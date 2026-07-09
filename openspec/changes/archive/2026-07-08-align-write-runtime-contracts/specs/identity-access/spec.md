## ADDED Requirements

### Requirement: Provision authenticated write callers as durable app identities

The KG Service MUST ensure that any app identity accepted on protected KG write endpoints is
compatible with the durable app registry and storage constraints used by the PostgreSQL-backed write
plane.

#### Scenario: Created app is write-ready after onboarding

- **GIVEN** an authorized admin creates an app through `POST /v1/tenants/{tenant_id}/apps`
- **WHEN** that app later calls a protected KG write endpoint with its active API key
- **THEN** the service resolves the request as the created `tenant_id` and `app_id`
- **AND** the write path can persist graph identity metadata without failing on app UUID or
  app-foreign-key mismatches

#### Scenario: Seeded write-capable credential is backed by a durable app row

- **GIVEN** the local runtime documents a seeded credential as valid for KG write flows
- **WHEN** that credential reaches graph-version or sync-session persistence
- **THEN** its resolved `app_id` is compatible with the durable `apps` registry used by the write
  schema
- **AND** local fixtures provide any required app row backing for write-path constraints

#### Scenario: Access and write identity stores do not diverge for supported flows

- **GIVEN** an app is active and authenticated successfully
- **WHEN** it reaches a supported write flow such as opening a sync session or creating a node
- **THEN** the service does not depend on a second unseen app-registration step outside the
  documented onboarding flow
- **AND** the caller is not exposed to an internal error caused only by identity-store mismatch
