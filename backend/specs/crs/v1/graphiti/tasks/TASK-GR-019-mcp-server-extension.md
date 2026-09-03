# TASK-GR-019 — MCP Server: 31 Tools Extension

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-019 |
| **Wave** | 3 |
| **Component** | `services/mcp-server/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-006 §4 |
| **Priority** | High |
| **Depends On** | TASK-GR-017, TASK-GR-018 |
| **Estimated** | 4h |

**Trạng thái:** 🔄 Partial  
**Ghi chú:** Gateway MCP server has graphiti tools; full extension pending  
---

## Context

Extend MCP server từ 22 lên 31 tools (+9 graphiti tools). MCP server dùng JSON-RPC 2.0 over stdio (hoặc SSE). Mỗi tool là một handler gọi gateway REST API.

---

## New Tools (9)

| Tool Name | Description |
|-----------|-------------|
| `graphiti_ingest` | Ingest text/message/json episode |
| `graphiti_add_triplet` | Add structured fact (S→P→O) |
| `graphiti_search` | Hybrid search (fact retrieval) |
| `graphiti_search_advanced` | Search with custom config/recipe |
| `graphiti_get_episodes` | List recent episodes |
| `graphiti_remove_episode` | Delete episode |
| `graphiti_build_communities` | Trigger community detection |
| `graphiti_get_ontology` | Get current ontology for group |
| `graphiti_set_ontology` | Set/update ontology for group |

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `services/mcp-server/internal/tools/registry.go` |
| CREATE | `services/mcp-server/internal/tools/graphiti_tools.go` |
| MODIFY | `services/mcp-server/internal/config/config.go` |

---

## Implementation

### File 1: `services/mcp-server/internal/tools/graphiti_tools.go`

```go
package tools

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/vnp-memory/services/mcp-server/internal/mcp"
)

type GraphitiToolSet struct {
    gatewayBaseURL string
    httpClient     *http.Client
    apiKey         string  // for gateway JWT/API-key auth
}

func NewGraphitiToolSet(gatewayBaseURL, apiKey string) *GraphitiToolSet {
    return &GraphitiToolSet{
        gatewayBaseURL: gatewayBaseURL,
        apiKey:         apiKey,
        httpClient:     &http.Client{Timeout: 30 * time.Second},
    }
}

// Register adds all 9 graphiti tools to the MCP registry
func (g *GraphitiToolSet) Register(reg *Registry) {
    reg.AddTool(g.GraphitiIngest())
    reg.AddTool(g.GraphitiAddTriplet())
    reg.AddTool(g.GraphitiSearch())
    reg.AddTool(g.GraphitiSearchAdvanced())
    reg.AddTool(g.GraphitiGetEpisodes())
    reg.AddTool(g.GraphitiRemoveEpisode())
    reg.AddTool(g.GraphitiBuildCommunities())
    reg.AddTool(g.GraphitiGetOntology())
    reg.AddTool(g.GraphitiSetOntology())
}

func (g *GraphitiToolSet) GraphitiIngest() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_ingest",
        Description: "Ingest a text episode into the temporal knowledge graph. Extracts entities and relationships automatically.",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "body": map[string]any{
                    "type":        "string",
                    "description": "The text content to ingest (can be conversation, document, or structured JSON)",
                },
                "source": map[string]any{
                    "type":        "string",
                    "enum":        []string{"text", "message", "json", "fact_triple"},
                    "default":     "text",
                    "description": "Content type of the episode body",
                },
                "source_description": map[string]any{
                    "type":        "string",
                    "description": "Optional description of the data source (e.g. 'user conversation', 'email thread')",
                },
                "saga_id": map[string]any{
                    "type":        "string",
                    "description": "Optional saga/thread ID to associate this episode with a narrative arc",
                },
                "reference_time": map[string]any{
                    "type":        "string",
                    "description": "ISO8601 timestamp of when events occurred (defaults to now)",
                },
            },
            Required: []string{"body"},
        },
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            return g.callGateway(ctx, "POST", "/v1/graphiti/episodes", params)
        },
    }
}

func (g *GraphitiToolSet) GraphitiAddTriplet() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_add_triplet",
        Description: "Add a structured fact as a (subject, predicate, object) triplet without LLM extraction.",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "source_entity": map[string]any{"type": "string", "description": "Subject entity name"},
                "relation":      map[string]any{"type": "string", "description": "Relationship/predicate type (e.g. WORKS_AT, REPORTS_TO)"},
                "target_entity": map[string]any{"type": "string", "description": "Object entity name"},
                "fact":          map[string]any{"type": "string", "description": "Natural language statement of the fact"},
                "valid_at":      map[string]any{"type": "string", "description": "ISO8601 when fact became true (optional)"},
            },
            Required: []string{"source_entity", "relation", "target_entity"},
        },
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            return g.callGateway(ctx, "POST", "/v1/graphiti/triplets", params)
        },
    }
}

func (g *GraphitiToolSet) GraphitiSearch() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_search",
        Description: "Search the temporal knowledge graph using hybrid BM25 + vector search. Returns relevant facts and entity relationships.",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "query": map[string]any{"type": "string", "description": "Natural language search query"},
                "num_results": map[string]any{
                    "type":        "integer",
                    "default":     10,
                    "description": "Number of results to return (max 100)",
                },
                "center_node_uuid": map[string]any{
                    "type":        "string",
                    "description": "Optional: UUID of a central entity to bias results toward",
                },
            },
            Required: []string{"query"},
        },
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            return g.callGateway(ctx, "POST", "/v1/graphiti/search", params)
        },
    }
}

func (g *GraphitiToolSet) GraphitiSearchAdvanced() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_search_advanced",
        Description: "Advanced knowledge graph search with configurable strategies. Supports 6 pre-built recipes: edge_hybrid_rrf, edge_hybrid_mmr, edge_hybrid_cross_encoder, node_hybrid_rrf, community_hybrid_rrf, combined_cross_encoder.",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "query": map[string]any{"type": "string"},
                "search_config_name": map[string]any{
                    "type": "string",
                    "enum": []string{
                        "edge_hybrid_rrf", "edge_hybrid_mmr", "edge_hybrid_cross_encoder",
                        "node_hybrid_rrf", "community_hybrid_rrf", "combined_cross_encoder",
                    },
                    "default": "edge_hybrid_rrf",
                },
                "num_results": map[string]any{"type": "integer", "default": 10},
                "filters": map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "valid_at": map[string]any{"type": "string", "description": "ISO8601 for point-in-time query"},
                    },
                },
            },
            Required: []string{"query"},
        },
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            return g.callGateway(ctx, "POST", "/v1/graphiti/search/advanced", params)
        },
    }
}

func (g *GraphitiToolSet) GraphitiGetEpisodes() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_get_episodes",
        Description: "Retrieve the most recent episodes from the knowledge graph for a group.",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "last_n": map[string]any{"type": "integer", "default": 10, "description": "Number of episodes to retrieve"},
                "saga_id": map[string]any{"type": "string", "description": "Optional: filter by saga/thread ID"},
            },
        },
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            lastN := 10
            if n, ok := params["last_n"].(float64); ok { lastN = int(n) }
            sagaID, _ := params["saga_id"].(string)
            path := fmt.Sprintf("/v1/graphiti/episodes?last_n=%d", lastN)
            if sagaID != "" { path += "&saga_id=" + sagaID }
            return g.callGateway(ctx, "GET", path, nil)
        },
    }
}

func (g *GraphitiToolSet) GraphitiRemoveEpisode() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_remove_episode",
        Description: "Remove an episode and its associated mentions from the knowledge graph.",
        InputSchema: mcp.ToolInputSchema{
            Type:     "object",
            Properties: map[string]any{"episode_uuid": map[string]any{"type": "string"}},
            Required: []string{"episode_uuid"},
        },
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            uuid, _ := params["episode_uuid"].(string)
            return g.callGateway(ctx, "DELETE", "/v1/graphiti/episodes/"+uuid, nil)
        },
    }
}

func (g *GraphitiToolSet) GraphitiBuildCommunities() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_build_communities",
        Description: "Trigger community detection to cluster related entities. Useful after bulk ingestion.",
        InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]any{}},
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            return g.callGateway(ctx, "POST", "/v1/graphiti/admin/communities/build", params)
        },
    }
}

func (g *GraphitiToolSet) GraphitiGetOntology() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_get_ontology",
        Description: "Get the current entity/edge type ontology for the current group.",
        InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]any{}},
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            return g.callGateway(ctx, "GET", "/v1/graphiti/ontology/current", nil)
        },
    }
}

func (g *GraphitiToolSet) GraphitiSetOntology() mcp.Tool {
    return mcp.Tool{
        Name:        "graphiti_set_ontology",
        Description: "Define or update the ontology (entity types and edge types) for the current group. Use preset names for quick setup.",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "preset_name": map[string]any{
                    "type":        "string",
                    "enum":        []string{"hr", "crm", "software_project"},
                    "description": "Apply a pre-built domain ontology preset",
                },
                "entity_types": map[string]any{
                    "type":        "object",
                    "description": "Custom entity type definitions (overrides preset)",
                },
                "edge_types": map[string]any{
                    "type":        "object",
                    "description": "Custom edge type definitions (overrides preset)",
                },
            },
        },
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            if presetName, ok := params["preset_name"].(string); ok && presetName != "" {
                return g.callGateway(ctx, "POST", "/v1/graphiti/ontology/current/preset",
                    map[string]any{"preset_name": presetName})
            }
            return g.callGateway(ctx, "POST", "/v1/graphiti/ontology/current", params)
        },
    }
}

// callGateway sends an HTTP request to the gateway
func (g *GraphitiToolSet) callGateway(ctx context.Context, method, path string, body any) (any, error) {
    var bodyReader io.Reader
    if body != nil {
        data, err := json.Marshal(body)
        if err != nil { return nil, err }
        bodyReader = bytes.NewReader(data)
    }

    req, err := http.NewRequestWithContext(ctx, method, g.gatewayBaseURL+path, bodyReader)
    if err != nil { return nil, err }
    if body != nil { req.Header.Set("Content-Type", "application/json") }
    if g.apiKey != "" { req.Header.Set("Authorization", "Bearer "+g.apiKey) }

    resp, err := g.httpClient.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()

    var result any
    json.NewDecoder(resp.Body).Decode(&result)
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("gateway error %d: %v", resp.StatusCode, result)
    }
    return result, nil
}
```

### MODIFY: `services/mcp-server/internal/tools/registry.go`

Add graphiti tools initialization:

```go
// In SetupRegistry() or equivalent init function:
func SetupRegistry(cfg *config.Config) *Registry {
    reg := NewRegistry()

    // ... existing 22 tools ...

    // Register 9 new graphiti tools
    graphitiTools := NewGraphitiToolSet(cfg.GraphitiGatewayURL, cfg.GraphitiAPIKey)
    graphitiTools.Register(reg)

    return reg
}
```

---

## Verification

```bash
cd services/mcp-server
go build ./...

# Test tool discovery
go test ./internal/tools/... -run TestGraphitiToolsRegistered -v
```

**Expected test:**
```go
func TestGraphitiToolsRegistered(t *testing.T) {
    reg := SetupRegistry(testConfig)
    expectedTools := []string{
        "graphiti_ingest", "graphiti_add_triplet", "graphiti_search",
        "graphiti_search_advanced", "graphiti_get_episodes", "graphiti_remove_episode",
        "graphiti_build_communities", "graphiti_get_ontology", "graphiti_set_ontology",
    }
    for _, name := range expectedTools {
        if _, ok := reg.GetTool(name); !ok {
            t.Errorf("tool %s not registered", name)
        }
    }
    if reg.Count() != 31 { t.Errorf("expected 31 tools, got %d", reg.Count()) }
}
```
