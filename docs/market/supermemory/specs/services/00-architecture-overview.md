# 00 — Supermemory Enterprise Architecture Overview

> **Version**: 5.0.0-enterprise | **Stack**: Go 1.23+ · gRPC · REST Gateway · PostgreSQL · Redis · NATS

---

## 1. Architecture Vision

Chuyển đổi Supermemory từ **TypeScript Serverless Monolith** (Cloudflare Workers) sang **Go Microservices** với mô hình **API Gateway + Domain Services**, tuân thủ Clean Architecture cho từng service. Mục tiêu: **enterprise-grade, production-ready, self-hosted**.

### 1.1. Design Principles

| Principle | Implementation |
|-----------|----------------|
| **Clean Architecture** | Domain → UseCase → Adapter → Infra (mỗi service) |
| **Domain Driven** | Mỗi service sở hữu 1 bounded context duy nhất |
| **Gateway Pattern** | Tất cả external traffic qua API Gateway |
| **gRPC Internal** | Inter-service communication qua gRPC + Protobuf |
| **Event-Driven** | Async processing qua NATS JetStream |
| **Multi-Tenant** | org_id isolation ở mọi tầng (DB, cache, queue) |

---

## 2. High-Level Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│                           CLIENT LAYER                                     │
├──────────┬──────────┬──────────┬──────────┬──────────┬─────────────────────┤
│ AI SDKs  │ MCP      │ Browser  │ Web      │ CLI      │ Framework           │
│ (TS/Py)  │ Clients  │ Extension│ Console  │ Tools    │ Integrations        │
└────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬───────────────┘
     │          │          │          │          │          │
     ▼          ▼          ▼          ▼          ▼          ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                        API GATEWAY (Go)                                    │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │  REST (chi/echo) · Rate Limiting · JWT/API Key Auth · RBAC          │  │
│  │  Request Validation · Tenant Resolution · Observability · CORS      │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│  Routes:                                                                   │
│  │  /api/v1/documents/*  → Document Service (gRPC)                        │
│  │  /api/v1/memories/*   → Memory Service (gRPC)                          │
│  │  /api/v1/search/*     → Search Service (gRPC)                          │
│  │  /api/v1/profiles/*   → Profile Service (gRPC)                         │
│  │  /api/v1/connections/*→ Connector Service (gRPC)                       │
│  │  /api/v1/projects/*   → Project Service (gRPC)                         │
│  │  /api/v1/analytics/*  → Analytics Service (gRPC)                       │
│  │  /api/v1/auth/*       → Auth Service (gRPC)                            │
│  │  /mcp/*               → MCP Service (SSE/JSON-RPC)                     │
└────────┬───────────────────────────────────────────────────────────────────┘
         │ gRPC (internal)
         ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                        SERVICE LAYER (Go Services)                         │
├────────────┬────────────┬────────────┬────────────┬────────────┬──────────┤
│  Document  │  Memory    │  Search    │  Profile   │  Connector │  MCP     │
│  Service   │  Service   │  Service   │  Service   │  Service   │  Service │
│  :9001     │  :9002     │  :9003     │  :9004     │  :9005     │  :9006   │
├────────────┴────────────┴────────────┴────────────┴────────────┴──────────┤
│  Auth Service :9007 │ Analytics Service :9008 │ Project Service :9009     │
└─────────────────────┴────────────────────────┴───────────────────────────┘
         │                    │                     │
         ▼                    ▼                     ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                      INFRASTRUCTURE LAYER                                  │
├──────────────┬──────────────┬──────────────┬──────────────┬────────────────┤
│  PostgreSQL  │  Redis       │  NATS        │  Object      │  AI Provider   │
│  (Primary)   │  (Cache)     │  JetStream   │  Storage     │  Gateway       │
│              │              │  (Events)    │  (S3/Minio)  │  (Embedding)   │
└──────────────┴──────────────┴──────────────┴──────────────┴────────────────┘
```

---

## 3. Service Inventory

| # | Service | Port (gRPC/Health) | Bounded Context | Owner |
|---|---------|-------------------|-----------------|-------|
| 01 | **API Gateway** | 8080 / 8081 | Routing, Auth, Rate Limit | Platform |
| 02 | **Document Service** | 9001 / 9081 | Document CRUD, Ingestion Pipeline | Core |
| 03 | **Memory Service** | 9002 / 9082 | Memory Engine, Knowledge Graph, Forgetting | Core |
| 04 | **Search Service** | 9003 / 9083 | Hybrid Search, RAG, Reranking | Core |
| 05 | **Profile Service** | 9004 / 9084 | User Profiles (Static + Dynamic) | Core |
| 06 | **Connector Service** | 9005 / 9085 | External Data Sync (GDrive, Notion, OneDrive) | Integration |
| 07 | **MCP Service** | 9006 / 9086 | Model Context Protocol Server | Integration |
| 08 | **Auth Service** | 9007 / 9087 | Authentication, API Keys, RBAC, Organizations | Platform |
| 09 | **Analytics Service** | 9008 / 9088 | Usage Tracking, Token Economics, Reporting | Platform |
| 10 | **Project Service** | 9009 / 9089 | Spaces, Container Tags, Membership | Core |

---

## 4. Shared Go Packages (`pkg/`)

```
pkg/
├── proto/                    # Generated Protobuf/gRPC stubs (shared contract)
│   ├── document/v1/
│   ├── memory/v1/
│   ├── search/v1/
│   ├── profile/v1/
│   ├── connector/v1/
│   ├── mcp/v1/
│   ├── auth/v1/
│   ├── analytics/v1/
│   └── project/v1/
├── middleware/               # Shared middleware
│   ├── auth.go              # JWT + API Key validation interceptor
│   ├── tenant.go            # Multi-tenant context injection
│   ├── logging.go           # Structured logging (slog)
│   ├── tracing.go           # OpenTelemetry tracing
│   ├── metrics.go           # Prometheus metrics
│   ├── ratelimit.go         # Token bucket / sliding window
│   └── recovery.go          # Panic recovery
├── adapters/                 # Shared infrastructure adapters
│   ├── postgres/            # pgxpool wrapper, migrations
│   ├── redis/               # go-redis wrapper
│   ├── nats/                # NATS JetStream publisher/subscriber
│   ├── s3/                  # Object storage (S3/Minio)
│   ├── embedding/           # AI embedding client (OpenAI, local)
│   └── llm/                 # LLM inference client
├── domain/                   # Shared domain types
│   ├── tenant.go            # OrgID, UserID, TenantContext
│   ├── pagination.go        # Cursor/Offset pagination
│   └── errors.go            # Domain error types
├── config/                   # Shared config loading (envconfig)
├── health/                   # Health check server (HTTP)
├── testutil/                 # Test helpers, fixtures, containers
└── observability/            # OTel setup, metrics registry
```

---

## 5. Inter-Service Communication

### 5.1. Synchronous (gRPC)

```
Gateway ──gRPC──► Document Service ──gRPC──► Memory Service
                                    ──gRPC──► Search Service
                                    ──gRPC──► Profile Service

Gateway ──gRPC──► Auth Service (token validation on every request)
```

### 5.2. Asynchronous (NATS JetStream)

| Event | Publisher | Subscribers |
|-------|-----------|-------------|
| `document.created` | Document Service | Memory Service (extract facts), Search Service (index) |
| `document.deleted` | Document Service | Memory Service (cleanup), Search Service (deindex) |
| `document.processed` | Document Service | Analytics Service (log), Connector Service (status) |
| `memory.created` | Memory Service | Search Service (index), Profile Service (update) |
| `memory.forgotten` | Memory Service | Search Service (deindex), Profile Service (update) |
| `memory.relation.created` | Memory Service | Search Service (reindex graph) |
| `connection.synced` | Connector Service | Document Service (ingest batch), Analytics Service |
| `auth.api_key.used` | Auth Service | Analytics Service (track) |

### 5.3. Event Schema (Protobuf)

```protobuf
message DomainEvent {
  string   event_id    = 1;  // UUID v7
  string   event_type  = 2;  // "document.created"
  string   org_id      = 3;  // Tenant scope
  string   user_id     = 4;
  bytes    payload     = 5;  // Type-specific Protobuf bytes
  int64    timestamp   = 6;  // Unix nanos
  map<string,string> metadata = 7;
}
```

---

## 6. Data Ownership

| Service | Owned Tables | Shared Read |
|---------|-------------|-------------|
| **Document** | documents, chunks | — |
| **Memory** | memory_entries, memory_relations, memory_document_sources | documents (read) |
| **Search** | search_index (materialized) | chunks, memory_entries (read) |
| **Profile** | user_profiles, profile_snapshots | memory_entries (read) |
| **Connector** | connections, connection_states, sync_logs | documents (write via event) |
| **Auth** | users, organizations, org_members, api_keys, sessions | — |
| **Analytics** | api_requests, usage_aggregations | — |
| **Project** | spaces, space_members, container_tags, documents_to_spaces | — |

> **Design Decision**: Shared PostgreSQL instance với schema isolation per service (`document.`, `memory.`, etc.). Production-grade có thể tách physical DB per service.

---

## 7. Deployment Architecture

```
┌─ Kubernetes Cluster ─────────────────────────────────────────────────────┐
│                                                                          │
│  ┌─ Ingress (Nginx/Traefik) ──────────────────────────────────────────┐ │
│  │  TLS termination, rate limiting (L7), WAF                          │ │
│  └────────────────────┬───────────────────────────────────────────────┘ │
│                       │                                                  │
│  ┌─ API Gateway ──────▼───────────────────────────────────────────────┐ │
│  │  Deployment: 3 replicas, HPA (CPU 70%)                            │ │
│  │  Resources: 256m/512Mi (request), 1000m/1Gi (limit)               │ │
│  └────────────────────┬───────────────────────────────────────────────┘ │
│                       │ gRPC (ClusterIP)                                 │
│  ┌────────────────────▼───────────────────────────────────────────────┐ │
│  │  Service Pods (per service):                                       │ │
│  │  • Deployment: 2+ replicas, HPA                                   │ │
│  │  • Graceful shutdown (SIGTERM → drain → close)                    │ │
│  │  • Liveness: /healthz, Readiness: /readyz                        │ │
│  │  • ConfigMap + Secrets (env vars)                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌─ Stateful Services ───────────────────────────────────────────────┐ │
│  │  PostgreSQL (StatefulSet/Operator) + Redis (Sentinel)             │ │
│  │  NATS JetStream (Helm chart, 3-node cluster)                      │ │
│  │  Minio (optional, S3-compatible object storage)                   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌─ Observability ────────────────────────────────────────────────────┐ │
│  │  OpenTelemetry Collector → Jaeger (traces) + Prometheus (metrics) │ │
│  │  Grafana dashboards + Loki (logs)                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Cross-Cutting Concerns

### 8.1. Authentication Flow

```
Client → Gateway (REST)
  │
  ├── Extract: Authorization: Bearer <token>
  ├── Classify: sm_ prefix → API Key | JWT → OAuth Token
  ├── Validate: Gateway → Auth Service (gRPC: ValidateToken)
  │   └── Response: { user_id, org_id, roles[], permissions[] }
  ├── Inject: TenantContext into gRPC metadata
  └── Forward: to target service with authenticated context
```

### 8.2. Multi-Tenant Isolation

```go
// Mọi gRPC request đều mang TenantContext trong metadata
type TenantContext struct {
    OrgID       string
    UserID      string
    Roles       []string
    Permissions []string
}

// Repository layer: tự động inject WHERE org_id = $1
func (r *DocumentRepo) List(ctx context.Context, ...) {
    tenant := middleware.TenantFromCtx(ctx)
    query := `SELECT * FROM documents WHERE org_id = $1 ...`
    // org_id luôn là filter đầu tiên
}
```

### 8.3. Observability

| Layer | Tool | Implementation |
|-------|------|----------------|
| **Traces** | OpenTelemetry + Jaeger | gRPC interceptor tự động trace mọi call |
| **Metrics** | Prometheus | Request count, latency histogram, error rate per service |
| **Logs** | slog + Loki | Structured JSON logs với trace_id correlation |
| **Health** | HTTP /healthz /readyz | Per-service health check (DB, Redis, NATS) |
| **Alerts** | Grafana Alerting | SLA-based alerts (p99 > 500ms, error rate > 1%) |

### 8.4. Error Handling

```go
// Domain errors (service-agnostic)
var (
    ErrNotFound       = errors.New("entity not found")
    ErrConflict       = errors.New("entity already exists")
    ErrForbidden      = errors.New("access denied")
    ErrInvalidInput   = errors.New("invalid input")
    ErrQuotaExceeded  = errors.New("quota exceeded")
)

// gRPC error mapping (adapter layer)
func toGRPCError(err error) error {
    switch {
    case errors.Is(err, domain.ErrNotFound):    return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, domain.ErrConflict):    return status.Error(codes.AlreadyExists, err.Error())
    case errors.Is(err, domain.ErrForbidden):   return status.Error(codes.PermissionDenied, err.Error())
    case errors.Is(err, domain.ErrInvalidInput):return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, domain.ErrQuotaExceeded):return status.Error(codes.ResourceExhausted, err.Error())
    default:                                     return status.Error(codes.Internal, "internal error")
    }
}

// Gateway REST error mapping
func toHTTPError(grpcErr error) (int, ErrorResponse) {
    // codes.NotFound → 404, codes.InvalidArgument → 400, etc.
}
```

---

## 9. Technology Stack

| Component | Technology | Justification |
|-----------|-----------|---------------|
| **Language** | Go 1.23+ | Performance, concurrency, deployment simplicity |
| **HTTP Router** | chi v5 | Lightweight, idiomatic, middleware-friendly |
| **gRPC** | google.golang.org/grpc | Inter-service, code-gen, streaming |
| **Database** | PostgreSQL 16 + pgx v5 | JSONB, pgvector, partitioning, proven at scale |
| **Vector Search** | pgvector extension | Embedded vector similarity (HNSW index) |
| **Cache** | Redis 7 (Sentinel) | Session, profile cache, rate limiting |
| **Message Queue** | NATS JetStream | Lightweight, at-least-once, durable subscriptions |
| **Object Storage** | S3 / Minio | File uploads (PDF, image, video) |
| **Auth** | JWT (RS256) + API Keys | Stateless, rotatable, cacheable |
| **Migrations** | golang-migrate | Versioned, idempotent, per-service schema |
| **Config** | envconfig + YAML | 12-factor, environment-aware |
| **Observability** | OpenTelemetry + Prometheus | Vendor-neutral, K8s-native |
| **Container** | Docker + K8s | Standard enterprise deployment |
| **CI/CD** | GitHub Actions + Helm | Automated build, test, deploy |
| **DI** | Wire (google/wire) | Compile-time dependency injection |
