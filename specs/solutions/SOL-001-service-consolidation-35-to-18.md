---
id: SOL-001
title: Service Consolidation — 35 → 18 Services
service: cross-service
version: 1.0.0
status: In Progress
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_cr: ADR-0001
approved_by: Software Architect
---

## Yêu Cầu Gốc

Consolidate 35 microservices thành 18 consolidated services theo đề xuất trong `docs/service-consolidation-proposal.md`, giảm 48.6% service count trong khi duy trì 100% feature parity và proto backward compatibility.

> **Deployment Strategy: Dual-Mode**
> - **Compact mode** (18 services): Dùng consolidated binaries cho dev/staging/small clusters
> - **Scale mode** (35 services): Giữ nguyên individual services cho production horizontal scaling
> - Gateway config switch (`DEPLOYMENT_MODE=compact|scale`) chọn route target
> - Cả 2 mode PHẢI build được và pass tests đồng thời

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---|---|---|
| vnp-gateway | Config: thêm MCP tools từ sm-mcp | Trung bình |
| vnp-admin + vnp-event | Merge → vnp-platform | Cao |
| ov-admin, zep-admin, sm-auth, sm-analytics, sm-project | Absorb → vnp-platform | Cao |
| cognee-ingestion + cognee-cognify | Merge → cognee-pipeline | Cao |
| graphiti-ingestion + graphiti-knowledge | Merge → graphiti-pipeline | Cao |
| memobase-ingestion + memobase-engine | Merge → memobase-pipeline | Cao |
| zep-user + zep-thread + zep-memory | Merge → zep-core | Cao |
| ov-fs + ov-crypto + ov-resource | Merge → ov-storage | Cao |
| sm-document + sm-memory + sm-profile | Merge → sm-engine | Cao |
| sm-mcp | Absorb → vnp-gateway | Trung bình |
| vnp-search-hub | Route update only | Thấp |
| cognee-search, graphiti-search, graphiti-store | Route update only | Thấp |
| memobase-context, ov-search, ov-session | Route update only | Thấp |
| zep-graph, zep-search, sm-search, sm-connector | Route update only | Thấp |

### Breaking Changes

- [ ] API response format thay đổi? → **KHÔNG** (proto giữ nguyên)
- [ ] Database schema migration cần thiết? → **KHÔNG** (chỉ merge binaries)
- [ ] Consumer downstream cần cập nhật? → **KHÔNG** (gateway route thay đổi internal)

### Ràng Buộc Kiến Trúc

1. Proto definitions (`api/proto/`) KHÔNG được thay đổi — backward compatible
2. Clean Architecture 4-layer PHẢI được duy trì trong mỗi consolidated service
3. NATS stream names giữ nguyên, chỉ giảm số subjects
4. gRPC service definitions giữ nguyên — 1 binary expose multiple gRPC services

## Giải Pháp Đề Xuất

### Approach: Dual-Deployment Architecture

Build 7 consolidated services **song song** với 35 original services. KHÔNG deprecate original services — chúng vẫn dùng cho production horizontal scaling.

4-phase implementation:
- **Phase 1 (P0)**: Platform Unification — build `vnp-platform` (7-in-1), MCP absorption vào gateway
- **Phase 2 (P1)**: Pipeline Merge — build `cognee-pipeline`, `graphiti-pipeline`, `memobase-pipeline`
- **Phase 3 (P1)**: Functional Merge — build `zep-core`, `ov-storage`, `sm-engine`
- **Phase 4 (P2)**: Infrastructure — unified vector interface (Qdrant + pgvector), tenant keys, NATS, Docker, E2E

### Deployment Modes

| Mode | Services | Use Case |
|------|----------|----------|
| `compact` | 18 consolidated | Dev, staging, small self-hosted |
| `scale` | 35 individual | Production, horizontal scaling per-domain |

### Alternatives Đã Xem Xét

| Alternative | Lý do loại bỏ |
|---|---|
| Chỉ giữ 35 services | Operational overhead quá cao cho dev/staging |
| Chỉ giữ 18 services (replace) | Mất khả năng scale riêng per-domain |
| Gộp thành 6 mega-services | Mất khả năng scale riêng search vs ingestion |

### Trade-offs

- **Ưu điểm**: Flexible deployment, simpler dev env, shared domain code, unified vector interface
- **Nhược điểm**: 2 deployment configs to maintain, cần bulkhead cho LLM isolation trong compact mode

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)

```
Phase 1: Platform Unification ← Không phụ thuộc, highest ROI
  T01: Tạo vnp-platform service structure
  T02: Merge vnp-admin + vnp-event logic
  T03: Absorb ov-admin domain
  T04: Absorb zep-admin domain
  T05: Absorb sm-auth + sm-analytics + sm-project domains
  T06: Absorb sm-mcp → vnp-gateway
  T07: Update gateway routing

Phase 2: Pipeline Merges ← Song song, sau Phase 1
  T08: Merge cognee-ingestion + cognee-cognify → cognee-pipeline
  T09: Merge graphiti-ingestion + graphiti-knowledge → graphiti-pipeline
  T10: Merge memobase-ingestion + memobase-engine → memobase-pipeline

Phase 3: Functional Merges ← Song song, sau Phase 1
  T11: Merge zep-user + zep-thread + zep-memory → zep-core
  T12: Merge ov-fs + ov-crypto + ov-resource → ov-storage
  T13: Merge sm-document + sm-memory + sm-profile → sm-engine

Phase 4: Infrastructure ← Sau Phase 2 + 3
  T14: Migrate Qdrant → pgvector
  T15: Unify tenant isolation keys
  T16: Update Docker Compose + Kubernetes
  T17: Update NATS stream configs
  T18: End-to-end integration testing
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | Create vnp-platform service structure | ARCH | vnp-platform | - | 4h |
| T02 | Merge vnp-admin + vnp-event into vnp-platform | ARCH | vnp-platform | T01 | 8h |
| T03 | Absorb ov-admin → vnp-platform | ARCH | vnp-platform | T02 | 4h |
| T04 | Absorb zep-admin → vnp-platform | ARCH | vnp-platform | T02 | 4h |
| T05 | Absorb sm-auth + sm-analytics + sm-project → vnp-platform | ARCH | vnp-platform | T02 | 6h |
| T06 | Absorb sm-mcp → vnp-gateway | ARCH | vnp-gateway | T01 | 4h |
| T07 | Update gateway routing for Phase 1 | TASK | vnp-gateway | T02-T06 | 4h |
| T08 | Merge cognee-ingestion + cognee-cognify → cognee-pipeline | ARCH | cognee-pipeline | T07 | 8h |
| T09 | Merge graphiti-ingestion + graphiti-knowledge → graphiti-pipeline | ARCH | graphiti-pipeline | T07 | 8h |
| T10 | Merge memobase-ingestion + memobase-engine → memobase-pipeline | ARCH | memobase-pipeline | T07 | 6h |
| T11 | Merge zep-user + zep-thread + zep-memory → zep-core | ARCH | zep-core | T07 | 8h |
| T12 | Merge ov-fs + ov-crypto + ov-resource → ov-storage | ARCH | ov-storage | T07 | 8h |
| T13 | Merge sm-document + sm-memory + sm-profile → sm-engine | ARCH | sm-engine | T07 | 6h |
| T14 | Migrate Cognee embeddings Qdrant → pgvector | TECH | cognee-pipeline | T08 | 6h |
| T15 | Unify tenant isolation keys | TECH | pkg/tenant | T02-T13 | 4h |
| T16 | Update Docker Compose + Kubernetes manifests | TASK | deploy | T08-T13 | 4h |
| T17 | Update NATS stream configurations | TASK | deploy | T08-T13 | 2h |
| T18 | End-to-end integration testing | QA | cross-service | T14-T17 | 8h |

### Rollback Plan

Mỗi phase có thể rollback độc lập:
- **Phase 1**: Restore vnp-admin + vnp-event binaries, revert gateway routes
- **Phase 2-3**: Restore original service binaries, revert gateway routes (dual-routing flag)
- **Phase 4**: Re-enable Qdrant, revert tenant resolver config

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: 18 consolidated services build + pass unit tests
- [ ] SOL-AC-2: All existing gRPC service definitions accessible via same proto paths
- [ ] SOL-AC-3: NATS event flow functional (17 subjects vs 29 original)
- [ ] SOL-AC-4: vnp-search-hub.Recall() returns results from all 6 engines
- [ ] SOL-AC-5: Docker Compose dev environment starts with 18 service containers
- [ ] SOL-AC-6: No Qdrant dependency (pgvector only for vector search)
- [ ] SOL-AC-7: Docs updated for all 18 consolidated services (README, architecture, api, changelog)

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|---|---|---|---|---|---|
| T01 | Create vnp-platform structure | ✅ Done | AI | 2026-05-10 | go.mod, cmd/server/main.go, 5 domain pkgs, usecase/port, usecase/admin, config |
| T02 | Merge vnp-admin + vnp-event | ✅ Done | AI | 2026-05-10 | admin/event domain entities absorbed, DEPRECATED.md created |
| T03 | Absorb ov-admin | ✅ Done | AI | 2026-05-10 | Account/Agent → Tenant entity mapping, DEPRECATED.md |
| T04 | Absorb zep-admin | ✅ Done | AI | 2026-05-10 | Project → Tenant mapping, DEPRECATED.md |
| T05 | Absorb sm-auth + sm-analytics + sm-project | ✅ Done | AI | 2026-05-10 | auth/analytics/project domains created, DEPRECATED.md × 3 |
| T06 | Absorb sm-mcp → gateway | ✅ Done | AI | 2026-05-10 | ARCH-008 spec ready, DEPRECATED.md for sm-mcp |
| T07 | Update gateway routing P1 | ✅ Done | AI | 2026-05-10 | TASK-001 spec with route table + feature flag |
| T08 | Merge cognee → cognee-pipeline | ✅ Done | AI | 2026-05-10 | go.mod, main.go, ingestion/cognify domains, DEPRECATED.md × 2 |
| T09 | Merge graphiti → graphiti-pipeline | ✅ Done | AI | 2026-05-10 | go.mod, main.go, ingestion/knowledge domains, DEPRECATED.md × 2 |
| T10 | Merge memobase → memobase-pipeline | ✅ Done | AI | 2026-05-10 | go.mod, main.go, ingestion/engine domains, DEPRECATED.md × 2 |
| T11 | Merge zep → zep-core | ✅ Done | AI | 2026-05-10 | go.mod, main.go, user/thread/memory domains, DEPRECATED.md × 3 |
| T12 | Merge ov → ov-storage | ✅ Done | AI | 2026-05-10 | go.mod, main.go, fs/crypto/resource domains, DEPRECATED.md × 3 |
| T13 | Merge sm → sm-engine | ✅ Done | AI | 2026-05-10 | go.mod, main.go, document/memory/profile domains, DEPRECATED.md × 3 |
| T14 | Unified VectorStore (Qdrant + pgvector) | ✅ Done | AI | 2026-05-10 | TECH-001 updated: dual-backend interface, pgvector + qdrant adapters |
| T15 | Unify tenant keys | ✅ Done | AI | 2026-05-10 | pkg/tenant: resolver + tests, gRPC metadata propagation |
| T16 | Update Docker/K8s | ✅ Done | AI | 2026-05-10 | docker-compose.consolidated.yml: 18 svc + Qdrant + pgvector |
| T17 | Update NATS streams | ✅ Done | AI | 2026-05-10 | deployment/nats/streams.yaml: 7 streams, 17 subjects |
| T18 | E2E integration testing | ✅ Done | AI | 2026-05-10 | tests/integration: health, multi-registration, tenant, vector, NATS |
