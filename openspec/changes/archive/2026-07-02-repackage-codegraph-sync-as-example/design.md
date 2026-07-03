# Design: Repackage CodeGraph Sync As A Repository Example

## Overview

This change reclassifies the implemented CodeGraph bridge from a root-level tooling package into a
repository example. The runtime behavior stays the same: the bridge still reads the local CodeGraph
SQLite index, maps symbols and relationships into the frozen `code-graph` ontology, syncs through the
existing `kg-service` API surface, and exposes the same three MCP tools.

The main change is repository organization and documentation ownership.

## Goals

- make `examples/codegraph/` the canonical home for the bridge implementation
- preserve the existing runnable workflow for contributors and validation scripts
- align all documentation with the new example path and the current HTTP contract

## Non-Goals

- rename the bridge capability itself
- redesign the bridge API adapter or mapping rules
- introduce new runtime features to `kg-service`

## Target Layout

The canonical example root becomes `examples/codegraph/`.

That example should own:

- the Go command entrypoint and internal bridge packages
- bridge tests
- executable helper scripts such as `build`, `sync`, `sync:dry`, and `mcp`
- example configuration such as `.env.example`
- example-local generated artifacts or ignored state directories where appropriate
- a local README that explains the example and links back to the operator runbook

The old root-level `codegraph-sync/` path should not remain the primary implementation location after
this change.

## Compatibility Strategy

The repository already documents and automates commands such as:

- `make codegraph-sync-build`
- `make codegraph-sync-sync-dry`
- `make codegraph-sync-sync`
- `make codegraph-sync-mcp`
- `scripts/validate-codegraph-runtime.sh`

To avoid breaking contributor workflows, this change should keep the command surface stable while
retargeting it to `examples/codegraph/`.

Acceptable compatibility patterns include:

- updating the `Makefile` targets to call the new example path directly
- updating validation scripts to resolve the example path directly
- keeping short compatibility wrappers only if they materially reduce migration churn

The important constraint is that docs and automation converge on one canonical example location.

## Documentation Updates

The documentation refresh should update every maintained reference that currently treats
`codegraph-sync/` as the canonical path. This includes:

- repository root overview docs
- `docs/codegraph/` bridge and integration docs
- deployment and runtime validation guides
- testing and MCP guides
- environment and maintenance references
- API companion docs where related guides or validation instructions reference the CodeGraph bridge

Older architecture notes may still mention historical names when describing evolution, but the active
operator and contributor guidance should prefer `examples/codegraph/`.

## API Documentation Impact

This change does not add or remove HTTP routes. Even so, the workstream should explicitly confirm that
the published API docs still match the live runtime after the doc refresh.

That means:

- rerun the route inventory check
- update `docs/api/README.md` and `docs/api/openapi.yaml` only if drift is found
- refresh any API-adjacent guide links or examples that still point to the old bridge location

## Verification Plan

1. move the example code and tests under `examples/codegraph/`
2. update the `Makefile` and repository-owned validation scripts to use the new path
3. update all maintained docs that reference the old root-level bridge path
4. run focused Go tests for the bridge package at its new location
5. run the route inventory check to confirm API docs still match the runtime
