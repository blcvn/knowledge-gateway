# TASK-CE-005 — NodeSets: Search Service (3 Retriever Filters)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-005 |
| **Wave** | 2 |
| **Component** | `services/cognee-search/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §2.5, §2.6, §2.7 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-CE-004 |
| **Estimated** | 3h |

---

## Context

Implement NodeSet filtering trong 3 retrievers của cognee-search:
1. **VectorRetriever** (SIMILARITY) — Qdrant `payload.node_sets` filter
2. **GraphRetriever** (GRAPH_COMPLETION) — Neo4j Cypher label predicate
3. **KeywordRetriever** (KEYWORD) — Neo4j fulltext + label filter

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `services/cognee-search/internal/usecase/search.go` |
| MODIFY | `services/cognee-search/internal/adapter/retriever/vector.go` |
| MODIFY | `services/cognee-search/internal/adapter/retriever/graph.go` |
| MODIFY | `services/cognee-search/internal/adapter/retriever/keyword.go` |
| MODIFY | `services/cognee-search/internal/adapter/grpc/handler.go` |
| MODIFY | `gateway/internal/adapter/handler/cognee_handler.go` |

---

## Implementation

### MODIFY `usecase/search.go` — Add NodeSets to SearchRequest

```go
// services/cognee-search/internal/usecase/search.go

type SearchRequest struct {
    Query           string
    Strategies      []SearchStrategy
    DatasetID       *uuid.UUID
    DatasetName     string
    TenantID        string
    NodeSets        []string   // [NEW] CR-002
    TopK            int
    SaveInteraction bool
    SessionID       *string
    FeedbackFor     *string
    FeedbackScore   *float64
    FeedbackText    string
}
```

### MODIFY `retriever/vector.go` — Qdrant Payload Filter

```go
// services/cognee-search/internal/adapter/retriever/vector.go

func (r *VectorRetriever) Retrieve(ctx context.Context, req usecase.SearchRequest) ([]usecase.SearchResult, error) {
    vec, err := r.embedder.Embed(ctx, req.Query)
    if err != nil { return nil, fmt.Errorf("embed query: %w", err) }

    // Build Qdrant filter
    filter := &qdrant.Filter{}

    // Existing: filter by dataset_id
    if req.DatasetID != nil {
        filter.Must = append(filter.Must, &qdrant.Condition{
            ConditionOneOf: &qdrant.Condition_Field{
                Field: &qdrant.FieldCondition{
                    Key:   "dataset_id",
                    Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: req.DatasetID.String()}},
                },
            },
        })
    }

    // [NEW] NodeSet filter — MUST match any of the given NodeSet tags in payload
    if len(req.NodeSets) > 0 {
        // Qdrant MatchAny: point payload.node_sets must contain at least one of the requested tags
        values := make([]string, len(req.NodeSets))
        copy(values, req.NodeSets)
        filter.Must = append(filter.Must, &qdrant.Condition{
            ConditionOneOf: &qdrant.Condition_Field{
                Field: &qdrant.FieldCondition{
                    Key: "node_sets",
                    Match: &qdrant.Match{
                        MatchValue: &qdrant.Match_Keywords{
                            Keywords: &qdrant.RepeatedStrings{Strings: values},
                        },
                    },
                },
            },
        })
    }

    collectionName := fmt.Sprintf("cognee_%s", req.TenantID)
    points, err := r.qdrantClient.Search(ctx, &qdrant.SearchPoints{
        CollectionName: collectionName,
        Vector:         vec,
        Filter:         filter,
        Limit:          uint64(req.TopK),
        WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
    })
    if err != nil { return nil, fmt.Errorf("qdrant search: %w", err) }

    return mapPointsToResults(points), nil
}
```

### MODIFY `retriever/graph.go` — Cypher Label Predicate Filter

```go
// services/cognee-search/internal/adapter/retriever/graph.go

func (r *GraphRetriever) Retrieve(ctx context.Context, req usecase.SearchRequest) ([]usecase.SearchResult, error) {
    params := map[string]any{
        "query":      req.Query,
        "tenant_id":  req.TenantID,
        "top_k":      req.TopK,
    }
    if req.DatasetID != nil { params["dataset_id"] = req.DatasetID.String() }

    var cypher string

    if len(req.NodeSets) > 0 {
        // [NEW] Filter by ALL NodeSet labels using Cypher label predicates
        // `all(tag IN $node_sets WHERE tag IN labels(n))` — ALL tags must be present
        params["node_sets"] = req.NodeSets
        cypher = `
            MATCH (n)-[r]->(m)
            WHERE n.tenant_id = $tenant_id
              AND ($dataset_id IS NULL OR n.dataset_id = $dataset_id)
              AND all(tag IN $node_sets WHERE tag IN labels(n))
            CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score
            WHERE node.id = n.id
            RETURN n, r, m, score
            ORDER BY score DESC
            LIMIT $top_k
        `
    } else {
        cypher = `
            MATCH (n)-[r]->(m)
            WHERE n.tenant_id = $tenant_id
              AND ($dataset_id IS NULL OR n.dataset_id = $dataset_id)
            CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score
            WHERE node.id = n.id
            RETURN n, r, m, score
            ORDER BY score DESC
            LIMIT $top_k
        `
    }

    return r.neo4jClient.QueryNodesAndEdges(ctx, cypher, params)
}
```

### MODIFY `retriever/keyword.go` — Keyword + Label Filter

```go
// services/cognee-search/internal/adapter/retriever/keyword.go

func (r *KeywordRetriever) Retrieve(ctx context.Context, req usecase.SearchRequest) ([]usecase.SearchResult, error) {
    params := map[string]any{
        "query":  req.Query,
        "top_k":  req.TopK,
    }

    cypher := `CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score`

    if len(req.NodeSets) > 0 {
        // [NEW] AND filter by NodeSet labels
        params["node_sets"] = req.NodeSets
        cypher += ` WHERE all(tag IN $node_sets WHERE tag IN labels(node))`
    }

    cypher += ` RETURN node, score ORDER BY score DESC LIMIT $top_k`

    return r.neo4jClient.QueryNodes(ctx, cypher, params)
}
```

### MODIFY `adapter/grpc/handler.go` — Pass NodeSets from proto

```go
// services/cognee-search/internal/adapter/grpc/handler.go

func (h *SearchHandler) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
    strategies := mapStrategiesToDomain(req.Strategies)

    var datasetID *uuid.UUID
    if req.DatasetId != "" {
        id, err := uuid.Parse(req.DatasetId)
        if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id") }
        datasetID = &id
    }

    result, err := h.searchUC.Execute(ctx, usecase.SearchRequest{
        Query:       req.Query,
        Strategies:  strategies,
        DatasetID:   datasetID,
        DatasetName: req.DatasetName,
        TenantID:    req.TenantId,
        NodeSets:    req.NodeSets,  // [NEW] propagate node_sets
        TopK:        int(req.TopK),
        SaveInteraction: req.SaveInteraction,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "search: %v", err) }

    return toProtoResponse(result), nil
}
```

### MODIFY `gateway/cognee_handler.go` — Parse node_sets from JSON

```go
// gateway/internal/adapter/handler/cognee_handler.go

func (h *CogneeHandler) Search(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Query      string   `json:"query"`
        Strategies []string `json:"strategies"`
        DatasetName string  `json:"dataset_name"`
        NodeSets   []string `json:"node_sets"`     // [NEW]
        TopK       int      `json:"top_k"`
        SaveInteraction bool `json:"save_interaction"` // from CR-005
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        respondError(w, http.StatusBadRequest, "invalid body")
        return
    }
    if body.Query == "" { respondError(w, http.StatusBadRequest, "query is required"); return }

    // Forward to cognee-search gRPC with node_sets
    resp, err := h.searchClient.Search(r.Context(), &searchpb.SearchRequest{
        Query:           body.Query,
        Strategies:      body.Strategies,
        DatasetName:     body.DatasetName,
        TenantId:        tenantIDFromContext(r.Context()),
        NodeSets:        body.NodeSets,         // [NEW]
        TopK:            int32(body.TopK),
        SaveInteraction: body.SaveInteraction,
    })
    if err != nil { respondError(w, http.StatusInternalServerError, err.Error()); return }
    respondJSON(w, http.StatusOK, resp)
}
```

---

## Verification

```bash
cd services/cognee-search
go build ./...
go test ./internal/adapter/retriever/... -run TestNodeSets -v
```

**Integration test:**
```go
func TestSearch_NodeSets_FiltersResults(t *testing.T) {
    // Setup: ingest data with node_sets=["project_alpha"] and ["project_beta"]
    // Search with node_sets=["project_alpha"]
    // Expect: only results tagged with "project_alpha" returned
    
    resp, err := searchClient.Search(ctx, &searchpb.SearchRequest{
        Query:    "technology stack",
        TenantId: "tenant-1",
        NodeSets: []string{"project_alpha"},
        TopK:     10,
    })
    require.NoError(t, err)
    // All results should have "project_alpha" label
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| SIMILARITY search với `node_sets` → Qdrant filtered results | ✅ |
| GRAPH_COMPLETION với `node_sets` → Cypher label predicate applied | ✅ |
| KEYWORD search với `node_sets` → fulltext + label filter applied | ✅ |
| Search không truyền `node_sets` → full results (no filter) | ✅ |
| Performance: Qdrant native payload filter (không scan toàn bộ) | ✅ |
