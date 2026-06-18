# Publish KG Service API Spec

## Why

The service already exposes a non-trivial HTTP contract for tenant management, ontology management, KG read/write/search, integrity checks, and MCP transport. That contract is currently discoverable only by reading router registration, handlers, DTO types, and the legacy TDD. This slows onboarding, increases the risk of integration drift, and makes it hard to tell which request/response shapes represent the current runtime baseline.

The repository needs a first-class API specification package that documents the current service surface in one place and can evolve with implementation changes.

## What Changes

- Publish a versioned API specification for the current HTTP surface, aligned to the routes registered in `internal/bootstrap/app.go`.
- Document shared API conventions including authentication, pagination, success envelopes, and structured error responses.
- Cover the current access, ontology, read, search, write, integrity, health, and MCP HTTP endpoints with concrete request/response schemas.
- Define a maintenance workflow so future route or schema changes update the API spec in the same change.

## Capabilities

- Versioned machine-readable API contract for the service
- Human-readable API reference for consumers and internal contributors
- Explicit documentation of auth, status-code, pagination, and dynamic template route conventions
- Change-time validation that the published spec stays aligned with runtime handlers

## Impact

- Makes the current API surface easier to consume without reverse-engineering the codebase.
- Reduces ambiguity between the historical TDD examples and the live service contract.
- Creates a stable baseline for future SDK generation, integration testing, and portal/client work.
