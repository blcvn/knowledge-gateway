# TASK-GR-017 — Gateway: Episode + Triplet + Saga REST Routes

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-017 |
| **Wave** | 3 |
| **Component** | `gateway/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-006 §2.1, §6 |
| **Priority** | High |
| **Depends On** | TASK-GR-011 |
| **Estimated** | 4h |

---

## Context

Implement REST handlers cho episode, triplet, và saga endpoints trong gateway. Mỗi handler translate HTTP request → gRPC call → HTTP response. JWT auth middleware tái dụng từ kiến trúc hiện tại.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `gateway/internal/adapter/handler/graphiti/episode_handler.go` |
| CREATE | `gateway/internal/adapter/handler/graphiti/triplet_handler.go` |
| CREATE | `gateway/internal/adapter/handler/graphiti/saga_handler.go` |
| CREATE | `gateway/internal/middleware/group_id.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### File 1: `gateway/internal/middleware/group_id.go`

```go
package middleware

import (
    "context"
    "net/http"

    "google.golang.org/grpc/metadata"
)

type groupIDKey struct{}

// ExtractGroupID extracts group_id from JWT claim, header, or query param (priority order).
func ExtractGroupID(r *http.Request) string {
    // 1. JWT claim "group_id"
    if claims := JWTClaimsFromCtx(r.Context()); claims != nil {
        if id, ok := claims["group_id"].(string); ok && id != "" { return id }
        if id, ok := claims["tenant_id"].(string); ok && id != "" { return id }
    }
    // 2. Header X-Group-ID
    if id := r.Header.Get("X-Group-ID"); id != "" { return id }
    // 3. Query param
    if id := r.URL.Query().Get("group_id"); id != "" { return id }
    return "default"
}

// WithGroupIDMeta injects group_id into gRPC outgoing metadata
func WithGroupIDMeta(ctx context.Context, groupID string) context.Context {
    return metadata.AppendToOutgoingContext(ctx, "x-group-id", groupID)
}
```

### File 2: `gateway/internal/adapter/handler/graphiti/episode_handler.go`

```go
package graphiti

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"
    ingestionpb "github.com/vnp-memory/api/proto/graphiti/ingestion/v1"
    "github.com/vnp-memory/gateway/internal/middleware"
)

type EpisodeHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
}

func NewEpisodeHandler(client ingestionpb.IngestionServiceClient) *EpisodeHandler {
    return &EpisodeHandler{ingestionClient: client}
}

// POST /v1/graphiti/episodes
func (h *EpisodeHandler) IngestEpisode(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Name              string `json:"name"`
        Body              string `json:"body"`
        Source            string `json:"source"`
        SourceDescription string `json:"source_description"`
        SagaID            string `json:"saga_id"`
        PrevEpisodeUUID   string `json:"prev_episode_uuid"`
        ReferenceTime     string `json:"reference_time"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if body.Body == "" {
        respondError(w, http.StatusBadRequest, "body is required")
        return
    }

    groupID := middleware.ExtractGroupID(r)
    ctx := middleware.WithGroupIDMeta(r.Context(), groupID)

    source := body.Source
    if source == "" { source = "text" }

    req := &ingestionpb.IngestEpisodeRequest{
        Name:              body.Name,
        Body:              body.Body,
        Source:            source,
        SourceDescription: body.SourceDescription,
        SagaId:            body.SagaID,
        PrevEpisodeUuid:   body.PrevEpisodeUUID,
    }
    if body.ReferenceTime != "" { req.ReferenceTime = body.ReferenceTime }

    resp, err := h.ingestionClient.IngestEpisode(ctx, req)
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }

    respondJSON(w, http.StatusOK, map[string]any{
        "episode_uuid": resp.EpisodeUuid,
        "stats": map[string]any{
            "entities_extracted": resp.Stats.EntitiesExtracted,
            "entities_new":       resp.Stats.EntitiesNew,
            "edges_extracted":    resp.Stats.EdgesExtracted,
            "edges_new":          resp.Stats.EdgesNew,
            "processing_time_ms": resp.Stats.ProcessingTimeMs,
        },
    })
}

// DELETE /v1/graphiti/episodes/{uuid}
func (h *EpisodeHandler) RemoveEpisode(w http.ResponseWriter, r *http.Request) {
    episodeUUID := chi.URLParam(r, "uuid")
    if episodeUUID == "" { respondError(w, http.StatusBadRequest, "uuid required"); return }

    groupID := middleware.ExtractGroupID(r)
    ctx := middleware.WithGroupIDMeta(r.Context(), groupID)

    _, err := h.ingestionClient.RemoveEpisode(ctx, &ingestionpb.RemoveEpisodeRequest{EpisodeUuid: episodeUUID})
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    w.WriteHeader(http.StatusNoContent)
}

// GET /v1/graphiti/episodes
func (h *EpisodeHandler) ListEpisodes(w http.ResponseWriter, r *http.Request) {
    groupID := middleware.ExtractGroupID(r)
    ctx := middleware.WithGroupIDMeta(r.Context(), groupID)

    lastN := 10
    if n := r.URL.Query().Get("last_n"); n != "" {
        if v, err := strconv.Atoi(n); err == nil && v > 0 { lastN = v }
    }
    if lastN > 100 { lastN = 100 }

    resp, err := h.ingestionClient.ListEpisodes(ctx, &ingestionpb.ListEpisodesRequest{
        GroupId: groupID, LastN: int32(lastN),
        SagaId:  r.URL.Query().Get("saga_id"),
        Source:  r.URL.Query().Get("source"),
    })
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, resp)
}

// POST /v1/graphiti/episodes/bulk
func (h *EpisodeHandler) IngestEpisodeBulk(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Episodes []struct {
            Name              string `json:"name"`
            Body              string `json:"body"`
            Source            string `json:"source"`
            SourceDescription string `json:"source_description"`
            SagaID            string `json:"saga_id"`
        } `json:"episodes"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    groupID := middleware.ExtractGroupID(r)
    ctx := middleware.WithGroupIDMeta(r.Context(), groupID)

    var results []map[string]any
    for _, ep := range body.Episodes {
        source := ep.Source
        if source == "" { source = "text" }
        resp, err := h.ingestionClient.IngestEpisode(ctx, &ingestionpb.IngestEpisodeRequest{
            Name: ep.Name, Body: ep.Body, Source: source,
            SourceDescription: ep.SourceDescription, SagaId: ep.SagaID,
        })
        if err != nil {
            results = append(results, map[string]any{"error": err.Error(), "name": ep.Name})
        } else {
            results = append(results, map[string]any{"episode_uuid": resp.EpisodeUuid, "name": ep.Name})
        }
    }
    respondJSON(w, http.StatusOK, map[string]any{"results": results, "total": len(results)})
}

// Helpers
func respondJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}

func respondError(w http.ResponseWriter, status int, msg string) {
    respondJSON(w, status, map[string]string{"error": msg})
}
```

### File 3: `gateway/internal/adapter/handler/graphiti/triplet_handler.go`

```go
package graphiti

import (
    "encoding/json"
    "net/http"

    ingestionpb "github.com/vnp-memory/api/proto/graphiti/ingestion/v1"
    "github.com/vnp-memory/gateway/internal/middleware"
)

type TripletHandler struct {
    ingestionClient ingestionpb.IngestionServiceClient
}

func NewTripletHandler(client ingestionpb.IngestionServiceClient) *TripletHandler {
    return &TripletHandler{ingestionClient: client}
}

// POST /v1/graphiti/triplets
func (h *TripletHandler) AddTriplet(w http.ResponseWriter, r *http.Request) {
    var body struct {
        SourceEntity string `json:"source_entity"`
        Relation     string `json:"relation"`
        TargetEntity string `json:"target_entity"`
        Fact         string `json:"fact"`
        ValidAt      string `json:"valid_at"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if body.SourceEntity == "" || body.Relation == "" || body.TargetEntity == "" {
        respondError(w, http.StatusBadRequest, "source_entity, relation, target_entity are required")
        return
    }

    fact := body.Fact
    if fact == "" { fact = body.SourceEntity + " " + body.Relation + " " + body.TargetEntity }

    groupID := middleware.ExtractGroupID(r)
    ctx     := middleware.WithGroupIDMeta(r.Context(), groupID)

    resp, err := h.ingestionClient.AddTriplet(ctx, &ingestionpb.AddTripletRequest{
        SourceEntity: body.SourceEntity,
        Relation:     body.Relation,
        TargetEntity: body.TargetEntity,
        Fact:         fact,
        ValidAt:      body.ValidAt,
    })
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, map[string]any{"episode_uuid": resp.EpisodeUuid})
}
```

### MODIFY: `gateway/internal/adapter/handler/router.go`

Add graphiti routes (insert into existing router setup):

```go
// In SetupRouter() or equivalent:
func setupGraphitiRoutes(r chi.Router, deps *GraphitiDeps) {
    r.Group(func(r chi.Router) {
        r.Use(deps.Auth.JWT)
        r.Use(deps.RateLimiter.Middleware)

        // Episodes
        r.Post("/v1/graphiti/episodes",         deps.Episode.IngestEpisode)
        r.Post("/v1/graphiti/episodes/bulk",     deps.Episode.IngestEpisodeBulk)
        r.Delete("/v1/graphiti/episodes/{uuid}", deps.Episode.RemoveEpisode)
        r.Get("/v1/graphiti/episodes",           deps.Episode.ListEpisodes)

        // Triplets
        r.Post("/v1/graphiti/triplets", deps.Triplet.AddTriplet)
    })
}
```

---

## Verification

```bash
cd gateway
go build ./...
go vet ./...
```

**Acceptance tests:**
1. `POST /v1/graphiti/episodes` valid JWT + body → 200 + `episode_uuid`
2. `POST /v1/graphiti/episodes` no JWT → 401 Unauthorized
3. `POST /v1/graphiti/episodes` empty body → 400 Bad Request
4. `DELETE /v1/graphiti/episodes/{uuid}` → 204 No Content
5. `GET /v1/graphiti/episodes?last_n=5` → array of 5 episodes
6. `POST /v1/graphiti/triplets` → 200 + `episode_uuid`
