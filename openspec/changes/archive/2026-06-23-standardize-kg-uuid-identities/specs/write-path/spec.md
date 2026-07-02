## MODIFIED Requirements

### Requirement: Support idempotent cross-store identifiers

The KG Service MUST support a stable external reference for nodes when provided by domain conventions so downstream sync can upsert deterministically, while preserving canonical internal UUID identities for persisted records.

#### Scenario: Node with external reference reuses canonical UUID on repeated sync

- **GIVEN** a write includes a valid domain-defined external reference
- **AND** a node already exists for that external reference
- **WHEN** the service persists and later projects the node
- **THEN** downstream graph/vector sync SHALL update the existing logical node
- **AND** the canonical internal node identifier SHALL remain the same UUID across repeated sync runs

#### Scenario: New node receives canonical UUID identity

- **GIVEN** a write creates a new node that does not yet exist
- **WHEN** the service persists the node
- **THEN** the source-of-truth record SHALL receive a canonical UUID node identifier
- **AND** the response identity SHALL be UUID-compatible

#### Scenario: Missing external reference still uses canonical UUID

- **GIVEN** a node write omits an external reference
- **WHEN** the service projects the node
- **THEN** it still projects successfully using the canonical internal UUID identifier

### Requirement: Return consistent write API success and failure responses

The KG Service MUST return predictable success payloads and typed validation/authorization errors for write endpoints, including UUID-compatible canonical identities for service-owned entities.

#### Scenario: `POST /v1/kg/write/nodes` returns canonical UUID identity

- **GIVEN** a valid authorized node write succeeds
- **WHEN** `POST /v1/kg/write/nodes` completes
- **THEN** the service returns `202 Accepted` or `201 Created` according to implementation policy
- **AND** the response includes the canonical UUID node identifier and asynchronous processing status when applicable
