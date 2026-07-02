# Tasks

- [x] **R1** — Audit the integration and CodeGraph validation entrypoints.
  Yêu cầu:
  - review `tests/integration/*`;
  - review `scripts/integration-test.sh`;
  - review CodeGraph-focused validation scripts, bridge state handling, and Make targets.

- [x] **R2** — Fix script behavior mismatches in the CodeGraph validation flow.
  Yêu cầu:
  - avoid unbound-variable aborts for missing `EMBEDDING_PROVIDER`;
  - make `--skip-verify` match the documented behavior.

- [x] **R3** — Extend CodeGraph validation coverage for create and update flows.
  Yêu cầu:
  - validate an initial sync that creates a CodeGraph probe entity;
  - validate a subsequent sync that updates that same logical entity;
  - assert the update path changes the relevant graph version signal instead of only returning 200s.

- [x] **R4** — Re-run the available validation checks and capture any environment blockers.
  Yêu cầu:
  - run `bash -n` for touched scripts;
  - run `go test ./tests/integration/...`;
  - run `go test ./codegraph-sync/...`;
  - note Docker/runtime limitations if full Compose validation cannot run locally.
