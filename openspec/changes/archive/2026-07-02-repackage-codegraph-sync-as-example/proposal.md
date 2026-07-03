# Proposal: Repackage CodeGraph Sync As A Repository Example

## Problem

`codegraph-sync/` currently lives at the repository root as if it were a first-class product surface,
but in practice it is an implementation example that demonstrates how `kg-service` can ingest a local
CodeGraph index and expose MCP tooling on top of the existing HTTP API.

That packaging creates three problems:

1. the root layout is harder to scan because example-only code sits beside core service code;
2. the repository already has `examples/codegraph/`, but the implemented bridge does not live there;
3. docs, runbooks, guides, and API companion references still point at the old root-level layout,
   which makes the current guidance drift from the example structure we want contributors to follow.

## Proposed Solution

Treat the current CodeGraph bridge as a repository-owned example and move its code, tests, wrapper
scripts, and example configuration under `examples/codegraph/`.

Refresh the surrounding documentation in the same workstream so the latest repository guidance points
to the new example path and still reflects the current live `kg-service` API contract.

## Scope

### In scope

- move the `codegraph-sync` implementation, tests, scripts, and example env files under
  `examples/codegraph/`
- keep a stable contributor command surface through `Makefile` targets and any necessary compatibility
  wrappers
- update repository docs, guides, runbooks, and maintenance guidance that reference the old location
- refresh API companion documentation and validation references so they stay aligned with the current
  runtime and the new example layout

### Out of scope

- changes to the `kg-service` HTTP routes or request/response contracts
- ontology changes for the frozen `code-graph` domain
- new bridge capabilities beyond the existing build, sync, dry-run sync, and MCP flows

## Success Criteria

- contributors can find the runnable CodeGraph bridge example under `examples/codegraph/`
- code, tests, env examples, and operator instructions all reference the same example location
- repository-owned validation and Make targets still provide a repeatable CodeGraph flow after the move
- API documentation remains validated against the live route inventory and no longer points at stale
  CodeGraph example paths
