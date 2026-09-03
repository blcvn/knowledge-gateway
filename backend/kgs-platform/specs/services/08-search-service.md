# search-service — Hybrid Search Service

> **Role:** Thực hiện hybrid search trên Knowledge Graph kết hợp vector semantic search (Qdrant), full-text search (PostgreSQL), và graph centrality scoring (Neo4j).

---

## 1. Trách Nhiệm (Single Responsibility)

`search-service` chịu trách nhiệm **duy nhất** cho:
- **Vector Search**: Tìm kiếm semantic qua Qdrant (embedding similarity)
- **Full-Text Search**: Tìm kiếm text qua PostgreSQL (full-text index)
- **Hybrid Blending**: Kết hợp kết quả bằng Reciprocal Rank Fusion (RRF)
- **Centrality Re-ranking**: Boost kết quả theo graph importance score từ Neo4j
- **Filter & Post-processing**: Filter theo entity type, domain, min confidence

---

## 2. Kiến Trúc Nội Tại

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         search-service                                   │
│                                                                         │
│  gRPC Server (port 9007)                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                  SearchServiceServer                              │   │
│  │                                                                  │   │
│  │  Search()                [main hybrid search]                    │   │
│  │  VectorSearch()          [pure vector search]                    │   │
│  │  TextSearch()            [pure full-text search]                 │   │
│  │  SimilarNodes()          [find similar nodes by embedding]       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Hybrid Search Pipeline                          │   │
│  │                                                                  │   │
│  │  Input: {namespace, query_text, options}                         │   │
│  │         │                                                        │   │
│  │         ├── EmbeddingProvider.Embed(query_text)                  │   │
│  │         │         ↓                                              │   │
│  │         ├── VectorRetriever.Search()   → Qdrant results (top K) │   │
│  │         ├── TextRetriever.Search()     → PG full-text (top K)   │   │
│  │         │                                                        │   │
│  │         ├── Blend() [Reciprocal Rank Fusion]                    │   │
│  │         │   score = alpha * vector_rank + (1-alpha) * text_rank │   │
│  │         │                                                        │   │
│  │         ├── CentralityScorer.Scores()  → Neo4j PageRank/Degree  │   │
│  │         ├── RerankWithCentrality()                               │   │
│  │         │   final_score = blended + beta * centrality_score      │   │
│  │         │                                                        │   │
│  │         └── ApplyFilters() → entity_types, domains, min_conf    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Data Access                                     │   │
│  │  Qdrant:     Vector similarity search                            │   │
│  │  PostgreSQL: Full-text search (tsvector columns)                 │   │
│  │  Neo4j:      Centrality scores (PageRank, Degree)                │   │
│  │  Redis:      Embedding cache (query → vector, TTL=1h)            │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Search Pipeline Detail

### 3.1 Step 1: Query Embedding

```go
// Cache-first embedding lookup
func (s *SearchService) GetEmbedding(query string) ([]float32, error) {
    cacheKey := fmt.Sprintf("embed:%s", sha256(query))
    if cached := s.redis.Get(cacheKey); cached != nil {
        return cached, nil
    }
    vector, err := s.embeddingProvider.Embed(query)
    s.redis.Set(cacheKey, vector, 1*time.Hour)
    return vector, err
}
```

### 3.2 Step 2: Parallel Retrieval

```go
// VectorRetriever (Qdrant)
results, _ = qdrantClient.Search(SearchRequest{
    CollectionName: "kgs_ba_agent",
    Vector:         queryVector,
    Filter: Filter{
        Must: []Condition{
            {Key: "namespace", Match: namespace},
            {Key: "entity_type", Match: entityTypeFilter}, // optional
        },
    },
    Limit: topK * 3, // Over-fetch for re-ranking
})

// TextRetriever (PostgreSQL full-text)
results, _ = db.Query(`
    SELECT id, entity_type, properties_json,
           ts_rank(search_vector, plainto_tsquery($1)) AS rank
    FROM kg_entities
    WHERE app_id = $2
      AND namespace = $3
      AND search_vector @@ plainto_tsquery($1)
    ORDER BY rank DESC
    LIMIT $4
`, queryText, appID, namespace, topK*3)
```

### 3.3 Step 3: Reciprocal Rank Fusion (RRF)

```go
// RRF formula: score(d) = Σ 1 / (k + rank(d))
// k = 60 (constant), alpha controls vector vs text weight

func Blend(vectorResults, textResults []SearchResult, alpha float64) []SearchResult {
    scores := make(map[string]float64)

    for rank, r := range vectorResults {
        scores[r.ID] += alpha * (1.0 / (60.0 + float64(rank+1)))
    }
    for rank, r := range textResults {
        scores[r.ID] += (1-alpha) * (1.0 / (60.0 + float64(rank+1)))
    }

    return sortByScore(scores)
}
```

### 3.4 Step 4: Centrality Re-ranking

```go
// CentralityScorer từ Neo4j
func (c *CentralityScorer) Scores(nodeIDs []string, namespace string) map[string]float64 {
    // Sử dụng Neo4j GDS hoặc tính toán degree centrality:
    result, _ := session.Run(`
        UNWIND $node_ids AS nid
        MATCH (n) WHERE n.id = nid
        RETURN n.id AS id, size((n)--()) AS degree
    `, map[string]any{"node_ids": nodeIDs})

    // Normalize degree to [0, 1]
    return normalizeDegrees(result)
}

// Re-rank: final_score = blended_score + beta * centrality
// beta = 0.2 (configurable)
```

### 3.5 Step 5: Filter & Return

```go
type SearchOptions struct {
    TopK            int
    Alpha           float64  // Vector weight [0,1], default 0.7
    Beta            float64  // Centrality weight [0,1], default 0.2
    EntityTypes     []string // Filter by entity type
    MinConfidence   float64  // Minimum final score threshold
    ProvenanceTypes []string // Filter by provenance (manual, agent, ...)
}
```

---

## 4. gRPC API

```protobuf
service SearchService {
  // Main hybrid search
  rpc Search(SearchRequest) returns (SearchResponse);

  // Specialized searches
  rpc VectorSearch(VectorSearchRequest) returns (SearchResponse);
  rpc TextSearch(TextSearchRequest) returns (SearchResponse);
  rpc SimilarNodes(SimilarNodesRequest) returns (SearchResponse);
}

message SearchRequest {
  string query_text = 1;
  SearchOptions options = 2;
  // app_id, namespace từ gRPC metadata
}

message SearchOptions {
  int32 top_k = 1;              // Default: 10
  double alpha = 2;             // Vector weight, default: 0.7
  double beta = 3;              // Centrality weight, default: 0.2
  repeated string entity_types = 4;    // Optional filter
  repeated string domains = 5;         // Optional domain filter
  double min_confidence = 6;   // Minimum score threshold
}

message SearchResponse {
  repeated SearchResult results = 1;
  int32 total_found = 2;
  SearchDebugInfo debug = 3;   // Optional debug info
}

message SearchResult {
  string node_id = 1;
  string entity_type = 2;
  bytes properties_json = 3;
  double score = 4;
  double vector_score = 5;     // Debug: vector component
  double text_score = 6;       // Debug: text component
  double centrality_score = 7; // Debug: centrality component
  string match_reason = 8;     // "vector" | "text" | "both"
}

message SimilarNodesRequest {
  string source_node_id = 1;   // Find nodes similar to this
  SearchOptions options = 2;
}
```

---

## 5. Search Examples

### 5.1 Hybrid Search

```json
// POST /v1/search
{
  "query_text": "authentication login user credentials",
  "options": {
    "top_k": 10,
    "alpha": 0.7,
    "beta": 0.2,
    "entity_types": ["Requirement", "UseCase"],
    "min_confidence": 0.5
  }
}

// Response:
{
  "results": [
    {
      "node_id": "ba_agent__Requirement__REQ-AUTH-001",
      "entity_type": "Requirement",
      "score": 0.92,
      "properties": { "req_id": "REQ-AUTH-001", "title": "Đăng nhập bằng email/password", ... },
      "match_reason": "both"
    },
    ...
  ],
  "total_found": 47
}
```

### 5.2 Similar Nodes

```json
// POST /v1/search/similar
{
  "source_node_id": "ba_agent__Requirement__REQ-AUTH-001",
  "options": {
    "top_k": 5,
    "entity_types": ["Requirement"]
  }
}
// Finds requirements semantically similar to REQ-AUTH-001
```

---

## 6. Embedding Providers

```go
// Factory pattern — same providers as sync-worker-service
type EmbeddingProvider interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
    Dimension() int
    Name() string
}

// Available:
// - openai:    OpenAI text-embedding-ada-002 (dim=1536)
// - aiproxy:   VNP AI Proxy (OpenAI-compatible)
// - air-vnp:   Local VNP embedding model
```

---

## 7. Qdrant Collection Setup

Mỗi app có một Qdrant collection riêng:

```
Collection name: "kgs_{app_id}"
  - Vector dimension: 1536 (hoặc theo embedding provider)
  - Distance metric: COSINE
  - Payload fields (indexed):
    - namespace: keyword
    - entity_type: keyword
    - app_id: keyword
    - created_at: datetime
```

---

## 8. HTTP REST Endpoints (Exposed qua Gateway)

| Method | Path | Scope | Mô tả |
|--------|------|-------|-------|
| POST | `/v1/search` | `graph:read` | Hybrid search |
| POST | `/v1/search/vector` | `graph:read` | Pure vector search |
| POST | `/v1/search/text` | `graph:read` | Pure text search |
| POST | `/v1/search/similar` | `graph:read` | Similar nodes |

---

## 9. Configuration

```yaml
# configs/search.yaml
search_service:
  grpc_port: 9007

  embedding:
    provider: air-vnp
    endpoint: http://air-vnp:8080
    dimension: 1536
    cache_ttl: 1h

  qdrant:
    addr: qdrant:6334
    api_key: ""
    default_collection_prefix: "kgs_"

  database:
    dsn: "postgres://kgs:password@postgres:5432/kgs_graph"
    full_text_language: "vietnamese"

  neo4j:
    uri: bolt://neo4j:7687
    username: neo4j
    password: secret
    centrality:
      algorithm: degree   # degree | pagerank
      cache_ttl: 5m

  redis:
    addr: redis:6379

  defaults:
    top_k: 10
    alpha: 0.7
    beta: 0.2
    min_confidence: 0.3
    max_top_k: 100

  observability:
    metrics_port: 9097
```

---

## 10. Observability

| Metric | Mô tả |
|--------|-------|
| `search_requests_total{app_id, search_type}` | Số search requests |
| `search_duration_seconds{app_id, phase}` | Latency theo từng phase |
| `search_results_total{app_id}` | Số kết quả trả về |
| `search_embedding_cache_hits_total` | Embedding cache hit rate |
| `search_vector_results_total{app_id}` | Số kết quả từ Qdrant |
| `search_text_results_total{app_id}` | Số kết quả từ PostgreSQL |
| `search_centrality_scores_fetched_total` | Số lần lấy centrality |
