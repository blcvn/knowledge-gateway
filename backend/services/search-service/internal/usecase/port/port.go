package port

import (
    "context"
    "vnp-memory/services/search-service/internal/domain"
)

type IEmbedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}

type IObservationStore interface {
    GetRecentSummaries(ctx context.Context, tenantID, project string, limit int) ([]domain.Summary, error)
    GetByIDs(ctx context.Context, ids []string) (map[string]domain.Observation, error)
}

type IAgentMemoryClient interface {
    ListLatest(ctx context.Context, tenantID, project string, days int) ([]domain.AgentMemory, error)
}
