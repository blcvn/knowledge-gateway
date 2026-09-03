package reranker

import (
	"context"

	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
)

type EpisodeMentionsReranker struct {
}

func NewEpisodeMentionsReranker() usecase.Reranker {
	return &EpisodeMentionsReranker{}
}

func (r *EpisodeMentionsReranker) Type() domain.RerankerType {
	return domain.RerankerEpisodeMentions
}

func (r *EpisodeMentionsReranker) Rerank(ctx context.Context, query string, results []domain.SearchResult) ([]domain.RankedResult, error) {
	ranked := make([]domain.RankedResult, len(results))
	for i, res := range results {
		ranked[i] = domain.RankedResult{
			EntityID: res.EntityID,
			Score:    res.Score,
			Rank:     i + 1,
			Content:  res.Content,
			Metadata: res.Metadata,
		}
	}
	return ranked, nil
}
