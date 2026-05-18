---
id: TASK-060
title: "[SOL-003 T06] Update Gateway api.md for Console Namespace"
service: gateway
type: TASK
priority: P1
status: Done
created: 2026-05-14
updated: 2026-05-14
linked_specs:
  - ui/specs/solutions/SOL-003-ui-gateway-hardening.md
  - gateway/specs/solutions/SOL-002-ux-console-api-upgrade.md
---

## Mục Tiêu
Cập nhật `gateway/docs/api.md` phản ánh đầy đủ 44+ console endpoints đã registered trong `router.go`.

## Bối Cảnh
SOL-002 đã thêm 44 console endpoints vào `router.go`. `api.md` đã được update +138 lines (T16) nhưng cần verify tính nhất quán sau SOL-003 changes.

## Phạm Vi Công Việc (Scope)
1. **Verify completeness**: So sánh `grep 'HandleFunc' router.go` với sections trong `api.md`
2. **Thêm missing entries**: Document bất kỳ endpoint nào thiếu
3. **Correct HTTP methods**: Ensure `POST /v1/console/memory/search` (không phải GET), `POST /v1/console/graph/subgraph`, etc.
4. **Add UI client notes**: Ghi chú cho developers rằng UI sử dụng `api.config.ts → console.*` paths

## Acceptance Criteria
- [ ] AC-1: Mọi `HandleFunc` trong `router.go` đều có entry tương ứng trong `api.md`
- [ ] AC-2: HTTP method (GET/POST/PUT/DELETE) khớp giữa `router.go` và `api.md`
- [ ] AC-3: Request/Response schema được document cho ít nhất 80% endpoints
- [ ] AC-4: Console namespace có section riêng, tổ chức theo module (dashboard, memory, graph, ...)

## Tài Liệu Tham Chiếu
- `gateway/internal/adapter/handler/router.go` — Source of truth
- `gateway/docs/api.md` — File cần update
- `ui/src/config/api.config.ts` — Consumer path config
