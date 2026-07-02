# Tasks

- [ ] **R1** — Add a benchmark comparing the legacy single-item path and the batch projection path.
  Yêu cầu:
  - measure claim/coalesce/project/commit behavior for representative node and relationship batches;
  - keep the benchmark small and deterministic enough to run in CI or locally.

- [ ] **R2** — Add integration verification for mixed-success and stale-event cases.
  Yêu cầu:
  - cover `graph success + vector fail`;
  - cover `vector success + graph fail`;
  - cover `delete after upsert`;
  - cover `stale event`.

- [ ] **R3** — Document canary and rollback guidance for batch projection rollout.
  Yêu cầu:
  - define what to monitor during canary;
  - define when to fall back to the legacy single-item projection path;
  - keep the guidance aligned with the feature flag already present in runtime.
