# Zep Context Engine — Enterprise Golang Architecture

> **Version**: 1.0 | **Date**: 2026-05-09 | **Status**: Approved  
> **Scope**: Zep Context Engineering Platform — Gateway + Services  
> **Stack**: Go 1.23+ · gRPC · NATS JetStream · Neo4j · PostgreSQL · Redis

---

## 1. Executive Summary

Chuyển đổi Zep từ monolith Go (legacy CE) thành kiến trúc **Gateway + 6 Services** enterprise-grade, tuân theo **4-layer Clean Architecture** chuẩn hóa. Zep cung cấp **context engineering cho AI agents** với sub-200ms latency, powered by **Graphiti** temporal knowledge graph.

### Design Principles

| # | Principle | Rationale |
|---|-----------|-----------|
| 1 | **Gateway + Services** | Tách biệt API routing khỏi business logic, scale independently |
| 2 | **Clean Architecture per Service** | 4 layers: domain → usecase(+port) → adapter → infra |
| 3 | **gRPC internal, REST external** | Type-safe inter-service; developer-friendly external API |
| 4 | **NATS JetStream async** | Decouple graph extraction (LLM-heavy, 10-20s) from API (sub-200ms) |
| 5 | **Multi-Tenant by Design** | `project_uuid` on all entities; advisory locks for concurrency |
| 6 | **Shared `pkg/`** | Reuse types, interfaces, middleware, adapters — NO business logic |
| 7 | **Temporal KG First** | Facts with `valid_at`/`invalid_at` for evolving context reasoning |

---

## 2. System Context

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          External Consumers                               │
│  Python SDK · TypeScript SDK · Go SDK · REST API · MCP Clients            │
│  AutoGen · CrewAI · Google ADK · LiveKit · AI Agents                      │
└───────────────────────────┬──────────────────────────────────────────────┘
                            │ REST / gRPC-Web / MCP(SSE+stdio) / WebSocket
                            ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                    ZEP API GATEWAY  (zep-gateway)                          │
│  Auth(JWT/APIKey/SharedSecret) · RateLimit · CORS · Protocol Translation  │
│  Circuit Breaker · Request Validation · Tenant Resolution · MCP Server    │
│  Request Size Limit (5MB) · Timeout (30s) · Request ID · OTel Tracing     │
└──┬──────┬──────┬──────┬──────┬──────┬────────────────────────────────────┘
   │      │      │      │      │      │
   ▼      ▼      ▼      ▼      ▼      ▼
┌──────┐┌──────┐┌──────┐┌──────┐┌──────┐┌──────┐
│ User ││Thread││Memory││Graph ││Search││Admin │
│ Svc  ││ Svc  ││ Svc  ││ Svc  ││ Svc  ││ Svc  │
└──┬───┘└──┬───┘└──┬───┘└──┬───┘└──┬───┘└──┬───┘
   └───────┴───────┴───────┴───────┴───────┘
                        │
        ┌───────────────▼───────────────────┐
        │     SHARED INFRASTRUCTURE          │
        │  PostgreSQL(pgvector) · Neo4j 5.x  │
        │  Redis 7+ · NATS JetStream         │
        │  Graphiti Service · Bifrost(LLM)    │
        │  OTel Collector                     │
        └───────────────────────────────────┘
```

---

## 3. Service Inventory

### 3.1 Complete Service Map

| # | Service | gRPC Port | Health Port | Origin Layer | Responsibility |
|---|---------|----------|-------------|--------------|----------------|
| 0 | `zep-gateway` | 8081 | 8083 | L1+L2 | REST→gRPC translation, Auth, MCP Server, Rate Limiting |
| 1 | `zep-user` | 9041 | 9141 | L3 (User DAO) | User CRUD, metadata management, project isolation |
| 2 | `zep-thread` | 9042 | 9142 | L3 (Session DAO) | Thread lifecycle, session state, ended_at management |
| 3 | `zep-memory` | 9043 | 9143 | L3 (Memory DAO) | Message ingestion, context assembly, memory overlay |
| 4 | `zep-graph` | 9044 | 9144 | L4 (Graphiti) | KG extraction, fact management, ontology, temporal reasoning |
| 5 | `zep-search` | 9045 | 9145 | L4 (Search) | Semantic search, reranking (5 strategies), graph traversal |
| 6 | `zep-admin` | 9046 | 9146 | Cross-cutting | Health aggregation, project/tenant management, API keys |

### 3.2 Service-to-Layer Mapping (from Zep Functional Layers)

| Zep Layer | Go Service(s) | Key Responsibility |
|-----------|--------------|-------------------|
| L1 — Client Access | `zep-gateway` (MCP Server) | SDK endpoint, MCP 13 tools |
| L2 — API & Routing | `zep-gateway` | chi Router, Middleware (10 layers) |
| L3 — Business Logic | `zep-user`, `zep-thread`, `zep-memory` | DAO orchestration, concurrency control |
| L4 — Graph Intelligence | `zep-graph`, `zep-search` | Graphiti client, ontology, reranking |
| L5 — Data Access | All services (internal) | PostgreSQL stores, bun ORM, advisory locks |
| L6 — External Services | Shared infrastructure | PostgreSQL, Neo4j, Graphiti, LLM |

---

## 4. Technology Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Language** | Go 1.23+ | All services |
| **External API** | chi/v5 + OpenAPI 3.1 | Gateway REST |
| **Internal RPC** | gRPC + Protobuf v3 | Inter-service communication |
| **Async** | NATS JetStream | Graph extraction pipeline (async 10-20s) |
| **Relational DB** | PostgreSQL 16 + pgvector | Users, Sessions, Messages, Metadata |
| **Graph DB** | Neo4j 5.22+ | Temporal Knowledge Graph (nodes, edges, episodes) |
| **Cache** | Redis 7+ | Session cache, rate limit state, fact cache |
| **ORM** | uptrace/bun | PostgreSQL struct-based query builder |
| **LLM Gateway** | Bifrost / Graphiti Service | Entity extraction, relationship mapping |
| **MCP SDK** | modelcontextprotocol/go-sdk | MCP Server (13 read-only tools) |
| **DI** | Google Wire | All services |
| **Observability** | OTel + Prometheus + Jaeger + slog | Distributed tracing, metrics, logging |
| **Config** | Viper + ENV + YAML | All services |
| **Validation** | go-playground/validator/v10 | Request validation |

---

## 5. Monorepo Structure

```
zep-platform/
├── api/proto/                          # ALL Protobuf definitions
│   ├── common/v1/                      #   Shared: pagination, temporal, errors, health
│   │   ├── pagination.proto
│   │   ├── temporal.proto              #   valid_at, invalid_at, expired_at
│   │   ├── errors.proto
│   │   └── health.proto
│   ├── gateway/v1/                     #   Gateway-specific
│   │   └── gateway.proto
│   ├── user/v1/                        #   User service
│   │   └── user.proto
│   ├── thread/v1/                      #   Thread/Session service
│   │   └── thread.proto
│   ├── memory/v1/                      #   Memory service
│   │   └── memory.proto
│   ├── graph/v1/                       #   Graph Intelligence service
│   │   ├── graph.proto
│   │   ├── fact.proto
│   │   └── ontology.proto
│   ├── search/v1/                      #   Search service
│   │   └── search.proto
│   └── admin/v1/                       #   Admin service
│       └── admin.proto
│
├── services/                           # Service binaries
│   ├── zep-gateway/                    #   API Gateway + MCP Server
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── adapter/
│   │   │   └── infra/
│   │   ├── Dockerfile
│   │   └── README.md
│   ├── zep-user/                       #   User CRUD service
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── domain/                #     User, UserID, ProjectUUID
│   │   │   ├── usecase/              #     CreateUser, UpdateUser, DeleteUser
│   │   │   ├── adapter/              #     gRPC handler, PostgreSQL repo
│   │   │   └── infra/
│   │   └── Dockerfile
│   ├── zep-thread/                     #   Thread/Session lifecycle
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── domain/                #     Session, SessionID, EndedAt
│   │   │   ├── usecase/              #     CreateThread, EndThread, ListThreads
│   │   │   ├── adapter/
│   │   │   └── infra/
│   │   └── Dockerfile
│   ├── zep-memory/                     #   Memory ingestion + retrieval
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── domain/                #     Memory, Message, Fact overlay
│   │   │   ├── usecase/              #     PutMemory, GetMemory, DeleteMemory
│   │   │   ├── adapter/
│   │   │   └── infra/
│   │   └── Dockerfile
│   ├── zep-graph/                      #   Graph Intelligence (Graphiti)
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── domain/                #     Node, Edge, Episode, Fact, Ontology
│   │   │   ├── usecase/              #     ExtractEntities, ManageFacts
│   │   │   ├── adapter/
│   │   │   └── infra/
│   │   └── Dockerfile
│   ├── zep-search/                     #   Semantic Search + Reranking
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── domain/                #     SearchQuery, SearchResult, Reranker
│   │   │   ├── usecase/              #     GraphSearch, SessionSearch
│   │   │   ├── adapter/
│   │   │   └── infra/
│   │   └── Dockerfile
│   └── zep-admin/                      #   Admin + Health Aggregation
│       ├── cmd/server/main.go
│       ├── internal/
│       │   ├── domain/
│       │   ├── usecase/
│       │   ├── adapter/
│       │   └── infra/
│       └── Dockerfile
│
├── pkg/                                # Shared packages (NO business logic)
│   ├── graph/                          #   Graph domain types
│   │   ├── node.go                     #     EntityNode (User, Preference, Organization...)
│   │   ├── edge.go                     #     Fact (temporal edge with valid_at/invalid_at)
│   │   ├── episode.go                  #     Episode (temporal event)
│   │   ├── temporal.go                 #     TemporalAnnotation (valid_at, invalid_at, expired_at)
│   │   └── ontology.go                #     Node priority hierarchy, edge type map
│   ├── adapters/                       #   Infrastructure adapter interfaces
│   │   ├── graphdb/                    #     GraphDB interface + Neo4j driver
│   │   ├── reldb/                      #     RelationalDB interface + PostgreSQL(bun)
│   │   ├── llm/                        #     LLMClient interface + Bifrost/Graphiti
│   │   ├── cache/                      #     CacheStore interface + Redis
│   │   └── graphiti/                   #     GraphitiClient interface (HTTP client)
│   ├── middleware/                      #   Shared gRPC/HTTP interceptors
│   │   ├── auth/                       #     JWT/APIKey/SharedSecret extraction
│   │   ├── logging/                    #     Structured access logging (slog)
│   │   ├── tracing/                    #     OTel trace propagation (otelchi)
│   │   ├── recovery/                   #     Panic recovery
│   │   ├── ratelimit/                  #     Redis sliding window
│   │   └── validation/                 #     go-playground/validator + custom validators
│   ├── resilience/                     #   Circuit breaker, retry, bulkhead, timeout
│   ├── observability/                  #   Tracer, metrics, logger, health helpers
│   ├── config/                         #   Viper loader + validator
│   ├── errors/                         #   Domain error types → gRPC status mapping
│   ├── nats/                           #   NATS client helpers (publisher, subscriber)
│   ├── auth/                           #   JWT provider, API key validator, RBAC
│   ├── tenant/                         #   Project/tenant context extraction
│   ├── pagination/                     #   Cursor/offset pagination
│   ├── metadata/                       #   JSONB merge utilities, advisory lock helpers
│   └── testutil/                       #   Fixtures, mocks, testcontainers
│
├── migrations/                         #   SQL + Cypher migration files
│   ├── postgres/                       #     PostgreSQL schema migrations
│   │   ├── 001_create_users.up.sql
│   │   ├── 002_create_sessions.up.sql
│   │   └── 003_create_messages.up.sql
│   └── neo4j/                          #     Neo4j constraint/index migrations
│
├── deploy/
│   ├── docker-compose/                 #   Dev environment
│   │   ├── docker-compose.yml
│   │   └── docker-compose.infra.yml
│   └── kubernetes/                     #   Kustomize base + overlays
│       ├── base/
│       └── overlays/
│           ├── dev/
│           ├── staging/
│           └── production/
│
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
│   │   │   ├── postgres/              #     PostgreSQL repository (bun ORM)
│   │   │   ├── neo4j/                 #     Neo4j repository
│   │   │   └── redis/                 #     Redis cache
│   │   ├── client/                     #   External service gRPC/HTTP clients
│   │   │   └── <service>_client.go
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

Memory Ingestion Flow (PutMemory):
  zep-gateway → zep-memory → zep-thread (upsert session)
                            → zep-graph (async via NATS)

Memory Retrieval Flow (GetMemory):
  zep-gateway → zep-memory → zep-thread (get session)
                            → zep-search (get relevant facts)

User Context Flow:
  zep-gateway → zep-user (get user)
              → zep-search (search user's graph)

Admin Health:
  zep-admin → all services (health checks)
```

### 7.2 Async Events (NATS JetStream)

| Stream | Subject | Publisher | Subscriber |
|--------|---------|-----------|------------|
| `zep` | `zep.memory.messages.ingested` | zep-memory | zep-graph |
| `zep` | `zep.graph.extraction.completed` | zep-graph | zep-search |
| `zep` | `zep.graph.fact.created` | zep-graph | zep-search |
| `zep` | `zep.graph.fact.invalidated` | zep-graph | zep-search |
| `zep` | `zep.thread.session.ended` | zep-thread | zep-memory |
| `zep` | `zep.user.deleted` | zep-user | zep-thread, zep-graph |
| `zep` | `zep.admin.project.created` | zep-admin | All (init schema) |
| `zep` | `zep.admin.project.deleted` | zep-admin | All (cascade delete) |

### 7.3 Critical Data Flow: Message Ingestion → Graph Extraction

```
Client POST /api/v2/sessions/{id}/memory
  │
  ▼ (synchronous — sub-200ms)
┌─────────────────────────────────────────────┐
│ zep-gateway                                  │
│  ├── Auth middleware (JWT/APIKey)            │
│  ├── Tenant resolution (project_uuid)       │
│  ├── Request validation (5MB limit)         │
│  └── Forward to zep-memory (gRPC)           │
└───────────────────┬─────────────────────────┘
                    ▼
┌─────────────────────────────────────────────┐
│ zep-memory (PutMemory)                       │
│  ├── 1. gRPC → zep-thread: UpsertSession    │
│  ├── 2. Check session.EndedAt               │
│  ├── 3. INSERT messages → PostgreSQL        │
│  └── 4. NATS Publish: zep.memory.messages   │
│         .ingested                            │
└───────────────────┬─────────────────────────┘
                    │ (async — 10-20s)
                    ▼
┌─────────────────────────────────────────────┐
│ zep-graph (NATS subscriber)                  │
│  ├── 1. Graphiti PutMemory(sessionID, msgs) │
│  ├── 2. Graphiti PutMemory(userID, msgs)    │
│  ├── 3. LLM entity extraction              │
│  ├── 4. Temporal annotation (valid_at/...)  │
│  ├── 5. Neo4j upsert (nodes, edges)        │
│  └── 6. NATS Publish: zep.graph.extraction  │
│         .completed                           │
└─────────────────────────────────────────────┘
```

---

## 8. Port Assignment

| Service | gRPC | Health/Metrics | REST |
|---------|------|---------------|------|
| zep-gateway | 8081 | 8083 | 8080 |
| zep-user | 9041 | 9141 | — |
| zep-thread | 9042 | 9142 | — |
| zep-memory | 9043 | 9143 | — |
| zep-graph | 9044 | 9144 | — |
| zep-search | 9045 | 9145 | — |
| zep-admin | 9046 | 9146 | — |

---

## 9. Cross-Cutting Concerns

| Concern | Package | Notes |
|---------|---------|-------|
| Auth (JWT + APIKey + SharedSecret) | `pkg/middleware/auth/` + `pkg/auth/` | Gateway validates; propagates via gRPC metadata |
| Multi-Tenancy | `pkg/tenant/` | `project_uuid` on all entities; schema-based isolation |
| Rate Limiting | `pkg/middleware/ratelimit/` | Redis sliding window, per-tenant per-endpoint |
| Circuit Breaker | `pkg/resilience/` | sony/gobreaker, per-downstream-service |
| Retry | `pkg/resilience/` | Exponential backoff 200ms→30s, max 15 retries |
| Advisory Locks | `pkg/metadata/` | PostgreSQL `pg_advisory_lock(SHA-256 hash)` |
| Concurrency Control | `pkg/metadata/` | JSONB merge-patch with lock acquisition |
| Observability | `pkg/observability/` | OTel traces + Prometheus metrics + slog JSON logs |
| Health | `pkg/observability/health.go` | gRPC Health v1 + HTTP /healthz /readyz /livez |
| Error Mapping | `pkg/errors/` | Domain errors → gRPC status → HTTP status |
| Soft Deletes | All services | `deleted_at` timestamp on all entities |
| Request Limiting | Gateway middleware | 5MB payload, 30s timeout |

---

## 10. Document Index

| Document | Description |
|----------|-------------|
| [01-gateway.md](./01-gateway.md) | API Gateway + MCP Server (13 tools) |
| [02-user-service.md](./02-user-service.md) | User CRUD, metadata, project isolation |
| [03-thread-service.md](./03-thread-service.md) | Thread/Session lifecycle, ended_at management |
| [04-memory-service.md](./04-memory-service.md) | Memory ingestion (PutMemory) + retrieval (GetMemory) |
| [05-graph-service.md](./05-graph-service.md) | Graph Intelligence — Graphiti integration, ontology |
| [06-search-service.md](./06-search-service.md) | Semantic search, 5 reranking strategies |
| [07-admin-service.md](./07-admin-service.md) | Admin, health aggregation, project management |
| [08-shared-packages.md](./08-shared-packages.md) | Shared `pkg/` — adapters, middleware, resilience |
| [09-data-models.md](./09-data-models.md) | Domain models + Protobuf definitions |
| [10-deployment.md](./10-deployment.md) | Docker Compose + Kubernetes |
