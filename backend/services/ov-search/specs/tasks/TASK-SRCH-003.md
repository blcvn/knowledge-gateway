---
id: TASK-SRCH-003
title: "Implement Infrastructure Layer and Persistence"
service: ov-search
status: Done
priority: High
created_at: 2026-05-11
---

# TASK-SRCH-003: Implement Infrastructure Layer and Persistence

## Objective
Implement the Infrastructure Layer (Layer 4) covering vector database integration (Qdrant), relational database persistence (PostgreSQL), and configuration management.

## Requirements

1. **Vector Repository (`internal/infra/persistence/qdrant_repo.go`)**:
   - Implement `VectorRepository` interface using Qdrant.
   - Configure collection `ov_embeddings` (1536-dim, Cosine distance, payload indexing on `account_id` and `parent_dir`).
   - Create `pgvector_repo.go` as a fallback implementation.

2. **Hotness Repository (`internal/infra/persistence/hotness_repo.go`)**:
   - Implement `HotnessRepository` interface using PostgreSQL.
   - Support queries on `ov_hotness_scores` and `ov_search_metadata` tables.

3. **Database Migrations**:
   - Create migration scripts for PostgreSQL tables: `ov_hotness_scores` and `ov_search_metadata`.
   - Apply necessary indexes as per the Data Model spec (`idx_hotness_account`, `idx_hotness_computed`, `idx_metadata_path`, `idx_metadata_stale`).

4. **Configuration (`internal/infra/config/config.go`)**:
   - Load application configurations using Viper or similar.
   - Core settings: `GRPC_PORT` (9052), `HEALTH_PORT` (9105), `LOG_LEVEL`, `OTEL_ENDPOINT`.
   - Resource endpoints: `NATS_URL`, `DB_DSN`, `QDRANT_URL`, `BIFROST_ADDR`, `OV_FS_ADDR`.
   - Algorithm tunables: `HOTNESS_DECAY_HALF_LIFE_H`, `HOTNESS_SESSION_BOOST`, `HOTNESS_RECOMPUTE_INTERVAL_M`, `RERANK_DEFAULT_STRATEGY`, `SEARCH_MAX_RESULTS`, `PROPAGATION_FACTOR`, `CONVERGENCE_THRESHOLD`.
   - Embedding settings: `EMBEDDING_MODEL`, `EMBEDDING_DIM` (1536), `QDRANT_COLLECTION`.

## Constraints
- Ensure query isolation by `account_id` for multi-tenancy compliance.
