// Package retriever implements FeedbackRetriever for edge weight reinforcement.
// TASK-CE-009: Feedback Loop — Neo4j edge weight adjustment
//
// Positive feedback (score > 0): edge weights × 1.1 (boost)
// Negative feedback (score < 0): edge weights × 0.9 (penalty)
// Weights clamped to [0.01, 10.0] to prevent runaway reinforcement.
package retriever

import (
	"context"
	"fmt"

	"vnp-memory/services/cognee-search/internal/usecase"
)

const (
	BoostFactor   = 1.1  // positive feedback multiplier
	PenaltyFactor = 0.9  // negative feedback multiplier
	MinWeight     = 0.01 // prevent edge weights reaching zero
	MaxWeight     = 10.0 // cap to prevent runaway boosting
)

// ApplyWeightRequest holds parameters for Neo4j edge weight adjustment.
type ApplyWeightRequest struct {
	NodeIDs  []string // result node IDs to adjust connected edges
	TenantID string
	Score    float64 // -1.0 to 1.0
}

// GraphWeightRepository abstracts Neo4j for weight update operations.
type GraphWeightRepository interface {
	RunCypherReturnIDs(ctx context.Context, cypher string, params map[string]any) ([]string, error)
}

// FeedbackRetriever updates Neo4j edge weights based on user feedback.
type FeedbackRetriever struct {
	graphRepo GraphWeightRepository
}

// NewFeedbackRetriever constructs a FeedbackRetriever.
func NewFeedbackRetriever(graphRepo GraphWeightRepository) *FeedbackRetriever {
	return &FeedbackRetriever{graphRepo: graphRepo}
}

// Strategy returns StrategyFeedback — marks this as a special non-retrieval handler.
func (r *FeedbackRetriever) Strategy() usecase.SearchStrategy {
	return usecase.StrategyFeedback
}

// Retrieve implements the Retriever interface but delegates to ApplyWeightAdjustment.
// The actual feedback logic is called directly from SearchUseCase.handleFeedback.
func (r *FeedbackRetriever) Retrieve(ctx context.Context, req usecase.SearchRequest) ([]usecase.SearchResult, error) {
	// Feedback is not a retrieval strategy — returns empty results
	return nil, nil
}

// ApplyWeightAdjustment updates Neo4j edge weights for all edges from the result nodes.
// Returns the IDs of nodes whose edges were adjusted.
func (r *FeedbackRetriever) ApplyWeightAdjustment(ctx context.Context, req ApplyWeightRequest) ([]string, error) {
	if len(req.NodeIDs) == 0 {
		return nil, nil
	}

	factor := BoostFactor
	if req.Score < 0 {
		factor = PenaltyFactor
	}

	// Cypher: multiply edge weights, clamp to [MinWeight, MaxWeight]
	cypher := `
		MATCH (n)-[r]->(m)
		WHERE n.id IN $node_ids AND n.tenant_id = $tenant_id
		SET r.weight = CASE
			WHEN (coalesce(r.weight, 1.0) * $factor) < $min_weight THEN $min_weight
			WHEN (coalesce(r.weight, 1.0) * $factor) > $max_weight THEN $max_weight
			ELSE (coalesce(r.weight, 1.0) * $factor)
		END
		RETURN n.id AS node_id
	`
	params := map[string]any{
		"node_ids":   req.NodeIDs,
		"tenant_id":  req.TenantID,
		"factor":     factor,
		"min_weight": MinWeight,
		"max_weight": MaxWeight,
	}

	affectedIDs, err := r.graphRepo.RunCypherReturnIDs(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("weight adjustment: %w", err)
	}
	return affectedIDs, nil
}
