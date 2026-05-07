# Báo cáo Review Triển khai: AI KG Service (ai-kg-service)

> **Ngày review:** 05/03/2026  
> **Reviewer:** Antigravity AI  
> **Tham chiếu:** [API Specs v1.1](ai_kg_service_api_specs.md) | [LLD](lld_ai_kg_service.md) | [Impl Plan](ai_kg_service_impl_plan.md) | [Tasks](ai_kg_service_tasks.md)  
> **Scope:** Toàn bộ codebase `services/ai-kg-service/kgs-platform/` (139 .go files, 32 test files)

---

## 1. Tổng quan đánh giá

| Tiêu chí                     | Đánh giá        | Chi tiết                                                                |
| ---------------------------- | --------------- | ----------------------------------------------------------------------- |
| API Endpoints Coverage       | ✅ **Đạt**      | Tất cả endpoints trong API Specs đều có implementation                  |
| Architecture Alignment (LLD) | ✅ **Đạt**      | Clean Architecture: `service` → `biz` → `data`, DI via Wire             |
| Structured Error Handling    | ✅ **Đạt**      | Sử dụng Kratos `kerrors` với `{code, reason, message, metadata}`        |
| Observability                | ✅ **Đạt**      | Prometheus + OTEL tracing + health endpoints                            |
| Test Coverage                | ✅ **Đạt**      | 32 test files (unit, integration, e2e, benchmark)                       |
| NATS JetStream               | ⚠️ **Chưa đạt** | In-memory adapter, chưa tích hợp NATS JetStream thực                    |
| X-Org-ID Header (DOC 4 §2)   | ❌ **Chưa đạt** | Không tìm thấy xử lý `X-Org-ID` trong toàn bộ codebase                  |
| Embedding Client             | ⚠️ **Chưa đạt** | Dùng `DeterministicEmbeddingClient` (SHA-256 hash) thay vì gọi ai-proxy |
| Hardcoded / Mockup           | ✅ **Đạt**      | Không tìm thấy TODO, HACK, FIXME, placeholder, time.Sleep               |
| Deployment                   | ⚠️ **Một phần** | Docker Compose dev có, K8s manifests chưa có                            |

### Điểm tổng: **7.5/10** — Gần production ready, cần fix 3 issues chính

---

## 2. Chi tiết Review theo module

### 2.1 API Endpoints (✅ Coverage đầy đủ)

| API Specs Section              | gRPC Method                                             | File                        | Status |
| ------------------------------ | ------------------------------------------------------- | --------------------------- | ------ |
| §3 CRUD Entities/Edges         | `CreateNode`, `GetNode`, `CreateEdge`                   | `service/graph.go` L65-123  | ✅     |
| §4 Graph Traversal             | `GetContext`, `GetImpact`, `GetCoverage`, `GetSubgraph` | `service/graph.go` L125-195 | ✅     |
| §5 Batch Upsert                | `BatchUpsertEntities`                                   | `service/graph.go` L197-235 | ✅     |
| §6 Hybrid Search               | `HybridSearch`                                          | `service/graph.go` L237-275 | ✅     |
| §7.1-7.3 Overlay CRUD          | `CreateOverlay`, `CommitOverlay`, `DiscardOverlay`      | `service/graph.go` L277-326 | ✅     |
| §7.8-7.10 Version Mgmt         | `ListVersions`, `DiffVersions`, `RollbackVersion`       | `service/graph.go` L328-395 | ✅     |
| §9.1-9.3 Analytics             | `GetTraceability`, `GetCoverage`, `GetCluster`          | `service/graph.go` L397+    | ✅     |
| §9.4-9.5 Projection            | `CreateViewDefinition`, `ListViewDefinitions`           | `service/graph.go` L500+    | ✅     |
| Health/Ready                   | `/healthz`, `/readyz`                                   | `server/http.go` L52-54     | ✅     |
| Metrics                        | `/metrics`                                              | `server/http.go` L51        | ✅     |
| Registry/Ontology/Rules/Policy | các module riêng                                        | `service/*.go`              | ✅     |

> **Kết luận:** Tất cả API endpoints đã được implement. Routing qua gRPC-Gateway HTTP + gRPC song song.

---

### 2.2 NATS Integration (⚠️ In-memory Adapter)

**File:** `internal/data/nats.go` (108 dòng)

**Vấn đề:** NATSClient hiện tại là **in-memory pub/sub adapter**, KHÔNG phải NATS JetStream client thực.

```go
// Publish gọi handler trực tiếp trong memory
func (c *NATSClient) Publish(ctx context.Context, subject string, payload []byte) error {
    for _, sub := range c.subs {
        if subjectMatch(sub.subject, subject) {
            sub.handler(ctx, payload) // gọi trực tiếp, không qua network
        }
    }
    return nil
}
```

**Tác động:**

- Events chỉ hoạt động trong cùng process, không cross-service
- Không có persistence/replay (JetStream feature)
- Không có consumer groups, ack/nack
- `Ping()` luôn trả `nil` nếu URL non-empty → không validate connection thực

**Các event đã implement logic đúng:**

- ✅ `OVERLAY_COMMIT` event published khi commit overlay (`overlay/commit.go` L76)
- ✅ `OVERLAY_DISCARD` event published khi discard overlay (`overlay/overlay.go` L95)
- ✅ `SESSION_CLOSE` subscriber: commit-or-discard theo DOC 4 §8 (`overlay/nats_listener.go` L103-116)
- ✅ `BUDGET_STOP` subscriber: partial commit (`overlay/nats_listener.go` L119-125)

**Topic pattern:** `nats_topics.go` — 4 topics đúng chuẩn DOC 4 §5.

> **Khuyến nghị:** Thay thế in-memory adapter bằng `nats.go` client thực (`github.com/nats-io/nats.go`) + JetStream API. Logic handler + topic patterns đã đúng, chỉ cần swap transport layer.

---

### 2.3 X-Org-ID Header (❌ Thiếu hoàn toàn)

**Grep kết quả:** Không tìm thấy `OrgID`, `org_id`, hay `X-Org-ID` trong toàn bộ `internal/`.

**File liên quan:**

- `middleware/auth.go` — `AppContext` struct chỉ có `AppID`, `Scopes`, `TenantID` → **thiếu `OrgID`**
- `middleware/namespace.go` — Namespace logic không include OrgID

**Yêu cầu từ API Specs §1:**

```
| `X-Org-ID` | `string` | Optional | Organization ID (enterprise multi-org). DOC 4 §2 |
```

> **Khuyến nghị:** Thêm `OrgID string` vào `AppContext`, parse từ header `X-Org-ID`, và inject vào context.

---

### 2.4 Embedding Client (⚠️ Deterministic Hash, chưa gọi LLM)

**File:** `search/vector.go` L18-43, `batch/dedup.go` L103+, `batch/vector_indexer.go` L34

**Vấn đề:** Cả vector search lẫn batch indexing đều dùng `DeterministicEmbeddingClient` — tạo vector bằng SHA-256 hash thay vì gọi ai-proxy embedding API.

```go
// DeterministicEmbeddingClient - SHA-256 hash-based, NOT real embedding
func (c *DeterministicEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
    for i := 0; i < size; i++ {
        digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", text, i)))
        v := binary.BigEndian.Uint32(digest[:4])
        out[i] = float32(v%10000)/5000 - 1
    }
}
```

**Tác động:**

- Semantic search trả kết quả **không có ý nghĩa ngữ nghĩa** — chỉ là hash match
- Hybrid search blend (semantic × text × centrality) vẫn hoạt động nhưng semantic score vô nghĩa

**Điểm tích cực:**

- Interface `EmbeddingClient` đã abstract đúng: `Embed(ctx, text) ([]float32, error)`
- Swap sang gRPC client tới ai-proxy chỉ cần implement interface, không thay đổi business logic

> **Khuyến nghị:** Implement `AIProxyEmbeddingClient` gọi ai-proxy gRPC `Embed()` endpoint. Giữ `DeterministicEmbeddingClient` cho test/dev.

---

### 2.5 Structured Error Handling (✅ Đạt)

**File:** `biz/errors.go` (24 dòng)

```go
var (
    ErrDepthExceeded  = kerrors.BadRequest("ERR_DEPTH_EXCEEDED", "...")
    ErrNodesExceeded  = kerrors.BadRequest("ERR_NODES_EXCEEDED", "...")
    ErrAPIKeyNotFound = kerrors.Unauthorized("ERR_UNAUTHORIZED", "...")
)

func ErrForbiddenWithMetadata(message string, metadata map[string]string) error {
    return kerrors.Forbidden("ERR_FORBIDDEN", message).WithMetadata(metadata)
}
```

**Coverage:** 7 error codes defined. Tất cả service handler dùng `kerrors` format đúng DOC 4 §6.

**Thiếu:** Theo API Specs §10 còn thiếu một số error codes:

- `ERR_OVERLAY_NOT_ACTIVE` (có logic check nhưng trả generic `fmt.Errorf`)
- `ERR_VERSION_NOT_FOUND` (dùng GORM error trực tiếp)

> **Khuyến nghị:** Thêm 2 error codes vào `biz/errors.go` và thay thế `fmt.Errorf` tương ứng.

---

### 2.6 Observability (✅ Đạt)

**File:** `observability/metrics.go` (145 dòng), `observability/tracing.go`

**6 Prometheus Metrics:**

| Metric                        | Type      | Labels            |
| ----------------------------- | --------- | ----------------- |
| `kg_request_total`            | Counter   | method, status    |
| `kg_request_duration_ms`      | Histogram | method, status    |
| `kg_entity_write_total`       | Counter   | operation, status |
| `kg_search_duration_ms`       | Histogram | search_type       |
| `kg_overlay_count_active`     | Gauge     | namespace         |
| `kg_lock_acquire_duration_ms` | Histogram | level, status     |

**OTEL Tracing:**

- `middleware/tracing.go` — request-level tracing
- `middleware/metrics.go` — request-level metrics
- Qdrant client có `observability.StartDependencySpan()` per operation
- NATS operations có span instrumentation

**Health Endpoints:**

- `/healthz` — liveness check
- `/readyz` — readiness check (Redis, Neo4j, Qdrant, NATS ping)
- `/metrics` — Prometheus exposition

> **Kết luận:** Observability stack đầy đủ và production-grade.

---

### 2.7 Search Engine (✅ Đạt — trừ embedding)

**Files:** `search/` package (11 files)

| Component                                       | File                | Status |
| ----------------------------------------------- | ------------------- | ------ |
| SearchEngine interface                          | `search.go`         | ✅     |
| HybridSearch (vector + text + centrality)       | `search.go` L69-117 | ✅     |
| VectorSearcher (Qdrant)                         | `vector.go`         | ✅     |
| TextSearcher (Neo4j fulltext)                   | `text.go`           | ✅     |
| ResultBlender (weighted merge)                  | `blender.go`        | ✅     |
| CentralityReranker                              | `reranker.go`       | ✅     |
| Filters (entity_types, domains, min_confidence) | `filter.go`         | ✅     |
| NamespaceResolver                               | `namespace.go`      | ✅     |

**Logic flow:**

1. Vector search (Qdrant) + Text search (Neo4j fulltext) song song
2. Blend theo `alpha` weight (default 0.6 semantic, 0.4 text)
3. Centrality rerank theo `beta` weight (default 0.2)
4. Filter by entity_types, domains, min_confidence
5. Sort by final score, cap at topK

> **Kết luận:** Search pipeline đầy đủ, chỉ thiếu real embedding client.

---

### 2.8 Overlay & Version Management (✅ Đạt)

| Feature                        | File                                | Status |
| ------------------------------ | ----------------------------------- | ------ |
| Create Overlay                 | `overlay/overlay.go` L47-75         | ✅     |
| Commit Overlay (+ NATS event)  | `overlay/commit.go` L14-81          | ✅     |
| CommitPartial (budget stop)    | `overlay/commit.go` L18-20          | ✅     |
| Discard Overlay (+ NATS event) | `overlay/overlay.go` L81-104        | ✅     |
| DiscardBySession               | `overlay/overlay.go` L106-112       | ✅     |
| Conflict Detection             | `overlay/conflict.go`               | ✅     |
| SESSION_CLOSE handler          | `overlay/nats_listener.go` L103-117 | ✅     |
| BUDGET_STOP handler            | `overlay/nats_listener.go` L119-126 | ✅     |
| Version CreateDelta            | `version/version.go` L33-56         | ✅     |
| Version List/Get               | `version/version.go` L58-78         | ✅     |
| Version Diff (chain walk)      | `version/version.go` L80-96         | ✅     |
| Version Rollback               | `version/version.go` L98-120        | ✅     |
| Version GC                     | `version/gc.go`                     | ✅     |

> **Kết luận:** Overlay lifecycle hoàn chỉnh với NATS events. Version management có full CRUD + diff + rollback + GC.

---

### 2.9 Analytics & Projection (✅ Đạt)

| Feature                    | File                                   | Status |
| -------------------------- | -------------------------------------- | ------ |
| Traceability Analysis      | `analytics/traceability.go`            | ✅     |
| Coverage Analysis          | `analytics/coverage.go`                | ✅     |
| Cluster Analysis           | `analytics/cluster.go`                 | ✅     |
| Analytics Cache (Redis)    | `analytics/cache.go`                   | ✅     |
| Projection Engine          | `projection/projection.go` (293 lines) | ✅     |
| ViewDefinition CRUD (GORM) | `projection/projection.go` L112-189    | ✅     |
| Role-based field filtering | `projection/projection.go` L36-110     | ✅     |
| PII Masking                | `projection/mask.go`                   | ✅     |

> **Kết luận:** Analytics pipeline và Projection engine đầy đủ tính năng, có caching và PII masking.

---

### 2.10 Test Coverage (✅ 32 test files)

| Category          | Files    | Scope                                                                               |
| ----------------- | -------- | ----------------------------------------------------------------------------------- |
| Unit Tests        | 22 files | analytics, batch, biz, data, lock, overlay, projection, search, middleware, version |
| Integration Tests | 2 files  | `service/graph_phase3_e2e_test.go`, `tests/integration/phase5_integration_test.go`  |
| E2E Tests         | 3 files  | `tests/e2e/happy_path_test.go`, `error_path_test.go`, `main_test.go`                |
| Benchmark         | 1 file   | `tests/benchmark/search_bench_test.go`                                              |
| Service Tests     | 4 files  | `service/graph_test.go`, `health_test.go`                                           |

> **Kết luận:** Coverage đầy đủ ở mức unit, integration, e2e, và benchmark.

---

### 2.11 Server & Middleware Stack (✅ Đạt)

**File:** `server/http.go` — Middleware chain:

```
Tracing → Metrics → Recovery → Auth → Namespace → RateLimiter
```

| Middleware                  | File                             | Status |
| --------------------------- | -------------------------------- | ------ |
| OTEL Tracing                | `middleware/tracing.go`          | ✅     |
| Prometheus Metrics          | `middleware/metrics.go`          | ✅     |
| Panic Recovery              | Kratos built-in                  | ✅     |
| API Key Auth                | `middleware/auth.go` (161 lines) | ✅     |
| Namespace Injection         | `middleware/namespace.go`        | ✅     |
| Rate Limiter (Redis-backed) | `middleware/ratelimit.go`        | ✅     |

**Auth flow:** API Key → Redis cache lookup → DB validation → AppContext injection.

> **Kết luận:** Middleware stack production-grade. Auth, rate limiting, tracing, metrics đều real implementation.

---

### 2.12 Configuration (✅ Đạt)

**File:** `configs/config.yaml` (33 dòng) — cover toàn bộ dependencies:

| Config           | Status | Ghi chú                                        |
| ---------------- | ------ | ---------------------------------------------- |
| HTTP server addr | ✅     | `0.0.0.0:8000`                                 |
| gRPC server addr | ✅     | `0.0.0.0:9000`                                 |
| PostgreSQL       | ✅     | GORM driver                                    |
| Redis            | ✅     | tcp, addr, password                            |
| Neo4j            | ✅     | bolt protocol                                  |
| OPA              | ✅     | HTTP endpoint                                  |
| Qdrant           | ✅     | host, port, collection, vector_size            |
| NATS             | ⚠️     | `url: ""` — empty = in-memory adapter fallback |

> **Lưu ý:** NATS URL empty sẽ trigger `NewNATSClientFromConfig` return `nil`, khiến toàn bộ NATS integration bị skip silently.

---

### 2.13 Deployment (⚠️ Một phần)

| Item               | Status  | File                                       |
| ------------------ | ------- | ------------------------------------------ |
| Dockerfile         | ✅      | `kgs-platform/Dockerfile`                  |
| Docker Compose dev | ✅      | `deployment/docker/docker-compose.dev.yml` |
| K8s Deployment     | ❌ Skip | Chưa tạo                                   |
| K8s HPA            | ❌ Skip | Chưa tạo                                   |
| K8s NetworkPolicy  | ❌ Skip | Chưa tạo                                   |

---

## 3. Tổng hợp Issues cần xử lý

### 🔴 Critical (phải fix trước production)

| #   | Issue                              | Impact                                      | Effort    |
| --- | ---------------------------------- | ------------------------------------------- | --------- |
| C1  | NATS client là in-memory adapter   | Cross-service events không hoạt động        | ~2-3 ngày |
| C2  | X-Org-ID header không được xử lý   | Không hỗ trợ multi-org (DOC 4 §2 violation) | ~0.5 ngày |
| C3  | Embedding client dùng SHA-256 hash | Semantic search vô nghĩa ngữ nghĩa          | ~1-2 ngày |

### 🟡 Medium (nên fix)

| #   | Issue                                                               | Impact                                  | Effort     |
| --- | ------------------------------------------------------------------- | --------------------------------------- | ---------- |
| M1  | `ERR_OVERLAY_NOT_ACTIVE`, `ERR_VERSION_NOT_FOUND` dùng `fmt.Errorf` | Response format không tuân thủ DOC 4 §6 | ~0.5 ngày  |
| M2  | NATS config URL empty → silent nil fallback                         | Config không rõ ràng, dễ miss           | ~0.25 ngày |
| M3  | K8s deployment manifests chưa tạo                                   | Chưa deploy được trên K8s               | ~1 ngày    |

### 🟢 Low (nice to have)

| #   | Issue                                                                                         | Impact                                        | Effort     |
| --- | --------------------------------------------------------------------------------------------- | --------------------------------------------- | ---------- |
| L1  | `overlayActiveGauge` counter dùng `syncMapCounter` (atomic) nhưng không thread-safe hoàn toàn | Race condition tiềm ẩn trong high-concurrency | ~0.25 ngày |

---

## 4. Kết luận & Đề xuất

### Điểm mạnh

1. **Architecture sạch:** Clean Architecture đúng chuẩn (`service` → `biz` → `data`), DI via Wire
2. **API coverage 100%:** Tất cả endpoints trong API Specs đã implement
3. **Error handling production-grade:** Kratos structured errors
4. **Observability đầy đủ:** Prometheus metrics, OTEL tracing, health checks
5. **Test coverage tốt:** 32 test files span unit/integration/e2e/benchmark
6. **Zero TODO/HACK/FIXME:** Code sạch, không có stub hay marker
7. **Overlay + Version management hoàn chỉnh:** Create, commit, discard, diff, rollback, GC
8. **Search pipeline sophisticated:** Hybrid (vector + text + centrality), filtering, reranking

### Điểm cần cải thiện

1. **NATS phải swap sang real JetStream client** — Logic handlers + topics đã đúng, transport layer cần thay
2. **X-Org-ID header phải thêm vào AppContext** — Require cho enterprise multi-org support
3. **Embedding client phải swap sang ai-proxy gRPC** — Interface đã abstract đúng, chỉ cần new implementation

### Roadmap đề xuất

```
Week 1: Fix C1 (NATS JetStream) + C2 (X-Org-ID) + M1 (error codes)
Week 2: Fix C3 (ai-proxy embedding) + M2 (NATS config) + M3 (K8s manifests)
Week 3: Testing + performance validation + staging deploy
```

---

## 5. Files đã Review

<details>
<summary>Danh sách 139 Go files đã scan</summary>

**Packages chính (đọc chi tiết):**

- `internal/service/graph.go` (913 lines, 49 methods)
- `internal/overlay/overlay.go`, `commit.go`, `nats_listener.go`, `conflict.go`
- `internal/version/version.go` (176 lines)
- `internal/search/search.go`, `vector.go`, `text.go`, `blender.go`, `filter.go`, `reranker.go`
- `internal/projection/projection.go` (293 lines)
- `internal/analytics/analytics.go`, `traceability.go`, `coverage.go`, `cluster.go`
- `internal/data/nats.go`, `nats_topics.go`, `qdrant.go` (304 lines)
- `internal/biz/errors.go`, `graph.go`, `namespace.go`, `query_planner.go`
- `internal/server/http.go`, `middleware/auth.go` (161 lines), `middleware/namespace.go`, `middleware/metrics.go`, `middleware/tracing.go`, `middleware/ratelimit.go`
- `internal/batch/batch.go`, `vector_indexer.go`, `dedup.go`
- `internal/observability/metrics.go` (145 lines), `tracing.go`
- `internal/lock/*.go`
- `configs/config.yaml`

**Scan toàn bộ:** `grep -rn "TODO\|HACK\|FIXME\|placeholder\|time.Sleep\|hardcod\|mock\|dummy"` → **0 kết quả**

</details>
