# Shared Packages (`backend/shared/pkg/`)

> **Rule:** NO business logic — chỉ chứa types, interfaces, middleware, adapters.
> **Updated**: 2026-09-03 — synced từ `go.work` và `backend/shared/pkg/` actual directory.

---

## 1. Package Inventory (từ `go.work` và filesystem)

| Package | Go Module | Description | Key consumers |
|---------|-----------|-------------|---------------|
| `shared/pkg/adapters` | `vnp-memory/shared/pkg/adapters` | External adapter implementations (LLM, DB, storage) | Memobase-engine, Cognify, Graphiti-knowledge |
| `shared/pkg/config` | `vnp-memory/shared/pkg/config` | Viper-based config loader + validator | All services |
| `shared/pkg/forward` | `vnp-memory/shared/pkg/forward` | HTTP forward proxy helpers | Gateway |
| `shared/pkg/graph` | `vnp-memory/shared/pkg/graph` | Shared graph domain types (Node, Edge, Community) | Cognee-cognify, Graphiti-store |
| `shared/pkg/privacy` | `vnp-memory/shared/pkg/privacy` | PII redaction engine | Observe-service (hook payload scrubbing) |
| `shared/pkg/resilience` | `vnp-memory/shared/pkg/resilience` | Circuit breaker, retry, bulkhead | Gateway, all services |
| `shared/pkg/search` | `vnp-memory/shared/pkg/search` | BM25 + vector + RRF fusion search | Observe-service search, search-service |
| `shared/pkg/telemetry` | `vnp-memory/shared/pkg/telemetry` | OTel tracer, metrics, structured logger | All services |
| `shared/pkg/tenant` | `vnp-memory/shared/pkg/tenant` | Tenant context propagation middleware | Gateway + all services |
| `shared/pkg/tokenizer` | `vnp-memory/shared/pkg/tokenizer` | tiktoken-go wrapper (token counting) | Memobase-engine |
| `shared/pkg/vectorstore` | `vnp-memory/shared/pkg/vectorstore` | Vector DB interface + pgvector implementation | Cognee-search, Graphiti-search |

> **Note:** Các package bên dưới **không tồn tại** trong codebase thực tế — đã removed hoặc chưa tạo:
> `pkg/graph/`, `pkg/viking/`, `pkg/vikingfs/`, `pkg/parse/`, `pkg/surrealdb/`
> (các TDD cũ tham chiếu sai path)

---

## 2. Key Interfaces

### 2.1 `shared/pkg/graph` — Graph Types

```go
// shared/pkg/graph/types.go
type Node struct {
    ID         string
    Name       string
    Type       string
    Labels     []string       // includes NodeSet tags
    Properties map[string]any
    Derived    bool
    VectorID   string
}

type Edge struct {
    ID         string
    SourceID   string
    TargetID   string
    Label      string
    Weight     float64
    Properties map[string]any
    Subject    string   // Memify alias
    Predicate  string   // Memify alias
    Object     string   // Memify alias
    Derived    bool
}

type Community struct {
    ID      string
    Summary string
    Members []string // entity IDs
}
```

### 2.2 `shared/pkg/adapters` — Adapter Interfaces

```go
// LLM Client (Bifrost-routed)
type LLMClient interface {
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

// Embedder
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float64, error)
    Dimensions() int
}

// Reranker (CrossEncoder)
type Reranker interface {
    Rerank(ctx context.Context, query string, passages []string) ([]RankedPassage, error)
}
```

### 2.3 `shared/pkg/vectorstore` — pgvector

```go
// pgvector implementation (PostgreSQL + pgvector extension)
type VectorStore interface {
    Upsert(ctx context.Context, item *VectorItem) error
    Search(ctx context.Context, vector []float64, opts SearchOpts) ([]SearchResult, error)
    Delete(ctx context.Context, id string) error
}

// SearchOpts
type SearchOpts struct {
    TenantID string
    Limit    int
    Threshold float64
    Filter   map[string]string
}
```

### 2.4 `shared/pkg/search` — BM25 + RRF

```go
// Hybrid search: BM25 + Vector results fused by RRF
type HybridSearcher interface {
    Search(ctx context.Context, query string, opts HybridOpts) ([]HybridResult, error)
}

// RRF fusion: score(d) = Σ 1/(k + rank_i(d)), k=60
func RRFFusion(bm25Results, vectorResults []SearchResult) []SearchResult
```

### 2.5 `shared/pkg/tenant` — Tenant Context

```go
// Inject TenantID from JWT into context
func TenantMiddleware(next http.Handler) http.Handler

// Extract TenantID — panics if missing (dev safety net)
func FromContext(ctx context.Context) string

// Context key
type contextKey int
const TenantKey contextKey = iota
```

### 2.6 `shared/pkg/privacy` — PII Redaction

```go
// Scrub PII từ hook payloads trước khi persist
type Redactor interface {
    Redact(input string) (string, []RedactedSpan, error)
}
// Detects: email, phone, credit card, SSN, IP addresses
```

### 2.7 `shared/pkg/resilience` — Circuit Breaker

```go
type CircuitBreaker interface {
    Call(ctx context.Context, fn func() error) error
    State() CircuitState // Closed | Open | HalfOpen
}

// Config: MaxFailures, Timeout, MaxRequests (from gateway CircuitConfig)
```

### 2.8 `shared/pkg/telemetry` — OTel

```go
// Structured logger (slog-based)
type Logger = *slog.Logger

// OTel tracer init
func InitTracer(ctx context.Context, cfg OTELConfig) (trace.Tracer, func(), error)

// Prometheus metrics
var (
    RequestDuration *prometheus.HistogramVec
    RequestTotal    *prometheus.CounterVec
    ErrorTotal      *prometheus.CounterVec
)
```

### 2.9 `shared/pkg/tokenizer` — tiktoken

```go
// Token counting cho Memobase flush budget
type Tokenizer interface {
    Count(text string) int
    CountMessages(messages []Message) int
}
// Supports: cl100k_base (GPT-4), p50k_base (GPT-3)
```

---

## 3. Module Naming Convention

```
// Shared packages (go.work registered):
module vnp-memory/shared/pkg/adapters
module vnp-memory/shared/pkg/config
module vnp-memory/shared/pkg/forward
module vnp-memory/shared/pkg/graph
module vnp-memory/shared/pkg/privacy
module vnp-memory/shared/pkg/resilience
module vnp-memory/shared/pkg/search
module vnp-memory/shared/pkg/telemetry
module vnp-memory/shared/pkg/tenant
module vnp-memory/shared/pkg/tokenizer
module vnp-memory/shared/pkg/vectorstore

// Services (go.work registered):
module vnp-memory/services/<service-name>    // most services
module github.com/vnp-community/vnp-memory/gateway  // gateway only
module github.com/vnp-community/vnp-memory/services/vnp-platform  // exception
```

---

## 4. Import Rule

```go
// Correct: service imports shared pkg
import "vnp-memory/shared/pkg/graph"
import "vnp-memory/shared/pkg/tenant"

// NEVER: cross-service imports
// import "vnp-memory/services/other-service/..."  ← FORBIDDEN
```

---

## 5. What Does NOT Exist (Common Mistakes)

Các paths sau **không tồn tại** trong codebase — đây là TDD lỗi thời:

| ❌ Path được ghi trong TDD cũ | ✅ Thực tế |
|---|---|
| `pkg/graph/` | `shared/pkg/graph/` |
| `pkg/viking/` | Không tồn tại (Viking code trong ov-* services) |
| `pkg/vikingfs/` | Không tồn tại |
| `pkg/surrealdb/` | Không tồn tại (SurrealDB integration removed) |
| `pkg/adapters/graphdb/` | `shared/pkg/adapters/` (flat structure) |
| `pkg/adapters/vectordb/` | `shared/pkg/vectorstore/` |
| `pkg/middleware/` | Trong mỗi service riêng |
| `pkg/auth/` | Trong `gateway/internal/infra/middleware/` |
