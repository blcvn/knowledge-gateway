# Tasks

- [x] **R1** — Align app provisioning with the durable write identity contract.
  Yêu cầu:
  - define and verify what makes an app `write-ready` for PostgreSQL-backed KG writes;
  - ensure app creation and any documented seeded write credentials satisfy UUID and FK-backed app
    identity requirements;
  - remove or reclassify any seeded/doc-only identities that can authenticate but cannot reach the
    durable write path safely.

- [x] **R2** — Add runtime coverage for write-path identity consistency.
  Yêu cầu:
  - cover the happy path where an authenticated app can open a sync session and persist graph
    identity metadata;
  - cover the mismatch class where auth and durable app provisioning diverge;
  - ensure the service no longer surfaces that mismatch as an opaque backend `500` during normal
    supported onboarding.

- [x] **R3** — Clarify and verify ontology write ownership semantics.
  Yêu cầu:
  - prove a tenant can read platform-visible baseline domains without automatically gaining write
    authority;
  - prove same-tenant owned domains are directly writable;
  - prove cross-tenant domain writes require an active `write` or `admin` grant for the relevant
    owner and scope.

- [x] **R4** — Update integration and troubleshooting guidance.
  Yêu cầu:
  - document the difference between `visible domain` and `writable domain`;
  - document the tenant-owned bootstrap path for integrations that plan to write under a tenant's
    own domain;
  - document the cross-tenant shared-domain path separately, including the need for explicit write
    grants when applicable;
  - keep seeded credential examples aligned with the actual runtime contract.
