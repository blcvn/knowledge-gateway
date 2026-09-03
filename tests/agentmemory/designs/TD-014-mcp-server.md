# TD-014: MCP Server Test Design

**Liên kết Requirements:** [TR-014-mcp-server.md](../requirements/TR-014-mcp-server.md)  
**Source:** `references/agentmemory/src/mcp/server.ts`, `mcp/standalone.ts`  
**Test file:** `tests/agentmemory/specs/mcp-server.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

MCP (Model Context Protocol) server expose agentmemory functions qua stdio/SSE transport cho AI IDE clients (Claude, Cursor, etc.).

---

## 2. Chiến lược kiểm thử

| Layer | Phương pháp |
|---|---|
| Unit | Mock MCP transport, test handler logic |
| Integration | Khởi động MCP server process, gửi JSONRPC messages |
| Contract | Verify response schema khớp MCP spec |

---

## 3. Test Cases

### Group A: Tool Registration

#### TC-001 — Server register đúng danh sách MCP tools
**Requirement:** TR-014-MCP-001 | **Type:** integration | 🔴 P0

**Given:** MCP server được khởi động  
**When:** Client gửi `tools/list` request  
**Then:** Response chứa ít nhất:
- `mem_observe`, `mem_recall`, `mem_remember`, `mem_forget`
- `mem_search`, `mem_status`, `mem_profile`

---

#### TC-002 — Mỗi tool có schema đầy đủ theo MCP spec
**Requirement:** TR-014-MCP-002 | **Type:** integration | 🔴 P0

**Given:** Tools list được trả về  
**When:** Schema của mỗi tool được inspect  
**Then:** Mỗi tool có:
- `name`: string
- `description`: string (không rỗng)
- `inputSchema`: JSON Schema object với `type: "object"`
- `inputSchema.properties`: object mô tả params

---

### Group B: Tool Invocation

#### TC-003 — `mem_recall` trả về kết quả hợp lệ
**Requirement:** TR-014-MCP-003 | **Type:** integration | 🔴 P0

**Given:** Server khởi động với data trong KV  
**When:** Client invoke `mem_recall` với `query: "auth"`, `sessionId: "sess_test"`  
**Then:**
- Response theo JSONRPC 2.0 format
- `result.content[0].type = "text"`
- `result.content[0].text` chứa recall results

---

#### TC-004 — `mem_remember` lưu memory và trả về confirmation
**Type:** integration | 🔴 P0

**Given:** Server running  
**When:** Client invoke `mem_remember` với `content: "Auth uses JWT"`  
**Then:**
- `result.content[0].text` chứa confirmation message
- Memory được lưu vào KV (verified qua `mem_recall`)

---

#### TC-005 — Tool invocation với invalid args trả về JSONRPC error
**Requirement:** TR-014-MCP-004 | **Type:** integration | 🔴 P0

**Given:** Server running  
**When:** `mem_recall` được gọi mà không có `sessionId` (required field)  
**Then:**
- Response có `error` field (không phải `result`)
- `error.code` là JSONRPC error code (ví dụ -32602 Invalid params)

---

### Group C: Transport

#### TC-006 — Stdio transport: stdin/stdout JSONRPC messages
**Requirement:** TR-014-MCP-005 | **Type:** integration | 🟠 P1

**Given:** Server được spawn với stdio transport  
**When:** JSONRPC message gửi qua stdin  
**Then:** Response đến qua stdout theo JSONRPC 2.0 format

---

#### TC-007 — SSE transport: server-sent events cho streaming
**Requirement:** TR-014-MCP-005 | **Type:** integration | 🟡 P2

**Given:** Server ở SSE mode trên port X  
**When:** Client connect `GET /sse`  
**Then:** Connection được establish, events stream qua SSE protocol

---

### Group D: Connection Lifecycle

#### TC-008 — Server graceful shutdown khi stdin closes
**Requirement:** TR-014-MCP-006 | **Type:** integration | 🟠 P1

**Given:** Server running với active connections  
**When:** stdin được close (client disconnects)  
**Then:**
- Server process exits gracefully (không crash)
- Exit code = 0

---

#### TC-009 — Concurrent tool invocations không conflict
**Requirement:** TR-014-MCP-007 | **Type:** integration | 🟠 P1

**Given:** Server running  
**When:** 5 concurrent `mem_recall` requests gửi đồng thời  
**Then:**
- Tất cả 5 responses nhận được
- Responses không bị interleave
- Không có error do concurrency

---

## 4. Coverage Notes

- MCP integration tests cần spawn real process (hoặc mock transport layer)
- Schema validation dùng JSON Schema validator (ajv)
- Stdio tests cần manage child process stdin/stdout pipes
