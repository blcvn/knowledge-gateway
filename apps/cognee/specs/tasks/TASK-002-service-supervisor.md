---
id: TASK-002
title: "Service Supervisor — Goroutine Lifecycle Manager"
app: apps/cognee
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-001]
estimated: 2h
---

## Mục Tiêu

Implement service supervisor quản lý lifecycle (start, health check, ordered shutdown) cho tất cả embedded services + gateway chạy như goroutines trong 1 process.

## Scope

### In Scope
- `internal/supervisor/supervisor.go` — Core supervisor
- Ordered startup: services trước (gRPC ready), gateway sau
- Ordered shutdown: gateway trước (drain HTTP), services sau (gRPC GracefulStop)
- Health aggregation: poll gRPC health per embedded service
- Panic recovery per goroutine

### Out of Scope
- Actual service startup logic (TASK-003, TASK-004)

## Thiết Kế Kỹ Thuật

### Supervisor API

```go
// internal/supervisor/supervisor.go
package supervisor

type Phase int
const (
    PhaseInfra   Phase = 0 // Services start first (gRPC servers)
    PhaseGateway Phase = 1 // Gateway starts after services ready
)

type ServiceSpec struct {
    Name    string
    Phase   Phase
    Port    int
    StartFn func(ctx context.Context) error  // Blocks until ctx cancelled
}

type Supervisor struct {
    services []ServiceSpec
    logger   *slog.Logger
    wg       sync.WaitGroup
    errors   chan error
}

func New(logger *slog.Logger) *Supervisor

func (s *Supervisor) Register(spec ServiceSpec)

// StartAll starts services in phase order:
// 1. Start Phase 0 (services) → wait for gRPC ports to be ready
// 2. Start Phase 1 (gateway)
// Blocks until ctx is cancelled or a critical error occurs
func (s *Supervisor) StartAll(ctx context.Context) error

// waitForPort polls tcp port until ready or timeout
func waitForPort(addr string, timeout time.Duration) error

// Shutdown stops in reverse phase order:
// 1. Signal Phase 1 (gateway) to stop → wait for drain
// 2. Signal Phase 0 (services) to stop → wait for GracefulStop
func (s *Supervisor) Shutdown(timeout time.Duration) error

// HealthCheck returns health status for each embedded service
func (s *Supervisor) HealthCheck() map[string]string
```

### Startup Sequence

```
1. Load config
2. Set ENV vars (config.SetServiceEnvVars())
3. Start Phase 0 (services):
   ├── goroutine: cognee-ingestion gRPC :9011
   ├── goroutine: cognee-cognify  gRPC :9012
   └── goroutine: cognee-search   gRPC :9013
4. Wait for all Phase 0 ports ready (TCP dial check)
5. Start Phase 1 (gateway):
   └── goroutine: gateway REST :8080 (services → localhost:PORT)
6. Log "cognee-app ready"
7. Wait for SIGTERM/SIGINT
```

### Shutdown Sequence

```
1. Receive SIGTERM
2. Cancel gateway context → gateway drains HTTP connections
3. Wait gateway stopped (or 10s timeout)
4. Cancel service contexts → services GracefulStop gRPC
5. Wait services stopped (or 15s timeout)
6. Close remaining cleanup functions
7. Log "cognee-app stopped"
```

### Error Handling

```go
// Each service goroutine:
go func() {
    defer func() {
        if r := recover(); r != nil {
            s.logger.Error("service panicked", "service", spec.Name, "error", r)
            s.errors <- fmt.Errorf("service %s panicked: %v", spec.Name, r)
        }
    }()
    if err := spec.StartFn(ctx); err != nil {
        s.logger.Error("service error", "service", spec.Name, "error", err)
        s.errors <- err
    }
}()
```

## Acceptance Criteria

- [x] AC-1: Services start in Phase 0 before gateway Phase 1
- [x] AC-2: Gateway waits until all service ports accepting TCP connections
- [x] AC-3: SIGTERM → gateway stops first, then services
- [x] AC-4: Individual goroutine panic does not crash whole process
- [x] AC-5: HealthCheck() returns per-service status
- [x] AC-6: Startup timeout — if service port not ready in 30s, fail with error

## Definition of Done

- [x] Supervisor with phased startup/shutdown
- [x] Port readiness check
- [x] Panic recovery per goroutine
- [x] Unit tests for ordering + error handling
