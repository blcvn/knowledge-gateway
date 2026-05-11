package external

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/port"
)

type dummyEmbedder struct{}

func NewDummyEmbedder() port.Embedder {
	return &dummyEmbedder{}
}

func (d *dummyEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}
