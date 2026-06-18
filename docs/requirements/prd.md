# Product Requirements Document

## 1. Product Summary

`kg-service` is a multi-tenant knowledge graph service. It accepts authenticated API requests, stores source-of-truth data in PostgreSQL, and projects that data into graph and vector backends for read, search, and reconciliation workflows.

The service is organized around organization-scoped knowledge operations:

- identity is resolved from an API key
- each caller belongs to a tenant and app
- ontology, access, and query templates are configured per tenant/domain
- graph and vector stores are read replicas, not direct user write targets

## 2. Product Vision

The product vision is to provide one knowledge layer that can serve multiple teams and deployment targets without changing application code for each new domain.

The service should make it easy to:

- model tenant-specific domains safely
- share knowledge explicitly across tenants and apps
- swap graph/vector backends through runtime profile configuration
- validate the full stack repeatably in Compose, Kubernetes, and VM environments

## 3. Problem Statement

Teams need a service that can:

- model tenant-specific knowledge domains without changing application code
- write data once and read it back through graph and vector retrieval paths
- enforce tenant/app visibility and sharing rules consistently
- support multiple deployment targets with the same runtime contract

The repo currently serves as the bootstrap implementation and target architecture reference for that capability.

The main problem is not simple persistence. It is the combination of:

- tenant identity
- access control
- ontology-driven behavior
- projection consistency
- backend portability
- operational repeatability

## 4. Goals

- Provide authenticated CRUD and retrieval APIs for knowledge graph data.
- Support tenant and app onboarding with explicit access grants.
- Allow domain-specific ontology, templates, and lifecycle rules without hardcoding domain logic.
- Keep write, graph projection, vector projection, and reconciliation behavior observable.
- Support repeatable deployment and validation on Docker Compose, Kubernetes, and VM targets.
- Allow backend selection for graph and vector stores through runtime profile configuration.
- Make tenant/app onboarding and validation steps understandable to operators and integrators.

## 5. Non-Goals

- Building a consumer UI or admin portal in this repository.
- Hardcoding business logic for a specific customer domain.
- Replacing the PostgreSQL source of truth with graph or vector stores.
- Providing a fully managed cloud control plane.
- Supporting raw graph queries from clients as the default read model.

## 6. Primary Personas

- Platform operator: deploys and validates the service.
- Tenant admin: creates tenant/app records, sets ontology, and manages grants.
- Application integrator: writes data, runs templates, and consumes search/read APIs.
- Agent consumer: uses REST or MCP to retrieve context for downstream automation.
- SRE: diagnoses drift, backend failures, and deployment regressions.

## 7. Core User Journeys

### 7.1 Onboard A Tenant

1. Create a tenant record.
2. Create one or more apps under the tenant.
3. Store the returned API key safely.
4. Confirm the caller identity with `GET /v1/access/resolve`.
5. Add grants if the tenant must collaborate with another tenant or app.

### 7.2 Model A Domain

1. Create the domain.
2. Define node types and relationship types.
3. Add query templates.
4. Activate the templates.
5. Configure lifecycle/status behavior if the domain uses it.

### 7.3 Write And Read Knowledge

1. Submit a node, relationship, or ingest request.
2. Wait for projection completion.
3. Execute a read template or fetch a node directly.
4. Search semantically for the same knowledge.
5. Compare results to the source state if the output looks stale.

### 7.4 Deploy And Validate

1. Start the selected deployment environment.
2. Select the appropriate runtime profile.
3. Run the repeatable validation script.
4. Confirm health, access resolution, read/search, and reconciliation behavior.

## 8. Product Capabilities

- Tenant and app management
- Access grants and visibility resolution
- Ontology management per tenant/domain
- Knowledge write flows
- Knowledge read and search flows
- Integrity and reconciliation checks
- MCP transport for tool-oriented consumers
- Deployment profiles for Compose, Kubernetes, and VM

## 9. Success Criteria

### Product Outcomes

- A new tenant/app can be created and used without code changes.
- A domain can be modeled with node types, relationship types, and query templates.
- A written node is visible in the read/search path after projection completes.
- Access grants change what the caller can see.
- The same service can run on Compose, Kubernetes, and VM with documented environment variables.
- Integration validation can run repeatedly and produce deterministic results.

### Operational Outcomes

- Operators can identify the active runtime profile from environment variables.
- Backend connectivity issues are visible in startup or validation checks.
- Smoke tests can run without destabilizing the default test suite.

### User Outcomes

- Tenant admins can onboard without editing service code.
- App integrators can use named templates instead of raw graph queries.
- Agent consumers can use REST or MCP with the same identity model.

## 10. Assumptions

- PostgreSQL remains the source of truth for writes.
- Graph and vector stores are projections that can be rebuilt from source-of-truth state.
- Runtime profiles are the supported mechanism for backend selection.
- Local bootstrap keys exist for development and docs, not as a production provisioning model.

## 11. Risks

- Real graph/vector backend setup can drift across environments.
- Different backend vendors use different client semantics and startup ordering.
- Multi-tenant permission bugs can surface as empty reads rather than explicit failures.
- Reconciliation bugs may only appear after partial projection or stale versions.
- Documentation can become stale if scripts, compose files, and docs are not updated together.

## 12. Release Completion Criteria

This workstream is considered complete when:

- supported runtime profiles are documented and runnable
- tenant/app onboarding is documented
- deployment validation scripts are documented
- the service behavior is observable through health, integrity, and reconciliation checks
