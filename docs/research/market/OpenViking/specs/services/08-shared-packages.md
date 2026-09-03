# 08 — Shared Packages (`pkg/`)

> **Role**: Shared infrastructure — NO business logic  
> **Rule**: Only types, interfaces, middleware, adapters, utilities  
> **Used by**: All 7 services (gateway + 6 domain services)

---

## 1. Package Map

```
pkg/
├── viking/          # Shared domain types (from L3 Core Domain)
├── adapters/        # Infrastructure adapter interfaces + implementations
├── vikingfs/        # Go-native filesystem engine (replacing RAGFS Rust)
├── middleware/      # Shared gRPC/HTTP interceptors
├── resilience/      # Circuit breaker, retry, bulkhead
├── observability/   # OTel tracer, Prometheus metrics, slog logger
├── config/          # Viper configuration loader
├── errors/          # Error types → gRPC/HTTP status mapping
├── nats/            # NATS JetStream client helpers
├── auth/            # API key validator, JWT provider, RBAC
├── tenant/          # Tenant context propagation
├── pagination/      # Cursor/offset pagination
├── parse/           # File parser registry (Go tree-sitter)
└── testutil/        # Test fixtures, mocks, testcontainers
```

---

## 2. `pkg/viking/` — Shared Domain Types

### 2.1 Context (Primary Record)

```go
type Context struct {
    URI            string            `json:"uri"`             // viking:// URI (PK)
    ParentURI      string            `json:"parent_uri"`
    ContextType    ContextType       `json:"context_type"`    // MEMORY|RESOURCE|SKILL|SESSION
    Level          int               `json:"level"`           // 0=Abstract, 1=Overview, 2=Detail
    OwnerAccountID string            `json:"owner_account_id"`
    OwnerUserID    string            `json:"owner_user_id"`
    OwnerAgentID   string            `json:"owner_agent_id"`
    Abstract       string            `json:"abstract"`        // L0 text (~100 tokens)
    Category       string            `json:"category"`
    ActiveCount    int64             `json:"active_count"`    // Hotness
    CreatedAt      time.Time         `json:"created_at"`
    UpdatedAt      time.Time         `json:"updated_at"`
    Meta           map[string]any    `json:"meta"`
}

type ContextType int
const (
    ContextTypeMemory   ContextType = 0
    ContextTypeResource ContextType = 1
    ContextTypeSkill    ContextType = 2
    ContextTypeSession  ContextType = 3
)

type ContextLevel int
const (
    LevelAbstract ContextLevel = 0  // .abstract.md (~100 tokens)
    LevelOverview ContextLevel = 1  // .overview.md (~2000 tokens)
    LevelDetail   ContextLevel = 2  // Original file (full)
)
```

### 2.2 Namespace

```go
// CanonicalizeURI normalizes URI to canonical form
func CanonicalizeURI(uri string) (string, error)

// ResolveOwner extracts (accountID, userID, agentID) from URI
func ResolveOwner(uri string) (string, string, string, error)

// IsAccessible checks if user can access URI based on role
func IsAccessible(uri string, ctx *RequestContext) bool

// ValidateURI rejects .., \, and drive-letter components
func ValidateURI(uri string) error

// Canonical root constants
const (
    RootResources = "viking://resources/"
    RootUser      = "viking://user/"
    RootAgent     = "viking://agent/"
    RootSession   = "viking://session/"
    RootTemp      = "viking://temp/"
)
```

### 2.3 Identity

```go
type UserIdentifier struct {
    AccountID string
    UserID    string
    AgentID   string
}

type Role int
const (
    RoleUser  Role = 0
    RoleAdmin Role = 1
    RoleRoot  Role = 2
)

type RequestContext struct {
    User            UserIdentifier
    Role            Role
    NamespacePolicy string
    APIKeyID        string
}
```

### 2.4 Error Hierarchy

```go
type OpenVikingError struct {
    Code    ErrorCode
    Message string
    Details map[string]any
}

type ErrorCode string
const (
    ErrInvalidArgument    ErrorCode = "INVALID_ARGUMENT"
    ErrUnauthenticated    ErrorCode = "UNAUTHENTICATED"
    ErrPermissionDenied   ErrorCode = "PERMISSION_DENIED"
    ErrNotFound           ErrorCode = "NOT_FOUND"
    ErrAlreadyExists      ErrorCode = "ALREADY_EXISTS"
    ErrFailedPrecondition ErrorCode = "FAILED_PRECONDITION"
    ErrResourceBusy       ErrorCode = "RESOURCE_BUSY"
    ErrNotInitialized     ErrorCode = "NOT_INITIALIZED"
    ErrInternal           ErrorCode = "INTERNAL"
)
```

---

## 3. `pkg/adapters/` — Infrastructure Interfaces

### 3.1 VectorDB

```go
type VectorStore interface {
    CreateCollection(ctx context.Context, name string, schema CollectionSchema) error
    CollectionExists(ctx context.Context, name string) (bool, error)
    SearchGlobalRoots(ctx context.Context, dense []float32, sparse map[string]float32, accountID string, topK int) ([]ScoredContext, error)
    SearchChildren(ctx context.Context, parentURI string, dense []float32, sparse map[string]float32, accountID string) ([]ScoredContext, error)
    UpsertContext(ctx context.Context, vec ContextVector) error
    DeleteContext(ctx context.Context, uri string) error
    UpdateActiveCount(ctx context.Context, uri string, delta int64) error
}
```

### 3.2 Embedder

```go
type EmbedderClient interface {
    Embed(ctx context.Context, text string, isQuery bool) (*EmbedResult, error)
    EmbedBatch(ctx context.Context, texts []string) ([]*EmbedResult, error)
    Dimension() int
    SupportsSparse() bool
}

type EmbedResult struct {
    DenseVector  []float32
    SparseVector map[string]float32
}
```

### 3.3 VLM (Vision-Language Model)

```go
type VLMClient interface {
    Generate(ctx context.Context, prompt string, opts ...VLMOption) (string, error)
    GenerateWithImage(ctx context.Context, prompt string, image []byte, opts ...VLMOption) (string, error)
    GenerateStructured(ctx context.Context, prompt string, schema any) (json.RawMessage, error)
}
```

### 3.4 KMS

```go
type KMSProvider interface {
    GetRootKey(ctx context.Context) ([]byte, error)
    DeriveAccountKey(ctx context.Context, accountID string) ([]byte, error)
    RotateRootKey(ctx context.Context) error
    ProviderType() byte
}
```

### 3.5 Reranker

```go
type RerankerClient interface {
    Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}
```

---

## 4. `pkg/vikingfs/` — Go-Native Filesystem

Replaces RAGFS (Rust) with pure Go implementation:

```go
type FileSystem interface {
    Read(path string, offset, size int64) ([]byte, error)
    Write(path string, data []byte) error
    Mkdir(path string) error
    Rm(path string, recursive bool) error
    Stat(path string) (*FileInfo, error)
    Exists(path string) (bool, error)
    Ls(path string) ([]DirEntry, error)
    Mv(oldPath, newPath string) error
    Cp(srcPath, dstPath string) error
}
```

---

## 5. `pkg/middleware/` — Shared Interceptors

| Middleware | Type | Description |
|-----------|------|-------------|
| `auth/` | gRPC + HTTP | Extract identity from metadata/headers |
| `logging/` | gRPC | Structured access logging with slog |
| `tracing/` | gRPC | OTel trace propagation |
| `recovery/` | gRPC | Panic recovery → gRPC INTERNAL |
| `ratelimit/` | gRPC + HTTP | Redis sliding window rate limiter |
| `validation/` | gRPC | Request field validation |

---

## 6. `pkg/resilience/` — Resilience Patterns

| Pattern | Implementation | Description |
|---------|---------------|-------------|
| Circuit Breaker | `sony/gobreaker` | Per-service, configurable thresholds |
| Retry | Exponential backoff + jitter | Max 3 attempts, configurable |
| Bulkhead | Go channel semaphore | Isolate LLM calls from DB calls |
| Timeout | gRPC deadline propagation | Cascade through service chain |

---

## 7. `pkg/observability/` — Observability Stack

| Signal | Technology | Export |
|--------|-----------|--------|
| Traces | OpenTelemetry SDK | OTLP endpoint (Jaeger/Tempo) |
| Metrics | OTel → Prometheus | `/metrics` endpoint |
| Logs | `log/slog` structured JSON | stdout + OTel log handler |
| Health | gRPC Health v1 | HTTP /healthz /readyz /livez |
| Profiling | `net/http/pprof` | Continuous profiling |

---

## 8. `pkg/nats/` — Event Bus Helpers

```go
type Publisher interface {
    Publish(ctx context.Context, subject string, data []byte) error
    PublishAsync(ctx context.Context, subject string, data []byte) error
}

type Subscriber interface {
    Subscribe(ctx context.Context, subject string, handler MessageHandler) error
    QueueSubscribe(ctx context.Context, subject, queue string, handler MessageHandler) error
}

type MessageHandler func(ctx context.Context, msg *nats.Msg) error
```

---

## 9. `pkg/parse/` — File Parser Registry

```go
type Parser interface {
    Parse(ctx context.Context, path string, content []byte) ([]Chunk, error)
    SupportedExtensions() []string
}

type Chunk struct {
    Content    string
    StartLine  int
    EndLine    int
    Language   string
    Metadata   map[string]any
}

// Registry maps file extensions to parsers
type Registry struct {
    parsers map[string]Parser
}

func NewRegistry() *Registry // registers 50+ extensions
```
