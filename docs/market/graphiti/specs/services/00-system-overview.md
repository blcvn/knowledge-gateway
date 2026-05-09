# VNP Graphiti — Enterprise Golang Architecture

**Version:** 2.0 | **Date:** 2026-05-09 | **Status:** Approved  
**Migration:** Python monolith → Golang Gateway + Microservices  
**Architecture Style:** API Gateway + Domain Services, Clean Architecture per service

---

## 1. Executive Summary

Chuyển đổi Graphiti từ Python monolith (FastAPI + graphiti_core) sang hệ thống **enterprise-grade Golang microservices** với:

- **API Gateway** — single entry point, authentication, rate limiting, protocol translation (REST/gRPC/MCP)
- **5 Domain Services** — mỗi service tách biệt theo bounded context, giao tiếp qua gRPC
- **Clean Architecture** per service — Entities → Use Cases → Interface Adapters → Frameworks
- **Production-grade** — observability, resilience, multi-tenancy, horizontal scaling

---

## 2. System Context Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            External Consumers                               │
│  AI Agents │ Chat Apps │ Data Pipelines │ MCP Clients │ Admin Dashboard     │
└──────┬─────────┬──────────────┬──────────────┬──────────────┬───────────────┘
       │         │              │              │              │
       │      REST/HTTP      gRPC          MCP/SSE       WebSocket
       │         │              │              │              │
┌──────▼─────────▼──────────────▼──────────────▼──────────────▼───────────────┐
│                        API Gateway (graphiti-gateway)                        │
│  ┌────────────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌───────────────┐  │
│  │ Auth/AuthZ │ │ Rate     │ │ Protocol  │ │ Circuit  │ │ Request       │  │
│  │ (JWT/mTLS) │ │ Limiter  │ │ Translate │ │ Breaker  │ │ Router        │  │
│  └────────────┘ └──────────┘ └───────────┘ └──────────┘ └───────────────┘  │
└──────┬─────────────┬──────────────┬──────────────┬──────────────┬───────────┘
       │             │              │              │              │
    gRPC          gRPC           gRPC           gRPC          gRPC
       │             │              │              │              │
┌──────▼──────┐ ┌────▼─────┐ ┌─────▼──────┐ ┌────▼─────┐ ┌──────▼──────┐
│  Ingestion  │ │  Search  │ │ Knowledge  │ │   Graph  │ │    Admin    │
│  Service    │ │  Service │ │ Processing │ │  Storage │ │   Service   │
│             │ │          │ │  Service   │ │  Service │ │             │
│ Episode     │ │ Hybrid   │ │ LLM Calls  │ │ Neo4j/   │ │ Tenant Mgmt │
│ Ingestion   │ │ Search   │ │ Entity Ext.│ │ FalkorDB │ │ Index Mgmt  │
│ Bulk Import │ │ Reranking│ │ Dedup/Res. │ │ CRUD Ops │ │ Health/Ops  │
│ Saga Mgmt   │ │ Filtering│ │ Community  │ │ Tx Mgmt  │ │ Migrations  │
└─────────────┘ └──────────┘ └────────────┘ └──────────┘ └─────────────┘
       │             │              │              │              │
       └─────────────┴──────────────┴──────┬───────┴──────────────┘
                                           │
                              ┌────────────▼────────────┐
                              │   Shared Infrastructure  │
                              │ ┌──────┐ ┌────────────┐ │
                              │ │ NATS │ │ Redis      │ │
                              │ │ /Kafka│ │ (Cache)    │ │
                              │ └──────┘ └────────────┘ │
                              │ ┌──────────┐ ┌────────┐ │
                              │ │ OTel     │ │ Vault  │ │
                              │ │ Collector│ │ /SOPS  │ │
                              │ └──────────┘ └────────┘ │
                              └─────────────────────────┘
```

---

## 3. Service Decomposition

### 3.1 Service Boundary Matrix

| Service | Bounded Context | Current Python Layers | Owns Data | Protocol |
|---------|----------------|----------------------|-----------|----------|
| **graphiti-gateway** | API Routing, Auth, Protocol Translation | L7 (REST, MCP) | — | REST/gRPC/MCP/WS |
| **graphiti-ingestion** | Episode Lifecycle, Pipeline Orchestration | L6 (Orchestration) + L5 partial | Episode queue state | gRPC |
| **graphiti-search** | Hybrid Search, Reranking, Filtering | L4 (Search & Retrieval) | Search indices cache | gRPC |
| **graphiti-knowledge** | LLM Integration, Entity/Edge Extraction, Resolution | L5 (Knowledge Processing) + L3 (AI Services) | Prompt templates | gRPC |
| **graphiti-store** | Graph DB Abstraction, CRUD, Transactions | L2 (Data Access) + L1 (Storage & Driver) | Graph database | gRPC |
| **graphiti-admin** | Tenant Management, Index Ops, Health | L6 partial (maintenance) | Config store | gRPC |

### 3.2 Inter-Service Communication

```
                    ┌──────────────────────┐
                    │   graphiti-gateway    │
                    └──────┬───────────────┘
                           │ gRPC
              ┌────────────┼────────────────┐
              │            │                │
      ┌───────▼──────┐ ┌──▼───────┐  ┌─────▼──────┐
      │  ingestion   │ │  search  │  │   admin    │
      └───────┬──────┘ └──┬───────┘  └─────┬──────┘
              │           │                │
              │     ┌─────▼──────┐         │
              ├────►│ knowledge  │◄────────┤
              │     └─────┬──────┘         │
              │           │                │
              │     ┌─────▼──────┐         │
              └────►│   store    │◄────────┘
                    └────────────┘
```

**Communication Rules:**
1. Gateway → tất cả services (fan-out)
2. Ingestion → Knowledge (extract, resolve) → Store (persist)
3. Search → Knowledge (rerank via cross-encoder) → Store (query)
4. Admin → Store (maintenance ops)
5. **KHÔNG** có circular dependencies giữa services

### 3.3 Async Event Bus (NATS JetStream)

| Event | Publisher | Subscribers | Purpose |
|-------|-----------|-------------|---------|
| `episode.ingested` | Ingestion | Search (reindex), Admin (metrics) | Post-ingestion notification |
| `entity.resolved` | Knowledge | Search (update cache) | Entity dedup completed |
| `community.rebuilt` | Knowledge | Search (update index) | Community detection done |
| `tenant.created` | Admin | Store (init schema), Search (init index) | New tenant onboarding |
| `health.degraded` | Any | Admin (alerting) | Circuit breaker triggered |

---

## 4. Technology Stack

### 4.1 Core

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Language** | Go 1.23+ | Performance, concurrency, single binary deployment |
| **API Gateway** | Custom (net/http + chi router) | Full control, no vendor lock-in |
| **Inter-service RPC** | gRPC + Protobuf v3 | Type-safe, efficient, streaming support |
| **Async Messaging** | NATS JetStream | Lightweight, Go-native, exactly-once delivery |
| **Configuration** | Viper + YAML/ENV | 12-factor compliance, hot reload |
| **DI Container** | Wire (google/wire) | Compile-time dependency injection |

### 4.2 Data

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Graph DB** | Neo4j 5.x (primary) | Mature, native vector index, real transactions |
| **Graph DB (alt)** | FalkorDB / Kuzu / Neptune | Pluggable via Store service driver interface |
| **Cache** | Redis 7+ (Cluster) | Embedding cache, search result cache, rate limiter backend |
| **Message Queue** | NATS JetStream | Event-driven inter-service communication |
| **Config/Secrets** | HashiCorp Vault / SOPS | Enterprise secret management |

### 4.3 AI/ML Integration

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **LLM Gateway** | Bifrost / LiteLLM proxy | Unified provider interface, failover, rate limiting |
| **Embedding** | OpenAI / Voyage / Local | Via Bifrost, configurable per tenant |
| **Cross-Encoder** | OpenAI Reranker / BGE local | Neural reranking support |
| **Structured Output** | JSON Schema enforcement | Go struct validation, no Pydantic equivalent needed |

### 4.4 Observability

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Tracing** | OpenTelemetry SDK for Go | Distributed tracing across all services |
| **Metrics** | Prometheus + OTel Metrics | Service-level SLIs, business metrics |
| **Logging** | zerolog / slog (structured JSON) | High-performance structured logging |
| **Health** | gRPC Health v1 + HTTP /healthz | K8s readiness/liveness probes |
| **Profiling** | pprof (continuous) | Production profiling |

### 4.5 Deployment

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Container** | Docker (multi-stage, distroless) | Minimal attack surface, <20MB images |
| **Orchestration** | Kubernetes / Docker Compose | Production K8s, dev Docker Compose |
| **CI/CD** | GitHub Actions | Automated testing, linting, deployment |
| **Service Mesh** | Istio / Linkerd (optional) | mTLS, traffic management |

---

## 5. Clean Architecture — Standard Per Service

Mỗi service tuân thủ **Clean Architecture** với 4 concentric layers:

```
┌─────────────────────────────────────────────────────────┐
│  Frameworks & Drivers  (outermost — replaceable)         │
│  ┌───────────────────────────────────────────────────┐   │
│  │  Interface Adapters  (controllers, presenters)     │   │
│  │  ┌─────────────────────────────────────────────┐   │   │
│  │  │  Use Cases / Application  (business rules)   │   │   │
│  │  │  ┌───────────────────────────────────────┐   │   │   │
│  │  │  │  Entities / Domain  (core models)      │   │   │   │
│  │  │  └───────────────────────────────────────┘   │   │   │
│  │  └─────────────────────────────────────────────┘   │   │
│  └───────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 5.1 Directory Convention (mỗi service)

```
services/<service-name>/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point, wire injection
├── internal/
│   ├── domain/                     # Layer 1: Entities
│   │   ├── entity.go               #   Domain models (pure Go structs)
│   │   ├── value_object.go         #   Value objects
│   │   ├── event.go                #   Domain events
│   │   └── errors.go               #   Domain-specific errors
│   ├── usecase/                    # Layer 2: Use Cases
│   │   ├── <usecase_name>.go       #   Application business rules
│   │   ├── port/                   #   Port interfaces (input + output)
│   │   │   ├── input.go            #     Input ports (use case interfaces)
│   │   │   └── output.go           #     Output ports (repository, external)
│   │   └── dto/                    #   Use case DTOs
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/                    # Layer 3: Interface Adapters
│   │   ├── grpc/                   #   gRPC handlers (controllers)
│   │   │   ├── handler.go
│   │   │   └── mapper.go           #   Proto ↔ Domain mapping
│   │   ├── repository/             #   Output adapter implementations
│   │   │   ├── neo4j/
│   │   │   ├── falkordb/
│   │   │   └── redis/
│   │   ├── client/                 #   External service clients
│   │   │   ├── knowledge_client.go
│   │   │   └── store_client.go
│   │   └── event/                  #   Event publisher/subscriber
│   │       ├── publisher.go
│   │       └── subscriber.go
│   └── infra/                      # Layer 4: Frameworks & Drivers
│       ├── config/                 #   Configuration loading
│       │   └── config.go
│       ├── server/                 #   gRPC server setup
│       │   └── grpc.go
│       ├── telemetry/              #   OTel, metrics, logging
│       │   ├── tracer.go
│       │   ├── metrics.go
│       │   └── logger.go
│       ├── middleware/             #   gRPC interceptors
│       │   ├── auth.go
│       │   ├── logging.go
│       │   ├── recovery.go
│       │   └── tracing.go
│       └── wire/                   #   Wire DI providers
│           ├── wire.go
│           └── wire_gen.go
├── api/
│   └── proto/                      # Protobuf definitions
│       └── <service>.proto
├── pkg/                            # Public packages (shared)
│   └── client/                     #   Generated gRPC client
├── migrations/                     # DB migrations
├── Dockerfile
├── Makefile
└── README.md
```

### 5.2 Dependency Rule

```
domain ← usecase ← adapter ← infra
  (inner)                    (outer)

- domain: ZERO external imports (no gRPC, no DB, no framework)
- usecase: imports domain only, defines port interfaces
- adapter: imports usecase (ports) + domain, implements interfaces
- infra: imports everything, wires dependencies
```

---

## 6. Shared Packages (Mono-repo)

```
pkg/
├── proto/                          # Shared Protobuf definitions
│   ├── common/                     #   Shared message types
│   │   ├── pagination.proto
│   │   ├── temporal.proto
│   │   └── errors.proto
│   ├── ingestion/v1/
│   │   └── ingestion.proto
│   ├── search/v1/
│   │   └── search.proto
│   ├── knowledge/v1/
│   │   └── knowledge.proto
│   ├── store/v1/
│   │   └── store.proto
│   └── admin/v1/
│       └── admin.proto
├── graph/                          # Shared graph domain types
│   ├── node.go                     #   EntityNode, EpisodicNode, etc.
│   ├── edge.go                     #   EntityEdge, EpisodicEdge, etc.
│   ├── temporal.go                 #   Bi-temporal model types
│   └── group.go                    #   Multi-tenancy primitives
├── middleware/                     # Shared gRPC interceptors
│   ├── auth.go
│   ├── logging.go
│   ├── tracing.go
│   ├── recovery.go
│   └── ratelimit.go
├── resilience/                     # Circuit breaker, retry, bulkhead
│   ├── circuit_breaker.go
│   ├── retry.go
│   └── bulkhead.go
├── observability/                  # OTel helpers
│   ├── tracer.go
│   ├── metrics.go
│   └── logger.go
├── config/                         # Configuration primitives
│   ├── loader.go
│   └── validator.go
└── testutil/                       # Testing utilities
    ├── fixtures.go
    ├── mocks.go
    └── integration.go
```

---

## 7. Deployment Topology

### 7.1 Development (Docker Compose)

```yaml
services:
  gateway:          # port 8080 (REST) + 8081 (gRPC) + 8082 (MCP)
  ingestion:        # port 9001 (gRPC)
  search:           # port 9002 (gRPC)
  knowledge:        # port 9003 (gRPC)
  store:            # port 9004 (gRPC)
  admin:            # port 9005 (gRPC)
  neo4j:            # port 7474/7687
  redis:            # port 6379
  nats:             # port 4222
  otel-collector:   # port 4317 (OTLP)
  jaeger:           # port 16686 (UI)
  prometheus:       # port 9090
```

### 7.2 Production (Kubernetes)

```
Namespace: graphiti-system
├── Deployments
│   ├── graphiti-gateway        (replicas: 3, HPA: 3-10)
│   ├── graphiti-ingestion      (replicas: 2, HPA: 2-8)
│   ├── graphiti-search         (replicas: 3, HPA: 3-12)
│   ├── graphiti-knowledge      (replicas: 2, HPA: 2-6)
│   ├── graphiti-store          (replicas: 2, HPA: 2-6)
│   └── graphiti-admin          (replicas: 1)
├── StatefulSets
│   ├── neo4j-core              (replicas: 3, cluster)
│   ├── redis-cluster           (replicas: 6)
│   └── nats-cluster            (replicas: 3)
├── Services (ClusterIP)
│   ├── graphiti-gateway-grpc
│   ├── graphiti-ingestion-grpc
│   ├── graphiti-search-grpc
│   ├── graphiti-knowledge-grpc
│   ├── graphiti-store-grpc
│   └── graphiti-admin-grpc
├── Ingress
│   └── graphiti-gateway (HTTPS, path-based routing)
├── ConfigMaps / Secrets
│   ├── graphiti-config
│   └── graphiti-secrets (from Vault)
└── ServiceMonitor (Prometheus)
    └── graphiti-metrics
```

---

## 8. Cross-Cutting Concerns

### 8.1 Authentication & Authorization

| Layer | Mechanism | Scope |
|-------|-----------|-------|
| External → Gateway | JWT (RS256) + API Key | Consumer authentication |
| Gateway → Services | mTLS + metadata propagation | Service-to-service trust |
| Service → Service | mTLS (Istio sidecar) | Zero-trust internal |
| Multi-tenant | `X-Tenant-ID` header → `group_id` | Partition isolation |

### 8.2 Resilience

| Pattern | Implementation | Config |
|---------|---------------|--------|
| **Circuit Breaker** | sony/gobreaker | per-service, per-endpoint thresholds |
| **Retry** | gRPC retry policy | exponential backoff, max 3 attempts |
| **Timeout** | gRPC deadline propagation | cascade through service chain |
| **Bulkhead** | Go channel-based semaphore | isolate LLM calls from DB calls |
| **Rate Limiting** | Redis-backed sliding window | per-tenant, per-endpoint |

### 8.3 Observability

| Signal | Tool | Granularity |
|--------|------|-------------|
| **Traces** | OTel → Jaeger/Tempo | Per-request, cross-service |
| **Metrics** | OTel → Prometheus | Per-endpoint latency, error rate, saturation |
| **Logs** | zerolog → stdout → Loki | Structured JSON, correlation IDs |
| **Health** | gRPC Health + HTTP /healthz | Liveness + Readiness + Startup |
| **Profiling** | pprof (continuous) | CPU, memory, goroutine |

### 8.4 Multi-Tenancy

```
Request flow:
  Client → JWT (tenant_id claim) → Gateway extracts tenant_id
    → Propagates as gRPC metadata "x-tenant-id"
    → Each service maps to group_id partition
    → Store service applies group_id filter/isolation per DB backend
```

| Backend | Isolation Strategy |
|---------|-------------------|
| Neo4j | Property filter (`group_id`) on all queries |
| FalkorDB | Separate graph per `group_id` |
| Kuzu | Separate database per `group_id` |
| Neptune | Property filter on all queries |

---

## 9. Data Flow Diagrams

### 9.1 Episode Ingestion (Full Pipeline)

```
Client POST /v1/episodes
  │
  ▼
Gateway (auth, rate-limit, validate)
  │ gRPC IngestEpisode
  ▼
Ingestion Service
  │ 1. Validate + enqueue (per group_id serialization)
  │ 2. gRPC → Knowledge.ExtractEntities(episode_content)
  │ 3. gRPC → Knowledge.ResolveEntities(extracted, group_id)
  │ 4. gRPC → Knowledge.ExtractEdges(episode, resolved_nodes)
  │ 5. gRPC → Knowledge.ResolveEdges(extracted_edges, group_id)
  │ 6. gRPC → Store.SaveBulk(nodes, edges, episode)
  │ 7. gRPC → Knowledge.UpdateCommunity(affected_entities)
  │ 8. Publish event: episode.ingested
  ▼
Response (episode_uuid, stats)
```

### 9.2 Hybrid Search

```
Client GET /v1/search?q=...
  │
  ▼
Gateway (auth, rate-limit)
  │ gRPC SearchQuery
  ▼
Search Service
  │ 1. gRPC → Knowledge.GenerateEmbedding(query)
  │ 2. Parallel:
  │    ├─ gRPC → Store.CosineSimilaritySearch(embedding)
  │    ├─ gRPC → Store.FulltextSearch(query)
  │    └─ gRPC → Store.BFSSearch(matched_nodes)
  │ 3. Merge + RRF/MMR/CrossEncoder rerank
  │    (if cross_encoder: gRPC → Knowledge.Rerank)
  │ 4. Apply temporal + property filters
  ▼
Response (SearchResults)
```

---

## 10. Migration Mapping — Python → Go

| Python Component | Go Service | Go Package |
|-----------------|-----------|-----------|
| `graphiti.py` (Graphiti class) | graphiti-ingestion | `internal/usecase/` |
| `server/graph_service/` (FastAPI) | graphiti-gateway | `internal/adapter/http/` |
| `mcp_server/` (FastMCP) | graphiti-gateway | `internal/adapter/mcp/` |
| `driver/` (GraphDriver + impls) | graphiti-store | `internal/adapter/repository/` |
| `driver/operations/` (ABCs) | graphiti-store | `internal/usecase/port/output.go` |
| `namespaces/` | graphiti-store | `internal/usecase/` |
| `search/` | graphiti-search | `internal/usecase/` |
| `llm_client/` | graphiti-knowledge | `internal/adapter/client/llm/` |
| `embedder/` | graphiti-knowledge | `internal/adapter/client/embedder/` |
| `cross_encoder/` | graphiti-knowledge | `internal/adapter/client/reranker/` |
| `prompts/` | graphiti-knowledge | `internal/domain/prompt/` |
| `utils/maintenance/` | graphiti-knowledge | `internal/usecase/` |
| `nodes.py`, `edges.py` (models) | pkg/graph | `pkg/graph/` |
| `tracer.py`, `telemetry/` | pkg/observability | `pkg/observability/` |

---

## 11. Document Index

| Document | Description |
|----------|-------------|
| [01-gateway-service.md](./01-gateway-service.md) | API Gateway — routing, auth, protocol translation |
| [02-ingestion-service.md](./02-ingestion-service.md) | Episode ingestion, pipeline orchestration, saga management |
| [03-search-service.md](./03-search-service.md) | Hybrid search, reranking, filtering |
| [04-knowledge-service.md](./04-knowledge-service.md) | LLM integration, entity extraction, resolution, community |
| [05-store-service.md](./05-store-service.md) | Graph DB abstraction, CRUD, transactions, driver implementations |
| [06-admin-service.md](./06-admin-service.md) | Tenant management, index operations, health |
| [07-shared-packages.md](./07-shared-packages.md) | Proto definitions, shared domain types, middleware |
| [08-deployment-guide.md](./08-deployment-guide.md) | Docker Compose, Kubernetes, CI/CD |
