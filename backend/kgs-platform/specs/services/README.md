# KGS Platform — Gateway + Services Architecture

> **Version:** 2.0 | **Date:** June 2026
> **Chuyển đổi từ:** Monolithic layered architecture → Gateway + Microservices

---

## Tổng Quan

KGS (Knowledge Graph Service) Platform được tái kiến trúc từ monolith 5 lớp sang mô hình **API Gateway + độc lập Services**. Mỗi service thực hiện đúng **một chức năng riêng biệt** (Single Responsibility Principle), giao tiếp qua gRPC nội bộ và expose ra ngoài thông qua API Gateway duy nhất.

---

## Sơ Đồ Kiến Trúc Tổng Thể

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Consumer Layer                                  │
│        BA Agent System · App B · App C · Platform Admin UI              │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │  HTTPS (REST / gRPC-Web)
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    kgs-gateway  (Port 8080)                              │
│  ┌─────────────┐  ┌───────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │  API Key    │  │   Rate    │  │   Request    │  │  Audit Logger  │  │
│  │  Auth       │  │  Limiter  │  │   Router     │  │                │  │
│  └─────────────┘  └───────────┘  └──────────────┘  └────────────────┘  │
└──────┬──────────┬──────────┬──────────┬──────────┬──────────┬───────────┘
       │          │          │          │          │          │
  gRPC │     gRPC │     gRPC │     gRPC │     gRPC │     gRPC │
       ▼          ▼          ▼          ▼          ▼          ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ registry │ │ ontology │ │  graph   │ │  query-  │ │  rule-   │ │ policy   │
│ -service │ │ -service │ │ -service │ │ intel-   │ │ engine-  │ │ -service │
│ :9001    │ │ :9002    │ │ :9003    │ │ service  │ │ service  │ │ :9005    │
│          │ │          │ │          │ │ :9004    │ │ :9006    │ │          │
└──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘
                                 │                       │
                          ┌──────┘──────┐         ┌──────┘─────┐
                          ▼             ▼          ▼            ▼
                    ┌──────────┐  ┌──────────┐ ┌──────────┐ ┌──────────┐
                    │  search  │  │   sync-  │ │  overlay │ │  vector  │
                    │ -service │  │  worker  │ │ -service │ │ -service │
                    │ :9007    │  │ (worker) │ │ :9008    │ │ :9009    │
                    └──────────┘  └──────────┘ └──────────┘ └──────────┘

                              ┌──────────────────────────────────────┐
                              │           Storage Layer               │
                              │  PostgreSQL · Neo4j · Qdrant          │
                              │  Redis · NATS · OPA Server            │
                              └──────────────────────────────────────┘
```

---

## Danh Sách Services

| # | Service | Port | Chức năng chính | Spec |
|---|---------|------|-----------------|------|
| 0 | **kgs-gateway** | 8080 | API Gateway: Auth, Rate Limit, Routing | [00-gateway.md](./00-gateway.md) |
| 1 | **registry-service** | 9001 | Quản lý App lifecycle, API Key, Quota | [01-registry-service.md](./01-registry-service.md) |
| 2 | **ontology-service** | 9002 | CRUD schema EntityType, RelationType | [02-ontology-service.md](./02-ontology-service.md) |
| 3 | **graph-service** | 9003 | CRUD Nodes/Edges, namespace isolation | [03-graph-service.md](./03-graph-service.md) |
| 4 | **query-intel-service** | 9004 | Query Planner, Analytics, Projection | [04-query-intelligence-service.md](./04-query-intelligence-service.md) |
| 5 | **rule-engine-service** | 9005 | CRUD Rules, Scheduled/Event-driven execution | [05-rule-engine-service.md](./05-rule-engine-service.md) |
| 6 | **policy-service** | 9006 | OPA Policy CRUD, Sync, Evaluation | [06-policy-service.md](./06-policy-service.md) |
| 7 | **sync-worker-service** | — | Outbox Worker, Batch Sync, Reconcile | [07-sync-worker-service.md](./07-sync-worker-service.md) |
| 8 | **search-service** | 9007 | Hybrid Search (vector + text + centrality) | [08-search-service.md](./08-search-service.md) |
| 9 | **overlay-service** | 9008 | Overlay Graphs, Commit/Discard, Conflict | [09-overlay-service.md](./09-overlay-service.md) |

---

## Nguyên Tắc Thiết Kế

### 1. Single Responsibility
Mỗi service chỉ làm **một việc duy nhất**. Graph Service chỉ CRUD nodes/edges. Query Intelligence Service chỉ xử lý query planning và phân tích.

### 2. Gateway là điểm vào duy nhất
Tất cả traffic từ external đều đi qua `kgs-gateway`. Gateway xử lý:
- API Key authentication → App Context (app_id, tenant_id, scopes)
- Rate limiting per app
- Request routing đến đúng service
- Audit logging

### 3. Giao tiếp nội bộ qua gRPC
Services giao tiếp với nhau qua gRPC (không qua Gateway). Ví dụ:
- `graph-service` gọi `ontology-service` để validate schema
- `graph-service` gọi `policy-service` để evaluate access
- `rule-engine-service` gọi `graph-service` để execute Cypher rules

### 4. PostgreSQL là Source-of-Truth
Mọi write operations đều ghi vào PostgreSQL trước. `sync-worker-service` async fan-out sang Neo4j và Qdrant qua Outbox pattern.

### 5. Tenant Context được truyền qua metadata
Gateway inject `x-app-id`, `x-tenant-id`, `x-scopes` vào gRPC metadata headers. Mọi service đều trust và sử dụng context này.

---

## Storage Ownership

| Storage | Owner Service | Vai trò |
|---------|--------------|---------|
| PostgreSQL `apps` table | registry-service | App lifecycle |
| PostgreSQL `api_keys` table | registry-service | API Key management |
| PostgreSQL `entity_types`, `relation_types` | ontology-service | Ontology schema |
| PostgreSQL `kg_entities`, `kg_edges`, `kg_sync_outbox` | graph-service | Graph data (source of truth) |
| PostgreSQL `rules`, `rule_executions` | rule-engine-service | Rule management |
| PostgreSQL `policies` | policy-service | OPA policies |
| PostgreSQL `view_definitions` | query-intel-service | Projection views |
| Neo4j | sync-worker-service (write) / query-intel-service, search-service (read) | Graph traversal |
| Qdrant | sync-worker-service (write) / search-service (read) | Vector search |
| Redis | overlay-service, sync-worker-service | Overlay store, cache, locks |
| NATS | overlay-service, graph-service | Event streaming |
| OPA Server | policy-service | Policy evaluation |

---

## Port Map

| Service | gRPC Port | HTTP Port (nếu có) |
|---------|-----------|-------------------|
| kgs-gateway | — | 8080 |
| registry-service | 9001 | — |
| ontology-service | 9002 | — |
| graph-service | 9003 | — |
| query-intel-service | 9004 | — |
| rule-engine-service | 9005 | — |
| policy-service | 9006 | — |
| search-service | 9007 | — |
| overlay-service | 9008 | — |
| sync-worker-service | — | — (background worker) |

---

## Inter-Service Call Graph

```
kgs-gateway
    ├──→ registry-service   (auth lookup, app context)
    ├──→ ontology-service   (schema CRUD)
    ├──→ graph-service      (node/edge CRUD)
    ├──→ query-intel-service (analytics, projection, context queries)
    ├──→ rule-engine-service (rule CRUD)
    ├──→ policy-service     (policy CRUD)
    ├──→ search-service     (hybrid search)
    └──→ overlay-service    (overlay CRUD)

graph-service
    ├──→ ontology-service   (validate schema trước khi write)
    ├──→ policy-service     (evaluate access policy)
    └──→ [NATS emit]        (trigger rule-engine, sync-worker)

rule-engine-service
    └──→ graph-service      (execute Cypher rules)

sync-worker-service
    ├── [listen NATS/Outbox from graph-service]
    ├──→ Neo4j              (sync entities/edges)
    └──→ Qdrant             (sync vectors)

policy-service
    └──→ OPA Server         (push policies, evaluate)

query-intel-service
    ├──→ Neo4j              (graph traversal queries)
    ├──→ ontology-service   (load schema for projection)
    └──→ policy-service     (check field-level access)

search-service
    ├──→ Qdrant             (vector search)
    ├──→ PostgreSQL          (full-text search)
    └──→ Neo4j              (centrality scoring)
```

---

## Deployment Model (Docker Compose / Kubernetes)

Mỗi service là một **independent binary** được build riêng:

```yaml
# docker-compose example
services:
  kgs-gateway:
    image: kgs/gateway:latest
    ports: ["8080:8080"]
    env: [REGISTRY_SERVICE_ADDR, GRAPH_SERVICE_ADDR, ...]

  registry-service:
    image: kgs/registry:latest
    ports: ["9001:9001"]
    depends_on: [postgres]

  ontology-service:
    image: kgs/ontology:latest
    ports: ["9002:9002"]
    depends_on: [postgres]

  graph-service:
    image: kgs/graph:latest
    ports: ["9003:9003"]
    depends_on: [postgres, nats]

  # ... các services khác
```

---

*Xem chi tiết từng service trong các file spec tương ứng.*
