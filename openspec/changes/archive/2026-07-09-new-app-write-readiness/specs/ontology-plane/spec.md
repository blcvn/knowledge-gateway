## ADDED Requirements

### Requirement: Relationship DB is authoritative for ontology metadata

The KG Service MUST use relationship DB as the durable source of truth for ontology domains,
versions, schemas, cross-domain rules, query templates, status configurations, search profiles, and
query strategies in supported runtimes.

#### Scenario: Ontology writes persist durably before runtime serves them

- **GIVEN** an authorized caller creates or updates ontology metadata through supported APIs or
  bootstrap flows
- **WHEN** the operation succeeds
- **THEN** the resulting ontology state is durably stored in relationship DB
- **AND** later runtime reads can reconstruct that same state without depending on process-local
  memory

#### Scenario: Ontology reads remain correct after restart or cache loss

- **GIVEN** ontology metadata has been provisioned successfully
- **WHEN** the service restarts or Redis and in-memory caches are empty
- **THEN** ontology resolution for write, read, search, integrity, and MCP flows still works from
  relationship DB-backed state

### Requirement: Memory and Redis act only as ontology caches

The KG Service MUST treat in-process ontology stores and Redis-backed ontology accelerators as
cache layers rather than authoritative data stores in supported runtimes.

#### Scenario: Cache miss refills from durable ontology state

- **GIVEN** a domain, schema, or query template is requested and the cache is empty
- **WHEN** the service reloads the requested ontology metadata
- **THEN** it reloads from relationship DB
- **AND** any rebuilt cache content remains derived from that durable source
