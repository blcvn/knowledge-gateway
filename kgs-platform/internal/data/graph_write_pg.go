package data

import (
	"context"
	"fmt"
	stdlog "log"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

type pgWriteRepo struct {
	db *gorm.DB
	tx *gorm.DB
}

func NewPGWriteRepo(db *gorm.DB) biz.GraphWriteRepo {
	return &pgWriteRepo{db: db}
}

func (r *pgWriteRepo) UpsertEntity(ctx context.Context, entity biz.WriteEntity) (biz.UpsertOp, error) {
	started := time.Now()
	stdlog.Printf("[KGS][PGWriteRepo] UpsertEntity start app_id=%s tenant_id=%s entity_id=%s entity_type=%s props_keys=%d",
		entity.AppID, entity.TenantID, entity.EntityID, entity.EntityType, len(entity.Properties))
	handle := r.dbHandle(ctx)
	op, err := upsertEntityTx(handle, KGEntity{
		EntityID:       entity.EntityID,
		AppID:          entity.AppID,
		TenantID:       entity.TenantID,
		EntityType:     entity.EntityType,
		Name:           entity.Name,
		Properties:     JSONMap(entity.Properties),
		Confidence:     entity.Confidence,
		SourceFile:     entity.SourceFile,
		ChunkID:        entity.ChunkID,
		SkillID:        entity.SkillID,
		VersionID:      entity.VersionID,
		ProvenanceType: entity.ProvenanceType,
		Domains:        StringArr(entity.Domains),
		Aliases:        StringArr(entity.Aliases),
		Version:        entity.Version,
	})
	if err != nil {
		stdlog.Printf("[KGS][PGWriteRepo] UpsertEntity failed app_id=%s tenant_id=%s entity_id=%s entity_type=%s err=%v",
			entity.AppID, entity.TenantID, entity.EntityID, entity.EntityType, err)
		return biz.UpsertOpConflict, err
	}
	result := toBizUpsertOp(op)
	stdlog.Printf("[KGS][PGWriteRepo] UpsertEntity done app_id=%s tenant_id=%s entity_id=%s entity_type=%s op=%s duration=%s",
		entity.AppID, entity.TenantID, entity.EntityID, entity.EntityType, result, time.Since(started))
	return result, nil
}

func (r *pgWriteRepo) UpsertEdge(ctx context.Context, edge biz.WriteEdge) (biz.UpsertOp, error) {
	started := time.Now()
	stdlog.Printf("[KGS][PGWriteRepo] UpsertEdge start app_id=%s tenant_id=%s edge_id=%s relation=%s from=%s to=%s props_keys=%d",
		edge.AppID, edge.TenantID, edge.EdgeID, edge.RelationType, edge.FromEntityID, edge.ToEntityID, len(edge.Properties))
	handle := r.dbHandle(ctx)
	op, err := upsertEdgeTx(handle, KGEdge{
		EdgeID:       edge.EdgeID,
		AppID:        edge.AppID,
		TenantID:     edge.TenantID,
		FromEntityID: edge.FromEntityID,
		ToEntityID:   edge.ToEntityID,
		RelationType: edge.RelationType,
		Properties:   JSONMap(edge.Properties),
		Confidence:   edge.Confidence,
		VersionID:    edge.VersionID,
	})
	if err != nil {
		stdlog.Printf("[KGS][PGWriteRepo] UpsertEdge failed app_id=%s tenant_id=%s edge_id=%s relation=%s from=%s to=%s err=%v",
			edge.AppID, edge.TenantID, edge.EdgeID, edge.RelationType, edge.FromEntityID, edge.ToEntityID, err)
		return biz.UpsertOpConflict, err
	}
	result := toBizUpsertOp(op)
	stdlog.Printf("[KGS][PGWriteRepo] UpsertEdge done app_id=%s tenant_id=%s edge_id=%s relation=%s from=%s to=%s op=%s duration=%s",
		edge.AppID, edge.TenantID, edge.EdgeID, edge.RelationType, edge.FromEntityID, edge.ToEntityID, result, time.Since(started))
	return result, nil
}

func (r *pgWriteRepo) SoftDeleteEntity(ctx context.Context, entityID, tenantID string) error {
	started := time.Now()
	stdlog.Printf("[KGS][PGWriteRepo] SoftDeleteEntity start tenant_id=%s entity_id=%s", tenantID, entityID)
	err := softDeleteEntityPG(ctx, r.dbHandle(ctx), entityID, tenantID)
	if err != nil {
		stdlog.Printf("[KGS][PGWriteRepo] SoftDeleteEntity failed tenant_id=%s entity_id=%s err=%v", tenantID, entityID, err)
		return err
	}
	stdlog.Printf("[KGS][PGWriteRepo] SoftDeleteEntity done tenant_id=%s entity_id=%s duration=%s", tenantID, entityID, time.Since(started))
	return nil
}

func (r *pgWriteRepo) SoftDeleteEdge(ctx context.Context, edgeID, tenantID string) error {
	started := time.Now()
	stdlog.Printf("[KGS][PGWriteRepo] SoftDeleteEdge start tenant_id=%s edge_id=%s", tenantID, edgeID)
	err := softDeleteEdgePG(ctx, r.dbHandle(ctx), edgeID, tenantID)
	if err != nil {
		stdlog.Printf("[KGS][PGWriteRepo] SoftDeleteEdge failed tenant_id=%s edge_id=%s err=%v", tenantID, edgeID, err)
		return err
	}
	stdlog.Printf("[KGS][PGWriteRepo] SoftDeleteEdge done tenant_id=%s edge_id=%s duration=%s", tenantID, edgeID, time.Since(started))
	return nil
}

func (r *pgWriteRepo) EnqueueOutbox(ctx context.Context, rec biz.OutboxRecord) error {
	started := time.Now()
	entityID := ""
	if rec.EntityID != nil {
		entityID = *rec.EntityID
	}
	edgeID := ""
	if rec.EdgeID != nil {
		edgeID = *rec.EdgeID
	}
	stdlog.Printf("[KGS][PGWriteRepo] EnqueueOutbox start op=%s app_id=%s tenant_id=%s entity_id=%s edge_id=%s payload_keys=%d",
		rec.Op, rec.AppID, rec.TenantID, entityID, edgeID, len(rec.Payload))
	traceCtx, span := observability.StartDependencySpan(ctx, "postgres", "kg.outbox.enqueue", attribute.String("kg.outbox.op", rec.Op))
	defer span.End()
	record := KGSyncOutbox{
		Op:       rec.Op,
		EntityID: rec.EntityID,
		EdgeID:   rec.EdgeID,
		TenantID: rec.TenantID,
		AppID:    rec.AppID,
		Payload:  JSONMap(rec.Payload),
		Status:   rec.Status,
	}
	err := enqueueOutboxTx(r.dbHandle(traceCtx), record)
	if err != nil {
		observability.RecordSpanError(span, err)
		stdlog.Printf("[KGS][PGWriteRepo] EnqueueOutbox failed op=%s app_id=%s tenant_id=%s entity_id=%s edge_id=%s err=%v",
			rec.Op, rec.AppID, rec.TenantID, entityID, edgeID, err)
		return err
	}
	stdlog.Printf("[KGS][PGWriteRepo] EnqueueOutbox done op=%s app_id=%s tenant_id=%s entity_id=%s edge_id=%s duration=%s",
		rec.Op, rec.AppID, rec.TenantID, entityID, edgeID, time.Since(started))
	return nil
}

func (r *pgWriteRepo) WithTx(ctx context.Context, fn func(txRepo biz.GraphWriteRepo) error) error {
	if fn == nil {
		return nil
	}
	if r == nil || r.db == nil {
		return fmt.Errorf("pgWriteRepo is not configured")
	}
	if r.tx != nil {
		return fn(r)
	}
	started := time.Now()
	stdlog.Printf("[KGS][PGWriteRepo] WithTx start")
	traceCtx, span := observability.StartDependencySpan(ctx, "postgres", "kg.postgres.tx")
	defer span.End()
	err := r.db.WithContext(traceCtx).Transaction(func(tx *gorm.DB) error {
		child := &pgWriteRepo{db: r.db, tx: tx}
		return fn(child)
	})
	if err != nil {
		observability.RecordSpanError(span, err)
		stdlog.Printf("[KGS][PGWriteRepo] WithTx failed err=%v duration=%s", err, time.Since(started))
		return err
	}
	stdlog.Printf("[KGS][PGWriteRepo] WithTx done duration=%s", time.Since(started))
	return nil
}

func (r *pgWriteRepo) dbHandle(ctx context.Context) *gorm.DB {
	if r == nil {
		return nil
	}
	if r.tx != nil {
		return r.tx.WithContext(ctx)
	}
	if r.db == nil {
		return nil
	}
	return r.db.WithContext(ctx)
}

func toBizUpsertOp(op upsertOp) biz.UpsertOp {
	switch op {
	case opCreated:
		return biz.UpsertOpCreated
	case opUpdated:
		return biz.UpsertOpUpdated
	default:
		return biz.UpsertOpConflict
	}
}
