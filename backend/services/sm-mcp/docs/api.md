---
id: DOC-S02
service: sm-mcp
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-mcp — API Reference

## gRPC Service Definition

### MCP Tools

| Tool | Target Service | Description |
|------|---------------|-------------|
| `add_memory` | sm-memory | Create a new memory |
| `search_memory` | sm-search | Hybrid search |
| `get_profile` | sm-profile | Get user profile |
| `list_documents` | sm-document | List documents |
| `rag_query` | sm-search | RAG completion |

## RPCs: N/A — MCP server (SSE/JSON-RPC), not gRPC

## NATS Events

None — MCP server dispatches to gRPC services directly.
