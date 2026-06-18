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
}

func (p HTTPEmbeddingProvider) Dimensions() int { return 0 }

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
		"model": p.Model,
		"text":  text,
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
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Embedding, nil
}
