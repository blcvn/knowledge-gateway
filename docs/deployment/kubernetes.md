# Kubernetes Deployment

Use Kubernetes when you want to run `kg-service` in a cluster and you already have reachable Postgres and Redis endpoints.

## What This Path Starts

- a `kg-service` Deployment
- a `kg-service` Service on port `8082`
- readiness and liveness checks against `/healthz`

## Prerequisites

- `kubectl` access to a target cluster
- an image that the cluster can pull or otherwise already has available
- reachable Postgres and Redis endpoints
- the SQL schema applied before the pod starts

## Deploy The App

```bash
KG_IMAGE=kg-service:local \
KG_POSTGRES_HOST=postgres.example.internal \
KG_POSTGRES_PASSWORD=... \
KG_REDIS_HOST=redis.example.internal \
make deploy-k8s
```

If your cluster uses a registry image instead of a local build, point `KG_IMAGE` at that image reference.

## Apply Schema

If Postgres is not already migrated, run the migration helper against the target database before or during rollout:

```bash
KG_MIGRATION_DSN='postgres://user:pass@postgres.example.internal:5432/kg_service?sslmode=disable' \
make migrate
```

## Verify The Deployment

```bash
KG_BASE_URL=http://service.example.internal:8082 \
KG_API_KEY=... \
make integration-test
```

## Notes

- This path keeps the runtime and deployment concerns separate from cluster-specific storage design.
- The manifests in `deploy/k8s/` are intentionally small and focus on the app workload rather than on managed database provisioning.
