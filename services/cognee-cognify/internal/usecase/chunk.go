package usecase

import (
	"context"
	"log/slog"
	"strings"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
)

// ChunkStage splits text into chunks based on classification strategy.
type ChunkStage struct {
	logger *slog.Logger
}

func NewChunkStage(logger *slog.Logger) *ChunkStage {
	return &ChunkStage{logger: logger.With("stage", "chunk")}
}

func (s *ChunkStage) Name() domain.StageType { return domain.StageChunk }

func (s *ChunkStage) Execute(ctx context.Context, job *domain.CognifyJob, state *PipelineState) error {
	chunkSize := job.Config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 1024
	}
	overlap := job.Config.ChunkOverlap
	if overlap <= 0 {
		overlap = 128
	}

	var allChunks []*domain.Chunk
	for _, item := range state.TextItems {
		if item.Text == "" {
			continue
		}
		chunks := recursiveChunk(job.ID, item.ID, item.Text, chunkSize, overlap)
		allChunks = append(allChunks, chunks...)
	}

	state.Chunks = allChunks
	job.Metrics.ChunksCreated = len(allChunks)
	s.logger.Info("chunking complete", "chunks_created", len(allChunks), "chunk_size", chunkSize)
	return nil
}

// recursiveChunk splits text into overlapping chunks based on character count.
// In production, this would use a proper tokenizer for token-based splitting.
func recursiveChunk(jobID interface{ String() string }, sourceItem, text string, size, overlap int) []*domain.Chunk {
	// Use runes to handle multibyte properly
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	var chunks []*domain.Chunk
	start := 0
	index := 0

	for start < len(runes) {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}

		// Try to break at a sentence boundary
		chunkText := string(runes[start:end])
		if end < len(runes) {
			if idx := strings.LastIndexAny(chunkText, ".!?\n"); idx > size/2 {
				end = start + idx + 1
				chunkText = string(runes[start:end])
			}
		}

		chunk := domain.NewChunk(
			parseUUID(jobID.String()),
			index,
			strings.TrimSpace(chunkText),
			len([]rune(chunkText)),
			sourceItem,
		)
		chunks = append(chunks, chunk)

		start = end - overlap
		if start <= 0 && end >= len(runes) {
			break
		}
		index++
	}

	return chunks
}

func parseUUID(s string) interface{ String() string } {
	// Simplified — in real code use uuid.Parse
	return uuidWrapper(s)
}

type uuidWrapper string

func (u uuidWrapper) String() string { return string(u) }
