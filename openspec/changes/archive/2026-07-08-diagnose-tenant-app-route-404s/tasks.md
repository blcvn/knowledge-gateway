# Tasks

- [x] **R1** — Add route verification for tenant/app endpoints.
  Yêu cầu:
  - prove `GET /v1/tenants/{tenant_id}` and `/v1/tenants/{tenant_id}/apps` routes are mounted;
  - prove gorilla-mux route vars become `r.PathValue(...)` in downstream handlers;
  - prove unmatched tenant/app paths still return router-level `404`;
  - keep the coverage focused on dispatch, not business logic.

- [x] **R2** — Add service/handler coverage for tenant/app not-found behavior.
  Yêu cầu:
  - cover missing tenant lookups for tenant/app endpoints;
  - cover app lookup with wrong `tenant_id`;
  - cover the slug-vs-ID local repro path;
  - preserve the current auth behavior while distinguishing resource-level `404`.

- [x] **R3** — Add lightweight local diagnostics for tenant/app `404`s.
  Yêu cầu:
  - make it visible whether the request missed the router, lost path vars in the bridge, or failed
    later in handler/service;
  - keep diagnostics safe and narrowly scoped to local debugging needs;
  - avoid changing the broader API contract unless the signal is intentionally documented.

- [x] **R4** — Document the local repro and debug checklist.
  Yêu cầu:
  - list the seeded bearer tokens, tenant IDs, and relevant app IDs already available locally;
  - show a known-good `/v1/access/resolve` check before tenant/app calls;
  - call out that `/v1/tenants/{tenant_id}` expects the seeded tenant ID, not the slug.
