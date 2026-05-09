---
id: FEAT-005
title: MCP Server — 16 Agent Tools via SSE/HTTP Streamable
service: vnp-gateway
version: 1.0.0
status: Done
priority: P1
created: 2026-05-09
updated: 2026-05-09
linked_sol: SOL-001
---

## Mục Tiêu

Implement Model Context Protocol (MCP) server on port 8082 with SSE and HTTP Streamable transports, exposing 16 tools for AI agent integration.

## Scope

### In Scope
- MCP server on separate port (8082) — protocol isolation from REST
- SSE transport for long-lived connections
- HTTP Streamable transport (JSON-RPC 2.0)
- 16 tool definitions (memory_store, memory_recall, ov_read_file, etc.)
- Tool dispatch → gRPC client registry forwarding
- Authentication via MCP session tokens

### Out of Scope
- stdio transport (server-side only)
- MCP Resources and Prompts (tools only for v1)
- Custom MCP extensions

## Acceptance Criteria
- [ ] AC-1: Given MCP SSE connection, When tools/list called, Then return 16 tools with schemas
- [ ] AC-2: Given tool call `memory_store`, When valid params, Then auto-route to correct service
- [ ] AC-3: Given tool call `ov_read_file`, When file exists, Then return file content
- [ ] AC-4: Given unauthenticated MCP session, When tool called, Then return auth error
- [ ] AC-5: Given downstream service timeout, When MCP tool called, Then return error with retry hint

## Test Requirements
- **Unit tests**: Tool dispatch, schema validation
- **Integration tests**: Full SSE connection lifecycle with mock tools
- **Minimum coverage**: 80%
