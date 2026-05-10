// Package qdrant implements the VectorStore interface using Qdrant vector database.
//
// Qdrant provides specialized ANN performance for high-throughput workloads
// (used by Cognee engine for entity embeddings).
package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/vnp-community/vnp-memory/pkg/vectorstore"
)

// Adapter implements vectorstore.VectorStore using Qdrant REST API.
type Adapter struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new Qdrant adapter.
func New(baseURL string) *Adapter {
	return &Adapter{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (a *Adapter) EnsureCollection(ctx context.Context, cfg vectorstore.CollectionConfig) error {
	// Check if collection exists
	resp, err := a.doRequest(ctx, "GET", fmt.Sprintf("/collections/%s", cfg.Name), nil)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil // Already exists
		}
	}

	// Create collection
	body := map[string]any{
		"vectors": map[string]any{
			"size":     cfg.Dimension,
			"distance": qdrantDistance(cfg.DistanceMetric),
		},
	}
	resp, err = a.doRequest(ctx, "PUT", fmt.Sprintf("/collections/%s", cfg.Name), body)
	if err != nil {
		return fmt.Errorf("qdrant: create collection %s: %w", cfg.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant: create collection %s: status %d: %s", cfg.Name, resp.StatusCode, b)
	}
	return nil
}

func (a *Adapter) Upsert(ctx context.Context, collection string, docs []vectorstore.Document) error {
	points := make([]map[string]any, len(docs))
	for i, doc := range docs {
		points[i] = map[string]any{
			"id":      doc.ID,
			"vector":  doc.Vector,
			"payload": doc.Metadata,
		}
		if doc.Content != "" {
			points[i]["payload"].(map[string]any)["_content"] = doc.Content
		}
	}

	body := map[string]any{"points": points}
	resp, err := a.doRequest(ctx, "PUT", fmt.Sprintf("/collections/%s/points", collection), body)
	if err != nil {
		return fmt.Errorf("qdrant: upsert %s: %w", collection, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant: upsert %s: status %d: %s", collection, resp.StatusCode, b)
	}
	return nil
}

func (a *Adapter) Search(ctx context.Context, params vectorstore.SearchParams) ([]vectorstore.Document, error) {
	body := map[string]any{
		"vector":          params.Vector,
		"limit":           params.TopK,
		"score_threshold": params.MinScore,
		"with_payload":    true,
	}
	if len(params.Filter) > 0 {
		body["filter"] = params.Filter
	}

	resp, err := a.doRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/search", params.Collection), body)
	if err != nil {
		return nil, fmt.Errorf("qdrant: search %s: %w", params.Collection, err)
	}
	defer resp.Body.Close()

	var result struct {
		Result []struct {
			ID      string         `json:"id"`
			Score   float64        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("qdrant: decode search response: %w", err)
	}

	docs := make([]vectorstore.Document, len(result.Result))
	for i, r := range result.Result {
		docs[i] = vectorstore.Document{
			ID:       r.ID,
			Score:    r.Score,
			Metadata: r.Payload,
		}
		if content, ok := r.Payload["_content"].(string); ok {
			docs[i].Content = content
			delete(docs[i].Metadata, "_content")
		}
	}
	return docs, nil
}

func (a *Adapter) Delete(ctx context.Context, collection string, ids []string) error {
	body := map[string]any{
		"points": ids,
	}
	resp, err := a.doRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", collection), body)
	if err != nil {
		return fmt.Errorf("qdrant: delete from %s: %w", collection, err)
	}
	defer resp.Body.Close()
	return nil
}

func (a *Adapter) DropCollection(ctx context.Context, collection string) error {
	resp, err := a.doRequest(ctx, "DELETE", fmt.Sprintf("/collections/%s", collection), nil)
	if err != nil {
		return fmt.Errorf("qdrant: drop %s: %w", collection, err)
	}
	defer resp.Body.Close()
	return nil
}

func (a *Adapter) Close() error {
	a.httpClient.CloseIdleConnections()
	return nil
}

func (a *Adapter) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return a.httpClient.Do(req)
}

func qdrantDistance(m vectorstore.DistanceMetric) string {
	switch m {
	case vectorstore.Euclidean:
		return "Euclid"
	case vectorstore.DotProduct:
		return "Dot"
	default:
		return "Cosine"
	}
}
