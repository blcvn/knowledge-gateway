# 13 — Deployment Guide

---

## 1. Docker Compose (Development)

```yaml
version: "3.8"

services:
  # ─── UNIFIED GATEWAY ───
  vnp-gateway:
    build: ./gateway
    ports: ["8080:8080", "8081:8081", "8082:8082"]
    depends_on: [redis, cognee-ingestion, cognee-cognify, cognee-search, cognee-memory,
                 graphiti-ingestion, graphiti-search, graphiti-knowledge, graphiti-store, vnp-admin]
    environment:
      - JWT_PUBLIC_KEY_PATH=/secrets/jwt/public.pem
      - REDIS_URL=redis://redis:6379/0

  # ─── COGNEE SERVICES ───
  cognee-ingestion:
    build: ./services/cognee-ingestion
    ports: ["9011:9011"]
    depends_on: [postgres, nats, minio]
    environment:
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/cognee
      - NATS_URL=nats://nats:4222
      - S3_ENDPOINT=http://minio:9000

  cognee-cognify:
    build: ./services/cognee-cognify
    ports: ["9012:9012"]
    depends_on: [postgres, nats, neo4j, qdrant]
    environment:
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/cognee
      - NATS_URL=nats://nats:4222
      - NEO4J_URI=bolt://neo4j:7687
      - QDRANT_URL=http://qdrant:6333
      - LLM_GATEWAY_URL=http://bifrost:8080

  cognee-search:
    build: ./services/cognee-search
    ports: ["9013:9013"]
    depends_on: [neo4j, qdrant]
    environment:
      - NEO4J_URI=bolt://neo4j:7687
      - QDRANT_URL=http://qdrant:6333

  cognee-memory:
    build: ./services/cognee-memory
    ports: ["9014:9014"]
    depends_on: [postgres, redis, nats]
    environment:
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/cognee
      - REDIS_URL=redis://redis:6379/1
      - NATS_URL=nats://nats:4222

  # ─── GRAPHITI SERVICES ───
  graphiti-ingestion:
    build: ./services/graphiti-ingestion
    ports: ["9021:9021"]
    depends_on: [postgres, nats]
    environment:
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/graphiti
      - NATS_URL=nats://nats:4222

  graphiti-search:
    build: ./services/graphiti-search
    ports: ["9022:9022"]
    depends_on: [neo4j]
    environment:
      - NEO4J_URI=bolt://neo4j:7687

  graphiti-knowledge:
    build: ./services/graphiti-knowledge
    ports: ["9023:9023"]
    depends_on: [neo4j]
    environment:
      - NEO4J_URI=bolt://neo4j:7687
      - LLM_GATEWAY_URL=http://bifrost:8080

  graphiti-store:
    build: ./services/graphiti-store
    ports: ["9024:9024"]
    depends_on: [neo4j, postgres]
    environment:
      - NEO4J_URI=bolt://neo4j:7687
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/graphiti

  # ─── SHARED ADMIN ───
  vnp-admin:
    build: ./services/vnp-admin
    ports: ["9030:9030"]
    depends_on: [postgres, nats]
    environment:
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/admin
      - NATS_URL=nats://nats:4222

  # ─── INFRASTRUCTURE ───
  postgres:
    image: postgres:16-alpine
    ports: ["5432:5432"]
    environment:
      POSTGRES_PASSWORD: password
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./migrations/init.sql:/docker-entrypoint-initdb.d/init.sql

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  nats:
    image: nats:2.10-alpine
    ports: ["4222:4222", "8222:8222"]
    command: ["--jetstream", "--store_dir=/data"]
    volumes: [natsdata:/data]

  neo4j:
    image: neo4j:5-community
    ports: ["7474:7474", "7687:7687"]
    environment:
      NEO4J_AUTH: neo4j/password
      NEO4J_PLUGINS: '["apoc"]'
    volumes: [neo4jdata:/data]

  qdrant:
    image: qdrant/qdrant:latest
    ports: ["6333:6333", "6334:6334"]
    volumes: [qdrantdata:/qdrant/storage]

  minio:
    image: minio/minio:latest
    ports: ["9000:9000", "9090:9090"]
    command: server /data --console-address :9090
    volumes: [miniodata:/data]

volumes:
  pgdata:
  natsdata:
  neo4jdata:
  qdrantdata:
  miniodata:
```

---

## 2. Kubernetes (Production)

### Directory Structure
```
deploy/kubernetes/
├── base/
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── secrets.yaml (sealed)
│   ├── vnp-gateway/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── hpa.yaml
│   │   └── ingress.yaml
│   ├── cognee-ingestion/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── hpa.yaml
│   ├── ... (each service)
│   └── vnp-admin/
├── overlays/
│   ├── dev/
│   ├── staging/
│   └── production/
│       ├── kustomization.yaml
│       ├── replicas-patch.yaml
│       └── resources-patch.yaml
```

### HPA Configuration (Per Service)
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: cognee-cognify-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: cognee-cognify
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

---

## 3. CI/CD Pipeline

```yaml
# .github/workflows/ci.yml
name: CI/CD
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: make lint           # golangci-lint

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: make test           # go test ./... -race -cover
      - run: make test-integration  # testcontainers

  proto:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: bufbuild/buf-setup-action@v1
      - run: buf lint
      - run: buf breaking --against '.git#branch=main'

  build:
    needs: [lint, test, proto]
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [vnp-gateway, cognee-ingestion, cognee-cognify, cognee-search,
                  cognee-memory, graphiti-ingestion, graphiti-search,
                  graphiti-knowledge, graphiti-store, vnp-admin]
    steps:
      - uses: actions/checkout@v4
      - run: docker build -t ${{ matrix.service }}:${{ github.sha }} ./services/${{ matrix.service }}
      - run: docker push ${{ matrix.service }}:${{ github.sha }}

  deploy:
    needs: [build]
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - run: kubectl apply -k deploy/kubernetes/overlays/production
```

---

## 4. Makefile Targets

```makefile
.PHONY: proto lint test build deploy

proto:
	buf generate

lint:
	golangci-lint run ./...

test:
	go test ./... -race -coverprofile=coverage.out

test-integration:
	go test ./... -tags=integration -v

build:
	@for svc in vnp-gateway cognee-ingestion cognee-cognify cognee-search \
		cognee-memory graphiti-ingestion graphiti-search graphiti-knowledge \
		graphiti-store vnp-admin; do \
		docker build -t $$svc:latest ./services/$$svc; \
	done

up:
	docker compose -f deploy/docker-compose/docker-compose.yml up -d

down:
	docker compose -f deploy/docker-compose/docker-compose.yml down

migrate:
	go run ./cmd/migrate/main.go up

wire:
	@for svc in services/*/; do \
		cd $$svc && wire ./internal/infra/wire/ && cd -; \
	done
```
