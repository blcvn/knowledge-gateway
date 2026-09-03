# Solution: SOL-007 — MCP Server Parity (16 → 22 Tools)

**CR ID:** CR-COGNEE-007  
**Solution ID:** SOL-007  
**Priority:** High (Wave 1)  
**Architecture:** EXTEND `gateway/internal/adapter/mcp/` (16 → 22 MCP tools)

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §4.3`:
- MCP server trên port `:8082`, transport SSE + HTTP Streamable.
- Hiện có **16 tools** trong `gateway/internal/adapter/mcp/tool_registry.go`.
- Tool handler pattern: `mcp.Tool{Name, Description, InputSchema, Handler}`.
- Gateway kết nối đến cognee services qua `InProcessRegistry` (bufconn gRPC), không qua HTTP.
- Existing cognee tools: `cognee_add` (AddData), `cognee_search` (Search) — tên khác với Cognee MCP standard.

**Chiến lược:** Thêm **6 tools mới** theo Cognee MCP standard naming. Existing 16 tools giữ nguyên (backward compat). Total: 22 tools.

---

## 2. Giải pháp chi tiết

### 2.1. [NEW] `gateway/internal/adapter/mcp/tools/cognee/`

```
gateway/internal/adapter/mcp/tools/cognee/
├── registry.go          # RegisterCogneeStandardTools() — 6 tools
├── cognify_handler.go   # cognify tool: AddData + StartCognify chain
├── search_handler.go    # search tool: proxy SearchUseCase
├── save_interaction_handler.go  # save_interaction: memory + feedback
├── list_data_handler.go         # list_data: ListDatasets + ListEntries
├── delete_dataset_handler.go    # delete_dataset: cascading delete
└── cognify_status_handler.go    # cognify_status: PipelineRun lookup
```

### 2.2. [NEW] Tool Definitions — `tools/cognee/registry.go`

```go
// gateway/internal/adapter/mcp/tools/cognee/registry.go
package cogneetools

import "github.com/vnp-memory/gateway/internal/adapter/mcp"

// RegisterCogneeStandardTools registers 6 Cognee-standard MCP tools.
// These coexist with existing 16 tools (no tool is removed).
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
                    Description: "Pipeline template to use. 'EMBED_ONLY' is fastest (no LLM). Default: 'STANDARD'.",
                },
            },
            Required: []string{"data"},
        },
        Handler: deps.CognifyHandler.Handle,
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
                    Description: "Search strategy. GRAPH_COMPLETION for knowledge graph traversal. SIMILARITY for semantic similarity. Default: 'GRAPH_COMPLETION'.",
                },
                "dataset_name": {
                    Type:        "string",
                    Description: "Restrict search to a specific dataset",
                },
                "node_sets": {
                    Type:        "array",
                    Items:       &mcp.Property{Type: "string"},
                    Description: "Filter results to nodes tagged with these NodeSets",
                },
                "top_k": {
                    Type:    "integer",
                    Default: 10,
                },
                "save_interaction": {
                    Type:        "boolean",
                    Description: "Log this search for feedback tracking. Returns interaction_id in response.",
                },
            },
            Required: []string{"query"},
        },
        Handler: deps.SearchHandler.Handle,
    }
}

func saveInteractionTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "save_interaction",
        Description: "Log a user-agent interaction (query + answer) for memory and feedback. Use after important Q&A exchanges to improve future retrieval quality.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query": {
                    Type:        "string",
                    Description: "The user's question or query",
                },
                "answer": {
                    Type:        "string",
                    Description: "The agent's answer or response",
                },
                "session_id": {
                    Type:        "string",
                    Description: "Session identifier for grouping related interactions",
                },
                "score": {
                    Type:        "number",
                    Description: "Optional quality score (-1.0 negative to 1.0 positive)",
                },
                "feedback_text": {
                    Type:        "string",
                    Description: "Optional text comment about the interaction quality",
                },
            },
            Required: []string{"query", "answer"},
        },
        Handler: deps.SaveInteractionHandler.Handle,
    }
}

func listDataTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "list_data",
        Description: "List available datasets or items within a specific dataset.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "dataset_id": {
                    Type:        "string",
                    Description: "If provided, lists data entries within this dataset. If omitted, lists all datasets.",
                },
                "dataset_name": {
                    Type:        "string",
                    Description: "Alternative to dataset_id: use dataset name",
                },
                "limit": {
                    Type:    "integer",
                    Default: 20,
                },
                "offset": {
                    Type:    "integer",
                    Default: 0,
                },
            },
        },
        Handler: deps.ListDataHandler.Handle,
    }
}

func deleteDatasetTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "delete_dataset",
        Description: "Delete a dataset and ALL associated data: graph nodes/edges in Neo4j, vector embeddings in Qdrant, and metadata in PostgreSQL. This is irreversible.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "dataset_id": {
                    Type:        "string",
                    Description: "UUID of the dataset to delete",
                },
                "dataset_name": {
                    Type:        "string",
                    Description: "Alternative: use dataset name instead of ID",
                },
            },
        },
        Handler: deps.DeleteDatasetHandler.Handle,
    }
}

func cognifyStatusTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "cognify_status",
        Description: "Check the status of a background cognify or memify pipeline job.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "pipeline_run_id": {
                    Type:        "string",
                    Description: "Pipeline run ID returned by cognify or memify",
                },
                "dataset_id": {
                    Type:        "string",
                    Description: "Alternative: check latest pipeline run for this dataset",
                },
            },
        },
        Handler: deps.CognifyStatusHandler.Handle,
    }
}
```

### 2.3. [NEW] Cognify Handler — Chain AddData + StartCognify

```go
// gateway/internal/adapter/mcp/tools/cognee/cognify_handler.go

type CognifyHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient  // cognee-ingestion bufconn
    cognifyClient   cognifypb.CognifyServiceClient      // cognee-cognify bufconn
    tenantResolver  port.TenantResolver
}

func (h *CognifyHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID := extractTenantFromContext(ctx)
    data     := getString(input, "data")
    dsName   := getStringOrDefault(input, "dataset_name", "default")
    nodeSets := toStringSlice(input["node_sets"])
    template := getStringOrDefault(input, "template", "STANDARD")

    // Step 1: AddData → cognee-ingestion
    addResp, err := h.ingestionClient.AddData(ctx, &ingestionpb.AddDataRequest{
        TenantId:    tenantID,
        DatasetName: dsName,
        Items: []*ingestionpb.DataItem{{
            Content: data,
            Type:    detectContentType(data),
        }},
        NodeSets: nodeSets,
    })
    if err != nil { return nil, fmt.Errorf("ingestion failed: %w", err) }

    // Step 2: StartCognify → cognee-cognify (background)
    cognifyResp, err := h.cognifyClient.StartCognify(ctx, &cognifypb.StartCognifyRequest{
        DatasetId: addResp.DatasetId,
        TenantId:  tenantID,
        EntryIds:  addResp.EntryIds,
        NodeSets:  nodeSets,
        Template:  template,
    })
    if err != nil {
        // Ingestion succeeded, cognify queued but failed? Return partial success
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

// detectContentType: URL → URL, else TEXT
func detectContentType(data string) string {
    if strings.HasPrefix(data, "http://") || strings.HasPrefix(data, "https://") { return "URL" }
    return "TEXT"
}
```

### 2.4. [NEW] SaveInteraction Handler

```go
// gateway/internal/adapter/mcp/tools/cognee/save_interaction_handler.go

type SaveInteractionHandler struct {
    searchClient cognee_search_pb.SearchServiceClient  // cognee-search
    memoryClient memory_pb.MemoryServiceClient          // cognee-memory (via memory-service)
}

func (h *SaveInteractionHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID := extractTenantFromContext(ctx)
    query     := getString(input, "query")
    answer    := getString(input, "answer")
    sessionID := getString(input, "session_id")
    score     := getFloat64Ptr(input, "score")
    fbText    := getString(input, "feedback_text")

    // 1. Save to cognee-memory as MemoryFact (QA pair)
    fact := fmt.Sprintf("Q: %s\nA: %s", query, answer)
    _, err := h.memoryClient.Remember(ctx, &memorypb.RememberRequest{
        TenantId: tenantID,
        Content:  fact,
        Source:   "interaction",
        Type:     "qa",
    })
    if err != nil { return nil, fmt.Errorf("save memory fact: %w", err) }

    // 2. If score provided: also submit as FEEDBACK to cognee-search
    if score != nil {
        _, err = h.searchClient.Search(ctx, &searchpb.SearchRequest{
            TenantId:      tenantID,
            Query:         fbText,
            Strategies:    []string{"FEEDBACK"},
            FeedbackScore: *score,
            FeedbackText:  fbText,
            SessionId:     sessionID,
        })
        // Non-fatal: feedback failure doesn't block save_interaction
        _ = err
    }

    return map[string]any{
        "saved":     true,
        "fact_type": "qa",
        "message":   "Interaction logged to memory",
    }, nil
}
```

### 2.5. [NEW] List/Delete/Status Handlers

```go
// gateway/internal/adapter/mcp/tools/cognee/list_data_handler.go

type ListDataHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
}

func (h *ListDataHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID  := extractTenantFromContext(ctx)
    datasetID := getString(input, "dataset_id")
    dsName    := getString(input, "dataset_name")
    limit     := getIntOrDefault(input, "limit", 20)
    offset    := getIntOrDefault(input, "offset", 0)

    if datasetID != "" || dsName != "" {
        // List entries within dataset
        resp, err := h.ingestionClient.ListDataEntries(ctx, &ingestionpb.ListDataEntriesRequest{
            TenantId:  tenantID,
            DatasetId: datasetID,
            DatasetName: dsName,
            Limit:     int32(limit),
            Offset:    int32(offset),
        })
        if err != nil { return nil, err }
        return resp, nil
    }

    // List all datasets
    resp, err := h.ingestionClient.ListDatasets(ctx, &ingestionpb.ListDatasetsRequest{
        TenantId: tenantID,
        Limit:    int32(limit),
        Offset:   int32(offset),
    })
    if err != nil { return nil, err }
    return resp, nil
}
```

```go
// gateway/internal/adapter/mcp/tools/cognee/delete_dataset_handler.go

type DeleteDatasetHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
}

func (h *DeleteDatasetHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    tenantID  := extractTenantFromContext(ctx)
    datasetID := getString(input, "dataset_id")
    dsName    := getString(input, "dataset_name")

    resp, err := h.ingestionClient.DeleteDataset(ctx, &ingestionpb.DeleteDatasetRequest{
        TenantId:    tenantID,
        DatasetId:   datasetID,
        DatasetName: dsName,
    })
    if err != nil { return nil, fmt.Errorf("delete dataset: %w", err) }
    return map[string]any{
        "deleted":   true,
        "dataset_id": resp.DatasetId,
        "message":    "Dataset and all associated data deleted",
    }, nil
}
```

```go
// gateway/internal/adapter/mcp/tools/cognee/cognify_status_handler.go

type CognifyStatusHandler struct {
    cognifyClient cognifypb.CognifyServiceClient
}

func (h *CognifyStatusHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    runID     := getString(input, "pipeline_run_id")
    datasetID := getString(input, "dataset_id")

    resp, err := h.cognifyClient.GetPipelineStatus(ctx, &cognifypb.GetPipelineStatusRequest{
        PipelineRunId: runID,
        DatasetId:     datasetID,
    })
    if err != nil { return nil, fmt.Errorf("get pipeline status: %w", err) }

    result := map[string]any{
        "pipeline_run_id": resp.PipelineRunId,
        "status":          resp.Status,
    }
    if resp.NewNodes > 0 { result["new_nodes"] = resp.NewNodes }
    if resp.NewEdges > 0 { result["new_edges"] = resp.NewEdges }
    if resp.Error != "" { result["error"] = resp.Error }
    return result, nil
}
```

### 2.6. [MODIFY] MCP Tool Registry — Registration

```go
// gateway/internal/adapter/mcp/tool_registry.go

func NewToolRegistry(deps *Dependencies) *ToolRegistry {
    reg := &ToolRegistry{}

    // Existing 16 tools (unchanged)
    reg.RegisterAll(
        memoryStoreTool(deps),
        memoryRecallTool(deps),
        memorySearchTool(deps),
        memoryTimelineTool(deps),
        memoryProfileTool(deps),
        memoryForgetTool(deps),
        graphQueryTool(deps),
        ovReadFileTool(deps),
        ovWriteFileTool(deps),
        ovSearchTool(deps),
        ovListDirTool(deps),
        ovGrepTool(deps),
        ovTreeTool(deps),
        ovSessionCommitTool(deps),
        ovIngestTool(deps),
        ovDeleteTool(deps),
    )

    // [NEW] 6 Cognee-standard tools
    cogneetools.RegisterCogneeStandardTools(reg, &cogneetools.Dependencies{
        CognifyHandler:         cogneetools.NewCognifyHandler(deps.IngestionClient, deps.CognifyClient),
        SearchHandler:          cogneetools.NewSearchHandler(deps.SearchClient),
        SaveInteractionHandler: cogneetools.NewSaveInteractionHandler(deps.SearchClient, deps.MemoryClient),
        ListDataHandler:        cogneetools.NewListDataHandler(deps.IngestionClient),
        DeleteDatasetHandler:   cogneetools.NewDeleteDatasetHandler(deps.IngestionClient),
        CognifyStatusHandler:   cogneetools.NewCognifyStatusHandler(deps.CognifyClient),
    })

    // Verify total tool count
    // assert reg.Count() == 22
    return reg
}
```

### 2.7. [MODIFY] MCP Server Config — Security

```go
// gateway/internal/infra/config/config.go

type MCPConfig struct {
    Port                 int      `yaml:"port" env:"VNP_MEMORY_SERVER_MCP_PORT" default:"8082"`
    Enabled              bool     `yaml:"enabled" default:"true"`
    // [NEW]
    DNSRebindingProtection bool   `yaml:"dns_rebinding_protection" default:"true"`
    AllowedHosts         []string `yaml:"allowed_hosts"`   // empty = allow localhost only
    CORSAllowOrigins     []string `yaml:"cors_allow_origins" default:"[\"http://localhost:*\"]"`
}
```

```go
// gateway/internal/adapter/mcp/server.go

// [ADD] DNS rebinding protection middleware
func dnsRebindingProtection(cfg MCPConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !cfg.DNSRebindingProtection {
                next.ServeHTTP(w, r)
                return
            }
            host := r.Host
            if host == "" || isLocalhost(host) || isAllowedHost(host, cfg.AllowedHosts) {
                next.ServeHTTP(w, r)
                return
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

### 2.8. Tool Count Verification Test

```go
// gateway/internal/adapter/mcp/tool_registry_test.go

func TestMCPToolCount_AfterCogneeParity(t *testing.T) {
    reg := NewToolRegistry(mockDeps)

    // Existing 16 + 6 new Cognee-standard = 22
    assert.Equal(t, 22, reg.Count(), "MCP server must have exactly 22 tools after Cognee parity")

    // Verify Cognee standard tools present
    cogneeStandard := []string{"cognify", "search", "save_interaction", "list_data", "delete_dataset", "cognify_status"}
    for _, name := range cogneeStandard {
        assert.True(t, reg.HasTool(name), "Missing Cognee standard tool: "+name)
    }

    // Verify existing tools not removed
    existing := []string{"memory_store", "memory_recall", "memory_search", "ov_read_file", "graph_query"}
    for _, name := range existing {
        assert.True(t, reg.HasTool(name), "Existing tool missing: "+name)
    }

    // No duplicate tool names
    names := reg.Names()
    assert.Equal(t, len(names), len(uniqueStrings(names)), "No duplicate tool names")
}
```

### 2.9. Claude Code Plugin Config Example

```json
// Generated by GET /v1/admin/plugin/claude-code
{
  "mcpServers": {
    "vnp-memory": {
      "type": "http",
      "url": "http://localhost:8082",
      "headers": {
        "Authorization": "Bearer <api_key>",
        "X-Tenant-ID": "<tenant_id>"
      }
    }
  }
}
```

---

## 3. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `gateway/internal/adapter/mcp/tools/cognee/registry.go` | 6 tool definitions |
| `gateway/internal/adapter/mcp/tools/cognee/cognify_handler.go` | cognify: AddData + StartCognify chain |
| `gateway/internal/adapter/mcp/tools/cognee/search_handler.go` | search: proxy SearchUseCase |
| `gateway/internal/adapter/mcp/tools/cognee/save_interaction_handler.go` | save_interaction: memory + feedback |
| `gateway/internal/adapter/mcp/tools/cognee/list_data_handler.go` | list_data: datasets + entries |
| `gateway/internal/adapter/mcp/tools/cognee/delete_dataset_handler.go` | delete_dataset: cascading delete |
| `gateway/internal/adapter/mcp/tools/cognee/cognify_status_handler.go` | cognify_status: PipelineRun |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `gateway/internal/adapter/mcp/tool_registry.go` | + RegisterCogneeStandardTools() call |
| `gateway/internal/adapter/mcp/server.go` | + DNS rebinding protection middleware |
| `gateway/internal/infra/config/config.go` | + MCPConfig security fields |
| `apps/memory/configs/config.yaml` | + mcp.dns_rebinding_protection: true |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-COGNEE-007 | Covered by |
|--------------------|-----------|
| `tools/list` → 6 Cognee-standard tools | TestMCPToolCount + registry.go |
| `cognify` với text → pipeline_run_id | CognifyHandler.Handle() chains |
| `search` với GRAPH_COMPLETION → results | SearchHandler proxy |
| `cognify_status` với run_id → {status:COMPLETED} | CognifyStatusHandler |
| `save_interaction` lưu QAEntry | SaveInteractionHandler → Remember() |
| `delete_dataset` cascade xóa | DeleteDatasetHandler → DeleteDataset() |
| Claude Code MCP plugin trỏ đến :8082 hoạt động | Plugin config + DNS rebinding protection |
| Existing tools (cognee_add, graphiti_ingest, v.v.) vẫn hoạt động | Không xóa existing tools |
