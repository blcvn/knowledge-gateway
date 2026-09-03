# SOL-COGNEE-007 — Solution: MCP Server Parity

| Field | Value |
|---|---|
| **Solution ID** | SOL-COGNEE-007 |
| **CR** | [CR-COGNEE-007](../../../../docs/crs/v1/cognee/CR-COGNEE-007*.md) |
| **TDD ref** | [02-cognee-services.md](../../../tdd/architecture/02-cognee-services.md) · [01-gateway.md](../../../tdd/architecture/01-gateway.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

---
## 1. Giải pháp

Expose full Cognee capabilities via 7 MCP tools in gateway/adapter/mcp/server.go.

### 1.1 MCP tools to add

```go
// gateway/adapter/mcp/handlers_cognee.go [NEW]
s.Register("cognee_ingest_text", h.handleCogneeIngestText,
    mcp.Schema{"dataset_id": "string", "content": "string", "metadata?": "object"})
s.Register("cognee_ingest_url", h.handleCogneeIngestURL,
    mcp.Schema{"dataset_id": "string", "url": "string"})
s.Register("cognee_cognify", h.handleCogneeCognify,
    mcp.Schema{"dataset_id": "string"})
s.Register("cognee_memify", h.handleCogneeMemify,
    mcp.Schema{"dataset_id": "string"})
s.Register("cognee_search", h.handleCogneeSearch,
    mcp.Schema{"query": "string", "dataset_id?": "string", "limit?": "integer"})
s.Register("cognee_get_graph", h.handleCogneeGetGraph,
    mcp.Schema{"dataset_id": "string", "entity_id?": "string"})
s.Register("cognee_list_datasets", h.handleCogneeListDatasets, mcp.Schema{})
```

## 2. Acceptance Criteria

- [ ] 7 MCP tools registered and functional
- [ ] All tools have JSON schema for validation
- [ ] Integration test for each tool
