# CodeGraph Sync Bridge

This runbook covers the implemented bridge in [examples/codegraph](/Users/anhdt/vnpay/knowledge/kg-service/examples/codegraph).

## What It Does

- reads the local CodeGraph SQLite index from `.codegraph/codegraph.db`
- maps supported symbols and edges to the frozen `code-graph` ontology surface in `kg-service`
- persists a local sync manifest so repeated runs stay idempotent
- exposes MCP tools for semantic search, full-text search, and template-backed traversal

## Required Environment

Set these variables before running the bridge:

- `KG_SERVICE_URL`
- `KG_API_KEY`

Useful optional variables:

- `PROJECT_PATH`
- `PROJECT_ID`
- `CODEGRAPH_DB_PATH`
- `KG_DOMAIN_ID`
- `KG_VISIBILITY`
- `KG_STATE_DIR`
- `KG_TEMPLATE_DOMAIN_ID`
- `KG_DEFAULT_TOP_K`
- `KG_TEMPLATE_TIMEOUT_SEC`

See [examples/codegraph/.env.example](/Users/anhdt/vnpay/knowledge/kg-service/examples/codegraph/.env.example) for a starter file.

Example `.env`:

```bash
PROJECT_PATH=/Users/anhdt/vnpay/knowledge/kg-service
PROJECT_ID=kg-service
KG_SERVICE_URL=http://127.0.0.1:8082
KG_API_KEY=kgsk_test_alpha_admin
KG_DOMAIN_ID=code-graph
KG_VISIBILITY=private
KG_STATE_DIR=examples/codegraph/.state
KG_TEMPLATE_DOMAIN_ID=code-graph
```

## Common Commands

From the repo root:

```bash
make codegraph-example-build
make codegraph-example-sync-dry
make codegraph-example-sync
make codegraph-example-mcp
```

Or run the bridge scripts directly:

```bash
./examples/codegraph/codegraph-example-build
./examples/codegraph/codegraph-example-sync-dry
./examples/codegraph/codegraph-example-sync
./examples/codegraph/codegraph-example-mcp
```

Use `./examples/codegraph/codegraph-example-sync --full` for a full reindex against the local manifest.

Typical first run:

```bash
make codegraph-example-build
make codegraph-example-sync-dry
make codegraph-example-sync
```

## Mapping Summary

### Nodes

Supported CodeGraph node kinds are mapped as follows:

- `function` -> `Function`
- `method` -> `Method`
- `struct` -> `Struct`
- `interface` -> `Interface`
- `file` -> `File`
- `import` -> `Package`
- `route` -> `Function`

Each node uses a deterministic external reference:

`<project_id>:<codegraph_node_id>`

### Relationships

Supported edge kinds are mapped as follows:

- `calls` -> `CALLS`
- `implements` -> `IMPLEMENTS`
- `contains` -> `CONTAINS`
- `references` -> `REFERENCES`
- `imports` -> `IMPORTS`
- `instantiates` -> `REFERENCES`
- `extends` -> `REFERENCES`

Relationship keys are tracked locally as:

`<project_id>:<from_external_ref>:<rel_type>:<to_external_ref>`

## MCP Tools

The MCP server exposes these tools:

- `kg_semantic_search`
- `kg_fulltext_search`
- `kg_code_template_query`

They call these `kg-service` routes:

- `POST /v1/kg/search/hybrid`
- `POST /v1/kg/search/fulltext`
- `POST /v1/kg/read/template/{domain_id}/{template_name}`

### Example Tool Listing

After starting `./examples/codegraph/mcp`, clients can list the bridge tools with a JSON-RPC request
like this:

```json
{
  "jsonrpc": "2.0",
  "id": 0,
  "method": "tools/list"
}
```

### Underlying REST Examples

The bridge calls the same `kg-service` routes directly. These examples mirror the bridge payloads:

```bash
curl -s \
  -X POST \
  -H "Authorization: Bearer ${KG_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"query":"function that builds MCP tools","domain_ids":["code-graph"],"top_k":5,"semantic_weight":0.7,"fts_operator":"all_tokens"}' \
  "${KG_SERVICE_URL}/v1/kg/search/hybrid"
```

```bash
curl -s \
  -X POST \
  -H "Authorization: Bearer ${KG_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"query":"sync bridge","domain_ids":["code-graph"],"top_k":5,"mode":"all_tokens","fields":["name","docstring"]}' \
  "${KG_SERVICE_URL}/v1/kg/search/fulltext"
```

```bash
curl -s \
  -X POST \
  -H "Authorization: Bearer ${KG_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"params":{"node_id":"kg-service:method:..."}}' \
  "${KG_SERVICE_URL}/v1/kg/read/template/code-graph/code_callers"
```

### Example Client Config

Use the bridge as a stdio MCP server in your client config:

```json
{
  "mcpServers": {
    "codegraph-example": {
      "command": "/Users/anhdt/vnpay/knowledge/kg-service/examples/codegraph/codegraph-example-mcp",
      "env": {
        "KG_SERVICE_URL": "http://127.0.0.1:8082",
        "KG_API_KEY": "kgsk_test_alpha_admin",
        "KG_DOMAIN_ID": "code-graph",
        "KG_TEMPLATE_DOMAIN_ID": "code-graph"
      }
    }
  }
}
```

### Example MCP Calls

`kg_code_template_query` requires a `template_name` and an object `params` payload:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "kg_code_template_query",
    "arguments": {
      "template_name": "code_callers",
      "params": {
        "node_id": "kg-service:method:..."
      }
    }
  }
}
```

If `template_name` is missing or empty, the bridge returns a validation error before calling
`kg-service`.

`kg_semantic_search` accepts a free-form query and optional filters:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "kg_semantic_search",
    "arguments": {
      "query": "function that builds MCP tools",
      "domain_ids": ["code-graph"],
      "top_k": 5,
      "semantic_weight": 0.7,
      "fts_operator": "all_tokens"
    }
  }
}
```

`kg_fulltext_search` uses the same JSON-RPC envelope, but passes the full-text search shape:

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "kg_fulltext_search",
    "arguments": {
      "query": "sync bridge",
      "domain_ids": ["code-graph"],
      "top_k": 5,
      "mode": "all_tokens",
      "fields": ["name", "docstring"]
    }
  }
}
```

## Verification

Recommended smoke checks:

```bash
make codegraph-example-sync-dry
make codegraph-example-build
```

If `sync` fails, confirm:

- `kg-service` is reachable at `KG_SERVICE_URL`
- `KG_API_KEY` is valid
- `.codegraph/codegraph.db` exists and is readable
- `sqlite3` is installed locally
