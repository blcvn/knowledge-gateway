package biz

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/lock"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const lockReleaseTimeout = 2 * time.Second

const (
	defaultNodeLockTTL = 30 * time.Second
	nodeLockTTLEnvKey  = "KGS_LOCK_TTL"
)

// GraphRepo defines the graph data persistence interface
type GraphRepo interface {
	CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error)
	GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error)
	CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error)
	ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error)
	GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*FullGraphResult, error)
	DeleteNode(ctx context.Context, appID, tenantID, nodeID string) (edgesRemoved int, err error)
	DeleteEdge(ctx context.Context, appID, tenantID, edgeID string) error
	BatchDeleteNodes(ctx context.Context, appID, tenantID string, nodeIDs []string) (deleted, edgesRemoved int, err error)
}

type GraphUsecase struct {
	repo        GraphRepo
	writeRepo   GraphWriteRepo
	reader      EntityReader
	validator   *OntologyValidator
	planner     *QueryPlanner
	opa         *OPAClient
	redisCli    *redis.Client
	lockMgr     lock.LockManager
	nodeLockTTL time.Duration
	overlay     OverlayDeltaWriter
	log         *log.Helper
}

type OverlayDeltaWriter interface {
	AddEntityDelta(ctx context.Context, overlayID, namespace, label string, properties map[string]any) (map[string]any, error)
	AddEdgeDelta(ctx context.Context, overlayID, namespace, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error)
	DeleteEntityDelta(ctx context.Context, overlayID, nodeID string) error
	DeleteEdgeDelta(ctx context.Context, overlayID, edgeID string) error
}

func NewGraphUsecase(
	repo GraphRepo,
	planner *QueryPlanner,
	opa *OPAClient,
	validator *OntologyValidator,
	redisCli *redis.Client,
	lockMgr lock.LockManager,
	overlay OverlayDeltaWriter,
	logger log.Logger,
) *GraphUsecase {
	return NewGraphUsecaseWithStorage(repo, nil, nil, planner, opa, validator, redisCli, lockMgr, overlay, logger)
}

func NewGraphUsecaseWithStorage(
	repo GraphRepo,
	writeRepo GraphWriteRepo,
	reader EntityReader,
	planner *QueryPlanner,
	opa *OPAClient,
	validator *OntologyValidator,
	redisCli *redis.Client,
	lockMgr lock.LockManager,
	overlay OverlayDeltaWriter,
	logger log.Logger,
) *GraphUsecase {
	return &GraphUsecase{
		repo:        repo,
		writeRepo:   writeRepo,
		reader:      reader,
		planner:     planner,
		opa:         opa,
		validator:   validator,
		redisCli:    redisCli,
		lockMgr:     lockMgr,
		nodeLockTTL: lockTTLFromEnv(),
		overlay:     overlay,
		log:         log.NewHelper(logger),
	}
}

func (uc *GraphUsecase) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
	started := time.Now()
	if properties == nil {
		properties = map[string]any{}
	}
	if _, ok := properties["id"].(string); !ok {
		properties["id"] = uuid.NewString()
	}
	nodeID := fmt.Sprint(properties["id"])
	uc.log.Infof("[KGS][GraphUsecase] CreateNode start app_id=%s tenant_id=%s label=%s node_id=%s props_keys=%d",
		appID, tenantID, label, nodeID, len(properties))
	if overlayID := extractOverlayID(properties); overlayID != "" {
		if uc.overlay == nil {
			err := ErrNotConfigured("overlay writer is not configured", map[string]string{"component": "overlay_writer"})
			uc.log.Errorf("[KGS][GraphUsecase] CreateNode overlay not configured app_id=%s tenant_id=%s label=%s node_id=%s err=%v",
				appID, tenantID, label, nodeID, err)
			observability.ObserveEntityWrite("create_node_overlay", err)
			return nil, err
		}
		namespace := ComputeNamespace(appID, tenantID)
		result, err := uc.overlay.AddEntityDelta(ctx, overlayID, namespace, label, properties)
		if err != nil {
			uc.log.Errorf("[KGS][GraphUsecase] CreateNode overlay failed app_id=%s tenant_id=%s overlay_id=%s label=%s node_id=%s err=%v",
				appID, tenantID, overlayID, label, nodeID, err)
		} else {
			uc.log.Infof("[KGS][GraphUsecase] CreateNode overlay done app_id=%s tenant_id=%s overlay_id=%s label=%s node_id=%s duration=%s",
				appID, tenantID, overlayID, label, nodeID, time.Since(started))
		}
		observability.ObserveEntityWrite("create_node_overlay", err)
		return result, err
	}

	lockCtx := lock.WithOwnerID(ctx, "graph-write-"+uuid.NewString())
	lockToken, err := uc.acquireNodeLock(lockCtx, appID, tenantID, nodeID)
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] CreateNode acquire lock failed app_id=%s tenant_id=%s node_id=%s err=%v",
			appID, tenantID, nodeID, err)
		return nil, err
	}
	defer uc.releaseLock(lockCtx, lockToken)

	// 1. OPA Policy Check
	allowed, err := uc.opa.EvaluatePolicy(lockCtx, appID, "CREATE_NODE", label)
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] CreateNode OPA evaluation failed app_id=%s tenant_id=%s label=%s node_id=%s err=%v",
			appID, tenantID, label, nodeID, err)
		observability.ObserveEntityWrite("create_node", err)
		return nil, err
	}
	if !allowed {
		err := ErrForbiddenWithMetadata("access denied by OPA policy", map[string]string{
			"action": "CREATE_NODE",
			"label":  label,
		})
		uc.log.Warnf("[KGS][GraphUsecase] CreateNode denied by OPA app_id=%s tenant_id=%s label=%s node_id=%s", appID, tenantID, label, nodeID)
		observability.ObserveEntityWrite("create_node", err)
		return nil, err
	}

	if err := uc.validator.ValidateEntity(lockCtx, appID, label, properties); err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] CreateNode ontology validation failed app_id=%s tenant_id=%s label=%s node_id=%s err=%v",
			appID, tenantID, label, nodeID, err)
		observability.ObserveEntityWrite("create_node", err)
		return nil, err
	}

	// 2. Data Persistence
	var result map[string]any
	if uc.writeRepo == nil {
		result, err = uc.repo.CreateNode(lockCtx, appID, tenantID, label, properties)
		if err != nil {
			uc.log.Errorf("[KGS][GraphUsecase] CreateNode repo write failed app_id=%s tenant_id=%s label=%s node_id=%s err=%v",
				appID, tenantID, label, nodeID, err)
			observability.ObserveEntityWrite("create_node", err)
			return nil, err
		}
		uc.log.Infof("[KGS][GraphUsecase] CreateNode repo write done app_id=%s tenant_id=%s label=%s node_id=%s duration=%s",
			appID, tenantID, label, nodeID, time.Since(started))
	} else {
		entity := mapNodeToWriteEntity(appID, tenantID, label, properties)
		err = uc.writeRepo.WithTx(lockCtx, func(txRepo GraphWriteRepo) error {
			op, err := txRepo.UpsertEntity(lockCtx, entity)
			if err != nil {
				uc.log.Errorf("[KGS][GraphUsecase] CreateNode postgres upsert failed app_id=%s tenant_id=%s label=%s node_id=%s err=%v",
					appID, tenantID, label, entity.EntityID, err)
				return err
			}
			uc.log.Infof("[KGS][GraphUsecase] CreateNode postgres upsert done app_id=%s tenant_id=%s label=%s node_id=%s op=%s",
				appID, tenantID, label, entity.EntityID, op)
			rec := buildEntityOutboxRecord(OutboxOpUpsertEntity, entity)
			if err := txRepo.EnqueueOutbox(lockCtx, rec); err != nil {
				uc.log.Errorf("[KGS][GraphUsecase] CreateNode enqueue outbox failed app_id=%s tenant_id=%s node_id=%s op=%s err=%v",
					appID, tenantID, entity.EntityID, rec.Op, err)
				return err
			}
			uc.log.Infof("[KGS][GraphUsecase] CreateNode enqueue outbox done app_id=%s tenant_id=%s node_id=%s op=%s",
				appID, tenantID, entity.EntityID, rec.Op)
			return nil
		})
		if err != nil {
			uc.log.Errorf("[KGS][GraphUsecase] CreateNode transactional write failed app_id=%s tenant_id=%s label=%s node_id=%s err=%v",
				appID, tenantID, label, entity.EntityID, err)
			observability.ObserveEntityWrite("create_node", err)
			return nil, err
		}
		result = cloneProperties(properties)
		result["id"] = entity.EntityID
		result["label"] = label
		result["entity_type"] = label
	}

	// 3. Trigger Event
	uc.redisCli.XAdd(lockCtx, &redis.XAddArgs{
		Stream: "kgs:events:nodes",
		Values: map[string]interface{}{
			"event_type": "node.created",
			"app_id":     appID,
			"tenant_id":  tenantID,
			"label":      label,
		},
	})
	uc.log.Infof("[KGS][GraphUsecase] CreateNode done app_id=%s tenant_id=%s label=%s node_id=%s duration=%s",
		appID, tenantID, label, nodeID, time.Since(started))

	observability.ObserveEntityWrite("create_node", nil)
	return result, nil
}

func (uc *GraphUsecase) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	if uc.reader != nil {
		return uc.reader.GetEntity(ctx, appID, tenantID, nodeID)
	}
	return uc.repo.GetNode(ctx, appID, tenantID, nodeID)
}

func (uc *GraphUsecase) EnrichWithFreshVersions(ctx context.Context, appID, tenantID string, entities []map[string]any) ([]map[string]any, error) {
	if uc.reader == nil {
		return entities, nil
	}
	return uc.reader.EnrichWithFreshVersions(ctx, appID, tenantID, entities)
}

func (uc *GraphUsecase) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	started := time.Now()
	uc.log.Infof("[KGS][GraphUsecase] CreateEdge start app_id=%s tenant_id=%s relation=%s source=%s target=%s props_keys=%d",
		appID, tenantID, relationType, sourceNodeID, targetNodeID, len(properties))
	if overlayID := extractOverlayID(properties); overlayID != "" {
		if uc.overlay == nil {
			err := ErrNotConfigured("overlay writer is not configured", map[string]string{"component": "overlay_writer"})
			uc.log.Errorf("[KGS][GraphUsecase] CreateEdge overlay not configured app_id=%s tenant_id=%s relation=%s source=%s target=%s err=%v",
				appID, tenantID, relationType, sourceNodeID, targetNodeID, err)
			observability.ObserveEntityWrite("create_edge_overlay", err)
			return nil, err
		}
		namespace := ComputeNamespace(appID, tenantID)
		result, err := uc.overlay.AddEdgeDelta(ctx, overlayID, namespace, relationType, sourceNodeID, targetNodeID, properties)
		if err != nil {
			uc.log.Errorf("[KGS][GraphUsecase] CreateEdge overlay failed app_id=%s tenant_id=%s overlay_id=%s relation=%s source=%s target=%s err=%v",
				appID, tenantID, overlayID, relationType, sourceNodeID, targetNodeID, err)
		} else {
			uc.log.Infof("[KGS][GraphUsecase] CreateEdge overlay done app_id=%s tenant_id=%s overlay_id=%s relation=%s source=%s target=%s duration=%s",
				appID, tenantID, overlayID, relationType, sourceNodeID, targetNodeID, time.Since(started))
		}
		observability.ObserveEntityWrite("create_edge_overlay", err)
		return result, err
	}

	lockCtx := lock.WithOwnerID(ctx, "graph-write-"+uuid.NewString())

	firstNodeID := sourceNodeID
	secondNodeID := targetNodeID
	if firstNodeID != secondNodeID && strings.Compare(firstNodeID, secondNodeID) > 0 {
		firstNodeID, secondNodeID = secondNodeID, firstNodeID
	}

	firstToken, err := uc.acquireNodeLock(lockCtx, appID, tenantID, firstNodeID)
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] CreateEdge acquire first lock failed app_id=%s tenant_id=%s node=%s err=%v",
			appID, tenantID, firstNodeID, err)
		return nil, err
	}
	defer uc.releaseLock(lockCtx, firstToken)

	if secondNodeID != firstNodeID {
		secondToken, acquireErr := uc.acquireNodeLock(lockCtx, appID, tenantID, secondNodeID)
		if acquireErr != nil {
			uc.log.Errorf("[KGS][GraphUsecase] CreateEdge acquire second lock failed app_id=%s tenant_id=%s node=%s err=%v",
				appID, tenantID, secondNodeID, acquireErr)
			return nil, acquireErr
		}
		defer uc.releaseLock(lockCtx, secondToken)
	}

	if err := uc.validator.ValidateEdge(lockCtx, appID, tenantID, relationType, sourceNodeID, targetNodeID); err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] CreateEdge validation failed app_id=%s tenant_id=%s relation=%s source=%s target=%s err=%v",
			appID, tenantID, relationType, sourceNodeID, targetNodeID, err)
		observability.ObserveEntityWrite("create_edge", err)
		return nil, err
	}

	if properties == nil {
		properties = map[string]any{}
	}
	var result map[string]any
	if uc.writeRepo == nil {
		result, err = uc.repo.CreateEdge(lockCtx, appID, tenantID, relationType, sourceNodeID, targetNodeID, properties)
	} else {
		edge := mapEdgeToWriteEdge(appID, tenantID, relationType, sourceNodeID, targetNodeID, properties)
		err = uc.writeRepo.WithTx(lockCtx, func(txRepo GraphWriteRepo) error {
			op, err := txRepo.UpsertEdge(lockCtx, edge)
			if err != nil {
				uc.log.Errorf("[KGS][GraphUsecase] CreateEdge postgres upsert failed app_id=%s tenant_id=%s relation=%s edge_id=%s source=%s target=%s err=%v",
					appID, tenantID, relationType, edge.EdgeID, edge.FromEntityID, edge.ToEntityID, err)
				return err
			}
			uc.log.Infof("[KGS][GraphUsecase] CreateEdge postgres upsert done app_id=%s tenant_id=%s relation=%s edge_id=%s source=%s target=%s op=%s",
				appID, tenantID, relationType, edge.EdgeID, edge.FromEntityID, edge.ToEntityID, op)
			rec := buildEdgeOutboxRecord(OutboxOpUpsertEdge, edge)
			if err := txRepo.EnqueueOutbox(lockCtx, rec); err != nil {
				uc.log.Errorf("[KGS][GraphUsecase] CreateEdge enqueue outbox failed app_id=%s tenant_id=%s edge_id=%s op=%s err=%v",
					appID, tenantID, edge.EdgeID, rec.Op, err)
				return err
			}
			uc.log.Infof("[KGS][GraphUsecase] CreateEdge enqueue outbox done app_id=%s tenant_id=%s edge_id=%s op=%s",
				appID, tenantID, edge.EdgeID, rec.Op)
			return nil
		})
		if err == nil {
			result = cloneProperties(properties)
			result["id"] = edge.EdgeID
			result["relation_type"] = edge.RelationType
			result["source_node_id"] = edge.FromEntityID
			result["target_node_id"] = edge.ToEntityID
		}
	}
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] CreateEdge failed app_id=%s tenant_id=%s relation=%s source=%s target=%s err=%v",
			appID, tenantID, relationType, sourceNodeID, targetNodeID, err)
	} else {
		uc.log.Infof("[KGS][GraphUsecase] CreateEdge done app_id=%s tenant_id=%s relation=%s source=%s target=%s edge_id=%s duration=%s",
			appID, tenantID, relationType, sourceNodeID, targetNodeID, fmt.Sprint(result["id"]), time.Since(started))
	}
	observability.ObserveEntityWrite("create_edge", err)
	return result, err
}

func (uc *GraphUsecase) GetContext(ctx context.Context, appID, tenantID string, nodeID string, depth int, direction string) (map[string]any, error) {
	if err := ValidateDepth(depth); err != nil {
		return nil, err
	}
	if depth > 3 {
		return uc.executeBatchedTraversal(ctx, "context", appID, tenantID, nodeID, depth, direction)
	}
	cypher := uc.planner.BuildContextQuery("", direction)
	params := map[string]any{
		"app_id":    appID,
		"tenant_id": tenantID,
		"node_id":   nodeID,
	}
	return uc.repo.ExecuteQuery(ctx, cypher, params)
}

func (uc *GraphUsecase) GetImpact(ctx context.Context, appID, tenantID string, nodeID string, maxDepth int) (map[string]any, error) {
	if err := ValidateDepth(maxDepth); err != nil {
		return nil, err
	}
	if maxDepth > 3 {
		return uc.executeBatchedTraversal(ctx, "impact", appID, tenantID, nodeID, maxDepth, "")
	}
	cypher := uc.planner.BuildImpactQuery("", maxDepth)
	params := map[string]any{
		"app_id":    appID,
		"tenant_id": tenantID,
		"node_id":   nodeID,
	}
	return uc.repo.ExecuteQuery(ctx, cypher, params)
}

func (uc *GraphUsecase) GetCoverage(ctx context.Context, appID, tenantID string, nodeID string, maxDepth int) (map[string]any, error) {
	if err := ValidateDepth(maxDepth); err != nil {
		return nil, err
	}
	if maxDepth > 3 {
		return uc.executeBatchedTraversal(ctx, "coverage", appID, tenantID, nodeID, maxDepth, "")
	}
	cypher := uc.planner.BuildCoverageQuery("", maxDepth)
	params := map[string]any{
		"app_id":    appID,
		"tenant_id": tenantID,
		"node_id":   nodeID,
	}
	return uc.repo.ExecuteQuery(ctx, cypher, params)
}

func (uc *GraphUsecase) GetSubgraph(ctx context.Context, appID, tenantID string, nodeIDs []string) (map[string]any, error) {
	// Guardrail for maximum bulk queries
	if err := ValidateNodeCount(len(nodeIDs)); err != nil {
		return nil, err
	}
	cypher := uc.planner.BuildSubgraphQuery()
	params := map[string]any{
		"app_id":    appID,
		"tenant_id": tenantID,
		"node_ids":  nodeIDs,
	}
	return uc.repo.ExecuteQuery(ctx, cypher, params)
}

func (uc *GraphUsecase) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*FullGraphResult, error) {
	if limit <= 0 {
		limit = MaxAllowedNodes
	}
	if limit > MaxAllowedNodes {
		limit = MaxAllowedNodes
	}
	if offset < 0 {
		offset = 0
	}
	return uc.repo.GetFullGraph(ctx, appID, tenantID, limit, offset)
}

func (uc *GraphUsecase) DeleteNode(ctx context.Context, appID, tenantID, nodeID string) (int, error) {
	started := time.Now()
	uc.log.Infof("[KGS][GraphUsecase] DeleteNode start app_id=%s tenant_id=%s node_id=%s", appID, tenantID, nodeID)
	if overlayID := overlayIDFromContext(ctx); overlayID != "" {
		if uc.overlay == nil {
			err := ErrNotConfigured("overlay writer is not configured", map[string]string{"component": "overlay_writer"})
			uc.log.Errorf("[KGS][GraphUsecase] DeleteNode overlay not configured app_id=%s tenant_id=%s node_id=%s err=%v", appID, tenantID, nodeID, err)
			observability.ObserveEntityWrite("delete_node_overlay", err)
			return 0, err
		}
		err := uc.overlay.DeleteEntityDelta(ctx, overlayID, nodeID)
		if err != nil {
			uc.log.Errorf("[KGS][GraphUsecase] DeleteNode overlay failed app_id=%s tenant_id=%s overlay_id=%s node_id=%s err=%v",
				appID, tenantID, overlayID, nodeID, err)
		} else {
			uc.log.Infof("[KGS][GraphUsecase] DeleteNode overlay done app_id=%s tenant_id=%s overlay_id=%s node_id=%s duration=%s",
				appID, tenantID, overlayID, nodeID, time.Since(started))
		}
		observability.ObserveEntityWrite("delete_node_overlay", err)
		return 0, err
	}

	lockCtx := lock.WithOwnerID(ctx, "graph-write-"+uuid.NewString())
	lockToken, err := uc.acquireNodeLock(lockCtx, appID, tenantID, nodeID)
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] DeleteNode acquire lock failed app_id=%s tenant_id=%s node_id=%s err=%v", appID, tenantID, nodeID, err)
		observability.ObserveEntityWrite("delete_node", err)
		return 0, err
	}
	defer uc.releaseLock(lockCtx, lockToken)

	var edgesRemoved int
	if uc.writeRepo == nil {
		edgesRemoved, err = uc.repo.DeleteNode(lockCtx, appID, tenantID, nodeID)
	} else {
		err = uc.writeRepo.WithTx(lockCtx, func(txRepo GraphWriteRepo) error {
			if err := txRepo.SoftDeleteEntity(lockCtx, nodeID, tenantID); err != nil {
				uc.log.Errorf("[KGS][GraphUsecase] DeleteNode postgres soft delete failed app_id=%s tenant_id=%s node_id=%s err=%v",
					appID, tenantID, nodeID, err)
				return err
			}
			uc.log.Infof("[KGS][GraphUsecase] DeleteNode postgres soft delete done app_id=%s tenant_id=%s node_id=%s", appID, tenantID, nodeID)
			rec := OutboxRecord{
				Op:       OutboxOpDeleteEntity,
				EntityID: ptr(nodeID),
				TenantID: tenantID,
				AppID:    appID,
				Payload: map[string]any{
					"entity_id": nodeID,
					"tenant_id": tenantID,
					"app_id":    appID,
				},
			}
			if err := txRepo.EnqueueOutbox(lockCtx, rec); err != nil {
				uc.log.Errorf("[KGS][GraphUsecase] DeleteNode enqueue outbox failed app_id=%s tenant_id=%s node_id=%s op=%s err=%v",
					appID, tenantID, nodeID, rec.Op, err)
				return err
			}
			uc.log.Infof("[KGS][GraphUsecase] DeleteNode enqueue outbox done app_id=%s tenant_id=%s node_id=%s op=%s",
				appID, tenantID, nodeID, rec.Op)
			return nil
		})
		edgesRemoved = 0
	}
	observability.ObserveEntityWrite("delete_node", err)
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] DeleteNode failed app_id=%s tenant_id=%s node_id=%s err=%v", appID, tenantID, nodeID, err)
		return 0, err
	}

	if uc.redisCli != nil {
		_ = uc.redisCli.XAdd(lockCtx, &redis.XAddArgs{
			Stream: fmt.Sprintf("kgs:events:%s:%s", appID, tenantID),
			Values: map[string]any{
				"event":         "node.deleted",
				"node_id":       nodeID,
				"edges_removed": edgesRemoved,
			},
		}).Err()
	}
	uc.log.Infof("[KGS][GraphUsecase] DeleteNode done app_id=%s tenant_id=%s node_id=%s edges_removed=%d duration=%s",
		appID, tenantID, nodeID, edgesRemoved, time.Since(started))
	return edgesRemoved, nil
}

func (uc *GraphUsecase) DeleteEdge(ctx context.Context, appID, tenantID, edgeID string) error {
	started := time.Now()
	uc.log.Infof("[KGS][GraphUsecase] DeleteEdge start app_id=%s tenant_id=%s edge_id=%s", appID, tenantID, edgeID)
	if overlayID := overlayIDFromContext(ctx); overlayID != "" {
		if uc.overlay == nil {
			err := ErrNotConfigured("overlay writer is not configured", map[string]string{"component": "overlay_writer"})
			uc.log.Errorf("[KGS][GraphUsecase] DeleteEdge overlay not configured app_id=%s tenant_id=%s edge_id=%s err=%v", appID, tenantID, edgeID, err)
			observability.ObserveEntityWrite("delete_edge_overlay", err)
			return err
		}
		err := uc.overlay.DeleteEdgeDelta(ctx, overlayID, edgeID)
		if err != nil {
			uc.log.Errorf("[KGS][GraphUsecase] DeleteEdge overlay failed app_id=%s tenant_id=%s overlay_id=%s edge_id=%s err=%v",
				appID, tenantID, overlayID, edgeID, err)
		} else {
			uc.log.Infof("[KGS][GraphUsecase] DeleteEdge overlay done app_id=%s tenant_id=%s overlay_id=%s edge_id=%s duration=%s",
				appID, tenantID, overlayID, edgeID, time.Since(started))
		}
		observability.ObserveEntityWrite("delete_edge_overlay", err)
		return err
	}

	lockCtx := lock.WithOwnerID(ctx, "graph-write-"+uuid.NewString())
	lockToken, err := uc.acquireNodeLock(lockCtx, appID, tenantID, edgeID)
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] DeleteEdge acquire lock failed app_id=%s tenant_id=%s edge_id=%s err=%v", appID, tenantID, edgeID, err)
		observability.ObserveEntityWrite("delete_edge", err)
		return err
	}
	defer uc.releaseLock(lockCtx, lockToken)

	if uc.writeRepo == nil {
		err = uc.repo.DeleteEdge(lockCtx, appID, tenantID, edgeID)
	} else {
		err = uc.writeRepo.WithTx(lockCtx, func(txRepo GraphWriteRepo) error {
			if err := txRepo.SoftDeleteEdge(lockCtx, edgeID, tenantID); err != nil {
				uc.log.Errorf("[KGS][GraphUsecase] DeleteEdge postgres soft delete failed app_id=%s tenant_id=%s edge_id=%s err=%v",
					appID, tenantID, edgeID, err)
				return err
			}
			uc.log.Infof("[KGS][GraphUsecase] DeleteEdge postgres soft delete done app_id=%s tenant_id=%s edge_id=%s", appID, tenantID, edgeID)
			rec := OutboxRecord{
				Op:       OutboxOpDeleteEdge,
				EdgeID:   ptr(edgeID),
				TenantID: tenantID,
				AppID:    appID,
				Payload: map[string]any{
					"edge_id":   edgeID,
					"tenant_id": tenantID,
					"app_id":    appID,
				},
			}
			if err := txRepo.EnqueueOutbox(lockCtx, rec); err != nil {
				uc.log.Errorf("[KGS][GraphUsecase] DeleteEdge enqueue outbox failed app_id=%s tenant_id=%s edge_id=%s op=%s err=%v",
					appID, tenantID, edgeID, rec.Op, err)
				return err
			}
			uc.log.Infof("[KGS][GraphUsecase] DeleteEdge enqueue outbox done app_id=%s tenant_id=%s edge_id=%s op=%s",
				appID, tenantID, edgeID, rec.Op)
			return nil
		})
	}
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] DeleteEdge failed app_id=%s tenant_id=%s edge_id=%s err=%v", appID, tenantID, edgeID, err)
	} else {
		uc.log.Infof("[KGS][GraphUsecase] DeleteEdge done app_id=%s tenant_id=%s edge_id=%s duration=%s",
			appID, tenantID, edgeID, time.Since(started))
	}
	observability.ObserveEntityWrite("delete_edge", err)
	return err
}

func (uc *GraphUsecase) BatchDeleteNodes(ctx context.Context, appID, tenantID string, nodeIDs []string) (int, int, error) {
	started := time.Now()
	if len(nodeIDs) == 0 {
		return 0, 0, nil
	}
	uc.log.Infof("[KGS][GraphUsecase] BatchDeleteNodes start app_id=%s tenant_id=%s nodes=%d", appID, tenantID, len(nodeIDs))

	lockCtx := lock.WithOwnerID(ctx, "graph-write-"+uuid.NewString())
	if uc.lockMgr != nil {
		namespace := ComputeNamespace(appID, tenantID)
		lockToken, err := uc.lockMgr.AcquireNamespaceLock(lockCtx, namespace, uc.nodeLockTTL)
		if err != nil {
			uc.log.Errorf("[KGS][GraphUsecase] BatchDeleteNodes acquire namespace lock failed app_id=%s tenant_id=%s namespace=%s err=%v",
				appID, tenantID, namespace, err)
			observability.ObserveEntityWrite("batch_delete_nodes", err)
			return 0, 0, err
		}
		defer uc.releaseLock(lockCtx, lockToken)
	}

	var deleted int
	var edgesRemoved int
	var err error
	if uc.writeRepo == nil {
		deleted, edgesRemoved, err = uc.repo.BatchDeleteNodes(lockCtx, appID, tenantID, nodeIDs)
	} else {
		err = uc.writeRepo.WithTx(lockCtx, func(txRepo GraphWriteRepo) error {
			for _, nodeID := range nodeIDs {
				if err := txRepo.SoftDeleteEntity(lockCtx, nodeID, tenantID); err != nil {
					uc.log.Errorf("[KGS][GraphUsecase] BatchDeleteNodes postgres soft delete failed app_id=%s tenant_id=%s node_id=%s err=%v",
						appID, tenantID, nodeID, err)
					return err
				}
				uc.log.Infof("[KGS][GraphUsecase] BatchDeleteNodes postgres soft delete done app_id=%s tenant_id=%s node_id=%s",
					appID, tenantID, nodeID)
				rec := OutboxRecord{
					Op:       OutboxOpDeleteEntity,
					EntityID: ptr(nodeID),
					TenantID: tenantID,
					AppID:    appID,
					Payload: map[string]any{
						"entity_id": nodeID,
						"tenant_id": tenantID,
						"app_id":    appID,
					},
				}
				if err := txRepo.EnqueueOutbox(lockCtx, rec); err != nil {
					uc.log.Errorf("[KGS][GraphUsecase] BatchDeleteNodes enqueue outbox failed app_id=%s tenant_id=%s node_id=%s op=%s err=%v",
						appID, tenantID, nodeID, rec.Op, err)
					return err
				}
				uc.log.Infof("[KGS][GraphUsecase] BatchDeleteNodes enqueue outbox done app_id=%s tenant_id=%s node_id=%s op=%s",
					appID, tenantID, nodeID, rec.Op)
			}
			return nil
		})
		if err == nil {
			deleted = len(nodeIDs)
			edgesRemoved = 0
		}
	}
	observability.ObserveEntityWrite("batch_delete_nodes", err)
	if err != nil {
		uc.log.Errorf("[KGS][GraphUsecase] BatchDeleteNodes failed app_id=%s tenant_id=%s nodes=%d err=%v",
			appID, tenantID, len(nodeIDs), err)
	} else {
		uc.log.Infof("[KGS][GraphUsecase] BatchDeleteNodes done app_id=%s tenant_id=%s nodes=%d deleted=%d edges_removed=%d duration=%s",
			appID, tenantID, len(nodeIDs), deleted, edgesRemoved, time.Since(started))
	}
	return deleted, edgesRemoved, err
}

func (uc *GraphUsecase) acquireNodeLock(ctx context.Context, appID, tenantID, nodeID string) (string, error) {
	if uc.lockMgr == nil || nodeID == "" {
		return "", nil
	}
	namespace := ComputeNamespace(appID, tenantID)
	ttl := uc.nodeLockTTL
	if ttl <= 0 {
		ttl = defaultNodeLockTTL
	}
	token, err := uc.lockMgr.AcquireNodeLock(ctx, namespace, nodeID, ttl)
	if err != nil {
		uc.log.Errorf("failed to acquire node lock namespace=%s node=%s: %v", namespace, nodeID, err)
		return "", err
	}
	return token, nil
}

func (uc *GraphUsecase) releaseLock(ctx context.Context, token string) {
	if uc.lockMgr == nil || token == "" {
		return
	}
	releaseCtx := context.WithoutCancel(ctx)
	releaseCtx, cancel := context.WithTimeout(releaseCtx, lockReleaseTimeout)
	defer cancel()

	if err := uc.lockMgr.Release(releaseCtx, token); err != nil {
		uc.log.Errorf("failed to release lock %s: %v", token, err)
	}
}

func (uc *GraphUsecase) executeBatchedTraversal(ctx context.Context, kind, appID, tenantID, nodeID string, depth int, direction string) (map[string]any, error) {
	queries := uc.planner.BuildBatchedTraversalQueries(kind, "", direction, depth, 3)
	merged := make([]map[string]any, 0)
	params := map[string]any{
		"app_id":    appID,
		"tenant_id": tenantID,
		"node_id":   nodeID,
	}
	for _, query := range queries {
		result, err := uc.repo.ExecuteQuery(ctx, query, params)
		if err != nil {
			return nil, err
		}
		if rows, ok := result["data"].([]map[string]any); ok {
			merged = append(merged, rows...)
		}
	}
	return map[string]any{"data": merged}, nil
}

func extractOverlayID(properties map[string]any) string {
	if properties == nil {
		return ""
	}
	raw, ok := properties["overlay_id"]
	if !ok || raw == nil {
		return ""
	}
	id, ok := raw.(string)
	if !ok || id == "" {
		return ""
	}
	delete(properties, "overlay_id")
	return id
}

func overlayIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	raw := ctx.Value("overlay_id")
	id, _ := raw.(string)
	return strings.TrimSpace(id)
}

func mapNodeToWriteEntity(appID, tenantID, label string, properties map[string]any) WriteEntity {
	props := cloneProperties(properties)
	entityID := toString(props["id"])
	if entityID == "" {
		entityID = uuid.NewString()
		props["id"] = entityID
	}
	name := firstNonEmpty(
		toString(props["name"]),
		toString(props["title"]),
		entityID,
	)

	return WriteEntity{
		EntityID:       entityID,
		AppID:          appID,
		TenantID:       tenantID,
		EntityType:     label,
		Name:           name,
		Properties:     props,
		Confidence:     toFloat64(props["confidence"], 1.0),
		SourceFile:     firstNonEmpty(toString(props["source_file"]), toString(props["sourceFile"])),
		ChunkID:        firstNonEmpty(toString(props["chunk_id"]), toString(props["chunkId"])),
		SkillID:        firstNonEmpty(toString(props["skill_id"]), toString(props["skillId"])),
		VersionID:      firstNonEmpty(toString(props["version_id"]), toString(props["versionId"])),
		ProvenanceType: firstNonEmpty(toString(props["provenance_type"]), toString(props["provenanceType"])),
		Domains:        toStringSlice(props["domains"]),
		Aliases:        toStringSlice(props["aliases"]),
		Version:        toInt(props["version"]),
	}
}

func mapEdgeToWriteEdge(appID, tenantID, relationType, sourceNodeID, targetNodeID string, properties map[string]any) WriteEdge {
	props := cloneProperties(properties)
	edgeID := toString(props["id"])
	if edgeID == "" {
		edgeID = uuid.NewString()
		props["id"] = edgeID
	}
	return WriteEdge{
		EdgeID:       edgeID,
		AppID:        appID,
		TenantID:     tenantID,
		FromEntityID: sourceNodeID,
		ToEntityID:   targetNodeID,
		RelationType: relationType,
		Properties:   props,
		Confidence:   toFloat64(props["confidence"], 1.0),
		VersionID:    firstNonEmpty(toString(props["version_id"]), toString(props["versionId"])),
	}
}

func buildEntityOutboxRecord(op string, entity WriteEntity) OutboxRecord {
	entityID := entity.EntityID
	payload := cloneProperties(entity.Properties)
	payload["id"] = entity.EntityID
	payload["label"] = entity.EntityType
	payload["entity_type"] = entity.EntityType
	payload["name"] = entity.Name
	payload["version"] = entity.Version
	payload["app_id"] = entity.AppID
	payload["tenant_id"] = entity.TenantID
	return OutboxRecord{
		Op:       op,
		EntityID: &entityID,
		TenantID: entity.TenantID,
		AppID:    entity.AppID,
		Payload:  payload,
	}
}

func buildEdgeOutboxRecord(op string, edge WriteEdge) OutboxRecord {
	edgeID := edge.EdgeID
	payload := cloneProperties(edge.Properties)
	payload["id"] = edge.EdgeID
	payload["edge_id"] = edge.EdgeID
	payload["from_entity_id"] = edge.FromEntityID
	payload["to_entity_id"] = edge.ToEntityID
	payload["relation_type"] = edge.RelationType
	payload["confidence"] = edge.Confidence
	payload["version_id"] = edge.VersionID
	payload["app_id"] = edge.AppID
	payload["tenant_id"] = edge.TenantID
	return OutboxRecord{
		Op:       op,
		EdgeID:   &edgeID,
		TenantID: edge.TenantID,
		AppID:    edge.AppID,
		Payload:  payload,
	}
}

func cloneProperties(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func ptr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	copied := v
	return &copied
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func toFloat64(v any, def float64) float64 {
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

func toStringSlice(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s := toString(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return []string{}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func lockTTLFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv(nodeLockTTLEnvKey))
	if raw == "" {
		return defaultNodeLockTTL
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return defaultNodeLockTTL
	}
	return parsed
}
