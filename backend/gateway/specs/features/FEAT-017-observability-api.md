---
id: FEAT-017
title: Observability Console API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P2
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.11 Observability"
---

## Mục Tiêu

REST APIs cho Observability screen — metrics dashboard, distributed trace viewer, error explorer.

## Scope

### In Scope
- `GET /v1/console/observability/metrics` — Aggregated metrics (per engine latency, token usage, LLM costs)
- `GET /v1/console/observability/traces` — List distributed traces
- `GET /v1/console/observability/traces/{id}` — Trace detail with spans
- `GET /v1/console/observability/errors` — Error explorer (stack traces, failed retrievals)
- `GET /v1/console/observability/costs` — Cost analytics (LLM calls, token usage)

### Out of Scope
- Metric alerting configuration (Grafana responsibility)
- Log aggregation (ELK/Loki responsibility)

## Thiết Kế Kỹ Thuật

### Internal Architecture
- **Handler:** `adapter/http/observability_handler.go`
- **Proxy to:** `vnp-platform` (metrics aggregation), OTel collector (traces)
- Metrics: scrape Prometheus endpoints from all 18 services
- Traces: query Jaeger/Tempo backend via `vnp-platform`
- Errors: query structured error logs from `vnp-event`

## Acceptance Criteria
- [ ] AC-1: Metrics returns per-engine API latency (p50/p95/p99)
- [ ] AC-2: Metrics includes token usage and LLM cost breakdown
- [ ] AC-3: Trace list filterable by service, duration, status
- [ ] AC-4: Trace detail shows full span tree with timing
- [ ] AC-5: Error explorer shows stack traces, failed retrievals per engine
- [ ] AC-6: Cost analytics includes Memobase 3-call budget tracking
- [ ] AC-7: All endpoints require admin role

## Test Requirements
- Unit tests: Metrics aggregation, trace formatting
- Integration tests: Mock Prometheus + trace backend
- Minimum coverage: 80%
