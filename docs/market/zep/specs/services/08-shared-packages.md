# 08 — Shared Packages (pkg/)

> **Location**: `pkg/`  
> **Rule**: Shared infrastructure code — **NO business logic**

---

## 1. Overview

```
pkg/
├── adapters/          # Infrastructure adapter interfaces + implementations
│   ├── graphdb/       #   Neo4j driver
│   ├── reldb/         #   PostgreSQL (bun ORM)
│   ├── llm/           #   LLM/Bifrost client
│   ├── cache/         #   Redis cache
│   └── graphiti/      #   Graphiti HTTP client
├── auth/              # JWT, API Key, Shared Secret
├── config/            # Viper loader + validator
├── errors/            # Domain error types → gRPC status
├── graph/             # Shared graph domain types
├── metadata/          # JSONB merge, advisory locks
├── middleware/        # gRPC/HTTP interceptors
├── nats/              # NATS client helpers
├── observability/     # OpenTelemetry setup
├── pagination/        # Cursor/offset pagination
├── tenant/            # Multi-tenant context
└── testutil/          # Test helpers, mocks
```

---

## 2. Graph Types (`pkg/graph/`)

Shared graph domain types used by zep-graph and zep-search:

```go
// pkg/graph/node.go
package graph

type EntityNode struct {
    UUID       string
    Name       string
    NodeType   NodeType
    GroupID    string
    Summary    string
    Labels     []string
    Properties map[string]any
    CreatedAt  time.Time
}

type NodeType string

const (
    NodeTypeUser         NodeType = "User"
    NodeTypeAssistant    NodeType = "Assistant"
    NodeTypePreference   NodeType = "Preference"
    NodeTypeOrganization NodeType = "Organization"
    NodeTypeEvent        NodeType = "Event"
    NodeTypeLocation     NodeType = "Location"
    NodeTypeDocument     NodeType = "Document"
    NodeTypeTopic        NodeType = "Topic"
    NodeTypeObject       NodeType = "Object"
)
```

```go
// pkg/graph/edge.go
type EntityEdge struct {
    UUID       string
    Name       string          // relationship label
    Fact       string          // human-readable fact statement
    SourceID   string
    TargetID   string
    EdgeType   EdgeType
    GroupID    string
    Temporal   TemporalAnnotation
    CreatedAt  time.Time
}

type EdgeType string

const (
    EdgeTypeLocatedAt  EdgeType = "LOCATED_AT"
    EdgeTypeOccurredAt EdgeType = "OCCURRED_AT"
)
```

```go
// pkg/graph/temporal.go
type TemporalAnnotation struct {
    ValidAt   *time.Time  // when fact became true
    InvalidAt *time.Time  // when fact ceased to be true
    ExpiredAt *time.Time  // when fact was superseded
}

func (t TemporalAnnotation) IsCurrentlyValid() bool {
    now := time.Now()
    if t.ValidAt != nil && now.Before(*t.ValidAt) {
        return false
    }
    if t.InvalidAt != nil && now.After(*t.InvalidAt) {
        return false
    }
    return true
}
```

```go
// pkg/graph/episode.go
type Episode struct {
    UUID      string
    Name      string
    Content   string
    GroupID   string
    SourceID  string
    CreatedAt time.Time
}
```

```go
// pkg/graph/ontology.go

// NodePriority defines extraction classification priority
var NodePriority = map[NodeType]int{
    NodeTypeUser:         1,  // Highest — singleton
    NodeTypeAssistant:    1,  // Highest — singleton
    NodeTypePreference:   2,  // Very High — LOW threshold
    NodeTypeOrganization: 3,
    NodeTypeEvent:        3,
    NodeTypeLocation:     4,
    NodeTypeDocument:     4,
    NodeTypeTopic:        5,
    NodeTypeObject:       6,  // Lowest — last resort
}

// EdgeTypeMap defines valid edge types between node type pairs
var EdgeTypeMap = map[[2]NodeType][]EdgeType{
    {NodeTypeEvent, NodeTypeOrganization}: {EdgeTypeOccurredAt},
    {NodeTypeEvent, NodeTypeLocation}:     {EdgeTypeOccurredAt},
    {NodeTypeOrganization, NodeTypeLocation}: {EdgeTypeLocatedAt},
    {NodeTypeUser, NodeTypeLocation}:      {EdgeTypeLocatedAt},
}
```

---

## 3. Infrastructure Adapters (`pkg/adapters/`)

### 3.1 GraphDB (`adapters/graphdb/`)

```go
type GraphDB interface {
    // Node operations
    AddNode(ctx context.Context, node graph.EntityNode) error
    GetNode(ctx context.Context, uuid string) (*graph.EntityNode, error)
    GetNodesByGroupID(ctx context.Context, groupID string) ([]graph.EntityNode, error)
    DeleteNode(ctx context.Context, uuid string) error
    
    // Edge operations
    AddEdge(ctx context.Context, edge graph.EntityEdge) error
    GetEdge(ctx context.Context, uuid string) (*graph.EntityEdge, error)
    GetEdgesByGroupID(ctx context.Context, groupID string) ([]graph.EntityEdge, error)
    GetEdgesForNode(ctx context.Context, nodeUUID string) ([]graph.EntityEdge, error)
    
    // Episode operations
    AddEpisode(ctx context.Context, episode graph.Episode) error
    GetEpisode(ctx context.Context, uuid string) (*graph.Episode, error)
    GetEpisodesByGroupID(ctx context.Context, groupID string) ([]graph.Episode, error)
    GetEpisodeMentions(ctx context.Context, episodeUUID string) ([]graph.EntityNode, []graph.EntityEdge, error)
    
    // Group operations
    DeleteGroup(ctx context.Context, groupID string) error
    
    // Lifecycle
    Close() error
    Ping(ctx context.Context) error
}

// Implementation
type Neo4jAdapter struct {
    driver neo4j.DriverWithContext
}
```

### 3.2 Relational DB (`adapters/reldb/`)

```go
type RelationalDB interface {
    DB() *bun.DB
    Ping(ctx context.Context) error
    Close() error
    RunMigrations(ctx context.Context) error
}

// Implementation wrapping uptrace/bun
type PostgresAdapter struct {
    db *bun.DB
}

func NewPostgresAdapter(dsn string, opts ...Option) (*PostgresAdapter, error) {
    sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
    db := bun.NewDB(sqldb, pgdialect.New())
    return &PostgresAdapter{db: db}, nil
}
```

### 3.3 Graphiti Client (`adapters/graphiti/`)

```go
type GraphitiClient interface {
    PutMemory(ctx context.Context, groupID string, messages []Message, addPrefix bool) error
    GetMemory(ctx context.Context, groupID string, maxFacts int, queryMessages []string) ([]graph.EntityEdge, error)
    Search(ctx context.Context, req SearchRequest) ([]graph.EntityEdge, error)
    AddNode(ctx context.Context, node graph.EntityNode) error
    GetFact(ctx context.Context, uuid string) (*graph.EntityEdge, error)
    DeleteFact(ctx context.Context, uuid string) error
    DeleteGroup(ctx context.Context, groupID string) error
    DeleteEpisode(ctx context.Context, uuid string) error
}

type HTTPGraphitiClient struct {
    baseURL    string
    httpClient *http.Client
    tracer     trace.Tracer
}
```

### 3.4 Cache Store (`adapters/cache/`)

```go
type CacheStore interface {
    Get(ctx context.Context, key string, dest any) error
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    DeleteByPrefix(ctx context.Context, prefix string) error
    Ping(ctx context.Context) error
    Close() error
}

type RedisAdapter struct {
    client *redis.Client
}
```

### 3.5 LLM Client (`adapters/llm/`)

```go
type LLMClient interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    StructuredOutput(ctx context.Context, req StructuredRequest) (json.RawMessage, error)
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Implementation: BifrostClient (calls Bifrost AI Gateway)
type BifrostClient struct {
    endpoint string
    apiKey   string
    client   *http.Client
}
```

---

## 4. Auth (`pkg/auth/`)

```go
// JWT Token Provider
type TokenProvider interface {
    ValidateToken(token string) (*Claims, error)
    GenerateToken(claims Claims) (string, error)
}

type Claims struct {
    ProjectUUID string
    UserID      string
    Scopes      []string
    ExpiresAt   time.Time
}

// API Key Validator
type APIKeyValidator interface {
    Validate(ctx context.Context, key string) (*APIKeyInfo, error)
}

type APIKeyInfo struct {
    ProjectUUID string
    Scopes      []string
    Name        string
}

// Shared Secret Validator (legacy CE compatibility)
type SharedSecretValidator struct {
    secret string
}

func (v *SharedSecretValidator) Validate(header string) error {
    if header != v.secret {
        return ErrInvalidSecret
    }
    return nil
}
```

---

## 5. Middleware (`pkg/middleware/`)

### HTTP Interceptors (for Gateway)

```go
func CORSMiddleware(cfg CORSConfig) func(http.Handler) http.Handler
func AccessLogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler
func HeartbeatMiddleware(path string) func(http.Handler) http.Handler
func RequestSizeLimiterMiddleware(maxBytes int64) func(http.Handler) http.Handler
func RequestIDMiddleware() func(http.Handler) http.Handler
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler
func RealIPMiddleware() func(http.Handler) http.Handler
func CleanPathMiddleware() func(http.Handler) http.Handler
func VersionHeaderMiddleware(version string) func(http.Handler) http.Handler
func OTelMiddleware(serviceName string) func(http.Handler) http.Handler
func AuthExtractMiddleware(validator auth.TokenProvider) func(http.Handler) http.Handler
func TenantResolveMiddleware() func(http.Handler) http.Handler
func RateLimitMiddleware(store cache.CacheStore, cfg RateLimitConfig) func(http.Handler) http.Handler
```

### gRPC Interceptors (for Services)

```go
func AuthUnaryInterceptor(auth auth.TokenProvider) grpc.UnaryServerInterceptor
func TenantUnaryInterceptor() grpc.UnaryServerInterceptor
func LoggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor
func RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor
func TracingUnaryInterceptor(tracer trace.Tracer) grpc.UnaryServerInterceptor
func MetricsUnaryInterceptor() grpc.UnaryServerInterceptor
func ValidationUnaryInterceptor() grpc.UnaryServerInterceptor
```

---

## 6. Metadata Utilities (`pkg/metadata/`)

```go
// JSONB merge-patch for metadata updates
func MergeMetadata(base, patch map[string]any) map[string]any {
    result := make(map[string]any)
    for k, v := range base {
        result[k] = v
    }
    for k, v := range patch {
        if v == nil {
            delete(result, k)
        } else {
            result[k] = v
        }
    }
    return result
}

// Advisory lock helpers
type AdvisoryLockManager struct {
    db     *bun.DB
    retry  RetryPolicy
}

type RetryPolicy struct {
    InitialInterval time.Duration  // 200ms
    MaxInterval     time.Duration  // 30s
    MaxRetries      int            // 15
    Multiplier      float64        // 2.0
}

func (m *AdvisoryLockManager) Acquire(ctx context.Context, key string) error {
    lockKey := hashToInt64(key)  // SHA-256 → int64
    return m.retryWithBackoff(ctx, func() error {
        _, err := m.db.ExecContext(ctx, "SELECT pg_advisory_lock(?)", lockKey)
        return err
    })
}

func (m *AdvisoryLockManager) Release(ctx context.Context, key string) error {
    lockKey := hashToInt64(key)
    _, err := m.db.ExecContext(ctx, "SELECT pg_advisory_unlock(?)", lockKey)
    return err
}
```

---

## 7. Observability (`pkg/observability/`)

```go
// OpenTelemetry setup
func InitTracer(serviceName string, cfg OTelConfig) (*sdktrace.TracerProvider, error)
func InitMeter(serviceName string, cfg OTelConfig) (*sdkmetric.MeterProvider, error)

// Structured logging
func NewLogger(serviceName string, level slog.Level, format string) *slog.Logger

// Span attributes
const (
    AttrSessionID   = "zep.session.id"
    AttrUserID      = "zep.user.id"
    AttrProjectUUID = "zep.project.uuid"
    AttrSearchQuery = "zep.search.query"
    AttrSearchScope = "zep.search.scope"
    AttrReranker    = "zep.search.reranker"
    AttrFactCount   = "zep.fact.count"
    AttrMessageCount = "zep.message.count"
    AttrLatencyMs   = "zep.latency.ms"
)

// Prometheus metrics
var (
    RequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "zep_request_duration_seconds"},
        []string{"service", "method", "status"},
    )
    RequestTotal = prometheus.NewCounterVec(
        prometheus.HistogramOpts{Name: "zep_requests_total"},
        []string{"service", "method", "status"},
    )
    GraphExtractionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "zep_graph_extraction_duration_seconds"},
        []string{"group_id"},
    )
    FactCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{Name: "zep_fact_count"},
        []string{"group_id"},
    )
    SearchLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "zep_search_latency_seconds"},
        []string{"scope", "reranker"},
    )
)
```

---

## 8. Resilience (`pkg/resilience/`)

```go
// Circuit Breaker (sony/gobreaker)
type CircuitBreakerConfig struct {
    MaxFailures  int           // default 5
    Timeout      time.Duration // default 60s
    HalfOpenMax  int           // default 3
}

func NewCircuitBreaker(name string, cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker

// Retry with exponential backoff + jitter
func Retry(ctx context.Context, fn func() error, opts ...RetryOption) error

type RetryOption func(*retryConfig)
func WithMaxRetries(n int) RetryOption
func WithInitialInterval(d time.Duration) RetryOption
func WithMaxInterval(d time.Duration) RetryOption
func WithMultiplier(m float64) RetryOption

// Bulkhead (semaphore-based concurrency limiter)
type Bulkhead struct {
    sem chan struct{}
}

func NewBulkhead(maxConcurrent int) *Bulkhead
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error
```

---

## 9. Tenant (`pkg/tenant/`)

```go
type contextKey string

const (
    ProjectUUIDKey contextKey = "project_uuid"
    UserIDKey      contextKey = "user_id"
)

type TenantContext struct {
    ProjectUUID string
    UserID      string
}

func FromContext(ctx context.Context) TenantContext {
    return TenantContext{
        ProjectUUID: ctx.Value(ProjectUUIDKey).(string),
        UserID:      ctx.Value(UserIDKey).(string),
    }
}

func WithProject(ctx context.Context, projectUUID string) context.Context {
    return context.WithValue(ctx, ProjectUUIDKey, projectUUID)
}
```

---

## 10. Errors (`pkg/errors/`)

```go
type DomainError struct {
    Code    ErrorCode
    Message string
    Details map[string]any
}

type ErrorCode string

const (
    ErrNotFound         ErrorCode = "NOT_FOUND"
    ErrAlreadyExists    ErrorCode = "ALREADY_EXISTS"
    ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"
    ErrUnauthenticated  ErrorCode = "UNAUTHENTICATED"
    ErrInvalidInput     ErrorCode = "INVALID_INPUT"
    ErrRateLimited      ErrorCode = "RATE_LIMITED"
    ErrSessionEnded     ErrorCode = "SESSION_ENDED"
    ErrLockTimeout      ErrorCode = "LOCK_TIMEOUT"
    ErrInternal         ErrorCode = "INTERNAL"
)

func ToGRPCStatus(err error) *status.Status
func ToHTTPStatus(err error) int
func FromGRPCStatus(st *status.Status) error
```

---

## 11. NATS Helpers (`pkg/nats/`)

```go
type Publisher interface {
    Publish(ctx context.Context, subject string, data any) error
    PublishAsync(ctx context.Context, subject string, data any) error
}

type Subscriber interface {
    Subscribe(subject string, handler MessageHandler) error
    QueueSubscribe(subject, queue string, handler MessageHandler) error
}

type MessageHandler func(ctx context.Context, msg *nats.Msg) error

func NewJetStreamClient(url string, opts ...nats.Option) (*JetStreamClient, error)
func CreateStream(js nats.JetStreamContext, cfg StreamConfig) error

type StreamConfig struct {
    Name       string
    Subjects   []string
    MaxAge     time.Duration  // default 24h
    MaxMsgs    int64
    Replicas   int            // default 1
    MaxDeliver int            // default 3
    AckWait    time.Duration  // default 30s
}
```

---

## 12. Pagination (`pkg/pagination/`)

```go
type PageRequest struct {
    Limit  int    // default 20, max 100
    Offset int    // 0-based
    OrderBy string // field name
    Order   string // "asc" | "desc"
}

type PageResponse struct {
    Total   int
    Limit   int
    Offset  int
    HasMore bool
}

func NormalizePagination(req *PageRequest) {
    if req.Limit <= 0 { req.Limit = 20 }
    if req.Limit > 100 { req.Limit = 100 }
    if req.Offset < 0 { req.Offset = 0 }
}
```

---

## 13. Test Utilities (`pkg/testutil/`)

```go
// Testcontainers
func NewPostgresContainer(t *testing.T) (*PostgresContainer, error)
func NewNeo4jContainer(t *testing.T) (*Neo4jContainer, error)
func NewRedisContainer(t *testing.T) (*RedisContainer, error)
func NewNATSContainer(t *testing.T) (*NATSContainer, error)

// Fixtures
func SeedUser(t *testing.T, db *bun.DB, opts ...UserOption) *domain.User
func SeedSession(t *testing.T, db *bun.DB, opts ...SessionOption) *domain.Session
func SeedMessages(t *testing.T, db *bun.DB, sessionID string, count int) []*domain.Message

// Mocks (generated by mockgen)
// mock_user_repository.go, mock_graphiti_client.go, etc.
```
