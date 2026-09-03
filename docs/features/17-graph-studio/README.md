# Feature 17 — Graph Studio

> **Loại:** Console UI | **Priority:** P1 | **Status:** Implemented (UI)

## Mô tả

Graph Studio là công cụ visualization và query cho knowledge graph. ML Engineer và Developer có thể explore graph structure, xem entities và relationships, edit ontology, và chạy Cypher/graph queries trực tiếp.

---

## Business Logic

### Subgraph Visualization

Bắt đầu từ một entity hoặc query, Graph Studio render interactive graph:
- **Nodes**: Entities với labels và properties
- **Edges**: Relationships với temporal validity (valid_at/invalid_at)
- **Colors**: Phân biệt node types (Person, Org, Concept, Event...)
- **Filters**: Filter theo entity type, time range, relationship type

### Entity Detail

Khi click vào entity:
- Full properties
- Connected edges (với temporal validity)
- Source episodes
- Provenance (extracted từ document nào)

### Timeline View

Hiển thị graph changes theo thời gian:
- Facts added/expired theo timeline
- Animation: xem graph evolve

### Ontology Management

ML Engineer có thể:
- Xem current ontology (entity types, relationship types, constraints)
- Edit ontology (thêm/sửa entity types, relationship types)
- Apply ontology thay đổi → ảnh hưởng future extractions

### Graph Query

Chạy custom queries:
- Cypher query cho Neo4j-backed engines (Graphiti, Zep)
- Custom query DSL cho other engines
- Result visualization inline

---

## Dataflow

```
Console UI (Graph Studio)
        │
        ├── POST /v1/console/graph/subgraph
        │         ├── Input: {seed_entity, depth, filters}
        │         └── Output: {nodes: [...], edges: [...]}
        │
        ├── GET /v1/console/graph/entity/{id}
        │         └── Full entity + properties + edges
        │
        ├── POST /v1/console/graph/timeline
        │         ├── Input: {entity_id, time_range}
        │         └── Output: Timeline of graph changes
        │
        ├── GET /v1/console/graph/ontology
        │         └── Current ontology definition
        │
        ├── PUT /v1/console/graph/ontology
        │         └── Update ontology
        │
        └── POST /v1/console/graph/query
                  ├── Input: {query: "MATCH (n)...", engine: "graphiti|zep"}
                  └── Output: Query results
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/console/graph/subgraph` | Render subgraph |
| `GET` | `/v1/console/graph/entity/{id}` | Entity detail |
| `POST` | `/v1/console/graph/timeline` | Graph timeline |
| `GET` | `/v1/console/graph/ontology` | Get ontology |
| `PUT` | `/v1/console/graph/ontology` | Update ontology |
| `POST` | `/v1/console/graph/query` | Run graph query |
