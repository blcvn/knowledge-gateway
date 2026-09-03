package usecase

import (
	"context"

	"vnp-memory/services/graphiti-store/domain"
)

// IndexServiceImpl implements port.IndexService.
type IndexServiceImpl struct {
	driver domain.GraphDriver
	dim    int
}

// NewIndexService creates an IndexService backed by the given graph driver.
func NewIndexService(driver domain.GraphDriver, vectorDim int) *IndexServiceImpl {
	return &IndexServiceImpl{driver: driver, dim: vectorDim}
}

// BuildIndices creates standard indexes for the graph.
func (s *IndexServiceImpl) BuildIndices(ctx context.Context, groupID string) error {
	defs := domain.DefaultIndexes(s.dim)
	return s.driver.BuildIndices(ctx, groupID, defs)
}

// DropIndices removes standard indexes for the graph.
func (s *IndexServiceImpl) DropIndices(ctx context.Context, groupID string) error {
	return s.driver.DropIndices(ctx, groupID)
}

// ListIndices returns definitions of standard indexes.
func (s *IndexServiceImpl) ListIndices(ctx context.Context) ([]domain.IndexDefinition, error) {
	return s.driver.ListIndices(ctx)
}
