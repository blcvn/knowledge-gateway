---
id: FEAT-009
title: Adaptive Memory Proxy API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.5 Adaptive Memory"
---

## Mục Tiêu

Proxy Supermemory APIs cho Console UI Adaptive Memory screen: memory versions, auto-forget rules, external connectors, analytics.

## Scope

### In Scope
- `GET /v1/console/adaptive/memories` — List adaptive memories (paginated)
- `GET /v1/console/adaptive/memories/{id}/versions` — Version chain
- `GET /v1/console/adaptive/connectors` — List external connectors
- `POST /v1/console/adaptive/connectors` — Create connector
- `POST /v1/console/adaptive/connectors/{id}/sync` — Trigger sync
- `GET /v1/console/adaptive/analytics` — Adaptive memory analytics
- `GET /v1/console/adaptive/forget-rules` — Auto-forget rules
- `PUT /v1/console/adaptive/forget-rules` — Update forget rules

### Out of Scope
- Direct memory graph visualization (Graph Studio handles this)

## Thiết Kế Kỹ Thuật

### Internal Architecture
- **Handler:** `adapter/http/adaptive_handler.go`
- **Proxy to:** `sm-memory`, `sm-connector`, `sm-analytics`, `sm-engine` via gRPC
- Version chain: calls `sm-memory.GetMemoryVersions()`
- Connectors: calls `sm-connector.ListConnections()`, `CreateConnection()`, `TriggerSync()`
- Analytics: calls `sm-analytics.GetStats()`

## Acceptance Criteria
- [ ] AC-1: Memory list returns paginated adaptive memories with `isLatest` flag
- [ ] AC-2: Version chain returns parent→root with relation types (updates/extends/derives)
- [ ] AC-3: Connector list shows status, last sync time, document count
- [ ] AC-4: Create connector validates OAuth credentials
- [ ] AC-5: Trigger sync returns job ID for tracking
- [ ] AC-6: Analytics returns creation/deletion rate, contradiction resolution frequency
- [ ] AC-7: Forget rules support per-memory-type TTL configuration

## Test Requirements
- Unit tests: Connector validation, version chain transformation
- Integration tests: Mock Supermemory gRPC services
- Minimum coverage: 80%
