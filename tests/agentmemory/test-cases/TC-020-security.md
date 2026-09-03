# TC-020: Security — Test Cases

**Test Design tham chiếu:** [TD-020](../designs/TD-020-security.md)  
**Requirements tham chiếu:** [TR-020](../requirements/TR-020-security.md)  
**Module:** Auth, Privacy Redaction, Data Isolation  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-020-001: Valid Bearer token → HTTP 200

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-020-SEC-001 |

**Điều kiện tiên quyết:** `AGENTMEMORY_SECRET = "valid-secret-key-minimum16"` được set

**HTTP Request:**

| | Giá trị |
|---|---|
| Method | `GET` |
| Path | `/status` |
| Header | `Authorization: Bearer valid-secret-key-minimum16` |

**Kết quả mong đợi:** HTTP 200

---

## TC-020-002: Wrong token → HTTP 401

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-020-SEC-001 |

**HTTP Request:**

| | Giá trị |
|---|---|
| Header | `Authorization: Bearer totally-wrong-key` |

**Kết quả mong đợi:**
- HTTP 401
- Body: `{error: "unauthorized"}` hoặc tương đương
- Không expose thông tin về secret

---

## TC-020-003: No Authorization header khi secret set → 401

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-003 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-020-SEC-001 |

**Điều kiện tiên quyết:** `AGENTMEMORY_SECRET` được set

**HTTP Request:** `GET /status` — không có Authorization header

**Kết quả mong đợi:** HTTP 401

---

## TC-020-004: Không có AGENTMEMORY_SECRET → local mode, không cần auth

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-020-SEC-002 |

**Điều kiện tiên quyết:** `AGENTMEMORY_SECRET` KHÔNG được set

**HTTP Request:** `GET /status` — không có Authorization header

**Kết quả mong đợi:** HTTP 200 (local mode, no auth required)

---

## TC-020-005: Timing-safe comparison trong auth middleware

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-005 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-020-SEC-003 |

**Phương pháp:** Code audit (không phải timing measurement — quá flaky)

**Các bước thực hiện:**
1. Mở file auth middleware
2. Tìm đoạn code so sánh token

**Kết quả mong đợi:**
- So sánh dùng `crypto.timingSafeEqual()` hoặc tương đương
- KHÔNG dùng `===` để compare secrets

---

## TC-020-006: Private data (API key) không lưu trong KV

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-020-SEC-004 |

**Dữ liệu đầu vào (hook payload):**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_sec` |
| `hookType` | `post_tool_use` |
| `data.tool_output` | `"Result: ANTHROPIC_API_KEY=sk-ant-api03-FAKEKEY_FOR_TESTING_ONLY"` |

**Các bước thực hiện:**
1. Gửi hook với tool_output chứa API key
2. Đọc observation từ KV
3. Kiểm tra `obs.narrative` và `obs.toolOutput`

**Kết quả mong đợi:**
- `obs.narrative` KHÔNG chứa `sk-ant-api03-`
- `obs.narrative` chứa `[REDACTED_SECRET]` tại vị trí đó
- Không có trace nào của real API key trong KV

---

## TC-020-007: Private data không xuất hiện trong recall response

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-020-SEC-004 |

**Điều kiện tiên quyết:** Observation đã được stored với redacted API key (từ TC-020-006)

**Các bước thực hiện:**
1. Gọi `mem::recall({sessionId: "sess_sec", query: "api key"})`
2. Kiểm tra response

**Kết quả mong đợi:**
- Recall response KHÔNG chứa `sk-ant-api03-`
- `[REDACTED_SECRET]` có thể xuất hiện (đó là dữ liệu đã được store)

---

## TC-020-008: Path traversal trong sessionId bị từ chối

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-008 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-020-SEC-006 |

**Dữ liệu đầu vào (test từng case):**

| sessionId | Lý do nguy hiểm | Expected |
|---|---|---|
| `"../../../etc/passwd"` | Path traversal | `{success: false}` |
| `"session; DROP TABLE--"` | SQL injection attempt | `{success: false}` |
| `"session<script>"` | XSS attempt | `{success: false}` |
| `"valid-session-id_123"` | Valid ID | `{success: true}` |

**Kết quả mong đợi:**
- Các IDs nguy hiểm đều bị từ chối với `success: false`
- Không tạo KV entry với malicious key
- Hệ thống không crash

---

## TC-020-009: API keys không xuất hiện trong error messages

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-009 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-020-SEC-007 |

**Setup:** `ANTHROPIC_API_KEY = "sk-ant-test-real-key-value"` được set

**Điều kiện:** Lỗi xảy ra khi call Anthropic API (mock trả về 500)

**Kết quả mong đợi:**
- Error message/response KHÔNG chứa `sk-ant-test-real-key-value`
- Error message có thể nói "API call failed" nhưng không expose key

---

## TC-020-010: API key không xuất hiện trong logs

| Trường | Giá trị |
|---|---|
| **ID** | TC-020-010 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-020-SEC-007 |

**Setup:** `AGENTMEMORY_SECRET = "my-super-secret-key"` set và logs được capture

**Các bước thực hiện:**
1. Start server
2. Capture tất cả log output (stdout/stderr)
3. Kiểm tra log content

**Kết quả mong đợi:** `"my-super-secret-key"` KHÔNG xuất hiện trong bất kỳ log line nào

---

## Tổng kết TC-020

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-020-001 | Valid token → 200 | 🔴 P0 | Integration |
| TC-020-002 | Wrong token → 401 | 🔴 P0 | Integration |
| TC-020-003 | No token → 401 | 🔴 P0 | Integration |
| TC-020-004 | No secret → local mode | 🔴 P0 | Integration |
| TC-020-005 | Timing-safe comparison | 🟠 P1 | Unit |
| TC-020-006 | API key not in KV | 🔴 P0 | Integration |
| TC-020-007 | API key not in recall | 🔴 P0 | Integration |
| TC-020-008 | Path traversal rejected | 🔴 P0 | Unit |
| TC-020-009 | Key not in error messages | 🟠 P1 | Unit |
| TC-020-010 | Key not in logs | 🟠 P1 | Unit |
