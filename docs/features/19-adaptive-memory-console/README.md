# Feature 19 — Adaptive Memory Console

> **Loại:** Console UI | **Priority:** P1 | **Status:** Implemented (UI)

## Mô tả

Adaptive Memory Console cho phép quản lý và visualize Supermemory adaptive KG. Bao gồm: memory version explorer, connector management, analytics, và forget-rules configuration.

---

## Business Logic

### Memory Version Explorer

Visualize memory versioning chain:
- List all memories với `isLatest` indicator
- Click memory → xem full version chain (root → latest)
- Mỗi version: content, timestamp, relation type, parent link
- Compare versions: content diff

### Connector Management

Manage external data connectors:
- List connected sources (Google Drive, Notion, GitHub...)
- Connector status: active/inactive/error
- Last sync timestamp + items synced
- Trigger manual sync
- Create new connector

### Analytics

Usage analytics cho Supermemory:
- Memory count over time
- Memory type distribution (static vs dynamic)
- Forget events timeline
- Most accessed memories

### Forget Rules Configuration

Configure auto-forgetting behavior:
- Default `forgetAfter` per memory type
- Static memories: no forget (persist indefinitely)
- Dynamic memories: configurable TTL (days/weeks)
- Manual forget: delete specific memory

---

## Dataflow

```
Console UI (Adaptive Memory)
        │
        ├── GET /v1/console/adaptive/memories
        │         └── List memories (isLatest=true by default)
        │
        ├── GET /v1/console/adaptive/memories/{id}/versions
        │         └── Full version chain
        │
        ├── GET /v1/console/adaptive/connectors
        │         └── List connectors với status
        │
        ├── POST /v1/console/adaptive/connectors
        │         └── Add new connector
        │
        ├── POST /v1/console/adaptive/connectors/{id}/sync
        │         └── Trigger sync
        │
        ├── GET /v1/console/adaptive/analytics
        │         └── Usage analytics timeseries
        │
        ├── GET /v1/console/adaptive/forget-rules
        │         └── Current forget configuration
        │
        └── PUT /v1/console/adaptive/forget-rules
                  └── Update configuration
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/adaptive/memories` | List memories |
| `GET` | `/v1/console/adaptive/memories/{id}/versions` | Version chain |
| `GET` | `/v1/console/adaptive/connectors` | List connectors |
| `POST` | `/v1/console/adaptive/connectors` | Add connector |
| `POST` | `/v1/console/adaptive/connectors/{id}/sync` | Trigger sync |
| `GET` | `/v1/console/adaptive/analytics` | Analytics |
| `GET` | `/v1/console/adaptive/forget-rules` | Forget config |
| `PUT` | `/v1/console/adaptive/forget-rules` | Update forget config |
