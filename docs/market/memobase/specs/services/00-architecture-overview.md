# Memobase Enterprise — Golang Architecture Overview

> **Version**: 1.0 | **Date**: 2026-05-09 | **Status**: Proposed
> **Stack**: Go 1.23+ · gRPC · NATS JetStream · PostgreSQL (pgvector) · Redis

---

## 1. Executive Summary

Chuyển đổi Memobase từ Python/FastAPI monolith sang **Go microservices** với mô hình **Gateway + Services**, tuân theo **4-layer Clean Architecture** per service. Mục tiêu: enterprise-grade, production-ready, horizontal scalable.

### Design Principles

| # | Principle | Rationale |
|---|-----------|-----------|
| 1 | **Gateway + Services** | Single entry point REST, gRPC nội bộ, tách biệt concern |
| 2 | **Clean Architecture per Service** | 4 layers: domain → usecase(+port) → adapter → infra |
| 3 | **gRPC internal, REST external** | Type-safe inter-service, developer-friendly external |
| 4 | **NATS JetStream async** | Buffer flush pipeline, event-driven processing |
| 5 | **Multi-Tenant by Design** | `project_id` partitioning, JWT/API Key per project |
| 6 | **Cold-path LLM Processing** | Buffer batching → async pipeline, tránh hot-path LLM calls |
| 7 | **Shared `pkg/` — NO business logic** | Only types, interfaces, middleware, adapters |

---

## 2. System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                       External Consumers                        │
│  Python SDK · TS SDK · Go SDK · MCP Clients · AI Agents        │
└──────────────────────────┬──────────────────────────────────────┘
                           │ REST / MCP(SSE) / WebSocket
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                 MEMOBASE API GATEWAY (memobase-gateway)         │
│  Auth(JWT/APIKey) · RateLimit · CORS · Protocol Translation    │
│  Circuit Breaker · Request Validation · Tenant Resolution      │
│  MCP Server (save_memory, get_profiles, search_memories)       │
└──┬──────────┬──────────────┬─────────────┬─────────────┬───────┘
   │          │              │             │             │
   ▼          ▼              ▼             ▼             ▼
┌──────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐
│Ingest│ │ Memory   │ │  Context  │ │  Event   │ │  Admin   │
│ ion  │ │ Engine   │ │  Service  │ │  Service │ │  Service │
│ Svc  │ │  Svc     │ │           │ │          │ │          │
└──┬───┘ └────┬─────┘ └─────┬─────┘ └────┬─────┘ └────┬─────┘
   └──────────┴─────────────┴────────────┴────────────┘
                            │
            ┌───────────────▼───────────────────┐
            │     SHARED INFRASTRUCTURE          │
            │  PostgreSQL + pgvector · Redis     │
            │  NATS JetStream · Bifrost (LLM)   │
            │  OTel Collector · Prometheus       │
            └───────────────────────────────────┘
```

---

## 3. Service Inventory

| # | Service | gRPC Port | Health Port | Responsibility |
|---|---------|----------|-------------|----------------|
| 0 | `memobase-gateway` | 8081 | 8083 | REST API, Auth, Rate Limit, MCP Server |
| 1 | `memobase-ingestion` | 9041 | 9091 | Blob insert, Buffer Zone, Flush trigger |
| 2 | `memobase-engine` | 9042 | 9092 | Profile extraction, YOLO merge, Event summary |
| 3 | `memobase-context` | 9043 | 9093 | Context assembly, Profile query, Event search |
| 4 | `memobase-event` | 9044 | 9094 | Event CRUD, Semantic search, Gist search |
| 5 | `memobase-admin` | 9045 | 9095 | Project, User, Billing, Config management |

### Service Boundary Rationale

| Current Python Module | → Go Service | Why |
|----------------------|-------------|-----|
| `api_layer/blob.py` + `controllers/blob.py` + `controllers/buffer.py` | **memobase-ingestion** | Hot-path ingestion isolated from cold-path processing |
| `controllers/modal/chat/` + `llms/` + `prompts/` | **memobase-engine** | CPU/LLM-intensive processing, independent scaling |
| `controllers/context.py` + `controllers/profile.py` (read) | **memobase-context** | Read-heavy, low-latency serving path |
| `controllers/event.py` + `controllers/event_gist.py` | **memobase-event** | Vector search workload, independent of profile ops |
| `controllers/user.py` + `controllers/project.py` + `controllers/billing.py` | **memobase-admin** | CRUD admin ops, isolated from data pipeline |

---

## 4. Technology Stack

| Layer | Technology | Justification |
|-------|-----------|---------------|
| **Language** | Go 1.23+ | High performance, concurrency, low memory |
| **External API** | chi/v5 + OpenAPI 3.1 | Gateway REST |
| **Internal RPC** | gRPC + Protobuf v3 | Type-safe, efficient serialization |
| **Async** | NATS JetStream | Pipeline orchestration, reliable delivery |
| **Relational DB** | PostgreSQL 17 + pgvector | JSONB + vector search |
| **Cache** | Redis 7+ | Profile caching, rate limiting |
| **LLM Gateway** | Bifrost / OpenAI-compatible | Multi-provider LLM abstraction |
| **Tokenizer** | tiktoken-go | Accurate token counting |
| **DI** | Google Wire | Compile-time dependency injection |
| **Observability** | OTel + Prometheus + slog | Traces, metrics, structured logs |
| **Config** | Viper + ENV | YAML + environment variable overrides |

---

## 5. Monorepo Structure

```
services/memobase/
├── api/proto/                          # ALL Protobuf definitions
│   ├── common/v1/
│   │   ├── pagination.proto
│   │   ├── errors.proto
│   │   └── health.proto
│   ├── memobase/
│   │   ├── ingestion/v1/
│   │   │   └── ingestion.proto         # InsertBlob, FlushBuffer, GetBufferCapacity
│   │   ├── engine/v1/
│   │   │   └── engine.proto            # ProcessBlobs, ExtractProfile, MergeProfile
│   │   ├── context/v1/
│   │   │   └── context.proto           # GetContext, GetProfiles, TruncateProfiles
│   │   ├── event/v1/
│   │   │   └── event.proto             # SearchEvents, SearchGists, FilterByTags
│   │   └── admin/v1/
│   │       └── admin.proto             # UserCRUD, ProjectConfig, Billing
│
├── services/                           # Service binaries
│   ├── memobase-gateway/               # REST → gRPC, Auth, MCP
│   ├── memobase-ingestion/             # Blob insert, Buffer zone
│   ├── memobase-engine/                # Profile extraction, YOLO merge
│   ├── memobase-context/               # Context assembly, Profile read
│   ├── memobase-event/                 # Event CRUD, Vector search
│   └── memobase-admin/                 # User, Project, Billing
│
├── pkg/                                # Shared packages (NO business logic)
│   ├── blob/                           # Blob types, ChatBlob, DocBlob
│   ├── profile/                        # Profile types (topic/sub_topic/content)
│   ├── adapters/                       # Infrastructure adapter interfaces
│   │   ├── llm/                        #   LLMClient interface + Bifrost/OpenAI
│   │   ├── embedder/                   #   EmbedderClient interface + providers
│   │   └── vectordb/                   #   VectorDB interface + pgvector
│   ├── middleware/                      # Shared gRPC/HTTP interceptors
│   │   ├── auth/                       #   JWT/APIKey extraction
│   │   ├── logging/                    #   Structured access logging
│   │   ├── tracing/                    #   OTel trace propagation
│   │   ├── recovery/                   #   Panic recovery
│   │   ├── ratelimit/                  #   Redis sliding window
│   │   └── validation/                 #   Request validation
│   ├── resilience/                     # Circuit breaker, retry, bulkhead
│   ├── observability/                  # Tracer, metrics, logger, health
│   ├── config/                         # Viper loader + validator
│   ├── errors/                         # Domain error → gRPC status mapping
│   ├── nats/                           # NATS client helpers
│   ├── tokenizer/                      # tiktoken-go wrapper
│   ├── prompt/                         # Prompt template engine (EN/ZH)
│   ├── tenant/                         # Project context propagation
│   └── testutil/                       # Fixtures, mocks, testcontainers
│
├── migrations/                         # SQL migrations (golang-migrate)
├── deploy/
│   ├── docker-compose/                 # Dev environment
│   └── kubernetes/                     # Kustomize base + overlays
├── go.mod
├── buf.yaml
├── Makefile
└── README.md
```

---

## 6. Clean Architecture — Standardized Per Service

```
services/<service-name>/
├── cmd/server/main.go                  # Entry point, wire injection
├── internal/
│   ├── domain/                         # Layer 1: ZERO external imports
│   │   ├── entity.go                   #   Domain models (pure Go structs)
│   │   ├── value_object.go             #   Value objects (immutable)
│   │   ├── event.go                    #   Domain events
│   │   └── errors.go                   #   Domain-specific errors
│   ├── usecase/                        # Layer 2: imports domain only
│   │   ├── <usecase_name>.go           #   One file per use case
│   │   ├── port/
│   │   │   ├── input.go               #   Use case interfaces
│   │   │   └── output.go              #   Repository/external interfaces
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/                        # Layer 3: implements ports
│   │   ├── grpc/                       #   gRPC handler (controller)
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── postgres/
│   │   │   └── redis/
│   │   ├── client/                     #   gRPC clients to other services
│   │   └── event/                      #   NATS publisher/subscriber
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       ├── telemetry/
│       └── wire/wire.go
├── Dockerfile
└── README.md
```

### Dependency Rule (STRICT)

```
domain ← usecase ← adapter ← infra
 (inner)                     (outer)

✅ domain: ZERO external imports
✅ usecase: imports domain only; defines port interfaces
✅ adapter: imports usecase(ports) + domain; implements interfaces
✅ infra: imports everything; wires via Google Wire
```

---

## 7. Inter-Service Communication

### 7.1 Synchronous (gRPC)

```
Gateway → All services (fan-out by route)
Gateway → ingestion (insert blob)
Gateway → context (get context, get profiles)
Gateway → event (search events)
Gateway → admin (user CRUD, project config)
ingestion → engine (flush trigger, gRPC)
context → event (semantic search for context assembly)
```

### 7.2 Async Events (NATS JetStream)

| Stream | Subject | Publisher | Subscriber |
|--------|---------|-----------|------------|
| `memobase` | `memobase.buffer.ready` | memobase-ingestion | memobase-engine |
| `memobase` | `memobase.engine.completed` | memobase-engine | memobase-context (cache invalidate) |
| `memobase` | `memobase.engine.failed` | memobase-engine | memobase-ingestion (status rollback) |
| `memobase` | `memobase.profile.changed` | memobase-engine | memobase-context (cache invalidate) |
| `memobase` | `memobase.event.created` | memobase-engine | memobase-event (index embeddings) |
| `memobase` | `memobase.admin.project.updated` | memobase-admin | All (config reload) |
| `memobase` | `memobase.admin.user.deleted` | memobase-admin | ingestion, context, event (cascade) |

### 7.3 Pipeline Flow (Buffer Flush)

```
[Insert Blob]  →  ingestion stores blob + buffer entry
                        │
            buffer full? │ (token_sum > threshold)
                        ▼
    NATS: memobase.buffer.ready {user_id, project_id, buffer_ids[]}
                        │
                        ▼
    [memobase-engine] subscribes:
        1. Fetch blobs from DB (via ingestion gRPC or direct DB)
        2. LLM Call #1: entry_chat_summary
        3. Parallel:
           a. LLM Call #2: extract_topics → LLM Call #3: merge_yolo
           b. tag_event (conditional)
        4. Persist profiles (DB write)
        5. Persist events + embeddings (DB write)
        6. Emit: memobase.engine.completed
                        │
                        ▼
    [memobase-ingestion] marks buffer "done", deletes blobs
    [memobase-context] invalidates Redis cache
    [memobase-event] indexes new embeddings (if needed)
```

---

## 8. Cross-Cutting Concerns

| Concern | Package | Implementation |
|---------|---------|----------------|
| Auth (JWT + API Key) | `pkg/middleware/auth/` | Gateway validates; propagates via gRPC metadata |
| Multi-Tenancy | `pkg/tenant/` | `project_id` in gRPC metadata, DB composite PK |
| Rate Limiting | `pkg/middleware/ratelimit/` | Redis sliding window, per-project per-endpoint |
| Circuit Breaker | `pkg/resilience/` | sony/gobreaker, per-downstream-service |
| Retry | `pkg/resilience/` | Exponential backoff + jitter |
| Bulkhead | `pkg/resilience/` | Semaphore for LLM calls |
| Observability | `pkg/observability/` | OTel traces + Prometheus + slog JSON |
| Health | `pkg/observability/health.go` | gRPC Health v1 + HTTP /healthz /readyz /livez |
| Error Mapping | `pkg/errors/` | Domain errors → gRPC status → HTTP status |
| Token Counting | `pkg/tokenizer/` | tiktoken-go (gpt-4o encoder) |
| Prompt Templates | `pkg/prompt/` | EN/ZH template registry, mustache-like |

---

## 9. Migration from Python

### 9.1 Component Mapping

| Python Component | Go Component | Notes |
|-----------------|-------------|-------|
| FastAPI + Uvicorn | chi/v5 (Gateway) + gRPC (Services) | — |
| SQLAlchemy ORM | pgx + sqlc / GORM | Raw SQL for performance |
| Redis (redis-py) | go-redis/v9 | Same patterns |
| OpenAI SDK | go-openai / Bifrost client | Multi-provider |
| tiktoken | tiktoken-go | Compatible |
| structlog | slog (stdlib) | Structured JSON |
| alembic | golang-migrate | SQL migrations |
| OpenTelemetry-py | OpenTelemetry-go | Same exporters |
| asyncio.gather | errgroup.Group | Parallel processing |
| Pydantic | Protobuf + validator | Type-safe DTOs |

### 9.2 Preserved Design Decisions

| Decision | Status |
|---------|--------|
| YOLO merge (3 fixed LLM calls per flush) | **Preserved** |
| Buffer zone FSM (idle → processing → done/failed) | **Preserved** |
| Profile caching (Redis TTL 20min) | **Preserved** |
| Composite PK (id, project_id) | **Preserved** |
| Non-persistent blobs (delete after processing) | **Preserved** |
| Profile schema (topic/sub_topic/content) | **Preserved** |
| Prompt templates (EN/ZH) | **Preserved** |

---

## 10. Document Index

| Document | Description |
|----------|-------------|
| [01-gateway.md](./01-gateway.md) | API Gateway — REST, Auth, MCP Server |
| [02-memobase-ingestion.md](./02-memobase-ingestion.md) | Blob ingestion, Buffer zone, Flush trigger |
| [03-memobase-engine.md](./03-memobase-engine.md) | Memory engine — Profile extraction, YOLO merge, Event summary |
| [04-memobase-context.md](./04-memobase-context.md) | Context assembly, Profile query, Read path |
| [05-memobase-event.md](./05-memobase-event.md) | Event CRUD, Semantic search, Gist search |
| [06-memobase-admin.md](./06-memobase-admin.md) | User, Project, Billing, Config management |
| [07-shared-packages.md](./07-shared-packages.md) | Shared `pkg/` packages |
| [08-data-models.md](./08-data-models.md) | Domain models + Protobuf definitions |
| [09-deployment.md](./09-deployment.md) | Docker Compose + Kubernetes |
