# KG Service

Bootstrap workspace for the multi-tenant, domain-agnostic KG Service described in [docs/KG_Service_TDD_v1.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/KG_Service_TDD_v1.md).

## Current Scope

- Go service bootstrap with [`main.go`](/Users/anhdt/vnpay/knowledge/kg-service/main.go) delegating to a repository-owned command package in [cmd](/Users/anhdt/vnpay/knowledge/kg-service/cmd)
- Initial PostgreSQL-oriented schema migrations in [migrations](/Users/anhdt/vnpay/knowledge/kg-service/migrations)
- In-memory access and ontology bootstrap layers used to implement the first protected API slices
- Shared HTTP envelopes and status-code helpers aligned with the OpenSpec API conventions

## Documentation By Audience

- New integrators: [docs/guides/quickstart.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/guides/quickstart.md)
- Application developers: [docs/guides/integration.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/guides/integration.md)
- MCP and agent consumers: [docs/guides/mcp.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/guides/mcp.md)
- Common integration failures: [docs/guides/troubleshooting.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/guides/troubleshooting.md)
- Deployment and verification: [docs/deployment/README.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/deployment/README.md)
- Current API surface: [docs/api/README.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/api/README.md)
- Service operators: [docs/operations](/Users/anhdt/vnpay/knowledge/kg-service/docs/operations)

## Bootstrap Highlights

- `GET /healthz` is public; current `/v1/*` routes require `Authorization: Bearer <api_key>`.
- Local bootstrap ships seeded credentials and sample ontology content for evaluation only.
- Read, search, integrity, and worker behavior are available for local validation, but some runtime paths remain in-memory by design.
- `sample-policy` is the easiest seeded domain for end-to-end template and search testing.

## Implemented Endpoint Groups

- Access and tenant management
- Ontology management
- KG write, read, and search
- Integrity checks
- MCP session and message transport

## Full Route Inventory

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

## Local Run

```bash
make test
make run
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
- User guides index: [docs/guides/README.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/guides/README.md)
- Deployment guides: [docs/deployment/README.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/deployment/README.md)
- API reference: [docs/api/README.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/api/README.md)
- Operations runbooks: [docs/operations](/Users/anhdt/vnpay/knowledge/kg-service/docs/operations)
