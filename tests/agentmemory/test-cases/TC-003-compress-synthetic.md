# TC-003: Synthetic Compression — Test Cases

**Test Design tham chiếu:** [TD-003](../designs/TD-003-compress-synthetic.md)  
**Requirements tham chiếu:** [TR-003](../requirements/TR-003-compress-synthetic.md)  
**Module:** buildSyntheticCompression, inferType, extractFiles, truncate  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

> **Lưu ý:** Các test case này follow **actual implementation**, không phải spec gốc.  
> `importance` = `5` (integer cố định), `confidence` = `0.3`, `facts/concepts` = `[]`

---

## NHÓM A: TYPE INFERENCE

---

## TC-003-001: hookType-based type mapping (5 special hookTypes)

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-001 |
| **Tên** | hookType đặc biệt được map thẳng sang ObservationType |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-003-SYN-001..SYN-004 |

**Dữ liệu đầu vào và expected (test từng row):**

| hookType | toolName | Expected `type` |
|---|---|---|
| `post_tool_failure` | `bash` | `error` |
| `prompt_submit` | *(undefined)* | `conversation` |
| `subagent_stop` | *(undefined)* | `subagent` |
| `task_completed` | `task_agent` | `subagent` |
| `notification` | *(undefined)* | `notification` |

**Các bước thực hiện (cho mỗi row):**
1. Tạo RawObservation với hookType và toolName tương ứng
2. Gọi `buildSyntheticCompression(raw)`
3. Kiểm tra `result.type`

**Tiêu chí Pass:** `result.type === expected_type` cho cả 5 rows.

---

## TC-003-002: Tool-name-based type mapping (`post_tool_use`)

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-002 |
| **Tên** | Các toolName khác nhau được map sang đúng ObservationType |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-003-SYN-005..SYN-009 |

**Điều kiện tiên quyết:** hookType = `post_tool_use` cho tất cả rows

**Dữ liệu đầu vào và expected:**

| toolName | Expected `type` |
|---|---|
| `edit_file` | `file_edit` |
| `update_file` | `file_edit` |
| `replace_text` | `file_edit` |
| `write_file` | `file_write` |
| `create_file` | `file_write` |
| `read_file` | `file_read` |
| `view_file` | `file_read` |
| `bash` | `command_run` |
| `shell` | `command_run` |
| `exec` | `command_run` |
| `run` | `command_run` |
| `grep` | `search` |
| `find` | `search` |
| `glob` | `search` |
| `WebFetch` | `web_fetch` |
| `http_get` | `web_fetch` |
| `web` | `web_fetch` |
| `task` | `subagent` |
| `agent` | `subagent` |

**Tiêu chí Pass:** Tất cả 19 rows đều cho đúng expected type.

---

## TC-003-003: toolName không khớp → `type = "other"`

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-003 |
| **Tên** | toolName không xác định được fallback sang type=other |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `hookType` | `post_tool_use` |
| `toolName` | `some_completely_unknown_tool_xyz` |

**Kết quả mong đợi:** `result.type = "other"`

---

## TC-003-004: toolName = undefined → `type = "other"`

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-004 |
| **Tên** | Không có toolName với post_tool_use → other |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `hookType` | `post_tool_use` |
| `toolName` | *(undefined)* |

**Kết quả mong đợi:** `result.type = "other"`

---

## NHÓM B: TITLE GENERATION

---

## TC-003-005: Title = toolName khi ngắn hơn 80 ký tự

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-005 |
| **Tên** | Title là toolName nguyên vẹn khi ≤ 80 chars |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-003-SYN-004 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `hookType` | `post_tool_use` |
| `toolName` | `edit_file` (9 chars) |

**Kết quả mong đợi:** `result.title = "edit_file"` (giữ nguyên)

---

## TC-003-006: Title bị truncate tại 80 ký tự (boundary)

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-006 |
| **Tên** | Title bị truncate tại đúng 80 ký tự với ellipsis |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào (test 3 boundary cases):**

| toolName | Độ dài | Expected title |
|---|---|---|
| `"a".repeat(79)` | 79 | Nguyên vẹn (79 chars) |
| `"a".repeat(80)` | 80 | Nguyên vẹn (80 chars) |
| `"a".repeat(81)` | 81 | 79 chars `"a"` + `"…"` = 80 chars |

**Các bước thực hiện:**
1. Test với toolName 79 chars → title.length = 79, không có ellipsis
2. Test với toolName 80 chars → title.length = 80, không có ellipsis
3. Test với toolName 81 chars → title.length = 80, kết thúc bằng `"…"`

**Tiêu chí Pass:** Truncation xảy ra chính xác tại boundary = 80.

---

## TC-003-007: Title = hookType khi không có toolName

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-007 |
| **Tên** | Title dùng hookType khi toolName undefined |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `hookType` | `session_start` |
| `toolName` | *(undefined)* |

**Kết quả mong đợi:** `result.title = "session_start"`

---

## NHÓM C: FILE EXTRACTION

---

## TC-003-008: Extract file path từ mỗi supported key

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-008 |
| **Tên** | extractFiles nhận diện tất cả 6 key variants |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-003-SYN-010 |

**Dữ liệu đầu vào (test từng row):**

| toolInput | Expected file in result.files |
|---|---|
| `{path: "src/auth.ts"}` | `"src/auth.ts"` |
| `{file_path: "src/logger.ts"}` | `"src/logger.ts"` |
| `{filepath: "src/db.ts"}` | `"src/db.ts"` |
| `{filePath: "src/Button.tsx"}` | `"src/Button.tsx"` |
| `{file: "src/utils.ts"}` | `"src/utils.ts"` |
| `{pattern: "src/**/*.ts"}` | `"src/**/*.ts"` |

**Các bước thực hiện (mỗi row):**
1. Tạo RawObservation với toolInput tương ứng
2. Gọi `buildSyntheticCompression(raw)`
3. Kiểm tra `result.files[]`

**Tiêu chí Pass:** Expected file path xuất hiện trong `result.files`.

---

## TC-003-009: File path trùng từ 2 keys → chỉ 1 entry (dedup)

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-009 |
| **Tên** | extractFiles dedup file paths trùng nhau |
| **Loại** | Unit |
| **Ưu tiên** | 🟡 P2 |

**Dữ liệu đầu vào:**
```json
{
  "path": "auth.ts",
  "file_path": "auth.ts"
}
```

**Kết quả mong đợi:** `result.files = ["auth.ts"]` (chỉ 1 entry, không phải 2)

---

## TC-003-010: toolInput = undefined → `files = []`

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-010 |
| **Tên** | Không extract files khi toolInput undefined |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `hookType` | `session_start` |
| `toolInput` | *(undefined)* |

**Kết quả mong đợi:** `result.files = []`

---

## TC-003-011: Path quá dài (≥ 512 chars) bị bỏ qua

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-011 |
| **Tên** | Path > 512 ký tự không được extract |
| **Loại** | Unit |
| **Ưu tiên** | 🟡 P2 |

**Dữ liệu đầu vào:**
```json
{
  "path": "aaaa...aaaa"
}
```
*(path = chuỗi 512 ký tự "a")*

**Kết quả mong đợi:** `result.files = []` (path quá dài bị bỏ qua)

---

## NHÓM D: FIXED VALUES

---

## TC-003-012: `importance` luôn là 5

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-012 |
| **Tên** | importance = 5 cố định bất kể hookType |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào (test với 3 hookTypes khác nhau):**
- `session_start`, `post_tool_use`, `post_tool_failure`

**Các bước thực hiện:**
1. Gọi `buildSyntheticCompression()` với mỗi hookType
2. Kiểm tra `result.importance`

**Kết quả mong đợi:** `result.importance === 5` trong cả 3 cases

---

## TC-003-013: `confidence` luôn là 0.3

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-013 |
| **Tên** | confidence = 0.3 cố định |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Kết quả mong đợi:** `result.confidence === 0.3`

---

## TC-003-014: `facts` và `concepts` luôn là mảng rỗng

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-014 |
| **Tên** | facts và concepts là empty arrays trong synthetic path |
| **Loại** | Unit |
| **Ưu tiên** | 🟡 P2 |

**Kết quả mong đợi:**
- `result.facts` = `[]` (mảng rỗng)
- `result.concepts` = `[]` (mảng rỗng)

---

## NHÓM E: OUTPUT STRUCTURE

---

## TC-003-015: Output có đủ required fields

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-015 |
| **Tên** | CompressedObservation output có đủ tất cả required fields |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-003-SYN-012 |

**Dữ liệu đầu vào (đầy đủ):**

| Trường | Giá trị |
|---|---|
| `id` | `obs_test_001` |
| `sessionId` | `sess_abc` |
| `timestamp` | `2026-06-10T14:00:00.000Z` |
| `hookType` | `post_tool_use` |
| `toolName` | `edit_file` |
| `toolInput` | `{path: "src/auth.ts"}` |
| `toolOutput` | `"File updated"` |
| `userPrompt` | `"Update auth"` |

**Các bước thực hiện:**
1. Gọi `buildSyntheticCompression(raw)`
2. Kiểm tra tồn tại của mỗi field

**Kết quả mong đợi — tất cả fields phải tồn tại:**
- `id` (string)
- `sessionId` (string)
- `timestamp` (string)
- `type` (string)
- `title` (string)
- `narrative` (string)
- `facts` (array)
- `concepts` (array)
- `files` (array)
- `importance` (number)
- `confidence` (number)

**Tiêu chí Pass:** Không có field nào là `undefined` (ngoại trừ optionals).

---

## TC-003-016: `id`, `sessionId`, `timestamp` được copy từ raw

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-016 |
| **Tên** | Identity fields được propagate từ raw observation |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `id` | `obs_unique_123` |
| `sessionId` | `sess_xyz_456` |
| `timestamp` | `2026-06-10T14:30:00.000Z` |

**Kết quả mong đợi:**
- `result.id = "obs_unique_123"`
- `result.sessionId = "sess_xyz_456"`
- `result.timestamp = "2026-06-10T14:30:00.000Z"`

---

## TC-003-017: `agentId` được propagate từ raw

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-017 |
| **Tên** | agentId được copy vào compressed observation |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `agentId` | `claude-code-1` |

**Kết quả mong đợi:** `result.agentId = "claude-code-1"`

---

## TC-003-018: Narrative truncate tại 400 ký tự

| Trường | Giá trị |
|---|---|
| **ID** | TC-003-018 |
| **Tên** | Narrative bị cắt tại 400 ký tự với ellipsis |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- `toolInput` có path dài 500 ký tự
- `toolOutput` có content 200 ký tự

**Kết quả mong đợi:**
- `result.narrative.length ≤ 400`
- Kết thúc bằng `"…"` nếu bị truncate

---

## Tổng kết Module TC-003

| TC ID | Tên ngắn | Priority |
|---|---|---|
| TC-003-001 | hookType → type (5 types) | 🔴 P0 |
| TC-003-002 | toolName → type (19 cases) | 🔴 P0 |
| TC-003-003 | Unknown tool → other | 🟠 P1 |
| TC-003-004 | No toolName → other | 🟠 P1 |
| TC-003-005 | Title = toolName ≤ 80 | 🔴 P0 |
| TC-003-006 | Title truncate boundary | 🔴 P0 |
| TC-003-007 | Title = hookType fallback | 🟠 P1 |
| TC-003-008 | Extract files (6 keys) | 🟠 P1 |
| TC-003-009 | File dedup | 🟡 P2 |
| TC-003-010 | No toolInput → files=[] | 🔴 P0 |
| TC-003-011 | Path too long ignored | 🟡 P2 |
| TC-003-012 | importance = 5 | 🔴 P0 |
| TC-003-013 | confidence = 0.3 | 🟠 P1 |
| TC-003-014 | facts/concepts = [] | 🟡 P2 |
| TC-003-015 | Output has all fields | 🔴 P0 |
| TC-003-016 | Identity fields propagated | 🔴 P0 |
| TC-003-017 | agentId propagated | 🟠 P1 |
| TC-003-018 | Narrative truncate | 🟠 P1 |
