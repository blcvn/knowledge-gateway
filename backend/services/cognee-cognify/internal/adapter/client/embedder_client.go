package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

type embedderClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewEmbedderClient(baseURL, model string, httpClient *http.Client) port.EmbedderClient {
	return &embedderClient{
		baseURL:    baseURL,
		model:      model,
		httpClient: httpClient,
	}
}

func (c *embedderClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	payload := map[string]any{
		"model": c.model,
		"input": texts,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/embeddings", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedder api error: status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(texts))
	for _, d := range result.Data {
		embeddings[d.Index] = d.Embedding
	}
	return embeddings, nil
}

func (c *embedderClient) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}
