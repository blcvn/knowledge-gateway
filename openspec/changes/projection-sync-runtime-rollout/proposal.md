# Proposal: Projection Sync Runtime Rollout and Verification

## Problem

The batch-first projection runtime has been implemented, but the repo still needs a dedicated rollout
change to validate it under load and define a safe canary/rollback path.

Without a separate follow-up change, benchmark work and rollout guidance will be mixed into the core
runtime optimization change, which makes scope harder to review and merge.

## Proposed Solution

Create a follow-up change focused only on rollout readiness:

1. benchmark single-item projection versus batch projection;
2. verify mixed-success and stale-event paths with integration tests;
3. document canary and rollback guidance for batch projection;
4. keep the runtime code path unchanged unless verification fails.

## Scope

### In scope

- benchmark harness for projection runtime comparison
- integration verification for batch projection failure modes
- canary and rollback guidance for batch projection rollout

### Out of scope

- further runtime refactors
- adapter contract changes
- new projection batching semantics

## Success Criteria

- benchmark results show the batch path is measurably better than the legacy single-item path
- verification covers graph success/vector fail, vector success/graph fail, delete-after-upsert, and stale-event handling
- rollout guidance clearly states how to canary batch projection and fall back to the legacy path
