# Các Services Còn Lại — Extract Plans

> **File này** tóm tắt kế hoạch extract cho 6 services còn lại:  
> query-intel-service, policy-service, rule-engine-service,  
> sync-worker-service, search-service, overlay-service

---

# query-intel-service — EXTRACT

> **Strategy:** 📤 EXTRACT  
> **Source:** `biz/query_planner.go` + `biz/graph.go` (read methods) + `biz/view_resolver.go`  
> **Priority:** P1

## Code Tái Sử Dụng

| File | Tái Sử Dụng | Ghi Chú |
|------|-----------|---------|
| `biz/query_planner.go` | ✅ Toàn bộ | BuildContextQuery, BuildImpactQuery, BuildCoverageQuery |
| `biz/graph_guardrails.go` | ✅ Toàn bộ | ValidateDepth, ValidateNodeCount |
| `biz/view_resolver.go` | ✅ Toàn bộ | ViewResolver logic |
| `biz/graph.go` (read) | ✅ GetContext, GetImpact, GetCoverage, GetSubgraph | Chỉ lấy read methods |
| `data/graph_query.go` | ✅ Toàn bộ | Neo4j query execution |
| `internal/projection/` | ✅ Toàn bộ | Projection engine |
| `internal/analytics/` | ✅ Toàn bộ | Analytics engine |

## Cấu Trúc

```
internal/query-intel/
├── biz/
│   ├── query_planner.go    ← TÁI SỬ DỤNG từ biz/query_planner.go
│   ├── guardrails.go       ← TÁI SỬ DỤNG từ biz/graph_guardrails.go
│   ├── analytics.go        ← TÁI SỬ DỤNG từ internal/analytics/
│   └── view_resolver.go    ← TÁI SỬ DỤNG từ biz/view_resolver.go + internal/projection/
├── data/
│   ├── neo4j_query.go      ← TÁI SỬ DỤNG từ data/graph_query.go
│   └── view_pg.go          ← MỚI: ViewDefinition CRUD
└── server/
    └── grpc.go             ← MỚI: gRPC server (port 9004)
```

## Key Proto RPCs

```protobuf
service QueryIntelService {
  rpc GetContext(GetContextRequest) returns (SubgraphResponse);
  rpc GetImpact(GetImpactRequest) returns (SubgraphResponse);
  rpc GetCoverage(GetCoverageRequest) returns (SubgraphResponse);
  rpc GetSubgraph(GetSubgraphRequest) returns (SubgraphResponse);
  rpc ExecuteQuery(ExecuteQueryRequest) returns (QueryResponse);
  rpc GetCoverageReport(CoverageReportRequest) returns (CoverageReportResponse);
  rpc GetTraceabilityMatrix(TraceabilityRequest) returns (TraceabilityResponse);
  rpc ClusterAnalysis(ClusterRequest) returns (ClusterResponse);
  rpc CreateView(CreateViewRequest) returns (ViewDefinition);
  rpc ResolveView(ResolveViewRequest) returns (ResolveViewResponse);
}
```

## Lý Do Tách Riêng
1. **Read-path vs Write-path** — Query intelligence là read-only, có thể scale riêng với nhiều replicas
2. **Neo4j intensive** — Traversal queries tốn tài nguyên Neo4j, cần separate resource budget
3. **Analytics isolation** — Coverage report, traceability chạy heavy aggregations, không nên block writes
4. **Projection engine** — Field-level filtering, PII masking là cross-cutting concern riêng
5. **Guardrails enforcement** — Tập trung guardrails validation ở một chỗ

**Effort:** 5 ngày (4 ngày tái sử dụng + 1 ngày gRPC server)

---

# policy-service — EXTRACT

> **Strategy:** 📤 EXTRACT  
> **Source:** `biz/policy.go` + `biz/opa_client.go` + `biz/policy_sync.go` + `data/policy.go`  
> **Priority:** P1 (graph-service cần policy-service)

## Code Tái Sử Dụng

| File | Tái Sử Dụng | Ghi Chú |
|------|-----------|---------|
| `biz/opa_client.go` | ✅ Toàn bộ | OPA REST client (3.2KB) |
| `biz/policy.go` | ✅ Toàn bộ | PolicyUsecase (1KB) |
| `biz/policy_sync.go` | ✅ Toàn bộ | Background sync runner (1.7KB) |
| `data/policy.go` | ✅ Toàn bộ | PostgreSQL policy repo (1.1KB) |

## OPA Client (Đã Có và Tốt)

```go
// biz/opa_client.go — TÁI SỬ DỤNG TRỰC TIẾP
type OPAClient struct {
    serverURL string
    appID     string
    httpClient *http.Client
    redisCli  *redis.Client
    cacheTTL  time.Duration
}

func (c *OPAClient) EvaluatePolicy(ctx context.Context, appID, action, resource string) (bool, error) {
    // Redis cache (30s TTL)
    // → OPA REST API call
    // Returns: allow/deny
}
```

## gRPC Server

```go
// Wrapper mỏng quanh OPAClient
func (s *PolicyServer) Evaluate(ctx context.Context, req *policypb.EvaluateRequest) (*policypb.EvaluateResponse, error) {
    allow, err := s.opaClient.EvaluatePolicy(ctx, req.AppId, req.Input.Action, req.Input.Resource)
    return &policypb.EvaluateResponse{Allow: allow}, err
}
```

## Lý Do Tách Riêng
1. **OPA dependency** — OPA server là external service, cần dedicated handler
2. **Security isolation** — Policy management phải tách biệt để audit
3. **Multiple callers** — graph-service VÀ query-intel-service đều cần evaluate
4. **Policy hot-reload** — Policy changes cần propagate ngay, cần NATS integration riêng
5. **Rego validation** — Compile Rego syntax cần OPA Go SDK, không nên bundle vào service khác

**Effort:** 3 ngày (tái sử dụng nhiều nhất)

---

# rule-engine-service — EXTRACT

> **Strategy:** 📤 EXTRACT  
> **Source:** `biz/rules.go` + `biz/rule_runner.go` + `biz/event_runner.go` + `data/rules.go`  
> **Priority:** P2

## Code Tái Sử Dụng

| File | Tái Sử Dụng | Ghi Chú |
|------|-----------|---------|
| `biz/rules.go` | ✅ Toàn bộ | Rule CRUD (3.36KB) |
| `biz/rule_runner.go` | ✅ Toàn bộ | Cron scheduler (2.2KB) |
| `biz/event_runner.go` | ✅ Toàn bộ | NATS event handler (2.94KB) |
| `data/rules.go` | ✅ Toàn bộ | PostgreSQL rule repo (1.6KB) |

## Rule Runner (Đã Có)

```go
// biz/rule_runner.go — TÁI SỬ DỤNG TRỰC TIẾP
type RuleRunner struct {
    repo      RuleRepo
    graph     GraphExecutor
    scheduler *cron.Cron
    log       *log.Helper
}

// Tự động load SCHEDULED rules và đăng ký vào cron
func (r *RuleRunner) Start(ctx context.Context) error
```

```go
// biz/event_runner.go — TÁI SỬ DỤNG TRỰC TIẾP  
type EventRunner struct {
    repo  RuleRepo
    graph GraphExecutor
    nats  *nats.Conn
    log   *log.Helper
}

// Subscribe NATS: graph.node.created, graph.node.updated, graph.edge.created
func (r *EventRunner) Start(ctx context.Context) error
```

## graph-service Client trong rule-engine

```go
// internal/rule-engine/biz/graph_client.go — MỚI
type RemoteGraphExecutor struct {
    client graphpb.GraphServiceClient
}

// Implement GraphExecutor interface (gọi graph-service thay vì local)
func (c *RemoteGraphExecutor) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
    resp, err := c.client.ExecuteRuleQuery(ctx, &graphpb.ExecuteRuleQueryRequest{
        CypherTemplate: cypher,
        Params:         paramsToProto(params),
    })
    ...
}
```

## Lý Do Tách Riêng
1. **Long-running workers** — Cron jobs và event subscribers là long-running processes, không nên bundle với API
2. **Independent scaling** — Rule execution load không liên quan đến API load
3. **Failure isolation** — Rule runner crash không ảnh hưởng API
4. **Hot-reload rules** — Thêm/sửa rules không cần restart API servers
5. **Resource isolation** — Heavy Cypher queries từ rules không chiếm resource của graph CRUD

**Effort:** 4 ngày (3 ngày tái sử dụng + 1 ngày gRPC + graph client)

---

# sync-worker-service — EXTRACT

> **Strategy:** 📤 EXTRACT  
> **Source:** `internal/outbox/` + `data/outbox.go` + `data/qdrant.go` + `data/neo4j_constraints.go`  
> **Priority:** P0 (cùng với graph-service)

## Code Tái Sử Dụng

| File | Tái Sử Dụng | Ghi Chú |
|------|-----------|---------|
| `internal/outbox/` | ✅ Toàn bộ | Outbox worker loop |
| `data/outbox.go` | ✅ Toàn bộ | PostgreSQL outbox repo (5.9KB) |
| `data/qdrant.go` | ✅ Toàn bộ | Qdrant client (8.87KB) |
| `data/graph_node.go` (Neo4j write) | ✅ Một phần | MERGE operations |
| `internal/batch/` | ✅ Toàn bộ | Batch processor |
| `pkg/vectorstore/` | ✅ Toàn bộ | Vector store abstraction |

## Outbox Worker (Đã Có)

```go
// internal/outbox/ — TÁI SỬ DỤNG TRỰC TIẾP
// Poll kg_sync_outbox mỗi 500ms
// Process UPSERT_ENTITY → Neo4j + Qdrant
// Process UPSERT_EDGE → Neo4j
// Distributed lock (Redis SETNX)
// Retry với exponential backoff
```

## Embedding Providers (Tái Sử Dụng)

```go
// pkg/vectorstore/ — TÁI SỬ DỤNG TRỰC TIẾP
// OpenAI, AIProxy, Air-VNP embedding providers
// Factory pattern đã implement
```

## Lý Do Tách Riêng
1. **Background-only** — Không có gRPC/HTTP server, pure background worker
2. **Database-heavy** — Read/write PostgreSQL outbox + Neo4j + Qdrant liên tục
3. **Scale riêng** — Có thể chạy nhiều instances với distributed lock
4. **Failure isolation** — Worker crash không ảnh hưởng API services
5. **Reconcile job** — Daily cron job phải chạy độc lập

**Effort:** 3 ngày (tái sử dụng outbox/ và data/ layer)

---

# search-service — REFACTOR

> **Strategy:** 🔨 REFACTOR  
> **Source:** `services/search-service/` + `kgs-platform/internal/search/`  
> **Priority:** P1

## Code Tái Sử Dụng

| Source | Tái Sử Dụng | Ghi Chú |
|--------|-----------|---------|
| `services/search-service/` | ✅ Toàn bộ | Existing search service |
| `kgs-platform/internal/search/` | ✅ Toàn bộ | Hybrid search logic |
| `data/qdrant.go` | ✅ Toàn bộ | Qdrant client (shared) |
| `pkg/vectorstore/` | ✅ Toàn bộ | Embedding providers |

## Kiến Trúc Hybrid Search (Đã Có)

```
Query Embedding (Redis cache) → Parallel Retrieval:
  ├── Qdrant vector search (semantic)
  ├── PostgreSQL full-text (ts_rank)
  └── RRF Blend + Centrality re-ranking (Neo4j degree)
```

## Thay Đổi Chính

1. **Thêm gRPC server** (port 9007) vào `services/search-service/`
2. **Namespace injection** — Tự động filter theo `app_id` namespace
3. **Qdrant collection** — Đổi collection name từ chung sang per-app `kgs_{app_id}`

```protobuf
service SearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc VectorSearch(VectorSearchRequest) returns (SearchResponse);
  rpc TextSearch(TextSearchRequest) returns (SearchResponse);
  rpc SimilarNodes(SimilarNodesRequest) returns (SearchResponse);
}
```

## Lý Do Tách Riêng
1. **Đã có service** — `services/search-service/` đã có, chỉ cần thêm gRPC
2. **Search-specific infrastructure** — Qdrant, embedding providers không cần ở graph-service
3. **Different SLA** — Search queries có latency SLA khác với CRUD
4. **Embedding cache** — Search query embedding cache (1h) không liên quan CRUD cache
5. **Centrality scoring** — Neo4j PageRank/degree queries không nên chạy trong graph CRUD

**Effort:** 4 ngày (2 ngày refactor + 2 ngày gRPC + tests)

---

# overlay-service — EXTRACT

> **Strategy:** 📤 EXTRACT  
> **Source:** `kgs-platform/internal/overlay/`  
> **Priority:** P2

## Code Tái Sử Dụng

| Source | Tái Sử Dụng | Ghi Chú |
|--------|-----------|---------|
| `internal/overlay/` | ✅ Toàn bộ | Overlay session management |

## Overlay Concept (Đã Implement)

```
OverlaySession (Redis Hash, TTL=1h):
  - session_id, app_id, tenant_id, status, created_by

OverlayDelta (Redis Hash):
  - delta_type: ENTITY|EDGE
  - operation: CREATE|UPDATE|DELETE
  - base_version (for conflict detection)
  - payload (new properties)

CommitExecutor:
  - Load deltas từ Redis
  - Check conflicts (version mismatch)
  - Call graph-service.CreateNode/UpdateNode/DeleteNode
  - Cleanup Redis
```

## gRPC Server

```protobuf
service OverlayService {
  rpc CreateSession(CreateSessionRequest) returns (OverlaySession);
  rpc GetSession(GetSessionRequest) returns (OverlaySession);
  rpc DiscardSession(DiscardSessionRequest) returns (google.protobuf.Empty);
  rpc AddEntityDelta(AddEntityDeltaRequest) returns (Delta);
  rpc AddEdgeDelta(AddEdgeDeltaRequest) returns (Delta);
  rpc CommitSession(CommitSessionRequest) returns (CommitResult);
  rpc ApplyOverlay(ApplyOverlayRequest) returns (ApplyOverlayResponse);
}
```

## CommitExecutor → graph-service gRPC

```go
// internal/overlay/biz/commit.go
type CommitExecutor struct {
    graphClient graphpb.GraphServiceClient  // gRPC call
    redis       *redis.Client
}

func (c *CommitExecutor) Execute(ctx context.Context, sessionID string) (*CommitResult, error) {
    // Load deltas từ Redis
    // For each delta:
    //   - Check conflict via graph-service.GetNode()
    //   - Apply via graph-service.CreateNode/UpdateNode/DeleteNode()
    // Return CommitResult{entities_committed, conflicts}
}
```

## Lý Do Tách Riêng
1. **Redis-backed state** — Overlay sessions là ephemeral state, cần Redis-first design riêng
2. **Conflict detection** — Version-based conflict logic phức tạp, không nên mix với main graph
3. **TTL management** — Session expiry, cleanup logic cần isolated
4. **Multi-user collaboration** — Overlay là feature riêng, không phải core graph operation
5. **Commit safety** — Commit flow cần atomic operations trên nhiều services

**Effort:** 5 ngày (overlay/ đã có, cần gRPC wrapper + graph-service integration)

---

# Tổng Hợp Effort

| Service | Strategy | Effort |
|---------|----------|--------|
| kgs-gateway | 🔄 UPGRADE | 3-4 ngày |
| registry-service | 🆕 NEW | 4.5 ngày |
| ontology-service | 📤 EXTRACT | 4.5 ngày |
| graph-service | 🔨 REFACTOR | 6 ngày |
| query-intel-service | 📤 EXTRACT | 5 ngày |
| policy-service | 📤 EXTRACT | 3 ngày |
| rule-engine-service | 📤 EXTRACT | 4 ngày |
| sync-worker-service | 📤 EXTRACT | 3 ngày |
| search-service | 🔨 REFACTOR | 4 ngày |
| overlay-service | 📤 EXTRACT | 5 ngày |
| **TOTAL** | | **~42 ngày** |

---

# Code Reuse Matrix

| Codebase Nguồn | Được Tái Sử Dụng Bởi |
|----------------|---------------------|
| `biz/graph.go` | graph-service (full), query-intel-service (read methods) |
| `biz/ontology_validator.go` | ontology-service (full) |
| `biz/opa_client.go` | policy-service (full) |
| `biz/query_planner.go` | query-intel-service (full) |
| `biz/rules.go` + `rule_runner.go` | rule-engine-service (full) |
| `biz/event_runner.go` | rule-engine-service (full) |
| `data/outbox.go` | sync-worker-service (full) |
| `data/qdrant.go` | sync-worker-service + search-service |
| `data/graph_query.go` | query-intel-service (full) |
| `internal/overlay/` | overlay-service (full) |
| `internal/batch/` | sync-worker-service (full) |
| `internal/search/` | search-service (full) |
| `pkg/vectorstore/` | sync-worker-service + search-service |
| `pkg/telemetry/` | ALL services |
| `pkg/tenant/` | ALL services (context extraction) |
| `gateway/usecase/auth.go` | kgs-gateway (full) |
| `gateway/usecase/ratelimit.go` | kgs-gateway (full) |
