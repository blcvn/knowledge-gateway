# Tasks

- [x] **R1** — Define and verify the write-owner identity contract.
  Yêu cầu:
  - document where `owner_tenant_id` and `owner_app_id` come from for supported write requests;
  - ensure the authenticated identity used by write endpoints maps directly to the durable owner
    IDs expected by PostgreSQL;
  - identify any remaining transformation or divergence point between access resolution, session
    identity, and write persistence.

- [x] **R2** — Detect owner-ID readiness mismatches before FK failure.
  Yêu cầu:
  - add pre-persistence checks for durable tenant/app owner rows used by graph identity writes;
  - ensure the service stops with a controlled contract/readiness error when owner IDs are missing
    or misaligned;
  - remove this failure class from the generic backend `500` path for supported onboarding flows.

- [x] **R3** — Add diagnostics that make owner-ID drift directly observable.
  Yêu cầu:
  - log or surface the authenticated `tenant_id`/`app_id` and the owner IDs about to be persisted;
  - make it clear whether the mismatch is tenant missing, app missing, or app-to-tenant mismatch;
  - keep diagnostics safe for supported troubleshooting and integration use.

- [x] **R4** — Add focused coverage for create-app -> authenticate -> first-write owner alignment.
  Yêu cầu:
  - cover the happy path where a newly created app authenticates and completes `POST /v1/kg/write/nodes`;
  - cover the negative path where durable owner rows are absent or mismatched;
  - assert that this regression class surfaces as a controlled identity-readiness failure instead
    of a raw FK-driven `500`.
