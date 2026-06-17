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
- `GET /v1/kg/read/templates`
- `POST /v1/kg/read/template/{domain_id}/{template_name}`
- `GET /v1/kg/read/nodes/{id}`

Current bootstrap flow seeds the core legal ontology through the ontology service APIs at startup, including domains, node types, relationship types, templates, and status configuration.
Current audit coverage in the bootstrap slice includes grant create/revoke events, read/write path mutations, and owner-scoped audit retrieval for tenant admins.

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
- OpenSpec change: [openspec/changes/bootstrap-kg-service-from-tdd/tasks.md](/Users/anhdt/vnpay/knowledge/kg-service/openspec/changes/bootstrap-kg-service-from-tdd/tasks.md)
