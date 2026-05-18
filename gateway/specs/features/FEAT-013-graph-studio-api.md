---
id: FEAT-013
title: Graph Studio API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.3 Graph Studio"
---

## Mục Tiêu

REST APIs cho Graph Studio — visual knowledge graph exploration. Bao gồm subgraph queries, timeline replay, entity explorer, ontology designer, và query playground.

## Bối Cảnh Nghiệp Vụ

Graph Studio (UX §6.3) là strategic visual layer:
1. **Interactive Graph Canvas** — zoom/pan, clustering, edge grouping, temporal playback
2. **Entity Inspector** — entity type, ontology schema, related facts, confidence
3. **Timeline Slider** — temporal replay of entity evolution
4. **Ontology Designer** — visual schema builder (node types, edge rules)
5. **Query Playground** — Cypher visual builder, natural language → Cypher

## Scope

### In Scope
- `POST /v1/console/graph/subgraph` — Query subgraph by entity/relationship
- `GET /v1/console/graph/entity/{id}` — Entity detail with neighbors
- `POST /v1/console/graph/timeline` — Temporal subgraph for time range
- `GET /v1/console/graph/ontology` — Get current ontology schema
- `PUT /v1/console/graph/ontology` — Update ontology schema
- `POST /v1/console/graph/query` — Execute Cypher/NL query

### Out of Scope
- Graph rendering (frontend responsibility)
- Real-time graph mutation subscriptions (future scope)

## Thiết Kế Kỹ Thuật

### API Contract

#### POST `/v1/console/graph/subgraph`

**Request:**
```json
{
  "center_entity_id": "entity_abc123",
  "depth": 2,
  "max_nodes": 100,
  "edge_types": ["related_to", "derived_from", "updates"],
  "engines": ["graphiti", "cognee", "supermemory"],
  "include_metadata": true
}
```

**Response (200):**
```json
{
  "nodes": [
    {
      "id": "entity_abc123",
      "label": "Knowledge Graph",
      "type": "concept",
      "engine": "graphiti",
      "confidence": 0.95,
      "properties": {},
      "memory_type": "episodic"
    }
  ],
  "edges": [
    {
      "source": "entity_abc123",
      "target": "entity_def456",
      "type": "related_to",
      "weight": 0.88,
      "engine": "graphiti",
      "temporal_validity": { "from": "2026-01-01", "to": null }
    }
  ],
  "total_nodes": 42,
  "total_edges": 78,
  "latency_ms": 250
}
```

#### POST `/v1/console/graph/timeline`

**Request:**
```json
{
  "entity_id": "entity_abc123",
  "from": "2026-01-01T00:00:00Z",
  "to": "2026-05-13T00:00:00Z",
  "granularity": "day|week|month"
}
```

**Response (200):**
```json
{
  "snapshots": [
    {
      "timestamp": "2026-01-15T00:00:00Z",
      "nodes_count": 5,
      "edges_count": 8,
      "changes": ["node_added: entity_xyz", "edge_updated: related_to"]
    }
  ]
}
```

#### POST `/v1/console/graph/query`

**Request:**
```json
{
  "mode": "cypher|natural_language",
  "query": "MATCH (n:Concept)-[r]->(m) WHERE n.name = 'Knowledge Graph' RETURN n, r, m LIMIT 50",
  "engine": "graphiti|cognee"
}
```

### Internal Architecture
- **Handler:** `adapter/http/graph_handler.go`
- **Proxy to:** `graphiti-store` (subgraph, entity, timeline), `cognee-search` (ontology)
- **Usecase:** `usecase/graph.go` — merge graph results from multiple engines
- Gateway merges subgraph from Graphiti + Cognee + Supermemory into unified graph format

## Acceptance Criteria
- [ ] AC-1: Subgraph query returns nodes and edges with engine badges
- [ ] AC-2: Entity detail includes type, ontology, confidence, source memories
- [ ] AC-3: Timeline returns temporal snapshots at configurable granularity
- [ ] AC-4: Ontology CRUD supports node types, edge rules, validation constraints
- [ ] AC-5: Cypher query execution returns graph results
- [ ] AC-6: Natural language query translated to Cypher via LLM
- [ ] AC-7: Multi-engine graph merge produces deduplicated nodes
- [ ] AC-8: All endpoints require auth; results scoped to tenant

## Test Requirements
- Unit tests: Graph merge logic, ontology validation, Cypher sanitization
- Integration tests: Multi-engine subgraph with mock backends
- Minimum coverage: 80%
