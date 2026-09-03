# Solution: SOL-ZEP-003 — Temporal Knowledge Graph (Graph Service)

**CR ID:** CR-ZEP-003  
**Solution ID:** SOL-ZEP-003  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `services/zep-graph/` (port gRPC 9044) tích hợp với **Graphiti** (Python service, LLM entity extraction) qua REST/gRPC bridge. Graph Service consume NATS events `zep.memory.messages.ingested` → trigger async extraction → upsert vào **Neo4j 5.22+**. VNP Memory đã có Neo4j trong infrastructure stack — cần upgrade lên 5.22+.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `GraphFact` domain | `services/memory-service/internal/domain/zep/` | Có: UUID, Name, Fact, Episodes[] |
| `zep-graph` service | `apps/memory/internal/bootstrap/` | Có: đăng ký, chưa implement |
| Neo4j infra | `deploy/dev/docker-compose.server.yaml` | Có: Neo4j (version cần upgrade lên 5.22+) |
| NATS events `zep.graph.*` | Architecture | Defined: `zep.graph.*`, `zep.search.*` |

### Gap phân tích

- `GraphFact` thiếu `ValidAt`, `InvalidAt`, `ExpiredAt` temporal annotations
- Không có entity extraction pipeline (Graphiti integration)
- Không có 9-node ontology system
- Không có custom ontology API
- Không có Graph Data ingestion (non-message data)
- Không có Fact invalidation (`DELETE /facts/:uuid`)

### Infrastructure Gap

> **⚠️ CẦN NÂNG CẤP:** Neo4j phải là 5.22+ (current version có thể cũ hơn). Graphiti service là Python service mới cần deploy.

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service

```
services/zep-graph/
├── internal/
│   ├── domain/
│   │   ├── node.go            # EntityNode với 9 NodeType hierarchy
│   │   ├── edge.go            # TemporalEdge (Fact) với ValidAt/InvalidAt
│   │   ├── episode.go         # Episode = source message window
│   │   ├── ontology.go        # Custom ontology definitions
│   │   └── repository.go      # GraphRepository port (Neo4j)
│   ├── usecase/
│   │   ├── extract_entities.go    # NATS consumer → Graphiti → Neo4j
│   │   ├── add_graph_data.go      # Direct data ingestion (non-message)
│   │   ├── get_fact.go            # Get temporal edge by UUID
│   │   ├── delete_fact.go         # Invalidate fact (set invalid_at = now())
│   │   ├── set_ontology.go        # Set custom ontology for a graph
│   │   └── get_graph_data.go      # Fetch nodes/edges/episodes for search
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── graph_server.go
│   │   ├── subscriber/
│   │   │   └── message_events.go  # NATS: memory.messages.ingested consumer
│   │   └── graphiti/
│   │       └── client.go          # HTTP client → Graphiti Python service
│   └── infra/
│       └── neo4j/
│           ├── node_repo.go
│           ├── edge_repo.go
│           └── episode_repo.go
```

### 3.2. Domain Model

```go
// services/zep-graph/internal/domain/node.go

package domain

import "time"

type NodeType string

const (
    // Priority hierarchy (lower number = higher priority for extraction)
    NodeTypeUser         NodeType = "User"         // Priority 1: singleton per conversation
    NodeTypeAssistant    NodeType = "Assistant"     // Priority 1: singleton
    NodeTypePreference   NodeType = "Preference"    // Priority 2: low extraction threshold
    NodeTypeOrganization NodeType = "Organization"  // Priority 3
    NodeTypeEvent        NodeType = "Event"         // Priority 3
    NodeTypeLocation     NodeType = "Location"      // Priority 4
    NodeTypeDocument     NodeType = "Document"      // Priority 4
    NodeTypeTopic        NodeType = "Topic"         // Priority 5
    NodeTypeObject       NodeType = "Object"        // Priority 6: last resort
)

var NodeTypePriority = map[NodeType]int{
    NodeTypeUser: 1, NodeTypeAssistant: 1,
    NodeTypePreference: 2,
    NodeTypeOrganization: 3, NodeTypeEvent: 3,
    NodeTypeLocation: 4, NodeTypeDocument: 4,
    NodeTypeTopic: 5,
    NodeTypeObject: 6,
}

type EntityNode struct {
    UUID      string
    GroupID   string      // session_id or user_id (scoping)
    Name      string      // entity name (e.g. "Alice", "Beta Inc")
    NodeType  NodeType
    Summary   string      // AI-generated summary
    Metadata  map[string]any
    CreatedAt time.Time
    UpdatedAt time.Time
}

// services/zep-graph/internal/domain/edge.go

type TemporalEdge struct {
    UUID         string
    GroupID      string
    SourceNodeID string      // source entity UUID
    TargetNodeID string      // target entity UUID
    Name         string      // relationship label (e.g. "WORKS_AT", "LIVES_IN")
    Fact         string      // human-readable: "Alice works at Beta Inc"
    FactRating   float64     // 0.0-1.0 quality score from LLM
    Episodes     []string    // episode UUIDs where this edge was extracted

    // Temporal annotations — the core Zep differentiator
    ValidAt    *time.Time  // when fact became true
    InvalidAt  *time.Time  // when fact stopped being true
    ExpiredAt  *time.Time  // when superseded by a newer fact

    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// Invalidate marks a fact as no longer valid (soft delete with timestamp)
func (e *TemporalEdge) Invalidate() {
    now := time.Now()
    e.InvalidAt = &now
}

// services/zep-graph/internal/domain/episode.go

type Episode struct {
    UUID       string
    GroupID    string
    Name       string      // "{groupID}-{messageUUID}" prefix to avoid collision
    Content    string      // raw text of the message window
    Source     string      // "message" | "graph_data"
    CreatedAt  time.Time
}
```

### 3.3. Graphiti Client (HTTP Bridge)

```go
// services/zep-graph/internal/adapter/graphiti/client.go

// Graphiti is a Python service that does LLM-powered entity extraction.
// VNP Memory calls it via HTTP REST.

type GraphitiClient struct {
    baseURL    string
    httpClient *http.Client
}

type PutMemoryRequest struct {
    GroupID  string    `json:"group_id"`   // session_id or user_id
    Messages []GraphitiMessage `json:"messages"`
    Ontology *OntologyConfig  `json:"ontology,omitempty"` // custom ontology
}

type GraphitiMessage struct {
    UUID    string `json:"uuid"`
    Content string `json:"content"`
    Role    string `json:"role"`
}

type GraphitiResponse struct {
    Nodes    []ExtractedNode `json:"nodes"`
    Edges    []ExtractedEdge `json:"edges"`
    Episodes []ExtractedEpisode `json:"episodes"`
}

type ExtractedEdge struct {
    Name         string     `json:"name"`
    Fact         string     `json:"fact"`
    FactRating   float64    `json:"fact_rating"`
    ValidAt      *time.Time `json:"valid_at"`
    InvalidAt    *time.Time `json:"invalid_at"`
    SourceEntity string     `json:"source_entity"`
    TargetEntity string     `json:"target_entity"`
}

// PutMemory sends messages to Graphiti for async entity extraction
// Expected latency: 10-20 seconds (LLM processing)
func (c *GraphitiClient) PutMemory(ctx context.Context, req PutMemoryRequest) (*GraphitiResponse, error) {
    body, _ := json.Marshal(req)
    resp, err := c.httpClient.Post(c.baseURL+"/v1/memories", "application/json", bytes.NewReader(body))
    if err != nil { return nil, fmt.Errorf("graphiti put_memory: %w", err) }
    defer resp.Body.Close()

    var result GraphitiResponse
    return &result, json.NewDecoder(resp.Body).Decode(&result)
}
```

### 3.4. Async Entity Extraction Pipeline

```go
// services/zep-graph/internal/usecase/extract_entities.go

type ExtractEntitiesUseCase struct {
    graphitiClient *graphiti.GraphitiClient
    nodeRepo       NodeRepository
    edgeRepo       EdgeRepository
    episodeRepo    EpisodeRepository
    publisher      EventPublisher    // NATS
    ontologyRepo   OntologyRepository
}

// Triggered by NATS "zep.memory.messages.ingested" event (async, 10-20s)
func (uc *ExtractEntitiesUseCase) Execute(ctx context.Context, event MessagesIngestedEvent) error {
    // 1. Load custom ontology for this group (if configured)
    ontology, _ := uc.ontologyRepo.GetForGroup(ctx, event.SessionID)

    msgs := convertToGraphiti(event.Messages)

    // 2a. Extract for session graph
    sessionResp, err := uc.graphitiClient.PutMemory(ctx, graphiti.PutMemoryRequest{
        GroupID:  event.SessionID,
        Messages: msgs,
        Ontology: ontology,
    })
    if err != nil {
        slog.Error("graphiti extraction failed for session", "session_id", event.SessionID, "error", err)
        return err  // Will be retried by NATS JetStream (max delivery = 3)
    }

    // 2b. Extract for user graph (if user linked)
    if event.UserID != nil {
        userResp, err := uc.graphitiClient.PutMemory(ctx, graphiti.PutMemoryRequest{
            GroupID:  *event.UserID,
            Messages: msgs,
            Ontology: ontology,
        })
        if err == nil {
            uc.upsertGraphData(ctx, *event.UserID, userResp)
        }
    }

    // 3. Upsert nodes, edges, episodes into Neo4j
    uc.upsertGraphData(ctx, event.SessionID, sessionResp)

    // 4. Publish extraction completed event → Search Service invalidates cache
    uc.publisher.Publish(ctx, "zep.graph.extraction.completed", GraphExtractionCompletedEvent{
        GroupID:   event.SessionID,
        EdgeCount: len(sessionResp.Edges),
        NodeCount: len(sessionResp.Nodes),
    })

    return nil
}

func (uc *ExtractEntitiesUseCase) upsertGraphData(ctx context.Context, groupID string, resp *graphiti.GraphitiResponse) {
    // Upsert nodes classified by NodeType priority
    for _, n := range resp.Nodes {
        node := &EntityNode{
            UUID:     n.UUID,
            GroupID:  groupID,
            Name:     n.Name,
            NodeType: classifyNodeType(n),
            Summary:  n.Summary,
        }
        uc.nodeRepo.Upsert(ctx, node)
    }

    // Upsert temporal edges with ValidAt/InvalidAt
    for _, e := range resp.Edges {
        edge := &TemporalEdge{
            UUID:         e.UUID,
            GroupID:      groupID,
            Name:         e.Name,
            Fact:         e.Fact,
            FactRating:   e.FactRating,
            ValidAt:      e.ValidAt,
            InvalidAt:    e.InvalidAt,
        }
        uc.edgeRepo.Upsert(ctx, edge)
    }

    // Upsert episodes (source message windows)
    for _, ep := range resp.Episodes {
        episode := &Episode{UUID: ep.UUID, GroupID: groupID, Content: ep.Content}
        uc.episodeRepo.Upsert(ctx, episode)
    }
}
```

### 3.5. Fact Invalidation (DELETE /facts/:uuid)

```go
// services/zep-graph/internal/usecase/delete_fact.go

func (uc *DeleteFactUseCase) Execute(ctx context.Context, factUUID, projectUUID string) error {
    edge, err := uc.edgeRepo.GetByUUID(ctx, factUUID)
    if err != nil { return ErrFactNotFound }

    // Soft-invalidate: set invalid_at = now()
    now := time.Now()
    edge.InvalidAt = &now
    edge.UpdatedAt = now

    if err := uc.edgeRepo.Update(ctx, edge); err != nil {
        return err
    }

    // Publish event → Search Service removes from cache
    uc.publisher.Publish(ctx, "zep.graph.fact.invalidated",
        FactInvalidatedEvent{FactUUID: factUUID, GroupID: edge.GroupID})

    return nil
}
```

### 3.6. Custom Ontology API

```go
// services/zep-graph/internal/domain/ontology.go

type EntityDefinition struct {
    Description string            `json:"description"`
    Fields      []string          `json:"fields"`      // expected attributes
}

type EdgeDefinition struct {
    Description string            `json:"description"`
    SourceTypes []NodeType        `json:"source_types"`
    TargetTypes []NodeType        `json:"target_types"`
}

type GraphOntology struct {
    GraphID  string
    Entities map[string]EntityDefinition  // custom node types
    Edges    map[string]EdgeDefinition    // custom edge types
    SetAt    time.Time
}

// POST /api/v2/graph/ontology
type SetOntologyRequest struct {
    GraphID  string                       `json:"graph_id"`
    Entities map[string]EntityDefinition  `json:"entities"`
    Edges    map[string]EdgeDefinition    `json:"edges"`
}

// Example: define "Product" entity
// {"entities": {"Product": {"description": "Product being discussed", "fields": ["name", "category"]}}}
```

### 3.7. Graph Data Ingestion (Non-Message)

```go
// services/zep-graph/internal/usecase/add_graph_data.go

type AddGraphDataUseCase struct {
    graphitiClient *graphiti.GraphitiClient
    nodeRepo       NodeRepository
    edgeRepo       EdgeRepository
    episodeRepo    EpisodeRepository
}

type AddGraphDataRequest struct {
    UserID  string  `json:"user_id"`   // target user's graph
    GraphID string  `json:"graph_id"`  // or specific graph scope
    Data    string  `json:"data"`      // raw text or JSON
    Type    string  `json:"type"`      // "text" | "json"
}

// POST /api/v2/graph/data — for CRM data, product catalogs, telemetry
func (uc *AddGraphDataUseCase) Execute(ctx context.Context, req AddGraphDataRequest) error {
    groupID := req.GraphID
    if req.UserID != "" { groupID = req.UserID }

    // Treat as synthetic message for extraction
    syntheticMsg := graphiti.GraphitiMessage{
        UUID:    newUUID(),
        Content: req.Data,
        Role:    "system",
    }

    resp, err := uc.graphitiClient.PutMemory(ctx, graphiti.PutMemoryRequest{
        GroupID:  groupID,
        Messages: []graphiti.GraphitiMessage{syntheticMsg},
    })
    if err != nil { return err }

    uc.upsertGraphData(ctx, groupID, resp)
    return nil
}
```

---

## 4. Neo4j Schema (Cypher)

```cypher
// Entity node constraints
CREATE CONSTRAINT entity_uuid IF NOT EXISTS
    FOR (n:Entity) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT entity_group_name IF NOT EXISTS
    FOR (n:Entity) REQUIRE (n.group_id, n.name) IS UNIQUE;

// Temporal edge properties (on relationship)
// WORKS_AT, LIVES_IN, MEMBER_OF, etc.
// Properties: uuid, fact, fact_rating, valid_at, invalid_at, expired_at

// Example temporal edge:
// (alice:User {name:"Alice"}) -[:WORKS_AT {
//   uuid: "edge-001",
//   fact: "Alice works at Beta Inc",
//   fact_rating: 0.95,
//   valid_at: datetime("2023-07-01"),
//   invalid_at: null
// }]-> (beta:Organization {name:"Beta Inc"})

// Index for temporal queries
CREATE INDEX entity_group_idx IF NOT EXISTS FOR (n:Entity) ON (n.group_id);
CREATE INDEX edge_valid_at_idx IF NOT EXISTS FOR ()-[r:*]-() ON (r.valid_at);
CREATE INDEX edge_invalid_at_idx IF NOT EXISTS FOR ()-[r:*]-() ON (r.invalid_at);
```

---

## 5. Temporal Query Examples (Cypher)

```cypher
-- Query: "Where did Alice work in 2022?"
MATCH (u:User {name:"Alice"}) -[r:WORKED_AT|WORKS_AT]-> (org:Organization)
WHERE (r.valid_at <= datetime("2022-06-01") OR r.valid_at IS NULL)
  AND (r.invalid_at IS NULL OR r.invalid_at > datetime("2022-06-01"))
RETURN org.name, r.fact, r.valid_at, r.invalid_at
ORDER BY r.valid_at DESC

-- Query: current facts only (no temporal filter)
MATCH (u:User {name:"Alice"}) -[r]-> (target)
WHERE r.invalid_at IS NULL AND r.expired_at IS NULL
RETURN u, r, target
```

---

## 6. Infrastructure Requirements

| Component | Version | Mục đích |
|-----------|---------|----------|
| **Neo4j** | **5.22+** (UPGRADE) | Temporal knowledge graph storage |
| **Graphiti** | Latest Python service | LLM entity extraction (external) |
| **NATS JetStream** | Embedded | Async pipeline (đã có) |

**Graphiti service deployment:**
```yaml
# deploy/dev/docker-compose.server.yaml (THÊM VÀO)
services:
  graphiti:
    image: zep-ai/graphiti:latest   # hoặc self-hosted
    ports: ["8100:8100"]
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}  # for entity extraction
      - NEO4J_URI=bolt://neo4j:7687
    depends_on: [neo4j]
```

---

## 7. API Endpoints

```
POST   /api/v2/graph/data        → AddGraphData (non-message ingestion)
POST   /api/v2/graph/ontology    → SetCustomOntology
GET    /api/v2/facts/{uuid}      → GetFact
DELETE /api/v2/facts/{uuid}      → InvalidateFact (set invalid_at=now())
GET    /api/v2/users/{id}/nodes  → GetUserNodes
GET    /api/v2/users/{id}/edges  → GetUserEdges
GET    /api/v2/users/{id}/episodes → GetUserEpisodes
```

---

## 8. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Neo4j 5.22+ upgrade + Graphiti service deploy | 1 ngày |
| **P2** | Domain model: Node, TemporalEdge, Episode, Ontology | 1 ngày |
| **P3** | Graphiti HTTP client (bridge) | 1 ngày |
| **P4** | NATS consumer + ExtractEntities pipeline | 3 ngày |
| **P5** | Neo4j repository (Cypher queries) | 2 ngày |
| **P6** | AddGraphData + SetOntology use cases | 1 ngày |
| **P7** | Fact CRUD (GetFact, InvalidateFact) | 1 ngày |
| **P8** | GroupID strategy + episode prefix | 0.5 ngày |
| **P9** | Gateway integration + tests | 2 ngày |

**Tổng:** ~12.5 ngày (Wave 4)

---

## 9. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| "Alice worked at Acme until last June, now at Beta" → 2 facts với temporal | Graphiti extracts ValidAt/InvalidAt from LLM |
| Query tháng 2022 → "worked at Acme"; 2024 → "works at Beta" | Cypher temporal filter: valid_at <= query_date AND (invalid_at IS NULL OR invalid_at > query_date) |
| POST /graph/data với JSON product catalog → graph | AddGraphData UC → synthetic message → Graphiti |
| Custom "Product" ontology → "MacBook Pro" extract đúng | SetOntology → Graphiti receives ontology config |
| DELETE /facts/:uuid → fact bị invalidate | InvalidateFact: set invalid_at = now() |
| Extraction 10-20s là OK | Async NATS pipeline, documented latency |
