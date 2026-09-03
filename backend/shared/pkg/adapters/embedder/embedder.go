package embedder

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
	ErrEmbeddingFailed   = errors.New("embedder: request failed")
	ErrEmbeddingDisabled = errors.New("embedder: embedding is disabled")
)

// Embedder is the interface for text embedding providers.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
	Dimension() int
	IsEnabled() bool
}

// JinaEmbedder uses the Jina AI embedding API.
type JinaEmbedder struct {
	apiKey     string
	model      string
	dim        int
	httpClient *http.Client
}

func NewJinaEmbedder(apiKey, model string, dim int) *JinaEmbedder {
	return &JinaEmbedder{
		apiKey: apiKey, model: model, dim: dim,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (j *JinaEmbedder) Dimension() int  { return j.dim }
func (j *JinaEmbedder) IsEnabled() bool { return true }

func (j *JinaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody, _ := json.Marshal(map[string]any{"model": j.model, "input": texts})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.jina.ai/v1/embeddings", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+j.apiKey)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: HTTP %d: %s", ErrEmbeddingFailed, resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: parse: %v", ErrEmbeddingFailed, err)
	}
	out := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		out[i] = d.Embedding
	}
	return out, nil
}

func (j *JinaEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	vecs, err := j.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, ErrEmbeddingFailed
	}
	return vecs[0], nil
}

// OllamaEmbedder uses Ollama's local embedding API.
type OllamaEmbedder struct {
	baseURL    string
	model      string
	dim        int
	httpClient *http.Client
}

func NewOllamaEmbedder(baseURL, model string, dim int) *OllamaEmbedder {
	return &OllamaEmbedder{
		baseURL: baseURL, model: model, dim: dim,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (o *OllamaEmbedder) Dimension() int  { return o.dim }
func (o *OllamaEmbedder) IsEnabled() bool { return true }

func (o *OllamaEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	reqBody, _ := json.Marshal(map[string]string{"model": o.model, "prompt": query})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/embeddings", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}
	defer resp.Body.Close()

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: parse: %v", ErrEmbeddingFailed, err)
	}
	return result.Embedding, nil
}

func (o *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, t := range texts {
		emb, err := o.EmbedQuery(ctx, t)
		if err != nil {
			return nil, err
		}
		results[i] = emb
	}
	return results, nil
}

// DisabledEmbedder is a no-op embedder used when embedding is turned off.
type DisabledEmbedder struct{}

func (d *DisabledEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, ErrEmbeddingDisabled
}
func (d *DisabledEmbedder) EmbedQuery(_ context.Context, _ string) ([]float32, error) {
	return nil, ErrEmbeddingDisabled
}
func (d *DisabledEmbedder) Dimension() int  { return 0 }
func (d *DisabledEmbedder) IsEnabled() bool { return false }
