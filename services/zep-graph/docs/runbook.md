---
id: DOC-S06
service: zep-graph
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-graph — Runbook

## Startup Sequence
1. Load config (Viper)
2. Connect Neo4j (validate with session test)
3. Connect NATS JetStream
4. Start NATS consumer for `zep.memory.messages.ingested`
5. Verify Graphiti HTTP connectivity
6. Start gRPC server on port 9064
7. Start health HTTP server on port 12064

## Health Checks

| Endpoint | Expected |
|----------|----------|
| `grpc.health.v1.Health/Check` | `SERVING` |
| `GET :12064/healthz` | `200 OK` |
| `GET :12064/readyz` | `200 OK` (Neo4j + NATS + Graphiti) |

## Common Errors

| Error | Diagnosis | Resolution |
|-------|-----------|------------|
| `graphiti timeout` | LLM extraction exceeds 60s | Check LLM service health; increase timeout |
| `neo4j connection refused` | Neo4j cluster down | Restart Neo4j; check connection pool |
| `NATS consumer lag` | Extraction queue building up | Scale zep-graph instances; check LLM throughput |
| `extraction failed (max retries)` | Graphiti returned 3 consecutive errors | Check Graphiti logs; message will be dead-lettered |

## Performance Notes

- Entity extraction: 10-20s per message batch (LLM-bound)
- NATS ack_wait: 120s to accommodate LLM processing
- Max 3 delivery attempts before dead-letter
- Neo4j pool size 50 to handle concurrent graph queries

## Monitoring

- **NATS consumer lag**: Alert if > 100 messages pending
- **Extraction latency**: P99 should be < 30s
- **Neo4j query time**: Alert if > 500ms for read queries

## Escalation

| Severity | Contact | SLA |
|----------|---------|-----|
| P0 | Zep on-call → Platform Team | 15 min |
| P1 | Zep team lead | 1 hour |
