package overlay

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"
)

func (m *Manager) Commit(ctx context.Context, overlayID, conflictPolicy string) (*CommitResult, error) {
	return m.commit(ctx, overlayID, conflictPolicy, StatusCommitted)
}

func (m *Manager) CommitPartial(ctx context.Context, overlayID, conflictPolicy string) (*CommitResult, error) {
	return m.commit(ctx, overlayID, conflictPolicy, StatusPartial)
}

func (m *Manager) commit(ctx context.Context, overlayID, conflictPolicy string, status Status) (*CommitResult, error) {
	overlay, err := m.store.Get(ctx, overlayID)
	if err != nil {
		return nil, err
	}
	if overlay.Status != StatusActive && overlay.Status != StatusCreated {
		return nil, fmt.Errorf("overlay is not active")
	}

	latestVersionID := m.resolveBaseVersion(ctx, overlay.Namespace, "current")
	conflicts := DetectConflicts(overlay.BaseVersionID, latestVersionID)
	resolved, err := ResolveConflicts(conflictPolicy, conflicts)
	if err != nil {
		return nil, err
	}

	entitiesDeleted := len(overlay.DeletedNodeIDs)
	edgesDeleted := len(overlay.DeletedEdgeIDs)
	if m.graphRepo != nil && (entitiesDeleted > 0 || edgesDeleted > 0) {
		appID, tenantID, err := parseNamespaceScope(overlay.Namespace)
		if err != nil {
			return nil, err
		}
		if entitiesDeleted > 0 {
			deleted, _, err := m.graphRepo.BatchDeleteNodes(ctx, appID, tenantID, overlay.DeletedNodeIDs)
			if err != nil {
				return nil, err
			}
			entitiesDeleted = deleted
		}
		if edgesDeleted > 0 {
			for _, edgeID := range overlay.DeletedEdgeIDs {
				if err := m.graphRepo.DeleteEdge(ctx, appID, tenantID, edgeID); err != nil {
					return nil, err
				}
			}
		}
	}

	var newVersionID string
	if m.versionMgr != nil {
		newVersionID, err = m.versionMgr.CreateDelta(ctx, overlay.Namespace, version.ChangeSet{
			EntitiesAdded:   len(overlay.EntitiesDelta),
			EdgesAdded:      len(overlay.EdgesDelta),
			EntitiesDeleted: entitiesDeleted,
			EdgesDeleted:    edgesDeleted,
			CommitMessage:   "overlay commit: " + overlay.OverlayID,
		})
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	overlay.Status = status
	overlay.CommittedAt = &now
	if err := m.store.Save(ctx, overlay, time.Until(overlay.ExpiresAt)); err != nil {
		return nil, err
	}
	if overlay.SessionID != "" {
		_ = m.store.UnbindSession(ctx, overlay.SessionID)
	}
	observability.DecOverlayActive(overlay.Namespace)

	result := &CommitResult{
		NewVersionID:      newVersionID,
		EntitiesCommitted: len(overlay.EntitiesDelta),
		EdgesCommitted:    len(overlay.EdgesDelta),
		ConflictsResolved: resolved,
	}
	if m.publisher != nil {
		payload, _ := json.Marshal(map[string]any{
			"overlay_id":         overlay.OverlayID,
			"namespace":          overlay.Namespace,
			"session_id":         overlay.SessionID,
			"new_version_id":     result.NewVersionID,
			"entities_committed": result.EntitiesCommitted,
			"edges_committed":    result.EdgesCommitted,
			"conflicts_resolved": result.ConflictsResolved,
			"status":             overlay.Status,
		})
		_ = m.publisher.Publish(ctx, data.TopicOverlayCommitted(overlay.Namespace), payload)
	}
	return result, nil
}

func parseNamespaceScope(namespace string) (appID, tenantID string, err error) {
	parts := strings.Split(strings.TrimSpace(namespace), "/")
	if len(parts) < 3 || parts[0] != "graph" {
		return "", "", fmt.Errorf("invalid namespace: %q", namespace)
	}
	// graph/{app}/{tenant}
	if len(parts) == 3 {
		return parts[1], parts[2], nil
	}
	// graph/{org}/{app}/{tenant}
	return parts[len(parts)-2], parts[len(parts)-1], nil
}
