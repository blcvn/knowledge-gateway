package knowledge

import "errors"

var (
	ErrLLMTimeout            = errors.New("LLM timeout")
	ErrPromptTooLong         = errors.New("prompt too long")
	ErrInvalidEdgeValidAt    = errors.New("valid_at is required")
	ErrInvalidEdgeTimeWindow = errors.New("valid_at must be before invalid_at")
)
