# zep-* — Data Models

> **Services**: `services/zep-user`, `services/zep-thread`, `services/zep-memory`, `services/zep-graph`, `services/zep-search`, `services/zep-admin`
> **Role**: Zep memory engine — conversation thread management, message storage, context assembly with priority-based fact overlay.

---

## zep-user — User Management

```go
type User struct {
    ID        uuid.UUID      `json:"id"`
    TenantID  uuid.UUID      `json:"tenant_id"`
    UserID    string         `json:"user_id"` // External user identifier
    Email     string         `json:"email,omitempty"`
    FirstName string         `json:"first_name,omitempty"`
    LastName  string         `json:"last_name,omitempty"`
    Metadata  map[string]any `json:"metadata,omitempty"` // JSONB merge-patch
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}
```

---

## zep-thread — Thread & Session

```go
type Thread struct {
    ID        uuid.UUID      `json:"id"`
    TenantID  uuid.UUID      `json:"tenant_id"`
    UserID    string         `json:"user_id"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    EndedAt   *time.Time     `json:"ended_at,omitempty"` // nil = active
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type Session struct {
    ID        uuid.UUID  `json:"id"`
    ThreadID  uuid.UUID  `json:"thread_id"`
    TenantID  uuid.UUID  `json:"tenant_id"`
    StartedAt time.Time  `json:"started_at"`
    EndedAt   *time.Time `json:"ended_at,omitempty"`
}
```

---

## zep-memory — Message Storage & Context Assembly

```go
type Message struct {
    ID        uuid.UUID      `json:"id"`
    ThreadID  uuid.UUID      `json:"thread_id"`
    TenantID  uuid.UUID      `json:"tenant_id"`
    Role      string         `json:"role"` // user | assistant | system
    Content   string         `json:"content"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
}

// ContextAssembly — assembled context window for retrieval (SLA: <200ms p95)
type ContextAssembly struct {
    ThreadID    uuid.UUID `json:"thread_id"`
    Messages    []Message `json:"messages"`
    Facts       []Fact    `json:"facts"`     // Priority-based fact overlay
    Summary     string    `json:"summary,omitempty"`
    TokenCount  int       `json:"token_count"`
    AssembledAt time.Time `json:"assembled_at"`
}

type Fact struct {
    ID         uuid.UUID `json:"id"`
    Content    string    `json:"content"`
    Confidence float64   `json:"confidence"`
    Source     string    `json:"source"` // graph | user | system
    CreatedAt  time.Time `json:"created_at"`
}
```

---

## zep-graph — Knowledge Graph Facts

> The zep-graph service extracts structured facts from conversations and stores them as graph nodes/edges.

```go
// Stub model (domain not yet fully elaborated)
type Node struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
}

type Edge struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
}

type Graph struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
}
```

---

## zep-admin — Project & API Key

> The zep-admin domain models are stubs; full models live in `vnp-platform/domain/admin`.

```go
type Project struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
}

type APIKey struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
}
```

---

## Notes

- `ContextAssembly` is the critical hot path — must be assembled in **sub-200ms p95**.
- Facts in `ContextAssembly` come from `zep-graph` extraction, with priority-based ordering.
- User metadata uses **JSONB merge-patch** semantics for partial updates.

---

## Sources
- [`services/zep-user/internal/domain/user/entity.go`](../../services/zep-user/internal/domain/user/entity.go)
- [`services/zep-thread/internal/domain/thread/entity.go`](../../services/zep-thread/internal/domain/thread/entity.go)
- [`services/zep-memory/internal/domain/memory/entity.go`](../../services/zep-memory/internal/domain/memory/entity.go)
- [`services/zep-graph/internal/domain/model/models.go`](../../services/zep-graph/internal/domain/model/models.go)
- [`services/zep-admin/internal/domain/model/models.go`](../../services/zep-admin/internal/domain/model/models.go)
