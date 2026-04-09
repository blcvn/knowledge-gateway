package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type entityEdgeSyncer interface {
	UpsertEntity(ctx context.Context, entity data.KGEntity) error
	UpsertEdge(ctx context.Context, edge data.KGEdge) error
	SoftDeleteEntity(ctx context.Context, entityID, tenantID string) error
	SoftDeleteEdge(ctx context.Context, edgeID, tenantID string) error
}

type vectorSyncer interface {
	UpsertVector(ctx context.Context, entity data.KGEntity) error
	DeleteVector(ctx context.Context, entityID, tenantID, appID string) error
}

type OutboxWorker struct {
	db           *gorm.DB
	neo4jSyncer  entityEdgeSyncer
	qdrantSyncer vectorSyncer
	pollInterval time.Duration
	batchSize    int
	maxAttempts  int
	log          *log.Helper
}

func NewOutboxWorker(cfg *conf.Data, db *gorm.DB, neo4jSyncer *Neo4jSyncer, qdrantSyncer *QdrantSyncer, logger log.Logger) *OutboxWorker {
	var neo entityEdgeSyncer = neo4jSyncer
	if neo == nil {
		neo = noopEntityEdgeSyncer{}
	}
	var qdrant vectorSyncer = qdrantSyncer
	if qdrant == nil {
		qdrant = noopVectorSyncer{}
	}
	pollInterval := 500 * time.Millisecond
	batchSize := 100
	maxAttempts := 10
	if cfg != nil && cfg.GetOutbox() != nil {
		outboxCfg := cfg.GetOutbox()
		if outboxCfg.GetPollIntervalMs() > 0 {
			pollInterval = time.Duration(outboxCfg.GetPollIntervalMs()) * time.Millisecond
		}
		if outboxCfg.GetBatchSize() > 0 {
			batchSize = int(outboxCfg.GetBatchSize())
		}
		if outboxCfg.GetMaxAttempts() > 0 {
			maxAttempts = int(outboxCfg.GetMaxAttempts())
		}
	}
	return &OutboxWorker{
		db:           db,
		neo4jSyncer:  neo,
		qdrantSyncer: qdrant,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		maxAttempts:  maxAttempts,
		log:          log.NewHelper(logger),
	}
}

func (w *OutboxWorker) Run(ctx context.Context) {
	if w == nil || w.db == nil {
		return
	}
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	if pending, err := data.CountPendingOutbox(ctx, w.db); err == nil {
		observability.SetOutboxPending(pending)
		if pending > 0 {
			w.log.Infof("[KGS][OutboxWorker] pending=%d", pending)
		}
	} else {
		w.log.Warnf("[KGS][OutboxWorker] count pending failed: %v", err)
	}

	recs, err := data.PollPendingOutbox(ctx, w.db, w.batchSize)
	if err != nil {
		w.log.Warnf("[KGS][OutboxWorker] poll failed batch_size=%d err=%v", w.batchSize, err)
		return
	}
	if len(recs) == 0 {
		observability.SetOutboxLagSeconds(0)
		return
	}
	w.log.Infof("[KGS][OutboxWorker] polled records=%d batch_size=%d", len(recs), w.batchSize)
	oldest := recs[0].CreatedAt
	if !oldest.IsZero() {
		observability.SetOutboxLagSeconds(time.Since(oldest).Seconds())
	}

	for _, rec := range recs {
		if w.maxAttempts > 0 && rec.Attempts >= w.maxAttempts {
			w.log.Warnf("[KGS][OutboxWorker] skip record id=%d op=%s attempts=%d max_attempts=%d",
				rec.ID, rec.Op, rec.Attempts, w.maxAttempts)
			continue
		}
		started := time.Now()
		w.log.Infof("[KGS][OutboxWorker] sync start id=%d op=%s tenant_id=%s app_id=%s entity_id=%s edge_id=%s attempts=%d",
			rec.ID, rec.Op, rec.TenantID, rec.AppID, outboxEntityID(rec), outboxEdgeID(rec), rec.Attempts)
		if err := data.MarkOutboxSyncing(ctx, w.db, rec.ID); err != nil {
			w.log.Warnf("[KGS][OutboxWorker] mark syncing failed id=%d op=%s err=%v", rec.ID, rec.Op, err)
			continue
		}
		err := w.syncOne(ctx, rec)
		observability.ObserveOutboxSync(rec.Op, time.Since(started), err)
		if err != nil {
			w.log.Errorf("[KGS][OutboxWorker] sync failed id=%d op=%s tenant_id=%s app_id=%s entity_id=%s edge_id=%s duration=%s err=%v",
				rec.ID, rec.Op, rec.TenantID, rec.AppID, outboxEntityID(rec), outboxEdgeID(rec), time.Since(started), err)
			observability.ObserveEntityWrite("outbox_sync", err)
			observability.IncOutboxFailed(rec.Op)
			if markErr := data.MarkOutboxFailed(ctx, w.db, rec.ID, truncateError(err)); markErr != nil {
				w.log.Warnf("[KGS][OutboxWorker] mark failed status update failed id=%d op=%s err=%v", rec.ID, rec.Op, markErr)
			}
			continue
		}
		w.log.Infof("[KGS][OutboxWorker] sync done id=%d op=%s tenant_id=%s app_id=%s entity_id=%s edge_id=%s duration=%s",
			rec.ID, rec.Op, rec.TenantID, rec.AppID, outboxEntityID(rec), outboxEdgeID(rec), time.Since(started))
		observability.ObserveEntityWrite("outbox_sync", nil)
		if err := data.MarkOutboxDone(ctx, w.db, rec.ID); err != nil {
			w.log.Warnf("[KGS][OutboxWorker] mark done failed id=%d op=%s err=%v", rec.ID, rec.Op, err)
		}
	}
}

func (w *OutboxWorker) syncOne(ctx context.Context, rec data.KGSyncOutbox) error {
	switch rec.Op {
	case data.OutboxOpUpsertEntity:
		var entity data.KGEntity
		if err := unmarshalPayload(rec.Payload, &entity); err != nil {
			return err
		}
		neo4jEntity := entity
		neo4jEntity.Properties, _ = data.NormalizePropertiesForNeo4j(entity.Properties)
		if err := w.neo4jSyncer.UpsertEntity(ctx, neo4jEntity); err != nil {
			return fmt.Errorf("neo4j upsert entity: %w", err)
		}
		if err := w.qdrantSyncer.UpsertVector(ctx, entity); err != nil {
			return fmt.Errorf("qdrant upsert vector: %w", err)
		}
		return nil
	case data.OutboxOpUpsertEdge:
		var edge data.KGEdge
		if err := unmarshalPayload(rec.Payload, &edge); err != nil {
			return err
		}
		neo4jEdge := edge
		neo4jEdge.Properties, _ = data.NormalizePropertiesForNeo4j(edge.Properties)
		if err := w.neo4jSyncer.UpsertEdge(ctx, neo4jEdge); err != nil {
			return fmt.Errorf("neo4j upsert edge: %w", err)
		}
		return nil
	case data.OutboxOpDeleteEntity:
		if rec.EntityID == nil || *rec.EntityID == "" {
			return fmt.Errorf("missing entity id for delete")
		}
		if err := w.neo4jSyncer.SoftDeleteEntity(ctx, *rec.EntityID, rec.TenantID); err != nil {
			return fmt.Errorf("neo4j delete entity: %w", err)
		}
		if err := w.qdrantSyncer.DeleteVector(ctx, *rec.EntityID, rec.TenantID, rec.AppID); err != nil {
			return fmt.Errorf("qdrant delete vector: %w", err)
		}
		return nil
	case data.OutboxOpDeleteEdge:
		if rec.EdgeID == nil || *rec.EdgeID == "" {
			return fmt.Errorf("missing edge id for delete")
		}
		if err := w.neo4jSyncer.SoftDeleteEdge(ctx, *rec.EdgeID, rec.TenantID); err != nil {
			return fmt.Errorf("neo4j delete edge: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported outbox op: %s", rec.Op)
	}
}

func unmarshalPayload(payload data.JSONMap, out any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) <= 1024 {
		return msg
	}
	return msg[:1024]
}

func outboxEntityID(rec data.KGSyncOutbox) string {
	if rec.EntityID == nil {
		return ""
	}
	return *rec.EntityID
}

func outboxEdgeID(rec data.KGSyncOutbox) string {
	if rec.EdgeID == nil {
		return ""
	}
	return *rec.EdgeID
}

type noopEntityEdgeSyncer struct{}

func (noopEntityEdgeSyncer) UpsertEntity(context.Context, data.KGEntity) error { return nil }
func (noopEntityEdgeSyncer) UpsertEdge(context.Context, data.KGEdge) error     { return nil }
func (noopEntityEdgeSyncer) SoftDeleteEntity(context.Context, string, string) error {
	return nil
}
func (noopEntityEdgeSyncer) SoftDeleteEdge(context.Context, string, string) error { return nil }

type noopVectorSyncer struct{}

func (noopVectorSyncer) UpsertVector(context.Context, data.KGEntity) error { return nil }
func (noopVectorSyncer) DeleteVector(context.Context, string, string, string) error {
	return nil
}
