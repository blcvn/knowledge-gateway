# vector-store-adapter

## MODIFIED Requirements

### Requirement: VectorAdapter interface for external vector store

The system SHALL define a `VectorAdapter` interface with `Upsert`, `Delete`, and `ANN` operations so that the vector store backend is swappable between test and production.

#### Scenario: Upsert a vector document with canonical UUID identity

- WHEN `workers.Runtime` processes a `NODE_UPSERTED` event
- THEN it SHALL call `VectorAdapter.Upsert` with a document whose service-owned identity is UUID-compatible
- AND the upsert SHALL be idempotent for repeated writes of the same canonical node UUID

#### Scenario: Qdrant-compatible point identity

- WHEN the active adapter is Qdrant
- THEN the adapter SHALL use a point id representation accepted by Qdrant
- AND that point identity SHALL map deterministically to the canonical node UUID
- AND the payload SHALL still preserve the node UUID and domain metadata for readback and reconciliation
