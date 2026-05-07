# KGS Platform — Service Inventory

> **Last updated:** 2026-04-08  
> **Module:** `kgs-platform` (Go / Kratos v2)  
> **Servers:** HTTP `:8000` · gRPC `:9000`

---

## 1. Overview

`kgs-platform` là **Knowledge Graph Service** — lõi lưu trữ và truy vấn đồ thị tri thức (Knowledge Graph) của VNP Design Platform. Service được xây dựng theo kiến trúc **Kratos** (BFF + Clean Architecture) gồm ba tầng chính: `internal/service` → `internal/biz` → `internal/data`.

```
                        ┌──────────────────────────────────────┐
                        │          kgs-platform                │
  Clients               │  ┌─────────┐   ┌──────────────────┐ │
  ──────────────────────┼─▶│  HTTP   │   │      gRPC        │ │
  (REST / Gateway)      │  │  :8000  │   │      :9000       │ │
                        │  └────┬────┘   └────────┬─────────┘ │
                        │       └────────┬─────────┘           │
                        │          ┌─────▼──────┐              │
                        │          │  Services  │              │
                        │          └─────┬──────┘              │
                        │          ┌─────▼──────┐              │
                        │          │    Biz     │              │
                        │          └─────┬──────┘              │
                        │          ┌─────▼──────┐              │
                        │          │    Data    │              │
                        │          └────────────┘              │
                        │  ┌──────────────────────────────┐    │
                        │  │       WorkerServer           │    │
                        │  │  RuleRunner │ EventRunner    │    │
                        │  │  PolicySync │ KafkaConsumer  │    │
                        │  └──────────────────────────────┘    │
                        └──────────────────────────────────────┘
```

---

## 2. Internal Services (API Layer)

Các API được expose qua cả HTTP và gRPC (protobuf), đăng ký trong `internal/server/grpc.go` và `internal/server/http.go`.

| Service | Proto Package | Vai trò |
|---|---|---|
| **GreeterService** | `api/helloworld/v1` | Health-check / ping endpoint |
| **RegistryService** | `api/registry/v1` | Quản lý App, API Key, Quota |
| **OntologyService** | `api/ontology/v1` | CRUD EntityType & RelationType (schema đồ thị) |
| **GraphService** | `api/graph/v1` | CRUD Node/Edge, query Context/Impact/Coverage/Subgraph |
| **RulesService** | `api/rules/v1` | CRUD Business Rules (SCHEDULED + ON_WRITE) |
| **PolicyService** | `api/accesscontrol/v1` | CRUD OPA Rego Policies |

### 2.1 GraphService — Endpoints chi tiết

| Method | Mô tả |
|---|---|
| `CreateNode` | Tạo node mới trong Neo4j (sau khi kiểm tra OPA) |
| `GetNode` | Lấy thông tin node theo ID |
| `UpdateNode` | Cập nhật properties của node |
| `DeleteNode` | Xoá node |
| `CreateEdge` | Tạo relationship giữa hai node |
| `GetContext` | Lấy neighbors (1 hop) của node, hỗ trợ direction INCOMING/OUTGOING/BOTH |
| `GetImpact` | Traversal downstream tới maxDepth |
| `GetCoverage` | Traversal upstream tới maxDepth |
| `GetSubgraph` | Fetch subgraph từ danh sách nodeIDs |

---

## 3. Internal Components (Biz Layer)

Các components chạy trong `internal/biz`, được wire qua Google Wire.

| Component | Struct | Mô tả |
|---|---|---|
| **GraphUsecase** | `biz.GraphUsecase` | Business logic CRUD node/edge; kiểm tra OPA trước ghi; publish event sang Redis Stream `kgs:events:nodes` |
| **OntologyUsecase** | `biz.OntologyService` | Quản lý EntityType & RelationType |
| **RegistryUsecase** | — (data layer direct) | Quản lý App, APIKey, Quota, AuditLog |
| **RulesUsecase** | `biz.RulesUsecase` | CRUD Rule objects (trigger: SCHEDULED, ON_WRITE) |
| **PolicyUsecase** | `biz.PolicyUsecase` | CRUD Policy objects (Rego content) |
| **OPAClient** | `biz.OPAClient` | HTTP client gọi sang OPA sidecar `/v1/data/kgs/allow` |
| **QueryPlanner** | `biz.QueryPlanner` | Tạo Cypher queries an toàn (scoped theo `app_id`) |
| **ViewResolver** | `biz.ViewResolver` | Xử lý view/projection của đồ thị |
| **OntologySyncManager** | `biz.OntologySyncManager` | Background sync constraints từ EntityType sang Neo4j |

---

## 4. Internal Workers (WorkerServer)

`WorkerServer` implement `transport.Server` của Kratos, chạy song song với HTTP/gRPC.

| Worker | Struct | Trigger | Mô tả |
|---|---|---|---|
| **RuleRunner** | `biz.RuleRunner` | Cron schedule (gocron) | Đọc Rule có `TriggerType=SCHEDULED` từ PostgreSQL → thực thi Cypher query trên Neo4j theo cron |
| **EventRunner** | `biz.EventRunner` | Redis Stream `kgs:events:nodes` | Consumer group `kgs-worker-group` — phản ứng với sự kiện node write, thực thi Rule có `TriggerType=ON_WRITE` |
| **PolicySyncRunner** | `biz.PolicySyncRunner` | Timer (30s interval) | Pull Policy records từ PostgreSQL → upload Rego lên OPA `/v1/policies/{id}` |
| **KafkaConsumer** | `kafka.Consumer` | Kafka topic `document.ingested` | Consumer group `knowledge-service` — lắng nghe document-ingested events từ `ai-orchestrator`, auto-create Node & Edge trên đồ thị |

---

## 5. Internal Middleware

| Middleware | File | Mô tả |
|---|---|---|
| **Auth** | `internal/server/middleware/` | Kiểm tra API Key / JWT trước khi vào business layer |
| **RateLimiter** | `internal/server/middleware/` | Giới hạn request theo Quota |
| **Recovery** | Kratos built-in | Panic recovery cho tất cả requests |

---

## 6. External Dependencies (Infrastructure)

### 6.1 Data Stores

| Service | Type | Địa chỉ mặc định | Vai trò trong kgs-platform |
|---|---|---|---|
| **PostgreSQL** | Relational DB | `localhost:5432` / db `kgs_platform` | State chính: App, APIKey, Quota, AuditLog, EntityType, RelationType, Rule, RuleExecution, Policy |
| **Neo4j** | Graph DB | `bolt://localhost:7687` / db `kgs` | Lưu trữ đồ thị tri thức (Node + Edge) với app_id namespacing |
| **Redis** | In-memory cache + Stream | `localhost:6379` | Cache; Redis Stream `kgs:events:nodes` cho event-driven rule execution |
| **Qdrant** | Vector DB | `http://localhost:6333` | Semantic search trên knowledge chunks (collection `knowledge_chunks`) |

### 6.2 Message Queue

| Service | Type | Địa chỉ mặc định | Topic / Group | Vai trò |
|---|---|---|---|---|
| **Apache Kafka** | Message Broker | `localhost:9092` | Topic: `document.ingested` · Group: `knowledge-service` | Nhận document-ingested events từ upstream services (ai-orchestrator, pipeline-shim) để tự động populate đồ thị |

### 6.3 Policy Engine

| Service | Type | Địa chỉ mặc định | Endpoint | Vai trò |
|---|---|---|---|---|
| **Open Policy Agent (OPA)** | Sidecar / Policy Engine | `http://localhost:8181` | `POST /v1/data/kgs/allow` · `PUT /v1/policies/{id}` | Đánh giá access control (cho phép/từ chối) trước mọi thao tác ghi vào Neo4j; nhận Rego policies từ PolicySyncRunner |

---

## 7. Upstream Producers (Services gọi vào kgs-platform)

| Service | Protocol | Mô tả |
|---|---|---|
| **pipeline-shim** | gRPC / HTTP | Gọi GraphService để query context/impact cho LLM pipeline |
| **ai-orchestrator** | Kafka (`document.ingested`) | Publish document-ingested events sau khi parse PRD/SRS/UI → kgs-platform tự động tạo node/edge |
| **service-kg-to-preview** | gRPC / HTTP | Sử dụng kgs-platform làm primary graph backend để lấy subgraph cho render UI preview |
| **vnp-design-platform (frontend / API Gateway)** | HTTP/gRPC | Gọi RegistryService để quản lý App/APIKey; gọi GraphService để truy vấn trực tiếp |

---

## 8. Data Models (PostgreSQL)

| Model | Bảng (auto-migrated) | Mô tả |
|---|---|---|
| `App` | `apps` | Client application đăng ký vào KGS |
| `APIKey` | `api_keys` | Khóa xác thực cho App (lưu hash SHA-256) |
| `Quota` | `quotas` | Rate limit & resource limits per App |
| `AuditLog` | `audit_logs` | Lịch sử hành động quản trị |
| `EntityType` | `entity_types` | Schema node label (JSON Schema) |
| `RelationType` | `relation_types` | Schema edge type (JSON Schema) |
| `Rule` | `rules` | Business rule (Cypher + cron/trigger) |
| `RuleExecution` | `rule_executions` | Lịch sử thực thi rule |
| `Policy` | `policies` | OPA Rego policy content |

---

## 9. Redis Streams & Keys

| Stream / Key | Hướng | Mô tả |
|---|---|---|
| `kgs:events:nodes` (Stream) | Produced by GraphUsecase | Publish events `node.created`, `node.updated`, `node.deleted` |
| `kgs:events:nodes` (Consumer Group: `kgs-worker-group`) | Consumed by EventRunner | Trigger ON_WRITE business rules |

---

## 10. Dependency Summary

```
kgs-platform
├── INTERNAL SERVICES (API)
│   ├── GreeterService       (HTTP+gRPC)
│   ├── RegistryService      (HTTP+gRPC)
│   ├── OntologyService      (HTTP+gRPC)
│   ├── GraphService         (HTTP+gRPC)
│   ├── RulesService         (HTTP+gRPC)
│   └── PolicyService        (HTTP+gRPC)
├── INTERNAL WORKERS
│   ├── RuleRunner           (gocron scheduler)
│   ├── EventRunner          (Redis Stream consumer)
│   ├── PolicySyncRunner     (30s ticker → OPA)
│   └── KafkaConsumer        (kafka-go, topic: document.ingested)
└── EXTERNAL DEPENDENCIES
    ├── PostgreSQL            (primary state store)
    ├── Neo4j                 (graph store)
    ├── Redis                 (cache + event stream)
    ├── Qdrant                (vector search)
    ├── Apache Kafka          (event bus, consumer)
    └── Open Policy Agent     (sidecar, access control)
```
