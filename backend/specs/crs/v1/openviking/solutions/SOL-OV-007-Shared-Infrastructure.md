# Solution: SOL-OV-007 — Shared Infrastructure (`pkg/`)

**CR:** [CR-OV-007](../CR-OV-007-Shared-Infrastructure.md)  
**Wave:** 1 (Foundation — phải xây trước mọi service)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng toàn bộ `pkg/` shared packages trong monorepo `vnp-memory/`. Đây là **prerequisite tuyệt đối** — mọi OpenViking service đều import từ đây. Xây dựng song song với hoặc trước Wave 2.

### Triết lý thiết kế

| Nguyên tắc | Áp dụng |
|---|---|
| Interface-first | Mọi adapter đều có interface trong `pkg/adapters/*/interface.go` |
| Zero business logic | `pkg/` chỉ chứa infrastructure, không chứa domain rules |
| Substitutability | Mọi provider đều swap được qua config (không cần recompile) |
| Testability | Mọi interface đều có mock trong `pkg/testutil/mocks/` |

---

## 2. Package Map & Build Order

```
Build Order (dependencies first):

pkg/viking/          ← No external deps (pure domain types)
    ↓
pkg/observability/   ← depends on: OTel SDK, Prometheus
pkg/nats/            ← depends on: nats.go
pkg/resilience/      ← depends on: sony/gobreaker
pkg/auth/            ← depends on: bcrypt, Redis
    ↓
pkg/adapters/vectordb/  ← depends on: Qdrant/Weaviate SDKs
pkg/adapters/embedder/  ← depends on: HTTP clients + Bifrost
pkg/adapters/vlm/       ← depends on: HTTP clients + Bifrost
pkg/adapters/reranker/  ← depends on: HTTP clients
pkg/adapters/kms/       ← depends on: Vault SDK, AWS/GCP SDKs
    ↓
pkg/vikingfs/        ← depends on: pkg/viking/
pkg/parse/           ← depends on: smacker/go-tree-sitter, goldmark, unidoc
    ↓
pkg/middleware/      ← depends on: pkg/auth/, pkg/observability/
    ↓
Services (Wave 2+)   ← depend on all pkg/
```

---

## 3. `pkg/viking/` — Shared Domain Types

### 3.1 URI System (`uri.go`)

```go
package viking

import (
    "fmt"
    "strings"
    "regexp"
)

// Namespace roots
const (
    RootResources = "viking://resources/"
    RootUser      = "viking://user/"
    RootAgent     = "viking://agent/"
    RootSession   = "viking://session/"
    RootTemp      = "viking://temp/"
)

var validURIRegex = regexp.MustCompile(`^viking://[a-zA-Z0-9/_.\-]+$`)

// ValidateURI rejects path traversal, backslash, Windows drive letters
func ValidateURI(uri string) error {
    if !strings.HasPrefix(uri, "viking://") {
        return &OpenVikingError{Code: ErrInvalidArgument, Message: "URI must start with viking://"}
    }
    if strings.Contains(uri, "..") {
        return &OpenVikingError{Code: ErrInvalidArgument, Message: "path traversal not allowed"}
    }
    if strings.Contains(uri, "\\") {
        return &OpenVikingError{Code: ErrInvalidArgument, Message: "backslash not allowed in URI"}
    }
    if !validURIRegex.MatchString(uri) {
        return &OpenVikingError{Code: ErrInvalidArgument, Message: "invalid URI characters"}
    }
    return nil
}

// CanonicalizeURI normalizes double slashes, trailing slash consistency
func CanonicalizeURI(uri string) (string, error) {
    if err := ValidateURI(uri); err != nil {
        return "", err
    }
    // Replace multiple slashes: viking:///foo → viking://foo
    normalized := regexp.MustCompile(`/{3,}`).ReplaceAllString(uri, "//")
    normalized = regexp.MustCompile(`//+`).ReplaceAllStringFunc(normalized, func(s string) string {
        if strings.Index(s, "//") == 0 && strings.HasPrefix(normalized, "viking:") {
            return s // preserve viking:// prefix
        }
        return "/"
    })
    return normalized, nil
}

// ResolveOwner extracts accountID, userID, agentID from URI
// viking://user/{account}/{user_id}/... → account, user_id, ""
// viking://agent/{account}/{user_id}/{agent_id}/... → account, user_id, agent_id
func ResolveOwner(uri string) (accountID, userID, agentID string, err error) {
    parts := strings.SplitN(strings.TrimPrefix(uri, "viking://"), "/", 5)
    switch parts[0] {
    case "user":
        if len(parts) >= 3 {
            return parts[1], parts[2], "", nil
        }
    case "agent":
        if len(parts) >= 4 {
            return parts[1], parts[2], parts[3], nil
        }
    case "resources", "session", "temp":
        return "", "", "", nil
    }
    return "", "", "", &OpenVikingError{Code: ErrInvalidArgument, Message: "cannot resolve owner from URI: " + uri}
}

// IsAccessible checks if the given request context has access to the URI
func IsAccessible(uri string, ctx *RequestContext) bool {
    if ctx.Role >= RoleRoot {
        return true
    }
    accountID, userID, agentID, _ := ResolveOwner(uri)
    if ctx.Role >= RoleAdmin && accountID == ctx.User.AccountID {
        return true
    }
    if ctx.Role == RoleUser && accountID == ctx.User.AccountID && userID == ctx.User.UserID {
        return true
    }
    if ctx.Role == RoleBot && agentID == ctx.User.AgentID {
        return true
    }
    // resources are publicly readable within account
    if strings.HasPrefix(uri, RootResources) && accountID == ctx.User.AccountID {
        return true
    }
    return false
}

// Tiered URI helpers — L0/L1 naming convention
func ToAbstractURI(fileURI string) string { return fileURI + ".abstract.md" }
func ToOverviewURI(fileURI string) string { return fileURI + ".overview.md" }

func IsAbstractURI(uri string) bool { return strings.HasSuffix(uri, ".abstract.md") }
func IsOverviewURI(uri string) bool { return strings.HasSuffix(uri, ".overview.md") }
```

### 3.2 Identity & RBAC (`identity.go`)

```go
package viking

type Role int
const (
    RoleUser  Role = 0  // Access own viking://user/{id}/ namespace
    RoleBot   Role = 1  // Access specific viking://agent/{id}/ namespace
    RoleAdmin Role = 2  // Account-scoped management
    RoleRoot  Role = 3  // Global, cross-tenant
)

type UserIdentifier struct {
    AccountID string
    UserID    string
    AgentID   string  // non-empty for Bot role
}

type RequestContext struct {
    User            UserIdentifier
    Role            Role
    NamespacePolicy string
    APIKeyID        string  // For audit logging
}

// Context key (unexported to avoid collisions)
type ctxKey struct{}
var RequestContextKey = ctxKey{}

func FromContext(ctx context.Context) (*RequestContext, bool) {
    rc, ok := ctx.Value(RequestContextKey).(*RequestContext)
    return rc, ok
}

func WithContext(ctx context.Context, rc *RequestContext) context.Context {
    return context.WithValue(ctx, RequestContextKey, rc)
}
```

### 3.3 Context Types (`context.go`)

```go
package viking

type ContextType int
const (
    ContextTypeMemory   ContextType = 0
    ContextTypeResource ContextType = 1
    ContextTypeSkill    ContextType = 2
    ContextTypeSession  ContextType = 3
)

func (c ContextType) RootURIs() []string {
    switch c {
    case ContextTypeMemory:
        return []string{"viking://user/", "viking://agent/"}
    case ContextTypeResource:
        return []string{"viking://resources/"}
    case ContextTypeSkill:
        return []string{"viking://agent/"}
    case ContextTypeSession:
        return []string{"viking://session/"}
    }
    return nil
}

type ContextLevel int
const (
    LevelAbstract ContextLevel = 0  // .abstract.md  (~100 tokens)
    LevelOverview ContextLevel = 1  // .overview.md  (~2K tokens)
    LevelDetail   ContextLevel = 2  // raw file      (full content)
)
```

### 3.4 Error Types (`errors.go`)

```go
package viking

type ErrorCode string
const (
    ErrInvalidArgument  ErrorCode = "INVALID_ARGUMENT"
    ErrUnauthenticated  ErrorCode = "UNAUTHENTICATED"
    ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"
    ErrNotFound         ErrorCode = "NOT_FOUND"
    ErrAlreadyExists    ErrorCode = "ALREADY_EXISTS"
    ErrResourceBusy     ErrorCode = "RESOURCE_BUSY"   // PathLock contention
    ErrNotInitialized   ErrorCode = "NOT_INITIALIZED"
    ErrInternal         ErrorCode = "INTERNAL"
)

type OpenVikingError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Details map[string]any
}

func (e *OpenVikingError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *OpenVikingError) Unwrap() error { return e.Cause }
```

---

## 4. `pkg/vikingfs/` — Go-Native Filesystem Engine

### 4.1 Filesystem Interface

```go
// pkg/vikingfs/fs.go

type FileSystem interface {
    Read(ctx context.Context, path string) ([]byte, error)
    ReadRange(ctx context.Context, path string, offset, size int64) ([]byte, error)
    Write(ctx context.Context, path string, data []byte) error
    Mkdir(ctx context.Context, path string, existOK bool) error
    Rm(ctx context.Context, path string, recursive bool) error
    Stat(ctx context.Context, path string) (*FileInfo, error)
    Exists(ctx context.Context, path string) (bool, error)
    Ls(ctx context.Context, path string) ([]DirEntry, error)
    Mv(ctx context.Context, oldPath, newPath string) error
    Cp(ctx context.Context, srcPath, dstPath string) error
}

type FileInfo struct {
    URI         string
    Name        string
    IsDirectory bool
    Size        int64
    ModTime     time.Time
    Mode        os.FileMode
}

type DirEntry struct {
    Name        string
    URI         string
    IsDirectory bool
    Size        int64
    ModTime     time.Time
    Abstract    string  // Content of .abstract.md if exists, empty otherwise
}

// LocalFileSystem implements FileSystem on local disk
// URI → LocalPath mapping: viking://resources/foo/ → {workspace}/data/resources/foo/
type LocalFileSystem struct {
    workspace string  // e.g., "~/.openviking/data"
}

func (fs *LocalFileSystem) uriToPath(uri string) (string, error) {
    if !strings.HasPrefix(uri, "viking://") {
        return "", fmt.Errorf("not a viking URI: %s", uri)
    }
    rel := strings.TrimPrefix(uri, "viking://")
    cleaned := filepath.Clean(rel)
    return filepath.Join(fs.workspace, cleaned), nil
}
```

### 4.2 PathLock — Distributed Locking

```go
// pkg/vikingfs/lock.go

// PathLock implements hierarchical path-based locking.
// Uses sync.Map for goroutine-local lock tracking.
// For distributed deployments, swap to Redis-based implementation.
type PathLock struct {
    mu    sync.Mutex
    locks map[string]*lockEntry
}

type lockEntry struct {
    mu       sync.RWMutex
    refCount int
}

type LockReleaser func()

// AcquirePoint acquires an exclusive lock on a single path.
// Used for: session commit writes, single file operations.
func (pl *PathLock) AcquirePoint(ctx context.Context, path string) (LockReleaser, error) {
    pl.mu.Lock()
    entry := pl.getOrCreateEntry(path)
    pl.mu.Unlock()
    
    // Try to acquire with context cancellation
    acquired := make(chan struct{}, 1)
    go func() {
        entry.mu.Lock()
        acquired <- struct{}{}
    }()
    
    select {
    case <-acquired:
        return func() { entry.mu.Unlock(); pl.cleanup(path) }, nil
    case <-ctx.Done():
        return nil, &viking.OpenVikingError{Code: viking.ErrResourceBusy, Message: "path lock timeout: " + path}
    }
}

// AcquireSubtree acquires an exclusive lock on a path and all its children.
// Used for: rm -rf on directories.
func (pl *PathLock) AcquireSubtree(ctx context.Context, path string) (LockReleaser, error) {
    // Lock both the exact path AND acquire a "subtree sentinel"
    // All child locks must check if a subtree lock on parent exists
    return pl.AcquirePoint(ctx, path+"/*") // Convention: /* suffix = subtree lock
}

// AcquireMv acquires locks on source path + destination parent.
// Used for: mv operations to prevent race between mv and rm.
func (pl *PathLock) AcquireMv(ctx context.Context, srcPath, dstParentPath string) (LockReleaser, error) {
    paths := []string{srcPath, dstParentPath}
    sort.Strings(paths) // Consistent order to prevent deadlock
    
    releasers := make([]LockReleaser, 0, len(paths))
    for _, p := range paths {
        r, err := pl.AcquirePoint(ctx, p)
        if err != nil {
            for _, release := range releasers {
                release()
            }
            return nil, err
        }
        releasers = append(releasers, r)
    }
    
    return func() {
        for _, r := range releasers {
            r()
        }
    }, nil
}
```

---

## 5. `pkg/adapters/` — Infrastructure Interfaces

### 5.1 VectorDB Adapter

```go
// pkg/adapters/vectordb/interface.go

type CollectionSchema struct {
    Name        string
    DenseDim    int
    SparseEnabled bool
}

type ScoredContext struct {
    URI         string
    ParentURI   string
    ContextType viking.ContextType
    Level       int
    Abstract    string
    Score       float64
    ActiveCount int64
}

type ContextVector struct {
    URI            string
    ParentURI      string
    ContextType    viking.ContextType
    Level          int
    OwnerAccountID string
    OwnerUserID    string
    Abstract       string
    ActiveCount    int64
    DenseVector    []float32
    SparseVector   map[string]float32
}

type VectorStore interface {
    CreateCollection(ctx context.Context, schema CollectionSchema) error
    CollectionExists(ctx context.Context, name string) (bool, error)
    DropCollection(ctx context.Context, name string) error
    
    // Global search across ALL L0/L1 nodes (no parent filter)
    SearchGlobalRoots(ctx context.Context, dense []float32, sparse map[string]float32,
        accountID string, topK int) ([]ScoredContext, error)
    
    // Children search within a specific parent directory
    SearchChildren(ctx context.Context, parentURI string, dense []float32,
        sparse map[string]float32, accountID string) ([]ScoredContext, error)
    
    UpsertContext(ctx context.Context, vec ContextVector) error
    DeleteContext(ctx context.Context, uri string) error
    UpdateActiveCount(ctx context.Context, uri string, delta int64) error
}

// pkg/adapters/vectordb/qdrant/client.go — Qdrant implementation
// Uses qdrant-go SDK; collection per account: "openviking_{account_id}"
// HNSW index params: m=16, ef_construction=128

// pkg/adapters/vectordb/weaviate/client.go — Weaviate implementation
// pkg/adapters/vectordb/embedded/client.go — In-process embedded (for testing)
```

### 5.2 Embedder Adapter (12 providers via Bifrost)

```go
// pkg/adapters/embedder/interface.go

type EmbedResult struct {
    DenseVector  []float32
    SparseVector map[string]float32  // BM25/SPLADE; empty if not supported
}

type EmbedderClient interface {
    Embed(ctx context.Context, text string, isQuery bool) (*EmbedResult, error)
    EmbedBatch(ctx context.Context, texts []string) ([]*EmbedResult, error)
    Dimension() int
    SupportsSparse() bool
    ProviderName() string
}

// pkg/adapters/embedder/bifrost/client.go
// Routes all 12 providers through Bifrost gateway:
// OpenAI, Volcengine, Gemini, Jina, Cohere, DashScope, MiniMax,
// Voyage, LiteLLM, VikingDB, ONNX local, Hugging Face

type BifrostEmbedder struct {
    httpClient  *http.Client
    baseURL     string  // Bifrost gateway URL
    model       string
    provider    string
    dimension   int
    apiKey      string
}

func (e *BifrostEmbedder) Embed(ctx context.Context, text string, isQuery bool) (*EmbedResult, error) {
    reqBody := map[string]any{
        "provider": e.provider,
        "model":    e.model,
        "input":    text,
        "is_query": isQuery,  // Bifrost handles query vs doc asymmetric encoding
    }
    // POST {bifrost_url}/v1/embeddings
    // Returns {dense_vector: [], sparse_vector: {}}
    resp, err := e.postJSON(ctx, "/v1/embeddings", reqBody)
    return parseEmbedResult(resp, err)
}
```

### 5.3 VLM Adapter

```go
// pkg/adapters/vlm/interface.go

type VLMOption func(*VLMConfig)
type VLMConfig struct {
    Model       string
    MaxTokens   int
    Temperature float64
    JSONSchema  any   // For structured output
}

func WithVLMModel(m string) VLMOption    { return func(c *VLMConfig) { c.Model = m } }
func WithVLMMaxTokens(n int) VLMOption   { return func(c *VLMConfig) { c.MaxTokens = n } }

type VLMClient interface {
    Generate(ctx context.Context, prompt string, opts ...VLMOption) (string, error)
    GenerateWithImage(ctx context.Context, prompt string, image []byte, opts ...VLMOption) (string, error)
    GenerateStructured(ctx context.Context, prompt string, schema any, opts ...VLMOption) (json.RawMessage, error)
}

// Providers: OpenAI (gpt-4o), Volcengine (Skylark2), Gemini, Kimi, GLM-4V, LiteLLM proxy
// All route through Bifrost for unified auth + fallback
```

### 5.4 Reranker Adapter

```go
// pkg/adapters/reranker/interface.go

type RerankResult struct {
    Index     int
    Score     float64
    Document  string
}

type RerankerClient interface {
    Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
    ProviderName() string
}

// Implementations:
// pkg/adapters/reranker/jina/     — jina-reranker-v2-base-multilingual
// pkg/adapters/reranker/cohere/   — rerank-multilingual-v3.0
// pkg/adapters/reranker/openai/   — via embedding similarity
// pkg/adapters/reranker/local/    — ONNX cross-encoder local model
// pkg/adapters/reranker/disabled/ — no-op, returns original order
```

### 5.5 KMS Adapter

```go
// pkg/adapters/kms/interface.go

type KMSProvider interface {
    GetRootKey(ctx context.Context) ([]byte, error)
    DeriveAccountKey(ctx context.Context, accountID string) ([]byte, error)  // HKDF(root, account_id, "openviking-account-key")
    RotateRootKey(ctx context.Context) error
    ProviderType() byte  // 0x01=Local, 0x02=Vault, 0x03=Cloud
}

// pkg/adapters/kms/local/    — Key from local file ~/.openviking/root.key
// pkg/adapters/kms/vault/    — HashiCorp Vault KV engine
// pkg/adapters/kms/cloud/    — AWS KMS | GCP KMS | Volcengine KMS
```

---

## 6. `pkg/resilience/` — Circuit Breaker, Retry, Bulkhead

```go
// pkg/resilience/circuit_breaker.go
// sony/gobreaker wrapper with OpenViking defaults:
// MaxRequests=3, Interval=10s, Timeout=30s, MinRequests=10

// pkg/resilience/retry.go
// Exponential backoff with jitter:
// Base=100ms, Max=10s, MaxAttempts=3, Jitter=±20%
// Only retries on: gRPC UNAVAILABLE, DEADLINE_EXCEEDED, RESOURCE_EXHAUSTED

// pkg/resilience/bulkhead.go
// Go channel-based semaphore:
// MaxConcurrentLLM=10, MaxConcurrentEmbed=20
// Returns ErrResourceBusy if semaphore full + ctx timeout
```

---

## 7. `pkg/middleware/` — Interceptors

### 7.1 Auth Middleware

```go
// pkg/middleware/auth/http.go

// Reads: X-OpenViking-Account, X-OpenViking-User, X-OpenViking-Agent, X-Api-Key
// Sets: pkg/viking.RequestContext in context.Context
// Modes:
//   DEV:     auto ROOT role, only accept 127.0.0.1/::1
//   API_KEY: resolve via Admin.ResolveAPIKey gRPC call + Redis cache
//   TRUSTED: read headers directly (for internal service mesh)

// pkg/middleware/auth/grpc.go
// Same logic but reads gRPC metadata:
// "x-account-id", "x-user-id", "x-agent-id", "x-api-key"
```

### 7.2 Rate Limit Middleware

```go
// pkg/middleware/ratelimit/redis_sliding_window.go

// Key: "ov_rl:{account_id}:{endpoint_bucket}:{minute}"
// Returns: 429 Too Many Requests + Retry-After header
// Config per endpoint, per tenant, configurable via Admin API
```

---

## 8. `pkg/parse/` — File Parser Registry

```go
// pkg/parse/registry.go

type Parser interface {
    Parse(ctx context.Context, path string, content []byte) ([]Chunk, error)
    SupportedExtensions() []string
}

type Chunk struct {
    Content   string
    StartLine int
    EndLine   int
    Language  string
    ChunkType string          // "function" | "class" | "paragraph" | "section" | "raw"
    Metadata  map[string]any  // function name, class name, etc.
}

type Registry struct {
    parsers map[string]Parser  // extension → parser
}

func NewRegistry(vlmClient adapters.VLMClient) *Registry {
    r := &Registry{parsers: make(map[string]Parser)}
    // Register all parsers
    ts := &TreeSitterParser{}
    r.registerAll(ts.SupportedExtensions(), ts)
    
    md := &MarkdownParser{}
    r.registerAll(md.SupportedExtensions(), md)
    
    doc := &DocumentParser{}
    r.registerAll(doc.SupportedExtensions(), doc)
    
    if vlmClient != nil {
        vlm := &VLMParser{client: vlmClient}
        r.registerAll(vlm.SupportedExtensions(), vlm)
    }
    
    txt := &TextParser{}
    r.registerAll(txt.SupportedExtensions(), txt)
    return r
}

// TreeSitterParser: .go, .py, .js, .ts, .rs, .java, .c, .cpp, .rb, .php, .swift, .kt (50+ langs)
// Uses smacker/go-tree-sitter — splits at function/class boundaries
// → Chunk{ChunkType:"function", Metadata:{"name":"ParseURI","file":"uri.go"}}

// MarkdownParser: .md, .mdx — splits at heading boundaries (## → new chunk)
// DocumentParser: .pdf (unidoc), .docx (gooxml), .xlsx, .epub
// VLMParser: .png, .jpg, .pptx — calls VLM to describe image content
// TextParser: .txt, .yaml, .toml, .json, .csv — sliding window chunking
```

---

## 9. `pkg/nats/` — JetStream Helpers

```go
// pkg/nats/publisher.go

type Publisher struct {
    js nats.JetStreamContext
}

func (p *Publisher) Publish(ctx context.Context, subject string, payload any) error {
    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    _, err = p.js.Publish(subject, data)
    return err
}

// pkg/nats/subscriber.go

type MessageHandler func(msg *nats.Msg) error

func Subscribe(js nats.JetStreamContext, subject, durable string, handler MessageHandler) (*nats.Subscription, error) {
    return js.Subscribe(subject, func(msg *nats.Msg) {
        if err := handler(msg); err != nil {
            msg.Nak()  // Redeliver
            return
        }
        msg.Ack()
    },
    nats.Durable(durable),
    nats.MaxDeliver(3),
    nats.AckWait(30*time.Second),
    )
}
```

---

## 10. `pkg/auth/` — API Key Validation

```go
// pkg/auth/key_resolver.go

type ResolvedKey struct {
    AccountID string
    UserID    string
    Role      viking.Role
    KeyID     string
}

type KeyResolver interface {
    Resolve(ctx context.Context, apiKey string) (*ResolvedKey, error)
}

// Cached implementation: Redis cache TTL=5min for successful resolutions
type CachedKeyResolver struct {
    adminClient admin.AdminServiceClient  // gRPC call to admin service
    redis       *redis.Client
    cacheTTL    time.Duration
}

func (r *CachedKeyResolver) Resolve(ctx context.Context, apiKey string) (*ResolvedKey, error) {
    // 1. Try Redis cache first
    cacheKey := "ovkey:" + hashPrefix(apiKey)  // Only hash first 16 chars
    cached, err := r.redis.Get(ctx, cacheKey).Bytes()
    if err == nil {
        var result ResolvedKey
        json.Unmarshal(cached, &result)
        return &result, nil
    }
    
    // 2. Admin service gRPC call
    resp, err := r.adminClient.ResolveAPIKey(ctx, &adminv1.ResolveAPIKeyRequest{Key: apiKey})
    if err != nil {
        return nil, err
    }
    
    // 3. Cache result
    data, _ := json.Marshal(toResolvedKey(resp))
    r.redis.Set(ctx, cacheKey, data, r.cacheTTL)
    
    return toResolvedKey(resp), nil
}
```

---

## 11. `pkg/observability/` — OTel + Prometheus

```go
// Key metrics:
var (
    VectorSearches = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "openviking_vector_searches_total",
    }, []string{"account", "collection", "search_type"})
    
    MemoryExtracted = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "openviking_memory_extracted_total",
    }, []string{"account", "category"})
    
    SessionCommits = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "openviking_session_commits_total",
    }, []string{"account", "phase"})
    
    HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "openviking_http_request_duration_seconds",
        Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1},
    }, []string{"method", "path", "status"})
    
    ResourceIngested = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "openviking_resource_ingested_total",
    }, []string{"account", "source_type"})
)
```

---

## 12. `ov` CLI — Developer Workflow

```go
// CLI implementation: Go (không dùng Rust, để giảm toolchain complexity)
// Package: cmd/ov/main.go — cobra-based CLI

// Core commands:
// ov status                          → GET /api/v1/system/status
// ov add-resource <url> [--watch]    → POST /api/v1/resources
// ov ls <viking-uri>                 → GET /api/v1/filesystem/ls
// ov tree <viking-uri> -L <depth>    → GET /api/v1/filesystem/tree
// ov find "<query>" [--type]         → POST /api/v1/search/find
// ov grep "<pattern>" --uri <uri>    → POST /api/v1/search/grep
// ov read <viking-uri> [--level 0|1|2] → GET /api/v1/content/read
// ov session commit                  → POST /api/v1/sessions/{id}/commit

// Config file: ~/.openviking/config.yaml
// Config init: ov config init → interactive wizard
// Config doctor: ov config doctor → validate connectivity
```

---

## 13. Python SDK

```python
# openviking/client.py — Generated from OpenAPI spec or manual wrapper

class OpenViking:
    def __init__(self, url: str, api_key: str, timeout: int = 30):
        self._base_url = url.rstrip('/')
        self._headers = {
            "X-Api-Key": api_key,
            "Content-Type": "application/json",
        }
        self._timeout = timeout
    
    def find(self, query: str, limit: int = 10, type: str = None, threshold: float = 0.0) -> list[SearchResult]:
        ...
    
    def read(self, uri: str, level: int = 2) -> str:
        ...
    
    def write(self, uri: str, content: str) -> None:
        ...
    
    def ls(self, uri: str) -> list[FileEntry]:
        ...
    
    def tree(self, uri: str, depth: int = 3) -> TreeNode:
        ...
    
    def add_resource(self, url_or_path: str, name: str, watch: bool = False) -> Task:
        ...
    
    def create_session(self, user_id: str = None) -> Session:
        ...
    
    def forget(self, uri: str) -> None:
        ...

class AsyncOpenViking:
    """Async version using aiohttp."""
    # Same interface but all methods are async
    
    async def find(self, query: str, **kwargs) -> list[SearchResult]:
        ...
```

---

## 14. Testing Strategy

### Unit Tests
- `TestValidateURI_PathTraversal` — `"viking://../escape"` → `ErrInvalidArgument`
- `TestValidateURI_Backslash` — `"viking://user\\alice"` → `ErrInvalidArgument`
- `TestCanonicalizeURI_DoubleSlash` — `"viking://user//alice/"` → `"viking://user/alice/"`
- `TestResolveOwner_UserURI` — `"viking://user/acct1/alice/mem/"` → `"acct1", "alice", ""`
- `TestPathLock_ConcurrentAccess` — 20 goroutines same path → only 1 enters critical section
- `TestPathLock_SubtreePreventsChild` — subtree lock → child lock blocks
- `TestPathLock_MvDeadlockPrevention` — paths sorted → no deadlock
- `TestRegistryParse_GoFile` — tree-sitter splits at function boundaries
- `TestCachedKeyResolver_CacheHit` — second call within TTL → no gRPC call
- `TestCircuitBreaker_OpenAfterFailures` — 10 failures → circuit open

### Integration Tests (testcontainers)
- `TestQdrantAdapter_UpsertAndSearch` — real Qdrant container
- `TestLocalFileSystem_WriteReadDelete` — real disk operations
- `TestPathLock_Distributed` — Redis-based locking across goroutines

---

## 15. Go Module & Dependency Summary

```go
// go.mod additions needed for pkg/

require (
    // Viking FS
    github.com/smacker/go-tree-sitter v0.0.0-20230720070738-0d0a9f78d8f8
    github.com/yuin/goldmark v1.7.1
    github.com/unidoc/unipdf/v3 v3.55.0
    
    // Vector DB
    github.com/qdrant/go-client v1.9.0
    
    // Resilience
    github.com/sony/gobreaker v0.5.0
    
    // Observability
    go.opentelemetry.io/otel v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0
    github.com/prometheus/client_golang v1.19.0
    
    // Auth
    golang.org/x/crypto v0.22.0        // bcrypt
    github.com/redis/go-redis/v9 v9.5.1
    
    // NATS
    github.com/nats-io/nats.go v1.34.1
    
    // Vault
    github.com/hashicorp/vault/api v1.12.2
)
```

---

## 16. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| tree-sitter grammars phải compile per OS/arch | Trung bình | Precompile grammars, bundle trong Docker image |
| unidoc (PDF parsing) license cost | Trung bình | Evaluate pdfcpu (MIT) hoặc unipdf community edition |
| Bifrost gateway single point of failure | Trung bình | Circuit breaker per provider + direct fallback to OpenAI |
| PathLock chỉ in-process (không distributed) | Trung bình | Acceptable cho monolith; swap to Redis Lua script khi horizontal scale |
| VLMParser latency tăng ingestion time | Thấp | Goroutine pool (max 2 VLM workers) + VLM optional |
