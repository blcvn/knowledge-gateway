# Feature 10 — Hybrid Search Engine

> **Loại:** Search | **Priority:** P0 | **Status:** Implemented (CR-AM-003)

## Mô tả

Hybrid Search Engine kết hợp 3 search strategies — **BM25 keyword search**, **vector semantic search**, và **graph traversal** — sau đó dùng **Reciprocal Rank Fusion (RRF)** để merge kết quả. Engine này đặc biệt quan trọng cho AgentMemory layer vì cần latency thấp (p50 ≤ 14ms) và không phụ thuộc vào Qdrant external service.

---

## Business Logic

### BM25 In-memory Index

Thay vì external search service, BM25 index được maintain **in-memory** với Go:

- **TF-IDF scoring** trên memory content tokens.
- **In-memory**: Zero network latency, cực nhanh.
- **Persistence**: Index được serialize (gob format) và persist sang disk → survive restarts.
- **Updates**: Incremental update khi memory được add/delete.

### Local Embedding

Thay vì gọi OpenAI API cho embeddings (latency + cost), Hybrid Search dùng local model:
- Model: **all-MiniLM-L6-v2** (384 dimensions)
- Zero cost, zero external dependency
- Latency: ~2ms per embedding (vs ~100ms external API)

### RRF Fusion

Reciprocal Rank Fusion merge ranked lists từ multiple sources:

```
RRF_score(doc) = Σ  1 / (k + rank_i(doc))
                  i

k = 60 (constant để prevent high scores from dominating)
```

Kết quả: documents xuất hiện cao trong nhiều lists → RRF score cao hơn, không bị dominated bởi 1 source duy nhất.

### Query Expansion

LLM-powered query expansion: khi nhận query ngắn, LLM generate synonyms và related terms → tăng recall:

```
Input: "database connection error"
Expanded: ["database connection error", "DB connection failure", 
           "connection timeout", "connection refused", "SQL connection"]
```

### Cross-Engine Search Hub

`vnp-search-hub` là component điều phối cross-engine recall:
- Fan-out gRPC calls tới tất cả 6 memory engines song song.
- Timeout: 500ms tổng.
- Engine nào fail → skip (không block toàn bộ response).
- Merge + rerank results từ tất cả engines.

---

## Dataflow

### AgentMemory Hybrid Search

```
POST /v1/observe/search  (hoặc qua Memory Recall)
        │
        ├── Input: {query: "...", filters: {type, agent_id}, limit: K}
        │
        ▼
observe-search service
        │
        ├── 1. Query Expansion (LLM, optional)
        │         └── Generate synonyms + related terms
        │
        ├── 2. BM25 Search (in-memory index)
        │         ├── Tokenize expanded query
        │         ├── TF-IDF score on memory content
        │         └── Return: [(memory_id, bm25_score, rank)]
        │
        ├── 3. Vector Search (local embedding)
        │         ├── Embed query with all-MiniLM-L6-v2
        │         ├── Cosine similarity against stored embeddings
        │         └── Return: [(memory_id, vector_score, rank)]
        │
        ├── 4. Graph Search (optional, if graph-connected)
        │         ├── Find memories connected to query entities
        │         └── Return: [(memory_id, graph_score, rank)]
        │
        └── 5. RRF Fusion
                  ├── RRF_score = Σ 1/(60 + rank_i)
                  └── Sort by RRF_score DESC → Top-K results
```

### Cross-Engine Recall (vnp-search-hub)

```
POST /v1/memory/recall
        │
        ▼
vnp-search-hub service
        │
        ├── Parallel gRPC fan-out (goroutines, WaitGroup)
        │         ├── cognee-search   → semantic results
        │         ├── graphiti-search → temporal results
        │         ├── zep-search      → session results
        │         ├── memobase-context → profile context
        │         ├── ov-search       → procedural results
        │         └── sm-search       → adaptive KG results
        │
        │   (all concurrent, max 500ms timeout)
        │
        ├── Collect results (partial results if some engines timeout)
        │
        └── Merge + Rerank
                  ├── Normalize scores per engine
                  ├── RRF fusion across all sources
                  └── Return Top-K unified results
```

### BM25 Index Persistence

```
On startup:
        └── Load BM25 index from disk (gob file)
                  └── Rebuild in-memory if file not found

On memory add/delete:
        ├── Update in-memory BM25 index (incremental)
        └── Async: serialize index to disk (gob)
                  (non-blocking, eventual consistency)
```

---

## Performance Targets

| Metric | Target |
|--------|--------|
| AgentMemory search latency (p50) | `≤ 14ms` |
| Cross-engine recall (p95) | `≤ 500ms` |
| BM25 index build | Background, non-blocking |
| Embedding latency (local) | `~2ms` per query |

---

## Services

| Service | Vai trò |
|---------|---------|
| `observe-search` | AgentMemory hybrid search (BM25 + vector + RRF) |
| `vnp-search-hub` | Cross-engine fan-out + merge |
| `search-service` | General search infrastructure |

---

## Related

- `shared/pkg/search/` — BM25 implementation, RRF fusion utilities
- Feature 01 (Unified Memory API) — uses vnp-search-hub for recall
- Feature 08 (Observe Service) — feeds data into search indexes

---

## Business Value

### Pain Points được giải quyết

- **PP-P1-02 (Memory fragmented)**
- **PP-P3-02 (Không benchmark engines)**

### Actors hưởng lợi

P1 AI Agent Developer, P3 ML/AI Engineer

### Giải pháp tham chiếu

- [S2 — Unified Memory API](../../bussiness/solutions/S2-unified-api.md)

### ROI / Kết quả đo được

> BM25 + Vector + RRF → tốt hơn any single strategy | Cross-engine recall < 500ms

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
