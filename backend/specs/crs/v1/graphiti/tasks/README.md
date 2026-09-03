# Graphiti Tasks — README

**Project:** VNP Memory · Feature Parity: Graphiti  
**Architecture Ref:** `specs/architecture.md` v3.0  
**Solutions Ref:** `specs/crs/v1/graphiti/solutions/`  
**Date:** 2026-06-17

---

## Execution Waves

| Wave | Tasks | Prerequisite | Description |
|------|-------|-------------|-------------|
| **Wave 1** | TASK-GR-001 → TASK-GR-006 | None | Core foundation: `pkg/graph/`, `graphiti-store`, `graphiti-knowledge` |
| **Wave 2** | TASK-GR-007 → TASK-GR-013 | Wave 1 | Ingestion pipeline + search engine |
| **Wave 3** | TASK-GR-014 → TASK-GR-020 | Wave 2 | Custom ontology + gateway REST + MCP tools |
| **Wave 4** | TASK-GR-021 → TASK-GR-026 | Wave 3 | Admin service + observability |

---

## Task Index

### Wave 1 — Foundation

| Task ID | File | Component | Estimated | Solution |
|---------|------|-----------|-----------|---------|
| TASK-GR-001 | [TASK-GR-001](./TASK-GR-001-shared-graph-types.md) | `pkg/graph/` shared types | 2h | SOL-002 §2 |
| TASK-GR-002 | [TASK-GR-002](./TASK-GR-002-neo4j-driver-interface.md) | `graphiti-store` GraphDriver interface | 3h | SOL-002 §3 |
| TASK-GR-003 | [TASK-GR-003](./TASK-GR-003-neo4j-entity-repositories.md) | `graphiti-store` Node/Edge repos | 4h | SOL-002 §4 |
| TASK-GR-004 | [TASK-GR-004](./TASK-GR-004-neo4j-bulk-save-bfs.md) | `graphiti-store` SaveBulk + BFS | 4h | SOL-002 §4.2, §4.3 |
| TASK-GR-005 | [TASK-GR-005](./TASK-GR-005-neo4j-migrations.md) | Neo4j schema migrations | 1h | SOL-002 §5 |
| TASK-GR-006 | [TASK-GR-006](./TASK-GR-006-llm-client-adapters.md) | `graphiti-knowledge` LLM clients | 4h | SOL-003 §2 |

### Wave 2 — Ingestion & Search

| Task ID | File | Component | Estimated | Solution |
|---------|------|-----------|-----------|---------|
| TASK-GR-007 | [TASK-GR-007](./TASK-GR-007-prompt-registry.md) | `graphiti-knowledge` prompt templates | 3h | SOL-003 §3 |
| TASK-GR-008 | [TASK-GR-008](./TASK-GR-008-extract-resolve-entities.md) | Entity extraction + resolution | 5h | SOL-003 §4, §5 |
| TASK-GR-009 | [TASK-GR-009](./TASK-GR-009-extract-resolve-edges.md) | Edge extraction + resolution | 4h | SOL-003 §6 |
| TASK-GR-010 | [TASK-GR-010](./TASK-GR-010-community-detection.md) | Community detection (LPA + LLM) | 4h | SOL-003 §7 |
| TASK-GR-011 | [TASK-GR-011](./TASK-GR-011-ingestion-pipeline.md) | 9-step ingestion pipeline | 6h | SOL-001 §4 |
| TASK-GR-012 | [TASK-GR-012](./TASK-GR-012-group-worker-pool.md) | GroupWorkerPool + chunking | 3h | SOL-001 §5, §6 |
| TASK-GR-013 | [TASK-GR-013](./TASK-GR-013-hybrid-search-engine.md) | Hybrid search (BM25+cosine+BFS) + 5 rerankers | 6h | SOL-004 §3, §4 |

### Wave 3 — Ontology & Gateway

| Task ID | File | Component | Estimated | Solution |
|---------|------|-----------|-----------|---------|
| TASK-GR-014 | [TASK-GR-014](./TASK-GR-014-search-cache-nats.md) | Search result cache + NATS invalidation | 2h | SOL-004 §5, §6 |
| TASK-GR-015 | [TASK-GR-015](./TASK-GR-015-custom-ontology.md) | Custom ontology (store + validation) | 4h | SOL-005 §2→§5 |
| TASK-GR-016 | [TASK-GR-016](./TASK-GR-016-ontology-presets.md) | Domain presets (HR, CRM, Software) | 2h | SOL-005 §3 |
| TASK-GR-017 | [TASK-GR-017](./TASK-GR-017-gateway-episode-routes.md) | Gateway: episode + triplet + saga REST | 4h | SOL-006 §2.1 |
| TASK-GR-018 | [TASK-GR-018](./TASK-GR-018-gateway-search-routes.md) | Gateway: search + entity REST + /healthz | 3h | SOL-006 §2.2, §2.3 |
| TASK-GR-019 | [TASK-GR-019](./TASK-GR-019-mcp-graphiti-tools.md) | MCP 9 graphiti tools (22→31) | 4h | SOL-006 §5 |
| TASK-GR-020 | [TASK-GR-020](./TASK-GR-020-rate-limiter-middleware.md) | Rate limiter + group_id propagation | 2h | SOL-006 §3, §4 |

### Wave 4 — Admin & Observability

| Task ID | File | Component | Estimated | Solution |
|---------|------|-----------|-----------|---------|
| TASK-GR-021 | [TASK-GR-021](./TASK-GR-021-admin-service-tenant.md) | `graphiti-admin` service #36 + tenant CRUD | 5h | SOL-007 §2→§5 |
| TASK-GR-022 | [TASK-GR-022](./TASK-GR-022-admin-community-index.md) | Admin: community rebuild + index ops | 2h | SOL-007 §6, §7 |
| TASK-GR-023 | [TASK-GR-023](./TASK-GR-023-token-usage-report.md) | Token usage report + cost estimation | 2h | SOL-007 §8 |
| TASK-GR-024 | [TASK-GR-024](./TASK-GR-024-otel-instrumentation.md) | OTel spans across all graphiti services | 4h | SOL-007 §9 |
| TASK-GR-025 | [TASK-GR-025](./TASK-GR-025-prometheus-metrics.md) | Prometheus metrics per service | 3h | SOL-007 §10 |
| TASK-GR-026 | [TASK-GR-026](./TASK-GR-026-anonymous-telemetry.md) | Anonymous telemetry (PostHog opt-out) | 1h | SOL-007 §11 |

---

## Dependency Graph

```
TASK-GR-001 ─┬──► TASK-GR-002 ──► TASK-GR-003 ──► TASK-GR-004
             │                                         │
             │    TASK-GR-005 (parallel)               │
             │                                         ▼
             └──► TASK-GR-006 ──► TASK-GR-007 ──► TASK-GR-008
                                                       │
                                               TASK-GR-009
                                                       │
                                               TASK-GR-010
                                                       │
                              TASK-GR-011 ◄────────────┘
                              TASK-GR-012 (parallel with 011)
                                    │
                              TASK-GR-013 ──► TASK-GR-014
                                    │
                              TASK-GR-015 ──► TASK-GR-016
                                    │
                              TASK-GR-017 ──► TASK-GR-018
                              TASK-GR-019 (parallel with 017)
                              TASK-GR-020 (parallel with 017)
                                    │
                              TASK-GR-021 ──► TASK-GR-022
                              TASK-GR-023 (parallel with 021)
                              TASK-GR-024 (parallel, needs 011-013)
                              TASK-GR-025 (parallel with 024)
                              TASK-GR-026 (independent)
```

---

## Total Estimate

| Wave | Tasks | Est. Hours |
|------|-------|-----------|
| Wave 1 | 6 | ~18h |
| Wave 2 | 7 | ~31h |
| Wave 3 | 7 | ~21h |
| Wave 4 | 6 | ~17h |
| **Total** | **26** | **~87h** |
