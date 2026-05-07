package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type QdrantClient struct {
	baseURL    string
	collection string
	http       *http.Client
}

type QdrantPoint struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type SearchResult struct {
	ID      string         `json:"id"`
	Score   float32        `json:"score"`
	Payload map[string]any `json:"payload"`
}

func NewQdrantClient(baseURL, collection string) *QdrantClient {
	return &QdrantClient{baseURL: baseURL, collection: collection, http: &http.Client{}}
}

func (q *QdrantClient) Upsert(ctx context.Context, points []QdrantPoint) error {
	body, _ := json.Marshal(map[string]any{"points": points})
	req, _ := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/collections/%s/points", q.baseURL, q.collection),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("qdrant upsert error: status %d", resp.StatusCode)
	}
	return nil
}

func (q *QdrantClient) Search(ctx context.Context, vector []float32, threshold float64, topK int) ([]SearchResult, error) {
	body, _ := json.Marshal(map[string]any{
		"vector":          vector,
		"limit":           topK,
		"score_threshold": threshold,
		"with_payload":    true,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, q.collection),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant search error: status %d", resp.StatusCode)
	}
	var result struct {
		Result []SearchResult `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Result, nil
}
