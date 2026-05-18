---
id: SOL-001
title: "Graphiti Monolith App — Embedded Multi-Service Process"
app: apps/graphiti
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_cr: null
approved_by: pending
---

## Yêu Cầu Gốc

Hợp nhất **5 Graphiti microservices** + **Gateway** thành **1 binary duy nhất** chạy ở mức **production grade** và **enterprise level**.

### Nguyên tắc thiết kế:
- **ZERO changes** đến code services (`services/graphiti-*`) và gateway (`gateway/`) đang có
- **Reuse toàn bộ** — Import từ services/gateway packages, chỉ viết orchestration code
- **Internal gRPC/NATS OK** — Các module giao tiếp qua gRPC localhost + NATS như thiết kế gốc
- **Single Process** — Mọi thứ chạy trong 1 process, deploy 1 binary
- **Minimal Code** — Ước tính ≤500 lines code mới (chỉ orchestration + config)

## Phân Tích Kiến Trúc

### Inventory — Graphiti Services Hiện Tại

| Service | gRPC Port | Bounded Context | Dependencies |
|---------|-----------|-----------------|--------------|
| `graphiti-ingestion` | 9021 | Episode Lifecycle, Pipeline Orchestration | → knowledge, store |
| `graphiti-search` | 9022 | Hybrid Search, Reranking, Filtering | → knowledge, store |
| `graphiti-knowledge` | 9023 | LLM/Embedding, Entity/Edge Extraction, Resolution | → store |
| `graphiti-store` | 9024 | Graph DB Abstraction (Neo4j), CRUD, Transactions | — |
| `graphiti-pipeline` | 9025 | Pipeline Orchestration | → ingestion, knowledge, store |
| `gateway` (shared) | 8080 (REST), 8082 (MCP) | API Routing, Auth, Protocol Translation | → all services |

### Cách hoạt động hiện tại (Microservices)

```
Client → [Gateway:8080] →─gRPC──→ [graphiti-ingestion:9021]   (separate process)
                         →─gRPC──→ [graphiti-search:9022]      (separate process)
                         →─gRPC──→ [graphiti-knowledge:9023]   (separate process)
                         →─gRPC──→ [graphiti-store:9024]       (separate process)
                         →─gRPC──→ [graphiti-pipeline:9025]    (separate process)
```

### Cách hoạt động mới (Monolith — 1 process)

```
                    ┌──── SINGLE PROCESS ──────────────────────────────────────┐
                    │                                                          │
Client → [Gateway REST :8080] →─gRPC localhost──→ [ingestion  goroutine:9021]  │
                    │          →─gRPC localhost──→ [search     goroutine:9022]  │
                    │          →─gRPC localhost──→ [knowledge  goroutine:9023]  │
                    │          →─gRPC localhost──→ [store      goroutine:9024]  │
                    │          →─gRPC localhost──→ [pipeline   goroutine:9025]  │
                    │          →─NATS────────────→ [async events]               │
                    │                                                          │
                    │  [MCP :8082] [Health :9090] [Metrics :9091]               │
                    └──────────────────────────────────────────────────────────┘
```

**Khác biệt duy nhất:** Tất cả services + gateway chạy trong 1 process thay vì 6+ processes.
- gRPC vẫn hoạt động → gọi qua `localhost` (zero-copy loopback, ~0.1ms overhead)
- NATS vẫn hoạt động → event-driven giữ nguyên
- **KHÔNG thay đổi bất kỳ code nào** trong services hay gateway

### Inter-Service Communication (giữ nguyên)

```
Gateway → tất cả services (fan-out via gRPC localhost)
Ingestion → Knowledge (extract, resolve) → Store (persist)
Search → Knowledge (rerank via cross-encoder) → Store (query)
Pipeline → Ingestion, Knowledge, Store
NATS events: episode.ingested, entity.resolved, community.rebuilt, tenant.created
```

### Tại sao approach này tốt nhất?

| Tiêu chí | Ưu điểm |
|----------|---------|
| **Zero code change** | Services + Gateway giữ nguyên 100%, không sửa 1 dòng nào |
| **Proven patterns** | gRPC localhost + NATS đã được test kỹ trong production |
| **Go `internal/` safe** | App KHÔNG import `internal/` packages (Go sẽ reject). Thay vào đó, replicate init logic |
| **Rollback trivial** | Chỉ cần deploy lại microservices riêng, code không bị ảnh hưởng |
| **Upgrade safe** | Khi services update, app tự động nhận code mới (same Go module) |
| **Same pattern as Cognee** | Đã triển khai thành công cho `apps/cognee`, reuse pattern đã prove |

### Code inventory: Cái gì KHÔNG thay đổi

```
✅ gateway/cmd/main.go                                — KHÔNG SỬA
✅ gateway/internal/**                                 — KHÔNG SỬA
✅ services/graphiti-ingestion/cmd/server/main.go      — KHÔNG SỬA
✅ services/graphiti-ingestion/internal/**              — KHÔNG SỬA
✅ services/graphiti-search/cmd/server/main.go         — KHÔNG SỬA
✅ services/graphiti-search/internal/**                — KHÔNG SỬA
✅ services/graphiti-knowledge/cmd/server/main.go      — KHÔNG SỬA
✅ services/graphiti-knowledge/internal/**             — KHÔNG SỬA
✅ services/graphiti-store/cmd/server/main.go          — KHÔNG SỬA
✅ services/graphiti-store/internal/**                 — KHÔNG SỬA
✅ services/graphiti-pipeline/cmd/server/main.go       — KHÔNG SỬA
✅ services/graphiti-pipeline/internal/**              — KHÔNG SỬA
```

### Code CẦN viết mới (chỉ orchestration)

```
🆕 apps/graphiti/cmd/graphiti/main.go              — Orchestrator: start all services + gateway
🆕 apps/graphiti/internal/supervisor/supervisor.go  — Process supervisor (start, health, shutdown)
🆕 apps/graphiti/internal/config/config.go          — Unified config for all embedded services
🆕 apps/graphiti/internal/embed/services.go         — Embedded service startup functions
🆕 apps/graphiti/internal/embed/gateway.go          — Embedded gateway startup function
🆕 apps/graphiti/internal/health/server.go          — Aggregated health probe server
🆕 apps/graphiti/config.yaml                        — Unified config file
🆕 apps/graphiti/.env.example                       — Environment variable reference
🆕 apps/graphiti/Dockerfile                         — Multi-stage build
🆕 apps/graphiti/Makefile                           — Build/run/test targets
🆕 apps/graphiti/docker-compose.yml                 — Local dev (Neo4j, Redis, NATS)
```

**Ước tính code mới: ~450 lines** (so với ~15,000+ lines reuse)

## Giải Pháp Chi Tiết

### Approach: Embedded Service Supervisor

App `graphiti` là **process supervisor** chạy trong 1 Go process:
1. Load unified config → phân phối thành ENV vars cho từng embedded service
2. Start mỗi service `main()` logic trong goroutine riêng (ordered: store → knowledge → search/ingestion/pipeline)
3. Start gateway trong goroutine riêng, config services map → `localhost:PORT`
4. Monitor health, aggregate metrics
5. Handle SIGTERM → graceful shutdown all goroutines (reverse order: gateway → services)

### Startup Order (Dependency-aware)

```
Phase 1: Data Layer
  └── graphiti-store        (9024) — Depends on: Neo4j
      Wait: gRPC health check ready

Phase 2: Intelligence Layer
  └── graphiti-knowledge    (9023) — Depends on: store, LLM provider
      Wait: gRPC health check ready

Phase 3: Application Layer (parallel)
  ├── graphiti-ingestion    (9021) — Depends on: knowledge, store
  ├── graphiti-search       (9022) — Depends on: knowledge, store
  └── graphiti-pipeline     (9025) — Depends on: ingestion, knowledge, store
      Wait: all gRPC health checks ready

Phase 4: Gateway
  └── vnp-gateway           (8080) — Depends on: all services ready
      Wait: REST + MCP servers listening
```

### Shutdown Order (Reverse)

```
1. Gateway (stop accepting new requests, drain in-flight)
2. Ingestion + Search + Pipeline (parallel, drain queues)
3. Knowledge (stop LLM calls)
4. Store (close DB connections)
```

### Supervisor Pattern

```go
// internal/supervisor/supervisor.go
package supervisor

type ServiceSpec struct {
    Name     string
    StartFn  func(ctx context.Context) error  // Each service's main() logic
    Port     int                               // gRPC port
    Phase    int                               // Startup phase (1-4)
}

type Supervisor struct {
    services []ServiceSpec
    logger   *slog.Logger
    wg       sync.WaitGroup
}

func New(logger *slog.Logger) *Supervisor

func (s *Supervisor) Register(spec ServiceSpec)

// StartAll launches services in phase order, waits for each phase health check
func (s *Supervisor) StartAll(ctx context.Context) error

// Graceful shutdown: reverse phase order, signal all services, wait with timeout
func (s *Supervisor) Shutdown(ctx context.Context) error

// HealthCheck returns aggregated health of all embedded services
func (s *Supervisor) HealthCheck() map[string]bool
```

### Config Unification

```go
// internal/config/config.go
type Config struct {
    // Shared infra
    Neo4jURI    string `env:"NEO4J_URI" default:"bolt://localhost:7687"`
    Neo4jUser   string `env:"NEO4J_USERNAME" default:"neo4j"`
    Neo4jPass   string `env:"NEO4J_PASSWORD"`
    RedisAddr   string `env:"REDIS_ADDR" default:"localhost:6379"`
    NATSURL     string `env:"NATS_URL" default:"nats://localhost:4222"`

    // LLM/AI
    LLMProvider    string `env:"LLM_PROVIDER" default:"openai"`
    LLMModel       string `env:"LLM_MODEL" default:"gpt-4o"`
    LLMSmallModel  string `env:"LLM_SMALL_MODEL" default:"gpt-4o-mini"`
    LLMAPIKey      string `env:"LLM_API_KEY"`
    EmbedderModel  string `env:"EMBEDDER_MODEL" default:"text-embedding-3-small"`

    // Service Ports
    IngestionGRPCPort int `env:"INGESTION_GRPC_PORT" default:"9021"`
    SearchGRPCPort    int `env:"SEARCH_GRPC_PORT" default:"9022"`
    KnowledgeGRPCPort int `env:"KNOWLEDGE_GRPC_PORT" default:"9023"`
    StoreGRPCPort     int `env:"STORE_GRPC_PORT" default:"9024"`
    PipelineGRPCPort  int `env:"PIPELINE_GRPC_PORT" default:"9025"`

    // Gateway
    GatewayRESTPort int  `env:"GATEWAY_REST_PORT" default:"8080"`
    GatewayMCPPort  int  `env:"GATEWAY_MCP_PORT" default:"8082"`
    HealthPort      int  `env:"HEALTH_PORT" default:"9090"`
    AuthDevMode     bool `env:"AUTH_DEV_MODE" default:"true"`
    LogLevel        string `env:"LOG_LEVEL" default:"info"`

    // OTel
    OTelEndpoint string `env:"OTEL_ENDPOINT"`
}

// SetServiceEnvVars exports config as ENV vars
// để các service đọc qua os.Getenv() (pattern đã có sẵn)
func (c *Config) SetServiceEnvVars()

// GatewayServicesMap returns services→localhost:PORT map cho GRPCRegistry
func (c *Config) GatewayServicesMap() map[string]string {
    return map[string]string{
        "graphiti-ingestion": fmt.Sprintf("localhost:%d", c.IngestionGRPCPort),
        "graphiti-search":    fmt.Sprintf("localhost:%d", c.SearchGRPCPort),
        "graphiti-knowledge": fmt.Sprintf("localhost:%d", c.KnowledgeGRPCPort),
        "graphiti-store":     fmt.Sprintf("localhost:%d", c.StoreGRPCPort),
        "graphiti-pipeline":  fmt.Sprintf("localhost:%d", c.PipelineGRPCPort),
    }
}
```

### Internal Communication Flow

```
┌────────────────────────────────────────────────────────────┐
│                    graphiti-app (single process)              │
│                                                              │
│  Client HTTP/MCP                                             │
│    │                                                         │
│    ▼                                                         │
│  Gateway (goroutine :8080 REST, :8082 MCP)                   │
│    │ GRPCRegistry("graphiti-ingestion" → "localhost:9021")   │
│    │                                                         │
│    ▼ gRPC via localhost loopback                              │
│  graphiti-store (goroutine :9024)                             │
│    │ Neo4j CRUD, search, transactions                        │
│    │                                                         │
│  graphiti-knowledge (goroutine :9023)                         │
│    │ LLM calls, entity extraction, embedding                 │
│    │ → store (gRPC localhost:9024)                            │
│    │                                                         │
│  graphiti-ingestion (goroutine :9021)                         │
│    │ Pipeline orchestrator                                   │
│    │ → knowledge (gRPC localhost:9023)                        │
│    │ → store (gRPC localhost:9024)                            │
│    │ PublishEvent → NATS JetStream                           │
│    │                                                         │
│  graphiti-search (goroutine :9022)                            │
│    │ Hybrid search, reranking                                │
│    │ → knowledge (gRPC localhost:9023)                        │
│    │ → store (gRPC localhost:9024)                            │
│    │                                                         │
│  graphiti-pipeline (goroutine :9025)                          │
│    │ Pipeline coordination                                   │
│                                                              │
│  External dependencies:                                      │
│  ├─ Neo4j (graph database)                                   │
│  ├─ Redis (cache, rate limiting)                             │
│  ├─ NATS JetStream (async events)                            │
│  └─ LLM Provider (OpenAI/Anthropic/Gemini via Bifrost)       │
└──────────────────────────────────────────────────────────────┘
```

### Alternatives Đã Xem Xét

| Alternative | Lý do loại bỏ |
|---|---|
| Import `internal/` packages trực tiếp | Go compiler reject cross-module `internal/` imports |
| Sửa services để export public packages | Vi phạm yêu cầu "KHÔNG sửa code services" |
| Replace NATS bằng in-process event bus | Phải sửa services code (EventPublisher interface) |
| Copy code từ services | Duplicate code, maintenance nightmare |
| Replace gRPC bằng in-process function calls | Phải sửa services, mất inter-service contract safety |

### Trade-offs

| Ưu điểm | Nhược điểm |
|----------|------------|
| Zero code changes | gRPC localhost có overhead nhỏ (~0.1ms vs in-process) |
| Proven gRPC/NATS paths | Cần NATS server chạy (external dependency) |
| Services update = app update | Startup thứ tự phải đúng (services trước gateway) |
| Single binary deploy | Memory footprint lớn hơn single microservice |
| Unified config | Config mapping: app config → service ENV vars |
| Same pattern as cognee app | Pattern đã prove, giảm risk |

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện

```
TASK-001: Project setup + Go module + unified config   ← Foundation
TASK-002: Service supervisor (goroutine lifecycle)      ← Sau TASK-001
TASK-003: Embed graphiti services (5 services)          ← Sau TASK-002
TASK-004: Embed gateway (config override localhost)     ← Sau TASK-003
TASK-005: Health aggregation + main.go entry point      ← Sau TASK-004
TASK-006: Dockerfile + Makefile + docker-compose        ← Sau TASK-005
TASK-007: Documentation (architecture, runbook)         ← Sau TASK-006
```

### Danh Sách Tác Vụ

| ID | Tên Task | Phụ thuộc | Ước tính | Mô tả |
|---|---|---|---|---|
| T01 | Go module + unified config | - | 1.5h | go.mod, config struct, ENV var mapping, config.yaml, .env.example |
| T02 | Service supervisor | T01 | 2h | Goroutine lifecycle manager, phase-ordered startup/shutdown, WaitForReady |
| T03 | Embed graphiti services | T02 | 3h | Replicate 5 service main() as StartFn, NO code changes to services |
| T04 | Embed gateway | T03 | 2h | Override services map → localhost:PORT, start REST + MCP servers |
| T05 | Health aggregation + main.go | T04 | 1.5h | Aggregated /healthz + /readyz, main.go entry point with signal handling |
| T06 | Dockerfile + Makefile + docker-compose | T05 | 2h | Multi-stage Dockerfile, build/run/test Makefile targets, dev compose |
| T07 | Documentation | T06 | 1h | Architecture docs, API reference, runbook, changelog |

**Tổng ước tính: ~13h** (giảm nhờ zero new business logic, pattern reuse từ cognee)

### Rollback Plan

Monolith app nằm trong `apps/graphiti/`. Services gốc KHÔNG bị sửa đổi.
Rollback = deploy microservices + gateway riêng biệt như cũ.

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: `go build ./cmd/graphiti/` build thành công, 1 binary duy nhất
- [ ] SOL-AC-2: Binary khởi động tất cả 5 graphiti services + gateway trong 1 process
- [ ] SOL-AC-3: REST API qua gateway hoạt động đúng (POST /v1/episodes, POST /v1/search, etc.)
- [ ] SOL-AC-4: MCP protocol qua gateway hoạt động đúng (add_memory, search_memory tools)
- [ ] SOL-AC-5: gRPC localhost communication hoạt động giữa gateway ↔ services
- [ ] SOL-AC-6: gRPC localhost communication hoạt động giữa services (ingestion → knowledge → store)
- [ ] SOL-AC-7: NATS events hoạt động (episode.ingested, entity.resolved, etc.)
- [ ] SOL-AC-8: Graceful shutdown tất cả goroutines (ordered: gateway → app services → knowledge → store)
- [ ] SOL-AC-9: Health checks aggregated từ tất cả embedded services (/healthz, /readyz)
- [ ] SOL-AC-10: **ZERO lines changed** trong `services/` và `gateway/` directories
- [ ] SOL-AC-11: Docker image build thành công, single binary
- [ ] SOL-AC-12: Code mới ≤ 500 lines (chỉ orchestration + config)
