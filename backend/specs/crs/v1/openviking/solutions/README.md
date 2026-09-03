# Solutions — OpenViking Feature Parity

**Project:** VNP Memory  
**Domain:** OpenViking — Context Database cho AI Agents (ByteDance/Volcengine Filesystem Paradigm)  
**Path:** `specs/crs/v1/openviking/solutions/`  
**Date:** 2026-06-17  
**Status:** Draft

> Các tài liệu Solution này mô tả giải pháp kỹ thuật chi tiết để đáp ứng từng Change Request trong `specs/crs/v1/openviking/`. Mỗi solution bao gồm: chiến lược triển khai, tích hợp với kiến trúc VNP Memory, design decisions, và kế hoạch thực thi.

---

## Innovation Core

**OpenViking** là một "Context Database" cho AI Agents, phân biệt với RAG thông thường ở **Viking URI paradigm** — tổ chức memories, resources, skills, sessions dưới một cây thư mục `viking://` thống nhất. Kết quả benchmark: **+49% task completion**, **83-91% token cost reduction** vs vanilla RAG.

---

## Danh sách Solutions

| Solution File | CR Tương ứng | Mô tả | Wave |
|---|---|---|---|
| [SOL-OV-001](./SOL-OV-001-Gateway-Service.md) | CR-OV-001 | Unified Gateway — REST 17 routes, MCP 9 tools, WebDAV, 3-mode Auth, RBAC, Rate Limit | Wave 6 |
| [SOL-OV-002](./SOL-OV-002-Filesystem-Service.md) | CR-OV-002 | Filesystem Service — VikingFS Go-native, L0/L1/L2 tiered, grep/glob, PathLock, transparent encryption | Wave 3 |
| [SOL-OV-003](./SOL-OV-003-Search-Service.md) | CR-OV-003 | Search Service — HierarchicalRetriever 6-step, score propagation α=0.7, hotness blending, convergence detection | Wave 4 |
| [SOL-OV-004](./SOL-OV-004-Session-Service.md) | CR-OV-004 | Session Service — Two-Phase Commit, Working Memory v2 (7 sections), 8 memory categories, redo log | Wave 5 |
| [SOL-OV-005](./SOL-OV-005-Resource-Service.md) | CR-OV-005 | Resource Service — Ingestion pipeline (git/HTTP/local/doc), parser registry (50+ ext), L0/L1 VLM, watch | Wave 5 |
| [SOL-OV-006](./SOL-OV-006-Crypto-Admin-Services.md) | CR-OV-006 | Crypto (OVE1 AES-256-GCM, KMS) + Admin (account/user/key CRUD, health aggregation) | Wave 2 |
| [SOL-OV-007](./SOL-OV-007-Shared-Infrastructure.md) | CR-OV-007 | Shared `pkg/` — viking types, VikingFS engine, adapter interfaces, middleware, resilience, OTel, CLI | Wave 1 |

---

## Kiến trúc Tổng quan

### NATS Event Bus (Async Backbone)

```
ov.content.written   → Search: embed + upsert vector index
ov.content.deleted   → Search: remove from vector index
ov.session.committed → Search: update hotness (used_uris)
ov.session.memory.extracted → FS: write memory files
ov.resource.ingested → Search: collection warm-up
ov.crypto.key.rotated → FS: re-wrap file headers
admin.account.created → FS + Search + Crypto: init account
admin.account.deleted → FS + Search + Session: cascade delete
```

### Implementation Wave Order

```
Wave 1 (Foundation): SOL-OV-007 — pkg/ shared types, VikingFS engine, adapters
Wave 2 (Security):   SOL-OV-006 — Crypto OVE1 + Admin multi-tenant
Wave 3 (Storage):    SOL-OV-002 — Filesystem service (VikingFS, tiered, PathLock)
Wave 4 (Search):     SOL-OV-003 — Search service (HierarchicalRetriever, vectors)
Wave 5 (Context):    SOL-OV-004 + SOL-OV-005 — Session (WM v2) + Resource (ingestion)
Wave 6 (Gateway):    SOL-OV-001 — Unified gateway (REST, MCP, WebDAV)
```

### Performance Targets

| Metric | Target |
|--------|--------|
| API response (p50, filesystem ops) | < 100ms |
| Semantic search latency (with rerank) | < 500ms |
| Session commit Phase 1 | < 1s |
| Concurrent sessions per instance | ≥ 1,000 |
| Token cost reduction vs RAG | ≥ 80% |

---

## Nguyên tắc chung cho mọi Solution

1. **Clean Architecture** — Domain → Usecase → Adapter → Infra (không có dependency ngược)
2. **gRPC Internal** — service-to-service communication qua gRPC trên fixed ports (9011-9030)
3. **NATS JetStream** — async events với WorkQueue retention; không có HTTP callbacks
4. **Viking URI** — mọi filesystem resource đều được định danh qua `viking://` namespace
5. **Fail-Fast** — startup validation cho crypto key, vectordb connection, embedding dimension
6. **Tích hợp monolith** — đăng ký vào `InProcessRegistry` trong `apps/memory/internal/bootstrap/`
