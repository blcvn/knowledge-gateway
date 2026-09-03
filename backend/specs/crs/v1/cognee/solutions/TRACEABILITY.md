# Traceability Matrix — Cognee Solutions

**Project:** VNP Memory  
**Domain:** Cognee Feature Parity  
**Architecture ref:** `specs/architecture.md` v3.0  
**Date:** 2026-06-17

---

## Solution Map

| CR | Solution File | Integration Point | Wave |
|----|--------------|-------------------|------|
| CR-COGNEE-001 | [SOL-001](./SOL-001-Memify-Enrichment.md) | EXTEND `services/cognee-cognify` + `api/proto/cognee/cognify/v1/` | Wave 2 |
| CR-COGNEE-002 | [SOL-002](./SOL-002-NodeSets-Scoping.md) | EXTEND `services/cognee-ingestion` + `cognee-search` + `cognee-cognify` | Wave 1 |
| CR-COGNEE-003 | [SOL-003](./SOL-003-DataPoint-Schema.md) | EXTEND `services/cognee-ingestion` + Neo4j + Qdrant | Wave 2 |
| CR-COGNEE-004 | [SOL-004](./SOL-004-Advanced-Loaders.md) | EXTEND `services/cognee-ingestion/internal/adapter/extractor/` | Wave 3 |
| CR-COGNEE-005 | [SOL-005](./SOL-005-Feedback-Loop.md) | EXTEND `services/cognee-search` + `cognee-memory` + PostgreSQL | Wave 3 |
| CR-COGNEE-006 | [SOL-006](./SOL-006-Custom-Pipelines.md) | REFACTOR `services/cognee-cognify/internal/usecase/start_cognify.go` | Wave 4 |
| CR-COGNEE-007 | [SOL-007](./SOL-007-MCP-Parity.md) | EXTEND `gateway/internal/adapter/mcp/tool_registry.go` (16 → 22 tools) | Wave 1 |

---

## Architecture Context

Tất cả Cognee services chạy **in-process** trong monolith (35 services, `InProcessRegistry` bufconn):

| Service | gRPC Port (external/Gateway-only) | In-process |
|---------|----------------------------------|-----------|
| `cognee-ingestion` | 9011 | bufconn |
| `cognee-cognify` | 9012 | bufconn |
| `cognee-search` | 9013 | bufconn |
| `cognee-memory` (memory-service) | 9014 | bufconn |

**Data stores dùng chung:**
- **Neo4j** — graph storage (nodes, edges, labels, weights)
- **Qdrant** — vector store (collections per tenant)
- **PostgreSQL** — metadata, interactions, feedback records
- **NATS JetStream** — async events giữa cognee services
- **Bifrost** — LLM gateway cho entity extraction, embedding

---

## New NATS Subjects (Cognee)

| Subject | Publisher | Consumer | CR |
|---------|-----------|----------|----|
| `cognee.cognify.memify.started` | cognee-cognify | obs-service | CR-001 |
| `cognee.cognify.memify.completed` | cognee-cognify | gateway (console) | CR-001 |
| `cognee.cognify.memify.failed` | cognee-cognify | obs-service | CR-001 |
| `cognee.ingestion.datapoints.added` | cognee-ingestion | cognee-cognify (partial) | CR-003 |
| `cognee.search.feedback.applied` | cognee-search | obs-service | CR-005 |

---

## Gateway Routes Added

| Method | Path | CR | Target Service |
|--------|------|----|---------------|
| POST | `/api/v1/cognee/datasets/{id}/memify` | CR-001 | cognee-cognify |
| GET | `/api/v1/cognee/datasets/{id}/memify/status` | CR-001 | cognee-cognify |
| POST | `/api/v1/cognee/datasets/{id}/datapoints` | CR-003 | cognee-ingestion |
| GET | `/api/v1/cognee/interactions` | CR-005 | cognee-search |
| GET | `/api/v1/cognee/pipeline/templates` | CR-006 | static/cognee-cognify |

---

## PostgreSQL Tables Added

| Table | CR | Service |
|-------|----|---------|
| `cognee_pipeline_runs` | CR-001 | cognee-cognify |
| `cognee_datapoints` | CR-003 | cognee-ingestion |
| `cognee_interactions` | CR-005 | cognee-search |
| `cognee_feedback_records` | CR-005 | cognee-search |

---

## MCP Tools: Before → After

| Before (16 tools) | After (22 tools) | Change |
|-------------------|--------------------|--------|
| `memory_store`, `memory_recall`, `memory_search`, `memory_timeline`, `memory_profile`, `memory_forget`, `graph_query`, `ov_read_file`, `ov_write_file`, `ov_search`, `ov_list_dir`, `ov_grep`, `ov_tree`, `ov_session_commit`, `ov_ingest`, `ov_delete` | +`cognify`, `search`, `save_interaction`, `list_data`, `delete_dataset`, `cognify_status` | +6 tools |
