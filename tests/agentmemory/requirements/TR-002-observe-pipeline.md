# TR-002: Observe Pipeline Test Requirements

**Module:** Observation Capture (observe.ts, compress-synthetic.ts, dedup.ts)  
**Nguồn:** SRS §3.2 (FR-OBS-001..004), Architecture §4.1, TDD §2.1..2.3  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho pipeline thu thập observations từ AI agent lifecycle hooks, bao gồm validation, deduplication, privacy redaction, image extraction và compression dispatch.

**Function chính:** `mem::observe` → `src/functions/observe.ts`

---

## TR-002-OBS-001 — Xử lý 12 loại hook events
🔴 P0 | `[INT]` | **FR-OBS-001**

**Given:** Worker đang chạy  
**When:** Mỗi trong 12 hook types được gửi:

| Hook Type | Required Data |
|---|---|
| `session_start` | sessionId, project, cwd |
| `prompt_submit` | sessionId, data.prompt |
| `pre_tool_use` | sessionId, data.tool_name, data.tool_input |
| `post_tool_use` | sessionId, data.tool_name, data.tool_output |
| `post_tool_failure` | sessionId, data.tool_name, data.error |
| `pre_compact` | sessionId |
| `subagent_start` | sessionId |
| `subagent_stop` | sessionId |
| `notification` | sessionId, data.content |
| `task_completed` | sessionId, data.result |
| `stop` | sessionId |
| `session_end` | sessionId |

**Then:** Mỗi hook được xử lý thành công, `observationId` được trả về

**Traceability:** FR-OBS-001, SRS §3.2

---

## TR-002-OBS-002 — Validation: required fields
🔴 P0 | `[UNIT]` | **FR-OBS-001**

**Given:** Payload thiếu required fields  
**When:** `mem::observe` được gọi  
**Then:**
- Thiếu `sessionId` → return `{success: false, error: "sessionId required"}`
- Thiếu `hookType` → return `{success: false, error: "hookType required"}`
- Thiếu `timestamp` → return `{success: false, error: "timestamp required"}`
- Không throw exception
- Không ghi bất kỳ data nào vào KV

**Traceability:** TDD §2.1 [1], SRS §3.2

---

## TR-002-OBS-003 — Deduplication trong 30s window
🔴 P0 | `[UNIT]` | **FR-OBS-002**

**Given:** `DedupMap` được khởi tạo với TTL 30 giây  
**When:** 2 observations với cùng `(sessionId, toolName, toolInput)` được gửi trong vòng 30 giây  
**Then:**
- Observation thứ nhất: được lưu, trả về `{observationId: "obs_xxx"}`
- Observation thứ hai: trả về `{deduplicated: true}`, KHÔNG được lưu vào KV
- `observationCount` của session không tăng cho duplicate

**Traceability:** FR-OBS-002, TDD §2.3

---

## TR-002-OBS-004 — Deduplication hash algorithm
🟠 P1 | `[UNIT]` | **FR-OBS-002**

**Given:** 2 observations  
**When:** Hash được compute  
**Then:** Hash = SHA-256(`${sessionId}:${toolName}:${JSON.stringify(toolInput)}`)

**Test cases (không được dedup):**
- Cùng toolName, khác sessionId → khác hash → cả 2 được lưu
- Cùng sessionId, khác toolInput → khác hash → cả 2 được lưu
- Cùng sessionId + toolName, `toolInput = undefined` vs `toolInput = {}` → khác hash

**Traceability:** FR-OBS-002, TDD §2.3

---

## TR-002-OBS-005 — Deduplication TTL expiry
🟠 P1 | `[UNIT]` | **FR-OBS-002**

**Given:** Observation A gửi lúc T=0  
**When:** Cùng observation gửi lại lúc T=31s (sau TTL 30s)  
**Then:** Observation thứ 2 KHÔNG bị dedup, được lưu bình thường

**Traceability:** FR-OBS-002, TDD §2.3

---

## TR-002-OBS-006 — Privacy redaction: API keys
🔴 P0 | `[UNIT]` | **FR-OBS-001**

**Given:** Hook payload chứa sensitive data:
```json
{
  "data": {
    "output": "ANTHROPIC_API_KEY=sk-ant-xxx123\nSuccess"
  }
}
```
**When:** `mem::observe` xử lý payload  
**Then:**
- `RawObservation.raw` được lưu với `[REDACTED]` thay cho value của API key
- Pattern được redact: `sk-ant-xxx123` → `[REDACTED]`
- Original payload KHÔNG được lưu

**Traceability:** FR-OBS-001, UR-010, Architecture §11.2

---

## TR-002-OBS-007 — Privacy redaction: Bearer tokens
🔴 P0 | `[UNIT]` | **FR-OBS-001**

**Given:** Payload chứa `"Authorization: Bearer eyJhbGci..."`  
**When:** `stripPrivateData()` chạy  
**Then:** Token bị replace bằng `[REDACTED]`

**Traceability:** Architecture §11.2, UR-010

---

## TR-002-OBS-008 — Privacy redaction: passwords
🔴 P0 | `[UNIT]`

**Given:** Payload chứa `"password": "my-secret-123"`  
**When:** `stripPrivateData()` chạy  
**Then:** Value bị replace bằng `[REDACTED]`

**Traceability:** Architecture §11.2

---

## TR-002-OBS-009 — Privacy redaction: không ảnh hưởng legitimate data
🟠 P1 | `[UNIT]`

**Given:** Payload với nội dung bình thường:
```json
{ "output": "File written successfully to src/auth.ts" }
```
**When:** `stripPrivateData()` chạy  
**Then:** Content được giữ nguyên, không bị modify

**Traceability:** SRS §8.2

---

## TR-002-OBS-010 — RawObservation được lưu đúng cấu trúc
🔴 P0 | `[INT]` | **FR-OBS-001**

**Given:** Hook `post_tool_use` với toolName="edit_file"  
**When:** Observation được lưu  
**Then:** `RawObservation` trong KV có đầy đủ:
```typescript
{
  id: string,           // "obs_<nanoid>"
  sessionId: string,
  timestamp: string,    // ISO
  hookType: "post_tool_use",
  toolName: "edit_file",
  toolInput: { path: "src/auth.ts" },
  toolOutput: { success: true },
  raw: object,          // sanitized original
  modality: "text"      // default
}
```

**Traceability:** SRS §6.1, TDD §2.1 [4]

---

## TR-002-OBS-011 — KV key schema cho observations
🟠 P1 | `[UNIT]`

**Given:** Observation với `sessionId = "sess_abc"` và `obsId = "obs_xyz"`  
**When:** Observation được lưu vào KV  
**Then:** KV key = `observations/sess_abc/obs_xyz`

**Traceability:** Architecture §9.1, TDD §10.1

---

## TR-002-OBS-012 — Image extraction: detect base64 PNG
🟠 P1 | `[UNIT]` | **FR-OBS-003**

**Given:** Hook payload chứa base64 PNG data (bắt đầu bằng `iVBORw0KGgo`)  
**When:** `mem::observe` xử lý  
**Then:**
- `raw.modality` = `"image"` hoặc `"mixed"`
- `raw.imageData` chứa base64 string
- Image được lưu vào disk với content-addressed path

**Traceability:** FR-OBS-003, TDD §2.1 [5]

---

## TR-002-OBS-013 — Image extraction: detect base64 JPEG
🟠 P1 | `[UNIT]` | **FR-OBS-003**

**Given:** Hook payload chứa JPEG data (bắt đầu bằng `/9j/`)  
**When:** `mem::observe` xử lý  
**Then:** Tương tự TR-002-OBS-012

**Traceability:** FR-OBS-003, TDD §2.1 [5]

---

## TR-002-OBS-014 — Observation types mapping
🟠 P1 | `[UNIT]` | **FR-OBS-004**

**Given:** Các hookType khác nhau  
**When:** Synthetic compression tạo CompressedObservation  
**Then:** ObservationType được map đúng:

| hookType | ObservationType |
|---|---|
| `post_tool_use` với tool "read_file" | `file_read` |
| `post_tool_use` với tool "write_file" | `file_write` |
| `post_tool_use` với tool "edit_file" | `file_edit` |
| `post_tool_use` với tool "bash" | `command_run` |
| `prompt_submit` | `conversation` |
| `post_tool_failure` | `error` |
| `session_start` | `other` |

**Traceability:** FR-OBS-004, TDD §2.2

---

## TR-002-OBS-015 — Keyed mutex ngăn race condition
🔴 P0 | `[UNIT]` | **FR-OBS-001**

**Given:** 10 observations được gửi đồng thời cho cùng sessionId  
**When:** Tất cả được xử lý song song  
**Then:**
- `observationCount` cuối cùng = 10 (không có lost update)
- Không có duplicate observations trong KV
- Không có panic/error

**Traceability:** TDD §3.3 Pattern, §2.1 [6], §10.3

---

## TR-002-OBS-016 — AgentId inheritance
🟠 P1 | `[INT]` | **FR-OBS-001**

**Given:** 
- Session tồn tại với `agentId = "claude-code-1"`
- Hook payload không có `agentId`
- `AGENT_ID` env var không set

**When:** Observation được tạo  
**Then:** `raw.agentId = "claude-code-1"` (kế thừa từ session)

**Traceability:** TDD §2.1 [8], SRS §3.10 FR-MULTI-005

---

## TR-002-OBS-017 — Synthetic compression path (default)
🔴 P0 | `[INT]` | **FR-COMPRESS-001**

**Given:** `AGENTMEMORY_AUTO_COMPRESS=false` (default)  
**When:** Observation được lưu  
**Then:**
- `buildSyntheticCompression(raw)` được gọi (không gọi LLM)
- `CompressedObservation` được tạo ngay lập tức
- Observation được index vào BM25
- Async vector indexing được trigger

**Traceability:** FR-COMPRESS-001, TDD §2.1 [14]

---

## TR-002-OBS-018 — LLM compression path (opt-in)
🟠 P1 | `[INT]` | **FR-COMPRESS-002**

**Given:** `AGENTMEMORY_AUTO_COMPRESS=true`, LLM provider được cấu hình  
**When:** Hook `post_tool_use` được gửi  
**Then:**
- `mem::compress` được trigger (async)
- LLM được gọi để tạo CompressedObservation
- Output có: `title`, `subtitle`, `facts[]`, `narrative`, `concepts[]`, `files[]`, `importance`

**Traceability:** FR-COMPRESS-002, TDD §2.1 [14]

---

## TR-002-OBS-019 — Stream broadcast sau observation
🟠 P1 | `[INT]`

**Given:** Viewer UI đang lắng nghe stream  
**When:** Observation được lưu thành công  
**Then:**
- `stream::set` được trigger (persisted stream, cho restart recovery)
- `stream::send` được trigger (viewer real-time, fire-and-forget)
- Viewer nhận event trong vòng <500ms

**Traceability:** TDD §2.1 [12], Architecture §12.1

---

## TR-002-OBS-020 — Session limit: warning không reject
🟠 P1 | `[INT]` | **FR-SESSION-003**

**Given:** Session đã đạt `MAX_OBS_PER_SESSION`  
**When:** Observation mới được gửi  
**Then:**
- Response trả về error message (không crash)
- Hook được xử lý xong (không block agent)
- Warning có thể được log

**Traceability:** FR-SESSION-003, SRS §3.1
