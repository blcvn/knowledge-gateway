# codegraph-ontology-bootstrap

## Requirements

### Requirement: Bootstrap script is generated and executable
A bootstrap script SHALL be generated that encapsulates all API calls to initialise the `code-graph` ontology domain.

#### Scenario: Script runs without errors
- GIVEN the bootstrap script `scripts/bootstrap-codegraph-ontology.sh` has been generated
- WHEN the script is executed against the target environment
- THEN all API calls SHALL complete with successful HTTP status codes and no error output

#### Scenario: Script is idempotent
- GIVEN the bootstrap script has already been run once
- WHEN the script is executed a second time
- THEN it SHALL complete without errors (upsert or check-before-create for existing entities)

### Requirement: `code-graph` is modeled as a first-class ontology domain
The system SHALL support a dedicated ontology domain `code-graph` for source-code entities.

#### Scenario: Initial code-graph domain supports the required node types
- WHEN the bootstrap script has been run and results are verified
- THEN the domain SHALL include node types `Function`, `Method`, `Struct`, `Interface`, `Package`, and `File`

#### Scenario: Initial code-graph domain supports the required relationship types
- WHEN the bootstrap script has been run and results are verified
- THEN the domain SHALL include relationship types `CALLS`, `IMPLEMENTS`, `CONTAINS`, `REFERENCES`, and `IMPORTS`

### Requirement: Search profile fields are backed by schema
The ontology SHALL reject a `code-graph` search profile that references fields not present in the
domain schema or built-in searchable fields.

#### Scenario: Unknown semantic field is rejected
- GIVEN a `code-graph` search profile references a field absent from the registered schema
- WHEN the profile is saved
- THEN the ontology API SHALL reject the request with a validation error

### Requirement: Code traversal is modeled through query templates
The baseline persistent graph traversal contract SHALL use active query templates in domain
`code-graph`.

#### Scenario: Active code_callers template exists
- WHEN the bootstrap script has been run and results are verified
- THEN the domain SHALL expose an active `code_callers` query template for persistent callers lookup

### Requirement: Verification confirms all bootstrapped entities
After the script runs, each created entity SHALL be verifiable via the read API.

#### Scenario: Verification reads back all entities
- GIVEN the bootstrap script has completed successfully
- WHEN verification queries are executed against the read API
- THEN domain, all node/relationship type schemas, search profile, and query templates SHALL return data matching the spec
