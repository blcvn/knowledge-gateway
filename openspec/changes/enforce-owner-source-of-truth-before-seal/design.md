# Design: Enforce Owner Source-of-Truth Sync Before Graph Version Seal

## Overview

The remaining failure is no longer about request payload ownership fields. The service already
sanitizes caller-supplied `tenant_id` and `app_id`, resolves identity from auth, and later checks
owner readiness. The unresolved gap is the lifecycle of that resolved identity across layers:

- PostgreSQL tenant/app tables are the durable owner registry;
- Redis may serve identity resolution first;
- request context and PostgreSQL session state carry copied tenant/app values through the request;
- write persistence eventually seals graph state using those copied owner IDs.

This change makes that pipeline explicit: source of truth flows outward to cache, memory, and
session copies; any doubt in those copies fails back to the same canonical owner registry; graph
version sealing cannot proceed until the owner pair is revalidated against that registry.

## Goals

- define one canonical owner registry for supported tenant/app writes
- define synchronization from canonical owner records into cache and request-local copies
- require cache-first auth to recover by re-reading canonical state when confidence is lost
- ensure session copies cannot drift into an independent authority
- stop graph version sealing before FK-backed owner mismatch reaches PostgreSQL

## Non-Goals

- replacing Redis with another cache technology
- introducing a second durable identity registry
- redesigning grant or ACL semantics

## Proposed Work

### 1. Declare one canonical owner registry

Supported runtimes must treat the PostgreSQL-backed tenant/app tables as the only authoritative
source of truth for owner identity.

That means:

- Redis identity entries are derived acceleration data;
- request context identity and PostgreSQL session variables are request-scoped copies;
- downstream checks must not treat cache contents or session copies as sufficient proof when owner
  durability is in question;
- all revalidation must return to the same canonical registry instead of mixing multiple partial
  checks across separate stores.

### 2. Define source-of-truth to cache and memory synchronization

Provisioning flows such as create-tenant, create-app, rotate-key, revoke-app, and equivalent owner
mutations must update canonical PostgreSQL rows first, then synchronize derived state.

The exact mechanism can stay implementation-sized, but the contract should require:

- cache warm or cache invalidation to happen from canonical tenant/app results;
- in-process identity views to be repopulated from canonical records rather than handwritten copies;
- stale cache entries that point to removed, moved, or revoked apps to be overwritten or evicted;
- no supported flow to return an app as active while its derived cache state still points at an old
  owner identity.

### 3. Strengthen cache-first auth with explicit failback

Cache-first auth is still acceptable as a performance optimization, but not as an independent
identity authority.

The resolver contract should require fallback to canonical tenant/app lookup when:

- the cache misses;
- the cache entry cannot be decoded or trusted;
- the cache says the app is active but downstream canonical checks detect owner mismatch;
- the cache entry points to an app that no longer exists, no longer belongs to the same tenant, or
  is no longer active;
- a supported write request reaches owner verification with evidence that the cached identity may be
  stale.

After fallback, the service should refresh or evict the cache so subsequent requests converge on
canonical state instead of repeating the drift.

### 4. Keep session copies derived and refreshable

Request context identity and PostgreSQL session variables are useful because they pin one identity
for the request, but they must remain copies of canonical state, not a separate source of truth.

This change should require:

- session identity to be created only from an auth result that is cache-verified or canonically
  reloaded;
- any canonical mismatch discovered before write persistence to fail or rebuild from canonical
  state before continuing;
- no later write stage to silently reinterpret session tenant/app IDs without returning to the same
  canonical owner registry.

### 5. Revalidate canonical owner identity before graph-version sealing

The write plane should not rely on `kg_graph_identifiers` foreign keys to become the first durable
owner check.

Before graph-version sealing or any first persistence that depends on owner foreign keys, the
service must verify:

- the tenant exists in the canonical owner registry;
- the app exists in the canonical owner registry;
- the app still belongs to that tenant in the canonical owner registry;
- the owner pair about to be persisted matches the request identity after any required fallback.

If this revalidation fails, the service should stop with a controlled readiness/sync error, record
the drift, and evict or refresh stale derived identity state as appropriate.

## Verification Plan

- integration coverage proves create-tenant -> create-app -> authenticate -> write uses the same
  canonical tenant/app pair through cache, request context, session scope, and write persistence
- coverage proves stale Redis identity can fall back to canonical PostgreSQL owner records and then
  self-heal cache state
- coverage proves revoked, cross-tenant, or missing-app cache entries do not survive into graph
  version sealing
- `POST /v1/kg/write/nodes` no longer reports this bug class first as a raw FK-driven `500`
