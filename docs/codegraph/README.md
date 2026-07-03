# CodeGraph Bootstrap

This folder documents the local CodeGraph bootstrap pattern for `kg-service` and for other Go
projects that want the same repo-local navigation workflow.

Start with [Bootstrap Guide](./bootstrap-guide.md) for the reusable setup steps.

Bootstrap guide legend:

- Required: CLI install, `codegraph init -i`, `.codegraph/config.json`, and an instruction file
- Optional convenience: refresh hook, helper script, Make target, and `codegraph serve --mcp`

Use [Integration Design](./codegraph-integration-design.md) when you need the full background,
including the future KG hybrid path.

For the implemented bridge and MCP tooling that sync the local index into `kg-service`, see
[CodeGraph Sync Bridge](./sync-bridge.md) and [examples/codegraph](/Users/anhdt/vnpay/knowledge/kg-service/examples/codegraph).
