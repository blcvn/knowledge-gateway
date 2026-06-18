# Reconciliation Incident Handling

## When To Use

Use this runbook when reconciliation reports drift that is not self-healing after worker replay.

## Symptoms

- Reconciliation shows persistent `graph_mismatch`, `vector_mismatch`, or orphan issues.
- Affected records are stable across repeated reconciliation runs.
- Operators need to decide whether to repair replica state or investigate source writes.

## Triage

- Confirm whether the source record changed recently.
- Compare source row, graph projection, and vector payload for the same node or relationship ID.
- Distinguish projection drift from intentional deletions or lifecycle transitions.
- If only graph state is stale, follow the graph-only repair path in [Replica Recovery Runbook](./replica-recovery.md).
- If only vector state is stale, follow the vector-only repair path in [Replica Recovery Runbook](./replica-recovery.md).
- If versions differ across PostgreSQL, graph, and vector stores, follow the mixed-version repair path in [Replica Recovery Runbook](./replica-recovery.md).

## Recovery Steps

1. Identify the smallest affected scope: a node, relationship, domain, or tenant.
2. Reproject the affected source rows into graph and vector stores.
3. Replay any missing outbox events for the same scope.
4. Re-run reconciliation and verify counts return to zero or an expected baseline.

## Escalation

- Escalate if source-of-truth rows and worker replay both look correct but drift persists.
- Escalate if the same issue reappears after two repair attempts.
