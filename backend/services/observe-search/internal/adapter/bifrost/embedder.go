package bifrost

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// Embedder uses Bifrost LLM gateway to generate text embeddings.
type Embedder struct {
    client *http.Client
    url    string
    model  string
    dims   int
}

func NewEmbedder(url, model string, dims int) *Embedder {
    return &Embedder{client: &http.Client{}, url: url, model: model, dims: dims}
}

func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
    body, _ := json.Marshal(map[string]any{
        "model": e.model,
        "input": text,
    })
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url+"/v1/embeddings", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := e.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("bifrost embedder: status %d", resp.StatusCode)
    }

    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    if len(result.Data) == 0 {
        return nil, fmt.Errorf("bifrost: empty embedding response")
    }
    return result.Data[0].Embedding, nil
}

// NullEmbedder disables vector search (BM25-only mode).
type NullEmbedder struct{}

func (n *NullEmbedder) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }
