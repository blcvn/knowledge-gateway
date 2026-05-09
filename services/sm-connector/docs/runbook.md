---
id: DOC-S06
service: sm-connector
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# sm-connector — Runbook

## Startup

```bash
# From monorepo root
make run-sm-connector
```

## Shutdown

Graceful shutdown via SIGTERM (30s timeout).

## Health Check

```bash
curl http://localhost:12075/healthz
# Expected: {"status": "serving"}
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|-----------|
| gRPC UNAVAILABLE | Service not started | Check logs, restart |
| High latency | DB connection pool exhausted | Scale DB connections |

## Monitoring

- **Metrics**: Prometheus at `:12075/metrics`
- **Traces**: Jaeger via OTel collector
- **Logs**: Structured JSON via slog

## Escalation

1. On-call engineer checks logs + metrics
2. If unresolved in 15min → escalate to Supermemory team lead
