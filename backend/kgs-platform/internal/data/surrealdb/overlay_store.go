package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// surrealOverlayStore implements overlay.Store using SurrealDB tables.
type surrealOverlayStore struct {
	client *Client
	log    *log.Helper
}

func NewSurrealOverlayStore(client *Client, logger log.Logger) *surrealOverlayStore {
	return &surrealOverlayStore{
		client: client,
		log:    log.NewHelper(logger),
	}
}

// SaveOverlay persists an overlay graph with TTL.
func (s *surrealOverlayStore) SaveOverlay(ctx context.Context, overlayID, namespace string, entityDeltas, edgeDeltas []map[string]any, deleteEntityIDs, deleteEdgeIDs []string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)
	sql := `UPDATE type::thing('kg_overlays', $overlay_id) MERGE {
		overlay_id: $overlay_id,
		namespace: $namespace,
		entity_deltas: $entity_deltas,
		edge_deltas: $edge_deltas,
		delete_entity_ids: $delete_entity_ids,
		delete_edge_ids: $delete_edge_ids,
		created_at: time::now(),
		expires_at: $expires_at
	}`
	_, err := s.client.Query(ctx, sql, map[string]any{
		"overlay_id":        overlayID,
		"namespace":         namespace,
		"entity_deltas":     entityDeltas,
		"edge_deltas":       edgeDeltas,
		"delete_entity_ids": deleteEntityIDs,
		"delete_edge_ids":   deleteEdgeIDs,
		"expires_at":        expiresAt,
	})
	return err
}

// GetOverlay retrieves an overlay by ID.
func (s *surrealOverlayStore) GetOverlay(ctx context.Context, overlayID string) (map[string]any, error) {
	sql := `SELECT * FROM kg_overlays WHERE overlay_id = $overlay_id AND expires_at > time::now() LIMIT 1`
	result, err := s.client.Query(ctx, sql, map[string]any{"overlay_id": overlayID})
	if err != nil {
		return nil, err
	}
	overlays, err := unmarshalSlice[map[string]any](result)
	if err != nil || len(overlays) == 0 {
		return nil, fmt.Errorf("overlay not found: %s", overlayID)
	}
	return overlays[0], nil
}

// DeleteOverlay removes an overlay.
func (s *surrealOverlayStore) DeleteOverlay(ctx context.Context, overlayID string) error {
	sql := `DELETE FROM kg_overlays WHERE overlay_id = $overlay_id`
	_, err := s.client.Query(ctx, sql, map[string]any{"overlay_id": overlayID})
	return err
}

// BindSession links a session ID to an overlay.
func (s *surrealOverlayStore) BindSession(ctx context.Context, sessionID, overlayID string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)
	sql := `UPDATE type::thing('kg_overlay_sessions', $session_id) MERGE {
		session_id: $session_id,
		overlay_id: $overlay_id,
		created_at: time::now(),
		expires_at: $expires_at
	}`
	_, err := s.client.Query(ctx, sql, map[string]any{
		"session_id": sessionID,
		"overlay_id": overlayID,
		"expires_at": expiresAt,
	})
	return err
}

// UnbindSession removes a session binding.
func (s *surrealOverlayStore) UnbindSession(ctx context.Context, sessionID string) error {
	sql := `DELETE FROM kg_overlay_sessions WHERE session_id = $session_id`
	_, err := s.client.Query(ctx, sql, map[string]any{"session_id": sessionID})
	return err
}

// FindBySession looks up the overlay ID bound to a session.
func (s *surrealOverlayStore) FindBySession(ctx context.Context, sessionID string) (string, error) {
	sql := `SELECT overlay_id FROM kg_overlay_sessions WHERE session_id = $session_id AND expires_at > time::now() LIMIT 1`
	result, err := s.client.Query(ctx, sql, map[string]any{"session_id": sessionID})
	if err != nil {
		return "", err
	}
	rows, err := unmarshalSlice[map[string]any](result)
	if err != nil || len(rows) == 0 {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}
	overlayID, _ := rows[0]["overlay_id"].(string)
	return overlayID, nil
}
