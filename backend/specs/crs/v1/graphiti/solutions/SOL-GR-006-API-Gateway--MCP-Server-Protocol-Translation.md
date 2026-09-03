# SOL-GR-006 — Solution: API Gateway & MCP Server Protocol Translation

| Field | Value |
|---|---|
| **Solution ID** | SOL-GR-006 |
| **CR** | CR-GR-006 |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `gateway/adapter/mcp` |

---

## 1. Phân tích

Expose Graphiti via 5 MCP tools: graph_add_episode, graph_search, graph_get_entity, graph_get_timeline, graph_set_ontology.

```go
// gateway/adapter/mcp/handlers_graph.go [NEW]
func (h *MCPHandlers) handleGraphAddEpisode(ctx context.Context, params map[string]any) (any, error) {
    conn := h.registry.Get("graphiti-ingestion")
    client := graphpb.NewGraphitiIngestionServiceClient(conn)
    res, err := client.IngestEpisode(ctx, &graphpb.EpisodeRequest{
        Content:   params["content"].(string),
        Source:    params["source"].(string),
        TenantId:  tenant.FromContext(ctx),
    })
    return map[string]any{"episode_id": res.EpisodeId}, err
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `gateway/adapter/mcp/handlers_graph.go` | NEW — 5 graph tools |
| `gateway/adapter/mcp/server.go` | MODIFY — register graph tools |

---

## 3. Acceptance Criteria

- [ ] 5 MCP tools functional
- [ ] All tools return structured JSON
- [ ] Tenant isolation enforced in all tools
