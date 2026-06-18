# API Spec Maintenance

Use this checklist whenever the HTTP or MCP transport surface changes.

## Normative Sources

- Runtime route inventory: [internal/bootstrap/app.go](/Users/anhdt/vnpay/knowledge/kg-service/internal/bootstrap/app.go)
- Shared REST envelopes: [internal/httpapi/respond/respond.go](/Users/anhdt/vnpay/knowledge/kg-service/internal/httpapi/respond/respond.go)
- Pagination normalization: [internal/httpapi/respond/list.go](/Users/anhdt/vnpay/knowledge/kg-service/internal/httpapi/respond/list.go)
- Published machine-readable spec: [openapi.yaml](./openapi.yaml)
- Published human-readable API reference: [README.md](./README.md)

## Required Update Rule

- If a route is added, removed, renamed, or moved in `internal/bootstrap/app.go`, update `docs/api/openapi.yaml` and `docs/api/README.md` in the same workstream.
- If request or response shapes change in access, ontology, read, search, write, integrity, or MCP handlers/types, update the relevant schemas in `docs/api/openapi.yaml` in the same workstream.
- If `respond.ErrorEnvelope`, list envelope fields, or pagination normalization changes, update the shared conventions in both `docs/api/openapi.yaml` and `docs/api/README.md`.

## Repeatable Validation

Run the route inventory check:

```bash
bash scripts/check-api-route-inventory.sh
```

This compares the registered routes in `internal/bootstrap/app.go` with the published method/path pairs in `docs/api/openapi.yaml`.

## Manual Review Checklist

- Confirm `/healthz` remains public and `/v1/*` auth requirements are still described correctly.
- Confirm list endpoints still document `limit`, `cursor`, default limit `20`, and max limit `100` where applicable.
- Confirm dynamic route behavior for `POST /v1/kg/read/template/{domain_id}/{template_name}` still matches runtime behavior.
- Confirm async acknowledgment notes for `POST /v1/kg/write/nodes` and `POST /v1/kg/write/ingest/document` still match handler status codes.
- Confirm MCP connect/message semantics still match the current HTTP transport.
