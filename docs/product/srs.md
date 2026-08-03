# Software Requirements Specification

## 1. Introduction

`kg-service` is a Go-based multi-tenant knowledge graph service. It accepts authenticated requests, stores source-of-truth data in PostgreSQL, and projects data into graph and vector backends for read/search workloads.

The SRS defines the system requirements that are already supported by the codebase or explicitly captured as deployment/runtime constraints.

## 2. System Scope

The system provides:

- tenant and app management
- access grant management
- ontology configuration
- knowledge write APIs
- knowledge read/search APIs
- integrity and reconciliation APIs
- MCP transport
- deployment and validation scripts for Compose, Kubernetes, and VM

The system does not provide:

- a web-based admin console
- a separate public data plane that bypasses PostgreSQL
- raw client-authored graph queries as the primary read interface

## 3. System Context

The service sits between:

- API clients and agents on the request side
- PostgreSQL as the source of truth
- graph backends as read replicas for traversal
- vector backends as read replicas for ANN and similarity search
- Redis for bootstrap and runtime support

## 4. Functional Requirements

### FR-1 Identity And Access

- The system shall resolve caller identity from the bearer token.
- The system shall support tenant and app creation.
- The system shall support key rotation and revocation.
- The system shall support explicit grants between tenants and apps.
- The system shall expose access-resolution and audit APIs.

### FR-2 Ontology Management

- The system shall allow per-tenant domain creation.
- The system shall allow node type and relationship type definitions.
- The system shall allow query template registration and activation.
- The system shall allow status/lifecycle configuration per domain.
- The system shall expose the effective ontology view for a tenant.

### FR-3 Knowledge Write

- The system shall accept node and relationship write requests.
- The system shall accept document ingest requests.
- The system shall reject malformed payloads and invalid ontology usage.
- The system shall preserve write auditability through the source-of-truth store.

### FR-4 Knowledge Read And Search

- The system shall list active query templates.
- The system shall execute named templates only.
- The system shall return visible nodes and relationships.
- The system shall support semantic search.
- The system shall support RAG-oriented retrieval on the same projection layer.

### FR-5 Projection And Reconciliation

- The system shall track projection sync versions.
- The system shall detect stale, missing, or mismatched projections.
- The system shall expose integrity reports.
- The system shall support graph and vector replication backends selected by runtime profile.

### FR-6 Deployment And Validation

- The system shall run on Docker Compose, Kubernetes, and VM deployment targets.
- The system shall support repeatable validation scripts for each deployment path.
- The system shall support profile-based backend selection.

### FR-7 MCP Surface

- The system shall expose an MCP connection endpoint on the protected HTTP surface.
- The system shall support tool discovery and invocation through MCP sessions.

## 5. Non-Functional Requirements

### NFR-1 Security

- No request body field shall override caller identity.
- Protected endpoints shall require bearer authentication.
- Access to data shall be filtered by visibility rules and grants.
- The system shall avoid silent privilege escalation when grants are absent.

### NFR-2 Reliability

- The source-of-truth write path shall be durable in PostgreSQL.
- Projection drift shall be observable and recoverable.
- Backend startup and health checks shall be explicit.
- Backend-specific failures shall surface as errors rather than implicit memory fallback in configured modes.

### NFR-3 Operability

- The system shall provide deployment and validation scripts.
- The system shall document environment variables and profile selection.
- The system shall expose health and integrity endpoints.
- The system shall support repeatable local and environment-backed smoke checks.

### NFR-4 Portability

- The runtime shall support multiple graph and vector backends.
- The same service binary shall run across supported deployment targets.
- Runtime profiles shall define backend compatibility in a documented way.

### NFR-5 Testability

- The codebase shall include conformance and integration tests.
- Backend-specific validation shall be reproducible in Compose or tag-gated tests.
- Default `go test ./...` shall remain green when backend-specific tests are added.

## 6. Data Requirements

### DR-1 Source Of Truth

- PostgreSQL shall remain the authoritative store for write operations and projection version tracking.

### DR-2 Projection Data

- Graph and vector stores shall represent projected views of the source data.
- Projection sync metadata shall be retained in graph/vector payloads.

### DR-3 Identity And ACL Data

- The service shall retain tenant/app identity in a normalized form.
- Access grants shall be queryable and auditable.

## 7. External Interface Requirements

- REST API on `KG_HTTP_HOST:KG_HTTP_PORT`
- MCP transport on the protected HTTP surface
- PostgreSQL for source-of-truth storage
- Redis for bootstrap support
- Graph backend and vector backend according to runtime profile

## 8. Deployment Requirements

- The service shall be deployable via Docker Compose, Kubernetes, and VM scripts.
- The deployment artifacts shall define the required environment variables.
- The deployment artifacts shall support runtime profile selection.
- The validation scripts shall map to the deployment path that started the service.

## 9. Constraints

- The repository currently treats PostgreSQL as source of truth.
- Graph and vector stores are projections and must remain consistent with the write model.
- Current deployment docs and scripts define the supported runtime profiles.
- The repository must continue to support both default test execution and backend-specific test execution.

## 10. Traceability Notes

- Product and user requirements live in the PRD and URD.
- API route details live in `docs/api`.
- Deployment and smoke commands live in `docs/deployment` and `scripts/`.
- Backend-specific runtime behavior is enforced by the code and the test suite.
