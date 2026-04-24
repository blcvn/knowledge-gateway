package batch

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ConflictPolicyFailFast = "FAIL_FAST"
	ConflictPolicySkip     = "SKIP"
)

type Edge struct {
	EdgeID       string         `json:"edgeId"`
	FromEntityID string         `json:"fromEntityId"`
	ToEntityID   string         `json:"toEntityId"`
	RelationType string         `json:"relationType"`
	Properties   map[string]any `json:"properties,omitempty"`
	Confidence   float64        `json:"confidence,omitempty"`
	VersionID    string         `json:"versionId,omitempty"`
}

type GraphBatchRequest struct {
	Entities       []Entity `json:"entities"`
	Edges          []Edge   `json:"edges"`
	OverlayID      *string  `json:"overlayId,omitempty"`
	ConflictPolicy string   `json:"conflictPolicy,omitempty"`
}

type GraphBatchResult struct {
	EntitiesCreated int      `json:"entitiesCreated"`
	EntitiesUpdated int      `json:"entitiesUpdated"`
	EntitiesSkipped int      `json:"entitiesSkipped"`
	EdgesCreated    int      `json:"edgesCreated"`
	EdgesSkipped    int      `json:"edgesSkipped"`
	Conflicted      int      `json:"conflicted"`
	Errors          []string `json:"errors,omitempty"`
}

type OverlayDeltaAppender interface {
	AddEntityDelta(ctx context.Context, overlayID, namespace, label string, properties map[string]any) (map[string]any, error)
	AddEdgeDelta(ctx context.Context, overlayID, namespace, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error)
}

type GraphBatchHandler struct {
	db      *gorm.DB
	overlay OverlayDeltaAppender
}

func NewGraphBatchHandler(db *gorm.DB, overlay OverlayDeltaAppender) *GraphBatchHandler {
	return &GraphBatchHandler{db: db, overlay: overlay}
}

func (h *GraphBatchHandler) UpsertGraph(ctx context.Context, req GraphBatchRequest, appID, tenantID string) (*GraphBatchResult, error) {
	started := time.Now()
	res := &GraphBatchResult{Errors: []string{}}
	if h == nil || h.db == nil {
		return nil, fmt.Errorf("graph batch handler is not configured")
	}
	stdlog.Printf("[KGS][GraphBatchHandler] UpsertGraph start app_id=%s tenant_id=%s entities=%d edges=%d overlay=%t conflict_policy=%s",
		appID, tenantID, len(req.Entities), len(req.Edges), req.OverlayID != nil, req.ConflictPolicy)

	if req.OverlayID != nil {
		if h.overlay == nil {
			return nil, fmt.Errorf("overlay store is not configured")
		}
		namespace := biz.ComputeNamespace(appID, tenantID)
		for _, e := range req.Entities {
			if _, err := h.overlay.AddEntityDelta(ctx, *req.OverlayID, namespace, e.Label, cloneProperties(e.Properties)); err != nil {
				stdlog.Printf("[KGS][GraphBatchHandler] overlay entity failed app_id=%s tenant_id=%s overlay_id=%s label=%s err=%v",
					appID, tenantID, *req.OverlayID, e.Label, err)
				return nil, err
			}
			stdlog.Printf("[KGS][GraphBatchHandler] overlay entity queued app_id=%s tenant_id=%s overlay_id=%s label=%s", appID, tenantID, *req.OverlayID, e.Label)
			res.EntitiesCreated++
		}
		for _, e := range req.Edges {
			if _, err := h.overlay.AddEdgeDelta(ctx, *req.OverlayID, namespace, e.RelationType, e.FromEntityID, e.ToEntityID, cloneProperties(e.Properties)); err != nil {
				stdlog.Printf("[KGS][GraphBatchHandler] overlay edge failed app_id=%s tenant_id=%s overlay_id=%s relation=%s from=%s to=%s err=%v",
					appID, tenantID, *req.OverlayID, e.RelationType, e.FromEntityID, e.ToEntityID, err)
				return nil, err
			}
			stdlog.Printf("[KGS][GraphBatchHandler] overlay edge queued app_id=%s tenant_id=%s overlay_id=%s relation=%s from=%s to=%s",
				appID, tenantID, *req.OverlayID, e.RelationType, e.FromEntityID, e.ToEntityID)
			res.EdgesCreated++
		}
		stdlog.Printf("[KGS][GraphBatchHandler] UpsertGraph overlay done app_id=%s tenant_id=%s entities_created=%d edges_created=%d duration=%s",
			appID, tenantID, res.EntitiesCreated, res.EdgesCreated, time.Since(started))
		return res, nil
	}

	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, e := range req.Entities {
			kgEntity := toKGEntity(e, appID, tenantID)
			op, err := data.UpsertEntityTx(tx, kgEntity)
			if err != nil {
				stdlog.Printf("[KGS][GraphBatchHandler] postgres upsert entity failed app_id=%s tenant_id=%s entity_id=%s label=%s err=%v",
					appID, tenantID, kgEntity.EntityID, kgEntity.EntityType, err)
				if isEntityConflict(err) {
					res.Conflicted++
					if strings.EqualFold(req.ConflictPolicy, ConflictPolicySkip) {
						res.EntitiesSkipped++
						continue
					}
				}
				return err
			}
			switch op {
			case data.UpsertOpCreated:
				res.EntitiesCreated++
			case data.UpsertOpUpdated:
				res.EntitiesUpdated++
			}
			stdlog.Printf("[KGS][GraphBatchHandler] postgres upsert entity done app_id=%s tenant_id=%s entity_id=%s label=%s op=%v",
				appID, tenantID, kgEntity.EntityID, kgEntity.EntityType, op)

			rec, err := toOutboxRecord(data.OutboxOpUpsertEntity, kgEntity)
			if err != nil {
				stdlog.Printf("[KGS][GraphBatchHandler] build outbox entity failed app_id=%s tenant_id=%s entity_id=%s err=%v",
					appID, tenantID, kgEntity.EntityID, err)
				return err
			}
			if err := data.EnqueueOutboxTx(tx, rec); err != nil {
				stdlog.Printf("[KGS][GraphBatchHandler] enqueue outbox entity failed app_id=%s tenant_id=%s entity_id=%s op=%s err=%v",
					appID, tenantID, kgEntity.EntityID, rec.Op, err)
				return err
			}
			stdlog.Printf("[KGS][GraphBatchHandler] enqueue outbox entity done app_id=%s tenant_id=%s entity_id=%s op=%s",
				appID, tenantID, kgEntity.EntityID, rec.Op)
		}

		for _, e := range req.Edges {
			kgEdge := toKGEdge(e, appID, tenantID)
			op, err := data.UpsertEdgeTx(tx, kgEdge)
			if err != nil {
				stdlog.Printf("[KGS][GraphBatchHandler] postgres upsert edge failed app_id=%s tenant_id=%s edge_id=%s relation=%s from=%s to=%s err=%v",
					appID, tenantID, kgEdge.EdgeID, kgEdge.RelationType, kgEdge.FromEntityID, kgEdge.ToEntityID, err)
				return fmt.Errorf("edge %s: %w", kgEdge.EdgeID, err)
			}
			if op == data.UpsertOpCreated {
				res.EdgesCreated++
			} else {
				res.EdgesSkipped++
			}
			stdlog.Printf("[KGS][GraphBatchHandler] postgres upsert edge done app_id=%s tenant_id=%s edge_id=%s relation=%s from=%s to=%s op=%v",
				appID, tenantID, kgEdge.EdgeID, kgEdge.RelationType, kgEdge.FromEntityID, kgEdge.ToEntityID, op)
			rec, err := toEdgeOutboxRecord(data.OutboxOpUpsertEdge, kgEdge)
			if err != nil {
				stdlog.Printf("[KGS][GraphBatchHandler] build outbox edge failed app_id=%s tenant_id=%s edge_id=%s err=%v",
					appID, tenantID, kgEdge.EdgeID, err)
				return err
			}
			if err := data.EnqueueOutboxTx(tx, rec); err != nil {
				stdlog.Printf("[KGS][GraphBatchHandler] enqueue outbox edge failed app_id=%s tenant_id=%s edge_id=%s op=%s err=%v",
					appID, tenantID, kgEdge.EdgeID, rec.Op, err)
				return err
			}
			stdlog.Printf("[KGS][GraphBatchHandler] enqueue outbox edge done app_id=%s tenant_id=%s edge_id=%s op=%s",
				appID, tenantID, kgEdge.EdgeID, rec.Op)
		}
		return nil
	})
	if err != nil {
		stdlog.Printf("[KGS][GraphBatchHandler] UpsertGraph failed app_id=%s tenant_id=%s err=%v duration=%s",
			appID, tenantID, err, time.Since(started))
		res.Errors = append(res.Errors, err.Error())
		return res, err
	}
	stdlog.Printf("[KGS][GraphBatchHandler] UpsertGraph done app_id=%s tenant_id=%s entities_created=%d entities_updated=%d entities_skipped=%d edges_created=%d edges_skipped=%d conflicted=%d duration=%s",
		appID, tenantID, res.EntitiesCreated, res.EntitiesUpdated, res.EntitiesSkipped, res.EdgesCreated, res.EdgesSkipped, res.Conflicted, time.Since(started))
	return res, nil
}

func toKGEdge(e Edge, appID, tenantID string) data.KGEdge {
	edgeID := strings.TrimSpace(e.EdgeID)
	if edgeID == "" {
		edgeID = uuid.NewString()
	}
	return data.KGEdge{
		EdgeID:       edgeID,
		AppID:        appID,
		TenantID:     tenantID,
		FromEntityID: e.FromEntityID,
		ToEntityID:   e.ToEntityID,
		RelationType: e.RelationType,
		Properties:   data.JSONMap(cloneProperties(e.Properties)),
		Confidence:   defaultFloat(e.Confidence, 1.0),
		VersionID:    e.VersionID,
	}
}

func toEdgeOutboxRecord(op string, edge data.KGEdge) (data.KGSyncOutbox, error) {
	payload, err := structToJSONMap(edge)
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

func isEntityConflict(err error) bool {
	return errors.Is(err, data.ErrAlreadyExists) || errors.Is(err, data.ErrVersionConflict) || errors.Is(err, data.ErrNameConflict)
}

func defaultFloat(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}
