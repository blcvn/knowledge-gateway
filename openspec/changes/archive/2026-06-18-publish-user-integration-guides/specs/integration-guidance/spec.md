## ADDED Requirements

### Requirement: Publish user-facing usage guidance for the current service surface

The KG Service MUST publish user-facing documentation that explains how consumers can successfully use the currently implemented bootstrap service.

#### Scenario: New users can find a quickstart path

- **GIVEN** a new user opens the repository without prior service knowledge
- **WHEN** they look for usage documentation
- **THEN** the repository provides a clear quickstart path for local bootstrap usage
- **AND** that path explains how to start the service, authenticate, and make an initial successful request

#### Scenario: Consumer guidance is distinct from operator runbooks

- **GIVEN** the repository also contains architecture material and operations runbooks
- **WHEN** user-facing guides are published
- **THEN** the guides clearly distinguish consumer usage documentation from maintainer recovery documentation
- **AND** users can tell which documentation is intended for integration versus internal operations

### Requirement: Document end-to-end integration workflows

The KG Service MUST document the practical sequence of calls and decisions required for common integration workflows.

#### Scenario: Tenant and app onboarding flow is documented

- **GIVEN** an integrator wants to onboard a tenant and application
- **WHEN** they follow the integration guide
- **THEN** the guide explains the sequence for tenant creation, app creation, API-key usage, API-key rotation, and visibility resolution
- **AND** it points to the API spec for exact request and response schemas

#### Scenario: Ontology and data lifecycle flow is documented

- **GIVEN** an integrator wants to model a domain and publish knowledge data
- **WHEN** they follow the integration guide
- **THEN** the guide explains the sequence for defining ontology artifacts before write operations
- **AND** it explains how to create nodes and relationships, ingest documents, and validate resulting read or search behavior

#### Scenario: Read and search workflows are documented in user terms

- **GIVEN** an integrator wants to retrieve data after onboarding
- **WHEN** they follow the integration guide
- **THEN** the guide explains how to list templates, execute template reads, fetch nodes directly, run semantic search, and use RAG search
- **AND** it explains how visibility and ACL behavior affect observed results

### Requirement: Document MCP integration in relation to REST

The KG Service MUST document how the MCP transport fits alongside the REST API for consumers.

#### Scenario: MCP connection and message flow is documented

- **GIVEN** a consumer wants to integrate through MCP
- **WHEN** they read the user-facing integration guide
- **THEN** the guide explains session establishment and message-posting flow for the current HTTP transport
- **AND** it explains that MCP uses the same underlying authorization and validation semantics as REST

#### Scenario: Users can choose the appropriate interface

- **GIVEN** the service offers both REST and MCP interfaces
- **WHEN** the integration guide compares them
- **THEN** it explains when direct REST integration is the simpler choice
- **AND** it explains when MCP is the better fit for tool-oriented or agent-style consumers

### Requirement: Label bootstrap-only assumptions clearly

The KG Service MUST make bootstrap-specific onboarding details explicit so users do not confuse them with production guarantees.

#### Scenario: Seeded local credentials are identified as bootstrap behavior

- **GIVEN** local development uses seeded credentials and sample ontology data
- **WHEN** those details appear in user-facing guides
- **THEN** the guides label them clearly as bootstrap or local-development behavior
- **AND** the guides avoid presenting them as universal production provisioning behavior

#### Scenario: In-memory runtime limitations are surfaced to users

- **GIVEN** current bootstrap behavior still relies on in-memory or simplified runtime paths in some areas
- **WHEN** user-facing guides describe expected behavior
- **THEN** the guides note relevant bootstrap limitations where they materially affect evaluation or integration expectations

### Requirement: Keep usage guidance aligned with the API contract

The KG Service MUST maintain user-facing guides in alignment with the published API specification and live implementation.

#### Scenario: Workflow guides reference the authoritative API spec

- **GIVEN** the API spec is the authoritative contract for request and response details
- **WHEN** user-facing guides show integration workflows
- **THEN** they reference the API spec for exact payload details
- **AND** they avoid redefining those details inconsistently in prose

#### Scenario: Integration changes update guides in the same workstream

- **GIVEN** onboarding or integration behavior changes materially
- **WHEN** the change is prepared for review or merge
- **THEN** the relevant user-facing guides are updated in the same workstream
- **AND** the documented workflows remain aligned with the live service behavior
