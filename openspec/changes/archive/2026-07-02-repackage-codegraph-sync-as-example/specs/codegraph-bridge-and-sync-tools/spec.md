## ADDED Requirements

### Requirement: The CodeGraph bridge is packaged as a repository-owned example

The repository SHALL publish the implemented CodeGraph-to-KG bridge as a runnable example under
`examples/codegraph/`.

#### Scenario: Contributors can find the canonical bridge implementation in examples

- **GIVEN** the repository ships a local CodeGraph bridge for `kg-service`
- **WHEN** a contributor looks for the implementation entrypoint, tests, or example configuration
- **THEN** the canonical bridge code SHALL live under `examples/codegraph/`
- **AND** the example SHALL include the runnable entrypoints and supporting tests needed for the
  documented bridge workflow

#### Scenario: Documentation treats the bridge as an example rather than a core root package

- **GIVEN** the repository documents CodeGraph sync and MCP workflows
- **WHEN** those active docs reference the implemented bridge
- **THEN** they SHALL describe it as a repository-owned example for `kg-service`
- **AND** they SHALL point to `examples/codegraph/` as the canonical implementation location

### Requirement: Example packaging preserves the current bridge workflow

Moving the bridge under `examples/codegraph/` SHALL NOT remove the documented build, sync, dry-run,
or MCP workflows used by contributors and validation automation.

#### Scenario: Repository-owned commands still expose the bridge lifecycle

- **GIVEN** contributors use repository-owned commands and scripts for the CodeGraph bridge
- **WHEN** the example packaging change is applied
- **THEN** the repository SHALL still provide repeatable entrypoints for build, sync, dry-run sync,
  and MCP server startup
- **AND** those entrypoints SHALL execute the bridge from its canonical example location
