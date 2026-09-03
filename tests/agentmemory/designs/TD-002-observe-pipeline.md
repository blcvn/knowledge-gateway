# TD-002: Observe Pipeline Test Design

**Liên kết Requirements:** [TR-002-observe-pipeline.md](../requirements/TR-002-observe-pipeline.md)  
**Source:**
- `references/agentmemory/src/functions/observe.ts`
- `references/agentmemory/src/functions/dedup.ts`
- `references/agentmemory/src/functions/privacy.ts`

**Test file:** `tests/agentmemory/specs/observe-pipeline.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Observe pipeline xử lý hook events từ agent tool và chuyển đổi thành `RawObservation` lưu vào KV. Pipeline bao gồm:
1. **Validation** — kiểm tra required fields
2. **Deduplication** — bỏ qua duplicates trong cửa sổ 5 phút
3. **Privacy redaction** — xóa sensitive data
4. **Image extraction** — detect và lưu base64 images
5. **KV write** — ghi raw observation
6. **Synthetic compression** — trigger compress-synthetic (hoặc LLM compress nếu enabled)

---

## 2. Chiến lược kiểm thử

| Khía cạnh | Phương pháp |
|---|---|
| Privacy regex | Unit test mỗi pattern riêng biệt |
| Dedup TTL | Unit test với fake timers |
| Image detection | Unit test mỗi format (PNG/JPEG/data:URI) |
| Pipeline flow | Integration test với MockKV + MockSdk |
| Concurrency | Integration test gửi N hooks đồng thời |

---

## 3. Test Cases

### Group A: Privacy Redaction (`stripPrivateData`)

#### TC-001 — Redact Anthropic API key (`sk-ant-*`)
**Requirement:** TR-002-OBS-006 | **Type:** unit | 🔴 P0

**Given:** String chứa `ANTHROPIC_API_KEY=sk-ant-api03-abcdef123...`  
**When:** `stripPrivateData()` được gọi  
**Then:**
- String kết quả không chứa pattern `sk-ant-`
- Vị trí bị redact có `[REDACTED_SECRET]`
- Phần còn lại của string không thay đổi

**Test Data:** `'api_key=sk-ant-api03-FAKEKEY-FOR-TESTING'`

---

#### TC-002 — Redact OpenAI key (`sk-proj-*`)
**Type:** unit | 🔴 P0

**Given:** String chứa `sk-proj-<20+ alphanumeric chars>`  
**When:** `stripPrivateData()` gọi  
**Then:** Pattern được thay thế bằng `[REDACTED_SECRET]`

---

#### TC-003 — Redact Bearer token
**Requirement:** TR-002-OBS-007 | **Type:** unit | 🔴 P0

**Given:** String có `Authorization: Bearer eyJhbGci...` (JWT-like)  
**When:** `stripPrivateData()` gọi  
**Then:** Bearer token bị redact, `Authorization:` label được giữ nguyên

---

#### TC-004 — Redact GitHub PAT (`ghp_*`, `github_pat_*`)
**Type:** unit | 🔴 P0

**Given:** String chứa `ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh`  
**When:** `stripPrivateData()` gọi  
**Then:** `ghp_...` bị thay bằng `[REDACTED_SECRET]`

---

#### TC-005 — Redact JWT 3-part structure (`eyJ...eyJ...sig`)
**Type:** unit | 🔴 P0

**Given:** String chứa JWT với 3 base64url parts ngăn bởi dấu `.`  
**When:** `stripPrivateData()` gọi  
**Then:** JWT bị redact

---

#### TC-006 — Redact `<private>...</private>` XML tags
**Requirement:** TR-002-OBS-006 | **Type:** unit | 🔴 P0

**Given:** String `"Public info <private>SECRET</private> more public"`  
**When:** `stripPrivateData()` gọi  
**Then:**
- `SECRET` bị thay bằng `[REDACTED]`
- `"Public info"` và `"more public"` được giữ nguyên

---

#### TC-007 — Normal content KHÔNG bị redact
**Requirement:** TR-002-OBS-009 | **Type:** unit | 🔴 P0

**Given:** String bình thường: `"File written to src/auth.ts. 42 bytes changed."`  
**When:** `stripPrivateData()` gọi  
**Then:** String trả về giống hệt string đầu vào

---

#### TC-008 — JSON structure được preserve sau redaction
**Requirement:** TR-002-OBS-009 | **Type:** unit | 🔴 P0

**Given:** JSON object hợp lệ có chứa `"api_key": "sk-ant-..."` trong đó  
**When:** `stripPrivateData(JSON.stringify(obj))` gọi  
**Then:**
- Output vẫn parseable (`JSON.parse()` không throw)
- Các field không sensitive giữ nguyên giá trị
- Field sensitive bị redact

---

#### TC-009 — AWS Access Key (`AKIA*`)
**Type:** unit | 🟠 P1

**Given:** String chứa `AKIAIOSFODNN7EXAMPLE`  
**When:** `stripPrivateData()` gọi  
**Then:** Pattern bị redact

---

#### TC-010 — npm token (`npm_*`)
**Type:** unit | 🟠 P1

**Given:** String chứa `npm_` + 36 alphanumeric chars  
**When:** `stripPrivateData()` gọi  
**Then:** Pattern bị redact

---

### Group B: Deduplication (`DedupMap`)

#### TC-011 — Cùng inputs → cùng hash (deterministic)
**Requirement:** TR-002-OBS-004 | **Type:** unit | 🔴 P0

**Given:** DedupMap mới  
**When:** `computeHash(sessId, toolName, toolInput)` gọi 2 lần với cùng args  
**Then:** 2 hash values giống hệt nhau

---

#### TC-012 — Khác sessionId → khác hash
**Type:** unit | 🔴 P0

**Given:** Cùng toolName và toolInput  
**When:** sessionId khác nhau  
**Then:** Hash khác nhau

---

#### TC-013 — Khác toolInput → khác hash
**Type:** unit | 🔴 P0

**Given:** Cùng sessionId và toolName  
**When:** toolInput khác nhau (`{path: "auth.ts"}` vs `{path: "other.ts"}`)  
**Then:** Hash khác nhau

---

#### TC-014 — `isDuplicate` = false trước khi record
**Requirement:** TR-002-OBS-003 | **Type:** unit | 🔴 P0

**Given:** DedupMap trống  
**When:** `isDuplicate(hash)` gọi với hash chưa được record  
**Then:** Trả về `false`

---

#### TC-015 — `isDuplicate` = true sau khi `record()`
**Type:** unit | 🔴 P0

**Given:** Hash vừa được `record()`  
**When:** `isDuplicate(hash)` gọi ngay sau đó  
**Then:** Trả về `true`

---

#### TC-016 — Dedup entry hết hạn sau 5 phút (TTL)
**Requirement:** TR-002-OBS-005 | **Type:** unit (với fake timers) | 🟠 P1

**Given:** Hash vừa được `record()`  
**When:** Giả lập thời gian qua 5 phút + 1ms  
**Then:** `isDuplicate(hash)` = `false` (entry đã expire)

**Kỹ thuật:** `vi.useFakeTimers()` để kiểm soát `Date.now()`

---

#### TC-017 — Integration: Duplicate observation → `{deduplicated: true}`, không ghi KV
**Requirement:** TR-002-OBS-003 | **Type:** integration | 🔴 P0

**Given:** Observation đã được gửi một lần  
**When:** Cùng payload gửi lần 2 (trong 5 phút)  
**Then:**
- Response `{deduplicated: true, sessionId: ...}`
- KV chỉ có 1 observation (không thêm)

---

### Group C: Image Extraction (`extractImage`)

#### TC-018 — Detect PNG base64 (prefix `iVBORw0KGgo`)
**Requirement:** TR-002-OBS-012 | **Type:** unit | 🔴 P0

**Given:** String bắt đầu bằng `iVBORw0KGgoAAAA...`  
**When:** `extractImage()` gọi  
**Then:** Trả về đúng string đó

---

#### TC-019 — Detect JPEG base64 (prefix `/9j/`)
**Requirement:** TR-002-OBS-013 | **Type:** unit | 🟠 P1

**Given:** String bắt đầu bằng `/9j/4AAQSkZJRg...`  
**When:** `extractImage()` gọi  
**Then:** String được trả về (recognized as JPEG)

---

#### TC-020 — Detect `data:image/` URI
**Type:** unit | 🟠 P1

**Given:** String `"data:image/png;base64,iVBORw0KGgo..."`  
**When:** `extractImage()` gọi  
**Then:** String được trả về

---

#### TC-021 — Non-image trả về `undefined`
**Type:** unit | 🔴 P0

**Given:** Các inputs sau: `"hello world"`, `""`, `null`, `42`, `{foo: "bar"}`  
**When:** `extractImage()` gọi với mỗi input  
**Then:** Tất cả trả về `undefined`

---

#### TC-022 — Extract từ nested object với key `image_data`
**Requirement:** TR-002-OBS-012 | **Type:** unit | 🟠 P1

**Given:** Object `{tool_name: "screenshot", image_data: "iVBORw0KGgo..."}`  
**When:** `extractImage(object)` gọi  
**Then:** Trả về giá trị của `image_data`

---

#### TC-023 — Extract từ nested object với key `imagePath`
**Type:** unit | 🟡 P2

**Given:** Object `{imagePath: "/path/to/screenshot.png"}`  
**When:** `extractImage()` gọi  
**Then:** Trả về `"/path/to/screenshot.png"`

---

### Group D: Hook Type Mapping

#### TC-024 — Mỗi hookType được map sang đúng ObservationType
**Requirement:** TR-002-OBS-014 | **Type:** unit | 🟠 P1

**Given:** Các cặp `(hookType, toolName)` khác nhau (xem bảng)  
**When:** Observation được processed (synthetic compression)  
**Then:** `type` field đúng với expected

| hookType | toolName | Expected type |
|---|---|---|
| `post_tool_failure` | * | `error` |
| `prompt_submit` | * | `conversation` |
| `subagent_stop` | * | `subagent` |
| `task_completed` | * | `subagent` |
| `notification` | * | `notification` |
| `post_tool_use` | `edit_file` | `file_edit` |
| `post_tool_use` | `write_file` | `file_write` |
| `post_tool_use` | `read_file` | `file_read` |
| `post_tool_use` | `bash` | `command_run` |
| `post_tool_use` | `grep` | `search` |
| `post_tool_use` | `WebFetch` | `web_fetch` |
| `post_tool_use` | (không có) | `other` |

---

### Group E: Concurrent Observations

#### TC-025 — 10 hooks đồng thời: không mất updates
**Requirement:** TR-002-OBS-015 | **Type:** integration | 🔴 P0

**Given:** Session tồn tại  
**When:** 10 hooks gửi đồng thời (cùng sessionId)  
**Then:**
- Tất cả 10 hooks được xử lý thành công
- `observationCount = 10`
- Mỗi hook có unique `observationId`

**Lý giải:** Test này xác nhận `withKeyedLock("obs:sessionId")` ngăn lost updates.

---

## 4. Coverage Notes

| Function | Branches cần cover |
|---|---|
| `stripPrivateData` | 13 regex patterns (mỗi cái 1 test), `<private>` tag |
| `extractImage` | string/object input, các keys khác nhau, null/undefined |
| `DedupMap` | `computeHash`, `isDuplicate` (new/recorded/expired), `record`, cleanup |
| `registerObserveFunction` | Validation branches, dedup path, image path, limit path |
