---
id: SOL-003
title: Service Bootstrap — Wiring 35 Services into Single Binary
version: 1.0.0
status: Proposed
priority: P0
created: 2026-05-14
linked_sol: SOL-001, SOL-002
---

# SOL-003: Service Bootstrap

## 1. Tổng Quan

Chi tiết cách wire 35 domain services + gateway vào single binary, tái sử dụng 100% code từ `gateway/` và `services/` mà không sửa đổi.

---

## 2. Bootstrap Pattern

Mỗi engine group có 1 bootstrapper file. Pattern chung:

```go
// apps/memory/internal/bootstrap/cognee.go
func Cognee(bus *bus.GRPCBus, infra *Infra, nats *bus.NATSBus, cfg *config.Config, logger *slog.Logger) {
    // 1. Create repositories (reuse service adapter/repository packages)
    ingRepo := cogneeIngRepo.NewPostgres(infra.PG, logger)
    
    // 2. Create use cases (reuse service usecase packages)  
    ingUC := cogneeIngUC.NewIngestUseCase(ingRepo, nats, logger)
    
    // 3. Create gRPC handlers (reuse service adapter/grpc packages)
    ingHandler := cogneeIngGRPC.NewHandler(ingUC, logger)
    
    // 4. Register on shared bus
    bus.Register(&pb.CogneeIngestionService_ServiceDesc, ingHandler)
    
    // Repeat for cognee-cognify, cognee-search
}
```

---

## 3. Bootstrap Files Map

| File | Services Wired | Count |
|------|---------------|-------|
| `bootstrap/infra.go` | PostgreSQL, Neo4j, Qdrant, Redis, MinIO pools | — |
| `bootstrap/gateway.go` | Gateway REST + MCP + WebDAV + Auth | 1 |
| `bootstrap/cognee.go` | cognee-ingestion, cognee-cognify, cognee-search | 3 |
| `bootstrap/graphiti.go` | graphiti-ingestion, graphiti-search, graphiti-knowledge, graphiti-store | 4 |
| `bootstrap/memobase.go` | memobase-ingestion, memobase-engine, memobase-context | 3 |
| `bootstrap/openviking.go` | ov-fs, ov-search, ov-session, ov-resource, ov-crypto, ov-admin | 6 |
| `bootstrap/zep.go` | zep-user, zep-thread, zep-memory, zep-graph, zep-search, zep-admin | 6 |
| `bootstrap/supermemory.go` | sm-document, sm-memory, sm-search, sm-profile, sm-connector, sm-mcp, sm-auth, sm-analytics, sm-project | 9 |
| `bootstrap/platform.go` | vnp-event, vnp-search-hub, vnp-admin | 3 |

---

## 4. Shared Infrastructure (infra.go)

```go
type Infra struct {
    PG      *pgxpool.Pool     // PostgreSQL + pgvector
    Neo4j   neo4j.DriverWithContext
    Qdrant  *qdrant.Client
    Redis   *redis.Client
    MinIO   *minio.Client
    Bifrost llm.LLMClient
}

func NewInfra(cfg *config.Config, logger *slog.Logger) (*Infra, error) {
    // Create shared connection pools — all services share same pools
    // Each service uses different schemas/collections for isolation
}

func (i *Infra) Close() {
    i.PG.Close()
    i.Neo4j.Close(context.Background())
    i.Redis.Close()
    // ...
}
```

---

## 5. Gateway Bootstrap

Gateway adapter cần `ServiceRegistry` để route requests. Trong monolithic mode, sử dụng `InProcessRegistry` (SOL-002) thay vì TCP registry:

```go
func Gateway(registry port.ServiceRegistry, infra *Infra, cfg *config.Config, logger *slog.Logger) *GatewayServers {
    // Reuse gateway usecase layer 100%
    authUC := gwUsecase.NewAuthUseCase(...)
    routeUC := gwUsecase.NewRouteUseCase(registry, ...)
    
    // Reuse gateway handler layer 100%
    memoryH := gwHandler.NewMemoryHandler(routeUC, registry, logger)
    cogneeH := gwHandler.NewCogneeHandler(registry, logger)
    // ... all other handlers
    
    // Reuse gateway router 100%
    router := gwHandler.Router(memoryH, cogneeH, ..., logger)
    
    // Reuse MCP server 100%
    mcpSrv := gwMCP.NewServer(registry, logger)
    
    return &GatewayServers{Router: router, MCP: mcpSrv}
}
```

---

## 6. NATS Subscriber Wiring

Mỗi service cần subscribe NATS events. Pattern:

```go
// Trong bootstrap/cognee.go
func wireCogneeSubscribers(nats *bus.NATSBus, cognifyUC usecase.CognifyUseCase, logger *slog.Logger) {
    nats.Subscribe("cognee.data.ingested", "cognee-cognify", func(msg *nats.Msg) {
        var evt CogneeDataIngested
        json.Unmarshal(msg.Data, &evt)
        if err := cognifyUC.ProcessIngested(context.Background(), evt); err != nil {
            logger.Error("cognify failed", "error", err)
            msg.Nak()
            return
        }
        msg.Ack()
    })
}
```

---

## 7. Go Workspace Setup

```
// go.work (project root)
go 1.25.0

use (
    ./gateway
    ./services/cognee-ingestion
    ./services/cognee-cognify
    ./services/cognee-search
    ./services/graphiti-ingestion
    ./services/graphiti-search
    ./services/graphiti-knowledge
    ./services/graphiti-store
    ./services/memobase-ingestion
    ./services/memobase-engine
    ./services/memobase-context
    ./services/ov-fs
    ./services/ov-search
    ./services/ov-session
    ./services/ov-resource
    ./services/ov-crypto
    ./services/ov-admin
    ./services/zep-user
    ./services/zep-thread
    ./services/zep-memory
    ./services/zep-graph
    ./services/zep-search
    ./services/zep-admin
    ./services/sm-document
    ./services/sm-memory
    ./services/sm-search
    ./services/sm-profile
    ./services/sm-connector
    ./services/sm-mcp
    ./services/sm-auth
    ./services/sm-analytics
    ./services/sm-project
    ./services/vnp-event
    ./services/vnp-search-hub
    ./services/vnp-admin
    ./apps/memory
    ./pkg
)
```

---

## 8. Acceptance Criteria

| # | Criteria |
|---|----------|
| AC-1 | All 35 gRPC service descriptors registered on shared bus |
| AC-2 | All NATS subscribers wired with correct subjects |
| AC-3 | Gateway uses InProcessRegistry seamlessly |
| AC-4 | Shared infra pools used across all services |
| AC-5 | `go.work` resolves all local module references |
| AC-6 | `go build ./apps/memory/cmd/server` succeeds |
