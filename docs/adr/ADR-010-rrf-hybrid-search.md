# ADR-010 — Hybrid Search: BM25 + Vector + RRF Fusion

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-03 |
| **Deciders** | ML Team |
| **Feature** | F10 (Hybrid Search Engine) |

---

## Context

AI Agents cần retrieve relevant memory. Hai approaches:
- **Keyword (BM25):** Exact term matching, fast, no embedding required
- **Vector (semantic):** Meaning-based, finds relevant even without exact match

Cả hai đều có blind spots riêng. Cần strategy tốt hơn.

---

## Decision

**Reciprocal Rank Fusion (RRF) để kết hợp BM25 và Vector scores.**

```go
// RRF Formula: score(d) = Σ 1/(k + rank_i(d))
// k = 60 (Cormack et al. 2009 recommendation)
const rrfK = 60

func rrfFusion(bm25Results, vectorResults []SearchResult) []SearchResult {
    scores := make(map[string]float64)
    
    // BM25 rank contribution
    for i, r := range bm25Results {
        scores[r.ID] += 1.0 / float64(rrfK + i + 1)
    }
    
    // Vector rank contribution
    for i, r := range vectorResults {
        scores[r.ID] += 1.0 / float64(rrfK + i + 1)
    }
    
    // Sort by fused score
    return sortByScore(scores)
}

// Cross-engine: run RRF across 6 engines' results
// cognee (BM25+vec) + graphiti (graph+vec) + memobase (SQL) + ...
```

---

## Consequences

**Positive:**
- RRF outperforms either BM25 or Vector alone (Cormack et al. research proven)
- Rank-based fusion: không cần normalize scores giữa engines
- Simple, interpretable, không cần ML training
- Works kể cả khi 1 engine returns 0 results

**Negative:**
- Cần maintain BM25 in-memory index (RAM usage)
- Cross-engine fan-out latency (500ms timeout)
- RRF không optimize cho specific query types

---

## Alternatives Considered

### A1 — Score normalization + weighted sum
- **Rejected:** Normalization across different score spaces unreliable; weight tuning cần offline evaluation

### A2 — LLM reranker
- **Rejected:** Additional LLM cost per recall; too slow for < 500ms SLA

### A3 — Vector only
- **Rejected:** Keyword queries ("tìm email của Nguyen Van A") miss với pure vector
