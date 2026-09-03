---
id: DOC-S01
service: sm-mcp
version: 1.1.0
status: Deprecated
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-mcp

> **Group**: Supermemory | **gRPC Port**: 9076 | **Health Port**: 9121 | **Origin**: Supermemory

## Purpose

> **🚨 DEPRECATION NOTICE**: This service has been merged into `vnp-gateway`. See [ARCH-008-absorb-sm-mcp-to-gateway](../../../specs/architecture/ARCH-008-absorb-sm-mcp-to-gateway.md) for details.


Model Context Protocol (MCP) server for AI agent integration. Provides a standardized **SSE/JSON-RPC** transport layer that exposes Supermemory capabilities as MCP tools, enabling any MCP-compatible AI agent (Claude, GPT, etc.) to store/search/manage memories.

### Business Capability

- **MCP Tool Registry**: `add_memory`, `search_memory`, `get_profile`, `list_documents`, `rag_query`
- **SSE Transport**: Server-Sent Events for streaming responses
- **JSON-RPC**: Standard MCP protocol compliance
- **Session Management**: Per-session state with API key authentication
- **Migration Support**: Migrate existing MCP data to Supermemory format

## API Surface — MCP Tools

| Tool | Target Service | Description |
|------|---------------|-------------|
| `add_memory` | sm-document | Add content (text/URL) as a memory |
| `search_memory` | sm-search | Search memories by natural language query |
| `get_profile` | sm-profile | Get user profile and preferences |
| `list_documents` | sm-document | List stored documents with pagination |
| `rag_query` | sm-search | RAG completion with context retrieval |

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v3/mcp/has-login` | Check if session has previous login |
| POST | `/v3/documents/migrate-mcp` | Migrate MCP data to Supermemory |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| sm-document | gRPC | Document CRUD for add_memory, list_documents |
| sm-search | gRPC | Search + RAG for search_memory, rag_query |
| sm-profile | gRPC | Profile for get_profile |
| sm-auth | gRPC | API key validation for MCP sessions |

## Owner

- **Team**: VNP Memory — Supermemory
