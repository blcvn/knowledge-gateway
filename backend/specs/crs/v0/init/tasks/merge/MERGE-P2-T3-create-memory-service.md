---
id: MERGE-P2-T3
title: "memory-service: Tạo mới — Merge memobase-* + zep-* + sm-memory/document/profile"
phase: P2
service: memory-service (NEW)
priority: P1
status: Done
estimated: 16h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P1-T1]
---

## Mục Tiêu

Tạo `memory-service` mới — service quản lý toàn bộ memory của agents/users: working memory (memobase), long-term episodic memory (zep qua SDK), và Supermemory entries.

## Services Bị Absorb

| Service | Lines | Chức Năng |
|---------|-------|-----------|
| `memobase-context` | 1,173 | User context retrieval |
| `memobase-engine` | 958 | Memory scoring + relevance |
| `memobase-ingestion` | 799 | Blob insertion + flush |
| `memobase-pipeline` | 1,051 | Batch processing |
| `zep-user` | 179 | Zep user management (proxy) |
| `zep-thread` | 179 | Zep session/thread (proxy) |
| `zep-memory` | 187 | Zep memory put/get (proxy) |
| `zep-search` | 616 | Zep search |
| `zep-graph` | 310 | Zep graph facts |
| `zep-core` | 850 | Zep core adapter (TODO skeleton) |
| `zep-admin` | 308 | Zep admin |
| `sm-memory` | 207 | Supermemory entries |
| `sm-document` | 144 | Document management |
| `sm-profile` | 137 | User profile |

**Tổng: ~7,098 lines** → 1 service

## Architecture

```
services/memory-service/
├── Dockerfile
├── go.mod
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── memobase/
│   │   │   ├── entity.go     # Blob, Context, Profile, Event, Buffer
│   │   │   └── errors.go
│   │   ├── zep/
│   │   │   ├── entity.go     # ZepUser, ZepSession, ZepMemory, GraphFact
│   │   │   └── errors.go
│   │   └── sm/
│   │       ├── entity.go     # SMMemory, SMDocument, SMProfile
│   │       └── errors.go
│   ├── usecase/
│   │   ├── memobase/
│   │   │   ├── ingest.go     # InsertBlob, Flush
│   │   │   ├── context.go    # GetContext, GetProfiles
│   │   │   └── port/         # Interfaces
│   │   ├── zep/
│   │   │   ├── user.go       # CreateUser, GetUser, UpdateUser
│   │   │   ├── memory.go     # PutMemory, GetMemory, SessionSearch
│   │   │   ├── graph.go      # GraphSearch, AddFact, SetOntology
│   │   │   └── port/         # ZepClient interface
│   │   └── sm/
│   │       ├── memory.go     # CreateMemory, RAG
│   │       ├── document.go   # CreateDocument, GetDocument
│   │       └── profile.go    # GetProfile
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── router.go     # ForwardService routes
│   │   └── zep/
│   │       └── client.go     # Zep Cloud HTTP client (uses zep-go SDK)
│   └── infra/
│       ├── pgvector/         # Memobase blob + embedding storage
│       ├── redis/            # Working memory cache
│       ├── nats/             # Event publishing
│       └── config/
└── migrations/
    └── 001_memory_init.sql
```

## Domain Entities — Memobase

```go
// domain/memobase/entity.go

type Blob struct {
    ID        string
    UserID    string
    TenantID  string
    Type      string      // "conversation" | "fact" | "document" | "image"
    Content   string      // Raw text content
    Metadata  map[string]any
    Embedding []float32
    CreatedAt time.Time
}

type UserContext struct {
    UserID    string
    TenantID  string
    Summary   string       // AI-generated user summary
    Profiles  []*Profile   // Structured user attributes
    Events    []*Event     // Recent activity events
    Tokens    int          // Context token count estimate
}

type Profile struct {
    Key       string
    Value     string
    Category  string    // "preference" | "fact" | "goal" | "habit"
    Score     float64   // Confidence score
    UpdatedAt time.Time
}

type Event struct {
    ID        string
    UserID    string
    EventType string
    Content   string
    Metadata  map[string]any
    CreatedAt time.Time
}

type Buffer struct {
    UserID    string
    Blobs     []*Blob
    TokenCount int
    FlushThreshold int
}
```

## Domain Entities — Zep

```go
// domain/zep/entity.go

type ZepUser struct {
    UserID    string
    Email     string
    FirstName string
    LastName  string
    Metadata  map[string]any
    CreatedAt time.Time
}

type ZepSession struct {
    SessionID string
    UserID    string
    Metadata  map[string]any
    CreatedAt time.Time
}

type ZepMemory struct {
    SessionID string
    Messages  []ZepMessage
    Summary   *ZepSummary
    Facts     []string
}

type ZepMessage struct {
    Role      string    // "user" | "assistant"
    Content   string
    Metadata  map[string]any
    CreatedAt time.Time
}

type GraphFact struct {
    UUID    string
    Name    string
    Fact    string
    Episodes []string
}
```

## Domain Entities — Supermemory

```go
// domain/sm/entity.go

type SMMemory struct {
    ID        string
    TenantID  string
    Content   string
    Tags      []string
    Metadata  map[string]any
    Embedding []float32
    CreatedAt time.Time
}

type SMDocument struct {
    ID        string
    TenantID  string
    Title     string
    Content   string
    Type      string      // "markdown" | "pdf" | "html"
    URL       string
    CreatedAt time.Time
}

type SMProfile struct {
    UserID   string
    TenantID string
    Memories []*SMMemory
    Tags     []string
    Stats    ProfileStats
}
```

## Usecase — Memobase

```go
// usecase/memobase/ingest.go
type MemobaseIngestUseCase struct {
    blobs    port.BlobRepository
    embedder port.EmbeddingService
    engine   port.MemoryEngine
    pub      port.EventPublisher
}

func (uc *MemobaseIngestUseCase) InsertBlob(ctx context.Context, userID, tenantID string, blob *Blob) (*Blob, error) {
    // 1. Generate embedding
    emb, _ := uc.embedder.Embed(ctx, blob.Content)
    blob.Embedding = emb
    blob.ID = uuid.New().String()

    // 2. Persist
    if err := uc.blobs.Create(ctx, blob); err != nil {
        return nil, err
    }

    // 3. Update buffer size, trigger flush if needed
    bufSize, _ := uc.blobs.GetBufferSize(ctx, userID)
    if bufSize >= flushThreshold {
        go uc.engine.ProcessBuffer(ctx, userID)
    }

    uc.pub.Publish(ctx, "memory.blob.inserted", blob)
    return blob, nil
}

func (uc *MemobaseIngestUseCase) Flush(ctx context.Context, userID string) error {
    return uc.engine.ProcessBuffer(ctx, userID)
}

// usecase/memobase/context.go
func (uc *MemobaseContextUseCase) GetContext(ctx context.Context, userID, tenantID string) (*UserContext, error) {
    // Aggregate blobs → summary, profiles, events
    blobs, _ := uc.blobs.List(ctx, userID)
    profiles, _ := uc.profiles.GetByUser(ctx, userID)
    events, _ := uc.events.GetByUser(ctx, userID, 10)
    summary := uc.summarizer.Summarize(blobs)
    return &UserContext{
        UserID: userID, Summary: summary,
        Profiles: profiles, Events: events,
    }, nil
}
```

## Usecase — Zep (via zep-go SDK)

```go
// usecase/zep/user.go
type ZepUserUseCase struct {
    client port.ZepClient   // Wraps zep-go SDK
}

func (uc *ZepUserUseCase) CreateUser(ctx context.Context, req CreateUserRequest) (*ZepUser, error) {
    // Delegate to Zep Cloud via SDK
    return uc.client.CreateUser(ctx, req.UserID, req.Email, req.FirstName, req.LastName, req.Metadata)
}

// adapter/zep/client.go
import zep "github.com/getzep/zep-go/v3"

type ZepSDKClient struct {
    client *zep.Client
}

func NewZepSDKClient(apiKey string) *ZepSDKClient {
    return &ZepSDKClient{
        client: zep.NewClient(zep.WithAPIKey(apiKey)),
    }
}

func (c *ZepSDKClient) CreateUser(ctx context.Context, userID, email, firstName, lastName string, meta map[string]any) (*ZepUser, error) {
    resp, err := c.client.User.Add(ctx, &zep.CreateUserRequest{
        UserID: zep.String(userID),
        Email:  zep.String(email),
        // ...
    })
    if err != nil { return nil, err }
    return &ZepUser{UserID: *resp.UserID, Email: *resp.Email}, nil
}
```

## ForwardService Routes

```go
// adapter/grpc/router.go
func RegisterRoutes(router *forward.Router, mb MemobaseHandlers, zepH ZepHandlers, smH SMHandlers) {
    // Memobase
    router.Handle("POST", "/v1/memobase/users/*/blobs",    mb.InsertBlob)
    router.Handle("POST", "/v1/memobase/users/*/flush",    mb.Flush)
    router.Handle("GET",  "/v1/memobase/users/*/context",  mb.GetContext)
    router.Handle("GET",  "/v1/memobase/users/*/profiles", mb.GetProfiles)
    router.Handle("GET",  "/v1/memobase/users/*/events",   mb.GetEvents)

    // Zep
    router.Handle("POST",  "/v1/zep/users",                zepH.CreateUser)
    router.Handle("GET",   "/v1/zep/users/*",              zepH.GetUser)
    router.Handle("PATCH", "/v1/zep/users/*",              zepH.UpdateUser)
    router.Handle("POST",  "/v1/zep/sessions/*/memory",    zepH.PutMemory)
    router.Handle("GET",   "/v1/zep/sessions/*/memory",    zepH.GetMemory)
    router.Handle("POST",  "/v1/zep/graph/search",         zepH.GraphSearch)
    router.Handle("POST",  "/v1/zep/sessions/*/search",    zepH.SessionSearch)
    router.Handle("POST",  "/v1/zep/graph/facts",          zepH.AddFact)
    router.Handle("POST",  "/v1/zep/graph/ontology",       zepH.SetOntology)

    // Supermemory
    router.Handle("POST", "/v1/sm/memories",               smH.CreateMemory)
    router.Handle("POST", "/v1/sm/rag",                    smH.RAG)
    router.Handle("GET",  "/v1/sm/profiles/*",             smH.GetProfile)
    router.Handle("POST", "/v1/sm/documents",              smH.CreateDocument)
    router.Handle("GET",  "/v1/sm/documents/*",            smH.GetDocument)
}
```

## Database Migration

```sql
-- migrations/001_memory_init.sql

-- Memobase blobs
CREATE TABLE IF NOT EXISTS memory_blobs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT NOT NULL,
    tenant_id  TEXT NOT NULL,
    type       TEXT NOT NULL,
    content    TEXT NOT NULL,
    metadata   JSONB NOT NULL DEFAULT '{}',
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_blobs_user ON memory_blobs(user_id, tenant_id);
CREATE INDEX idx_blobs_embedding ON memory_blobs USING ivfflat (embedding vector_cosine_ops);

-- User profiles (memobase)
CREATE TABLE IF NOT EXISTS memory_profiles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT NOT NULL,
    tenant_id  TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    category   TEXT NOT NULL DEFAULT 'fact',
    score      FLOAT NOT NULL DEFAULT 1.0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, tenant_id, key)
);

-- SM memories
CREATE TABLE IF NOT EXISTS sm_memories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT NOT NULL,
    content    TEXT NOT NULL,
    tags       TEXT[] NOT NULL DEFAULT '{}',
    metadata   JSONB NOT NULL DEFAULT '{}',
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sm_memories_tenant ON sm_memories(tenant_id);

-- SM documents
CREATE TABLE IF NOT EXISTS sm_documents (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT NOT NULL,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'markdown',
    url        TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Config Environment Variables

```bash
GRPC_PORT=9090
HEALTH_PORT=9130
DATABASE_URL=postgres://...
REDIS_URL=redis://redis:6379/1
NATS_URL=nats://nats:4222
EMBEDDING_URL=http://llm-proxy:8080
EMBEDDING_MODEL=text-embedding-3-small
# Zep Cloud
ZEP_API_KEY=...                       # Required for Zep features
ZEP_BASE_URL=https://api.getzep.com  # Default Zep Cloud
ZEP_ENABLED=true
# Memory engine config
MEMOBASE_FLUSH_THRESHOLD=20           # Blob count before auto-flush
MEMOBASE_CONTEXT_MAX_TOKENS=4096      # Max tokens in context response
```

## go.mod

```
module vnp-memory/services/memory-service

go 1.25.0

require (
    vnp-memory/pkg/forward      v0.0.0
    vnp-memory/pkg/telemetry    v0.0.0
    vnp-memory/pkg/tenant       v0.0.0
    vnp-memory/services/zep-go  v0.0.0  // Zep SDK as local dep
    google.golang.org/grpc      v1.72.1
    github.com/jackc/pgx/v5     v5.7.0
    github.com/pgvector/pgvector-go v0.2.0
    github.com/redis/go-redis/v9    v9.x.x
)
```

## Acceptance Criteria

- [ ] `POST /v1/memobase/users/{uid}/blobs` → insert blob, generate embedding, persist
- [ ] `POST /v1/memobase/users/{uid}/flush` → trigger buffer processing
- [ ] `GET /v1/memobase/users/{uid}/context` → returns UserContext JSON với summary + profiles
- [ ] `GET /v1/memobase/users/{uid}/profiles` → returns user profiles list
- [ ] `POST /v1/zep/users` → create user in Zep Cloud (nếu ZEP_API_KEY set)
- [ ] `POST /v1/zep/sessions/{id}/memory` → store memory in Zep
- [ ] `GET /v1/zep/sessions/{id}/memory` → retrieve memory from Zep
- [ ] `POST /v1/sm/memories` → create SM memory với embedding
- [ ] When `ZEP_ENABLED=false` → Zep routes return 503 gracefully
- [ ] Embedding stored in pgvector, semantic search functional
- [ ] Redis cache cho hot user contexts
- [ ] `/healthz` returns 200
- [ ] `go build ./services/memory-service/...` passes
- [ ] Unit tests pass (mock Zep SDK + mock embedder)

## Ghi Chú

- **zep-go SDK** (`services/zep-go`) được dùng như Go module dependency, không phải deployed service
- **Memory Engine:** `memobase-engine` scoring logic cần được implement — extract từ existing code trong `services/memobase-engine/internal/`
- **Zep integration là optional** — nếu `ZEP_API_KEY` không set → memobase vẫn hoạt động
- Tất cả 13 services gốc giữ nguyên cho đến P4 cleanup
