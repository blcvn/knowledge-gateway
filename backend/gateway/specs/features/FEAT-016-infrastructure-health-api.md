---
id: FEAT-016
title: Infrastructure Health API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.10 Infrastructure View"
---

## Mục Tiêu

REST APIs cho Infrastructure View — service topology, database health, resource monitoring.

## Scope

### In Scope
- `GET /v1/console/infra/topology` — Service topology with connections
- `GET /v1/console/infra/services` — All 18 services status
- `GET /v1/console/infra/services/{name}` — Service detail
- `GET /v1/console/infra/databases` — DB health (PG, Neo4j, Redis, NATS)
- `GET /v1/console/infra/resources` — Resource usage per service

### Out of Scope
- Service management (start/stop/restart)
- Infrastructure provisioning

## Thiết Kế Kỹ Thuật

### Internal Architecture
- **Handler:** `adapter/http/infra_handler.go`
- **Proxy to:** `vnp-platform` (infrastructure aggregation)
- **Source:** gRPC health checks, Prometheus scrape, NATS admin API

## Acceptance Criteria
- [ ] AC-1: Topology returns all 18 services with connection graph
- [ ] AC-2: Service detail shows health, gRPC port, latency
- [ ] AC-3: Database health includes PG, Neo4j, Redis, NATS details
- [ ] AC-4: Resource monitoring returns CPU/memory/disk per service
- [ ] AC-5: All endpoints require admin role

## Test Requirements
- Unit tests: Health aggregation, resource metric parsing
- Integration tests: Mock infrastructure health
- Minimum coverage: 80%
