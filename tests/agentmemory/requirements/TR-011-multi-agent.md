# TR-011: Multi-Agent Coordination Test Requirements

**Module:** Multi-Agent (leases.ts, signals.ts, mesh.ts, team.ts)  
**Nguồn:** SRS §3.10 (FR-MULTI-001..005), Architecture §7, TDD §7.2, URD §3.6  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-011-MAC-001 — Shared memory pool: cross-agent access
🔴 P0 | `[E2E]` | **FR-MULTI-001**

**Given:**
- Agent A (Claude Code) lưu memory: "auth uses jose middleware"
- Agent B (Cursor) connect đến cùng agentmemory server

**When:** Agent B search: "jose authentication"  
**Then:** Memory từ Agent A xuất hiện trong Agent B's search results

**Traceability:** FR-MULTI-001, UR-024, UC-4

---

## TR-011-MAC-002 — Agent scoping: shared mode (default)
🔴 P0 | `[INT]` | **FR-MULTI-005**

**Given:** `AGENTMEMORY_AGENT_SCOPE=shared` (default), 2 agents  
**When:** Agent A lưu memory với agentId="claude-1"  
**Then:**
- Memory được tag với `agentId="claude-1"`
- Agent B có thể recall memory (không bị filter)
- Memory visible to all agents

**Traceability:** FR-MULTI-005, SRS §3.10

---

## TR-011-MAC-003 — Agent scoping: isolated mode
🟠 P1 | `[INT]` | **FR-MULTI-005**

**Given:** `AGENTMEMORY_AGENT_SCOPE=isolated`, 2 agents  
**When:** Agent A lưu memory, Agent B search  
**Then:**
- Agent B KHÔNG thấy Agent A's memories (filtered by agentId)
- Agent B chỉ thấy memories với `agentId = Agent B's ID` hoặc không có agentId

**Traceability:** FR-MULTI-005

---

## TR-011-MAC-004 — Lease acquisition: thành công
🔴 P0 | `[UNIT]` | **FR-MULTI-002**

**Given:** Không có lease active cho actionId "task-123"  
**When:** Agent A gọi `mem::lease-acquire({actionId: "task-123", agentId: "A", ttl: 300})`  
**Then:**
- Lease được tạo với `status: "active"`
- `acquiredAt` = now, `expiresAt` = now + 300s
- `agentId = "A"`
- Lease ID được trả về

**Traceability:** FR-MULTI-002, TDD §7.2

---

## TR-011-MAC-005 — Lease acquisition: conflict
🔴 P0 | `[UNIT]` | **FR-MULTI-002**

**Given:** Lease active cho actionId "task-123" (held by Agent A)  
**When:** Agent B gọi `mem::lease-acquire({actionId: "task-123", agentId: "B"})`  
**Then:**
- Return `null` hoặc `{conflict: true}`
- Không tạo second lease
- Không crash

**Traceability:** FR-MULTI-002, TDD §7.2

---

## TR-011-MAC-006 — Lease renewal
🟠 P1 | `[UNIT]` | **FR-MULTI-002**

**Given:** Active lease với `expiresAt = T+300s`  
**When:** Agent gọi `mem::lease-renew(leaseId)` lúc T+200s  
**Then:**
- `expiresAt` được extend thêm `ttl` giây
- `renewedAt` được set
- `status` vẫn là "active"

**Traceability:** TDD §7.2

---

## TR-011-MAC-007 — Lease expiry: auto-cleanup
🟠 P1 | `[INT]` | **FR-MULTI-002**

**Given:** Lease với `expiresAt = T` (đã qua)  
**When:** `mem::lease-acquire` cho cùng actionId được gọi sau T  
**Then:**
- Expired lease được recognize
- New lease có thể được acquire (expired lock không block)

**Traceability:** TDD §7.2 auto-expiry

---

## TR-011-MAC-008 — Signal send và receive
🔴 P0 | `[UNIT]` | **FR-MULTI-003**

**Given:** Agent A muốn notify Agent B  
**When:** `mem::signal-send({to: "agent-B", type: "handoff", content: "Task done, your turn"})`  
**Then:**
- Signal được lưu vào KV với unique ID
- `mem::signal-list({to: "agent-B"})` trả về signal
- Signal có: `type`, `content`, `from`, `to`, `createdAt`

**Traceability:** FR-MULTI-003, UR-025

---

## TR-011-MAC-009 — Signal types: 5 types
🟠 P1 | `[UNIT]` | **FR-MULTI-003**

**Given:** Signal được gửi  
**When:** Type được validated  
**Then:** Chỉ accept: `info`, `request`, `response`, `alert`, `handoff`

**Traceability:** FR-MULTI-003, SRS §3.10

---

## TR-011-MAC-010 — Signal threads
🟡 P2 | `[UNIT]` | **FR-MULTI-003**

**Given:** Signal A đã được gửi (signalId = "sig_A")  
**When:** Agent B reply: `mem::signal-send({replyTo: "sig_A", ...})`  
**Then:**
- Reply signal có `threadId = "sig_A"` (hoặc root thread ID)
- `replyTo = "sig_A"` preserved

**Traceability:** FR-MULTI-003

---

## TR-011-MAC-011 — Lease mutual exclusion: concurrent acquisition
🔴 P0 | `[UNIT]`

**Given:** 10 agents tất cả gọi `acquire-lease` cho cùng actionId đồng thời  
**When:** All requests processed  
**Then:**
- Đúng 1 agent nhận được lease (thắng race)
- 9 agents còn lại nhận conflict response
- Không có 2 active leases cùng lúc

**Traceability:** FR-MULTI-002, TDD §7.2 keyed mutex

---

## TR-011-MAC-012 — Project scope isolation
🔴 P0 | `[INT]` | **FR-MULTI-001**

**Given:**
- Memory M_A: project="project-A"
- Memory M_B: project="project-B"

**When:** `mem::smart-search({query: "auth", project: "project-A"})`  
**Then:**
- M_A xuất hiện trong results
- M_B KHÔNG xuất hiện

**Traceability:** FR-MULTI-001, UR-015

---

## TR-011-MAC-013 — Mesh sync: HMAC authentication
🟡 P2 | `[INT]` | **FR-MULTI-004**

**Given:** Node A muốn sync với Node B, HMAC secret configured  
**When:** `mem::mesh-sync({targetUrl: "http://node-b:3111", direction: "push"})`  
**Then:**
- Request đến Node B có `Authorization: Bearer {HMAC_signature}`
- Node B validates signature
- Sync chỉ proceed nếu auth pass

**Traceability:** FR-MULTI-004, Architecture §7.2

---

## TR-011-MAC-014 — Multi-agent handoff use case (UC-4)
🔴 P0 | `[E2E]` | **UC-4**

**Given:**
- Session 1 (Cursor): API implementation done, memory saved
- Session 2 (Claude Code): starts to write tests

**When:** Session 2 searches: "API implementation details"  
**Then:** Memories và summaries từ Session 1 xuất hiện trong Session 2's context

**Traceability:** UC-4, PRD §5

---

## TR-011-MAC-015 — AgentId từ AGENT_ID env var
🟠 P1 | `[UNIT]` | **FR-MULTI-005**

**Given:** `AGENT_ID=cursor-agent-1` env var  
**When:** Observation được tạo  
**Then:** `raw.agentId = "cursor-agent-1"` (từ env var, nếu session không có agentId)

**Traceability:** FR-MULTI-005, TDD §2.1 [8]
