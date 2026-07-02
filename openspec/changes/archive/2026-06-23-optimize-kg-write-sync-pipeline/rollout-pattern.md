# Rollout Pattern

## Standard Flow

1. Add-compatible schema or code path.
2. Backfill or reconcile data if needed.
3. Verify the new state with a read-only check.
4. Cut over producers/consumers to the new path.
5. Drop the old path only after metrics and verification stay green.

## Applied To This Change

- FK removal follows add-compatible projection cleanup first, then verify, then cut over to app-managed integrity, then drop old FK constraints.
- Bulk write indexes are safe to add before traffic changes.
- Projection maintenance changes should ship with verification and repair scripts before any drop of legacy behavior.

## Operational Guardrails

- Never fold verification or repair into the same migration file that changes hot-path constraints.
- Keep forward-only steps explicit when rollback would require a full data rebuild.
- Treat orphan counts and lag metrics as release gates before cutover.
