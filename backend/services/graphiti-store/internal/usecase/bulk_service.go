package usecase

import (
	"context"

	"vnp-memory/services/graphiti-store/domain"
	"vnp-memory/services/graphiti-store/usecase/port"
)

// BulkServiceImpl implements port.BulkService.
type BulkServiceImpl struct {
	driver    domain.GraphDriver
	publisher port.EventPublisher
	vectorDim int
}

// NewBulkService creates a BulkService backed by the given graph driver.
func NewBulkService(driver domain.GraphDriver, publisher port.EventPublisher, vectorDim int) *BulkServiceImpl {
	return &BulkServiceImpl{
		driver:    driver,
		publisher: publisher,
		vectorDim: vectorDim,
	}
}

// SaveBulk validates all items and persists them atomically.
func (s *BulkServiceImpl) SaveBulk(ctx context.Context, req port.SaveBulkRequest) (*port.SaveBulkResponse, error) {
	// Validate episode
	if err := req.Episode.Validate(); err != nil {
		return nil, err
	}

	// Validate all nodes
	for i := range req.Nodes {
		if err := req.Nodes[i].Validate(); err != nil {
			return nil, err
		}
	}

	// Validate all edges
	for i := range req.Edges {
		if err := req.Edges[i].Validate(); err != nil {
			return nil, err
		}
	}

	// Delegate to driver (atomic transaction)
	if err := s.driver.SaveBulk(ctx, req.Nodes, req.Edges, req.Episode); err != nil {
		return nil, &domain.ErrBulkOperationFailed{
			Operation: "SaveBulk",
			EpisodeID: req.Episode.UUID,
			Cause:     err,
		}
	}

	// Publish event (best-effort, don't fail the operation)
	if s.publisher != nil {
		_ = s.publisher.PublishBulkSaved(ctx, req.Episode.GroupID, req.Episode.UUID,
			len(req.Nodes), len(req.Edges))
	}

	return &port.SaveBulkResponse{
		NodeCount: len(req.Nodes),
		EdgeCount: len(req.Edges),
		EpisodeID: req.Episode.UUID,
	}, nil
}

// RollbackBulk removes all data created by a specific episode.
func (s *BulkServiceImpl) RollbackBulk(ctx context.Context, episodeID string) error {
	return s.driver.RollbackBulk(ctx, episodeID)
}

// DeleteByGroupID purges all tenant data.
func (s *BulkServiceImpl) DeleteByGroupID(ctx context.Context, groupID string) error {
	return s.driver.DeleteByGroupID(ctx, groupID)
}

// IndexServiceImpl implements port.IndexService.
type IndexServiceImpl struct {
	driver    domain.GraphDriver
	vectorDim int
}

// NewIndexService creates an IndexService with the configured vector dimensions.
func NewIndexService(driver domain.GraphDriver, vectorDim int) *IndexServiceImpl {
	return &IndexServiceImpl{driver: driver, vectorDim: vectorDim}
}

// BuildIndices creates all standard indexes using the default definitions.
func (s *IndexServiceImpl) BuildIndices(ctx context.Context, groupID string) error {
	defs := domain.DefaultIndexes(s.vectorDim)
	return s.driver.BuildIndices(ctx, groupID, defs)
}
