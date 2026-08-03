# Solutions — Operator / SRE

> **Actor**: Operator / SRE  
> **Pain Points nguồn**: [operator-sre.md](../painpoints/operator-sre.md)  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Phân loại giải pháp

| Ký hiệu | Ý nghĩa |
|:---:|:---|
| ✅ **Đã có** | Sản phẩm đã hỗ trợ |
| 🔧 **Cần bổ sung** | Skeleton có, cần hoàn thiện |
| 🆕 **Đề xuất mới** | Chưa có, cần phát triển |

---

## PP-OPS-01 — Backend startup ordering không được orchestrate

### ✅ Giải pháp đã có trong sản phẩm

**Health check tại startup** — service validate connections khi boot:

```bash
GET /healthz
# → { "service": "ok", "postgres": "ok", "redis": "ok" }
# Nếu postgres/redis chưa ready → service báo trong health response
```

**Docker Compose với migration container** — ordering được handle:

```yaml
# deploy/compose/ setup:
# 1. migration container runs SQL schema → exitCode 0
# 2. kg-service starts AFTER migration complete (depends_on)
# → PostgreSQL guaranteed ready trước kg-service
```

**Environment variable startup validation**:

> "Invalid integer and duration values fail startup with configuration errors instead of panicking the process."

**Xem thêm**: [Deployment — Docker Compose](../../deployment/compose.md), [Environment Variables](../../deployment/environment.md#notes)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Backend connectivity wait mode**

```bash
# Environment variable để control startup behavior:
KG_STARTUP_WAIT_BACKENDS=true        # Chờ đến khi backends healthy
KG_STARTUP_WAIT_TIMEOUT_S=120        # Timeout tổng
KG_STARTUP_GRAPH_WAIT_TIMEOUT_S=90   # Timeout riêng cho graph backend (Neo4j chậm hơn)
KG_STARTUP_VECTOR_WAIT_TIMEOUT_S=60  # Timeout riêng cho vector backend
```

Startup behavior:
```
kg-service starting...
[00:01] ✓ postgres: connected (2ms)
[00:01] ✓ redis: connected (1ms)
[00:05] ⏳ graph_backend (memgraph): waiting... (attempt 3/30)
[00:30] ✓ graph_backend (memgraph): connected (28s)
[00:32] ✓ vector_backend (qdrant): connected (2s)
[00:32] ✓ All backends ready. Service starting.
```

**2. Kubernetes startup probe configuration**

```yaml
# Recommended Kubernetes probe:
startupProbe:
  httpGet:
    path: /healthz
    port: 8082
  failureThreshold: 30
  periodSeconds: 5
# → 30 × 5s = 150s cho backends khởi động
# → Service chỉ nhận traffic khi /healthz trả về ok cho tất cả subsystems
```

**3. Degraded mode documentation**

Document rõ behavior khi từng backend unavailable:

| Scenario | Write API | Graph Read | Vector Search | Full-text Search |
|:---|:---:|:---:|:---:|:---:|
| All backends up | ✅ | ✅ | ✅ | ✅ |
| Graph backend down | ✅ | ⚠️ realtime fallback | ✅ | ✅ |
| Vector backend down | ✅ | ✅ | ❌ 503 | ✅ |
| Redis down | ⚠️ degraded | ✅ | ✅ | ✅ |
| Postgres down | ❌ 503 | ✅ (stale) | ✅ (stale) | ✅ (stale) |

---

## PP-OPS-02 — Không có unified validation tool — validation không CI/CD friendly

### ✅ Giải pháp đã có trong sản phẩm

**Integration test script** với exit code:

```bash
KG_BASE_URL=http://127.0.0.1:8082 \
KG_API_KEY=kgsk_platform_admin \
make integration-test
# → Checks: /healthz, /v1/access/resolve, /v1/kg/read/templates, template execute
# → Exit code 0/1 → CI/CD friendly
```

**Runtime profile validation script**:

```bash
KG_BASE_URL=http://127.0.0.1:8082 \
KG_API_KEY=kgsk_platform_admin \
KG_RUNTIME_PROFILE=qdrant-nebula \
make validate-runtime-profile
# → Checks: write, read, semantic search, integrity, reconciliation
```

**CodeGraph full validation**:

```bash
make validate-codegraph-runtime
# → Full end-to-end: health, ontology bootstrap, sync, query validation
# → Supports --skip-compose, --skip-sync flags cho reruns
```

**API route inventory check**:

```bash
bash scripts/check-api-route-inventory.sh
# → Verify routes trong code match openapi.yaml
```

**Xem thêm**: [Testing Guide](../../guides/testing.md), [Integration Test](../../deployment/integration-test.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. kg-validate CLI — structured output**

```bash
kg-validate \
  --base-url http://staging.kg-service.internal:8082 \
  --api-key ${KG_API_KEY} \
  --profile qdrant-memgraph \
  --output json

# Structured output:
{
  "status": "partial_pass",
  "environment": "staging",
  "profile": "qdrant-memgraph",
  "checks": [
    { "name": "health", "status": "pass", "duration_ms": 12 },
    { "name": "postgres_connectivity", "status": "pass", "duration_ms": 3 },
    { "name": "graph_connectivity", "status": "pass", "duration_ms": 45 },
    { "name": "vector_connectivity", "status": "fail", "error": "connection refused", "duration_ms": 5001 },
    { "name": "write_node", "status": "skip", "reason": "depends on: vector_connectivity" },
    { "name": "read_template", "status": "pass", "duration_ms": 67 },
    { "name": "semantic_search", "status": "fail", "error": "vector backend unavailable" }
  ],
  "summary": { "total": 7, "pass": 4, "fail": 2, "skip": 1 }
}

# Exit code: 0 = all pass, 1 = some fail, 2 = critical fail
```

**2. Minimal validation suite** cho smoke check nhanh

```bash
kg-validate --quick \
  --base-url http://127.0.0.1:8082 \
  --api-key kgsk_platform_admin
# → Chỉ check: health, auth, read — trong <5s
# → Dùng trong CI/CD gate sau deployment
```

**3. Validation matrix per environment**

```yaml
# validation-matrix.yaml:
environments:
  local-dev:
    profile: memory
    checks: [health, auth, write, read]
    timeout_s: 30
  staging:
    profile: qdrant-memgraph
    checks: [health, auth, write, read, search, integrity]
    timeout_s: 120
  production:
    profile: qdrant-memgraph
    checks: [health, auth, read, search]  # Không write vào production
    timeout_s: 60
```

---

## PP-OPS-03 — Reconciliation drift không có automated detection và alert

### ✅ Giải pháp đã có trong sản phẩm

**Reconciliation và integrity endpoints** — đã có đầy đủ:

```bash
# Tenant-level integrity report:
GET /v1/kg/integrity/tenant/{tenant_id}
# → drift counts, sync version mismatches

# Missing bridges:
GET /v1/kg/integrity/missing-bridges?tenant_id={t}

# Orphaned data:
GET /v1/kg/integrity/orphans?tenant_id={t}

# Metrics — projection lag:
GET /v1/kg/metrics
# → worker lag, projection queue depth, realtime fallback counters
```

**Repair endpoints** — manual trigger:

```bash
# Rebuild projections từ source-of-truth:
POST /v1/kg/integrity/repair/rebuild?tenant_id={t}

# Purge orphaned projection data:
POST /v1/kg/integrity/repair/purge-orphans?tenant_id={t}
```

**Reconciliation runbook** — documented procedure:
[Reconciliation Incident Handling](../../operations/reconciliation-incident-handling.md), [Replica Recovery](../../operations/replica-recovery.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Prometheus metrics export** — cho alert rules:

```bash
GET /v1/kg/metrics/prometheus
# → text/plain Prometheus exposition format:
kg_projection_lag_total{tenant="payment-team",backend="graph"} 3
kg_projection_lag_total{tenant="payment-team",backend="vector"} 0
kg_orphan_count{tenant="payment-team",type="graph_node"} 0
kg_integrity_drift_count{tenant="payment-team"} 0
```

**Prometheus alert rule ví dụ**:
```yaml
- alert: KGProjectionDriftHigh
  expr: kg_projection_lag_total > 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "KG projection lag high for {{ $labels.tenant }}"
```

**2. Scheduled reconciliation job**

```bash
# Environment variable để bật reconciliation cron:
KG_RECONCILIATION_ENABLED=true
KG_RECONCILIATION_INTERVAL_MINUTES=60  # Check mỗi giờ
KG_RECONCILIATION_AUTO_REPAIR=false    # Chỉ report, không auto-repair
KG_RECONCILIATION_ALERT_WEBHOOK=https://hooks.slack.com/...
```

**3. Drift alert webhook**

```bash
# Khi drift vượt threshold → POST đến webhook:
POST ${KG_RECONCILIATION_ALERT_WEBHOOK}
{
  "alert": "projection_drift",
  "tenant_id": "payment-team",
  "drift_count": 47,
  "threshold": 10,
  "oldest_unsynced_ms": 45000,
  "repair_link": "POST /v1/kg/integrity/repair/rebuild?tenant_id=payment-team"
}
```

**4. Auto-reconnect reconciliation trigger**

Khi graph/vector backend reconnect sau downtime → tự động trigger reconciliation:
```
graph_backend disconnected at 14:30:00
graph_backend reconnected at 15:02:00
→ Auto-trigger: POST /v1/kg/integrity/repair/rebuild?tenant_id=* (all tenants)
→ Log: "Auto-reconciliation triggered after graph backend reconnect"
```

---

## PP-OPS-04 — Multi-environment config không có isolation guard

### ✅ Giải pháp đã có trong sản phẩm

**Profile-based backend selection** — giảm per-env manual config:

```bash
# Thay vì set GRAPH_ADAPTER, KG_GRAPH_ENDPOINT riêng lẻ:
KG_RUNTIME_PROFILE=qdrant-memgraph make deploy-compose
# → scripts/runtime-profile.sh tự derive GRAPH_ADAPTER và KG_GRAPH_ENDPOINT
```

**Environment validation at startup**:

> "Invalid integer and duration values fail startup with configuration errors instead of panicking the process."

**Dedicated .env per environment**:

```
deploy/compose/integration-test/         → có .env riêng
deploy/compose/runtime-validation/       → có .env riêng
deploy/compose/codegraph-runtime/.env    → source of truth cho CodeGraph stack
```

**Xem thêm**: [Environment Variables](../../deployment/environment.md), [Docker Compose](../../deployment/compose.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Startup env validation với clear errors**

```bash
# Khi start với thiếu/sai env vars:
kg-service --validate-env
# Output:
✓ KG_POSTGRES_HOST = "staging-db.internal"
✓ KG_REDIS_HOST = "staging-redis.internal"
✓ KG_RUNTIME_PROFILE = "qdrant-memgraph"
✗ KG_GRAPH_ENDPOINT = "http://prod-memgraph:7687"  # WARNING: looks like production URL in staging config
✓ KG_VECTOR_ENDPOINT = "http://staging-qdrant:6333"

WARNING: 1 validation warning found. Review before proceeding.
```

**2. Environment tagging**

```bash
# Environment variable:
KG_ENVIRONMENT=production  # | staging | development | local

# Effect:
# 1. Logs mọi request đều có: [env=staging] prefix
# 2. Metrics: kg_environment="staging" label
# 3. Health response:
GET /healthz → { "service": "ok", "environment": "staging", "profile": "qdrant-memgraph" }
```

**3. Config linting script**

```bash
bash scripts/validate-config.sh --env-file .env.staging
# → Check:
# ✓ No placeholder values (like "REPLACE_ME" or "TODO")
# ✓ No localhost URLs in non-development environments
# ✓ KG_ENVIRONMENT is set and valid
# ✓ All required vars for selected KG_RUNTIME_PROFILE are present
# ✗ KG_POSTGRES_PASSWORD contains "postgres" (default) — use strong password in staging
```

---

## PP-OPS-05 — Logs không có correlation ID xuyên write → projection → read

### ✅ Giải pháp đã có trong sản phẩm

**Structured logging** — service đã log theo format có thể parse:

```
2026-08-03T01:30:00Z INFO write_node node_id=abc-123 tenant_id=payment-team domain_id=payment
2026-08-03T01:30:01Z INFO projection_queued node_id=abc-123 backend=graph
2026-08-03T01:30:02Z INFO projection_completed node_id=abc-123 backend=graph version=42
```

**Integrity endpoints** — Operator có thể check state của specific node:

```bash
GET /v1/kg/integrity/tenant/{tenant_id}
# → drift counts cho phép narrow down investigation
```

**Xem thêm**: [Troubleshooting — Search Or Read Results Look Wrong](../../guides/troubleshooting.md#search-or-read-results-look-wrong), [Replica Recovery](../../operations/replica-recovery.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. W3C Trace Context propagation**

```http
POST /v1/kg/write/nodes
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
# → Request ID propagated through: HTTP → PostgreSQL write → outbox → projection worker → graph write → vector write
```

**Write response với trace ID**:

```json
POST /v1/kg/write/nodes → 202 Accepted
{
  "node_id": "abc-123",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "projection_job_id": "proj-job-xyz"
}
```

**2. Request ID trong mọi response**

```http
HTTP/1.1 202 Accepted
X-Request-ID: req-4bf92f35
X-Trace-ID: 4bf92f3577b34da6a3ce929d0e0e4736
Content-Type: application/json
```

**3. Trace lookup endpoint**

```bash
GET /v1/admin/traces/{trace_id}
# → End-to-end trace của một write operation:
{
  "trace_id": "4bf92f35...",
  "events": [
    { "step": "write_received", "timestamp": "14:30:00.001", "node_id": "abc-123" },
    { "step": "postgres_committed", "timestamp": "14:30:00.015", "version": 42 },
    { "step": "outbox_queued", "timestamp": "14:30:00.016" },
    { "step": "graph_projection_started", "timestamp": "14:30:00.340", "backend": "memgraph" },
    { "step": "graph_projection_completed", "timestamp": "14:30:00.780", "graph_version": 42 },
    { "step": "vector_projection_started", "timestamp": "14:30:00.341" },
    { "step": "vector_projection_completed", "timestamp": "14:30:01.120", "vector_version": 42 }
  ],
  "total_latency_ms": 1119,
  "status": "fully_projected"
}
```

**4. OpenTelemetry export**

```bash
# Environment variable:
KG_OTEL_ENDPOINT=http://jaeger:4317
KG_OTEL_SERVICE_NAME=kg-service

# → Traces exported to Jaeger/Tempo/Grafana
# → Operator dùng Jaeger UI để trace incident mà không cần grep logs
```

---

## Summary — Operator / SRE Solutions

| Pain Point | Đã có | Đề xuất mới | Priority |
|:---|:---:|:---:|:---:|
| PP-OPS-01: Backend startup race condition | ✅ Compose ordering + health check | 🆕 Wait-for-backend mode + degraded mode table | 🔴 P0 |
| PP-OPS-02: Validation không CI/CD friendly | ✅ make integration-test, validate-runtime-profile | 🆕 kg-validate CLI + structured output | 🔴 P0 |
| PP-OPS-03: Reconciliation drift thụ động | ✅ integrity endpoints + repair endpoints + runbook | 🆕 Prometheus export + scheduled recon + alert webhook | 🔴 P0 |
| PP-OPS-04: Multi-env config không có isolation | ✅ Profile-based config + per-env .env files | 🆕 Env tagging + config linting script | 🟠 P1 |
| PP-OPS-05: Không có correlation ID | ✅ Structured logs + node_id tracking | 🆕 W3C Trace Context + trace lookup API + OTel | 🟠 P1 |
