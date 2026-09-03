package domain

import "errors"

var (
	ErrIndexNotFound    = errors.New("index not found")
	ErrEmbeddingFailed  = errors.New("embedding generation failed")
	ErrRetrievalFailed  = errors.New("context retrieval failed")
	ErrInvalidRequest   = errors.New("invalid search request")
)
