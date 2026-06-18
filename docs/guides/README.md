# User Guides

These guides are for people integrating with `kg-service` during the current bootstrap phase.

## Start Here

- [Quickstart](./quickstart.md) for the fastest path from local run to the first authenticated request.
- [Integration Workflows](./integration.md) for tenant/app onboarding, ontology setup, write flows, and read/search flows.
- [Tenant And App Setup](./tenant-app-setup.md) for identity onboarding, app creation, grants, and visibility checks.
- [Testing Guide](./testing.md) for unit tests, integration stacks, and runtime profile validation.
- [Deployment Guide](./deployment.md) for Compose, Kubernetes, and VM user-facing deployment steps.
- [MCP Integration](./mcp.md) for session-based MCP usage over the current HTTP transport.
- [Troubleshooting](./troubleshooting.md) for common integration failures before switching to operator runbooks.

## Choose The Right Document

- Use these guides when you are consuming the service as an API or MCP client.
- Use [API Reference](../api/README.md) when you need the current endpoint groups, common conventions, and exact route inventory.
- Use [Operations Runbooks](../operations) when the service itself needs recovery or operator action.
- Use [TDD](../KG_Service_TDD_v1.md) for architecture and target-state context rather than day-to-day bootstrap integration steps.
- Use [Requirements](../requirements/README.md) for the PRD, URD, and SRS documents.

## Bootstrap Scope Note

The current guides describe the repository's local bootstrap environment:

- seeded test credentials are available only for local development and tests
- some backing stores and projection paths are still in-memory
- production provisioning, persistence, and operational hardening are follow-up work

## Maintenance Rule

- Update these guides in the same workstream whenever onboarding, auth, ontology setup, write flows, read/search behavior, or MCP integration behavior changes materially.
- Update [API Reference](../api/README.md) alongside these guides when route inventory or shared request/response conventions change.
