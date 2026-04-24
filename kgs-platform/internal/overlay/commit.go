package overlay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
	if (overlay.Status == status || overlay.Status == StatusCommitted || overlay.Status == StatusPartial) && overlay.CommittedAt != nil {
		return &CommitResult{
			NewVersionID:      overlay.CommittedVersionID,
			EntitiesCommitted: len(overlay.EntitiesDelta),
			EdgesCommitted:    len(overlay.EdgesDelta),
			ConflictsResolved: 0,
		}, nil
	}
	if overlay.Status != StatusActive && overlay.Status != StatusCreated {
		return nil, fmt.Errorf("overlay is not active")
	}
	if m.db == nil {
		return nil, fmt.Errorf("overlay manager db is not configured")
	}
	appID, tenantID, err := parseNamespaceScope(overlay.Namespace)
	if err != nil {
		return nil, err
	}

	latestVersionID := m.resolveBaseVersion(ctx, overlay.Namespace, "current")
	conflicts := DetectConflicts(overlay.BaseVersionID, latestVersionID)
	resolved, err := ResolveConflicts(conflictPolicy, conflicts)
	if err != nil {
		return nil, err
	}

	entitiesDeleted := len(overlay.DeletedNodeIDs)
	edgesDeleted := len(overlay.DeletedEdgeIDs)
	entityDeltas := dedupeEntityDeltas(overlay.EntitiesDelta)
	edgeDeltas := dedupeEdgeDeltas(overlay.EdgesDelta)

	if err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, delta := range entityDeltas {
			entity, err := entityDeltaToKGEntity(delta, overlay.Namespace)
			if err != nil {
				return err
			}
			if _, err := upsertEntityForCommit(tx, entity, conflictPolicy); err != nil {
				return err
			}
			rec, err := entityOutboxRecord(data.OutboxOpUpsertEntity, entity)
			if err != nil {
				return err
			}
			if err := data.EnqueueOutboxTx(tx, rec); err != nil {
				return err
			}
		}

		for _, delta := range edgeDeltas {
			edge, err := edgeDeltaToKGEdge(delta, overlay.Namespace)
			if err != nil {
				return err
			}
			if _, err := data.UpsertEdgeTx(tx, edge); err != nil {
				return err
			}
			rec, err := edgeOutboxRecord(data.OutboxOpUpsertEdge, edge)
			if err != nil {
				return err
			}
			if err := data.EnqueueOutboxTx(tx, rec); err != nil {
				return err
			}
		}

		for _, nodeID := range overlay.DeletedNodeIDs {
			if err := data.SoftDeleteEntityPG(ctx, tx, nodeID, tenantID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			rec := data.KGSyncOutbox{
				Op:       data.OutboxOpDeleteEntity,
				EntityID: stringPtr(nodeID),
				TenantID: tenantID,
				AppID:    appID,
				Payload: data.JSONMap{
					"entity_id": nodeID,
					"tenant_id": tenantID,
					"app_id":    appID,
				},
				Status: data.OutboxStatusPending,
			}
			if err := data.EnqueueOutboxTx(tx, rec); err != nil {
				return err
			}
		}

		for _, edgeID := range overlay.DeletedEdgeIDs {
			if err := data.SoftDeleteEdgePG(ctx, tx, edgeID, tenantID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			rec := data.KGSyncOutbox{
				Op:       data.OutboxOpDeleteEdge,
				EdgeID:   stringPtr(edgeID),
				TenantID: tenantID,
				AppID:    appID,
				Payload: data.JSONMap{
					"edge_id":   edgeID,
					"tenant_id": tenantID,
					"app_id":    appID,
				},
				Status: data.OutboxStatusPending,
			}
			if err := data.EnqueueOutboxTx(tx, rec); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	var newVersionID string
	if m.versionMgr != nil {
		newVersionID, err = m.versionMgr.CreateDelta(ctx, overlay.Namespace, version.ChangeSet{
			EntitiesAdded:   len(entityDeltas),
			EdgesAdded:      len(edgeDeltas),
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
	overlay.CommittedVersionID = newVersionID
	if err := m.store.Save(ctx, overlay, time.Until(overlay.ExpiresAt)); err != nil {
		return nil, err
	}
	if err := m.store.Delete(ctx, overlayID); err != nil {
		return nil, err
	}
	if overlay.SessionID != "" {
		_ = m.store.UnbindSession(ctx, overlay.SessionID)
	}
	observability.DecOverlayActive(overlay.Namespace)

	result := &CommitResult{
		NewVersionID:      newVersionID,
		EntitiesCommitted: len(entityDeltas),
		EdgesCommitted:    len(edgeDeltas),
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
		go func(subject string, body []byte) {
			_ = m.publisher.Publish(context.Background(), subject, body)
		}(data.TopicOverlayCommitted(overlay.Namespace), payload)
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

func entityDeltaToKGEntity(delta EntityDelta, namespace string) (data.KGEntity, error) {
	appID, tenantID, err := parseNamespaceScope(namespace)
	if err != nil {
		return data.KGEntity{}, err
	}
	props := cloneMap(delta.Properties)
	entityID := strings.TrimSpace(delta.ID)
	if entityID == "" {
		entityID = strings.TrimSpace(fmt.Sprint(props["id"]))
	}
	if entityID == "" {
		return data.KGEntity{}, fmt.Errorf("entity delta missing id")
	}
	props["id"] = entityID
	name := strings.TrimSpace(fmt.Sprint(props["name"]))
	if name == "" || name == "<nil>" {
		name = entityID
	}
	version := toInt(props["version"])
	return data.KGEntity{
		EntityID:   entityID,
		AppID:      appID,
		TenantID:   tenantID,
		EntityType: delta.Label,
		Name:       name,
		Properties: data.JSONMap(props),
		Confidence: toFloat(props["confidence"], 1.0),
		Version:    version,
	}, nil
}

func edgeDeltaToKGEdge(delta EdgeDelta, namespace string) (data.KGEdge, error) {
	appID, tenantID, err := parseNamespaceScope(namespace)
	if err != nil {
		return data.KGEdge{}, err
	}
	props := cloneMap(delta.Properties)
	edgeID := strings.TrimSpace(delta.ID)
	if edgeID == "" {
		edgeID = strings.TrimSpace(fmt.Sprint(props["id"]))
	}
	if edgeID == "" {
		return data.KGEdge{}, fmt.Errorf("edge delta missing id")
	}
	props["id"] = edgeID
	return data.KGEdge{
		EdgeID:       edgeID,
		AppID:        appID,
		TenantID:     tenantID,
		FromEntityID: delta.SourceID,
		ToEntityID:   delta.TargetID,
		RelationType: delta.Type,
		Properties:   data.JSONMap(props),
		Confidence:   toFloat(props["confidence"], 1.0),
		VersionID:    normalizeOptionalUUIDValue(props["version_id"]),
	}, nil
}

func entityOutboxRecord(op string, entity data.KGEntity) (data.KGSyncOutbox, error) {
	payload, err := toPayload(entity)
	if err != nil {
		return data.KGSyncOutbox{}, err
	}
	entityID := entity.EntityID
	return data.KGSyncOutbox{
		Op:       op,
		EntityID: &entityID,
		TenantID: entity.TenantID,
		AppID:    entity.AppID,
		Payload:  payload,
		Status:   data.OutboxStatusPending,
	}, nil
}

func edgeOutboxRecord(op string, edge data.KGEdge) (data.KGSyncOutbox, error) {
	payload, err := toPayload(edge)
	if err != nil {
		return data.KGSyncOutbox{}, err
	}
	edgeID := edge.EdgeID
	return data.KGSyncOutbox{
		Op:       op,
		EdgeID:   &edgeID,
		TenantID: edge.TenantID,
		AppID:    edge.AppID,
		Payload:  payload,
		Status:   data.OutboxStatusPending,
	}, nil
}

func toPayload(v any) (data.JSONMap, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return data.JSONMap(out), nil
}

func stringPtr(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	out := trimmed
	return &out
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float32:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

func toFloat(v any, def float64) float64 {
	switch x := v.(type) {
	case float32:
		return float64(x)
	case float64:
		return x
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	default:
		return def
	}
}

func normalizeOptionalUUIDValue(v any) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" {
		return ""
	}
	switch strings.ToLower(s) {
	case "<nil>", "nil", "null":
		return ""
	}
	if _, err := uuid.Parse(s); err != nil {
		return ""
	}
	return s
}

func upsertEntityForCommit(tx *gorm.DB, entity data.KGEntity, conflictPolicy string) (data.UpsertOp, error) {
	if strings.EqualFold(strings.TrimSpace(conflictPolicy), PolicyRequireManual) {
		return data.UpsertEntityTx(tx, entity)
	}
	return data.UpsertEntityTxOverlay(tx, entity)
}

func dedupeEntityDeltas(in []EntityDelta) []EntityDelta {
	if len(in) <= 1 {
		return in
	}
	lastByID := make(map[string]int, len(in))
	for i, delta := range in {
		id := normalizeDeltaEntityID(delta)
		if id == "" {
			continue
		}
		lastByID[id] = i
	}

	out := make([]EntityDelta, 0, len(in))
	for i, delta := range in {
		id := normalizeDeltaEntityID(delta)
		if id == "" {
			out = append(out, delta)
			continue
		}
		if lastByID[id] == i {
			out = append(out, delta)
		}
	}
	return out
}

func dedupeEdgeDeltas(in []EdgeDelta) []EdgeDelta {
	if len(in) <= 1 {
		return in
	}
	lastByID := make(map[string]int, len(in))
	lastBySemanticKey := make(map[string]int, len(in))
	for i, delta := range in {
		if id := normalizeDeltaEdgeID(delta); id != "" {
			lastByID[id] = i
		}
		if key := normalizeDeltaEdgeSemanticKey(delta); key != "" {
			lastBySemanticKey[key] = i
		}
	}

	out := make([]EdgeDelta, 0, len(in))
	for i, delta := range in {
		if id := normalizeDeltaEdgeID(delta); id != "" && lastByID[id] != i {
			continue
		}
		if key := normalizeDeltaEdgeSemanticKey(delta); key != "" && lastBySemanticKey[key] != i {
			continue
		}
		out = append(out, delta)
	}
	return out
}

func normalizeDeltaEntityID(delta EntityDelta) string {
	id := normalizeOptionalString(delta.ID)
	if id != "" {
		return id
	}
	return normalizeOptionalString(delta.Properties["id"])
}

func normalizeDeltaEdgeID(delta EdgeDelta) string {
	id := normalizeOptionalString(delta.ID)
	if id != "" {
		return id
	}
	return normalizeOptionalString(delta.Properties["id"])
}

func normalizeDeltaEdgeSemanticKey(delta EdgeDelta) string {
	sourceID := normalizeOptionalString(delta.SourceID)
	targetID := normalizeOptionalString(delta.TargetID)
	relationType := normalizeOptionalString(delta.Type)
	if sourceID == "" || targetID == "" || relationType == "" {
		return ""
	}
	return sourceID + "|" + targetID + "|" + strings.ToUpper(relationType)
}

func normalizeOptionalString(v any) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	switch strings.ToLower(s) {
	case "", "<nil>", "nil", "null":
		return ""
	default:
		return s
	}
}
