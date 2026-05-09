package surrealdb

import (
	"context"
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// surrealGraphWriteRepo implements biz.GraphWriteRepo using SurrealDB.
// Key difference from PG adapter: EnqueueOutbox is a no-op because SurrealDB is a unified store.
type surrealGraphWriteRepo struct {
	client *Client
	log    *log.Helper
}

func NewSurrealGraphWriteRepo(client *Client, logger log.Logger) biz.GraphWriteRepo {
	return &surrealGraphWriteRepo{
		client: client,
		log:    log.NewHelper(logger),
	}
}

func (r *surrealGraphWriteRepo) UpsertEntity(ctx context.Context, entity biz.WriteEntity) (biz.UpsertOp, error) {
	if entity.EntityID == "" {
		entity.EntityID = uuid.NewString()
	}

	sql := `UPDATE type::thing('kg_entities', $entity_id) MERGE {
		entity_id: $entity_id,
		app_id: $app_id,
		tenant_id: $tenant_id,
		entity_type: $entity_type,
		name: $name,
		properties: $properties,
		confidence: $confidence,
		source_file: $source_file,
		chunk_id: $chunk_id,
		skill_id: $skill_id,
		version_id: $version_id,
		provenance_type: $provenance_type,
		domains: $domains,
		aliases: $aliases,
		version: $version,
		is_deleted: false,
		updated_at: time::now()
	}`

	_, err := r.client.Query(ctx, sql, map[string]any{
		"entity_id":       entity.EntityID,
		"app_id":          entity.AppID,
		"tenant_id":       entity.TenantID,
		"entity_type":     entity.EntityType,
		"name":            entity.Name,
		"properties":      entity.Properties,
		"confidence":      entity.Confidence,
		"source_file":     entity.SourceFile,
		"chunk_id":        entity.ChunkID,
		"skill_id":        entity.SkillID,
		"version_id":      entity.VersionID,
		"provenance_type": entity.ProvenanceType,
		"domains":         entity.Domains,
		"aliases":         entity.Aliases,
		"version":         entity.Version,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] UpsertEntity failed entity_id=%s err=%v", entity.EntityID, err)
		return "", fmt.Errorf("upsert entity: %w", err)
	}

	r.log.Infof("[KGS][SurrealDB] UpsertEntity done entity_id=%s type=%s", entity.EntityID, entity.EntityType)
	return biz.UpsertOpCreated, nil
}

func (r *surrealGraphWriteRepo) UpsertEdge(ctx context.Context, edge biz.WriteEdge) (biz.UpsertOp, error) {
	if edge.EdgeID == "" {
		edge.EdgeID = uuid.NewString()
	}

	sql := `UPDATE type::thing('kg_edges', $edge_id) MERGE {
		edge_id: $edge_id,
		app_id: $app_id,
		tenant_id: $tenant_id,
		from_entity_id: $from_entity_id,
		to_entity_id: $to_entity_id,
		relation_type: $relation_type,
		properties: $properties,
		confidence: $confidence,
		version_id: $version_id,
		is_deleted: false,
		updated_at: time::now()
	}`

	_, err := r.client.Query(ctx, sql, map[string]any{
		"edge_id":        edge.EdgeID,
		"app_id":         edge.AppID,
		"tenant_id":      edge.TenantID,
		"from_entity_id": edge.FromEntityID,
		"to_entity_id":   edge.ToEntityID,
		"relation_type":  edge.RelationType,
		"properties":     edge.Properties,
		"confidence":     edge.Confidence,
		"version_id":     edge.VersionID,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] UpsertEdge failed edge_id=%s err=%v", edge.EdgeID, err)
		return "", fmt.Errorf("upsert edge: %w", err)
	}

	r.log.Infof("[KGS][SurrealDB] UpsertEdge done edge_id=%s rel=%s", edge.EdgeID, edge.RelationType)
	return biz.UpsertOpCreated, nil
}

func (r *surrealGraphWriteRepo) SoftDeleteEntity(ctx context.Context, entityID, tenantID string) error {
	sql := `UPDATE kg_entities SET is_deleted = true, deleted_at = time::now(), updated_at = time::now()
		WHERE entity_id = $entity_id AND tenant_id = $tenant_id`
	_, err := r.client.Query(ctx, sql, map[string]any{
		"entity_id": entityID,
		"tenant_id": tenantID,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] SoftDeleteEntity failed entity_id=%s err=%v", entityID, err)
	}
	return err
}

func (r *surrealGraphWriteRepo) SoftDeleteEdge(ctx context.Context, edgeID, tenantID string) error {
	sql := `UPDATE kg_edges SET is_deleted = true, deleted_at = time::now(), updated_at = time::now()
		WHERE edge_id = $edge_id AND tenant_id = $tenant_id`
	_, err := r.client.Query(ctx, sql, map[string]any{
		"edge_id":   edgeID,
		"tenant_id": tenantID,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] SoftDeleteEdge failed edge_id=%s err=%v", edgeID, err)
	}
	return err
}

// EnqueueOutbox is a NO-OP for SurrealDB mode.
// SurrealDB is a unified store — there is no CQRS fan-out to Neo4j/Qdrant.
func (r *surrealGraphWriteRepo) EnqueueOutbox(ctx context.Context, rec biz.OutboxRecord) error {
	return nil // NO-OP: no outbox needed
}

// WithTx executes fn within a SurrealDB transaction.
func (r *surrealGraphWriteRepo) WithTx(ctx context.Context, fn func(txRepo biz.GraphWriteRepo) error) error {
	// SurrealDB Go SDK doesn't yet expose a native transaction API in the same way as GORM.
	// For now, execute directly — SurrealDB ensures atomicity per statement.
	// TODO: Use BEGIN TRANSACTION / COMMIT when SDK supports it.
	return fn(r)
}

var _ biz.GraphWriteRepo = (*surrealGraphWriteRepo)(nil)
