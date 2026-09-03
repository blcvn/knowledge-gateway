# Feature 16 — Memory Explorer

> **Loại:** Console UI | **Priority:** P1 | **Status:** Implemented (UI)

## Mô tả

Memory Explorer là công cụ tìm kiếm và inspect memory trong Console UI. Developer có thể search across tất cả engines, xem chi tiết từng memory, xem neighbors trong graph, và track version history.

---

## Business Logic

### Cross-Engine Search

Memory Explorer cho phép search với multiple filters:
- **Query text**: Semantic search
- **Memory type**: Filter theo engine (semantic/episodic/conversational/profile/procedural/adaptive)
- **Time range**: Filter theo thời gian
- **Tenant/User**: Filter theo scope

### Memory Detail View

Khi chọn một memory:
- Full content
- Metadata: created_at, engine, type, score
- Source: Từ episode nào, document nào
- Provenance chain

### Neighbors View

Với graph-based memories (Graphiti, Zep, Supermemory):
- Hiển thị connected nodes/edges
- Depth configurable (1-3 hops)
- Visual graph representation

### Version History

Với Supermemory và AgentMemory:
- Hiển thị full version chain (root → latest)
- Mỗi version: content diff, timestamp, relation type (updates/extends/derives)

---

## Dataflow

```
Console UI (Memory Explorer)
        │
        ├── POST /v1/console/memory/search
        │         ├── Input: {query, type, time_range, engine, limit}
        │         └── Output: [{id, content_preview, engine, score, created_at}]
        │
        ├── GET /v1/console/memory/{id}
        │         └── Full memory detail + metadata
        │
        ├── GET /v1/console/memory/{id}/neighbors
        │         └── Graph neighbors (configurable depth)
        │
        └── GET /v1/console/memory/{id}/versions
                  └── Version chain (parent → root)
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/console/memory/search` | Search memories |
| `GET` | `/v1/console/memory/{id}` | Memory detail |
| `GET` | `/v1/console/memory/{id}/neighbors` | Graph neighbors |
| `GET` | `/v1/console/memory/{id}/versions` | Version history |

---

## Business Value

### Pain Points được giải quyết

- **PP-P4-01 (Không biết AI nhớ gì)**
- **PP-P7-04 (No transparency)**

### Actors hưởng lợi

P4 Enterprise Architect, P7 AI Power User

### Giải pháp tham chiếu

- [S9 — Enterprise Governance](../../bussiness/solutions/S9-governance-compliance.md)

### ROI / Kết quả đo được

> Full visibility: inspect, neighbors, versions của từng memory

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
