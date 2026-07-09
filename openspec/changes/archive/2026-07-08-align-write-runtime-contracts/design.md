# Design: Align Write Runtime Identity and Domain Ownership Contracts

## Overview

This change closes two runtime contract gaps that currently show up as integration blockers:

- identity resolution and app provisioning succeed in the access layer, but the write plane can
  still fail later because `owner_app_id` must satisfy PostgreSQL UUID and foreign-key
  requirements;
- effective ontology visibility includes platform-owned domains, but write authorization still
  depends on domain ownership or an explicit cross-tenant write grant.

The goal is not to redesign tenancy. The goal is to make the current contract explicit, durable,
and testable so integrations do not discover hidden write-path assumptions only after reaching the
real persistence layer.

## Goals

- ensure an authenticated app identity is compatible with the durable write path
- ensure `CreateApp` and seeded fixtures produce write-capable app identities when the service
  advertises them for local or integration use
- distinguish domain visibility from write authority in both requirements and guides
- make tenant-specific ontology bootstrap deterministic for downstream integrations

## Non-Goals

- changing the tenant/app/grant permission model
- introducing slug-based domain ownership resolution
- defining BAS-specific bootstrap behavior inside `kg-service`

## Proposed Work

### 1. Unify write-ready app identity semantics

The service currently has an access-plane identity store and a PostgreSQL-backed write plane. This
change should require a shared contract between them:

- an app that can authenticate into protected write endpoints must have a durable app identity that
  satisfies the write-plane schema;
- app IDs used by graph identity, scope leases, and related write metadata must be valid UUIDs when
  stored in PostgreSQL tables that reference `apps(id)`;
- app-creation flows and local seed fixtures must either provision that durable app row or avoid
  being presented as write-capable credentials.

This closes the class of bugs where authentication succeeds but `kg_graph_identifiers.owner_app_id`
fails at insert time.

### 2. Make seeded and documented credentials consistent with the durable schema

The local runtime and docs currently expose seeded admin credentials that do not match the UUID
contract enforced by the graph-version schema. This change should require the seeded identities used
for write-path validation to be aligned with the authoritative `apps` table shape.

That alignment can be achieved by whichever implementation is smallest and safest, but the contract
should be:

- documented write-capable seed apps are UUID-backed;
- local fixtures include the durable rows needed by write-path FK checks;
- onboarding docs do not promote credentials that are incompatible with write persistence.

### 3. Clarify domain visibility versus write authority

`effective ontology` and `GetVisibleDomain(...)` intentionally expose platform-owned baseline
domains to tenant apps, but write authorization remains stricter:

- same-tenant ownership can write without an extra grant;
- cross-tenant writes require an active `write` or `admin` grant for the relevant owner and scope;
- visibility alone does not make a foreign-owned domain writable.

This should be captured explicitly in spec text, tests, and integration docs so callers stop
inferring that a visible platform domain is also a valid tenant write target.

### 4. Define tenant-scoped ontology bootstrap guidance

Integrations that create a tenant-specific ontology domain for subsequent tenant writes must create
that domain under the tenant that will own the writes, unless the integration intentionally wants a
shared foreign-owned domain with explicit grants.

The docs and requirements should make this sequence clear:

1. create tenant and app;
2. confirm identity with `/v1/access/resolve`;
3. create tenant-owned domain when the tenant will write into it directly;
4. use cross-tenant grants only when sharing into another owner domain is intentional;
5. treat platform-owned baseline domains as shared read-visible ontology unless explicit write
   delegation exists.

## Verification Plan

- tests cover app-creation and seeded-app write readiness against the durable write contract
- tests cover foreign-owned visible domains rejecting writes without `write` or `admin` grants
- docs show a tenant-owned bootstrap path and a separate cross-tenant shared-domain path
- local seeded identities listed in guides remain consistent with the durable schema used by
  graph-version writes
