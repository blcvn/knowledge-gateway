# Shared Packages — Mono-repo Common Code

**Version:** 2.0 | **Date:** 2026-05-09  
**Purpose:** Shared Protobuf definitions, domain types, middleware, and utilities

---

## 1. Overview

Shared packages cung cấp code dùng chung giữa tất cả services trong mono-repo. Tuân thủ nguyên tắc: **shared code KHÔNG chứa business logic** — chỉ types, interfaces, utilities, và infrastructure concerns.

---

## 2. Package Layout

```
pkg/
├── proto/                              # Shared Protobuf definitions
│   ├── common/v1/
│   │   ├── pagination.proto            #   Pagination messages
│   │   ├── temporal.proto              #   Bi-temporal model messages
│   │   ├── errors.proto                #   Standardized error messages
│   │   ├── health.proto                #   Health check messages
│   │   └── metadata.proto              #   Request metadata
│   ├── graph/v1/
│   │   ├── node.proto                  #   Node type messages (shared)
│   │   └── edge.proto                  #   Edge type messages (shared)
│   ├── ingestion/v1/
│   │   └── ingestion.proto             #   Ingestion service contract
│   ├── search/v1/
│   │   └── search.proto                #   Search service contract
│   ├── knowledge/v1/
│   │   └── knowledge.proto             #   Knowledge service contract
│   ├── store/v1/
│   │   └── store.proto                 #   Store service contract
│   └── admin/v1/
│       └── admin.proto                 #   Admin service contract
│
├── graph/                              # Shared graph domain types
│   ├── node.go                         #   Node type definitions
│   ├── edge.go                         #   Edge type definitions
│   ├── temporal.go                     #   Bi-temporal model
│   ├── group.go                        #   Multi-tenancy primitives
│   └── embedding.go                    #   Embedding vector type
│
├── middleware/                         # Shared gRPC interceptors
│   ├── auth/
│   │   ├── interceptor.go              #   Auth extraction from metadata
│   │   └── tenant.go                   #   Tenant ID propagation
│   ├── logging/
│   │   └── interceptor.go              #   Structured access logging
│   ├── tracing/
│   │   └── interceptor.go              #   OTel trace propagation
│   ├── recovery/
│   │   └── interceptor.go              #   Panic recovery
│   ├── ratelimit/
│   │   └── interceptor.go              #   gRPC rate limiting
│   └── validation/
│       └── interceptor.go              #   Request validation
│
├── resilience/                         # Resilience patterns
│   ├── circuit_breaker.go              #   sony/gobreaker wrapper
│   ├── retry.go                        #   Retry with exponential backoff
│   ├── bulkhead.go                     #   Channel-based semaphore
│   └── timeout.go                      #   Context-based timeout
│
├── observability/                      # OTel helpers
│   ├── tracer.go                       #   Tracer initialization
│   ├── metrics.go                      #   Metrics registration helpers
│   ├── logger.go                       #   zerolog/slog factory
│   └── health.go                       #   gRPC health server helpers
│
├── config/                             # Configuration primitives
│   ├── loader.go                       #   Viper config loader
│   ├── validator.go                    #   Config validation
│   └── env.go                          #   Environment helpers
│
└── testutil/                           # Testing utilities
    ├── fixtures/
    │   ├── nodes.go                    #   Test node fixtures
    │   └── edges.go                    #   Test edge fixtures
    ├── mocks/
    │   ├── mock_store.go               #   Mock Store client
    │   ├── mock_knowledge.go           #   Mock Knowledge client
    │   └── mock_driver.go              #   Mock GraphDriver
    ├── containers/
    │   ├── neo4j.go                    #   Testcontainers Neo4j
    │   ├── redis.go                    #   Testcontainers Redis
    │   └── nats.go                     #   Testcontainers NATS
    └── helpers.go                      #   Common test helpers
```

---

## 3. Shared Graph Domain Types

```go
// pkg/graph/node.go

package graph

import "time"

// EntityNode represents a named entity in the graph
type EntityNode struct {
    UUID          string            `json:"uuid"`
    Name          string            `json:"name"`
    GroupID       string            `json:"group_id"`
    Labels        []string          `json:"labels"`
    Summary       string            `json:"summary"`
    Attributes    map[string]any    `json:"attributes,omitempty"`
    NameEmbedding []float32         `json:"name_embedding,omitempty"`
    CreatedAt     time.Time         `json:"created_at"`
}

// EpisodicNode represents a raw ingested episode
type EpisodicNode struct {
    UUID        string      `json:"uuid"`
    Name        string      `json:"name"`
    GroupID     string      `json:"group_id"`
    Content     string      `json:"content"`
    Source      string      `json:"source"`
    ValidAt     time.Time   `json:"valid_at"`
    Metadata    EpMetadata  `json:"ep_metadata,omitempty"`
    CreatedAt   time.Time   `json:"created_at"`
}

// CommunityNode represents a cluster of co-occurring entities
type CommunityNode struct {
    UUID          string    `json:"uuid"`
    Name          string    `json:"name"`
    GroupID       string    `json:"group_id"`
    Summary       string    `json:"summary"`
    NameEmbedding []float32 `json:"name_embedding,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
}

// SagaNode represents a named sequence of episodes
type SagaNode struct {
    UUID             string    `json:"uuid"`
    Name             string    `json:"name"`
    GroupID          string    `json:"group_id"`
    Summary          string    `json:"summary"`
    FirstEpisodeUUID string    `json:"first_episode_uuid"`
    LastEpisodeUUID  string    `json:"last_episode_uuid"`
    LastSummarizedAt time.Time `json:"last_summarized_at"`
    CreatedAt        time.Time `json:"created_at"`
}
```

```go
// pkg/graph/edge.go

package graph

import "time"

// EntityEdge represents a temporal fact triple between entities
type EntityEdge struct {
    UUID          string    `json:"uuid"`
    SourceUUID    string    `json:"source_uuid"`
    TargetUUID    string    `json:"target_uuid"`
    Name          string    `json:"name"`           // relation type
    GroupID       string    `json:"group_id"`
    Fact          string    `json:"fact"`
    FactEmbedding []float32 `json:"fact_embedding,omitempty"`
    
    // Bi-temporal model
    ValidAt       time.Time  `json:"valid_at"`       // when fact became true
    InvalidAt     *time.Time `json:"invalid_at"`     // when fact stopped being true
    ExpiredAt     *time.Time `json:"expired_at"`     // when record was superseded
    CreatedAt     time.Time  `json:"created_at"`     // when record was written
    
    Attributes    map[string]any `json:"attributes,omitempty"`
}

// EpisodicEdge (MENTIONS) links episode to entity
type EpisodicEdge struct {
    UUID        string `json:"uuid"`
    SourceUUID  string `json:"source_uuid"`  // episode
    TargetUUID  string `json:"target_uuid"`  // entity
    GroupID     string `json:"group_id"`
}

// CommunityEdge (HAS_MEMBER) links community to entity
type CommunityEdge struct {
    UUID        string `json:"uuid"`
    SourceUUID  string `json:"source_uuid"`  // community
    TargetUUID  string `json:"target_uuid"`  // entity
    GroupID     string `json:"group_id"`
}

// HasEpisodeEdge links saga to episode
type HasEpisodeEdge struct {
    UUID        string `json:"uuid"`
    SourceUUID  string `json:"source_uuid"`  // saga
    TargetUUID  string `json:"target_uuid"`  // episode
    GroupID     string `json:"group_id"`
}

// NextEpisodeEdge links episodes in temporal order
type NextEpisodeEdge struct {
    UUID        string `json:"uuid"`
    SourceUUID  string `json:"source_uuid"`  // earlier episode
    TargetUUID  string `json:"target_uuid"`  // later episode
    GroupID     string `json:"group_id"`
}
```

```go
// pkg/graph/temporal.go

package graph

import "time"

// BiTemporal represents the four temporal dimensions of a fact
type BiTemporal struct {
    ValidAt   time.Time  `json:"valid_at"`    // real-world validity start
    InvalidAt *time.Time `json:"invalid_at"`  // real-world validity end
    ExpiredAt *time.Time `json:"expired_at"`  // system-level superseded time
    CreatedAt time.Time  `json:"created_at"`  // system write time
}

// IsCurrentlyValid returns true if the fact is valid at the given point in time
func (bt BiTemporal) IsCurrentlyValid(at time.Time) bool {
    if at.Before(bt.ValidAt) {
        return false
    }
    if bt.InvalidAt != nil && !at.Before(*bt.InvalidAt) {
        return false
    }
    return true
}

// IsActive returns true if the record has not been expired
func (bt BiTemporal) IsActive() bool {
    return bt.ExpiredAt == nil
}
```

---

## 4. Shared Protobuf Definitions

```protobuf
// pkg/proto/common/v1/pagination.proto
syntax = "proto3";
package graphiti.common.v1;

message PaginationRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message PaginationResponse {
  string next_page_token = 1;
  int32 total_count = 2;
}
```

```protobuf
// pkg/proto/common/v1/temporal.proto
syntax = "proto3";
package graphiti.common.v1;

import "google/protobuf/timestamp.proto";

message BiTemporalFilter {
  optional google.protobuf.Timestamp valid_at = 1;
  optional google.protobuf.Timestamp invalid_at = 2;
  optional google.protobuf.Timestamp created_at_start = 3;
  optional google.protobuf.Timestamp created_at_end = 4;
}
```

```protobuf
// pkg/proto/common/v1/errors.proto
syntax = "proto3";
package graphiti.common.v1;

message ErrorDetail {
  string code = 1;        // machine-readable error code
  string message = 2;     // human-readable description
  string field = 3;       // field that caused the error (if applicable)
  map<string, string> metadata = 4;
}
```

---

## 5. Shared Middleware

### 5.1 Tenant Propagation

```go
// pkg/middleware/auth/tenant.go

const TenantIDKey = "x-tenant-id"

// UnaryTenantInterceptor extracts tenant_id from gRPC metadata
// and injects it into context for downstream use
func UnaryTenantInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler) (interface{}, error) {
        
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }
        
        tenantIDs := md.Get(TenantIDKey)
        if len(tenantIDs) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing tenant ID")
        }
        
        ctx = context.WithValue(ctx, tenantIDContextKey, tenantIDs[0])
        return handler(ctx, req)
    }
}

// TenantFromContext extracts tenant_id from context
func TenantFromContext(ctx context.Context) (string, error)
```

### 5.2 Tracing Interceptor

```go
// pkg/middleware/tracing/interceptor.go

// Automatically creates OTel spans for all gRPC calls
// Propagates trace context between services
func UnaryTracingInterceptor(tracer trace.Tracer) grpc.UnaryServerInterceptor
func StreamTracingInterceptor(tracer trace.Tracer) grpc.StreamServerInterceptor
```

### 5.3 Recovery Interceptor

```go
// pkg/middleware/recovery/interceptor.go

// Catches panics, logs stack trace, returns Internal error
func UnaryRecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor
```

---

## 6. Resilience Patterns

```go
// pkg/resilience/circuit_breaker.go

type CircuitBreakerConfig struct {
    Name            string
    MaxRequests     uint32        // half-open state max requests
    Interval        time.Duration // cyclic period for counters
    Timeout         time.Duration // open → half-open transition time
    ReadyToTrip     func(counts gobreaker.Counts) bool
    OnStateChange   func(name string, from, to gobreaker.State)
}

func NewCircuitBreaker(cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker
```

```go
// pkg/resilience/retry.go

type RetryConfig struct {
    MaxAttempts  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    RetryIf      func(error) bool
}

func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error
```

```go
// pkg/resilience/bulkhead.go

// Semaphore-based concurrency limiter
type Bulkhead struct {
    sem chan struct{}
}

func NewBulkhead(maxConcurrent int) *Bulkhead
func (b *Bulkhead) Acquire(ctx context.Context) error
func (b *Bulkhead) Release()
```

---

## 7. Observability Helpers

```go
// pkg/observability/tracer.go

func InitTracer(serviceName, otelEndpoint string) (func(), error)

// pkg/observability/metrics.go  

func InitMetrics(serviceName, otelEndpoint string) (func(), error)

// pkg/observability/logger.go

func NewLogger(level, format string) *slog.Logger

// pkg/observability/health.go

func RegisterHealthServer(server *grpc.Server, checker func() error)
```

---

## 8. Testing Utilities

```go
// pkg/testutil/containers/neo4j.go

// Testcontainers-based Neo4j for integration tests
func StartNeo4j(ctx context.Context) (*Neo4jContainer, error)

// pkg/testutil/fixtures/nodes.go

func NewTestEntityNode(overrides ...func(*graph.EntityNode)) *graph.EntityNode
func NewTestEpisodicNode(overrides ...func(*graph.EpisodicNode)) *graph.EpisodicNode

// pkg/testutil/mocks/mock_store.go

// Auto-generated or hand-written mocks using mockery/gomock
type MockStoreClient struct { ... }
```

---

## 9. Build System

```makefile
# Root Makefile
.PHONY: proto build test lint

# Generate Protobuf code for all services
proto:
	@buf generate

# Build all services
build:
	@for svc in gateway ingestion search knowledge store admin; do \
		go build -o bin/graphiti-$$svc ./services/graphiti-$$svc/cmd/server; \
	done

# Run all tests
test:
	go test ./... -race -count=1

# Lint
lint:
	golangci-lint run ./...
	buf lint

# Generate Wire DI
wire:
	@for svc in gateway ingestion search knowledge store admin; do \
		wire ./services/graphiti-$$svc/internal/infra/wire/; \
	done
```

---

## 10. Go Module Structure

```
go.mod
  module github.com/vnp/graphiti

  require (
    google.golang.org/grpc
    google.golang.org/protobuf
    github.com/go-chi/chi/v5
    github.com/neo4j/neo4j-go-driver/v5
    github.com/redis/go-redis/v9
    github.com/nats-io/nats.go
    github.com/sony/gobreaker
    github.com/google/wire
    github.com/spf13/viper
    go.opentelemetry.io/otel
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
    github.com/rs/zerolog
  )
```

Single `go.mod` at repo root — all services share the same dependency graph. Services are built as separate binaries from `./services/<name>/cmd/server/`.
