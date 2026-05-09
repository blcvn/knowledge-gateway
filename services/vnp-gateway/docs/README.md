---
id: DOC-S01
service: vnp-gateway
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Platform Team
---

# vnp-gateway

> **Group**: Platform | **Ports**: 8080(REST) 8081(gRPC) 8082(MCP) 8083(Health) | **Origin**: Unified

## Purpose

Single entry point for ALL VNP Memory APIs. Routes to **35 domain services** across 6 engines. Handles **JWT/API Key authentication**, **tenant resolution**, **rate limiting** (Redis sliding window), **protocol translation** (REST ↔ gRPC), **MCP server** (15 tools), **WebDAV proxy**, **WebSocket**, and **circuit breaking**.

### Business Capability

- **Unified Routing**: REST/gRPC/MCP/WebSocket/WebDAV → 35 services
- **Auto-Classification**: `memory.store()` auto-routes by content type (semantic/episodic/conversational/procedural)
- **MCP Server**: 15 registered tools for AI agent integration
- **WebDAV Proxy**: Filesystem interoperability via ov-fs
- **Auth**: JWT RS256 + API Key with tenant_id extraction
- **Rate Limiting**: Per-tenant, per-endpoint Redis sliding window
- **Circuit Breaking**: sony/gobreaker per downstream service
- **Cross-Engine Recall**: Single API → vnp-search-hub fan-out

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: chi/v5 (HTTP), gRPC-Web proxy
- **Cache**: Redis (rate limiting, session)
- **Architecture**: 4-layer Clean Architecture

## API Groups

| Prefix | Engine | Routes |
|--------|--------|--------|
| `/v1/memory/*` | Unified | store, recall, forget, timeline |
| `/v1/cognee/*` | Cognee | datasets, cognify, search |
| `/v1/graphiti/*` | Graphiti | episodes, search, nodes, edges |
| `/v1/memobase/*` | Memobase | blobs, flush, context, profiles |
| `/v1/ov/*` | OpenViking | files, tree, grep, search, sessions |
| `/v1/zep/*` | Zep | users, sessions, memory, graph |
| `/v1/sm/*` | Supermemory | documents, memories, search, connections |
| `/v3/*` | Supermemory (native) | documents, search, projects, analytics |
| `/v1/admin/*` | Admin | tenants, keys, health |
| `/webdav/*` | OpenViking | WebDAV file access |

## MCP Tools (15)

`memory_store`, `memory_recall`, `memory_search`, `memory_timeline`, `memory_profile`, `memory_forget`, `graph_query`, `ov_read_file`, `ov_write_file`, `ov_search`, `ov_list_dir`, `ov_grep`, `ov_tree`, `ov_session_commit`, `ov_ingest`

## Cross-Cutting Concerns

| Feature | Implementation |
|---------|---------------|
| Auth | JWT RS256 + API Key → tenant_id extraction |
| Rate Limit | Redis sliding window, per-tenant per-endpoint |
| Circuit Breaker | sony/gobreaker per downstream service |
| CORS | Configurable origins + credentials |
| Request ID | UUID v7, X-Request-ID header |
| Timeout | 30s default, 120s ingestion, 10s search |

## Links

- [API](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md) · [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Gateway Spec](../../../specs/architecture/01-gateway.md)

## Owner

- **Team**: VNP Memory — Platform
