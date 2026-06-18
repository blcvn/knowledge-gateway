# KG Service

Bootstrap workspace for the multi-tenant, domain-agnostic KG Service described in [docs/KG_Service_TDD_v1.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/KG_Service_TDD_v1.md).

## Current Scope

- Go service bootstrap with stdlib HTTP server in [cmd/kg-service/main.go](/Users/anhdt/vnpay/knowledge/kg-service/cmd/kg-service/main.go)
- Initial PostgreSQL-oriented schema migrations in [migrations](/Users/anhdt/vnpay/knowledge/kg-service/migrations)
- In-memory access and ontology bootstrap layers used to implement the first protected API slices
- Shared HTTP envelopes and status-code helpers aligned with the OpenSpec API conventions

## Implemented Endpoints

- `GET /healthz`
- `GET /v1/access/resolve`
- `POST /v1/access/grants`
- `GET /v1/access/grants`
- `DELETE /v1/access/grants/{id}`
- `GET /v1/access/audit`
- `POST /v1/tenants`
- `GET /v1/tenants/{tenant_id}`
- `PUT /v1/tenants/{tenant_id}`
- `DELETE /v1/tenants/{tenant_id}`
- `POST /v1/tenants/{tenant_id}/apps`
- `POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key`
- `GET /v1/tenants/{tenant_id}/apps`
- `DELETE /v1/tenants/{tenant_id}/apps/{app_id}`
- `POST /v1/tenants/{tenant_id}/ontology/domains`
- `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types`
- `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types`
- `GET /v1/tenants/{tenant_id}/ontology/effective`
- `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates`
- `PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate`
- `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config`
- `GET /v1/ontology/domains/{domain_id}`
- `POST /v1/kg/write/nodes`
- `PUT /v1/kg/write/nodes/{id}`
- `DELETE /v1/kg/write/nodes/{id}`
- `POST /v1/kg/write/relationships`
- `POST /v1/kg/write/ingest/document`
- `GET /v1/kg/write/ingest/jobs/{job_id}`
- `GET /v1/kg/read/templates`
- `POST /v1/kg/read/template/{domain_id}/{template_name}`
- `GET /v1/kg/read/nodes/{id}`
- `POST /v1/kg/search/semantic`
- `POST /v1/kg/search/rag`
- `GET /v1/kg/integrity/tenant/{tenant_id}`
- `GET /v1/kg/integrity/missing-bridges?tenant_id=...`
- `GET /v1/mcp/connect`
- `POST /v1/mcp/messages/{session_id}`

Current bootstrap flow seeds a small domain-neutral sample ontology through the ontology service APIs at startup, including sample domains, node types, relationship types, templates, and status configuration.
The bootstrap now registers five active templates for `sample-policy`, and the generic read route exercises them through the same template execution path used by arbitrary tenant-defined domains.
Current audit coverage in the bootstrap slice includes grant create/revoke events, read/write path mutations, and owner-scoped audit retrieval for tenant admins.
List endpoints now return standard envelopes with `data`, `next_cursor`, and `has_more`, and accept `limit`/`cursor` query parameters.

## Search Projection Notes

The current semantic search slice uses the in-memory write projection as the source for the `kg_vectors` payload shape during bootstrap. The result metadata follows the projected vector contract:

- `node_id`
- `node_type`
- `domain_id`
- `owner_tenant_id`
- `owner_app_id`
- `acl_visible_to`
- `is_deleted`
- `status_value`
- `authority_score`
- `domain_props`

Search requests always enforce server-side ACL filtering and deletion-state filtering, and explicit `domain_ids` must refer to visible domains.

## Read Safeguards

The read service now enforces a small max-row guard and a timeout guard while walking compiled templates. Those safeguards keep bootstrap tests predictable and provide an early stop signal even before a dedicated graph backend is wired in.

## Integrity Notes

Integrity endpoints are currently backed by the in-memory write projection and ontology cross-domain rules during bootstrap. They report tenant-scoped drift summaries and missing-bridge candidates so the TDD contract is present while the worker-side reconciliation report is available during bootstrap.

## Worker Notes

The repository now includes an in-memory worker runtime that polls outbox events, projects nodes and relationships into graph/vector mirrors, applies status cascades, performs ACL fanout/invalidation hooks for grant changes, and produces reconciliation drift reports over the projected state. It is intentionally lightweight and test-driven, so the production queueing layer remains separate follow-up work.

## Rate Limiting

REST endpoints and MCP tool calls share a simple tenant-tier limiter in bootstrap. The current bootstrap policy is intentionally lightweight and in-memory, with tier-specific windows applied after identity resolution so REST and MCP behave consistently during local development and tests.

## MCP Notes

The MCP surface uses connection-level session creation over SSE and JSON-RPC message posting for tool invocation. Tool implementations are thin wrappers over the existing REST/service layer so MCP and REST share the same ACL and validation behavior during bootstrap.

## Local Run

```bash
go test ./...
go run ./cmd/kg-service
```

Default HTTP address: `0.0.0.0:8082`

## Bootstrap Test Credentials

These are seeded only in the in-memory bootstrap layer for local development and tests:

- Platform admin API key: `kgsk_platform_admin`
- Tenant alpha admin API key: `kgsk_test_alpha_admin`
- Tenant alpha app API key: `kgsk_test_alpha`
- Tenant beta app API key: `kgsk_test_beta`

## Current Architecture Note

The repo currently uses in-memory stores and an in-process TTL cache abstraction for the access and ontology slices. That is intentional for bootstrap speed and testability; the PostgreSQL and Redis integrations are not wired into runtime persistence yet.

## Reference

- TDD: [docs/KG_Service_TDD_v1.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/KG_Service_TDD_v1.md)
- OpenSpec change: [openspec/changes/remove-legal-domain-coupling/tasks.md](/Users/anhdt/vnpay/knowledge/kg-service/openspec/changes/remove-legal-domain-coupling/tasks.md)
- Operations runbooks: [docs/operations](/Users/anhdt/vnpay/knowledge/kg-service/docs/operations)
