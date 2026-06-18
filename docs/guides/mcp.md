# MCP Integration

Use this guide when your consumer is tool-oriented or agent-style and you want to interact with `kg-service` through the current MCP HTTP transport.

## When To Use MCP

Prefer REST when:

- you control direct service-to-service calls
- you want ordinary HTTP request/response integration
- your workflow maps cleanly to individual endpoints

Prefer MCP when:

- your client expects tool discovery and tool invocation
- you are integrating an assistant, agent, or tool-calling runtime
- you want a session-based wrapper over the existing KG capabilities

## Shared Behavior With REST

- MCP uses the same `Authorization: Bearer <api_key>` header on the connect request.
- The same identity, ACL, and validation model applies underneath.
- The same tenant-tier limiter is applied after identity resolution.

## 1. Open A Session

Create an SSE session:

```bash
curl -N \
  -H "Authorization: Bearer kgsk_test_alpha_admin" \
  http://127.0.0.1:8082/v1/mcp/connect
```

The server responds as `text/event-stream` and emits a `session` event containing `session_id`.

## 2. List Available Tools

Use the returned session ID with JSON-RPC:

```bash
curl -s \
  -X POST \
  -H "Authorization: Bearer kgsk_test_alpha_admin" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
  http://127.0.0.1:8082/v1/mcp/messages/<session_id>
```

Current tool set includes:

- `kg_search`
- `kg_search_rag`
- `kg_read_pattern`
- `kg_list_domains`
- `kg_list_templates`
- `kg_get_node`
- `kg_write_node`
- `kg_check_access`
- `kg_integrity`

## 3. Call A Tool

Example: list templates in `sample-policy`.

```bash
curl -s \
  -X POST \
  -H "Authorization: Bearer kgsk_test_alpha_admin" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"kg_list_templates","arguments":{"domain_id":"sample-policy"}}}' \
  http://127.0.0.1:8082/v1/mcp/messages/<session_id>
```

Example: execute a template.

```bash
curl -s \
  -X POST \
  -H "Authorization: Bearer kgsk_test_alpha_admin" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"kg_read_pattern","arguments":{"domain_id":"sample-policy","template_name":"action-guide","params":{"topic_key":"returns"}}}}' \
  http://127.0.0.1:8082/v1/mcp/messages/<session_id>
```

## Common MCP Errors

- Invalid or expired session: JSON-RPC error with code `-32000`
- Rate limit exceeded: JSON-RPC error with code `-32029`
- Invalid JSON body: JSON-RPC error with code `-32600`
- Unknown method or tool: JSON-RPC error with code `-32601`

## Practical Advice

- Treat MCP as a thin wrapper over the REST and service layers, not as a separate data plane.
- Reuse the same expectations for visibility, validation, and async write behavior that you use with REST.
- Use [Integration Workflows](./integration.md) for the domain and data lifecycle order; MCP does not remove those prerequisites.
