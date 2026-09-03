// Package cognee implements Cognee adapter usecases.
//
// kg-service acts as a thin HTTP proxy layer to the Cognee Python service.
// Dataset metadata is tracked locally in PostgreSQL for tenant isolation.
// (MERGE-P2-T2)
//
// Extensions:
//   - CR-COGNEE-001: MemifyUseCase — non-destructive graph enrichment
//   - CR-COGNEE-002: NodeSetsSearchUseCase — tag-based scoped search
//   - CR-COGNEE-003: AddDataPointsUseCase — schema-defined ingestion (zero LLM)
package cognee

import (
	"context"
	"fmt"
	"time"

	cogclient "vnp-memory/services/kg-service/internal/adapter/cognee"
	"vnp-memory/services/kg-service/internal/domain/cognee"
	"vnp-memory/services/kg-service/internal/usecase/port"
)

// DatasetUseCase manages Cognee datasets.
type DatasetUseCase struct {
	client  cogclient.Interface
	store   port.DatasetRepository
	enabled bool
}

// NewDatasetUseCase creates a DatasetUseCase.
func NewDatasetUseCase(client cogclient.Interface, store port.DatasetRepository, enabled bool) *DatasetUseCase {
	return &DatasetUseCase{client: client, store: store, enabled: enabled}
}

func (uc *DatasetUseCase) checkEnabled() error {
	if !uc.enabled {
		return fmt.Errorf("cognee: feature disabled (set COGNEE_ENABLED=true)")
	}
	return nil
}

// CreateDataset creates a dataset in Cognee and tracks it locally.
func (uc *DatasetUseCase) CreateDataset(ctx context.Context, tenantID, name string) (*cognee.Dataset, error) {
	if err := uc.checkEnabled(); err != nil {
		return nil, err
	}
	resp, err := uc.client.CreateDataset(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("cognee create dataset: %w", err)
	}
	ds := &cognee.Dataset{
		ID:        resp.ID,
		Name:      name,
		TenantID:  tenantID,
		Status:    "empty",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return ds, uc.store.Save(ctx, ds)
}

// UploadData uploads data to a Cognee dataset.
func (uc *DatasetUseCase) UploadData(ctx context.Context, datasetID string, item *cognee.DataItem) error {
	if err := uc.checkEnabled(); err != nil {
		return err
	}
	_ = uc.store.UpdateStatus(ctx, datasetID, "uploading")
	if err := uc.client.UploadData(ctx, datasetID, item); err != nil {
		return fmt.Errorf("cognee upload data: %w", err)
	}
	return uc.store.UpdateStatus(ctx, datasetID, "ready")
}

// ListDatasets retrieves all datasets for a tenant.
func (uc *DatasetUseCase) ListDatasets(ctx context.Context, tenantID string) ([]*cognee.Dataset, error) {
	return uc.store.ListByTenant(ctx, tenantID)
}

// CognifyUseCase triggers cognification pipeline.
type CognifyUseCase struct {
	client  cogclient.Interface
	store   port.DatasetRepository
	enabled bool
}

// NewCognifyUseCase creates a CognifyUseCase.
func NewCognifyUseCase(client cogclient.Interface, store port.DatasetRepository, enabled bool) *CognifyUseCase {
	return &CognifyUseCase{client: client, store: store, enabled: enabled}
}

// Cognify triggers the cognification pipeline for a dataset.
// CR-COGNEE-006: Support for PipelineConfig
func (uc *CognifyUseCase) Cognify(ctx context.Context, datasetID string, cfg cognee.PipelineConfig) (*cognee.CognifyJob, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("cognee: feature disabled")
	}
	_ = uc.store.UpdateStatus(ctx, datasetID, "cognifying")
	job, err := uc.client.Cognify(ctx, datasetID, cfg)
	if err != nil {
		_ = uc.store.UpdateStatus(ctx, datasetID, "ready")
		return nil, fmt.Errorf("cognee cognify: %w", err)
	}
	return job, nil
}

// CogneeSearchUseCase handles search via Cognee Python.
type CogneeSearchUseCase struct {
	client  cogclient.Interface
	enabled bool
}

// NewCogneeSearchUseCase creates a CogneeSearchUseCase.
func NewCogneeSearchUseCase(client cogclient.Interface, enabled bool) *CogneeSearchUseCase {
	return &CogneeSearchUseCase{client: client, enabled: enabled}
}

// Search delegates to Cognee Python search API.
// CR-COGNEE-005: Full SearchRequest support
func (uc *CogneeSearchUseCase) Search(ctx context.Context, req cognee.SearchRequest) ([]*cognee.SearchResult, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("cognee: feature disabled")
	}
	results, err := uc.client.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cognee search: %w", err)
	}
	return results, nil
}

// ─── CR-COGNEE-001: MemifyUseCase ───────────────────────────────────────────

// MemifyUseCase triggers non-destructive graph enrichment (Memify).
// Memify derives new facts and re-embeds triplets without rebuilding the graph.
type MemifyUseCase struct {
	client  cogclient.Interface
	store   port.DatasetRepository
	enabled bool
}

// NewMemifyUseCase creates a MemifyUseCase.
func NewMemifyUseCase(client cogclient.Interface, store port.DatasetRepository, enabled bool) *MemifyUseCase {
	return &MemifyUseCase{client: client, store: store, enabled: enabled}
}

// Memify triggers the Memify pipeline for a dataset and returns a pipeline job.
// Returns 202 Accepted semantics — callers should poll GetMemifyStatus.
func (uc *MemifyUseCase) Memify(ctx context.Context, datasetID, tenantID string, cfg cognee.MemifyConfig) (*cognee.MemifyJob, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("cognee: feature disabled")
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	// Default: derive facts and embed triplets
	if !cfg.DeriveFacts && !cfg.EmbedTriplets {
		cfg.DeriveFacts = true
		cfg.EmbedTriplets = true
	}
	job, err := uc.client.Memify(ctx, datasetID, tenantID, cfg)
	if err != nil {
		return nil, fmt.Errorf("cognee memify: %w", err)
	}
	return job, nil
}

// GetMemifyStatus polls the status of a running Memify job.
func (uc *MemifyUseCase) GetMemifyStatus(ctx context.Context, datasetID, pipelineRunID string) (*cognee.MemifyJob, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("cognee: feature disabled")
	}
	status, err := uc.client.GetMemifyStatus(ctx, datasetID, pipelineRunID)
	if err != nil {
		return nil, fmt.Errorf("cognee memify status: %w", err)
	}
	return status, nil
}

// ─── CR-COGNEE-002: NodeSetsSearchUseCase ───────────────────────────────────

// NodeSetsSearchUseCase handles tag-based scoped search via Cognee Python.
// NodeSets filter both Qdrant vector payload and Neo4j labels for fast scoping.
type NodeSetsSearchUseCase struct {
	client  cogclient.Interface
	enabled bool
}

// NewNodeSetsSearchUseCase creates a NodeSetsSearchUseCase.
func NewNodeSetsSearchUseCase(client cogclient.Interface, enabled bool) *NodeSetsSearchUseCase {
	return &NodeSetsSearchUseCase{client: client, enabled: enabled}
}

// SearchWithNodeSets performs a scoped search filtered by the given NodeSet tags.
// Falls back to full dataset search when NodeSets is empty.
func (uc *NodeSetsSearchUseCase) SearchWithNodeSets(ctx context.Context, req cognee.SearchRequest) ([]*cognee.SearchResult, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("cognee: feature disabled")
	}
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	results, err := uc.client.SearchWithNodeSets(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cognee search with node_sets: %w", err)
	}
	return results, nil
}

// ─── CR-COGNEE-003: AddDataPointsUseCase ────────────────────────────────────

// AddDataPointsUseCase handles schema-defined DataPoint ingestion.
// DataPoints bypass LLM entity extraction — zero token cost for structured data.
type AddDataPointsUseCase struct {
	client  cogclient.Interface
	store   port.DatasetRepository
	enabled bool
}

// NewAddDataPointsUseCase creates an AddDataPointsUseCase.
func NewAddDataPointsUseCase(client cogclient.Interface, store port.DatasetRepository, enabled bool) *AddDataPointsUseCase {
	return &AddDataPointsUseCase{client: client, store: store, enabled: enabled}
}

// AddDataPoints ingests structured DataPoints directly into the graph.
// Each DataPoint with a defined Type is mapped to a Neo4j node;
// Relations are mapped to Neo4j edges. Only index_fields are embedded.
func (uc *AddDataPointsUseCase) AddDataPoints(ctx context.Context, req cognee.AddDataPointsRequest) (*cognee.AddDataPointsResponse, error) {
	if !uc.enabled {
		return nil, fmt.Errorf("cognee: feature disabled")
	}
	if len(req.DataPoints) == 0 {
		return nil, fmt.Errorf("at least one data_point is required")
	}
	// Validate: each DataPoint must have a non-empty Type
	for i, dp := range req.DataPoints {
		if dp.Type == "" {
			return nil, fmt.Errorf("data_points[%d]: type is required", i)
		}
		if dp.ID == "" {
			return nil, fmt.Errorf("data_points[%d]: id is required", i)
		}
	}
	resp, err := uc.client.AddDataPoints(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cognee add_data_points: %w", err)
	}
	return resp, nil
}
