# CodeGraph Example

This repository example packages the local CodeGraph bridge and sync tooling under
`examples/codegraph/`.

For the operator-facing walkthrough, see [CodeGraph Sync Bridge](/Users/anhdt/vnpay/knowledge/kg-service/docs/codegraph/sync-bridge.md).

## What it does

- reads the CodeGraph SQLite index from `.codegraph/codegraph.db`
- maps supported CodeGraph nodes and edges into `kg-service` node and relationship writes
- keeps a local sync manifest so repeated runs stay idempotent
- exposes three MCP tools for persistent lookup:
  - `kg_semantic_search`
  - `kg_fulltext_search`
  - `kg_code_template_query`

## Scripts

The canonical example scripts live here:

- `examples/codegraph/codegraph-example-build`
- `examples/codegraph/codegraph-example-sync`
- `examples/codegraph/codegraph-example-sync-dry`
- `examples/codegraph/codegraph-example-mcp`
- `examples/codegraph/codegraph-refresh.sh`
- `examples/codegraph/bootstrap-codegraph-ontology.sh`
- `examples/codegraph/verify-codegraph-ontology.sh`
- `examples/codegraph/deploy-compose-codegraph-runtime.sh`
- `examples/codegraph/validate-codegraph-runtime.sh`

`examples/codegraph/codegraph-example-sync --full` performs a full reindex by deleting the project nodes tracked in
the local manifest before the new batch is written.

## Environment

Use `examples/codegraph/.env.example` as the starting point.

Required values:

- `KG_SERVICE_URL`
- `KG_API_KEY`

Useful overrides:

- `PROJECT_PATH`
- `PROJECT_ID`
- `CODEGRAPH_DB_PATH`
- `KG_DOMAIN_ID`
- `KG_VISIBILITY`
- `KG_STATE_DIR`
- `KG_TEMPLATE_DOMAIN_ID`

## Mapping

### Node mapping

The bridge writes one KG node per supported CodeGraph node and uses a deterministic external ref:

`<project_id>:<codegraph_node_id>`

Supported node kinds:

- `function` -> `Function`
- `method` -> `Method`
- `struct` -> `Struct`
- `interface` -> `Interface`
- `file` -> `File`
- `import` -> `Package`
- `route` -> `Function`

Unsupported kinds are skipped during sync.

Node properties written to `kg-service` stay within the frozen `code-graph` schema surface:

- `name`
- `kind`
- `file`
- `line`
- `language`
- `project_id`
- `commit_sha`
- `signature` when available
- `docstring` when available
- `package` when available

### Relationship mapping

Supported CodeGraph edge kinds are mapped like this:

- `calls` -> `CALLS`
- `implements` -> `IMPLEMENTS`
- `contains` -> `CONTAINS`
- `references` -> `REFERENCES`
- `imports` -> `IMPORTS`
- `instantiates` -> `REFERENCES`
- `extends` -> `REFERENCES`

Relationship properties include the original CodeGraph edge kind and provenance when present.

### Idempotency

Nodes are upserted through the local manifest keyed by `external_ref`.

Relationships do not have an `external_ref` field in the current `kg-service` contract, so the bridge keeps a deterministic local relationship key:

`<project_id>:<from_external_ref>:<rel_type>:<to_external_ref>`

## MCP tools

The `mcp` script registers three tools against the bridge service:

- `kg_semantic_search` -> `POST /v1/kg/search/hybrid`
- `kg_fulltext_search` -> `POST /v1/kg/search/fulltext`
- `kg_code_template_query` -> `POST /v1/kg/read/template/code-graph/{template_name}`

See [docs/codegraph/sync-bridge.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/codegraph/sync-bridge.md) for example MCP client config and JSON-RPC calls.

## Notes

- The bridge uses `sqlite3` to query the local CodeGraph index.
- The bridge assumes `kg-service` is already bootstrapped with the frozen `code-graph` ontology.
