// Package usecase implements the graphiti-store business logic.
// All usecases depend on domain.GraphDriver (output port) and are called by gRPC handlers.
package usecase

import (
	"context"
	"time"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/usecase/port"
)

// NodeServiceImpl implements port.NodeService.
type NodeServiceImpl struct {
	driver domain.GraphDriver
}

// NewNodeService creates a NodeService backed by the given graph driver.
func NewNodeService(driver domain.GraphDriver) *NodeServiceImpl {
	return &NodeServiceImpl{driver: driver}
}

// SaveNode validates and persists an entity node.
func (s *NodeServiceImpl) SaveNode(ctx context.Context, req port.SaveNodeRequest) (*domain.EntityNode, error) {
	now := time.Now().UTC()
	node := domain.EntityNode{
		UUID:          req.UUID,
		Name:          req.Name,
		GroupID:       req.GroupID,
		Summary:       req.Summary,
		NameEmbedding: req.NameEmbedding,
		Labels:        req.Labels,
		Attributes:    req.Attributes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := node.Validate(); err != nil {
		return nil, err
	}

	if err := s.driver.SaveNode(ctx, node); err != nil {
		return nil, err
	}

	return &node, nil
}

// GetNode retrieves an entity node by UUID.
func (s *NodeServiceImpl) GetNode(ctx context.Context, groupID, uuid string) (*domain.EntityNode, error) {
	return s.driver.GetNode(ctx, groupID, uuid)
}

// DeleteNode removes a node and its relationships.
func (s *NodeServiceImpl) DeleteNode(ctx context.Context, groupID, uuid string) error {
	return s.driver.DeleteNode(ctx, groupID, uuid)
}
