// Package usecase implements the community rebuild use case for graphiti-admin.
// SOL-007: Admin Service & Observability Stack (CR-GR-007)
package usecase

import (
	"context"
	"fmt"

	"vnp-memory/services/graphiti-admin/internal/usecase/port"
)

// RebuildCommunitiesUseCase rebuilds community summaries for a tenant group.
type RebuildCommunitiesUseCase struct {
	knowledgePort port.KnowledgePort
	storePort     port.StorePort
	publisher     port.EventPublisher
}

// NewRebuildCommunitiesUseCase constructs the use case.
func NewRebuildCommunitiesUseCase(
	knowledgePort port.KnowledgePort,
	storePort port.StorePort,
	publisher port.EventPublisher,
) *RebuildCommunitiesUseCase {
	return &RebuildCommunitiesUseCase{
		knowledgePort: knowledgePort,
		storePort:     storePort,
		publisher:     publisher,
	}
}

// Execute removes old communities and triggers full community detection.
func (uc *RebuildCommunitiesUseCase) Execute(ctx context.Context, groupID string) error {
	// 1. Remove existing communities for group
	if err := uc.storePort.RemoveCommunities(ctx, groupID); err != nil {
		return fmt.Errorf("remove existing communities: %w", err)
	}

	// 2. Trigger full community detection via knowledge service
	if _, err := uc.knowledgePort.BuildCommunities(ctx, port.BuildCommunitiesReq{GroupID: groupID}); err != nil {
		return fmt.Errorf("build communities: %w", err)
	}

	// 3. Publish event → search service invalidates community caches
	_ = uc.publisher.Publish(ctx, "graphiti.community.rebuilt", map[string]any{
		"group_id": groupID,
		"trigger":  "admin_manual",
	})

	return nil
}
