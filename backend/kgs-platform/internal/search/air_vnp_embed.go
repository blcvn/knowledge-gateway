package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	defaultVNPEmbedURL        = "https://genai.vnpay.vn/aigateway/embed/v1/embeddings"
	defaultVNPEmbedVectorSize = 1024
)

// VNPEmbeddingClient implements EmbeddingClient by calling the VNPay GenAI embedding API.
// The VNPay embedding model only supports 1024 dimensions.
type VNPEmbeddingClient struct {
	url        string
	apiKey     string
	httpClient *http.Client
}

// NewVNPEmbeddingClient creates a new VNPEmbeddingClient.
// baseURL is the API endpoint (use "" for default).
// apiKey is the Bearer token used for authorization.
// timeout is the HTTP request timeout (use 0 for default 15s).
// Note: VNPay embedding model only supports 1024 dimensions.
func NewVNPEmbeddingClient(baseURL, apiKey string, timeout time.Duration) *VNPEmbeddingClient {
	if baseURL == "" {
		baseURL = defaultVNPEmbedURL
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &VNPEmbeddingClient{
		url:        baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// --- request / response DTOs ---

type vnpEmbedRequest struct {
	Input string `json:"input"`
}

type vnpEmbedResponse struct {
	Data []vnpEmbedData `json:"data"`
}

type vnpEmbedData struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// Embed sends the text to the VNPay embedding API and returns the embedding vector.
func (c *VNPEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("vnp embed: empty text")
	}

	reqBody := vnpEmbedRequest{
		Input: text,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("vnp embed: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("vnp embed: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("http_x_request_id", uuid.New().String())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vnp embed: do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vnp embed: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vnp embed: unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	var embedResp vnpEmbedResponse
	if err := json.Unmarshal(respBytes, &embedResp); err != nil {
		return nil, fmt.Errorf("vnp embed: unmarshal response: %w", err)
	}

	if len(embedResp.Data) == 0 || len(embedResp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("vnp embed: empty embedding in response")
	}

	// Convert []float64 → []float32
	raw := embedResp.Data[0].Embedding
	out := make([]float32, len(raw))
	for i, v := range raw {
		out[i] = float32(v)
	}
	return out, nil
}
