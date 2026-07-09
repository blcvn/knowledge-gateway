# Design: Diagnose Tenant/App Route 404s in Local Runtime

## Overview

This change is an investigation and observability follow-up for tenant/app access APIs.

The current code already registers tenant/app routes in `internal/bootstrap/app.go`, and
`RequireIdentity` in `internal/access/middleware.go` returns `401`, `429`, or `400` rather than
`404`. A confirmed root cause is the bridge between Kratos/gorilla-mux routing and the repo's
`net/http` handlers: route params are attached by gorilla-mux, while tenant/app handlers read
`r.PathValue(...)`. If mux vars are not copied onto the request path-value map, handlers observe
empty `tenant_id` or `app_id` values and fall through to service-level not-found behavior.

There is also a second likely local failure path inside `internal/access/handler.go` and
`internal/access/service.go`, where valid `PathValue` reads can still map to `ErrNotFound` when the
path uses a slug or a missing tenant/app ID.

Seed data in `internal/access/seed.go` also separates `Tenant.ID` from `Tenant.Slug`, which makes a
slug-vs-ID mismatch a likely local repro for "route exists but response is 404".

## Goals

- prove the tenant/app routes are mounted and dispatch into the expected handlers
- prove tenant/app path params survive the Kratos/gorilla-mux to `net/http` bridge
- prove which tenant/app `404` responses are emitted by service lookup instead of router mismatch
- make the local happy path explicit with exact seeded IDs and bearer tokens
- reduce time-to-diagnosis for future route/debug regressions

## Non-Goals

- introducing slug-based routing
- changing seeded identities or auth semantics
- expanding diagnostics into a general production tracing framework

## Proposed Work

### 1. Route dispatch verification

Add focused HTTP tests around the tenant/app endpoints to show:

- registered routes accept the expected methods and path shapes;
- mux-provided route vars are visible through `r.PathValue(...)` in downstream handlers;
- matched requests reach the access handlers;
- unmatched paths still produce router-level `404`.

This keeps route registration regressions separate from handler/service regressions.

### 2. Path-value bridge verification

Add focused coverage for the bridge in `internal/bootstrap/app.go` so the repo explicitly proves:

- tenant/app routes registered through Kratos/gorilla-mux populate mux vars;
- the adapter layer copies those vars into `net/http.Request.PathValue(...)`;
- downstream handlers reading `r.PathValue("tenant_id")`, `r.PathValue("app_id")`, or similar keys
  receive the routed values.

This is the confirmed root cause behind at least one class of "route exists but response is 404"
bugs.

### 3. Service-layer `404` verification

Add or extend tests in access handler/service coverage to show:

- `GET /v1/tenants/{tenant_id}` returns not found when the tenant ID is missing;
- `POST /v1/tenants/{tenant_id}/apps` and `GET /v1/tenants/{tenant_id}/apps` return not found when
  the tenant does not exist;
- app operations return not found when the app does not belong to the requested tenant;
- a slug-shaped path value such as `test-alpha` reproduces the current local not-found behavior.

### 4. Local debug diagnostics

Add lightweight diagnostics that help local debugging without broadening error exposure. The exact
mechanism can stay implementation-sized, but it should make these states distinguishable:

- route was never matched;
- route matched but handler saw empty path vars because propagation failed;
- route matched and auth passed;
- handler/service returned `ErrNotFound` for tenant/app lookup.

Examples include structured logs, test-only assertions, or a narrowly scoped debug field in the
response path that is safe for local use.

### 5. Local docs and repro checklist

Document a short local checklist for tenant/app APIs:

1. call `/v1/access/resolve` with the bearer token to confirm the effective `tenant_id` and `app_id`;
2. if a route is registered but still returns `404`, confirm whether the handler is receiving empty
   `PathValue` fields;
3. use seeded tenant IDs, not slugs, in `/v1/tenants/{tenant_id}` paths;
4. use a tenant-admin token when exercising tenant-admin routes;
5. compare a known-good request and a slug-based request to confirm the current failure mode.

The docs should include the exact seeded local identities already present in `internal/access/seed.go`
so the debug path is reproducible without code spelunking.

## Verification Plan

- unit/integration tests cover route match, path-var propagation, and handler-level not-found for
  tenant/app endpoints
- docs are sufficient to reproduce both the happy path and the known slug-vs-ID failure locally
- a developer can identify the 404 source without reading bootstrap and access internals first
