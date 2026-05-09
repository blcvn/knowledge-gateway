---
id: DOC-S06
service: ov-session
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# ov-session — Runbook

## Startup

```bash
make run-ov-session
docker compose up ov-session postgresql nats bifrost ov-fs
```

**Prerequisites**: PostgreSQL + NATS + Bifrost + ov-fs must be running.

## Shutdown

Graceful shutdown via SIGTERM (60s timeout). Active commits complete both phases before shutdown; async extraction jobs are re-queued via NATS.

## Health Check

```bash
curl http://localhost:9106/healthz
# Expected: {"status": "serving"}

curl http://localhost:9106/readyz
# Expected: {"status": "ready", "checks": {"db": "ok", "nats": "ok", "bifrost": "ok", "ov-fs": "ok"}}
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| gRPC UNAVAILABLE | Service not started | Check logs, verify port 9053 |
| `session not found` | Stale session ID | Verify session exists in DB |
| `session already committed` | Re-commit attempt | Session is immutable after commit |
| Commit timeout | LLM extraction too slow | Check Bifrost latency, switch to `async` mode |
| Memory dedup failures | Embedding service down | Check Bifrost health, memories saved without dedup |
| ov-fs unreachable | ov-fs service down | Check ov-fs health; archives queued for retry |
| High memory usage | Large sessions (>1000 messages) | Enforce `MAX_MESSAGES_PER_SESSION` |
| WM update failures | JSONB serialization error | Check message format compliance |

## Monitoring

- **Metrics**: Prometheus at `:9106/metrics`
  - `ov_session_created_total` — sessions created
  - `ov_session_committed_total` — sessions committed
  - `ov_session_commit_duration_seconds` — commit latency (both phases)
  - `ov_session_extraction_total` — memory extractions by category
  - `ov_session_dedup_total` — dedup actions (create/merge/skip/archive)
  - `ov_session_messages_total` — messages added
- **Traces**: Jaeger via OTel — trace per commit with spans for archive + extract + dedup
- **Logs**: Structured JSON via slog with `request_id`, `session_id`, `account_id`

## Deployment Checklist

- [ ] PostgreSQL migrations applied (`make migrate-ov-session`)
- [ ] NATS JetStream stream `openviking` exists
- [ ] ov-fs service healthy
- [ ] Bifrost reachable and LLM model loaded
- [ ] Prompt templates deployed to `pkg/prompt/`

## Escalation

1. On-call checks logs + metrics dashboard
2. If LLM extraction fails → check Bifrost + prompt templates
3. If ov-fs unreachable → escalate to ov-fs team
4. If unresolved in 15min → escalate to OpenViking team lead
