# graph-db-adapter

## MODIFIED Requirements

### Requirement: Neo4jGraphAdapter / MemgraphGraphAdapter for production

The system SHALL provide selectable production graph adapters for `neo4j` and `memgraph`, both translating `GraphQuery` to Cypher and executing it through backend-specific clients.

#### Scenario: Switch between Neo4j and Memgraph

- GIVEN `read.Service` and `workers.Runtime` depend only on `GraphAdapter`
- WHEN the configured graph backend changes from `neo4j` to `memgraph`
- THEN bootstrap and deployment configuration SHALL change
- AND `read.Service`, `workers.Runtime`, and `read.QueryTemplateCompiler` SHALL require no service-layer code changes

### Requirement: NebulaGraph adapter for non-Cypher deployments

The system SHALL provide a `NebulaGraphAdapter` that accepts the same `GraphQuery` input and translates it to NebulaGraph's query model without exposing that model to service-layer callers.

#### Scenario: Read template runs on NebulaGraph

- GIVEN a compiled `GraphQuery`
- WHEN `GraphAdapter` is configured as `nebula`
- THEN the adapter SHALL execute the traversal against NebulaGraph
- AND callers SHALL continue to pass a structured `GraphQuery`, not backend-specific query text

### Requirement: Graph adapters expose snapshot and version visibility

Production graph adapters SHALL support reconciliation by returning replica snapshots and per-entity sync-version metadata.

#### Scenario: Reconciliation checks a real graph backend

- GIVEN a configured production graph adapter
- WHEN `workers.Runtime.Reconcile` runs
- THEN it SHALL load nodes and relationships from that backend through `ListNodes` and `ListRelationships`
- AND it SHALL read replica sync version metadata for compared entities
- AND it SHALL NOT silently succeed with an empty `nil` snapshot
