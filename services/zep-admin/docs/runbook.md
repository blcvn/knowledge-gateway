---
id: DOC-S06
service: zep-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-admin — Runbook

## Startup Sequence
1. Load config (Viper)
2. Connect PostgreSQL
3. Run migrations (projects, api_keys tables)
4. Connect NATS JetStream
5. Establish gRPC connections to all 5 backend services
6. Start gRPC server on port 9066
7. Start health HTTP on port 12066

## Health Checks

| Endpoint | Expected |
|----------|----------|
| `grpc.health.v1.Health/Check` | `SERVING` |
| `GET :12066/healthz` | `200 OK` |

## AggregatedHealth Behavior

```
SERVING → all 5 backend services reporting SERVING
DEGRADED → at least 1 backend NOT_SERVING but others SERVING
NOT_SERVING → all backends NOT_SERVING
```

## Common Errors

| Error | Diagnosis | Resolution |
|-------|-----------|------------|
| `DEGRADED status` | One+ backend unhealthy | Check individual service health |
| `API key validation failed` | Key revoked or expired | Generate new key |
| `Project cascade failed` | Dependent data in other services | Check cascade events in NATS |

## API Key Security Notes

- Raw key shown only once at creation (never stored)
- Only SHA-256 hash stored in database
- Key prefix (8 chars) stored for identification
- Revocation is immediate (revoked_at timestamp)

## Escalation

| Severity | Contact | SLA |
|----------|---------|-----|
| P0 | Zep on-call → Platform Team | 15 min |
| P1 | Zep team lead | 1 hour |
