package llm

import "context"

// ModelSize hints at which model tier to use (affects cost/quality tradeoff)
type ModelSize int

const (
	// ModelSizeMedium — best quality (gpt-4o, claude-3-5-sonnet, gemini-2.0-flash)
	ModelSizeMedium ModelSize = iota
	// ModelSizeSmall — cheaper/faster (gpt-4o-mini, haiku, flash-lite)
	// Used for resolution decisions where cost matters more than quality
	ModelSizeSmall
)

// Message represents a single chat turn
type Message struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// GenerateOpts configures a single LLM call
type GenerateOpts struct {
	ResponseSchema interface{} // JSON schema for structured output (nil = free text)
	PromptName     string      // for token tracking (e.g. "extract_nodes")
	ModelSize      ModelSize
	MaxTokens      int     // 0 = use default (4096)
	Temperature    float64 // 0.0 = deterministic
	Schema         interface{} // alias for ResponseSchema (community use)
}

// TokenUsage captures token consumption for a single LLM call
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (u *TokenUsage) Add(other TokenUsage) {
	u.PromptTokens += other.PromptTokens
	u.CompletionTokens += other.CompletionTokens
	u.TotalTokens += other.TotalTokens
}

// LLMResponse contains the parsed LLM output
type LLMResponse struct {
	Content    []byte     // raw JSON bytes (parsed into target struct by caller)
	TokenUsage TokenUsage
	Cached     bool   // true if served from Redis cache
	Provider   string // "bifrost" | "openai" | "anthropic"
	Model      string // actual model used
}

// LLMClient — unified interface for all LLM providers.
// All implementations must be safe for concurrent use.
// Content is always returned as raw JSON bytes (structured output).
type LLMClient interface {
	GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error)
	Provider() string
}
