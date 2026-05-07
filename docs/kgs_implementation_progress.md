# KGS Platform Implementation Progress

This document tracks the detailed implementation progress of the Knowledge Graph Service (KGS) Platform, completely conforming to the **Golang + Go-Kratos + Neo4j + Postgres** architecture pattern.

> **Status Reference**
> 🔴 NOT STARTED | 🟡 IN PROGRESS | 🟢 COMPLETED | 🚫 BLOCKED

## 0. Project Setup & Core Infrastructure (� COMPLETED)

- [x] **0.1. Initialize Kratos Project:**
  - Create the base layout using `kratos new kgs-platform`.
  - Set up `cmd/`, `internal/` (biz, data, server, service), and `api/` directories.
- [x] **0.2. Dependency Definitions:**
  - `go.mod` configuration (Kratos, GORM, Neo4j Driver, Redis, OPA Client).
  - [x] Setup Wire dependency injection in `cmd/kgs-platform/wire.go`.
- [x] **0.3. Configuration Management:**
  - Define protobuf for `configs/config.yaml` containing settings for Postgres, Neo4j, Redis, and OPA.
- [x] **0.4. Database Drivers Integration:**
  - **Postgres:** Integrate GORM in `internal/data/data.go`.
  - **Neo4j:** Integrate Neo4j Go Driver.
  - **Redis:** Integrate `go-redis/v9`.

---

## 1. Sprint 1 — App Registry & Gateway (🟡 IN PROGRESS)

### Task 1.1 — PostgreSQL Schema: Registry
- [x] Create GORM Models (`App`, `ApiKey`, `Quota`, `AuditLog`) in `internal/biz/registry.go`.
- [x] Implement database auto-migration script in Kratos startup or separate CLI command.

### Task 1.2 — App Registry API (gRPC + HTTP)
- [x] **Protobuf Definition:** Define `api/registry/v1/registry.proto`.
- [x] **Service Layer:** Implement `internal/service/registry.go` (CreateApp, ListApps, GetApp, IssueApiKey, RevokeApiKey).
- [x] **Biz Logic:** Implement `internal/biz/registry.go` containing core business rules (e.g., generating API key hash).
- [x] **Data Repository:** Implement `internal/data/registry.go` using GORM.

### Task 1.3 — Gateway Auth Middleware (Kratos Middleware)
- [x] Implement HTTP/gRPC middleware `internal/server/middleware/auth.go`.
- [x] Extract API key from headers, validate against Redis cache / Postgres.
- [x] Inject `AppContext` (AppID, Scopes) into `context.Context`.

### Task 1.4 — Rate Limiter
- [x] Implement Redis-based sliding window rate limiter middleware in `internal/server/middleware/ratelimit.go`.

### Task 1.5 — Namespace Reservation in Neo4j
- [x] Implement Neo4j query inside `internal/data/registry.go` to reserve/release `__KGS_Namespace` upon App creation/deletion.

---

## 2. Sprint 2 — Ontology Service (� IN PROGRESS)

### Task 2.1 — PostgreSQL Schema: Ontology
- [ ] Create GORM Models (`EntityType`, `RelationType`) with JSONB fields in `internal/biz/ontology.go`.

### Task 2.2 — Ontology CRUD API
- [x] **Protobuf Definition:** Define `api/ontology/v1/ontology.proto`.
- [x] **Service Layer:** Implement `internal/service/ontology.go`.
- [x] **Biz & Data:** Implement creation logic, ensuring backward compatibility on updates (additive only).

### Task 2.3 — JSON Schema Validator
- [x] Implement `internal/biz/validator.go` using a Golang JSON Schema validator (e.g., `github.com/xeipuuv/gojsonschema`).
- [x] Implement robust error formatting (matching `KGSError` hierarchy).

### Task 2.4 — Ontology Cache
- [x] Implement `internal/data/ontology_cache.go` using Redis with 5-minute TTL.

### Task 2.5 — Neo4j Constraint Auto-Sync
- [x] Implement a worker in `internal/biz/ontology_sync.go` that runs on Kratos application startup to sync constraints to Neo4j.

---

## 3. Sprint 3 — Graph API (🔴 NOT STARTED)

### Task 3.1 — Query Planner
- [x] Implement `internal/biz/query_planner.go`.
- [x] Develop Golang string builders to dynamically construct safe, namespaced Cypher queries (Node CRUD, Subgraph, Impact, Coverage).

### Task 3.2 — Node CRUD
- [x] **Protobuf Definition:** Define `api/graph/v1/graph.proto`.
- [ ] **Validation Pipeline:** Chain Auth Middleware $\rightarrow$ JSON Schema Validator $\rightarrow$ OPA Policy Check $\rightarrow$ Quota Check $\rightarrow$ Query Planner.
- [x] **Data Layer:** Implement Neo4j execution logic in `internal/data/graph_node.go`.
- [x] **Events:** Publish `node.created` to Redis Stream securely.

### Task 3.3 — Edge CRUD
- [x] Implement `CreateEdge`, `DeleteEdge` inside the Graph Service.
- [ ] Validate relation whitelist before creating edges.

### Task 3.4 — Context & Query API
- [x] Implement `/context`, `/impact`, `/coverage`, `/path` endpoints.
- [x] Implement hardcoded guardrails (depth limitations, node limits) in `internal/biz/graph_guardrails.go`.
- [x] Implement safe whitelist query evaluation (prevent JSON $\rightarrow$ Raw Cypher injection).

### Task 3.5 — Response View Resolver
- [x] Implement `internal/biz/view_resolver.go` to shape JSON responses based on App-defined views stored in Postgres.

---

## 4. Sprint 4 — Rule Engine & Access Control (🔴 NOT STARTED)

### Task 4.1 — PostgreSQL Schema: Rules & Policies
- [ ] Create GORM models (`Rule`, `RuleExecution`, `Policy`).

### Task 4.2 — Rule CRUD API
- [x] **Protobuf Definition:** Define `api/rules/v1/rules.proto`.
- [x] Implement Service, Biz, and Data layers for Rules.

### Task 4.3 — Rule Runner: Scheduled
- [x] Implement distributed cron jobs matching Kratos design (e.g., leveraging `github.com/go-co-op/gocron` + Redis lock).

### Task 4.4 — Rule Runner: ON_WRITE (Event-Driven)
- [x] Implement Redis Stream consumer pattern in `cmd/kgs-worker/` or run asynchronously within the main Kratos app.

### Task 4.5 — OPA Integration: Policy Decision Point
- [x] Implement `internal/biz/opa_client.go`.
- [x] Send HTTP requests to local OPA sidecar (`http://localhost:8181/v1/data/kgs/allow`).
- [x] Write `kgs.rego` policies.

### Task 4.6 — Policy CRUD API
- [x] **Protobuf Definition:** Define `api/accesscontrol/v1/policy.proto`.
- [x] Implement CRUD operations.
- [x] Implement background worker to sync SQL policies $\rightarrow$ OPA data bundle every 30s.

---

## Verification Plan

Because this is an implementation checklist document based exclusively on a Go-Kratos paradigm:
1. **Directory Structure Verification:** Ensure `kratos new` can be replicated, checking for standard Kratos structure (`api/`, `internal/biz`, `internal/data`, `internal/service`).
2. **Schema & Model Validation:** Verify that GORM models map cleanly to the documented PostgreSQL tables.
3. **Protobuf Compilation:** Successfully compile protobuf definitions to Go and Swagger JSON using `make api`.
4. **Middleware Tests:** Unit-test the auth middleware passing mock contexts.
