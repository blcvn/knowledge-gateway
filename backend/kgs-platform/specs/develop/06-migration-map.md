# Migration Map Revised — Monolith Logic → Existing Services

> **Chiến lược mới:** Tái sử dụng code từ `kgs-platform/internal/` vào 3 services hiện có thay vì tạo 8 services mới

---

## 1. Mapping Tổng Quan

```
kgs-platform/internal/biz/         →   Vào 3 services theo domain
├── graph.go ─────────────────────────→  services/kg-service/internal/usecase/kgs/
├── ontology*.go ─────────────────────→  services/kg-service/internal/usecase/kgs/
├── overlay/ ─────────────────────────→  services/kg-service/internal/usecase/kgs/
├── outbox/ ──────────────────────────→  services/kg-service/internal/infra/outbox/
│
├── query_planner.go ─────────────────→  services/search-service/internal/usecase/kgs/
├── analytics/ ───────────────────────→  services/search-service/internal/usecase/kgs/
├── projection/ ──────────────────────→  services/search-service/internal/usecase/kgs/
│
├── rules.go ─────────────────────────→  services/pipeline-service/internal/usecase/kgs/
├── rule_runner.go ───────────────────→  services/pipeline-service/internal/usecase/kgs/
├── event_runner.go ──────────────────→  services/pipeline-service/internal/usecase/kgs/
├── opa_client.go ────────────────────→  services/pipeline-service/internal/usecase/kgs/
├── policy*.go ───────────────────────→  services/pipeline-service/internal/usecase/kgs/
│
└── registry*.go ─────────────────────→  services/registry-service/internal/usecase/  [NEW]
```

---

## 2. Chi Tiết: `kgs-platform/internal/biz/` → Destination

| File Nguồn | Kích Thước | → Destination Service | Package Đích |
|-----------|------------|----------------------|-------------|
| `biz/graph.go` | 38.8KB | **kg-service** | `usecase/kgs/graph.go` |
| `biz/graph_write.go` | 1.8KB | **kg-service** | `usecase/kgs/graph_write.go` |
| `biz/graph_guardrails.go` | 639B | **kg-service** + **search-service** | `usecase/kgs/guardrails.go` |
| `biz/namespace.go` | 405B | **kg-service** | `usecase/kgs/namespace.go` |
| `biz/full_graph.go` | 363B | **kg-service** | `usecase/kgs/graph.go` (merged) |
| `biz/ontology.go` | 1.9KB | **kg-service** | `usecase/kgs/ontology.go` |
| `biz/ontology_validator.go` | 5.5KB | **kg-service** | `usecase/kgs/ontology_validator.go` |
| `biz/ontology_sync.go` | 988B | **kg-service** | `usecase/kgs/ontology_sync.go` |
| `biz/query_planner.go` | 4.0KB | **search-service** | `usecase/kgs/query_planner.go` |
| `biz/view_resolver.go` | 764B | **search-service** | `usecase/kgs/view.go` |
| `biz/rules.go` | 3.4KB | **pipeline-service** | `usecase/kgs/rules.go` |
| `biz/rule_runner.go` | 2.2KB | **pipeline-service** | `usecase/kgs/rule_runner.go` |
| `biz/event_runner.go` | 2.9KB | **pipeline-service** | `usecase/kgs/event_runner.go` |
| `biz/opa_client.go` | 3.3KB | **pipeline-service** | `usecase/kgs/opa_client.go` |
| `biz/policy.go` | 1.0KB | **pipeline-service** | `usecase/kgs/policy.go` |
| `biz/policy_sync.go` | 1.7KB | **pipeline-service** | `usecase/kgs/policy_sync.go` |
| `biz/registry.go` | 2.3KB | **registry-service** [NEW] | `usecase/app.go` |
| `biz/registry_usecase.go` | 4.0KB | **registry-service** [NEW] | `usecase/apikey.go` |
| `biz/errors.go` | 1.1KB | ALL services | shared `domain/errors.go` |
| `biz/validator.go` | 1.1KB | **kg-service** | `usecase/kgs/validator.go` |

---

## 3. Chi Tiết: `kgs-platform/internal/data/` → Destination

| File Nguồn | Kích Thước | → Destination Service | Package Đích |
|-----------|------------|----------------------|-------------|
| `data/models_kg.go` | 9.8KB | **kg-service** | `domain/kgs/entity.go` + `infra/postgres/models.go` |
| `data/entity_pg.go` | 10.7KB | **kg-service** | `infra/postgres/entity_pg.go` |
| `data/edge_pg.go` | 4.5KB | **kg-service** | `infra/postgres/edge_pg.go` |
| `data/graph_write_pg.go` | 7.0KB | **kg-service** | `infra/postgres/graph_write_pg.go` |
| `data/entity_reader.go` | 4.9KB | **kg-service** | `infra/postgres/entity_reader.go` |
| `data/entity_reader_kg_namespace.go` | 16.1KB | **kg-service** | `infra/postgres/entity_reader_ns.go` |
| `data/entity_query.go` | 4.0KB | **kg-service** | `infra/postgres/entity_query.go` |
| `data/graph_node.go` | 9.9KB | **kg-service** | `infra/neo4j/graph_node.go` |
| `data/graph_edge.go` | 3.9KB | **kg-service** | `infra/neo4j/graph_edge.go` |
| `data/graph_query.go` | 10.5KB | **search-service** | `infra/neo4j/graph_query.go` |
| `data/outbox.go` | 5.9KB | **kg-service** | `infra/outbox/outbox.go` |
| `data/qdrant.go` | 8.9KB | **kg-service** + **search-service** | `infra/qdrant/client.go` |
| `data/nats.go` | 2.9KB | **kg-service** | `infra/nats/nats.go` |
| `data/nats_topics.go` | 1.4KB | ALL | `domain/kgs/events.go` (constants) |
| `data/ontology.go` | 2.8KB | **kg-service** | `infra/postgres/ontology_pg.go` |
| `data/policy.go` | 1.1KB | **pipeline-service** | `infra/postgres/policy_pg.go` |
| `data/rules.go` | 1.6KB | **pipeline-service** | `infra/postgres/rules_pg.go` |
| `data/registry.go` | 3.9KB | **registry-service** [NEW] | `infra/postgres/registry_pg.go` |
| `data/neo4j_constraints.go` | 1.1KB | **kg-service** | `infra/neo4j/constraints.go` |

---

## 4. Chi Tiết: `kgs-platform/internal/` Subdirs → Destination

| Directory | Size | → Destination |
|-----------|------|--------------|
| `internal/search/` | — | **search-service** `usecase/kgs/hybrid_search.go` |
| `internal/overlay/` | — | **kg-service** `usecase/kgs/overlay.go` |
| `internal/batch/` | — | **kg-service** `infra/outbox/batch.go` |
| `internal/outbox/` | — | **kg-service** `infra/outbox/worker.go` |
| `internal/projection/` | — | **search-service** `usecase/kgs/view.go` |
| `internal/analytics/` | — | **search-service** `usecase/kgs/analytics.go` |
| `internal/lock/` | — | **kg-service** `infra/redis/lock.go` |
| `internal/observability/` | — | ALL services (pkg) |
| `internal/version/` | — | ALL services |

---

## 5. Feature Preservation — Với Services Đã Có

| KGS Feature | Phương Án Cũ | Phương Án Mới (Revised) |
|-------------|-------------|-------------------------|
| Node CRUD | graph-service (mới) | **kg-service** `/v1/graph/nodes` |
| Edge CRUD | graph-service (mới) | **kg-service** `/v1/graph/edges` |
| Overlay sessions | overlay-service (mới) | **kg-service** `/v1/overlay` |
| Outbox sync (PG→Neo4j→Qdrant) | sync-worker (mới) | **kg-service** background worker |
| Schema validation | ontology-service (mới) | **kg-service** `/v1/ontology` |
| Neo4j constraints | ontology-service (mới) | **kg-service** (background on startup) |
| Hybrid search (vector+text) | search-service (mới) | **search-service** `/v1/kgs/search` |
| Context/Impact/Coverage | query-intel (mới) | **search-service** `/v1/graph/nodes/*/context` |
| Analytics & Views | query-intel (mới) | **search-service** `/v1/analytics` + `/v1/views` |
| Business rules (cron+event) | rule-engine (mới) | **pipeline-service** `/v1/rules` |
| Policy management (OPA) | policy-service (mới) | **pipeline-service** `/v1/policies` |
| App + API Key management | registry-service (mới) | **registry-service** (mới — không tránh được) |
| JWT + API Key auth | kgs-gateway | **kgs-gateway** (upgrade) |
| Episode ingestion | kg-service (existing) | **kg-service** (giữ nguyên) |
| Cognee datasets | kg-service (existing) | **kg-service** (giữ nguyên) |
| Cross-engine search | search-service (existing) | **search-service** (giữ nguyên) |
| MCP tools | search-service (existing) | **search-service** (giữ nguyên) |
| Knowledge pipeline | pipeline-service (existing) | **pipeline-service** (giữ nguyên) |

---

## 6. Code Không Cần Di Chuyển (Giữ Trong kgs-platform)

`kgs-platform` hiện tại vẫn chạy song song trong giai đoạn migration. Sau khi migrate xong các features, kgs-platform có thể được deprecated.

| Code Giữ Lại | Lý Do |
|-------------|-------|
| `kgs-platform/internal/biz/greeter.go` | Legacy, bỏ |
| `kgs-platform/internal/data/greeter.go` | Legacy, bỏ |
| `kgs-platform/internal/data/surrealdb/` | Experimental, bỏ |
| `kgs-platform/internal/data/shadow/` | Không dùng |

---

## 7. Shared Utilities — Đưa Vào `pkg/`

Code dùng chung bởi nhiều services nên đặt vào `pkg/`:

```
pkg/
├── forward/        ← Đã có — HTTP adapter pattern (dùng bởi kg-service, search-service)
├── telemetry/      ← Đã có — OpenTelemetry
├── tenant/         ← Đã có — Tenant context extraction
├── vectorstore/    ← Đã có — Embedding providers (thêm vào search-service + kg-service)
└── kgs/            ← MỚI: KGS shared constants
    ├── events.go   ← NATS topic constants (từ data/nats_topics.go)
    ├── namespace.go← Namespace computation (từ biz/namespace.go)
    └── errors.go   ← KGS error codes (từ biz/errors.go)
```
