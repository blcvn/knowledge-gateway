# Change Request: CR-OV-007 — Shared Infrastructure (`pkg/`) & CLI Integration

**CR ID:** CR-OV-007  
**Component:** `pkg/` (Shared Packages) + `ov` CLI + Python SDK [NEW PACKAGES]  
**Priority:** High  
**Status:** Implemented
**Reference:** OpenViking PRD §5.2 §7.2, SRS §2.8-2.11, specs/services/08-shared-packages.md, URD §2.1-2.2

---

## 1. Mô tả

Xây dựng **toàn bộ shared infrastructure** — code dùng chung giữa 7 services, không chứa business logic:

1. **`pkg/viking/`**: Shared domain types (Context, ContextType, ContextLevel, Namespace, Identity, Errors).
2. **`pkg/adapters/`**: Infrastructure interfaces + implementations (VectorDB, Embedder, VLM, KMS, Reranker).
3. **`pkg/vikingfs/`**: Go-native filesystem engine (thay thế RAGFS Rust).
4. **`pkg/middleware/`**: Shared gRPC/HTTP interceptors (auth, logging, tracing, ratelimit, recovery).
5. **`pkg/resilience/`**: Circuit breaker, retry, bulkhead patterns.
6. **`pkg/observability/`**: OTel tracer, Prometheus metrics, slog structured logger, health endpoints.
7. **`pkg/nats/`**: NATS JetStream publisher/subscriber helpers.
8. **`pkg/parse/`**: File parser registry (50+ extensions, tree-sitter Go bindings).
9. **`pkg/auth/`**: API key validator, bcrypt comparison, Redis cache for key resolution.
10. **`ov` CLI**: Rust CLI (via `crates/ov_cli`) — hoặc Go rewrite — cho developer workflow.
11. **Python SDK**: `openviking` Python package (sync + async clients).
12. **IDE Plugins**: Claude Code, OpenCode, Codex memory plugin examples.

---

## 2. Vấn đề hiện tại

- VNP Memory thiếu shared `pkg/` library chuẩn hóa → mỗi service tự implement utilities → code duplication.
- Thiếu unified EmbedderClient interface cho 12+ providers.
- Thiếu standardized middleware stack cho gRPC.
- Chưa có CLI tool cho developer workflow.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `pkg/viking/` — Shared Domain Types

```go
// pkg/viking/context.go
type Context struct {
    URI            string            `json:"uri"`           // viking:// URI (Primary Key)
    ParentURI      string            `json:"parent_uri"`
    ContextType    ContextType       `json:"context_type"`  // MEMORY|RESOURCE|SKILL|SESSION
    Level          int               `json:"level"`         // 0=Abstract, 1=Overview, 2=Detail
    OwnerAccountID string            `json:"owner_account_id"`
    OwnerUserID    string            `json:"owner_user_id"`
    OwnerAgentID   string            `json:"owner_agent_id"`
    Abstract       string            `json:"abstract"`      // L0 text (~100 tokens)
    Category       string            `json:"category"`
    ActiveCount    int64             `json:"active_count"`  // Usage counter (hotness)
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
    LevelAbstract ContextLevel = 0  // .abstract.md
    LevelOverview ContextLevel = 1  // .overview.md
    LevelDetail   ContextLevel = 2  // Raw file
)

// pkg/viking/namespace.go
const (
    RootResources = "viking://resources/"
    RootUser      = "viking://user/"
    RootAgent     = "viking://agent/"
    RootSession   = "viking://session/"
    RootTemp      = "viking://temp/"
)

func CanonicalizeURI(uri string) (string, error)
func ResolveOwner(uri string) (accountID, userID, agentID string, err error)
func IsAccessible(uri string, ctx *RequestContext) bool
func ValidateURI(uri string) error  // Rejects: .., \, drive letters

// pkg/viking/tiered.go — File naming conventions
func ToAbstractURI(fileURI string) string { return fileURI + ".abstract.md" }
func ToOverviewURI(fileURI string) string { return fileURI + ".overview.md" }

// pkg/viking/errors.go
type OpenVikingError struct {
    Code    ErrorCode
    Message string
    Details map[string]any
}

const (
    ErrInvalidArgument    ErrorCode = "INVALID_ARGUMENT"
    ErrUnauthenticated    ErrorCode = "UNAUTHENTICATED"
    ErrPermissionDenied   ErrorCode = "PERMISSION_DENIED"
    ErrNotFound           ErrorCode = "NOT_FOUND"
    ErrAlreadyExists      ErrorCode = "ALREADY_EXISTS"
    ErrResourceBusy       ErrorCode = "RESOURCE_BUSY"  // PathLock busy
    ErrNotInitialized     ErrorCode = "NOT_INITIALIZED"
    ErrInternal           ErrorCode = "INTERNAL"
)

// pkg/viking/identity.go
type UserIdentifier struct {
    AccountID string
    UserID    string
    AgentID   string
}

type Role int
const (
    RoleUser  Role = 0
    RoleBot   Role = 1  // Agent-scoped access
    RoleAdmin Role = 2
    RoleRoot  Role = 3
)

type RequestContext struct {
    User            UserIdentifier
    Role            Role
    NamespacePolicy string
    APIKeyID        string
}
```

### 3.2. [NEW] `pkg/adapters/` — Infrastructure Interfaces

```go
// pkg/adapters/vectordb/interface.go
type VectorStore interface {
    CreateCollection(ctx context.Context, name string, schema CollectionSchema) error
    CollectionExists(ctx context.Context, name string) (bool, error)
    SearchGlobalRoots(ctx context.Context, dense []float32, sparse map[string]float32,
        accountID string, topK int) ([]ScoredContext, error)
    SearchChildren(ctx context.Context, parentURI string, dense []float32,
        sparse map[string]float32, accountID string) ([]ScoredContext, error)
    UpsertContext(ctx context.Context, vec ContextVector) error
    DeleteContext(ctx context.Context, uri string) error
    UpdateActiveCount(ctx context.Context, uri string, delta int64) error
}
// Implementations: Qdrant, Weaviate, embedded

// pkg/adapters/embedder/interface.go
type EmbedderClient interface {
    Embed(ctx context.Context, text string, isQuery bool) (*EmbedResult, error)
    EmbedBatch(ctx context.Context, texts []string) ([]*EmbedResult, error)
    Dimension() int
    SupportsSparse() bool
}
type EmbedResult struct {
    DenseVector  []float32
    SparseVector map[string]float32  // empty if provider doesn't support sparse
}
// 12 Providers via Bifrost:
// OpenAI, Volcengine, Gemini, Jina, Cohere, DashScope, MiniMax, Voyage, LiteLLM, VikingDB, Local(ONNX)

// pkg/adapters/vlm/interface.go
type VLMClient interface {
    Generate(ctx context.Context, prompt string, opts ...VLMOption) (string, error)
    GenerateWithImage(ctx context.Context, prompt string, image []byte, opts ...VLMOption) (string, error)
    GenerateStructured(ctx context.Context, prompt string, schema any) (json.RawMessage, error)
}
// Providers: OpenAI, Volcengine, Gemini, Kimi, GLM, LiteLLM

// pkg/adapters/reranker/interface.go
type RerankerClient interface {
    Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}
type RerankResult struct {
    Index     int
    Score     float64
    Document  string
}
// Providers: Volcengine, OpenAI, Cohere, Jina, local models
```

### 3.3. [NEW] `pkg/vikingfs/` — Go-Native Filesystem Engine

```go
// pkg/vikingfs/fs.go
type FileSystem interface {
    Read(path string, offset, size int64) ([]byte, error)
    Write(path string, data []byte) error
    Mkdir(path string, existOK bool) error
    Rm(path string, recursive bool) error
    Stat(path string) (*FileInfo, error)
    Exists(path string) (bool, error)
    Ls(path string) ([]DirEntry, error)
    Mv(oldPath, newPath string) error
    Cp(srcPath, dstPath string) error
}

// URI → local path mapping
// viking://resources/myrepo/ → {workspace}/data/resources/myrepo/
// viking://user/acct1/alice/ → {workspace}/data/user/acct1/alice/

// pkg/vikingfs/lock.go
type PathLock struct {
    mu    sync.Map
    locks map[string]*pathEntry
}

func (pl *PathLock) AcquirePoint(path string) (releaser func(), err error)
func (pl *PathLock) AcquireSubtree(path string) (releaser func(), err error)
func (pl *PathLock) AcquireMv(srcPath, dstParentPath string) (releaser func(), err error)
```

### 3.4. [NEW] `pkg/middleware/` — Interceptors

```go
// pkg/middleware/auth/
// HTTP: reads X-OpenViking-Account, X-OpenViking-User, X-OpenViking-Agent, X-Api-Key
// gRPC: reads metadata "x-account-id", "x-user-id", "x-agent-id", "x-api-key"
// Sets RequestContext in context.Context

// pkg/middleware/logging/
// Structured access log per request:
// {"level":"info","method":"POST","path":"/api/v1/search","duration_ms":45,"status":200,"account":"acct1"}

// pkg/middleware/tracing/
// OTel span per gRPC call, propagate trace context

// pkg/middleware/ratelimit/
// Redis sliding window: per tenant + per endpoint
// Returns 429 with Retry-After header

// pkg/middleware/recovery/
// Catch panics → log stack trace → return gRPC INTERNAL error
```

### 3.5. [NEW] `pkg/resilience/`

```go
// pkg/resilience/circuit_breaker.go
// sony/gobreaker per downstream service
// Settings: maxRequests=3, interval=10s, timeout=30s, minRequests=10

// pkg/resilience/retry.go
// Exponential backoff: base=100ms, max=10s, maxAttempts=3
// Jitter: ±20% random to avoid thundering herd

// pkg/resilience/bulkhead.go
// Go channel semaphore for LLM/Embed calls
// Configurable: max_concurrent_llm=10, max_concurrent_embed=20
```

### 3.6. [NEW] `pkg/observability/`

```go
// Traces: OTel SDK → OTLP → Jaeger/Tempo
// Metrics: OTel → Prometheus → Grafana dashboards
// Logs: log/slog structured JSON + OTel log bridge
// Health: gRPC Health v1 + HTTP /healthz /readyz /livez /metrics

// Key Metrics:
// openviking_vector_searches_total{account, collection}
// openviking_vector_scored{account, collection}
// openviking_memory_extracted_total{account, category}
// openviking_http_request_duration_seconds{method, path, status}
// openviking_session_commits_total{account}
// openviking_resource_ingested_total{account, source_type}
```

### 3.7. [NEW] `pkg/parse/` — File Parser Registry

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
    Metadata   map[string]any  // function name, class name, etc.
}

// Registry: auto-register all parsers, 50+ extensions
type Registry struct{ parsers map[string]Parser }
func NewRegistry() *Registry  // registers all parsers

// Parsers:
// TreeSitterParser: .go, .py, .js, .ts, .rs, .java, .c, .cpp, ... (smacker/go-tree-sitter)
// MarkdownParser:   .md, .mdx (goldmark)
// DocumentParser:   .pdf, .docx, .xlsx, .epub (unidoc, gooxml)
// VLMParser:        .png, .jpg, .pptx (via VLM client)
// TextParser:       .txt, .yaml, .toml, .json, .csv (native)
```

### 3.8. `ov` CLI — Developer Workflow Tool

```
# Core commands (maps to REST API):
ov status                              → GET /api/v1/system/status
ov add-resource <url|path> [--watch]   → POST /api/v1/resources
ov ls <viking-uri>                     → GET /api/v1/filesystem/ls
ov tree <viking-uri> -L <depth>        → GET /api/v1/filesystem/tree
ov find "<query>" [--type memory|resource|skill]  → POST /api/v1/search/find
ov grep "<pattern>" --uri <viking-uri> → POST /api/v1/search/grep
ov read <viking-uri> [--level 0|1|2]   → GET /api/v1/content/read
ov chat                                → Interactive VikingBot session
ov session commit                      → POST /api/v1/sessions/{id}/commit

# Config
ov config init                         → Interactive setup wizard
ov config doctor                       → Validate configuration
```

### 3.9. Python SDK

```python
# Sync client — openviking.py
from openviking import OpenViking, AsyncOpenViking

# Sync
client = OpenViking(url="http://localhost:1933", api_key="ovu_xxx")
result = client.find("authentication flow", limit=5)
session = client.create_session()
session.add_message("user", "How does auth work?")
session.commit()

# Async
async def main():
    client = AsyncOpenViking(url="http://localhost:1933", api_key="ovu_xxx")
    results = await client.find("database schema", limit=10)
    await client.add_resource("https://github.com/org/repo", name="my-repo")

# SDK Methods:
# find(query, limit, type) → List[SearchResult]
# read(uri, level=2) → str
# ls(uri) → List[FileEntry]
# tree(uri, depth) → TreeNode
# write(uri, content) → None
# forget(uri) → None
# add_resource(url_or_path, name, watch=False) → Task
# create_session(user_id) → Session
# session.add_message(role, content) → None
# session.commit() → CommitResult
```

### 3.10. IDE Plugin Examples

| Plugin | Giao thức | Repo path |
|--------|----------|-----------|
| **Claude Code Memory** | MCP + CLI | `examples/claude-code-memory-plugin/` |
| **OpenCode Memory** | MCP + CLI | `examples/opencode-memory-plugin/` |
| **Codex Memory** | MCP | `examples/codex-memory-plugin/` |
| **OpenClaw Context** | Plugin API | `examples/openclaw-plugin/` |

**Claude Code setup:**
```json
// ~/.claude/settings.json
{
  "mcpServers": {
    "openviking": {
      "type": "http",
      "url": "http://localhost:8082/mcp",
      "headers": {
        "X-OpenViking-Account": "my-account",
        "X-OpenViking-User": "alice"
      }
    }
  }
}
```

---

## 4. Monorepo Structure

```
openviking-go/
├── api/proto/
│   ├── common/v1/    # pagination.proto, errors.proto, health.proto
│   ├── viking/v1/    # context.proto, namespace.proto, identity.proto
│   ├── gateway/v1/
│   ├── fs/v1/        # FileSystemService
│   ├── search/v1/    # SearchService
│   ├── session/v1/   # SessionService
│   ├── resource/v1/  # ResourceService
│   ├── crypto/v1/    # CryptoService
│   └── admin/v1/     # AdminService
├── services/
│   ├── openviking-gateway/
│   ├── openviking-fs/
│   ├── openviking-search/
│   ├── openviking-session/
│   ├── openviking-resource/
│   ├── openviking-crypto/
│   └── openviking-admin/
├── pkg/
│   ├── viking/         # Shared domain types
│   ├── adapters/       # VectorDB, Embedder, VLM, KMS, Reranker
│   ├── vikingfs/       # Go-native FS engine
│   ├── middleware/     # auth, logging, tracing, ratelimit, recovery
│   ├── resilience/     # circuit breaker, retry, bulkhead
│   ├── observability/  # OTel, Prometheus, slog, health
│   ├── nats/           # NATS JetStream helpers
│   ├── auth/           # API key validation + Redis cache
│   ├── tenant/         # Tenant context extraction
│   ├── pagination/     # Cursor/offset pagination
│   ├── parse/          # File parser registry
│   └── testutil/       # Fixtures, mocks, testcontainers
├── examples/
│   ├── claude-code-memory-plugin/
│   ├── opencode-memory-plugin/
│   ├── codex-memory-plugin/
│   └── openclaw-plugin/
├── deploy/
│   ├── docker-compose/  # dev environment
│   └── kubernetes/      # Kustomize overlays
├── go.mod
├── buf.yaml             # Protobuf buf toolchain config
└── Makefile
```

---

## 5. Acceptance Criteria

### Shared Packages
- [ ] `pkg/viking.ValidateURI("../escape")` → returns ErrInvalidArgument (path traversal blocked).
- [ ] `pkg/viking.CanonicalizeURI("viking://user//alice/")` → returns `"viking://user/alice/"` (normalized).
- [ ] `pkg/adapters/embedder.OpenAIEmbedder.Embed("hello world", false)` → returns dense vector of correct dimension.
- [ ] `pkg/resilience.CircuitBreaker`: after 5 failures → circuit opens → subsequent calls fail immediately without calling downstream.
- [ ] `pkg/parse.Registry.Parse("main.go", content)` → chunks split at function boundaries (tree-sitter).
- [ ] `pkg/observability`: Prometheus `/metrics` endpoint has `openviking_http_request_duration_seconds` histogram.

### CLI
- [ ] `ov status` → prints server health when server running; prints error when server not running.
- [ ] `ov add-resource https://github.com/golang/go` → shows progress, completes with resource URI.
- [ ] `ov find "goroutine scheduling"` → returns ranked results with relevance scores.
- [ ] `ov read viking://resources/golang/src/runtime/proc.go --level 0` → prints L0 abstract.

### Python SDK
- [ ] `client.find("auth flow", limit=5)` → returns list of SearchResult with uri, abstract, score.
- [ ] `client.create_session()` → returns Session object with `session_id`.
- [ ] `AsyncOpenViking.find("test")` → works in async context (pytest-asyncio).

### IDE Integration
- [ ] Claude Code connects to MCP at `http://localhost:8082/mcp` → sees 9 tools.
- [ ] Claude uses `search` tool → response includes viking URIs and abstract summaries.
