---
id: TASK-006
title: "Dockerfile + Makefile + Docker Compose"
app: apps/memobase
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-005]
---

## Mục Tiêu

Tạo deployment artifacts: Dockerfile (multi-stage build), Makefile (build/run/test), docker-compose.yml (local dev with NATS, PostgreSQL, Redis).

## Scope

### In Scope
- `apps/memobase/Dockerfile` — Multi-stage Go build → scratch/distroless
- `apps/memobase/Makefile` — build, run, test, lint, docker targets
- `apps/memobase/docker-compose.yml` — Local dev environment

## Thiết Kế Kỹ Thuật

### Docker Compose Services
| Service | Image | Ports |
|---------|-------|-------|
| memobase-app | build: . | 8080, 8082, 9090 |
| postgres | postgres:17-alpine | 5432 |
| redis | redis:7-alpine | 6379 |
| nats | nats:2-alpine | 4222 |

### Makefile Targets
- `make build` — `go build -o bin/memobase ./cmd/memobase/`
- `make run` — Build + run binary
- `make test` — `go test -race ./...`
- `make lint` — `go vet ./...`
- `make docker-build` — `docker build -t memobase-app .`
- `make compose-up` — `docker compose up -d`
- `make compose-down` — `docker compose down`

## Acceptance Criteria

- [x] AC-1: `make build` produces single binary
- [x] AC-2: `docker build` produces image < 50MB
- [x] AC-3: `docker compose up` starts app + infra
- [x] AC-4: All Makefile targets work correctly
- [x] AC-5: `.env.example` documented for docker compose
