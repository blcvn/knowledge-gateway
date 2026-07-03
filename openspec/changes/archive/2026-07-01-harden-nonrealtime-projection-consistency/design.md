# Design: Harden Non-Realtime Projection Consistency

## Overview

This change is a narrow correctness pass over projection-backed reads and sync observability.
It does not alter the relationshipdb-first write path. Instead, it tightens the contract between:

- graph projection persistence;
- graph-version heads;
- non-realtime read behavior;
- per-entity sync status;
- repository-owned runtime validation.

## Design Decisions

### Graph sync status must follow graph heads, not per-entity version equality

Today the runtime can classify a node as graph-`SYNCED` from backend version metadata even when the
projection-backed read path still cannot fetch the node payload. That is too weak for operator use.

The repo already has graph-level versioning concepts such as:

- logical graph identifiers;
- graph versions and `GRAPH_VERSION_SEALED`;
- per-backend projection heads.

The hardened contract should make `graph_lag_class="SYNCED"` mean:

1. the relevant logical graph version has been fully applied to the graph backend;
2. the graph projection head for that backend has advanced to that graph version;
3. the affected entity is readable through a graph projection probe consistent with non-realtime
   reads.

This keeps `SYNCED` aligned with a whole-graph version handoff, not a single node's reported
`_kg_sync_version`.

### In-flight graph work needs an explicit non-synced state

A new source write can be durable in relationshipdb while its graph version is still:

- `PENDING_ENTITIES`;
- sealed but waiting in the outbox;
- being actively applied by the graph projection worker.

Those states should never surface as `SYNCED`. The contract should explicitly treat them as
non-synced graph states, using the existing lag taxonomy or a new graph-specific syncing state, so
operators can distinguish:

1. work that is still being projected;
2. work that is lagging or stuck;
3. work that is actually synced.

The important design constraint is semantic rather than naming: there must be a distinct state for
"still projecting this graph version" that is not `SYNCED`.

### Non-realtime read errors should surface projection inconsistency explicitly

`mode=non-realtime` is intentionally projection-only, so it should not fall back to relationshipdb.
However, a projection miss is not always equivalent to “resource does not exist”.

When the source entity still exists in relationshipdb and the service can determine that the graph
projection head should already have made the relevant graph version queryable, the read path should
return a dedicated projection error
instead of a generic `404`. This keeps the projection-only contract while making backend corruption,
partial persistence, or adapter drift visible to callers and validation tooling.

The `404` contract should remain unchanged only when the entity is truly missing or invisible from
the caller's allowed source scope.

### Validation should continue past CodeGraph CLI unavailability

The current environment may not have the `codegraph` CLI available. That should not block the
separate relationshipdb/projection timing checks, because the false-`SYNCED` failure can be exposed
without the CodeGraph update probe.

The validation flow should therefore keep two behaviors explicit:

1. log that CodeGraph create/update validation is skipped when the CLI is unavailable;
2. still execute the relationshipdb write and projection timing checks, and fail if sync status and
   non-realtime readability disagree.

### Verification plan should focus on graph-head readiness

The most valuable verification for this change is not broad end-to-end coverage; it is proving the
specific contract holds:

1. source write succeeds in relationshipdb;
2. graph sync status stays non-`SYNCED` while the relevant graph version is still in flight;
3. graph sync status becomes `SYNCED` only after graph-head advancement and a successful
   non-realtime graph read;
4. non-realtime read returns a projection-specific error rather than a false `404` for inconsistent
   states;
5. validation catches the mismatch deterministically.

## Spec Impact

This change updates three spec areas:

1. `read-templates` to distinguish projection inconsistency from true not-found behavior in
   non-realtime mode;
2. `sync-lag-guard` so graph sync status is derived from graph-head readiness rather than
   per-entity version equality;
3. `codegraph-runtime-validation` so repository-owned validation explicitly covers the
   relationshipdb/projection timing contract and the CodeGraph-CLI-unavailable skip path.
