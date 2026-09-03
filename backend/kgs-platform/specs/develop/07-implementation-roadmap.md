# Roadmap Revised — 3 Upgrades + 1 New Service

> **Chiến lược:** Tối thiểu binaries mới, tối đa tái sử dụng existing services  
> **Tổng thời gian:** ~25 ngày (giảm từ 42 ngày)

---

## Tổng Quan Services

| # | Service | Chiến Lược | Binary Mới? | Effort |
|---|---------|-----------|------------|--------|
| 0 | `gateway/` | 🔄 UPGRADE | Không | 3 ngày |
| 1 | `services/registry-service` | 🆕 NEW | **Có** (bắt buộc) | 4 ngày |
| 2 | `services/kg-service` | 🔄 UPGRADE | Không | 8 ngày |
| 3 | `services/search-service` | 🔄 UPGRADE | Không | 5 ngày |
| 4 | `services/pipeline-service` | 🔄 UPGRADE | Không | 5 ngày |

---

## Phase 1: Foundation (Week 1) — 7 ngày

### 1.1 registry-service [NEW] — 4 ngày

**Mục tiêu:** API Key management để gateway có thể xác thực `kgs_xxx` keys

**Tasks:**
```
[ ] Tạo services/registry-service/ structure
[ ] Copy + adapt biz/registry.go → usecase/app.go  
[ ] Copy + adapt biz/registry_usecase.go → usecase/apikey.go
[ ] Tạo usecase/quota.go (mới hoàn toàn)
[ ] Tạo usecase/audit.go (mới hoàn toàn)
[ ] Copy data/registry.go → infra/postgres/registry_pg.go
[ ] Tạo adapter/grpc/router.go với ForwardService pattern (giống kg-service)
[ ] Tạo migrations/001_registry.sql
[ ] Wire cmd/server/main.go
[ ] NATS: publish apikey.revoked event khi revoke key
[ ] Unit tests (≥80%)
```

**Key API:**
- `POST /v1/apps` — Register app  
- `POST /v1/apps/:id/keys` — Issue API key (`kgs_xxx` format)
- `DELETE /v1/apps/:id/keys/:key_id` — Revoke key
- `GET /v1/apps/:id/keys/validate?hash=xxx` — Validate (used by gateway)

---

### 1.2 gateway/ Upgrade — 3 ngày

**Mục tiêu:** Hỗ trợ API Key `kgs_xxx` song song với JWT; route đến 3 services mới

**Tasks:**
```
[ ] gateway/domain/entity.go: thêm AppContext struct
[ ] gateway/usecase/auth.go: thêm AuthenticateKGSKey() → gọi registry-service
[ ] gateway/adapter/client/registry_client.go: HTTP/gRPC client cho registry
[ ] gateway/usecase/ratelimit.go: thêm per-app-id quota check  
[ ] gateway/usecase/route.go: thêm routing rules cho KGS routes
[ ] gateway/infra/nats.go: subscribe registry.apikey.revoked → invalidate cache
[ ] Test dual auth (JWT vẫn work, API key mới work)
```

**Routing rules thêm:**
```
/v1/graph/**     → kg-service (mới)
/v1/ontology/**  → kg-service (mới)
/v1/overlay/**   → kg-service (mới)
/v1/kgs/search** → search-service (mới)
/v1/query/**     → search-service (mới)
/v1/analytics/** → search-service (mới)
/v1/views/**     → search-service (mới)
/v1/rules/**     → pipeline-service (mới)
/v1/policies/**  → pipeline-service (mới)
/v1/apps/**      → registry-service
```

---

## Phase 2: kg-service Upgrade (Week 2-3) — 8 ngày

### Mục tiêu
`kg-service` đảm nhận thêm: graph CRUD, ontology schema, overlay sessions, outbox sync

### Structure Mới

```
services/kg-service/
├── cmd/server/
│   └── main.go              ← MODIFY: wire KGS usecases + background workers
├── internal/
│   ├── usecase/
│   │   ├── graphiti/        ← UNCHANGED (IngestUseCase, StoreUseCase, SearchUseCase)
│   │   ├── cognee/          ← UNCHANGED (DatasetUseCase, CognifyUseCase)
│   │   └── kgs/             ← NEW PACKAGE
│   │       ├── graph.go     ← copy từ kgs-platform/internal/biz/graph.go
│   │       ├── ontology.go  ← copy từ biz/ontology.go + ontology_validator.go
│   │       ├── overlay.go   ← copy từ kgs-platform/internal/overlay/
│   │       ├── namespace.go ← copy từ biz/namespace.go
│   │       └── guardrails.go← copy từ biz/graph_guardrails.go
│   ├── domain/
│   │   ├── graphiti/        ← UNCHANGED
│   │   ├── cognee/          ← UNCHANGED
│   │   └── kgs/             ← NEW
│   │       ├── entity.go    ← từ data/models_kg.go
│   │       └── events.go    ← từ data/nats_topics.go
│   ├── adapter/
│   │   └── grpc/
│   │       ├── router.go        ← MODIFY: thêm KGS routes
│   │       ├── kgs_graph.go     ← NEW: graph handlers
│   │       ├── kgs_ontology.go  ← NEW: ontology handlers
│   │       └── kgs_overlay.go   ← NEW: overlay handlers
│   └── infra/
│       ├── postgres/        ← EXTEND
│       │   ├── entity_pg.go ← NEW: từ data/entity_pg.go
│       │   ├── edge_pg.go   ← NEW: từ data/edge_pg.go
│       │   └── ontology_pg.go ← NEW: từ data/ontology.go
│       ├── neo4j/           ← NEW
│       │   └── client.go    ← từ data/graph_node.go + graph_edge.go
│       ├── qdrant/          ← NEW
│       │   └── client.go    ← từ data/qdrant.go
│       ├── redis/           ← NEW
│       │   └── lock.go      ← từ internal/lock/
│       ├── outbox/          ← NEW
│       │   └── worker.go    ← từ internal/outbox/ + data/outbox.go
│       └── nats/            ← NEW
│           └── publisher.go ← từ data/nats.go
└── migrations/
    ├── kgs/
    │   ├── 001_entities.sql  ← NEW: kg_entities table
    │   ├── 002_edges.sql     ← NEW: kg_edges table
    │   ├── 003_outbox.sql    ← NEW: kg_sync_outbox table
    │   └── 004_ontology.sql  ← NEW: entity_types, relation_types tables
```

### Tasks

```
Day 1: Core domain
[ ] Tạo internal/domain/kgs/ với entity models (từ models_kg.go)
[ ] Tạo internal/usecase/kgs/graph.go (copy + adapt từ biz/graph.go)
[ ] Tạo internal/usecase/kgs/namespace.go

Day 2: Infrastructure
[ ] Tạo infra/postgres/entity_pg.go (copy từ data/entity_pg.go)
[ ] Tạo infra/postgres/edge_pg.go (copy từ data/edge_pg.go)
[ ] Tạo infra/postgres/graph_write_pg.go (copy)
[ ] Tạo infra/neo4j/client.go (copy từ data/graph_node.go)
[ ] Tạo infra/redis/lock.go (copy từ internal/lock/)

Day 3: Outbox Worker
[ ] Tạo infra/outbox/worker.go (copy từ internal/outbox/)
[ ] Tạo infra/qdrant/client.go (copy từ data/qdrant.go)
[ ] Tạo infra/nats/publisher.go (copy từ data/nats.go)

Day 4: Graph Handlers + Routes
[ ] Tạo adapter/grpc/kgs_graph.go (HTTP handlers cho graph CRUD)
[ ] Modify adapter/grpc/router.go: thêm /v1/graph/** routes
[ ] Tạo migrations/kgs/*.sql

Day 5: Ontology
[ ] Tạo internal/usecase/kgs/ontology.go (copy từ biz/ontology*.go)
[ ] Tạo infra/postgres/ontology_pg.go (copy từ data/ontology.go)
[ ] Tạo adapter/grpc/kgs_ontology.go (HTTP handlers)
[ ] Modify router.go: thêm /v1/ontology/** routes

Day 6: Overlay
[ ] Tạo internal/usecase/kgs/overlay.go (copy từ internal/overlay/)
[ ] Tạo adapter/grpc/kgs_overlay.go (HTTP handlers)
[ ] Modify router.go: thêm /v1/overlay/** routes

Day 7: Wire + Config
[ ] Modify cmd/server/main.go: wire tất cả KGS dependencies
[ ] Thêm background goroutines: outbox worker, ontology neo4j sync

Day 8: Tests
[ ] Unit tests cho kgs/graph.go
[ ] Unit tests cho kgs/ontology.go
[ ] Integration test: create node → check outbox → verify neo4j sync
```

---

## Phase 3: search-service Upgrade (Week 3-4) — 5 ngày

### Mục tiêu
`search-service` đảm nhận thêm: KGS hybrid search, query intelligence, analytics, views

### Structure Mới

```
services/search-service/
├── internal/
│   ├── usecase/
│   │   ├── orchestrator/    ← UNCHANGED (cross-engine search)
│   │   ├── connector/       ← UNCHANGED
│   │   ├── mcp/             ← UNCHANGED
│   │   └── kgs/             ← NEW PACKAGE
│   │       ├── hybrid.go    ← từ kgs-platform/internal/search/ + biz/query_planner.go
│   │       ├── query_intel.go ← traversal (Context/Impact/Coverage)
│   │       ├── analytics.go ← từ internal/analytics/
│   │       └── view.go      ← từ internal/projection/
│   ├── adapter/
│   │   └── grpc/
│   │       ├── router.go         ← MODIFY: thêm KGS routes
│   │       ├── kgs_search.go     ← NEW: hybrid search handlers
│   │       └── kgs_query.go      ← NEW: query intel + analytics handlers
│   └── infra/
│       ├── kg/              ← Existing (client gọi kg-service)
│       ├── qdrant/          ← NEW: Qdrant direct search client
│       ├── neo4j/           ← NEW: traversal + centrality
│       └── redis/           ← NEW: embedding cache
```

### Tasks

```
Day 1: KGS Hybrid Search
[ ] Tạo infra/qdrant/client.go (copy từ data/qdrant.go)
[ ] Tạo infra/redis/embed_cache.go
[ ] Tạo usecase/kgs/hybrid.go:
    - Embed query (Redis cache-first)
    - Parallel: Qdrant vector + PG full-text
    - RRF blend (tái sử dụng rrfMerge từ orchestrator/search.go)
    - Centrality re-rank

Day 2: Query Intelligence  
[ ] Tạo infra/neo4j/client.go (copy từ data/graph_query.go)
[ ] Tạo usecase/kgs/query_intel.go (copy từ biz/query_planner.go + graph.go read methods)

Day 3: Analytics + Views
[ ] Tạo usecase/kgs/analytics.go (copy từ internal/analytics/)
[ ] Tạo usecase/kgs/view.go (copy từ internal/projection/)

Day 4: Handlers + Routes
[ ] Tạo adapter/grpc/kgs_search.go (handlers: /v1/kgs/search/**, /v1/kgs/search/similar)
[ ] Tạo adapter/grpc/kgs_query.go (handlers: /v1/graph/nodes/*/context|impact|coverage)
[ ] Modify router.go: thêm tất cả KGS routes

Day 5: Tests + Wire
[ ] Modify cmd/server/main.go: wire KGS deps
[ ] Unit tests cho hybrid.go (mock Qdrant + PG)
[ ] Integration test: search với real Qdrant
```

---

## Phase 4: pipeline-service Upgrade (Week 4-5) — 5 ngày

### Mục tiêu
`pipeline-service` đảm nhận thêm: KGS rule engine + policy management

### Structure Mới

```
services/pipeline-service/
├── internal/
│   ├── usecase/
│   │   ├── pipeline/        ← UNCHANGED
│   │   ├── knowledge/       ← UNCHANGED
│   │   └── kgs/             ← NEW PACKAGE
│   │       ├── rules.go     ← copy từ biz/rules.go
│   │       ├── rule_runner.go ← copy từ biz/rule_runner.go
│   │       ├── event_runner.go ← copy từ biz/event_runner.go
│   │       ├── opa_client.go ← copy từ biz/opa_client.go
│   │       ├── policy.go    ← copy từ biz/policy*.go
│   │       └── graph_client.go ← NEW: gRPC/HTTP client gọi kg-service
│   ├── adapter/
│   │   └── grpc/
│   │       ├── router.go        ← MODIFY
│   │       ├── kgs_rules.go     ← NEW
│   │       └── kgs_policy.go    ← NEW
│   └── infra/
│       ├── postgres/            ← EXTEND
│       │   ├── rules_pg.go      ← copy từ data/rules.go
│       │   └── policy_pg.go     ← copy từ data/policy.go
│       └── nats/                ← NEW (subscribe graph events)
└── migrations/
    └── kgs/
        ├── 001_rules.sql
        └── 002_policies.sql
```

### Tasks

```
Day 1: Rule Data + Business Logic
[ ] Tạo infra/postgres/rules_pg.go (copy từ data/rules.go)
[ ] Tạo usecase/kgs/rules.go (copy từ biz/rules.go)
[ ] Tạo usecase/kgs/graph_client.go (HTTP client gọi kg-service /v1/query)

Day 2: Rule Runners
[ ] Tạo usecase/kgs/rule_runner.go (copy từ biz/rule_runner.go)
[ ] Tạo usecase/kgs/event_runner.go (copy từ biz/event_runner.go)
[ ] Cấu hình NATS subscriptions: graph.node.created, graph.node.updated

Day 3: Policy
[ ] Tạo infra/postgres/policy_pg.go (copy từ data/policy.go)
[ ] Tạo usecase/kgs/opa_client.go (copy từ biz/opa_client.go)
[ ] Tạo usecase/kgs/policy.go (copy từ biz/policy*.go)

Day 4: Handlers + Routes
[ ] Tạo adapter/grpc/kgs_rules.go
[ ] Tạo adapter/grpc/kgs_policy.go
[ ] Modify router.go: thêm /v1/rules/**, /v1/policies/**
[ ] Migrations: rules + policies tables

Day 5: Wire + Tests
[ ] Modify cmd/server/main.go: khởi động RuleRunner + EventRunner goroutines
[ ] Unit tests cho rules.go
[ ] Integration test: tạo rule → trigger event → verify execution history
```

---

## Phân Tích Effort So Sánh

| Task | Kế Hoạch Cũ | Kế Hoạch Mới | Tiết Kiệm |
|------|------------|-------------|-----------|
| Binary/service mới | 8 binaries | 1 binary | **-7** |
| Proto files mới | 8 proto files | 0 (dùng HTTP/JSON) | **-8 files** |
| gRPC server code | 8 gRPC servers | 0 (dùng ForwardService pattern) | **-8 servers** |
| DI wiring | 8 main.go | 3 main.go (extend existing) | **-5** |
| Docker images | +8 images | +1 image | **-7 images** |
| Tổng effort | 42 ngày | 25 ngày | **-17 ngày (-40%)** |

---

## Deployment

```yaml
# docker-compose.yml — chỉ thêm 1 service mới
services:
  # EXISTING (unchanged)
  memory-service:   # unchanged
  obs-service:      # unchanged
  storage-service:  # unchanged

  # UPGRADED (same binary, more features)
  kg-service:
    build: ./services/kg-service
    ports: ["9003:8080"]
    environment:
      - NEO4J_URI=bolt://neo4j:7687
      - QDRANT_ADDR=qdrant:6334
      - OUTBOX_ENABLED=true        # Enable outbox background worker
      - REGISTRY_SERVICE_URL=http://registry-service:8080

  search-service:
    build: ./services/search-service
    ports: ["9007:8080"]
    environment:
      - QDRANT_ADDR=qdrant:6334
      - NEO4J_URI=bolt://neo4j:7687
      - KG_SERVICE_URL=http://kg-service:8080

  pipeline-service:
    build: ./services/pipeline-service
    ports: ["9005:8080"]
    environment:
      - OPA_SERVER=http://opa-server:8181
      - KG_SERVICE_URL=http://kg-service:8080
      - NATS_ADDR=nats:4222
      - RULE_ENGINE_ENABLED=true   # Enable rule runner goroutines

  # NEW (only 1 new service)
  registry-service:
    build: ./services/registry-service
    ports: ["9001:8080"]
    environment:
      - DATABASE_DSN=postgres://kgs:password@postgres:5432/kgs_registry
      - NATS_ADDR=nats:4222

  # GATEWAY (upgraded, same binary)
  kgs-gateway:
    build: ./gateway
    ports: ["8080:8080"]
    environment:
      - REGISTRY_SERVICE_URL=http://registry-service:8080
      - KG_SERVICE_ADDR=kg-service:8080
      - SEARCH_SERVICE_ADDR=search-service:8080
      - PIPELINE_SERVICE_ADDR=pipeline-service:8080
      - KGS_NEW_ROUTING=true

  # NEW infrastructure
  neo4j:
    image: neo4j:5
  qdrant:
    image: qdrant/qdrant:latest
  opa-server:
    image: openpolicyagent/opa:latest
```

---

## Rollback Plan

Vì ta chỉ **extend** services hiện có (không thay đổi existing routes), rollback cực kỳ đơn giản:

```bash
# Rollback gateway: disable KGS routes
KGS_NEW_ROUTING=false gateway restart

# Rollback kg-service: disable KGS handlers (feature flag)
KGS_GRAPH_ENABLED=false kg-service restart

# Không có gì bị phá vỡ vì legacy routes không bị chạm
```
