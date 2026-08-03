## ADDED Requirements

### Requirement: Authenticated write identities resolve to durable owner records

The KG Service MUST ensure that any tenant/app identity accepted on supported KG write endpoints
resolves to the same durable tenant/app ownership records expected by the write plane.

#### Scenario: Created app resolves to the same owner identity used by writes

- **GIVEN** an authorized caller creates an app through `POST /v1/tenants/{tenant_id}/apps`
- **WHEN** that app later authenticates on a protected KG write endpoint
- **THEN** the resolved `tenant_id` and `app_id` are the exact durable owner IDs the write plane
  will use
- **AND** the service does not require an undocumented repair or registration step before the first
  supported write

#### Scenario: Access and write identity divergence is surfaced explicitly

- **GIVEN** authentication succeeds for an app identity
- **AND** the corresponding durable owner tenant/app records are missing or inconsistent for the
  write plane
- **WHEN** the request attempts a supported KG write
- **THEN** the service surfaces that condition as an identity-readiness mismatch
- **AND** it does not treat the caller as fully write-ready just because authentication passed
