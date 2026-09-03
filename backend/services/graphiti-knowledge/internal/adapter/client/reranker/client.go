package reranker

import "context"

// CrossEncoderClient — reranking interface for neural cross-encoder reranking.
// Returns a score per passage (higher = more relevant to query).
type CrossEncoderClient interface {
	Rerank(ctx context.Context, query string, passages []string) ([]float64, error)
}

// NoopReranker — passthrough reranker (returns equal scores).
// Used when no reranker service is configured.
type NoopReranker struct{}

func (n *NoopReranker) Rerank(ctx context.Context, query string, passages []string) ([]float64, error) {
	scores := make([]float64, len(passages))
	for i := range scores {
		scores[i] = 1.0
	}
	return scores, nil
}
