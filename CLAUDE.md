# kg-service - Agent instructions

## CodeGraph first for repo-local navigation

For code structure questions, callers/callees, impact, or exact symbol lookup in this repo, use CodeGraph first. Do not begin with grep or direct file reads when CodeGraph can answer the question.

Required order:

1. `codegraph_explore`
2. `codegraph_search`
3. `codegraph_callers` / `codegraph_callees`
4. direct file reads only when the graph is incomplete or implementation detail is needed

## Repo pointers

- `docs/codegraph/bootstrap-guide.md` documents the reusable bootstrap pattern for other Go repos.
- `examples/codegraph/codegraph-refresh.sh` refreshes the local index when you want it on demand.
- `.githooks/post-commit` triggers an incremental refresh after commits when enabled via
  `git config core.hooksPath .githooks`.
