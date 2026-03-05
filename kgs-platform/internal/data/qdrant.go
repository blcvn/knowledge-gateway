package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kgs-platform/internal/conf"
	"kgs-platform/internal/observability"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/attribute"
)

const (
	defaultQdrantPort       = 6333
	defaultQdrantVectorSize = 1536
)

type VectorPoint struct {
	ID      string
	Vector  []float32
	Payload map[string]any
}

type ScoredPoint struct {
	ID      string
	Score   float64
	Payload map[string]any
}

type QdrantClient struct {
	baseURL           string
	defaultCollection string
	defaultVectorSize int
	httpClient        *http.Client
	log               *log.Helper
}

func NewQdrantClientFromConfig(cfg *conf.Data_Qdrant, logger *log.Helper) (*QdrantClient, error) {
	if cfg == nil {
		return nil, nil
	}
	host := strings.TrimSpace(cfg.GetHost())
	if host == "" {
		return nil, nil
	}
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	base, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant host: %w", err)
	}
	port := cfg.GetPort()
	if port <= 0 {
		port = defaultQdrantPort
	}
	if base.Port() == "" {
		base.Host = fmt.Sprintf("%s:%d", base.Hostname(), port)
	}
	vectorSize := int(cfg.GetVectorSize())
	if vectorSize <= 0 {
		vectorSize = defaultQdrantVectorSize
	}
	return &QdrantClient{
		baseURL:           strings.TrimSuffix(base.String(), "/"),
		defaultCollection: cfg.GetCollection(),
		defaultVectorSize: vectorSize,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
		log: logger,
	}, nil
}

func (c *QdrantClient) EnsureCollection(ctx context.Context, collection string, vectorSize int) error {
	if c == nil {
		return nil
	}
	collection = c.normalizeCollection(collection)
	if collection == "" {
		return nil
	}
	if vectorSize <= 0 {
		vectorSize = c.defaultVectorSize
	}
	body := map[string]any{
		"vectors": map[string]any{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}
	return c.doJSON(ctx, http.MethodPut, "/collections/"+url.PathEscape(collection), body, nil)
}

func (c *QdrantClient) UpsertVectors(ctx context.Context, collection string, points []VectorPoint) error {
	if c == nil || len(points) == 0 {
		return nil
	}
	collection = c.normalizeCollection(collection)
	if collection == "" {
		return nil
	}
	upsertPoints := make([]map[string]any, 0, len(points))
	for _, point := range points {
		if point.ID == "" || len(point.Vector) == 0 {
			continue
		}
		upsertPoints = append(upsertPoints, map[string]any{
			"id":      point.ID,
			"vector":  point.Vector,
			"payload": point.Payload,
		})
	}
	if len(upsertPoints) == 0 {
		return nil
	}
	body := map[string]any{"points": upsertPoints}
	return c.doJSON(ctx, http.MethodPut, "/collections/"+url.PathEscape(collection)+"/points", body, nil)
}

func (c *QdrantClient) SearchVectors(ctx context.Context, collection string, vector []float32, limit int, scoreThreshold float64) ([]ScoredPoint, error) {
	if c == nil || len(vector) == 0 {
		return nil, nil
	}
	collection = c.normalizeCollection(collection)
	if collection == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	body := map[string]any{
		"vector":          vector,
		"limit":           limit,
		"score_threshold": scoreThreshold,
		"with_payload":    true,
		"with_vector":     false,
	}
	var out []qdrantPoint
	err := c.doJSON(ctx, http.MethodPost, "/collections/"+url.PathEscape(collection)+"/points/search", body, &out)
	if err != nil {
		return nil, err
	}
	return mapScoredPoints(out), nil
}

func (c *QdrantClient) BatchSearch(ctx context.Context, collection string, vectors [][]float32, limit int) ([][]ScoredPoint, error) {
	if c == nil || len(vectors) == 0 {
		return nil, nil
	}
	collection = c.normalizeCollection(collection)
	if collection == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	searches := make([]map[string]any, 0, len(vectors))
	for _, vector := range vectors {
		if len(vector) == 0 {
			continue
		}
		searches = append(searches, map[string]any{
			"vector":       vector,
			"limit":        limit,
			"with_payload": true,
			"with_vector":  false,
		})
	}
	if len(searches) == 0 {
		return nil, nil
	}
	body := map[string]any{"searches": searches}
	var out [][]qdrantPoint
	err := c.doJSON(ctx, http.MethodPost, "/collections/"+url.PathEscape(collection)+"/points/search/batch", body, &out)
	if err != nil {
		return nil, err
	}
	results := make([][]ScoredPoint, 0, len(out))
	for _, list := range out {
		results = append(results, mapScoredPoints(list))
	}
	return results, nil
}

func (c *QdrantClient) DeleteVectors(ctx context.Context, collection string, ids []string) error {
	if c == nil || len(ids) == 0 {
		return nil
	}
	collection = c.normalizeCollection(collection)
	if collection == "" {
		return nil
	}
	body := map[string]any{
		"points": ids,
	}
	return c.doJSON(ctx, http.MethodPost, "/collections/"+url.PathEscape(collection)+"/points/delete", body, nil)
}

func (c *QdrantClient) Ping(ctx context.Context) error {
	if c == nil {
		return nil
	}
	return c.doJSON(ctx, http.MethodGet, "/collections", nil, nil)
}

type qdrantPoint struct {
	ID      any            `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
}

type qdrantEnvelope struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
	Time   float64         `json:"time"`
	Error  string          `json:"error"`
}

func (c *QdrantClient) doJSON(ctx context.Context, method, path string, reqBody any, out any) error {
	traceCtx, span := observability.StartDependencySpan(ctx, "qdrant", "qdrant."+strings.ToLower(method), attribute.String("http.path", path))
	defer span.End()

	var bodyReader io.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequestWithContext(traceCtx, method, c.baseURL+path, bodyReader)
	if err != nil {
		observability.RecordSpanError(span, err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		observability.RecordSpanError(span, err)
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		observability.RecordSpanError(span, err)
		return err
	}
	if resp.StatusCode >= 400 {
		err := fmt.Errorf("qdrant %s %s failed: status=%d body=%s", method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
		observability.RecordSpanError(span, err)
		return err
	}
	if out == nil {
		return nil
	}
	var env qdrantEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		observability.RecordSpanError(span, err)
		return err
	}
	if env.Error != "" {
		err := fmt.Errorf("qdrant error: %s", env.Error)
		observability.RecordSpanError(span, err)
		return err
	}
	if len(env.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Result, out); err != nil {
		observability.RecordSpanError(span, err)
		return err
	}
	return nil
}

func mapScoredPoints(points []qdrantPoint) []ScoredPoint {
	out := make([]ScoredPoint, 0, len(points))
	for _, point := range points {
		out = append(out, ScoredPoint{
			ID:      fmt.Sprint(point.ID),
			Score:   point.Score,
			Payload: point.Payload,
		})
	}
	return out
}

func (c *QdrantClient) normalizeCollection(collection string) string {
	trimmed := strings.TrimSpace(collection)
	if trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(c.defaultCollection)
}
