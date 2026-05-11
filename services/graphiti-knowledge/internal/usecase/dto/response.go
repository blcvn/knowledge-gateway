package dto

import "vnp-memory/services/graphiti-knowledge/internal/domain"

type ResolveEdgesResponse struct {
	ResolvedEdges []domain.ExtractedEdge
	Invalidated   []string
}

type UpdateCommunityResponse struct {
	UpdatedCommunities []domain.CommunityNode
}
