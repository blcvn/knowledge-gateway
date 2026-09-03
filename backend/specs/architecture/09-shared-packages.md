# Shared Packages (`pkg/`)

> NO business logic — only types, interfaces, middleware, adapters

---

## 1. Package Inventory

| Package | Description | Used By |
|---------|-------------|---------|
| `pkg/graph/` | Shared graph domain types | Cognee, Graphiti |
| `pkg/profile/` | Profile/Blob/Event types | Memobase, Search Hub |
| `pkg/viking/` | OpenViking domain: Context, Namespace, URI, Identity | OV-* services |
| `pkg/vikingfs/` | Go-native filesystem engine (read/write/tree/lock) | OV-FS, OV-Session |
| `pkg/parse/` | File parser registry (tree-sitter, markdown, PDF) | OV-Resource |
| `pkg/adapters/graphdb/` | GraphDB interface + implementations | Graphiti Store, Cognee Cognify |
| `pkg/adapters/vectordb/` | VectorDB interface + implementations | Cognee Search, Event, OV-Search |
| `pkg/adapters/reldb/` | RelationalDB interface (PostgreSQL/SurrealDB) | All services with relational data |
| `pkg/adapters/llm/` | LLMClient interface + Bifrost | Cognify, Knowledge, Engine, OV-Session |
| `pkg/adapters/embedder/` | Embedder interface + 12 providers | Knowledge, Cognify, OV-Resource |
| `pkg/adapters/reranker/` | CrossEncoder interface | Search Hub, Graphiti Search, OV-Search |
| `pkg/adapters/vlm/` | VLM interface (vision models) | OV-Resource |
| `pkg/adapters/kms/` | KMS interface (Local/Vault/Cloud) | OV-Crypto |
| `pkg/adapters/storage/` | Object storage interface | Cognee Ingestion |
| `pkg/surrealdb/` | SurrealDB multi-model adapter (graph+vector+relational) | All (when SurrealDB mode enabled) |
| `pkg/middleware/` | gRPC/HTTP interceptors | All |
| `pkg/resilience/` | Circuit breaker, retry, bulkhead | All |
| `pkg/observability/` | OTel tracer, metrics, logger | All |
| `pkg/config/` | Viper loader + validator | All |
| `pkg/errors/` | Error types → gRPC/HTTP mapping | All |
| `pkg/nats/` | NATS publisher/subscriber helpers | All |
| `pkg/auth/` | JWT provider, API key validator, RBAC | Gateway, OV-Admin |
| `pkg/tenant/` | Tenant context propagation | All |
| `pkg/tokenizer/` | tiktoken-go wrapper | Memobase |
| `pkg/prompt/` | Prompt template engine | Engine, Knowledge, Cognify |
| `pkg/pagination/` | Cursor/offset pagination | All |
| `pkg/testutil/` | Fixtures, mocks, testcontainers | All (test) |

---

## 2. Key Interfaces

### 2.1 GraphDB Interface

```go
// pkg/adapters/graphdb/interface.go
type GraphDB interface {
    SaveNode(ctx context.Context, node *graph.Node) error
    SaveEdge(ctx context.Context, edge *graph.Edge) error
    GetNode(ctx context.Context, id, groupID string) (*graph.Node, error)
    CosineSimilarity(ctx context.Context, vec []float64, groupID string, limit int) ([]*graph.Node, error)
    FulltextSearch(ctx context.Context, query, groupID string, limit int) ([]*graph.Node, error)
    BFSTraversal(ctx context.Context, startID string, depth int) ([]*graph.Node, error)
    DeleteByGroup(ctx context.Context, groupID string) error
    Close() error
}
```

### 2.2 LLM Client Interface

```go
// pkg/adapters/llm/interface.go
type LLMClient interface {
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    CompleteStructured(ctx context.Context, req *StructuredRequest, schema any) error
    CountTokens(ctx context.Context, text string) (int, error)
}
```

### 2.3 VectorDB Interface

```go
// pkg/adapters/vectordb/interface.go
type VectorDB interface {
    Upsert(ctx context.Context, collection string, vectors []Vector) error
    Search(ctx context.Context, collection string, query []float64, limit int) ([]SearchResult, error)
    Delete(ctx context.Context, collection string, ids []string) error
}
```

### 2.4 RelationalDB Interface

```go
// pkg/adapters/reldb/interface.go
type RelationalDB interface {
    Create(ctx context.Context, table string, record any) error
    Get(ctx context.Context, table, id string, dest any) error
    Update(ctx context.Context, table, id string, patch any) error
    Delete(ctx context.Context, table, id string) error
    Query(ctx context.Context, query string, args ...any) ([]map[string]any, error)
    List(ctx context.Context, table string, filter Filter, dest any) error
    Tx(ctx context.Context, fn func(tx TxHandle) error) error
    Migrate(ctx context.Context, migrations []Migration) error
    Close() error
}
```

### 2.5 SurrealDB Unified Adapter

```go
// pkg/surrealdb/client.go
// SurrealDB implements GraphDB, VectorDB, AND RelationalDB simultaneously.
// Activated via config: db.backend = "surrealdb"
type SurrealClient struct {
    conn     *surrealdb.DB
    ns       string          // Namespace (cluster-level isolation)
    db       string          // Database (tenant-level isolation)
}

// pkg/surrealdb/graph_adapter.go
// Implements pkg/adapters/graphdb.GraphDB
func (s *SurrealClient) SaveNode(ctx context.Context, node *graph.Node) error {
    // SurrealQL: CREATE node:{id} SET name=$name, type=$type, embedding=$embedding
}
func (s *SurrealClient) SaveEdge(ctx context.Context, edge *graph.Edge) error {
    // SurrealQL: RELATE node:{from} -> edge_type -> node:{to} SET ...
}
func (s *SurrealClient) BFSTraversal(ctx context.Context, startID string, depth int) ([]*graph.Node, error) {
    // SurrealQL: SELECT ->edge_type->node.* FROM node:{startID} LIMIT $depth
}

// pkg/surrealdb/vector_adapter.go
// Implements pkg/adapters/vectordb.VectorDB
func (s *SurrealClient) Search(ctx context.Context, collection string, query []float64, limit int) ([]SearchResult, error) {
    // SurrealQL: SELECT *, vector::similarity::cosine(embedding, $query) AS score
    //           FROM {collection} WHERE embedding <|{limit},40|> $query
}

// pkg/surrealdb/relational_adapter.go
// Implements pkg/adapters/reldb.RelationalDB
func (s *SurrealClient) Query(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
    // Executes raw SurrealQL with parameterized queries
}
```

---

## 3. Middleware Stack

```go
// Applied to all gRPC services
func DefaultInterceptors() []grpc.UnaryServerInterceptor {
    return []grpc.UnaryServerInterceptor{
        middleware.Recovery(),
        middleware.RequestID(),
        middleware.Logging(),
        middleware.Tracing(),
        middleware.TenantExtraction(),
        middleware.Validation(),
    }
}
```

---

## 4. Resilience Patterns

```go
// pkg/resilience/circuit_breaker.go
type CircuitBreakerConfig struct {
    Name          string
    MaxRequests   uint32        // half-open max
    Interval      time.Duration // reset interval
    Timeout       time.Duration // open → half-open
    ReadyToTrip   func(counts gobreaker.Counts) bool
}

// pkg/resilience/retry.go
type RetryConfig struct {
    MaxAttempts int
    InitialWait time.Duration
    MaxWait     time.Duration
    Multiplier  float64
    Jitter      bool
}

// pkg/resilience/bulkhead.go
type BulkheadConfig struct {
    MaxConcurrent int
    MaxWait       time.Duration
}
```
