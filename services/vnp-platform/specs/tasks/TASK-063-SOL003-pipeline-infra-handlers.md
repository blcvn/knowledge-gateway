---
id: TASK-063
title: "[SOL-003 T09] vnp-platform — Pipeline Status + Infrastructure Probe Handlers"
service: vnp-platform
type: FEAT
priority: P1
status: Done
created: 2026-05-14
updated: 2026-05-14
linked_specs:
  - ui/specs/solutions/SOL-003-ui-gateway-hardening.md
  - gateway/specs/solutions/SOL-002-ux-console-api-upgrade.md
---

## Mục Tiêu
Expose HTTP handlers trong `vnp-platform` service để Gateway có thể proxy calls cho:
- Pipeline status aggregation (jobs, queues, workers)
- Infrastructure health probes (topology, services, databases, resources, deployments)

## Bối Cảnh Nghiệp Vụ
Gateway đã implement `console_pipeline_usecase.go` (SOL-002 T10) và `console_infra_usecase.go` (SOL-002 T11) với fan-out health check + topology probe logic. Cần downstream handler để serve actual data.

## Phạm Vi Công Việc (Scope)

### In Scope
1. **Pipeline Handlers**:
   - `GET /api/v1/pipelines/status` — Aggregated pipeline status across engines
   - `GET /api/v1/pipelines/queues` — Queue depths per engine
   - `GET /api/v1/pipelines/workers` — Worker pool status
   - `GET /api/v1/pipelines/{engine}/jobs` — Job list per engine
2. **Infrastructure Handlers**:
   - `GET /api/v1/infra/topology` — Service dependency graph
   - `GET /api/v1/infra/services` — All service health status
   - `GET /api/v1/infra/services/{name}` — Individual service detail
   - `GET /api/v1/infra/databases` — Database connection pool + latency
   - `GET /api/v1/infra/resources` — CPU/Memory/Disk per node
   - `GET /api/v1/infra/deployments` — Recent deployment history
3. **Health probe runner**: Probe 34 services (6 engine groups × 5-7 services + platform services)

### Out of Scope
- WebSocket streaming for realtime infra updates (FEAT-012)
- Auto-scaling triggers

## Thiết Kế Kỹ Thuật

### Internal Architecture
```
handler/pipeline_handler.go → usecase/pipeline_usecase.go → probers/engine_prober.go
handler/infra_handler.go    → usecase/infra_usecase.go    → probers/service_prober.go
                                                           → store/deployment_store.go
```

### Fan-out Health Check Pattern
```
infra_usecase.GetTopology():
  → Concurrent gRPC calls to 34 services
  → 5-second timeout per service
  → Return partial results if some services timeout
  → Cache results for 30 seconds
```

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/pipelines/status` returns aggregated status object
- [ ] AC-2: `GET /api/v1/infra/topology` returns service dependency graph with health status
- [ ] AC-3: `GET /api/v1/infra/services` returns health for all probed services (partial results on timeout)
- [ ] AC-4: Health probe completes within 5s (fan-out with timeout)
- [ ] AC-5: Results cached 30s to avoid thundering herd
- [ ] AC-6: Unit tests ≥ 80% coverage for handler + usecase

## Test Requirements
- Unit tests: Handler contracts, usecase fan-out logic, timeout handling
- Integration tests: Service prober with mock gRPC servers
- Minimum coverage: 80%
