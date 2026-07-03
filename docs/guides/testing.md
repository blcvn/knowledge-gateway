# Testing Guide

This guide covers the repeatable test and validation commands for `kg-service`.

## 1. Unit And Package Tests

Run the full Go test suite:

```bash
make test
```

Use this after code changes that affect request handlers, runtime wiring, adapters, or validation logic.

## 2. Default Integration Smoke

Start the integration Compose stack:

```bash
make deploy-compose-integration
```

Then run the repeatable integration validation:

```bash
KG_BASE_URL=http://127.0.0.1:8082 \
KG_API_KEY=kgsk_platform_admin \
make integration-test
```

This checks:

- `GET /healthz`
- `GET /v1/access/resolve`
- `GET /v1/kg/read/templates`
- `POST /v1/kg/read/template/{domain_id}/{template_name}`

## 3. Runtime Profile Validation

Start the runtime validation Compose stack:

```bash
KG_RUNTIME_PROFILE=qdrant-nebula make deploy-compose-runtime-validation
```

Then run the profile-aware validation:

```bash
KG_BASE_URL=http://127.0.0.1:8082 \
KG_API_KEY=kgsk_platform_admin \
KG_RUNTIME_PROFILE=qdrant-nebula \
make validate-runtime-profile
```

This checks:

- write
- read
- semantic search
- integrity
- reconciliation

## 4. CodeGraph Runtime Validation

Use the dedicated Compose flow when you need to verify the `code-graph` ontology, sync, and query path together:

```bash
export EMBEDDING_PROVIDER=http
export EMBEDDING_URL=https://genai.vnpay.vn/aigateway/embed/v1/embeddings
export EMBEDDING_MODEL=v_search
export EMBEDDING_API_KEY=replace-with-local-secret
make validate-codegraph-runtime
```

This flow:

- boots the dedicated Postgres + Memgraph + Qdrant Compose stack
- sets Memgraph `vm.max_map_count` to at least `524288` so the graph backend can start cleanly on Docker Desktop
- creates or reuses a tenant admin app for CodeGraph validation
- waits for `GET /healthz`
- bootstraps and verifies the `code-graph` ontology
- runs the `examples/codegraph` bridge in dry-run and sync modes using upsert semantics
- when the CodeGraph CLI is available, refreshes a dedicated probe symbol and verifies the same node advances to a newer sync version after an update
- confirms get/list, hybrid search, full-text search, and template-backed queries work

The endpoint and model values come from `tests/llm/embedding-vnp.txt`, but secrets must stay outside repo-owned files.

For reruns, you can skip completed setup steps:

```bash
make validate-codegraph-runtime ARGS="--skip-compose"
make validate-codegraph-runtime ARGS="--skip-compose --skip-sync"
```

## 5. Backend-Specific Tests

Some backend checks are tag-gated or env-gated:

- `go test -tags nebula ./internal/platform/graphstore`
- `go test ./internal/platform/graphstore`
- `go test ./internal/platform/vectorstore`

These are useful when changing adapter translation or conformance behavior.

## 6. Tenant And App Test Setup

For integration tests that need a tenant-specific identity:

1. Create a tenant.
2. Create an app under that tenant.
3. Use the returned API key.
4. Verify identity with `GET /v1/access/resolve`.
5. Add grants before testing cross-tenant visibility.

When a test writes or reads tenant-scoped data, always prefer a dedicated test tenant/app pair over reusing a shared key.
