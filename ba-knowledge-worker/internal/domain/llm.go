package domain

import (
	"context"
)

type LLMService interface {
	Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
