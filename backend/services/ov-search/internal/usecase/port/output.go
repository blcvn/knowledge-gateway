package port

import (
	"context"

	"vnp-memory/ov-search/internal/domain/model"
)

type EmbedderPort interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, []float32, error)
}

type FileReaderPort interface {
	ReadContext(ctx context.Context, path string, level string) (string, error)
}

type RerankPort interface {
	Rerank(ctx context.Context, query string, docs []model.SearchResult, strategy string) ([]model.SearchResult, error)
}
