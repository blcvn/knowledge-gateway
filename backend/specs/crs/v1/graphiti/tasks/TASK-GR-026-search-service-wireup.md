# TASK-GR-026 — graphiti-search gRPC Service Wire-Up

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-026 |
| **Wave** | 4 (Admin & Observability) |
| **Component** | `services/graphiti-search/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-004 §8 |
| **Priority** | High |
| **Depends On** | TASK-GR-013, TASK-GR-025 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-search: 32 .go - service wire-up complete  
---

## Context

Wire up `graphiti-search` gRPC service: `main.go`, DI container, gRPC server handler, connection to `graphiti-store` gRPC. Include search metrics middleware.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-search/main.go` |
| CREATE | `services/graphiti-search/internal/adapter/grpc/handler.go` |
| CREATE | `services/graphiti-search/internal/adapter/grpc/store_client.go` |
| CREATE | `services/graphiti-search/internal/infra/metrics/prometheus.go` |

---

## Implementation

### File 1: `services/graphiti-search/internal/infra/metrics/prometheus.go`

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    SearchRequests = promauto.NewCounterVec(prometheus.CounterOpts{
        Namespace: "graphiti", Subsystem: "search", Name: "requests_total",
        Help: "Total search requests",
    }, []string{"recipe", "status"})

    SearchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Namespace: "graphiti", Subsystem: "search", Name: "duration_seconds",
        Help: "Search request duration",
        Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5},
    }, []string{"recipe"})

    SearchCacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
        Namespace: "graphiti", Subsystem: "search", Name: "cache_hits_total",
        Help: "Search cache hits",
    }, []string{"recipe"})

    SearchResultsReturned = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Namespace: "graphiti", Subsystem: "search", Name: "results_returned",
        Help: "Number of results returned per search",
        Buckets: []float64{0, 1, 5, 10, 20, 50},
    }, []string{"result_type"})
)
```

### File 2: `services/graphiti-search/internal/adapter/grpc/store_client.go`

```go
package grpc

import (
    "context"

    storepb "github.com/vnp-memory/api/proto/graphiti/store/v1"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-search/internal/usecase"
)

// StoreGRPCClient adapts storepb.StoreServiceClient to usecase.StorePort interface
type StoreGRPCClient struct {
    client storepb.StoreServiceClient
}

func NewStoreGRPCClient(client storepb.StoreServiceClient) *StoreGRPCClient {
    return &StoreGRPCClient{client: client}
}

func (s *StoreGRPCClient) EdgeSimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.EntityEdge, error) {
    resp, err := s.client.EdgeSimilaritySearch(ctx, &storepb.EdgeSimilaritySearchRequest{
        Vector: vector, GroupIds: groupIDs, Limit: int32(limit), MinScore: minScore,
    })
    if err != nil { return nil, err }
    return protoEdgesToGraph(resp.Edges), nil
}

func (s *StoreGRPCClient) EdgeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, filters any) ([]*graph.EntityEdge, error) {
    resp, err := s.client.EdgeFulltextSearch(ctx, &storepb.EdgeFulltextSearchRequest{
        Query: query, GroupIds: groupIDs, Limit: int32(limit),
    })
    if err != nil { return nil, err }
    return protoEdgesToGraph(resp.Edges), nil
}

func (s *StoreGRPCClient) NodeSimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.EntityNode, error) {
    resp, err := s.client.NodeSimilaritySearch(ctx, &storepb.NodeSimilaritySearchRequest{
        Vector: vector, GroupIds: groupIDs, Limit: int32(limit), MinScore: minScore,
    })
    if err != nil { return nil, err }
    return protoNodesToGraph(resp.Nodes), nil
}

func (s *StoreGRPCClient) NodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.EntityNode, error) {
    resp, err := s.client.NodeFulltextSearch(ctx, &storepb.NodeFulltextSearchRequest{
        Query: query, GroupIds: groupIDs, Limit: int32(limit),
    })
    if err != nil { return nil, err }
    return protoNodesToGraph(resp.Nodes), nil
}

func (s *StoreGRPCClient) EdgeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityEdge, error) {
    resp, err := s.client.EdgeBFSSearch(ctx, &storepb.EdgeBFSSearchRequest{
        OriginUuids: originUUIDs, MaxDepth: int32(maxDepth), GroupIds: groupIDs, Limit: int32(limit),
    })
    if err != nil { return nil, err }
    return protoEdgesToGraph(resp.Edges), nil
}

func (s *StoreGRPCClient) NodeDistanceReranker(ctx context.Context, nodeUUIDs []string, centerUUID string) (map[string]float64, error) {
    resp, err := s.client.NodeDistanceReranker(ctx, &storepb.NodeDistanceRerankerRequest{
        NodeUuids: nodeUUIDs, CenterUuid: centerUUID,
    })
    if err != nil { return nil, err }
    return resp.Scores, nil
}

func (s *StoreGRPCClient) EpisodeMentionsReranker(ctx context.Context, nodeUUIDs []string) (map[string]int, error) {
    resp, err := s.client.EpisodeMentionsReranker(ctx, &storepb.EpisodeMentionsRerankerRequest{
        NodeUuids: nodeUUIDs,
    })
    if err != nil { return nil, err }
    counts := make(map[string]int, len(resp.Counts))
    for k, v := range resp.Counts { counts[k] = int(v) }
    return counts, nil
}

// Proto conversion helpers
func protoEdgesToGraph(edges []*storepb.EntityEdge) []*graph.EntityEdge {
    result := make([]*graph.EntityEdge, 0, len(edges))
    for _, e := range edges {
        edge := &graph.EntityEdge{
            UUID:           e.Uuid,
            SourceNodeUUID: e.SourceNodeUuid,
            TargetNodeUUID: e.TargetNodeUuid,
            Name:           e.Name,
            Fact:           e.Fact,
            FactEmbedding:  e.FactEmbedding,
            GroupID:        e.GroupId,
            Episodes:       e.Episodes,
        }
        result = append(result, edge)
    }
    return result
}

func protoNodesToGraph(nodes []*storepb.EntityNode) []*graph.EntityNode {
    result := make([]*graph.EntityNode, 0, len(nodes))
    for _, n := range nodes {
        node := &graph.EntityNode{
            UUID:          n.Uuid,
            Name:          n.Name,
            Labels:        n.Labels,
            Summary:       n.Summary,
            GroupID:       n.GroupId,
            NameEmbedding: n.NameEmbedding,
        }
        result = append(result, node)
    }
    return result
}
```

### File 3: `services/graphiti-search/internal/adapter/grpc/handler.go`

```go
package grpc

import (
    "context"
    "time"

    pb "github.com/vnp-memory/api/proto/graphiti/search/v1"
    "github.com/vnp-memory/services/graphiti-search/internal/domain"
    "github.com/vnp-memory/services/graphiti-search/internal/infra/metrics"
    "github.com/vnp-memory/services/graphiti-search/internal/usecase"
)

type SearchHandler struct {
    pb.UnimplementedSearchServiceServer
    searchUC *usecase.SearchUseCase
}

func NewSearchHandler(uc *usecase.SearchUseCase) *SearchHandler {
    return &SearchHandler{searchUC: uc}
}

func (h *SearchHandler) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
    start := time.Now()

    result, err := h.searchUC.Execute(ctx, domain.SearchRequest{
        Query:          req.Query,
        Config:         domain.EdgeHybridSearchRRF,
        CenterNodeUUID: req.CenterNodeUuid,
        Filters: domain.SearchFilters{
            GroupIDs: req.GroupIds,
        },
    })
    if result != nil { result.LatencyMs = time.Since(start).Milliseconds() }

    if err != nil {
        metrics.SearchRequests.WithLabelValues("edge_hybrid_rrf", "error").Inc()
        return nil, err
    }

    metrics.SearchRequests.WithLabelValues("edge_hybrid_rrf", "success").Inc()
    metrics.SearchDuration.WithLabelValues("edge_hybrid_rrf").Observe(time.Since(start).Seconds())
    metrics.SearchResultsReturned.WithLabelValues("edge").Observe(float64(len(result.Edges)))

    return toProtoResponse(result), nil
}

func (h *SearchHandler) SearchAdvanced(ctx context.Context, req *pb.SearchAdvancedRequest) (*pb.SearchResponse, error) {
    start := time.Now()
    recipe := req.SearchConfigName
    if recipe == "" { recipe = "edge_hybrid_rrf" }

    cfg, ok := domain.RecipeByName[recipe]
    if !ok { cfg = domain.EdgeHybridSearchRRF }

    searchReq := domain.SearchRequest{
        Query:          req.Query,
        Config:         cfg,
        CenterNodeUUID: req.CenterNodeUuid,
        Filters: domain.SearchFilters{
            GroupIDs: req.GroupIds,
        },
    }
    if req.ValidAt != "" {
        if t, err := time.Parse(time.RFC3339, req.ValidAt); err == nil { searchReq.Filters.ValidAt = &t }
    }

    result, err := h.searchUC.Execute(ctx, searchReq)
    if err != nil {
        metrics.SearchRequests.WithLabelValues(recipe, "error").Inc()
        return nil, err
    }

    metrics.SearchRequests.WithLabelValues(recipe, "success").Inc()
    metrics.SearchDuration.WithLabelValues(recipe).Observe(time.Since(start).Seconds())

    if result != nil { result.LatencyMs = time.Since(start).Milliseconds() }
    return toProtoResponse(result), nil
}

func (h *SearchHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
    return &pb.HealthCheckResponse{Status: "healthy"}, nil
}

func toProtoResponse(r *domain.SearchResults) *pb.SearchResponse {
    if r == nil { return &pb.SearchResponse{} }
    resp := &pb.SearchResponse{LatencyMs: r.LatencyMs}
    for _, e := range r.Edges {
        if edge, ok := e.(*graph.EntityEdge); ok {
            resp.Edges = append(resp.Edges, &pb.EdgeResult{
                Uuid: edge.UUID, SourceUuid: edge.SourceNodeUUID,
                TargetUuid: edge.TargetNodeUUID, Fact: edge.Fact, Name: edge.Name,
            })
        }
    }
    return resp
}
```

### File 4: `services/graphiti-search/main.go`

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/prometheus/client_golang/prometheus/promhttp"
    "google.golang.org/grpc"
    pb "github.com/vnp-memory/api/proto/graphiti/search/v1"
    storepb "github.com/vnp-memory/api/proto/graphiti/store/v1"
    graphitigrpc "github.com/vnp-memory/services/graphiti-search/internal/adapter/grpc"
    "github.com/vnp-memory/services/graphiti-search/internal/adapter/cache"
    "github.com/vnp-memory/services/graphiti-search/internal/usecase"
    "github.com/vnp-memory/services/graphiti-search/internal/domain"
    "github.com/redis/go-redis/v9"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Connect to graphiti-store
    storeAddr := os.Getenv("GRAPHITI_STORE_ADDR")
    if storeAddr == "" { storeAddr = "localhost:9090" }
    storeConn, err := grpc.Dial(storeAddr, grpc.WithInsecure())
    if err != nil { log.Fatalf("connect store: %v", err) }
    defer storeConn.Close()

    storeClient := storepb.NewStoreServiceClient(storeConn)
    storePort   := graphitigrpc.NewStoreGRPCClient(storeClient)

    // Redis cache
    redisAddr := os.Getenv("REDIS_ADDR")
    if redisAddr == "" { redisAddr = "localhost:6379" }
    redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
    searchCache := cache.NewRedisSearchCache(redisClient)

    // Knowledge client (for embeddings + rerank)
    knowledgeAddr := os.Getenv("GRAPHITI_KNOWLEDGE_ADDR")
    knowledgeClient := newKnowledgeGRPCClient(knowledgeAddr)

    // Wire up use cases
    edgeUC   := usecase.NewSearchEdgesUseCase(storePort, knowledgeClient)
    nodeUC   := usecase.NewSearchNodesUseCase(storePort, knowledgeClient)
    searchUC := usecase.NewSearchUseCase(edgeUC, nodeUC, nil, nil, searchCache)

    // gRPC server
    grpcPort := os.Getenv("GRAPHITI_SEARCH_GRPC_PORT")
    if grpcPort == "" { grpcPort = "9098" }

    lis, err := net.Listen("tcp", ":"+grpcPort)
    if err != nil { log.Fatalf("listen: %v", err) }

    server := grpc.NewServer()
    pb.RegisterSearchServiceServer(server, graphitigrpc.NewSearchHandler(searchUC))

    // Prometheus metrics server
    go func() {
        metricsPort := os.Getenv("METRICS_PORT")
        if metricsPort == "" { metricsPort = "9099" }
        http.Handle("/metrics", promhttp.Handler())
        http.ListenAndServe(":"+metricsPort, nil)
    }()

    go func() {
        log.Printf("graphiti-search gRPC listening on :%s", grpcPort)
        if err := server.Serve(lis); err != nil { log.Fatalf("serve: %v", err) }
    }()

    <-ctx.Done()
    server.GracefulStop()
    log.Println("graphiti-search stopped")
}
```

---

## Verification

```bash
cd services/graphiti-search
go build ./...
go vet ./...

# Verify gRPC service reflection
grpcurl -plaintext localhost:9098 list
# Expected: graphiti.search.v1.SearchService
```
