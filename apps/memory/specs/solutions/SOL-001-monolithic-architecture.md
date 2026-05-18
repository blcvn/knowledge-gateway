---
id: SOL-001
title: Monolithic App Architecture — VNP Memory Unified Binary
version: 1.0.0
status: Proposed
priority: P0
created: 2026-05-14
linked_prd: docs/product/PRD.md
linked_urd: docs/product/URD.md
linked_arch: specs/architecture/00-architecture-overview.md
---

# SOL-001: Monolithic App Architecture

## 1. Tổng Quan

Xây dựng **single Go binary** (`apps/memory`) hợp nhất toàn bộ 35 domain services + gateway thành **1 process** trong khi **tái sử dụng 100%** code hiện có trong `gateway/` và `services/` mà không sửa đổi.

### Motivation

| Vấn Đề | Giải Pháp |
|---------|-----------|
| 35+ binaries phức tạp cho dev/staging | 1 binary, 1 container |
| gRPC network overhead giữa services | In-process function calls |
| Docker Compose 40+ containers cho dev | `go run ./apps/memory` |
| Khó debug cross-service flows | Single process, unified logs |
| Startup time 30-60s (35 containers) | ~3s (single binary) |

### Ràng buộc

- ❌ **KHÔNG** sửa đổi bất kỳ code nào trong `gateway/` và `services/`
- ✅ Import và tái sử dụng trực tiếp các package internal
- ✅ Sử dụng gRPC in-process (bufconn) hoặc NATS embedded cho inter-module communication
- ✅ Tất cả 35 gRPC services chạy trong cùng 1 process

---

## 2. Kiến Trúc Tổng Thể

```
                     External Clients
                          │
                   ┌──────┴──────┐
                   │  apps/memory │  ← Single Go Binary
                   │  (main.go)  │
                   └──────┬──────┘
                          │
           ┌──────────────┼──────────────┐
           │              │              │
     ┌─────▼─────┐ ┌─────▼─────┐ ┌─────▼─────┐
     │   REST     │ │    MCP    │ │  Health/   │
     │  :8080     │ │  :8082   │ │  Metrics   │
     │ (chi/v5)   │ │ (SSE)    │ │  :8083     │
     └─────┬─────┘ └─────┬─────┘ └───────────┘
           │              │
           └──────┬───────┘
                  │
     ┌────────────▼────────────────┐
     │   GATEWAY (reuse gateway/)  │
     │   Auth · RateLimit · Route  │
     └────────────┬────────────────┘
                  │ In-Process gRPC (bufconn)
     ┌────────────▼────────────────────────────────────┐
     │              SERVICE BUS (In-Process)            │
     │                                                  │
     │  ┌─────────┐ ┌─────────┐ ┌─────────┐           │
     │  │ Cognee  │ │Graphiti │ │Memobase │           │
     │  │ 3 svcs  │ │ 4 svcs  │ │ 3 svcs  │           │
     │  └────┬────┘ └────┬────┘ └────┬────┘           │
     │       │            │           │                 │
     │  ┌────▼────┐ ┌────▼────┐ ┌────▼────┐           │
     │  │OpenVikg │ │  Zep    │ │  Super  │           │
     │  │ 6 svcs  │ │ 6 svcs  │ │ memory  │           │
     │  └────┬────┘ └────┬────┘ │ 9 svcs  │           │
     │       │            │      └────┬────┘           │
     │  ┌────▼────────────▼───────────▼────┐           │
     │  │         Platform Services        │           │
     │  │  vnp-admin · vnp-event · hub     │           │
     │  └──────────────────────────────────┘           │
     └─────────────────────┬───────────────────────────┘
                           │
              ┌────────────▼────────────┐
              │  NATS Embedded / gRPC   │
              │  In-Process Messaging   │
              └────────────┬────────────┘
                           │
     ┌─────────────────────▼─────────────────────────┐
     │              SHARED INFRASTRUCTURE              │
     │  PostgreSQL · Neo4j · Qdrant · Redis            │
     │  MinIO · Bifrost(LLM)                           │
     └─────────────────────────────────────────────────┘
```

---

## 3. Chiến Lược Tái Sử Dụng Code

### 3.1 Import Strategy

Vì tất cả services nằm trong cùng monorepo và sử dụng Go modules, ta import trực tiếp internal packages:

```go
// apps/memory/main.go
import (
    // Gateway layer — reuse 100%
    gwConfig  "github.com/vnp-community/vnp-memory/gateway/internal/infra/config"
    gwHandler "github.com/vnp-community/vnp-memory/gateway/internal/adapter/handler"
    gwClient  "github.com/vnp-community/vnp-memory/gateway/internal/adapter/client"
    gwMCP     "github.com/vnp-community/vnp-memory/gateway/internal/adapter/mcp"
    gwUsecase "github.com/vnp-community/vnp-memory/gateway/internal/usecase"

    // Cognee services — reuse internal adapters
    cogneeIngGRPC   "github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/adapter/grpc"
    cogneeCognGRPC  "github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/adapter/grpc"
    cogneeSearchGRPC "github.com/vnp-community/vnp-memory/services/cognee-search/internal/adapter/grpc"

    // Graphiti services
    graphitiIngGRPC  "github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/adapter/grpc"
    // ... all other services
)
```

### 3.2 Go Module Strategy

```
apps/memory/go.mod
├── require github.com/vnp-community/vnp-memory/gateway => ../../gateway
├── require github.com/vnp-community/vnp-memory/services/cognee-ingestion => ../../services/cognee-ingestion
├── require github.com/vnp-community/vnp-memory/services/cognee-cognify => ../../services/cognee-cognify
├── ... (all 34 services + gateway)
└── require github.com/vnp-community/vnp-memory/pkg => ../../pkg
```

> **Lưu ý**: Sử dụng Go workspace (`go.work`) ở root level để manage local module references.

### 3.3 Không Sửa Đổi Code Hiện Tại

| Component | Cách Tái Sử Dụng |
|-----------|------------------|
| `gateway/internal/adapter/handler/` | Import trực tiếp Router, các Handler |
| `gateway/internal/usecase/` | Import AuthUseCase, RouteUseCase |
| `gateway/internal/infra/config/` | Import Config struct, Load() |
| `gateway/internal/adapter/mcp/` | Import MCP Server |
| `services/*/internal/adapter/grpc/` | Register handlers vào shared gRPC server |
| `services/*/internal/usecase/` | Import use case implementations |
| `services/*/internal/domain/` | Import domain models |
| `services/*/internal/infra/` | Import config, wire setup |

---

## 4. Inter-Module Communication

### 4.1 gRPC In-Process (bufconn) — Primary

Thay vì TCP connections giữa services, sử dụng `google.golang.org/grpc/test/bufconn`:

```go
// apps/memory/internal/bus/inprocess.go
package bus

import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

type InProcessBus struct {
    listener *bufconn.Listener
    server   *grpc.Server
    conns    map[string]*grpc.ClientConn  // service_name → conn
}

func NewInProcessBus() *InProcessBus {
    lis := bufconn.Listen(bufSize)
    srv := grpc.NewServer(
        // Shared interceptors
        grpc.ChainUnaryInterceptor(
            middleware.Recovery(),
            middleware.RequestID(),
            middleware.Logging(),
            middleware.Tracing(),
            middleware.TenantExtraction(),
        ),
    )
    return &InProcessBus{
        listener: lis,
        server:   srv,
        conns:    make(map[string]*grpc.ClientConn),
    }
}

// RegisterService registers a gRPC service handler
func (b *InProcessBus) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
    b.server.RegisterService(desc, impl)
}

// GetConn returns in-process connection (zero network overhead)
func (b *InProcessBus) GetConn(serviceName string) (*grpc.ClientConn, error) {
    if conn, ok := b.conns[serviceName]; ok {
        return conn, nil
    }
    conn, err := grpc.DialContext(context.Background(), "bufnet",
        grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
            return b.listener.DialContext(ctx)
        }),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, err
    }
    b.conns[serviceName] = conn
    return conn, nil
}
```

### 4.2 NATS Embedded — Async Events

Sử dụng NATS embedded server cho async event communication:

```go
// apps/memory/internal/bus/nats_embedded.go
package bus

import (
    natsserver "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
)

type EmbeddedNATS struct {
    server *natsserver.Server
    conn   *nats.Conn
    js     nats.JetStreamContext
}

func NewEmbeddedNATS() (*EmbeddedNATS, error) {
    opts := &natsserver.Options{
        DontListen: true,  // In-process only, no TCP
        JetStream:  true,
        StoreDir:   "./data/nats",
    }
    srv, err := natsserver.NewServer(opts)
    if err != nil {
        return nil, err
    }
    go srv.Start()
    if !srv.ReadyForConnections(5 * time.Second) {
        return nil, fmt.Errorf("NATS embedded not ready")
    }

    // In-process client connection
    nc, err := nats.Connect("", nats.InProcessServer(srv))
    if err != nil {
        return nil, err
    }
    js, err := nc.JetStream()
    if err != nil {
        return nil, err
    }
    return &EmbeddedNATS{server: srv, conn: nc, js: js}, nil
}
```

### 4.3 Communication Strategy Matrix

| Communication Type | Implementation | Latency |
|-------------------|---------------|---------|
| Gateway → Service (sync) | gRPC bufconn | ~0.1ms |
| Service → Service (sync) | gRPC bufconn | ~0.1ms |
| Service → Service (async) | NATS embedded | ~0.5ms |
| External client → Gateway | HTTP/gRPC TCP | Network-dependent |

---

## 5. Monolithic App Structure

```
apps/memory/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point — wires everything
├── internal/
│   ├── bus/                           # In-process communication
│   │   ├── inprocess.go              #   bufconn gRPC bus
│   │   ├── nats_embedded.go          #   Embedded NATS JetStream
│   │   └── registry.go              #   ServiceRegistry impl for gateway
│   ├── bootstrap/                     # Service bootstrappers
│   │   ├── gateway.go                #   Wire gateway handlers + usecase
│   │   ├── cognee.go                 #   Wire cognee-* services
│   │   ├── graphiti.go               #   Wire graphiti-* services
│   │   ├── memobase.go               #   Wire memobase-* services
│   │   ├── openviking.go             #   Wire ov-* services
│   │   ├── zep.go                    #   Wire zep-* services
│   │   ├── supermemory.go            #   Wire sm-* services
│   │   ├── platform.go              #   Wire vnp-event, vnp-search-hub, vnp-admin
│   │   └── infra.go                  #   Shared infra: PostgreSQL, Neo4j, Redis, Qdrant
│   └── config/
│       └── config.go                 #   Unified config (merge all service configs)
├── configs/
│   ├── config.yaml                   #   Default config
│   └── config.dev.yaml              #   Dev overrides
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
├── README.md
└── specs/
    ├── solutions/                    #   This directory
    └── tasks/                        #   Implementation tasks
```

---

## 6. Unified Config

Merge tất cả service configs thành 1 file:

```yaml
# apps/memory/configs/config.yaml

server:
  rest_port: 8080
  grpc_port: 8081
  mcp_port: 8082
  health_port: 8083
  log_level: info

auth:
  dev_mode: true
  jwt_public_key: ""
  jwt_issuer: "vnp-memory"
  jwt_audience: "vnp-memory"

# Shared Infrastructure
postgres:
  dsn: "postgres://vnp:vnp@localhost:5432/vnp_memory?sslmode=disable"
  max_conns: 50
  min_conns: 5

neo4j:
  uri: "bolt://localhost:7687"
  user: "neo4j"
  password: "password"

qdrant:
  addr: "localhost:6333"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

nats:
  mode: "embedded"  # "embedded" | "external"
  url: ""           # Only for external mode
  store_dir: "./data/nats"

minio:
  endpoint: "localhost:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  bucket: "vnp-memory"

bifrost:
  url: "http://localhost:8443"

# Per-Engine Configs
cognee:
  chunk_size: 1024
  max_concurrent_jobs: 4

graphiti:
  community_detection: true
  max_bfs_depth: 3

memobase:
  buffer_flush_threshold: 1024
  buffer_idle_timeout: 1h
  max_llm_calls_per_flush: 3

openviking:
  data_dir: "./data/vikingfs"
  encryption_enabled: false

zep:
  context_assembly_timeout: 200ms
  max_messages_per_thread: 10000

supermemory:
  forgetting_curve_enabled: true
  decay_half_life: 168h

platform:
  search_hub_timeout: 2s
  max_fan_out: 7
```

---

## 7. Bootstrap Flow

```go
// apps/memory/cmd/server/main.go (simplified)
func main() {
    cfg := config.Load()
    logger := setupLogger(cfg)

    // 1. Shared Infrastructure
    infra := bootstrap.NewInfra(cfg, logger)
    defer infra.Close()

    // 2. In-Process Bus
    bus := bus.NewInProcessBus()

    // 3. Embedded NATS (async events)
    embeddedNATS := bus.NewEmbeddedNATS(cfg)
    defer embeddedNATS.Close()

    // 4. Bootstrap Domain Services (register gRPC handlers)
    bootstrap.Cognee(bus, infra, embeddedNATS, cfg, logger)
    bootstrap.Graphiti(bus, infra, embeddedNATS, cfg, logger)
    bootstrap.Memobase(bus, infra, embeddedNATS, cfg, logger)
    bootstrap.OpenViking(bus, infra, embeddedNATS, cfg, logger)
    bootstrap.Zep(bus, infra, embeddedNATS, cfg, logger)
    bootstrap.Supermemory(bus, infra, embeddedNATS, cfg, logger)
    bootstrap.Platform(bus, infra, embeddedNATS, cfg, logger)

    // 5. Start bus (begins serving in-process gRPC)
    go bus.Serve()

    // 6. Bootstrap Gateway (uses bus as ServiceRegistry)
    registry := bus.AsServiceRegistry()
    gw := bootstrap.Gateway(registry, infra, cfg, logger)

    // 7. Start HTTP servers
    startServers(ctx, gw, cfg, logger)
}
```

---

## 8. Deployment

### 8.1 Development (Single Binary)

```bash
# Start infra only
docker compose -f docker-compose.infra.yml up -d

# Run monolithic app
cd apps/memory
go run ./cmd/server
```

### 8.2 Docker Compose (Compact)

```yaml
# apps/memory/docker-compose.yml
services:
  vnp-memory:
    build:
      context: ../..
      dockerfile: apps/memory/Dockerfile
    ports:
      - "8080:8080"   # REST
      - "8082:8082"   # MCP
      - "8083:8083"   # Health/Metrics
    environment:
      - POSTGRES_DSN=postgres://vnp:vnp@postgresql:5432/vnp_memory
      - NEO4J_URI=bolt://neo4j:7687
      - REDIS_ADDR=redis:6379
      - QDRANT_ADDR=qdrant:6333
    depends_on: [postgresql, neo4j, redis, qdrant, nats]

  # Infrastructure (shared)
  postgresql:
    image: pgvector/pgvector:pg17
    ports: ["5432:5432"]
  neo4j:
    image: neo4j:5-enterprise
    ports: ["7474:7474", "7687:7687"]
  qdrant:
    image: qdrant/qdrant:latest
    ports: ["6333:6333"]
  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
  minio:
    image: minio/minio:latest
    ports: ["9000:9000"]
```

### 8.3 Production (Kubernetes)

Có thể chạy cả monolithic (1 deployment) hoặc microservices (35 deployments) — sử dụng cùng code base:

```
# Monolithic
kubectl apply -f deploy/k8s/monolithic/

# Microservices
kubectl apply -f deploy/k8s/microservices/
```

---

## 9. Invariants

> Những điều KHÔNG ĐƯỢC thay đổi:

- [ ] Code trong `gateway/` — không sửa bất kỳ file nào
- [ ] Code trong `services/` — không sửa bất kỳ file nào
- [ ] Code trong `pkg/` — không sửa bất kỳ file nào
- [ ] Proto definitions trong `proto/` — không sửa
- [ ] gRPC API contracts — không thay đổi
- [ ] NATS event subjects — không thay đổi
- [ ] Database schemas — không thay đổi

---

## 10. Acceptance Criteria

| # | Criteria | Verification |
|---|----------|-------------|
| AC-1 | `go build ./apps/memory/cmd/server` compiles successfully | `make build` |
| AC-2 | All 35 gRPC services registered and callable | Integration test |
| AC-3 | Gateway REST API routes work identically | `api_tests.http` |
| AC-4 | MCP server responds to all 15 tools | MCP client test |
| AC-5 | NATS events flow between services in-process | Event trace logs |
| AC-6 | Health endpoint reports all 35 services | `curl :8083/healthz` |
| AC-7 | Config loading from single YAML + ENV override | Config test |
| AC-8 | Graceful shutdown in <5 seconds | Signal test |
| AC-9 | Memory usage < 512MB at startup | Profiling |
| AC-10 | Zero code modifications in gateway/ and services/ | `git diff` verification |
