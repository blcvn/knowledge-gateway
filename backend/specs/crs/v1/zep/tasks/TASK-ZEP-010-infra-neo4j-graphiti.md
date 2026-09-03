# TASK-ZEP-010 — Infrastructure: Neo4j 5.22+ Upgrade & Graphiti Deploy

**Task ID:** TASK-ZEP-010  
**Wave:** 4 (Graph Intelligence — Infrastructure)  
**Solution:** [SOL-ZEP-003](../solutions/SOL-ZEP-003-Temporal-Knowledge-Graph.md)  
**Depends on:** TASK-ZEP-008 (NATS events from PutMemory)  
**Ước tính:** 2h  
**Priority:** Critical — prerequisite cho zep-graph service

**Trạng thái:** ✅ Implemented  
**Ghi chú:** zep-graph: 6 .go - Neo4j + graphiti infra  
---

## Mục tiêu

Chuẩn bị infrastructure cho Temporal Knowledge Graph:
1. Upgrade Neo4j lên 5.22+ trong docker-compose
2. Deploy Graphiti service (Python — LLM entity extraction)
3. Tạo Neo4j schema (constraints + vector indexes)

---

## Công việc cụ thể

### 1. Cập nhật `deploy/dev/docker-compose.server.yaml`

Upgrade Neo4j và thêm Graphiti service:

```yaml
# UPGRADE: neo4j version → 5.22 (hoặc latest 5.x)
services:
  neo4j:
    image: neo4j:5.22-community     # UPGRADE từ version hiện tại
    environment:
      NEO4J_AUTH: neo4j/${NEO4J_PASSWORD}
      NEO4J_PLUGINS: '["apoc", "graph-data-science"]'
      NEO4J_dbms_security_procedures_unrestricted: "gds.*,apoc.*"
      # Enable vector index (5.22+)
      NEO4J_dbms_memory_heap_initial__size: "512m"
      NEO4J_dbms_memory_heap_max__size: "2g"
    ports:
      - "7474:7474"   # HTTP browser
      - "7687:7687"   # Bolt

  # NEW: Graphiti Python service (LLM entity extraction)
  graphiti:
    image: ghcr.io/getzep/graphiti:latest
    ports:
      - "8100:8100"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - NEO4J_URI=bolt://neo4j:7687
      - NEO4J_USER=neo4j
      - NEO4J_PASSWORD=${NEO4J_PASSWORD}
      - GRAPHITI_LOG_LEVEL=info
    depends_on:
      neo4j:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8100/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 30s
```

### 2. Tạo `.env.example` additions

```bash
# Thêm vào .env.example
NEO4J_PASSWORD=your_neo4j_password
OPENAI_API_KEY=sk-...   # required for Graphiti entity extraction
GRAPHITI_URL=http://graphiti:8100
```

### 3. Tạo Neo4j Schema Script `deploy/dev/neo4j/init_schema.cypher`

```cypher
// Entity node constraints
CREATE CONSTRAINT entity_uuid IF NOT EXISTS
    FOR (n:Entity) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT entity_group_name IF NOT EXISTS
    FOR (n:Entity) REQUIRE (n.group_id, n.name) IS UNIQUE;

// Vector index for semantic search (Neo4j 5.22+)
CREATE VECTOR INDEX entity_embedding_idx IF NOT EXISTS
    FOR (n:Entity) ON (n.embedding)
    OPTIONS {
        indexConfig: {
            `vector.dimensions`: 1536,
            `vector.similarity_function`: 'cosine'
        }
    };

// NOTE: Edge vector index via Neo4j 5.22+ relationship vector index
// (syntax depends on Neo4j version)
CREATE INDEX entity_group_idx IF NOT EXISTS FOR (n:Entity) ON (n.group_id);
```

### 4. Tạo Neo4j Health Check Script `deploy/dev/neo4j/health_check.sh`

```bash
#!/bin/bash
# Verify Neo4j 5.22+ is running with vector index support
cypher-shell -u neo4j -p ${NEO4J_PASSWORD} "CALL db.indexes() YIELD name, type WHERE type = 'VECTOR' RETURN count(*) as vector_indexes" 
```

### 5. Tạo Graphiti Client Config

**`apps/memory/configs/config.yaml` — thêm section:**
```yaml
graphiti:
  url: "http://graphiti:8100"  # hoặc localhost:8100 khi dev local
  timeout_seconds: 30
  max_retries: 3
```

### 6. Tạo `services/zep-graph/internal/adapter/graphiti/client.go`

```go
// GraphitiClient là HTTP client → Graphiti Python service
type GraphitiClient struct {
    baseURL    string
    httpClient *http.Client
    retrier    *resilience.CircuitBreaker // từ TASK-ZEP-001
}

// PutMemory gửi messages tới Graphiti để extract entities (10-20s)
// Returns: nodes, edges (với ValidAt/InvalidAt), episodes
func (c *GraphitiClient) PutMemory(ctx context.Context, req PutMemoryRequest) (*GraphitiResponse, error) { ... }

// Health checks Graphiti service availability
func (c *GraphitiClient) Health(ctx context.Context) error { ... }
```

---

## Acceptance Criteria

- [ ] `docker-compose up neo4j graphiti` không có lỗi
- [ ] Neo4j browser tại `http://localhost:7474` hiển thị version 5.22+
- [ ] Vector index `entity_embedding_idx` được tạo (verify với `SHOW INDEXES`)
- [ ] Graphiti health endpoint `http://localhost:8100/health` trả về 200
- [ ] GraphitiClient.Health() → nil error khi service running
- [ ] `go build ./services/zep-graph/...` (chỉ client file) không có lỗi

---

## Files tạo/thay đổi

```
deploy/dev/
├── docker-compose.server.yaml    (MODIFY: upgrade neo4j + add graphiti)
├── .env.example                  (MODIFY: thêm NEO4J_PASSWORD, GRAPHITI_URL)
└── neo4j/
    ├── init_schema.cypher        (NEW)
    └── health_check.sh           (NEW)

services/zep-graph/
├── internal/adapter/graphiti/
│   └── client.go                 (NEW)
└── ...

apps/memory/configs/config.yaml   (MODIFY: thêm graphiti section)
```

## Sau khi hoàn thành

Chạy: 
```bash
docker-compose -f deploy/dev/docker-compose.server.yaml up neo4j graphiti -d
# Verify:
curl http://localhost:8100/health
```
