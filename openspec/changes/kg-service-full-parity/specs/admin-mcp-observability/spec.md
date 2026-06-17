# admin-mcp-observability

## ADDED Requirements

### Requirement: Parity between REST and MCP backend behavior
The system SHALL use the same underlying access, read, search, ontology, and integrity services for both REST and MCP surfaces.

#### Scenario: Compare REST and MCP visibility
- WHEN an actor resolves access through REST and through MCP
- THEN the returned visible owners and domain visibility SHALL match
- AND authorization failures SHALL map to the same semantic outcome.

### Requirement: Parity-friendly error mapping
The system SHALL preserve stable tool and endpoint error semantics while backend adapters are swapped to their production implementations.

#### Scenario: An invalid MCP tool call occurs
- WHEN the client submits a malformed or unauthorized MCP request
- THEN the service SHALL return the existing MCP error shape
- AND SHALL not leak backend implementation details.
