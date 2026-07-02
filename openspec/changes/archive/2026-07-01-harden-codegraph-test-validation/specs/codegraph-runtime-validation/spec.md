## MODIFIED Requirements

### Requirement: Validation covers ontology bootstrap and CodeGraph queryability

The CodeGraph runtime validation flow MUST cover ontology readiness and both create and update
behavior against the running stack.

#### Scenario: Validation upserts CodeGraph data into KG backends on the first sync

- **GIVEN** the `code-graph` ontology is ready and the local CodeGraph index is available
- **WHEN** the repository-owned validation flow performs its initial data update step
- **THEN** it creates CodeGraph data in the `kg-service` runtime for a fresh probe symbol
- **AND** the created data is available for graph traversal and search validation checks

#### Scenario: Validation proves a rerun updates an existing symbol

- **GIVEN** the validation flow has already synced a probe symbol into domain `code-graph`
- **WHEN** the same flow mutates the probe symbol and runs sync again
- **THEN** the service SHALL update the same logical symbol instead of creating a duplicate
- **AND** repository-owned validation checks SHALL confirm the updated content is readable after sync

#### Scenario: Validation confirms the update advances graph version state

- **GIVEN** the validation flow can observe version metadata for the synced probe symbol or sync scope
- **WHEN** the probe symbol changes and the follow-up sync completes successfully
- **THEN** the observed graph version state SHALL differ from the pre-update value
- **AND** the validation flow SHALL fail if the content changes but the version signal does not advance
