---
id: FEAT-002
title: Infrastructure Health Probes
service: vnp-platform
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: gateway/SOL-002 (T11)
linked_ux: "ux_spec.md §6.10 Infrastructure View"
---

## Mục Tiêu

Probe health và metrics của shared infrastructure (PostgreSQL, Neo4j, Qdrant, Redis, NATS) cho Infrastructure View trong Console UI.

## Scope

### In Scope
- gRPC `InfraService.GetServiceTopology()` — service map with connections
- gRPC `InfraService.GetDatabaseHealth()` — DB cluster status
- gRPC `InfraService.GetResourceMetrics()` — CPU/RAM/disk per service
- gRPC `InfraService.GetDeploymentTimeline()` — recent deployments

### Out of Scope
- Database migration execution
- Resource scaling (Kubernetes scope)

## Thiết Kế Kỹ Thuật

### Business Logic

1. **Service Topology:** Query all 35 service endpoints + gateway for connectivity status
2. **Database Health:**
   - PostgreSQL: `SELECT pg_is_in_recovery()`, replication lag
   - Neo4j: bolt driver status, cluster members
   - Qdrant: collection info, vector count
   - Redis: `INFO server`, memory usage
   - NATS: JetStream stream/consumer info
3. **Resource Metrics:** Scrape from Prometheus (if available) or process `/proc` stats

## Acceptance Criteria
- [ ] AC-1: Service topology returns all 35+ services with status
- [ ] AC-2: Database health includes PostgreSQL, Neo4j, Qdrant, Redis, NATS
- [ ] AC-3: Resource metrics include CPU, RAM, disk per service
- [ ] AC-4: Deployment timeline returns last 20 deployments
- [ ] AC-5: Unhealthy components highlighted with error details

## Test Requirements
- Unit tests: Health probe logic
- Integration tests: Mock infrastructure endpoints
- Minimum coverage: 80%
