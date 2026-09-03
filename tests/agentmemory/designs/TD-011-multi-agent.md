# TD-011: Multi-Agent Test Design

**Liên kết Requirements:** [TR-011-multi-agent.md](../requirements/TR-011-multi-agent.md)  
**Source:** `references/agentmemory/src/functions/leases.ts`, `signals.ts`  
**Test file:** `tests/agentmemory/specs/multi-agent.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Multi-agent coordination cho phép nhiều agents chia sẻ memory state và tránh conflict.

**Cơ chế chính:**
- **Leases:** Agent "acquire" lease trước khi ghi vào shared resource
- **Signals:** Pub/sub giữa agents trong cùng team
- **AgentId scoping:** Mỗi agent có observations riêng, nhưng memories có thể shared

---

## 2. Test Cases

### Group A: Lease Acquisition

#### TC-001 — Agent acquire lease thành công khi resource free
**Requirement:** TR-011-MAG-001 | **Type:** integration | 🔴 P0

**Given:** Resource `"shared-state-123"` không có lease nào  
**When:** Agent "agent-A" acquire lease với TTL 30s  
**Then:**
- `{success: true, leaseId: "..."}`
- KV `mem:leases` có entry cho resource

---

#### TC-002 — Second agent bị từ chối khi resource đã có lease
**Requirement:** TR-011-MAG-002 | **Type:** integration | 🔴 P0

**Given:** Agent A đang giữ lease cho resource "shared-123"  
**When:** Agent B cố acquire cùng resource  
**Then:**
- `{success: false, error: "resource locked"}`
- Không có lease thứ 2 được tạo

---

#### TC-003 — Lease hết hạn theo TTL, resource released tự động
**Requirement:** TR-011-MAG-003 | **Type:** unit (fake timers) | 🔴 P0

**Given:** Agent A acquire lease với TTL = 5s  
**When:** Thời gian giả lập qua 5s + 1ms  
**Then:**
- Agent B có thể acquire lease (resource được released)
- `isExpired(lease) = true`

---

#### TC-004 — Release lease trước TTL: resource available ngay
**Type:** integration | 🟠 P1

**Given:** Agent A đang giữ lease  
**When:** Agent A release lease explicitly  
**Then:** Agent B có thể acquire ngay (không cần chờ TTL)

---

#### TC-005 — Extend lease TTL trước khi expire
**Requirement:** TR-011-MAG-004 | **Type:** integration | 🟠 P1

**Given:** Agent A giữ lease với 2s còn lại  
**When:** Agent A extend lease thêm 30s  
**Then:**
- `{success: true}`
- Lease TTL được reset, expires 30s từ now

---

### Group B: Signal Pub/Sub

#### TC-006 — Agent A publish signal, Agent B nhận được
**Requirement:** TR-011-MAG-005 | **Type:** integration | 🟠 P1

**Given:** Agents A và B cùng teamId  
**When:**
1. Agent B subscribe signal type "file-changed"
2. Agent A publish signal `{type: "file-changed", file: "auth.ts"}`

**Then:** Agent B nhận signal với đúng payload

---

#### TC-007 — Signal chỉ delivery đến cùng teamId
**Type:** integration | 🔴 P0

**Given:**
- Team "team-A": Agents A, B
- Team "team-B": Agent C

**When:** Agent A publish signal  
**Then:** Agent B nhận, Agent C KHÔNG nhận

---

#### TC-008 — AgentId filtering: self-exclude
**Type:** unit | 🟡 P2

**Given:** Agent A publish signal với `excludeSelf: true`  
**When:** A cũng subscribe cùng signal type  
**Then:** A không nhận signal của chính mình

---

### Group C: AgentId Scoping

#### TC-009 — Observations mang `agentId` từ AGENT_ID env
**Requirement:** TR-011-MAG-007 | **Type:** integration | 🔴 P0

**Given:** `AGENT_ID=cursor-agent-1` env var  
**When:** Hook được observed  
**Then:** `observation.agentId = "cursor-agent-1"`

---

#### TC-010 — Recall với `agentScope="own"` chỉ trả về obs của agent đó
**Requirement:** TR-011-MAG-008 | **Type:** integration | 🟠 P1

**Given:**
- Obs từ Agent A (agentId="agent-A")
- Obs từ Agent B (agentId="agent-B")

**When:** `mem::recall({agentScope: "own", agentId: "agent-A"})`  
**Then:** Chỉ obs của agent-A được trả về

---

#### TC-011 — Recall với `agentScope="all"` trả về obs từ mọi agents
**Type:** integration | 🟠 P1

**Given:** Cùng setup như TC-010  
**When:** `mem::recall({agentScope: "all"})`  
**Then:** Obs từ cả agent-A và agent-B được trả về

---

### Group D: Shared Team Memory

#### TC-012 — Team memory được ghi vào `mem:team:{teamId}:shared`
**Requirement:** TR-011-MAG-009 | **Type:** integration | 🟠 P1

**Given:** Memory được lưu với `scope="team"`, `teamId="team-xyz"`  
**When:** Memory saved  
**Then:** Entry tồn tại tại `mem:team:team-xyz:shared`

---

#### TC-013 — Team members đều đọc được shared memory
**Type:** integration | 🟠 P1

**Given:** Agent A write shared memory cho team  
**When:** Agent B (cùng team) recall  
**Then:** Agent B nhận được shared memory

---

## 3. Coverage Notes

- Leases cần test TTL với fake timers
- Signals cần test in-process delivery (không qua network trong unit tests)
- Cross-agent isolation là regression test quan trọng cho production scenarios
