# Design: Search, Sync, and Index Enhancements

## Current Behavior

| Concern | Current State | Gap |
|---|---|---|
| Embedding | `DeterministicProvider` (hash-based, 8-dim) | Not a real LLM; scores are meaningless |
| Vector store | `workers.VectorStore` in-memory map | No persistence; lost on restart |
| Graph store | `workers.GraphStore` in-memory map | No persistence; lost on restart |
| Full-text search | Absent | Only cosine similarity + token-overlap |
| Index configuration | Hardcoded in `buildEmbeddingText` / `nodeContent` | No per-domain/tenant/app customization |
| Sync (outbox → projection) | Logic correct; writes to in-memory stores | Never reaches an external backend |

## Architecture

```
Write Path
  Postgres (source of truth)
      │  outbox events
      ▼
  workers.Runtime
      ├─► GraphAdapter ──► Neo4j (prod) / InMemoryGraphAdapter (test)
      └─► VectorAdapter ──► PgVector (prod) / InMemoryVectorAdapter (test)
                                │ uses EmbeddingProvider
                                └─► LLM HTTP (prod) / DeterministicProvider (test)

Read / Search Path
  search.Service
      ├─► SemanticSearch  ──► VectorAdapter.ANN
      ├─► FullTextSearch  ──► FTSAdapter (Postgres tsvector / pluggable)
      └─► HybridSearch    ──► fuse(SemanticSearch, FullTextSearch) via RRF

  read.Service
      └─► GraphIndex  ──► GraphAdapter.ExecuteQuery(GraphQuery)
```

## Key Design Decisions

### 1. Adapter interfaces with production and test implementations

Every new external dependency (embedding, vector store, graph DB, FTS) is introduced as an interface. Production implementations are wired at bootstrap. Tests continue to use the existing in-memory/deterministic implementations. This preserves test coverage without requiring external services in CI.

**EmbeddingProvider** (extends existing `vector.Provider`):
```go
type EmbeddingProvider interface {
    Embed(ctx context.Context, text string) ([]float64, error)
    Dimensions() int
    ModelID() string
}
```
Production implementation: HTTP client to LLM endpoint (configurable URL, model, API key).
Test implementation: existing `DeterministicProvider` wrapped to satisfy the new signature.

**EmbeddingRouter** (proxy/middleware layer above `EmbeddingProvider`):
```go
type EmbeddingRouter interface {
    EmbeddingProvider
    // RouteContext selects the active provider for a given (tenant, domain) pair.
    RouteContext(tenantID, domainID string) EmbeddingProvider
}
```
`EmbeddingRouter` is the value injected into `workers.Runtime` and `search.Service`. The default `DirectRouter` wraps a single `EmbeddingProvider`. A `ProxyRouter` implementation can forward through an HTTP proxy, insert a cache layer, or A/B test between models — all without changing service code. The middleware chain pattern (Cache → Retry → Proxy → HTTP) is expressed by composing `EmbeddingProvider` implementations, not by modifying the router.

**VectorAdapter**:
```go
type VectorAdapter interface {
    Upsert(ctx context.Context, doc VectorDocument) error
    Delete(ctx context.Context, nodeID string) error
    ANN(ctx context.Context, query []float64, filter VectorFilter, opts ANNOptions) ([]VectorResult, error)
}

// ANNOptions carries query-time tuning parameters.
// Adding a new field is backward-compatible; adapters ignore fields they do not support.
type ANNOptions struct {
    TopK      int     // required: maximum results to return
    MinScore  float64 // optional: minimum similarity threshold (0.0 = disabled)
    IndexHint string  // optional: adapter-specific index hint, e.g. "hnsw", "ivfflat", "flat"
    EfSearch  int     // optional: HNSW ef_search (accuracy vs. speed tradeoff)
    // FilterMode controls ACL/domain filter placement:
    // "pre"  = filter before ANN (exact ACL, slower for small ACL sets)
    // "post" = filter after ANN (approx ACL, faster at scale)
    FilterMode string
}
```
Production: `PgVectorAdapter` (uses `pgvector` extension; stores vectors in a `kg_vector_documents` table with `hnsw` index).
Test: `InMemoryVectorAdapter` that wraps the existing `workers.VectorStore` map.
Adding fields to `ANNOptions` never breaks the interface; adapters that do not understand a field simply ignore it.

**GraphAdapter**:
```go
type GraphAdapter interface {
    UpsertNode(ctx context.Context, node GraphNode) error
    DeleteNode(ctx context.Context, nodeID string) error
    UpsertRelationship(ctx context.Context, rel GraphRelationship) error
    DeleteRelationship(ctx context.Context, relID string) error
    // ExecuteQuery accepts a structured GraphQuery, not a raw query string.
    // Each adapter translates GraphQuery into its native language
    // (Cypher for Neo4j/Memgraph, Gremlin for Neptune, AQL for ArangoDB).
    ExecuteQuery(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error)
}

// GraphQuery is the language-neutral representation of a graph traversal.
// The compiler produces a GraphQuery; the adapter translates it.
type GraphQuery struct {
    StartNodeType string
    StartMatch    map[string]any
    Hops          []GraphQueryHop
    ReturnFields  []string
    ACLTokensParam string  // name of the bound parameter carrying the caller's ACL tokens
    MaxDepth      int
    Strategy      string   // resolver hint: "default", "deep_traversal", or named key
}

type GraphQueryHop struct {
    RelType      string
    ToNodeType   string
    Direction    string  // "out" | "in" | "both"
    Filter       map[string]any
    FilterStatus string
}
```
Production: `Neo4jGraphAdapter` translates `GraphQuery` to Cypher. `MemgraphGraphAdapter` is identical since both speak Cypher; a future `NeptuneGraphAdapter` would translate to Gremlin/openCypher.
Test: `InMemoryGraphAdapter` that wraps the existing `workers.GraphStore`.
Switching graph databases requires only a new adapter; `read.QueryTemplateCompiler` and `workers.Runtime` never change.

**FTSAdapter**:
```go
type FTSAdapter interface {
    Index(ctx context.Context, doc FTSDocument) error
    Delete(ctx context.Context, nodeID string) error
    Search(ctx context.Context, query FTSQuery, filter FTSFilter) ([]FTSResult, error)
}

// FTSQuery uses backend-neutral mode names.
// Each adapter translates these into its native query form.
type FTSQuery struct {
    Text   string
    // Mode is backend-neutral:
    //   "all_tokens"  → Postgres `&`, Elasticsearch `must`, Meilisearch default
    //   "any_token"   → Postgres `|`, Elasticsearch `should`
    //   "phrase"      → Postgres `<->`, Elasticsearch `match_phrase`
    Mode   string
    Fields []string // optional: restrict FTS to specific fields
}
```
Production: `PgFTSAdapter` translates `Mode` to `tsquery` operators.
A future `ElasticFTSAdapter` translates the same `Mode` to Elasticsearch DSL.
Test: `InMemoryFTSAdapter` (simple substring + token match).
`FTSQuery` never carries Postgres-specific syntax; dialects stay inside their adapter.

### 2. Domain Search Profile stored in the ontology layer

Add `SearchProfile` to `ontology.Domain`. It is stored as a JSONB column on `kg_domains` (Postgres) and loaded by `ontology.Service`.

```go
type SearchProfile struct {
    // Semantic index fields: ordered list; earlier fields get higher weight
    SemanticFields    []IndexedField    `json:"semantic_fields"`
    // FTS: backend-neutral language hint (e.g. "english", "simple", "vi")
    FTSLanguage       string            `json:"fts_language"`
    // QueryStrategyRef is the key of a registered QueryStrategy in the ontology.
    // Built-in keys: "default", "deep_traversal". Custom keys reference stored strategies.
    QueryStrategyRef  string            `json:"query_strategy_ref"`
    // Per-tenant overrides keyed by tenant_id
    TenantOverrides   map[string]SearchProfileOverride `json:"tenant_overrides,omitempty"`
    // Per-app overrides keyed by "tenant_id:app_id"
    AppOverrides      map[string]SearchProfileOverride `json:"app_overrides,omitempty"`
}

type IndexedField struct {
    FieldName  string  `json:"field_name"`  // node property key or built-in: "id", "node_type", "external_ref"
    Weight     float64 `json:"weight"`      // multiplier for embedding text repetition; 1.0 = normal
    Prefix     string  `json:"prefix,omitempty"` // optional label prepended: "title: <value>"
}

type SearchProfileOverride struct {
    SemanticFields   []IndexedField `json:"semantic_fields,omitempty"`
    FTSLanguage      string         `json:"fts_language,omitempty"`
    QueryStrategyRef string         `json:"query_strategy_ref,omitempty"`
}

// QueryStrategy is a versioned, typed strategy object stored in the ontology.
// Separating it from SearchProfile means strategies can be shared across domains
// and updated independently without touching per-domain profiles.
type QueryStrategy struct {
    Key         string `json:"key"`          // unique name, referenced by QueryStrategyRef
    Version     int    `json:"version"`
    MaxDepth    int    `json:"max_depth"`    // traversal depth cap; ignored by flat strategies
    // Params holds strategy-specific tuning values (e.g. direction hints, index type).
    // New params never break the interface; the compiler ignores unknown keys.
    Params      map[string]any `json:"params,omitempty"`
}
```

`SearchProfile` and `QueryStrategy` are separate objects. A domain references a strategy by key; updating the strategy record propagates to all domains that reference it without touching their profile. New strategy variants are registered in the ontology layer — no code change required in the compiler.

**`SearchProfileResolver` interface** decouples the resolution algorithm from the struct:
```go
type SearchProfileResolver interface {
    Resolve(domainID, tenantID, appID string) (ResolvedSearchProfile, error)
}
```
The default implementation applies app override → tenant override → domain baseline → hardcoded defaults. A future implementation can pull overrides from a runtime config store, respect feature flags, or consult an external policy engine — all without changing `workers.Runtime` or `search.Service`.

The `workers.Runtime.projectNode` and `search.Service` call `SearchProfileResolver.Resolve` before building embedding text or executing FTS indexing.

### 3. Embedding text construction respects the search profile

Current `buildEmbeddingText` joins all fields with equal weight. With a `SearchProfile`, the function:
1. Resolves the effective `SemanticFields` list for the (domain, tenant, app) triple via `SearchProfileResolver.Resolve`
2. Builds text by appending each field value, repeated `ceil(weight)` times, with optional prefix
3. Falls back to the **system default field list** when no profile is configured (see below)

**System default `SemanticFields`** (used when `SearchProfile` is nil or `SemanticFields` is nil/empty):
```
id          weight=1.0  (built-in)
node_type   weight=1.0  (built-in)
domain_id   weight=1.0  (built-in)
external_ref weight=1.0 (built-in)
status_value weight=1.0 (built-in)
<all node.Properties keys> weight=1.0, no prefix
```
This exactly matches the current `buildEmbeddingText` behavior, so the change is backward-compatible.

**Nil vs. empty `SemanticFields`**:
- `nil` → use system defaults (same as no profile configured)
- `[]` (empty slice) → index nothing; treated as a configuration error and rejected at save time

**System default `QueryStrategy "default"`** (pre-seeded, immutable):
```json
{ "key": "default", "version": 1, "max_depth": 5,
  "params": { "direction": "out", "depth_mode": "fixed", "acl_predicate": "any_hop" } }
```
- `direction: "out"` — only follow outgoing relationships
- `depth_mode: "fixed"` — each hop is a distinct `GraphQueryHop`, no variable-length paths
- `acl_predicate: "any_hop"` — ACL WHERE clause applied at every hop

**System default `QueryStrategy "deep_traversal"`** (pre-seeded, immutable):
```json
{ "key": "deep_traversal", "version": 1, "max_depth": 10,
  "params": { "direction": "out", "depth_mode": "variable", "acl_predicate": "start_only" } }
```
- `depth_mode: "variable"` — compiler emits variable-length path `*1..max_depth`
- `acl_predicate: "start_only"` — ACL WHERE applied only on the start node

### 4. Full-text search is a first-class search mode

`search.Service` gains `FullTextSearch(actor, req FullTextSearchRequest)` and `HybridSearch(actor, req HybridSearchRequest)`.

`FullTextSearchRequest`:
```go
type FullTextSearchRequest struct {
    Query     string   `json:"query"`
    DomainIDs []string `json:"domain_ids"`
    TopK      int      `json:"top_k"`
    // Mode is backend-neutral: "all_tokens" (default) | "any_token" | "phrase"
    Mode      string   `json:"mode"`
    Fields    []string `json:"fields,omitempty"` // restrict to specific node properties
}
```

`HybridSearchRequest` embeds both request types plus a fusion weight:
```go
type HybridSearchRequest struct {
    Query          string   `json:"query"`
    DomainIDs      []string `json:"domain_ids"`
    TopK           int      `json:"top_k"`
    FTSOperator    string   `json:"fts_operator"`
    SemanticWeight float64  `json:"semantic_weight"` // 0.0–1.0; remainder goes to FTS
}
```

Fusion uses reciprocal rank fusion (RRF): `score(d) = Σ 1/(k + rank_i(d))` with k=60.

ACL, domain, lifecycle, and authority-score filtering is applied identically to existing `SemanticSearch`.

### 5. Query strategy customization per domain/tenant/app

`read.QueryTemplateCompiler.Compile` produces a `GraphQuery` (not a raw Cypher string). After this change:
- It accepts a resolved `QueryStrategy` object from the ontology layer (fetched via `QueryStrategyRef`)
- `"default"` strategy: existing behavior (depth ≤ 5, ACL predicate on each hop) — compiled to `GraphQuery` with `MaxDepth=5`
- `"deep_traversal"` strategy: `MaxDepth` from the strategy's `Params`; expressed as variable-length hops in `GraphQuery.Hops`
- Named strategy key: loaded from the ontology, which carries typed `Params` — no string parsing required

`GraphAdapter.ExecuteQuery` is the single execution point. Each adapter translates the `GraphQuery` struct into its native language:
- `Neo4jGraphAdapter` / `MemgraphGraphAdapter` → Cypher
- A future `NeptuneGraphAdapter` → Gremlin or openCypher
- `InMemoryGraphAdapter` → in-process node/relationship walk

**Switching graph databases never requires touching `read.QueryTemplateCompiler`.**

### 6. Sync correctness is maintained through the new adapters

`workers.Runtime.projectNode` is updated to call `GraphAdapter.UpsertNode` and `VectorAdapter.Upsert` (with embedding computed via `EmbeddingProvider`) instead of writing to the in-memory stores. ACL grant change handling and status cascade call `GraphAdapter.UpsertNode` for affected nodes and `VectorAdapter.Upsert` for updated documents. Reconciliation compares the source Postgres store against the real adapter snapshots.

### 7. Backward compatibility

- No public API surface changes for existing endpoints
- `DeterministicProvider`, `InMemoryVectorAdapter`, `InMemoryGraphAdapter`, `InMemoryFTSAdapter` remain the default wiring in tests
- Configuration keys `embedding.provider`, `vector.adapter`, `graph.adapter`, `fts.adapter` select the production implementations at startup

## Extensibility Model

The design is structured around three extension points. Each can be changed independently without touching service logic:

### Extension Point 1 — Adapter swap (backend replacement)
| Adapter | How to swap |
|---|---|
| `GraphAdapter` | Implement `UpsertNode`, `DeleteNode`, `UpsertRelationship`, `DeleteRelationship`, `ExecuteQuery`. Register at bootstrap. |
| `VectorAdapter` | Implement `Upsert`, `Delete`, `ANN`. Register at bootstrap. |
| `FTSAdapter` | Implement `Index`, `Delete`, `Search`. Register at bootstrap. |
| `EmbeddingProvider` | Implement `Embed`, `Dimensions`, `ModelID`. Register in `EmbeddingRouter`. |

No changes to `workers.Runtime`, `search.Service`, or `read.Service` are needed.

### Extension Point 2 — Middleware chain (proxy, cache, rate limit)
`EmbeddingRouter` wraps one or more `EmbeddingProvider` implementations. Middleware providers implement `EmbeddingProvider` and delegate to the next in chain:

```
search.Service → EmbeddingRouter
                    └─ CachingProvider(ttl=5m)
                         └─ RetryProvider(maxAttempts=3)
                              └─ ProxyHTTPProvider(url="https://proxy.internal/embed")
                                   └─ HTTPEmbeddingProvider(url="...", model="...")
```

Each layer is a separate struct implementing `EmbeddingProvider`. Adding or removing a layer requires only changing the bootstrap wiring.

### Extension Point 3 — Query strategy registry (customizable index/search logic)
`QueryStrategy` objects are stored in the ontology layer and referenced by key. Adding a new strategy:
1. Register a `QueryStrategy` record in the ontology (API call, no code change)
2. Reference it by key in the domain's `SearchProfile.QueryStrategyRef`
3. If the strategy needs new `Params` interpretation, register a `StrategyHandler` in the compiler — the existing handlers remain untouched

`SearchProfileResolver` is an interface; the resolution algorithm (precedence order, override merge rules) can be replaced without touching service code.

## Migration Notes

- `kg_vector_documents` table: new Postgres table with `nodeID text PK`, `embedding vector(N)`, `acl_visible_to text[]`, `domain_id text`, `is_deleted bool`, indexed with `ivfflat` or `hnsw`
- `kg_domains.search_profile` column: new nullable JSONB column on the existing domains table
- Full-text index: `GIN` index on a generated `tsvector` column on `kg_nodes`
- No existing data migration required for the first rollout (new tables/columns are empty; workers backfill on next reconciliation run)

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| LLM embedding API latency increases write latency | Embed asynchronously via outbox worker, not in the write-path transaction |
| pgvector `hnsw` index memory usage at scale | Expose `IndexHint` in `ANNOptions`; operators can switch to `ivfflat` or `flat` without code change |
| Neo4j driver dependency increases binary size | Gate Neo4j adapter behind a build tag (`neo4j`); default build uses in-memory adapter |
| Switching graph DB mid-flight breaks existing Cypher snippets | `GraphQuery` is the stable contract; old Cypher snippets are owned by the adapter, not by service code |
| Domain search profile misconfiguration silently degrades results | Validate `SearchProfile` on write; log resolved profile at DEBUG level per request |
| FTS language mismatch for non-English domains | Default to `simple` analyzer when `FTSLanguage` is unset; each adapter maps the hint to its own analyzer config |
| `EmbeddingRouter` middleware chain grows unbounded | Cap chain depth at 5; log chain structure at startup |
| `QueryStrategy` registry grows stale (unused strategies) | Add `last_used_at` tracking; surface unused strategies in the admin MCP tool |
