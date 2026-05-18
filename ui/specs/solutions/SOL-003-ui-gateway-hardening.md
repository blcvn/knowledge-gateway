---
id: SOL-003
title: UI/Gateway Hardening — Service Alignment, Type Safety & Resilience
service: cross-service
version: 1.1.0
status: Approved
priority: P0
created: 2026-05-14
updated: 2026-05-14
linked_cr: ui-gateway-audit-report (2026-05-14)
approved_by: Architecture Team
---

## Yêu Cầu Gốc

Audit report (2026-05-14) phát hiện **17 issues** (5 Critical, 6 Medium, 6 Low) trong UI/Gateway layer:

1. **C1–C4 (Critical)**: 10 UI service files gọi sai API path — dùng `/v1/admin/*` hoặc engine-direct paths thay vì `/v1/console/*` của Gateway → mọi API call đều trả 404 khi `VITE_USE_MOCK_DATA=false`.
2. **C2–C3**: `usePolicies()` trả `Promise.resolve([])`, `useProfileList()` gọi `getEvents('all')` — hợp đồng (contract) giữa hook và service bị gãy.
3. **M1–M6**: Thiếu **22 service methods** so với Gateway router; mock data dùng `any` thay vì typed.
4. **L1–L6**: Dead code (`Placeholder.tsx`), thiếu `ErrorBoundary`, `cognee.service.ts` / `zep.service.ts` không có hook tương ứng.

**Phạm vi mở rộng**: Audit cũng chỉ ra Gateway cần update `api.md` & `changelog.md`, và downstream services (`vnp-admin`, `vnp-platform`, `vnp-event`, `vnp-search-hub`) cần expose actual handlers để các Gateway console proxy calls có thể hoạt động thực tế.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---|---|---|
| **ui** | Service path alignment, type safety, ErrorBoundary, dead code cleanup | Cao |
| **vnp-gateway** | Cập nhật api.md, changelog.md, verify router parity | Trung bình |
| **vnp-admin** | Expose audit log query + OPA policy CRUD handlers | Cao |
| **vnp-platform** | Expose pipeline status + infra health probe handlers | Cao |
| **vnp-search-hub** | Expose unified cross-engine search handler | Trung bình |
| **vnp-event** | Expose GDPR cascading forget + timeline handlers | Trung bình |

### Breaking Changes
- [x] API response format thay đổi? — Không (chỉ align paths, không đổi format)
- [ ] Database schema migration cần thiết? — Không (đã migrate ở SOL-002 T08–T09)
- [x] Consumer downstream cần cập nhật? — Có (UI services phải dùng `/v1/console/*`)

### Ràng Buộc Kiến Trúc
- Mọi UI service PHẢI gọi qua `/v1/console/*` — KHÔNG gọi trực tiếp engine API
- Gateway router.go (`handler/router.go`) là source of truth cho endpoints
- Mock data phải type-safe (`satisfies` thay `as`), không dùng `any`
- Hooks pattern: `useQuery` + `useMutation` via `@tanstack/react-query`

## Giải Pháp Đề Xuất

### Approach

**Phase 1 — UI Hardening (T01–T05)**: Sửa toàn bộ UI layer — config, services, hooks, types, mocks, error boundary. Đã thực hiện bước đầu trong audit resolution nhưng cần formalize thành tasks có AC để verify.

**Phase 2 — Gateway Documentation (T06–T07)**: Cập nhật `api.md` và `changelog.md` phản ánh console namespace + SOL-003 changes.

**Phase 3 — Downstream Service Handlers (T08–T11)**: Ensure các downstream services expose real HTTP handlers cho Gateway gRPC clients đã được wired ở SOL-002 T14.

### Alternatives Đã Xem Xét

| Alternative | Lý do loại bỏ |
|---|---|
| Proxy trực tiếp từ UI → engine services | Vỡ kiến trúc gateway, không có auth/audit layer |
| Giữ `/v1/admin/*` paths | Xung đột namespace khi admin namespace đã có tenant/key mgmt |

### Trade-offs
- **Ưu điểm**: 100% type-safe, single source of truth cho paths, resilient error handling
- **Nhược điểm**: Downstream services chưa có real handlers → vẫn cần mock mode cho dev

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)

```
Phase 1 — UI (song song)
  T01: API config + service alignment    ← Không phụ thuộc
  T02: Hook fixes                        ← Sau T01
  T03: Type safety + mock cleanup        ← Song song T02
  T04: ErrorBoundary + dead code         ← Song song T02
  T05: Build verification + smoke test   ← Sau T01–T04

Phase 2 — Gateway docs (sau Phase 1)
  T06: Gateway api.md update             ← Sau T05
  T07: Gateway changelog.md update       ← Sau T06

Phase 3 — Downstream (song song với Phase 2)
  T08: vnp-admin audit + policy handlers ← Không phụ thuộc
  T09: vnp-platform pipeline + infra     ← Không phụ thuộc
  T10: vnp-search-hub unified search     ← Không phụ thuộc
  T11: vnp-event GDPR + timeline         ← Không phụ thuộc
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | Align API config + 10 service paths | TASK | ui | - | 2h |
| T02 | Fix hook contracts (profiles, governance, adaptive) | TASK | ui | T01 | 1h |
| T03 | Type safety: extend types + cleanup mocks | TASK | ui | - | 1h |
| T04 | ErrorBoundary + remove dead code | TASK | ui | - | 1h |
| T05 | Build verification + integration smoke test | QA | ui | T01–T04 | 1h |
| T06 | Update gateway api.md for console namespace | TASK | gateway | T05 | 2h |
| T07 | Update gateway changelog.md | TASK | gateway | T06 | 0.5h |
| T08 | vnp-admin: audit log query + policy CRUD handlers | FEAT | vnp-admin | - | 4h |
| T09 | vnp-platform: pipeline status + infra probe handlers | FEAT | vnp-platform | - | 4h |
| T10 | vnp-search-hub: unified search orchestrator handler | FEAT | vnp-search-hub | - | 3h |
| T11 | vnp-event: GDPR cascade forget + event timeline | FEAT | vnp-event | - | 3h |

### Rollback Plan
- UI: Revert `api.config.ts` to use legacy paths, set `VITE_USE_MOCK_DATA=true`
- Gateway: api.md/changelog.md are documentation-only, no code rollback needed
- Downstream: Each service has independent rollback via git revert

## Acceptance Criteria (Solution Level)
- [ ] SOL-AC-1: `vite build` passes cleanly with 0 TypeScript errors
- [ ] SOL-AC-2: All 10 UI services call `/v1/console/*` paths (verified by grep)
- [ ] SOL-AC-3: No `any` type in service layer (verified by grep)
- [ ] SOL-AC-4: ErrorBoundary wraps all lazy-loaded modules
- [ ] SOL-AC-5: Gateway `api.md` documents all 44+ console endpoints
- [ ] SOL-AC-6: All 4 downstream services expose HTTP handlers matching Gateway gRPC client expectations

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify |
|---|---|---|---|---|
| T01 | API config + service alignment | ✅ Done | AI | 10/10 services aligned |
| T02 | Hook contract fixes | ✅ Done | AI | 3/3 hooks fixed |
| T03 | Type safety + mock cleanup | ✅ Done | AI | 0 `any` in services |
| T04 | ErrorBoundary + dead code | ✅ Done | AI | Placeholder deleted, EB added |
| T05 | Build verification | ✅ Done | AI | vite build 2.15s, 22 chunks |
| T06 | Gateway api.md update | ✅ Done | AI | v2.0.0 Active, SOL-003 cross-ref |
| T07 | Gateway changelog.md | ✅ Done | AI | v0.7.0 SOL-003 entry added |
| T08 | vnp-admin handlers | ✅ Done | AI | 2 models, 2 usecases, 2 test files |
| T09 | vnp-platform handlers | ✅ Done | AI | Console handler + topology cache |
| T10 | vnp-search-hub handler | ✅ Done | AI | Orchestrator + 4 test cases |
| T11 | vnp-event handlers | ✅ Done | AI | GDPR cascade + 4 test cases |

**Progress: 11/11 (100%) — ALL PHASES COMPLETE ✅**
