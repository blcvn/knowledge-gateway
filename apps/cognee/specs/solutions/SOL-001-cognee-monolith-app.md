---
id: SOL-001
title: "Cognee Monolith App — Embedded Multi-Service Process"
app: apps/cognee
version: 3.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_cr: null
approved_by: pending
---

## Yêu Cầu Gốc

Hợp nhất **4 Cognee microservices** + **Gateway** thành **1 binary duy nhất** chạy ở mức **production grade** và **enterprise level**.

### Nguyên tắc thiết kế:
- **ZERO changes** đến code services và gateway đang có
- **Reuse toàn bộ** — Import từ services/gateway packages, chỉ viết orchestration code
- **Internal gRPC/NATS OK** — Các module giao tiếp qua gRPC localhost + NATS như thiết kế gốc
- **Single Process** — Mọi thứ chạy trong 1 process, deploy 1 binary

## Phân Tích Kiến Trúc

### Cách hoạt động hiện tại (Microservices)

```
Client → [Gateway:8080] →─gRPC──→ [cognee-ingestion:9011]  (separate process)
                         →─gRPC──→ [cognee-cognify:9012]   (separate process)
                         →─gRPC──→ [cognee-search:9013]    (separate process)
                         →─NATS──→ [cognee-memory:9014]    (separate process)
```

Mỗi service là 1 process riêng, gateway route qua gRPC, async events qua NATS.

### Cách hoạt động mới (Monolith — 1 process)

```
                    ┌──── SINGLE PROCESS ─────────────────────────────────────┐
                    │                                                         │
Client → [Gateway REST :8080] →─gRPC localhost──→ [ingestion  goroutine:9011] │
                    │          →─gRPC localhost──→ [cognify    goroutine:9012] │
                    │          →─gRPC localhost──→ [search     goroutine:9013] │
                    │          →─NATS────────────→ [async events]              │
                    │                                                         │
                    │  [Health :9090] [Metrics :9091]                          │
                    └─────────────────────────────────────────────────────────┘
```

**Khác biệt duy nhất:** Tất cả services + gateway chạy trong 1 process thay vì 5 processes.
- gRPC vẫn hoạt động → gọi qua `localhost` (zero-copy loopback, ~0.1ms overhead)
- NATS vẫn hoạt động → event-driven giữ nguyên
- **KHÔNG thay đổi bất kỳ code nào** trong services hay gateway

### Tại sao approach này tốt nhất?

| Tiêu chí | Ưu điểm |
|----------|---------|
| **Zero code change** | Services + Gateway giữ nguyên 100%, không sửa 1 dòng nào |
| **Proven patterns** | gRPC localhost + NATS đã được test kỹ trong production |
| **Go `internal/` safe** | App KHÔNG import `internal/` packages (Go sẽ reject). Thay vào đó, embed từng service bằng cách gọi init functions |
| **Rollback trivial** | Chỉ cần deploy lại microservices riêng, code không bị ảnh hưởng |
| **Upgrade safe** | Khi services update, app tự động nhận code mới (same Go module) |

### Code inventory: Cái gì KHÔNG thay đổi

```
✅ gateway/cmd/main.go                         — KHÔNG SỬA (extract init logic thành callable func)
✅ gateway/internal/**                          — KHÔNG SỬA (import as-is)
✅ services/cognee-ingestion/cmd/server/main.go — KHÔNG SỬA (extract init logic)
✅ services/cognee-ingestion/internal/**        — KHÔNG SỬA
✅ services/cognee-cognify/cmd/server/main.go   — KHÔNG SỬA
✅ services/cognee-cognify/internal/**          — KHÔNG SỬA
✅ services/cognee-search/cmd/server/main.go    — KHÔNG SỬA
✅ services/cognee-search/internal/**           — KHÔNG SỬA
```

### Code CẦN viết mới (chỉ orchestration)

```
🆕 apps/cognee/cmd/cognee/main.go              — Orchestrator: start all services + gateway as goroutines
🆕 apps/cognee/internal/supervisor/supervisor.go — Process supervisor (start, health check, shutdown)
🆕 apps/cognee/internal/config/config.go        — Unified config for all embedded services
🆕 apps/cognee/Dockerfile                       — Multi-stage build
🆕 apps/cognee/Makefile                         — Build/run targets
🆕 apps/cognee/docker-compose.yml               — Local dev (NATS, Postgres, Neo4j, Qdrant)
🆕 apps/cognee/config.yaml                      — Unified config file
```

**Ước tính code mới: ~400 lines** (so với ~8,000+ lines reuse)

## Giải Pháp Chi Tiết

### Approach: Embedded Service Supervisor

App `cognee` là **process supervisor** chạy trong 1 Go process:
1. Load unified config → phân phối thành ENV vars cho từng embedded service
2. Start mỗi service `main()` logic trong goroutine riêng
3. Start gateway trong goroutine riêng, config services map → `localhost:PORT`
4. Monitor health, aggregate metrics
5. Handle SIGTERM → graceful shutdown all goroutines

### Supervisor Pattern

```go
// internal/supervisor/supervisor.go
package supervisor

type ServiceSpec struct {
    Name     string
    StartFn  func(ctx context.Context) error  // Each service's main() logic
    Port     int                               // gRPC port
}

type Supervisor struct {
    services []ServiceSpec
    logger   *slog.Logger
    wg       sync.WaitGroup
}

func New(logger *slog.Logger) *Supervisor

func (s *Supervisor) Register(spec ServiceSpec)

// StartAll launches all services as goroutines, returns when ctx is cancelled
func (s *Supervisor) StartAll(ctx context.Context) error

// Graceful shutdown: signal all services, wait with timeout
func (s *Supervisor) Shutdown(ctx context.Context) error

// HealthCheck returns aggregated health of all embedded services
func (s *Supervisor) HealthCheck() map[string]bool
```

### main.go Pattern

```go
// cmd/cognee/main.go
func main() {
    cfg := config.Load()
    logger := setupLogger(cfg)

    // Set ENV vars cho các embedded services
    os.Setenv("DATABASE_URL", cfg.DatabaseURL)
    os.Setenv("NATS_URL", cfg.NATSURL)
    os.Setenv("NEO4J_URI", cfg.Neo4jURI)
    os.Setenv("QDRANT_URL", cfg.QdrantURL)
    // ... other env vars

    sv := supervisor.New(logger)

    // Register cognee services (sử dụng logic từ services/*/cmd/server/main.go)
    sv.Register(supervisor.ServiceSpec{
        Name: "cognee-ingestion",
        Port: 9011,
        StartFn: func(ctx context.Context) error {
            // Replicate the startup logic from
            // services/cognee-ingestion/cmd/server/main.go
            // WITHOUT modifying that file
            return startIngestionService(ctx, cfg)
        },
    })

    sv.Register(supervisor.ServiceSpec{
        Name: "cognee-cognify",
        Port: 9012,
        StartFn: startCognifyService,
    })

    sv.Register(supervisor.ServiceSpec{
        Name: "cognee-search",
        Port: 9013,
        StartFn: startSearchService,
    })

    // Register gateway (sử dụng logic từ gateway/cmd/main.go)
    sv.Register(supervisor.ServiceSpec{
        Name: "vnp-gateway",
        Port: 8080,
        StartFn: func(ctx context.Context) error {
            // Config gateway to route to localhost:PORT
            return startGateway(ctx, cfg)
        },
    })

    // Start all
    ctx, cancel := signal.NotifyContext(context.Background(),
        os.Interrupt, syscall.SIGTERM)
    defer cancel()

    if err := sv.StartAll(ctx); err != nil {
        logger.Error("supervisor error", "error", err)
        os.Exit(1)
    }

    sv.Shutdown(context.Background())
}
```

### Config Unification

```go
// internal/config/config.go
type Config struct {
    // Shared infra
    DatabaseURL string `env:"DATABASE_URL"`
    NATSURL     string `env:"NATS_URL"`
    Neo4jURI    string `env:"NEO4J_URI"`
    Neo4jUser   string `env:"NEO4J_USERNAME" default:"neo4j"`
    Neo4jPass   string `env:"NEO4J_PASSWORD"`
    QdrantURL   string `env:"QDRANT_URL"`
    RedisAddr   string `env:"REDIS_ADDR"`
    MinIOEndpoint string `env:"MINIO_ENDPOINT"`
    MinIOAccess   string `env:"MINIO_ACCESS_KEY"`
    MinIOSecret   string `env:"MINIO_SECRET_KEY"`

    // LLM
    LLMGatewayURL string `env:"LLM_GATEWAY_URL"`

    // Ports
    GatewayRESTPort  int `env:"GATEWAY_REST_PORT" default:"8080"`
    IngestionGRPCPort int `env:"INGESTION_GRPC_PORT" default:"9011"`
    CognifyGRPCPort   int `env:"COGNIFY_GRPC_PORT" default:"9012"`
    SearchGRPCPort    int `env:"SEARCH_GRPC_PORT" default:"9013"`
    HealthPort        int `env:"HEALTH_PORT" default:"9090"`

    // Gateway overrides
    AuthDevMode bool   `env:"AUTH_DEV_MODE" default:"true"`
    LogLevel    string `env:"LOG_LEVEL" default:"info"`

    // OTel
    OTelEndpoint string `env:"OTEL_ENDPOINT"`
}

// SetServiceEnvVars exports config as ENV vars
// để các service đọc qua os.Getenv() (pattern đã có sẵn)
func (c *Config) SetServiceEnvVars()

// GatewayServicesMap returns services→localhost:PORT map
// cho GRPCRegistry
func (c *Config) GatewayServicesMap() map[string]string {
    return map[string]string{
        "cognee-ingestion": fmt.Sprintf("localhost:%d", c.IngestionGRPCPort),
        "cognee-cognify":   fmt.Sprintf("localhost:%d", c.CognifyGRPCPort),
        "cognee-search":    fmt.Sprintf("localhost:%d", c.SearchGRPCPort),
    }
}
```

### Service Startup Functions

Mỗi service có `startXxxService()` function trong app, replicate logic từ
`services/*/cmd/server/main.go` **mà không sửa file gốc**:

```go
// cmd/cognee/services.go

func startIngestionService(ctx context.Context, cfg *config.Config) error {
    // Replicate: services/cognee-ingestion/cmd/server/main.go lines 21-85
    // 1. Init OTel
    shutdownTracer, _ := telemetry.InitProvider(ctx, "cognee-ingestion")
    defer shutdownTracer(context.Background())

    // 2. gRPC server + interceptors
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
    )

    // 3. Health probes
    healthCheck := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

    // 4. Listen on configured port
    lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", cfg.IngestionGRPCPort))
    healthCheck.SetServingStatus("cognee-ingestion", grpc_health_v1.HealthCheckResponse_SERVING)

    // 5. Serve (blocks until ctx cancelled)
    go grpcServer.Serve(lis)
    <-ctx.Done()
    grpcServer.GracefulStop()
    return nil
}

// startCognifyService, startSearchService — tương tự
// startGateway — replicate gateway/cmd/main.go logic, override services map
```

### Internal Communication Flow (giữ nguyên gRPC + NATS)

```
┌─────────────────────────────────────────────────────────┐
│                    cognee-app (single process)            │
│                                                          │
│  Client HTTP                                             │
│    │                                                     │
│    ▼                                                     │
│  Gateway (goroutine :8080)                               │
│    │ GRPCRegistry("cognee-ingestion" → "localhost:9011") │
│    │                                                     │
│    ▼ gRPC via localhost loopback                          │
│  cognee-ingestion (goroutine :9011)                      │
│    │ PublishDataIngested → NATS JetStream                │
│    │                                                     │
│    ▼ NATS subscription                                   │
│  cognee-cognify (goroutine :9012)                        │
│    │ runs 8-stage pipeline                               │
│    │ → Neo4j (external), Qdrant (external)               │
│    │                                                     │
│  cognee-search (goroutine :9013)                         │
│    │ query Neo4j + Qdrant                                │
│                                                          │
│  External dependencies:                                  │
│  ├─ PostgreSQL (shared connection)                       │
│  ├─ Neo4j (shared connection)                            │
│  ├─ Qdrant (shared connection)                           │
│  ├─ Redis (rate limiting)                                │
│  ├─ MinIO/S3 (file storage)                              │
│  └─ NATS JetStream (async events)                        │
└─────────────────────────────────────────────────────────┘
```

### Alternatives Đã Xem Xét

| Alternative | Lý do loại bỏ |
|---|---|
| Import `internal/` packages trực tiếp | Go compiler reject cross-module `internal/` imports |
| Sửa services để export public packages | Vi phạm yêu cầu "KHÔNG sửa code services" |
| Replace NATS bằng in-process event bus | Phải sửa services code (EventPublisher interface) |
| Copy code từ services | Duplicate code, maintenance nightmare |

### Trade-offs

| Ưu điểm | Nhược điểm |
|----------|------------|
| Zero code changes | gRPC localhost có overhead nhỏ (~0.1ms vs in-process) |
| Proven gRPC/NATS paths | Cần NATS server chạy (external dependency) |
| Services update = app update | Startup thứ tự phải đúng (services trước gateway) |
| Single binary deploy | Memory footprint lớn hơn single microservice |
| Unified config | Config mapping: app config → service ENV vars |

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện

```
TASK-001: Project setup + unified config           ← Foundation
TASK-002: Service supervisor (goroutine lifecycle)  ← Sau TASK-001
TASK-003: Embed services (ingestion, cognify, search) ← Sau TASK-002
TASK-004: Embed gateway (config override localhost) ← Sau TASK-003
TASK-005: Health aggregation + main.go             ← Sau TASK-004
TASK-006: Dockerfile + Makefile + docker-compose   ← Sau TASK-005
TASK-007: Service documentation                    ← Sau TASK-006
```

### Danh Sách Tác Vụ

| ID | Tên Task | Phụ thuộc | Ước tính | Mô tả |
|---|---|---|---|---|
| T01 | Go module + unified config | - | 1.5h | go.mod, config struct, ENV var mapping |
| T02 | Service supervisor | T01 | 2h | Goroutine lifecycle manager, ordered startup/shutdown |
| T03 | Embed cognee services | T02 | 3h | Replicate 3 service main() as StartFn, NO code changes |
| T04 | Embed gateway | T03 | 2h | Override services map → localhost:PORT |
| T05 | Health aggregation + main.go | T04 | 1.5h | Aggregated /healthz, main.go entry point |
| T06 | Dockerfile + Makefile + docker-compose | T05 | 2h | Build, run, test targets |
| T07 | Documentation | T06 | 1h | Architecture, API, runbook docs |

**Tổng ước tính: ~13h** (giảm nhờ zero new business logic)

### Rollback Plan
Monolith app nằm trong `apps/cognee/`. Services gốc KHÔNG bị sửa đổi.
Rollback = deploy microservices + gateway riêng biệt như cũ.

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: `go build ./cmd/cognee/` build thành công, 1 binary duy nhất
- [ ] SOL-AC-2: Binary khởi động tất cả 3 cognee services + gateway trong 1 process
- [ ] SOL-AC-3: REST API qua gateway hoạt động đúng (POST /v1/cognee/*)
- [ ] SOL-AC-4: gRPC localhost communication hoạt động giữa gateway và services
- [ ] SOL-AC-5: NATS events hoạt động (ingestion → cognify pipeline)
- [ ] SOL-AC-6: Graceful shutdown tất cả goroutines (ordered: gateway → services)
- [ ] SOL-AC-7: Health checks aggregated từ tất cả embedded services
- [ ] SOL-AC-8: **ZERO lines changed** trong services/ và gateway/ directories
- [ ] SOL-AC-9: Docker image build thành công, single binary
- [ ] SOL-AC-10: Code mới ≤ 500 lines (chỉ orchestration + config)
