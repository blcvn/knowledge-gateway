---
id: TASK-007
title: "Service Documentation"
app: apps/cognee
version: 1.0.0
status: Done
priority: P2
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-006]
estimated: 1h
---

## Mục Tiêu

Tạo documentation cho cognee monolith app.

## Scope

```
apps/cognee/docs/
├── README.md           — Quick start, purpose
├── architecture.md     — Embedded services diagram, communication patterns
├── configuration.md    — All ENV vars, defaults
└── runbook.md          — Startup, shutdown, troubleshooting
```

## Key Content

### README.md
- What is cognee-app (single binary embedding 3 services + gateway)
- Quick start: `docker compose up -d` → `curl localhost:8080/healthz`
- Architecture overview (link to architecture.md)

### architecture.md
- Embedded service supervisor model
- gRPC localhost communication
- NATS event flow
- Startup/shutdown sequence diagram
- Comparison with microservices deployment

### configuration.md
- Complete ENV var table (from Config struct)
- Default values
- Config precedence
- Per-service port configuration

### runbook.md
- How to start locally
- How to build Docker image
- How to debug individual embedded services
- Common error scenarios + resolution
- How to rollback to microservices deployment

## Acceptance Criteria

- [x] AC-1: README has working quick start
- [x] AC-2: architecture.md documents embedded model
- [x] AC-3: configuration.md lists all ENV vars
- [x] AC-4: runbook.md has troubleshooting guide

## Definition of Done

- [x] All 4 docs created
- [x] Quick start verified
