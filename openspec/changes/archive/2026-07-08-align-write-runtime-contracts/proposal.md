# Proposal: Align Write Runtime Identity and Domain Ownership Contracts

## Problem

End-to-end integrations can currently authenticate successfully and still fail on the KG write path
for reasons that are not part of the documented client contract.

Current investigation shows two separate contract gaps:

- the access/auth layer accepts seeded and newly created apps from an in-memory store, while the
  graph-version write path persists `owner_app_id` through PostgreSQL tables that require a UUID app
  row to exist in `apps`;
- ontology visibility rules allow a tenant app to see platform-owned baseline domains, but write
  authorization still requires either same-tenant ownership or an explicit `write`/`admin` grant.

The current onboarding docs imply that `POST /v1/tenants/{tenant_id}/apps` is enough to make an app
ready for writes and that a visible domain can be used for tenant write bootstrap. In practice,
those assumptions are incomplete, which causes `403` and `500` failures during integration work.

## Proposed Solution

Create a focused follow-up change that tightens the KG write contract:

1. define a single write-ready app identity contract shared by authentication, app provisioning, and
   PostgreSQL-backed write persistence;
2. require seeded or documented write-capable app identities to be UUID-backed and durably
   provisioned in the authoritative app registry used by write-path constraints;
3. clarify and verify ontology ownership semantics so platform-owned baseline domains remain visible
   but not tenant-writable without an explicit cross-tenant write grant;
4. update integration guidance so tenant-specific ontology bootstrap uses the tenant that will own
   the write path unless the integration intentionally relies on shared-domain grants.

## Scope

### In scope

- write-ready identity contract for app creation, seeded app fixtures, and graph-version writes
- verification that authenticated write callers have a durable UUID-backed app record
- requirements and docs for platform-owned versus tenant-owned ontology domains on the write path
- integration guidance for tenant-scoped ontology bootstrap and cross-tenant write grants

### Out of scope

- BAS-specific adapter implementation details outside the `kg-service` contract
- broader tenancy redesign beyond the current tenant/app/grant model
- new sharing models beyond existing same-tenant ownership and grant-based write access

## Success Criteria

- an app that is documented or returned as active by the service is also write-ready for the
  PostgreSQL-backed graph identity path
- seeded local write-capable app identities are compatible with UUID and FK constraints
- platform-visible domains are clearly documented and tested as read-visible but not tenant-writable
  by default
- tenant bootstrap guidance makes it explicit when a domain must be tenant-owned versus
  cross-tenant grant-backed
