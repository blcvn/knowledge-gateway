package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kgs-platform/internal/lock"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// GraphRepo defines the graph data persistence interface
type GraphRepo interface {
	CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error)
	GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error)
	CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error)
	ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error)
}

type GraphUsecase struct {
	repo     GraphRepo
	ontology *OntologySyncManager
	planner  *QueryPlanner
	opa      *OPAClient
	redisCli *redis.Client
	lockMgr  lock.LockManager
	overlay  OverlayDeltaWriter
	log      *log.Helper
}

type OverlayDeltaWriter interface {
	AddEntityDelta(ctx context.Context, overlayID, namespace, label string, properties map[string]any) (map[string]any, error)
	AddEdgeDelta(ctx context.Context, overlayID, namespace, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error)
}

func NewGraphUsecase(
	repo GraphRepo,
	planner *QueryPlanner,
	opa *OPAClient,
	redisCli *redis.Client,
	lockMgr lock.LockManager,
	overlay OverlayDeltaWriter,
	logger log.Logger,
) *GraphUsecase {
	return &GraphUsecase{
		repo:     repo,
		planner:  planner,
		opa:      opa,
		redisCli: redisCli,
		lockMgr:  lockMgr,
		overlay:  overlay,
		log:      log.NewHelper(logger),
	}
}

func (uc *GraphUsecase) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
	if properties == nil {
		properties = map[string]any{}
	}
	if _, ok := properties["id"].(string); !ok {
		properties["id"] = uuid.NewString()
	}
	if overlayID := extractOverlayID(properties); overlayID != "" {
		if uc.overlay == nil {
			return nil, fmt.Errorf("overlay writer is not configured")
		}
		namespace := ComputeNamespace(appID, tenantID)
		return uc.overlay.AddEntityDelta(ctx, overlayID, namespace, label, properties)
	}

	lockCtx := lock.WithOwnerID(ctx, "graph-write-"+uuid.NewString())
	lockToken, err := uc.acquireNodeLock(lockCtx, appID, tenantID, properties["id"].(string))
	if err != nil {
		return nil, err
	}
	defer uc.releaseLock(lockCtx, lockToken)

	// 1. OPA Policy Check
	allowed, err := uc.opa.EvaluatePolicy(lockCtx, appID, "CREATE_NODE", label)
	if err != nil {
		uc.log.Errorf("OPA evaluation failed: %v", err)
		return nil, err
	}
	if !allowed {
		return nil, errors.New("access denied by OPA policy")
	}

	// 2. Data Persistence
	result, err := uc.repo.CreateNode(lockCtx, appID, tenantID, label, properties)
	if err != nil {
		return nil, err
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

	return result, nil
}

func (uc *GraphUsecase) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	return uc.repo.GetNode(ctx, appID, tenantID, nodeID)
}

func (uc *GraphUsecase) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	if overlayID := extractOverlayID(properties); overlayID != "" {
		if uc.overlay == nil {
			return nil, fmt.Errorf("overlay writer is not configured")
		}
		namespace := ComputeNamespace(appID, tenantID)
		return uc.overlay.AddEdgeDelta(ctx, overlayID, namespace, relationType, sourceNodeID, targetNodeID, properties)
	}

	lockCtx := lock.WithOwnerID(ctx, "graph-write-"+uuid.NewString())
	sourceToken, err := uc.acquireNodeLock(lockCtx, appID, tenantID, sourceNodeID)
	if err != nil {
		return nil, err
	}
	defer uc.releaseLock(lockCtx, sourceToken)

	targetToken, err := uc.acquireNodeLock(lockCtx, appID, tenantID, targetNodeID)
	if err != nil {
		return nil, err
	}
	defer uc.releaseLock(lockCtx, targetToken)

	// TODO: Validate relation whitelist
	return uc.repo.CreateEdge(lockCtx, appID, tenantID, relationType, sourceNodeID, targetNodeID, properties)
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

func (uc *GraphUsecase) acquireNodeLock(ctx context.Context, appID, tenantID, nodeID string) (string, error) {
	if uc.lockMgr == nil || nodeID == "" {
		return "", nil
	}
	namespace := ComputeNamespace(appID, tenantID)
	token, err := uc.lockMgr.AcquireNodeLock(ctx, namespace, nodeID, 10*time.Second)
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
	if err := uc.lockMgr.Release(ctx, token); err != nil {
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
