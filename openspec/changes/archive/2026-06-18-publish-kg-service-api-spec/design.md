# Design

## Current Behavior

`kg-service` already serves a coherent HTTP API, but the contract is fragmented across several sources:

- route registration in `internal/bootstrap/app.go`
- request/response DTOs in `internal/access`, `internal/ontology`, `internal/read`, `internal/search`, `internal/write`, and `internal/integrity`
- shared response helpers in `internal/httpapi/respond`
- historical examples in `docs/KG_Service_TDD_v1.md`

That fragmentation makes the runtime contract harder to understand than it needs to be. It also creates a drift risk: developers can change a route, query parameter, or response shape in code without a clear requirement to update external-facing documentation at the same time.

## Problem Statement

The repository does not yet publish a current, versioned API spec that answers these questions in one place:

- which HTTP routes exist today
- which routes require `Authorization`
- what request bodies, path parameters, and query parameters each route accepts
- which status codes and response shapes each route returns
- how common behaviors such as pagination, malformed JSON handling, rate limiting, and structured errors work

Without a dedicated change, consumers must infer the contract from implementation details, and future API drift will remain easy to miss.

## Goals

- Publish a current API specification for the existing HTTP surface without redesigning endpoint behavior.
- Align the documented schemas with the DTOs and envelopes currently used by handlers.
- Make shared conventions explicit once instead of repeating them inconsistently across endpoint descriptions.
- Define a maintainable update pattern so the API spec remains part of normal feature work.

## Non-Goals

- Changing route structure, payload semantics, or status codes as part of this change.
- Introducing a new external API version.
- Redesigning MCP tool semantics beyond documenting the HTTP transport endpoints that expose MCP sessions and messages.

## Key Decisions

### 1. Treat the current router as the contract inventory

`internal/bootstrap/app.go` is the authoritative list of live routes. The API spec should enumerate every registered endpoint, including:

- public health endpoint
- tenant/app/grant/audit/resolve access endpoints
- ontology management endpoints
- read/search/write endpoints
- integrity endpoints
- MCP connect/message endpoints

This prevents the spec from documenting aspirational routes that are not actually wired in the service.

### 2. Derive schemas from existing DTOs and envelopes

Request and response shapes should be documented from the current Go types and response helpers:

- DTO packages define route-specific payload fields
- `respond.ErrorEnvelope` defines the common error shape
- `respond.ListEnvelope[T]` defines the collection envelope and pagination metadata
- handler status codes define the current success/error matrix

This keeps the published contract aligned with implementation instead of relying on stale examples from the TDD.

### 3. Publish both machine-readable and human-readable artifacts

The change should produce:

- a machine-readable API description suitable for validation and future tooling
- a concise human-readable guide that explains auth, conventions, and endpoint groupings

The machine-readable artifact should be the normative contract, while the human-readable document helps onboarding and quick consumption.

### 4. Explicitly document dynamic route conventions

Some behaviors are not obvious from static endpoint lists alone:

- `POST /v1/kg/read/template/{domain_id}/{template_name}` is a generic dynamic execution route
- most `/v1/*` routes require `Authorization`, while `/healthz` does not
- list endpoints share cursor/limit pagination with default and maximum limits
- middleware strips caller-supplied `tenant_id` and `app_id` fields from JSON bodies before handlers execute

These conventions should be documented as shared rules rather than left implicit in middleware and handler code.

### 5. Add a drift-prevention workflow

The spec must not become a one-time snapshot. The implementation plan should include validation so API-affecting changes update the published spec in the same workstream. Validation can be as simple as a route inventory check plus schema review gates, but it must make undocumented API changes visible during development and review.

## Risks And Mitigations

- Documenting current behavior may reveal inconsistencies across handlers.
  - Mitigation: document the runtime truth first, then open follow-up changes for any cleanup work rather than mixing redesign into this change.
- Machine-readable spec authoring can become verbose and slow to maintain.
  - Mitigation: keep shared components centralized and group endpoints by bounded capability area.
- Historical TDD examples may differ from runtime payloads.
  - Mitigation: treat code and shared responders as the baseline for this change, and use TDD examples only as supplemental context.

## Validation Strategy

- Verify the published endpoint inventory matches every route in `internal/bootstrap/app.go`.
- Verify documented request/response schemas match the active DTOs and response envelopes.
- Verify auth, pagination, and error conventions are documented once and referenced consistently.
- Add a repeatable validation step so future API-affecting changes cannot quietly bypass the spec.
