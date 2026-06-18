# Publish User Integration Guides

## Why

The repository currently documents the service mostly from an implementation or operator perspective. Consumers can see route inventories in `README.md`, operational recovery steps under `docs/operations/`, and architectural intent in the TDD, but they still lack a practical guide that explains how to start using the service and how to integrate against it end to end.

That gap makes first-time adoption harder than necessary. Integrators must infer the correct order of operations, bootstrap credentials, authentication flow, ontology setup sequence, and read/write/search usage from scattered code and reference material.

## What Changes

- Add user-facing usage and integration documentation for the current bootstrap service surface.
- Document the recommended onboarding journey from obtaining credentials through first successful API and MCP calls.
- Explain common integration flows for tenant/app setup, ontology management, data write/read/search, and integrity checks.
- Distinguish consumer-facing integration guides from internal operator runbooks and architecture documents.

## Capabilities

- Quickstart guide for first-time users of `kg-service`
- Integration guide for application developers consuming REST and MCP surfaces
- Task-oriented examples that show the expected sequence of calls
- Clear boundaries between consumer docs, API reference docs, and operations runbooks

## Impact

- Reduces onboarding time for internal and external integrators.
- Makes the bootstrap environment easier to evaluate without reverse-engineering the codebase.
- Provides a documentation layer that complements the API spec with usage-oriented guidance.
