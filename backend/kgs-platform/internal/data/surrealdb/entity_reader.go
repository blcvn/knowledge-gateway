package surrealdb

import (
	"context"
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

// surrealEntityReader implements biz.EntityReader using SurrealDB.
// Unlike the PG+Neo4j adapter, there is no stale-version detection since SurrealDB is the single store.
type surrealEntityReader struct {
	client *Client
	log    *log.Helper
}

func NewSurrealEntityReader(client *Client, logger log.Logger) biz.EntityReader {
	return &surrealEntityReader{
		client: client,
		log:    log.NewHelper(logger),
	}
}

func (r *surrealEntityReader) GetEntity(ctx context.Context, appID, tenantID, entityID string) (map[string]any, error) {
	sql := `SELECT * FROM kg_entities WHERE entity_id = $entity_id AND app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{
		"entity_id": entityID,
		"app_id":    appID,
		"tenant_id": tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("get entity %s: %w", entityID, err)
	}

	entities, err := unmarshalSlice[map[string]any](result)
	if err != nil || len(entities) == 0 {
		return nil, fmt.Errorf("entity not found: %s", entityID)
	}

	return normalizeEntityMap(entities[0]), nil
}

// EnrichWithFreshVersions returns entities as-is because SurrealDB is the single source-of-truth.
// There is no version drift between PG and Neo4j to reconcile.
func (r *surrealEntityReader) EnrichWithFreshVersions(ctx context.Context, appID, tenantID string, entities []map[string]any) ([]map[string]any, error) {
	// Single store = always fresh — no enrichment needed
	out := make([]map[string]any, 0, len(entities))
	for _, entity := range entities {
		out = append(out, normalizeEntityMap(entity))
	}
	return out, nil
}

func normalizeEntityMap(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	// Normalize field aliases
	if out["id"] == nil && out["entity_id"] != nil {
		out["id"] = out["entity_id"]
	}
	if out["entity_type"] == nil && out["label"] != nil {
		out["entity_type"] = out["label"]
	}
	if out["label"] == nil && out["entity_type"] != nil {
		out["label"] = out["entity_type"]
	}
	return out
}

var _ biz.EntityReader = (*surrealEntityReader)(nil)
