package shadow

import (
	"context"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

// ShadowGraphWriteRepo wraps primary and secondary GraphWriteRepo for shadow mode.
// All writes go to primary (sync) + secondary (async).
// EnqueueOutbox: primary handles outbox normally, secondary ignores it.
type ShadowGraphWriteRepo struct {
	primary   biz.GraphWriteRepo
	secondary biz.GraphWriteRepo
	log       *log.Helper
}

func NewShadowGraphWriteRepo(primary, secondary biz.GraphWriteRepo, logger log.Logger) biz.GraphWriteRepo {
	return &ShadowGraphWriteRepo{
		primary:   primary,
		secondary: secondary,
		log:       log.NewHelper(logger),
	}
}

func (s *ShadowGraphWriteRepo) UpsertEntity(ctx context.Context, entity biz.WriteEntity) (biz.UpsertOp, error) {
	op, err := s.primary.UpsertEntity(ctx, entity)
	go func() {
		if _, e := s.secondary.UpsertEntity(ctx, entity); e != nil {
			s.log.Warnf("[Shadow] UpsertEntity secondary failed entity_id=%s: %v", entity.EntityID, e)
		}
	}()
	return op, err
}

func (s *ShadowGraphWriteRepo) UpsertEdge(ctx context.Context, edge biz.WriteEdge) (biz.UpsertOp, error) {
	op, err := s.primary.UpsertEdge(ctx, edge)
	go func() {
		if _, e := s.secondary.UpsertEdge(ctx, edge); e != nil {
			s.log.Warnf("[Shadow] UpsertEdge secondary failed edge_id=%s: %v", edge.EdgeID, e)
		}
	}()
	return op, err
}

func (s *ShadowGraphWriteRepo) SoftDeleteEntity(ctx context.Context, entityID, tenantID string) error {
	err := s.primary.SoftDeleteEntity(ctx, entityID, tenantID)
	go func() {
		if e := s.secondary.SoftDeleteEntity(ctx, entityID, tenantID); e != nil {
			s.log.Warnf("[Shadow] SoftDeleteEntity secondary failed entity_id=%s: %v", entityID, e)
		}
	}()
	return err
}

func (s *ShadowGraphWriteRepo) SoftDeleteEdge(ctx context.Context, edgeID, tenantID string) error {
	err := s.primary.SoftDeleteEdge(ctx, edgeID, tenantID)
	go func() {
		if e := s.secondary.SoftDeleteEdge(ctx, edgeID, tenantID); e != nil {
			s.log.Warnf("[Shadow] SoftDeleteEdge secondary failed edge_id=%s: %v", edgeID, e)
		}
	}()
	return err
}

func (s *ShadowGraphWriteRepo) EnqueueOutbox(ctx context.Context, rec biz.OutboxRecord) error {
	// Primary handles outbox (Specialized mode). Secondary ignores it (SurrealDB mode).
	return s.primary.EnqueueOutbox(ctx, rec)
}

func (s *ShadowGraphWriteRepo) WithTx(ctx context.Context, fn func(txRepo biz.GraphWriteRepo) error) error {
	// Transaction only on primary — shadow writes are async/best-effort
	return s.primary.WithTx(ctx, fn)
}

var _ biz.GraphWriteRepo = (*ShadowGraphWriteRepo)(nil)
