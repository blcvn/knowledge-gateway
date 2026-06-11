package overlay

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (m *Manager) AddEntityDelta(ctx context.Context, overlayID, namespace, label string, properties map[string]any) (map[string]any, error) {
	item, err := m.store.Get(ctx, overlayID)
	if err != nil {
		return nil, err
	}
	if err := validateWritableOverlay(item, namespace); err != nil {
		return nil, err
	}

	id := ensureID(properties)
	delta := EntityDelta{
		ID:         id,
		Label:      label,
		Properties: cloneMap(properties),
	}
	item.EntitiesDelta = append(item.EntitiesDelta, delta)
	if err := m.store.Save(ctx, item, 0); err != nil {
		return nil, err
	}

	out := cloneMap(properties)
	out["id"] = id
	out["label"] = label
	out["overlay_id"] = overlayID
	return out, nil
}

func (m *Manager) AddEdgeDelta(ctx context.Context, overlayID, namespace, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error) {
	item, err := m.store.Get(ctx, overlayID)
	if err != nil {
		return nil, err
	}
	if err := validateWritableOverlay(item, namespace); err != nil {
		return nil, err
	}

	id := ensureID(properties)
	delta := EdgeDelta{
		ID:         id,
		SourceID:   sourceNodeID,
		TargetID:   targetNodeID,
		Type:       relationType,
		Properties: cloneMap(properties),
	}
	item.EdgesDelta = append(item.EdgesDelta, delta)
	if err := m.store.Save(ctx, item, 0); err != nil {
		return nil, err
	}

	out := cloneMap(properties)
	out["id"] = id
	out["relation_type"] = relationType
	out["source_node_id"] = sourceNodeID
	out["target_node_id"] = targetNodeID
	out["overlay_id"] = overlayID
	return out, nil
}

func validateWritableOverlay(item *OverlayGraph, namespace string) error {
	if item == nil {
		return fmt.Errorf("overlay not found")
	}
	if item.Status != StatusActive && item.Status != StatusCreated {
		return fmt.Errorf("overlay is not active")
	}
	if namespace != "" && item.Namespace != namespace {
		return fmt.Errorf("overlay namespace mismatch")
	}
	return nil
}

func ensureID(properties map[string]any) string {
	if properties == nil {
		return uuid.NewString()
	}
	if id, ok := properties["id"].(string); ok && id != "" {
		return id
	}
	id := uuid.NewString()
	properties["id"] = id
	return id
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
