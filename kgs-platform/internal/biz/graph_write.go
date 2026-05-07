package biz

import "context"

type UpsertOp string

const (
	UpsertOpCreated  UpsertOp = "CREATED"
	UpsertOpUpdated  UpsertOp = "UPDATED"
	UpsertOpConflict UpsertOp = "CONFLICT"

	OutboxOpUpsertEntity = "UPSERT_ENTITY"
	OutboxOpUpsertEdge   = "UPSERT_EDGE"
	OutboxOpDeleteEntity = "DELETE_ENTITY"
	OutboxOpDeleteEdge   = "DELETE_EDGE"
)

type WriteEntity struct {
	EntityID       string
	AppID          string
	TenantID       string
	EntityType     string
	Name           string
	Properties     map[string]any
	Confidence     float64
	SourceFile     string
	ChunkID        string
	SkillID        string
	VersionID      string
	ProvenanceType string
	Domains        []string
	Aliases        []string
	Version        int
}

type WriteEdge struct {
	EdgeID       string
	AppID        string
	TenantID     string
	FromEntityID string
	ToEntityID   string
	RelationType string
	Properties   map[string]any
	Confidence   float64
	VersionID    string
}

type OutboxRecord struct {
	Op       string
	EntityID *string
	EdgeID   *string
	TenantID string
	AppID    string
	Payload  map[string]any
	Status   string
}

type GraphWriteRepo interface {
	UpsertEntity(ctx context.Context, entity WriteEntity) (UpsertOp, error)
	UpsertEdge(ctx context.Context, edge WriteEdge) (UpsertOp, error)
	SoftDeleteEntity(ctx context.Context, entityID, tenantID string) error
	SoftDeleteEdge(ctx context.Context, edgeID, tenantID string) error
	EnqueueOutbox(ctx context.Context, rec OutboxRecord) error
	WithTx(ctx context.Context, fn func(txRepo GraphWriteRepo) error) error
}

type EntityReader interface {
	GetEntity(ctx context.Context, appID, tenantID, entityID string) (map[string]any, error)
	EnrichWithFreshVersions(ctx context.Context, appID, tenantID string, entities []map[string]any) ([]map[string]any, error)
}
