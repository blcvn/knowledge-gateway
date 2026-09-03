package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"vnp-memory/services/graphiti-knowledge/internal/adapter/cache"
	"vnp-memory/services/graphiti-knowledge/internal/infra/telemetry"
)

type OpenAIConfig struct {
	APIKey      string
	MediumModel string // default: gpt-4o
	SmallModel  string // default: gpt-4o-mini
	BaseURL     string // optional custom endpoint
}

type OpenAILLMClient struct {
	client       *openai.Client
	mediumModel  string
	smallModel   string
	llmCache     cache.LLMCache
	tokenTracker *telemetry.TokenTracker
	retry        RetryConfig
}

func NewOpenAILLMClient(cfg OpenAIConfig, llmCache cache.LLMCache, tracker *telemetry.TokenTracker) *OpenAILLMClient {
	config := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}
	mediumModel := cfg.MediumModel
	if mediumModel == "" {
		mediumModel = "gpt-4o"
	}
	smallModel := cfg.SmallModel
	if smallModel == "" {
		smallModel = "gpt-4o-mini"
	}

	return &OpenAILLMClient{
		client:       openai.NewClientWithConfig(config),
		mediumModel:  mediumModel,
		smallModel:   smallModel,
		llmCache:     llmCache,
		tokenTracker: tracker,
		retry:        DefaultRetryConfig,
	}
}

func (c *OpenAILLMClient) Provider() string { return "openai" }

func (c *OpenAILLMClient) GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error) {
	// Check cache
	cacheKey := computeCacheKey(messages, opts)
	if cached, ok := c.llmCache.Get(ctx, cacheKey); ok {
		return &LLMResponse{Content: cached.Content, Cached: true, Provider: "openai"}, nil
	}

	model := c.mediumModel
	if opts.ModelSize == ModelSizeSmall {
		model = c.smallModel
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	openaiMsgs := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		openaiMsgs[i] = openai.ChatCompletionMessage{Role: m.Role, Content: m.Content}
	}

	req := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    openaiMsgs,
		MaxTokens:   maxTokens,
		Temperature: float32(opts.Temperature),
	}

	// Use JSON schema response format if schema provided
	schema := opts.ResponseSchema
	if schema == nil {
		schema = opts.Schema
	}
	if schema != nil {
		schemaBytes, _ := json.Marshal(schema)
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   opts.PromptName,
				Schema: json.RawMessage(schemaBytes),
				Strict: true,
			},
		}
	}

	var resp openai.ChatCompletionResponse
	var lastErr error
	for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
		resp, lastErr = c.client.CreateChatCompletion(ctx, req)
		if lastErr == nil {
			break
		}
		if !isRetryable(lastErr) {
			break
		}
		delay := c.retry.InitialDelay * time.Duration(1<<(attempt-1))
		if delay > c.retry.MaxDelay {
			delay = c.retry.MaxDelay
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("openai: %w", lastErr)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices returned")
	}

	result := &LLMResponse{
		Content:  []byte(resp.Choices[0].Message.Content),
		Provider: "openai",
		Model:    model,
		TokenUsage: TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	if opts.PromptName != "" {
		c.tokenTracker.Track(opts.PromptName, telemetry.TokenUsage{
			PromptTokens:     result.TokenUsage.PromptTokens,
			CompletionTokens: result.TokenUsage.CompletionTokens,
			TotalTokens:      result.TokenUsage.TotalTokens,
		})
	}
	c.llmCache.Set(ctx, cacheKey, result.Content, c.retry.CacheTTL)
	return result, nil
}
