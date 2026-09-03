# TR-014: MCP Server Test Requirements

**Module:** MCP Server (server.ts, tools-registry.ts, standalone.ts)  
**Nguồn:** SRS §7.2, §11, Architecture §3.2, §8.2, TDD §9, URD §3.9  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-014-MCP-001 — MCP: 53 tools registered
🔴 P0 | `[INT]` | **SRS §7.2**

**Given:** MCP server khởi động  
**When:** Tools list được query  
**Then:** Đúng 53 tools được registered (không hơn, không kém)

**Traceability:** SRS §7.2, SRS §11

---

## TR-014-MCP-002 — MCP: 6 resources
🟠 P1 | `[INT]`

**Given:** MCP server running  
**When:** Resources list được query  
**Then:** 6 resources:
- `agentmemory://sessions`
- `agentmemory://memories`
- `agentmemory://graph`
- `agentmemory://health`
- `agentmemory://config`
- `agentmemory://stats`

**Traceability:** TDD §9.1

---

## TR-014-MCP-003 — MCP: 3 prompts
🟡 P2 | `[INT]`

**Given:** MCP server running  
**When:** Prompts list được query  
**Then:**
- `recall-context`
- `summarize-session`
- `memory-audit`

**Traceability:** TDD §9.1

---

## TR-014-MCP-004 — MCP tool: memory_smart_search
🔴 P0 | `[INT]`

**Given:** Data trong memory  
**When:** `memory_smart_search({query: "auth", limit: 5})`  
**Then:**
- Proxy đến `mem::smart-search`
- Return JSON string trong content text
- Latency ≤ 50ms

**Traceability:** SRS §7.2, SRS §11

---

## TR-014-MCP-005 — MCP tool: memory_save
🔴 P0 | `[INT]`

**Given:** MCP client  
**When:** `memory_save({type: "architecture", title: "T", content: "C", concepts: ["a"]})`  
**Then:**
- Proxy đến `mem::remember`
- Memory được tạo trong KV
- Return `{memoryId: "mem_xxx"}` trong content

**Traceability:** SRS §7.2

---

## TR-014-MCP-006 — MCP tool: memory_governance_delete
🔴 P0 | `[INT]`

**Given:** Memory M tồn tại  
**When:** `memory_governance_delete({memoryId: M.id})`  
**Then:** Cascade delete xảy ra (như TR-013-GOV-001)

**Traceability:** FR-GOV-001, UR-016

---

## TR-014-MCP-007 — MCP standalone mode: 7 core tools
🔴 P0 | `[INT]` | **TDD §9.2**

**Given:** `AGENTMEMORY_URL` unreachable  
**When:** MCP package khởi động  
**Then:**
- Standalone mode active (in-memory KV)
- 7 core tools available (search, save, recall, forget, sessions, context, health)
- Không block agent workflow

**Traceability:** TDD §9.2, Architecture §3.2

---

## TR-014-MCP-008 — MCP proxy mode: forward to server
🟠 P1 | `[INT]`

**Given:** `AGENTMEMORY_URL=http://localhost:3111` reachable  
**When:** MCP tool được gọi  
**Then:**
- Request được proxy qua REST đến running server
- Full 53 tools available
- Auth header forwarded nếu AGENTMEMORY_SECRET set

**Traceability:** TDD §9.2, Architecture §8.2

---

## TR-014-MCP-009 — MCP transport: stdio
🟠 P1 | `[INT]`

**Given:** MCP package được chạy qua `npx @agentmemory/mcp`  
**When:** Transport configured  
**Then:** stdio transport được dùng (default)

**Traceability:** SRS §7.2, Architecture §8.2

---

## TR-014-MCP-010 — MCP: tool schema validation
🟠 P1 | `[UNIT]`

**Given:** MCP tool call với missing required args  
**When:** Request được processed  
**Then:** MCP protocol error được trả về (không crash server)

**Traceability:** TDD §9.1

---

## TR-014-MCP-011 — MCP: memory_slot_read/write/list/delete
🟠 P1 | `[INT]`

**Given:** Memory slot "preferences" tồn tại  
**When:** Slot operations qua MCP  
**Then:** CRUD operations hoạt động (xem TR-017)

**Traceability:** SRS §11 Memory Slots section

---

## TR-014-MCP-012 — MCP: memory_replay operations
🟡 P2 | `[INT]`

**Given:** Sessions có recorded observations  
**When:** `memory_replay_sessions` rồi `memory_replay_load`  
**Then:** Session playback data được trả về với đầy đủ events

**Traceability:** SRS §11

---

## TR-014-MCP-013 — MCP: tools đếm đúng sau update
🟡 P2 | `[UNIT]`

**Given:** `src/mcp/tools-registry.ts`  
**When:** Tool count được check  
**Then:** Số tools trong registry = 53 (consistency test)

**Traceability:** SRS §7.2

---

## TR-014-MCP-014 — MCP: AGENTMEMORY_URL configuration
🟠 P1 | `[UNIT]`

**Given:** `AGENTMEMORY_URL=http://remote-server:3111`  
**When:** MCP package khởi động  
**Then:** Proxy target = `http://remote-server:3111/agentmemory/`

**Traceability:** SRS §7.2, Architecture §8.2
