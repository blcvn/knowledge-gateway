---
id: TASK-002
title: "Service Supervisor — Goroutine Lifecycle Manager"
app: apps/memobase
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-001]
---

## Mục Tiêu

Implement `internal/supervisor/supervisor.go` — process supervisor quản lý lifecycle của tất cả embedded services + gateway trong 1 process. **Reuse pattern đã proven** từ `apps/graphiti/internal/supervisor/supervisor.go`.

## Scope

### In Scope
- `apps/memobase/internal/supervisor/supervisor.go` — Supervisor struct, Register, StartAll, Shutdown, HealthCheck
- Phased startup: Data → Intelligence → Application → Gateway
- Ordered shutdown: Gateway → Application → Intelligence → Data
- TCP port readiness probing
- Panic recovery per goroutine
- Race-safe status tracking

### Out of Scope
- Service start functions (TASK-003)
- Gateway start function (TASK-004)
- Health HTTP server (TASK-005)

## Thiết Kế Kỹ Thuật

### Phase Order (Memobase-specific)

```
Phase 0 (Data):           memobase-ingestion   — Must start first (blob storage)
Phase 1 (Intelligence):   memobase-engine      — Depends on ingestion (blob fetch)
Phase 2 (Application):    memobase-context      — Depends on engine (profile read)
                          memobase-pipeline    — Pipeline orchestration
Phase 3 (Gateway):        vnp-gateway          — After all gRPC services ready
```

### API (consistent with graphiti pattern)

```go
type Phase int
const (
    PhaseData         Phase = 0
    PhaseIntelligence Phase = 1
    PhaseApplication  Phase = 2
    PhaseGateway      Phase = 3
)

type ServiceSpec struct {
    Name    string
    Phase   Phase
    Port    int
    StartFn func(ctx context.Context) error
}

type Supervisor struct { ... }

func New(logger *slog.Logger) *Supervisor
func (s *Supervisor) Register(spec ServiceSpec)
func (s *Supervisor) StartAll(parentCtx context.Context) error
func (s *Supervisor) Shutdown(timeout time.Duration)
func (s *Supervisor) HealthCheck() map[string]string
```

### Key Behaviors
1. **StartAll** groups services by phase, launches each phase concurrently, waits for port readiness before next phase
2. **Shutdown** cancels contexts in reverse phase order (gateway first, data last)
3. **HealthCheck** returns map of service name → status ("registered"/"starting"/"serving"/"stopping"/"stopped"/"failed")
4. **Panic recovery** per goroutine — service crash doesn't kill process
5. **waitForPort** — TCP dial with 30s timeout per service

## Acceptance Criteria

- [x] AC-1: Phase 0 services start before Phase 1 (verified by port readiness order)
- [x] AC-2: Gateway (Phase 3) only starts after all gRPC services are ready
- [x] AC-3: Shutdown stops gateway first, then services in reverse phase order
- [x] AC-4: Panic in one goroutine doesn't crash the process
- [x] AC-5: `HealthCheck()` returns correct status for each service
- [x] AC-6: Unit tests with race detection pass (`go test -race`)
- [x] AC-7: API consistent with `apps/graphiti/internal/supervisor/supervisor.go`

## Test Requirements
- Unit tests: phased startup order, shutdown order, panic recovery, health check
- Race detection: `go test -race`
- Minimum coverage: 80%
