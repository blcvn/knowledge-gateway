# TC-016: Session Replay — Test Cases

**Test Design tham chiếu:** [TD-016](../designs/TD-016-session-replay.md)  
**Requirements tham chiếu:** [TR-016](../requirements/TR-016-session-replay.md)  
**Module:** JSONL Import, Fingerprint Dedup, Viewer Server  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-016-001: Import JSONL với valid transcript

| Trường | Giá trị |
|---|---|
| **ID** | TC-016-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-016-REP-001 |

**Dữ liệu đầu vào (file `transcript.jsonl`):**
- 10 lines, mỗi line là JSON object hợp lệ với các fields: `sessionId`, `hookType`, `timestamp`, `data`

**Ví dụ một line:**
```
{"sessionId":"replay_sess_1","hookType":"post_tool_use","timestamp":"2026-06-10T10:00:00.000Z","data":{"tool_name":"edit_file","tool_input":{"path":"auth.ts"}}}
```

**Các bước thực hiện:**
1. Gọi `mem::import-replay({path: "transcript.jsonl"})`
2. Kiểm tra response
3. Đọc KV `mem:obs:replay_sess_1`
4. Đọc KV `mem:sessions["replay_sess_1"]`

**Kết quả mong đợi:**
- `{success: true, importedCount: 10}`
- KV có đúng 10 observations cho session
- Session tồn tại với `source = "replay"`

---

## TC-016-002: Malformed JSONL — skip invalid lines, count valid

| Trường | Giá trị |
|---|---|
| **ID** | TC-016-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-016-REP-002 |

**Dữ liệu đầu vào (file `mixed.jsonl`):**

| Line | Nội dung | Type |
|---|---|---|
| 1-8 | JSON hợp lệ | Valid |
| 9 | `{broken json` | Invalid |
| 10 | `not json at all` | Invalid |

**Các bước thực hiện:**
1. Gọi `mem::import-replay({path: "mixed.jsonl"})`
2. Kiểm tra response

**Kết quả mong đợi:**
- `importedCount = 8` (chỉ valid lines)
- `skippedCount = 2` (hoặc có warning về 2 lines)
- Không crash

---

## TC-016-003: Import idempotent — gọi 2 lần không tạo duplicate

| Trường | Giá trị |
|---|---|
| **ID** | TC-016-003 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-016-REP-003 |

**Điều kiện tiên quyết:** File `transcript.jsonl` đã được import lần 1 (TC-016-001)

**Các bước thực hiện:**
1. Gọi `mem::import-replay({path: "transcript.jsonl"})` lần 2
2. Kiểm tra response
3. Đếm obs trong KV

**Kết quả mong đợi:**
- `{importedCount: 0, skippedCount: 10}` — tất cả đã tồn tại
- KV count không tăng (vẫn là 10, không phải 20)

**Cơ chế:** Fingerprint dedup dùng `fingerprintId(sessionId + hookType + timestamp + toolName)`

---

## TC-016-004: Observation data được preserved đầy đủ sau import

| Trường | Giá trị |
|---|---|
| **ID** | TC-016-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-016-REP-004 |

**Dữ liệu đầu vào (JSONL line với đầy đủ fields):**

| Field | Giá trị |
|---|---|
| `sessionId` | `replay_verify` |
| `hookType` | `post_tool_use` |
| `timestamp` | `2026-06-10T12:00:00.000Z` |
| `data.tool_name` | `edit_file` |
| `data.tool_input` | `{"path": "src/auth.ts"}` |
| `data.tool_output` | `"File updated successfully"` |
| `agentId` | `claude-3-5-sonnet` |

**Các bước thực hiện:**
1. Import JSONL với line trên
2. Đọc observation từ KV
3. Kiểm tra từng field

**Kết quả mong đợi:**
- `obs.type` = `"file_edit"` (mapped từ tool_name)
- `obs.files` chứa `"src/auth.ts"`
- `obs.agentId = "claude-3-5-sonnet"`
- `obs.timestamp = "2026-06-10T12:00:00.000Z"`
- Nội dung tool_input được preserve trong `narrative`

---

## TC-016-005: Viewer server trả về session list

| Trường | Giá trị |
|---|---|
| **ID** | TC-016-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-016-REP-005 |

**Điều kiện tiên quyết:** 3 sessions trong KV (bao gồm cả replay sessions), Viewer server đang chạy tại port 4000

**HTTP Request:** `GET http://localhost:4000/api/sessions`

**Kết quả mong đợi:**
- HTTP 200
- Body là JSON array với 3 sessions
- Mỗi item có `id`, `source`, `observationCount`

---

## TC-016-006: Observations được serve theo timeline order

| Trường | Giá trị |
|---|---|
| **ID** | TC-016-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-016-REP-006 |

**Setup:** Session với 5 observations, timestamps không theo thứ tự

**HTTP Request:** `GET /api/sessions/replay_sess_1/observations`

**Kết quả mong đợi:** Observations được sorted theo `timestamp` ascending

---

## Tổng kết TC-016

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-016-001 | Import valid JSONL | 🔴 P0 | Integration |
| TC-016-002 | Skip invalid lines | 🔴 P0 | Integration |
| TC-016-003 | Import idempotent | 🟠 P1 | Integration |
| TC-016-004 | Data preserved after import | 🔴 P0 | Integration |
| TC-016-005 | Viewer session list | 🟠 P1 | Integration |
| TC-016-006 | Timeline order | 🟠 P1 | Integration |
