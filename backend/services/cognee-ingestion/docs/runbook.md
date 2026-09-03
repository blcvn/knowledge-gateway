---
id: DOC-S06
service: cognee-ingestion
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# cognee-ingestion — Runbook

## Startup

```bash
# From monorepo root
make run-cognee-ingestion

# Or via Docker Compose
docker compose up cognee-ingestion
```

**Prerequisites**: PostgreSQL, MinIO/S3, Redis, NATS must be running and accessible.

## Shutdown

Graceful shutdown via SIGTERM (30s timeout). In-flight uploads will complete before shutdown.

```bash
# Graceful stop
docker compose stop cognee-ingestion

# Force stop (after timeout)
docker compose kill cognee-ingestion
```

## Health Check

```bash
# HTTP health endpoint
curl http://localhost:9091/healthz
# Expected: {"status": "serving"}

# Readiness check (includes DB connectivity)
curl http://localhost:9091/readyz
# Expected: {"status": "ready", "checks": {"postgres": "ok", "minio": "ok", "nats": "ok"}}

# gRPC health check
grpcurl -plaintext localhost:9011 grpc.health.v1.Health/Check
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| gRPC UNAVAILABLE | Service not started or port conflict | Check logs: `docker compose logs cognee-ingestion`. Verify port 9011 is free |
| Upload timeout | MinIO/S3 unreachable | Check S3_ENDPOINT connectivity. Verify MinIO is running |
| Dataset creation fails | PostgreSQL connection error | Check PG_HOST/PG_PORT. Verify DB exists and user has permissions |
| NATS publish error | NATS JetStream not configured | Verify NATS is running. Check stream `cognee` exists |
| High memory usage | Large file uploads buffering | Adjust `MAX_FILE_SIZE`. Check concurrent upload count |
| Slow text extraction | CPU-bound PDF/DOCX parsing | Scale horizontally. Consider offloading to worker pool |

## Deployment Procedure

1. Build new image: `make docker-build-cognee-ingestion`
2. Run migrations: `make migrate-cognee-ingestion`
3. Deploy to staging: `kubectl apply -k deploy/kubernetes/overlays/staging`
4. Verify health: `curl https://staging.vnp-memory.example/healthz`
5. Deploy to production: `kubectl apply -k deploy/kubernetes/overlays/production`

## Rollback Procedure

1. Revert Kubernetes deployment: `kubectl rollout undo deployment/cognee-ingestion`
2. Verify health endpoint returns 200
3. Check logs for errors: `kubectl logs -l app=cognee-ingestion --tail=100`

## Monitoring

- **Metrics**: Prometheus at `:9091/metrics`
  - `cognee_ingestion_uploads_total` — total upload count by status
  - `cognee_ingestion_upload_duration_seconds` — upload latency histogram
  - `cognee_ingestion_file_size_bytes` — uploaded file size distribution
- **Traces**: Jaeger via OTel collector
- **Logs**: Structured JSON via slog with `request_id`, `tenant_id`, `dataset_id`

## Escalation

1. On-call engineer checks logs + metrics dashboard
2. If unresolved in 15min → escalate to Cognee team lead
3. If data loss suspected → escalate to platform engineering
