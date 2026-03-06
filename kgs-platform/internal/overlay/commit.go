package overlay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kgs-platform/internal/data"
	"kgs-platform/internal/observability"
	"kgs-platform/internal/version"
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

	var newVersionID string
	if m.versionMgr != nil {
		newVersionID, err = m.versionMgr.CreateDelta(ctx, overlay.Namespace, version.ChangeSet{
			EntitiesAdded: len(overlay.EntitiesDelta),
			EdgesAdded:    len(overlay.EdgesDelta),
			CommitMessage: "overlay commit: " + overlay.OverlayID,
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
