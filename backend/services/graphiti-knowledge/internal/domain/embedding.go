package domain

import "errors"

type EmbeddingVector []float32

func (e EmbeddingVector) Validate(expectedDimension int) error {
	if len(e) != expectedDimension {
		return errors.New("invalid embedding dimension")
	}
	return nil
}

type EmbeddingRequest struct {
	Text  string
	Model string
}

type EmbeddingResult struct {
	Vector EmbeddingVector
	Usage  TokenUsage
}
