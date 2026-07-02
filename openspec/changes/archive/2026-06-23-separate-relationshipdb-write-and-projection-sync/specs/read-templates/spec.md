## MODIFIED Requirements

### Requirement: Graph-backed template execution

The KG Service MUST execute graph reads for `app_id` + `kg_id` scoped entities through the graph query path,
with explicit `realtime` and `non-realtime` read modes.

#### Scenario: Non-realtime read always uses graphdb

- **GIVEN** a caller requests a graph read in `non-realtime` mode
- **WHEN** the target entity is identified by the caller's app scope and `kg_id`
- **THEN** the service SHALL execute the read against `graphdb`
- **AND** SHALL NOT fall back to `relationshipdb` only because the projection may be stale

#### Scenario: Realtime read uses graphdb only when projection version matches source version

- **GIVEN** a caller requests a graph read in `realtime` mode
- **AND** the target entity is identified by the caller's app scope and `kg_id`
- **WHEN** the service compares the entity's `graphdb` projection version with its source version in `relationshipdb`
- **THEN** the service SHALL return the `graphdb` result only when those versions are equal

#### Scenario: Realtime read falls back to relationshipdb when graphdb is behind

- **GIVEN** a caller requests a graph read in `realtime` mode
- **AND** the entity exists in `relationshipdb`
- **WHEN** the entity's `graphdb` projection version is lower than the source version in `relationshipdb`, or the graph projection is missing
- **THEN** the service SHALL return the source-backed result from `relationshipdb`
- **AND** SHALL treat that response as a freshness-preserving fallback rather than a write failure

### Requirement: Backend-enforced query safeguards

The KG Service MUST preserve ACL, lifecycle, timeout, and row-limit safeguards regardless of whether the
final read result comes from `graphdb` or a `relationshipdb` fallback in `realtime` mode.

#### Scenario: Realtime fallback still enforces read constraints

- **GIVEN** a `realtime` read falls back from `graphdb` to `relationshipdb`
- **WHEN** the fallback result is built
- **THEN** the same caller visibility and lifecycle constraints SHALL still apply
- **AND** the service SHALL continue to enforce the documented timeout and bounded-result behavior
