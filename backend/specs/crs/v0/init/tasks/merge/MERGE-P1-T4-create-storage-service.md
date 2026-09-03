---
id: MERGE-P1-T4
title: "storage-service: Tạo mới — Merge ov-fs + ov-crypto + ov-resource + ov-session"
phase: P1
service: storage-service (NEW)
priority: P0
status: Done
estimated: 8h
created: 2026-06-11
linked_sol: SOL-003
depends_on: []
---

## Mục Tiêu

Tạo service `storage-service` mới bằng cách merge toàn bộ OpenViking file system services. Service này xử lý tất cả operations liên quan đến file, encryption, resource ingestion, và chat sessions trên file system.

## Services Bị Absorb

| Service | Lines | Chức Năng |
|---------|-------|-----------|
| `ov-fs` | 1,562 | File read/write/delete/tree/grep |
| `ov-crypto` | 926 | Encrypt/decrypt file content |
| `ov-resource` | 1,406 | Resource ingestion pipeline |
| `ov-session` | 1,683 | Chat session + message history |
| `ov-storage` | 1,084 | Storage abstraction layer (base) |

**Tổng: 6,661 lines** → 1 service

## Cấu Trúc Service Mới

```
services/storage-service/
├── Dockerfile
├── go.mod
├── cmd/server/main.go          # Single binary entry point
├── api/
│   └── proto/v1/               # Protobuf definitions (if needed)
├── internal/
│   ├── domain/
│   │   ├── fs/                 # File, Directory, TreeNode, GrepResult
│   │   ├── crypto/             # EncryptedContent, KeyDerivation
│   │   ├── resource/           # Resource, IngestJob, IngestStatus
│   │   └── session/            # ChatSession, Message, CommitRecord
│   ├── usecase/
│   │   ├── fs/                 # ReadFile, WriteFile, DeleteFile, Tree, Grep
│   │   ├── crypto/             # Encrypt, Decrypt, HashFile
│   │   ├── resource/           # Ingest, GetStatus, ListResources
│   │   └── session/            # CreateSession, AddMessage, CommitSession, GetHistory
│   ├── adapter/
│   │   └── grpc/               # ForwardService router + route handlers
│   └── infra/
│       ├── localfs/            # Local filesystem implementation
│       ├── s3/                 # Optional S3/MinIO backend (feature-flagged)
│       ├── pgvector/           # Resource embedding index
│       └── config/             # Config loader
└── migrations/
    └── 001_storage_init.sql
```

## Domain Entities

### `domain/fs/entity.go`

```go
package fs

type File struct {
    Path      string
    Content   []byte
    Size      int64
    MimeType  string
    Encrypted bool
    ModTime   time.Time
}

type Directory struct {
    Path     string
    Children []TreeNode
}

type TreeNode struct {
    Name     string
    IsDir    bool
    Size     int64
    Children []TreeNode
}

type GrepResult struct {
    Path    string
    Line    int
    Content string
    Match   string
}
```

### `domain/session/entity.go`

```go
package session

type ChatSession struct {
    ID        string
    TenantID  string
    BaseDir   string   // Working directory path
    CreatedAt time.Time
    Status    string   // "open" | "committed"
}

type Message struct {
    ID        string
    SessionID string
    Role      string  // "user" | "assistant" | "system"
    Content   string
    CreatedAt time.Time
}

type CommitRecord struct {
    SessionID string
    FilesDiff []FileDiff
    Summary   string
    CommittedAt time.Time
}
```

### `domain/resource/entity.go`

```go
package resource

type Resource struct {
    ID        string
    TenantID  string
    URI       string   // file:// | http:// | s3://
    Type      string   // "document" | "image" | "code" | "web"
    Status    string   // "pending" | "processing" | "indexed" | "failed"
    EmbedPath string   // Where embedding is stored
    CreatedAt time.Time
}

type IngestJob struct {
    ResourceID string
    URI        string
    Options    IngestOptions
    CreatedAt  time.Time
}
```

## Usecase Interfaces

```go
// usecase/fs/service.go
type FSUseCase interface {
    ReadFile(ctx context.Context, tenantID, path string) (*fs.File, error)
    WriteFile(ctx context.Context, tenantID, path string, content []byte) error
    DeleteFile(ctx context.Context, tenantID, path string) error
    Tree(ctx context.Context, tenantID, path string, depth int) (*fs.Directory, error)
    Grep(ctx context.Context, tenantID, path, pattern string) ([]*fs.GrepResult, error)
}

// usecase/resource/service.go
type ResourceUseCase interface {
    Ingest(ctx context.Context, tenantID string, job *resource.IngestJob) (*resource.Resource, error)
    GetStatus(ctx context.Context, resourceID string) (*resource.Resource, error)
    List(ctx context.Context, tenantID string) ([]*resource.Resource, error)
}

// usecase/session/service.go
type SessionUseCase interface {
    Create(ctx context.Context, tenantID, baseDir string) (*session.ChatSession, error)
    AddMessage(ctx context.Context, sessionID string, msg *session.Message) error
    Commit(ctx context.Context, sessionID string) (*session.CommitRecord, error)
    GetHistory(ctx context.Context, sessionID string) ([]*session.Message, error)
}
```

## ForwardService Routes

```go
// cmd/server/main.go — route registration
func registerRoutes(router *forward.Router, fs FSHandler, crypto CryptoHandler, resource ResourceHandler, session SessionHandler) {
    // File System
    router.Handle("GET",    "/v1/ov/files/*",             fs.ReadFile)
    router.Handle("PUT",    "/v1/ov/files/*",             fs.WriteFile)
    router.Handle("DELETE", "/v1/ov/files/*",             fs.DeleteFile)
    router.Handle("GET",    "/v1/ov/tree/*",              fs.Tree)
    router.Handle("POST",   "/v1/ov/grep",                fs.Grep)

    // Sessions
    router.Handle("POST",   "/v1/ov/sessions",            session.Create)
    router.Handle("POST",   "/v1/ov/sessions/*/messages", session.AddMessage)
    router.Handle("POST",   "/v1/ov/sessions/*/commit",   session.Commit)

    // Resources
    router.Handle("POST",   "/v1/ov/resources/ingest",    resource.Ingest)

    // Search (delegates to search-service — forward call, not handled here)
    // /v1/ov/search → handled by search-service
}
```

## Infrastructure

### `infra/localfs/` — Local Filesystem

```go
type LocalFSRepo struct {
    BaseDir string  // root sandbox directory
}

func (r *LocalFSRepo) Read(ctx context.Context, relPath string) ([]byte, error) {
    fullPath := filepath.Join(r.BaseDir, relPath)
    // Security: validate no path traversal
    return os.ReadFile(fullPath)
}
```

### `infra/pgvector/` — Resource Embedding Index

```go
// Store resource embeddings for semantic search
type PGResourceIndex struct {
    pool *pgxpool.Pool
}
func (r *PGResourceIndex) IndexResource(ctx context.Context, res *resource.Resource, embedding []float32) error
func (r *PGResourceIndex) Search(ctx context.Context, query []float32, limit int) ([]*resource.Resource, error)
```

## Database Migration

```sql
-- migrations/001_storage_init.sql
CREATE TABLE IF NOT EXISTS ov_sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT NOT NULL,
    base_dir   TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ov_messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES ov_sessions(id),
    role       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ov_resources (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT NOT NULL,
    uri        TEXT NOT NULL,
    type       TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending',
    embedding  vector(1536),   -- pgvector
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ov_resources_tenant ON ov_resources(tenant_id);
```

## Config Environment Variables

```bash
GRPC_PORT=9090
HEALTH_PORT=9140
DATABASE_URL=postgres://...
FS_BASE_DIR=/data/storage          # Root directory for file sandbox
FS_MAX_FILE_SIZE_MB=100            # Max file size in MB
CRYPTO_KEY_DERIVATION_SECRET=...   # Secret for key derivation
S3_ENABLED=false                   # Optional S3 backend
S3_ENDPOINT=...
S3_BUCKET=...
EMBEDDING_MODEL=text-embedding-3-small
NATS_URL=nats://nats:4222
```

## go.mod

```
module vnp-memory/services/storage-service

go 1.25.0

require (
    vnp-memory/pkg/forward   v0.0.0
    vnp-memory/pkg/telemetry v0.0.0
    vnp-memory/pkg/tenant    v0.0.0
    google.golang.org/grpc   v1.72.1
    github.com/jackc/pgx/v5  v5.7.0
    github.com/pgvector/pgvector-go v0.2.0
)
```

## Acceptance Criteria

- [ ] `PUT /v1/ov/files/test.txt` với body content → creates file
- [ ] `GET /v1/ov/files/test.txt` returns file content
- [ ] `DELETE /v1/ov/files/test.txt` removes file
- [ ] `GET /v1/ov/tree/` returns directory tree JSON
- [ ] `POST /v1/ov/grep` với pattern → returns matching lines
- [ ] `POST /v1/ov/sessions` tạo session trong PostgreSQL
- [ ] `POST /v1/ov/resources/ingest` queues ingest job
- [ ] `/healthz` returns 200 OK
- [ ] gRPC health check returns SERVING
- [ ] Path traversal bị blocked (security test: `../../../etc/passwd`)
- [ ] `go build ./services/storage-service/...` passes
- [ ] `go test ./services/storage-service/...` passes

## Ghi Chú

- **Crypto domain:** ov-crypto xử lý encrypt/decrypt — implement với AES-256-GCM. Key được derive từ tenant ID + secret
- **Search:** `/v1/ov/search` route KHÔNG implement ở đây — sẽ delegate sang `search-service`
- **Resource ingestion:** Async via NATS — service publish job, worker xử lý embedding riêng
- Tất cả 5 service gốc (ov-*) giữ nguyên cho đến P4 cleanup
