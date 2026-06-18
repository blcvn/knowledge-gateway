# VM Deployment

Use the VM path when you want to run `kg-service` on a standalone host under your own supervisor, such as `systemd`.

## What This Path Starts

- a locally built `kg-service` binary
- the HTTP listener on `KG_HTTP_HOST:KG_HTTP_PORT`
- connectivity to reachable Postgres and Redis endpoints
- a runtime profile selected from `KG_RUNTIME_PROFILE`

## Prerequisites

- Go toolchain or a built service binary
- access to the target VM
- reachable Postgres and Redis endpoints
- the SQL schema applied before the service starts

## Build And Run

```bash
make deploy-vm
```

The script builds the binary when needed and starts it with the current environment.

Set `KG_RUNTIME_PROFILE` before starting the service to choose the graph and vector adapter pair.

## Apply Schema

If the VM points at a fresh database, apply the schema first:

```bash
KG_MIGRATION_DSN='postgres://user:pass@postgres.example.internal:5432/kg_service?sslmode=disable' \
make migrate
```

## Verify The Deployment

```bash
KG_BASE_URL=http://127.0.0.1:8082 \
KG_API_KEY=... \
make integration-test
```

For the full runtime profile validation flow, run `scripts/validate-runtime-profile.sh` after the service is reachable.

## Notes

- The VM path is intentionally thin: the script prepares and launches the process, while your supervisor handles restarts and persistence.
- Use the same environment variables as the Compose and Kubernetes paths so the runtime behavior stays comparable.
