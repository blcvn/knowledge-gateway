package dto

import "vnp-memory/services/graphiti-knowledge/internal/domain"

type ExtractEntitiesRequest struct {
	Content          string
	PreviousEpisodes []string
	EntityTypes      []string
}

type ResolveEntitiesRequest struct {
	ExtractedEntities []domain.ExtractedEntity
	GroupID           string
}

type ExtractEdgesRequest struct {
	Content          string
	Entities         []domain.ExtractedEntity
	PreviousEpisodes []string
}

type ResolveEdgesRequest struct {
	ExtractedEdges []domain.ExtractedEdge
	GroupID        string
}

type EmbedRequest struct {
	Text  string
	Model string
}

type UpdateCommunityRequest struct {
	EntityIDs []string
	GroupID   string
}
