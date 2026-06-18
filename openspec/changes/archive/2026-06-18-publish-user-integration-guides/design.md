# Design

## Current Behavior

The repository has several useful documentation assets, but none of them serves as a complete usage guide for consumers:

- `README.md` lists implemented endpoints and local bootstrap credentials
- `docs/KG_Service_TDD_v1.md` captures the target architecture and many illustrative API examples
- `docs/operations/` contains internal runbooks for incidents and recovery
- the new API spec change defines the contract surface

This is strong raw material, but it is not organized around the questions an integrating user typically asks:

- How do I authenticate and what credentials do I use locally?
- What is the order of steps to onboard a tenant or app?
- How do I define ontology before writing data?
- How do I test read and search successfully?
- When should I use REST versus MCP?
- Which docs are for integrators versus operators?

## Problem Statement

The repository lacks a user-oriented documentation layer that translates the existing API surface into an actionable integration journey. As a result, onboarding depends on implementation knowledge and informal guidance rather than a maintained path in the repo.

## Goals

- Publish user-facing guides that help a consumer get from zero to a working integration.
- Organize guidance by real workflows rather than by internal package structure.
- Reuse the API spec as the normative contract while keeping the integration guides task-oriented.
- Clarify the difference between bootstrap-only details and long-term production expectations.

## Non-Goals

- Redesigning endpoint behavior or changing auth semantics.
- Replacing the machine-readable API spec with prose-only documentation.
- Writing operator runbooks or production deployment manuals in this change.

## Key Decisions

### 1. Split consumer guidance from reference and operations material

The documentation set should have three distinct roles:

- API spec documents the contract precisely
- user integration guides explain how to use that contract in realistic workflows
- operations runbooks explain how maintainers recover from failures

This keeps onboarding content approachable without weakening the formal API contract or polluting user docs with operator concerns.

### 2. Organize guides by integration journey

The user-facing documentation should follow the order an integrator actually experiences:

- local bootstrap and authentication
- tenant and app management
- ontology setup
- data ingestion and graph mutations
- read and semantic search
- MCP usage for agent-style consumers
- troubleshooting and common integration mistakes

This is easier to follow than grouping docs only by package or route family.

### 3. Make bootstrap-specific assumptions explicit

The current runtime seeds sample credentials and a sample ontology for local development. Those conveniences are valuable for onboarding, but the guide must label them clearly as bootstrap behavior so users do not confuse them with production provisioning flows.

### 4. Use task-oriented examples instead of route inventories alone

The guide should not duplicate the API reference route-by-route. Instead, it should show call sequences and decision points, for example:

- create tenant, then create app, then use the returned API key
- create a domain, then define node types and relationship types, then activate query templates
- write nodes, then query them through template execution or semantic search
- use integrity endpoints to verify projection health during evaluation

This gives users enough context to succeed while deferring exact schema details to the API spec.

### 5. Link REST and MCP guidance intentionally

The service exposes both REST and MCP surfaces. The integration docs should explain when each surface is appropriate and how they share the same identity, ACL, and validation model, so users do not treat them as unrelated products.

## Risks And Mitigations

- User guides can drift from the API contract.
  - Mitigation: make the API spec the normative contract and have guides reference it for exact payload schemas.
- Bootstrap examples can accidentally be mistaken for production commitments.
  - Mitigation: label seeded credentials, sample ontology, and in-memory behavior explicitly as local-development bootstrap context.
- Documentation can become repetitive across README, guides, and API spec.
  - Mitigation: keep README concise, place workflows in dedicated guides, and keep exact schemas in the API spec.

## Validation Strategy

- Verify the user guides cover the end-to-end onboarding path for local bootstrap usage.
- Verify every workflow in the guides maps to currently implemented endpoints.
- Verify the guides link to the API spec for precise request/response details rather than re-defining those schemas inconsistently.
- Verify bootstrap-only assumptions are labeled clearly so readers can distinguish demo behavior from production architecture direction.
