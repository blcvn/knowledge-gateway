# VNP Cognitive Platform — Unified Golang Architecture

> **Version**: 2.0 | **Date**: 2026-05-09 | **Status**: Approved  
> **Scope**: Cognee + Graphiti unified monorepo  
> **Stack**: Go 1.23+ · gRPC · NATS JetStream · Neo4j · Redis · PostgreSQL

---

## 1. Executive Summary

Hợp nhất **Cognee** (semantic KG, RAG, session memory) và **Graphiti** (temporal episodic KG) vào **single Go monorepo** với shared infrastructure, shared `pkg/`, và **unified API Gateway**. Mỗi service tuân theo **4-layer Clean Architecture** chuẩn hóa.

### Design Principles

| # | Principle | Rationale |
|---|-----------|-----------|
| 1 | **Single Monorepo** | Shared proto, shared `pkg/`, single `go.mod`, unified CI |
| 2 | **Unified Gateway** | One entry point for all Cognee + Graphiti APIs |
| 3 | **Clean Architecture per Service** | 4 layers: domain → usecase(+port) → adapter → infra |
| 4 | **gRPC internal, REST external** | Type-safe inter-service, developer-friendly external |
| 5 | **NATS JetStream async** | Pipeline orchestration, event-driven coupling |
| 6 | **Shared `pkg/` — NO business logic** | Only types, interfaces, middleware, adapters |
| 7 | **Multi-Tenant by Design** | Cognee: tenant_id + RLS. Graphiti: group_id partition |

---

## 2. System Context

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          External Consumers                               │
│  Web UI · CLI · SDK(Go/Py/JS) · MCP Clients · AI Agents · Chat Apps     │
└───────────────────────────┬──────────────────────────────────────────────┘
                            │ REST / gRPC-Web / MCP(SSE) / WebSocket
                            ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                    UNIFIED API GATEWAY  (vnp-gateway)                      │
│  Auth(JWT/APIKey) · RateLimit · CORS · Protocol Translation · Routing    │
│  Circuit Breaker · Request Validation · Tenant Resolution · MCP Server   │
└──┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬───────┘
   │      │      │      │      │      │      │      │      │      │
   ▼      ▼      ▼      ▼      ▼      ▼      ▼      ▼      ▼      ▼
┌───── COGNEE DOMAIN ─────┐  ┌─────── GRAPHITI DOMAIN ───────┐  ┌───────┐
│Ingest│Cognify│Search│Mem │  │Ingest│Search│Knowledge│ Store │  │ Admin │
│ ion  │  ion  │      │ory │  │ ion  │      │Process. │       │  │(shared│
│ Svc  │  Svc  │ Svc  │Svc │  │ Svc  │ Svc  │  Svc    │  Svc  │  │  )    │
└──┬───┘───┬───┘──┬───┘─┬──┘  └──┬───┘──┬───┘───┬─────┘──┬───┘  └──┬────┘
   └───────┴──────┴─────┴────────┴──────┴───────┴────────┴─────────┘
                                    │
                    ┌───────────────▼───────────────────┐
                    │     SHARED INFRASTRUCTURE          │
                    │  PostgreSQL · Neo4j · Qdrant       │
                    │  Redis · NATS · MinIO/S3           │
                    │  Bifrost(LLM) · OTel Collector     │
                    └───────────────────────────────────┘
```

---

## 3. Service Inventory

### 3.1 Complete Service Map

| # | Service | gRPC Port | Domain | Origin |
|---|---------|----------|--------|--------|
| 0 | `vnp-gateway` | 8080(HTTP) 8081(gRPC) 8082(MCP) | Routing, Auth | Both |
| 1 | `cognee-ingestion` | 9011 | Data pipeline, file extract | Cognee L2-L4 |
| 2 | `cognee-cognify` | 9012 | KG build, chunking, ontology | Cognee L3-L5 |
| 3 | `cognee-search` | 9013 | 15 retriever strategies, RAG | Cognee L5 |
| 4 | `cognee-memory` | 9014 | Session, agent mem, V2 API | Cognee L2+L5 |
| 5 | `graphiti-ingestion` | 9021 | Episode pipeline, saga | Graphiti L6 |
| 6 | `graphiti-search` | 9022 | Hybrid search, reranking | Graphiti L4 |
| 7 | `graphiti-knowledge` | 9023 | LLM extraction, resolution | Graphiti L5+L3 |
| 8 | `graphiti-store` | 9024 | Graph DB abstraction, CRUD | Graphiti L2+L1 |
| 9 | `vnp-admin` | 9030 | Users, tenants, datasets, health | Shared |

### 3.2 What Is Shared vs Separate

| Component | Strategy | Rationale |
|-----------|----------|-----------|
| **Gateway** | **Unified** | Single entry point, single auth/rate-limit |
| **Admin** | **Unified** | Users, tenants, API keys, health — same for both domains |
| **`pkg/`** | **Unified** | Proto, middleware, resilience, observability, config, testutil |
| **`pkg/graph/`** | **Unified** | Shared graph types (Graphiti's bi-temporal model + Cognee's KG model) |
| **`pkg/adapters/`** | **Unified** | GraphDB, VectorDB, LLM, Storage, Embedding interfaces |
| **Cognee services** | **Separate** | Cognee-specific business logic |
| **Graphiti services** | **Separate** | Graphiti-specific business logic |
| **Infrastructure** | **Shared** | Same Neo4j, Redis, NATS, PostgreSQL instances |

---

## 4. Technology Stack (Unified)

| Layer | Technology | Used By |
|-------|-----------|---------|
| **Language** | Go 1.23+ | All |
| **External API** | chi/v5 + OpenAPI 3.1 | Gateway |
| **Internal RPC** | gRPC + Protobuf v3 | All services |
| **Async** | NATS JetStream | All services |
| **Relational DB** | PostgreSQL 16 | Admin, Cognee metadata |
| **Graph DB** | Neo4j 5.x (primary), FalkorDB/Kuzu (pluggable) | Cognee KG + Graphiti episodic graph |
| **Vector DB** | Qdrant / PGVector | Cognee search |
| **Cache** | Redis 7+ | Gateway, Search, Memory |
| **Object Storage** | MinIO / S3 | Cognee file storage |
| **LLM Gateway** | Bifrost | Cognee Cognify + Graphiti Knowledge |
| **DI** | Google Wire | All services |
| **Observability** | OTel + Prometheus + Jaeger + slog | All services |
| **Config** | Viper + ENV | All services |

---

## 5. Monorepo Structure

```
vnp-cognitive/
├── api/proto/                          # ALL Protobuf definitions
│   ├── common/v1/                      #   Shared: pagination, temporal, errors, health
│   ├── graph/v1/                       #   Shared: node.proto, edge.proto
│   ├── gateway/v1/                     #   Gateway-specific
│   ├── cognee/                         #   Cognee domain protos
│   │   ├── ingestion/v1/
│   │   ├── cognify/v1/
│   │   ├── search/v1/
│   │   └── memory/v1/
│   ├── graphiti/                       #   Graphiti domain protos
│   │   ├── ingestion/v1/
│   │   ├── search/v1/
│   │   ├── knowledge/v1/
│   │   └── store/v1/
│   └── admin/v1/                       #   Shared admin proto
│
├── services/                           # Service binaries (each has cmd/ + internal/)
│   ├── vnp-gateway/                    #   Unified API Gateway
│   ├── cognee-ingestion/               #   Cognee: data pipeline
│   ├── cognee-cognify/                 #   Cognee: KG construction
│   ├── cognee-search/                  #   Cognee: 15 retrievers
│   ├── cognee-memory/                  #   Cognee: session + agent
│   ├── graphiti-ingestion/             #   Graphiti: episode pipeline
│   ├── graphiti-search/                #   Graphiti: hybrid search
│   ├── graphiti-knowledge/             #   Graphiti: LLM processing
│   ├── graphiti-store/                 #   Graphiti: graph DB abstraction
│   └── vnp-admin/                      #   Shared: users, tenants, health
│
├── pkg/                                # Shared packages (NO business logic)
│   ├── graph/                          #   Shared graph domain types
│   │   ├── node.go                     #     EntityNode, EpisodicNode, CommunityNode, SagaNode
│   │   ├── edge.go                     #     EntityEdge, EpisodicEdge, CommunityEdge
│   │   ├── temporal.go                 #     BiTemporal model
│   │   ├── group.go                    #     Multi-tenancy primitives (group_id)
│   │   └── embedding.go               #     EmbeddingVector type
│   ├── adapters/                       #   Infrastructure adapter interfaces
│   │   ├── graphdb/                    #     GraphDB interface + Neo4j/FalkorDB/Kuzu
│   │   ├── vectordb/                   #     VectorDB interface + Qdrant/PGVector
│   │   ├── llm/                        #     LLMClient interface + Bifrost/OpenAI/Anthropic
│   │   ├── embedder/                   #     EmbedderClient interface + providers
│   │   ├── reranker/                   #     CrossEncoder interface + providers
│   │   └── storage/                    #     ObjectStorage interface + S3/MinIO/Local
│   ├── middleware/                      #   Shared gRPC/HTTP interceptors
│   │   ├── auth/                       #     JWT/APIKey extraction + tenant propagation
│   │   ├── logging/                    #     Structured access logging
│   │   ├── tracing/                    #     OTel trace propagation
│   │   ├── recovery/                   #     Panic recovery
│   │   ├── ratelimit/                  #     gRPC rate limiting
│   │   └── validation/                 #     Request validation
│   ├── resilience/                     #   Circuit breaker, retry, bulkhead, timeout
│   ├── observability/                  #   Tracer, metrics, logger, health helpers
│   ├── config/                         #   Viper loader + validator
│   ├── errors/                         #   Domain error types → gRPC status mapping
│   ├── nats/                           #   NATS client helpers (publisher, subscriber)
│   ├── auth/                           #   JWT provider, API key validator, RBAC
│   ├── tenant/                         #   Tenant context extraction/propagation
│   ├── pagination/                     #   Cursor/offset pagination
│   └── testutil/                       #   Fixtures, mocks, testcontainers
│
├── migrations/                         #   SQL + Cypher migration files
├── deploy/
│   ├── docker-compose/                 #   Dev environment
│   └── kubernetes/                     #   Kustomize base + overlays
├── go.mod                              #   Single go.mod for entire monorepo
├── buf.yaml                            #   Protobuf management
├── Makefile
└── README.md
```

---

## 6. Clean Architecture — Standardized Per Service

Mỗi service trong `services/<name>/` tuân theo cấu trúc 4-layer **chuẩn hóa hoàn toàn**:

```
services/<service-name>/
├── cmd/
│   └── server/
│       └── main.go                     # Entry point, wire injection
├── internal/
│   ├── domain/                         # Layer 1: Entities (innermost — ZERO imports)
│   │   ├── entity.go                   #   Domain models (pure Go structs)
│   │   ├── value_object.go             #   Value objects (immutable)
│   │   ├── event.go                    #   Domain events
│   │   └── errors.go                   #   Domain-specific error types
│   │
│   ├── usecase/                        # Layer 2: Use Cases (imports domain only)
│   │   ├── <usecase_name>.go           #   One file per use case
│   │   ├── port/                       #   Port interfaces (dependency inversion)
│   │   │   ├── input.go               #     Input ports (use case interfaces)
│   │   │   └── output.go             #     Output ports (repository, external service)
│   │   └── dto/                        #   Data Transfer Objects
│   │       ├── request.go
│   │       └── response.go
│   │
│   ├── adapter/                        # Layer 3: Interface Adapters
│   │   ├── grpc/                       #   gRPC handlers (controllers)
│   │   │   ├── handler.go
│   │   │   └── mapper.go             #   Proto ↔ Domain mapping
│   │   ├── repository/                 #   Output adapter implementations
│   │   │   ├── postgres/              #     PostgreSQL repository
│   │   │   ├── neo4j/                 #     Neo4j repository
│   │   │   └── redis/                 #     Redis cache
│   │   ├── client/                     #   External service gRPC clients
│   │   │   ├── <service>_client.go
│   │   │   └── ...
│   │   ├── event/                      #   NATS publisher/subscriber
│   │   │   ├── publisher.go
│   │   │   └── subscriber.go
│   │   └── mapper/                     #   Additional data mappers
│   │
│   └── infra/                          # Layer 4: Frameworks & Drivers (outermost)
│       ├── config/
│       │   └── config.go               #   Service-specific configuration
│       ├── server/
│       │   └── grpc.go                 #   gRPC server setup + graceful shutdown
│       ├── telemetry/
│       │   ├── tracer.go
│       │   ├── metrics.go
│       │   └── logger.go
│       ├── middleware/                  #   Service-specific interceptors
│       └── wire/                       #   Google Wire DI providers
│           ├── wire.go
│           └── wire_gen.go
│
├── api/proto/                          # Service-specific proto (or symlink to api/proto/)
├── Dockerfile
├── Makefile
└── README.md
```

### Dependency Rule (STRICT)

```
domain ← usecase ← adapter ← infra
 (inner)                     (outer)

✅ domain: ZERO external imports (no gRPC, no DB, no framework)
✅ usecase: imports domain only; defines port interfaces
✅ adapter: imports usecase(ports) + domain; implements interfaces
✅ infra: imports everything; wires dependencies via Wire
```

---

## 7. Inter-Service Communication

### 7.1 Synchronous (gRPC)

```
Gateway → All services (fan-out by route)
Cognee: ingestion → cognify → (search for reindex)
Graphiti: ingestion → knowledge → store; search → knowledge + store
Admin: health checks to all services
```

### 7.2 Async Events (NATS JetStream)

| Stream | Subject | Publisher | Subscriber |
|--------|---------|-----------|------------|
| `cognee` | `cognee.ingestion.data.ingested` | cognee-ingestion | cognee-cognify |
| `cognee` | `cognee.cognify.pipeline.completed` | cognee-cognify | cognee-search |
| `cognee` | `cognee.memory.session.persisted` | cognee-memory | cognee-cognify |
| `graphiti` | `graphiti.episode.ingested` | graphiti-ingestion | graphiti-search |
| `graphiti` | `graphiti.entity.resolved` | graphiti-knowledge | graphiti-search |
| `graphiti` | `graphiti.community.rebuilt` | graphiti-knowledge | graphiti-search |
| `admin` | `admin.tenant.created` | vnp-admin | All (init schema/cache) |
| `admin` | `admin.dataset.deleted` | vnp-admin | Cognee services (cascade) |
| `admin` | `admin.tenant.deleted` | vnp-admin | Graphiti (clear group) |

---

## 8. Port Assignment

| Service | gRPC | Health/Metrics | REST |
|---------|------|---------------|------|
| vnp-gateway | 8081 | 8083 | 8080 |
| cognee-ingestion | 9011 | 9091 | — |
| cognee-cognify | 9012 | 9092 | — |
| cognee-search | 9013 | 9093 | — |
| cognee-memory | 9014 | 9094 | — |
| graphiti-ingestion | 9021 | 9095 | — |
| graphiti-search | 9022 | 9096 | — |
| graphiti-knowledge | 9023 | 9097 | — |
| graphiti-store | 9024 | 9098 | — |
| vnp-admin | 9030 | 9099 | — |

---

## 9. Cross-Cutting Concerns

| Concern | Package | Notes |
|---------|---------|-------|
| Auth (JWT + API Key) | `pkg/middleware/auth/` + `pkg/auth/` | Gateway validates; propagates via gRPC metadata |
| Multi-Tenancy | `pkg/tenant/` | Cognee: `tenant_id` + PG RLS. Graphiti: `group_id` property filter |
| Rate Limiting | `pkg/middleware/ratelimit/` | Redis sliding window, per-tenant per-endpoint |
| Circuit Breaker | `pkg/resilience/` | sony/gobreaker, per-downstream-service |
| Retry | `pkg/resilience/` | Exponential backoff + jitter |
| Bulkhead | `pkg/resilience/` | Channel-based semaphore for LLM calls |
| Observability | `pkg/observability/` | OTel traces + Prometheus metrics + slog JSON logs |
| Health | `pkg/observability/health.go` | gRPC Health v1 + HTTP /healthz /readyz /livez |
| Error Mapping | `pkg/errors/` | Domain errors → gRPC status → HTTP status |

---

## 10. Document Index

| Document | Description |
|----------|-------------|
| [01-gateway.md](./01-gateway.md) | Unified API Gateway — Cognee + Graphiti routing |
| [02-cognee-ingestion.md](./02-cognee-ingestion.md) | Cognee data ingestion pipeline |
| [03-cognee-cognify.md](./03-cognee-cognify.md) | Cognee KG construction |
| [04-cognee-search.md](./04-cognee-search.md) | Cognee 15-strategy retrieval |
| [05-cognee-memory.md](./05-cognee-memory.md) | Cognee session + agent memory |
| [06-graphiti-ingestion.md](./06-graphiti-ingestion.md) | Graphiti episode pipeline |
| [07-graphiti-search.md](./07-graphiti-search.md) | Graphiti hybrid search |
| [08-graphiti-knowledge.md](./08-graphiti-knowledge.md) | Graphiti LLM processing |
| [09-graphiti-store.md](./09-graphiti-store.md) | Graphiti graph DB abstraction |
| [10-admin.md](./10-admin.md) | Shared admin service |
| [11-shared-packages.md](./11-shared-packages.md) | Shared `pkg/` packages |
| [12-data-models.md](./12-data-models.md) | Unified domain models + Protobuf |
| [13-deployment.md](./13-deployment.md) | Docker Compose + Kubernetes |
