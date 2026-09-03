# TASK-CORE-010 — MCP Graph Tools (graphiti)

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-010 |
| **Wave** | 3 |
| **Solution** | [SOL-CORE-006](../solutions/SOL-CORE-006-MCP-Server.md) §2.1 |
| **Component** | `gateway/adapter/mcp/handlers_graph.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-CORE-009 |
| **Estimated** | 4h |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** gateway/adapter/mcp/tools/cognee: 6 Cognee tools registered; graphiti MCP tools not in mcp/tools dir
---

## Mục tiêu

Register 5 Graph Operation MCP tools cho graphiti.

---

## Công việc cụ thể

### `gateway/adapter/mcp/handlers_graph.go` [NEW]

```go
// graph_add_episode — ingest new episode to Graphiti
func (h *Handlers) handleGraphAddEpisode(ctx context.Context, params map[string]any) (any, error) {
    conn := h.registry.Get("graphiti-ingestion")
    client := graphpb.NewGraphitiIngestionServiceClient(conn)
    resp, err := client.IngestEpisode(ctx, &graphpb.EpisodeRequest{
        Content:  params["content"].(string),
        Source:   stringParam(params, "source"),
        TenantId: tenant.FromContext(ctx),
    })
    return map[string]string{"episode_id": resp.EpisodeId}, err
}

// graph_search — semantic + keyword search over episodes
func (h *Handlers) handleGraphSearch(ctx context.Context, params map[string]any) (any, error) {
    conn := h.registry.Get("graphiti-search")
    client := graphpb.NewGraphitiSearchServiceClient(conn)
    limit := int32(10)
    if v, ok := params["limit"]; ok { limit = int32(v.(float64)) }
    resp, err := client.Search(ctx, &graphpb.SearchRequest{
        Query: params["query"].(string), Limit: limit,
        TenantId: tenant.FromContext(ctx),
    })
    return resp, err
}

// graph_get_entity — get entity with neighbors
func (h *Handlers) handleGraphGetEntity(ctx context.Context, params map[string]any) (any, error) { ... }

// graph_get_timeline — temporal episode query
func (h *Handlers) handleGraphTimeline(ctx context.Context, params map[string]any) (any, error) {
    conn := h.registry.Get("graphiti-search")
    client := graphpb.NewGraphitiSearchServiceClient(conn)
    resp, err := client.SearchTemporal(ctx, &graphpb.TemporalSearchRequest{
        TenantId: tenant.FromContext(ctx),
        From: stringParam(params, "from"),
        To:   stringParam(params, "to"),
        Limit: 20,
    })
    return resp, err
}

// graph_set_ontology — define custom entity/edge types
func (h *Handlers) handleGraphSetOntology(ctx context.Context, params map[string]any) (any, error) { ... }
```

### Register in `server.go`

```go
s.Register("graph_add_episode", h.handleGraphAddEpisode,
    mcp.Schema{"content": "string", "source?": "string"})
s.Register("graph_search", h.handleGraphSearch,
    mcp.Schema{"query": "string", "limit?": "integer"})
s.Register("graph_get_entity", h.handleGraphGetEntity,
    mcp.Schema{"entity_id": "string"})
s.Register("graph_get_timeline", h.handleGraphTimeline,
    mcp.Schema{"from?": "string", "to?": "string", "limit?": "integer"})
s.Register("graph_set_ontology", h.handleGraphSetOntology,
    mcp.Schema{"entity_types": "array", "edge_types": "array"})
```

---

## Acceptance Criteria

- [ ] 5 graph tools registered
- [ ] `graph_add_episode` stores episode and returns episode_id
- [ ] `graph_get_timeline` supports from/to ISO8601 params
- [ ] All tools enforce tenant isolation

## Files

```
gateway/adapter/mcp/handlers_graph.go  [NEW]
gateway/adapter/mcp/server.go          [MODIFY — register 5 tools]
```
