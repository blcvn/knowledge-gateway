package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/usecase/port"
)

// ManageDatasetUseCase handles dataset CRUD lifecycle operations.
type ManageDatasetUseCase struct {
	datasetRepo port.DatasetRepository
	itemRepo    port.DataItemRepository
	fileStorage port.FileStorage
	logger      *slog.Logger
}

// NewManageDatasetUseCase constructs the dataset management use case.
func NewManageDatasetUseCase(
	datasetRepo port.DatasetRepository,
	itemRepo port.DataItemRepository,
	fileStorage port.FileStorage,
	logger *slog.Logger,
) *ManageDatasetUseCase {
	return &ManageDatasetUseCase{
		datasetRepo: datasetRepo,
		itemRepo:    itemRepo,
		fileStorage: fileStorage,
		logger:      logger.With("usecase", "manage_dataset"),
	}
}

// Create creates a new dataset for a tenant. Returns error if name is duplicate.
func (uc *ManageDatasetUseCase) Create(ctx context.Context, tenantID, name, description string) (*domain.Dataset, error) {
	// Check for duplicate name
	exists, err := uc.datasetRepo.ExistsByName(ctx, tenantID, name)
	if err != nil {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}
	if exists {
		return nil, domain.ErrDuplicateDataset
	}

	ds, err := domain.NewDataset(tenantID, name, description)
	if err != nil {
		return nil, fmt.Errorf("create dataset: %w", err)
	}

	if err := uc.datasetRepo.Create(ctx, ds); err != nil {
		return nil, fmt.Errorf("persist dataset: %w", err)
	}

	uc.logger.Info("dataset created", "dataset_id", ds.ID, "name", name, "tenant_id", tenantID)
	return ds, nil
}

// Get retrieves a dataset by ID, scoped to the given tenant.
func (uc *ManageDatasetUseCase) Get(ctx context.Context, tenantID string, id uuid.UUID) (*domain.Dataset, error) {
	ds, err := uc.datasetRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	if ds == nil {
		return nil, domain.ErrDatasetNotFound
	}
	return ds, nil
}

// List returns datasets for a tenant with cursor-based pagination.
func (uc *ManageDatasetUseCase) List(ctx context.Context, tenantID, cursor string, limit int) ([]*domain.Dataset, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return uc.datasetRepo.List(ctx, tenantID, cursor, limit)
}

// Delete removes a dataset and all its data items (cascade), including files from storage.
func (uc *ManageDatasetUseCase) Delete(ctx context.Context, tenantID string, id uuid.UUID) error {
	// Verify dataset exists
	ds, err := uc.datasetRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get dataset: %w", err)
	}
	if ds == nil {
		return domain.ErrDatasetNotFound
	}

	// Delete files from object storage (best-effort, log errors)
	storagePrefix := fmt.Sprintf("%s/%s/", tenantID, id.String())
	if err := uc.fileStorage.DeletePrefix(ctx, storagePrefix); err != nil {
		uc.logger.Error("failed to delete files from storage", "prefix", storagePrefix, "error", err)
	}

	// Delete data items (DB cascade will handle this if FK is set, but explicit is safer)
	if err := uc.itemRepo.DeleteByDataset(ctx, id); err != nil {
		return fmt.Errorf("delete data items: %w", err)
	}

	// Delete dataset
	if err := uc.datasetRepo.Delete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete dataset: %w", err)
	}

	uc.logger.Info("dataset deleted", "dataset_id", id, "tenant_id", tenantID)
	return nil
}

// GetStatus returns dataset status with item statistics.
func (uc *ManageDatasetUseCase) GetStatus(ctx context.Context, tenantID string, id uuid.UUID) (*dto.DatasetStatusResponse, error) {
	ds, err := uc.datasetRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	if ds == nil {
		return nil, domain.ErrDatasetNotFound
	}

	return &dto.DatasetStatusResponse{
		ID:             ds.ID,
		Name:           ds.Name,
		Status:         ds.Status,
		FileCount:      ds.FileCount,
		TotalSizeBytes: ds.TotalSizeBytes,
		CreatedAt:      ds.CreatedAt,
		UpdatedAt:      ds.UpdatedAt,
	}, nil
}
