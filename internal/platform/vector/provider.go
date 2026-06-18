package vector

import (
	"math"
	"math/bits"
	"strings"
)

type Provider interface {
	Embed(text string) []float64
	Dimensions() int
}

type DeterministicProvider struct {
	dimensions int
}

func NewDeterministicProvider(dimensions int) DeterministicProvider {
	if dimensions <= 0 {
		dimensions = 8
	}
	return DeterministicProvider{dimensions: dimensions}
}

func (p DeterministicProvider) Dimensions() int {
	return p.dimensions
}

func (p DeterministicProvider) Embed(text string) []float64 {
	vec := make([]float64, p.dimensions)
	if len(vec) == 0 {
		return vec
	}

	tokens := strings.Fields(strings.ToLower(text))
	if len(tokens) == 0 {
		return vec
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
		return vec
	}
	for i := range vec {
		vec[i] /= norm
	}
	return vec
}
