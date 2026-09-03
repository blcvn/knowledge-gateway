package domain

import "errors"

var (
	ErrLLMTimeout          = errors.New("llm request timed out")
	ErrPromptTooLong       = errors.New("prompt exceeds maximum token limit")
	ErrProviderUnavailable = errors.New("llm provider is unavailable")
	ErrMalformedLLMResponse = errors.New("malformed response from llm")
)
