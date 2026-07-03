# CodeGraph Bootstrap Guide for Go Projects

This guide is the reusable part of the `kg-service` CodeGraph bootstrap.
It intentionally separates generic steps from repo-specific values so the same pattern can be
applied to another Go repository.

## Required vs Optional

### Required

- install the CodeGraph CLI
- run `codegraph init -i`
- add `.codegraph/config.json`
- add a repo instruction file such as `CLAUDE.md` or `.claude/CLAUDE.md`
- verify with `codegraph status` and `codegraph query`

### Optional convenience

- `examples/codegraph/codegraph-refresh.sh`
- `.githooks/post-commit`
- `make codegraph-refresh`
- `codegraph serve --mcp` for a local agent smoke test

## What is reusable

- the CodeGraph init flow
- the ignore pattern shape
- the `CLAUDE.md` instruction pattern
- the incremental refresh hook
- the verification commands

## What is repo-specific

- the exact repo root path
- the ignore list details for generated files in that codebase
- the directory layout summary in `CLAUDE.md`
- any project-specific search examples

## Bootstrap Steps

1. Install the CodeGraph CLI and make sure `codegraph` is on `PATH`.
2. From the repository root, run `codegraph init -i`.
3. Add a `.codegraph/config.json` that keeps the index focused on source code.
4. Add a root `CLAUDE.md` that tells agents to use CodeGraph first and avoid grep or direct reads unless the graph is incomplete.
5. Add a refresh helper and wire it into a post-commit hook or an equivalent local hook path.
6. Verify the index and MCP entrypoint with `codegraph status`, `codegraph query`, and
   `codegraph serve --mcp`.

Note: in CodeGraph v1.0.1, the CLI command is `codegraph query`; the older `search` wording is the
same idea and appears in some task/spec text.

## Suggested config

Use this baseline for a Go repository:

```json
{
  "languages": ["go"],
  "ignore": [
    ".codegraph/**",
    ".git/**",
    "bin/**",
    "dist/**",
    "vendor/**",
    "**/*_test.go",
    "**/*.pb.go"
  ],
  "autoSync": true
}
```

Adjust the ignore list if your repository has other generated files that should stay out of the
graph, such as generated clients, snapshots, or vendored third-party code.

## Suggested `CLAUDE.md`

Keep the instruction file short and explicit:

```md
# <repo-name> - Agent instructions

## CodeGraph first for repo-local navigation

For questions about structure, callers, callees, impact, or exact symbol lookup in this repo,
prefer CodeGraph before manual grep or direct file reads.

Required order:

1. `codegraph_explore`
2. `codegraph_search`
3. `codegraph_callers` / `codegraph_callees`
4. direct file reads only when the graph is incomplete or implementation detail is needed
```

## Suggested refresh hook

If your Git setup uses a tracked hooks directory, point `core.hooksPath` at it and keep a
`post-commit` hook that starts an incremental refresh in the background.

One simple setup is:

```bash
git config core.hooksPath .githooks
```

Example hook body:

```bash
#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
"$repo_root/examples/codegraph/codegraph-refresh.sh" --background >/dev/null 2>&1 || true
```

The helper should be safe to run even when CodeGraph is missing, so commits are never blocked by
local tooling gaps. That safety net is for refresh automation only; repo navigation should still
prefer CodeGraph first.

## Verification checklist

- `codegraph status`
- `codegraph query "<known symbol>"`
- `codegraph serve --mcp`

If any of those fail, confirm that the CLI is installed, the repo has been initialized with
`codegraph init -i`, and the `.codegraph/` directory is present at the repository root.
