# Cognee Task Planning — README

**Project:** VNP Memory  
**Domain:** Cognee Feature Parity  
**Solutions ref:** `specs/crs/v1/cognee/solutions/`  
**Architecture ref:** `specs/architecture.md` v3.0  
**Date:** 2026-06-17  

---

## Wave Map

| Wave | Scope | Tasks |
|------|-------|-------|
| **Wave 1** (Foundation) | Proto contracts + MCP Server | TASK-CE-001, TASK-CE-002 |
| **Wave 2** (Core Features) | NodeSets, Memify, DataPoint Schema | TASK-CE-003, TASK-CE-004, TASK-CE-005 |
| **Wave 3** (Advanced) | Advanced Loaders, Feedback Loop | TASK-CE-006, TASK-CE-007 |
| **Wave 4** (Orchestration & Infra) | Custom Pipelines, DB Migrations | TASK-CE-008, TASK-CE-009 |

---

## Task List (26 tasks total)

| Task | Wave | Solution | Component | Status | Estimate |
|------|------|----------|-----------|--------|----------|
| [TASK-CE-001](./TASK-CE-001-protobuf-contracts.md) | 1 | SOL-001, SOL-002, SOL-003, SOL-006 | `api/proto/cognee/` | 🔲 Pending | 3h |
| [TASK-CE-002](./TASK-CE-002-mcp-tool-registry.md) | 1 | SOL-007 | `gateway/internal/adapter/mcp/` | 🔲 Pending | 5h |
| [TASK-CE-003](./TASK-CE-003-nodesets-ingestion.md) | 2 | SOL-002 | `services/cognee-ingestion/` | 🔲 Pending | 4h |
| [TASK-CE-004](./TASK-CE-004-nodesets-cognify.md) | 2 | SOL-002 | `services/cognee-cognify/` | 🔲 Pending | 3h |
| [TASK-CE-005](./TASK-CE-005-nodesets-search.md) | 2 | SOL-002 | `services/cognee-search/` | 🔲 Pending | 3h |
| [TASK-CE-006](./TASK-CE-006-memify-usecase.md) | 2 | SOL-001 | `services/cognee-cognify/` | 🔲 Pending | 5h |
| [TASK-CE-007](./TASK-CE-007-datapoint-schema.md) | 2 | SOL-003 | `services/cognee-ingestion/` | 🔲 Pending | 5h |
| [TASK-CE-008](./TASK-CE-008-advanced-loaders.md) | 3 | SOL-004 | `services/cognee-ingestion/` | 🔲 Pending | 4h |
| [TASK-CE-009](./TASK-CE-009-feedback-loop.md) | 3 | SOL-005 | `services/cognee-search/` + `cognee-memory/` | 🔲 Pending | 6h |
| [TASK-CE-010](./TASK-CE-010-custom-pipelines.md) | 4 | SOL-006 | `services/cognee-cognify/` | 🔲 Pending | 6h |
| [TASK-CE-011](./TASK-CE-011-db-migrations.md) | 4 | SOL-001, SOL-003, SOL-005 | `db/migrations/` | 🔲 Pending | 1h |

---

## Architecture Context

```
Monolith (InProcessRegistry bufconn):
  cognee-ingestion  → port 9011  (AddData, AddDataPoints)
  cognee-cognify    → port 9012  (StartCognify, Memify, GetPipelineStatus)
  cognee-search     → port 9013  (Search, ListInteractions)
  cognee-memory     → port 9014  (Remember)
  
Data Stores:
  Neo4j    — graph (nodes + edges + labels + weights)
  Qdrant   — vectors (collections per tenant + payload filtering)
  PostgreSQL — metadata (datasets, pipeline runs, interactions, feedback)
  NATS JetStream — async events
  Bifrost  — LLM gateway
```

## Execution Order

```
Wave 1: [CE-001] → [CE-002]
         ↓
Wave 2: [CE-003] + [CE-006] + [CE-007] (parallel)
         ↓ CE-003 done
        [CE-004] → [CE-005]
         ↓ all Wave 2 done
Wave 3: [CE-008] + [CE-009] (parallel)
         ↓
Wave 4: [CE-010] + [CE-011] (parallel)
```

## New NATS Subjects

| Subject | Publisher | Consumer |
|---------|-----------|----------|
| `cognee.cognify.memify.started` | cognee-cognify | obs-service |
| `cognee.cognify.memify.completed` | cognee-cognify | gateway |
| `cognee.cognify.memify.failed` | cognee-cognify | obs-service |
| `cognee.ingestion.datapoints.added` | cognee-ingestion | cognee-cognify |
| `cognee.search.feedback.applied` | cognee-search | obs-service |

## New DB Tables

| Table | Migration | CR |
|-------|-----------|-----|
| `cognee_pipeline_runs` | `0020_cognee_pipeline_runs.up.sql` | CR-001 |
| `cognee_datapoints` | `0021_cognee_datapoints.up.sql` | CR-003 |
| `cognee_interactions` | `0022_cognee_interactions.up.sql` | CR-005 |
| `cognee_feedback_records` | `0022_cognee_interactions.up.sql` | CR-005 |
