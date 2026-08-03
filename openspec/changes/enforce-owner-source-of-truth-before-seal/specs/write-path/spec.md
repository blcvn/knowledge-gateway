## ADDED Requirements

### Requirement: Graph version sealing requires canonical owner revalidation

The KG Service MUST revalidate the tenant/app owner identity against the canonical PostgreSQL owner
registry before persisting graph identity metadata that depends on owner foreign keys.

#### Scenario: First write seals graph state only after canonical owner verification

- **GIVEN** a tenant and app were provisioned through the supported flow
- **AND** authentication resolved that app for a protected KG write
- **WHEN** the write path is about to persist or seal graph identity/version metadata
- **THEN** the tenant/app pair to be written is revalidated against the canonical PostgreSQL owner
  registry
- **AND** the owner pair used for graph persistence matches that canonical tenant/app identity
- **AND** the write does not rely on `kg_graph_identifiers` foreign-key rejection as the first
  owner-consistency check

#### Scenario: Stale cached identity fails back before graph version sealing

- **GIVEN** a cached identity entry points to a tenant/app mapping that no longer matches the
  canonical owner registry
- **WHEN** a supported KG write reaches owner verification before graph-version sealing
- **THEN** the service reloads canonical owner state and refreshes or evicts the stale cache entry
- **AND** it either continues with the canonical owner pair or rejects the request with a controlled
  readiness/sync error
- **AND** it does not proceed to a raw FK-driven `500` for this mismatch class
