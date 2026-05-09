---
id: DOC-S06
service: zep-user
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-user — Runbook

## Startup / Shutdown

### Startup Sequence

```
1. Load config (Viper: YAML + ENV)
2. Connect PostgreSQL (validate with ping)
3. Run migrations (bun migration runner)
4. Connect NATS JetStream
5. Start gRPC server on port 9061
6. Start health HTTP server on port 12061
7. Register gRPC health check (grpc.health.v1)
```

### Graceful Shutdown

```
1. Stop accepting new gRPC connections
2. Drain in-flight requests (30s timeout)
3. Close NATS publisher
4. Close PostgreSQL pool
5. Flush OTel spans
```

## Health Checks

| Endpoint | Expected Response |
|----------|-------------------|
| `grpc.health.v1.Health/Check` | `SERVING` |
| `GET http://host:12061/healthz` | `200 OK {"status": "serving"}` |
| `GET http://host:12061/readyz` | `200 OK` (DB + NATS connected) |
| `GET http://host:12061/livez` | `200 OK` (process alive) |

## Common Errors & Resolution

| Error | Diagnosis | Resolution |
|-------|-----------|------------|
| `ErrUserNotFound` | user_id doesn't exist or soft-deleted | Verify user_id and project scope |
| `ErrUserAlreadyExists` | Duplicate user_id in same project | Use different user_id or update existing |
| `ErrInvalidUserID` | user_id contains non-alphanumeric chars | Ensure user_id matches `[a-zA-Z0-9_]+` |
| `ErrLockTimeout` | Advisory lock acquisition failed after 15 retries | Check for long-running transactions; consider increasing timeout |
| `NATS publish failed` | NATS JetStream unavailable | Check NATS cluster health; events will be retried |
| `PostgreSQL connection refused` | DB pool exhausted or down | Check `max_open_connections`; verify DB status |

## Deployment

### Docker

```bash
docker build -t zep-user -f services/zep-user/Dockerfile .
docker run -d --name zep-user \
  -e PG_DSN="postgres://..." \
  -e NATS_URL="nats://nats:4222" \
  -p 9061:9061 -p 12061:12061 \
  zep-user
```

### Rollback

```bash
# Revert to previous image tag
kubectl set image deployment/zep-user zep-user=zep-user:v1.0.0

# Verify rollback
kubectl rollout status deployment/zep-user
```

## Monitoring

- **Prometheus**: gRPC request duration, error rates, advisory lock contention
- **Jaeger**: Distributed traces via OTel
- **Alerts**: Advisory lock timeout rate > 5% → investigate concurrent metadata updates

## Escalation

| Severity | Contact | SLA |
|----------|---------|-----|
| P0 (service down) | Zep on-call → Platform Team | 15 min response |
| P1 (degraded) | Zep team lead | 1 hour response |
| P2 (bug) | Zep dev team | Next business day |
