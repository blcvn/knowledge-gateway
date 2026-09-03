# TD-003: Synthetic Compression Test Design

**Liên kết Requirements:** [TR-003-compress-synthetic.md](../requirements/TR-003-compress-synthetic.md)  
**Source:** `references/agentmemory/src/functions/compress-synthetic.ts`  
**Test file:** `tests/agentmemory/specs/compress-synthetic.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Synthetic compression chuyển đổi `RawObservation` thành `CompressedObservation` **không dùng LLM** — đây là default path (khi `AGENTMEMORY_AUTO_COMPRESS=false`).

Hàm chính: `buildSyntheticCompression(raw: RawObservation): CompressedObservation`

**Các điểm kiểm thử:**
- `inferType()` — ánh xạ hookType/toolName → ObservationType
- `extractFiles()` — trích xuất file paths từ toolInput
- `truncate()` — cắt chuỗi đúng độ dài với ellipsis
- Output structure đầy đủ các fields bắt buộc

---

## 2. Ghi chú Implementation (quan trọng)

Sau khi đọc source code thực tế (`compress-synthetic.ts`):

| Field | Spec gốc (TR-003) | Actual Implementation |
|---|---|---|
| `title` | "Wrote: src/auth.ts" | `truncate(toolName \|\| hookType, 80)` — chỉ toolName |
| `importance` | 0.0-1.0 float | `5` (integer cố định) |
| `confidence` | dynamic | `0.3` (cố định) |
| `facts` | từ toolOutput | `[]` (luôn empty trong synthetic path) |
| `concepts` | từ keywords | `[]` (luôn empty trong synthetic path) |

> **⚠️ Test design này follow actual implementation, không phải spec gốc.**  
> Nên cập nhật TR-003 để phản ánh thực tế.

---

## 3. Chiến lược kiểm thử

| Khía cạnh | Kỹ thuật |
|---|---|
| `inferType` mapping | Parameterized tests với bảng 20+ cases |
| Truncation | Boundary value: n-1, n, n+1 chars |
| File extraction | Equivalence partition: mỗi key hỗ trợ |
| Output structure | Structural assertion |

---

## 4. Test Cases

### Group A: Type Inference (`inferType`)

#### TC-001 — hookType-based type mapping (không phụ thuộc toolName)
**Requirement:** TR-003-SYN-001..SYN-004 | **Type:** unit | 🔴 P0

**Given:** Raw observation với hookType nhất định  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.type` theo bảng sau:

| hookType | toolName | Expected `type` |
|---|---|---|
| `post_tool_failure` | bất kỳ | `error` |
| `prompt_submit` | bất kỳ | `conversation` |
| `subagent_stop` | bất kỳ | `subagent` |
| `task_completed` | bất kỳ | `subagent` |
| `notification` | bất kỳ | `notification` |

**Kỹ thuật:** Bảng test (it.each)

---

#### TC-002 — Tool-name-based type mapping
**Requirement:** TR-003-SYN-005..SYN-009 | **Type:** unit | 🔴 P0

**Given:** hookType = `post_tool_use`, toolName khác nhau  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.type` theo bảng:

| toolName | Expected `type` |
|---|---|
| `edit_file`, `update_file`, `replace_text` | `file_edit` |
| `write_file`, `create_file` | `file_write` |
| `read_file`, `view_file` | `file_read` |
| `bash`, `shell`, `exec`, `run` | `command_run` |
| `grep`, `search`, `glob`, `find` | `search` |
| `WebFetch`, `http_get`, `web` | `web_fetch` |
| `task`, `agent` | `subagent` |
| (không có toolName) | `other` |
| `unknown_tool_xyz` | `other` |

**Ghi chú:** Normalization: `WebFetch` → lowercase `webfetch`, split camelCase bằng `_`

---

#### TC-003 — Fallback: toolName không có từ nào khớp → `other`
**Type:** unit | 🟠 P1

**Given:** hookType = `post_tool_use`, toolName = `some_completely_unknown_tool_xyz`  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.type = "other"`

---

### Group B: Title Generation

#### TC-004 — Title = toolName được truncate đến 80 ký tự
**Requirement:** TR-003-SYN-004 | **Type:** unit | 🔴 P0

**Given:** toolName là chuỗi dài hơn 80 ký tự  
**When:** `buildSyntheticCompression()` gọi  
**Then:**
- `result.title` dài đúng 80 ký tự
- Ký tự cuối là `…` (ellipsis U+2026)

**Boundary values:** len=79 (pass-through), len=80 (pass-through), len=81 (truncated)

---

#### TC-005 — Title = hookType khi không có toolName
**Type:** unit | 🟠 P1

**Given:** hookType = `session_start`, toolName = undefined  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.title = "session_start"` (hookType được dùng thay toolName)

---

### Group C: Subtitle

#### TC-006 — Subtitle = toolInput stringify, truncated 120 chars
**Type:** unit | 🟠 P1

**Given:** toolInput = `{path: "src/auth.ts", changes: "Added JWT validation"}`  
**When:** `buildSyntheticCompression()` gọi  
**Then:**
- `result.subtitle` là string chứa path và changes
- Dài tối đa 120 ký tự

---

#### TC-007 — Subtitle là `undefined` khi không có toolInput
**Type:** unit | 🟠 P1

**Given:** hookType = `prompt_submit`, toolInput = undefined  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.subtitle = undefined`

---

### Group D: File Extraction

#### TC-008 — Extract từ mỗi supported key
**Requirement:** TR-003-SYN-010 | **Type:** unit | 🟠 P1

**Given:** toolInput với từng key sau (test riêng):
- `{path: "src/auth.ts"}`
- `{file_path: "src/logger.ts"}`
- `{filepath: "src/db.ts"}`
- `{filePath: "src/Button.tsx"}`
- `{file: "src/utils.ts"}`
- `{pattern: "src/**/*.ts"}`

**When:** `buildSyntheticCompression()` gọi  
**Then:** Path tương ứng xuất hiện trong `result.files`

---

#### TC-009 — Dedup file paths: cùng path từ 2 keys khác nhau → chỉ 1 entry
**Type:** unit | 🟡 P2

**Given:** toolInput = `{path: "auth.ts", file_path: "auth.ts"}` (2 keys, cùng giá trị)  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.files` chỉ có 1 entry `"auth.ts"`

---

#### TC-010 — Không extract khi toolInput = null/undefined
**Type:** unit | 🔴 P0

**Given:** hookType = `session_start`, toolInput = undefined  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.files = []`

---

#### TC-011 — Path quá dài (≥ 512 chars) bị bỏ qua
**Type:** unit | 🟡 P2

**Given:** toolInput = `{path: "a".repeat(512)}`  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.files = []` (path quá dài không được extract)

---

### Group E: Narrative

#### TC-012 — Narrative kết hợp prompt, input và output với `|`
**Type:** unit | 🟠 P1

**Given:** Raw observation có cả `userPrompt`, `toolInput` và `toolOutput`  
**When:** `buildSyntheticCompression()` gọi  
**Then:**
- Narrative chứa nội dung của tất cả 3 phần
- Các phần ngăn cách bởi ` | `

---

#### TC-013 — Narrative bị truncate tại 400 ký tự
**Type:** unit | 🟠 P1

**Given:** toolInput có path dài 500 ký tự  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.narrative.length ≤ 400` và kết thúc bằng `…`

---

### Group F: Fixed Values

#### TC-014 — `importance = 5` (integer cố định, không phụ thuộc loại hook)
**Type:** unit | 🔴 P0

**Given:** Nhiều loại hookType khác nhau  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.importance = 5` trong mọi trường hợp

---

#### TC-015 — `confidence = 0.3` (cố định)
**Type:** unit | 🟠 P1

**Given:** Bất kỳ RawObservation nào  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.confidence = 0.3`

---

#### TC-016 — `facts = []`, `concepts = []` (luôn empty trong synthetic path)
**Type:** unit | 🟡 P2

**Given:** Bất kỳ RawObservation nào  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.facts = []`, `result.concepts = []`

---

### Group G: Output Structure và Data Propagation

#### TC-017 — Output có đủ required fields của `CompressedObservation`
**Requirement:** TR-003-SYN-012 | **Type:** unit | 🔴 P0

**Given:** Một RawObservation hợp lệ với đầy đủ fields  
**When:** `buildSyntheticCompression()` gọi  
**Then:** Output có đủ: `id`, `sessionId`, `timestamp`, `type`, `title`, `facts`, `narrative`, `concepts`, `files`, `importance`, `confidence`

---

#### TC-018 — `id`, `sessionId`, `timestamp` được copy từ raw
**Type:** unit | 🔴 P0

**Given:** Raw có id=`obs_abc`, sessionId=`sess_xyz`, timestamp=`2026-06-10T14:00:00Z`  
**When:** `buildSyntheticCompression()` gọi  
**Then:** Output có đúng `id`, `sessionId`, `timestamp` từ raw

---

#### TC-019 — `agentId` được propagate từ raw observation
**Requirement:** TR-002-OBS-016 | **Type:** unit | 🟠 P1

**Given:** Raw có `agentId = "claude-code-1"`  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.agentId = "claude-code-1"`

---

#### TC-020 — `modality` được propagate từ raw
**Type:** unit | 🟠 P1

**Given:** Raw có `modality = "image"`  
**When:** `buildSyntheticCompression()` gọi  
**Then:** `result.modality = "image"`

---

## 5. Coverage Notes

| Function | Branches cần cover |
|---|---|
| `inferType` | hookType-based (5 cases), tool word matching (7 groups), fallback |
| `extractFiles` | mỗi 6 key, null input, path length limit |
| `truncate` | n<limit, n=limit, n>limit |
| `buildSyntheticCompression` | all 3 narrative parts, modality, agentId, imageData |
