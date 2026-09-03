# TR-010: Context Injection Test Requirements

**Module:** Context Building (context.ts)  
**Nguồn:** SRS §3.7 (FR-CTX-001..002), TDD §12.3, URD §3.3  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-010-CTX-001 — Context injection off by default
🔴 P0 | `[UNIT]` | **FR-CTX-002**

**Given:** `AGENTMEMORY_INJECT_CONTEXT=false` (default)  
**When:** Hook `session_start` hoặc `pre_tool_use` được gửi  
**Then:**
- Không có context được inject vào agent response
- Hook xử lý bình thường mà không thêm memory context
- Không có warning về chi phí token

**Traceability:** FR-CTX-002, SRS §3.7

---

## TR-010-CTX-002 — Token budget enforcement (default 2000)
🔴 P0 | `[UNIT]` | **FR-CTX-001**

**Given:** `TOKEN_BUDGET=2000`, 20 memories sẵn có  
**When:** `mem::context` được gọi để build context block  
**Then:**
- Tổng tokens của context blocks ≤ 2000
- Greedy fill: add blocks cho đến khi budget hết
- Recent/high-strength memories được ưu tiên

**Traceability:** FR-CTX-001, TDD §12.3

---

## TR-010-CTX-003 — Priority order: memories > summaries > observations
🟠 P1 | `[UNIT]` | **FR-CTX-001**

**Given:** Available: 5 recent memories, 3 session summaries, 10 observations  
**When:** Context được built với token budget 500  
**Then:**
- Memories được chọn trước (high strength + recent)
- Rồi session summaries (last 3)
- Cuối cùng là relevant observations (by search)

**Traceability:** TDD §12.3

---

## TR-010-CTX-004 — ContextBlock types
🟠 P1 | `[UNIT]` | **FR-CTX-001**

**Given:** Mixed context sources  
**When:** Context blocks được tạo  
**Then:** Mỗi block có `type: "summary" | "observation" | "memory"`

**Traceability:** FR-CTX-001, SRS §3.7

---

## TR-010-CTX-005 — Token estimation
🟡 P2 | `[UNIT]`

**Given:** Context block với 400 ký tự  
**When:** Token count được estimate  
**Then:** `estimatedTokens ≈ 400/4 = 100` (chars/4 approximation)

**Traceability:** TDD §12.3

---

## TR-010-CTX-006 — Context injection ~4000 chars
🟠 P1 | `[INT]` | **FR-CTX-002**

**Given:** `AGENTMEMORY_INJECT_CONTEXT=true`  
**When:** Hook `session_start` processed  
**Then:** Context injected ≤ 4000 ký tự vào agent turn

**Traceability:** FR-CTX-002, SRS §3.7

---

## TR-010-CTX-007 — Context injection: cross-session recall
🔴 P0 | `[E2E]` | **UC-1**

**Given:**
- Session 1: solved N+1 query bug (memory saved)
- Session 2: new session starts

**When:** `mem::context` được build cho Session 2  
**Then:** Bug fix memory xuất hiện trong context (cross-session recall)

**Traceability:** UC-1, UR-011, SRS §3.7

---

## TR-010-CTX-008 — Context: empty khi không có memories
🟡 P2 | `[UNIT]`

**Given:** Không có memories, summaries hoặc observations  
**When:** `mem::context` được gọi  
**Then:** Trả về empty context (`[]`), không throw error

**Traceability:** SRS §3.7

---

## TR-010-CTX-009 — Context tìm kiếm semantic relevance
🟠 P1 | `[INT]`

**Given:** Current session đang làm về "database optimization"  
**When:** Context được build  
**Then:** Memories về database/SQL xuất hiện trong context (semantic match)

**Traceability:** UR-012, UR-013

---

## TR-010-CTX-010 — Token budget configurable
🟡 P2 | `[UNIT]`

**Given:** `TOKEN_BUDGET=5000`  
**When:** Context được built  
**Then:** Context có thể đạt 5000 tokens (không bị cap ở 2000)

**Traceability:** SRS §9.3, FR-CTX-001
