# Quickstart

This guide gets you from a fresh checkout to the first authenticated `kg-service` request.

## Before You Start

- The current bootstrap server listens on `0.0.0.0:8082`.
- `GET /healthz` is public.
- The current `/v1/*` routes require `Authorization: Bearer <api_key>`.
- Bootstrap credentials and sample ontology data exist only for local development and tests.

## Run The Service

```bash
make test
make run
```

In another terminal:

```bash
curl -s http://127.0.0.1:8082/healthz
```

Expected result: a JSON response with `service`, `postgres`, and `redis` fields.

## Use A Bootstrap API Key

Available local bootstrap keys:

- Platform admin: `kgsk_platform_admin`
- Tenant alpha admin: `kgsk_test_alpha_admin`
- Tenant alpha app: `kgsk_test_alpha`
- Tenant beta app: `kgsk_test_beta`

Example:

```bash
export KG_BASE_URL=http://127.0.0.1:8082
export KG_API_KEY=kgsk_test_alpha_admin

curl -s \
  -H "Authorization: Bearer ${KG_API_KEY}" \
  "${KG_BASE_URL}/v1/access/resolve"
```

`/v1/access/resolve` is a good first check because it proves authentication and shows the caller's visible owners.

## First Read Workflow

List visible templates for the seeded `sample-policy` domain:

```bash
curl -s \
  -H "Authorization: Bearer ${KG_API_KEY}" \
  "${KG_BASE_URL}/v1/kg/read/templates?domain_id=sample-policy"
```

Run one of the active templates:

```bash
curl -s \
  -X POST \
  -H "Authorization: Bearer ${KG_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"params":{"topic_key":"returns"}}' \
  "${KG_BASE_URL}/v1/kg/read/template/sample-policy/action-guide"
```

This is the quickest way to confirm that auth, domain visibility, template activation, and the generic read route are working together.

When you need the freshest direct node view, use realtime mode:

```bash
curl -s \
  -H "Authorization: Bearer ${KG_API_KEY}" \
  "${KG_BASE_URL}/v1/kg/read/nodes/<node_id>?app_id=<app_id>&mode=realtime"
```

Realtime reads use the graph projection only when its sync version matches `relationshipdb`; otherwise the service falls back to `relationshipdb`.

## First Search Workflow

```bash
curl -s \
  -X POST \
  -H "Authorization: Bearer ${KG_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"query":"returns policy","domain_ids":["sample-policy"],"top_k":5}' \
  "${KG_BASE_URL}/v1/kg/search/semantic"
```

Search results are filtered by the caller's visibility and by deletion state.

## What To Read Next

- Go to [Integration Workflows](./integration.md) for tenant/app onboarding and ontology-first data modeling.
- Go to [Tenant And App Setup](./tenant-app-setup.md) when you want a dedicated checklist for creating tenants, apps, and grants.
- Go to [MCP Integration](./mcp.md) if your consumer is tool-oriented or agent-style.
- Go to [API Reference](../api/README.md) for route inventory, conventions, and endpoint grouping.

## Bootstrap Caveats

- Writes land in `relationshipdb` first, and graph/vector/full-text projections are synchronized asynchronously.
- Some read, search, sync, and integrity behavior still relies on in-memory bootstrap implementations.
- Caller-supplied `tenant_id` and `app_id` fields in JSON bodies are ignored by middleware; identity comes from the API key.
- The local seeded credentials are not a production provisioning model.
