# vector-store-adapter

## MODIFIED Requirements

### Requirement: Vector adapters support batch upsert and batch delete

Production vector adapters SHALL support bulk document writes and deletes so projection runtime can batch
embedding-backed sync work.

#### Scenario: Worker projects a node batch into the vector backend

- GIVEN the worker has produced embeddings for 24 node documents
- WHEN it dispatches them to the configured vector adapter
- THEN the adapter SHALL accept them through a batch upsert contract
- AND the runtime SHALL NOT be forced to call single-document upsert 24 times on the hot path

### Requirement: Vector adapters avoid per-document flush semantics on batch-capable backends

Batch-capable vector backends SHALL flush or commit according to batch or short time-window policy, not
after every individual document.

#### Scenario: Milvus batch projection completes

- GIVEN the worker has upserted a batch of vector documents into Milvus
- WHEN the batch finishes
- THEN the adapter SHALL flush once for the batch or according to a bounded flush interval
- AND it SHALL NOT flush once per document in that batch

### Requirement: Vector adapters preserve sync-version monotonicity in batch mode

Vector adapters SHALL store `_kg_sync_version` for every projected document and SHALL reject stale writes
that would move a document backwards in source version.

#### Scenario: Older vector projection arrives after a newer document already exists

- GIVEN a vector document for `node-789` already stores `_kg_sync_version=21`
- AND the runtime later tries to write source version `20` for the same node
- WHEN the adapter evaluates the write
- THEN it SHALL keep the existing document version at `21`
- AND it SHALL report the stale write as a no-op or idempotent success
