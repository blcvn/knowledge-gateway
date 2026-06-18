# Tasks

## Milestone: `docs/api`

- [ ] Create a versioned machine-readable API specification for the live HTTP surface.
- [ ] Create a human-readable API reference that explains endpoint groups, authentication, pagination, envelopes, and dynamic template routing.
- [ ] Document that `/healthz` is public and that all current `/v1/*` routes require `Authorization`.

## Milestone: `access`

- [ ] Document tenant, app, grant, audit, and resolve endpoints with their path/query parameters and current response schemas.
- [ ] Document list pagination semantics for app, grant, audit, and template listing endpoints, including default and maximum limits.
- [ ] Document rate-limit and authentication failure responses used by the access middleware.

## Milestone: `ontology`

- [ ] Document domain, node type, relationship type, query template, activation, status-field-config, and effective ontology endpoints.
- [ ] Capture current create and fetch response shapes from the ontology DTOs and handlers.

## Milestone: `kg-data`

- [ ] Document read template execution, node read, semantic search, RAG search, node write, relationship write, document ingest, and ingest-job status endpoints.
- [ ] Document current success status codes, especially `202 Accepted` for asynchronous write and ingest operations.
- [ ] Document dynamic template route semantics and the visibility/auth rules that affect read and search results.

## Milestone: `integrity-and-mcp`

- [ ] Document integrity endpoints and list envelope usage for missing-bridge results.
- [ ] Document the MCP HTTP transport endpoints, including session creation and message posting semantics at the HTTP layer.

## Milestone: `validation`

- [ ] Add a repeatable check or review step that compares the published route inventory with `internal/bootstrap/app.go`.
- [ ] Add a repeatable check or review step that keeps shared error and list envelope documentation aligned with `internal/httpapi/respond`.
- [ ] Reconcile any runtime/documentation mismatches discovered during spec authoring before the change is archived.
