# Design: Projection Sync Runtime Rollout and Verification

## Overview

This change is the operational follow-up to `optimize-projection-sync-runtime`.
It does not change the batching implementation itself. Instead, it validates the runtime and gives the
team a controlled rollout path.

## Goals

- compare the legacy single-item projection path with the batch path
- verify the runtime behaves correctly under partial backend failure
- document a canary plan and rollback trigger for production rollout

## Verification Plan

### Benchmark

Add a small benchmark that exercises:

- single-item event projection
- batch claim plus coalescing
- mixed node and relationship batches

The benchmark should focus on relative behavior, not absolute performance targets.

### Integration coverage

Add or extend tests for:

- graph success and vector failure
- vector success and graph failure
- delete after upsert
- stale event after newer projection version

## Rollout Plan

1. run the benchmark against the legacy path and the batch path;
2. canary batch projection on a small slice of workers or environments;
3. monitor backlog, queue age, stale skips, and partial failures;
4. roll back to the legacy single-item path if batch adapters regress correctness or stability.
