package reranker

import (
	"context"

	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
	"google.golang.org/grpc"
)

type PipelineClient interface {
	Rerank(ctx context.Context, query string, docs []string) ([]float32, error)
}

type CrossEncoderReranker struct {
	client PipelineClient
}

func NewCrossEncoderReranker(conn *grpc.ClientConn) usecase.Reranker {
	return &CrossEncoderReranker{client: nil} // placeholder
}

func (r *CrossEncoderReranker) Type() domain.RerankerType {
	return domain.RerankerCrossEncoder
}

func (r *CrossEncoderReranker) Rerank(ctx context.Context, query string, results []domain.SearchResult) ([]domain.RankedResult, error) {
	if r.client == nil {
		return mockCrossEncoderFallback(results), nil
	}

	docs := make([]string, len(results))
	for i, res := range results {
		docs[i] = res.Content
	}

	scores, err := r.client.Rerank(ctx, query, docs)
	if err != nil {
		return nil, err
	}

	ranked := make([]domain.RankedResult, len(results))
	for i, res := range results {
		ranked[i] = domain.RankedResult{
			EntityID: res.EntityID,
			Score:    float64(scores[i]),
			Rank:     i + 1,
			Content:  res.Content,
			Metadata: res.Metadata,
		}
	}

	return ranked, nil
}

func mockCrossEncoderFallback(results []domain.SearchResult) []domain.RankedResult {
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
	return ranked
}
