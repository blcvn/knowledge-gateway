# Unified Data Models + Deployment

---

## 1. Shared Graph Types (`pkg/graph/`)

```go
// EntityNode — shared across Cognee KG + Graphiti episodic
type EntityNode struct {
    ID         string            `json:"id"`
    GroupID    string            `json:"group_id"`    // tenant isolation
    Name       string            `json:"name"`
    Type       string            `json:"type"`        // person, org, concept...
    Summary    string            `json:"summary"`
    Embedding  []float64         `json:"embedding"`
    Properties map[string]any    `json:"properties"`
    CreatedAt  time.Time         `json:"created_at"`
    ValidAt    *time.Time        `json:"valid_at"`    // bi-temporal
    InvalidAt  *time.Time        `json:"invalid_at"`
}

// EntityEdge — relationship between nodes
type EntityEdge struct {
    ID         string    `json:"id"`
    SourceID   string    `json:"source_id"`
    TargetID   string    `json:"target_id"`
    Type       string    `json:"type"`
    Fact       string    `json:"fact"`
    Weight     float64   `json:"weight"`
    Embedding  []float64 `json:"embedding"`
    CreatedAt  time.Time `json:"created_at"`
    ValidAt    *time.Time
    InvalidAt  *time.Time
}

// EpisodicNode — Graphiti-specific
type EpisodicNode struct {
    ID        string    `json:"id"`
    GroupID   string    `json:"group_id"`
    Content   string    `json:"content"`
    Source    string    `json:"source"`     // text, message, json
    SourceDesc string  `json:"source_description"`
    ValidAt   time.Time `json:"valid_at"`
}
```

## 2. Profile Types (`pkg/profile/`)

```go
// Profile — Memobase structured profile
type Profile struct {
    ID        uuid.UUID `json:"id"`
    UserID    string    `json:"user_id"`
    ProjectID string    `json:"project_id"`
    Topic     string    `json:"topic"`
    SubTopic  string    `json:"sub_topic"`
    Content   string    `json:"content"`
    UpdatedAt time.Time `json:"updated_at"`
}

// ChatBlob — conversation input
type ChatBlob struct {
    ID        uuid.UUID      `json:"id"`
    UserID    string         `json:"user_id"`
    ProjectID string         `json:"project_id"`
    Messages  []ChatMessage  `json:"messages"`
    TokenCount int           `json:"token_count"`
    CreatedAt time.Time      `json:"created_at"`
}

type ChatMessage struct {
    Role    string `json:"role"`    // user, assistant, system
    Content string `json:"content"`
    Alias   string `json:"alias,omitempty"`
}

// UserEvent — cross-domain event
type UserEvent struct {
    ID        uuid.UUID  `json:"id"`
    UserID    string     `json:"user_id"`
    TenantID  string     `json:"tenant_id"`
    Source    string     `json:"source"`   // memobase, graphiti, cognee
    Content   string     `json:"content"`
    Tags      []string   `json:"tags"`
    Embedding []float64  `json:"embedding"`
    CreatedAt time.Time  `json:"created_at"`
}
```

---

## 3. Deployment — Docker Compose (Dev)

```yaml
version: "3.9"
services:
  # === Gateway ===
  vnp-gateway:
    build: ./gateway
    ports: ["8080:8080", "8081:8081", "8082:8082"]
    depends_on: [postgresql, redis, nats]

  # === Cognee Domain ===
  cognee-ingestion:
    build: ./services/cognee-ingestion
    ports: ["9011:9011"]
    depends_on: [postgresql, minio, nats]
  cognee-cognify:
    build: ./services/cognee-cognify
    ports: ["9012:9012"]
    depends_on: [postgresql, neo4j, qdrant, nats]
  cognee-search:
    build: ./services/cognee-search
    ports: ["9013:9013"]
    depends_on: [neo4j, qdrant, redis]

  # === Graphiti Domain ===
  graphiti-ingestion:
    build: ./services/graphiti-ingestion
    ports: ["9021:9021"]
    depends_on: [nats]
  graphiti-search:
    build: ./services/graphiti-search
    ports: ["9022:9022"]
    depends_on: [redis]
  graphiti-knowledge:
    build: ./services/graphiti-knowledge
    ports: ["9023:9023"]
  graphiti-store:
    build: ./services/graphiti-store
    ports: ["9024:9024"]
    depends_on: [neo4j]

  # === Memobase Domain ===
  memobase-ingestion:
    build: ./services/memobase-ingestion
    ports: ["9031:9031"]
    depends_on: [postgresql, redis, nats]
  memobase-engine:
    build: ./services/memobase-engine
    ports: ["9032:9032"]
    depends_on: [postgresql, nats]
  memobase-context:
    build: ./services/memobase-context
    ports: ["9033:9033"]
    depends_on: [postgresql, redis]

  # === Platform ===
  vnp-event:
    build: ./services/vnp-event
    ports: ["9041:9041"]
    depends_on: [postgresql, nats]
  vnp-search-hub:
    build: ./services/vnp-search-hub
    ports: ["9042:9042"]
    depends_on: [redis]
  vnp-admin:
    build: ./services/vnp-admin
    ports: ["9050:9050"]
    depends_on: [postgresql, nats]

  # === Infrastructure ===
  postgresql:
    image: pgvector/pgvector:pg17
    ports: ["5432:5432"]
  neo4j:
    image: neo4j:5-enterprise
    ports: ["7474:7474", "7687:7687"]
  qdrant:
    image: qdrant/qdrant:latest
    ports: ["6333:6333"]
  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
  nats:
    image: nats:2-alpine
    command: ["--jetstream", "--store_dir=/data"]
    ports: ["4222:4222"]
  minio:
    image: minio/minio:latest
    ports: ["9000:9000"]
  bifrost:
    image: bifrost:latest
    ports: ["8443:8443"]
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    ports: ["4317:4317"]
```

---

## 4. Infrastructure Requirements

| Component | Dev | Production |
|-----------|-----|-----------|
| PostgreSQL | Single instance | 3-node Patroni cluster |
| Neo4j | Single instance | 3-node Core cluster |
| Qdrant | Single instance | 3-node cluster |
| Redis | Single instance | 6-node cluster |
| NATS | Single instance | 3-node JetStream cluster |
| MinIO | Single instance | Distributed (4+ nodes) |
