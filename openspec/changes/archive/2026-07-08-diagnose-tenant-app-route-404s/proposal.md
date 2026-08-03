# Proposal: Diagnose Tenant/App Route 404s in Local Runtime

## Problem

Local debugging is blocked by tenant/app APIs returning `404` even though the HTTP routes are
registered in `internal/bootstrap/app.go`.

Current investigation shows the problem is hard to localize because several different failure modes
collapse into the same user-visible outcome:

- the router may not match the incoming path;
- the route may match in Kratos/gorilla-mux, but path params may not be propagated into
  `net/http.Request.PathValue(...)`;
- the request may match the route but fail later in the handler/service layer;
- local callers may provide a tenant slug where the service expects a tenant ID.

The current API docs list the endpoints, but they do not give a focused debug path for local access
flows or clearly show which seeded tenant IDs, app IDs, and bearer tokens should be used.

## Proposed Solution

Create a focused change that makes tenant/app `404` behavior observable and reproducible in local
development:

1. verify the confirmed bridge mismatch between gorilla-mux path vars and `r.PathValue(...)`;
2. add route-level and handler-level verification for tenant/app endpoints;
3. capture the known local failure mode where route params use a slug instead of the seeded tenant
   ID;
4. add lightweight diagnostics so local debugging can tell whether `404` came from route matching,
   path-var propagation, or resource lookup;
5. document the local seed identities and a short debug checklist built around `/v1/access/resolve`.

## Scope

### In scope

- tests for tenant/app route registration and dispatch
- tests for mux-vars to `PathValue` propagation on tenant/app routes
- tests for service-level `ErrNotFound` on missing tenant/app lookups
- docs for seeded local tenant IDs, app IDs, and bearer tokens
- safe diagnostics for distinguishing router misses, path-var propagation failures, and resource
  misses during local debugging

### Out of scope

- redesigning tenant/app authorization rules
- changing tenant path parameters from IDs to slugs
- broader API error model refactors outside tenant/app access flows

## Success Criteria

- local debugging can prove whether a tenant/app `404` came from router mismatch, path-var
  propagation failure, or service lookup
- automated coverage includes the gorilla-mux to `PathValue` propagation failure mode
- automated coverage includes the slug-vs-ID failure mode for tenant/app routes
- docs show the exact seeded credentials and tenant IDs needed to exercise local tenant/app APIs
- the change narrows the likely root cause of current local `404`s without widening access scope
