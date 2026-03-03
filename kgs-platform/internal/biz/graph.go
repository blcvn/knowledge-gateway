package biz

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

// GraphRepo defines the graph data persistence interface
type GraphRepo interface {
	CreateNode(ctx context.Context, appID string, label string, properties map[string]any) (map[string]any, error)
	CreateEdge(ctx context.Context, appID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error)
	ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error)
}

type GraphUsecase struct {
	repo     GraphRepo
	ontology *OntologySyncManager
	planner  *QueryPlanner
	opa      *OPAClient
	redisCli *redis.Client
	log      *log.Helper
}

func NewGraphUsecase(repo GraphRepo, planner *QueryPlanner, opa *OPAClient, redisCli *redis.Client, logger log.Logger) *GraphUsecase {
	return &GraphUsecase{
		repo:     repo,
		planner:  planner,
		opa:      opa,
		redisCli: redisCli,
		log:      log.NewHelper(logger),
	}
}

func (uc *GraphUsecase) CreateNode(ctx context.Context, appID string, label string, properties map[string]any) (map[string]any, error) {
	// 1. OPA Policy Check
	allowed, err := uc.opa.EvaluatePolicy(ctx, appID, "CREATE_NODE", label)
	if err != nil {
		uc.log.Errorf("OPA evaluation failed: %v", err)
		return nil, err
	}
	if !allowed {
		return nil, errors.New("access denied by OPA policy")
	}

	// 2. Data Persistence
	result, err := uc.repo.CreateNode(ctx, appID, label, properties)
	if err != nil {
		return nil, err
	}

	// 3. Trigger Event
	uc.redisCli.XAdd(ctx, &redis.XAddArgs{
		Stream: "kgs:events:nodes",
		Values: map[string]interface{}{
			"event_type": "node.created",
			"app_id":     appID,
			"label":      label,
		},
	})

	return result, nil
}

func (uc *GraphUsecase) CreateEdge(ctx context.Context, appID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	// TODO: Validate relation whitelist
	return uc.repo.CreateEdge(ctx, appID, relationType, sourceNodeID, targetNodeID, properties)
}

func (uc *GraphUsecase) GetContext(ctx context.Context, appID string, nodeID string, depth int, direction string) (map[string]any, error) {
	if err := ValidateDepth(depth); err != nil {
		return nil, err
	}
	cypher := uc.planner.BuildContextQuery("", direction)
	params := map[string]any{
		"app_id":  appID,
		"node_id": nodeID,
	}
	return uc.repo.ExecuteQuery(ctx, cypher, params)
}

func (uc *GraphUsecase) GetImpact(ctx context.Context, appID string, nodeID string, maxDepth int) (map[string]any, error) {
	if err := ValidateDepth(maxDepth); err != nil {
		return nil, err
	}
	cypher := uc.planner.BuildImpactQuery("", maxDepth)
	params := map[string]any{
		"app_id":  appID,
		"node_id": nodeID,
	}
	return uc.repo.ExecuteQuery(ctx, cypher, params)
}

func (uc *GraphUsecase) GetCoverage(ctx context.Context, appID string, nodeID string, maxDepth int) (map[string]any, error) {
	if err := ValidateDepth(maxDepth); err != nil {
		return nil, err
	}
	cypher := uc.planner.BuildCoverageQuery("", maxDepth)
	params := map[string]any{
		"app_id":  appID,
		"node_id": nodeID,
	}
	return uc.repo.ExecuteQuery(ctx, cypher, params)
}

func (uc *GraphUsecase) GetSubgraph(ctx context.Context, appID string, nodeIDs []string) (map[string]any, error) {
	// Guardrail for maximum bulk queries
	if err := ValidateNodeCount(len(nodeIDs)); err != nil {
		return nil, err
	}
	cypher := uc.planner.BuildSubgraphQuery()
	params := map[string]any{
		"app_id":   appID,
		"node_ids": nodeIDs,
	}
	return uc.repo.ExecuteQuery(ctx, cypher, params)
}
