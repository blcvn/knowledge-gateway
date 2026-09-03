package reranker

import (
	"context"

	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
)

type NodeDistanceReranker struct {
	weight float64
}

func NewNodeDistanceReranker(weight float64) usecase.Reranker {
	return &NodeDistanceReranker{weight: weight}
}

func (r *NodeDistanceReranker) Type() domain.RerankerType {
	return domain.RerankerNodeDistance
}

func (r *NodeDistanceReranker) Rerank(ctx context.Context, query string, results []domain.SearchResult) ([]domain.RankedResult, error) {
	ranked := make([]domain.RankedResult, len(results))
	for i, res := range results {
		ranked[i] = domain.RankedResult{
			EntityID: res.EntityID,
			Score:    res.Score + r.weight,
			Rank:     i + 1,
			Content:  res.Content,
			Metadata: res.Metadata,
		}
	}
	return ranked, nil
}
