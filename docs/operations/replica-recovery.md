# Replica Recovery Runbook

## When To Use

Use this runbook when graph or vector projections fall behind, are partially empty, or need to be rebuilt after an outage.

## Symptoms

- Integrity endpoints report non-zero drift counts.
- Worker reconciliation reports missing projections or orphaned replicas.
- Search or read routes return stale or missing projected data while PostgreSQL source data is correct.

## Immediate Checks

- Confirm PostgreSQL writes are healthy and outbox rows continue to accumulate.
- Confirm worker polling is running and the worker runtime is not stuck in retry or dead-letter states.
- Compare `internal/workers` reconciliation output against source `kg_nodes` and `kg_relationships`.

## Recovery Steps

1. Stop any stale projection workers that are writing corrupt or partial replica state.
2. Restart worker polling so pending outbox events are replayed.
3. Re-run reconciliation after the queue drains.
4. If graph/vector state still diverges, rebuild the projection from source-of-truth rows and replay outbox events in order.

## Drift-Specific Repair Paths

### Graph-only drift

- Use this when reconciliation reports `graph_mismatch`, `orphan_graph_node`, `orphan_graph_relationship`, or `stale_projection_version` only for graph records.
- Verify the source row in PostgreSQL, then re-run the worker projection for the affected node or relationship IDs.
- If a graph backend was manually edited, remove the affected node or relationship from the replica first and then replay the source outbox event so the `_kg_sync_version` value is rewritten from source truth.
- Re-run reconciliation and confirm graph drift returns to zero while vector drift remains unchanged.

### Vector-only drift

- Use this when reconciliation reports `vector_mismatch`, `orphan_vector_doc`, or stale vector sync versions without graph drift.
- Confirm the embedding payload is still valid for the current source record and the tenant ACL set.
- Delete the stale vector document for the affected node ID, then replay the latest outbox event to reinsert the document with the authoritative `_kg_sync_version`.
- Re-run reconciliation and confirm vector drift returns to zero while graph drift remains unchanged.

### Mixed-version drift

- Use this when graph and vector replicas are both present but one or more entities have different `_kg_sync_version` values across stores.
- Compare the source version in PostgreSQL with the projection ledger in `kg_projection_versions`.
- If the ledger is stale, replay the outbox events in order until the source version, graph version, and vector version match.
- If the replica payload looks correct but the version is behind, force a reproject by deleting the affected replica record and replaying the latest source event.
- Re-run reconciliation and confirm the three-store version alignment is monotonic and identical for the repaired entities.

## Validation

- `GET /v1/kg/integrity/tenant/{tenant_id}` returns pass or acceptable drift.
- Search and read responses match the source records visible to the tenant.
