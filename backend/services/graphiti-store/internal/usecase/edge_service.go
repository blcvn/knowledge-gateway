package usecase

import (
	"context"
	"time"

	"vnp-memory/services/graphiti-store/domain"
	"vnp-memory/services/graphiti-store/usecase/port"
)

// EdgeServiceImpl implements port.EdgeService.
type EdgeServiceImpl struct {
	driver domain.GraphDriver
}

// NewEdgeService creates an EdgeService backed by the given graph driver.
func NewEdgeService(driver domain.GraphDriver) *EdgeServiceImpl {
	return &EdgeServiceImpl{driver: driver}
}

// SaveEdge validates bi-temporal constraints and persists an entity edge.
func (s *EdgeServiceImpl) SaveEdge(ctx context.Context, req port.SaveEdgeRequest) (*domain.EntityEdge, error) {
	now := time.Now().UTC()
	edge := domain.EntityEdge{
		UUID:          req.UUID,
		SourceNodeID:  req.SourceNodeID,
		TargetNodeID:  req.TargetNodeID,
		Name:          req.Name,
		GroupID:       req.GroupID,
		Fact:          req.Fact,
		FactEmbedding: req.FactEmbedding,
		ValidAt:       req.ValidAt,
		InvalidAt:     req.InvalidAt,
		Attributes:    req.Attributes,
		EpisodeID:     req.EpisodeID,
		CreatedAt:     now,
	}

	if err := edge.Validate(); err != nil {
		return nil, err
	}

	if err := s.driver.SaveEdge(ctx, edge); err != nil {
		return nil, err
	}

	return &edge, nil
}

// GetEdge retrieves an entity edge by UUID.
func (s *EdgeServiceImpl) GetEdge(ctx context.Context, uuid string) (*domain.EntityEdge, error) {
	return s.driver.GetEdge(ctx, uuid)
}

// DeleteEdge removes an edge.
func (s *EdgeServiceImpl) DeleteEdge(ctx context.Context, uuid string) error {
	return s.driver.DeleteEdge(ctx, uuid)
}

// InvalidateEdge marks an edge as no longer valid in the real world.
func (s *EdgeServiceImpl) InvalidateEdge(ctx context.Context, req port.InvalidateEdgeRequest) error {
	return s.driver.InvalidateEdge(ctx, req.UUID, req.InvalidAt)
}
