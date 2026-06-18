# vector-store-adapter

## Requirements

### Requirement: VectorAdapter interface for external vector store
The system SHALL define a `VectorAdapter` interface with `Upsert`, `Delete`, and `ANN` operations so that the vector store backend is swappable between test and production.

The `ANN` method SHALL accept an `ANNOptions` struct rather than a bare `topK int` so that adapter-specific tuning parameters can be added over time without changing the interface signature.

```go
type ANNOptions struct {
    TopK       int     // required
    MinScore   float64 // 0.0 = disabled
    IndexHint  string  // "hnsw" | "ivfflat" | "flat" | "" (adapter default)
    EfSearch   int     // HNSW ef_search; 0 = adapter default
    FilterMode string  // "pre" | "post" | "" (adapter default)
}
```

Adapters that do not support a given field SHALL ignore it silently. Adding a new field to `ANNOptions` is always backward-compatible.

#### Scenario: Upsert a vector document
- WHEN `workers.Runtime` processes a `NODE_UPSERTED` event
- THEN it SHALL call `VectorAdapter.Upsert` with the document embedding and ACL metadata
- AND the upsert SHALL be idempotent (re-upserting the same node ID converges to the latest state)

#### Scenario: Delete a vector document
- WHEN `workers.Runtime` processes a `NODE_DELETED` event
- THEN it SHALL call `VectorAdapter.Delete` for the node ID
- AND subsequent ANN queries SHALL not return the deleted document

#### Scenario: Query nearest neighbours with ACL filter
- WHEN `search.Service.SemanticSearch` is called
- THEN it SHALL call `VectorAdapter.ANN` with the query vector, a `VectorFilter` containing the caller's visible ACL tokens and domain IDs, and an `ANNOptions` derived from the resolved `SearchProfile`
- AND the adapter SHALL only return documents whose `acl_visible_to` intersects the caller's token set

#### Scenario: Tune ANN accuracy vs. speed via options
- WHEN an operator sets `ANNOptions.EfSearch=200` and `ANNOptions.IndexHint="hnsw"` in the domain's `SearchProfile`
- THEN `PgVectorAdapter` SHALL apply those values to the query-time index parameters
- AND other adapters (e.g. a future Qdrant adapter) SHALL read the same `ANNOptions` and map them to their own equivalent parameters

### Requirement: InMemoryVectorAdapter for tests
The system SHALL provide an `InMemoryVectorAdapter` that implements `VectorAdapter` using the existing `workers.VectorStore` map, ignoring all `ANNOptions` fields except `TopK`, so that all current tests continue to pass without external dependencies.

### Requirement: PgVectorAdapter for production
The system SHALL provide a `PgVectorAdapter` that stores vectors in a `kg_vector_documents` table using the `pgvector` extension.

#### Scenario: Production ANN query with pre-filter
- WHEN `PgVectorAdapter.ANN` is called with `FilterMode="pre"` (or default)
- THEN it SHALL execute a parameterised SQL query that applies domain and ACL predicates as `WHERE` clauses before the `ORDER BY embedding <=> $query` step

#### Scenario: Switch index type without code change
- WHEN an operator changes `ANNOptions.IndexHint` from `"hnsw"` to `"ivfflat"` in the domain search profile
- THEN `PgVectorAdapter` SHALL issue the query without modifying any Go source files (the hint drives a SET/session-level index preference, not a code branch)

### Requirement: Sync correctness
- WHEN `workers.Runtime.Reconcile` is called
- THEN it SHALL compare source Postgres node records against the state in `VectorAdapter`
- AND SHALL report drift if any node is missing from or stale in the adapter
- AND SHALL NOT read from the legacy in-memory `VectorStore` struct during reconciliation
