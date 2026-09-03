package port

import (
    "context"
    "vnp-memory/services/memory-service/internal/domain/agentmemory"
)

type ILLMProvider interface {
    Chat(ctx context.Context, system, user string) (string, error)
}

type IObservationRepo interface {
    ListCompressed(ctx context.Context, sessionID string) ([]agentmemory.CompressedObs, error)
    ListSessionsNeedingCompression(ctx context.Context) ([]Session, error)
    ListRawUncompressed(ctx context.Context, sessionID string) ([]agentmemory.RawObs, error)
    SaveCompressed(ctx context.Context, obs agentmemory.CompressedObs) error
    ListCompletedSessionsWithoutSummary(ctx context.Context) ([]Session, error)
}

type IAgentMemoryRepo interface {
    Save(ctx context.Context, mem agentmemory.AgentMemory) error
}

type ISessionSummaryRepo interface {
    Save(ctx context.Context, summary agentmemory.SessionSummary) error
    ListByProject(ctx context.Context, project string, limit int) ([]agentmemory.SessionSummary, error)
    ListRecent(ctx context.Context, limit int) ([]agentmemory.SessionSummary, error)
    ListAll(ctx context.Context) ([]agentmemory.SessionSummary, error)
}

type IProceduralRepo interface {
    FindByStepHash(ctx context.Context, hash string) (*agentmemory.ProceduralMemory, error)
    IncrementFrequency(ctx context.Context, id string) error
    Save(ctx context.Context, proc agentmemory.ProceduralMemory) error
    List(ctx context.Context, project string) ([]agentmemory.ProceduralMemory, error)
    ListByTenant(ctx context.Context, tenantID string) ([]agentmemory.ProceduralMemory, error)
}

type ILessonRepo interface {
    Save(ctx context.Context, lesson agentmemory.Lesson) error
    ListAll(ctx context.Context) ([]agentmemory.Lesson, error)
    UpdateConfidence(ctx context.Context, id string, conf float64) error
    ListHighConfidence(ctx context.Context, threshold float64, limit int) ([]agentmemory.Lesson, error)
    ListByTenant(ctx context.Context, tenantID string) ([]agentmemory.Lesson, error)
}

type IInsightRepo interface {
    Save(ctx context.Context, insight agentmemory.Insight) error
    ListByTenant(ctx context.Context, tenantID string) ([]agentmemory.Insight, error)
}

type IEventPublisher interface {
    Publish(ctx context.Context, topic string, data any) error
}

type Session struct { ID string }
