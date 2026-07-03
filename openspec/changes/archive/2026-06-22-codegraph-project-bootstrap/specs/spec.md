# codegraph-project-bootstrap

## Requirements

### Requirement: The repo supports CodeGraph as a local navigation path
The project SHALL support CodeGraph indexing and local source-code exploration without depending on
`kg-service` backend integration.

#### Scenario: Local index is initialized for the repo
- WHEN a developer initializes CodeGraph for the repo
- THEN the repo SHALL contain the configuration needed to build and reuse the local CodeGraph index

### Requirement: Agent guidance prefers CodeGraph before manual file reading
The repo SHALL document that local structural navigation prefers CodeGraph tools before grep/read.

#### Scenario: Local structural query uses CodeGraph first
- GIVEN a caller asks for callers, callees, impact, or exact symbol lookup in this repo
- WHEN local CodeGraph data is available
- THEN the recommended workflow SHALL prefer CodeGraph local tools before manual file reads

### Requirement: Bootstrap guidance is reusable for other projects
The project SHALL include a user guide describing how to apply the same CodeGraph bootstrap pattern
to another Go project.

#### Scenario: Another project follows the same bootstrap guide
- WHEN a developer reads the user guide
- THEN they SHALL be able to reproduce the setup pattern on another comparable repository
