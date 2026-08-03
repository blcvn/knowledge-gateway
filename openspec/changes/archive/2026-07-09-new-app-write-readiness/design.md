# Design: Make Identity and Ontology Data Durable in Relationship DB

## Overview

This change closes a broader runtime gap in `kg-service` data architecture:

- tenant/app creation succeeds in an in-memory access plane;
- ontology configuration is served from an in-memory ontology plane;
- the returned app API key authenticates successfully;
- the first KG write still fails because the PostgreSQL write plane requires durable owner rows
  that were not created or not aligned with the authenticated identity;
- multiple runtime paths can observe different truths depending on whether they read from memory,
  Redis-backed cached state, or PostgreSQL tables.

The target is not a general tenancy redesign. The target is to make relationship DB the single
authoritative store for durable control-plane state so onboarding, ontology bootstrap, and first
write all behave as one contract.

## Goals

- make tenant/app/access state durable in relationship DB
- make ontology metadata durable in relationship DB
- ensure auth-resolved `tenant_id` and `app_id` match what write persistence expects
- prevent control-plane drift bugs from surfacing as opaque runtime `500`s
- document and verify end-to-end onboarding, ontology bootstrap, and first-write flows

## Non-Goals

- changing BAS request payload semantics
- redesigning cross-tenant grants or ontology ownership rules
- changing unrelated projection or sync semantics

## Proposed Work

### 1. Define relationship DB as the durable source of truth

The repo already defines durable PostgreSQL tables for identity/access and ontology state, but the
runtime currently relies on `MemoryStore` implementations for those areas. This change should make
the relationship DB contract explicit:

- `tenants`, `apps`, `access_grants`, and `access_audit_log` are authoritative for access state;
- `domains`, `ontology_versions`, `node_type_schemas`, `rel_type_schemas`,
  `cross_domain_rel_rules`, `domain_query_templates`, `domain_status_field_configs`, and
  `query_strategies` are authoritative for ontology state;
- Redis and in-process memory may cache derived or recently read values, but they must not be the
  only durable home of those entities in supported runtime flows.

### 2. Define write-ready onboarding as one service contract

`POST /v1/tenants/{tenant_id}/apps` currently produces an identity that can pass authentication,
but that alone is not enough for the durable write path.

This change should define a stricter contract:

- if the service returns an active app for a supported onboarding flow, that app is already
  write-ready for PostgreSQL-backed owner persistence;
- the service must not require an undocumented second registration or repair step before the first
  supported write;
- the tenant/app identifiers used by auth, ACL resolution, and write metadata must be the same
  identifiers recognized by the authoritative database rows behind FK constraints.

### 3. Materialize durable tenant/app ownership records during provisioning

The write plane persists `owner_tenant_id` and `owner_app_id` into tables that reference durable
tenant/app state. The create-tenant and create-app flow must therefore provision or synchronize the
same durable records before the service advertises the resulting identity as active.

Implementation details can stay small, but the contract should require:

- a created tenant is durably present anywhere `owner_tenant_id` FK checks depend on;
- a created app is durably present anywhere `owner_app_id` FK checks depend on;
- the IDs returned to clients are the same IDs the write plane will persist;
- local or mixed-runtime stores do not drift between authenticated identity and durable ownership.

### 4. Materialize ontology state durably before runtime serves it

Ontology is currently a control plane for domain visibility, schema validation, search tuning,
query templates, and integrity checks. It therefore cannot remain memory-only in supported
environments if the service expects consistent behavior across requests, restarts, and workers.

This change should require:

- ontology creation and update APIs persist durable rows in relationship DB;
- runtime reads for domain lookup, schema validation, query templates, status config, search
  profiles, and query strategies resolve from relationship DB or caches derived from it;
- memory-only ontology state is limited to cache/test usage and is rebuildable from durable data;
- bootstrap flows that seed ontology for local or example runtimes write those same records into
  relationship DB instead of relying on process-local seeding alone.

### 5. Fail safely if write readiness cannot be established

The supported path should succeed end-to-end, but the service still needs a defined behavior when
provisioning is incomplete or a runtime drift is detected.

This change should require:

- the service detects unsupported identity mismatch before surfacing a raw FK failure to callers;
- supported onboarding flows no longer end in a backend `500` caused only by missing durable owner
  rows;
- any remaining mismatch path is surfaced as an intentional contract/runtime failure that points to
  service readiness, not caller payload shape.

The exact status code can remain implementation-sized as long as it is explicit, documented, and
not a generic internal-error leak.

### 6. Audit repo-wide reads and caches for source-of-truth correctness

Because access and ontology state currently appear in memory, Redis, service handlers, search,
write, integrity, workers, and MCP code paths, this change needs a repo-wide correctness pass.

That pass should verify:

- runtime reads do not silently prefer stale in-memory state over durable relationship DB rows;
- Redis keys such as identity or ACL entries are treated as cache artifacts with invalidation and
  refill behavior anchored to relationship DB writes;
- test helpers and local bootstrap paths are clearly separated from supported runtime storage
  architecture;
- any remaining in-memory store is either a cache facade over durable data or a test-only double.

### 7. Verify the create -> authenticate -> write sequence

Coverage should prove the sequence that currently fails in integration:

1. create tenant;
2. create app under that tenant;
3. authenticate with the returned API key;
4. call `/v1/kg/write/nodes`;
5. persist owner identity metadata without tenant/app FK violations.

The repo should also cover:

- ontology bootstrap followed by domain/schema-backed writes using relationship DB as the source of
  truth;
- negative cases where access or ontology state diverges from durable rows, so future regressions
  are caught before they become integration-only `500`s or restart-sensitive behavior.

## Verification Plan

- end-to-end or integration-style tests cover create-tenant/create-app followed by authenticated
  node write on the PostgreSQL-backed path
- coverage proves owner-tenant and owner-app FK dependencies are satisfied for newly created apps
- coverage proves ontology APIs and ontology-backed runtime reads resolve against relationship DB
- coverage proves identity or ontology drift is detected as a controlled contract failure rather
  than an opaque internal error
- troubleshooting guidance states that durable relationship DB state, not process-local memory, is
  the supported source of truth for identity and ontology
