# TASK-CE-002 — MCP Server: 6 Cognee Standard Tools (16 → 22)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-002 |
| **Wave** | 1 (Foundation) |
| **Component** | `gateway/internal/adapter/mcp/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-007 §2 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-CE-001 |
| **Estimated** | 5h |

---

## Context

Thêm **6 Cognee-standard MCP tools** vào MCP server hiện có (16 tools → 22 tools). Tạo package mới `gateway/internal/adapter/mcp/tools/cognee/` với 7 files. Cũng thêm DNS rebinding protection middleware.

Existing tools **không bị xóa hay thay đổi** — backward compatible hoàn toàn.

---

## Goal

- 6 tools mới: `cognify`, `search`, `save_interaction`, `list_data`, `delete_dataset`, `cognify_status`
- DNS rebinding protection middleware cho MCP server
- `TestMCPToolCount` → 22 tools, không trùng tên
- Existing 16 tools còn nguyên

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `gateway/internal/adapter/mcp/tools/cognee/registry.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/cognee/cognify_handler.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/cognee/search_handler.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/cognee/save_interaction_handler.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/cognee/list_data_handler.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/cognee/delete_dataset_handler.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/cognee/cognify_status_handler.go` |
| MODIFY | `gateway/internal/adapter/mcp/tool_registry.go` |
| MODIFY | `gateway/internal/adapter/mcp/server.go` |
| MODIFY | `gateway/internal/infra/config/config.go` |

---

## Implementation

### File 1: `gateway/internal/adapter/mcp/tools/cognee/registry.go`

```go
package cogneetools

import (
    "github.com/vnp-memory/gateway/internal/adapter/mcp"
    ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
    cognifypb   "github.com/vnp-memory/api/proto/cognee/cognify/v1"
    searchpb    "github.com/vnp-memory/api/proto/cognee/search/v1"
    memorypb    "github.com/vnp-memory/api/proto/memory/v1"
)

// Dependencies — all gRPC clients needed by cognee tools
type Dependencies struct {
    IngestionClient ingestionpb.IngestionServiceClient
    CognifyClient   cognifypb.CognifyServiceClient
    SearchClient    searchpb.SearchServiceClient
    MemoryClient    memorypb.MemoryServiceClient
}

// RegisterCogneeStandardTools registers 6 Cognee-standard MCP tools.
// These coexist with existing 16 tools — no tool is removed.
// After: total = 22 tools.
func RegisterCogneeStandardTools(reg *mcp.ToolRegistry, deps *Dependencies) {
    tools := []mcp.Tool{
        cognifyTool(deps),
        searchTool(deps),
        saveInteractionTool(deps),
        listDataTool(deps),
        deleteDatasetTool(deps),
        cognifyStatusTool(deps),
    }
    for _, t := range tools {
        reg.Register(t)
    }
}

func cognifyTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "cognify",
        Description: "Transform data into a knowledge graph. Ingests text/URL/file and builds semantic graph with entity extraction and relationship mapping. Use this to add knowledge to the memory system.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "data": {
                    Type:        "string",
                    Description: "Text content, URL, or file path to ingest into the knowledge graph",
                },
                "dataset_name": {
                    Type:        "string",
                    Description: "Target dataset name (created if not exists). Defaults to 'default'.",
                },
                "node_sets": {
                    Type:        "array",
                    Items:       &mcp.Property{Type: "string"},
                    Description: "Optional NodeSet tags for memory scoping (e.g. ['user_123', 'project_alpha'])",
                },
                "template": {
                    Type:        "string",
                    Enum:        []string{"STANDARD", "EMBED_ONLY", "FAST_INDEX", "TEMPORAL", "GRAPH_ONLY"},
                    Description: "Pipeline template. 'EMBED_ONLY' is fastest (no LLM). Default: 'STANDARD'.",
                },
            },
            Required: []string{"data"},
        },
        Handler: NewCognifyHandler(deps.IngestionClient, deps.CognifyClient).Handle,
    }
}

func searchTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "search",
        Description: "Query the knowledge graph with multiple search strategies. Returns semantically relevant information from ingested documents and memories.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query": {
                    Type:        "string",
                    Description: "Natural language search query",
                },
                "query_type": {
                    Type: "string",
                    Enum: []string{
                        "SIMILARITY", "GRAPH_COMPLETION", "GRAPH_SUMMARY",
                        "KEYWORD", "CHUNKS", "TEMPORAL", "MULTI_HOP",
                        "HYBRID", "FEELING_LUCKY",
                    },
                    Description: "Search strategy. Default: 'GRAPH_COMPLETION'.",
                },
                "dataset_name": {Type: "string", Description: "Restrict search to a specific dataset"},
                "node_sets":    {Type: "array", Items: &mcp.Property{Type: "string"}, Description: "Filter by NodeSet tags"},
                "top_k":        {Type: "integer", Default: 10},
                "save_interaction": {
                    Type:        "boolean",
                    Description: "Log this search for feedback tracking. Returns interaction_id.",
                },
            },
            Required: []string{"query"},
        },
        Handler: NewSearchHandler(deps.SearchClient).Handle,
    }
}

func saveInteractionTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "save_interaction",
        Description: "Log a user-agent interaction (query + answer) for memory and feedback. Use after important Q&A exchanges to improve future retrieval quality.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query":         {Type: "string", Description: "The user's question or query"},
                "answer":        {Type: "string", Description: "The agent's answer or response"},
                "session_id":    {Type: "string", Description: "Session identifier for grouping related interactions"},
                "score":         {Type: "number", Description: "Quality score (-1.0 to 1.0)"},
                "feedback_text": {Type: "string", Description: "Optional text comment about interaction quality"},
            },
            Required: []string{"query", "answer"},
        },
        Handler: NewSaveInteractionHandler(deps.SearchClient, deps.MemoryClient).Handle,
    }
}

func listDataTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "list_data",
        Description: "List available datasets or items within a specific dataset.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "dataset_id":   {Type: "string", Description: "If provided, lists entries within this dataset"},
                "dataset_name": {Type: "string", Description: "Alternative to dataset_id"},
                "limit":        {Type: "integer", Default: 20},
                "offset":       {Type: "integer", Default: 0},
            },
        },
        Handler: NewListDataHandler(deps.IngestionClient).Handle,
    }
}

func deleteDatasetTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "delete_dataset",
        Description: "Delete a dataset and ALL associated data (graph nodes in Neo4j, vectors in Qdrant, metadata in PostgreSQL). Irreversible.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "dataset_id":   {Type: "string", Description: "UUID of the dataset to delete"},
                "dataset_name": {Type: "string", Description: "Alternative: use dataset name"},
            },
        },
        Handler: NewDeleteDatasetHandler(deps.IngestionClient).Handle,
    }
}

func cognifyStatusTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "cognify_status",
        Description: "Check the status of a background cognify or memify pipeline job.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "pipeline_run_id": {Type: "string", Description: "Pipeline run ID returned by cognify"},
                "dataset_id":      {Type: "string", Description: "Alternative: check latest pipeline run for this dataset"},
            },
        },
        Handler: NewCognifyStatusHandler(deps.CognifyClient).Handle,
    }
}
```

### File 2: `gateway/internal/adapter/mcp/tools/cognee/cognify_handler.go`

```go
package cogneetools

import (
    "context"
    "fmt"
    "strings"

    ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
    cognifypb   "github.com/vnp-memory/api/proto/cognee/cognify/v1"
)

type CognifyHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
    cognifyClient   cognifypb.CognifyServiceClient
}

func NewCognifyHandler(ing ingestionpb.IngestionServiceClient, cog cognifypb.CognifyServiceClient) *CognifyHandler {
    return &CognifyHandler{ingestionClient: ing, cognifyClient: cog}
}

func (h *CognifyHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID := extractTenantFromContext(ctx)
    data     := getString(input, "data")
    dsName   := getStringOrDefault(input, "dataset_name", "default")
    nodeSets := toStringSlice(input["node_sets"])
    template := getStringOrDefault(input, "template", "STANDARD")

    if data == "" { return nil, fmt.Errorf("data is required") }

    // Step 1: AddData → cognee-ingestion
    addResp, err := h.ingestionClient.AddData(ctx, &ingestionpb.AddDataRequest{
        TenantId:    tenantID,
        DatasetName: dsName,
        Items: []*ingestionpb.DataItem{{
            Content:     data,
            ContentType: detectContentType(data),
        }},
        NodeSets: nodeSets,
    })
    if err != nil { return nil, fmt.Errorf("ingestion failed: %w", err) }

    // Step 2: StartCognify → cognee-cognify (returns quickly, runs in background)
    cognifyResp, err := h.cognifyClient.StartCognify(ctx, &cognifypb.StartCognifyRequest{
        DatasetId: addResp.DatasetId,
        TenantId:  tenantID,
        EntryIds:  addResp.EntryIds,
        NodeSets:  nodeSets,
        Template:  template,
    })
    if err != nil {
        // Partial success: ingestion succeeded, cognify queued but may have failed
        return map[string]any{
            "dataset_id": addResp.DatasetId,
            "status":     "INGESTED_NOT_PROCESSED",
            "error":      err.Error(),
        }, nil
    }

    return map[string]any{
        "dataset_id":      addResp.DatasetId,
        "pipeline_run_id": cognifyResp.PipelineRunId,
        "status":          cognifyResp.Status,
        "message":         fmt.Sprintf("Data ingested. Processing with template '%s' in background.", template),
    }, nil
}

func detectContentType(data string) string {
    if strings.HasPrefix(data, "http://") || strings.HasPrefix(data, "https://") { return "URL" }
    return "TEXT"
}

// Helpers
func extractTenantFromContext(ctx context.Context) string {
    if id, ok := ctx.Value("tenant_id").(string); ok && id != "" { return id }
    return "default"
}

func getString(m map[string]any, key string) string {
    if v, ok := m[key].(string); ok { return v }
    return ""
}

func getStringOrDefault(m map[string]any, key, def string) string {
    if v := getString(m, key); v != "" { return v }
    return def
}

func getIntOrDefault(m map[string]any, key string, def int) int {
    if v, ok := m[key].(float64); ok { return int(v) }
    return def
}

func toStringSlice(v any) []string {
    if v == nil { return nil }
    arr, ok := v.([]any)
    if !ok { return nil }
    result := make([]string, 0, len(arr))
    for _, item := range arr {
        if s, ok := item.(string); ok { result = append(result, s) }
    }
    return result
}

func getFloat64Ptr(m map[string]any, key string) *float64 {
    if v, ok := m[key].(float64); ok { return &v }
    return nil
}
```

### File 3: `gateway/internal/adapter/mcp/tools/cognee/search_handler.go`

```go
package cogneetools

import (
    "context"
    "fmt"

    searchpb "github.com/vnp-memory/api/proto/cognee/search/v1"
)

type SearchHandler struct {
    searchClient searchpb.SearchServiceClient
}

func NewSearchHandler(client searchpb.SearchServiceClient) *SearchHandler {
    return &SearchHandler{searchClient: client}
}

func (h *SearchHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID  := extractTenantFromContext(ctx)
    query     := getString(input, "query")
    queryType := getStringOrDefault(input, "query_type", "GRAPH_COMPLETION")
    dsName    := getString(input, "dataset_name")
    nodeSets  := toStringSlice(input["node_sets"])
    topK      := getIntOrDefault(input, "top_k", 10)
    saveInter := false
    if v, ok := input["save_interaction"].(bool); ok { saveInter = v }

    if query == "" { return nil, fmt.Errorf("query is required") }

    req := &searchpb.SearchRequest{
        Query:           query,
        Strategies:      []string{queryType},
        DatasetName:     dsName,
        TenantId:        tenantID,
        NodeSets:        nodeSets,
        TopK:            int32(topK),
        SaveInteraction: saveInter,
    }

    resp, err := h.searchClient.Search(ctx, req)
    if err != nil { return nil, fmt.Errorf("search failed: %w", err) }
    return resp, nil
}
```

### File 4: `gateway/internal/adapter/mcp/tools/cognee/save_interaction_handler.go`

```go
package cogneetools

import (
    "context"
    "fmt"

    searchpb "github.com/vnp-memory/api/proto/cognee/search/v1"
    memorypb "github.com/vnp-memory/api/proto/memory/v1"
)

type SaveInteractionHandler struct {
    searchClient searchpb.SearchServiceClient
    memoryClient memorypb.MemoryServiceClient
}

func NewSaveInteractionHandler(search searchpb.SearchServiceClient, memory memorypb.MemoryServiceClient) *SaveInteractionHandler {
    return &SaveInteractionHandler{searchClient: search, memoryClient: memory}
}

func (h *SaveInteractionHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID := extractTenantFromContext(ctx)
    query    := getString(input, "query")
    answer   := getString(input, "answer")
    score    := getFloat64Ptr(input, "score")
    fbText   := getString(input, "feedback_text")

    if query == "" || answer == "" { return nil, fmt.Errorf("query and answer are required") }

    // 1. Save Q&A pair to cognee-memory as MemoryFact
    fact := fmt.Sprintf("Q: %s\nA: %s", query, answer)
    _, err := h.memoryClient.Remember(ctx, &memorypb.RememberRequest{
        TenantId: tenantID,
        Content:  fact,
        Source:   "interaction",
        Type:     "qa",
    })
    if err != nil { return nil, fmt.Errorf("save memory fact: %w", err) }

    // 2. If score provided, submit as FEEDBACK to cognee-search (non-fatal)
    if score != nil {
        fbScore := *score
        h.searchClient.Search(ctx, &searchpb.SearchRequest{
            TenantId:      tenantID,
            Query:         fbText,
            Strategies:    []string{"FEEDBACK"},
            FeedbackScore: &fbScore,
            FeedbackText:  fbText,
        })
    }

    return map[string]any{
        "saved":     true,
        "fact_type": "qa",
        "message":   "Interaction logged to memory",
    }, nil
}
```

### File 5: `gateway/internal/adapter/mcp/tools/cognee/list_data_handler.go`

```go
package cogneetools

import (
    "context"
    "fmt"

    ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
)

type ListDataHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
}

func NewListDataHandler(client ingestionpb.IngestionServiceClient) *ListDataHandler {
    return &ListDataHandler{ingestionClient: client}
}

func (h *ListDataHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID  := extractTenantFromContext(ctx)
    datasetID := getString(input, "dataset_id")
    dsName    := getString(input, "dataset_name")
    limit     := getIntOrDefault(input, "limit", 20)
    offset    := getIntOrDefault(input, "offset", 0)

    if datasetID != "" || dsName != "" {
        resp, err := h.ingestionClient.ListDataEntries(ctx, &ingestionpb.ListDataEntriesRequest{
            TenantId:    tenantID,
            DatasetId:   datasetID,
            DatasetName: dsName,
            Limit:       int32(limit),
            Offset:      int32(offset),
        })
        if err != nil { return nil, fmt.Errorf("list data entries: %w", err) }
        return resp, nil
    }

    resp, err := h.ingestionClient.ListDatasets(ctx, &ingestionpb.ListDatasetsRequest{
        TenantId: tenantID,
        Limit:    int32(limit),
        Offset:   int32(offset),
    })
    if err != nil { return nil, fmt.Errorf("list datasets: %w", err) }
    return resp, nil
}
```

### File 6: `gateway/internal/adapter/mcp/tools/cognee/delete_dataset_handler.go`

```go
package cogneetools

import (
    "context"
    "fmt"

    ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
)

type DeleteDatasetHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
}

func NewDeleteDatasetHandler(client ingestionpb.IngestionServiceClient) *DeleteDatasetHandler {
    return &DeleteDatasetHandler{ingestionClient: client}
}

func (h *DeleteDatasetHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID  := extractTenantFromContext(ctx)
    datasetID := getString(input, "dataset_id")
    dsName    := getString(input, "dataset_name")

    if datasetID == "" && dsName == "" {
        return nil, fmt.Errorf("dataset_id or dataset_name is required")
    }

    resp, err := h.ingestionClient.DeleteDataset(ctx, &ingestionpb.DeleteDatasetRequest{
        TenantId:    tenantID,
        DatasetId:   datasetID,
        DatasetName: dsName,
    })
    if err != nil { return nil, fmt.Errorf("delete dataset: %w", err) }
    return map[string]any{
        "deleted":    resp.Deleted,
        "dataset_id": resp.DatasetId,
        "message":    "Dataset and all associated data deleted",
    }, nil
}
```

### File 7: `gateway/internal/adapter/mcp/tools/cognee/cognify_status_handler.go`

```go
package cogneetools

import (
    "context"
    "fmt"

    cognifypb "github.com/vnp-memory/api/proto/cognee/cognify/v1"
)

type CognifyStatusHandler struct {
    cognifyClient cognifypb.CognifyServiceClient
}

func NewCognifyStatusHandler(client cognifypb.CognifyServiceClient) *CognifyStatusHandler {
    return &CognifyStatusHandler{cognifyClient: client}
}

func (h *CognifyStatusHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    runID     := getString(input, "pipeline_run_id")
    datasetID := getString(input, "dataset_id")

    if runID == "" && datasetID == "" {
        return nil, fmt.Errorf("pipeline_run_id or dataset_id is required")
    }

    resp, err := h.cognifyClient.GetPipelineStatus(ctx, &cognifypb.GetPipelineStatusRequest{
        PipelineRunId: runID,
        DatasetId:     datasetID,
    })
    if err != nil { return nil, fmt.Errorf("get pipeline status: %w", err) }

    result := map[string]any{
        "pipeline_run_id": resp.PipelineRunId,
        "status":          resp.Status,
    }
    if resp.NewNodes > 0  { result["new_nodes"] = resp.NewNodes }
    if resp.NewEdges > 0  { result["new_edges"] = resp.NewEdges }
    if resp.Error != ""   { result["error"] = resp.Error }
    return result, nil
}
```

### MODIFY `tool_registry.go` — add RegisterCogneeStandardTools

```go
// In NewToolRegistry():
// ... existing 16 tools ...

// [NEW] 6 Cognee-standard tools
cogneetools.RegisterCogneeStandardTools(reg, &cogneetools.Dependencies{
    IngestionClient: deps.IngestionClient,
    CognifyClient:   deps.CognifyClient,
    SearchClient:    deps.CogneeSearchClient,
    MemoryClient:    deps.MemoryClient,
})
// Total: 22 tools
```

### MODIFY `server.go` + `config.go` — DNS rebinding protection

```go
// config.go: MCPConfig struct
type MCPConfig struct {
    Port                    int      `yaml:"port" env:"VNP_MEMORY_SERVER_MCP_PORT" default:"8082"`
    Enabled                 bool     `yaml:"enabled" default:"true"`
    DNSRebindingProtection  bool     `yaml:"dns_rebinding_protection" default:"true"`
    AllowedHosts            []string `yaml:"allowed_hosts"`
    CORSAllowOrigins        []string `yaml:"cors_allow_origins"`
}

// server.go: Add DNS rebinding middleware
func dnsRebindingProtection(cfg MCPConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !cfg.DNSRebindingProtection {
                next.ServeHTTP(w, r); return
            }
            host := r.Host
            if host == "" || isLocalhost(host) {
                next.ServeHTTP(w, r); return
            }
            http.Error(w, "DNS rebinding protection: rejected host "+host, http.StatusForbidden)
        })
    }
}

func isLocalhost(host string) bool {
    h := strings.Split(host, ":")[0]
    return h == "localhost" || h == "127.0.0.1" || h == "::1"
}
```

---

## Test

```go
// gateway/internal/adapter/mcp/tool_registry_test.go

func TestMCPToolCount_AfterCogneeParity(t *testing.T) {
    reg := NewToolRegistry(mockDeps)
    assert.Equal(t, 22, reg.Count())

    cogneeStandard := []string{"cognify", "search", "save_interaction", "list_data", "delete_dataset", "cognify_status"}
    for _, name := range cogneeStandard {
        assert.True(t, reg.HasTool(name), "missing tool: "+name)
    }

    existing := []string{"memory_store", "memory_recall", "memory_search", "ov_read_file", "graph_query"}
    for _, name := range existing {
        assert.True(t, reg.HasTool(name), "removed existing tool: "+name)
    }
}
```

## Verification

```bash
cd gateway
go build ./...
go test ./internal/adapter/mcp/... -run TestMCPToolCount -v
```

**Expected:**
- 22 tools total
- 6 new cognee tools present
- 16 existing tools still present
- No duplicate names
