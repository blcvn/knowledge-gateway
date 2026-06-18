# graph-db-adapter

## Requirements

### Requirement: GraphAdapter interface must not expose a query language
The system SHALL define a `GraphAdapter` interface whose query execution method accepts a structured `GraphQuery` value, not a raw query string, so that the adapter — not the caller — owns the translation to the target database's native language.

#### Scenario: Switching from Memgraph to Neo4j
- GIVEN `workers.Runtime` and `read.Service` call `GraphAdapter.ExecuteQuery(ctx, query, params)`
- WHEN the active `GraphAdapter` implementation is replaced with `Neo4jGraphAdapter`
- THEN `workers.Runtime`, `read.Service`, and `read.QueryTemplateCompiler` SHALL require no code changes
- AND the new adapter SHALL translate `GraphQuery` to Cypher using the Neo4j Go driver

#### Scenario: Switching to a non-Cypher graph database
- GIVEN the same `GraphQuery` struct
- WHEN a `NeptuneGraphAdapter` (Gremlin) or `ArangoGraphAdapter` (AQL) is registered at bootstrap
- THEN `workers.Runtime` and `read.Service` SHALL be unaware of the language change
- AND only the new adapter file needs to be added

### Requirement: GraphAdapter operations
The system SHALL define the following operations on `GraphAdapter`:
- `UpsertNode(ctx, GraphNode) error` — idempotent node create/update
- `DeleteNode(ctx, nodeID string) error` — soft or hard delete depending on adapter
- `UpsertRelationship(ctx, GraphRelationship) error` — idempotent
- `DeleteRelationship(ctx, relID string) error`
- `ExecuteQuery(ctx, GraphQuery, params map[string]any) ([]map[string]any, error)` — structured traversal

#### Scenario: Upsert a graph node
- WHEN `workers.Runtime` processes a `NODE_UPSERTED` event
- THEN it SHALL call `GraphAdapter.UpsertNode` with the node payload and ACL metadata
- AND the upsert SHALL be idempotent (second call with the same node ID converges to the same state)

#### Scenario: Execute a compiled query template
- WHEN `read.Service` executes a query template
- THEN `read.GraphIndex` SHALL call `GraphAdapter.ExecuteQuery` with a `GraphQuery` produced by `read.QueryTemplateCompiler`
- AND the adapter SHALL return result rows as `[]map[string]any`

#### Scenario: ACL enforcement is expressed in GraphQuery, not in adapter code
- WHEN `GraphAdapter.ExecuteQuery` is called
- THEN the ACL token predicate SHALL be part of the `GraphQuery` struct (via `ACLTokensParam`)
- AND the adapter SHALL bind the caller's visible tokens as a query parameter, never interpolating them into the query string
- AND ACL enforcement SHALL not be re-implemented inside individual adapter files

#### Scenario: Unknown Strategy value in GraphQuery
- WHEN `GraphAdapter.ExecuteQuery` receives a `GraphQuery` with `Strategy` set to an unrecognized key
- THEN the adapter SHALL emit a WARNING log with the unknown key
- AND SHALL execute the query using the `"default"` strategy behavior (`MaxDepth=5`, `depth_mode="fixed"`, `acl_predicate="any_hop"`)
- AND SHALL NOT return an error; the query proceeds with safe defaults

### Requirement: InMemoryGraphAdapter for tests
The system SHALL provide an `InMemoryGraphAdapter` that implements `GraphAdapter` by walking the existing `workers.GraphStore` maps so that all current tests pass without an external graph database.

`InMemoryGraphAdapter.ExecuteQuery` SHALL implement the same traversal logic as the current `ProjectionGraphIndex.walkTemplate`, driven by the `GraphQuery` struct rather than a Cypher string.

### Requirement: Neo4jGraphAdapter / MemgraphGraphAdapter for production
The system SHALL provide a `Neo4jGraphAdapter` (and an identical `MemgraphGraphAdapter`) that translates `GraphQuery` to Cypher and executes it via the official Go driver. Both SHALL be compiled only when the `neo4j` build tag is present.

The Cypher translation layer SHALL be a standalone function `graphQueryToCypher(query GraphQuery, params map[string]any) (string, map[string]any)` so it can be unit-tested without a live database.

### Requirement: Sync correctness via adapter
- WHEN `workers.Runtime.Reconcile` is called
- THEN it SHALL compare source Postgres records against the state reported by `GraphAdapter` (snapshot via `ListNodes` / `ListRelationships` on the adapter)
- AND SHALL report drift for any node or relationship missing from or stale in the adapter
- AND SHALL NOT read from the legacy in-memory `GraphStore` struct during reconciliation
