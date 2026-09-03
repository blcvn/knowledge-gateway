package port

import (
	"time"
	"context"

	"vnp-memory/services/memory-service/internal/domain/agentmemory"
)

// IMemoryRepo is the output port for persisting AgentMemory entities.
type IMemoryRepo interface {
	Save(ctx context.Context, mem agentmemory.AgentMemory) error
	Get(ctx context.Context, id string) (*agentmemory.AgentMemory, error)
	ListLatestByType(ctx context.Context, tenantID, project, memType string) ([]agentmemory.AgentMemory, error)
	ListLatestByProject(ctx context.Context, tenantID, project string) ([]agentmemory.AgentMemory, error)
	ListAll(ctx context.Context) ([]agentmemory.AgentMemory, error)
	FindExpired(ctx context.Context) ([]agentmemory.AgentMemory, error)
	SetNotLatest(ctx context.Context, id string) error
	UpdateStrength(ctx context.Context, id string, strength float64) error
	FlagForEviction(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

// ISearchNotifier is the output port for notifying the search service of index updates.
type ISearchNotifier interface {
	IndexMemory(ctx context.Context, mem agentmemory.AgentMemory) error
	RemoveMemory(ctx context.Context, id string) error
}

// IEventPublisher is the output port for publishing domain events to NATS.
type IEventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

// IAuditRepo is the output port for persisting audit entries.
type IAuditRepo interface {
	Save(ctx context.Context, entry agentmemory.AuditEntry) error
	List(ctx context.Context, filter AuditFilter) ([]agentmemory.AuditEntry, error)
}

// AuditFilter defines filter criteria for listing audit entries.
type AuditFilter struct {
	TenantID  string
	Project   string
	Operation string
	TargetID  string
	FromTime  *time.Time
	ToTime    *time.Time
	Limit     int
	Offset    int
}

// IGraphClient is the output port for interacting with the knowledge graph.
type IGraphClient interface {
	RemoveBySourceID(ctx context.Context, sourceID string) error
}

// ISlotsRepo is the output port for persisting MemorySlot entities.
type ISlotsRepo interface {
	GetSlot(ctx context.Context, tenantID, scope, label string) (*agentmemory.MemorySlot, error)
	CreateSlot(ctx context.Context, slot agentmemory.MemorySlot) error
	UpdateSlot(ctx context.Context, slot agentmemory.MemorySlot) error
	DeleteSlot(ctx context.Context, tenantID, scope, label string) error
	ListSlots(ctx context.Context, tenantID, project string) ([]agentmemory.MemorySlot, error)
}
