---
id: TASK-006
title: "Dockerfile + Makefile + Docker Compose"
app: apps/graphiti
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-005]
---

## Mục Tiêu

Tạo deployment artifacts cho development và production.

## Scope

### In Scope
- `Dockerfile` — Multi-stage build, distroless base
- `Makefile` — build, run, test, lint, docker targets
- `docker-compose.yml` — Local dev (Neo4j, Redis, NATS, app)

### Out of Scope
- Kubernetes manifests (future)
- CI/CD pipeline (future)

## Thiết Kế Kỹ Thuật

### Dockerfile (Multi-stage, Distroless)

```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /graphiti ./cmd/graphiti/

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /graphiti /graphiti
EXPOSE 8080 8082 9021 9022 9023 9024 9025 9090
USER nonroot:nonroot
ENTRYPOINT ["/graphiti"]
```

### Makefile Targets

```makefile
.PHONY: build run test lint docker

build:
	go build -o bin/graphiti ./cmd/graphiti/

run: build
	./bin/graphiti

test:
	go test ./... -v -race -cover

lint:
	golangci-lint run ./...

docker-build:
	docker build -t graphiti-app:latest .

docker-run:
	docker compose up -d

docker-down:
	docker compose down
```

### Docker Compose (Local Dev)

```yaml
services:
  graphiti-app:
    build: .
    ports:
      - "8080:8080"   # REST
      - "8082:8082"   # MCP
      - "9090:9090"   # Health
    environment:
      - NEO4J_URI=bolt://neo4j:7687
      - NEO4J_PASSWORD=graphiti
      - REDIS_ADDR=redis:6379
      - NATS_URL=nats://nats:4222
      - AUTH_DEV_MODE=true
      - LLM_API_KEY=${LLM_API_KEY}
    depends_on:
      neo4j:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_started

  neo4j:
    image: neo4j:5-community
    ports: ["7474:7474", "7687:7687"]
    environment:
      NEO4J_AUTH: neo4j/graphiti
      NEO4J_PLUGINS: '["apoc"]'
    healthcheck:
      test: ["CMD", "neo4j", "status"]
      interval: 10s
      retries: 5
    volumes:
      - neo4j_data:/data

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      retries: 3

  nats:
    image: nats:2.10-alpine
    ports: ["4222:4222", "8222:8222"]
    command: ["--jetstream", "--monitor", "8222"]

volumes:
  neo4j_data:
```

## Acceptance Criteria

- [x] AC-1: `docker build` produces image <50MB
- [x] AC-2: `docker compose up` starts app + all dependencies
- [x] AC-3: `make build` produces binary in `bin/`
- [x] AC-4: `make test` runs all tests
- [x] AC-5: `make lint` runs golangci-lint

## Definition of Done

- [x] Docker image builds successfully
- [x] Docker compose starts clean environment
- [x] Không có lint errors
