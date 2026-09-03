# TASK-OV-004 — `pkg/` Infrastructure Packages (nats, resilience, middleware, observability, auth, parse)

**Wave:** 1 (Foundation)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-001, TASK-OV-003  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-OV-007 §6-§10](../solutions/SOL-OV-007-Shared-Infrastructure.md)

**Trạng thái:** ✅ Implemented  
**Ghi chú:** shared/pkg/telemetry + shared/pkg/resilience infra  
---

## Mục tiêu

Tạo 6 infrastructure packages dùng chung cho tất cả OpenViking services: NATS helpers, resilience patterns, HTTP/gRPC middleware, OTel observability, API key auth, và file parser registry.

---

## 1. `pkg/nats/` — JetStream Publisher/Subscriber

**File: `publisher.go`**
```go
package natspkg

type Publisher struct {
    js nats.JetStreamContext
}

func NewPublisher(nc *nats.Conn) (*Publisher, error)
// Tạo JetStream context từ connection

func (p *Publisher) Publish(ctx context.Context, subject string, payload any) error
// json.Marshal(payload) → js.Publish(subject, data)
// Timeout từ ctx
```

**File: `subscriber.go`**
```go
type MessageHandler func(msg *nats.Msg) error

func Subscribe(js nats.JetStreamContext, subject, durable string, handler MessageHandler) (*nats.Subscription, error)
// Durable consumer với: MaxDeliver=3, AckWait=30s
// handler return error → msg.Nak(); success → msg.Ack()

func MustCreateStream(js nats.JetStreamContext, config *nats.StreamConfig) error
// Idempotent stream creation (ignore ErrStreamNameAlreadyInUse)
```

**File: `config.go`**
```go
// Subjects constants
const (
    SubjectContentWritten  = "ov.content.written"
    SubjectContentDeleted  = "ov.content.deleted"
    SubjectSessionCommitted = "ov.session.committed"
    SubjectMemoryExtracted = "ov.session.memory.extracted"
    SubjectResourceIngested = "ov.resource.ingested"
    SubjectKeyRotated      = "ov.crypto.key.rotated"
    SubjectAccountCreated  = "admin.account.created"
    SubjectAccountDeleted  = "admin.account.deleted"
)

// Common payloads
type ContentWrittenPayload struct {
    URI         string `json:"uri"`
    AccountID   string `json:"account_id"`
    ContextType string `json:"context_type"`
    Level       int    `json:"level"`
}

type ContentDeletedPayload struct {
    URI       string `json:"uri"`
    AccountID string `json:"account_id"`
}

type SessionCommittedPayload struct {
    SessionID  string   `json:"session_id"`
    ArchiveURI string   `json:"archive_uri"`
    UsedURIs   []string `json:"used_uris"`
}

type AccountCreatedPayload struct {
    AccountID          string `json:"account_id"`
    EncryptionEnabled  bool   `json:"encryption_enabled"`
}

type AccountDeletedPayload struct {
    AccountID string `json:"account_id"`
}
```

---

## 2. `pkg/resilience/` — Circuit Breaker, Retry, Bulkhead

**File: `circuit_breaker.go`**
```go
package resilience

// Wraps sony/gobreaker with OpenViking defaults
type CircuitBreaker struct {
    cb *gobreaker.CircuitBreaker
}

type CircuitBreakerConfig struct {
    Name        string
    MaxRequests uint32        // default: 3
    Interval    time.Duration // default: 10s
    Timeout     time.Duration // default: 30s
    MinRequests uint32        // default: 10
}

func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker

func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error)
// ErrOpenState → wrap as viking.ErrInternal "circuit open"
```

**File: `retry.go`**
```go
type RetryConfig struct {
    MaxAttempts int           // default: 3
    BaseDelay   time.Duration // default: 100ms
    MaxDelay    time.Duration // default: 10s
    JitterPct   float64       // default: 0.20 (±20%)
}

func WithRetry(ctx context.Context, config RetryConfig, fn func() error) error
// Only retry: gRPC UNAVAILABLE, DEADLINE_EXCEEDED, RESOURCE_EXHAUSTED
// Exponential backoff + jitter
```

**File: `bulkhead.go`**
```go
type Bulkhead struct {
    sem chan struct{}
}

func NewBulkhead(maxConcurrent int) *Bulkhead

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error
// Acquire semaphore → run fn → release
// ctx timeout → return ErrResourceBusy
```

---

## 3. `pkg/middleware/` — HTTP + gRPC Interceptors

**File: `logging/http.go`**
```go
// Structured access log per request (log/slog):
// {"level":"INFO","method":"POST","path":"/api/v1/search/find",
//  "duration_ms":45,"status":200,"account":"acct1","request_id":"uuid"}

func HTTPLogger(next http.Handler) http.Handler
```

**File: `recovery/http.go`**
```go
// Catch panics → log stack trace → return HTTP 500
func HTTPRecovery(next http.Handler) http.Handler
```

**File: `recovery/grpc.go`**
```go
// gRPC server interceptor: catch panics → return gRPC INTERNAL error
func GRPCRecovery() grpc.UnaryServerInterceptor
```

**File: `ratelimit/redis_sliding_window.go`**
```go
type RateLimiter struct {
    redis     *redis.Client
    window    time.Duration
    maxReqs   int
}

type RateLimitConfig struct {
    RedisURL          string
    Window            time.Duration // default: 60s
    RequestsPerWindow int           // default: 1000
}

func NewRateLimiter(config RateLimitConfig) (*RateLimiter, error)

func (rl *RateLimiter) HTTPMiddleware(next http.Handler) http.Handler
// Key: "ov_rl:{account_id}:{path_bucket}:{window_minute}"
// Sliding window với INCR + EXPIRE
// On limit exceeded: 429 + Retry-After header
```

---

## 4. `pkg/observability/` — OTel + Prometheus

**File: `tracer.go`**
```go
func InitTracer(serviceName, otelEndpoint string) (func(), error)
// Setup OTel SDK → OTLP HTTP exporter → Jaeger/Tempo
// Returns shutdown function
```

**File: `metrics.go`**
```go
// Pre-defined metrics (promauto auto-register)
var (
    VectorSearchesTotal    *prometheus.CounterVec
    MemoryExtractedTotal   *prometheus.CounterVec
    SessionCommitsTotal    *prometheus.CounterVec
    HTTPRequestDuration    *prometheus.HistogramVec
    ResourceIngestedTotal  *prometheus.CounterVec
    GRPCRequestDuration    *prometheus.HistogramVec
    FSOperationsTotal      *prometheus.CounterVec
)

func init() { /* register all metrics */ }
```

**File: `health.go`**
```go
// Standard health handlers
func HealthzHandler(w http.ResponseWriter, r *http.Request)  // Always 200
func ReadyzHandler(ready *atomic.Bool) http.HandlerFunc       // 200 if ready, 503 if not
```

---

## 5. `pkg/auth/` — API Key Validation

**File: `key_resolver.go`**
```go
package auth

type ResolvedKey struct {
    AccountID string
    UserID    string
    Role      viking.Role
    KeyID     string
}

type KeyResolver interface {
    Resolve(ctx context.Context, apiKey string) (*ResolvedKey, error)
    Invalidate(ctx context.Context, keyID string) error  // Called on revoke
}

// CachedKeyResolver: Redis cache TTL=5min
// Cache key: "ovkey:" + sha256(apiKey)[:16]
type CachedKeyResolver struct {
    // requires grpc AdminClient — defined in interface to avoid circular import
    resolver KeyResolver
    redis    *redis.Client
    cacheTTL time.Duration
}

func NewCachedKeyResolver(resolver KeyResolver, redis *redis.Client, ttl time.Duration) *CachedKeyResolver
```

---

## 6. `pkg/parse/` — File Parser Registry

**File: `registry.go`**
```go
package parse

type Parser interface {
    Parse(ctx context.Context, path string, content []byte) ([]Chunk, error)
    SupportedExtensions() []string
}

type Chunk struct {
    Content   string
    StartLine int
    EndLine   int
    Language  string
    ChunkType string          // "function"|"class"|"paragraph"|"section"|"raw"
    Metadata  map[string]any
}

type Registry struct {
    parsers map[string]Parser
    vlm     vlm.VLMClient  // optional, for VLMParser
}

func NewRegistry(vlmClient vlm.VLMClient) *Registry
// Registers: TreeSitter, Markdown, Document, VLM (if vlmClient != nil), Text

func (r *Registry) Parse(ctx context.Context, filePath string, content []byte) ([]Chunk, error)
func (r *Registry) ParserFor(ext string) Parser
func (r *Registry) TextParser() Parser  // Fallback
```

**File: `treesitter.go`** — TreeSitterParser
- Import: `github.com/smacker/go-tree-sitter`
- Extensions: `.go`, `.py`, `.js`, `.ts`, `.rs`, `.java`, `.c`, `.cpp`, `.rb`, `.php`, `.swift`, `.kt` (50+ tổng)
- Split at function/class boundaries → `Chunk{ChunkType:"function", Metadata:{"name":"..."}}`

**File: `markdown.go`** — MarkdownParser
- Extensions: `.md`, `.mdx`, `.markdown`
- Split at `##` heading boundaries

**File: `text.go`** — TextParser (fallback)
- Extensions: `.txt`, `.yaml`, `.toml`, `.json`, `.csv`
- Sliding window chunking (max 1000 tokens per chunk, 20% overlap)

**File: `document.go`** — DocumentParser stub
- Extensions: `.pdf`, `.docx`, `.xlsx`, `.epub`
- Phase 1: Return content as single raw chunk (full implementation later)
- Comment: `// TODO: integrate unidoc/unipdf for full parsing`

---

## Unit Tests

```
// nats (dùng nats-server embedded hoặc mock)
TestPublish_SerializesPayload
TestSubscribe_AcksOnSuccess
TestSubscribe_NaksOnError
TestMustCreateStream_Idempotent

// resilience
TestCircuitBreaker_OpensAfterFailures → 10 failures → circuit open
TestCircuitBreaker_ClosesAfterTimeout → wait timeout → try again → closes
TestRetry_ExponentialBackoff → 3 failures → delays increase
TestRetry_NoRetryOnPermissionDenied → non-retryable error → no retry
TestBulkhead_ConcurrencyLimit → 20 concurrent, max=5 → 5 run, rest wait

// ratelimit
TestRateLimiter_AllowsUnderLimit
TestRateLimiter_Blocks429OverLimit
TestRateLimiter_SeparateKeyPerAccount

// parse
TestRegistry_GoFile_SplitsAtFunctions
TestRegistry_MarkdownFile_SplitsAtHeadings
TestRegistry_UnknownExtension_TextFallback
TestRegistry_TextParser_SlidingWindow
TestRegistry_ParseEmpty_ReturnsEmpty
```

---

## Cấu trúc thư mục kết quả

```
pkg/
├── nats/
│   ├── publisher.go
│   ├── subscriber.go
│   ├── config.go
│   └── publisher_test.go
├── resilience/
│   ├── circuit_breaker.go
│   ├── retry.go
│   ├── bulkhead.go
│   └── *_test.go
├── middleware/
│   ├── logging/http.go
│   ├── recovery/http.go
│   ├── recovery/grpc.go
│   └── ratelimit/redis_sliding_window.go
├── observability/
│   ├── tracer.go
│   ├── metrics.go
│   └── health.go
├── auth/
│   ├── key_resolver.go
│   └── key_resolver_test.go
└── parse/
    ├── registry.go
    ├── treesitter.go
    ├── markdown.go
    ├── text.go
    ├── document.go
    └── registry_test.go
```

---

## Go Dependencies cần thêm vào `go.mod`

```
github.com/sony/gobreaker v0.5.0
github.com/nats-io/nats.go v1.34.1
go.opentelemetry.io/otel v1.24.0
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0
github.com/prometheus/client_golang v1.19.0
github.com/smacker/go-tree-sitter v0.0.0-20230720070738-0d0a9f78d8f8
github.com/redis/go-redis/v9 v9.5.1
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go mod tidy
go build ./pkg/...
go test ./pkg/... -v -count=1
```
