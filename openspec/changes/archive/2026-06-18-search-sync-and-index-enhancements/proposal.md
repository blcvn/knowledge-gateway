# Proposal: Search, Sync, and Index Enhancements

## Problem

The current service passes all tests but relies on several architectural stubs that make it unsuitable for production deployment:

1. **No real LLM embedding**: `DeterministicProvider` produces hash-based pseudo-vectors. Semantic search and RAG operate on these fake embeddings, producing meaningless similarity scores in production.

2. **No real vector store adapter**: `VectorStore` in `workers/` is an in-memory `map[string]VectorDocument`. There is no adapter to persist or query vectors in an external store (pgvector, Qdrant, Weaviate, etc.).

3. **No real graph DB adapter**: `GraphStore` in `workers/` is an in-memory `map[string]GraphNode`. There is no adapter to persist or query graph data in a real graph database (Neo4j, etc.).

4. **No full-text search**: The service only provides semantic (cosine similarity) search. There is no full-text search capability, which is necessary for exact keyword queries, structured filters, and hybrid retrieval patterns.

5. **No per-domain / per-tenant / per-app search customization**: Index field selection, embedding text construction, Cypher query strategy, and FTS analyzer are hardcoded globally. Different domains (legal, finance, HR) or different tenants need to customize which fields contribute to their index, how similarity is computed, and which Cypher patterns are executed.

## Proposed Solution

Introduce five focused additions:

### 1. LLM Embedding Adapter (`llm-embedding-adapter`)
Define a pluggable `EmbeddingProvider` interface backed by a real HTTP LLM endpoint (Claude, OpenAI, local Ollama, etc.). Keep `DeterministicProvider` as the test/fallback implementation. `NewService` and `workers.Runtime` accept the provider via dependency injection.

### 2. Vector Store Adapter (`vector-store-adapter`)
Define a `VectorAdapter` interface for upsert, delete, and ANN (approximate nearest neighbor) queries. Provide an `InMemoryVectorAdapter` for tests and a `PgVectorAdapter` as the first production implementation (using PostgreSQL + pgvector extension). Workers write embeddings through the adapter on outbox projection events.

### 3. Graph DB Adapter (`graph-db-adapter`)
Define a `GraphAdapter` interface for node/relationship upsert, delete, and Cypher execution. Provide an `InMemoryGraphAdapter` for tests and a `Neo4jGraphAdapter` for production. The `read.GraphIndex` and `workers.Runtime` accept the adapter via injection.

### 4. Full-Text Search (`full-text-search`)
Add a `FullTextSearch` operation on the search service backed by PostgreSQL `tsvector`/`tsquery` (or a pluggable FTS adapter). Support `AND`, `OR`, phrase queries, and field-level weighting. Include hybrid search that fuses FTS rank with semantic score (reciprocal rank fusion). Respect ACL, domain, lifecycle, and authority-score semantics identically to semantic search.

### 5. Domain Search Configuration (`domain-search-config`)
Extend the ontology layer with a per-domain `SearchProfile` that controls:
- Which node fields are indexed for semantic search and their relative weights
- Embedding text template (ordered field list with optional prefix labels)
- FTS language/analyzer per domain
- Cypher query strategy per domain (default pattern vs. custom traversal depth/direction)
- Per-tenant and per-app overrides on top of the domain baseline

## Out of Scope

- Changing the public REST or MCP API surface (new search endpoints are additive)
- Migrating existing production data
- A/B testing or gradual rollout infrastructure
- Adding new ontology entity types beyond `SearchProfile`

## Success Criteria

- A real LLM embedding provider can be wired at startup without changing service logic
- Vectors are persisted to and queried from a real external store (pgvector in CI)
- Graph mutations are persisted to and queried from a real graph store (in-memory adapter in CI, Neo4j in staging/prod)
- Full-text search returns results for keyword queries with correct ACL and lifecycle filtering
- Hybrid search combines FTS and semantic scores
- Domain search profiles are stored in the ontology layer and applied at index time and query time
- Per-tenant and per-app overrides on the search profile are honored
- All existing tests remain green; new tests cover the new adapter boundaries
