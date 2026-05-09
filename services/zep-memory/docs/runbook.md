---
id: DOC-S06
service: zep-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-memory — Runbook

## Startup Sequence
1. Load config (Viper)
2. Connect PostgreSQL
3. Run migrations (bun)
4. Connect NATS JetStream
5. Establish gRPC connections to zep-thread, zep-search
6. Start gRPC server on port 9063
7. Start health HTTP server on port 12063

## Health Checks

| Endpoint | Expected |
|----------|----------|
| `grpc.health.v1.Health/Check` | `SERVING` |
| `GET :12063/healthz` | `200 OK` |
| `GET :12063/readyz` | `200 OK` (DB + NATS + thread-client + search-client) |

## Common Errors

| Error | Diagnosis | Resolution |
|-------|-----------|------------|
| `ErrSessionEnded` | Session ended_at set | Create new session |
| `ErrEmptyMessages` | Empty messages in PutMemory | Include at least 1 message |
| `ErrMessageNotFound` | Invalid message UUID | Verify UUID and project scope |
| `thread_client timeout` | zep-thread unreachable | Check zep-thread health |
| `search_client timeout` | zep-search unreachable | GetMemory will degrade gracefully (messages-only) |
| `NATS publish failed` | JetStream unavailable | Graph extraction will be retried |

## Performance Notes

- PutMemory target: < 200ms (sync path only)
- Graph extraction: 10-20s async via NATS → zep-graph
- GetMemory with facts: ~50ms (depends on zep-search response)

## Escalation

| Severity | Contact | SLA |
|----------|---------|-----|
| P0 | Zep on-call → Platform Team | 15 min |
| P1 | Zep team lead | 1 hour |
| P2 | Zep dev team | Next business day |
