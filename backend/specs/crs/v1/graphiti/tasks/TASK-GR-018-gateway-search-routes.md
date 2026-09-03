# TASK-GR-018 — Gateway: Search + Entity REST + /healthz

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-018 |
| **Wave** | 3 |
| **Component** | `gateway/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-006 §2.2, §2.3 |
| **Priority** | High |
| **Depends On** | TASK-GR-017 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** gateway graphiti search routes implemented  
---

## Context

Implement search handlers (`POST /v1/graphiti/search`, `POST /v1/graphiti/search/advanced`), entity/edge read handlers, và aggregate health check `/healthz` endpoint.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `gateway/internal/adapter/handler/graphiti/search_handler.go` |
| CREATE | `gateway/internal/adapter/handler/graphiti/ontology_handler.go` |
| CREATE | `gateway/internal/adapter/handler/health_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### File 1: `gateway/internal/adapter/handler/graphiti/search_handler.go`

```go
package graphiti

import (
    "encoding/json"
    "net/http"

    searchpb "github.com/vnp-memory/api/proto/graphiti/search/v1"
    "github.com/vnp-memory/gateway/internal/middleware"
)

type SearchHandler struct {
    searchClient searchpb.SearchServiceClient
}

func NewSearchHandler(client searchpb.SearchServiceClient) *SearchHandler {
    return &SearchHandler{searchClient: client}
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
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if body.Query == "" { respondError(w, http.StatusBadRequest, "query is required"); return }

    groupID := middleware.ExtractGroupID(r)
    if len(body.GroupIDs) == 0 { body.GroupIDs = []string{groupID} }
    if body.NumResults == 0 { body.NumResults = 10 }
    if body.NumResults > 100 { body.NumResults = 100 }

    resp, err := h.searchClient.Search(r.Context(), &searchpb.SearchRequest{
        Query:          body.Query,
        GroupIds:       body.GroupIDs,
        NumResults:     int32(body.NumResults),
        CenterNodeUuid: body.CenterNodeUUID,
    })
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, resp)
}

// POST /v1/graphiti/search/advanced
func (h *SearchHandler) SearchAdvanced(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Query    string `json:"query"`
        GroupIDs []string `json:"group_ids"`
        NumResults int   `json:"num_results"`
        CenterNodeUUID string `json:"center_node_uuid"`
        SearchConfigName string `json:"search_config_name"` // recipe name
        Filters struct {
            ValidAt  string `json:"valid_at"`
            GroupIDs []string `json:"group_ids"`
        } `json:"filters"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    groupID := middleware.ExtractGroupID(r)
    if len(body.GroupIDs) == 0 { body.GroupIDs = []string{groupID} }
    if body.NumResults == 0 { body.NumResults = 10 }

    resp, err := h.searchClient.SearchAdvanced(r.Context(), &searchpb.SearchAdvancedRequest{
        Query:            body.Query,
        GroupIds:         body.GroupIDs,
        NumResults:       int32(body.NumResults),
        CenterNodeUuid:   body.CenterNodeUUID,
        SearchConfigName: body.SearchConfigName,
        ValidAt:          body.Filters.ValidAt,
    })
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, resp)
}

// GET /v1/graphiti/search/nodes
func (h *SearchHandler) SearchNodes(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" { respondError(w, http.StatusBadRequest, "q is required"); return }

    groupID := middleware.ExtractGroupID(r)
    resp, err := h.searchClient.SearchNodes(r.Context(), &searchpb.SearchNodesRequest{
        Query:    query,
        GroupIds: []string{groupID},
    })
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, resp)
}
```

### File 2: `gateway/internal/adapter/handler/graphiti/ontology_handler.go`

```go
package graphiti

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    knowledgepb "github.com/vnp-memory/api/proto/graphiti/knowledge/v1"
    "github.com/vnp-memory/gateway/internal/middleware"
)

type OntologyHandler struct {
    knowledgeClient knowledgepb.KnowledgeServiceClient
}

func NewOntologyHandler(client knowledgepb.KnowledgeServiceClient) *OntologyHandler {
    return &OntologyHandler{knowledgeClient: client}
}

// POST /v1/graphiti/ontology/{group_id}
func (h *OntologyHandler) Save(w http.ResponseWriter, r *http.Request) {
    groupID := chi.URLParam(r, "group_id")
    if groupID == "" { groupID = middleware.ExtractGroupID(r) }

    var body struct {
        EntityTypes map[string]any `json:"entity_types"`
        EdgeTypes   map[string]any `json:"edge_types"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    entityJSON, _ := json.Marshal(body.EntityTypes)
    edgeJSON, _   := json.Marshal(body.EdgeTypes)

    _, err := h.knowledgeClient.SaveOntology(r.Context(), &knowledgepb.SaveOntologyRequest{
        GroupId:         groupID,
        EntityTypesJson: entityJSON,
        EdgeTypesJson:   edgeJSON,
    })
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    w.WriteHeader(http.StatusCreated)
}

// GET /v1/graphiti/ontology/{group_id}
func (h *OntologyHandler) Get(w http.ResponseWriter, r *http.Request) {
    groupID := chi.URLParam(r, "group_id")
    resp, err := h.knowledgeClient.GetOntology(r.Context(), &knowledgepb.GetOntologyRequest{GroupId: groupID})
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    if resp == nil { respondJSON(w, http.StatusOK, map[string]any{"entity_types": map[string]any{}, "edge_types": map[string]any{}}); return }

    var entityTypes, edgeTypes map[string]any
    json.Unmarshal(resp.EntityTypesJson, &entityTypes)
    json.Unmarshal(resp.EdgeTypesJson,   &edgeTypes)
    respondJSON(w, http.StatusOK, map[string]any{"entity_types": entityTypes, "edge_types": edgeTypes})
}

// DELETE /v1/graphiti/ontology/{group_id}
func (h *OntologyHandler) Delete(w http.ResponseWriter, r *http.Request) {
    groupID := chi.URLParam(r, "group_id")
    _, err := h.knowledgeClient.DeleteOntology(r.Context(), &knowledgepb.DeleteOntologyRequest{GroupId: groupID})
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    w.WriteHeader(http.StatusNoContent)
}

// POST /v1/graphiti/ontology/{group_id}/preset
func (h *OntologyHandler) ApplyPreset(w http.ResponseWriter, r *http.Request) {
    groupID := chi.URLParam(r, "group_id")
    var body struct { PresetName string `json:"preset_name"` }
    json.NewDecoder(r.Body).Decode(&body)
    if body.PresetName == "" { respondError(w, http.StatusBadRequest, "preset_name is required"); return }

    _, err := h.knowledgeClient.ApplyOntologyPreset(r.Context(), &knowledgepb.ApplyOntologyPresetRequest{
        GroupId: groupID, PresetName: body.PresetName,
    })
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, map[string]any{"applied": body.PresetName, "group_id": groupID})
}
```

### File 3: `gateway/internal/adapter/handler/health_handler.go`

```go
package handler

import (
    "context"
    "encoding/json"
    "net/http"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/nats-io/nats.go"
    ingestionpb "github.com/vnp-memory/api/proto/graphiti/ingestion/v1"
    searchpb    "github.com/vnp-memory/api/proto/graphiti/search/v1"
)

type ServiceHealth struct {
    Status    string `json:"status"`
    LatencyMs int64  `json:"latency_ms,omitempty"`
    Error     string `json:"error,omitempty"`
}

type HealthHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
    searchClient    searchpb.SearchServiceClient
    redisClient     redis.UniversalClient
    natsConn        *nats.Conn
}

func NewHealthHandler(
    ingestion ingestionpb.IngestionServiceClient,
    search searchpb.SearchServiceClient,
    redisClient redis.UniversalClient,
    natsConn *nats.Conn,
) *HealthHandler {
    return &HealthHandler{
        ingestionClient: ingestion,
        searchClient:    search,
        redisClient:     redisClient,
        natsConn:        natsConn,
    }
}

// GET /healthz
func (h *HealthHandler) HealthZ(w http.ResponseWriter, r *http.Request) {
    services := map[string]*ServiceHealth{
        "graphiti-ingestion": {},
        "graphiti-search":    {},
        "redis":              {},
        "nats":               {},
    }

    var wg sync.WaitGroup
    for svcName := range services {
        wg.Add(1)
        go func(name string) {
            defer wg.Done()
            svc := services[name]
            start := time.Now()
            var err error

            ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
            defer cancel()

            switch name {
            case "graphiti-ingestion":
                _, err = h.ingestionClient.GetPipelineStatus(ctx, &ingestionpb.GetPipelineStatusRequest{})
            case "graphiti-search":
                _, err = h.searchClient.HealthCheck(ctx, &searchpb.HealthCheckRequest{})
            case "redis":
                _, err = h.redisClient.Ping(ctx).Result()
            case "nats":
                if !h.natsConn.IsConnected() { err = fmt.Errorf("not connected") }
            }

            svc.LatencyMs = time.Since(start).Milliseconds()
            if err != nil { svc.Status = "unhealthy"; svc.Error = err.Error() } else { svc.Status = "healthy" }
        }(svcName)
    }
    wg.Wait()

    overall := "healthy"
    for _, svc := range services {
        if svc.Status == "unhealthy" { overall = "unhealthy"; break }
    }

    statusCode := http.StatusOK
    if overall == "unhealthy" { statusCode = http.StatusServiceUnavailable }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]any{
        "status":    overall,
        "services":  services,
        "timestamp": time.Now().UTC(),
    })
}
```

### MODIFY `router.go` — add search + health + ontology routes

```go
// Additional routes to add:
r.Post("/v1/graphiti/search",          deps.Search.Search)
r.Post("/v1/graphiti/search/advanced", deps.Search.SearchAdvanced)
r.Get("/v1/graphiti/search/nodes",     deps.Search.SearchNodes)

r.Post("/v1/graphiti/ontology/{group_id}",        deps.Ontology.Save)
r.Get("/v1/graphiti/ontology/{group_id}",         deps.Ontology.Get)
r.Delete("/v1/graphiti/ontology/{group_id}",      deps.Ontology.Delete)
r.Post("/v1/graphiti/ontology/{group_id}/preset", deps.Ontology.ApplyPreset)

// Health (no auth)
r.Get("/healthz", deps.Health.HealthZ)
```

---

## Verification

```bash
cd gateway
go build ./...
```

**Acceptance tests:**
1. `POST /v1/graphiti/search` → 200 with facts array
2. `GET /healthz` all services up → `{"status":"healthy"}` 200
3. `GET /healthz` Redis down → `{"status":"unhealthy","services":{"redis":{"status":"unhealthy"}}}` 503
4. Search with `valid_at` filter → only edges valid at that time returned
