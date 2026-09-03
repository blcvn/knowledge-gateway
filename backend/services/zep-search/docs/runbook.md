---
id: DOC-S06
service: zep-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-search — Runbook

## Startup Sequence
1. Load config (Viper)
2. Connect Redis (validate with PING)
3. Connect NATS JetStream
4. Verify Graphiti HTTP connectivity
5. Start NATS consumers for graph cache invalidation events
6. Start gRPC server on port 9065
7. Start health HTTP on port 12065

## Health Checks

| Endpoint | Expected |
|----------|----------|
| `grpc.health.v1.Health/Check` | `SERVING` |
| `GET :12065/readyz` | `200 OK` (Redis + NATS + Graphiti) |

## Common Errors

| Error | Diagnosis | Resolution |
|-------|-----------|------------|
| `redis timeout` | Redis unreachable | Check Redis cluster; search still works (cache miss) |
| `graphiti search timeout` | Search takes > 30s | Check Graphiti service health |
| `cache miss storm` | Many concurrent cache misses | Scale instances; consider cache warming |

## Performance Notes
- Cache TTL: 30s (short for temporal freshness)
- RRF/MMR/EpisodeMentions: Low latency (~10ms)
- CrossEncoder: Higher latency (~100ms, neural reranking)
- NodeDistance: Medium (~50ms, graph traversal)

## Escalation

| Severity | Contact | SLA |
|----------|---------|-----|
| P0 | Zep on-call | 15 min |
| P1 | Zep team lead | 1 hour |
