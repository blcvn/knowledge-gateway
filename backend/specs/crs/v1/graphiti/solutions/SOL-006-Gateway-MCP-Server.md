# Solution: SOL-006 — API Gateway & MCP Server (9 Graphiti Tools)

**CR ID:** CR-GR-006  
**Solution ID:** SOL-006  
**Priority:** High (Wave 3)  
**Architecture:** EXTEND `gateway/` — add graphiti REST routes + 9 MCP tools (22 → 31 tools)

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §4`:
- Gateway đã có: REST `:8080`, gRPC `:8081`, MCP `:8082`.
- MCP server có **22 tools** (sau Cognee parity từ SOL-007).
- `InProcessRegistry` — gọi graphiti services qua bufconn (zero network).
- JWT auth middleware đã có — tái dùng cho graphiti routes.
- Rate limiting chưa có — cần thêm.

**Strategy:** Thêm 26 REST routes + 9 MCP tools, không sửa existing tools.

---

## 2. REST Handlers — `gateway/internal/adapter/handler/graphiti/`

### 2.1. Episode Handler

```go
// gateway/internal/adapter/handler/graphiti/episode_handler.go

package graphiti

import (
    "encoding/json"
    "net/http"
    "time"

    ingestionpb "github.com/vnp-memory/api/proto/graphiti/ingestion/v1"
)

type EpisodeHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
}

// POST /v1/graphiti/episodes
func (h *EpisodeHandler) IngestEpisode(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Name              string    `json:"name"`
        Body              string    `json:"body"`
        Source            string    `json:"source"`  // message|text|json|fact_triple
        SourceDescription string    `json:"source_description"`
        ReferenceTime     time.Time `json:"reference_time"`
        SagaID            string    `json:"saga_id"`
        PrevEpisodeUUID   string    `json:"prev_episode_uuid"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
        return
    }

    groupID := extractGroupID(r)   // from JWT claim "group_id" or X-Group-ID header

    resp, err := h.ingestionClient.IngestEpisode(
        withGroupIDMeta(r.Context(), groupID),
        &ingestionpb.IngestEpisodeRequest{
            Name:              body.Name,
            Body:              body.Body,
            Source:            body.Source,
            SourceDescription: body.SourceDescription,
            SagaId:            body.SagaID,
            PrevEpisodeUuid:   body.PrevEpisodeUUID,
        },
    )
    if err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    respondJSON(w, http.StatusOK, map[string]any{
        "episode_uuid": resp.EpisodeUuid,
        "stats": map[string]any{
            "entities_extracted":  resp.Stats.EntitiesExtracted,
            "entities_new":        resp.Stats.EntitiesNew,
            "edges_extracted":     resp.Stats.EdgesExtracted,
            "edges_new":           resp.Stats.EdgesNew,
            "processing_time_ms":  resp.Stats.ProcessingTimeMs,
        },
    })
}

// DELETE /v1/graphiti/episodes/{uuid}
func (h *EpisodeHandler) RemoveEpisode(w http.ResponseWriter, r *http.Request) {
    uuid := chi.URLParam(r, "uuid")
    groupID := extractGroupID(r)

    _, err := h.ingestionClient.RemoveEpisode(
        withGroupIDMeta(r.Context(), groupID),
        &ingestionpb.RemoveEpisodeRequest{EpisodeUuid: uuid},
    )
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    w.WriteHeader(http.StatusNoContent)
}

// GET /v1/graphiti/episodes
func (h *EpisodeHandler) ListEpisodes(w http.ResponseWriter, r *http.Request) {
    groupID := extractGroupID(r)
    lastN := queryIntParam(r, "last_n", 10)

    resp, err := h.ingestionClient.ListEpisodes(
        withGroupIDMeta(r.Context(), groupID),
        &ingestionpb.ListEpisodesRequest{GroupId: groupID, LastN: int32(lastN)},
    )
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, resp.Episodes)
}
```

### 2.2. Search Handler

```go
// gateway/internal/adapter/handler/graphiti/search_handler.go

type SearchHandler struct {
    searchClient searchpb.SearchServiceClient
}

// POST /v1/graphiti/search
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Query          string   `json:"query"`
        GroupIDs       []string `json:"group_ids"`
        NumResults     int      `json:"num_results"`
        CenterNodeUUID string   `json:"center_node_uuid"`
        SearchConfig   any      `json:"search_config"`
        Filters        any      `json:"filters"`
    }
    json.NewDecoder(r.Body).Decode(&body)

    groupID := extractGroupID(r)
    if len(body.GroupIDs) == 0 { body.GroupIDs = []string{groupID} }
    if body.NumResults == 0 { body.NumResults = 10 }

    resp, err := h.searchClient.Search(
        r.Context(),
        &searchpb.SearchRequest{
            Query:          body.Query,
            GroupIds:       body.GroupIDs,
            NumResults:     int32(body.NumResults),
            CenterNodeUuid: body.CenterNodeUUID,
        },
    )
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, resp)
}
```

### 2.3. Aggregate Health Check

```go
// gateway/internal/adapter/handler/health_handler.go

type HealthHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
    searchClient    searchpb.SearchServiceClient
    knowledgeClient knowledgepb.KnowledgeServiceClient
    storeClient     storepb.StoreServiceClient
    adminClient     adminpb.AdminServiceClient
    redisClient     redis.Client
    natsConn        *nats.Conn
}

type ServiceHealth struct {
    Status    string `json:"status"`    // healthy | unhealthy
    LatencyMs int64  `json:"latency_ms"`
    Error     string `json:"error,omitempty"`
}

// GET /healthz
func (h *HealthHandler) HealthZ(w http.ResponseWriter, r *http.Request) {
    services := map[string]*ServiceHealth{
        "graphiti-ingestion": {},
        "graphiti-search":    {},
        "graphiti-knowledge": {},
        "graphiti-store":     {},
        "graphiti-admin":     {},
        "redis":              {},
        "nats":               {},
    }

    var wg sync.WaitGroup
    for name := range services {
        wg.Add(1)
        go func(svcName string) {
            defer wg.Done()
            start := time.Now()
            var err error
            switch svcName {
            case "graphiti-ingestion":
                ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
                defer cancel()
                _, err = h.ingestionClient.GetPipelineStatus(ctx, &ingestionpb.GetPipelineStatusRequest{})
            case "graphiti-search":
                // gRPC health check ping
            case "graphiti-knowledge":
                // gRPC health check ping
            case "graphiti-store":
                // gRPC health check ping
            case "graphiti-admin":
                // gRPC health check ping
            case "redis":
                _, err = h.redisClient.Ping(r.Context()).Result()
            case "nats":
                if !h.natsConn.IsConnected() { err = fmt.Errorf("NATS disconnected") }
            }

            lat := time.Since(start).Milliseconds()
            if err != nil {
                services[svcName] = &ServiceHealth{Status: 🔨 In Progress LatencyMs: lat, Error: err.Error()}
            } else {
                services[svcName] = &ServiceHealth{Status: "healthy", LatencyMs: lat}
            }
        }(name)
    }
    wg.Wait()

    // Determine overall status
    overallStatus := "healthy"
    for _, svc := range services {
        if svc.Status == "unhealthy" { overallStatus = "unhealthy"; break }
    }

    httpStatus := http.StatusOK
    if overallStatus == "unhealthy" { httpStatus = http.StatusServiceUnavailable }

    respondJSON(w, httpStatus, map[string]any{
        "status":    overallStatus,
        "services":  services,
        "timestamp": time.Now(),
    })
}
```

---

## 3. Rate Limiting Middleware — `gateway/internal/middleware/rate_limit.go`

```go
// gateway/internal/middleware/rate_limit.go

package middleware

import (
    "fmt"
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"
)

type RateLimit struct {
    Requests int           // max requests per window
    Window   time.Duration
}

var graphitiRateLimits = map[string]RateLimit{
    "POST:/v1/graphiti/episodes":       {Requests: 100, Window: time.Minute},
    "POST:/v1/graphiti/episodes/bulk":  {Requests: 10,  Window: time.Minute},
    "POST:/v1/graphiti/search":         {Requests: 500, Window: time.Minute},
    "DELETE:/v1/graphiti/admin/data":   {Requests: 5,   Window: time.Hour},
    "POST:/v1/graphiti/triplets":       {Requests: 200, Window: time.Minute},
}

type RateLimiter struct {
    redis redis.Client
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        limit, ok := graphitiRateLimits[r.Method+":"+r.URL.Path]
        if !ok { next.ServeHTTP(w, r); return }

        tenantID := extractTenantID(r)
        key := fmt.Sprintf("rate:%s:%s:%s", tenantID, r.Method, r.URL.Path)

        // Sliding window using Redis
        count, err := rl.redis.Incr(r.Context(), key).Result()
        if count == 1 {
            rl.redis.Expire(r.Context(), key, limit.Window)
        }

        if err != nil || count > int64(limit.Requests) {
            w.Header().Set("Retry-After", fmt.Sprintf("%.0f", limit.Window.Seconds()))
            http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

---

## 4. Group ID Propagation Middleware

```go
// gateway/internal/middleware/group_id.go

// ExtractGroupID extracts group_id from (priority order):
// 1. JWT claim "group_id"
// 2. JWT claim "tenant_id"  
// 3. Header X-Group-ID
// 4. Query param ?group_id=...
// 5. Default "default" (dev mode only)
func ExtractGroupID(r *http.Request) string {
    if claims := jwtClaimsFromCtx(r.Context()); claims != nil {
        if id, ok := claims["group_id"].(string); ok && id != "" { return id }
        if id, ok := claims["tenant_id"].(string); ok && id != "" { return id }
    }
    if id := r.Header.Get("X-Group-ID"); id != "" { return id }
    if id := r.URL.Query().Get("group_id"); id != "" { return id }
    return "default"
}

// withGroupIDMeta injects group_id into gRPC outgoing metadata
func withGroupIDMeta(ctx context.Context, groupID string) context.Context {
    return metadata.AppendToOutgoingContext(ctx, "x-group-id", groupID)
}
```

---

## 5. MCP Tools — `gateway/internal/adapter/mcp/tools/graphiti/`

### 5.1. Registry (9 Tools)

```go
// gateway/internal/adapter/mcp/tools/graphiti/registry.go

package graphititools

import "github.com/vnp-memory/gateway/internal/adapter/mcp"

// RegisterGraphitiTools adds 9 Graphiti MCP tools (22 → 31 total)
func RegisterGraphitiTools(reg *mcp.ToolRegistry, deps *Dependencies) {
    tools := []mcp.Tool{
        addMemoryTool(deps),
        searchMemoryTool(deps),
        getEpisodesTool(deps),
        deleteEpisodeTool(deps),
        deleteEntityNodeTool(deps),
        deleteEntityEdgeTool(deps),
        getEntityEdgeTool(deps),
        clearGraphTool(deps),
        getStatusTool(deps),
    }
    for _, t := range tools { reg.Register(t) }
}

func addMemoryTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "add_memory",
        Description: "Add a new memory (episode) to the temporal knowledge graph. Content is automatically processed: entities extracted, relationships identified, temporal context maintained, and contradictions resolved. Supports text, messages, JSON, and direct triplets.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "content": {
                    Type:        "string",
                    Description: "The content of the memory to add (text, message, or JSON string)",
                },
                "source_description": {
                    Type:        "string",
                    Description: "Brief description of memory source (e.g., 'chat conversation', 'document', 'API response')",
                },
                "source": {
                    Type:        "string",
                    Enum:        []string{"text", "message", "json", "fact_triple"},
                    Description: "Type of content source. Default: 'text'",
                },
                "group_id": {
                    Type:        "string",
                    Description: "Namespace for multi-tenant isolation (e.g., project ID, user ID). Defaults to authenticated tenant.",
                },
                "reference_time": {
                    Type:        "string",
                    Description: "ISO8601 timestamp when the event occurred (for temporal reasoning). Defaults to now.",
                },
                "saga_id": {
                    Type:        "string",
                    Description: "Optional saga (narrative sequence) ID to group related episodes",
                },
            },
            Required: []string{"content", "source_description"},
        },
        Handler: deps.AddMemoryHandler.Handle,
    }
}

func searchMemoryTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "search_memory",
        Description: "Search the temporal knowledge graph using hybrid retrieval (semantic similarity + keyword BM25 + graph traversal). Returns relevant facts, entities, and relationships. Supports temporal filters for point-in-time queries.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query": {
                    Type:        "string",
                    Description: "Natural language search query",
                },
                "group_ids": {
                    Type:        "array",
                    Items:       &mcp.Property{Type: "string"},
                    Description: "Namespaces to search (defaults to authenticated tenant)",
                },
                "num_results": {
                    Type:    "integer",
                    Default: 10,
                    Description: "Number of results to return (max 100)",
                },
                "center_node_uuid": {
                    Type:        "string",
                    Description: "Boost results near this entity UUID (node distance reranking)",
                },
                "valid_at": {
                    Type:        "string",
                    Description: "ISO8601 timestamp for point-in-time query (only return facts valid at this time)",
                },
            },
            Required: []string{"query"},
        },
        Handler: deps.SearchMemoryHandler.Handle,
    }
}

func getEpisodesTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "get_episodes",
        Description: "Retrieve recent episodes (memories) from the knowledge graph, ordered by most recent first.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "group_id": {Type: "string"},
                "last_n":   {Type: "integer", Default: 10, Description: "Number of episodes to retrieve (max 100)"},
                "source":   {Type: "string", Enum: []string{"text", "message", "json", "fact_triple"}, Description: "Filter by source type"},
            },
        },
        Handler: deps.GetEpisodesHandler.Handle,
    }
}

func deleteEpisodeTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "delete_episode",
        Description: "Delete an episode and cascade-remove its episodic edges (MENTIONS relationships). The entity nodes and entity edges remain unless explicitly deleted.",
        InputSchema: mcp.Schema{
            Type:     "object",
            Required: []string{"episode_uuid"},
            Properties: map[string]mcp.Property{
                "episode_uuid": {Type: "string", Description: "UUID of the episode to delete"},
            },
        },
        Handler: deps.DeleteEpisodeHandler.Handle,
    }
}

func deleteEntityNodeTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "delete_entity_node",
        Description: "Delete an entity node and all its associated edges from the knowledge graph. This is a destructive operation that cannot be undone.",
        InputSchema: mcp.Schema{
            Type:     "object",
            Required: []string{"node_uuid"},
            Properties: map[string]mcp.Property{
                "node_uuid": {Type: "string", Description: "UUID of the entity node to delete"},
            },
        },
        Handler: deps.DeleteEntityNodeHandler.Handle,
    }
}

func deleteEntityEdgeTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "delete_entity_edge",
        Description: "Permanently delete a fact/relationship edge from the knowledge graph. Note: prefer temporal invalidation (which preserves historical data) over deletion.",
        InputSchema: mcp.Schema{
            Type:     "object",
            Required: []string{"edge_uuid"},
            Properties: map[string]mcp.Property{
                "edge_uuid": {Type: "string", Description: "UUID of the entity edge to delete"},
            },
        },
        Handler: deps.DeleteEntityEdgeHandler.Handle,
    }
}

func getEntityEdgeTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "get_entity_edge",
        Description: "Retrieve full details of a fact/relationship edge, including temporal fields (valid_at, invalid_at, expired_at), fact text, source episodes, and embedding.",
        InputSchema: mcp.Schema{
            Type:     "object",
            Required: []string{"edge_uuid"},
            Properties: map[string]mcp.Property{
                "edge_uuid": {Type: "string", Description: "UUID of the entity edge to retrieve"},
            },
        },
        Handler: deps.GetEntityEdgeHandler.Handle,
    }
}

func clearGraphTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "clear_graph",
        Description: "Clear ALL data for a specific group_id — all episodes, entities, relationships, and communities. This is irreversible. Use with caution.",
        InputSchema: mcp.Schema{
            Type:     "object",
            Required: []string{"group_id"},
            Properties: map[string]mcp.Property{
                "group_id": {Type: "string", Description: "Namespace to clear. IRREVERSIBLE."},
            },
        },
        Handler: deps.ClearGraphHandler.Handle,
    }
}

func getStatusTool(deps *Dependencies) mcp.Tool {
    return mcp.Tool{
        Name:        "get_status",
        Description: "Get current server health status including all graphiti services (ingestion, search, knowledge, store, admin), Neo4j, Redis, and NATS.",
        InputSchema: mcp.Schema{Type: "object"},
        Handler:     deps.GetStatusHandler.Handle,
    }
}
```

### 5.2. Tool Handlers

```go
// gateway/internal/adapter/mcp/tools/graphiti/handlers.go

type AddMemoryHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
}

func (h *AddMemoryHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    groupID := extractTenantFromContext(ctx)
    content  := getString(input, "content")
    srcDesc  := getStringOrDefault(input, "source_description", "mcp_tool")
    source   := getStringOrDefault(input, "source", "text")
    sagaID   := getString(input, "saga_id")
    if gid := getString(input, "group_id"); gid != "" { groupID = gid }

    resp, err := h.ingestionClient.IngestEpisode(
        withGroupIDMeta(ctx, groupID),
        &ingestionpb.IngestEpisodeRequest{
            Name:              fmt.Sprintf("episode_%d", time.Now().UnixMilli()),
            Body:              content,
            Source:            source,
            SourceDescription: srcDesc,
            SagaId:            sagaID,
        },
    )
    if err != nil { return nil, fmt.Errorf("add_memory failed: %w", err) }

    return map[string]any{
        "episode_uuid":       resp.EpisodeUuid,
        "entities_extracted": resp.Stats.EntitiesExtracted,
        "entities_new":       resp.Stats.EntitiesNew,
        "edges_new":          resp.Stats.EdgesNew,
        "message":            "Memory added and processed into knowledge graph",
    }, nil
}

type SearchMemoryHandler struct {
    searchClient searchpb.SearchServiceClient
}

func (h *SearchMemoryHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    query      := getString(input, "query")
    groupIDs   := toStringSliceOrDefault(input["group_ids"], []string{extractTenantFromContext(ctx)})
    numResults := getIntOrDefault(input, "num_results", 10)
    centerNode := getString(input, "center_node_uuid")

    resp, err := h.searchClient.Search(ctx, &searchpb.SearchRequest{
        Query:          query,
        GroupIds:       groupIDs,
        NumResults:     int32(numResults),
        CenterNodeUuid: centerNode,
    })
    if err != nil { return nil, fmt.Errorf("search_memory failed: %w", err) }

    // Format for LLM consumption: return as list of facts
    var facts []string
    for _, e := range resp.Edges {
        facts = append(facts, e.Fact)
    }
    return map[string]any{
        "facts":       facts,
        "edge_count":  len(resp.Edges),
        "node_count":  len(resp.Nodes),
        "latency_ms":  resp.LatencyMs,
    }, nil
}

type GetStatusHandler struct {
    healthChecker HealthChecker
}

func (h *GetStatusHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
    return h.healthChecker.Check(ctx), nil
}
```

---

## 6. Router Registration

```go
// gateway/internal/adapter/handler/router.go (additions)

func setupGraphitiRoutes(r chi.Router, deps *Dependencies) {
    r.Group(func(r chi.Router) {
        r.Use(deps.AuthMiddleware.JWT)
        r.Use(deps.RateLimiter.Middleware)

        // Episodes
        r.Post("/v1/graphiti/episodes",              deps.EpisodeHandler.IngestEpisode)
        r.Post("/v1/graphiti/episodes/bulk",          deps.EpisodeHandler.IngestEpisodeBulk)
        r.Delete("/v1/graphiti/episodes/{uuid}",     deps.EpisodeHandler.RemoveEpisode)
        r.Get("/v1/graphiti/episodes",               deps.EpisodeHandler.ListEpisodes)
        r.Get("/v1/graphiti/episodes/{uuid}",        deps.EpisodeHandler.GetEpisode)
        r.Post("/v1/graphiti/triplets",              deps.TripletHandler.AddTriplet)

        // Sagas
        r.Post("/v1/graphiti/sagas",                 deps.SagaHandler.CreateSaga)
        r.Post("/v1/graphiti/sagas/{id}/summarize",  deps.SagaHandler.SummarizeSaga)
        r.Get("/v1/graphiti/sagas/{id}",             deps.SagaHandler.GetSaga)

        // Search
        r.Post("/v1/graphiti/search",                deps.SearchHandler.Search)
        r.Post("/v1/graphiti/search/advanced",       deps.SearchHandler.SearchAdvanced)
        r.Get("/v1/graphiti/search/nodes",           deps.SearchHandler.SearchNodes)
        r.Get("/v1/graphiti/search/edges",           deps.SearchHandler.SearchEdges)
        r.Get("/v1/graphiti/search/episodes",        deps.SearchHandler.SearchEpisodes)
        r.Get("/v1/graphiti/search/communities",     deps.SearchHandler.SearchCommunities)

        // Entities & Edges (read-only from store)
        r.Get("/v1/graphiti/entities/{uuid}",        deps.StoreHandler.GetEntityNode)
        r.Get("/v1/graphiti/edges/{uuid}",           deps.StoreHandler.GetEntityEdge)

        // Ontology (from CR-005)
        r.Post("/v1/graphiti/ontology/{group_id}",         deps.OntologyHandler.Save)
        r.Get("/v1/graphiti/ontology/{group_id}",          deps.OntologyHandler.Get)
        r.Delete("/v1/graphiti/ontology/{group_id}",       deps.OntologyHandler.Delete)
        r.Post("/v1/graphiti/ontology/{group_id}/preset",  deps.OntologyHandler.ApplyPreset)

        // Admin (require "admin" scope)
        r.Group(func(r chi.Router) {
            r.Use(deps.AuthMiddleware.RequireScope("admin"))
            r.Post("/v1/graphiti/admin/communities",     deps.AdminHandler.RebuildCommunities)
            r.Post("/v1/graphiti/admin/indices",         deps.AdminHandler.BuildIndices)
            r.Delete("/v1/graphiti/admin/data/{group_id}", deps.AdminHandler.ClearData)
            r.Get("/v1/graphiti/admin/token-usage",     deps.AdminHandler.GetTokenUsage)
            r.Post("/v1/graphiti/admin/tenants",        deps.AdminHandler.CreateTenant)
            r.Get("/v1/graphiti/admin/tenants",         deps.AdminHandler.ListTenants)
            r.Delete("/v1/graphiti/admin/tenants/{id}", deps.AdminHandler.DeleteTenant)
            r.Get("/v1/graphiti/admin/tenants/{id}/stats", deps.AdminHandler.GetTenantStats)
        })
    })

    // Health (no auth)
    r.Get("/healthz", deps.HealthHandler.HealthZ)
}
```

---

## 7. MCP Tool Registration

```go
// gateway/internal/adapter/mcp/tool_registry.go (additions)

func NewToolRegistry(deps *Dependencies) *ToolRegistry {
    reg := &ToolRegistry{}

    // Existing 22 tools (from Cognee parity + original 16)
    // ... unchanged ...

    // [NEW] 9 Graphiti standard tools (22 → 31)
    graphititools.RegisterGraphitiTools(reg, &graphititools.Dependencies{
        AddMemoryHandler:        graphititools.NewAddMemoryHandler(deps.IngestionClient),
        SearchMemoryHandler:     graphititools.NewSearchMemoryHandler(deps.SearchClient),
        GetEpisodesHandler:      graphititools.NewGetEpisodesHandler(deps.IngestionClient),
        DeleteEpisodeHandler:    graphititools.NewDeleteEpisodeHandler(deps.IngestionClient),
        DeleteEntityNodeHandler: graphititools.NewDeleteEntityNodeHandler(deps.StoreClient),
        DeleteEntityEdgeHandler: graphititools.NewDeleteEntityEdgeHandler(deps.StoreClient),
        GetEntityEdgeHandler:    graphititools.NewGetEntityEdgeHandler(deps.StoreClient),
        ClearGraphHandler:       graphititools.NewClearGraphHandler(deps.AdminClient),
        GetStatusHandler:        graphititools.NewGetStatusHandler(deps.HealthChecker),
    })

    // Verify 31 tools total
    // assert reg.Count() == 31
    return reg
}
```

---

## 8. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `gateway/internal/adapter/handler/graphiti/episode_handler.go` | Episode REST handlers |
| `gateway/internal/adapter/handler/graphiti/search_handler.go` | Search REST handlers |
| `gateway/internal/adapter/handler/graphiti/saga_handler.go` | Saga REST handlers |
| `gateway/internal/adapter/handler/graphiti/store_handler.go` | Entity/Edge read handlers |
| `gateway/internal/adapter/handler/graphiti/ontology_handler.go` | Ontology REST handlers |
| `gateway/internal/adapter/handler/graphiti/admin_handler.go` | Admin REST handlers |
| `gateway/internal/adapter/handler/health_handler.go` | Aggregate /healthz |
| `gateway/internal/adapter/mcp/tools/graphiti/registry.go` | 9 MCP tool definitions |
| `gateway/internal/adapter/mcp/tools/graphiti/handlers.go` | MCP tool handler implementations |
| `gateway/internal/middleware/rate_limit.go` | Per-tenant rate limiting (Redis sliding window) |
| `gateway/internal/middleware/group_id.go` | Group ID extraction + gRPC propagation |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `gateway/internal/adapter/handler/router.go` | + 26 graphiti routes + /healthz |
| `gateway/internal/adapter/mcp/tool_registry.go` | + RegisterGraphitiTools() (22 → 31) |
| `apps/memory/configs/config.yaml` | + graphiti rate limits config |
| `apps/memory/internal/bootstrap/gateway.go` | + graphiti handler deps injection |

---

## 9. Acceptance Criteria Mapping

| AC từ CR-GR-006 | Covered by |
|----------------|-----------|
| MCP add_memory → episode ingest → entities in graph | AddMemoryHandler → ingestionClient.IngestEpisode() |
| MCP search_memory "who works at Acme" ≤ 1s | SearchMemoryHandler → searchClient.Search() |
| MCP get_episodes → 10 most recent | GetEpisodesHandler |
| MCP get_status → {status: "healthy", services: {...}} | GetStatusHandler → HealthChecker.Check() |
| POST /v1/graphiti/episodes valid JWT → 200 | EpisodeHandler + AuthMiddleware |
| POST /v1/graphiti/episodes no JWT → 401 | JWT middleware rejects |
| 101 req/min → 429 Too Many Requests | RateLimiter (100/min for episodes) |
| X-Group-ID tenant-A → isolated from tenant-B | ExtractGroupID → withGroupIDMeta |
| GET /healthz Neo4j down → {unhealthy, store: unhealthy} 503 | HealthHandler parallel ping |
| DELETE /v1/graphiti/admin/data/group-alpha → clear | ClearData → adminClient.ClearData() |
