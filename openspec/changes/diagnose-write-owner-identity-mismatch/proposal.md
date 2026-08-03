# Proposal: Diagnose and Align Write Owner Identity Mapping

## Problem

`kg-service` still does not pass scope 1 end-to-end for the supported onboarding and first-write
flow.

Current runtime evidence shows:

- `POST /v1/kg/write/nodes` still returns `500 Internal Server Error`;
- the underlying PostgreSQL error is `kg_graph_identifiers_owner_tenant_id_fkey` and may also
  include the paired `owner_app_id` foreign-key contract;
- the caller has already authenticated successfully, so the failure happens after auth resolution
  and inside the write-plane owner persistence path.

This means the current service contract is still incomplete: an authenticated `tenant_id`/`app_id`
pair is reaching the write plane, but `kg-service` has not yet proven that the exact IDs carried by
that authenticated identity are the same durable owner IDs expected by PostgreSQL.

The broader `new-app-write-readiness` change already aims to make onboarding durable, but scope 1
is still blocked by a narrower unresolved question: which owner IDs are actually being set on the
write path, where do they diverge from durable records, and how does the service surface that
mismatch before it becomes an opaque backend `500`?

## Proposed Solution

Create a focused follow-up change that narrows the remaining blocker to write-owner identity
mapping:

1. define one explicit contract from authenticated access identity to the `owner_tenant_id` and
   `owner_app_id` persisted by the write plane;
2. instrument and verify the handoff points between app provisioning, auth resolution, session
   identity, and write persistence so the service can show which IDs it is using;
3. require supported onboarding flows to resolve the same durable tenant/app records that
   PostgreSQL foreign keys enforce;
4. replace raw FK-driven `500` outcomes with a controlled identity-readiness mismatch signal that
   makes the owner-ID gap diagnosable.

## Scope

### In scope

- authenticated identity to write-owner identity mapping
- owner-tenant and owner-app durability checks before graph identity persistence
- diagnostics and error handling for write-path owner-ID mismatches
- tests covering create-tenant -> create-app -> authenticate -> write with explicit owner-ID
  verification

### Out of scope

- broader ontology durability work
- redesigning the tenant/app/grant model
- unrelated projection, search, or sync optimizations

## Success Criteria

- the service can show exactly which `tenant_id` and `app_id` an authenticated write request will
  persist as `owner_tenant_id` and `owner_app_id`
- supported onboarding flows either provision matching durable owner rows or fail with an explicit
  identity-readiness contract error before a raw FK violation occurs
- `POST /v1/kg/write/nodes` no longer hides owner-identity drift behind a generic `500` for this
  known failure class
- integration coverage catches owner-ID mismatches before scope 1 is declared fully passing
