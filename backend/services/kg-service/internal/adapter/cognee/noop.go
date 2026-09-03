// Package cognee — CogneeClient interface and noop stub.
package cognee

import (
	"context"

	cogdomain "vnp-memory/services/kg-service/internal/domain/cognee"
)

// Interface mirrors port.CogneeClient for adapter implementations.
// Extensions:
//   - CR-COGNEE-001: Memify — non-destructive graph enrichment
//   - CR-COGNEE-002: SearchWithNodeSets — scoped search by node tags
//   - CR-COGNEE-003: AddDataPoints — schema-defined ingestion without LLM
type Interface interface {
	CreateDataset(ctx context.Context, name string) (*cogdomain.DatasetResponse, error)
	UploadData(ctx context.Context, datasetID string, item *cogdomain.DataItem) error
	Cognify(ctx context.Context, datasetID string, cfg cogdomain.PipelineConfig) (*cogdomain.CognifyJob, error)
	Search(ctx context.Context, req cogdomain.SearchRequest) ([]*cogdomain.SearchResult, error)
	// CR-COGNEE-001: Non-destructive graph enrichment
	Memify(ctx context.Context, datasetID, tenantID string, cfg cogdomain.MemifyConfig) (*cogdomain.MemifyJob, error)
	GetMemifyStatus(ctx context.Context, datasetID, pipelineRunID string) (*cogdomain.MemifyJob, error)
	// CR-COGNEE-002: NodeSet-scoped search
	SearchWithNodeSets(ctx context.Context, req cogdomain.SearchRequest) ([]*cogdomain.SearchResult, error)
	// CR-COGNEE-003: Schema-defined DataPoint ingestion (zero LLM tokens)
	AddDataPoints(ctx context.Context, req cogdomain.AddDataPointsRequest) (*cogdomain.AddDataPointsResponse, error)
}

// NoopClient is a no-op implementation used when Cognee is disabled.
type NoopClient struct{}

func (c *NoopClient) CreateDataset(_ context.Context, name string) (*cogdomain.DatasetResponse, error) {
	return &cogdomain.DatasetResponse{ID: "noop", Name: name}, nil
}
func (c *NoopClient) UploadData(_ context.Context, _ string, _ *cogdomain.DataItem) error {
	return nil
}
func (c *NoopClient) Cognify(_ context.Context, datasetID string, _ cogdomain.PipelineConfig) (*cogdomain.CognifyJob, error) {
	return &cogdomain.CognifyJob{JobID: "noop", DatasetID: datasetID, Status: "completed"}, nil
}
func (c *NoopClient) Search(_ context.Context, _ cogdomain.SearchRequest) ([]*cogdomain.SearchResult, error) {
	return nil, nil
}

// CR-COGNEE-001: Memify no-op — returns a queued job immediately.
func (c *NoopClient) Memify(_ context.Context, datasetID, tenantID string, cfg cogdomain.MemifyConfig) (*cogdomain.MemifyJob, error) {
	return &cogdomain.MemifyJob{
		PipelineRunID: "noop-memify",
		DatasetID:     datasetID,
		TenantID:      tenantID,
		Status:        "QUEUED",
		DeriveFacts:   cfg.DeriveFacts,
		EmbedTriplets: cfg.EmbedTriplets,
		BatchSize:     cfg.BatchSize,
	}, nil
}

// CR-COGNEE-001: GetMemifyStatus no-op — always reports completed.
func (c *NoopClient) GetMemifyStatus(_ context.Context, datasetID, pipelineRunID string) (*cogdomain.MemifyJob, error) {
	return &cogdomain.MemifyJob{
		PipelineRunID: pipelineRunID,
		DatasetID:     datasetID,
		Status:        "COMPLETED",
	}, nil
}

// CR-COGNEE-002: SearchWithNodeSets no-op.
func (c *NoopClient) SearchWithNodeSets(_ context.Context, _ cogdomain.SearchRequest) ([]*cogdomain.SearchResult, error) {
	return nil, nil
}

// CR-COGNEE-003: AddDataPoints no-op — acknowledges without persisting.
func (c *NoopClient) AddDataPoints(_ context.Context, req cogdomain.AddDataPointsRequest) (*cogdomain.AddDataPointsResponse, error) {
	ids := make([]string, len(req.DataPoints))
	for i, dp := range req.DataPoints {
		ids[i] = dp.ID
	}
	return &cogdomain.AddDataPointsResponse{Accepted: len(req.DataPoints), IDs: ids}, nil
}
