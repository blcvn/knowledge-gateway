# Change Request: CR-ZEP-004 — Semantic Graph Search & 5 Reranking Strategies

**CR ID:** CR-ZEP-004  
**Component:** `services/search-service` [UPGRADE SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** Zep PRD §6.1 F4, SRS §5.4, specs/services/06-search-service.md

---

## 1. Mô tả

Nâng cấp Search Service của VNP Memory để hỗ trợ **Graph Search** đa scope với 5 reranking strategies:

1. **Multi-Scope Search**: `edges` (facts), `nodes` (entities), `episodes` (events), `all`.
2. **5 Reranking Strategies**: RRF, MMR, Cross-Encoder, Node Distance, Episode Mentions.
3. **Search Filters**: Filter theo `node_labels`, `edge_types`, `min_fact_rating`.
4. **Session Search**: Tìm kiếm qua nhiều sessions.
5. **Redis Cache**: Cache kết quả search (TTL 30s) với invalidation từ graph events.
6. **GetRelevantFacts**: Internal endpoint cho Memory Service dùng để assemble context.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện chỉ có vector similarity search, chưa có **graph-aware search**.
- Chỉ có 1 ranking strategy, chưa có 5 strategies của Zep.
- Thiếu **label filtering** và **edge type filtering** để tìm kiếm chính xác.
- Chưa có **Session Search** (tìm qua nhiều sessions).

---

## 3. Thay đổi đề xuất

### 3.1. [UPGRADE] `services/search-service/` (Port gRPC: 9045)

### 3.2. Search Scope

```go
type SearchScope string
const (
    SearchScopeEdges    SearchScope = "edges"    // search temporal facts
    SearchScopeNodes    SearchScope = "nodes"    // search entity nodes
    SearchScopeEpisodes SearchScope = "episodes" // search temporal events
    SearchScopeAll      SearchScope = "all"      // expensive — search everything
)
```

### 3.3. 5 Reranking Strategies

| Strategy | Thuật toán | Latency | Tốt nhất cho |
|----------|-----------|---------|-------------|
| `rrf` | Reciprocal Rank Fusion | Thấp | General-purpose, balanced |
| `mmr` | Maximal Marginal Relevance | Thấp | Diverse results, tránh trùng lặp |
| `cross_encoder` | Neural cross-encoder | Cao | Best accuracy, critical queries |
| `node_distance` | Graph shortest path | Trung bình | Relationship-aware search |
| `episode_mentions` | Episode co-occurrence | Thấp | Recency-aware, trending topics |

### 3.4. Search Query Model

```go
type GraphSearchQuery struct {
    Query          string
    UserID         *string       // scope to user's graph
    GroupIDs       []string      // scope to specific groups
    Scope          SearchScope
    Reranker       RerankerType
    NodeLabels     []string      // filter: only return these node types
    EdgeTypes      []string      // filter: only return these edge types
    Limit          int
    MinFactRating  *float64      // filter by quality rating
    MmrLambda      *float64      // MMR: 0.0=diversity, 1.0=relevance
    CenterNodeUUID *string       // for node_distance: center reference
}
```

### 3.5. Redis Cache Strategy

```go
// Cache với TTL 30s (temporal data cần fresh nhanh)
// Key: sha256(query + scope + reranker + filters)
// Invalidation triggers:
//   - NATS: "graph.extraction.completed" → invalidate group's cache
//   - NATS: "graph.fact.created" → update cache
//   - NATS: "graph.fact.invalidated" → remove from cache
```

### 3.6. API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/api/v2/graph/search` | Search knowledge graph |
| `POST` | `/api/v2/sessions/search` | Search across sessions |

**Graph Search Request:**
```json
{
  "query": "Where did Alice work?",
  "user_id": "alice",
  "scope": "edges",
  "reranker": "node_distance",
  "node_labels": ["Organization", "Event"],
  "edge_types": ["WORKS_AT", "WORKED_AT"],
  "limit": 10,
  "min_fact_rating": 0.7,
  "center_node_uuid": "node_alice_uuid"
}
```

**Graph Search Response:**
```json
{
  "items": [
    {
      "uuid": "edge_001",
      "score": 0.95,
      "fact": {
        "name": "WORKED_AT",
        "fact": "Alice worked at Acme Corp",
        "valid_at": "2020-01-01T00:00:00Z",
        "invalid_at": "2023-06-30T00:00:00Z"
      }
    }
  ],
  "total": 1,
  "query": "Where did Alice work?",
  "scope": "edges",
  "reranker": "node_distance",
  "latency_ms": 45
}
```

### 3.7. Reranker Configuration

```yaml
search:
  reranker:
    default: "rrf"
    rrf:
      k: 60                    # fusion constant
    mmr:
      default_lambda: 0.5      # 0.5 = balanced diversity vs relevance
    cross_encoder:
      model: "cross-encoder/ms-marco-MiniLM-L-12-v2"
      batch_size: 32
    node_distance:
      max_depth: 3              # graph traversal depth
    episode_mentions:
      time_decay: 0.95          # exponential decay
```

---

## 4. Acceptance Criteria

- [ ] Search với `scope=edges` → chỉ trả về facts (temporal edges), không có nodes.
- [ ] Search với `reranker=mmr` và 20 kết quả → kết quả diverse hơn so với `rrf`.
- [ ] Search với `node_labels=["Organization"]` → chỉ trả về Organization nodes.
- [ ] Search với `min_fact_rating=0.8` → chỉ trả về high-quality facts.
- [ ] Cache hit: cùng query → response từ Redis, latency < 10ms.
- [ ] Sau graph extraction → cache tự động invalidate, query lần sau trả về facts mới.
- [ ] Session search: query trả về matched facts grouped theo session_id.
