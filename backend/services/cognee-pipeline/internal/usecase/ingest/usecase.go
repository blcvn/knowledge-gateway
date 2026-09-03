package ingest

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/domain/ingestion"
	"github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/usecase/port"
)

type ingestionUseCase struct {
	datasetRepo port.DatasetRepository
	cognifyUC   port.CognifyUseCase
}

// NewIngestionUseCase creates a new ingestion use case
func NewIngestionUseCase(repo port.DatasetRepository, cognifyUC port.CognifyUseCase) port.IngestionUseCase {
	return &ingestionUseCase{
		datasetRepo: repo,
		cognifyUC:   cognifyUC,
	}
}

func (uc *ingestionUseCase) CreateDataset(ctx context.Context, tenantID uuid.UUID, name, desc string) (*ingestion.Dataset, error) {
	ds := &ingestion.Dataset{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: desc,
		Status:      "active",
	}
	err := uc.datasetRepo.CreateDataset(ctx, ds)
	return ds, err
}

func (uc *ingestionUseCase) AddDataItem(ctx context.Context, datasetID uuid.UUID, sourceType, sourceURI, mimeType string) (*ingestion.DataItem, error) {
	item := &ingestion.DataItem{
		ID:         uuid.New(),
		DatasetID:  datasetID,
		SourceType: sourceType,
		SourceURI:  sourceURI,
		MimeType:   mimeType,
	}
	err := uc.datasetRepo.CreateDataItem(ctx, item)
	if err != nil {
		return nil, err
	}

	// Wait, we need the tenantID to trigger cognify. Let's get the dataset first.
	ds, err := uc.datasetRepo.FindDatasetByID(ctx, datasetID)
	if err == nil && ds != nil {
		// Directly trigger cognify pipeline via local function call (instead of NATS)
		// We launch this in a goroutine so it doesn't block the ingestion response.
		// NOTE: In production, pass a background context, not the request context which might be canceled.
		go func(bgCtx context.Context, tenant uuid.UUID, dsId uuid.UUID) {
			_, _ = uc.cognifyUC.StartCognify(bgCtx, tenant, dsId)
		}(context.Background(), ds.TenantID, datasetID)
	}

	return item, nil
}

func (uc *ingestionUseCase) GetDataset(ctx context.Context, id uuid.UUID) (*ingestion.Dataset, error) {
	return uc.datasetRepo.FindDatasetByID(ctx, id)
}

func (uc *ingestionUseCase) ListDatasets(ctx context.Context, tenantID uuid.UUID) ([]*ingestion.Dataset, error) {
	return uc.datasetRepo.ListDatasetsByTenant(ctx, tenantID)
}

func (uc *ingestionUseCase) DeleteDataset(ctx context.Context, id uuid.UUID) error {
	return uc.datasetRepo.DeleteDataset(ctx, id)
}
