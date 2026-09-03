<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **knowledge-gateway** (70952 symbols, 133328 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/knowledge-gateway/context` | Codebase overview, check index freshness |
| `gitnexus://repo/knowledge-gateway/clusters` | All functional areas |
| `gitnexus://repo/knowledge-gateway/processes` | All execution flows |
| `gitnexus://repo/knowledge-gateway/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

<!-- folder-architecture:start -->
# Folder Architecture — Mandatory Rules

> These rules MUST be followed by ALL agents (Claude Code, Antigravity) on every code generation task.
> Violating these rules breaks the Go workspace and module resolution.

## Canonical Layout

| Directory | Purpose | Rule |
|---|---|---|
| `backend/backend/shared/pkg/<name>/` | Shared Go packages (2+ services) | ONLY place for shared Go code |
| `backend/backend/services/<name>/` | Individual microservice code | ONLY place for service Go code |
| `backend/api/proto/` | Generated protobuf Go files | ONLY place for `*.pb.go` files |
| `deployment/dev/` | Docker, Nginx, migrations, scripts | ONLY place for deploy configs |
| `tools/` | Binary tools (protoc, etc.) | ONLY place for binary tools |
| `shared/proto/` | Proto source definitions | ONLY place for `.proto` sources |

## Never Do

- **NEVER** create Go files or `go.mod` directly at the repository root (`vnp-memory/`)
- **NEVER** create a `/pkg/` directory at root — use `backend/backend/shared/pkg/<name>/` instead
- **NEVER** create `/deploy/` — the canonical name is `deployment/`
- **NEVER** place `*.pb.go` files at root or inside services — use `backend/api/proto/`
- **NEVER** place binary tools at root — use `tools/<name>/`
- **NEVER** use `replace => ../../pkg/` in any `go.mod` — correct path is `../../shared/pkg/`

## Always Do

- **ALWAYS** check if a shared package already exists in `shared/pkg/` before creating a new one
- **ALWAYS** add new Go modules to `go.work` under the correct section
- **ALWAYS** run `go work sync` after adding/moving any Go module
- **ALWAYS** use `module vnp-memory/backend/shared/pkg/<name>` for shared package `go.mod`
- **ALWAYS** use `module vnp-memory/backend/services/<name>` for service `go.mod`
- **ALWAYS** place SQL migrations in `deployment/dev/migrations/`

## Decision Tree — Where Does My Code Go?

```
New Go code?
  ├─ Used by 1 service only?    → backend/services/<name>/internal/
  ├─ Used by 2+ services?       → backend/shared/pkg/<name>/
  └─ Is it a proto generated?   → api/proto/<domain>/<version>/

New deployment file?
  └─ Always → deployment/dev/<type>/

New binary tool?
  └─ Always → tools/<name>/
```

For full rules see: `.agent/rules/folder-architecture.md`
<!-- folder-architecture:end -->
