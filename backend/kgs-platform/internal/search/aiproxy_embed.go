package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AIProxyEmbeddingClient struct {
	client     *http.Client
	baseURL    string
	path       string
	apiKey     string
	model      string
	vectorSize int
}

func NewAIProxyEmbeddingClient(baseURL, path, apiKey, model string, vectorSize int, timeout time.Duration) *AIProxyEmbeddingClient {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultAIProxyBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	path = strings.TrimSpace(path)
	if path == "" {
		path = defaultAIProxyEmbedPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if model == "" {
		model = defaultOpenAIModel
	}
	if vectorSize <= 0 {
		vectorSize = defaultEmbeddingVectorSize
	}
	if timeout <= 0 {
		timeout = defaultEmbeddingTimeout
	}

	return &AIProxyEmbeddingClient{
		client:     &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		path:       path,
		apiKey:     strings.TrimSpace(apiKey),
		model:      model,
		vectorSize: vectorSize,
	}
}

func (c *AIProxyEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if c == nil {
		return nil, fmt.Errorf("ai-proxy embed client is nil")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	reqBody, err := json.Marshal(map[string]any{
		"model": c.model,
		"input": text,
		"text":  text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ai-proxy embeddings returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	vector, err := parseEmbeddingVector(raw)
	if err != nil {
		return nil, fmt.Errorf("parse ai-proxy embedding response: %w", err)
	}
	if c.vectorSize > 0 && len(vector) != c.vectorSize {
		return nil, fmt.Errorf("ai-proxy vector size mismatch: got=%d want=%d", len(vector), c.vectorSize)
	}
	normalizeVector(vector)
	return vector, nil
}

func parseEmbeddingVector(raw []byte) ([]float32, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	// Format 1: [0.1, 0.2, ...]
	if vector := toFloat32Slice(decoded); len(vector) > 0 {
		return vector, nil
	}

	root, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("response must be object or array")
	}

	// Format 2: {"embedding":[...]} / {"vector":[...]}
	for _, key := range []string{"embedding", "vector"} {
		if vector := toFloat32Slice(root[key]); len(vector) > 0 {
			return vector, nil
		}
	}

	// Format 3: OpenAI style {"data":[{"embedding":[...]}]}
	if dataSlice, ok := root["data"].([]any); ok && len(dataSlice) > 0 {
		if first, ok := dataSlice[0].(map[string]any); ok {
			if vector := toFloat32Slice(first["embedding"]); len(vector) > 0 {
				return vector, nil
			}
		}
	}

	// Format 4: ai-proxy complete response with completion.text containing JSON array
	if completion, ok := root["completion"].(map[string]any); ok {
		if text, ok := completion["text"].(string); ok && strings.TrimSpace(text) != "" {
			if vector := parseTextVector(text); len(vector) > 0 {
				return vector, nil
			}
		}
	}
	return nil, fmt.Errorf("embedding vector not found in response")
}

func parseTextVector(text string) []float32 {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		return nil
	}
	return toFloat32Slice(decoded)
}

func toFloat32Slice(v any) []float32 {
	items, ok := v.([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	vector := make([]float32, 0, len(items))
	for _, item := range items {
		switch n := item.(type) {
		case float64:
			vector = append(vector, float32(n))
		case float32:
			vector = append(vector, n)
		case int:
			vector = append(vector, float32(n))
		case int32:
			vector = append(vector, float32(n))
		case int64:
			vector = append(vector, float32(n))
		default:
			return nil
		}
	}
	return vector
}
