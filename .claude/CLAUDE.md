# kg-service - Claude Code instructions

## CodeGraph first for repo-local navigation

For code structure questions, callers/callees, impact, or exact symbol lookup in this repo, use CodeGraph first. Do not begin with grep or direct file reads when CodeGraph can answer the question.

Required order:

1. `codegraph_explore`
2. `codegraph_search`
3. `codegraph_callers` / `codegraph_callees`
4. direct file reads only when the graph is incomplete or deeper implementation detail is needed

## Local setup

- The repo is indexed by CodeGraph in `.codegraph/`.
- Refresh the local index with `make codegraph-refresh` or `scripts/codegraph-refresh.sh`.
- See `docs/codegraph/bootstrap-guide.md` for the reusable bootstrap pattern and repo-specific notes.
