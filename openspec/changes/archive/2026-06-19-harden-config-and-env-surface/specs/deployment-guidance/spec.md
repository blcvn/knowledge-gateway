## MODIFIED Requirements

### Requirement: Provide repeatable deployment entrypoints

The KG Service MUST provide executable scripts or entrypoints that make the supported deployment paths repeatable.

#### Scenario: Deployment documentation aligns with runtime prerequisites

- **GIVEN** the deployment docs describe required environment variables, ports, or external dependencies
- **WHEN** an operator follows the guidance
- **THEN** the documented prerequisites match the actual runtime expectations
- **AND** bootstrap-only conveniences are clearly labeled as such
- **AND** deployment manifests and scripts pass through the variables required by the selected runtime profile, including graph database selection when applicable

### Requirement: Publish operator-facing configuration inventory

The KG Service MUST publish an operator-facing inventory of supported environment variables for deployment and runtime startup.

#### Scenario: Operators can discover the environment contract in one place

- **GIVEN** an operator needs to configure `kg-service` for Compose, Kubernetes, or VM deployment
- **WHEN** they open the deployment guidance
- **THEN** they can find one repository-owned inventory of supported environment variables
- **AND** the inventory explains defaults, conditional requirements, and deployment notes for each variable
