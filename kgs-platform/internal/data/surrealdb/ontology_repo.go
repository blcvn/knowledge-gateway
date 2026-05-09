package surrealdb

import (
	"context"
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

// surrealOntologyRepo implements the ontology repository using SurrealDB.
// Unlike the PG+Redis adapter, this does NOT cache in Redis since SurrealDB lookups are fast enough.
type surrealOntologyRepo struct {
	client *Client
	log    *log.Helper
}

func NewSurrealOntologyRepo(client *Client, logger log.Logger) *surrealOntologyRepo {
	return &surrealOntologyRepo{client: client, log: log.NewHelper(logger)}
}

func (r *surrealOntologyRepo) GetEntityType(ctx context.Context, appID, name string) (*biz.EntityType, error) {
	sql := `SELECT * FROM kgs_entity_types WHERE app_id = $app_id AND name = $name AND deleted_at IS NONE LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{"app_id": appID, "name": name})
	if err != nil {
		return nil, err
	}
	et, err := unmarshalOne[biz.EntityType](result)
	if err != nil || et == nil {
		return nil, fmt.Errorf("entity type not found: app=%s name=%s", appID, name)
	}
	return et, nil
}

func (r *surrealOntologyRepo) GetRelationType(ctx context.Context, appID, name string) (*biz.RelationType, error) {
	sql := `SELECT * FROM kgs_relation_types WHERE app_id = $app_id AND name = $name AND deleted_at IS NONE LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{"app_id": appID, "name": name})
	if err != nil {
		return nil, err
	}
	rt, err := unmarshalOne[biz.RelationType](result)
	if err != nil || rt == nil {
		return nil, fmt.Errorf("relation type not found: app=%s name=%s", appID, name)
	}
	return rt, nil
}

// InvalidateEntityType is a no-op for SurrealDB — no Redis cache to invalidate.
func (r *surrealOntologyRepo) InvalidateEntityType(ctx context.Context, appID, name string) error {
	return nil
}

// InvalidateRelationType is a no-op for SurrealDB — no Redis cache to invalidate.
func (r *surrealOntologyRepo) InvalidateRelationType(ctx context.Context, appID, name string) error {
	return nil
}
