# 11 — Shared Packages (`pkg/`)

> Tất cả shared packages — KHÔNG chứa business logic, chỉ infrastructure abstractions

---

## 1. Package Map

```
pkg/
├── adapters/          # Infrastructure adapter interfaces + implementations
│   ├── graphdb/       # Graph database abstraction
│   ├── vectordb/      # Vector database abstraction
│   ├── llm/           # LLM client abstraction
│   ├── embedder/      # Embedding generation
│   ├── reranker/      # Cross-encoder reranking
│   └── storage/       # Object storage (S3/MinIO)
├── middleware/         # gRPC/HTTP interceptors
│   ├── auth/          # JWT/APIKey extraction
│   ├── logging/       # Structured access logging
│   ├── tracing/       # OTel trace propagation
│   ├── recovery/      # Panic recovery
│   ├── ratelimit/     # Redis-backed rate limiting
│   └── validation/    # Request validation
├── resilience/         # Fault tolerance patterns
├── observability/      # OTel, Prometheus, health
├── graph/             # Shared graph domain types
├── auth/              # JWT provider, API key validator
├── config/            # Viper config loader
├── errors/            # Domain errors → gRPC status
├── nats/              # NATS JetStream helpers
├── tenant/            # Multi-tenant context
├── pagination/        # Cursor/offset pagination
└── testutil/          # Test helpers, mocks, fixtures
```

---

## 2. Key Interfaces

### 2.1 GraphDB Adapter (`pkg/adapters/graphdb/`)

```go
// Interface shared by both Cognee and Graphiti
type GraphDB interface {
    // Node CRUD
    CreateNode(ctx context.Context, label string, props map[string]any) (string, error)
    GetNode(ctx context.Context, id string) (map[string]any, error)
    UpdateNode(ctx context.Context, id string, props map[string]any) error
    DeleteNode(ctx context.Context, id string) error

    // Edge CRUD
    CreateEdge(ctx context.Context, from, to, label string, props map[string]any) (string, error)
    GetEdges(ctx context.Context, nodeID string, dir Direction) ([]Edge, error)

    // Query
    Query(ctx context.Context, cypher string, params map[string]any) ([]Record, error)

    // Fulltext
    FullTextSearch(ctx context.Context, index, query string, limit int) ([]Record, error)

    // Management
    EnsureIndex(ctx context.Context, label, property string) error
    EnsureFullTextIndex(ctx context.Context, name string, labels, properties []string) error
    Ping(ctx context.Context) error
    Close() error
}

// Implementations: neo4j.New(), falkordb.New(), kuzu.New()
```

### 2.2 VectorDB Adapter (`pkg/adapters/vectordb/`)

```go
type VectorDB interface {
    CreateCollection(ctx context.Context, name string, dim int) error
    Upsert(ctx context.Context, collection string, points []VectorPoint) error
    Search(ctx context.Context, collection string, vector []float32, topK int, filter Filter) ([]ScoredPoint, error)
    Delete(ctx context.Context, collection string, ids []string) error
    Ping(ctx context.Context) error
    Close() error
}

// Implementations: qdrant.New(), pgvector.New()
```

### 2.3 LLM Client (`pkg/adapters/llm/`)

```go
type LLMClient interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    CompleteStructured(ctx context.Context, req StructuredRequest) (*StructuredResponse, error)
    StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}

type CompletionRequest struct {
    Model       string
    Messages    []Message
    Temperature float64
    MaxTokens   int
    Provider    string  // bifrost, openai, anthropic, openrouter
}

// Implementations: bifrost.New(), openai.New(), anthropic.New()
```

### 2.4 Embedder (`pkg/adapters/embedder/`)

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
}

// Implementations: openai.New("text-embedding-3-large"), bifrost.New()
```

### 2.5 Object Storage (`pkg/adapters/storage/`)

```go
type ObjectStorage interface {
    Upload(ctx context.Context, bucket, key string, reader io.Reader) error
    Download(ctx context.Context, bucket, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, bucket, key string) error
    Exists(ctx context.Context, bucket, key string) (bool, error)
    List(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error)
}

// Implementations: s3.New(), minio.New(), local.New()
```

---

## 3. Resilience Package (`pkg/resilience/`)

```go
// Circuit Breaker (sony/gobreaker)
type CircuitBreaker struct {
    breaker *gobreaker.CircuitBreaker
}
func NewCircuitBreaker(name string, opts ...Option) *CircuitBreaker

// Retry with exponential backoff
func Retry(ctx context.Context, maxAttempts int, fn func() error) error

// Bulkhead (channel-based semaphore)
type Bulkhead struct {
    sem chan struct{}
}
func NewBulkhead(maxConcurrent int) *Bulkhead
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error

// Timeout wrapper
func WithTimeout(ctx context.Context, d time.Duration, fn func(context.Context) error) error
```

---

## 4. Observability (`pkg/observability/`)

```go
// Unified tracer setup
func InitTracer(serviceName string, cfg TracerConfig) (*sdktrace.TracerProvider, error)

// Prometheus metrics registry
func InitMetrics(serviceName string) *prometheus.Registry

// Structured logger (slog)
func InitLogger(cfg LogConfig) *slog.Logger

// Health check helpers
type HealthChecker interface {
    Check(ctx context.Context) (Status, error)
}
func NewCompositeHealth(checks map[string]HealthChecker) *CompositeHealth
```

---

## 5. Tenant Package (`pkg/tenant/`)

```go
// Extract tenant from context (set by middleware)
func FromContext(ctx context.Context) (TenantInfo, error)

// Set tenant in context
func WithContext(ctx context.Context, info TenantInfo) context.Context

// gRPC metadata key
const MetadataKey = "x-tenant-id"

type TenantInfo struct {
    TenantID string  // Cognee tenant isolation
    GroupID  string  // Graphiti group isolation
    UserID   string
    Role     string
}
```

---

## 6. NATS Helpers (`pkg/nats/`)

```go
// Publisher with retry + tracing
type Publisher struct {
    js nats.JetStreamContext
}
func (p *Publisher) Publish(ctx context.Context, subject string, data []byte) error

// Subscriber with consumer group + error handling
type Subscriber struct {
    js nats.JetStreamContext
}
func (s *Subscriber) Subscribe(subject, consumer string, handler MessageHandler) error

type MessageHandler func(ctx context.Context, msg *nats.Msg) error
```

---

## 7. Test Utilities (`pkg/testutil/`)

```go
// Testcontainers for integration tests
func NewPostgresContainer(ctx context.Context) (*PostgresContainer, error)
func NewNeo4jContainer(ctx context.Context) (*Neo4jContainer, error)
func NewRedisContainer(ctx context.Context) (*RedisContainer, error)
func NewNATSContainer(ctx context.Context) (*NATSContainer, error)
func NewQdrantContainer(ctx context.Context) (*QdrantContainer, error)

// Mock generators (via mockgen)
//go:generate mockgen -source=../adapters/graphdb/interface.go -destination=mocks/mock_graphdb.go
//go:generate mockgen -source=../adapters/vectordb/interface.go -destination=mocks/mock_vectordb.go
//go:generate mockgen -source=../adapters/llm/interface.go -destination=mocks/mock_llm.go

// Fixture helpers
func LoadFixture(t *testing.T, path string) []byte
func SeedTenant(t *testing.T, db *sql.DB) TenantInfo
```
