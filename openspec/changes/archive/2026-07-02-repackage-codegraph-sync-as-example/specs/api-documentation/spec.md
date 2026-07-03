## MODIFIED Requirements

### Requirement: Keep the API specification synchronized with implementation changes

The KG Service MUST update the published API specification and companion API references whenever
runtime API behavior changes, and MUST refresh related documentation when repository-owned example
layouts materially change.

#### Scenario: Route changes update the published spec in the same workstream

- **GIVEN** a change adds, removes, or renames an HTTP route
- **WHEN** that change is prepared for review or merge
- **THEN** the published API specification SHALL be updated in the same workstream
- **AND** the route inventory SHALL remain aligned with the runtime router

#### Scenario: Companion API references stay aligned with active example guidance

- **GIVEN** the machine-readable and human-readable API docs link to surrounding guides and validation
  workflows
- **WHEN** a repository-owned integration example is repackaged or relocated
- **THEN** those companion API references SHALL be reviewed in the same workstream
- **AND** any maintained links or example-path references SHALL point to the canonical current layout
