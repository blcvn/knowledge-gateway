# Tasks

## llm-embedding-adapter

- [x] **T1** — Extend `internal/platform/vector/provider.go`: rename `Provider` to `EmbeddingProvider`, add `Embed(ctx context.Context, text string) ([]float64, error)` and `ModelID() string`; wrap `DeterministicProvider` to satisfy the new interface.
- [x] **T2** — Implement `internal/platform/vector/http_provider.go`: `HTTPEmbeddingProvider` struct with configurable URL, model ID, API key, and HTTP timeout.
- [x] **T3** — Implement `internal/platform/vector/router.go`: `EmbeddingRouter` interface and `DirectRouter` (single-provider default). `workers.Runtime` and `search.NewService` accept `EmbeddingRouter`, not bare `EmbeddingProvider`.
- [x] **T4** — Implement middleware providers in `internal/platform/vector/`: `CachingProvider` (TTL-based, keyed by text hash), `RetryProvider` (exponential backoff), `ProxyHTTPProvider` (rewrites request through a proxy URL). Each wraps an inner `EmbeddingProvider`.
- [x] **T5** — Update `workers.Runtime.projectNode` to call `EmbeddingRouter.RouteContext(tenantID, domainID).Embed(ctx, text)` and propagate errors.
- [x] **T6** — Remove `buildEmbeddingVector` from `workers/runtime.go`; consolidate embedding text construction into a helper that respects the resolved `SearchProfile` (see T36).
- [x] **T7** — Add unit tests: `HTTPEmbeddingProvider` with mock server (success, 5xx, timeout); `CachingProvider` (cache hit, cache miss, TTL expiry); `RetryProvider` (retries on error, stops after max); `ProxyHTTPProvider` (rewrites URL); `DirectRouter` routes all to single provider; `RoutingRouter` routes by tenant.

## vector-store-adapter

- [x] **T8** — Define `internal/platform/vectorstore/adapter.go`: `VectorAdapter` interface (`Upsert`, `Delete`, `ANN(ctx, []float64, VectorFilter, ANNOptions)`) and supporting types (`VectorDocument`, `VectorFilter`, `VectorResult`, `ANNOptions`).
- [x] **T9** — Implement `internal/platform/vectorstore/memory.go`: `InMemoryVectorAdapter` — honors `TopK` from `ANNOptions`, ignores other fields; used as default in tests.
- [x] **T10** — Implement `internal/platform/vectorstore/pgvector.go`: `PgVectorAdapter` — `kg_vector_documents` table, `hnsw` index, parameterised `Upsert`/`Delete`; `ANN` applies `IndexHint` and `EfSearch` via session-level `SET` statements before the query, `FilterMode` determines WHERE clause position.
- [x] **T11** — Update `workers.Runtime` to call `VectorAdapter.Upsert` / `VectorAdapter.Delete` instead of writing to the in-memory `VectorStore` map.
- [x] **T12** — Update `search.Service.SemanticSearch` and `RagSearch` to call `VectorAdapter.ANN` with `ANNOptions` derived from the resolved `SearchProfile`.
- [x] **T13** — Update `workers.Runtime.Reconcile` to snapshot vector state via `VectorAdapter`.
- [x] **T14** — Add migration file for `kg_vector_documents` table.
- [x] **T15** — Add integration tests: upsert → ANN returns result; delete → ANN no longer returns result; ACL pre-filter respected; `ANNOptions.MinScore` filters low-similarity results; `InMemoryVectorAdapter` ignores unknown `ANNOptions` fields.

## graph-db-adapter

- [x] **T16** — Define `internal/platform/graphstore/adapter.go`: `GraphAdapter` interface (`UpsertNode`, `DeleteNode`, `UpsertRelationship`, `DeleteRelationship`, `ExecuteQuery(ctx, GraphQuery, map[string]any)`) and supporting types (`GraphQuery`, `GraphQueryHop`).
- [x] **T17** — Implement `internal/platform/graphstore/memory.go`: `InMemoryGraphAdapter` — `ExecuteQuery` drives the existing `ProjectionGraphIndex.walkTemplate` logic, translated to operate on `GraphQuery` instead of a compiled Cypher string.
- [x] **T18** — Implement `internal/platform/graphstore/neo4j.go` (build tag `neo4j`): `Neo4jGraphAdapter`; extract `graphQueryToCypher(GraphQuery, params) (string, params)` as a pure function (unit-testable without a live DB); `ExecuteQuery` calls it then passes the result to the driver session.
- [x] **T19** — Update `workers.Runtime` to call `GraphAdapter.UpsertNode` / `DeleteNode` / `UpsertRelationship` / `DeleteRelationship`.
- [x] **T20** — Update `read.QueryTemplateCompiler.Compile` to produce a `GraphQuery` instead of a raw Cypher string; update `read.GraphIndex` to call `GraphAdapter.ExecuteQuery`.
- [x] **T21** — Update `workers.Runtime.Reconcile` to snapshot graph state via `GraphAdapter`.
- [x] **T22** — Add unit tests for `graphQueryToCypher`: default strategy, deep_traversal, ACL param binding, named strategy params.
- [x] **T23** — Add integration tests: upsert node → ExecuteQuery returns it; delete node → not returned; ACL predicate honored; `InMemoryGraphAdapter` and `Neo4jGraphAdapter` produce equivalent results for the same `GraphQuery`.

## full-text-search

- [x] **T24** — Define `internal/platform/fts/adapter.go`: `FTSAdapter` interface (`Index`, `Delete`, `Search`) and types (`FTSDocument`, `FTSQuery{Text, Mode, Fields}`, `FTSFilter`, `FTSResult`). `Mode` values are `"all_tokens"`, `"any_token"`, `"phrase"` — no Postgres syntax.
- [x] **T25** — Implement `internal/platform/fts/memory.go`: `InMemoryFTSAdapter` — honors `Mode` using token matching; honors `Fields` by restricting match to named properties.
- [x] **T26** — Implement `internal/platform/fts/postgres.go`: `PgFTSAdapter` — add `fts_vector` generated `tsvector` column to `kg_nodes` via migration, `GIN` index; map `Mode` → `plainto_tsquery` / `to_tsquery` (OR) / `phraseto_tsquery`; use `ts_rank_cd` for ordering; apply `Fields` as weighted `tsvector` sub-queries.
- [x] **T27** — Add `FullTextSearch(actor, req FullTextSearchRequest) (FullTextSearchResponse, error)` to `search.Service`; `FullTextSearchRequest` has `Query`, `DomainIDs`, `TopK`, `Mode`, `Fields`; apply ACL, domain, lifecycle, and authority-score filtering identical to `SemanticSearch`.
- [x] **T28** — Add `HybridSearch(actor, req HybridSearchRequest) (HybridSearchResponse, error)` to `search.Service`; run FTS and semantic search concurrently; fuse with RRF (k=60) weighted by `SemanticWeight`.
- [x] **T29** — Update `workers.Runtime.projectNode` to call `FTSAdapter.Index` alongside the vector upsert.
- [x] **T30** — Update `workers.Runtime.handleEvent` for `NODE_DELETED` to call `FTSAdapter.Delete`.
- [x] **T31** — Add HTTP handler and routes `POST /search/fulltext` and `POST /search/hybrid` in `internal/search/handler.go`.
- [x] **T32** — Add migration file for `fts_vector` column and `GIN` index.
- [x] **T33** — Add unit tests: all three `Mode` values on `InMemoryFTSAdapter`; `PgFTSAdapter` mode→tsquery mapping (no live DB needed, test the translation function in isolation); ACL filter; lifecycle filter; hybrid RRF fusion math.

## domain-search-config

- [x] **T34** — Add `SearchProfile`, `IndexedField`, `SearchProfileOverride`, `QueryStrategy`, `ResolvedSearchProfile` types to `internal/ontology/types.go`. `SearchProfile.CypherStrategy` is renamed to `QueryStrategyRef`.
- [x] **T35** — Add `search_profile` nullable JSONB column to `kg_domains` and `kg_query_strategies` table via migrations; update `ontology.Store` to read and write both.
- [x] **T36** — Define `SearchProfileResolver` interface in `internal/ontology/`; implement `DefaultSearchProfileResolver` that applies app > tenant > domain > hardcoded default precedence.
- [x] **T37** — Wire `SearchProfileResolver` into `workers.Runtime` and `search.Service` via dependency injection; replace all inline profile logic with `resolver.Resolve(domainID, tenantID, appID)`.
- [x] **T38** — Move `buildEmbeddingText` out of `workers/runtime.go` into a shared helper package; rewrite it to accept a `ResolvedSearchProfile` and build text from `SemanticFields` with weights and prefixes.
- [x] **T39** — Update `read.QueryTemplateCompiler.Compile` to accept a `QueryStrategy` (resolved from ontology via `QueryStrategyRef`); emit `GraphQuery` with `MaxDepth` and `Strategy` from the strategy object; support `"default"`, `"deep_traversal"`, and named keys via a registered handler map (adding a handler does not change `Compile`).
- [x] **T40** — Seed built-in `QueryStrategy` records (`"default"`, `"deep_traversal"`) in `internal/bootstrap/bootstrap_seed.go`.
- [x] **T41** — Add ontology API endpoints: `PUT /ontology/domains/{domain_id}/search-profile`, `GET /ontology/domains/{domain_id}/search-profile`, `POST /ontology/query-strategies`, `PUT /ontology/query-strategies/{key}`, `GET /ontology/query-strategies`; validate field names, weights, and `QueryStrategyRef` existence.
- [x] **T42** — Add unit tests:
  - `DefaultSearchProfileResolver` precedence (app > tenant > domain > system default)
  - System default `SemanticFields` matches the current `buildEmbeddingText` field list exactly
  - `SemanticFields = nil` → system defaults applied; `SemanticFields = []` → rejected with 422
  - `QueryStrategyRef` referencing missing key → WARNING logged, `"default"` strategy used, no error returned
  - `Compile` for `"default"` (MaxDepth=5, fixed, acl=any_hop), `"deep_traversal"` (MaxDepth=10, variable, acl=start_only), and a custom strategy with `Params`
  - `SearchProfile` validation: invalid field names, weights outside [0.1, 10.0], empty `FTSLanguage`
  - `QueryStrategy` version increment on update; built-in strategies reject update/delete with 403

## Cross-cutting

- [x] **T43** — Add new config fields to `internal/config/config.go` and `env.go`: `EMBEDDING_PROVIDER` (`"deterministic"` | `"http"`), `EMBEDDING_URL`, `EMBEDDING_MODEL`, `EMBEDDING_API_KEY`, `EMBEDDING_PROXY_URL`, `EMBEDDING_CACHE_TTL_S`, `VECTOR_ADAPTER` (`"memory"` | `"pgvector"`), `GRAPH_ADAPTER` (`"memory"` | `"neo4j"`), `FTS_ADAPTER` (`"memory"` | `"postgres"`).
- [x] **T44** — Update `bootstrap/app.go` to build the `EmbeddingRouter` middleware chain from config (order: cache → retry → proxy → HTTP), select vector/graph/FTS adapter, select `SearchProfileResolver` implementation.
- [x] **T45** — Log the full middleware chain for `EmbeddingRouter` at startup (INFO level) so operators can verify the wiring without inspecting code.
- [x] **T46** — Verify all existing tests pass with in-memory adapters and `DeterministicProvider` as defaults; no test should require external services.
- [x] **T47** — Add end-to-end integration test: create node → outbox event → worker projects to vector and FTS → `SemanticSearch`, `FullTextSearch`, and `HybridSearch` all return the node with correct ACL.
- [x] **T48** — Add adapter conformance test suite (`internal/platform/*/conformance_test.go`): a shared set of scenarios that any `VectorAdapter`, `GraphAdapter`, `FTSAdapter`, or `EmbeddingProvider` implementation must pass; run against in-memory adapters in CI, against production adapters in staging.
