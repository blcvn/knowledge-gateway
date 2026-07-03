## MODIFIED Requirements

### Requirement: Keep usage guidance aligned with the API contract

The KG Service MUST maintain user-facing guides in alignment with the published API specification,
live implementation, and canonical repository-owned examples.

#### Scenario: Workflow guides reference the authoritative API spec and current example locations

- **GIVEN** the API spec is the authoritative contract for request and response details
- **WHEN** user-facing guides show integration workflows or local bridge examples
- **THEN** they SHALL reference the API spec for exact payload details
- **AND** they SHALL point to the current canonical example location in the repository rather than a
  stale historical path

#### Scenario: Integration changes update guides in the same workstream

- **GIVEN** onboarding, bridge packaging, or integration behavior changes materially
- **WHEN** the change is prepared for review or merge
- **THEN** the relevant user-facing guides are updated in the same workstream
- **AND** the documented workflows remain aligned with the live service behavior and current example
  layout
