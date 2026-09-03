package bifrost

import (
    "context"
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type BifrostEmbedder struct {
    url    string
    model  string
    dims   int
    client *http.Client
}

func NewBifrostEmbedder(url, model string, dims int) *BifrostEmbedder {
    return &BifrostEmbedder{
        url: url, model: model, dims: dims,
        client: &http.Client{Timeout: 5 * time.Second},
    }
}

type embedRequest struct {
    Model string   `json:"model"`
    Input []string `json:"input"`
}

type embedResponse struct {
    Data []struct {
        Embedding []float32 `json:"embedding"`
    } `json:"data"`
}

func (b *BifrostEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    payload, _ := json.Marshal(embedRequest{Model: b.model, Input: []string{text}})
    req, _ := http.NewRequestWithContext(ctx, "POST", b.url+"/v1/embeddings", bytes.NewReader(payload))
    req.Header.Set("Content-Type", "application/json")

    resp, err := b.client.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("bifrost embed: status %d", resp.StatusCode)
    }

    var result embedResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return nil, err }
    if len(result.Data) == 0 { return nil, fmt.Errorf("empty embedding response") }
    return result.Data[0].Embedding, nil
}

// NullEmbedder — no-op embedder when EMBEDDING_PROVIDER=none
type NullEmbedder struct{}
func (n *NullEmbedder) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }
