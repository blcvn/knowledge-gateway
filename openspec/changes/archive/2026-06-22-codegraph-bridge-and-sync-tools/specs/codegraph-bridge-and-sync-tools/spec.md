# codegraph-bridge-and-sync-tools

## Requirements

### Requirement: The bridge uses the current kg-service API surface
The bridge SHALL integrate with the existing `/v1/*` API surface of `kg-service`.

#### Scenario: Adapter probes health using the public route
- WHEN the bridge checks service health
- THEN it SHALL use `GET /healthz`

#### Scenario: Node sync uses the current write contract
- WHEN the bridge writes a source-code symbol node
- THEN the payload SHALL conform to the current node write contract with `domain_id`, `node_type`,
  `properties`, `visibility`, and `external_ref`

#### Scenario: Relationship sync uses the current relationship contract
- WHEN the bridge writes a source-code relationship
- THEN the payload SHALL conform to the current relationship write contract with `rel_type`,
  `from_node_id`, `to_node_id`, `domain_id`, and optional `properties`

### Requirement: Persistent traversal uses template-backed queries
The bridge tooling SHALL use template-backed traversal as the baseline persistent graph lookup path.

#### Scenario: MCP tool executes a code-graph template
- WHEN the MCP tooling performs persistent callers, callees, impact, or implements lookup
- THEN it SHALL execute `POST /v1/kg/read/template/{domain_id}/{template_name}`

### Requirement: Sync is idempotent by external_ref
Repeated sync runs SHALL use deterministic `external_ref` values so the same symbol is not created
multiple times as duplicate logical nodes.

#### Scenario: Repeated sync of the same symbol
- GIVEN a symbol maps to `<project_id>:<symbol_id>`
- WHEN the bridge syncs the symbol multiple times
- THEN the sync workflow SHALL treat subsequent writes as updates of the same logical symbol
