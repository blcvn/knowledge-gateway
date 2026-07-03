# Tasks

- [x] **R1** — Audit compile-time fallout introduced by the `main` to `develop` rebase.
  Yêu cầu:
  - identify packages failing from stale helper signatures, removed functions, or partial merges;
  - fix the smallest set of mismatches needed to restore a clean build.

- [x] **R2** — Reconcile or remove redundant post-rebase config/runtime fragments.
  Yêu cầu:
  - keep the merged code aligned with current structs and validation rules;
  - avoid changing intended defaults or feature behavior.

- [x] **R3** — Re-run automated verification after cleanup.
  Yêu cầu:
  - run `go test ./...` from the repo root;
  - confirm the existing test suite remains green after the stabilization pass.
