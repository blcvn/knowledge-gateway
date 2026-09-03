# Traceability Matrix — Graphiti Solutions

**Project:** VNP Memory  
**Domain:** Graphiti — Temporal Context Graph Engine  
**Architecture ref:** `specs/architecture.md` v3.0  
**Date:** 2026-06-17

---

## Architecture Context

VNP Memory hiện tại (35 services, monolith) đã có 4 graphiti services trong `InProcessRegistry`:

| Service hiện tại | gRPC port (external/GW-only) | In-process |
|-----------------|------------------------------|-----------|
| `graphiti-ingestion` | 9001 | bufconn |
| `graphiti-search` | 9002 | bufconn |
| `graphiti-knowledge` | 9003 | bufconn |
| `graphiti-store` | 9004 | bufconn |

> **Không có** `graphiti-admin` (service #5 cần thêm → monolith 35 → 36 services)

---

## Solution Map

| CR | Solution File | Type | Wave |
|----|--------------|------|------|
| CR-GR-001 | [SOL-001](./SOL-001-Episode-Ingestion-Pipeline.md) | REBUILD `services/graphiti-ingestion/` | Wave 2 |
| CR-GR-002 | [SOL-002](./SOL-002-Temporal-Knowledge-Graph-Store.md) | REBUILD `services/graphiti-store/` | Wave 1 |
| CR-GR-003 | [SOL-003](./SOL-003-Knowledge-Processing-Service.md) | REBUILD `services/graphiti-knowledge/` | Wave 1 |
| CR-GR-004 | [SOL-004](./SOL-004-Hybrid-Search-Engine.md) | REBUILD `services/graphiti-search/` | Wave 2 |
| CR-GR-005 | [SOL-005](./SOL-005-Custom-Ontology.md) | EXTEND knowledge + ingestion | Wave 3 |
| CR-GR-006 | [SOL-006](./SOL-006-Gateway-MCP-Server.md) | EXTEND `gateway/` (22 → 31 MCP tools) | Wave 3 |
| CR-GR-007 | [SOL-007](./SOL-007-Admin-Observability.md) | NEW `services/graphiti-admin/` (#36) + OTel | Wave 4 |

---

## New Service Added to Monolith

| Service | Action | Monolith Count |
|---------|--------|---------------|
| `graphiti-admin` | NEW — add to InProcessRegistry | 35 → 36 |

---

## Shared `pkg/graph/` Package

Tất cả graphiti services dùng chung shared types từ `pkg/graph/`:

```
pkg/graph/
├── node.go          # EntityNode, EpisodicNode, CommunityNode, SagaNode
├── edge.go          # EntityEdge, EpisodicEdge, CommunityEdge, HasEpisodeEdge, NextEpisodeEdge
├── ontology.go      # EntityTypeSchema, EdgeTypeSchema, OntologyRegistry
└── presets/
    ├── hr.go        # HRPreset
    ├── crm.go       # CRMPreset
    └── software.go  # SoftwareProjectPreset
```

---

## New NATS Subjects (Graphiti)

| Subject | Publisher | Consumer |
|---------|-----------|---------|
| `graphiti.episode.ingested` | graphiti-ingestion | graphiti-search (invalidate cache), obs-service |
| `graphiti.episode.bulk_ingested` | graphiti-ingestion | graphiti-admin (trigger community) |
| `graphiti.episode.removed` | graphiti-ingestion | graphiti-search (invalidate cache) |
| `graphiti.saga.updated` | graphiti-ingestion | obs-service |
| `graphiti.entity.resolved` | graphiti-knowledge | graphiti-search (invalidate cache) |
| `graphiti.community.rebuilt` | graphiti-admin | graphiti-search (invalidate cache) |
| `graphiti.tenant.created` | graphiti-admin | graphiti-store (init schema) |
| `graphiti.health.degraded` | any service | graphiti-admin (alerting) |

---

## Gateway Routes Added (26 new routes → /v1/graphiti/*)

| Method | Path | Target Service |
|--------|------|---------------|
| POST | `/v1/graphiti/episodes` | graphiti-ingestion |
| POST | `/v1/graphiti/episodes/bulk` | graphiti-ingestion |
| DELETE | `/v1/graphiti/episodes/{uuid}` | graphiti-ingestion |
| GET | `/v1/graphiti/episodes` | graphiti-ingestion |
| GET | `/v1/graphiti/episodes/{uuid}` | graphiti-ingestion |
| POST | `/v1/graphiti/triplets` | graphiti-ingestion |
| POST | `/v1/graphiti/sagas` | graphiti-ingestion |
| POST | `/v1/graphiti/sagas/{id}/summarize` | graphiti-ingestion |
| GET | `/v1/graphiti/sagas/{id}` | graphiti-ingestion |
| POST | `/v1/graphiti/search` | graphiti-search |
| POST | `/v1/graphiti/search/advanced` | graphiti-search |
| GET | `/v1/graphiti/entities/{uuid}` | graphiti-store |
| GET | `/v1/graphiti/edges/{uuid}` | graphiti-store |
| POST | `/v1/graphiti/ontology/{group_id}` | graphiti-knowledge |
| GET | `/v1/graphiti/ontology/{group_id}` | graphiti-knowledge |
| DELETE | `/v1/graphiti/ontology/{group_id}` | graphiti-knowledge |
| POST | `/v1/graphiti/ontology/{group_id}/preset` | graphiti-knowledge |
| POST | `/v1/graphiti/admin/communities` | graphiti-admin |
| POST | `/v1/graphiti/admin/indices` | graphiti-admin |
| DELETE | `/v1/graphiti/admin/data/{group_id}` | graphiti-admin |
| GET | `/v1/graphiti/admin/token-usage` | graphiti-admin |
| POST | `/v1/graphiti/admin/tenants` | graphiti-admin |
| GET | `/v1/graphiti/admin/tenants` | graphiti-admin |
| DELETE | `/v1/graphiti/admin/tenants/{id}` | graphiti-admin |
| GET | `/v1/graphiti/admin/tenants/{id}/stats` | graphiti-admin |
| GET | `/healthz` | aggregate (all services) |

---

## MCP Tools: Before → After (22 → 31)

| Added Tools | Target |
|------------|--------|
| `add_memory` | graphiti-ingestion |
| `search_memory` | graphiti-search |
| `get_episodes` | graphiti-ingestion |
| `delete_episode` | graphiti-ingestion |
| `delete_entity_node` | graphiti-store |
| `delete_entity_edge` | graphiti-store |
| `get_entity_edge` | graphiti-store |
| `clear_graph` | graphiti-admin |
| `get_status` | graphiti-admin |

---

## Neo4j Schema Changes

| Object | Change |
|--------|--------|
| Node labels | `:Entity`, `:Episodic`, `:Community`, `:Saga` |
| Entity edges | RELATES_TO (with valid_at/invalid_at/expired_at) |
| Episodic edges | MENTIONS |
| Community edges | HAS_MEMBER |
| Saga edges | HAS_EPISODE, NEXT_EPISODE |
| Vector indices | `entity_name_embedding` (1536d), `entity_edge_fact_embedding` |
| Fulltext indices | `entity_fulltext`, `episode_fulltext`, `community_fulltext` |
