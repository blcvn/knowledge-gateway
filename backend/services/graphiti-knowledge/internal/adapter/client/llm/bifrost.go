package llm

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"vnp-memory/services/graphiti-knowledge/internal/adapter/cache"
	"vnp-memory/services/graphiti-knowledge/internal/infra/telemetry"
)

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	CacheTTL     time.Duration
}

var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 500 * time.Millisecond,
	MaxDelay:     10 * time.Second,
	CacheTTL:     1 * time.Hour,
}

type BifrostLLMClient struct {
	baseURL      string
	apiKey       string
	mediumModel  string
	smallModel   string
	llmCache     cache.LLMCache
	tokenTracker *telemetry.TokenTracker
	retry        RetryConfig
}

type BifrostConfig struct {
	BaseURL     string
	APIKey      string
	MediumModel string
	SmallModel  string
}

func NewBifrostLLMClient(cfg BifrostConfig, llmCache cache.LLMCache, tracker *telemetry.TokenTracker) *BifrostLLMClient {
	return &BifrostLLMClient{
		baseURL:      cfg.BaseURL,
		apiKey:       cfg.APIKey,
		mediumModel:  cfg.MediumModel,
		smallModel:   cfg.SmallModel,
		llmCache:     llmCache,
		tokenTracker: tracker,
		retry:        DefaultRetryConfig,
	}
}

func (c *BifrostLLMClient) Provider() string { return "bifrost" }

func (c *BifrostLLMClient) GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error) {
	// 1. Check cache
	cacheKey := computeCacheKey(messages, opts)
	if cached, ok := c.llmCache.Get(ctx, cacheKey); ok {
		return &LLMResponse{Content: cached.Content, Cached: true, Provider: "bifrost"}, nil
	}

	// 2. Select model
	model := c.mediumModel
	if opts.ModelSize == ModelSizeSmall {
		model = c.smallModel
	}

	// 3. Build request payload
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	schema := opts.ResponseSchema
	if schema == nil {
		schema = opts.Schema
	}

	payload := map[string]any{
		"model":       model,
		"messages":    mapMessagesToMaps(messages),
		"max_tokens":  maxTokens,
		"temperature": opts.Temperature,
	}
	if schema != nil {
		payload["response_format"] = map[string]any{
			"type":        "json_schema",
			"json_schema": schema,
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal bifrost request: %w", err)
	}

	// 4. Execute with retry + exponential backoff
	var result *LLMResponse
	var lastErr error
	for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
		result, lastErr = c.doHTTPCall(ctx, bodyBytes, model)
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
		return nil, fmt.Errorf("bifrost after %d attempts: %w", c.retry.MaxAttempts, lastErr)
	}

	// 5. Track usage + cache
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

func (c *BifrostLLMClient) doHTTPCall(ctx context.Context, body []byte, model string) (*LLMResponse, error) {
	req, err := newHTTPRequest(ctx, "POST", c.baseURL+"/v1/chat/completions", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := defaultHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("bifrost status %d: %s", resp.StatusCode, errBody.String())
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode bifrost response: %w", err)
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("no choices in bifrost response")
	}

	return &LLMResponse{
		Content:  []byte(out.Choices[0].Message.Content),
		Provider: "bifrost",
		Model:    model,
		TokenUsage: TokenUsage{
			PromptTokens:     out.Usage.PromptTokens,
			CompletionTokens: out.Usage.CompletionTokens,
			TotalTokens:      out.Usage.TotalTokens,
		},
	}, nil
}

// computeCacheKey generates a stable cache key from messages + prompt config
func computeCacheKey(messages []Message, opts GenerateOpts) string {
	h := md5.New()
	json.NewEncoder(h).Encode(messages)
	fmt.Fprintf(h, "|%s|%d", opts.PromptName, opts.ModelSize)
	return hex.EncodeToString(h.Sum(nil))
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsStr(msg, "429") || containsStr(msg, "503") ||
		containsStr(msg, "timeout") || containsStr(msg, "connection")
}

func mapMessagesToMaps(msgs []Message) []map[string]string {
	result := make([]map[string]string, len(msgs))
	for i, m := range msgs {
		result[i] = map[string]string{"role": m.Role, "content": m.Content}
	}
	return result
}

func containsStr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
