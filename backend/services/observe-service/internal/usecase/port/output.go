package port

import (
    "context"
    "github.com/vnp-memory/services/observe-service/internal/domain"
)

type IKVStore interface {
    GetSessionObsCount(ctx context.Context, sessionID string) int
    GetSessionAgentID(ctx context.Context, sessionID string) string
    IncrementObsCount(ctx context.Context, sessionID string)
    SaveRawObservation(ctx context.Context, raw domain.RawObservation) error
    SaveCompressedObservation(ctx context.Context, comp domain.CompressedObservation)
}

type ISearchIndexer interface {
    IndexObservation(ctx context.Context, comp domain.CompressedObservation) error
}

type IEventPublisher interface {
    Publish(ctx context.Context, subject string, payload any) error
}

type IStreamBroker interface {
    Subscribe(sessionFilter string) (chan any, func())
    Broadcast(event any)
}

type ISessionRepo interface {
    Save(ctx context.Context, session domain.Session) error
    GetByID(ctx context.Context, id string) (*domain.Session, error)
    List(ctx context.Context, tenantID, status, project string, limit, offset int) ([]domain.Session, error)
    UpdateStatus(ctx context.Context, id, status string) error
    IncrementObsCount(ctx context.Context, id string) error
    GetObsCount(ctx context.Context, id string) (int, error)
}

type IObservationRepo interface {
    SaveRaw(ctx context.Context, raw domain.RawObservation) error
    SaveCompressed(ctx context.Context, comp domain.CompressedObservation) error
    ListCompressed(ctx context.Context, sessionID string, limit, offset int) ([]domain.CompressedObservation, error)
    DeleteBySessionID(ctx context.Context, sessionID string) error
}
