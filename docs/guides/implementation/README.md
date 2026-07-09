# kg-service Integration Bundle (For Partners)

This folder is a self-contained set of docs for a partner team integrating with
`kg-service` over REST or MCP. Send this whole folder when a new system wants
to integrate — it does not depend on the rest of the repo.

## Scope

- Supported protocols: REST (HTTP+JSON) and MCP (HTTP/SSE, JSON-RPC tool-calling).
- gRPC is **not** exposed to external clients. `kg-service` uses gRPC only
  internally to talk to its vector store (Milvus); there is no gRPC surface
  for integrators.
- The service is currently in its bootstrap phase: seeded credentials,
  in-memory projection paths, and async projection sync reflect current
  behavior, not a finalized production contract. Flag this to partners so
  they set expectations accordingly.

## Read In This Order

1. [Quickstart](./quickstart.md) — run one authenticated request end to end.
2. [Integration Workflows](./integration.md) — auth, onboarding, ontology setup,
   write flows, read/search flows, sharing grants, integrity checks.
3. [Tenant And App Setup](./tenant-app-setup.md) — condensed onboarding
   checklist (tenant, app, grants, visibility).
4. [MCP Integration](./mcp.md) — use this instead of, or alongside, REST if the
   partner's consumer is an agent or tool-calling runtime.
5. [API Reference](./api-reference.md) and [openapi.yaml](./openapi.yaml) —
   endpoint-level contract; `openapi.yaml` is the normative source.
6. [Troubleshooting](./troubleshooting.md) — common consumer-side failures
   before escalating back to the kg-service team.

## Source Of Truth

These files are curated, partner-facing copies of:

- `docs/guides/quickstart.md`, `integration.md`, `tenant-app-setup.md`,
  `mcp.md`, `troubleshooting.md`
- `docs/api/README.md`, `docs/api/openapi.yaml`

Internal-only references (operator runbooks under `docs/operations`,
`deployment.md`, `testing.md`, the CodeGraph sync bridge, and repo-maintenance
scripts) were removed or replaced with plain-text notes since partners don't
have access to those paths.

When the source files change materially, refresh this bundle in the same
workstream so it doesn't drift.
