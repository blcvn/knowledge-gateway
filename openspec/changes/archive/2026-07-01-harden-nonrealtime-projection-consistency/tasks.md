# Tasks

- [x] **R1** — Audit the projection-backed non-realtime read path and sync-status resolver.
  Yêu cầu:
  - review `internal/read/service.go` and `internal/read/handler.go`;
  - review `internal/workers/runtime.go`, `internal/write/service.go`, and graph-head repository
    paths;
  - confirm where per-entity version checks can diverge from whole-graph projection readiness.

- [x] **R2** — Harden graph sync semantics for projection-backed reads.
  Yêu cầu:
  - ensure sync status does not report `graph_lag_class="SYNCED"` unless the relevant graph version
    has advanced the graph backend head and the projected entity is actually readable through the
    graph path used by non-realtime reads;
  - introduce or reuse a distinct non-synced in-flight state for graph projection work instead of
    collapsing it into `SYNCED`;
  - add a distinct non-realtime projection inconsistency error path instead of returning a generic
    `404` when the source row still exists;
  - keep the true not-found contract unchanged for missing or invisible source entities.

- [x] **R3** — Extend repository-owned runtime validation for projection consistency.
  Yêu cầu:
  - verify relationshipdb write and projection sync timing against graph-head readiness and
    non-realtime readability;
  - fail when sync status claims `SYNCED` but the graph projection still cannot return the entity;
  - verify in-flight graph versions do not surface as synced before graph-head advancement;
  - log an explicit skip when CodeGraph create/update validation cannot run because the CodeGraph
    CLI is unavailable.

- [x] **R4** — Re-run available checks and record environment blockers.
  Yêu cầu:
  - run the repo-local tests that cover read-mode and sync-status behavior;
  - run script-level validation that is deterministic in the current environment;
  - record CodeGraph CLI unavailability or runtime-stack limitations if full validation cannot run.
