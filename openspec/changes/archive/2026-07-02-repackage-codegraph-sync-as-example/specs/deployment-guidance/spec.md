## MODIFIED Requirements

### Requirement: Deployment documentation aligns with runtime prerequisites

The deployment docs MUST match the actual runtime prerequisites and repository-owned automation used
by supported environments.

#### Scenario: CodeGraph runtime guidance follows the canonical example packaging

- **GIVEN** the deployment docs describe required environment variables, ports, external dependencies,
  or repository-owned validation entrypoints
- **WHEN** an operator follows the guidance
- **THEN** the documented prerequisites SHALL match the actual runtime expectations
- **AND** bootstrap-only conveniences SHALL be clearly labeled as such
- **AND** maintained CodeGraph runtime guidance SHALL reference the current canonical example path and
  state-file location used by repository-owned automation
