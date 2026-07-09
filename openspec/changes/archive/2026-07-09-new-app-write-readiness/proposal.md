# Proposal: Make Identity and Ontology Data Durable in Relationship DB

## Problem

`kg-service` currently splits core control-plane state across incompatible storage layers.

The most visible symptom is that tenant and app creation can succeed, and the returned app can
authenticate successfully, but the same app can still fail on `POST /v1/kg/write/nodes` with
`500 Internal Server Error`.

Current runtime evidence shows the write plane reaches PostgreSQL and fails on foreign-key
constraints in `kg_graph_identifiers`, especially:

- `kg_graph_identifiers_owner_tenant_id_fkey`
- `kg_graph_identifiers_owner_app_id_fkey`

This means the failure is not caused by BAS payload shape. It is a contract/runtime gap inside
`kg-service`: the access plane issues an identity that is accepted by authentication, but the
write plane does not recognize that identity as durably provisioned and write-ready.

Current repo review also shows the problem is broader than first-write onboarding:

- tenant/app identity state is managed in memory at runtime even though relationship DB already has
  `tenants`, `apps`, `access_grants`, and `access_audit_log` tables;
- ontology state is managed in memory at runtime even though relationship DB already has
  `domains`, `ontology_versions`, `node_type_schemas`, `rel_type_schemas`,
  `cross_domain_rel_rules`, `domain_query_templates`, `domain_status_field_configs`, and
  `query_strategies` tables;
- Redis and in-process memory are currently used beyond caching responsibilities, which allows
  durable state and runtime state to drift.

As a result, the service does not yet have a single durable source of truth for identity and
ontology data even though the write path and schema assume one exists.

## Proposed Solution

Strengthen this change so `relationshipdb`/PostgreSQL becomes the authoritative source of truth for
identity, access, and ontology control-plane data, with memory/Redis limited to cache roles:

1. require tenant/app provisioning to write durable rows into relationship DB tables that the write
   plane and FK constraints depend on;
2. require ontology domains, schemas, templates, strategies, and related metadata to be persisted
   in relationship DB as the authoritative store;
3. restrict in-memory stores and Redis to cache or test-double responsibilities, not primary
   runtime ownership of durable control-plane data;
4. ensure supported write requests from a newly created app do not fail with backend FK-driven
   `500` errors because auth/runtime state diverged from relationship DB;
5. add a repo-wide verification pass so all identity and ontology reads/writes resolve against the
   same durable contract.

## Scope

### In scope

- durable source-of-truth storage for tenant/app/access data in relationship DB
- durable source-of-truth storage for ontology metadata in relationship DB
- alignment between auth/access/ontology runtime behavior and relationship DB rows
- explicit handling of identity or ontology drift before it becomes opaque runtime failure
- coverage for onboarding, ontology bootstrap, and first-write flows against relationship DB

### Out of scope

- BAS-side payload or adapter changes
- broader tenancy or permission-model redesign beyond moving authoritative state into
  relationship DB
- unrelated graph/vector projection optimizations

## Success Criteria

- an app returned by `POST /v1/tenants/{tenant_id}/apps` can authenticate and complete a supported
  `POST /v1/kg/write/nodes` write without FK failures caused by missing tenant/app ownership rows
- tenant/app/access data resolve consistently from relationship DB across authentication,
  authorization, and PostgreSQL-backed writes
- ontology domains and schema metadata resolve consistently from relationship DB across ontology,
  write, read, search, integrity, and MCP flows
- in-memory stores and Redis are documented and implemented as caches or test doubles rather than
  runtime sources of truth
- tests and docs make the durable control-plane contract reproducible for integrations
