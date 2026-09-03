// Package usecase implements the index management use case for graphiti-admin.
// SOL-007: Admin Service & Observability Stack (CR-GR-007)
package usecase

import (
	"context"
	"fmt"

	"vnp-memory/services/graphiti-admin/internal/usecase/port"
)

// IndexManagementUseCase manages Neo4j vector + fulltext indices.
type IndexManagementUseCase struct {
	storePort port.StorePort
}

// NewIndexManagementUseCase constructs the use case.
func NewIndexManagementUseCase(storePort port.StorePort) *IndexManagementUseCase {
	return &IndexManagementUseCase{storePort: storePort}
}

// BuildIndicesAndConstraints creates Neo4j vector indices and uniqueness constraints.
func (uc *IndexManagementUseCase) BuildIndicesAndConstraints(ctx context.Context) error {
	if err := uc.storePort.BuildIndicesAndConstraints(ctx); err != nil {
		return fmt.Errorf("build indices: %w", err)
	}
	return nil
}

// DeleteAllIndexes removes all non-constraint indices (useful for reindex).
func (uc *IndexManagementUseCase) DeleteAllIndexes(ctx context.Context) error {
	if err := uc.storePort.DeleteAllIndexes(ctx); err != nil {
		return fmt.Errorf("delete indexes: %w", err)
	}
	return nil
}
