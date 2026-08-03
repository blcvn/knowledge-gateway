## ADDED Requirements

### Requirement: Write owner identity is verified before graph identity persistence

The KG Service MUST verify that the owner identity it is about to persist for a supported KG write
matches durable tenant/app records before graph identity metadata reaches PostgreSQL foreign-key
enforcement.

#### Scenario: First write uses the authenticated durable owner identity

- **GIVEN** a tenant and app were created through the supported provisioning flow
- **AND** the app authenticates successfully on `POST /v1/kg/write/nodes`
- **WHEN** the write path prepares to persist graph identity metadata
- **THEN** the `owner_tenant_id` and `owner_app_id` used for that persistence match the
  authenticated durable tenant/app identity
- **AND** the write does not fail on owner foreign-key violations caused by service-side identity
  drift

#### Scenario: Owner mismatch fails before raw foreign-key violation

- **GIVEN** the write path resolves an authenticated identity whose durable owner tenant or app row
  is missing or misaligned
- **WHEN** the request reaches owner-identity verification
- **THEN** the service rejects the write as a controlled identity-readiness contract failure
- **AND** it does not expose this mismatch first as a generic internal `500`

### Requirement: Write-path diagnostics identify owner-ID mismatches

The KG Service MUST make owner-ID mismatches on supported write flows directly diagnosable.

#### Scenario: Diagnostics reveal the failing owner identity mapping

- **GIVEN** a supported write request cannot satisfy the durable owner-ID contract
- **WHEN** the service reports the failure
- **THEN** troubleshooting artifacts identify the authenticated `tenant_id` and `app_id`
- **AND** they identify the owner IDs the write path attempted to validate or persist
- **AND** they distinguish tenant-row absence, app-row absence, and app-to-tenant mismatch
