# API Reference

This is the human-readable reference for the current bootstrap `kg-service` HTTP surface.

## Published Artifacts

- Machine-readable contract: [openapi.yaml](./openapi.yaml)
- Maintenance workflow: [maintenance.md](./maintenance.md)
- User workflows: [docs/guides](../guides)

`openapi.yaml` is the normative contract. This document summarizes the same surface for faster scanning.

## Shared Conventions

### Authentication

- `GET /healthz` is public.
- Current `/v1/*` routes require `Authorization: Bearer <api_key>`.
- `GET /v1/mcp/connect` also requires the same bearer token because it is under the protected surface.

### Error Envelope

4xx and 5xx REST responses use:

```json
{
  "error": {
    "code": "MACHINE_READABLE_CODE",
    "message": "Human-readable message",
    "details": {}
  }
}
```

Current common codes include:

- `BAD_REQUEST`
- `INVALID_API_KEY`
- `FORBIDDEN`
- `NOT_FOUND`
- `VALIDATION_FAILED`
- `TOO_MANY_REQUESTS`
- `REQUEST_TIMEOUT`
- `INTERNAL_ERROR`

### Pagination

List endpoints use:

- `data`
- `next_cursor`
- `has_more`

Current list endpoints normalize pagination as:

- default `limit = 20`
- maximum `limit = 100`
- `cursor` is an opaque marker taken from the previous response

This applies to:

- `GET /v1/tenants/{tenant_id}/apps`
- `GET /v1/access/grants`
- `GET /v1/access/audit`
- `GET /v1/kg/read/templates`

### Identity Sanitization

Middleware strips caller-supplied `tenant_id` and `app_id` fields from JSON bodies before handler execution. Caller identity is derived from the bearer token, not trusted from request payloads.

## Endpoint Reference

### Health

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| `GET` | `/healthz` | `200` | Public health payload with `service`, `postgres`, and `redis` fields. |

### Access And Tenant Management

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| `GET` | `/v1/access/resolve` | `200` | Returns `tenant_id`, `app_id`, and `visible_owners`. |
| `POST` | `/v1/access/grants` | `201` | Creates a sharing grant from JSON request body. |
| `GET` | `/v1/access/grants` | `200` | Supports `grantor_tenant_id`, `grantee_tenant_id`, `limit`, `cursor`. |
| `DELETE` | `/v1/access/grants/{id}` | `200` | Returns revoke status and `revoked_at` when present. |
| `GET` | `/v1/access/audit` | `200` | Supports `resource_owner_tenant_id`, `limit`, `cursor`. |
| `POST` | `/v1/tenants` | `201` | Creates a tenant from `slug`, `name`, `tier`. |
| `GET` | `/v1/tenants/{tenant_id}` | `200` | Returns a tenant record. |
| `PUT` | `/v1/tenants/{tenant_id}` | `200` | Updates `tier` and `default_sharing_policy`. |
| `DELETE` | `/v1/tenants/{tenant_id}` | `200` | Returns `{ id, status }` for suspended tenant. |
| `POST` | `/v1/tenants/{tenant_id}/apps` | `201` | Creates an app and may include plaintext `api_key`. |
| `POST` | `/v1/tenants/{tenant_id}/apps/{app_id}/rotate-key` | `200` | Returns new plaintext `api_key` and `rotated_at`. |
| `GET` | `/v1/tenants/{tenant_id}/apps` | `200` | Supports `limit` and `cursor`. |
| `DELETE` | `/v1/tenants/{tenant_id}/apps/{app_id}` | `200` | Returns revoke status and `revoked_at` when present. |

### Ontology

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| `POST` | `/v1/tenants/{tenant_id}/ontology/domains` | `201` | Creates a domain. |
| `POST` | `/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types` | `201` | Creates a node type schema. |
| `POST` | `/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types` | `201` | Creates a relationship type schema. |
| `GET` | `/v1/tenants/{tenant_id}/ontology/effective` | `200` | Returns effective visible domains. |
| `POST` | `/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates` | `201` | Creates a query template. |
| `PUT` | `/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate` | `200` | Activates a query template. |
| `POST` | `/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config` | `201` | Upserts lifecycle and authority config. |
| `PUT` | `/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/search-profile` | `201` | Upserts the domain search profile. |
| `GET` | `/v1/ontology/domains/{domain_id}/search-profile` | `200` | Resolves the effective search profile for the caller. |
| `POST` | `/v1/tenants/{tenant_id}/ontology/query-strategies` | `201` | Creates a query strategy. |
| `PUT` | `/v1/tenants/{tenant_id}/ontology/query-strategies/{key}` | `200` | Updates a query strategy. |
| `GET` | `/v1/ontology/query-strategies` | `200` | Lists available query strategies. |
| `GET` | `/v1/ontology/domains/{domain_id}` | `200` | Returns domain, node types, relationship types, templates, and status config when present. |

### Knowledge Read And Search

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| `GET` | `/v1/kg/read/templates` | `200` | Supports `domain_id`, `limit`, `cursor`. |
| `POST` | `/v1/kg/read/template/{domain_id}/{template_name}` | `200` | Generic dynamic route for active templates only; body may include `app_id` and `mode`. |
| `GET` | `/v1/kg/read/nodes/{id}` | `200` | Returns visible node details and relationship IDs; supports query params `app_id` and `mode=realtime|non-realtime`. |
| `POST` | `/v1/kg/search/semantic` | `200` | Uses `query`, `domain_ids`, `top_k`; retrieval comes from projection search stores. |
| `POST` | `/v1/kg/search/rag` | `200` | Uses same request shape as semantic search in current runtime. |
| `POST` | `/v1/kg/search/fulltext` | `200` | Uses `query`, `domain_ids`, `top_k`, `mode`, `fields`. |
| `POST` | `/v1/kg/search/hybrid` | `200` | Combines semantic and full-text ranking with `semantic_weight` and `fts_operator`. |
| `POST` | `/v1/kg/search/graph` | `200` | Executes a raw graph query with the same ACL and profile guardrails as template-backed reads. |

Read and search behavior notes:

- visibility and ACL rules filter observed results
- deleted content is excluded from normal search responses
- `POST /v1/kg/read/template/{domain_id}/{template_name}` is generic and resolved from active query templates, not hard-coded per domain
- `GET /v1/kg/read/nodes/{id}?mode=non-realtime` always reads from the graph projection
- `GET /v1/kg/read/nodes/{id}?mode=realtime` first compares graph sync version with `relationshipdb`; if the graph projection is stale, the response falls back to `relationshipdb`
- template-backed graph reads accept the same `mode` semantics through the JSON request body
- search flows read from vector/full-text projection stores rather than `relationshipdb`
- `POST /v1/kg/search/graph` reuses the same visibility and search-profile checks as the template-backed read path

### Knowledge Write

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| `POST` | `/v1/kg/write/nodes` | `202` | Accepted after persisting to `relationshipdb`; downstream projection sync remains async. |
| `POST` | `/v1/kg/write/nodes/bulk` | `202` | Bulk async acknowledgments with per-node `node_id` entries after `relationshipdb` persistence. |
| `PUT` | `/v1/kg/write/nodes/{id}` | `200` | Updates node properties, visibility, or external ref. |
| `DELETE` | `/v1/kg/write/nodes/{id}` | `200` | Soft-delete response with `node_id` and `is_deleted`. |
| `POST` | `/v1/kg/write/relationships` | `201` | Creates relationship in `relationshipdb` and returns `relationship_id`. |
| `POST` | `/v1/kg/write/relationships/bulk` | `201` | Bulk relationship creation in `relationshipdb` with per-item `relationship_id` entries. |
| `DELETE` | `/v1/kg/write/relationships/bulk` | `200` | Bulk relationship delete by relationship IDs. |
| `DELETE` | `/v1/kg/write/nodes:by-external-ref-prefix` | `200` | Soft-deletes matching nodes by external ref prefix. |
| `POST` | `/v1/kg/write/ingest/document` | `202` | Async ingest acknowledgment with `job_id` and `status`. |
| `GET` | `/v1/kg/write/ingest/jobs/{job_id}` | `200` | Returns ingest job status, created node count, and errors. |

Write behavior notes:

- application requests write only to `relationshipdb`
- graph, vector, and full-text stores are maintained by async sync jobs/workers
- scope-sensitive writes validate an existing graph scope identifier before creating cross-graph edges

### Integrity

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| `GET` | `/v1/kg/integrity/tenant/{tenant_id}` | `200` | Tenant-level integrity report. |
| `GET` | `/v1/kg/integrity/missing-bridges` | `200` | Requires query `tenant_id`; returns list envelope of bridge candidates. |
| `GET` | `/v1/kg/integrity/orphans` | `200` | Requires query `tenant_id`; returns orphaned relationships and vector docs. |
| `POST` | `/v1/kg/integrity/repair/rebuild` | `200` | Requires query `tenant_id`; rebuilds graph/vector/full-text projections. |
| `POST` | `/v1/kg/integrity/repair/purge-orphans` | `200` | Requires query `tenant_id`; purges orphaned projection data. |
| `GET` | `/v1/kg/metrics` | `200` | Returns worker and projection lag metrics, including realtime fallback and graph scope conflict counters. |

### MCP Transport

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| `GET` | `/v1/mcp/connect` | `200` | Returns `text/event-stream` and emits `session` event with `session_id`. |
| `POST` | `/v1/mcp/messages/{session_id}` | `200` | Accepts JSON-RPC request and returns JSON-RPC result or error. |

MCP transport notes:

- `tools/list` enumerates the current tool set
- `tools/call` invokes a named tool with JSON arguments
- rate-limit and invalid-session failures are returned as JSON-RPC errors rather than REST error envelopes

## Validation Workflow

Use the repeatable route inventory check:

```bash
bash scripts/check-api-route-inventory.sh
```

If this check fails, update `docs/api/openapi.yaml` and this reference in the same workstream as the runtime change.

## Related Docs

- [Quickstart](../guides/quickstart.md)
- [Integration Workflows](../guides/integration.md)
- [MCP Integration](../guides/mcp.md)
- [Troubleshooting](../guides/troubleshooting.md)
- [Operations Runbooks](../operations)
- [API Spec Maintenance](./maintenance.md)
