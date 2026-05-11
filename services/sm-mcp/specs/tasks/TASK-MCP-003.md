---
id: TASK-MCP-003
title: Data Models & Repositories
service: sm-mcp
status: Done
priority: P0
created: 2026-05-11
---

# Data Models & Repositories

## Objective
Implement the storage and persistence adapters.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-mcp
title: Technical Design — sm-mcp
service: sm-mcp
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-mcp

> **Group**: Supermemory | **gRPC Port**: 9076 | **Health Port**: 9121

> **🚨 DEPRECATION NOTICE**: This specification is obsolete. The service has been merged into `vnp-gateway` (Ref: [ARCH-008-absorb-sm-mcp-to-gateway]).


## 1. Service Overview

MCP (Model Context Protocol) server for AI agent integration. SSE/JSON-RPC transport, 5 registered tools delegating to peer Supermemory services.

## 2. Domain Layer

- **MCPTool**: name, description, input_schema, target_service, target_rpc
- **MCPSession**: session_id, org_id, user_id, api_key_id, created_at, last_active_at
- **MCPRequest**: jsonrpc version, method, params, id
- **MCPResponse**: result, error, id
- **MigrateMCPRequest**: source_format, data[], container_tags[]

## 3. MCP Tool Registry

| Tool | Input Schema | Target |
|------|-------------|--------|
| `add_memory` | `{content: string, metadata?: object, containerTags?: string[]}` | sm-document.CreateDocument |
| `search_memory` | `{q: string, limit?: int, containerTags?: string[]}` | sm-search.HybridSearch |
| `get_profile` | `{}` | sm-profile.GetProfile |
| `list_documents` | `{page?: int, limit?: int}` | sm-document.ListDocuments |
| `rag_query` | `{q: string, limit?: int}` | sm-search.RAGComplete |

## 4. Transport

- **SSE**: `GET /mcp/sse` — Server-Sent Events stream for real-time responses
- **JSON-RPC**: `POST /mcp/message` — Standard MCP protocol messages
- Session lifecycle: connect → authenticate (API key) → tool calls → disconnect

## 5. Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| sm-document | gRPC | add_memory, list_documents |
| sm-search | gRPC | search_memory, rag_query |
| sm-profile | gRPC | get_profile |
| sm-auth | gRPC | API key validation |

## 6. Observability

- **Metrics**: mcp_sessions_active (gauge), mcp_tool_calls_total, mcp_tool_latency_seconds
- **Health**: gRPC + HTTP /healthz on port 9121

---

> **Next Steps**: FEAT-001 (SSE Transport), FEAT-002 (Tool Registry), FEAT-003 (MCP Data Migration)

```

## Acceptance Criteria
- [x] Database schema / migrations created.
- [x] Repository implementations accurately query the data models.
