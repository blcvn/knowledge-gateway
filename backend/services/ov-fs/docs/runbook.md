---
id: DOC-S06
service: ov-fs
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# ov-fs — Runbook

## Startup

```bash
# From monorepo root
make run-ov-fs

# With dependency services
docker compose up ov-fs postgresql nats ov-crypto
```

**Prerequisites**: PostgreSQL + NATS + ov-crypto must be running.

## Shutdown

Graceful shutdown via SIGTERM (30s timeout). Pending writes complete before shutdown; PathLock releases all held locks.

## Health Check

```bash
curl http://localhost:9104/healthz
# Expected: {"status": "serving"}

curl http://localhost:9104/readyz
# Expected: {"status": "ready", "checks": {"db": "ok", "nats": "ok", "crypto": "ok"}}
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| gRPC UNAVAILABLE | Service not started | Check logs, verify port 9051 |
| `PathLock timeout` | Lock contention on busy paths | Increase `PATHLOCK_TIMEOUT_MS`, check for stuck operations |
| `ov-crypto unreachable` | Crypto service down | Check ov-crypto health, fallback reads plaintext |
| `File too large` | Exceeds MAX_FILE_SIZE_MB | Increase limit or chunk file |
| High latency on reads | DB connection pool exhausted | Scale `DB_MAX_CONNECTIONS` |
| `OVE1 parse error` | Corrupted encrypted file | Check ov-crypto key rotation status, verify envelope integrity |
| NATS publish failures | NATS connection lost | Check NATS cluster health, reconnect backoff active |

## Monitoring

- **Metrics**: Prometheus at `:9104/metrics`
  - `ov_fs_read_total` / `ov_fs_write_total` — operation counts
  - `ov_fs_read_duration_seconds` — read latency histogram
  - `ov_fs_pathlock_wait_seconds` — lock acquisition time
  - `ov_fs_encryption_duration_seconds` — encrypt/decrypt time
- **Traces**: Jaeger via OTel collector — trace per file operation
- **Logs**: Structured JSON via slog with `request_id`, `account_id`, `path`

## Deployment Checklist

- [ ] PostgreSQL migrations applied (`make migrate-ov-fs`)
- [ ] NATS JetStream stream `openviking` exists
- [ ] ov-crypto service healthy
- [ ] Encryption keys provisioned in KMS

## Escalation

1. On-call checks logs + metrics dashboard
2. If PathLock contention → check for stuck operations (>30s locks)
3. If encryption failures → escalate to ov-crypto / KMS team
4. If unresolved in 15min → escalate to OpenViking team lead
