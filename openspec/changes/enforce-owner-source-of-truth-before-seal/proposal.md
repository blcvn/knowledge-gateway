# Proposal: Enforce Owner Source-of-Truth Sync Before Graph Version Seal

## Problem

`kg-service` still returns `500 Internal Server Error` on the supported first-write flow because
`kg_graph_identifiers` insert can still hit owner foreign-key rejection.

The latest runtime conclusion narrows the bug further:

- BAS payload shape is not the blocker;
- authentication can succeed and produce a request identity;
- write persistence still fails because the tenant/app identity carried into the write plane is not
  guaranteed to stay synchronized with the durable owner records that PostgreSQL enforces.

The repo already moved tenant/app provisioning toward PostgreSQL-backed durability and added
write-readiness checks, but one contract is still incomplete: how tenant/app identity propagates
from source of truth into Redis cache, in-memory/session copies, and finally the write path.

Today that path is still fragmented:

- provisioning writes durable tenant/app rows;
- auth resolution may prefer cached identity data;
- request context and PostgreSQL session variables carry a copy of that resolved identity;
- write readiness later checks owner rows again, but only after the request has already committed
  to one copied identity.

Without one explicit sync and fallback contract, stale cache or stale session copies can remain
internally consistent with themselves while diverging from the durable owner registry.

## Proposed Solution

Create a focused follow-up change that defines the full owner-identity synchronization contract:

1. declare one authoritative source of truth for tenant/app owner identity in supported runtimes;
2. define how cache and memory copies are derived from that source of truth and how they are
   refreshed or invalidated after provisioning changes;
3. require cache-first identity resolution to fail back to source of truth when cache entries are
   missing, stale, unverifiable, or contradicted by downstream owner checks;
4. require session/context identity copies to remain derived data only, never an independent truth;
5. block graph-version sealing until the exact tenant/app pair to be persisted has been verified
   against the canonical owner registry.

## Scope

### In scope

- canonical source-of-truth definition for owner tenant/app identity
- source-of-truth to cache/memory synchronization for supported tenant/app lifecycle events
- auth-resolution fallback from cache to canonical store
- session-copy consistency rules for protected write requests
- pre-seal owner verification before `kg_graph_identifiers` / graph-version persistence
- tests for cache drift, fallback, and first-write owner alignment

### Out of scope

- redesigning tenant/app/grant semantics
- changing BAS request payloads
- unrelated ontology, search, or projection optimizations

## Success Criteria

- supported runtimes treat one canonical tenant/app registry as the only source of truth for owner
  identity
- Redis or in-memory identity copies are refreshed or invalidated from that source instead of
  becoming independent owner registries
- cache-first auth can recover safely by reloading canonical tenant/app state before a stale cache
  entry causes a write-plane mismatch
- session-bound write identity is provably derived from canonical owner state and does not bypass
  revalidation
- `POST /v1/kg/write/nodes` no longer reaches graph-version sealing with a tenant/app pair that the
  canonical owner registry would reject
