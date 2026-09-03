# P2 — Platform / DevOps Engineer

> **Vai trò:** Triển khai, vận hành và scale VNP Memory infrastructure cho production.
> **Kỹ năng:** Docker, Kubernetes, monitoring, CI/CD, security hardening.
> **Tần suất sử dụng VNP Memory:** Hàng tuần.

---

## Bức tranh công việc hàng ngày

Platform Engineer nhận yêu cầu: "Deploy memory system cho 50 AI teams, mỗi team là 1 tenant riêng, cần SLA 99.9%, phải monitor được, phải scale được."

Họ nhìn vào danh sách dependencies: PostgreSQL + Neo4j + Redis + Qdrant + MinIO + NATS + 35 microservices... và thở dài.

---

## Pain Points

### PP-P2-01 — Vận hành 35+ microservices độc lập là ác mộng

**Mô tả:**
Hệ thống AI memory truyền thống yêu cầu deploy riêng lẻ từng component. Với 6 memory engines, mỗi engine 3-9 services, cộng platform services = 35+ processes cần quản lý. Monitoring, logging, health check đều phải tự configure riêng.

**Hậu quả thực tế:**
- Onboarding engineer mới: 2-3 tuần chỉ để hiểu topology
- Service discovery complexity: mỗi service cần biết endpoint của 10 services khác
- Cascading failures: 1 service crash → không biết nó ảnh hưởng service nào

**Features giải quyết:**
- [F01] Monolith Mode: 35+ services trong 1 process (bufconn), 1 binary, 1 `make dev`
- InProcessRegistry: services communicate qua in-process gRPC — zero network hop
- `GET /healthz` (port :8083): aggregated health cho tất cả services

---

### PP-P2-02 — Monitoring và observability fragmented

**Mô tả:**
Mỗi engine có logging format riêng. Metrics không standardized. Khi có sự cố, engineer phải SSH vào từng container, grep log từng service, tự correlate.

**Hậu quả thực tế:**
- Mean Time to Detection (MTTD): 15-30 phút
- Mean Time to Resolution (MTTR): 2-4 giờ
- Không có unified dashboard — phải dùng 5 tab browser cùng lúc

**Features giải quyết:**
- [F25] Observability & Tracing:
  - OpenTelemetry distributed tracing across all engines
  - Prometheus metrics: latency, throughput, error rates, LLM costs
  - Structured JSON logging (slog) với secret redaction
  - `GET /v1/console/observability/{metrics|traces|errors|costs}`
- [F24] Infrastructure Health: service topology, database health, resource usage
- [F15] Console Dashboard: real-time metrics, memory heatmap, throughput

---

### PP-P2-03 — Multi-tenant isolation khó đảm bảo

**Mô tả:**
Khi chạy nhiều tenant trên cùng infrastructure, phải đảm bảo tenant A không truy cập được data của tenant B. Với 6 engines riêng lẻ, mỗi engine có isolation mechanism khác nhau — rất dễ có gap.

**Hậu quả thực tế:**
- Security audit fail → mất khách hàng enterprise
- Data leak → incident + fines (GDPR)
- Phải review code isolation logic của 6 engines khác nhau

**Features giải quyết:**
- [F14] Authentication & Multi-tenancy:
  - TenantID injected vào MỌI domain entity
  - MỌI query có `WHERE tenant_id = :current` — không có global queries
  - Integration tests verify: cross-tenant queries return 0 results
  - Rate limiting per tenant (Redis sliding window)

---

### PP-P2-04 — Không có single entry point cho API management

**Mô tả:**
Trong hệ thống truyền thống, mỗi engine expose API riêng. Client phải biết endpoint của từng engine. API key management phân tán. Khó audit ai đang call gì.

**Features giải quyết:**
- [F01] Gateway: single entry point REST :8080, MCP :8082
- [F14] API Key Lifecycle: Create → Active → Revoked/Expired, SHA-256 hash, prefix identification
- [F27] Organization & SDK Manager: quản lý API keys, engine aliases, quota per tenant

---

### PP-P2-05 — Pipeline failures im lặng — không biết khi nào bị

**Mô tả:**
Background pipelines (knowledge graph construction, profile extraction, consolidation) thất bại không có alert. Developer chỉ biết khi user complain "AI không nhớ gì cả".

**Features giải quyết:**
- [F23] Pipeline Monitor: job status, queues, workers, failed jobs
- [F28] WebSocket Real-time Events: push alerts khi pipeline fails
- Circuit breaker events qua NATS: `gateway.circuit.opened`
- NATS event `consolidation.failed` → alert system

---

## Summary

| Pain | Giải pháp |
|---|---|
| 35+ services phức tạp | Monolith mode — 1 binary, `make dev` |
| Monitoring fragmented | OpenTelemetry + Prometheus + unified console |
| Tenant isolation gap | TenantID trên mọi query |
| API management phân tán | Gateway duy nhất + API key lifecycle |
| Pipeline failures im lặng | Pipeline Monitor + WebSocket alerts |
