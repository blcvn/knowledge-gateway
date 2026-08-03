# Tasks

- [x] **R1** — Move tenant/app/access runtime authority into relationship DB.
  Yêu cầu:
  - ensure `POST /v1/tenants` and `POST /v1/tenants/{tenant_id}/apps` write durable rows into
    relationship DB tables used by auth and FK-backed owner persistence;
  - ensure access grants, revocation, and audit writes are anchored to relationship DB;
  - ensure created identities returned to clients match the IDs used later by the write path;
  - avoid supported runtime states where auth succeeds only because memory state exists.

- [x] **R2** — Move ontology runtime authority into relationship DB.
  Yêu cầu:
  - ensure domain, schema, cross-domain rule, template, status config, search profile, and query
    strategy data are persisted and reloaded from relationship DB;
  - ensure ontology bootstrap for supported runtimes does not depend on process-local seeding as the
    only source of truth;
  - ensure ontology reads in write/read/search/integrity/MCP paths do not depend on memory-only
    state.

- [x] **R3** — Restrict memory and Redis to cache responsibilities.
  Yêu cầu:
  - audit all current `MemoryStore` and Redis usages across access, ontology, write, search,
    integrity, workers, and MCP code;
  - reclassify supported runtime usages as cache layers or replace them with relationship DB-backed
    implementations;
  - ensure cache invalidation/refill semantics are derived from durable writes.

- [x] **R4** — Add runtime protection against durable-state drift.
  Yêu cầu:
  - detect mismatches between authenticated identity or ontology reads and durable relationship DB
    state before surfacing raw FK or missing-schema failures;
  - ensure supported writes no longer return generic `500` just because durable control-plane rows
    are missing or misaligned;
  - keep the failure contract explicit enough that integrations can distinguish service readiness
    issues from payload errors.

- [x] **R5** — Add end-to-end coverage for durable control-plane correctness.
  Yêu cầu:
  - [x] cover create-tenant -> create-app -> authenticate -> `POST /v1/kg/write/nodes`;
  - [x] prove `kg_graph_identifiers.owner_tenant_id` and `owner_app_id` can be persisted for that
    new app;
  - [x] cover ontology bootstrap -> ontology-backed write/read flows against relationship DB at the
    service/integration boundary;
  - [x] cover regression classes where runtime state would previously succeed from memory but fail
    after restart or durable lookup.

- [x] **R6** — Update integration guidance and troubleshooting.
  Yêu cầu:
  - document that BAS payload is not expected to carry service-authoritative tenant/app or ontology
    ownership data;
  - document that tenant/app and ontology APIs are expected to durably provision relationship DB
    state without hidden follow-up registration;
  - document how to diagnose source-of-truth mismatches using auth resolution, ontology lookups, and
    write-path error signals.
