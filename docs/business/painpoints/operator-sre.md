# Pain Points — Operator / SRE

> **Actor**: Operator / SRE (Site Reliability Engineer)  
> **Phạm vi**: Người deploy, validate, và vận hành kg-service — đảm bảo service chạy ổn định, diagnose drift và backend failures, maintain deployment repeatability  
> **Loại**: Operational reliability & deployment pain points  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Tổng quan

Operator/SRE là người **đảm bảo kg-service vận hành trong production**. Họ không viết application code — họ deploy, monitor, và diagnose. Với kg-service, thách thức đặc biệt là kiến trúc multi-backend (PostgreSQL + Redis + Graph backend + Vector backend) dưới nhiều môi trường khác nhau (Compose, Kubernetes, VM).

Mỗi backend thêm vào là một **failure surface** và một **complexity dimension** mà Operator phải manage.

---

## Pain Points chi tiết

### PP-OPS-01 — Backend startup ordering không được orchestrate — service có thể start trước khi dependencies sẵn sàng

**Mức độ**: 🔴 Critical  
**Tần suất**: Mỗi lần deploy hoặc restart  

**Mô tả**:  
SRS nêu risk: "Different backend vendors use different client semantics and startup ordering." Trong thực tế:

```
Docker Compose startup order:
→ postgres: starts (15-30 seconds to ready)
→ redis: starts (2-5 seconds to ready)  
→ graph-backend: starts (30-120 seconds to ready — Neo4j especially slow)
→ vector-backend: starts (10-60 seconds to ready)
→ kg-service: starts → ngay lập tức try connect tất cả backends

Problem: kg-service start ở giây thứ 5 → graph-backend chưa ready (second 30-120)
→ kg-service log: "Failed to connect to graph backend: connection refused"
→ Behavior: 
   Option A: service crash-loop → 503 cho toàn bộ API
   Option B: service start với degraded mode → graph queries fail silently
   Option C: service retry indefinitely → confusing logs
```

**Ví dụ thực tế**:
```bash
# Operator chạy:
docker-compose up

# Logs:
kg-service | 2026-08-03 00:10:05 ERROR: graph backend connection failed
kg-service | 2026-08-03 00:10:05 ERROR: vector backend connection failed  
kg-service | 2026-08-03 00:10:05 INFO: service started (degraded mode)

# Operator: 
# "degraded mode là gì? Graph queries fail ngay hay retry? Vector search available không?"
# Không có rõ ràng documentation về degraded behavior
```

**Hệ quả kinh doanh**:
- False-positive health failures: service "running" nhưng graph/vector không hoạt động
- Operator phải manually verify từng backend sau startup → không scalable
- Production incidents vì startup race condition sau pod restart trong Kubernetes

**Giải pháp cần có**:
- Wait-for-backend mode: `KG_STARTUP_WAIT_BACKENDS=true` → service không start cho đến khi tất cả backends healthy
- Per-backend health check với timeout và retry config
- Startup probe Kubernetes-compatible: chỉ mark Ready khi tất cả backends connected
- Clear degraded mode documentation: "Graph backend down → these endpoints fail, these still work"

---

### PP-OPS-02 — Không có unified validation tool — phải chạy nhiều curl commands để verify deployment

**Mức độ**: 🔴 Critical  
**Tần suất**: Mỗi lần deploy mới (staging, production, rollback)  

**Mô tả**:  
URD task flow "Deploy And Validate" liệt kê: "Run the appropriate validation script." Nhưng validation scripts hiện tại:
- Là collection of curl commands — phải chain output thủ công
- Không có expected vs. actual comparison
- Không có exit code để dùng trong CI/CD
- Nếu một step fail → tiếp tục hay stop? Không rõ

```bash
# Validation script hiện tại (giả định):
curl $KG_HOST/health
# → Operator nhìn output, judge "ok" hay không

curl -X POST $KG_HOST/v1/access/resolve -H "Authorization: Bearer $TEST_KEY"
# → Operator nhìn, check tenant/app fields

curl -X GET $KG_HOST/v1/tenants/test-tenant/ontology/domains
# → ...

# 10-15 bước như vậy, manual observation
# Không có: PASS/FAIL per check
# Không có: tổng kết ở cuối
# Không có: exit code 0/1 → CI/CD không dùng được
```

**Hệ quả kinh doanh**:
- Validation không reliable → drift giữa environments không được detect
- CI/CD không thể auto-validate deployment → manual gate → slow deploy cycle
- "Validation passed" là subjective, không reproducible

**Giải pháp cần có**:
- Validation tool: `kg-validate --profile compose --env .env.staging` → structured output: `✓ health, ✓ postgres, ✗ vector-backend (timeout), ...`
- Exit code: 0 = all pass, 1 = some fail → CI/CD friendly
- JSON output mode: `kg-validate --output json | jq .failures`
- Validation test suite: không chỉ connectivity, còn test write→project→read→search full flow

---

### PP-OPS-03 — Reconciliation drift không có automated detection và alert

**Mức độ**: 🔴 Critical  
**Tần suất**: Ongoing, sau bất kỳ backend restart hoặc data migration  

**Mô tả**:  
SRS mô tả: "FR-5 Projection And Reconciliation — detect stale, missing, or mismatched projections." Nhưng detection là passive (người phải hỏi) chứ không phải proactive:

```
Tình huống phổ biến:
1. Neo4j (graph backend) restart unplanned (OOM killed)
2. kg-service tiếp tục write → PostgreSQL nhận writes
3. Graph backend recover nhưng projection events missed
4. Projection sync bị gap: PostgreSQL có data mới hơn graph backend
5. App integrators thấy stale data qua graph queries

Hiện tại operator biết được vấn đề này khi nào?
→ Khi app integrators complain "tôi write rồi nhưng template không trả về data mới"
→ Operator chạy GET /v1/integrity/report → thấy drift
→ MTTR cao vì phát hiện muộn

Không có:
→ Automated reconciliation job chạy định kỳ
→ Alert khi sync_version lag > threshold
→ Auto-healing: detect drift → trigger re-projection automatically
```

**Hệ quả kinh doanh**:
- Stale data in production → wrong AI agent answers → business decisions based on stale knowledge
- Manual reconciliation trigger → operator phải monitor và intervene
- After backend migration → có thể weeks of drift trước khi detected

**Giải pháp cần có**:
- Scheduled reconciliation: cron-based reconciliation job, configurable interval
- Drift alert: webhook/alert khi `projection_lag_count > N` hoặc `oldest_unsynced_ms > T`
- Auto-reconcile trigger: khi graph/vector backend reconnect → automatically trigger reconciliation
- `GET /v1/integrity/report?format=prometheus` → metrics cho Prometheus/Grafana

---

### PP-OPS-04 — Multi-environment configuration không có clear isolation — dễ point staging config vào production backend

**Mức độ**: 🟠 High  
**Tần suất**: Khi manage multiple environments (dev, staging, prod)  

**Mô tả**:  
SRS "FR-6 Deployment And Validation" và risk: "Real graph/vector backend setup can drift across environments." Trong thực tế:

```
Env vars cần set per environment:
KG_HTTP_PORT, KG_DB_DSN, KG_REDIS_URL, 
KG_GRAPH_BACKEND_URL, KG_VECTOR_BACKEND_URL,
KG_PROFILE, ...

Danger:
→ Staging env file có KG_GRAPH_BACKEND_URL trỏ vào production Neo4j (typo)
→ Developer test trên staging → writes production data
→ Không có service-level guard: "This is production backend. Staging service should not connect here."

Thêm vào:
→ Không có env var validation at startup: service start với invalid URL → fail at runtime
→ Không có environment tagging: service không biết nó đang ở staging hay production
```

**Hệ quả kinh doanh**:
- Data pollution: staging tests contaminate production graph/vector
- Debugging nightmare: "Data này đến từ đâu? Staging hay prod?"
- Không có circuit breaker giữa environments

**Giải pháp cần có**:
- Startup env validation: check tất cả required vars present, URLs parseable, không placeholder values
- Environment tag: `KG_ENVIRONMENT=production|staging|development` → log, metrics, và guard
- Backend environment assertion: graph/vector backend có thể expose environment tag → kg-service check khớp không
- Env file linting: `kg-config validate --env .env.staging` → pre-flight check trước deploy

---

### PP-OPS-05 — Logs không có correlation ID xuyên suốt write → projection → read flow

**Mức độ**: 🟠 High  
**Tần suất**: Khi diagnose incidents — tại sao data bị stale hoặc missing  

**Mô tả**:  
Khi App Integrator báo "tôi write node lúc 14:32 nhưng 15:00 vẫn không thấy qua template", Operator cần trace:

```
Cần trace:
1. Write request có được nhận không? → PostgreSQL write log
2. PostgreSQL record có được pick up bởi projection worker không? → worker log
3. Graph backend có nhận projection event không? → graph connector log
4. Vector backend embedding có được compute không? → vector connector log

Hiện tại:
→ Mỗi step có separate logs
→ Không có correlation ID xuyên suốt
→ Operator phải:
   - Grep write log theo timestamp
   - Find node_id
   - Grep projection log theo node_id
   - Tìm graph log theo node_id (có thể không consistent)
→ 30-60 phút để trace một incident
```

**Hệ quả kinh doanh**:
- MTTR cao: incident investigation slow
- Only senior SREs biết cách correlate logs → knowledge silo
- Production postmortems thiếu evidence vì logs không liên kết được

**Giải pháp cần có**:
- Distributed trace ID (W3C Trace Context) xuyên suốt: write → queue → projection → graph write → vector write
- Structured logging với `trace_id`, `span_id`, `node_id`, `tenant_id` ở mọi log line
- `GET /v1/admin/trace/{trace_id}` — trace viewer cho một write operation từ đầu đến cuối
- OpenTelemetry export: traces → Jaeger/Tempo

---

## Ma trận Pain Points — Operator / SRE

| ID | Pain Point | Mức độ | Impact | Giải pháp cần có |
|:---|:---|:---:|:---|:---|
| PP-OPS-01 | Backend startup race condition | 🔴 | Silent degraded mode, false healthy | Wait-for-backend mode + startup probe |
| PP-OPS-02 | Validation không có PASS/FAIL output | 🔴 | Manual gate, unreproducible | kg-validate tool với exit code |
| PP-OPS-03 | Reconciliation drift không proactive detect | 🔴 | Stale data in prod, high MTTR | Scheduled recon + drift alerts |
| PP-OPS-04 | Multi-env config không có isolation guard | 🟠 | Staging → prod contamination | Env validation + environment tagging |
| PP-OPS-05 | Không có correlation ID xuyên write→projection | 🟠 | Slow incident diagnosis | Distributed tracing + W3C Trace Context |

---

## Tại sao Operator/SRE phải dùng kg-service

1. **Observability built-in**: Service có health, integrity, reconciliation endpoints — không phải tự implement monitoring
2. **Repeatable deployment**: Cùng một service binary chạy trên Compose, K8s, VM — giảm environment-specific configurations
3. **Backend portability**: Swap graph backend (Neo4j → MemGraph → FalkorDB) thông qua runtime profile — không phải redeploy application code
4. **Source-of-truth rebuild**: Nếu graph hoặc vector backend fail hoàn toàn → rebuild từ PostgreSQL — không mất data
5. **Profile-based config**: Runtime profiles tách biệt config complexity — Operator chỉ cần chọn profile, không cần manage từng env var riêng lẻ

> **Kết luận**: Operator/SRE đau nhiều nhất trong ngày đầu deploy và trong production incidents. Giải quyết PP-OPS-01 (startup ordering) và PP-OPS-02 (validation tool) là prerequisite để Operator có thể tự tin run kg-service trong production.
