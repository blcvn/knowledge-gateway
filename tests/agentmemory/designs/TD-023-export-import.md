# TD-023: Export & Import Test Design

**Liên kết Requirements:** [TR-023-export-import.md](../requirements/TR-023-export-import.md)  
**Source:** `references/agentmemory/src/functions/export-import.ts`, `functions/migrate.ts`  
**Test file:** `tests/agentmemory/specs/export-import.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Export/Import cho phép backup, migration giữa các instances, và data recovery.

---

## 2. Test Cases

### Group A: Export

#### TC-001 — Export toàn bộ KV state sang JSON
**Requirement:** TR-023-EXP-001 | **Type:** integration | 🔴 P0

**Given:**
- 3 sessions, 30 observations, 10 memories, 5 graph nodes

**When:** `mem::export({format: "json"})` gọi  
**Then:**
- Output JSON có đủ: `sessions`, `observations`, `memories`, `graph`
- JSON parseable
- Data counts match

---

#### TC-002 — Export format có version marker
**Requirement:** TR-023-EXP-002 | **Type:** unit | 🔴 P0

**Given:** Export được thực hiện  
**When:** JSON parsed  
**Then:** `{version: "1.0", exportedAt: "...", data: {...}}`

---

#### TC-003 — Export không bao gồm expired memories (`forgetAfter` trong quá khứ)
**Type:** integration | 🟠 P1

**Given:** Memory A (expires yesterday), Memory B (no expiry)  
**When:** Export  
**Then:** JSON chứa Memory B, không chứa Memory A

---

#### TC-004 — Partial export: chỉ export theo sessionId
**Requirement:** TR-023-EXP-003 | **Type:** integration | 🟠 P1

**Given:** 3 sessions  
**When:** `mem::export({sessionId: "sess_A"})`  
**Then:** Chỉ data của sess_A trong output

---

#### TC-005 — Export không include sensitive data đã redacted
**Requirement:** TR-023-EXP-004 | **Type:** integration | 🔴 P0

**Given:** Observation với redacted API key `[REDACTED_SECRET]`  
**When:** Export  
**Then:** `[REDACTED_SECRET]` trong output (không reveal original), không có un-redacted secrets

---

### Group B: Import

#### TC-006 — Import từ valid export JSON: restore đầy đủ
**Requirement:** TR-023-EXP-005 | **Type:** integration | 🔴 P0

**Given:** Clean KV state, valid export JSON từ TC-001  
**When:** `mem::import({json: exportData})`  
**Then:**
- 3 sessions, 30 observations, 10 memories, 5 nodes được restore
- `{success: true, importedSessions: 3, importedObservations: 30, ...}`

---

#### TC-007 — Import idempotent: import 2 lần không tạo duplicates
**Requirement:** TR-023-EXP-006 | **Type:** integration | 🟠 P1

**Given:** Export data đã được import lần 1  
**When:** Import lần 2 với cùng data  
**Then:**
- Không có duplicate entries trong KV
- `{importedCount: 0, skippedCount: N}` cho all resources

---

#### TC-008 — Import với version mismatch: warning nhưng vẫn proceed
**Requirement:** TR-023-EXP-007 | **Type:** integration | 🟠 P1

**Given:** Export JSON với `version: "0.9"` (cũ hơn)  
**When:** Import  
**Then:**
- Import proceeds (không hard fail)
- Response có warning về version mismatch

---

#### TC-009 — Import với corrupt JSON: fail gracefully
**Type:** integration | 🔴 P0

**Given:** JSON string bị truncate (malformed)  
**When:** Import  
**Then:** `{success: false, error: "invalid JSON"}` — không partial import corrupt data

---

#### TC-010 — Import không overwrite newer data
**Requirement:** TR-023-EXP-008 | **Type:** integration | 🟠 P1

**Given:**
- Current KV: Memory M1 với `updatedAt = 2026-06-10T14:00:00Z`
- Import data: Memory M1 với `updatedAt = 2026-06-09T14:00:00Z` (cũ hơn)

**When:** Import  
**Then:** Existing M1 (newer) được giữ nguyên — import M1 bị skipped

---

### Group C: Migration

#### TC-011 — Migration tự động khi detect older data format
**Requirement:** TR-023-EXP-009 | **Type:** integration | 🟠 P1

**Given:** KV chứa data ở format cũ (thiếu `isLatest` field trên memories)  
**When:** Server startup hoặc `mem::migrate` gọi  
**Then:**
- Tất cả memories được backfill `isLatest = true`
- Migration log được tạo
- Không có data loss

---

#### TC-012 — Migration idempotent: chạy 2 lần an toàn
**Type:** integration | 🟠 P1

**Given:** Migration đã chạy lần 1 trên data  
**When:** Migration chạy lần 2  
**Then:** Data không bị corrupt, không có duplicate fields

---

#### TC-013 — Migration rollback: nếu fail, data ở trạng thái ban đầu
**Requirement:** TR-023-EXP-010 | **Type:** integration | 🔴 P0

**Given:** Migration script inject lỗi ở step 5/10  
**When:** Migration chạy  
**Then:**
- Migration dừng tại error
- Data trước step 5 được rollback về original state
- `{success: false, rolledBack: true}`

---

### Group D: JSONL Format

#### TC-014 — Export sang JSONL (streaming format)
**Requirement:** TR-023-EXP-011 | **Type:** integration | 🟡 P2

**Given:** 1000 observations  
**When:** `mem::export({format: "jsonl", outputPath: "export.jsonl"})`  
**Then:**
- File được tạo
- Mỗi line là valid JSON object
- Line count = 1000

---

#### TC-015 — Import từ JSONL: streaming read
**Type:** integration | 🟡 P2

**Given:** JSONL file với 1000 observations  
**When:** `mem::import({path: "export.jsonl", format: "jsonl"})`  
**Then:**
- 1000 observations được import
- Memory usage ổn định (streaming, không load toàn bộ)

---

## 3. Coverage Notes

- Export tests cần verify data integrity bằng cách round-trip: export → import → verify
- Migration tests cần seed "old format" data manually vào MockKV
- Version mismatch tests cần fixture export files với cả old và new versions
