## ADDED Requirements

### Requirement: Publish deployment guidance for supported runtime targets

The KG Service MUST publish operator-facing deployment guidance for Docker Compose, Kubernetes, and VM-based deployments.

#### Scenario: Compose deployment path is documented

- **GIVEN** an operator wants the simplest supported deployment path
- **WHEN** they open the deployment docs
- **THEN** they can find a Docker Compose deployment flow
- **AND** the flow explains how to start the service and verify that it is healthy

#### Scenario: Kubernetes deployment path is documented

- **GIVEN** an operator is deploying the service into a Kubernetes cluster
- **WHEN** they open the deployment docs
- **THEN** they can find a Kubernetes deployment flow
- **AND** the flow explains the expected configuration inputs and verification steps

#### Scenario: VM deployment path is documented

- **GIVEN** an operator is deploying the service onto a standalone VM
- **WHEN** they open the deployment docs
- **THEN** they can find a VM deployment flow
- **AND** the flow explains how to install, start, and verify the service on that host

### Requirement: Provide repeatable deployment entrypoints

The KG Service MUST provide executable scripts or entrypoints that make the supported deployment paths repeatable.

#### Scenario: Each supported target has a repository-owned deployment entrypoint

- **GIVEN** an operator chooses Compose, Kubernetes, or VM deployment
- **WHEN** they run the documented entrypoint for that target
- **THEN** the repository provides a consistent way to deploy the service
- **AND** the entrypoint uses the documented configuration surface rather than ad hoc local steps

#### Scenario: Deployment documentation aligns with runtime prerequisites

- **GIVEN** the deployment docs describe required environment variables, ports, or external dependencies
- **WHEN** an operator follows the guidance
- **THEN** the documented prerequisites match the actual runtime expectations
- **AND** bootstrap-only conveniences are clearly labeled as such

### Requirement: Provide a repository-owned command and build interface

The KG Service MUST expose a minimal repository-owned command surface and a `Makefile` that standardizes common build and run actions.

#### Scenario: The service starts through an explicit serve command

- **GIVEN** an operator or contributor runs the service binary
- **WHEN** they inspect the command interface
- **THEN** they can see a `serve` command for starting the HTTP server
- **AND** the repository documents that the deployment paths invoke that command explicitly where needed

#### Scenario: Common actions are available through Makefile targets

- **GIVEN** a contributor or operator wants a repeatable local workflow
- **WHEN** they use the repository `Makefile`
- **THEN** they can build, run, deploy, migrate, and validate the service through documented targets
- **AND** those targets wrap the same command and scripts used by the deployment paths
