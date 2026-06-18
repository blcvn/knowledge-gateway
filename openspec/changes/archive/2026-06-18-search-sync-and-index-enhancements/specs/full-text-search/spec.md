# full-text-search

## Requirements

### Requirement: Full-text search operation
The system SHALL expose a `FullTextSearch` operation on `search.Service` that accepts a keyword query and returns ranked results with correct ACL, domain, lifecycle, and authority-score filtering.

#### Scenario: Keyword search with all-tokens mode
- WHEN an actor calls `FullTextSearch` with `mode: "all_tokens"`
- THEN the system SHALL return only nodes whose indexed text contains all query tokens
- AND SHALL rank results by FTS relevance score (descending)
- AND SHALL apply the caller's ACL visibility, domain filter, and lifecycle status filter

#### Scenario: Keyword search with any-token mode
- WHEN an actor calls `FullTextSearch` with `mode: "any_token"`
- THEN the system SHALL return nodes whose indexed text contains any query token

#### Scenario: Phrase search
- WHEN an actor calls `FullTextSearch` with `mode: "phrase"`
- THEN the system SHALL return only nodes whose indexed text contains the query tokens in order

### Requirement: FTSQuery uses backend-neutral mode names
The system SHALL define `FTSQuery.Mode` as one of `"all_tokens"`, `"any_token"`, or `"phrase"` — never as Postgres operator syntax. Each `FTSAdapter` implementation translates these to its own query form.

| Mode | Postgres tsquery | Elasticsearch DSL | InMemory |
|---|---|---|---|
| `all_tokens` | `token1 & token2` | `must` | all tokens present |
| `any_token` | `token1 \| token2` | `should` | any token present |
| `phrase` | `token1 <-> token2` | `match_phrase` | substring match |

#### Scenario: Same FTSQuery drives a different FTS backend
- GIVEN `FTSQuery{Mode: "all_tokens", Text: "payment gateway"}` is issued
- WHEN the active `FTSAdapter` is swapped from `PgFTSAdapter` to a future `ElasticFTSAdapter`
- THEN `search.Service` SHALL require no code changes
- AND the new adapter SHALL translate `Mode` to the appropriate Elasticsearch DSL

### Requirement: FTSQuery supports field-level restriction
The system SHALL allow callers to restrict FTS to a subset of indexed fields via `FTSQuery.Fields`. When empty, all indexed fields are searched.

#### Scenario: Search restricted to title field only
- WHEN `FTSQuery.Fields = ["title"]`
- THEN `PgFTSAdapter` SHALL apply the query only to the `title` tsvector sub-weight
- AND other adapters SHALL apply field restriction using their own mechanism

### Requirement: Hybrid search combining FTS and semantic scores
The system SHALL expose a `HybridSearch` operation that fuses FTS and semantic search results using reciprocal rank fusion (RRF).

#### Scenario: Hybrid search returns fused results
- WHEN an actor calls `HybridSearch`
- THEN the system SHALL execute both `FullTextSearch` and `SemanticSearch` concurrently
- AND SHALL fuse their ranked lists using RRF with k=60
- AND the final ranking SHALL reflect the configured `SemanticWeight` parameter (0.0–1.0)
- AND ACL, domain, and lifecycle filtering SHALL be identical to the individual search modes

### Requirement: FTSAdapter interface
The system SHALL define an `FTSAdapter` interface with `Index`, `Delete`, and `Search` operations.

#### Scenario: FTS document indexed on node upsert
- WHEN `workers.Runtime` processes a `NODE_UPSERTED` event
- THEN it SHALL call `FTSAdapter.Index` with the node's searchable text fields
- AND the FTS language SHALL be resolved from the domain's `SearchProfile.FTSLanguage` via `SearchProfileResolver`

### Requirement: PgFTSAdapter for production
The system SHALL provide a `PgFTSAdapter` that:
- Stores FTS data in a `fts_vector` generated `tsvector` column on `kg_nodes` with a `GIN` index
- Constructs `tsquery` from `FTSQuery.Mode` using `plainto_tsquery` (`all_tokens`), `to_tsquery` with `|` (`any_token`), or `phraseto_tsquery` (`phrase`)
- Ranks results via `ts_rank_cd`
- Uses the language from `FTSLanguage` in the resolved `SearchProfile`; defaults to `"simple"` when unset

### Requirement: InMemoryFTSAdapter for tests
The system SHALL provide an `InMemoryFTSAdapter` that implements `FTSAdapter` using simple token matching, honoring `Mode` without any Postgres dependency, so that tests pass without a database connection.
