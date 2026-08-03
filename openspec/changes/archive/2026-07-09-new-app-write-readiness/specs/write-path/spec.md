## ADDED Requirements

### Requirement: First writes from newly created apps do not fail on hidden owner FK gaps

The KG Service MUST treat tenant/app write readiness as a prerequisite of the supported write path
and MUST NOT expose newly created apps to opaque internal errors caused only by missing durable
owner records.

#### Scenario: First node write persists owner identity metadata successfully

- **GIVEN** a tenant and app were created through the supported provisioning flow
- **AND** the app authenticates successfully on `POST /v1/kg/write/nodes`
- **WHEN** the write path persists graph identity metadata such as `owner_tenant_id` and
  `owner_app_id`
- **THEN** the persistence succeeds without foreign-key violations caused by missing tenant/app
  owner rows

#### Scenario: Owner readiness mismatch is surfaced as a controlled contract failure

- **GIVEN** the authenticated identity does not have the durable owner records required by the
  PostgreSQL-backed write plane
- **WHEN** the caller attempts a supported KG write
- **THEN** the service detects that mismatch without leaking a raw backend FK failure as a generic
  `500`
- **AND** the response identifies the problem as a service-side identity/readiness contract issue
  rather than a caller payload-shape problem

### Requirement: Write-time ontology validation resolves from durable control-plane state

The KG Service MUST validate write requests against ontology metadata sourced from relationship DB
or caches derived from it.

#### Scenario: Write path uses durably persisted ontology metadata

- **GIVEN** a domain and its node or relationship schemas were provisioned successfully
- **WHEN** a supported KG write targets that ontology
- **THEN** the write path resolves the required domain and schema metadata from relationship DB
  state or a cache rebuilt from it
- **AND** the request does not depend on process-local ontology seeding to remain valid
