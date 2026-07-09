# Tasks

- [x] **R1** — Define the canonical owner source-of-truth contract.
  Yêu cầu:
  - document which PostgreSQL tenant/app records are authoritative for supported write ownership;
  - state explicitly that Redis, memory, request context, and session variables are derived copies;
  - ensure all owner revalidation paths return to the same canonical registry instead of fragmented
    checks.

- [x] **R2** — Synchronize source of truth into cache and memory copies.
  Yêu cầu:
  - update supported tenant/app lifecycle flows so cache and other derived identity views are
    refreshed or invalidated from canonical PostgreSQL results;
  - prevent active responses from returning while derived identity state still reflects stale owner
    mapping;
  - make stale owner cache entries recoverable without manual repair.

- [x] **R3** — Add cache failback and session-copy revalidation.
  Yêu cầu:
  - keep cache-first auth as an optimization but require fallback to canonical lookup on cache miss,
    decode failure, stale status, or downstream owner mismatch;
  - ensure request/session identity copies are rebuilt from or rejected by canonical owner state
    before protected writes continue;
  - evict or refresh stale cache entries once drift is detected.

- [x] **R4** — Block graph-version sealing on canonical owner mismatch.
  Yêu cầu:
  - verify the tenant/app pair about to persist into graph identity metadata against canonical owner
    records before seal/write persistence;
  - surface controlled readiness/sync errors instead of raw FK-driven `500`;
  - add focused coverage for cache drift, fallback, revoke/mismatch, and first-write success.
