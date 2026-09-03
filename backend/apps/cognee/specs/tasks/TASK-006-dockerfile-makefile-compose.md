---
id: TASK-006
title: "Dockerfile + Makefile + Docker Compose"
app: apps/cognee
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-005]
estimated: 2h
---

## Mục Tiêu

Tạo deployment artifacts: multi-stage Dockerfile, Makefile, docker-compose.yml cho local dev.

## Scope

### In Scope

```
🆕 Dockerfile              — Multi-stage build (builder + alpine runtime)
🆕 Makefile                 — build, run, test, docker targets
🆕 docker-compose.yml       — Local dev: NATS, PostgreSQL, Neo4j, Qdrant, MinIO, Redis
🆕 config.yaml              — Example config for docker-compose environment
🆕 .env.example             — Example ENV vars
```

## Thiết Kế Kỹ Thuật

### Dockerfile

```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
WORKDIR /workspace

# Copy shared packages first (cacheable)
COPY pkg/ pkg/

# Copy app
COPY apps/cognee/ apps/cognee/

WORKDIR /workspace/apps/cognee
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /cognee-app ./cmd/cognee/

# Stage 2: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /cognee-app /usr/local/bin/cognee-app
EXPOSE 8080 9011 9012 9013 9090
ENTRYPOINT ["cognee-app"]
```

### docker-compose.yml

```yaml
version: "3.8"
services:
  cognee-app:
    build:
      context: ../..
      dockerfile: apps/cognee/Dockerfile
    ports:
      - "8080:8080"   # Gateway REST
      - "9090:9090"   # Health
    depends_on: [postgres, nats, neo4j, qdrant, redis, minio]
    env_file: .env.example

  postgres:
    image: postgres:16-alpine
    ports: ["5432:5432"]
    environment: { POSTGRES_PASSWORD: password, POSTGRES_DB: cognee }

  nats:
    image: nats:2.10-alpine
    ports: ["4222:4222"]
    command: ["--jetstream", "--store_dir=/data"]

  neo4j:
    image: neo4j:5-community
    ports: ["7474:7474", "7687:7687"]
    environment: { NEO4J_AUTH: "neo4j/password" }

  qdrant:
    image: qdrant/qdrant:latest
    ports: ["6333:6333"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  minio:
    image: minio/minio:latest
    ports: ["9000:9000"]
    command: server /data
```

### Makefile

```makefile
.PHONY: build run test lint docker clean

build:
	go build -o bin/cognee-app ./cmd/cognee/

run: build
	./bin/cognee-app

test:
	go test ./internal/... -race -coverprofile=coverage.out

lint:
	golangci-lint run ./...

docker:
	docker build -t cognee-app:latest -f Dockerfile ../..

up:
	docker compose up -d

down:
	docker compose down

clean:
	rm -rf bin/ coverage.out
```

## Acceptance Criteria

- [x] AC-1: `make build` produces binary
- [x] AC-2: `docker build` succeeds, image < 50MB
- [x] AC-3: `docker compose up -d` starts all infra + app
- [x] AC-4: App connects to all infra services
- [x] AC-5: `curl localhost:8080/healthz` returns OK

## Definition of Done

- [x] Dockerfile, Makefile, docker-compose.yml, .env.example created
- [x] Docker build verified
- [x] Local dev stack works end-to-end
