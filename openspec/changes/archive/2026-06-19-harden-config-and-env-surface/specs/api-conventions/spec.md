## MODIFIED Requirements

### Requirement: Expose a public health endpoint

The KG Service MUST expose a public health endpoint for probes and basic reachability checks.

#### Scenario: Health endpoint returns only safe operational metadata

- **GIVEN** an unauthenticated caller requests `GET /healthz`
- **WHEN** the service responds
- **THEN** the response includes only non-sensitive operational metadata needed for probes or basic diagnostics
- **AND** the response does not expose raw DSNs, secrets, API keys, or credential-bearing connection strings
