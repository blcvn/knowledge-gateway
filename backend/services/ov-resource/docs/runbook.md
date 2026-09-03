---
id: DOC-S06
service: ov-resource
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# ov-resource — Runbook

## Startup

```bash
make run-ov-resource
docker compose up ov-resource postgresql nats ov-fs
```

**Prerequisites**: PostgreSQL + NATS + ov-fs must be running. tree-sitter requires CGo build.

## Shutdown

Graceful shutdown via SIGTERM (30s timeout). Active ingestion jobs complete; watch tasks stop polling.

## Health Check

```bash
curl http://localhost:9107/healthz
# Expected: {"status": "serving"}

curl http://localhost:9107/readyz
# Expected: {"status": "ready", "checks": {"db": "ok", "nats": "ok", "ov-fs": "ok"}}
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| gRPC UNAVAILABLE | Service not started | Check logs, verify port 9054 |
| Parse failures | Unsupported file format | Check parser registry, verify `filename` extension |
| tree-sitter crash | CGo segfault | Check tree-sitter grammar version, rebuild with CGo debug |
| Watch task stale | Polling stopped | Check `last_poll_at`, restart watch manager |
| ov-fs write failures | ov-fs down or path lock | Check ov-fs health |
| Large file timeout | File exceeds MAX_INGESTION_SIZE_MB | Increase limit or split file |
| Duplicate chunks | Re-ingestion without hash check | Verify `content_hash` dedup logic |

## Monitoring

- **Metrics**: Prometheus at `:9107/metrics`
  - `ov_resource_ingest_total` — ingestion count by parser type
  - `ov_resource_ingest_duration_seconds` — pipeline latency
  - `ov_resource_chunks_produced_total` — chunks generated
  - `ov_resource_watch_polls_total` — watch polling count
  - `ov_resource_parse_errors_total` — parse failures
- **Traces**: Jaeger via OTel — trace per ingestion pipeline
- **Logs**: Structured JSON via slog with `request_id`, `account_id`, `filename`

## Deployment Checklist

- [ ] PostgreSQL migrations applied (`make migrate-ov-resource`)
- [ ] NATS JetStream stream `openviking` exists
- [ ] ov-fs service healthy
- [ ] tree-sitter grammars built (if CGo enabled)

## Escalation

1. On-call checks logs + metrics dashboard
2. If parser issues → check tree-sitter + grammar versions
3. If ov-fs unreachable → escalate to ov-fs team
4. If unresolved in 15min → escalate to OpenViking team lead
