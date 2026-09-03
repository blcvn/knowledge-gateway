---
id: FEAT-002
title: REST Router + 8 Namespace Handlers
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
linked_sol: SOL-001
---

## Mục Tiêu

Implement chi/v5 HTTP router with 50+ REST routes across 8 API namespaces, each forwarding to the appropriate downstream gRPC service.

## Scope

### In Scope
- chi/v5 router setup with route groups
- 8 handler files (memory, cognee, graphiti, memobase, ov, zep, sm, admin)
- REST → gRPC transcoding (JSON body → protobuf → JSON response)
- Path parameter extraction and forwarding
- Timeout policy per route group (30s default, 120s ingestion, 10s search)

### Out of Scope
- Authentication (FEAT-001)
- Rate limiting (FEAT-003)
- MCP server (FEAT-005)

## Thiết Kế Kỹ Thuật

### API Contract

See [api.md](../../docs/api.md) for complete route table.

### Business Logic

```go
// internal/adapter/http/router.go
func NewRouter(
    memory  *MemoryHandler,
    cognee  *CogneeHandler,
    graphiti *GraphitiHandler,
    memobase *MemobaseHandler,
    ov       *OpenVikingHandler,
    zep      *ZepHandler,
    sm       *SMHandler,
    admin    *AdminHandler,
    mw       *MiddlewareStack,
) *chi.Mux {
    r := chi.NewRouter()
    r.Use(mw.Recovery, mw.RequestID, mw.Logger, mw.CORS, mw.Auth)
    
    r.Route("/v1/memory", memory.Routes)
    r.Route("/v1/cognee", cognee.Routes)
    r.Route("/v1/graphiti", graphiti.Routes)
    r.Route("/v1/memobase", memobase.Routes)
    r.Route("/v1/ov", ov.Routes)
    r.Route("/v1/zep", zep.Routes)
    r.Route("/v1/sm", sm.Routes)
    r.Route("/v1/admin", admin.Routes)
    
    return r
}
```

### Internal Architecture
- `internal/adapter/http/router.go` — Main router, Wire-injected
- `internal/adapter/http/memory_handler.go` — `/v1/memory/*` (4 routes)
- `internal/adapter/http/cognee_handler.go` — `/v1/cognee/*` (4 routes)
- `internal/adapter/http/graphiti_handler.go` — `/v1/graphiti/*` (4 routes)
- `internal/adapter/http/memobase_handler.go` — `/v1/memobase/*` (5 routes)
- `internal/adapter/http/openviking_handler.go` — `/v1/ov/*` (11 routes)
- `internal/adapter/http/zep_handler.go` — `/v1/zep/*` (9 routes)
- `internal/adapter/http/sm_handler.go` — `/v1/sm/*` (9 routes)
- `internal/adapter/http/admin_handler.go` — `/v1/admin/*` (4 routes)

## Acceptance Criteria
- [ ] AC-1: Given valid request to any route, When dispatched, Then correct handler invoked
- [ ] AC-2: Given unknown route, When request arrives, Then return 404 JSON error
- [ ] AC-3: Given path params (e.g., `{id}`), When request arrives, Then params extracted and forwarded
- [ ] AC-4: Given ingestion route, When timeout not specified, Then 120s timeout applied
- [ ] AC-5: Given search route, When timeout not specified, Then 10s timeout applied

## Test Requirements
- **Unit tests**: Route matching, param extraction, timeout selection
- **Integration tests**: Full router with mock handlers
- **Minimum coverage**: 80%
