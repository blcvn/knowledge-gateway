package domain

import "errors"

var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrAlreadyCommitted    = errors.New("session already committed")
	ErrInvalidMessageRole  = errors.New("invalid message role")
	ErrLLMExtractionFailed = errors.New("llm extraction failed")
)
