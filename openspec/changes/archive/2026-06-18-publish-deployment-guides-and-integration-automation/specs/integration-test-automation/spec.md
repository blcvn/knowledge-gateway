## ADDED Requirements

### Requirement: Provide a repeatable integration test script

The KG Service MUST provide a repeatable integration test script that can validate a deployed instance.

#### Scenario: Operators can run the script against a deployed target

- **GIVEN** the service is already deployed in Compose, Kubernetes, or VM form
- **WHEN** an operator runs the integration test script with the target service address or equivalent configuration
- **THEN** the script executes the documented integration checks against that deployment
- **AND** the script returns a clear pass or fail result

#### Scenario: The integration script fails on core regressions

- **GIVEN** the deployed service has a health, authentication, or core workflow regression
- **WHEN** the integration test script runs
- **THEN** the script exits with a non-zero status
- **AND** it surfaces enough context for the operator to identify the failing step

### Requirement: Document the integration validation contract

The KG Service MUST document what the integration test script validates and what it requires from the operator.

#### Scenario: Operators know the prerequisites before running validation

- **GIVEN** an operator wants to run post-deploy validation
- **WHEN** they read the integration test docs
- **THEN** they can see the required inputs such as base URL, credentials, or environment variables
- **AND** they can see which service behaviors are covered by the validation

#### Scenario: The same validation flow works across supported deployments

- **GIVEN** the service can run on Compose, Kubernetes, or a VM
- **WHEN** the operator uses the integration test script for any of those targets
- **THEN** the validation flow stays the same
- **AND** only the target connection details change
