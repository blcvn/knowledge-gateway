# TC-002: Observe Pipeline — Test Cases

**Test Design tham chiếu:** [TD-002](../designs/TD-002-observe-pipeline.md)  
**Requirements tham chiếu:** [TR-002](../requirements/TR-002-observe-pipeline.md)  
**Module:** Privacy Redaction, Deduplication, Image Extraction, Hook Type Mapping  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## NHÓM A: PRIVACY REDACTION

---

## TC-002-001: Redact Anthropic API key (`sk-ant-*`)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-001 |
| **Tên** | stripPrivateData xóa Anthropic API key pattern |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-006 |

**Điều kiện tiên quyết:** Module `privacy.ts` được import

**Dữ liệu đầu vào:**
```
ANTHROPIC_API_KEY=sk-ant-api03-FAKE_KEY_FOR_TESTING_ONLY_NOT_REAL-xxxxxxxxxx
```

**Các bước thực hiện:**
1. Gọi `stripPrivateData()` với chuỗi đầu vào trên
2. Kiểm tra kết quả trả về

**Kết quả mong đợi:**
- Kết quả KHÔNG chứa chuỗi `sk-ant-api03-`
- Vị trí bị redact có chuỗi `[REDACTED_SECRET]`
- Phần `ANTHROPIC_API_KEY=` vẫn còn (không bị redact phần này)

**Tiêu chí Pass:**
- `result.includes("sk-ant-") === false`
- `result.includes("[REDACTED_SECRET]") === true`

---

## TC-002-002: Redact OpenAI key (`sk-proj-*`)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-002 |
| **Tên** | stripPrivateData xóa OpenAI sk-proj key |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-006 |

**Dữ liệu đầu vào:**
```
api_key=sk-proj-fakekeyfakekeyfakekeyfakekeyfakekeyfakekeyfakekeyfakek
```

**Các bước thực hiện:**
1. Gọi `stripPrivateData()` với chuỗi trên
2. Kiểm tra output

**Kết quả mong đợi:**
- `sk-proj-` không xuất hiện trong output
- `[REDACTED_SECRET]` xuất hiện thay thế

---

## TC-002-003: Redact Bearer token (Authorization header)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-003 |
| **Tên** | stripPrivateData xóa Bearer token trong Authorization header |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-007 |

**Dữ liệu đầu vào:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMSJ9.SflKxwRJSMeKKF2QT4fwpMeJf36P
```

**Các bước thực hiện:**
1. Gọi `stripPrivateData()` với header string trên
2. Kiểm tra output

**Kết quả mong đợi:**
- Token JWT bị redact
- `Authorization:` prefix vẫn còn
- `Bearer ` text có thể còn hoặc bị redact tùy implementation

**Tiêu chí Pass:** `eyJhbGci` không xuất hiện trong output.

---

## TC-002-004: Redact GitHub Personal Access Token (`ghp_*`)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-004 |
| **Tên** | stripPrivateData xóa GitHub PAT |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-006 |

**Dữ liệu đầu vào:**
```
token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh
```

**Kết quả mong đợi:**
- `ghp_ABCDEFGHI...` không xuất hiện trong output
- `[REDACTED_SECRET]` xuất hiện

---

## TC-002-005: Redact JWT token (3-part base64url structure)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-005 |
| **Tên** | stripPrivateData xóa JWT token dạng 3 phần |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-007 |

**Dữ liệu đầu vào:**
```
token=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyMSIsImlhdCI6MTYwMDAwMH0.SflKxwRJSMeKKF2QT4fwpMeJf36P
```

**Kết quả mong đợi:**
- JWT string bị redact
- `token=` label vẫn hiện

---

## TC-002-006: Redact nội dung trong `<private>` XML tags

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-006 |
| **Tên** | stripPrivateData xóa nội dung trong thẻ private |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-006 |

**Dữ liệu đầu vào:**
```
Public info <private>SECRET_VALUE_HERE</private> more public content
```

**Các bước thực hiện:**
1. Gọi `stripPrivateData()` với chuỗi trên
2. Kiểm tra output

**Kết quả mong đợi:**
- `SECRET_VALUE_HERE` không xuất hiện
- `[REDACTED]` xuất hiện ở vị trí private block
- `Public info` vẫn còn trong output
- `more public content` vẫn còn trong output

---

## TC-002-007: Nội dung bình thường KHÔNG bị redact

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-007 |
| **Tên** | stripPrivateData giữ nguyên nội dung bình thường |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-009 |

**Dữ liệu đầu vào:**
```
File written to src/auth.ts. 42 bytes changed. Status: success.
```

**Các bước thực hiện:**
1. Gọi `stripPrivateData()` với chuỗi trên
2. So sánh output với input

**Kết quả mong đợi:**
- Output giống hệt input (không có thay đổi)

**Tiêu chí Pass:** `output === input`

---

## TC-002-008: JSON structure được preserve sau redaction

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-008 |
| **Tên** | Sau redaction, JSON string vẫn parse được |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-009 |

**Dữ liệu đầu vào (JSON string):**
```json
{
  "success": true,
  "api_key": "sk-ant-api03-FAKEKEY",
  "message": "Done",
  "count": 42
}
```

**Các bước thực hiện:**
1. Gọi `stripPrivateData(JSON.stringify(inputObj))`
2. Gọi `JSON.parse()` trên kết quả
3. Kiểm tra fields của parsed object

**Kết quả mong đợi:**
- `JSON.parse()` không throw (valid JSON)
- `parsed.success = true`
- `parsed.message = "Done"`
- `parsed.count = 42`
- `parsed.api_key` chứa `[REDACTED_SECRET]` (không phải key thật)

---

## TC-002-009: Redact AWS Access Key (`AKIA*`)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-009 |
| **Tên** | stripPrivateData xóa AWS access key ID |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
```
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
```

**Kết quả mong đợi:**
- `AKIAIOSFODNN7EXAMPLE` không xuất hiện trong output

---

## NHÓM B: DEDUPLICATION

---

## TC-002-010: Cùng inputs → cùng hash (deterministic)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-010 |
| **Tên** | DedupMap.computeHash() là deterministic |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-004 |

**Dữ liệu đầu vào (lần 1 và lần 2):**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_test` |
| `toolName` | `edit_file` |
| `toolInput` | `{"path": "src/auth.ts"}` |

**Các bước thực hiện:**
1. Gọi `computeHash("sess_test", "edit_file", {path: "src/auth.ts"})` lần 1
2. Gọi `computeHash("sess_test", "edit_file", {path: "src/auth.ts"})` lần 2
3. So sánh 2 hashes

**Kết quả mong đợi:**
- `hash1 === hash2` (chuỗi hex giống nhau)

---

## TC-002-011: Khác sessionId → khác hash

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-011 |
| **Tên** | sessionId khác nhau tạo hash khác nhau |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**

| | sessionId | toolName | toolInput |
|---|---|---|---|
| Hash A | `sess_A` | `edit_file` | `{path: "auth.ts"}` |
| Hash B | `sess_B` | `edit_file` | `{path: "auth.ts"}` |

**Kết quả mong đợi:** `hashA !== hashB`

---

## TC-002-012: `isDuplicate` = false khi chưa record

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-012 |
| **Tên** | isDuplicate trả về false cho hash mới (chưa record) |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-003 |

**Dữ liệu đầu vào:**
- DedupMap mới (chưa có entries)
- Hash = `computeHash("sess_1", "edit_file", {})`

**Các bước thực hiện:**
1. Tạo DedupMap mới
2. Compute hash
3. Gọi `isDuplicate(hash)` (chưa record)

**Kết quả mong đợi:** `isDuplicate(hash) === false`

---

## TC-002-013: `isDuplicate` = true sau khi `record()`

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-013 |
| **Tên** | isDuplicate trả về true sau khi hash được record |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Các bước thực hiện:**
1. Compute hash từ inputs
2. Gọi `record(hash)`
3. Gọi `isDuplicate(hash)` ngay sau

**Kết quả mong đợi:** `isDuplicate(hash) === true`

---

## TC-002-014: Dedup TTL — entry hết hạn sau 5 phút

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-014 |
| **Tên** | Dedup entry tự động expire sau TTL 5 phút |
| **Loại** | Unit (với fake timers) |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-002-OBS-005 |

**Điều kiện tiên quyết:** Fake timer được kích hoạt (vi.useFakeTimers hoặc equivalent)

**Các bước thực hiện:**
1. Tạo DedupMap, record hash H
2. Verify `isDuplicate(H) === true`
3. Advance time by **4 minutes 59 seconds** (TTL chưa hết)
4. Verify `isDuplicate(H) === true` (còn valid)
5. Advance time thêm **2 seconds** (total > 5 minutes)
6. Verify `isDuplicate(H) === false` (đã expire)

**Kết quả mong đợi:**
- Tại bước 4: `true`
- Tại bước 6: `false`

---

## TC-002-015: Integration — Duplicate observation trả về `{deduplicated: true}`

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-015 |
| **Tên** | Observation trùng lặp bị bỏ qua và không ghi KV |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-003 |

**Dữ liệu đầu vào (payload — gửi 2 lần):**

| Trường | Giá trị |
|---|---|
| `sessionId` | `sess_dedup` |
| `hookType` | `post_tool_use` |
| `timestamp` | `2026-06-10T14:00:00.000Z` |
| `data.tool_name` | `edit_file` |
| `data.tool_input` | `{"path": "auth.ts"}` |

**Các bước thực hiện:**
1. Gửi observation lần 1
2. Verify response có `observationId` → obs được ghi KV
3. Đọc count của `mem:obs:sess_dedup` = 1
4. Gửi cùng observation lần 2 (trong vòng 5 phút)
5. Kiểm tra response lần 2
6. Đọc count của `mem:obs:sess_dedup`

**Kết quả mong đợi (bước 5):**
- `response.deduplicated = true`
- `response.sessionId = "sess_dedup"`
- Không có `observationId` mới

**Kết quả mong đợi (bước 6):**
- Count vẫn = 1 (không tạo obs thứ 2)

---

## NHÓM C: IMAGE EXTRACTION

---

## TC-002-016: Detect PNG base64 (prefix `iVBORw0KGgo`)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-016 |
| **Tên** | extractImage nhận diện PNG base64 string |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-012 |

**Dữ liệu đầu vào:**
```
iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==
```
*(PNG 1x1 pixel base64)*

**Các bước thực hiện:**
1. Gọi `extractImage(inputString)`
2. Kiểm tra return value

**Kết quả mong đợi:** Trả về chính string đó (không phải undefined)

---

## TC-002-017: Detect `data:image/` URI

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-017 |
| **Tên** | extractImage nhận diện data:image/ URI |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
```
data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAY...
```

**Kết quả mong đợi:** Trả về string đầu vào (được nhận diện)

---

## TC-002-018: Non-image input trả về undefined

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-018 |
| **Tên** | extractImage trả về undefined với các inputs không phải image |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-012 |

**Dữ liệu đầu vào (test từng case):**

| Input | Type | Expected |
|---|---|---|
| `"hello world"` | string | `undefined` |
| `""` | empty string | `undefined` |
| `null` | null | `undefined` |
| `42` | number | `undefined` |
| `{foo: "bar"}` | plain object | `undefined` |

**Các bước thực hiện:**
1. Gọi `extractImage()` với mỗi input
2. Kiểm tra return value

**Tiêu chí Pass:** Tất cả 5 cases đều trả về `undefined`

---

## TC-002-019: Extract image từ nested object (key `image_data`)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-019 |
| **Tên** | extractImage tìm image trong nested object với key image_data |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
```json
{
  "tool_name": "screenshot",
  "image_data": "iVBORw0KGgoAAAANSUh..."
}
```

**Kết quả mong đợi:** Trả về giá trị của key `image_data`

---

## NHÓM D: HOOK TYPE MAPPING

---

## TC-002-020: Mapping hookType → ObservationType (toàn bộ cases)

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-020 |
| **Tên** | Mỗi hookType được map sang đúng ObservationType |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-002-OBS-014 |

**Dữ liệu đầu vào và expected (test từng row):**

| hookType | toolName | Expected `type` |
|---|---|---|
| `post_tool_failure` | *(bất kỳ)* | `error` |
| `prompt_submit` | *(bất kỳ)* | `conversation` |
| `subagent_stop` | *(bất kỳ)* | `subagent` |
| `task_completed` | *(bất kỳ)* | `subagent` |
| `notification` | *(bất kỳ)* | `notification` |
| `post_tool_use` | `edit_file` | `file_edit` |
| `post_tool_use` | `update_file` | `file_edit` |
| `post_tool_use` | `write_file` | `file_write` |
| `post_tool_use` | `create_file` | `file_write` |
| `post_tool_use` | `read_file` | `file_read` |
| `post_tool_use` | `view_file` | `file_read` |
| `post_tool_use` | `bash` | `command_run` |
| `post_tool_use` | `shell` | `command_run` |
| `post_tool_use` | `grep` | `search` |
| `post_tool_use` | `glob` | `search` |
| `post_tool_use` | `WebFetch` | `web_fetch` |
| `post_tool_use` | `http_get` | `web_fetch` |
| `post_tool_use` | `task_agent` | `subagent` |
| `post_tool_use` | *(undefined)* | `other` |
| `session_start` | *(undefined)* | `other` |

**Các bước thực hiện (cho mỗi row):**
1. Tạo RawObservation với `hookType` và `toolName` từ row
2. Gọi `buildSyntheticCompression(raw)`
3. Kiểm tra `result.type`

**Tiêu chí Pass:** Tất cả 20 rows đều cho đúng expected type.

---

## NHÓM E: CONCURRENT OBSERVATIONS

---

## TC-002-021: 10 hooks đồng thời không mất updates

| Trường | Giá trị |
|---|---|
| **ID** | TC-002-021 |
| **Tên** | Concurrent observations được xử lý đầy đủ nhờ keyed mutex |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-002-OBS-015 |

**Điều kiện tiên quyết:** Session `sess_parallel` tồn tại

**Dữ liệu đầu vào:**
- 10 payloads với cùng `sessionId = "sess_parallel"`
- Mỗi payload có `tool_input.n = {0, 1, 2, ..., 9}` (để phân biệt)
- Timestamp mỗi payload lệch nhau 1ms

**Các bước thực hiện:**
1. Dispatch tất cả 10 hooks đồng thời (không đợi cái trước xong)
2. Chờ tất cả 10 responses
3. Kiểm tra mỗi response
4. Đọc `session.observationCount`
5. Đọc danh sách observations trong `mem:obs:sess_parallel`

**Kết quả mong đợi:**
- Tất cả 10 responses có `observationId` (không có errors)
- `session.observationCount = 10`
- `mem:obs:sess_parallel` có đúng 10 entries
- Không có observation nào bị ghi đè (10 unique IDs)

**Tiêu chí Pass:** count = 10, 10 unique observationIds.

---

## Tổng kết Module TC-002

| TC ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-002-001 | Redact sk-ant- key | 🔴 P0 | Unit |
| TC-002-002 | Redact sk-proj- key | 🔴 P0 | Unit |
| TC-002-003 | Redact Bearer token | 🔴 P0 | Unit |
| TC-002-004 | Redact GitHub PAT | 🔴 P0 | Unit |
| TC-002-005 | Redact JWT | 🔴 P0 | Unit |
| TC-002-006 | Redact private tags | 🔴 P0 | Unit |
| TC-002-007 | Normal content preserved | 🔴 P0 | Unit |
| TC-002-008 | JSON preserved after redact | 🔴 P0 | Unit |
| TC-002-009 | Redact AWS key | 🟠 P1 | Unit |
| TC-002-010 | Hash deterministic | 🔴 P0 | Unit |
| TC-002-011 | Different session → diff hash | 🔴 P0 | Unit |
| TC-002-012 | isDuplicate false before record | 🔴 P0 | Unit |
| TC-002-013 | isDuplicate true after record | 🔴 P0 | Unit |
| TC-002-014 | TTL expiry after 5 min | 🟠 P1 | Unit |
| TC-002-015 | Duplicate obs → deduplicated | 🔴 P0 | Integration |
| TC-002-016 | Detect PNG base64 | 🔴 P0 | Unit |
| TC-002-017 | Detect data:image/ URI | 🟠 P1 | Unit |
| TC-002-018 | Non-image → undefined | 🔴 P0 | Unit |
| TC-002-019 | Extract from nested object | 🟠 P1 | Unit |
| TC-002-020 | hookType → ObservationType mapping | 🟠 P1 | Unit |
| TC-002-021 | Concurrent hooks mutex | 🔴 P0 | Integration |
