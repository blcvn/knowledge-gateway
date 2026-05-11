package usecase

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
)

type KnowledgeClient interface {
	ExtractEntities(ctx context.Context, episode domain.Episode) ([]map[string]interface{}, error)
	ResolveEntities(ctx context.Context, groupID string, entities []map[string]interface{}) error
	ExtractEdges(ctx context.Context, episode domain.Episode, entities []map[string]interface{}) ([]map[string]interface{}, error)
	ResolveEdges(ctx context.Context, groupID string, edges []map[string]interface{}) error
	UpdateCommunity(ctx context.Context, groupID string) error
}

type StoreClient interface {
	SaveBulk(ctx context.Context, groupID string, data map[string]interface{}) error
	RollbackBulk(ctx context.Context, groupID string, sagaID string) error
}

type SagaStateRepo interface {
	Create(ctx context.Context, state *domain.SagaState) error
	Get(ctx context.Context, id string) (*domain.SagaState, error)
	Update(ctx context.Context, state *domain.SagaState) error
	GetStuckSagas(ctx context.Context, timeoutMinutes int, limit int) ([]*domain.SagaState, error)
}

type EpisodeRepo interface {
	Create(ctx context.Context, episode *domain.Episode) error
	GetByHash(ctx context.Context, contentHash string) (*domain.Episode, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event domain.DomainEvent) error
}
