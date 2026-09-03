# P4 — Enterprise Architect

> **Vai trò:** Governance, compliance, security architecture cho hệ thống AI của tổ chức.
> **Kỹ năng:** Security policies, ABAC, audit frameworks, GDPR/compliance.
> **Tần suất sử dụng VNP Memory:** Hàng tháng (audit cycles).

---

## Pain Points

### PP-P4-01 — Không kiểm soát được AI đang nhớ gì, từ đâu, ai tạo

**Mô tả:**
"AI của chúng ta đang giữ thông tin gì về khách hàng?" — câu hỏi này không có câu trả lời rõ ràng. Memory được lưu ở 6 engines khác nhau, không có unified audit trail, không có "xem tất cả memory về user X".

**Hậu quả thực tế:**
- Không pass được GDPR audit
- Không trả lời được Data Subject Access Request (DSAR) trong 30 ngày theo luật
- Risk: data leak không phát hiện được

**Features giải quyết:**
- [F22] Governance Center:
  - Audit Trail: `GET /v1/console/governance/audit` — searchable theo actor, action, entity, tenant, engine
  - GDPR Forget: `POST /v1/console/governance/gdpr/forget` — cascading deletion across 6 engines
  - GDPR Preview: dry-run mode — xem impact trước khi xóa
- [F16] Memory Explorer: `GET /v1/console/memory/*` — inspect, neighbors, versions của từng memory
- [F14] API Key Lifecycle: biết ai đang truy cập gì

---

### PP-P4-02 — GDPR Right to be Forgotten không thể thực thi

**Mô tả:**
Khi user yêu cầu xóa data, team phải manually xóa data từ 6 engines khác nhau. Dễ bỏ sót engine nào đó. Không có bằng chứng (audit log) rằng việc xóa đã hoàn thành.

**Hậu quả thực tế:**
- GDPR violation fine: up to 4% annual turnover
- Legal liability
- Trust damage

**Features giải quyết:**
- [F22] GDPR Forget: 1 API call → cascading delete across ALL 6 engines
- Preview trước: xem chính xác data nào sẽ bị xóa
- Audit log: evidence rằng deletion đã xảy ra (timestamp, actor, scope)

---

### PP-P4-03 — Không có policy enforcement cho AI memory access

**Mô tả:**
"AI không được phép đọc memory của user khác." — Nghe đơn giản nhưng không có cơ chế technical enforce điều này ngoài "hy vọng developer viết code đúng".

**Features giải quyết:**
- [F14] TenantID injection: không thể query cross-tenant
- [F22] OPA Policies: `POST/PUT /v1/console/governance/policies` — define access rules per entity type
- [F14] User roles per tenant: admin/editor/viewer — role-based access

---

### PP-P4-04 — Không biết AI nhớ gì đang ảnh hưởng đến quyết định

**Mô tả:**
"Tại sao AI đưa ra recommendation X?" — Không trace được context nào đã được inject vào LLM prompt, từ engine nào, có accuracy không.

**Features giải quyết:**
- [F08] Agent Observe: capture `memory_read` events — biết AI đọc memory gì trước khi quyết định
- [F20] Agent Context Debugger: trace context assembly
- [F26] Session Replay: replay toàn bộ decision process

---

## Summary

| Pain | Giải pháp |
|---|---|
| Không biết AI nhớ gì | Memory Explorer + Audit Trail |
| GDPR forget không thể thực thi | Cascading GDPR Forget + audit evidence |
| Thiếu policy enforcement | OPA Policies + TenantID isolation |
| Không trace được AI decision | Agent Observe + Context Debugger |
