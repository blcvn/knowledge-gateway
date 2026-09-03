# P3 — ML / AI Engineer

> **Vai trò:** Tối ưu context quality, thiết kế ontology, đánh giá retrieval accuracy.
> **Kỹ năng:** Python, graph theory, NLP, evaluation frameworks.
> **Tần suất sử dụng VNP Memory:** Hàng tuần.

---

## Pain Points

### PP-P3-01 — Generic ontology miss domain-specific entities

**Mô tả:**
Default ontology của memory engines nhận diện entity type "Person", "Organization", "Location" — nhưng trong domain y tế cần "Drug", "Symptom", "Diagnosis". Generic ontology dẫn đến knowledge graph kém chất lượng.

**Features giải quyết:**
- [F04] Zep Custom Ontology: `POST /v1/zep/graph/ontology` — define custom entity types
- [F04] Custom Facts: `POST /v1/zep/graph/facts` — seed domain-specific facts
- [F03] Cognee Multi-strategy Search: 15+ strategies, configurable pipeline

---

### PP-P3-02 — Không đo được retrieval quality cross-engine

**Mô tả:**
Cognee search vs Graphiti search vs Supermemory search — kết quả khác nhau với cùng query. Không có unified benchmark để biết engine nào tốt hơn cho use case cụ thể.

**Features giải quyết:**
- [F10] Hybrid Search Engine: BM25 + Vector + RRF — có thể so sánh scores
- [F20] Agent Context Debugger: `POST /v1/console/debugger/trace` — xem pipeline từng bước
- [F25] Observability: latency breakdown per engine

---

### PP-P3-03 — Không debug được tại sao agent recall sai context

**Mô tả:**
Agent trả lời không chính xác nhưng không biết vấn đề ở retrieval (sai context được fetch) hay ở LLM (context đúng nhưng LLM hiểu sai). Cần transparency vào context assembly process.

**Features giải quyết:**
- [F08] Agent Observe: capture `memory_read` events — biết chính xác agent đọc gì
- [F20] Agent Context Debugger: trace toàn bộ context assembly
- [F26] Session Replay: replay session step-by-step, filter theo `memory_read` hook

---

### PP-P3-04 — Hybrid search tuning là black box

**Mô tả:**
BM25 weight vs vector similarity weight — không biết tune thế nào cho domain cụ thể. RRF fusion parameters mặc định không phù hợp với mọi use case.

**Features giải quyết:**
- [F10] Hybrid Search Engine: BM25 in-memory index + local embedding + RRF fusion — observable
- [F23] Pipeline Monitor: job status per search strategy

---

## Summary

| Pain | Giải pháp |
|---|---|
| Generic ontology | Custom ontology + facts (Zep) |
| Không so sánh được engines | Unified search + observability |
| Debug context assembly | Agent Debugger + Session Replay |
| Hybrid search tuning khó | Observable BM25+Vector+RRF pipeline |
