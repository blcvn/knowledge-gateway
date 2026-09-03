// Package steps implements the CHUNK pipeline step.
// Splits raw content into overlapping sliding-window chunks.
package steps

import (
	"context"
	"strings"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase"
)

// ChunkStep splits raw content into overlapping text chunks.
// Uses a sliding window with 20% overlap for context continuity.
type ChunkStep struct {
	defaultSize int
}

func NewChunkStep(defaultSize int) *ChunkStep {
	if defaultSize <= 0 {
		defaultSize = 512
	}
	return &ChunkStep{defaultSize: defaultSize}
}

func (s *ChunkStep) Name() domain.PipelineStep { return domain.StepChunk }

func (s *ChunkStep) Execute(ctx context.Context, state *usecase.PipelineState) (*usecase.PipelineState, error) {
	size := s.defaultSize
	if state.Options.ChunkSize > 0 {
		size = state.Options.ChunkSize
	}
	overlap := size / 5 // 20% overlap

	var allChunks []usecase.Chunk
	for _, content := range state.RawContent {
		allChunks = append(allChunks, slidingWindowChunk(content, size, overlap)...)
	}
	state.Chunks = allChunks
	return state, nil
}

// slidingWindowChunk splits text into word-boundary chunks with overlap.
func slidingWindowChunk(text string, size, overlap int) []usecase.Chunk {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	step := size - overlap
	if step <= 0 {
		step = size
	}

	var chunks []usecase.Chunk
	for i := 0; i < len(words); i += step {
		end := i + size
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, usecase.Chunk{Content: chunk})
		if end == len(words) {
			break
		}
	}
	return chunks
}
