---
trigger: always_on
---

# Folder Architecture Rules

> These rules are MANDATORY and apply to ALL AI agents (Antigravity, Claude Code) and contributors.
> Violating these rules breaks the monorepo build and module resolution.

---

## Canonical Folder Map

```
vnp-memory/
├── backend/api/proto/          ← Generated protobuf Go files (module: github.com/vnp-memory/api/proto)
├── backend/apps/               ← Frontend applications
├── backend/db/                 ← Database schemas, seed data
├── deployment/         ← ALL deployment configs (Docker, Nginx, migrations, scripts)
│   └── dev/
│       ├── migrations/ ← SQL migration files
│       └── ...
├── docs/               ← Project-level documentation
├── backend/gateway/            ← API gateway service
├── backend/kgs-platform/       ← READ-ONLY external submodule
├── references/         ← Reference materials (read-only)
├── scripts/            ← Build/CI shell scripts
├── backend/services/           ← Individual microservice code (one dir per service)
│   └── <service-name>/
│       ├── cmd/
│       ├── internal/
│       ├── go.mod
│       └── docs/
├── backend/shared/             ← ALL shared Go code used by 2+ services
│   ├── pkg/            ← Shared Go packages
│   │   ├── adapters/
│   │   ├── config/
│   │   ├── forward/
│   │   ├── graph/
│   │   ├── privacy/
│   │   ├── resilience/
│   │   ├── search/
│   │   ├── telemetry/
│   │   ├── tenant/
│   │   ├── tokenizer/
│   │   └── vectorstore/
│   └── proto/          ← Shared proto source definitions
├── backend/specs/              ← Specification documents, CRs, tasks
├── tests/              ← Integration tests
├── backend/tools/              ← Binary tools (protoc, etc.)
│   └── protoc3/
└── ui/                 ← Shared UI components/assets
```

---

## ARCH-001: No Go Code at Repository Root

**Rule:** NEVER create Go files (`.go`), `go.mod`, or Go package directories directly under the repository root (`vnp-memory/`).

**Forbidden patterns:**
```
vnp-memory/pkg/          ← FORBIDDEN (was incorrectly created)
vnp-memory/cognify/      ← FORBIDDEN
vnp-memory/ingestion/    ← FORBIDDEN
vnp-memory/search/       ← FORBIDDEN
vnp-memory/api/          ← FORBIDDEN (except api/proto which is canonical)
```

**Correct locations:**
- Shared Go package → `backend/backend/shared/pkg/<name>/`
- Service-specific code → `backend/backend/services/<service-name>/`
- Generated proto Go files → `backend/api/proto/<domain>/<version>/`

---

## ARCH-002: Shared Go Packages Must Live in `shared/pkg/`

**Rule:** Any Go package used by **2 or more services** MUST be placed in `backend/backend/shared/pkg/<name>/`, NOT at the root or inside any single service.

**Decision tree:**
```
Is this Go code?
  ├─ Used by only 1 service?  → backend/services/<service-name>/internal/
  ├─ Used by 2+ services?     → backend/shared/pkg/<name>/
  └─ Is it a binary tool?     → tools/<name>/
```

**When creating a new shared package:**
1. Create directory: `backend/backend/shared/pkg/<name>/`
2. Initialize: `go mod init vnp-memory/backend/shared/pkg/<name>` (use consistent naming)
3. Add to `go.work` under the `// Shared packages` section
4. Add `replace` directives in each consuming service's `go.mod`

**Allowed `go.mod` module names for shared packages:**
```
vnp-memory/backend/shared/pkg/<name>
```

**NEVER use these patterns for shared packages:**
```
vnp-memory/pkg/<name>          ← WRONG
github.com/vnp-memory/pkg/<name>  ← WRONG (inconsistent prefix)
```

---

## ARCH-003: All Deployment Files Go in `deployment/`

**Rule:** NEVER create a `/deploy/` directory at the repository root. The canonical deployment directory is `/deployment/`.

**Forbidden:**
```
vnp-memory/deploy/           ← FORBIDDEN
vnp-memory/deploy/dev/       ← FORBIDDEN
```

**Correct:**
```
vnp-memory/deployment/dev/
vnp-memory/deployment/dev/migrations/   ← SQL migrations here
vnp-memory/deployment/dev/config/       ← Service configs here
```

---

## ARCH-004: Proto Generated Files Must Go in `api/proto/`

**Rule:** Protobuf-generated Go files (`*.pb.go`, `*_grpc.pb.go`) MUST be placed inside `backend/api/proto/<domain>/<version>/`, NOT at the repository root or inside individual services.

**Forbidden patterns:**
```
vnp-memory/cognify/v1/cognify.pb.go     ← FORBIDDEN
vnp-memory/ingestion/v1/ingestion.pb.go ← FORBIDDEN
vnp-memory/search/v1/search.pb.go       ← FORBIDDEN
services/<name>/proto/*.pb.go           ← FORBIDDEN
```

**Correct pattern:**
```
api/proto/cognee/cognify/v1/cognify.pb.go
api/proto/cognee/ingestion/v1/ingestion.pb.go
api/proto/observe/v1/observe.pb.go
```

**Proto source files (`.proto`) go in `shared/proto/` or alongside `api/proto/` files.**

---

## ARCH-005: Binary Tools Go in `tools/`

**Rule:** Binary executables, extracted archives, and build tools MUST be placed in `tools/`, NOT at the repository root.

**Forbidden:**
```
vnp-memory/protoc3/     ← FORBIDDEN (has been moved)
vnp-memory/bin/         ← FORBIDDEN
```

**Correct:**
```
vnp-memory/tools/protoc3/
```

---

## ARCH-006: `go.work` Must Stay Consistent

**Rule:** After adding, moving, or removing any Go module, the agent MUST update `go.work` to reflect the correct paths. All `use` directives MUST point to valid directories.

**Enforcement checklist (run after any Go file/module changes):**
1. `grep "pkg/" go.work` — must only show `./backend/shared/pkg/` paths, never `./pkg/`
2. `go work sync` — must complete without errors
3. No `replace` directive in any `go.mod` should point to `../../pkg/`

---

## ARCH-007: New Service Scaffolding Rules

**Rule:** When creating a new Go microservice, ALWAYS place it in `backend/services/<name>/` with this structure:

```
backend/services/<service-name>/
├── cmd/
│   └── main.go           ← Entry point
├── internal/
│   ├── adapter/          ← gRPC/HTTP handlers, DB adapters
│   ├── domain/           ← Business entities
│   └── usecase/          ← Business logic + ports
├── go.mod                ← module vnp-memory/services/<service-name>
├── Dockerfile
└── docs/
    ├── README.md
    ├── api.md
    ├── architecture.md
    └── configuration.md
```

**`go.mod` module naming convention:**
```
module vnp-memory/services/<service-name>
```

---

## VIOLATION QUICK REFERENCE

| Action | Correct Location | WRONG (never do) |
|---|---|---|
| New shared Go package | `backend/backend/shared/pkg/<name>/` | `pkg/<name>/`, root level |
| New service | `backend/services/<name>/` | root level, `api/<name>/` |
| SQL migrations | `deployment/dev/migrations/` | `deploy/dev/migrations/` |
| Proto generated Go | `api/proto/<domain>/v1/` | `<domain>/v1/` at root |
| Binary tools | `tools/<name>/` | `<name>/` at root |
| Deployment configs | `deployment/dev/` | `deploy/dev/` |
| Shared proto source | `shared/proto/` | root level |

---

## Pre-Code-Generation Checklist

Before writing any new Go file, the agent MUST answer:

1. **Is this a shared package?** → If yes, does `backend/backend/shared/pkg/<name>/` already exist?
2. **Is this service-specific?** → Is it inside `services/<name>/internal/`?
3. **Is this a proto generated file?** → Will it land in `api/proto/`?
4. **Does `go.work` need updating?** → Will the new module be reachable?
5. **Are `go.mod replace` directives correct?** → Do they point to `../../backend/shared/pkg/` not `../../pkg/`?
