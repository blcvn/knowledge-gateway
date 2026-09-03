# TC-013: Governance & Audit — Test Cases

**Test Design tham chiếu:** [TD-013](../designs/TD-013-governance-audit.md)  
**Requirements tham chiếu:** [TR-013](../requirements/TR-013-governance-audit.md)  
**Module:** Audit Trail, Retention Sweep, Auth  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-013-001: mem::remember tạo audit record với đầy đủ fields

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-013-AUD-001 |

**Điều kiện tiên quyết:** KV `mem:audit` trống

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| content | `Auth system uses JWT with RS256` |
| type | `architecture` |
| sessionId | `sess_audit` |

**Các bước thực hiện:**
1. Gọi `mem::remember({content, type, sessionId})`
2. Đọc KV `mem:audit`
3. Tìm entry mới nhất

**Kết quả mong đợi:**
- Có ít nhất 1 audit record mới
- `audit.operation = "remember"`
- `audit.memoryId` là string (ID của memory vừa tạo)
- `audit.timestamp` là ISO timestamp hợp lệ
- `audit.sessionId = "sess_audit"`

---

## TC-013-002: mem::forget tạo audit record

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-013-AUD-002 |

**Điều kiện tiên quyết:** Memory `mem_target` tồn tại trong KV

**Các bước thực hiện:**
1. Gọi `mem::forget({memoryId: "mem_target"})`
2. Đọc KV `mem:audit`, tìm entry cho "mem_target"

**Kết quả mong đợi:**
- Audit record có `operation = "forget"`
- `audit.memoryId = "mem_target"`
- `audit.timestamp` là ISO timestamp

---

## TC-013-003: Audit records không bị xóa khi memory bị forget

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-003 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-013-AUD-003 |

**Các bước thực hiện:**
1. `mem::remember(...)` → tạo Memory M và audit record R₁
2. Ghi lại `auditRecordId` của R₁
3. `mem::forget({memoryId: M.id})` → tạo audit record R₂ (forget event)
4. Kiểm tra R₁ vẫn còn trong `mem:audit`
5. Kiểm tra R₂ cũng tồn tại trong `mem:audit`

**Kết quả mong đợi:**
- R₁ vẫn tồn tại (không bị xóa khi memory M bị forget)
- R₂ tồn tại (forget event được ghi nhận)
- `mem:audit` có ít nhất 2 records liên quan đến M

---

## TC-013-004: Audit records được giữ ít nhất 30 ngày

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-013-AUD-004 |

**Điều kiện:** Audit record có `timestamp = 29 ngày trước`

**Kết quả mong đợi:** Sau retention sweep, audit record này vẫn tồn tại (29 < 30 ngày)

---

## TC-013-005: Retention sweep xóa expired memories, giữ active

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-013-AUD-005 |

**Setup:**

| Memory | forgetAfter | Expected sau sweep |
|---|---|---|
| Memory A | `[hôm qua]` (expired) | ❌ Bị xóa |
| Memory B | `[tuần sau]` (active) | ✅ Còn |
| Memory C | *(không set)* | ✅ Còn |

**Các bước thực hiện:**
1. Tạo Memory A, B, C với forgetAfter như trên
2. Gọi `mem::retention-sweep` (hoặc trigger tự động)
3. Kiểm tra mỗi memory trong KV

**Kết quả mong đợi:**
- Memory A: không tồn tại trong `mem:memories`
- Memory B: tồn tại
- Memory C: tồn tại

---

## TC-013-006: API key hợp lệ → HTTP 200

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-013-AUD-006 |

**Điều kiện tiên quyết:** `AGENTMEMORY_SECRET = "valid-secret-key-16chars"` được set

**HTTP Request:**

| | Giá trị |
|---|---|
| Method | `GET` |
| Path | `/status` |
| Header | `Authorization: Bearer valid-secret-key-16chars` |

**Kết quả mong đợi:** HTTP 200

---

## TC-013-007: API key sai → HTTP 401

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-013-AUD-006 |

**HTTP Request:**

| | Giá trị |
|---|---|
| Header | `Authorization: Bearer wrong-key` |

**Kết quả mong đợi:**
- HTTP 401
- Body: `{error: "unauthorized"}` hoặc tương đương

---

## TC-013-008: Không có AGENTMEMORY_SECRET → local mode (không cần auth)

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-008 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-013-AUD-007 |

**Điều kiện tiên quyết:** `AGENTMEMORY_SECRET` không được set (undefined)

**HTTP Request:** `GET /status` — không có Authorization header

**Kết quả mong đợi:** HTTP 200 (local mode, không yêu cầu auth)

---

## TC-013-009: Recall operations được ghi vào audit

| Trường | Giá trị |
|---|---|
| **ID** | TC-013-009 |
| **Loại** | Integration |
| **Ưu tiên** | 🟡 P2 |
| **Requirement** | TR-013-AUD-008 |

**Các bước thực hiện:**
1. Gọi `mem::recall({sessionId: "sess_audit", query: "auth"})`
2. Đọc `mem:audit`

**Kết quả mong đợi:** Có audit record với `operation = "recall"`, `sessionId = "sess_audit"`

---

## Tổng kết TC-013

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-013-001 | remember → audit record | 🔴 P0 | Integration |
| TC-013-002 | forget → audit record | 🔴 P0 | Integration |
| TC-013-003 | Audit immutable after forget | 🟠 P1 | Integration |
| TC-013-004 | Audit retained 30 days | 🟠 P1 | Integration |
| TC-013-005 | Retention sweep | 🟠 P1 | Integration |
| TC-013-006 | Valid API key → 200 | 🔴 P0 | Integration |
| TC-013-007 | Wrong API key → 401 | 🔴 P0 | Integration |
| TC-013-008 | No secret → local mode | 🟠 P1 | Integration |
| TC-013-009 | Recall → audit record | 🟡 P2 | Integration |
