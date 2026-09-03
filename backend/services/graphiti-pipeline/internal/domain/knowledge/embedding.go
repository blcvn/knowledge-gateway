package knowledge

import "errors"

type EmbeddingVector []float64

func (v EmbeddingVector) Validate(expectedDimension int) error {
	if len(v) != expectedDimension {
		return errors.New("invalid embedding dimension")
	}
	return nil
}

type EmbeddingRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}
