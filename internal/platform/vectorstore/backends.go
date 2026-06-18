package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type QdrantConfig struct {
	Endpoint   string
	Collection string
	Client     *http.Client
}

type delegatedVectorAdapter struct {
	delegate VectorAdapter
}

func (a delegatedVectorAdapter) Upsert(ctx context.Context, doc VectorDocument) error {
	return a.delegate.Upsert(ctx, doc)
}

func (a delegatedVectorAdapter) Delete(ctx context.Context, nodeID string) error {
	return a.delegate.Delete(ctx, nodeID)
}

func (a delegatedVectorAdapter) ANN(ctx context.Context, query []float64, filter VectorFilter, opts ANNOptions) ([]VectorResult, error) {
	return a.delegate.ANN(ctx, query, filter, opts)
}

func (a delegatedVectorAdapter) Snapshot(ctx context.Context) ([]VectorDocument, error) {
	return a.delegate.Snapshot(ctx)
}

func (a delegatedVectorAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	return a.delegate.ReadSyncVersion(ctx, entityID)
}

type QdrantVectorAdapter struct {
	Client     *http.Client
	Endpoint   string
	Collection string
}

func NewQdrantVectorAdapter(cfg QdrantConfig) *QdrantVectorAdapter {
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &QdrantVectorAdapter{
		Client:     client,
		Endpoint:   cfg.Endpoint,
		Collection: cfg.Collection,
	}
}

func (a *QdrantVectorAdapter) Upsert(ctx context.Context, doc VectorDocument) error {
	if a == nil || a.Client == nil || strings.TrimSpace(a.Endpoint) == "" || strings.TrimSpace(a.Collection) == "" {
		return errors.New("qdrant vector adapter is not configured")
	}
	body := map[string]any{
		"points": []map[string]any{
			{
				"id":     doc.NodeID,
				"vector": doc.Embedding,
				"payload": map[string]any{
					"node_id":          doc.NodeID,
					"node_type":        doc.NodeType,
					"domain_id":        doc.DomainID,
					"owner_tenant_id":  doc.OwnerTenantID,
					"owner_app_id":     doc.OwnerAppID,
					"acl_visible_to":   append([]string(nil), doc.ACLVisibleTo...),
					"is_deleted":       doc.IsDeleted,
					"status_value":     doc.StatusValue,
					"authority_score":  doc.AuthorityScore,
					"_kg_sync_version": doc.SyncVersion,
					"domain_props":     cloneMap(doc.DomainProps),
					"embedding":        append([]float64(nil), doc.Embedding...),
				},
			},
		},
	}
	return a.do(ctx, http.MethodPut, a.pointsPath(), body, nil)
}

func (a *QdrantVectorAdapter) Delete(ctx context.Context, nodeID string) error {
	if a == nil || a.Client == nil || strings.TrimSpace(a.Endpoint) == "" || strings.TrimSpace(a.Collection) == "" {
		return errors.New("qdrant vector adapter is not configured")
	}
	body := map[string]any{
		"points": []string{nodeID},
	}
	return a.do(ctx, http.MethodPost, a.pointsDeletePath(), body, nil)
}

func (a *QdrantVectorAdapter) ANN(ctx context.Context, query []float64, filter VectorFilter, opts ANNOptions) ([]VectorResult, error) {
	if a == nil || a.Client == nil || strings.TrimSpace(a.Endpoint) == "" || strings.TrimSpace(a.Collection) == "" {
		return nil, errors.New("qdrant vector adapter is not configured")
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	body := map[string]any{
		"vector":       query,
		"filter":       qdrantFilter(filter),
		"limit":        opts.TopK,
		"with_payload": true,
		"with_vector":  true,
	}
	var points []qdrantPoint
	if err := a.do(ctx, http.MethodPost, a.searchPath(), body, &points); err != nil {
		return nil, err
	}
	results := make([]VectorResult, 0, len(points))
	for _, point := range points {
		doc := qdrantPointToDocument(point)
		results = append(results, VectorResult{Document: doc, Score: point.Score})
	}
	return results, nil
}

func (a *QdrantVectorAdapter) Snapshot(ctx context.Context) ([]VectorDocument, error) {
	if a == nil || a.Client == nil || strings.TrimSpace(a.Endpoint) == "" || strings.TrimSpace(a.Collection) == "" {
		return nil, errors.New("qdrant vector adapter is not configured")
	}
	var result qdrantScrollResult
	var docs []VectorDocument
	var offset any
	for {
		body := map[string]any{
			"limit":        128,
			"with_payload": true,
			"with_vector":  true,
		}
		if offset != nil {
			body["offset"] = offset
		}
		result = qdrantScrollResult{}
		if err := a.do(ctx, http.MethodPost, a.scrollPath(), body, &result); err != nil {
			return nil, err
		}
		for _, point := range result.Result.Points {
			docs = append(docs, qdrantPointToDocument(point))
		}
		if result.Result.NextPageOffset == nil {
			return docs, nil
		}
		offset = result.Result.NextPageOffset
	}
}

func (a *QdrantVectorAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	if a == nil || a.Client == nil || strings.TrimSpace(a.Endpoint) == "" || strings.TrimSpace(a.Collection) == "" {
		return 0, errors.New("qdrant vector adapter is not configured")
	}
	var result qdrantPointResponse
	err := a.do(ctx, http.MethodGet, a.pointPath(entityID), nil, &result)
	if err != nil {
		if errors.Is(err, errQdrantNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return qdrantPointToDocument(result.Result).SyncVersion, nil
}

func (a *QdrantVectorAdapter) pointsPath() string {
	return fmt.Sprintf("/collections/%s/points?wait=true", url.PathEscape(a.Collection))
}

func (a *QdrantVectorAdapter) pointsDeletePath() string {
	return fmt.Sprintf("/collections/%s/points/delete?wait=true", url.PathEscape(a.Collection))
}

func (a *QdrantVectorAdapter) searchPath() string {
	return fmt.Sprintf("/collections/%s/points/search", url.PathEscape(a.Collection))
}

func (a *QdrantVectorAdapter) scrollPath() string {
	return fmt.Sprintf("/collections/%s/points/scroll", url.PathEscape(a.Collection))
}

func (a *QdrantVectorAdapter) pointPath(entityID string) string {
	return fmt.Sprintf("/collections/%s/points/%s?with_payload=true&with_vector=true", url.PathEscape(a.Collection), url.PathEscape(entityID))
}

func (a *QdrantVectorAdapter) do(ctx context.Context, method, targetPath string, body any, dst any) error {
	base := strings.TrimRight(strings.TrimSpace(a.Endpoint), "/")
	if base == "" {
		return errors.New("qdrant vector adapter is not configured")
	}
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, base+targetPath, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errQdrantNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant request failed: %s", resp.Status)
	}
	if dst == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

type qdrantPoint struct {
	ID      any            `json:"id"`
	Version int64          `json:"version"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
	Vector  any            `json:"vector"`
}

type qdrantPointResponse struct {
	Result qdrantPoint `json:"result"`
}

type qdrantScrollResult struct {
	Result struct {
		Points         []qdrantPoint `json:"points"`
		NextPageOffset any           `json:"next_page_offset"`
	} `json:"result"`
}

var errQdrantNotFound = errors.New("qdrant point not found")

func qdrantFilter(filter VectorFilter) map[string]any {
	must := make([]any, 0, 5)
	if len(filter.DomainIDs) > 0 {
		must = append(must, qdrantAnyMatch("domain_id", filter.DomainIDs))
	}
	if len(filter.NodeTypes) > 0 {
		must = append(must, qdrantAnyMatch("node_type", filter.NodeTypes))
	}
	if len(filter.OwnerTenantIDs) > 0 {
		must = append(must, qdrantAnyMatch("owner_tenant_id", filter.OwnerTenantIDs))
	}
	if len(filter.OwnerAppIDs) > 0 {
		must = append(must, qdrantAnyMatch("owner_app_id", filter.OwnerAppIDs))
	}
	if len(filter.ACLVisibleTo) > 0 {
		must = append(must, qdrantAnyMatch("acl_visible_to", filter.ACLVisibleTo))
	}
	if len(must) == 0 {
		return nil
	}
	return map[string]any{"must": must}
}

func qdrantAnyMatch(key string, values []string) map[string]any {
	return map[string]any{
		"key": key,
		"match": map[string]any{
			"any": values,
		},
	}
}

func qdrantPointToDocument(point qdrantPoint) VectorDocument {
	doc := VectorDocument{}
	if len(point.Payload) > 0 {
		if raw, err := json.Marshal(point.Payload); err == nil {
			_ = json.Unmarshal(raw, &doc)
		}
	}
	if len(doc.Embedding) == 0 {
		doc.Embedding = qdrantVector(point.Vector)
	}
	if doc.DomainProps == nil {
		doc.DomainProps = map[string]any{}
	}
	if doc.ACLVisibleTo == nil {
		doc.ACLVisibleTo = []string{}
	}
	return doc
}

func qdrantVector(value any) []float64 {
	switch v := value.(type) {
	case []float64:
		return append([]float64(nil), v...)
	case []any:
		result := make([]float64, 0, len(v))
		for _, item := range v {
			if f, ok := item.(float64); ok {
				result = append(result, f)
			}
		}
		return result
	case map[string]any:
		for _, raw := range v {
			if nested, ok := raw.([]any); ok {
				return qdrantVector(nested)
			}
		}
	}
	return nil
}
