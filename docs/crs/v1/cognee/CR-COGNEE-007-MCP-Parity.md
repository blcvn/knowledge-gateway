# Change Request: CR-COGNEE-007 — MCP Server Parity

**CR ID:** CR-COGNEE-007  
**Component:** `services/vnp-gateway` (MCP adapter)  
**Priority:** High  
**Status:** Implemented  
**Reference:** Cognee PRD §6.3, SRS §3.4, URD UR-AGENT-02  
**Spec:** `references/cognee/specs/services/01-gateway.md` §5 (MCP Server)

---

## 1. Mô tả

Bổ sung đầy đủ **6 MCP Tools chuẩn** của Cognee vào MCP Server tại Gateway (port 8082) để đạt parity hoàn toàn với hệ sinh thái Cognee gốc, cho phép Claude Code, Cursor, và các AI Agent khác sử dụng Cognee MCP tools mà không cần cài Python.

---

## 2. Vấn đề hiện tại

`services/vnp-gateway/internal/adapter/mcp/tool_registry.go` hiện đăng ký **6 MCP tools** nhưng không khớp với MCP spec chuẩn của Cognee:

| Tool hiện tại (gateway) | Tool chuẩn Cognee MCP | Status |
|---|---|---|
| `cognee_add` | `cognify` (ingest + cognify) | ❌ Tên khác, flow khác |
| `cognee_search` | `search` | ⚠️ Tên khác |
| `memory_remember` | `save_interaction` | ❌ Khác hoàn toàn |
| `memory_recall` | — | ❌ Không có tương đương |
| `graphiti_ingest` | `list_data` | ❌ Không liên quan |
| `graphiti_search` | `delete_dataset` | ❌ Không liên quan |
| — | `cognify_status` | ❌ Thiếu hoàn toàn |

Kết quả: Plugin cognee-mcp cho Claude Code, Cursor không thể kết nối đến gateway vì tool names không match.

---

## 3. Thay đổi đề xuất

### 3.1. Service: `services/vnp-gateway`

**[MODIFY]** `internal/adapter/mcp/tool_registry.go`

Thêm 6 Cognee-standard tools (có thể coexist với existing tools):

```go
// Cognee Standard MCP Tools — 1:1 parity với cognee-mcp Python server
var cogneeStandardTools = []mcp.Tool{

    // Tool 1: cognify — ingest + process data into knowledge graph
    {
        Name:        "cognify",
        Description: "Transform data into a knowledge graph. Ingests content (text/URL/file) and builds semantic graph with entity extraction.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "data":         {Type: "string", Description: "Text content, URL, or file reference to ingest"},
                "dataset_name": {Type: "string", Description: "Target dataset name"},
                "node_sets":    {Type: "array",  Items: &mcp.Property{Type: "string"}},
            },
            Required: []string{"data"},
        },
        Handler: proxyCogneeAddThenCognify,  // chains: AddData → StartCognify
    },

    // Tool 2: search — query the knowledge graph
    {
        Name:        "search",
        Description: "Query the knowledge graph with multiple search strategies (semantic, graph traversal, keyword).",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query":        {Type: "string", Description: "Search query"},
                "query_type":   {Type: "string", Enum: []string{
                    "SIMILARITY","GRAPH_COMPLETION","GRAPH_SUMMARY","KEYWORD",
                    "CHUNKS","TEMPORAL","MULTI_HOP","HYBRID","FEELING_LUCKY",
                }, Default: "GRAPH_COMPLETION"},
                "dataset_name": {Type: "string"},
                "top_k":        {Type: "integer", Default: 10},
            },
            Required: []string{"query"},
        },
        Handler: proxyCogneeSearch,
    },

    // Tool 3: save_interaction — log user-agent interactions
    {
        Name:        "save_interaction",
        Description: "Log a user-agent interaction (query + answer) for feedback and learning purposes.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query":      {Type: "string"},
                "answer":     {Type: "string"},
                "session_id": {Type: "string"},
                "score":      {Type: "number", Description: "Feedback score -1.0 to 1.0"},
            },
            Required: []string{"query", "answer"},
        },
        Handler: proxySaveInteraction,  // → cognee-memory:9014 Remember(QAEntry)
    },

    // Tool 4: list_data — list datasets and data items
    {
        Name:        "list_data",
        Description: "List available datasets or items within a dataset.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "dataset_id": {Type: "string", Description: "If provided, lists items in this dataset"},
                "limit":      {Type: "integer", Default: 20},
            },
        },
        Handler: proxyListDatasets,  // → cognee-ingestion:9011 ListDatasets()
    },

    // Tool 5: delete_dataset — delete a dataset and all its data
    {
        Name:        "delete_dataset",
        Description: "Delete a dataset and all associated data, graph nodes, and vector embeddings.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "dataset_id":   {Type: "string"},
                "dataset_name": {Type: "string", Description: "Alternative to dataset_id"},
            },
        },
        Handler: proxyDeleteDataset,  // → cognee-ingestion:9011 DeleteDataset()
    },

    // Tool 6: cognify_status — check background cognify task status
    {
        Name:        "cognify_status",
        Description: "Check the status of a background cognify pipeline task.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "pipeline_run_id": {Type: "string"},
                "dataset_id":      {Type: "string"},
            },
        },
        Handler: proxyGetPipelineStatus,  // → cognee-cognify:9012 GetPipelineStatus()
    },
}
```

**[NEW]** `internal/adapter/mcp/handlers/cognify_handler.go`

```go
// proxyCogneeAddThenCognify — chains 2 gRPC calls:
// 1. cognee-ingestion:9011 AddData()
// 2. cognee-cognify:9012 StartCognify() with template=STANDARD
// Returns: {dataset_id, pipeline_run_id, status: "QUEUED"}
```

**[NEW]** `internal/adapter/mcp/handlers/save_interaction_handler.go`

```go
// proxySaveInteraction:
// → cognee-memory:9014 Remember(QAEntry{question, answer, session_id})
// if score provided: → cognee-search:9013 Search(FEEDBACK strategy)
```

### 3.2. MCP Transport Security

**[VERIFY]** Hiện tại MCP server đã có cấu hình trong `services/vnp-gateway/infra/config/config.go`:
```yaml
gateway:
  mcp:
    port: 8082
    enabled: true
```

Cần thêm:
```yaml
    dns_rebinding_protection: true         # [NEW]
    allowed_hosts: []                      # [NEW] whitelist (empty = allow all on localhost)
    cors_allow_origins: ["http://localhost:*"] # [NEW]
```

### 3.3. Transport Mode Support

**[MODIFY]** `internal/adapter/mcp/server.go`

```go
// Support both transport modes:
// - SSE (Server-Sent Events) — for Claude Desktop, Cursor, VS Code
// - stdio — for CLI integration (optional, via separate binary)
```

---

## 4. Backward Compatibility

Các tools hiện tại (`cognee_add`, `cognee_search`, `memory_remember`, `memory_recall`, `graphiti_ingest`, `graphiti_search`) **KHÔNG bị xóa**. Chỉ thêm 6 new tools theo Cognee standard naming.

---

## 5. Traceability

| Item | Ref |
|---|---|
| MCP Server port | `:8082` (đã có) |
| Transport | SSE (primary) + stdio (optional) |
| New tools | 6 tools theo Cognee MCP standard |
| gRPC calls | ingestion:9011, cognify:9012, search:9013, memory:9014 |
| Security | DNS rebinding protection, CORS config |
| Consumers | Claude Code (hooks), Cursor MCP, VS Code MCP |

---

## 6. Acceptance Criteria

- [x] MCP `tools/list` request trả về 6 Cognee-standard tools (cognify, search, save_interaction, list_data, delete_dataset, cognify_status).
- [x] Các tools mới được route qua MCP `forwardTool` thành công tới `kg-service`.
- [x] Existing tools (cognee_add, graphiti_ingest, v.v.) vẫn hoạt động bình thường (không bị xóa).

---

## 7. Implementation Notes

**Implemented in:** `gateway`

| File | Change |
|------|--------|
| `gateway/adapter/mcp/server.go` | `[MODIFY]` Thêm 6 tools (cognify, search, save_interaction, list_data, delete_dataset, cognify_status) vào MCP `registerTools()` |
