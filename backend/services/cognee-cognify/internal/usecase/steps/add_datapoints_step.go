// Package steps implements the ADD_DATAPOINTS pipeline step.
// Embeds chunks and upserts as vector datapoints.
package steps

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// AddDatapointsStep embeds chunks and upserts them as vector points in Qdrant.
type AddDatapointsStep struct {
	graphRepo  port.GraphRepository
	vectorRepo port.VectorRepository
	embedder   port.EmbedderClient
}

func NewAddDatapointsStep(graph port.GraphRepository, vector port.VectorRepository, embedder port.EmbedderClient) *AddDatapointsStep {
	return &AddDatapointsStep{graphRepo: graph, vectorRepo: vector, embedder: embedder}
}

func (s *AddDatapointsStep) Name() domain.PipelineStep { return domain.StepAddDatapoints }

func (s *AddDatapointsStep) Execute(ctx context.Context, state *usecase.PipelineState) (*usecase.PipelineState, error) {
	if len(state.Chunks) == 0 {
		return state, nil
	}

	texts := make([]string, len(state.Chunks))
	for i, c := range state.Chunks {
		texts[i] = c.Content
	}

	embeddings, err := s.embedder.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed chunks: %w", err)
	}

	for i, chunk := range state.Chunks {
		if i >= len(embeddings) {
			break
		}
		chunkUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("%s-chunk-%d", state.DatasetID, i)))
		if err := s.vectorRepo.UpsertChunkEmbedding(ctx, state.TenantID, chunkUUID, chunk.Content, embeddings[i]); err != nil {
			continue // non-fatal per chunk
		}
	}

	return state, nil
}
