## ADDED Requirements

### Requirement: Build effective ontology from visible domains

The KG Service MUST compute an app's effective ontology from platform-owned domains, tenant-owned domains, and explicitly shared domains visible to that app.

#### Scenario: Tenant sees platform and owned domains

- **GIVEN** a tenant owns at least one domain and the platform tenant owns shared baseline domains
- **WHEN** the tenant requests its effective ontology
- **THEN** the response includes platform-visible domains and the tenant's own domains

#### Scenario: Shared domain is added to effective ontology

- **GIVEN** a domain owner grants another tenant access to a domain
- **WHEN** the grantee requests effective ontology after grant propagation
- **THEN** the granted domain appears in the effective ontology

#### Scenario: `GET /v1/tenants/{tenant_id}/ontology/effective` returns only visible domains

- **GIVEN** a tenant has a mix of owned, shared, and inaccessible domains in the system
- **WHEN** it calls `GET /v1/tenants/{tenant_id}/ontology/effective`
- **THEN** the response includes only platform, owned, and granted domains visible to that tenant/app context

### Requirement: Validate node and relationship definitions against registered schemas

The KG Service MUST reject writes that reference domains, node types, or relationship types not declared in the effective ontology.

#### Scenario: Unknown node type is rejected

- **GIVEN** a write request targets a domain in the effective ontology
- **WHEN** the node type is not registered for that domain
- **THEN** the service rejects the request as a validation error

#### Scenario: Required property is missing

- **GIVEN** a node type schema declares required properties
- **WHEN** the write payload omits one of those properties
- **THEN** the service rejects the request with validation details

#### Scenario: Validation rule fails

- **GIVEN** a node type schema includes a domain validation rule
- **WHEN** the payload violates that rule
- **THEN** the write is rejected before persistence

### Requirement: Support generic cross-domain relationship rules

The KG Service MUST enforce configured cross-domain relationship requirements without hardcoding domain-specific relationship names into service logic.

#### Scenario: Required bridge property is missing

- **GIVEN** a domain has an active required cross-domain relationship rule
- **WHEN** a node write omits the corresponding bridge reference field
- **THEN** the service rejects the write as invalid

#### Scenario: Bridge relationships are created from rule-driven properties

- **GIVEN** a node write includes bridge identifiers that satisfy a configured cross-domain rule
- **WHEN** the write transaction succeeds
- **THEN** the service persists the node
- **AND** it creates the corresponding graph relationships defined by the rule

### Requirement: Register query templates through a safe DSL

The KG Service MUST store graph-read templates as validated DSL definitions rather than raw graph query text.

#### Scenario: Valid template is registered

- **GIVEN** a tenant admin submits a query template DSL document for a visible domain
- **WHEN** the DSL passes schema and safety validation
- **THEN** the service stores the template with versioned metadata

#### Scenario: Raw Cypher is not accepted as a template definition

- **GIVEN** a tenant admin attempts to register raw Cypher instead of the supported DSL structure
- **WHEN** the ontology API validates the payload
- **THEN** the request is rejected

#### Scenario: Template traversal depth is bounded

- **GIVEN** a template definition includes more hops than the service safety limit
- **WHEN** the template is submitted for registration
- **THEN** the service rejects it as too complex

#### Scenario: `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates` stores a draft template

- **GIVEN** a valid DSL payload is submitted for a visible domain
- **WHEN** the caller posts it to `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates`
- **THEN** the service stores the template in draft status with version metadata

#### Scenario: `PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate` promotes a valid draft

- **GIVEN** a draft template exists and has passed validation
- **WHEN** the caller activates it through `PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate`
- **THEN** the template becomes available for execution through the generic read API

### Requirement: Model lifecycle and ranking behavior as domain configuration

The KG Service MUST store lifecycle/status semantics and authority/ranking semantics as domain-level configuration rather than service code constants.

#### Scenario: Domain with status configuration exposes valid values

- **GIVEN** a domain registers a status field configuration
- **WHEN** downstream services read domain configuration
- **THEN** they can determine which field holds lifecycle state
- **AND** which values are valid, warning, or excluded

#### Scenario: Domain without status configuration remains supported

- **GIVEN** a domain registers no lifecycle status configuration
- **WHEN** its data is read or searched
- **THEN** the service treats status-based filtering as a no-op for that domain

#### Scenario: `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config` registers lifecycle semantics

- **GIVEN** a tenant admin defines the domain lifecycle field, valid values, and optional ranking map
- **WHEN** the payload is submitted to `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config`
- **THEN** the configuration becomes available to read, search, and sync components for that domain

### Requirement: Return consistent ontology API responses and validation errors

The KG Service MUST return predictable success payloads and validation errors for ontology-management endpoints.

#### Scenario: `POST /v1/tenants/{tenant_id}/ontology/domains` returns created domain metadata

- **GIVEN** a valid domain-create payload is submitted by an authorized caller
- **WHEN** `POST /v1/tenants/{tenant_id}/ontology/domains` succeeds
- **THEN** the service returns `201 Created`
- **AND** the response includes the created domain identifier, owner tenant, status, and version metadata

#### Scenario: Invalid query template DSL returns validation errors

- **GIVEN** a caller submits malformed or unsafe DSL to `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates`
- **WHEN** the service validates the payload
- **THEN** it returns `422 Unprocessable Entity`
- **AND** the response identifies the invalid fields or rule violations

#### Scenario: Inaccessible domain returns forbidden or not found

- **GIVEN** a caller targets a domain outside its manageable scope
- **WHEN** it invokes an ontology-management endpoint for that domain
- **THEN** the service returns `403 Forbidden` or `404 Not Found` according to the service visibility policy
