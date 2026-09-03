package usecase

import (
	"github.com/google/uuid"
	"context"
	"log/slog"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// EmbedStage generates vector embeddings for chunks and entities.
type EmbedStage struct {
	embedder  port.EmbedderClient
	vectorRepo port.VectorRepository
	logger     *slog.Logger
}

func NewEmbedStage(embedder port.EmbedderClient, vectorRepo port.VectorRepository, logger *slog.Logger) *EmbedStage {
	return &EmbedStage{embedder: embedder, vectorRepo: vectorRepo, logger: logger.With("stage", "embed")}
}

func (s *EmbedStage) Name() domain.StageType { return domain.StageEmbed }

func (s *EmbedStage) Execute(ctx context.Context, job *domain.CognifyJob, state *CognifyPipelineState) error {
	embeddingsCount := 0

	// Embed chunks in batches
	const batchSize = 32
	for i := 0; i < len(state.Chunks); i += batchSize {
		end := i + batchSize
		if end > len(state.Chunks) {
			end = len(state.Chunks)
		}
		batch := state.Chunks[i:end]

		texts := make([]string, len(batch))
		for j, chunk := range batch {
			texts[j] = chunk.Content
		}

		embeddings, err := s.embedder.Embed(ctx, texts)
		if err != nil {
			return &domain.ErrPipelineFailed{Stage: domain.StageEmbed, Cause: err}
		}

		for j, chunk := range batch {
			if j >= len(embeddings) {
				break
			}
			if err := s.vectorRepo.UpsertChunkEmbedding(ctx, job.TenantID, chunk.ID, chunk.Content, embeddings[j]); err != nil {
				s.logger.Warn("chunk embedding upsert failed", "chunk_id", chunk.ID, "error", err)
			}
			embeddingsCount++
		}
	}

	// Embed entities (name + description)
	for _, entity := range state.Entities {
		text := entity.Name + ": " + entity.Description
		embedding, err := s.embedder.EmbedSingle(ctx, text)
		if err != nil {
			s.logger.Warn("entity embedding failed", "entity_id", entity.ID, "error", err)
			continue
		}
		if err := s.vectorRepo.UpsertEntityEmbedding(ctx, job.TenantID, uuid.MustParse(entity.ID), text, embedding); err != nil {
			s.logger.Warn("entity embedding upsert failed", "entity_id", entity.ID, "error", err)
		}
		embeddingsCount++
	}

	job.Metrics.EmbeddingsGenerated = embeddingsCount
	s.logger.Info("embedding complete", "embeddings_generated", embeddingsCount)
	return nil
}
