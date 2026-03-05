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

type OpenAIEmbeddingClient struct {
	client     *http.Client
	baseURL    string
	apiKey     string
	model      string
	vectorSize int
}

type openAIEmbeddingRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func NewOpenAIEmbeddingClient(baseURL, apiKey, model string, vectorSize int, timeout time.Duration) *OpenAIEmbeddingClient {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	if model == "" {
		model = defaultOpenAIModel
	}
	if vectorSize <= 0 {
		vectorSize = defaultEmbeddingVectorSize
	}
	if timeout <= 0 {
		timeout = defaultEmbeddingTimeout
	}
	return &OpenAIEmbeddingClient{
		client:     &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		apiKey:     strings.TrimSpace(apiKey),
		model:      model,
		vectorSize: vectorSize,
	}
}

func (c *OpenAIEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if c == nil {
		return nil, fmt.Errorf("openai embed client is nil")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}
	if c.apiKey == "" {
		return nil, fmt.Errorf("openai api key is empty")
	}

	reqBody, err := json.Marshal(openAIEmbeddingRequest{
		Model:          c.model,
		Input:          text,
		EncodingFormat: "float",
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("openai embeddings returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var body openAIEmbeddingResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("decode embeddings response: %w", err)
	}
	if body.Error != nil {
		return nil, fmt.Errorf("openai embeddings error: %s (%s)", body.Error.Message, body.Error.Type)
	}
	if len(body.Data) == 0 || len(body.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("openai embeddings response has empty vector")
	}

	vector := append([]float32(nil), body.Data[0].Embedding...)
	if c.vectorSize > 0 && len(vector) != c.vectorSize {
		return nil, fmt.Errorf("openai vector size mismatch: got=%d want=%d", len(vector), c.vectorSize)
	}
	normalizeVector(vector)
	return vector, nil
}
