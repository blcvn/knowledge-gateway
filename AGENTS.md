# kg-service - Codex instructions

## CodeGraph first for repo-local navigation

For code structure questions, callers/callees, impact, or exact symbol lookup in this repo, use CodeGraph first. Do not start with grep or direct file reads when CodeGraph can answer the question.

Required order:

1. `codegraph_explore`
2. `codegraph_search`
3. `codegraph_callers` / `codegraph_callees`
4. direct file reads only when the graph is incomplete or deeper implementation detail is needed

## Local setup

- The repo is indexed by CodeGraph in `.codegraph/`.
- The MCP server is configured in `~/.codex/config.toml` through `codegraph serve --mcp`.
- Refresh the local index with `make codegraph-refresh` or `scripts/codegraph-refresh.sh`.

## Repo pointers

- `CLAUDE.md` and `.claude/CLAUDE.md` carry the same repo-local navigation guidance.
- `docs/codegraph/bootstrap-guide.md` documents the reusable bootstrap pattern for other Go repos.
