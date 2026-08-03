# Business Logic — AI Agent / MCP Client

> **Actor**: AI Agent / MCP Client  
> **Vai trò**: Hệ thống AI tự động (LLM agent, RAG pipeline) giao tiếp với KG Service qua MCP protocol để tìm kiếm và khai thác tri thức  
> **Phạm vi quyền**: Như App — theo API key của session MCP

---

## Nghiệp vụ 1: Kết nối MCP Session

### BL-AG-01: Thiết lập MCP Session

**Mô tả**: Agent kết nối và xác thực một lần duy nhất khi bắt đầu session — toàn bộ tool calls trong session dùng chung identity đó.

**Business Rules**:
- Xác thực xảy ra tại **connection level** (`GET /v1/mcp/connect`) — không phải tại từng tool call
- Mỗi connection sinh ra một `session_id` duy nhất
- Session dùng SSE (Server-Sent Events) — agent lắng nghe events trên stream
- Toàn bộ tool calls trong session mang cùng `(tenant_id, app_id)` identity
- Không có cơ chế "switch identity" giữa chừng trong session

**Luồng kết nối**:
```
GET /v1/mcp/connect
Authorization: Bearer {api_key}
→ SSE stream: { event: "session", session_id: "sess-xyz", tenant_id: "...", app_id: "..." }
```

**Sau khi connect**:
- Agent gửi tool calls qua: `POST /v1/mcp/messages/{session_id}`
- Format: JSON-RPC 2.0
- Connection mất → cần reconnect và lấy `session_id` mới

---

## Nghiệp vụ 2: Khám phá Capabilities

### BL-AG-02: Liệt kê MCP Tools

**Mô tả**: Agent khám phá các tools khả dụng trước khi gọi — không nên hardcode tool names.

```
{ "jsonrpc": "2.0", "method": "tools/list" }
→ { tools: [
    { name: "kg_search", description: "...", inputSchema: {...} },
    { name: "kg_read_pattern", description: "...", inputSchema: {...} },
    ...
  ]}
```

**9 tools khả dụng**:

| Tool | Mục đích |
|:---|:---|
| `kg_search` | Semantic search trong vector store |
| `kg_search_rag` | RAG retrieval — trả về structured context + citations |
| `kg_read_pattern` | Đọc theo named template từ graph |
| `kg_list_domains` | Liệt kê domains agent có thể access |
| `kg_list_templates` | Liệt kê templates của một domain |
| `kg_get_node` | Lấy node + relationships theo ID |
| `kg_write_node` | Ghi node mới vào KG |
| `kg_check_access` | Kiểm tra identity và visible domains |
| `kg_integrity` | Kiểm tra drift/inconsistency (cho agent giám sát) |

---

### BL-AG-03: Khám phá Domains và Templates

**Business Rules**:
- Agent nên gọi `kg_list_domains` trước để biết domain nào có thể search/read
- Agent nên gọi `kg_list_templates` với `domain_id` cụ thể để biết template nào khả dụng
- **Không hardcode tên template** trong agent code — template có thể thay đổi theo từng tenant

**Workflow khám phá**:
```
1. kg_check_access → biết mình là ai, thấy domain nào
2. kg_list_domains → lấy danh sách domain_id
3. kg_list_templates({ domain_id }) → lấy template_name + param_schema
4. → Giờ mới biết có thể gọi gì
```

---

## Nghiệp vụ 3: Tìm kiếm Tri thức

### BL-AG-04: Semantic Search

**Mô tả**: Agent tìm kiếm nodes liên quan đến câu query bằng vector similarity.

**Business Rules**:
- Kết quả tự động filter theo ACL của session — agent không thể truy cập data ngoài scope
- Nếu domain có status config: nodes invalid status bị loại bỏ khỏi kết quả
- `domain_ids` để giới hạn phạm vi search (recommended để tránh noise)
- `top_k` giới hạn số kết quả

```json
{ "name": "kg_search", "arguments": {
    "query": "payment timeout errors in checkout flow",
    "domain_ids": ["payment-errors"],
    "top_k": 5
}}
→ { results: [{ node_ref, score, content, domain_id, ... }] }
```

---

### BL-AG-05: RAG Retrieval

**Mô tả**: Agent lấy structured context để cung cấp cho LLM — bao gồm data có cấu trúc, citations, và conflict notes.

**Business Rules**:
- Kết hợp semantic search + graph traversal tự động
- Luôn trả về `disclaimer` — agent không được bỏ disclaimer khi trình bày với user
- `citations` phải được hiển thị cùng với câu trả lời (traceability)
- `conflict_notes` báo hiệu khi có data mâu thuẫn — agent nên cảnh báo user

```json
{ "name": "kg_search_rag", "arguments": {
    "query": "What are the tax rates for retail business with revenue 800M VND?"
}}
→ { answer_context: {
    structured_data: { ... },
    citations: [{ source, document_id, section }],
    conflict_notes: [],
    disclaimer: "Thông tin mang tính tham khảo..."
  }}
```

---

## Nghiệp vụ 4: Đọc Graph Pattern

### BL-AG-06: Đọc theo Named Template

**Mô tả**: Agent gọi named template để traverse graph theo pattern được Tenant Admin định nghĩa.

**Business Rules**:
- Phải biết `template_name` trước (dùng `kg_list_templates` để khám phá)
- Params phải đúng với `param_schema` của template
- ACL tự động inject — agent không thấy nodes ngoài scope
- Không có raw Cypher trong tool call

```json
{ "name": "kg_read_pattern", "arguments": {
    "domain_id": "payment-errors",
    "template_name": "errors-by-severity",
    "params": { "severity": "critical" }
}}
→ { results: [{ code, description, flow_id, requirement_title }] }
```

---

### BL-AG-07: Lấy Node Theo ID

**Business Rules**:
- `mode=realtime` có thể dùng nếu agent vừa write và cần đọc lại ngay
- Response bao gồm node + relationships liền kề
- 403 nếu node không trong visible scope

---

## Nghiệp vụ 5: Ghi Dữ liệu (nếu được phép)

### BL-AG-08: Ghi Node qua MCP

**Mô tả**: Agent có thể ghi node nếu được cấp quyền write (app type = `ingestion_producer` hoặc `hybrid`).

**Business Rules**:
- Validation giống REST write — required props, ontology, cross-domain rules
- 202 Accepted — async, không chắc visible ngay
- Agent nên kiểm tra `kg_check_access` trước để xác nhận có quyền write
- Không write vào domain không thuộc effective ontology

---

## Nghiệp vụ 6: Xử lý Lỗi trong MCP

### BL-AG-09: Error Handling

**Business Rules**:
- MCP error format: JSON-RPC `{ "error": { "code": -32000, "message": "..." } }` — **khác** với REST error envelope
- Agent phải handle các error code:

| Scenario | Behavior đề xuất |
|:---|:---|
| 401 / INVALID_API_KEY | Thông báo cần reconnect, không retry vô hạn |
| 403 / NO_READ_ACCESS | Thông báo không có quyền — không cố escalate |
| 404 / UNKNOWN_TEMPLATE | Gọi `kg_list_templates` để kiểm tra template hiện tại |
| 422 / VALIDATION_FAILED | Log chi tiết lỗi, thông báo data không hợp lệ |
| 503 / GRAPH_DB_TIMEOUT | Retry sau 1-2s, tối đa 3 lần |
| 429 / RATE_LIMIT_EXCEEDED | Backoff theo `X-RateLimit-Reset` header |

---

## Tóm tắt Business Rules — AI Agent / MCP Client

| ID | Rule |
|:---:|:---|
| **BR-AG-01** | Xác thực một lần tại connection — không per tool call |
| **BR-AG-02** | Khám phá tools/domains/templates trước khi gọi — không hardcode |
| **BR-AG-03** | Không gửi raw query/Cypher trong bất kỳ tool nào |
| **BR-AG-04** | ACL tự động filter — agent không cần và không thể bypass |
| **BR-AG-05** | Luôn hiển thị `disclaimer` và `citations` từ RAG kết quả |
| **BR-AG-06** | Không escalate permission — 403 là 403, không thử lại với params khác |
| **BR-AG-07** | Session mất → phải reconnect lấy `session_id` mới |
| **BR-AG-08** | Write chỉ khi app type có quyền — kiểm tra `kg_check_access` trước |
| **BR-AG-09** | Handle 503 với exponential backoff — graph/vector có thể tạm thời không khả dụng |
