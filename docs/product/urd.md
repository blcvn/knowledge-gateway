# User Requirements Document

## 1. Purpose

This document describes what users need from `kg-service` in practical terms. It focuses on workflows, onboarding, and the operational expectations people have when they use the service directly or through automation.

## 2. User Outcomes

Users of `kg-service` want to:

- authenticate with a bearer token
- create or select a tenant and app
- configure ontology for a domain
- write and retrieve graph-backed knowledge
- search semantically against vector-backed projections
- share data safely across tenant/app boundaries
- deploy the service in the environment they already use

They also want the service to behave predictably when a backend is changed, when a grant is added or removed, and when a projection is stale.

## 3. User Types

- Platform admin
- Tenant admin
- App integrator
- Agent runtime client
- Operator / SRE

## 4. User Needs

### 4.1 Platform Admin

Platform admins need to:

- create tenants and apps
- rotate and revoke keys
- define access grants
- validate onboarding with an access-resolution call
- confirm the service is reachable and healthy after deployment

### 4.2 Tenant Admin

Tenant admins need to:

- create domains and ontology definitions
- publish query templates
- configure lifecycle/status fields
- confirm effective visibility for the tenant's apps
- understand which apps can see which domains

### 4.3 App Integrator

App integrators need to:

- use the current API key to write nodes and relationships
- read and search by domain
- execute named templates rather than raw graph queries
- confirm that caller identity is derived from the key, not request payloads
- understand which deployment profile they are targeting

### 4.4 Operator

Operators need to:

- deploy the service on Docker Compose, Kubernetes, or VM
- run repeatable integration validation
- verify backend connectivity for PostgreSQL, Redis, graph, and vector stores
- diagnose integrity or reconciliation drift
- know which compose file or script corresponds to which validation path

## 5. Task Flows

### 5.1 Onboard A New Tenant And App

1. Create a tenant.
2. Create an app under that tenant.
3. Store the generated API key.
4. Call `GET /v1/access/resolve`.
5. Confirm the app can see the intended domains after grants are added.

Expected result:

- the service accepts the key
- the identity is resolved correctly
- the app has the expected visibility

### 5.2 Prepare A New Domain

1. Create the domain.
2. Define node and relationship types.
3. Add query templates.
4. Activate templates.
5. Configure lifecycle/status behavior if the domain uses it.

Expected result:

- the domain is visible in the effective ontology
- write validation can use the ontology
- read/search can execute named templates

### 5.3 Write And Read Data

1. Submit a write request.
2. Wait for projection to complete.
3. Read back a node or execute a template.
4. Search by semantic query.
5. Check integrity if results look stale or incomplete.

Expected result:

- the same data is visible across write, graph, and vector paths after projection
- ACL rules continue to filter results

### 5.4 Deploy And Validate

1. Start the target deployment environment.
2. Run the appropriate validation script.
3. Check health, access resolution, read/search, and reconciliation.

Expected result:

- the deployment path starts the correct dependencies
- the service becomes reachable
- the validation script can prove the environment is usable

## 6. User Expectations

- The service should reject missing or invalid keys.
- The service should be predictable across environments.
- The service should explain validation failures clearly.
- The service should not require raw graph query authoring from the user.
- The service should preserve tenant isolation unless an explicit grant exists.
- The service should provide a clear checklist for onboarding a new tenant/app.
- The service should make environment-specific deployment instructions easy to find.

## 7. Acceptance Expectations

Users consider the product usable when:

- a tenant admin can onboard a new app and obtain a working key
- an app integrator can write and retrieve knowledge without asking for code changes
- an operator can choose the correct deployment and validation script
- a consumer can understand which API surface to use for REST or MCP

## 8. User Friction To Avoid

- Hidden environment variables that are not documented
- Unclear mapping between profile names and backend services
- Silent fallback from a configured backend to memory-backed behavior
- Need to author raw graph queries for ordinary read use cases
- Tenant/app identity leaked from request payloads instead of the API key
