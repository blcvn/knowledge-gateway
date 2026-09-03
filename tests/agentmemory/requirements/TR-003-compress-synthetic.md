# TR-003: Synthetic Compression Test Requirements

**Module:** Compress Synthetic (compress-synthetic.ts)  
**Nguồn:** SRS §3.3 (FR-COMPRESS-001), TDD §2.2  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho quá trình tạo `CompressedObservation` từ `RawObservation` mà **không cần LLM**. Đây là path mặc định (zero-cost).

---

## TR-003-SYN-001 — Title generation: file_write
🔴 P0 | `[UNIT]` | **FR-COMPRESS-001**

**Given:** RawObservation với hookType=`post_tool_use`, toolName=`write_file`, toolInput=`{path: "src/auth.ts"}`  
**When:** `buildSyntheticCompression(raw)` được gọi  
**Then:** `CompressedObservation.title` = `"Wrote: src/auth.ts"`

**Traceability:** TDD §2.2

---

## TR-003-SYN-002 — Title generation: command_run
🔴 P0 | `[UNIT]`

**Given:** toolName=`bash`, toolInput=`{command: "npm test -- --coverage"}`  
**When:** Synthetic compression chạy  
**Then:** `title` = `"Ran: npm test -- --coverage"` (truncate tại 80 ký tự)

**Traceability:** TDD §2.2

---

## TR-003-SYN-003 — Title generation: prompt_submit
🟠 P1 | `[UNIT]`

**Given:** hookType=`prompt_submit`, `userPrompt = "Add JWT authentication to the API"`  
**When:** Synthetic compression chạy  
**Then:** `title` = `"User: Add JWT authentication to the API"` (truncate tại 100 ký tự)

**Traceability:** TDD §2.2

---

## TR-003-SYN-004 — Title truncation
🟡 P2 | `[UNIT]`

**Given:** prompt dài 200 ký tự  
**When:** Title được tạo cho `prompt_submit`  
**Then:** `title` bị truncate tại 100 ký tự, không bị cut giữa chữ

**Traceability:** TDD §2.2

---

## TR-003-SYN-005 — Importance scoring: session_start
🟠 P1 | `[UNIT]` | **FR-COMPRESS-001**

**Given:** hookType=`session_start`  
**When:** Synthetic compression tính importance  
**Then:** `importance = 0.9`

**Traceability:** TDD §2.2 Importance scoring

---

## TR-003-SYN-006 — Importance scoring: prompt_submit
🟠 P1 | `[UNIT]`

**Given:** hookType=`prompt_submit`  
**When:** Synthetic compression tính importance  
**Then:** `importance = 0.8`

**Traceability:** TDD §2.2

---

## TR-003-SYN-007 — Importance scoring: post_tool_failure
🟠 P1 | `[UNIT]`

**Given:** hookType=`post_tool_failure`  
**When:** Synthetic compression tính importance  
**Then:** `importance = 0.7`

**Traceability:** TDD §2.2

---

## TR-003-SYN-008 — Importance scoring: post_tool_use (write)
🟠 P1 | `[UNIT]`

**Given:** hookType=`post_tool_use`, toolName is write/edit operation  
**When:** Synthetic compression tính importance  
**Then:** `importance = 0.7`

**Traceability:** TDD §2.2

---

## TR-003-SYN-009 — Importance scoring: post_tool_use (read)
🟡 P2 | `[UNIT]`

**Given:** hookType=`post_tool_use`, toolName=`read_file`  
**When:** Synthetic compression tính importance  
**Then:** `importance = 0.4`

**Traceability:** TDD §2.2

---

## TR-003-SYN-010 — File path extraction
🔴 P0 | `[UNIT]` | **FR-COMPRESS-001**

**Given:** RawObservation với toolInput=`{path: "src/middleware/auth.ts"}`  
**When:** Synthetic compression extract files  
**Then:** `CompressedObservation.files = ["src/middleware/auth.ts"]`

**Traceability:** TDD §2.2

---

## TR-003-SYN-011 — Concept extraction từ file paths
🟠 P1 | `[UNIT]`

**Given:** toolInput=`{path: "src/middleware/auth.ts"}`  
**When:** Synthetic compression extract concepts  
**Then:** `concepts` bao gồm: `["middleware", "auth"]` (từ path segments)

**Traceability:** TDD §2.2

---

## TR-003-SYN-012 — CompressedObservation structure đầy đủ
🔴 P0 | `[UNIT]`

**Given:** Bất kỳ RawObservation hợp lệ  
**When:** `buildSyntheticCompression()` chạy  
**Then:** Return object đầy đủ:
```typescript
{
  id: string,              // = raw.id
  sessionId: string,       // = raw.sessionId
  timestamp: string,       // = raw.timestamp
  type: ObservationType,   // mapped từ hookType
  title: string,           // generated
  subtitle?: string,
  facts: string[],         // extracted
  narrative: string,       // generated summary
  concepts: string[],      // extracted
  files: string[],         // file paths
  importance: number,      // 0.0-1.0
  modality?: string        // from raw.modality
}
```

**Traceability:** SRS §6.1, TDD §2.2
