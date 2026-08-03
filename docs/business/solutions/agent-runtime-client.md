# Solutions — Agent Runtime Client

> **Actor**: Agent Runtime Client (AI Agent / Automation System)  
> **Pain Points nguồn**: [agent-runtime-client.md](../painpoints/agent-runtime-client.md)  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Phân loại giải pháp

| Ký hiệu | Ý nghĩa |
|:---:|:---|
| ✅ **Đã có** | Sản phẩm đã hỗ trợ |
| 🔧 **Cần bổ sung** | Skeleton có, cần hoàn thiện |
| 🆕 **Đề xuất mới** | Chưa có, cần phát triển |

---

## PP-ARC-01 — MCP tool discovery không đủ để agent tự biết cách dùng

### ✅ Giải pháp đã có trong sản phẩm

**MCP tools/list** — tool discovery built-in:

```bash
# Mở session MCP:
curl -N -H "Authorization: Bearer kgsk_test_alpha_admin" \
  http://127.0.0.1:8082/v1/mcp/connect
# → session_id trong SSE event

# Discover tools:
curl -X POST -H "Authorization: Bearer kgsk_test_alpha_admin" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
  http://127.0.0.1:8082/v1/mcp/messages/{session_id}
```

**Current MCP tool set**:
- `kg_search` — semantic search
- `kg_search_rag` — RAG-oriented retrieval
- `kg_read_pattern` — execute named template
- `kg_list_domains` — list visible domains
- `kg_list_templates` — list templates per domain
- `kg_get_node` — fetch node by ID
- `kg_write_node` — create/update node
- `kg_check_access` — verify caller visibility
- `kg_integrity` — check projection integrity

**Realtime mode** cho freshness-sensitive agent retrieval:

```bash
# MCP: kg_get_node với mode=realtime
{
  "jsonrpc": "2.0", "id": 4,
  "method": "tools/call",
  "params": {
    "name": "kg_get_node",
    "arguments": { "id": "<node_id>", "app_id": "...", "mode": "realtime" }
  }
}
# realtime fallback về PostgreSQL nếu graph projection stale → agent luôn nhận data mới nhất
```

**Same identity model REST và MCP** — agent dùng cùng API key:

> "MCP uses the same `Authorization: Bearer <api_key>` header on the connect request. The same identity, ACL, and validation model applies underneath."

**Xem thêm**: [MCP Integration Guide](../../guides/mcp.md), [API Reference — MCP Transport](../../api/README.md#mcp-transport)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Rich tool descriptions với examples**

Mỗi MCP tool cần enhanced description để LLM agent tự biết cách dùng:

```json
{
  "name": "kg_read_pattern",
  "description": "Execute a named query template to retrieve knowledge from a specific domain. Use this when you need structured data retrieval by a known pattern (e.g., 'requirements by status', 'impact analysis'). Returns an array of nodes matching the template.",
  "inputSchema": {
    "type": "object",
    "required": ["domain_id", "template_name"],
    "properties": {
      "domain_id": { "type": "string", "description": "Domain to query. Use kg_list_domains to discover available domains.", "example": "payment-errors" },
      "template_name": { "type": "string", "description": "Active template name. Use kg_list_templates to discover available templates.", "example": "errors-by-severity" },
      "params": { "type": "object", "description": "Template parameters. Check template spec for required params.", "example": { "severity": "high" } },
      "mode": { "type": "string", "enum": ["realtime", "non-realtime"], "default": "non-realtime", "description": "Use 'realtime' when freshness is critical — falls back to source-of-truth if graph projection is stale." }
    }
  },
  "when_to_use": "Prefer kg_read_pattern over kg_search when you know the exact query pattern needed. Use kg_search for exploratory/semantic retrieval.",
  "output_description": "Array of node objects with id, node_type, attributes, and relationship_ids."
}
```

**2. MCP context initialization — tenant knowledge brief**

Khi agent mở session → kg-service tự động emit tenant context:

```json
{
  "event": "context",
  "data": {
    "tenant": "payment-team",
    "available_domains": [
      {
        "id": "payment-errors",
        "description": "Payment error codes and severity levels",
        "node_count": 245,
        "active_templates": ["errors-by-severity", "errors-by-code", "error-impact-chain"]
      }
    ],
    "recommended_tools": {
      "explore": "kg_list_domains → kg_list_templates → kg_read_pattern",
      "search": "kg_search (semantic) or kg_search_rag (answer-oriented)"
    }
  }
}
```

**3. Decision guide — when to use which tool**

Document tĩnh agent system prompt có thể reference:

```
kg_search: khi không biết exact data muốn tìm → semantic similarity
kg_search_rag: khi cần tổng hợp answer từ knowledge
kg_read_pattern: khi biết exact template cần dùng → structured retrieval
kg_get_node: khi có node_id cụ thể → direct fetch
kg_list_templates: khi cần biết templates nào available trong domain
kg_write_node: khi cần cập nhật knowledge từ agent
kg_integrity: khi agent cần verify projection health
```

---

## PP-ARC-02 — Semantic search thiếu relevance score

### ✅ Giải pháp đã có trong sản phẩm

**Semantic search** — search engine built-in:

```bash
POST /v1/kg/search/semantic
{
  "query": "QR payment timeout error handling",
  "domain_ids": ["payment-errors"],
  "top_k": 5
}
# → Results filtered by caller visibility, excluded deleted content
```

**RAG-oriented retrieval** — synthesized context:

```bash
POST /v1/kg/search/rag
{
  "query": "QR payment timeout error handling",
  "domain_ids": ["payment-errors"],
  "top_k": 5
}
# Same request shape, answer-synthesis oriented output
```

**Hybrid search** — semantic + full-text:

```bash
POST /v1/kg/search/hybrid
{
  "query": "QR timeout",
  "domain_ids": ["payment-errors"],
  "top_k": 5,
  "semantic_weight": 0.7,
  "fts_operator": "AND"
}
```

**Domain-scoped search** — narrower scope để tăng precision:

```bash
# Restrict domain_ids để giảm noise
POST /v1/kg/search/semantic
{ "query": "...", "domain_ids": ["payment-errors"], "top_k": 3 }
# Thay vì để domain_ids empty → search toàn bộ → nhiều noise
```

**Xem thêm**: [API Reference — Knowledge Read And Search](../../api/README.md#knowledge-read-and-search), [Integration Workflows — Read And Search](../../guides/integration.md#5-read-and-search-data)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Relevance score trong search response**

```json
POST /v1/kg/search/semantic → response hiện tại:
[ { "id": "node-123", "node_type": "ErrorCode", "attributes": {...} } ]

// Đề xuất thêm:
[
  {
    "id": "node-123",
    "node_type": "ErrorCode",
    "attributes": { "code": "E007", "severity": "high" },
    "relevance_score": 0.94,
    "match_reasons": ["matched: QR payment, timeout, error handling"],
    "snippet": "QR timeout error (E007) occurs when..."  // excerpt từ description
  }
]
```

**2. Min score threshold filter**

```bash
POST /v1/kg/search/semantic
{
  "query": "...",
  "domain_ids": ["payment-errors"],
  "top_k": 10,
  "min_score": 0.75  # Chỉ trả về kết quả có relevance >= 0.75
}
# Agent không cần filter manually → nhận đúng số lượng relevant results
```

**3. Include relationships trong search result**

```bash
POST /v1/kg/search/semantic
{
  "query": "QR timeout handling",
  "domain_ids": ["payment-errors"],
  "top_k": 5,
  "include_relationships": true,
  "relationship_depth": 1
}
# → Mỗi result node bao gồm cả connected nodes (1 hop)
# → Agent có thêm context mà không cần thêm API calls
```

---

## PP-ARC-03 — Không có token-budget-aware retrieval

### ✅ Giải pháp đã có trong sản phẩm

**Pagination** — giới hạn số lượng kết quả:

```bash
POST /v1/kg/search/semantic
{ "query": "...", "top_k": 3 }  # Giới hạn 3 kết quả → ít tokens hơn
```

**domain_ids filtering** — giảm scope → giảm noise và token:

```bash
POST /v1/kg/search/semantic
{ "query": "...", "domain_ids": ["payment-errors"] }
# Thay vì search toàn bộ tenant → narrow scope → fewer, more relevant results
```

**Template-based retrieval** — structured, predictable output size:

```bash
POST /v1/kg/read/template/{domain_id}/{template_name}
{ "params": { "severity": "high" } }
# Template định nghĩa sẵn projection → output schema predictable → agent biết token cost
```

**Xem thêm**: [API Reference — Pagination conventions](../../api/README.md#pagination)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Token budget parameter**

```bash
POST /v1/kg/search/semantic
{
  "query": "QR payment implementation guide",
  "domain_ids": ["payment"],
  "top_k": 20,
  "token_budget": 2000,      # ← tối đa 2000 tokens trong response
  "format": "summary"        # ← "full" | "summary" | "ids_only"
}
# → Service tự cắt/summarize results để fit trong token_budget
# → { "results": [...], "tokens_used": 1890, "truncated": true, "total_matches": 45 }
```

**2. Format modes**

```bash
# format=ids_only: chỉ trả về IDs → agent fetch chi tiết khi cần
{ "results": ["node-123", "node-456"], "total": 45 }

# format=summary: mỗi node chỉ có key fields
{ "results": [{ "id": "node-123", "title": "QR Timeout Error", "score": 0.94 }] }

# format=full (default): full node object với tất cả attributes
```

**3. Context package endpoint**

```bash
GET /v1/kg/context?query=QR+payment+implementation&domains=payment,payment-errors&token_budget=3000
# → Single API call trả về curated context package:
{
  "query": "QR payment implementation",
  "context": {
    "primary_nodes": [...],         # top 3 most relevant
    "related_relationships": [...], # key relationships
    "domain_summary": "...",        # domain overview
    "suggested_templates": [...]    # templates agent có thể dùng tiếp
  },
  "token_estimate": 2847,
  "sources": ["payment/Requirement#123", "payment-errors/ErrorCode#456"]
}
```

---

## PP-ARC-04 — REST và MCP không có cùng data model

### ✅ Giải pháp đã có trong sản phẩm

**Shared identity model** — REST và MCP dùng cùng API key:

> "The same identity, ACL, and validation model applies underneath."

**Same tool logic** — MCP tools gọi cùng service layer với REST:

> "Treat MCP as a thin wrapper over the REST and service layers, not as a separate data plane."

**Shared error model** — MCP errors là JSON-RPC format nhưng underlying domain errors giống nhau:

```
MCP error code -32000: invalid/expired session
MCP error code -32029: rate limit exceeded  
MCP error code -32601: unknown tool
```

**Xem thêm**: [MCP Guide — Shared Behavior With REST](../../guides/mcp.md#shared-behavior-with-rest)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Canonical node format — cùng JSON structure REST và MCP**

MCP tool results hiện tại có thể return dạng text content hoặc JSON. Chuẩn hóa:

```json
// MCP tool result (tools/call response) — canonical format:
{
  "jsonrpc": "2.0", "id": 3,
  "result": {
    "content": [
      {
        "type": "application/json",
        "data": {
          "nodes": [
            {
              "id": "node-123",
              "node_type": "ErrorCode",
              "domain_id": "payment-errors",
              "attributes": { "code": "E001", "severity": "high" },
              "relationship_ids": ["rel-456"],
              "projection_version": 42
            }
          ],
          "total": 1
        }
      }
    ]
  }
}

// REST response — same structure bên trong:
{
  "nodes": [ { "id": "node-123", "node_type": "ErrorCode", ... } ],
  "total": 1
}
```

**2. Unified SDK — abstract REST và MCP**

```python
# Cùng Python objects dù dùng REST hay MCP transport:
from kg_sdk import KGClient

# REST transport:
client = KGClient(api_key="...", transport="rest")

# MCP transport:
client = KGClient(api_key="...", transport="mcp")

# Same API:
results = client.search.semantic("QR timeout", domains=["payment-errors"], top_k=5)
# → List[KGNode] — same object model regardless of transport

node = client.nodes.get("node-123", mode="realtime")
# → KGNode object với .id, .node_type, .attributes, .relationships
```

---

## PP-ARC-05 — Không có caching — repeated queries hit backend mỗi lần

### ✅ Giải pháp đã có trong sản phẩm

**Embedding cache** — `EMBEDDING_CACHE_TTL_S`:

```bash
# environment.md: EMBEDDING_CACHE_TTL_S=300
# Cache embedding computations trong 5 phút
# → Repeated semantic searches với cùng query text → không recompute embedding
```

**Redis** — đã available trong stack:

> "Redis for bootstrap and runtime support" — đã có infrastructure cho caching.

**Read modes** — agent kiểm soát freshness:

```bash
mode=non-realtime  # Reads from graph projection (potentially cached)
mode=realtime      # Checks against PostgreSQL (fresher, slightly more expensive)
```

**Xem thêm**: [Deployment — Environment Variables](../../deployment/environment.md#shared-runtime-variables)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. HTTP cache headers cho read endpoints**

```http
GET /v1/ontology/domains/{domain_id}
→ Cache-Control: max-age=300, must-revalidate
→ ETag: "ontology-v42"

GET /v1/kg/read/templates?domain_id=payment
→ Cache-Control: max-age=60
→ ETag: "templates-payment-v7"
```

Agent có thể:
```http
GET /v1/kg/read/templates?domain_id=payment
If-None-Match: "templates-payment-v7"
→ 304 Not Modified (zero bandwidth) khi templates chưa thay đổi
```

**2. Server-side read-through cache qua Redis**

Extension của Redis hiện có để cache template query results:

```
Agent calls: POST /v1/kg/read/template/payment/errors-by-severity {"params": {"severity": "high"}}
→ Cache key: "template:payment:errors-by-severity:severity=high:tenant=payment-team"
→ TTL: 60s (configurable per domain)
→ Cache hit → response trong <5ms thay vì 45ms graph query
```

**3. Conditional fetch với projection version**

```bash
GET /v1/kg/read/nodes/{id}?since_version=40
# → Nếu node version vẫn là 40 → 304 Not Modified
# → Nếu node updated lên version 41+ → 200 với new data
# Agent chỉ re-process khi data thực sự changed
```

---

## Summary — Agent Runtime Client Solutions

| Pain Point | Đã có | Đề xuất mới | Priority |
|:---|:---:|:---:|:---:|
| PP-ARC-01: MCP tool discovery không đủ | ✅ tools/list + 9 MCP tools | 🆕 Rich descriptions + context init | 🔴 P0 |
| PP-ARC-02: Semantic search thiếu relevance score | ✅ semantic/rag/hybrid search endpoints | 🆕 relevance_score + min_score + relationships | 🔴 P0 |
| PP-ARC-03: Không có token-budget retrieval | ✅ top_k + domain filtering | 🆕 token_budget param + format modes | 🔴 P0 |
| PP-ARC-04: REST và MCP format khác | ✅ Shared identity + service layer | 🆕 Canonical JSON format + unified SDK | 🟠 P1 |
| PP-ARC-05: Không có caching | ✅ EMBEDDING_CACHE_TTL_S + Redis infra | 🆕 HTTP cache headers + read-through cache | 🟡 P2 |
