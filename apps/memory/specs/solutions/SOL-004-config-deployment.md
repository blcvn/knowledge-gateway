---
id: SOL-004
title: Unified Configuration & Deployment
version: 1.0.0
status: Proposed
priority: P1
created: 2026-05-14
linked_sol: SOL-001
---

# SOL-004: Unified Configuration & Deployment

## 1. Config Strategy

Single YAML config file merges all 35 service configs + gateway config. Sử dụng Viper với ENV override.

### Config Loading Priority
```
1. config.yaml (defaults)
2. config.{env}.yaml (environment-specific)
3. Environment variables (VNP_MEMORY_*)
4. Command-line flags
```

### Config Struct

```go
type Config struct {
    Server    ServerConfig    `mapstructure:"server"`
    Auth      AuthConfig      `mapstructure:"auth"`
    Postgres  PostgresConfig  `mapstructure:"postgres"`
    Neo4j     Neo4jConfig     `mapstructure:"neo4j"`
    Qdrant    QdrantConfig    `mapstructure:"qdrant"`
    Redis     RedisConfig     `mapstructure:"redis"`
    NATS      NATSConfig      `mapstructure:"nats"`
    MinIO     MinIOConfig     `mapstructure:"minio"`
    Bifrost   BifrostConfig   `mapstructure:"bifrost"`
    Cognee    CogneeConfig    `mapstructure:"cognee"`
    Graphiti  GraphitiConfig  `mapstructure:"graphiti"`
    Memobase  MemobaseConfig  `mapstructure:"memobase"`
    OpenViking OVConfig       `mapstructure:"openviking"`
    Zep       ZepConfig       `mapstructure:"zep"`
    Supermemory SMConfig      `mapstructure:"supermemory"`
    Platform  PlatformConfig  `mapstructure:"platform"`
}
```

---

## 2. Deployment Models

### 2.1 Development — Single Binary

```bash
# Start infra only (5 containers instead of 40+)
docker compose -f apps/memory/docker-compose.infra.yml up -d

# Run monolithic binary
go run ./apps/memory/cmd/server
```

### 2.2 Docker — Compact (1 app container + infra)

```yaml
# apps/memory/docker-compose.yml
services:
  vnp-memory:
    build:
      context: ../..
      dockerfile: apps/memory/Dockerfile
    ports: ["8080:8080", "8082:8082", "8083:8083"]
    depends_on: [postgresql, neo4j, redis, qdrant]
    
  postgresql:
    image: pgvector/pgvector:pg17
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: vnp_memory
      POSTGRES_USER: vnp
      POSTGRES_PASSWORD: vnp
    volumes: [pgdata:/var/lib/postgresql/data]

  neo4j:
    image: neo4j:5-enterprise
    ports: ["7474:7474", "7687:7687"]
    environment:
      NEO4J_AUTH: neo4j/password
      NEO4J_ACCEPT_LICENSE_AGREEMENT: "yes"

  qdrant:
    image: qdrant/qdrant:latest
    ports: ["6333:6333"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  minio:
    image: minio/minio:latest
    command: server /data
    ports: ["9000:9000"]

volumes:
  pgdata:
```

### 2.3 Dockerfile (Multi-stage)

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY gateway/ gateway/
COPY services/ services/
COPY pkg/ pkg/
COPY proto/ proto/
COPY apps/memory/ apps/memory/
RUN cd apps/memory && go build -o /vnp-memory ./cmd/server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /vnp-memory /usr/local/bin/vnp-memory
COPY apps/memory/configs/ /etc/vnp-memory/
EXPOSE 8080 8082 8083
ENTRYPOINT ["vnp-memory"]
```

### 2.4 Makefile

```makefile
.PHONY: build run dev docker test

build:
	cd apps/memory && go build -o ../../bin/vnp-memory ./cmd/server

run: build
	./bin/vnp-memory

dev:
	cd apps/memory && go run ./cmd/server

docker:
	docker build -t vnp-memory:latest -f apps/memory/Dockerfile .

test:
	cd apps/memory && go test ./...

infra-up:
	docker compose -f apps/memory/docker-compose.infra.yml up -d

infra-down:
	docker compose -f apps/memory/docker-compose.infra.yml down
```

---

## 3. Acceptance Criteria

| # | Criteria |
|---|----------|
| AC-1 | Single YAML config loads all engine-specific settings |
| AC-2 | ENV vars override config values (`VNP_MEMORY_POSTGRES_DSN`) |
| AC-3 | `docker compose up` starts 1 app + 5 infra containers |
| AC-4 | Dockerfile builds successfully with multi-stage |
| AC-5 | `make dev` starts the app in development mode |
