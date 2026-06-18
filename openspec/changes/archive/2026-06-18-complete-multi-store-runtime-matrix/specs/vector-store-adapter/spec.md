# vector-store-adapter

## MODIFIED Requirements

### Requirement: VectorAdapter interface for external vector store

The system SHALL support production vector adapters for `pgvector`, `qdrant`, and `milvus`, all behind the same `VectorAdapter` interface.

#### Scenario: Switch vector backends without changing search service

- GIVEN `search.Service` calls `VectorAdapter.ANN`
- WHEN the configured vector backend changes between `pgvector`, `qdrant`, and `milvus`
- THEN `search.Service` SHALL require no backend-specific code changes
- AND the adapter SHALL map `ANNOptions` and `VectorFilter` to its own backend primitives

### Requirement: Vector adapters expose reconciliation snapshots

Production vector adapters SHALL expose enough state for reconciliation to compare payload and sync version against PostgreSQL.

#### Scenario: Reconciliation checks vector replica state

- GIVEN a configured production vector adapter
- WHEN `workers.Runtime.Reconcile` runs
- THEN it SHALL load vector documents from that backend rather than from legacy in-memory mirrors
- AND it SHALL compare `_kg_sync_version` metadata together with ACL/domain/status payload fields

### Requirement: Qdrant and Milvus preserve sync metadata

The system SHALL persist projection sync metadata in `qdrant` and `milvus` payload/document fields so replay and reconciliation can prove convergence.

#### Scenario: Replay updates a document in Qdrant

- GIVEN a node has already been projected once
- WHEN a later outbox event reprojects the node into `qdrant`
- THEN the adapter SHALL upsert the document idempotently
- AND SHALL update the stored `_kg_sync_version` to the latest applied source version
