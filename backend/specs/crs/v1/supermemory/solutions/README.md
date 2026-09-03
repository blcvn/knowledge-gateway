# Solutions — Supermemory Change Requests

**Project:** VNP Memory  
**Domain:** Supermemory Feature Parity  
**Path:** `specs/crs/v1/supermemory/solutions/`  
**Date:** 2026-06-17  
**Status:** Draft

> Các Solution documents này mô tả giải pháp kỹ thuật chi tiết để đáp ứng từng Change Request trong `specs/crs/v1/supermemory/`.  
> Mỗi solution bao gồm: phân tích kiến trúc hiện tại (gap analysis), thiết kế domain model, code Go mẫu, database schema, API endpoints, lộ trình triển khai, và mapping với Acceptance Criteria.

---

## Danh sách Solutions

| Solution | CR | Tên | Loại | Wave | Ước tính |
|----------|-----|-----|------|------|---------|
| [SOL-SM-001](./SOL-SM-001-Document-Ingestion-Pipeline.md) | CR-SM-001 | **Document Ingestion Pipeline** | New Service | 2 | 16 ngày |
| [SOL-SM-002](./SOL-SM-002-Memory-Engine-Knowledge-Graph.md) | CR-SM-002 | **Memory Engine & Knowledge Graph** | Upgrade Service | 2 | 13 ngày |
| [SOL-SM-003](./SOL-SM-003-Hybrid-Search-Engine.md) | CR-SM-003 | **Hybrid Search Engine** | Upgrade Service | 3 | 13 ngày |
| [SOL-SM-004](./SOL-SM-004-User-Profile-Service.md) | CR-SM-004 | **User Profile Service** | New Service | 3 | 8 ngày |
| [SOL-SM-005](./SOL-SM-005-Connector-Service.md) | CR-SM-005 | **External Connector Service** | New Service | 4 | 17 ngày |
| [SOL-SM-006](./SOL-SM-006-MCP-Server.md) | CR-SM-006 | **MCP Server** | Upgrade Service | 4 | 12 ngày |
| [SOL-SM-007](./SOL-SM-007-Auth-Organization-RBAC.md) | CR-SM-007 | **Auth & Organization RBAC** | Upgrade Service | 1 | 14 ngày |
| [SOL-SM-008](./SOL-SM-008-Project-Space-Management.md) | CR-SM-008 | **Project & Space Management** | New Service | 1 | 7.5 ngày |
| [SOL-SM-009](./SOL-SM-009-Analytics-Token-Economics.md) | CR-SM-009 | **Analytics & Token Economics** | New Service | 5 | 9 ngày |
| [SOL-SM-010](./SOL-SM-010-Framework-Integrations-SDK.md) | CR-SM-010 | **Framework Integrations & SDK** | New Packages | 5 | 13 ngày |

**Tổng ước tính:** ~122.5 ngày developer-days (5 Waves, có thể parallel)

---

## Kiến trúc Tổng thể Giải pháp

```
                         External Clients
         ┌──────────────────────────────────────────────┐
         │  AI SDKs (Go/TS/Py) · MCP (Claude/Cursor)   │
         │  Vercel AI · LangChain · LangGraph · Mastra  │
         └──────────────────────┬───────────────────────┘
                                │
                ┌───────────────▼───────────────┐
                │     VNP Gateway :8080          │
                │  JWT/API Key (sm_) · RBAC     │  ← SOL-007
                │  OAuth2 Server · Rate Limit    │
                └──┬───┬───┬───┬───┬───┬───┬───┘
                   │   │   │   │   │   │   │  gRPC (bufconn)
    ┌──────────────┘   │   │   │   │   │   └──────────────┐
    │         ┌────────┘   │   │   │   └────────────┐     │
    ▼         ▼            ▼   ▼   ▼                ▼     ▼
┌──────┐  ┌──────┐  ┌──────┐ ┌────┐ ┌────────┐ ┌──────┐ ┌─────┐
│ Doc  │  │ Mem  │  │Search│ │Prof│ │Connector│ │Auth  │ │ MCP │
│ :9001│  │ :9002│  │ :9003│ │:9004│ │  :9005 │ │ :9007│ │:9006│
└──────┘  └──────┘  └──────┘ └────┘ └────────┘ └──────┘ └─────┘
    ▲         ▲            ▲                              
    │         │            │   ┌──────────┐  ┌──────────┐
    └─────────┴────────────┘   │ Project  │  │Analytics │
                               │  :9009   │  │  :9008   │
                               └──────────┘  └──────────┘
                   │ NATS JetStream Events │
    ┌──────────────────────────────────────────────────────┐
    │  PostgreSQL + pgvector │ Redis │ NATS JetStream │ S3  │
    └──────────────────────────────────────────────────────┘
```

---

## NATS Event Flow (Cross-Service)

```
CreateDocument
    → document.ingest.requested
        → IngestionWorker (SOL-001)
            → document.processed
                → FactExtraction (SOL-002)
                    → memory.created
                        → Profile rebuild (SOL-004)
                        → Analytics update (SOL-009)
                    → memory.relation.created

ForgetRequest
    → memory.forgotten
        → Profile rebuild (SOL-004)
        → Analytics update (SOL-009)

ConnectorSync
    → sm.connector.synced
        → Analytics log (SOL-009)

SpaceCreated
    → space.created
        → (MCP session cache refresh - SOL-006)
```

---

## Lộ trình Theo Wave

### Wave 1 — Foundation (SOL-007 + SOL-008)
- **Mục tiêu:** Auth/RBAC + Project/Space namespace
- **Ước tính:** 14 + 7.5 = ~21.5 ngày
- **Lý do ưu tiên:** Auth là foundation cho tất cả services, Space management cần có trước khi ingest documents

### Wave 2 — Core Memory (SOL-001 + SOL-002)
- **Mục tiêu:** Document ingestion pipeline + Knowledge Graph với fact extraction
- **Ước tính:** 16 + 13 = ~29 ngày (có thể parallel một phần)
- **Dependencies:** Cần Wave 1 (OrgID, SpaceID, containerTags)

### Wave 3 — Intelligence (SOL-003 + SOL-004)
- **Mục tiêu:** Hybrid Search (RAG+Memory) + User Profile < 100ms
- **Ước tính:** 13 + 8 = ~21 ngày (có thể parallel)
- **Dependencies:** Cần Wave 2 (memory_entries + chunks tồn tại)

### Wave 4 — Integrations (SOL-005 + SOL-006)
- **Mục tiêu:** External Connectors (Google Drive, Notion) + MCP Server hoàn chỉnh
- **Ước tính:** 17 + 12 = ~29 ngày (có thể parallel)
- **Dependencies:** Cần Wave 1 (Auth) + Wave 2 (Document Service)

### Wave 5 — Ecosystem (SOL-009 + SOL-010)
- **Mục tiêu:** Analytics + Token Economics + SDK/Framework integrations
- **Ước tính:** 9 + 13 = ~22 ngày
- **Dependencies:** Tất cả waves trước (analytics cần events từ tất cả services)

---

## Key Design Decisions

| Quyết định | Lý do |
|-----------|-------|
| Clean Architecture 4 lớp cho tất cả services | Nhất quán với codebase hiện tại của VNP Memory |
| InProcessRegistry (bufconn) trong Monolith mode | Không có network hop → latency thấp hơn |
| pgvector HNSW cho vector search | Đã có PostgreSQL infra, tránh thêm Qdrant dependency |
| NATS JetStream cho async events | Đã có embedded NATS trong monolith |
| Redis cache cho profiles + sessions | Đã có Redis infra, p95 < 100ms |
| AES-GCM cho OAuth token encryption | Industry standard, key rotation friendly |
| SHA-256 contentHash để dedup | Deterministic, không cần lookup thêm |
| `sm_` prefix cho API keys | Format chuẩn, dễ phân biệt với internal tokens |
