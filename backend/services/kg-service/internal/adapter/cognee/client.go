// Package cognee implements the CogneeHTTPClient adapter.
//
// This is an HTTP client proxy to the Cognee Python service.
// (MERGE-P2-T2)
package cognee

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"vnp-memory/services/kg-service/internal/domain/cognee"
)

// HTTPClient implements port.CogneeClient via HTTP calls to Cognee Python.
type HTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewHTTPClient creates a Cognee HTTP client.
func NewHTTPClient(baseURL, apiKey string, timeoutSec int) *HTTPClient {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &HTTPClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

// CreateDataset calls POST /api/v1/datasets on Cognee Python.
func (c *HTTPClient) CreateDataset(ctx context.Context, name string) (*cognee.DatasetResponse, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	resp, err := c.do(ctx, "POST", "/api/v1/datasets", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result cognee.DatasetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode dataset response: %w", err)
	}
	return &result, nil
}

// UploadData calls POST /api/v1/datasets/{id}/data on Cognee Python.
func (c *HTTPClient) UploadData(ctx context.Context, datasetID string, item *cognee.DataItem) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// If content type is text, send as field
	if item.ContentType == "text" && len(item.Content) > 0 {
		fw, _ := writer.CreateFormField("data")
		_, _ = fw.Write(item.Content)
	} else if item.URI != "" {
		fw, _ := writer.CreateFormField("uri")
		_, _ = fw.Write([]byte(item.URI))
	}
	_ = writer.Close()

	resp, err := c.do(ctx, "POST",
		fmt.Sprintf("/api/v1/datasets/%s/data", datasetID),
		writer.FormDataContentType(), &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// Cognify calls POST /api/v1/datasets/{id}/cognify on Cognee Python.
// CR-COGNEE-006: Custom Pipelines via PipelineConfig
func (c *HTTPClient) Cognify(ctx context.Context, datasetID string, cfg cognee.PipelineConfig) (*cognee.CognifyJob, error) {
	body, _ := json.Marshal(cfg)
	resp, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/datasets/%s/cognify", datasetID), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	job := &cognee.CognifyJob{
		DatasetID: datasetID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	// Try to decode job ID from response
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
		if id, ok := raw["job_id"].(string); ok {
			job.JobID = id
		}
	}
	if job.JobID == "" {
		job.JobID = fmt.Sprintf("cognify_%s_%d", datasetID, time.Now().UnixNano())
	}
	return job, nil
}

// Search calls POST /api/v1/search on Cognee Python.
// CR-COGNEE-005: Full SearchRequest is passed (including feedback/interaction flags).
func (c *HTTPClient) Search(ctx context.Context, req cognee.SearchRequest) ([]*cognee.SearchResult, error) {
	if req.SearchType == "" {
		req.SearchType = "GRAPH_COMPLETION"
	}
	body, _ := json.Marshal(req)
	resp, err := c.do(ctx, "POST", "/api/v1/search", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []*cognee.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		// Cognee may return different shapes — try array of strings
		return results, nil
	}
	return results, nil
}

// Memify calls POST /api/v1/datasets/{id}/memify on Cognee Python.
// CR-COGNEE-001: Non-destructive graph enrichment (derive facts, embed triplets).
func (c *HTTPClient) Memify(ctx context.Context, datasetID, tenantID string, cfg cognee.MemifyConfig) (*cognee.MemifyJob, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	payload := map[string]any{
		"tenant_id":      tenantID,
		"derive_facts":   cfg.DeriveFacts,
		"embed_triplets": cfg.EmbedTriplets,
		"batch_size":     cfg.BatchSize,
	}
	body, _ := json.Marshal(payload)
	resp, err := c.do(ctx, "POST",
		fmt.Sprintf("/api/v1/datasets/%s/memify", datasetID),
		"application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	job := &cognee.MemifyJob{
		DatasetID:     datasetID,
		TenantID:      tenantID,
		Status:        "QUEUED",
		DeriveFacts:   cfg.DeriveFacts,
		EmbedTriplets: cfg.EmbedTriplets,
		BatchSize:     cfg.BatchSize,
		StartedAt:     time.Now(),
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
		if id, ok := raw["pipeline_run_id"].(string); ok {
			job.PipelineRunID = id
		}
		if status, ok := raw["status"].(string); ok {
			job.Status = status
		}
	}
	if job.PipelineRunID == "" {
		job.PipelineRunID = fmt.Sprintf("memify_%s_%d", datasetID, time.Now().UnixNano())
	}
	return job, nil
}

// GetMemifyStatus calls GET /api/v1/datasets/{id}/memify/status on Cognee Python.
// CR-COGNEE-001
func (c *HTTPClient) GetMemifyStatus(ctx context.Context, datasetID, pipelineRunID string) (*cognee.MemifyJob, error) {
	path := fmt.Sprintf("/api/v1/datasets/%s/memify/status?pipeline_run_id=%s", datasetID, pipelineRunID)
	resp, err := c.do(ctx, "GET", path, "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	job := &cognee.MemifyJob{
		DatasetID:     datasetID,
		PipelineRunID: pipelineRunID,
		Status:        "UNKNOWN",
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
		if status, ok := raw["status"].(string); ok {
			job.Status = status
		}
		if nn, ok := raw["new_nodes"].(float64); ok {
			job.NewNodes = int(nn)
		}
		if ne, ok := raw["new_edges"].(float64); ok {
			job.NewEdges = int(ne)
		}
	}
	return job, nil
}

// SearchWithNodeSets calls POST /api/v1/search with NodeSet filters.
// CR-COGNEE-002: Scoped search by node tags (faster than full dataset scan).
func (c *HTTPClient) SearchWithNodeSets(ctx context.Context, req cognee.SearchRequest) ([]*cognee.SearchResult, error) {
	searchType := req.SearchType
	if searchType == "" {
		searchType = "GRAPH_COMPLETION"
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}
	payload := map[string]any{
		"query":      req.Query,
		"searchType": searchType,
		"top_k":      topK,
	}
	if len(req.NodeSets) > 0 {
		payload["node_sets"] = req.NodeSets
	}
	if req.DatasetID != "" {
		payload["dataset_id"] = req.DatasetID
	}
	body, _ := json.Marshal(payload)
	resp, err := c.do(ctx, "POST", "/api/v1/search", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []*cognee.SearchResult
	_ = json.NewDecoder(resp.Body).Decode(&results)
	return results, nil
}

// AddDataPoints calls POST /api/v1/datasets/{id}/datapoints on Cognee Python.
// CR-COGNEE-003: Schema-defined ingestion that bypasses LLM entity extraction.
func (c *HTTPClient) AddDataPoints(ctx context.Context, req cognee.AddDataPointsRequest) (*cognee.AddDataPointsResponse, error) {
	if req.DatasetID == "" {
		return nil, fmt.Errorf("dataset_id is required for AddDataPoints")
	}
	body, _ := json.Marshal(req)
	resp, err := c.do(ctx, "POST",
		fmt.Sprintf("/api/v1/datasets/%s/datapoints", req.DatasetID),
		"application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result cognee.AddDataPointsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Fallback: acknowledge all submitted DataPoint IDs.
		ids := make([]string, len(req.DataPoints))
		for i, dp := range req.DataPoints {
			ids[i] = dp.ID
		}
		return &cognee.AddDataPointsResponse{Accepted: len(req.DataPoints), IDs: ids}, nil
	}
	return &result, nil
}

// do executes an HTTP request against the Cognee Python service.
func (c *HTTPClient) do(ctx context.Context, method, path, contentType string, body io.Reader) (*http.Response, error) {
	if body == nil {
		body = http.NoBody
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cognee http %s %s: %w", method, path, err)
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("cognee http %s %s: status %d: %s", method, path, resp.StatusCode, string(b))
	}
	return resp, nil
}
