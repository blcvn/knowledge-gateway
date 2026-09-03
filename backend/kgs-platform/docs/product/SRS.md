# KGS Platform — Software Requirements Specification (SRS)

**Version:** 1.0 | **Date:** 2026-05-07  
**Module:** `github.com/blcvn/knowledge-gateway/kgs-platform`

---

## 1. Introduction

### 1.1 Purpose
Tài liệu SRS mô tả chi tiết kiến trúc phần mềm, các thành phần hệ thống, giao diện API, mô hình dữ liệu, và yêu cầu kỹ thuật của KGS Platform — nền tảng Knowledge Graph as a Service multi-tenant.

### 1.2 Scope
KGS Platform là microservice Go (Kratos Framework) cung cấp CRUD đồ thị tri thức, hybrid search, overlay graph, business rules, OPA access control, và analytics. Hệ thống sử dụng polyglot persistence: PostgreSQL (source of truth), Neo4j (graph traversal), Qdrant (vector search), Redis (cache/lock/events), NATS (event bus).

### 1.3 Technology Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Language | Go | 1.25 |
| Framework | go-kratos/kratos/v2 | 2.9.2 |
| DI | google/wire | 0.7.0 |
| ORM | gorm.io/gorm | 1.31.1 |
| Graph DB | neo4j-go-driver/v5 | 5.28.4 |
| Vector DB | Qdrant (gRPC client) | — |
| Cache/Lock | redis/go-redis/v9 | 9.18.0 |
| Event Bus | nats-io/nats.go | 1.49.0 |
| Scheduler | go-co-op/gocron/v2 | 2.19.1 |
| Metrics | prometheus/client_golang | 1.23.2 |
| Tracing | opentelemetry/otel | 1.42.0 |
| Schema Validation | xeipuuv/gojsonschema | 1.2.0 |

---

## 2. System Architecture

### 2.1 Layered Architecture

```
┌─────────────────────────────────────────────────┐
│  cmd/server         Entry point (main + wire)   │
├─────────────────────────────────────────────────┤
│  api/               Protobuf definitions        │
│  ├── graph/v1       Graph CRUD + traversal      │
│  ├── ontology/v1    EntityType + RelationType    │
│  ├── registry/v1    App + API Key management    │
│  ├── rules/v1       Business rules              │
│  ├── accesscontrol/v1  OPA policies             │
│  └── helloworld/v1  Health check                │
├─────────────────────────────────────────────────┤
│  internal/server    HTTP(8000) + gRPC(9000)     │
│  internal/server/middleware                      │
│    Tracing→AccessLog→Metrics→Recovery→Auth      │
│    →Namespace→RateLimiter                       │
├─────────────────────────────────────────────────┤
│  internal/service   API↔Biz translation layer   │
├─────────────────────────────────────────────────┤
│  internal/biz       Business logic (use cases)  │
├─────────────────────────────────────────────────┤
│  internal/data      Data access (repositories)  │
├─────────────────────────────────────────────────┤
│  Cross-cutting modules:                         │
│  search│analytics│overlay│outbox│projection     │
│  batch│lock│observability│version               │
└─────────────────────────────────────────────────┘
```

### 2.2 Module Descriptions

| Module | Path | Responsibility |
|--------|------|----------------|
| **biz** | `internal/biz/` | Domain models, use cases, repository interfaces |
| **data** | `internal/data/` | Repository implementations (PG, Neo4j, Qdrant, Redis, NATS) |
| **service** | `internal/service/` | Proto-to-domain mapping, API handlers |
| **server** | `internal/server/` | HTTP/gRPC server bootstrap, middleware stack |
| **search** | `internal/search/` | Hybrid search engine (vector + text + centrality reranking) |
| **analytics** | `internal/analytics/` | Coverage reports, traceability matrix, cluster analysis |
| **overlay** | `internal/overlay/` | Session-scoped overlay graph lifecycle (create/commit/discard) |
| **outbox** | `internal/outbox/` | Transactional outbox worker (PG → Neo4j + Qdrant sync) |
| **projection** | `internal/projection/` | Role-based view filtering, PII masking, ontology sync |
| **batch** | `internal/batch/` | Bulk graph operations (dedup, PG writer, Neo4j writer, vector indexer) |
| **lock** | `internal/lock/` | Distributed locking (Redis-based node/namespace locks) |
| **observability** | `internal/observability/` | Prometheus metrics, OpenTelemetry tracing |
| **conf** | `internal/conf/` | Protobuf-based configuration (Bootstrap, Server, Data) |
| **version** | `internal/version/` | Graph version management |

---

## 3. Component Specifications

### 3.1 Graph Use Case (`internal/biz/graph.go`)

**Interfaces:**

```go
// GraphRepo — graph data persistence (Neo4j-backed)
type GraphRepo interface {
    CreateNode(ctx, appID, tenantID, label string, properties map[string]any) (map[string]any, error)
    GetNode(ctx, appID, tenantID, nodeID string) (map[string]any, error)
    CreateEdge(ctx, appID, tenantID, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error)
    ExecuteQuery(ctx, cypher string, params map[string]any) (map[string]any, error)
    GetFullGraph(ctx, appID, tenantID string, limit, offset int) (*FullGraphResult, error)
    DeleteNode(ctx, appID, tenantID, nodeID string) (edgesRemoved int, err error)
    DeleteEdge(ctx, appID, tenantID, edgeID string) error
    BatchDeleteNodes(ctx, appID, tenantID string, nodeIDs []string) (deleted, edgesRemoved int, err error)
}

// GraphWriteRepo — PostgreSQL write-side (CQRS)
type GraphWriteRepo interface {
    UpsertEntity(ctx, entity WriteEntity) (UpsertOp, error)
    UpsertEdge(ctx, edge WriteEdge) (UpsertOp, error)
    SoftDeleteEntity(ctx, entityID, tenantID string) error
    SoftDeleteEdge(ctx, edgeID, tenantID string) error
    EnqueueOutbox(ctx, rec OutboxRecord) error
    WithTx(ctx, fn func(txRepo GraphWriteRepo) error) error
}

// EntityReader — read-side entity retrieval
type EntityReader interface {
    GetEntity(ctx, appID, tenantID, entityID string) (map[string]any, error)
    EnrichWithFreshVersions(ctx, appID, tenantID string, entities []map[string]any) ([]map[string]any, error)
}
```

**Write Flow:** OPA Check → Ontology Validation → Lock Acquire → PG Write (Entity + Outbox in TX) → Redis Event → Lock Release

**Overlay Detection:** If `properties["overlay_id"]` exists, route to `OverlayDeltaWriter` instead of base graph.

### 3.2 Ontology Validator (`internal/biz/ontology_validator.go`)

**Configuration:**
```go
type OntologyValidatorConfig struct {
    Enabled             bool  // master toggle
    StrictMode          bool  // strict=reject, soft=warn
    SchemaValidation    bool  // JSON Schema property validation
    EdgeConstraintCheck bool  // source/target type enforcement
}
```

**Validation Pipeline:**
1. Entity validation: lookup EntityType → check existence → (optional) validate properties against JSON Schema
2. Edge validation: lookup RelationType → check existence → lookup source/target node labels → verify against sourceTypes/targetTypes constraints
3. Violation handling: StrictMode=true → return error; StrictMode=false → log warning, continue

### 3.3 Query Planner (`internal/biz/query_planner.go`)

Generates parameterized Cypher queries scoped by `app_id` and `tenant_id`:

| Method | Query Type | Parameters |
|--------|-----------|------------|
| `BuildContextQuery` | Neighborhood traversal | label, direction (INCOMING/OUTGOING/BOTH) |
| `BuildImpactQuery` | Downstream path | label, maxDepth |
| `BuildCoverageQuery` | Upstream path | label, maxDepth |
| `BuildSubgraphQuery` | Multi-node subgraph | node_ids list |
| `BuildBatchedTraversalQueries` | Depth-windowed batches | kind, depth, batchWindow(3) |

### 3.4 Search Engine (`internal/search/`)

**Architecture:**
```
HybridSearch(namespace, query, opts)
    ├── VectorRetriever.Search()  → semantic results
    ├── TextRetriever.Search()    → text results
    ├── Blend(semantic, text, alpha)
    ├── CentralityScorer.Scores() → centrality map
    ├── RerankWithCentrality(blended, centrality, beta)
    ├── ApplyFilters(entityTypes, domains, minConfidence, provenanceTypes)
    └── Sort + Truncate(topK)
```

**Defaults:** topK=10, alpha=0.6, beta=0.2, maxTopK=10,000

**Embedding Providers:** `embed_factory.go` supports: OpenAI, AI Proxy, VNP Air, Deterministic (testing)

### 3.5 Overlay Manager (`internal/overlay/`)

**States:** `CREATED → ACTIVE → COMMITTED | PARTIAL | DISCARDED`

**Storage:** Redis with TTL (default 1 hour)

**Commit Protocol:**
1. Load overlay from Redis
2. Detect conflicts (baseVersionID vs currentVersion)
3. Resolve conflicts per policy: `KEEP_OVERLAY` | `KEEP_BASE` | `MERGE` | `REQUIRE_MANUAL`
4. Deduplicate entity/edge deltas (last-write-wins per ID)
5. PostgreSQL transaction: upsert entities + edges + outbox records + delete nodes/edges
6. Create version delta via VersionManager
7. Cleanup overlay from Redis, unbind session
8. Publish commit event via NATS

### 3.6 Outbox Worker (`internal/outbox/worker.go`)

**Configuration:** pollInterval=500ms, batchSize=100, maxAttempts=10

**Sync Operations:**

| Op | Neo4j Action | Qdrant Action |
|----|-------------|---------------|
| `UPSERT_ENTITY` | Upsert node | Upsert vector |
| `UPSERT_EDGE` | Upsert relationship | — |
| `DELETE_ENTITY` | Soft delete node | Delete vector |
| `DELETE_EDGE` | Soft delete relationship | — |

**Metrics:** `kgs_outbox_pending`, `kgs_outbox_lag_seconds`, `kgs_outbox_sync_duration`, `kgs_outbox_failed_total`

### 3.7 Lock Manager (`internal/lock/`)

**Interface:**
```go
type LockManager interface {
    AcquireNodeLock(ctx, namespace, nodeID string, ttl time.Duration) (token string, err error)
    AcquireNamespaceLock(ctx, namespace string, ttl time.Duration) (token string, err error)
    Release(ctx, token string) error
}
```

**Implementation:** Redis-based with configurable TTL (default 30s, env `KGS_LOCK_TTL`)

**Deadlock Prevention:** Edge creation acquires locks on both source/target nodes in sorted ID order.

### 3.8 Analytics Engine (`internal/analytics/`)

| Capability | Method | Description |
|-----------|--------|-------------|
| Coverage | `CoverageReport()` | Entity coverage analysis per domain |
| Traceability | `TraceabilityMatrix()` | Source→target path analysis with max hops |
| Clustering | `ClusterAnalysis()` | Community detection per entity type |

**Caching:** In-memory TTL cache to avoid repeated heavy Cypher queries.

---

## 4. API Specification

### 4.1 REST API Endpoints (from OpenAPI)

| Method | Path | Service | Description |
|--------|------|---------|-------------|
| POST | `/v1/apps` | Registry | Create application |
| GET | `/v1/apps` | Registry | List applications |
| GET | `/v1/apps/{appId}` | Registry | Get application details |
| POST | `/v1/apps/{appId}/keys` | Registry | Issue API key |
| DELETE | `/v1/keys/{keyHash}` | Registry | Revoke API key |
| POST | `/v1/ontology/entities` | Ontology | Create EntityType |
| GET | `/v1/ontology/entities` | Ontology | List EntityTypes |
| POST | `/v1/ontology/relations` | Ontology | Create RelationType |
| GET | `/v1/ontology/relations` | Ontology | List RelationTypes |
| POST | `/v1/graph/nodes` | Graph | Create node |
| GET | `/v1/graph/nodes/{nodeId}` | Graph | Get node |
| POST | `/v1/graph/edges` | Graph | Create edge |
| GET | `/v1/graph/nodes/{nodeId}/context` | Graph | Context traversal |
| GET | `/v1/graph/nodes/{nodeId}/impact` | Graph | Impact analysis |
| GET | `/v1/graph/nodes/{nodeId}/coverage` | Graph | Coverage analysis |
| POST | `/v1/graph/subgraph` | Graph | Subgraph extraction |
| POST | `/v1/rules` | Rules | Create rule |
| GET | `/v1/rules` | Rules | List rules |
| GET | `/v1/rules/{id}` | Rules | Get rule |
| POST | `/v1/policies` | AccessControl | Create policy |
| GET | `/v1/policies` | AccessControl | List policies |
| GET | `/v1/policies/{id}` | AccessControl | Get policy |
| PREFIX | `/kg/*` | Graph | KG Namespace HTTP (extended operations) |
| GET | `/metrics` | System | Prometheus metrics |
| GET | `/healthz` | System | Liveness probe |
| GET | `/readyz` | System | Readiness probe |

### 4.2 gRPC Services

Tất cả REST endpoints đều có tương ứng gRPC service definitions trong `api/` protobuf files.

---

## 5. Data Model

### 5.1 PostgreSQL Tables

| Table | Primary Key | Key Fields |
|-------|-------------|------------|
| `kgs_apps` | AppID (varchar) | AppName, Owner, Status |
| `kgs_api_keys` | KeyHash (varchar) | AppID, KeyPrefix, Scopes, IsRevoked, ExpiresAt |
| `kgs_quotas` | ID (uint) | AppID, QuotaType, Limit |
| `kgs_audit_logs` | ID (uint) | AppID, Action, Actor, Details |
| `kgs_entity_types` | ID (uint) | AppID, TenantID, Name, Schema (JSONB) |
| `kgs_relation_types` | ID (uint) | AppID, TenantID, Name, SourceTypes, TargetTypes (JSONB) |
| `kgs_entities` | EntityID (varchar) | AppID, TenantID, EntityType, Name, Properties (JSONB), Confidence, Version |
| `kgs_edges` | EdgeID (varchar) | AppID, TenantID, FromEntityID, ToEntityID, RelationType, Properties (JSONB) |
| `kgs_rules` | ID (uint) | AppID, TenantID, TriggerType, Cron, CypherQuery, Action |
| `kgs_rule_executions` | ID (uint) | RuleID, Status, StartedAt, EndedAt |
| `kgs_policies` | ID (uint) | AppID, TenantID, Name, RegoContent, IsActive |
| `kgs_sync_outbox` | ID (uint) | Op, EntityID, EdgeID, TenantID, AppID, Payload, Status, Attempts |

### 5.2 Neo4j Graph Model

- **Nodes:** Properties include `id`, `app_id`, `tenant_id`, `label/entity_type`, user-defined properties
- **Relationships:** Properties include `id`, `app_id`, `tenant_id`, `relation_type`
- **Constraints:** Unique index on `(app_id, tenant_id, id)` per node

### 5.3 Qdrant Vector Model

- **Collection:** Configurable (default `kgs-vectors-default`)
- **Vector Size:** Configurable (default 1536 for OpenAI text-embedding-3-small)
- **Payload:** Entity metadata (id, label, name, namespace, properties)

### 5.4 Redis Keys

| Pattern | Purpose | TTL |
|---------|---------|-----|
| `kgs:lock:node:{namespace}:{nodeId}` | Node-level distributed lock | 30s |
| `kgs:lock:ns:{namespace}` | Namespace-level lock | 30s |
| `kgs:overlay:{overlayId}` | Overlay graph data | 1h |
| `kgs:session:{sessionId}` | Session→Overlay binding | 1h |
| `kgs:events:{appId}:{tenantId}` | Event stream | — |
| `kgs:events:nodes` | Global node event stream | — |
| `kgs:ratelimit:{appId}` | Rate limit counter | 60s |

---

## 6. Middleware Pipeline

| Order | Middleware | Function |
|-------|-----------|----------|
| 1 | **Tracing** | OpenTelemetry span creation |
| 2 | **AccessLog** | Structured request/response logging |
| 3 | **Metrics** | Prometheus request counters and histograms |
| 4 | **Recovery** | Panic recovery |
| 5 | **Auth** | API Key validation (Redis cache → PG fallback) |
| 6 | **Namespace** | Extract and inject namespace from headers/path |
| 7 | **RateLimiter** | Redis sliding window rate limiting per app |

---

## 7. Configuration Schema

```yaml
server:
  http: { addr: "0.0.0.0:8000", timeout: "180s" }
  grpc: { addr: "0.0.0.0:9000", timeout: "180s" }
data:
  database: { driver: "postgres", source: "..." }
  redis: { addr: "redis:6379", read_timeout: "0.2s", write_timeout: "0.2s" }
  neo4j: { uri: "bolt://kgs-neo4j:7687", user: "neo4j", database: "kgs" }
  opa: { url: "http://kgs-opa:8181" }
  qdrant: { host: "kgs-qdrant", port: 6333, collection: "kgs-vectors-default", vector_size: 1536 }
  nats: { url: "nats://kgs-nats:4222", stream: "kgs-events" }
  embedding: { provider: "deterministic|openai|aiproxy|vnp_air", vector_size: 1536, timeout: "15s" }
  ontology: { validation_enabled: true, strict_mode: false, schema_validation: false, edge_constraint_check: true, sync_projection: true }
  outbox: { poll_interval_ms: 500, batch_size: 100, max_attempts: 10 }
  reconcile: { cron: "0 3 * * *" }
```

---

## 8. Guardrails & Limits

| Guardrail | Value | Enforcement |
|-----------|-------|-------------|
| Max traversal depth | 10 | `ValidateDepth()` returns `ErrDepthExceeded` |
| Max nodes per query | 10,000 | `ValidateNodeCount()` returns `ErrNodesExceeded` |
| Max search topK | 10,000 | Clamped in `withDefaults()` |
| Outbox max retry | 10 | Skip record after max attempts |
| Lock TTL (default) | 30s | Configurable via `KGS_LOCK_TTL` env |
| Overlay TTL | 1h | Redis key expiry |
| Rate limit | 1000 req/min/app | Configurable per-app quota |
| Batch traversal window | 3 | Depth > 3 split into windowed queries |

---

## 9. Observability

### 9.1 Prometheus Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `kgs_entity_write_total` | Counter | op, status |
| `kgs_entity_write_duration_seconds` | Histogram | op |
| `kgs_outbox_pending` | Gauge | — |
| `kgs_outbox_lag_seconds` | Gauge | — |
| `kgs_outbox_sync_duration_seconds` | Histogram | op |
| `kgs_outbox_failed_total` | Counter | op |
| `kgs_search_duration_seconds` | Histogram | type |
| `kgs_overlay_active` | Gauge | namespace |
| `http_request_duration_seconds` | Histogram | method, path, code |
| `http_requests_total` | Counter | method, path, code |

### 9.2 OpenTelemetry Tracing

- Span per HTTP/gRPC request (auto-instrumented via middleware)
- Custom spans for Neo4j queries, Qdrant operations, OPA evaluations
- Trace context propagation across NATS events

---

## 10. Deployment

### 10.1 Docker

```dockerfile
FROM golang:1.25 AS builder
COPY . /src && WORKDIR /src && RUN make build

FROM debian:stable-slim
COPY --from=builder /src/bin /app
EXPOSE 8000 9000
VOLUME /data/conf
CMD ["./server", "-conf", "/data/conf/config.yaml"]
```

### 10.2 Infrastructure Topology

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  KGS Server │────▶│  PostgreSQL  │     │    Neo4j     │
│  (Go/Kratos)│     │  (Primary)   │     │  (Cluster)   │
│  HTTP:8000  │     └──────────────┘     └──────────────┘
│  gRPC:9000  │            ▲                    ▲
└─────┬───────┘            │                    │
      │              ┌─────┴─────┐         ┌────┴────┐
      ├─────────────▶│   Redis   │         │  Outbox │
      │              │  (Cluster)│         │  Worker │
      ├─────────────▶│           │         └─────────┘
      │              └───────────┘
      ├─────────────▶┌───────────┐
      │              │  Qdrant   │
      ├─────────────▶┌───────────┐
      │              │   NATS    │
      └─────────────▶┌───────────┐
                     │  OPA      │
                     │ (sidecar) │
                     └───────────┘
```

---

## 11. Testing Strategy

| Level | Location | Framework |
|-------|----------|-----------|
| Unit Tests | `*_test.go` in each package | Go testing + testify |
| Integration | `internal/data/*_test.go` | testcontainers-go (PG, Redis) |
| E2E | `internal/service/graph_phase3_e2e_test.go` | Full stack with mocked deps |
| API Testing | `cmd/api-tester/` | Binary for manual API testing |

**Test Coverage Areas:** Graph CRUD, ontology validation, outbox sync, overlay commit/discard, lock management, namespace middleware, auth middleware, rate limiting, search blending, analytics caching.
