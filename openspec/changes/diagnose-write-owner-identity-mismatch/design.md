# Design: Diagnose and Align Write Owner Identity Mapping

## Overview

This change isolates the remaining scope 1 blocker to one concrete runtime seam:

- `CreateApp` and auth resolution produce an identity that can reach protected write endpoints;
- `CreateNodeWithContext` passes `actor.TenantID` and `actor.AppID` into the write/session path;
- PostgreSQL later rejects owner persistence because the effective `owner_tenant_id` and/or
  `owner_app_id` do not match the durable rows referenced by foreign keys.

The goal is not to finish every durable control-plane migration in this change. The goal is to
make write-owner identity mapping observable, contractually explicit, and safe enough that the
service either writes with the correct durable owner IDs or fails before the database reports an
opaque FK violation.

## Goals

- define the exact source of `owner_tenant_id` and `owner_app_id` for supported write requests
- verify the authenticated identity matches durable owner rows before graph identity persistence
- make owner-ID mismatch diagnosable in logs, errors, and tests
- unblock scope 1 by proving or falsifying the remaining owner-identity gap quickly

## Non-Goals

- redesigning authorization semantics
- migrating unrelated ontology or cache behavior
- changing BAS payload shape

## Proposed Work

### 1. Define one owner-identity handoff contract

The service currently relies on `access.Identity` values flowing into the write/session layer.
This change should make that handoff explicit:

- the authenticated `tenant_id` and `app_id` accepted on a supported write endpoint are the same
  IDs the service intends to persist as `owner_tenant_id` and `owner_app_id`;
- the service must not rewrite, infer, or silently substitute a different owner identity later in
  the write pipeline without making that transformation explicit and verified;
- any supported app returned by onboarding APIs must resolve to durable owner rows compatible with
  the PostgreSQL foreign keys behind graph identity persistence.

### 2. Add pre-persistence owner-readiness checks

The failing class should be detected before the database becomes the first place that notices it.
This change should require a preflight check that confirms:

- the authenticated `tenant_id` exists in the durable owner registry referenced by
  `owner_tenant_id`;
- the authenticated `app_id` exists in the durable owner registry referenced by `owner_app_id`;
- the app belongs to the same tenant identity the write plane is about to persist;
- the write path stops with a controlled contract error when any of those conditions fail.

The implementation can remain small, but the runtime behavior should no longer depend on a raw FK
violation to identify this mismatch.

### 3. Make mismatch diagnostics first-class

The current `500 Internal Server Error` is too coarse to close the bug quickly. This change should
require diagnostics that identify:

- the authenticated `tenant_id` and `app_id`;
- the owner IDs about to be written;
- which durable lookup failed or mismatched;
- whether the gap comes from tenant provisioning, app provisioning, or app-to-tenant ownership
  alignment.

The exact mechanism may be structured logs, surfaced error details, integration assertions, or a
combination of them. The contract requirement is that this bug class becomes directly diagnosable
without attaching to raw PostgreSQL internals.

### 4. Prove the create -> authenticate -> write sequence with explicit owner assertions

The repo already tests pieces of onboarding and writes, but scope 1 needs a sharper proof for this
exact regression class. Coverage should:

1. create a tenant;
2. create an app under that tenant;
3. authenticate using the returned API key;
4. resolve the resulting identity;
5. attempt a supported node write;
6. assert that the owner IDs used by the write path match durable tenant/app rows and do not fail
   on FK constraints.

Negative coverage should also prove that an owner mismatch is surfaced as a controlled readiness
error rather than a generic internal failure.

## Verification Plan

- integration coverage proves the authenticated identity and persisted owner identity are the same
  durable tenant/app pair for the supported flow
- tests cover owner-tenant missing, owner-app missing, and cross-tenant app mismatch cases
- logs or structured diagnostics expose the concrete owner IDs involved in a mismatch
- `POST /v1/kg/write/nodes` no longer returns a generic `500` for this known owner-identity drift
  class
