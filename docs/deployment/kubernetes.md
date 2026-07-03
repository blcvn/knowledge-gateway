# Kubernetes Deployment

Use Kubernetes when you want to run `kg-service` in a cluster and you already have reachable Postgres and Redis endpoints.

## What This Path Starts

- a `kg-service` Deployment
- a `kg-service` Service on port `8082`
- readiness and liveness checks against `/healthz`
- a runtime profile injected through `KG_RUNTIME_PROFILE`, `GRAPH_ADAPTER`, `VECTOR_ADAPTER`, and any required profile-specific variables such as `KG_GRAPH_DATABASE`

## Prerequisites

- `kubectl` access to a target cluster
- an image that the cluster can pull or otherwise already has available
- reachable Postgres and Redis endpoints
- the SQL schema applied before the pod starts

## Deploy The App

```bash
KG_IMAGE=kg-service:local \
KG_RUNTIME_PROFILE=pgvector-neo4j \
KG_POSTGRES_HOST=postgres.example.internal \
KG_POSTGRES_PASSWORD=... \
KG_REDIS_HOST=redis.example.internal \
make deploy-k8s
```

If your cluster uses a registry image instead of a local build, point `KG_IMAGE` at that image reference.

For profiles that use a graph database name or vector endpoint override, export the corresponding variables before deployment. See [Environment Variables](./environment.md) for the full inventory.

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

For a profile-aware smoke test that also exercises write, read, search, and reconciliation checks, run `scripts/validate-runtime-profile.sh`.

## Notes

- This path keeps the runtime and deployment concerns separate from cluster-specific storage design, but now makes the selected backend profile explicit.
- The manifests in `deploy/k8s/` are intentionally small and focus on the app workload rather than on managed database provisioning.
