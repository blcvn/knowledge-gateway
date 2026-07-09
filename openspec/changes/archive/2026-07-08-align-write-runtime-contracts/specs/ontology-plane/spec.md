## ADDED Requirements

### Requirement: Distinguish visible domains from writable domains

The KG Service MUST treat ontology visibility and write authority as separate concerns.
A caller may see a domain in its effective ontology without being allowed to write into that
domain.

#### Scenario: Platform-owned baseline domain is visible but not tenant-writable by default

- **GIVEN** the platform tenant owns a baseline domain that appears in another tenant app's
  effective ontology
- **WHEN** that tenant app attempts a write into the platform-owned domain without a matching
  cross-tenant write grant
- **THEN** the service rejects the write as forbidden

#### Scenario: Tenant-owned domain is writable by the owning tenant

- **GIVEN** a tenant admin creates a domain under its own tenant
- **WHEN** an app from the same tenant writes into that domain through the supported KG write path
- **THEN** the service allows the write subject to the existing ontology schema validation rules

#### Scenario: Cross-tenant write grant enables shared-domain writes

- **GIVEN** a domain is owned by another tenant
- **AND** the caller has an active `write` or `admin` grant whose owner and scope match that domain
- **WHEN** the caller performs a KG write targeting that domain
- **THEN** the service allows the write subject to the existing ontology and payload validation
  rules

### Requirement: Document tenant-scoped ontology bootstrap for write integrations

The KG Service MUST document a tenant-owned ontology bootstrap path for integrations that intend to
write under a tenant's own authority, and MUST document the separate grant-backed path for
cross-tenant shared domains.

#### Scenario: Tenant-specific write integration uses tenant-owned domain bootstrap

- **GIVEN** an integration creates a new tenant/app pair and plans to write tenant-owned data
- **WHEN** it follows the documented onboarding and ontology bootstrap flow
- **THEN** the flow creates the write target domain under that tenant unless a shared foreign-owned
  domain is explicitly intended

#### Scenario: Shared-domain integration calls out explicit write delegation

- **GIVEN** an integration intentionally uses a domain owned by the platform tenant or another
  foreign tenant
- **WHEN** it follows the documented shared-domain path
- **THEN** the docs require an explicit write-capable grant before the tenant app is treated as
  allowed to write into that foreign-owned domain
