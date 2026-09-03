package usecase

import (
	"context"
	"time"

	"vnp-memory/services/graphiti-store/domain"
	"vnp-memory/services/graphiti-store/usecase/port"
)

// CommunityServiceImpl implements port.CommunityService.
type CommunityServiceImpl struct {
	driver domain.GraphDriver
}

// NewCommunityService creates a CommunityService backed by the given graph driver.
func NewCommunityService(driver domain.GraphDriver) *CommunityServiceImpl {
	return &CommunityServiceImpl{driver: driver}
}

// SaveCommunity validates and persists a community node.
func (s *CommunityServiceImpl) SaveCommunity(ctx context.Context, req port.SaveCommunityRequest) (*domain.CommunityNode, error) {
	now := time.Now().UTC()
	node := domain.CommunityNode{
		UUID:          req.UUID,
		Name:          req.Name,
		GroupID:       req.GroupID,
		Summary:       req.Summary,
		NameEmbedding: req.NameEmbedding,
		Level:         req.Level,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := node.Validate(); err != nil {
		return nil, err
	}

	if err := s.driver.SaveCommunityNode(ctx, node); err != nil {
		return nil, err
	}

	return &node, nil
}

// GetCommunity retrieves a community node by UUID.
func (s *CommunityServiceImpl) GetCommunity(ctx context.Context, groupID, uuid string) (*domain.CommunityNode, error) {
	return s.driver.GetCommunity(ctx, groupID, uuid)
}

// DeleteCommunity removes a community node and its relationships.
func (s *CommunityServiceImpl) DeleteCommunity(ctx context.Context, groupID, uuid string) error {
	return s.driver.DeleteCommunity(ctx, groupID, uuid)
}
