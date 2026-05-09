---
id: DOC-S06
service: ov-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# ov-admin — Runbook

## Startup

```bash
make run-ov-admin
docker compose up ov-admin postgresql
```

**Prerequisites**: PostgreSQL must be running.

## Shutdown

Graceful shutdown via SIGTERM (15s timeout). Lightweight service — no long-running background tasks.

## Health Check

```bash
curl http://localhost:9109/healthz
# Expected: {"status": "serving"}

curl http://localhost:9109/readyz
# Expected: {"status": "ready", "checks": {"db": "ok"}}
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| gRPC UNAVAILABLE | Service not started | Check logs, verify port 9056 |
| `account not found` | Invalid account_id | Verify account exists in DB |
| `duplicate user` | User already exists in account | Check UNIQUE constraint |
| API key validation slow (~100ms) | Argon2id is CPU-intensive | Expected; consider Redis cache for validated keys |
| Health aggregation timeout | Downstream OV service slow/down | Check individual service health |
| `permission denied` | Role insufficient for operation | Verify caller's role vs required role |
| Auth mode mismatch | Running `api_key` mode without keys | Switch to `dev` mode for local development |

## Monitoring

- **Metrics**: Prometheus at `:9109/metrics`
  - `ov_admin_accounts_total` — total accounts
  - `ov_admin_users_total` — total users
  - `ov_admin_api_key_validations_total` — key validations (success/failure)
  - `ov_admin_health_check_duration_seconds` — aggregated health check time
- **Traces**: Jaeger via OTel — trace per CRUD operation
- **Logs**: Structured JSON via slog with `request_id`, `account_id`

## Deployment Checklist

- [ ] PostgreSQL migrations applied (`make migrate-ov-admin`)
- [ ] `AUTH_MODE` configured correctly
- [ ] Initial ROOT API key created (bootstrap)
- [ ] All OV services reachable for health aggregation

## Bootstrap Procedure

```bash
# 1. Start in dev mode
AUTH_MODE=dev make run-ov-admin

# 2. Create first account
grpcurl -d '{"name": "Default"}' ov-admin:9056 openviking.admin.v1.OvAdminService/CreateAccount

# 3. Create ROOT API key
grpcurl -d '{"account_id": "default", "role": "root"}' ov-admin:9056 openviking.admin.v1.OvAdminService/CreateAPIKey
# Save the raw_key — it will not be shown again!

# 4. Switch to api_key mode
AUTH_MODE=api_key make run-ov-admin
```

## Escalation

1. On-call checks logs + metrics
2. If auth issues → verify AUTH_MODE + API key status
3. If DB connectivity → escalate to DBA team
4. If unresolved in 15min → escalate to OpenViking team lead
