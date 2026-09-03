# Solutions — VNP Memory Console UI v0

Thư mục này chứa các **giải pháp kỹ thuật** (Solutions) cho từng Change Request, được xây dựng dựa trên kiến trúc hệ thống tại [`specs/architecture.md`](../../architecture.md).

---

## Danh sách Solutions

| Solution | CR liên quan | Nội dung | Status |
|---|---|---|---|
| [SOL-001](./SOL-001-Migration-Strategy.md) | CR-001 → CR-011 | Chiến lược migration tổng thể (pattern, env config, checklist) | ✅ Implemented |
| [SOL-002](./SOL-002-Auth-Solution.md) | CR-001 Auth | JWT RS256 login/logout/refresh, users + refresh_tokens schema, token auto-refresh | ✅ Implemented |
| [SOL-003](./SOL-003-Dashboard-Solution.md) | CR-002 Dashboard | InProcessRegistry health check, Prometheus metrics, NATS throughput, Redis caching | ✅ Implemented |
| [SOL-004](./SOL-004-Sessions-Solution.md) | CR-003 Sessions | sessions PostgreSQL schema, Zep messages via bufconn, vnp-event timeline | ✅ Implemented |
| [SOL-005](./SOL-005-Memory-Solution.md) | CR-004 Memory | vnp-search-hub fan-out, engine routing by ID prefix, sm version chain | ✅ Implemented |
| [SOL-006](./SOL-006-Adaptive-to-Org-Solutions.md) | CR-005 → CR-011 | Supermemory connectors, Memobase profiles, OPA governance, OTEL observability, NATS pipelines, infra DB pings, APIKey/webhook schema | ✅ Implemented |
| [SOL-007](./SOL-007-Gap-Fixes.md) | CR-001 → CR-011 | Service files đầy đủ, hook code hoàn chỉnh, schemas còn thiếu (governance, observability, pipeline, infra, org, sdk) | ✅ Implemented |

---

## Nguyên tắc kiến trúc áp dụng

Tất cả solutions đều bám sát vào các quyết định kiến trúc tại `specs/architecture.md`:

| Nguyên tắc | Chi tiết |
|---|---|
| **gRPC qua bufconn** | Mọi console handler giao tiếp với engine service qua `InProcessRegistry` — zero network hop trong monolith mode (§2.1) |
| **Tenant isolation** | `TenantID` được inject từ `AuthContext` vào mọi gRPC request — không có query nào thiếu tenantID (§3.1) |
| **Redis caching** | Các metrics thay đổi chậm (health, throughput) được cache Redis 15-30s để tránh overload Prometheus (§8) |
| **NATS events** | Async operations (connector sync, GDPR forget, profile flush) trigger qua NATS subjects (§6.4) |
| **camelCase JSON** | Go struct fields dùng `json:"camelCase"` tags để khớp với TypeScript interface trong `ui/src/types/*.ts` |
| **Error fallback** | Fan-out queries (search hub, dashboard metrics) tolerate individual engine failures — partial result thay vì fail toàn bộ |

---

## Luồng dữ liệu tổng quan

```
Browser (React Query)
    │  fetch Bearer JWT + x-tenant-id
    ▼
Gateway :8080 (Go net/http)
    │  Auth middleware → inject TenantID
    │  Console Handlers (/v1/console/*)
    │
    ├─ bufconn gRPC ──→ 35 Engine Services (in-process)
    │                    ├── memobase-context  (profiles, context)
    │                    ├── zep-thread/memory (sessions, working memory)
    │                    ├── sm-memory/connector/analytics (adaptive)
    │                    ├── vnp-search-hub   (cross-engine search)
    │                    ├── vnp-event        (timeline, heatmap)
    │                    ├── vnp-admin        (tenants, API keys)
    │                    └── pipeline-service (jobs, queues)
    │
    ├─ PostgreSQL ────→ sessions, messages, audit_logs, users, api_keys, webhooks
    ├─ Neo4j ─────────→ graph nodes/edges count (dashboard stats)
    ├─ Redis ─────────→ response cache (dashboard), rate limits
    ├─ NATS ──────────→ JetStream metrics (pipeline queues), event publish
    └─ Prometheus ────→ latency/throughput metrics queries
```
