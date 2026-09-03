# Change Request: CR-CONSOLE-002 — Memory Explorer Backend APIs

**CR ID:** CR-CONSOLE-002
**Component:** `backend/gateway`, `backend/services/vnp-search-hub`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Console
**Feature:** [F16](../../../features/16-memory-explorer/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P4-01 | Enterprise Architect | Không biết AI nhớ gì về user |
| PP-P7-04 | AI Power User | No transparency into memory state |

---

## 2. APIs

### `POST /v1/console/memory/search`

```json
// Request
{
  "query": "meeting with John",
  "types": ["episodic", "semantic"],   // filter by engine
  "user_id": "user-123",
  "time_range": {"from": "2026-09-01", "to": "2026-09-03"},
  "limit": 20,
  "offset": 0
}

// Response
{
  "results": [
    {
      "id": "mem-abc",
      "engine": "graphiti",
      "type": "episodic",
      "content": "Meeting with John about project Alpha",
      "score": 0.92,
      "created_at": "2026-09-02T...",
      "user_id": "user-123"
    }
  ],
  "total_hits": 47,
  "engines_queried": ["graphiti", "cognee"]
}
```

### `GET /v1/console/memory/{id}`

```json
{
  "id": "mem-abc",
  "engine": "graphiti",
  "type": "episodic",
  "content": "...",
  "metadata": {...},
  "created_at": "...",
  "version": 2,
  "salience_score": 0.78
}
```

### `GET /v1/console/memory/{id}/neighbors`

```json
{
  "entity": {"id": "mem-abc", "content": "..."},
  "neighbors": [
    {"id": "ent-xyz", "type": "Person", "name": "John", "relationship": "MENTIONED_IN"},
    {"id": "ent-abc", "type": "Project", "name": "Alpha", "relationship": "RELATED_TO"}
  ]
}
```

### `GET /v1/console/memory/{id}/versions`

```json
{
  "current_version": 2,
  "history": [
    {"version": 1, "content": "...", "created_at": "...", "superseded_by": 2},
    {"version": 2, "content": "...", "created_at": "...", "superseded_by": null}
  ]
}
```

---

## 3. Implementation Notes

- `search` → calls `vnp-search-hub` (cross-engine RRF search)
- `neighbors` → calls `graphiti-search.GetSubgraph(seed=id, depth=1)`
- `versions` → calls `sm-memory.GetVersionHistory(id)`
- Tenant isolation: always inject tenant_id

---

## 4. Acceptance Criteria

- [ ] Search supports cross-engine query
- [ ] Filter by type (engine) works
- [ ] Time range filter applied correctly
- [ ] Neighbors show entity relationships
- [ ] Version history shows all versions with timestamps
- [ ] p95 < 500ms
