# graphiti-* — Data Models

> **Services**: `services/graphiti-knowledge`, `services/graphiti-admin`, `services/graphiti-ingestion`, `services/graphiti-search`, `services/graphiti-pipeline`, `services/graphiti-store`
> **Role**: Graphiti knowledge graph engine — temporal episode ingestion, entity/edge extraction, community detection, and semantic search.

---

## graphiti-knowledge — Graph Entities

### ExtractedEntity & Edge

```go
type ExtractedEntity struct {
    Name    string `json:"name"`
    Label   string `json:"label"`
    Summary string `json:"summary"`
}

type ExtractedEdge struct {
    Source   string   `json:"source"`
    Target   string   `json:"target"`
    Relation string   `json:"relation"`
    Fact     string   `json:"fact"`
    Temporal []string `json:"temporal,omitempty"`
}
```

### Resolution (Entity Deduplication)

```go
type Resolution struct {
    ExistingEntityID string            `json:"existing_entity_id"`
    ExtractedEntity  ExtractedEntity   `json:"extracted_entity"`
    Decision         DuplicateDecision `json:"decision"`
    Confidence       float64           `json:"confidence"`
}

type DuplicateDecision string
// merge | create | skip
```

### CommunityNode

```go
type CommunityNode struct {
    ID        string
    Name      string
    Summary   string
    Level     CommunityLevel
    MemberIDs []string
}

type CommunityMember struct {
    ID   string
    Type string
}
```

### Value Objects

```go
type PromptTemplate struct {
    ID           string
    SystemPrompt string
    UserPrompt   string
    Model        string
    MaxTokens    int
}

type ModelConfig struct {
    Provider string
    Model    string
    APIKey   string
}

type TokenUsage struct {
    PromptTokens     int    `json:"prompt_tokens"`
    CompletionTokens int    `json:"completion_tokens"`
    TotalTokens      int    `json:"total_tokens"`
    Model            string `json:"model"`
}
```

---

## graphiti-admin — Tenant Management

### Tenant

```go
type Tenant struct {
    GroupID   string
    Name      string
    CreatedAt time.Time
    Config    TenantConfig
}

type TenantConfig struct {
    MaxEpisodes   int    // 0 = unlimited
    LLMProvider   string // override default (e.g. "anthropic")
    EmbedProvider string
}
```

### TenantStats

```go
type TenantStats struct {
    GroupID             string
    EpisodeCount        int64
    EntityCount         int64
    EdgeCount           int64
    CommunityCount      int64
    StorageSizeEstimate string // e.g. "2.3 MB"
}
```

### MaintenanceTask

```go
type MaintenanceTask struct {
    ID        string
    GroupID   string
    TaskType  string // "rebuild_communities" | "build_indices" | "delete_data"
    Status    string // "pending" | "running" | "done" | "failed"
    StartedAt time.Time
    DoneAt    *time.Time
    Error     string
}
```

### PostgresTenant (DB model)

```go
type PostgresTenant struct {
    GroupID       string    `db:"group_id"`
    Name          string    `db:"name"`
    MaxEpisodes   int       `db:"max_episodes"`
    LLMProvider   string    `db:"llm_provider"`
    EmbedProvider string    `db:"embed_provider"`
    CreatedAt     time.Time `db:"created_at"`
    UpdatedAt     time.Time `db:"updated_at"`
}
```

---

## Sources
- [`services/graphiti-knowledge/internal/domain/entity.go`](../../services/graphiti-knowledge/internal/domain/entity.go)
- [`services/graphiti-knowledge/internal/domain/community.go`](../../services/graphiti-knowledge/internal/domain/community.go)
- [`services/graphiti-knowledge/internal/domain/value_object.go`](../../services/graphiti-knowledge/internal/domain/value_object.go)
- [`services/graphiti-admin/internal/domain/tenant.go`](../../services/graphiti-admin/internal/domain/tenant.go)
