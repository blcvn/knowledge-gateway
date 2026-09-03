package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrLLMRequestFailed = errors.New("llm: request failed")
	ErrLLMBadResponse   = errors.New("llm: bad response format")
)

// Option configures an LLM call.
type Option func(*callOptions)

type callOptions struct {
	maxTokens   int
	temperature float64
	jsonMode    bool
}

// WithJSONMode requests a JSON-mode response.
func WithJSONMode() Option { return func(o *callOptions) { o.jsonMode = true } }

// WithMaxTokens sets the max token limit.
func WithMaxTokens(n int) Option { return func(o *callOptions) { o.maxTokens = n } }

// LLMClient is the primary interface for language model calls.
type LLMClient interface {
	CompleteJSON(ctx context.Context, prompt string, opts ...Option) (json.RawMessage, error)
	Complete(ctx context.Context, prompt string, opts ...Option) (string, error)
}

// BifrostClient calls an OpenAI-compatible Bifrost proxy (internal gateway).
type BifrostClient struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

// NewBifrostClient creates a new BifrostClient.
func NewBifrostClient(baseURL, model, apiKey string) *BifrostClient {
	return &BifrostClient{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type chatCompletionRequest struct {
	Model          string              `json:"model"`
	Messages       []map[string]string `json:"messages"`
	ResponseFormat *map[string]string  `json:"response_format,omitempty"`
	MaxTokens      int                 `json:"max_tokens,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// CompleteJSON sends a prompt and parses the content field as JSON.
func (c *BifrostClient) CompleteJSON(ctx context.Context, prompt string, opts ...Option) (json.RawMessage, error) {
	text, err := c.doRequest(ctx, prompt, true, opts...)
	if err != nil {
		return nil, err
	}
	return extractJSONContent(text)
}

// Complete sends a prompt and returns the raw text content.
func (c *BifrostClient) Complete(ctx context.Context, prompt string, opts ...Option) (string, error) {
	return c.doRequest(ctx, prompt, false, opts...)
}

func (c *BifrostClient) doRequest(ctx context.Context, prompt string, jsonMode bool, opts ...Option) (string, error) {
	o := &callOptions{temperature: 0.2}
	for _, opt := range opts {
		opt(o)
	}

	reqBody := chatCompletionRequest{
		Model: c.model,
		Messages: []map[string]string{
			{"role": "user", "content": prompt},
		},
		MaxTokens: o.maxTokens,
	}
	if jsonMode || o.jsonMode {
		rf := map[string]string{"type": "json_object"}
		reqBody.ResponseFormat = &rf
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrLLMRequestFailed, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("%w: HTTP %d: %s", ErrLLMRequestFailed, resp.StatusCode, string(respBytes))
	}

	var result chatCompletionResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("%w: %v", ErrLLMBadResponse, err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("%w: empty choices", ErrLLMBadResponse)
	}
	return result.Choices[0].Message.Content, nil
}

// extractJSONContent tries to parse text as JSON, returning raw message.
func extractJSONContent(text string) (json.RawMessage, error) {
	// Try parsing directly
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err == nil {
		return raw, nil
	}
	// It might be a stringified JSON wrapped in backticks or quotes
	// Try to find JSON object within
	start := bytes.IndexByte([]byte(text), '{')
	end := bytes.LastIndexByte([]byte(text), '}')
	if start >= 0 && end > start {
		candidate := text[start : end+1]
		if err := json.Unmarshal([]byte(candidate), &raw); err == nil {
			return raw, nil
		}
	}
	// Return as a JSON string
	b, _ := json.Marshal(text)
	return b, nil
}

// OpenAICompatClient wraps BifrostClient for Ollama-style local deployments.
func NewOpenAICompatClient(baseURL, model string) *BifrostClient {
	return NewBifrostClient(baseURL, model, "")
}
