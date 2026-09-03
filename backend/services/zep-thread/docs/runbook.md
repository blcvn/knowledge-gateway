---
id: DOC-S06
service: zep-thread
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-thread — Runbook

## Startup / Shutdown

### Startup Sequence
1. Load config (Viper: YAML + ENV)
2. Connect PostgreSQL (validate with ping)
3. Run migrations
4. Connect NATS JetStream
5. Start gRPC server on port 9062
6. Start health HTTP server on port 12062

### Graceful Shutdown
1. Stop accepting new connections
2. Drain in-flight requests (30s)
3. Release advisory locks
4. Close NATS, PostgreSQL
5. Flush OTel spans

## Health Checks

| Endpoint | Expected Response |
|----------|-------------------|
| `grpc.health.v1.Health/Check` | `SERVING` |
| `GET :12062/healthz` | `200 OK` |
| `GET :12062/readyz` | `200 OK` (DB + NATS connected) |

## Common Errors & Resolution

| Error | Diagnosis | Resolution |
|-------|-----------|------------|
| `ErrSessionNotFound` | session_id not found or soft-deleted | Verify session_id and project scope |
| `ErrSessionEnded` | Attempting to modify ended session | Session was ended; create new session |
| `ErrLockTimeout` | Advisory lock timeout after 15 retries | Check for long-running transactions; increase timeout |
| `ErrSessionAlreadyExists` | Duplicate session_id in project | Use UpsertSession instead or different session_id |

## Escalation

| Severity | Contact | SLA |
|----------|---------|-----|
| P0 | Zep on-call → Platform Team | 15 min |
| P1 | Zep team lead | 1 hour |
| P2 | Zep dev team | Next business day |
