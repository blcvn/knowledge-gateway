# TC-015: REST API — Test Cases

**Test Design tham chiếu:** [TD-015](../designs/TD-015-rest-api.md)  
**Requirements tham chiếu:** [TR-015](../requirements/TR-015-rest-api.md)  
**Module:** Sessions, Observations, Search, Memories, Health  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-015-001: GET /sessions → danh sách sessions

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-015-API-001 |

**Điều kiện tiên quyết:** 3 sessions trong KV: `sess_1`, `sess_2`, `sess_3`

**HTTP Request:**

| | Giá trị |
|---|---|
| Method | `GET` |
| Path | `/sessions` |

**Kết quả mong đợi:**
- HTTP 200
- `Content-Type: application/json`
- Body: `{sessions: [...]}` với `sessions.length = 3`
- Mỗi item có: `id`, `project`, `status`, `observationCount`, `startedAt`

---

## TC-015-002: GET /sessions/:id/observations → obs của session

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-015-API-002 |

**Điều kiện tiên quyết:** Session `sess_abc` có 5 observations

**HTTP Request:** `GET /sessions/sess_abc/observations`

**Kết quả mong đợi:**
- HTTP 200
- `observations.length = 5`
- Mỗi obs có: `id`, `timestamp`, `type`, `title`
- Sorted by `timestamp` (ascending)

---

## TC-015-003: GET /sessions/:id không tồn tại → 404

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-003 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-015-API-003 |

**HTTP Request:** `GET /sessions/does_not_exist_999`

**Kết quả mong đợi:**
- HTTP 404
- Body có `error` field

---

## TC-015-004: GET /search?q=auth → sorted results

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-015-API-004 |

**Điều kiện tiên quyết:** 5 observations có "auth" trong content, indexed trong BM25

**HTTP Request:** `GET /search?q=auth`

**Kết quả mong đợi:**
- HTTP 200
- `results` là array với ít nhất 1 item
- Mỗi result có: `observation`, `bm25Score`, `vectorScore`, `combinedScore`
- Results sorted: `results[i].combinedScore >= results[i+1].combinedScore`

---

## TC-015-005: GET /search?q=auth&limit=5 → đúng limit

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-015-API-005 |

**Điều kiện tiên quyết:** 20 observations liên quan "auth"

**HTTP Request:** `GET /search?q=auth&limit=5`

**Kết quả mong đợi:** `results.length = 5`

---

## TC-015-006: GET /memories → chỉ isLatest=true

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-015-API-006 |

**Setup:**
- M1: `isLatest = false` (bị supersede)
- M2: `isLatest = true` (active)

**HTTP Request:** `GET /memories`

**Kết quả mong đợi:**
- Response body chứa M2
- M1 KHÔNG xuất hiện trong response

---

## TC-015-007: POST /memories → tạo memory mới (201)

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-015-API-007 |

**HTTP Request:**

| | Giá trị |
|---|---|
| Method | `POST` |
| Path | `/memories` |
| Content-Type | `application/json` |
| Body | `{"content": "Auth uses JWT with RS256", "type": "architecture"}` |

**Kết quả mong đợi:**
- HTTP 201
- Body chứa memory object với:
  - `id`: string
  - `version = 1`
  - `isLatest = true`
  - `content = "Auth uses JWT with RS256"`

---

## TC-015-008: DELETE /memories/:id → xóa memory

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-008 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-015-API-008 |

**Điều kiện tiên quyết:** Memory `mem_del_target` tồn tại

**Các bước thực hiện:**
1. `DELETE /memories/mem_del_target` → verify 200
2. `GET /memories/mem_del_target` → verify 404

**Kết quả mong đợi (bước 1):**
- HTTP 200
- `{success: true, deleted: 1}`

**Kết quả mong đợi (bước 2):** HTTP 404

---

## TC-015-009: POST /memories thiếu content → 422

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-009 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-015-API-009 |

**HTTP Request:**

| | Giá trị |
|---|---|
| Body | `{"type": "fact"}` — thiếu `content` |

**Kết quả mong đợi:**
- HTTP 422 (Unprocessable Entity)
- Body có `error` field chứa thông tin về trường bị thiếu

---

## TC-015-010: GET /health → 200 khi healthy

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-010 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-015-API-010 |

**HTTP Request:** `GET /health`

**Kết quả mong đợi:**
- HTTP 200
- `{status: "ok", uptime: <number>, version: "..."}`

---

## TC-015-011: Internal error → 500 không expose stack trace

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-011 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-015-API-011 |

**Setup:** KV được inject lỗi (simulate KV crash)

**HTTP Request:** `GET /sessions`

**Kết quả mong đợi:**
- HTTP 500
- Body: `{error: "internal server error"}` (generic message)
- Body KHÔNG chứa: stack trace, file paths, error details

---

## TC-015-012: CORS headers tồn tại trong response

| Trường | Giá trị |
|---|---|
| **ID** | TC-015-012 |
| **Loại** | Integration |
| **Ưu tiên** | 🟡 P2 |

**HTTP Request:** `GET /health` với `Origin: http://localhost:3000`

**Kết quả mong đợi:**
- Response có `Access-Control-Allow-Origin` header

---

## Tổng kết TC-015

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-015-001 | GET /sessions | 🔴 P0 | Integration |
| TC-015-002 | GET /sessions/:id/observations | 🔴 P0 | Integration |
| TC-015-003 | GET /sessions/:id 404 | 🔴 P0 | Integration |
| TC-015-004 | GET /search sorted | 🔴 P0 | Integration |
| TC-015-005 | GET /search limit | 🟠 P1 | Integration |
| TC-015-006 | GET /memories isLatest | 🔴 P0 | Integration |
| TC-015-007 | POST /memories 201 | 🔴 P0 | Integration |
| TC-015-008 | DELETE /memories | 🟠 P1 | Integration |
| TC-015-009 | POST /memories 422 | 🔴 P0 | Integration |
| TC-015-010 | GET /health 200 | 🔴 P0 | Integration |
| TC-015-011 | 500 no stack trace | 🟠 P1 | Integration |
| TC-015-012 | CORS headers | 🟡 P2 | Integration |
