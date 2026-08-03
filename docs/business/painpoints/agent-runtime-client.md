# Pain Points — Agent Runtime Client

> **Actor**: Agent Runtime Client  
> **Phạm vi**: AI Agent / Automation system — sử dụng kg-service qua REST hoặc MCP để retrieve context, execute tool calls, query knowledge cho downstream automation  
> **Loại**: AI/Automation integration pain points  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Tổng quan

Agent Runtime Client là **AI agent hoặc automation system** — không phải human user. Đây là actor "tương lai" của kg-service: khi AI agents cần grounded knowledge để hoạt động hiệu quả, kg-service là nơi cung cấp context đó.

Đây là actor khác biệt hoàn toàn với human users vì:
- **Không có human trong loop** để interpret ambiguous responses
- **Latency sensitive**: AI pipeline cần context nhanh — mỗi 100ms delay = degraded user experience
- **Reliability critical**: Nếu knowledge retrieval fail → agent hallucinate, give wrong answers
- **Token budget aware**: Agent có context window limit — response từ kg-service cần compact và relevant

---

## Pain Points chi tiết

### PP-ARC-01 — MCP tool discovery chưa đủ để agent tự biết cách dùng kg-service

**Mức độ**: 🔴 Critical  
**Tần suất**: Mỗi lần agent mới được integrate với kg-service  

**Mô tả**:  
SRS mô tả "FR-7 MCP Surface" với "tool discovery and invocation through MCP sessions." Nhưng để AI agent tự biết cách dùng kg-service qua MCP, cần nhiều hơn chỉ tool names:

```
Agent nhận: tools = ["kg_query", "kg_write", "kg_search"]

Agent cần biết để dùng đúng:
→ "kg_query" — parameters gì? template_name là mandatory? domain_id cũng cần?
→ Kết quả có structure gì? List? Object? Có pagination không?
→ Khi nào dùng "kg_query" vs "kg_search"? (graph traversal vs. semantic search)
→ Rate limits? Error codes có ý nghĩa gì?

Nếu MCP tool description không đủ → agent phải:
1. Trial-and-error → gây unexpected writes/reads
2. Hallucinate parameters → call tool với wrong args → error
3. Fall back: không dùng tool → mất knowledge grounding
```

**Hệ quả kinh doanh**:
- Agent integration tốn nhiều iteration để tune tool descriptions
- Agent errors khi dùng wrong tool/wrong params → bad AI output → user distrust
- RAG quality thấp vì agent không biết cách query knowledge đúng

**Giải pháp cần có**:
- Rich MCP tool descriptions với examples: `kg_query(template_name="requirements-by-status", params={"status": "approved"})` → returns list of Requirement nodes
- MCP context initialization: khi agent connect → kg-service tự cung cấp "Here are the available domains, templates, and data types for your tenant"
- Agent-optimized response format: compact JSON với only essential fields, không trailing metadata

---

### PP-ARC-02 — Semantic search kết quả không có confidence score hoặc relevance explanation

**Mức độ**: 🔴 Critical  
**Tần suất**: Mỗi lần agent dùng semantic search để retrieve context  

**Mô tả**:  
Agent dùng semantic search để tìm relevant knowledge cho RAG. Nhưng response hiện tại (theo SRS: "support semantic search") thiếu critical metadata cho agent decision-making:

```
Agent query: "What are the error handling requirements for QR payment?"

Response (hiện tại):
[
  { "id": "node-123", "type": "Requirement", "title": "QR Payment Error Handling" },
  { "id": "node-456", "type": "TestCase", "title": "TC-QR-001 Timeout Test" },
  { "id": "node-789", "type": "UserStory", "title": "QR Scan Failure Handling" }
]

Agent vấn đề:
→ Không biết node nào relevant nhất để include trong context
→ Nếu include tất cả → exceed token budget
→ Không có score → agent chọn ngẫu nhiên → inconsistent answers
→ Không có explanation → agent không thể tell user "tôi tìm thấy 3 kết quả liên quan"
```

**Hệ quả kinh doanh**:
- RAG quality inconsistent — đôi khi tốt, đôi khi miss relevant knowledge
- Agent phải retrieve nhiều hơn cần → waste token budget → expensive
- Agent không thể cite sources với confidence → less trustworthy output

**Giải pháp cần có**:
- Search response với `relevance_score` (0.0–1.0), `match_reason` ("matched on: error handling, QR payment, timeout")
- Search options: `max_results`, `min_score_threshold`, `include_relationships: true`
- Reranking: `POST /v1/kg/search/rerank` — input: query + candidates → output: sorted by relevance

---

### PP-ARC-03 — Không có context window-aware retrieval — agent có thể nhận response quá lớn

**Mức độ**: 🔴 Critical  
**Tần suất**: Khi domain knowledge phong phú, nhiều relationships  

**Mô tả**:  
Khi agent retrieve knowledge từ kg-service, response có thể quá lớn:

```
Agent query: "Get context for implementing payment QR flow"

kg-service response (naive):
- 45 Requirement nodes × 200 tokens each = 9,000 tokens
- 30 UserStory nodes × 150 tokens each = 4,500 tokens  
- 100 Relationship descriptions = 2,000 tokens
Total: ~15,500 tokens

Agent context window: 16,000 tokens (GPT-4 turbo)
→ Context cạn → agent không thể process user message + system prompt + tools
→ Agent phải truncate context → mất thông tin quan trọng
→ Or: agent không gọi kg-service để tránh token overflow
```

**Hệ quả kinh doanh**:
- Agent unreliable khi domain có nhiều data
- Agents có implicit token budget → viết workarounds → technical debt
- Cost tăng vì over-retrieve

**Giải pháp cần có**:
- Token-budget parameter: `GET /v1/kg/context?token_budget=2000&format=summary`
- Progressive disclosure: trả về top-k relevant với summary, agent có thể drill down
- Summarization mode: kg-service pre-summarize node clusters, trả về executive summary thay vì raw nodes

---

### PP-ARC-04 — REST và MCP không có cùng data model — agent phải handle two formats

**Mức độ**: 🟠 High  
**Tần suất**: Khi build agent system dùng cả REST và MCP  

**Mô tả**:  
SRS nói: "Agent consumers can use REST or MCP with the same identity model." Identity model giống nhau — nhưng data format của response có thể khác giữa:
- REST: `{"id": "node-123", "attributes": {"title": "..."}, "relationships": [...]}`
- MCP tool result: có thể là plain text, JSON string trong content block, hoặc structured data

Agent phải handle:
```python
# REST path
node = kg_client.get_node(id)
title = node["attributes"]["title"]

# MCP path
result = mcp_tool.call("kg_get_node", {"id": id})
# result.content is string? dict? depends on MCP server implementation
title = ???  # agent phải guess format
```

**Hệ quả kinh doanh**:
- Agent code complex hơn cần thiết
- Format mismatch bugs khó debug vì production không có clear error
- Switching từ REST sang MCP (hoặc ngược lại) → agent code phải refactor

**Giải pháp cần có**:
- Canonical knowledge format: same JSON structure dù qua REST hay MCP
- MCP tool results always return structured JSON with schema, không text blob
- SDK: `KGClient` abstract cả REST và MCP — same Python objects regardless of transport

---

### PP-ARC-05 — Không có caching layer cho agent — repeated queries hit backend mỗi lần

**Mức độ**: 🟡 Medium  
**Tần suất**: Khi agent process nhiều similar requests trong một session  

**Mô tả**:  
Trong một AI session, agent thường query cùng knowledge nhiều lần:
```
User session:
Message 1: "Explain QR payment error codes"
→ Agent: GET /v1/kg/templates/error-codes-by-domain?domain=payment → 50ms

Message 2: "Compare with Interbank payment error codes"  
→ Agent: GET /v1/kg/templates/error-codes-by-domain?domain=interbank → 50ms

Message 3: "Which QR error codes require manual intervention?"
→ Agent: GET /v1/kg/templates/error-codes-by-domain?domain=payment → 50ms (same query!)
```

Không có:
- Client-side cache hint: `Cache-Control: max-age=300` trong response
- Server-side read cache (Redis là available theo SRS nhưng không biết có cache reads không)
- ETags để agent check "data changed kể từ lần cuối tôi hỏi?"

**Hệ quả kinh doanh**:
- Latency tăng: mỗi message trong session → multiple backend queries
- Cost tăng nếu pricing by query
- Redis đã có nhưng không tận dụng cho read caching

**Giải pháp cần có**:
- HTTP cache headers trên read endpoints: `ETag`, `Cache-Control: max-age=60`
- Server-side read-through cache qua Redis cho template query results
- `GET /v1/kg/nodes/{id}?version={v}` — conditional fetch, trả về 304 nếu không đổi

---

## Ma trận Pain Points — Agent Runtime Client

| ID | Pain Point | Mức độ | Impact | Giải pháp cần có |
|:---|:---|:---:|:---|:---|
| PP-ARC-01 | MCP tool discovery không đủ cho agent tự dùng | 🔴 | Poor tool usage, hallucination | Rich tool descriptions + context init |
| PP-ARC-02 | Semantic search thiếu relevance score | 🔴 | Poor RAG quality, inconsistent answers | Score + match_reason in response |
| PP-ARC-03 | Không có token-budget-aware retrieval | 🔴 | Context overflow, agent unreliable | Token budget param + summarization |
| PP-ARC-04 | REST và MCP có khác biệt data format | 🟠 | Complex agent code, format bugs | Canonical format + unified SDK |
| PP-ARC-05 | Không có caching → repeated queries | 🟡 | Latency, cost | HTTP cache headers + Redis read cache |

---

## Tại sao Agent Runtime Client phải dùng kg-service

1. **Grounded knowledge, không hallucinate**: Agent có access vào kg-service → cite exact nodes, relationships, versions thay vì generate từ training data → factual accuracy cao hơn
2. **Tenant-isolated context**: Agent của "payment-team" chỉ thấy knowledge của payment-team — không leakage sang compliance hay lending → safe for multi-tenant AI systems
3. **Graph traversal = deep context**: Query "tất cả dependencies của UserStory US-045" → kg-service traverse graph → trả về full dependency tree — không thể làm với flat search
4. **Ontology-aware retrieval**: AI không chỉ search by text — biết "Requirement" và "UserStory" là different types → filter và retrieve đúng knowledge type cho từng query
5. **MCP standard**: Agent ecosystem (Claude, Cursor, custom agents) đều support MCP → zero integration effort nếu kg-service MCP surface tốt

> **Kết luận**: Agent Runtime Client là actor của tương lai — khi AI agents trở thành first-class users của knowledge systems. Giải quyết PP-ARC-01 và PP-ARC-02 sẽ unlock toàn bộ RAG/Agentic AI use cases cho platform.
