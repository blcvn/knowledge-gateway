# Proposal: Harden Non-Realtime Projection Consistency

## Problem

The current validation flow exposed a contract mismatch in the projection-backed read path:

1. `examples/codegraph/validate-codegraph-runtime.sh` successfully reached the relationshipdb write and
   projection timing step even when the local CodeGraph CLI was unavailable, but the timing check
   failed on a more serious condition:
   - `kg_entity_sync_status` reported `graph_lag_class="SYNCED"` for a freshly written node;
   - the same node still returned `404` on `mode=non-realtime`.
2. In the current runtime, graph sync status is still inferred too close to per-entity version
   equality, even though the repo already has graph-level versioning and backend projection heads.
   This allows a node-level version signal to look current before the full graph version is
   actually queryable through the non-realtime graph path.
3. Because non-realtime reads currently return a plain `404`, operators cannot distinguish:
   - a truly missing source entity;
   - a projection backend that claims the entity is synced but did not persist a readable node.

This leaves a dangerous false-positive state: sync metadata says the graph projection is current,
but the projection-backed read contract is still broken.

## Proposed Solution

Create a focused follow-up change that hardens projection-read consistency without changing the
separate relationshipdb-first write model:

1. audit the non-realtime read/query path, sync-status resolver, graph-version workflow, and
   runtime validation flow for graph-head-based consistency;
2. redefine graph sync status so `SYNCED` means the graph backend has applied the relevant logical
   graph version at the graph-head level, not merely that one entity reports a matching version;
3. require an explicit non-synced status for in-flight graph projection work so a version being
   written or projected cannot be reported as `SYNCED` prematurely;
4. return a distinct projection-consistency error for non-realtime reads when the source row exists
   but the graph projection is unreadable, instead of collapsing that case into a generic `404`;
5. extend repository-owned validation so it explicitly catches any state where sync status claims
   the graph is synced before the non-realtime graph read is actually queryable;
6. document and preserve the current environment behavior where CodeGraph create/update validation
   is skipped when the CodeGraph CLI is unavailable, while the projection timing checks still run.

## Scope

### In scope

- non-realtime node read and query error handling
- graph-head-based sync-status semantics for graph projection
- explicit in-flight graph sync state before graph head advancement
- repository-owned validation for relationshipdb write and projection timing
- explicit handling of CodeGraph CLI unavailability in validation output and docs

### Out of scope

- changing the relationshipdb-first write contract
- redesigning CodeGraph sync semantics
- adding new graph backends or changing deployment topology
- broad runtime reconciliation redesign beyond graph-head sync semantics

## Success Criteria

- `kg_entity_sync_status` no longer reports `graph_lag_class="SYNCED"` until the graph backend has
  advanced the logical graph head for the relevant version and the non-realtime graph path is
  queryable for the affected entity
- graph projection work that is still being applied is reported with a distinct non-synced state
  rather than being collapsed into `SYNCED`
- non-realtime reads distinguish projection inconsistency from a true not-found source entity
- runtime validation fails deterministically on false-`SYNCED` projection states
- validation logs clearly state when CodeGraph create/update validation is skipped because the
  CodeGraph CLI is unavailable
