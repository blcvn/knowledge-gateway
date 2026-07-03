package vector

import (
	"context"
	"math"
	"math/bits"
	"strings"
)

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)
	Dimensions() int
	ModelID() string
}

type DeterministicProvider struct {
	dimensions int
	modelID    string
}

func NewDeterministicProvider(dimensions int) DeterministicProvider {
	if dimensions <= 0 {
		dimensions = 8
	}
	return DeterministicProvider{dimensions: dimensions, modelID: "deterministic"}
}

func (p DeterministicProvider) Dimensions() int {
	return p.dimensions
}

func (p DeterministicProvider) ModelID() string {
	return p.modelID
}

func (p DeterministicProvider) Embed(_ context.Context, text string) ([]float64, error) {
	vec := make([]float64, p.dimensions)
	if len(vec) == 0 {
		return vec, nil
	}

	tokens := strings.Fields(strings.ToLower(text))
	if len(tokens) == 0 {
		return vec, nil
	}

	for _, token := range tokens {
		var acc uint64
		for _, r := range token {
			acc ^= uint64(r)
			acc = bits.RotateLeft64(acc*1099511628211, 13)
		}
		idx := int(acc % uint64(len(vec)))
		vec[idx] += 1.0
	}

	norm := 0.0
	for _, value := range vec {
		norm += value * value
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return vec, nil
	}
	for i := range vec {
		vec[i] /= norm
	}
	return vec, nil
}

func (p DeterministicProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	vectors := make([][]float64, 0, len(texts))
	for _, text := range texts {
		vec, err := p.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vec)
	}
	return vectors, nil
}
