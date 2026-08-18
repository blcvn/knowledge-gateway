package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
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

	// MaxInputChars caps the text sent per input. 0 uses defaultMaxInputChars.
	//
	// This is not a tuning knob, it is a correctness guard. Embedding models reject input past
	// their context window rather than truncating it: BGE-m3 answers
	//
	//	400 "This model's maximum context length is 8192 tokens. However, you requested 22003
	//	tokens in the input for embedding generation."
	//
	// and a rejection propagates — one oversized node aborts the projection run it is part of, so
	// every node after it goes unembedded too. Observed on real BAS data, where a rebuild stopped
	// after 290 of ~6.500 nodes on a single 42.735-character node. Truncating loses the tail of one
	// document; not truncating loses the index.
	MaxInputChars int
}

// defaultMaxInputChars keeps input under an 8192-token window without knowing the tokenizer.
//
// Two characters per token is the worst case seen from this provider (it counted 22.003 tokens for
// 44.006 characters of ASCII); Vietnamese prose measures closer to three. Sizing for the worst case
// means the cap holds for both, at the cost of truncating some Vietnamese text earlier than strictly
// necessary — the right trade, because the failure mode on the other side is a failed run rather
// than a shorter vector.
const defaultMaxInputChars = 16000

// truncateForModel bounds one input, reporting whether it had to.
func truncateForModel(text string, limit int) (string, bool) {
	if limit <= 0 {
		limit = defaultMaxInputChars
	}
	if len(text) <= limit {
		return text, false
	}
	// Cut on a rune boundary so the payload stays valid UTF-8.
	cut := limit
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut], true
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
	input, truncated := truncateForModel(text, p.MaxInputChars)
	if truncated {
		log.Printf("embedding input truncated from %d to %d characters to stay inside the model context window", len(text), len(input))
	}
	payload := map[string]any{
		"input": []string{input},
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
		return nil, statusError(resp.StatusCode, strings.TrimSpace(string(snippet)))
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
