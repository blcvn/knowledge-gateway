## ADDED Requirements

### Requirement: Relationship DB is authoritative for identity and access state

The KG Service MUST use relationship DB as the durable source of truth for tenant, app, grant, and
access-audit data in supported runtimes.

#### Scenario: Tenant and app provisioning writes durable relationship DB rows

- **GIVEN** an authorized caller creates a tenant or app through the supported access APIs
- **WHEN** the request succeeds
- **THEN** the resulting tenant or app exists durably in relationship DB tables that the rest of the
  service depends on
- **AND** supported runtime behavior does not rely on process-local memory as the only persistent
  home of that identity

#### Scenario: Redis and memory do not replace durable access state

- **GIVEN** the service uses Redis or in-process memory for identity or ACL acceleration
- **WHEN** the runtime serves authentication or authorization decisions
- **THEN** those layers behave as caches derived from relationship DB state
- **AND** the service remains correct after cache loss, restart, or cache rebuild

### Requirement: Newly created apps are write-ready when returned as active

The KG Service MUST ensure that any tenant/app identity returned as active by the supported
provisioning APIs is already compatible with the durable tenant/app ownership records required by
the PostgreSQL-backed write plane.

#### Scenario: Created app is immediately write-ready for supported writes

- **GIVEN** an authorized caller creates a tenant and then creates an app through the supported
  tenant/app APIs
- **WHEN** that app authenticates with the API key returned by the service
- **THEN** the resolved `tenant_id` and `app_id` correspond to durable ownership records recognized
  by the write plane
- **AND** the app can proceed to a supported KG write without requiring an undocumented identity
  registration step

#### Scenario: Access identity does not drift from durable ownership identity

- **GIVEN** an app is accepted by authentication on a protected KG write endpoint
- **WHEN** the request reaches write-path ownership persistence
- **THEN** the service does not discover for the first time that the tenant or app is missing from
  the authoritative durable registry
- **AND** supported onboarding flows do not split identity between access-only and write-capable
  stores
