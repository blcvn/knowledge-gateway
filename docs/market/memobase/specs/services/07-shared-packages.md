# 07 — Shared Packages (pkg/)

---

## 1. Package Map

```
pkg/
├── blob/                   # Blob type definitions
│   ├── types.go            # BlobType, ChatMessage, ChatBlob, DocBlob
│   └── parser.go           # Blob → string conversion
│
├── profile/                # Profile domain types (shared)
│   ├── types.go            # ProfileSlot, ProfileAttributes, TopicDef
│   └── schema.go           # Default profile topics (basic_info, interest, etc.)
│
├── adapters/               # Infrastructure adapter interfaces
│   ├── llm/
│   │   ├── client.go       # LLMClient interface
│   │   ├── bifrost.go      # Bifrost adapter
│   │   ├── openai.go       # OpenAI adapter
│   │   └── options.go      # WithModel, WithJSON, WithMaxTokens
│   ├── embedder/
│   │   ├── client.go       # EmbedderClient interface
│   │   ├── openai.go       # OpenAI embeddings
│   │   ├── jina.go         # Jina embeddings
│   │   ├── ollama.go       # Ollama embeddings
│   │   └── lmstudio.go     # LMStudio embeddings
│   └── vectordb/
│       ├── client.go       # VectorDB interface
│       └── pgvector.go     # pgvector implementation
│
├── middleware/              # Shared gRPC/HTTP interceptors
│   ├── auth/
│   │   ├── extractor.go    # JWT/APIKey extraction
│   │   └── propagator.go   # gRPC metadata propagation
│   ├── logging/
│   │   └── interceptor.go  # Structured access log (gRPC + HTTP)
│   ├── tracing/
│   │   └── interceptor.go  # OTel trace context propagation
│   ├── recovery/
│   │   └── interceptor.go  # Panic → gRPC Internal error
│   ├── ratelimit/
│   │   └── interceptor.go  # Redis sliding window per-tenant
│   └── validation/
│       └── interceptor.go  # Protobuf validation
│
├── resilience/             # Circuit breaker, retry, bulkhead
│   ├── breaker.go          # sony/gobreaker wrapper
│   ├── retry.go            # Exponential backoff + jitter
│   └── bulkhead.go         # Channel-based semaphore
│
├── observability/          # Telemetry helpers
│   ├── tracer.go           # OTel tracer provider setup
│   ├── metrics.go          # Prometheus metrics registry
│   ├── logger.go           # slog JSON logger setup
│   └── health.go           # gRPC Health v1 + HTTP endpoints
│
├── config/                 # Configuration loader
│   ├── loader.go           # Viper: YAML + ENV override
│   └── validator.go        # Config validation
│
├── errors/                 # Error types + mapping
│   ├── domain.go           # DomainError, ErrorCode enum
│   ├── grpc.go             # DomainError → gRPC Status
│   └── http.go             # gRPC Status → HTTP status
│
├── nats/                   # NATS JetStream helpers
│   ├── publisher.go        # Typed publisher with retry
│   ├── subscriber.go       # Consumer group subscriber
│   └── stream.go           # Stream/subject management
│
├── tokenizer/              # Token counting
│   ├── tiktoken.go         # tiktoken-go (gpt-4o encoder)
│   └── counter.go          # CountTokens, TruncateToTokens
│
├── prompt/                 # Prompt template engine
│   ├── registry.go         # Language → PromptSet mapping
│   ├── types.go            # PromptFunc signatures
│   └── templates/
│       ├── en/             # English templates
│       └── zh/             # Chinese templates
│
├── tenant/                 # Multi-tenant context
│   ├── context.go          # ProjectID extraction from ctx
│   └── metadata.go         # gRPC metadata keys
│
└── testutil/               # Testing utilities
    ├── fixtures.go         # Test data generators
    ├── mocks.go            # Interface mocks (gomock)
    └── containers.go       # testcontainers (PG, Redis, NATS)
```

---

## 2. Key Interfaces

### 2.1 LLMClient

```go
type LLMClient interface {
    Complete(ctx context.Context, prompt string, opts ...Option) (string, error)
    CompleteJSON(ctx context.Context, prompt string, opts ...Option) (json.RawMessage, error)
}

type Option func(*RequestConfig)
func WithModel(m string) Option          { ... }
func WithMaxTokens(n int) Option         { ... }
func WithTemperature(t float64) Option   { ... }
func WithJSONMode() Option               { ... }
func WithSystemPrompt(s string) Option   { ... }
```

### 2.2 EmbedderClient

```go
type EmbedderClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, query string) ([]float32, error)
    Dimension() int
    IsEnabled() bool
}
```

### 2.3 Resilience

```go
// Circuit Breaker
type Breaker struct {
    cb *gobreaker.CircuitBreaker
}
func NewBreaker(name string, maxFail int, timeout time.Duration) *Breaker

// Retry
func WithRetry(ctx context.Context, fn func() error, opts ...RetryOption) error
// Options: MaxAttempts(3), BackoffBase(100ms), BackoffMax(5s), Jitter(true)

// Bulkhead (semaphore)
type Bulkhead struct {
    sem chan struct{}
}
func NewBulkhead(maxConcurrent int) *Bulkhead
func (b *Bulkhead) Acquire(ctx context.Context) error
func (b *Bulkhead) Release()
```

---

## 3. Error Mapping

```go
// Domain errors (pkg/errors/domain.go)
type ErrorCode int
const (
    ErrBadRequest       ErrorCode = 400
    ErrUnauthorized     ErrorCode = 401
    ErrForbidden        ErrorCode = 403
    ErrNotFound         ErrorCode = 404
    ErrConflict         ErrorCode = 409
    ErrInternal         ErrorCode = 500
    ErrServiceUnavailable ErrorCode = 503
    ErrParseError       ErrorCode = 520
)

// gRPC mapping (pkg/errors/grpc.go)
var grpcMapping = map[ErrorCode]codes.Code{
    ErrBadRequest:     codes.InvalidArgument,
    ErrUnauthorized:   codes.Unauthenticated,
    ErrForbidden:      codes.PermissionDenied,
    ErrNotFound:       codes.NotFound,
    ErrInternal:       codes.Internal,
    ErrParseError:     codes.Internal,
}
```

---

## 4. Observability Metrics

```go
// Shared metrics (all services register these)
var (
    RequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "memobase_requests_total"},
        []string{"service", "method", "project_id", "status"},
    )
    RequestLatency = promauto.NewHistogramVec(
        prometheus.HistogramOpts{Name: "memobase_request_latency_ms",
            Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000}},
        []string{"service", "method"},
    )
    LLMInvocations = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "memobase_llm_invocations_total"},
        []string{"service", "project_id", "model"},
    )
    LLMTokensInput = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "memobase_llm_input_tokens_total"},
        []string{"project_id"},
    )
    LLMLatency = promauto.NewHistogramVec(
        prometheus.HistogramOpts{Name: "memobase_llm_latency_ms",
            Buckets: []float64{100, 250, 500, 1000, 2500, 5000, 10000}},
        []string{"project_id", "model"},
    )
    EmbeddingLatency = promauto.NewHistogramVec(
        prometheus.HistogramOpts{Name: "memobase_embedding_latency_ms"},
        []string{"project_id", "provider"},
    )
)
```
