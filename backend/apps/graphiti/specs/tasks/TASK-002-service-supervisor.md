---
id: TASK-002
title: "Service Supervisor — Goroutine Lifecycle Manager"
app: apps/graphiti
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-001]
---

## Mục Tiêu

Implement Supervisor pattern quản lý lifecycle (startup, health monitoring, shutdown) của tất cả embedded services + gateway goroutines.

## Scope

### In Scope
- `internal/supervisor/supervisor.go` — Core supervisor logic
- Phase-ordered startup (4 phases theo dependency graph)
- Reverse-order graceful shutdown
- gRPC health check wait-for-ready per service
- Panic recovery per goroutine

### Out of Scope
- Specific service startup functions (TASK-003)
- Gateway embedding (TASK-004)

## Thiết Kế Kỹ Thuật

### Core Types

```go
type ServiceSpec struct {
    Name    string
    StartFn func(ctx context.Context) error
    Port    int  // gRPC port, 0 if no gRPC
    Phase   int  // 1=data, 2=intelligence, 3=app, 4=gateway
}

type Supervisor struct {
    services []ServiceSpec
    logger   *slog.Logger
    running  map[string]context.CancelFunc
    errors   map[string]error
}
```

### Phase Ordering

```
Phase 1: graphiti-store        (depends: Neo4j)
Phase 2: graphiti-knowledge    (depends: store, LLM)
Phase 3: graphiti-ingestion, graphiti-search, graphiti-pipeline (parallel)
Phase 4: vnp-gateway           (depends: all services)
```

### Key Methods

- `StartAll(ctx)` — Launch by phase, wait for gRPC health per phase
- `Shutdown(ctx)` — Reverse phase order, cancel contexts, wait with timeout
- `HealthCheck()` — Return map[string]bool from gRPC health checks
- `waitForReady(name, port, timeout)` — Poll gRPC health until SERVING

## Acceptance Criteria

- [x] AC-1: StartAll launches services in correct phase order (0→1→2→3)
- [x] AC-2: Each phase waits for TCP port readiness before next phase
- [x] AC-3: Shutdown stops services in reverse order (3→2→1→0)
- [x] AC-4: Panic in one goroutine is recovered, does not crash process
- [x] AC-5: HealthCheck returns accurate status per service
- [x] AC-6: waitForPort times out after 30s if service fails to start
- [x] AC-7: Context cancellation propagates to all goroutines

## Definition of Done

- [x] Unit tests pass, coverage 92.8% ≥ 80% ✅
- [x] Không có lint errors
