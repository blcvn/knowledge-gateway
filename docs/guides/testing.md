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

## 4. Backend-Specific Tests

Some backend checks are tag-gated or env-gated:

- `go test -tags nebula ./internal/platform/graphstore`
- `go test ./internal/platform/graphstore`
- `go test ./internal/platform/vectorstore`

These are useful when changing adapter translation or conformance behavior.

## 5. Tenant And App Test Setup

For integration tests that need a tenant-specific identity:

1. Create a tenant.
2. Create an app under that tenant.
3. Use the returned API key.
4. Verify identity with `GET /v1/access/resolve`.
5. Add grants before testing cross-tenant visibility.

When a test writes or reads tenant-scoped data, always prefer a dedicated test tenant/app pair over reusing a shared key.

