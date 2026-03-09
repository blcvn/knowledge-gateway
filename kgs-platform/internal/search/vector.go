package search

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"kgs-platform/internal/data"
)

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type DeterministicEmbeddingClient struct {
	vectorSize int
}

func NewDeterministicEmbeddingClient(vectorSize int) *DeterministicEmbeddingClient {
	if vectorSize <= 0 {
		vectorSize = 1536
	}
	return &DeterministicEmbeddingClient{vectorSize: vectorSize}
}

func (c *DeterministicEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	if text == "" {
		return nil, errors.New("empty text")
	}
	size := c.vectorSize
	if size <= 0 {
		size = 1536
	}
	out := make([]float32, size)
	for i := 0; i < size; i++ {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", text, i)))
		v := binary.BigEndian.Uint32(digest[:4])
		out[i] = float32(v%10000)/5000 - 1
	}
	normalizeVector(out)
	return out, nil
}

type VectorSearcher struct {
	qdrant   *data.QdrantClient
	embedder EmbeddingClient
}

func NewVectorSearcher(qdrant *data.QdrantClient, embedder EmbeddingClient) *VectorSearcher {
	return &VectorSearcher{
		qdrant:   qdrant,
		embedder: embedder,
	}
}

func (s *VectorSearcher) Search(ctx context.Context, namespace, query string, topK int) ([]Result, error) {
	if s == nil || s.qdrant == nil || s.embedder == nil {
		return nil, nil
	}
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	collection := collectionName(namespace)
	if err := s.qdrant.EnsureCollection(ctx, collection, len(vector)); err != nil {
		return nil, err
	}
	points, err := s.qdrant.SearchVectors(ctx, collection, vector, topK, 0)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(points))
	for _, point := range points {
		properties := readProperties(point.Payload)
		results = append(results, Result{
			ID:            resolveResultID(point.ID, point.Payload),
			Label:         readPayloadString(point.Payload, "label"),
			Properties:    properties,
			SemanticScore: point.Score,
			Score:         point.Score,
		})
	}
	return results, nil
}

func normalizeVector(vector []float32) {
	if len(vector) == 0 {
		return
	}
	norm := 0.0
	for _, value := range vector {
		norm += float64(value * value)
	}
	if norm == 0 {
		return
	}
	norm = math.Sqrt(norm)
	for i := range vector {
		vector[i] = float32(float64(vector[i]) / norm)
	}
}

func resolveResultID(pointID string, payload map[string]any) string {
	if payload == nil {
		if pointID != "" && pointID != "<nil>" {
			return pointID
		}
		return ""
	}
	if raw, ok := payload["id"]; ok {
		id := fmt.Sprint(raw)
		if id != "" && id != "<nil>" {
			return id
		}
	}
	if pointID != "" && pointID != "<nil>" {
		return pointID
	}
	return ""
}

func readPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	if out, ok := raw.(string); ok {
		return out
	}
	return fmt.Sprint(raw)
}

func readProperties(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	raw, ok := payload["properties"]
	if !ok || raw == nil {
		return payload
	}
	props, ok := raw.(map[string]any)
	if !ok {
		return payload
	}
	return props
}
