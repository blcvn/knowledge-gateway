# Change Request: CR-UI-001 — Graph Studio (Knowledge Graph Visualization)

**CR ID:** CR-UI-001
**Component:** `backend/gateway`, `backend/services/graphiti-search`, `ui/`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / UI
**Feature:** [F17](../../../features/17-graph-studio/README.md)
**Solution:** [S3 — Temporal Reasoning](../../../bussiness/solutions/S3-temporal-reasoning.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P3-02 | ML Engineer | Không thể visualize knowledge graph — debug mù |
| PP-P3-01 | ML Engineer | Không biết graph quality khi tuning extraction |

**Before:** Engineer phải chạy raw Cypher queries để inspect graph.
**After:** Interactive graph visualization với entity detail, timeline, ontology editor.

---

## 2. Features

### 2.1 Subgraph Visualization
- Seed: entity ID hoặc query → render interactive force-directed graph
- Nodes: colored by entity type (Person=blue, Org=green, Concept=purple, Event=orange)
- Edges: labeled with relationship type + temporal validity badges
- Filters: entity type, time range, relationship type, max depth (1-5)

### 2.2 Entity Detail Panel
- Click node → show: all properties, connected edges, source episodes, provenance document
- Temporal validity per edge: valid_from, valid_to timestamps

### 2.3 Timeline View
- Horizontal timeline: facts added/expired over time
- Animation: play evolution of graph changes (scrub to any point in time)

### 2.4 Ontology Management
- View current: entity types, relationship types, property constraints
- Edit: add/modify entity and relationship types
- Apply: affects future LLM extractions (immediate)

### 2.5 Graph Query Console
- Input: Cypher query (MATCH/RETURN only — write operations blocked)
- Result: table view + inline graph visualization
- Query history: last 20 queries per session

---

## 3. Backend API Endpoints (NEW)

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/console/graph/subgraph` | Subgraph from seed entity |
| `GET`  | `/v1/console/graph/entity/{id}` | Entity detail + edges |
| `POST` | `/v1/console/graph/timeline` | Graph changes over time range |
| `GET`  | `/v1/console/graph/ontology` | Current ontology |
| `PUT`  | `/v1/console/graph/ontology` | Update ontology |
| `POST` | `/v1/console/graph/query` | Execute Cypher query |

---

## 4. Security

- Cypher: whitelist MATCH, RETURN, WITH, WHERE, ORDER BY, LIMIT only
- Result cap: max 200 nodes, 500 edges per response
- Tenant isolation: all queries inject `AND n.tenant_id = $tenant_id`

---

## 5. Acceptance Criteria

- [ ] Subgraph renders within 2s for depth=3
- [ ] Entity detail panel shows all properties + temporal edges
- [ ] Timeline animation plays at 1x/2x/5x speed
- [ ] Ontology update applied to future extractions (not retroactive)
- [ ] Cypher write operations (CREATE/DELETE/MERGE) return 403
- [ ] Result capped at 200 nodes — shows "truncated" warning
- [ ] Tenant isolation: queries cannot return other tenant's data
