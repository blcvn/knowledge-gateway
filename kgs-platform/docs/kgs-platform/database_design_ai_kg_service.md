# Database Design & Data Models — ai-kg-service (kgs-platform)

> Tham chiếu: [LLD ai-kg-service](./lld_ai_kg_service.md)
> Source code: `services/ai-kg-service/kgs-platform/`

---

## 1. Tổng quan Storage Architecture

kgs-platform sử dụng **5 storage backend**, mỗi loại phục vụ mục đích riêng:

```
┌─────────────────────────────────────────────────────────────────┐
│                        kgs-platform                             │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌───────┐  ┌─────┐│
│  │ Neo4j    │  │ Qdrant   │  │ PostgreSQL│  │ Redis │  │NATS ││
│  │ (Graph)  │  │ (Vector) │  │ (Metadata)│  │(Cache)│  │(Msg)││
│  └──────────┘  └──────────┘  └───────────┘  └───────┘  └─────┘│
└─────────────────────────────────────────────────────────────────┘
```

| Storage    | Vai trò                                                  | Port mặc định |
|------------|----------------------------------------------------------|----------------|
| **Neo4j**  | Lưu trữ graph (nodes + edges), traversal, fulltext index | 7687 (bolt)    |
| **Qdrant** | Lưu embedding vectors cho semantic search                | 6333 (REST)    |
| **PostgreSQL** | Metadata: ontology, registry, versions, projections, rules | 5432     |
| **Redis**  | Cache (ontology, centrality), overlay staging, locks     | 6379           |
| **NATS**   | Event streaming (overlay lifecycle, rule events)         | 4222           |

### Configuration (conf.proto)

```protobuf
message Data {
  message Database {
    string driver = 1;  // "postgres"
    string source = 2;  // connection string
  }
  message Neo4j {
    string uri = 1;      // "bolt://localhost:7687"
    string user = 2;     // "neo4j"
    string password = 3;
    string database = 4; // "neo4j"
  }
  message Qdrant {
    string host = 1;       // "localhost"
    int32 port = 2;        // 6333
    string collection = 3; // collection prefix
    int32 vector_size = 4; // 1536 (default)
  }
  message Redis {
    string network = 1;
    string addr = 2;
    string password = 3;
    google.protobuf.Duration read_timeout = 4;
    google.protobuf.Duration write_timeout = 5;
  }
  message NATS {
    string url = 1;
    string stream = 2;
  }
  message Embedding {
    string provider = 1;   // "openai" | "aiproxy" | "deterministic"
    string api_key = 2;
    string model = 3;
    string base_url = 4;
    int32 vector_size = 5; // 1536
    google.protobuf.Duration timeout = 6;
    string path = 7;
  }

  Database database = 1;
  Redis redis = 2;
  Neo4j neo4j = 3;
  Qdrant qdrant = 5;
  NATS nats = 6;
  Embedding embedding = 7;
}
```

---

## 2. Multi-Tenancy & Namespace

### 2.1 Namespace Computation

Source: `internal/biz/namespace.go`

```go
func ComputeNamespace(appID, tenantID string, orgID ...string) string {
    if tenantID == "" {
        tenantID = "default"
    }
    if len(orgID) > 0 && orgID[0] != "" {
        return "graph/" + orgID[0] + "/" + appID + "/" + tenantID
    }
    return "graph/" + appID + "/" + tenantID
}
```

| Pattern | Ví dụ |
|---------|-------|
| `graph/{appID}/{tenantID}` | `graph/ba-agent-service/doc-123` |
| `graph/{orgID}/{appID}/{tenantID}` | `graph/acme/ba-agent-service/doc-123` |

### 2.2 Tenant Isolation theo Storage

| Storage | Isolation mechanism |
|---------|---------------------|
| **Neo4j** | Property `app_id` + `tenant_id` trên mỗi node/edge. Mọi query PHẢI filter theo 2 field này |
| **Qdrant** | Collection per app: `kgs-vectors-{app_id_sanitized}` |
| **PostgreSQL** | Column `app_id` + `tenant_id`, composite UNIQUE index |
| **Redis** | Key prefix: `overlay:{namespace}:*`, `lock:{namespace}:*` |

### 2.3 Namespace Reservation (Neo4j)

```cypher
MERGE (n:__KGS_Namespace {app_id: $app_id})
ON CREATE SET n.created_at = datetime()
RETURN n
```

---

## 3. Neo4j — Graph Database

### 3.1 Node Model

Source: `internal/data/graph_node.go`

**Node Creation:**

```cypher
CREATE (n:{label} {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})
SET n += $props
RETURN n
```

- `{label}` là dynamic Neo4j label — tương ứng EntityType (vd: `USER_STORY`, `API`, `PERSONA`)
- `id` auto-generate UUID nếu không có trong `$props`
- `$props` là `map[string]any` — toàn bộ properties user-defined

**Node Properties (system):**

| Property | Type | Mô tả | Required |
|----------|------|--------|----------|
| `app_id` | string | Application ID | Yes |
| `tenant_id` | string | Tenant ID, default `"default"` | Yes |
| `id` | string | UUID node ID | Yes (auto-gen) |

**Node Properties (user-defined, ví dụ):**

| Property | Type | Mô tả |
|----------|------|--------|
| `name` | string | Node name — indexed cho fulltext search |
| `description` | string | Description — indexed cho fulltext search |
| `content` | string | Content — indexed cho fulltext search |
| `domain` | string | Domain tag |
| `domains` | []string | Multiple domain tags |
| `confidence` | float64 | Confidence score 0.0–1.0 |
| `version` | int | Optimistic lock version |
| Bất kỳ key nào | any | Custom properties |

**Node Retrieval:**

```cypher
MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})
RETURN n
LIMIT 1
```

### 3.2 Edge (Relationship) Model

Source: `internal/data/graph_edge.go`

**Edge Creation:**

```cypher
MATCH (a {app_id: $app_id, tenant_id: $tenant_id, id: $source_node_id})
MATCH (b {app_id: $app_id, tenant_id: $tenant_id, id: $target_node_id})
CREATE (a)-[rel:{relation_type} {app_id: $app_id, tenant_id: $tenant_id, id: $edge_id}]->(b)
SET rel += $props
RETURN rel
```

- `{relation_type}` là dynamic relationship type (vd: `REQUIRES`, `CONTAINS`, `MAPS_TO`)
- `$edge_id` auto-generate UUID
- `$props` custom properties trên edge

**Edge Properties (system):**

| Property | Type | Mô tả |
|----------|------|--------|
| `app_id` | string | Application ID |
| `tenant_id` | string | Tenant ID |
| `id` | string | UUID edge ID |

### 3.3 Indexes & Constraints

**Fulltext Index:**

```cypher
CREATE FULLTEXT INDEX kgs_fti_global IF NOT EXISTS
FOR (n) ON EACH [n.name, n.content, n.description]
```

- Dùng cho text search component trong HybridSearch
- Lucene-based — hỗ trợ scoring
- Tạo tự động khi service khởi động

> **Lưu ý**: Không có composite index trên `(app_id, tenant_id)` ở Neo4j. Tenant isolation dựa hoàn toàn vào property filter trong Cypher WHERE clause.

### 3.4 Cypher Query Patterns

**GetContext (Neighborhood Traversal):**

```cypher
-- direction = "both", depth = 2
MATCH p=(n {app_id:$app_id, tenant_id:$tenant_id, id:$node_id})-[*1..2]-(m)
WHERE m.app_id = $app_id AND m.tenant_id = $tenant_id
RETURN n, m, relationships(p) AS rels
```

**GetImpact (Downstream):**

```cypher
MATCH p=(n {app_id:$app_id, tenant_id:$tenant_id, id:$node_id})-[*1..{maxDepth}]->(m)
WHERE m.app_id = $app_id AND m.tenant_id = $tenant_id
RETURN n, m, relationships(p) AS rels
```

**GetCoverage (Upstream):**

```cypher
MATCH p=(m)-[*1..{maxDepth}]->(n {app_id:$app_id, tenant_id:$tenant_id, id:$node_id})
WHERE m.app_id = $app_id AND m.tenant_id = $tenant_id
RETURN n, m, relationships(p) AS rels
```

**GetSubgraph:**

```cypher
MATCH (n)
WHERE n.app_id = $app_id AND n.tenant_id = $tenant_id AND n.id IN $node_ids
OPTIONAL MATCH (n)-[r]-(m)
WHERE m.app_id = $app_id AND m.tenant_id = $tenant_id AND m.id IN $node_ids
RETURN n, r, m
```

**Bulk Create (Batch Upsert):**

```cypher
UNWIND $entities AS e
CREATE (n:{label} {app_id: $app_id, tenant_id: $tenant_id, id: e.id})
SET n += e
RETURN count(n) AS created
```

- Group entities by label → 1 query per label
- Batch size: 200 (configurable trong `Neo4jWriter`)

### 3.5 Graph Data Science (GDS)

**PageRank:**

```cypher
CALL gds.pageRank.stream($graph_name)
YIELD nodeId, score
RETURN gds.util.asNode(nodeId).id AS id, score
```

**Degree Centrality:**

```cypher
CALL gds.degree.stream($graph_name)
YIELD nodeId, score
RETURN gds.util.asNode(nodeId).id AS id, score
```

- Cache kết quả trong Redis: `kgs:gds:pagerank:{namespace}` (TTL: 15 phút)
- Dùng cho reranking trong HybridSearch (beta weight)

---

## 4. Qdrant — Vector Database

Source: `internal/data/qdrant.go`, `internal/search/vector.go`, `internal/batch/vector_indexer.go`

### 4.1 Collection Schema

**Collection naming:** `kgs-vectors-{app_id_sanitized}`
- `app_id` → lowercase, non-alphanumeric → `_`
- Fallback: `"default"` nếu empty

**Collection creation:**

```json
{
  "vectors": {
    "size": 1536,
    "distance": "Cosine"
  }
}
```

| Thuộc tính | Giá trị |
|-----------|---------|
| Vector size | 1536 (configurable, default cho OpenAI ada-002) |
| Distance metric | Cosine similarity |
| Collection per | App (không per tenant) |

### 4.2 Vector Point Structure

```go
type VectorPoint struct {
    ID      string         // Point ID (= node ID)
    Vector  []float32      // Embedding vector (1536 dims)
    Payload map[string]any // Metadata
}
```

**Payload structure khi index:**

```json
{
  "id": "entity-uuid",
  "label": "USER_STORY",
  "properties": {
    "name": "FR-001 Payment Gateway",
    "description": "...",
    "content": "..."
  },
  "app_id": "ba-agent-service"
}
```

### 4.3 Embedding Pipeline

Source: `internal/batch/vector_indexer.go`

1. Entity → text: concatenate `label + name + title + content + description`
2. Text → vector: EmbeddingClient (3 providers)
3. Vector + payload → Qdrant upsert

**Embedding Providers:**

| Provider | Implementation | Vector size |
|----------|----------------|-------------|
| `deterministic` | SHA-256 hash → normalize to float32 vector | Configurable |
| `openai` | OpenAI Embeddings API | 1536 (ada-002) |
| `aiproxy` | Internal AI Proxy service | Configurable |

### 4.4 Search Operations

**Vector Search:**

```
POST /collections/{collection}/points/search
{
  "vector": [0.123, 0.456, ...],
  "limit": 100,
  "score_threshold": 0.0,
  "with_payload": true,
  "with_vector": false
}
```

**Batch Search (cho semantic dedup):**

```
POST /collections/{collection}/points/search/batch
{
  "searches": [
    { "vector": [...], "limit": 1, "score_threshold": 0.95 }
  ]
}
```

- Semantic dedup threshold: **0.95** cosine similarity

### 4.5 Scored Result

```go
type ScoredPoint struct {
    ID      string         // Node ID
    Score   float64        // Cosine similarity score
    Payload map[string]any // Full metadata
}
```

---

## 5. PostgreSQL — Relational Metadata

Source: `internal/biz/ontology.go`, `internal/biz/registry.go`, `internal/biz/rules.go`, `internal/version/model.go`, `internal/projection/model.go`

Tất cả tables dùng GORM AutoMigrate. Soft delete via `deleted_at`.

### 5.1 Ontology

#### EntityType

```sql
CREATE TABLE entity_types (
    id          SERIAL PRIMARY KEY,
    app_id      VARCHAR(50) NOT NULL,
    tenant_id   VARCHAR(50) NOT NULL DEFAULT 'default',
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    schema      JSONB NOT NULL,       -- JSON Schema validation
    created_at  TIMESTAMP,
    updated_at  TIMESTAMP,
    deleted_at  TIMESTAMP,
    UNIQUE (app_id, tenant_id, name)
);
```

```go
type EntityType struct {
    ID          uint           `gorm:"primaryKey"`
    AppID       string         `gorm:"column:app_id;size:50"`
    TenantID    string         `gorm:"column:tenant_id;size:50;default:default"`
    Name        string         `gorm:"column:name;size:100"`
    Description string         `gorm:"column:description;type:text"`
    Schema      datatypes.JSON `gorm:"column:schema;type:jsonb"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

#### RelationType

```sql
CREATE TABLE relation_types (
    id           SERIAL PRIMARY KEY,
    app_id       VARCHAR(50) NOT NULL,
    tenant_id    VARCHAR(50) NOT NULL DEFAULT 'default',
    name         VARCHAR(100) NOT NULL,
    description  TEXT,
    properties   JSONB,              -- Edge property schema
    source_types JSONB,              -- Valid source EntityType names
    target_types JSONB,              -- Valid target EntityType names
    created_at   TIMESTAMP,
    updated_at   TIMESTAMP,
    deleted_at   TIMESTAMP,
    UNIQUE (app_id, tenant_id, name)
);
```

```go
type RelationType struct {
    ID          uint           `gorm:"primaryKey"`
    AppID       string         `gorm:"column:app_id;size:50"`
    TenantID    string         `gorm:"column:tenant_id;size:50;default:default"`
    Name        string         `gorm:"column:name;size:100"`
    Description string         `gorm:"column:description;type:text"`
    Properties  datatypes.JSON `gorm:"column:properties;type:jsonb"`
    SourceTypes datatypes.JSON `gorm:"column:source_types;type:jsonb"`
    TargetTypes datatypes.JSON `gorm:"column:target_types;type:jsonb"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

### 5.2 Registry

#### App

```sql
CREATE TABLE apps (
    app_id      VARCHAR(50) PRIMARY KEY,
    app_name    VARCHAR(200) NOT NULL,
    description TEXT,
    owner       VARCHAR(100) NOT NULL,
    status      VARCHAR(20) DEFAULT 'ACTIVE',  -- ACTIVE | INACTIVE | SUSPENDED
    created_at  TIMESTAMP,
    updated_at  TIMESTAMP,
    deleted_at  TIMESTAMP
);
```

```go
type App struct {
    AppID       string    `gorm:"column:app_id;primaryKey;size:50"`
    AppName     string    `gorm:"column:app_name;size:200"`
    Description string    `gorm:"column:description;type:text"`
    Owner       string    `gorm:"column:owner;size:100"`
    Status      string    `gorm:"column:status;size:20;default:ACTIVE"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt
    APIKeys     []APIKey  `gorm:"foreignKey:AppID"`
    Quotas      []Quota   `gorm:"foreignKey:AppID"`
}
```

#### APIKey

```sql
CREATE TABLE api_keys (
    key_hash   VARCHAR(80) PRIMARY KEY,  -- SHA-256 hash
    app_id     VARCHAR(50) NOT NULL,
    key_prefix VARCHAR(10) NOT NULL,     -- First 10 chars for display
    name       VARCHAR(100),
    scopes     VARCHAR(500),             -- Comma-separated: read,write,admin
    is_revoked BOOLEAN DEFAULT false,
    expires_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
CREATE INDEX idx_api_keys_app_id ON api_keys(app_id);
```

```go
type APIKey struct {
    KeyHash   string     `gorm:"column:key_hash;primaryKey;size:80"`
    AppID     string     `gorm:"column:app_id;size:50;index"`
    KeyPrefix string     `gorm:"column:key_prefix;size:10"`
    Name      string     `gorm:"column:name;size:100"`
    Scopes    string     `gorm:"column:scopes;size:500"`
    IsRevoked bool       `gorm:"column:is_revoked;default:false"`
    ExpiresAt *time.Time `gorm:"column:expires_at"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt
}
```

**API Key format:** `kgs_ak_{random_hex}` → SHA-256 → lưu `key_hash`
**Key prefix:** 10 ký tự đầu cho display (vd: `kgs_ak_a3b`)

#### Quota

```sql
CREATE TABLE quotas (
    id         SERIAL PRIMARY KEY,
    app_id     VARCHAR(50) NOT NULL,
    quota_type VARCHAR(50) NOT NULL,  -- requests_per_minute, max_nodes, etc.
    "limit"    BIGINT NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE (app_id, quota_type)
);
```

```go
type Quota struct {
    ID        uint   `gorm:"primaryKey"`
    AppID     string `gorm:"column:app_id;size:50"`
    QuotaType string `gorm:"column:quota_type;size:50"`
    Limit     int64  `gorm:"column:limit"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

#### AuditLog

```sql
CREATE TABLE audit_logs (
    id         SERIAL PRIMARY KEY,
    app_id     VARCHAR(50),
    action     VARCHAR(100) NOT NULL,
    actor      VARCHAR(100) NOT NULL,
    details    TEXT,
    created_at TIMESTAMP
);
CREATE INDEX idx_audit_app ON audit_logs(app_id);
CREATE INDEX idx_audit_time ON audit_logs(created_at);
```

### 5.3 Version Control

Source: `internal/version/model.go`

#### GraphVersion

```sql
CREATE TABLE graph_versions (
    id                VARCHAR(64) PRIMARY KEY,
    namespace         VARCHAR(255) NOT NULL,
    parent_id         VARCHAR(64),
    commit_message    VARCHAR(1024),
    entities_added    INTEGER,
    entities_modified INTEGER,
    entities_deleted  INTEGER,
    edges_added       INTEGER,
    edges_modified    INTEGER,
    edges_deleted     INTEGER,
    rollback_from_id  VARCHAR(64),
    created_at        TIMESTAMP
);
CREATE INDEX idx_version_ns_time ON graph_versions(namespace, created_at);
```

```go
type GraphVersion struct {
    ID               string    `gorm:"column:id;primaryKey;size:64"`
    Namespace        string    `gorm:"column:namespace;size:255;index:idx_ns_time"`
    ParentID         string    `gorm:"column:parent_id;size:64"`
    CommitMessage    string    `gorm:"column:commit_message;size:1024"`
    EntitiesAdded    int       `gorm:"column:entities_added"`
    EntitiesModified int       `gorm:"column:entities_modified"`
    EntitiesDeleted  int       `gorm:"column:entities_deleted"`
    EdgesAdded       int       `gorm:"column:edges_added"`
    EdgesModified    int       `gorm:"column:edges_modified"`
    EdgesDeleted     int       `gorm:"column:edges_deleted"`
    RollbackFromID   string    `gorm:"column:rollback_from_id;size:64"`
    CreatedAt        time.Time `gorm:"column:created_at;index:idx_ns_time"`
}
```

**Related types:**

```go
type ChangeSet struct {
    EntitiesAdded    int
    EntitiesModified int
    EntitiesDeleted  int
    EdgesAdded       int
    EdgesModified    int
    EdgesDeleted     int
    CommitMessage    string
}

type DiffResult struct {
    FromVersionID    string
    ToVersionID      string
    EntitiesAdded    int
    EntitiesModified int
    EntitiesDeleted  int
    EdgesAdded       int
    EdgesModified    int
    EdgesDeleted     int
}
```

### 5.4 Projection (Role-Based Views)

Source: `internal/projection/model.go`

#### ViewDefinition

```sql
CREATE TABLE view_definitions (
    id                   VARCHAR(64) PRIMARY KEY,
    app_id               VARCHAR(128) NOT NULL,
    tenant_id            VARCHAR(128) NOT NULL,
    role_name            VARCHAR(64) NOT NULL,
    allowed_entity_types TEXT,    -- JSON array serialized
    allowed_fields       TEXT,    -- JSON array serialized
    pii_mask_fields      TEXT,    -- JSON array serialized
    created_at           TIMESTAMP
);
CREATE INDEX idx_view_app_tenant_role ON view_definitions(app_id, tenant_id, role_name);
```

```go
type ViewDefinition struct {
    ID                 string    `gorm:"column:id;primaryKey;size:64"`
    AppID              string    `gorm:"column:app_id;size:128"`
    TenantID           string    `gorm:"column:tenant_id;size:128"`
    RoleName           string    `gorm:"column:role_name;size:64"`
    AllowedEntityTypes []string  `gorm:"column:allowed_entity_types;serializer:json"`
    AllowedFields      []string  `gorm:"column:allowed_fields;serializer:json"`
    PIIMaskFields      []string  `gorm:"column:pii_mask_fields;serializer:json"`
    CreatedAt          time.Time `gorm:"column:created_at"`
}
```

**Projection logic:**
- Filter nodes: chỉ giữ nodes có label nằm trong `AllowedEntityTypes`
- Filter fields: chỉ giữ properties có key trong `AllowedFields`
- PII masking: replace giá trị fields trong `PIIMaskFields` bằng `"***"`

### 5.5 Rules & Policies

#### Rule

```sql
CREATE TABLE rules (
    id           SERIAL PRIMARY KEY,
    app_id       VARCHAR(50) NOT NULL,
    tenant_id    VARCHAR(50) NOT NULL DEFAULT 'default',
    name         VARCHAR(100) NOT NULL,
    description  TEXT,
    trigger_type VARCHAR(20) NOT NULL,  -- SCHEDULED | ON_WRITE
    cron         VARCHAR(50),           -- Cron expression
    cypher_query TEXT NOT NULL,          -- Cypher to execute
    action       VARCHAR(50),           -- webhook, notification
    payload      JSONB,
    is_active    BOOLEAN DEFAULT true,
    created_at   TIMESTAMP,
    updated_at   TIMESTAMP,
    deleted_at   TIMESTAMP
);
```

#### RuleExecution

```sql
CREATE TABLE rule_executions (
    id         SERIAL PRIMARY KEY,
    app_id     VARCHAR(50) NOT NULL,
    tenant_id  VARCHAR(50) NOT NULL DEFAULT 'default',
    rule_id    INTEGER NOT NULL,
    status     VARCHAR(20) NOT NULL,  -- SUCCESS | FAILED
    message    TEXT,
    started_at TIMESTAMP,
    ended_at   TIMESTAMP
);
CREATE INDEX idx_rule_exec_time ON rule_executions(started_at);
```

#### Policy (OPA)

```sql
CREATE TABLE policies (
    id           SERIAL PRIMARY KEY,
    app_id       VARCHAR(50) NOT NULL,
    tenant_id    VARCHAR(50) NOT NULL DEFAULT 'default',
    name         VARCHAR(100) NOT NULL,
    description  TEXT,
    rego_content TEXT NOT NULL,  -- OPA Rego policy code
    is_active    BOOLEAN DEFAULT true,
    created_at   TIMESTAMP,
    updated_at   TIMESTAMP,
    deleted_at   TIMESTAMP
);
```

---

## 6. Redis — Cache & Overlay Storage

### 6.1 Key Patterns

| Key pattern | Mô tả | TTL |
|-------------|--------|-----|
| `ontology:entity:{app_id}:{name}` | Cached EntityType | 5 min |
| `ontology:relation:{app_id}:{name}` | Cached RelationType | 5 min |
| `kgs:gds:pagerank:{namespace}` | PageRank scores cache | 15 min |
| `overlay:{namespace}:{overlay_id}` | Overlay graph data | 1 hour |
| `lock:{namespace}:{entity_id}` | Entity-level distributed lock | Variable |

### 6.2 Overlay Model

Source: `internal/overlay/model.go`

```go
type OverlayGraph struct {
    OverlayID     string        // UUID
    Namespace     string        // graph/{appID}/{tenantID}
    SessionID     string        // User session binding
    BaseVersionID string        // Reference base version
    Status        Status        // CREATED | ACTIVE | COMMITTED | PARTIAL | DISCARDED
    EntitiesDelta []EntityDelta // Staged entity changes
    EdgesDelta    []EdgeDelta   // Staged edge changes
    CreatedAt     time.Time
    ExpiresAt     time.Time
    CommittedAt   *time.Time
}

type EntityDelta struct {
    ID         string
    Label      string
    Properties map[string]any
}

type EdgeDelta struct {
    ID         string
    SourceID   string
    TargetID   string
    Type       string
    Properties map[string]any
}

type CommitResult struct {
    NewVersionID      string
    EntitiesCommitted int
    EdgesCommitted    int
    ConflictsResolved int
}
```

**Overlay Status Flow:**

```
CREATED → ACTIVE → COMMITTED
                  → PARTIAL (partial commit)
                  → DISCARDED
```

---

## 7. Batch Processing

Source: `internal/batch/batch.go`

### 7.1 Data Structures

```go
type BatchUpsertRequest struct {
    AppID    string
    TenantID string
    Entities []Entity
}

type Entity struct {
    Label      string         // Neo4j label
    Properties map[string]any // All properties
}

type BatchUpsertResult struct {
    Created int
    Updated int
    Skipped int
}

const MaxBatchSize = 1000
```

### 7.2 Batch Pipeline

```
Input Entities
    │
    ▼
┌──────────────┐
│ Exact Dedup  │ ── Loại bỏ duplicate (label + properties hash)
└──────┬───────┘
       ▼
┌──────────────┐
│Semantic Dedup│ ── Qdrant vector similarity (threshold 0.95)
└──────┬───────┘
       ▼
┌──────────────┐
│ Neo4j Bulk   │ ── UNWIND + CREATE, batch size 200, group by label
│ Write        │
└──────┬───────┘
       ▼
┌──────────────┐
│ Qdrant       │ ── Embed + upsert vectors
│ Vector Index │
└──────────────┘
```

---

## 8. Hybrid Search Engine

Source: `internal/search/search.go`

### 8.1 Architecture

```go
type Engine struct {
    vector     VectorRetriever    // Qdrant semantic search
    text       TextRetriever      // Neo4j fulltext search
    centrality CentralityScorer   // PageRank/Degree from GDS
}

type Options struct {
    TopK            int      // Default: 10, Max: 100
    Alpha           float64  // Semantic weight (default: 0.6)
    Beta            float64  // Centrality weight (default: 0.2)
    EntityTypes     []string // Filter by types
    Domains         []string // Filter by domains
    MinConfidence   float64
    ProvenanceTypes []string
}

type Result struct {
    ID            string
    Label         string
    Properties    map[string]any
    SemanticScore float64
    TextScore     float64
    Centrality    float64
    Score         float64  // Final blended score
}
```

### 8.2 Scoring Formula

```
FinalScore = Alpha × SemanticScore + (1 - Alpha) × TextScore
           + Beta × Centrality
```

### 8.3 Hard Limits

Source: `internal/search/search.go:16`

```go
const (
    defaultTopK   = 10
    defaultAlpha  = 0.6
    defaultBeta   = 0.2
    maxSearchTopK = 100  // HARD CAP — không thể vượt qua
)
```

### 8.4 Search Pipeline

```
Query
  │
  ├──► Vector Search (Qdrant)  ──► SemanticScore per node
  │         embed(query) → cosine similarity search
  │
  ├──► Text Search (Neo4j FTI) ──► TextScore per node
  │         CALL db.index.fulltext.queryNodes(...)
  │
  └──► Blend Results
          │
          ▼
       Fetch Centrality (PageRank cache)
          │
          ▼
       Apply Filters (EntityTypes, Domains, MinConfidence)
          │
          ▼
       Sort by FinalScore DESC
          │
          ▼
       Limit to TopK (max 100)
```

### 8.5 Text Search Cypher

Source: `internal/search/text.go`

```cypher
CALL db.index.fulltext.queryNodes($index_name, $query)
YIELD node, score
WHERE node.app_id = $app_id AND node.tenant_id = $tenant_id
RETURN node.id AS id, labels(node)[0] AS label,
       properties(node) AS props, score
ORDER BY score DESC
LIMIT $limit
```

> **Quan trọng**: `query="*"` KHÔNG phải wildcard match-all trong Lucene. Kết quả không xác định (0 hoặc rất ít results).

---

## 9. Analytics

Source: `internal/analytics/`

### 9.1 Coverage Report

```cypher
MATCH (n {app_id: $app_id, tenant_id: $tenant_id})
WHERE $domain = '' OR coalesce(n.domain, '') = $domain
  OR $domain IN coalesce(n.domains, [])
WITH n, coalesce(head(labels(n)), 'Entity') AS entity_type
OPTIONAL MATCH (n)-[r]->()
WITH entity_type, n, count(r) AS outgoing_edges
WITH entity_type,
     count(n) AS total_entities,
     sum(CASE WHEN outgoing_edges > 0 THEN 1 ELSE 0 END) AS covered_entities
RETURN entity_type, total_entities, covered_entities
ORDER BY entity_type ASC
```

```go
type CoverageReport struct {
    Domain          string
    TotalEntities   int
    CoveredEntities int
    CoveragePercent float64
    UncoveredTypes  []string
    ByType          []CoverageByType
    GeneratedAt     time.Time
}

type CoverageByType struct {
    EntityType      string
    TotalEntities   int
    CoveredEntities int
    CoveragePercent float64
}
```

### 9.2 Traceability Matrix

```cypher
MATCH p=(s)-[*1..{maxHops}]->(t)
WHERE s.app_id = $app_id AND s.tenant_id = $tenant_id
  AND t.app_id = $app_id AND t.tenant_id = $tenant_id
  AND any(lbl IN labels(s) WHERE lbl IN $source_types)
  AND any(lbl IN labels(t) WHERE lbl IN $target_types)
RETURN s.id AS source_id,
       coalesce(s.name, s.id) AS source_name,
       coalesce(head(labels(s)), 'Entity') AS source_type,
       t.id AS target_id,
       coalesce(t.name, t.id) AS target_name,
       coalesce(head(labels(t)), 'Entity') AS target_type,
       length(p) AS hops,
       [rel IN relationships(p) | type(rel)] AS path
ORDER BY source_id ASC, hops ASC
LIMIT $limit  -- default 2000
```

```go
type TraceabilityMatrix struct {
    Matrix            []TraceabilityRow
    TotalSources      int
    TotalTargets      int
    ComputeDurationMs float64
}

type TraceabilityRow struct {
    SourceID   string
    SourceName string
    SourceType string
    Targets    []TraceabilityTarget
}

type TraceabilityTarget struct {
    EntityID string
    Name     string
    Type     string
    Hops     int
    Path     []string  // Relationship types along path
}
```

---

## 10. ERD Tổng hợp

### 10.1 Neo4j Graph Schema

```
(:EntityType {app_id, tenant_id, id, name, description, content, ...})
     │
     │ -[:RELATION_TYPE {app_id, tenant_id, id, ...}]->
     ▼
(:EntityType {app_id, tenant_id, id, name, description, content, ...})

Special:
(:__KGS_Namespace {app_id, created_at})
```

- Label = EntityType name (dynamic: `USER_STORY`, `API`, `PERSONA`, ...)
- Relationship type = RelationType name (dynamic: `REQUIRES`, `CONTAINS`, ...)
- Mọi node/edge đều có `app_id` + `tenant_id` cho tenant isolation

### 10.2 PostgreSQL Tables

```
┌──────────────┐     ┌──────────────┐
│ apps         │────<│ api_keys     │
│ (PK: app_id) │     │ (PK: key_hash│
└──────┬───────┘     └──────────────┘
       │
       ├────<┌──────────────┐
       │     │ quotas       │
       │     └──────────────┘
       │
       ├────<┌──────────────┐
       │     │ entity_types │
       │     │ (UK: app_id, │
       │     │  tenant_id,  │
       │     │  name)       │
       │     └──────────────┘
       │
       ├────<┌───────────────┐
       │     │ relation_types│
       │     │ (UK: app_id,  │
       │     │  tenant_id,   │
       │     │  name)        │
       │     └───────────────┘
       │
       ├────<┌──────────────┐      ┌──────────────────┐
       │     │ rules        │─────<│ rule_executions   │
       │     └──────────────┘      └──────────────────┘
       │
       ├────<┌──────────────┐
       │     │ policies     │
       │     └──────────────┘
       │
       ├────<┌──────────────────┐
       │     │ view_definitions │
       │     └──────────────────┘
       │
       └────<┌──────────────────┐
             │ graph_versions   │
             │ (namespace-scoped│
             └──────────────────┘
```

### 10.3 Qdrant Collections

```
Collection: kgs-vectors-{app_id}
  ├── Point: { id: "node-uuid", vector: float32[1536], payload: {...} }
  ├── Point: { id: "node-uuid", vector: float32[1536], payload: {...} }
  └── ...
```

### 10.4 Redis Keys

```
ontology:entity:{app_id}:{name}     → EntityType JSON (TTL: 5m)
ontology:relation:{app_id}:{name}   → RelationType JSON (TTL: 5m)
kgs:gds:pagerank:{namespace}        → PageRank scores (TTL: 15m)
overlay:{namespace}:{overlay_id}    → OverlayGraph JSON (TTL: 1h)
lock:{namespace}:{entity_id}        → Distributed lock (variable TTL)
```

---

## 11. Gaps & Lưu ý cho tích hợp

### 11.1 Thiếu RPC "Get Full Graph by Tenant"

Hiện tại kgs-platform **không có RPC** nào cho phép lấy toàn bộ nodes + edges theo `tenant_id`. Các lựa chọn hiện có:

| Approach | Vấn đề |
|----------|--------|
| `HybridSearch(query="*", topK=10000)` | TopK cap 100, `"*"` không match all |
| `GetSubgraph(nodeIDs)` | Cần biết trước nodeIDs |
| `GetContext(nodeID, depth=100)` | Cần root nodeID, không đảm bảo cover hết |

**Giải pháp đề xuất**: Thêm RPC `ListNodesByTenant` hoặc `GetFullGraph` query trực tiếp Neo4j:

```cypher
MATCH (n {app_id: $app_id, tenant_id: $tenant_id})
OPTIONAL MATCH (n)-[r]-(m {app_id: $app_id, tenant_id: $tenant_id})
RETURN n, r, m
```

### 11.2 Thiếu Batch Edge Creation

Chỉ có `CreateEdge` (single). Không có `BatchCreateEdges`.
→ Adapter cần concurrent goroutine pool.

### 11.3 Search TopK Hard Cap

`maxSearchTopK = 100` — nếu cần nhiều hơn phải sửa source hoặc thêm pagination.
