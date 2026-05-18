---
id: FEAT-007
title: Unified Memory Explorer API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.2 Memory Explorer"
---

## Mục Tiêu

API thống nhất để search và inspect memory across tất cả 6 engines. Proxy yêu cầu tới `vnp-search-hub` để fan-out search và merge kết quả.

## Scope

### In Scope
- `POST /v1/console/memory/search` — Unified cross-engine search
- `GET /v1/console/memory/{id}` — Memory detail with provenance
- `GET /v1/console/memory/{id}/neighbors` — Graph neighbors of a memory
- `GET /v1/console/memory/{id}/versions` — Version chain (Supermemory)

### Out of Scope
- Graph Studio visualization API (FEAT-008 scope)
- Memory mutation/delete (covered by existing engine APIs)

## Thiết Kế Kỹ Thuật

### API Contract

#### POST `/v1/console/memory/search`

**Request:**
```json
{
  "query": "What is knowledge graph?",
  "mode": "semantic|hybrid|graph|timeline|profile",
  "engines": ["cognee", "graphiti", "zep", "openviking", "memobase", "supermemory"],
  "filters": {
    "tenant_id": "optional",
    "user_id": "optional",
    "memory_type": "episodic|semantic|conversational|procedural|profile|adaptive",
    "time_range": { "from": "2026-01-01", "to": "2026-05-13" },
    "confidence_min": 0.5,
    "ontology_class": "optional",
    "version_latest_only": true
  },
  "limit": 20,
  "offset": 0,
  "reranking": "rrf|mmr|cross_encoder"
}
```

**Response (200):**
```json
{
  "results": [
    {
      "id": "mem_abc123",
      "engine": "cognee",
      "memory_type": "semantic",
      "title": "Knowledge Graph Definition",
      "summary": "...",
      "content": "...",
      "score": 0.95,
      "entities": ["KnowledgeGraph", "Memory"],
      "source_sessions": ["sess_001"],
      "temporal_validity": { "from": "2026-01-01", "to": null },
      "policy_tags": ["public"],
      "version_chain": null,
      "metadata": {}
    }
  ],
  "total": 142,
  "facets": {
    "by_engine": { "cognee": 42, "graphiti": 38, "zep": 30, "openviking": 15, "memobase": 12, "supermemory": 5 },
    "by_type": { "semantic": 42, "episodic": 38, "conversational": 30, "procedural": 15, "profile": 12, "adaptive": 5 }
  },
  "latency_ms": 280
}
```

### Internal Architecture
- **Handler:** `adapter/http/explorer_handler.go`
- **Proxy to:** `vnp-search-hub` service via gRPC
- Gateway does NOT perform search — delegates entirely to `vnp-search-hub`

## Acceptance Criteria
- [ ] AC-1: Search returns results from multiple engines in single response
- [ ] AC-2: Filters by engine, memory_type, time_range work correctly
- [ ] AC-3: Reranking modes (rrf, mmr) produce different orderings
- [ ] AC-4: Memory detail returns provenance (source engine, sessions, confidence)
- [ ] AC-5: Version chain endpoint returns parent→root chain for Supermemory memories
- [ ] AC-6: All endpoints require auth; results scoped to tenant

## Test Requirements
- Unit tests: Filter building, response merging
- Integration tests: Multi-engine search with mock backends
- Minimum coverage: 80%
