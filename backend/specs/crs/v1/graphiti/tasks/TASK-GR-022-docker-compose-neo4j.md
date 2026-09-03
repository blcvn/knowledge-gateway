# TASK-GR-022 — Docker Compose + Neo4j + Deploy Config

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-022 |
| **Wave** | 4 (Admin & Observability) |
| **Component** | `deploy/dev/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-007 §4 |
| **Priority** | Medium |
| **Depends On** | TASK-GR-005, TASK-GR-020 |
| **Estimated** | 2h |

---

## Context

Thêm 4 services mới vào Docker Compose dev stack: `graphiti-store`, `graphiti-knowledge`, `graphiti-ingestion`, `graphiti-admin`. Thêm `neo4j` container. Tạo service config YAMLs.

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `deploy/dev/docker-compose.server.yaml` |
| CREATE | `deploy/dev/configs/graphiti-store.yaml` |
| CREATE | `deploy/dev/configs/graphiti-knowledge.yaml` |
| CREATE | `deploy/dev/configs/graphiti-ingestion.yaml` |
| CREATE | `deploy/dev/configs/graphiti-admin.yaml` |

---

## Implementation

### MODIFY: `deploy/dev/docker-compose.server.yaml`

Add to existing compose file:

```yaml
services:

  # ── Neo4j (Temporal Knowledge Graph Store) ──────────────────────────────
  neo4j:
    image: neo4j:5.21-community
    container_name: neo4j
    restart: unless-stopped
    environment:
      NEO4J_AUTH: neo4j/password123
      NEO4J_PLUGINS: '["apoc", "graph-data-science"]'
      NEO4J_dbms_security_procedures_allowlist: "apoc.*,gds.*"
      NEO4J_server_memory_heap_initial__size: "512m"
      NEO4J_server_memory_heap_max__size: "2g"
      NEO4J_server_memory_pagecache_size: "512m"
    ports:
      - "7474:7474"   # Neo4j Browser
      - "7687:7687"   # Bolt protocol
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:7474/db/neo4j/cluster/available"]
      interval: 10s
      timeout: 5s
      retries: 10
    networks: [backend]

  # ── Graphiti Store (Neo4j gRPC adapter) ──────────────────────────────────
  graphiti-store:
    image: ghcr.io/vnp-memory/graphiti-store:${VERSION:-latest}
    build:
      context: ../..
      dockerfile: services/graphiti-store/Dockerfile
    restart: unless-stopped
    depends_on:
      neo4j: { condition: service_healthy }
    environment:
      GRAPHITI_STORE_NEO4J_URI: bolt://neo4j:7687
      GRAPHITI_STORE_NEO4J_USER: neo4j
      GRAPHITI_STORE_NEO4J_PASSWORD: password123
      GRAPHITI_STORE_NEO4J_DATABASE: neo4j
      GRAPHITI_STORE_GRPC_PORT: "9090"
      GRAPHITI_AUTO_MIGRATE: "true"
      OTEL_EXPORTER_OTLP_ENDPOINT: http://jaeger:4317
    ports:
      - "9090:9090"   # gRPC
      - "9091:9091"   # Prometheus metrics
    networks: [backend]

  # ── Graphiti Knowledge (LLM orchestration) ───────────────────────────────
  graphiti-knowledge:
    image: ghcr.io/vnp-memory/graphiti-knowledge:${VERSION:-latest}
    build:
      context: ../..
      dockerfile: services/graphiti-knowledge/Dockerfile
    restart: unless-stopped
    depends_on:
      - graphiti-store
      - redis
    env_file:
      - configs/graphiti-knowledge.yaml
    environment:
      GRAPHITI_STORE_ADDR: graphiti-store:9090
      REDIS_ADDR: redis:6379
      OTEL_EXPORTER_OTLP_ENDPOINT: http://jaeger:4317
    ports:
      - "9092:9092"   # gRPC
      - "9093:9093"   # Prometheus metrics
    networks: [backend]

  # ── Graphiti Ingestion (Pipeline orchestrator) ────────────────────────────
  graphiti-ingestion:
    image: ghcr.io/vnp-memory/graphiti-ingestion:${VERSION:-latest}
    build:
      context: ../..
      dockerfile: services/graphiti-ingestion/Dockerfile
    restart: unless-stopped
    depends_on:
      - graphiti-store
      - graphiti-knowledge
      - nats
    env_file:
      - configs/graphiti-ingestion.yaml
    environment:
      GRAPHITI_STORE_ADDR:     graphiti-store:9090
      GRAPHITI_KNOWLEDGE_ADDR: graphiti-knowledge:9092
      NATS_URL: nats://nats:4222
      OTEL_EXPORTER_OTLP_ENDPOINT: http://jaeger:4317
      GRAPHITI_WORKER_POOL_BUFFER_SIZE: "100"
    ports:
      - "9094:9094"   # gRPC
      - "9095:9095"   # Prometheus metrics
    networks: [backend]

  # ── Graphiti Admin (Community + data management) ─────────────────────────
  graphiti-admin:
    image: ghcr.io/vnp-memory/graphiti-admin:${VERSION:-latest}
    build:
      context: ../..
      dockerfile: services/graphiti-admin/Dockerfile
    restart: unless-stopped
    depends_on:
      - graphiti-store
      - graphiti-knowledge
      - nats
    environment:
      GRAPHITI_STORE_ADDR:     graphiti-store:9090
      GRAPHITI_KNOWLEDGE_ADDR: graphiti-knowledge:9092
      NATS_URL: nats://nats:4222
      ADMIN_GRPC_PORT: "9096"
    ports:
      - "9096:9096"   # gRPC (internal admin only)
    networks: [backend]

volumes:
  neo4j_data:
  neo4j_logs:
```

### File 1: `deploy/dev/configs/graphiti-knowledge.yaml`

```yaml
# Graphiti Knowledge Service Configuration
OPENAI_API_KEY: "${OPENAI_API_KEY}"
BIFROST_BASE_URL: "${BIFROST_BASE_URL:-http://bifrost:8080}"
BIFROST_API_KEY: "${BIFROST_API_KEY}"

# Model selection
GRAPHITI_MEDIUM_MODEL: "openai/gpt-4o"
GRAPHITI_SMALL_MODEL: "openai/gpt-4o-mini"
GRAPHITI_EMBEDDER_MODEL: "text-embedding-3-small"

# LLM Cache (Redis)
REDIS_ADDR: "redis:6379"
GRAPHITI_LLM_CACHE_TTL_MINUTES: "60"
GRAPHITI_LLM_CACHE_ENABLED: "true"

# gRPC
GRAPHITI_KNOWLEDGE_GRPC_PORT: "9092"

# Service addresses
GRAPHITI_STORE_ADDR: "graphiti-store:9090"

# Postgres (for ontology storage)
DB_URL: "${DB_URL:-postgres://user:password@postgres:5432/vnp_memory}"
```

### File 2: `deploy/dev/configs/graphiti-ingestion.yaml`

```yaml
# Graphiti Ingestion Service Configuration

# Worker pool
GRAPHITI_WORKER_POOL_BUFFER_SIZE: "100"

# Chunker
GRAPHITI_CHUNK_SIZE_WORDS: "200"
GRAPHITI_CHUNK_OVERLAP_WORDS: "50"

# gRPC server
GRAPHITI_INGESTION_GRPC_PORT: "9094"

# Retry
GRAPHITI_INGESTION_MAX_RETRY: "3"
GRAPHITI_INGESTION_RETRY_DELAY_MS: "500"

# NATS subjects
GRAPHITI_NATS_EPISODE_SUBJECT: "graphiti.episode.ingested"
```

### File 3: `deploy/dev/configs/graphiti-store.yaml`

```yaml
# Graphiti Store Service Configuration

# Neo4j
GRAPHITI_STORE_NEO4J_URI: "bolt://neo4j:7687"
GRAPHITI_STORE_NEO4J_USER: "neo4j"
GRAPHITI_STORE_NEO4J_PASSWORD: "password123"
GRAPHITI_STORE_NEO4J_DATABASE: "neo4j"
GRAPHITI_STORE_NEO4J_MAX_POOL: "50"
GRAPHITI_STORE_NEO4J_CONN_TIMEOUT_SEC: "30"

# Migration
GRAPHITI_AUTO_MIGRATE: "true"
GRAPHITI_MIGRATION_DIR: "/app/migrations/graphiti"

# gRPC
GRAPHITI_STORE_GRPC_PORT: "9090"

# Batch sizes
GRAPHITI_BULK_SAVE_BATCH_SIZE: "100"
GRAPHITI_CLEAR_BATCH_SIZE: "1000"
```

### File 4: `deploy/dev/configs/graphiti-admin.yaml`

```yaml
# Graphiti Admin Service Configuration
ADMIN_GRPC_PORT: "9096"
GRAPHITI_STORE_ADDR: "graphiti-store:9090"
GRAPHITI_KNOWLEDGE_ADDR: "graphiti-knowledge:9092"
NATS_URL: "nats://nats:4222"

# Community detection
GRAPHITI_COMMUNITY_MIN_CLUSTER_SIZE: "3"
GRAPHITI_COMMUNITY_LPA_MAX_ITERATIONS: "10"
```

---

## Verification

```bash
# Start all graphiti services
cd deploy/dev
docker compose -f docker-compose.server.yaml up -d neo4j graphiti-store graphiti-knowledge graphiti-ingestion graphiti-admin

# Health checks
docker compose logs graphiti-store | grep "ONLINE\|started"
docker compose logs graphiti-knowledge | grep "started\|connected"

# Verify Neo4j browser accessible
open http://localhost:7474

# Verify gRPC services running
grpcurl -plaintext localhost:9090 list
grpcurl -plaintext localhost:9094 list
```

**Expected:**
- Neo4j Browser accessible at `http://localhost:7474`
- All 4 graphiti services healthy (gRPC responding)
- Neo4j indices created (check: `SHOW INDEXES` in Neo4j Browser)
