# TC-014: MCP Server — Test Cases

**Test Design tham chiếu:** [TD-014](../designs/TD-014-mcp-server.md)  
**Requirements tham chiếu:** [TR-014](../requirements/TR-014-mcp-server.md)  
**Module:** JSONRPC 2.0, tools/list, Tool Invocation  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-014-001: tools/list trả về đầy đủ tools

| Trường | Giá trị |
|---|---|
| **ID** | TC-014-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-014-MCP-001 |

**Điều kiện tiên quyết:** MCP server đang chạy qua stdio transport

**JSONRPC Request:**

| Trường | Giá trị |
|---|---|
| `jsonrpc` | `2.0` |
| `method` | `tools/list` |
| `id` | `1` |

**Các bước thực hiện:**
1. Start MCP server
2. Gửi JSONRPC request `tools/list`
3. Parse response
4. Kiểm tra danh sách tools

**Kết quả mong đợi:** Response `result.tools` chứa tất cả tools sau:
- `mem_observe`
- `mem_recall`
- `mem_remember`
- `mem_forget`
- `mem_search`
- `mem_status`
- `mem_profile`

---

## TC-014-002: Mỗi tool có JSON Schema hợp lệ

| Trường | Giá trị |
|---|---|
| **ID** | TC-014-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-014-MCP-002 |

**Điều kiện tiên quyết:** Response từ `tools/list` đã có

**Các bước thực hiện:**
1. Lấy list tools từ TC-014-001
2. Với mỗi tool, kiểm tra schema

**Kết quả mong đợi — mỗi tool phải có:**
- `name`: string không rỗng
- `description`: string không rỗng (mô tả có nghĩa)
- `inputSchema.type = "object"`
- `inputSchema.properties`: object không rỗng (ít nhất 1 property)

---

## TC-014-003: mem_recall trả về JSONRPC 2.0 response

| Trường | Giá trị |
|---|---|
| **ID** | TC-014-003 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-014-MCP-003 |

**Setup:** Session `sess_mcp` có 3 observations chứa "auth"

**JSONRPC Request:**

| Trường | Giá trị |
|---|---|
| `method` | `tools/call` |
| `params.name` | `mem_recall` |
| `params.arguments.query` | `auth` |
| `params.arguments.sessionId` | `sess_mcp` |

**Kết quả mong đợi:**
- `response.jsonrpc = "2.0"`
- `response.result.content` là array
- `response.result.content[0].type = "text"`
- `response.result.content[0].text` là string không rỗng (chứa recall results)
- Không có `response.error`

---

## TC-014-004: mem_remember lưu memory và trả về confirmation

| Trường | Giá trị |
|---|---|
| **ID** | TC-014-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-014-MCP-004 |

**JSONRPC Request:**

| Trường | Giá trị |
|---|---|
| `params.name` | `mem_remember` |
| `params.arguments.content` | `Auth system uses JWT with RS256` |
| `params.arguments.type` | `architecture` |
| `params.arguments.sessionId` | `sess_mcp` |

**Các bước thực hiện:**
1. Gọi `tools/call` với mem_remember
2. Kiểm tra response
3. Verify memory tồn tại trong KV (via `mem_recall`)

**Kết quả mong đợi:**
- `result.content[0].text` chứa confirmation message
- Memory xuất hiện khi `mem_recall` sau đó

---

## TC-014-005: Tool với invalid args → JSONRPC error

| Trường | Giá trị |
|---|---|
| **ID** | TC-014-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-014-MCP-005 |

**JSONRPC Request (thiếu sessionId):**

| Trường | Giá trị |
|---|---|
| `params.name` | `mem_recall` |
| `params.arguments` | `{}` (payload rỗng, thiếu sessionId) |

**Kết quả mong đợi:**
- Response có `error` field (JSONRPC error)
- `result` field không có
- `error.code` là integer error code

---

## TC-014-006: mem_observe ghi observation qua MCP

| Trường | Giá trị |
|---|---|
| **ID** | TC-014-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-014-MCP-006 |

**JSONRPC Request:**

| Trường | Giá trị |
|---|---|
| `params.name` | `mem_observe` |
| `params.arguments.sessionId` | `sess_mcp` |
| `params.arguments.hookType` | `post_tool_use` |
| `params.arguments.timestamp` | `2026-06-11T08:00:00.000Z` |
| `params.arguments.data.tool_name` | `edit_file` |

**Kết quả mong đợi:**
- `result.content[0].text` chứa `observationId`
- Observation tồn tại trong KV

---

## TC-014-007: Concurrent tool invocations không conflict

| Trường | Giá trị |
|---|---|
| **ID** | TC-014-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |

**Setup:** 5 concurrent mem_recall requests với cùng sessionId

**Các bước thực hiện:**
1. Dispatch 5 JSONRPC requests đồng thời
2. Chờ tất cả responses

**Kết quả mong đợi:**
- Tất cả 5 responses nhận được
- Không có error do concurrency
- Mỗi response có `result.content`

---

## Tổng kết TC-014

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-014-001 | tools/list có đủ tools | 🔴 P0 | Integration |
| TC-014-002 | JSON Schema hợp lệ | 🔴 P0 | Integration |
| TC-014-003 | mem_recall JSONRPC response | 🔴 P0 | Integration |
| TC-014-004 | mem_remember + confirmation | 🔴 P0 | Integration |
| TC-014-005 | Invalid args → JSONRPC error | 🔴 P0 | Integration |
| TC-014-006 | mem_observe via MCP | 🔴 P0 | Integration |
| TC-014-007 | Concurrent invocations | 🟠 P1 | Integration |
