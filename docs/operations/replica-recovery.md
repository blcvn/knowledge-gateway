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

## Validation

- `GET /v1/kg/integrity/tenant/{tenant_id}` returns pass or acceptable drift.
- Search and read responses match the source records visible to the tenant.

