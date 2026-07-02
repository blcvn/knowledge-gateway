package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPEmbeddingProvider struct {
	URL        string
	Model      string
	APIKey     string
	Timeout    time.Duration
	HTTPClient *http.Client
	// Dims is the expected vector dimension for this provider.
	// Set via EMBEDDING_DIMENSIONS. 0 means the dimension is inferred
	// from the provider's response (works with unbounded vector columns).
	Dims int
}

func (p HTTPEmbeddingProvider) Dimensions() int { return p.Dims }

func (p HTTPEmbeddingProvider) ModelID() string { return p.Model }

func (p HTTPEmbeddingProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	if strings.TrimSpace(p.URL) == "" {
		return nil, errors.New("embedding url is required")
	}
	client := p.HTTPClient
	if client == nil {
		timeout := p.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	payload := map[string]any{
		"input": []string{text},
	}
	if strings.TrimSpace(p.Model) != "" {
		payload["model"] = p.Model
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(p.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("embedding provider status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	var out struct {
		Embedding []float64 `json:"embedding"`
		Data      []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embedding) > 0 {
		return out.Embedding, nil
	}
	if len(out.Data) > 0 && len(out.Data[0].Embedding) > 0 {
		return out.Data[0].Embedding, nil
	}
	return nil, errors.New("embedding provider returned no embedding")
}

func (p HTTPEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	vectors := make([][]float64, 0, len(texts))
	for _, text := range texts {
		vec, err := p.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vec)
	}
	return vectors, nil
}
