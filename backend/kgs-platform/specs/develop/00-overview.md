# Giải Pháp Cập Nhật: Tối Thiểu Services Mới — Ưu Tiên Nâng Cấp Existing Services

> **Nguyên tắc mới:** Mỗi service trong `services/` chịu trách nhiệm thêm một nhóm tính năng KGS mới thay vì tạo service độc lập.  
> **Kết quả:** Từ 10 services đề xuất ban đầu → còn **3 services cần nâng cấp** + **1 service mới bắt buộc**

---

## 1. Phân Tích Services Hiện Có

### Inventory `services/`

| Service | Đang Làm | Routes/Endpoints | Có Thể Absorb KGS Role |
|---------|----------|------------------|------------------------|
| `kg-service` | Graph CRUD, Episode Ingestion, Cognee Dataset | `/v1/graphiti/**`, `/v1/cognee/**`, `/v1/console/graph/**` | ✅ **graph-service + sync-worker + overlay** |
| `search-service` | Cross-engine search, RAG, Connectors, MCP | `/v1/search/**`, `/v1/mcp/**`, `/v1/connectors/**` | ✅ **search-service + query-intel** |
| `pipeline-service` | Knowledge pipeline, orchestration | Knowledge pipeline, MCP pipeline | ✅ **rule-engine-service** |
| `memory-service` | Memobase, SM, Zep memory backends | Memory CRUD per backend | ❌ Domain khác biệt quá |
| `obs-service` | Observability, metrics | Metrics collection | ❌ Không liên quan |
| `storage-service` | File storage | File CRUD | ❌ Domain khác biệt |

### Kết Luận Mapping

```
kg-service      → graph-service + ontology-service + sync-worker + overlay-service
search-service  → search-service (KGS hybrid) + query-intel-service  
pipeline-service → rule-engine-service + policy-service
```

**Services hoàn toàn mới cần tạo:** chỉ **1** — `registry-service` (vì không có service nào quản lý App/API Key)

---

## 2. Kiến Trúc Revised

```
┌──────────────────────────────────────────────────────────────────────┐
│                       Consumer Layer                                  │
└──────────────────────┬───────────────────────────────────────────────┘
                       │  HTTPS
                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│              kgs-gateway  [gateway/ UPGRADE]                          │
│  JWT + API Key auth, Rate limiting, Routing                           │
└──────┬────────────────┬────────────────┬────────────────┬────────────┘
       │gRPC            │gRPC            │gRPC            │gRPC
       ▼                ▼                ▼                ▼
┌────────────┐   ┌────────────────┐  ┌──────────────┐  ┌────────────┐
│ registry-  │   │   kg-service   │  │search-service│  │ pipeline-  │
│ service    │   │   [UPGRADE]    │  │  [UPGRADE]   │  │ service    │
│  [NEW]     │   │                │  │              │  │ [UPGRADE]  │
│ Port 9001  │   │ Port 9003      │  │ Port 9007    │  │ Port 9005  │
│            │   │ Absorbs:       │  │ Absorbs:     │  │ Absorbs:   │
│ App + Key  │   │ - graph-svc    │  │ - search-svc │  │ - rule-eng │
│ management │   │ - ontology-svc │  │ - query-intel│  │ - policy-svc│
│            │   │ - sync-worker  │  │              │  │            │
│            │   │ - overlay-svc  │  │              │  │            │
└────────────┘   └────────────────┘  └──────────────┘  └────────────┘
```

---

## 3. Kế Hoạch Nâng Cấp Chi Tiết

---

### 3.1 ✅ kg-service → KGS Knowledge Graph Service

**Absorbs:** `graph-service` + `ontology-service` + `sync-worker-service` + `overlay-service`

**Lý do kg-service phù hợp nhất:**
- Đã có `IngestUseCase` (episode) → gốc của graph write path
- Đã có `StoreUseCase` (node/edge CRUD) → gốc của graph-service  
- Đã có `KnowledgeUseCase` (ontology, subgraph) → gốc của ontology + query
- Đã có `KGHandler` với full ForwardService adapter
- Ko cần tạo binary mới, chỉ expand routes + usecases

**Thêm vào `kg-service/`:**

```
kg-service/
├── internal/
│   ├── usecase/
│   │   ├── graphiti/           ← Existing (giữ nguyên)
│   │   │   ├── service.go
│   │   │   └── interfaces.go
│   │   ├── cognee/             ← Existing (giữ nguyên)
│   │   ├── kgs/                ← NEW: KGS-spec usecases
│   │   │   ├── graph.go        ← Từ kgs-platform/internal/biz/graph.go
│   │   │   ├── ontology.go     ← Từ kgs-platform/internal/biz/ontology*.go
│   │   │   ├── overlay.go      ← Từ kgs-platform/internal/overlay/
│   │   │   ├── policy.go       ← Từ kgs-platform/internal/biz/opa_client.go
│   │   │   └── sync.go         ← Từ kgs-platform/internal/outbox/
│   │   └── port/
│   ├── adapter/
│   │   └── grpc/
│   │       └── router.go       ← EXTEND: Thêm /v1/graph/**, /v1/ontology/**, /v1/overlay/**
│   ├── domain/
│   │   ├── graphiti/           ← Existing
│   │   ├── cognee/             ← Existing
│   │   └── kgs/                ← NEW: KGS domain entities
│   │       ├── entity.go       ← Từ kgs-platform/internal/data/models_kg.go
│   │       ├── ontology.go
│   │       └── overlay.go
│   └── infra/
│       ├── postgres/           ← Existing
│       ├── neo4j/              ← NEW: Từ kgs-platform/internal/data/graph_query.go
│       ├── qdrant/             ← NEW: Từ kgs-platform/internal/data/qdrant.go
│       ├── redis/              ← NEW: Lock manager + overlay sessions
│       └── outbox/             ← NEW: Từ kgs-platform/internal/outbox/
└── migrations/
    └── kgs/                    ← NEW: Migrations cho kg_entities, kg_edges, kg_sync_outbox
```

**Routes mới thêm vào `router.go`:**

```go
// ── KGS Graph API (New KGS-spec graph-service) ──
router.Handle("POST", "/v1/graph/nodes",               h.adaptHTTP(h.KGSCreateNode))
router.Handle("GET",  "/v1/graph/nodes/*",             h.adaptHTTP(h.KGSGetNode))
router.Handle("PUT",  "/v1/graph/nodes/*",             h.adaptHTTP(h.KGSUpdateNode))
router.Handle("DELETE", "/v1/graph/nodes/*",           h.adaptHTTP(h.KGSDeleteNode))
router.Handle("POST", "/v1/graph/edges",               h.adaptHTTP(h.KGSCreateEdge))
router.Handle("DELETE", "/v1/graph/edges/*",           h.adaptHTTP(h.KGSDeleteEdge))
router.Handle("DELETE", "/v1/graph/nodes",             h.adaptHTTP(h.KGSBatchDeleteNodes))
router.Handle("GET",  "/v1/graph",                     h.adaptHTTP(h.KGSGetFullGraph))

// ── KGS Ontology API (New ontology-service logic) ──
router.Handle("POST", "/v1/ontology/entity-types",     h.adaptHTTP(h.KGSCreateEntityType))
router.Handle("GET",  "/v1/ontology/entity-types",     h.adaptHTTP(h.KGSListEntityTypes))
router.Handle("PUT",  "/v1/ontology/entity-types/*",   h.adaptHTTP(h.KGSUpdateEntityType))
router.Handle("DELETE", "/v1/ontology/entity-types/*", h.adaptHTTP(h.KGSDeleteEntityType))
router.Handle("POST", "/v1/ontology/relation-types",   h.adaptHTTP(h.KGSCreateRelationType))
router.Handle("GET",  "/v1/ontology",                  h.adaptHTTP(h.KGSGetFullOntology))

// ── KGS Overlay API (New overlay-service logic) ──
router.Handle("POST", "/v1/overlay",                    h.adaptHTTP(h.KGSCreateSession))
router.Handle("GET",  "/v1/overlay",                    h.adaptHTTP(h.KGSListSessions))
router.Handle("DELETE", "/v1/overlay/*",                h.adaptHTTP(h.KGSDiscardSession))
router.Handle("POST", "/v1/overlay/*/deltas/entity",    h.adaptHTTP(h.KGSAddEntityDelta))
router.Handle("POST", "/v1/overlay/*/deltas/edge",      h.adaptHTTP(h.KGSAddEdgeDelta))
router.Handle("POST", "/v1/overlay/*/commit",           h.adaptHTTP(h.KGSCommitSession))

// Legacy routes KHÔNG thay đổi:
// /v1/graphiti/**, /v1/cognee/**, /v1/console/graph/** vẫn hoạt động
```

**Handler mới — thin adapter gọi KGS usecases:**

```go
// internal/adapter/grpc/kgs_graph_handler.go
func (h *KGHandler) KGSCreateNode(w http.ResponseWriter, r *http.Request) {
    appID  := r.Header.Get("X-App-ID")
    tenantID := r.Header.Get("X-Tenant-ID")
    
    var req struct {
        EntityType     string         `json:"entity_type"`
        PropertiesJson map[string]any `json:"properties"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    result, err := h.kgsGraph.CreateNode(r.Context(), appID, tenantID, req.EntityType, req.PropertiesJson)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusCreated, result)
}
```

---

### 3.2 ✅ search-service → KGS Search + Query Intelligence Service

**Absorbs:** `search-service` (KGS hybrid) + `query-intel-service`

**Lý do search-service phù hợp nhất:**
- Đã có `SearchOrchestrator` với RRF merge — logic tương tự hybrid search KGS spec
- Đã có fan-out pattern cho multiple engines (graphiti, cognee, memobase, sm)
- Đã có MCP tool support
- Đã có connector pattern cho external data sources
- Ko cần engine mới, chỉ thêm KGS-specific search modes + query intel

**Thêm vào `search-service/`:**

```
search-service/
├── internal/
│   ├── usecase/
│   │   ├── orchestrator/         ← Existing (giữ nguyên: cross-engine search)
│   │   ├── connector/            ← Existing (giữ nguyên)
│   │   ├── mcp/                  ← Existing (giữ nguyên)
│   │   └── kgs/                  ← NEW: KGS-specific search + query intel
│   │       ├── hybrid_search.go  ← Từ kgs-platform/internal/search/ + biz/query_planner.go
│   │       ├── query_intel.go    ← Traversal: Context/Impact/Coverage
│   │       ├── analytics.go      ← Từ kgs-platform/internal/analytics/
│   │       └── view.go           ← Từ kgs-platform/internal/projection/
│   ├── adapter/
│   │   └── grpc/
│   │       └── router.go         ← EXTEND: Thêm /v1/search (KGS), /v1/query/**
│   └── infra/
│       ├── qdrant/               ← NEW: Vector search client
│       ├── neo4j/                ← NEW: Centrality scoring + traversal
│       └── redis/                ← NEW: Embedding cache
```

**Routes mới thêm:**

```go
// ── KGS Hybrid Search (replaces graphiti search) ──
router.Handle("POST", "/v1/kgs/search",         h.adapt(h.KGSHybridSearch))
router.Handle("POST", "/v1/kgs/search/vector",  h.adapt(h.KGSVectorSearch))
router.Handle("POST", "/v1/kgs/search/text",    h.adapt(h.KGSTextSearch))
router.Handle("POST", "/v1/kgs/search/similar", h.adapt(h.KGSSimilarNodes))

// ── KGS Query Intelligence ──
router.Handle("GET",  "/v1/graph/nodes/*/context",  h.adapt(h.KGSGetContext))
router.Handle("GET",  "/v1/graph/nodes/*/impact",   h.adapt(h.KGSGetImpact))
router.Handle("GET",  "/v1/graph/nodes/*/coverage", h.adapt(h.KGSGetCoverage))
router.Handle("POST", "/v1/graph/subgraph",          h.adapt(h.KGSGetSubgraph))
router.Handle("POST", "/v1/query",                   h.adapt(h.KGSExecuteQuery))

// ── KGS Analytics ──
router.Handle("GET", "/v1/analytics/coverage-report",    h.adapt(h.KGSCoverageReport))
router.Handle("GET", "/v1/analytics/traceability-matrix",h.adapt(h.KGSTraceabilityMatrix))

// ── KGS View (Projection Engine) ──
router.Handle("POST", "/v1/views",         h.adapt(h.KGSCreateView))
router.Handle("GET",  "/v1/views",         h.adapt(h.KGSListViews))
router.Handle("GET",  "/v1/views/*/query", h.adapt(h.KGSResolveView))

// Legacy routes KHÔNG thay đổi:
// /v1/search, /v1/search/rag, /v1/ov/search, /v1/sm/search
// /v1/connectors/**, /v1/mcp/** vẫn hoạt động
```

**KGS Hybrid Search — extend SearchOrchestrator:**

```go
// internal/usecase/kgs/hybrid_search.go
// Tái sử dụng SearchOrchestrator pattern nhưng dùng Qdrant + PG + Neo4j

type KGSHybridSearch struct {
    qdrant     QdrantClient      // Từ kgs-platform/internal/data/qdrant.go
    pgFullText PGFullTextClient  // PostgreSQL tsvector search
    neo4j      CentralityScorer  // Degree centrality từ Neo4j
    embedder   EmbeddingProvider // Từ pkg/vectorstore/
    redis      *redis.Client     // Embedding cache
}

func (s *KGSHybridSearch) Search(ctx context.Context, q *KGSSearchQuery) (*KGSSearchResult, error) {
    // Step 1: Embed query (Redis cache)
    vector := s.getEmbedding(ctx, q.QueryText)
    
    // Step 2: Parallel retrieval (tái sử dụng fan-out pattern từ SearchOrchestrator)
    vectorResults := s.qdrant.Search(ctx, q.AppID, vector, q.TopK*3)
    textResults   := s.pgFullText.Search(ctx, q.AppID, q.QueryText, q.TopK*3)
    
    // Step 3: RRF Blend (tái sử dụng rrfMerge từ orchestrator/search.go)
    blended := rrfMerge(vectorResults, textResults, q.Alpha)
    
    // Step 4: Centrality re-ranking
    centrality := s.neo4j.GetDegrees(ctx, nodeIDs(blended))
    reranked    := rerankWithCentrality(blended, centrality, q.Beta)
    
    return applyFilters(reranked, q.Options), nil
}
```

---

### 3.3 ✅ pipeline-service → KGS Rule Engine + Policy Service

**Absorbs:** `rule-engine-service` + `policy-service`

**Lý do pipeline-service phù hợp nhất:**
- Đã có `PipelineUseCase` với orchestration logic → tương tự rule execution
- Đã có `KnowledgeUseCase` pipeline → có thể extend sang rule-triggered pipelines
- Đã có background worker pattern
- MCP pipeline support → tương tự rule action webhook

**Thêm vào `pipeline-service/`:**

```
pipeline-service/
├── internal/
│   ├── usecase/
│   │   ├── pipeline/           ← Existing (giữ nguyên)
│   │   ├── knowledge/          ← Existing (giữ nguyên)
│   │   └── kgs/                ← NEW: KGS rule engine + policy
│   │       ├── rules.go        ← Từ kgs-platform/internal/biz/rules.go
│   │       ├── rule_runner.go  ← Từ kgs-platform/internal/biz/rule_runner.go
│   │       ├── event_runner.go ← Từ kgs-platform/internal/biz/event_runner.go
│   │       ├── policy.go       ← Từ kgs-platform/internal/biz/policy*.go
│   │       └── opa_client.go   ← Từ kgs-platform/internal/biz/opa_client.go
│   └── adapter/
│       └── grpc/
│           └── router.go       ← EXTEND: Thêm /v1/rules/**, /v1/policies/**
```

**Routes mới thêm:**

```go
// ── KGS Rule Engine ──
router.Handle("POST", "/v1/rules",                    h.adapt(h.KGSCreateRule))
router.Handle("GET",  "/v1/rules",                    h.adapt(h.KGSListRules))
router.Handle("GET",  "/v1/rules/*",                  h.adapt(h.KGSGetRule))
router.Handle("PUT",  "/v1/rules/*",                  h.adapt(h.KGSUpdateRule))
router.Handle("DELETE", "/v1/rules/*",                h.adapt(h.KGSDeleteRule))
router.Handle("POST", "/v1/rules/*/activate",         h.adapt(h.KGSActivateRule))
router.Handle("POST", "/v1/rules/*/deactivate",       h.adapt(h.KGSDeactivateRule))
router.Handle("POST", "/v1/rules/*/trigger",          h.adapt(h.KGSTriggerRule))
router.Handle("GET",  "/v1/rules/*/executions",       h.adapt(h.KGSListExecutions))

// ── KGS Policy ──
router.Handle("POST", "/v1/policies",                 h.adapt(h.KGSCreatePolicy))
router.Handle("GET",  "/v1/policies",                 h.adapt(h.KGSListPolicies))
router.Handle("GET",  "/v1/policies/*",               h.adapt(h.KGSGetPolicy))
router.Handle("PUT",  "/v1/policies/*",               h.adapt(h.KGSUpdatePolicy))
router.Handle("DELETE", "/v1/policies/*",             h.adapt(h.KGSDeletePolicy))
router.Handle("POST", "/v1/policies/evaluate",        h.adapt(h.KGSEvaluatePolicy))

// Legacy routes KHÔNG thay đổi
```

**RuleRunner integration — extend existing pipeline worker:**

```go
// cmd/pipeline-service → thêm background workers
func main() {
    // Existing pipeline workers (giữ nguyên)
    
    // NEW: KGS Rule Engine workers
    ruleRunner := kgs.NewRuleRunner(ruleRepo, graphClient, nats)
    go ruleRunner.Start(ctx)  // Cron-based rule execution

    eventRunner := kgs.NewEventRunner(ruleRepo, graphClient, nats)
    go eventRunner.Start(ctx)  // NATS event-triggered rules
    
    // NEW: OPA Policy sync
    policySyncRunner := kgs.NewPolicySyncRunner(policyRepo, opaClient, redis)
    go policySyncRunner.Start(ctx)
}
```

---

### 3.4 🆕 registry-service — Service Mới Bắt Buộc

**Tại sao không thể absorb vào service có sẵn:**
- Không có service nào trong `services/` quản lý App + API Key
- `memory-service` dùng để nhớ user contexts, không phải app management
- `obs-service` chỉ collect metrics
- `storage-service` chỉ quản lý file

**Absorb vào `vnp-platform` không phù hợp** vì vnp-platform là platform admin, không phải KGS registry.

**→ Bắt buộc tạo mới `services/registry-service/`**

```
services/registry-service/         ← NEW service
├── cmd/server/main.go
├── internal/
│   ├── usecase/
│   │   ├── app.go               ← App lifecycle
│   │   ├── apikey.go            ← API Key management
│   │   ├── quota.go             ← Quota management
│   │   └── audit.go             ← Audit log writer
│   ├── domain/
│   │   └── registry.go
│   ├── adapter/grpc/router.go
│   └── infra/postgres/
└── migrations/
```

Tái sử dụng code từ:
- `kgs-platform/internal/biz/registry.go` + `registry_usecase.go`
- `kgs-platform/internal/data/registry.go`

---

### 3.5 🔄 gateway/ → kgs-gateway Upgrade

Không tạo mới, chỉ upgrade `gateway/` như kế hoạch trước.

---

## 4. So Sánh: Kế Hoạch Cũ vs Mới

| Kế hoạch cũ | Kế hoạch mới | Savings |
|------------|-------------|---------|
| 10 services (1 upgrade + 1 mới + 8 extract) | 4 services (3 upgrade + 1 mới) | **-6 services** |
| `graph-service` (mới) | Absorbed vào **kg-service** | ✅ |
| `ontology-service` (mới) | Absorbed vào **kg-service** | ✅ |
| `sync-worker-service` (mới) | Absorbed vào **kg-service** | ✅ |
| `overlay-service` (mới) | Absorbed vào **kg-service** | ✅ |
| `query-intel-service` (mới) | Absorbed vào **search-service** | ✅ |
| `search-service` (mới) | Absorbed vào **search-service** (upgrade) | ✅ |
| `rule-engine-service` (mới) | Absorbed vào **pipeline-service** | ✅ |
| `policy-service` (mới) | Absorbed vào **pipeline-service** | ✅ |
| `registry-service` (mới) | **registry-service** (vẫn cần tạo mới) | — |
| `kgs-gateway` (upgrade) | **kgs-gateway** (upgrade) | — |

---

## 5. Danh Sách Thay Đổi File

### 5.1 `services/kg-service/` — UPGRADE

| File | Thay Đổi |
|------|---------|
| `internal/usecase/kgs/graph.go` | MỚI — copy từ `kgs-platform/internal/biz/graph.go` |
| `internal/usecase/kgs/ontology.go` | MỚI — copy từ `kgs-platform/internal/biz/ontology*.go` |
| `internal/usecase/kgs/overlay.go` | MỚI — copy từ `kgs-platform/internal/overlay/` |
| `internal/usecase/kgs/sync.go` | MỚI — copy từ `kgs-platform/internal/outbox/` |
| `internal/domain/kgs/entity.go` | MỚI — copy từ `kgs-platform/internal/data/models_kg.go` |
| `internal/adapter/grpc/kgs_graph_handler.go` | MỚI — graph routes adapter |
| `internal/adapter/grpc/kgs_ontology_handler.go` | MỚI — ontology routes adapter |
| `internal/adapter/grpc/kgs_overlay_handler.go` | MỚI — overlay routes adapter |
| `internal/adapter/grpc/router.go` | MODIFY — thêm KGS routes |
| `internal/infra/neo4j/client.go` | MỚI — copy từ `kgs-platform/internal/data/graph_query.go` |
| `internal/infra/qdrant/client.go` | MỚI — copy từ `kgs-platform/internal/data/qdrant.go` |
| `internal/infra/outbox/worker.go` | MỚI — copy từ `kgs-platform/internal/outbox/` |
| `migrations/kgs/001_init.sql` | MỚI — kg_entities, kg_edges, kg_sync_outbox tables |
| `cmd/server/main.go` | MODIFY — wire KGS usecases + background workers |

### 5.2 `services/search-service/` — UPGRADE

| File | Thay Đổi |
|------|---------|
| `internal/usecase/kgs/hybrid_search.go` | MỚI — KGS hybrid search pipeline |
| `internal/usecase/kgs/query_intel.go` | MỚI — copy từ `kgs-platform/internal/biz/query_planner.go` |
| `internal/usecase/kgs/analytics.go` | MỚI — copy từ `kgs-platform/internal/analytics/` |
| `internal/usecase/kgs/view.go` | MỚI — copy từ `kgs-platform/internal/projection/` |
| `internal/adapter/grpc/kgs_search_handler.go` | MỚI — KGS search routes adapter |
| `internal/adapter/grpc/kgs_query_handler.go` | MỚI — query intel routes adapter |
| `internal/adapter/grpc/router.go` | MODIFY — thêm /v1/kgs/search/**, /v1/query/**, /v1/analytics/** |
| `internal/infra/qdrant/client.go` | MỚI — Qdrant client |
| `internal/infra/neo4j/client.go` | MỚI — Neo4j traversal client |
| `internal/infra/redis/embed_cache.go` | MỚI — Embedding cache |

### 5.3 `services/pipeline-service/` — UPGRADE

| File | Thay Đổi |
|------|---------|
| `internal/usecase/kgs/rules.go` | MỚI — copy từ `kgs-platform/internal/biz/rules.go` |
| `internal/usecase/kgs/rule_runner.go` | MỚI — copy từ `kgs-platform/internal/biz/rule_runner.go` |
| `internal/usecase/kgs/event_runner.go` | MỚI — copy từ `kgs-platform/internal/biz/event_runner.go` |
| `internal/usecase/kgs/policy.go` | MỚI — copy từ `kgs-platform/internal/biz/policy*.go` + `opa_client.go` |
| `internal/adapter/grpc/kgs_rules_handler.go` | MỚI — rules routes adapter |
| `internal/adapter/grpc/kgs_policy_handler.go` | MỚI — policy routes adapter |
| `internal/adapter/grpc/router.go` | MODIFY — thêm /v1/rules/**, /v1/policies/** |
| `cmd/server/main.go` | MODIFY — wire RuleRunner + EventRunner + PolicySyncRunner goroutines |
| `migrations/kgs/001_rules.sql` | MỚI — rules, rule_executions tables |
| `migrations/kgs/002_policies.sql` | MỚI — policies table |

### 5.4 `services/registry-service/` — NEW

| File | Thay Đổi |
|------|---------|
| `cmd/server/main.go` | MỚI |
| `internal/usecase/app.go` | MỚI — copy từ `kgs-platform/internal/biz/registry_usecase.go` |
| `internal/usecase/apikey.go` | MỚI — API Key management |
| `internal/usecase/quota.go` | MỚI — Quota management |
| `internal/usecase/audit.go` | MỚI — Audit log |
| `internal/adapter/grpc/router.go` | MỚI |
| `internal/infra/postgres/` | MỚI |
| `migrations/001_registry.sql` | MỚI |

### 5.5 `gateway/` — UPGRADE

| File | Thay Đổi |
|------|---------|
| `domain/entity.go` | MODIFY — thêm AppContext |
| `usecase/auth.go` | MODIFY — thêm AuthenticateKGSKey() |
| `usecase/route.go` | MODIFY — thêm routing đến 3 services |
| `adapter/client/registry_client.go` | MỚI — gRPC client cho registry-service |
| `infra/nats_subscriber.go` | MỚI — cache invalidation |

---

## 6. Thứ Tự Triển Khai (Revised)

### Phase 1: Foundation (Week 1) — 7 ngày
1. **registry-service** — Tạo mới (4 ngày)
2. **gateway upgrade** — Thêm API Key auth (3 ngày)

### Phase 2: kg-service Upgrade (Week 2-3) — 8 ngày
3. **kg-service: KGS Graph + Outbox** — Core write path (5 ngày)
4. **kg-service: Ontology** — Schema management (2 ngày)
5. **kg-service: Overlay** — Draft sessions (1 ngày)

### Phase 3: search-service Upgrade (Week 3-4) — 5 ngày
6. **search-service: KGS Hybrid Search** — Vector + Text + Centrality (3 ngày)
7. **search-service: Query Intel** — Traversal + Analytics (2 ngày)

### Phase 4: pipeline-service Upgrade (Week 4-5) — 5 ngày
8. **pipeline-service: Rule Engine** — Cron + Event-driven rules (3 ngày)
9. **pipeline-service: Policy** — OPA policy management (2 ngày)

**Tổng:** ~25 ngày (giảm từ 42 ngày → tiết kiệm ~40%)

---

## 7. Rủi Ro và Giảm Thiểu

| Rủi Ro | Giảm Thiểu |
|--------|-----------|
| kg-service quá lớn (god service) | Tách rõ packages: `usecase/kgs/`, `usecase/graphiti/`, `usecase/cognee/` |
| Test isolation khó khi nhiều domains trong 1 binary | Unit test theo package, integration test theo route group |
| Deploy chậm hơn nếu chỉ update ontology | Feature flag: deploy toàn bộ, enable từng route group |
| Memory footprint tăng | Ko đáng kể — Go binary overhead nhỏ |
| Schema conflict giữa cognee và KGS entity models | Dùng separate DB schemas: `public` cho legacy, `kgs` cho KGS entities |
