# OpenViking — Enterprise Golang Architecture

> **Version**: 2.0 | **Date**: 2026-05-09 | **Status**: Approved  
> **Migration**: Python/Rust monolith → Golang Gateway + Microservices  
> **Stack**: Go 1.23+ · gRPC · NATS JetStream · Embedded VectorDB · Redis · AES-256-GCM

---

## 1. Executive Summary

Chuyển đổi OpenViking từ Python/Rust monolith (FastAPI + RAGFS) sang hệ thống **enterprise-grade Golang microservices** với:

- **API Gateway** — single entry point: REST/gRPC/MCP/WebDAV, authentication (DEV/API_KEY/TRUSTED), rate limiting, protocol translation
- **6 Domain Services** — tách biệt theo bounded context, giao tiếp qua gRPC
- **Clean Architecture** per service — Domain → Use Cases → Interface Adapters → Frameworks
- **Production-grade** — observability (OTel), resilience (circuit breaker), multi-tenancy, envelope encryption, horizontal scaling

### Design Principles

| # | Principle | Rationale |
|---|-----------|-----------|
| 1 | **Single Monorepo** | Shared proto, shared `pkg/`, single `go.mod`, unified CI |
| 2 | **Unified Gateway** | One entry point for REST + MCP + WebDAV + SDK |
| 3 | **Clean Architecture per Service** | 4 layers: domain → usecase(+port) → adapter → infra |
| 4 | **gRPC internal, REST external** | Type-safe inter-service, developer-friendly external |
| 5 | **NATS JetStream async** | Pipeline orchestration, background task processing |
| 6 | **Viking URI as First-Class Citizen** | `viking://` protocol preserved in all services |
| 7 | **Multi-Tenant by Design** | Account/User/Agent namespace isolation with RBAC |
| 8 | **Envelope Encryption Native** | Per-file AES-256-GCM, KMS-pluggable |

---

## 2. System Context

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           External Consumers                                │
│  ov CLI · Python SDK · MCP Clients (Claude/OpenCode) · WebDAV · VikingBot  │
└──────────┬─────────┬──────────────┬──────────────┬──────────────┬──────────┘
           │         │              │              │              │
        REST/HTTP  gRPC-Web     MCP(SSE)       WebDAV       Bot Gateway
           │         │              │              │              │
┌──────────▼─────────▼──────────────▼──────────────▼──────────────▼──────────┐
│                    UNIFIED API GATEWAY  (openviking-gateway)                 │
│  Auth(DEV/API_KEY/TRUSTED) · RateLimit · CORS · Protocol Translation       │
│  Circuit Breaker · Request Validation · Tenant Resolution · MCP Server     │
│  WebDAV Proxy · Namespace Access Check                                      │
└──┬──────────┬──────────┬──────────┬──────────┬──────────┬─────────────────┘
   │          │          │          │          │          │
gRPC       gRPC       gRPC       gRPC       gRPC       gRPC
   │          │          │          │          │          │
┌──▼───┐ ┌───▼────┐ ┌───▼───┐ ┌───▼────┐ ┌───▼────┐ ┌──▼─────┐
│ FS   │ │ Search │ │Session│ │Resource│ │Encrypt │ │ Admin  │
│ Svc  │ │  Svc   │ │  Svc  │ │  Svc   │ │  Svc   │ │  Svc   │
│      │ │        │ │       │ │        │ │        │ │        │
│CRUD  │ │Hierarc.│ │2-Phase│ │Ingest  │ │Envelope│ │Account │
│Tree  │ │Retriev.│ │Commit │ │Pipeline│ │Per-File│ │API Key │
│Grep  │ │Rerank  │ │WM v2  │ │Parse   │ │KMS     │ │Health  │
│Glob  │ │Hotness │ │Memory │ │Embed   │ │Rotate  │ │Metrics │
└──┬───┘ └───┬────┘ └───┬───┘ └───┬────┘ └───┬────┘ └──┬─────┘
   └─────────┴──────────┴─────────┴──────────┴─────────┘
                                │
               ┌────────────────▼────────────────────┐
               │      SHARED INFRASTRUCTURE           │
               │  VikingFS(Go) · VectorDB(Embedded)   │
               │  Redis · NATS · Bifrost(LLM/Embed)   │
               │  KMS(Local/Vault) · OTel Collector    │
               └─────────────────────────────────────┘
```

---

## 3. Service Inventory

### 3.1 Complete Service Map

| # | Service | gRPC Port | Bounded Context | Origin Layer |
|---|---------|----------|-----------------|-------------|
| 0 | `openviking-gateway` | 8080(HTTP) 8081(gRPC) 8082(MCP) | Routing, Auth, MCP, WebDAV | L1 Presentation |
| 1 | `openviking-fs` | 9011 | File CRUD, Tree, Grep, Glob, Relations | L2 FSService + L5 VikingFS |
| 2 | `openviking-search` | 9012 | Hierarchical Retrieval, Rerank, Hotness | L2 SearchService + L4 Retriever |
| 3 | `openviking-session` | 9013 | Session lifecycle, 2-phase commit, WM v2, Memory Extract | L2 SessionService + L4 Session |
| 4 | `openviking-resource` | 9014 | Resource ingestion, Parse engine, Watch/Refresh | L2 ResourceService + L4 Processor |
| 5 | `openviking-crypto` | 9015 | Envelope encryption, KMS adapters, Key rotation | L5 Crypto |
| 6 | `openviking-admin` | 9030 | Account/User/Key CRUD, Health, Metrics, Maintenance | L2 Admin + TaskTracker |

### 3.2 What Is Shared vs Separate

| Component | Strategy | Rationale |
|-----------|----------|-----------|
| **Gateway** | **Unified** | Single entry point, single auth/rate-limit, MCP+WebDAV proxy |
| **Admin** | **Unified** | Account, API keys, health — same for all domains |
| **`pkg/`** | **Unified** | Proto, middleware, resilience, observability, VikingFS core, URI resolution |
| **`pkg/viking/`** | **Unified** | Shared domain types: Context, Namespace, URI, ContextType/Level |
| **`pkg/adapters/`** | **Unified** | VectorDB, Embedder, VLM, KMS, Reranker interfaces |
| **FS/Search/Session/Resource** | **Separate** | Domain-specific business logic |

---

## 4. Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Language** | Go 1.23+ | Performance, concurrency, single binary |
| **External API** | chi/v5 + OpenAPI 3.1 | Gateway REST |
| **Internal RPC** | gRPC + Protobuf v3 | Type-safe inter-service |
| **Async** | NATS JetStream | Pipeline orchestration, background tasks |
| **Vector DB** | Embedded (Weaviate-lite/custom) or Qdrant | Hybrid dense+sparse search |
| **Cache** | Redis 7+ | Search cache, rate limiter, session cache |
| **Filesystem** | Go-native VikingFS | Replace RAGFS Rust with Go implementation |
| **LLM/Embed Gateway** | Bifrost | Unified provider interface (12+ embedding providers) |
| **Encryption** | Go `crypto/aes` + `crypto/cipher` | AES-256-GCM envelope encryption |
| **KMS** | Local / HashiCorp Vault / Cloud KMS | Pluggable key management |
| **DI** | Google Wire | Compile-time dependency injection |
| **Observability** | OTel + Prometheus + slog | Traces, metrics, structured logs |
| **Config** | Viper + ENV | 12-factor compliance |

---

## 5. Monorepo Structure

```
openviking-go/
├── api/proto/                              # ALL Protobuf definitions
│   ├── common/v1/                          #   Shared: pagination, errors, health
│   │   ├── pagination.proto
│   │   ├── errors.proto
│   │   └── health.proto
│   ├── viking/v1/                          #   Shared: context, namespace, URI
│   │   ├── context.proto                   #     Context, ContextType, ContextLevel
│   │   ├── namespace.proto                 #     URI resolution messages
│   │   └── identity.proto                  #     UserIdentifier, Role, RequestContext
│   ├── gateway/v1/                         #   Gateway-specific
│   │   └── gateway.proto
│   ├── fs/v1/                              #   Filesystem service
│   │   └── fs.proto
│   ├── search/v1/                          #   Search service
│   │   └── search.proto
│   ├── session/v1/                         #   Session service
│   │   └── session.proto
│   ├── resource/v1/                        #   Resource service
│   │   └── resource.proto
│   ├── crypto/v1/                          #   Crypto service
│   │   └── crypto.proto
│   └── admin/v1/                           #   Admin service
│       └── admin.proto
│
├── services/                               # Service binaries
│   ├── openviking-gateway/                 #   Unified API Gateway
│   │   ├── cmd/server/main.go
│   │   └── internal/
│   │       ├── domain/
│   │       ├── usecase/
│   │       ├── adapter/
│   │       │   ├── http/                   #     REST handlers (17 route groups)
│   │       │   ├── mcp/                    #     MCP Streamable HTTP (9 tools)
│   │       │   ├── webdav/                 #     WebDAV proxy
│   │       │   └── grpc/                   #     gRPC-Web handler
│   │       └── infra/
│   ├── openviking-fs/                      #   Filesystem Service
│   ├── openviking-search/                  #   Search Service
│   ├── openviking-session/                 #   Session Service
│   ├── openviking-resource/                #   Resource Ingestion Service
│   ├── openviking-crypto/                  #   Encryption Service
│   └── openviking-admin/                   #   Admin Service
│
├── pkg/                                    # Shared packages (NO business logic)
│   ├── viking/                             #   Shared domain types
│   │   ├── context.go                      #     Context, ContextType, ContextLevel
│   │   ├── namespace.go                    #     URI resolution, ownership, canonical roots
│   │   ├── uri.go                          #     URI validation, canonicalization
│   │   ├── identity.go                     #     UserIdentifier, Role, RequestContext
│   │   ├── errors.go                       #     OpenVikingError hierarchy
│   │   └── tiered.go                       #     L0/L1/L2 constants, file conventions
│   ├── adapters/                           #   Infrastructure adapter interfaces
│   │   ├── vectordb/                       #     VectorDB interface + implementations
│   │   ├── embedder/                       #     EmbedderClient interface + 12 providers
│   │   ├── vlm/                            #     VLMClient interface + providers
│   │   ├── reranker/                       #     Reranker interface + providers
│   │   ├── kms/                            #     KMS interface (Local/Vault/Cloud)
│   │   └── storage/                        #     FileStorage interface (local/S3)
│   ├── vikingfs/                           #   Go-native filesystem engine
│   │   ├── fs.go                           #     Core FS operations (read/write/mkdir/rm)
│   │   ├── tree.go                         #     Directory tree operations
│   │   └── lock.go                         #     PathLock (point/subtree/mv)
│   ├── middleware/                          #   Shared gRPC/HTTP interceptors
│   │   ├── auth/                           #     DEV/API_KEY/TRUSTED auth
│   │   ├── logging/                        #     Structured access logging
│   │   ├── tracing/                        #     OTel trace propagation
│   │   ├── recovery/                       #     Panic recovery
│   │   ├── ratelimit/                      #     Rate limiting
│   │   └── validation/                     #     Request validation
│   ├── resilience/                         #   Circuit breaker, retry, bulkhead
│   ├── observability/                      #   Tracer, metrics, logger, health
│   ├── config/                             #   Viper loader + validator
│   ├── errors/                             #   Domain error → gRPC status mapping
│   ├── nats/                               #   NATS client helpers
│   ├── auth/                               #   API key manager, RBAC
│   ├── tenant/                             #   Tenant context extraction
│   ├── pagination/                         #   Cursor/offset pagination
│   ├── parse/                              #   File parser registry (Go tree-sitter)
│   │   ├── registry.go                     #     Extension → parser routing
│   │   ├── treesitter.go                   #     tree-sitter Go bindings
│   │   ├── markdown.go                     #     Markdown parser
│   │   └── document.go                     #     PDF/DOCX parser
│   └── testutil/                           #   Fixtures, mocks, testcontainers
│
├── migrations/                             #   SQL migration files
├── deploy/
│   ├── docker-compose/                     #   Dev environment
│   └── kubernetes/                         #   Kustomize overlays
├── go.mod
├── buf.yaml
├── Makefile
└── README.md
```

---

## 6. Clean Architecture — Standardized Per Service

Mỗi service trong `services/<name>/` tuân theo 4-layer **chuẩn hóa**:

```
services/<service-name>/
├── cmd/
│   └── server/
│       └── main.go                     # Entry point, wire injection
├── internal/
│   ├── domain/                         # Layer 1: Entities (ZERO imports)
│   │   ├── entity.go                   #   Domain models (pure Go structs)
│   │   ├── value_object.go             #   Value objects (immutable)
│   │   ├── event.go                    #   Domain events
│   │   └── errors.go                   #   Domain-specific errors
│   │
│   ├── usecase/                        # Layer 2: Use Cases (imports domain only)
│   │   ├── <usecase_name>.go           #   One file per use case
│   │   ├── port/                       #   Port interfaces
│   │   │   ├── input.go               #     Input ports (use case interfaces)
│   │   │   └── output.go             #     Output ports (repository, external)
│   │   └── dto/                        #   Data Transfer Objects
│   │       ├── request.go
│   │       └── response.go
│   │
│   ├── adapter/                        # Layer 3: Interface Adapters
│   │   ├── grpc/                       #   gRPC handlers (controllers)
│   │   │   ├── handler.go
│   │   │   └── mapper.go             #   Proto ↔ Domain mapping
│   │   ├── repository/                 #   Output adapter implementations
│   │   ├── client/                     #   External service gRPC clients
│   │   └── event/                      #   NATS publisher/subscriber
│   │
│   └── infra/                          # Layer 4: Frameworks & Drivers
│       ├── config/config.go
│       ├── server/grpc.go
│       ├── telemetry/
│       └── wire/
├── Dockerfile
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
FS ↔ Crypto (encrypt/decrypt on read/write)
Search → FS (read context) + VectorDB adapter
Session → FS (read/write messages) + Search (context-aware retrieval)
Resource → FS (write parsed content) + Search (embed + index)
Admin → All (health checks)
```

### 7.2 Async Events (NATS JetStream)

| Stream | Subject | Publisher | Subscriber |
|--------|---------|-----------|------------|
| `openviking` | `ov.resource.ingested` | Resource | Search (reindex) |
| `openviking` | `ov.resource.parsed` | Resource | FS (write content) |
| `openviking` | `ov.session.committed` | Session | Search (update hotness) |
| `openviking` | `ov.session.memory.extracted` | Session | FS (write memories) |
| `openviking` | `ov.content.written` | FS | Search (embed + upsert) |
| `openviking` | `ov.content.deleted` | FS | Search (remove from index) |
| `openviking` | `ov.crypto.key.rotated` | Crypto | FS (re-wrap files) |
| `admin` | `admin.account.created` | Admin | FS (init dirs), Search (init collection) |
| `admin` | `admin.account.deleted` | Admin | All (cascade cleanup) |

---

## 8. Port Assignment

| Service | gRPC | Health/Metrics | REST |
|---------|------|---------------|------|
| openviking-gateway | 8081 | 8083 | 8080 |
| openviking-fs | 9011 | 9091 | — |
| openviking-search | 9012 | 9092 | — |
| openviking-session | 9013 | 9093 | — |
| openviking-resource | 9014 | 9094 | — |
| openviking-crypto | 9015 | 9095 | — |
| openviking-admin | 9030 | 9099 | — |

---

## 9. Cross-Cutting Concerns

| Concern | Package | Notes |
|---------|---------|-------|
| Auth (DEV/API_KEY/TRUSTED) | `pkg/middleware/auth/` + `pkg/auth/` | Gateway validates; propagates via gRPC metadata |
| Multi-Tenancy | `pkg/tenant/` | Account/User/Agent namespace isolation |
| Rate Limiting | `pkg/middleware/ratelimit/` | Redis sliding window, per-tenant per-endpoint |
| Circuit Breaker | `pkg/resilience/` | sony/gobreaker, per-downstream-service |
| Retry | `pkg/resilience/` | Exponential backoff + jitter |
| Bulkhead | `pkg/resilience/` | Channel-based semaphore for LLM/Embed calls |
| Observability | `pkg/observability/` | OTel traces + Prometheus metrics + slog JSON logs |
| Health | `pkg/observability/health.go` | gRPC Health v1 + HTTP /healthz /readyz /livez |
| Error Mapping | `pkg/errors/` | OpenVikingError → gRPC status → HTTP status |
| Encryption | `pkg/adapters/kms/` | Envelope encryption transparent to services |

---

## 10. Migration Mapping — Python → Go

| Python Component | Go Service | Go Package |
|-----------------|-----------|-----------|
| `server/app.py` (FastAPI) | openviking-gateway | `internal/adapter/http/` |
| `server/mcp_endpoint.py` (9 tools) | openviking-gateway | `internal/adapter/mcp/` |
| `server/routers/webdav.py` | openviking-gateway | `internal/adapter/webdav/` |
| `server/auth.py` (3 modes) | openviking-gateway | `pkg/middleware/auth/` |
| `service/fs_service.py` | openviking-fs | `internal/usecase/` |
| `storage/viking_fs.py` (2199 lines) | openviking-fs + `pkg/vikingfs/` | `internal/adapter/repository/` |
| `retrieve/hierarchical_retriever.py` | openviking-search | `internal/usecase/` |
| `retrieve/memory_lifecycle.py` | openviking-search | `internal/domain/` |
| `service/search_service.py` | openviking-search | `internal/usecase/` |
| `session/session.py` (2629 lines) | openviking-session | `internal/usecase/` |
| `session/compressor.py` | openviking-session | `internal/usecase/` |
| `service/session_service.py` | openviking-session | `internal/usecase/` |
| `utils/resource_processor.py` | openviking-resource | `internal/usecase/` |
| `parse/` (9 files) | openviking-resource + `pkg/parse/` | `internal/adapter/` |
| `resource/watch_manager.py` | openviking-resource | `internal/usecase/` |
| `crypto/encryptor.py` | openviking-crypto | `internal/usecase/` |
| `crypto/providers.py` | openviking-crypto | `internal/adapter/repository/` |
| `models/embedder/` (13 files) | `pkg/adapters/embedder/` | Shared adapter |
| `models/vlm/` | `pkg/adapters/vlm/` | Shared adapter |
| `models/rerank.py` | `pkg/adapters/reranker/` | Shared adapter |
| `core/context.py` | `pkg/viking/context.go` | Shared domain |
| `core/namespace.py` | `pkg/viking/namespace.go` | Shared domain |
| `storage/vikingdb_manager.py` | `pkg/adapters/vectordb/` | Shared adapter |

---

## 11. Document Index

| Document | Description |
|----------|-------------|
| [01-gateway.md](./01-gateway.md) | API Gateway — REST (17 routes), MCP (9 tools), WebDAV, Auth |
| [02-fs-service.md](./02-fs-service.md) | Filesystem Service — CRUD, Tree, Grep, Glob, Relations |
| [03-search-service.md](./03-search-service.md) | Search Service — HierarchicalRetriever, Rerank, Hotness |
| [04-session-service.md](./04-session-service.md) | Session Service — 2-Phase Commit, WM v2, Memory Extract |
| [05-resource-service.md](./05-resource-service.md) | Resource Service — Ingestion Pipeline, Parse, Watch |
| [06-crypto-service.md](./06-crypto-service.md) | Crypto Service — Envelope Encryption, KMS, Key Rotation |
| [07-admin-service.md](./07-admin-service.md) | Admin Service — Account/User/Key CRUD, Health, Maintenance |
| [08-shared-packages.md](./08-shared-packages.md) | Shared `pkg/` — Viking domain, adapters, middleware |
| [09-deployment.md](./09-deployment.md) | Docker Compose + Kubernetes deployment |
