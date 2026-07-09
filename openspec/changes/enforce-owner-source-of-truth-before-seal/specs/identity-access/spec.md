## ADDED Requirements

### Requirement: Canonical owner identity flows from source of truth into derived caches and copies

The KG Service MUST treat the PostgreSQL-backed tenant/app registry as the authoritative source of
truth for owner identity in supported runtimes. Redis caches, in-memory views, request context, and
session copies MUST be derived from that source and MUST NOT become independent owner registries.

#### Scenario: Provisioning synchronizes derived identity state from canonical tenant/app rows

- **GIVEN** an authorized caller creates or mutates a tenant/app through a supported access flow
- **WHEN** the request succeeds
- **THEN** the canonical PostgreSQL tenant/app rows are the committed owner state for that identity
- **AND** any Redis or in-memory identity state exposed to later auth or write flows is refreshed or
  invalidated from those canonical rows
- **AND** the service does not return the identity as active while a stale derived owner mapping is
  still preferred

#### Scenario: Cache-first auth falls back to source of truth when confidence is lost

- **GIVEN** identity resolution prefers a cached tenant/app entry for performance
- **WHEN** the cache misses, cannot be trusted, or is contradicted by canonical owner verification
- **THEN** the service reloads tenant/app identity from the canonical PostgreSQL registry
- **AND** it refreshes or evicts the stale derived cache entry
- **AND** it does not continue protected write handling based only on the stale cached identity

#### Scenario: Session copies remain derived from canonical owner identity

- **GIVEN** a protected request carries tenant/app identity in request context or PostgreSQL session
  state
- **WHEN** the write flow needs owner verification
- **THEN** those values are treated as request-scoped copies derived from canonical owner state
- **AND** the service does not treat the session copy itself as the authoritative owner registry
- **AND** any detected mismatch triggers canonical revalidation before the write can continue
