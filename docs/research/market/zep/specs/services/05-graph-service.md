# 05 — Graph Service (zep-graph)

> **gRPC**: 9044 | **Health**: 9144  
> **Origin**: L4 — Graph Intelligence Layer (Graphiti Client + Ontology)

---

## 1. Purpose

Temporal Knowledge Graph management — extraction, storage, and retrieval of relationship-aware facts. Cung cấp:
- **Async Entity Extraction**: Consume messages từ NATS → LLM-powered extraction → Neo4j upsert
- **Graphiti Integration**: HTTP client wrapping Graphiti service endpoints
- **Ontology Management**: 9-type node hierarchy + edge type mapping
- **Fact CRUD**: Temporal facts with `valid_at`/`invalid_at`/`expired_at`
- **Graph Data Operations**: Add nodes, manage episodes, delete groups

---

## 2. Clean Architecture Layout

```
services/zep-graph/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── node.go                # EntityNode (User, Preference, Organization...)
│   │   ├── edge.go                # EntityEdge (Fact with temporal annotations)
│   │   ├── episode.go             # Episode (temporal event/message)
│   │   ├── fact.go                # Fact entity (name, fact string, temporal)
│   │   ├── ontology.go            # NodeOntology, EdgeOntology, priority hierarchy
│   │   ├── group.go               # GroupID, prefix strategy
│   │   ├── temporal.go            # TemporalAnnotation (valid_at, invalid_at, expired_at)
│   │   ├── event.go               # ExtractionCompleted, FactCreated, FactInvalidated
│   │   └── errors.go
│   │
│   ├── usecase/
│   │   ├── extract_entities.go    # NATS consumer: messages → LLM extraction → Neo4j
│   │   ├── put_memory.go          # Forward to Graphiti PutMemory
│   │   ├── get_fact.go            # Get fact by UUID
│   │   ├── delete_fact.go         # Delete (invalidate) fact
│   │   ├── add_node.go            # Add entity node to graph
│   │   ├── add_graph_data.go      # Add text/JSON to user/graph
│   │   ├── set_ontology.go        # Set custom ontology
│   │   ├── get_user_nodes.go      # MCP: get nodes for user
│   │   ├── get_user_edges.go      # MCP: get edges for user
│   │   ├── get_episodes.go        # MCP: get episodes
│   │   ├── get_node.go            # MCP: get node by UUID
│   │   ├── get_edge.go            # MCP: get edge by UUID
│   │   ├── get_episode.go         # MCP: get episode by UUID
│   │   ├── get_node_edges.go      # MCP: get all edges for a node
│   │   ├── get_episode_mentions.go # MCP: get nodes/edges in episode
│   │   ├── delete_group.go        # Delete all data for a group
│   │   ├── port/
│   │   │   ├── input.go
│   │   │   └── output.go          # GraphitiClient, GraphRepository, EventPublisher
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   │
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── neo4j/
│   │   │       ├── node_repo.go   # Neo4j node operations
│   │   │       ├── edge_repo.go   # Neo4j edge operations
│   │   │       └── episode_repo.go # Neo4j episode operations
│   │   ├── client/
│   │   │   └── graphiti_client.go  # HTTP client → Graphiti service
│   │   └── event/
│   │       ├── publisher.go       # NATS publisher
│   │       └── subscriber.go     # NATS consumer (messages.ingested)
│   │
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Layer

### 3.1 Node Ontology (Priority Hierarchy)

```go
package domain

// NodeType represents the ontology classification for extracted entities
type NodeType string

const (
    NodeTypeUser         NodeType = "User"         // Priority 1 (Highest) — singleton
    NodeTypeAssistant    NodeType = "Assistant"     // Priority 1 (Highest) — singleton
    NodeTypePreference   NodeType = "Preference"    // Priority 2 — LOW threshold
    NodeTypeOrganization NodeType = "Organization"  // Priority 3
    NodeTypeEvent        NodeType = "Event"         // Priority 3
    NodeTypeLocation     NodeType = "Location"      // Priority 4
    NodeTypeDocument     NodeType = "Document"      // Priority 4
    NodeTypeTopic        NodeType = "Topic"         // Priority 5
    NodeTypeObject       NodeType = "Object"        // Priority 6 (Lowest) — last resort
)

// NodePriority returns extraction classification priority
var NodePriority = map[NodeType]int{
    NodeTypeUser:         1,
    NodeTypeAssistant:    1,
    NodeTypePreference:   2,
    NodeTypeOrganization: 3,
    NodeTypeEvent:        3,
    NodeTypeLocation:     4,
    NodeTypeDocument:     4,
    NodeTypeTopic:        5,
    NodeTypeObject:       6,
}
```

### 3.2 Edge Ontology

```go
type EdgeType string

const (
    EdgeTypeLocatedAt  EdgeType = "LOCATED_AT"   // Entity → Location
    EdgeTypeOccurredAt EdgeType = "OCCURRED_AT"  // Event → Entity/Location
)

// EdgeTypeMap defines valid edge types between node type pairs
var EdgeTypeMap = map[[2]NodeType][]EdgeType{
    {NodeTypeEvent, NodeTypeOrganization}: {EdgeTypeOccurredAt},
    {NodeTypeEvent, NodeTypeLocation}:     {EdgeTypeOccurredAt},
    {NodeTypeOrganization, NodeTypeLocation}: {EdgeTypeLocatedAt},
    {NodeTypeUser, NodeTypeLocation}:      {EdgeTypeLocatedAt},
}
```

### 3.3 Temporal Annotations

```go
type TemporalAnnotation struct {
    ValidAt   *time.Time  // when the fact became true
    InvalidAt *time.Time  // when the fact ceased to be true
    ExpiredAt *time.Time  // when the fact was superseded by a newer fact
}

// Example:
// "Alice worked at Acme" → ValidAt: 2020-01-01, InvalidAt: 2023-06-30
// "Alice works at Beta"  → ValidAt: 2023-07-01, InvalidAt: nil (current)
```

### 3.4 Entity Definitions

```go
type EntityNode struct {
    UUID     string
    Name     string
    NodeType NodeType
    GroupID  string            // multi-tenant partition
    Summary  string            // LLM-generated summary
    Labels   []string
    Properties map[string]any
    CreatedAt  time.Time
}

type EntityEdge struct {
    UUID       string
    Name       string           // relationship label
    Fact       string           // human-readable fact statement
    SourceID   string           // source node UUID
    TargetID   string           // target node UUID
    EdgeType   EdgeType
    GroupID    string
    Temporal   TemporalAnnotation
    CreatedAt  time.Time
}

type Episode struct {
    UUID      string
    Name      string
    Content   string           // original message content
    GroupID   string
    SourceID  string           // message UUID (potentially prefixed)
    CreatedAt time.Time
}
```

### 3.5 Group ID Strategy

```go
type GroupID string

// WithPrefix generates prefixed episode UUID: {groupID}-{messageUUID}
// Used to namespace episodes across different groups (session/user)
func (g GroupID) WithPrefix(messageUUID string) string {
    return string(g) + "-" + messageUUID
}
```

---

## 4. Use Case Layer

### 4.1 Port Interfaces

```go
package port

type GraphService interface {
    // Graphiti forwarding
    PutMemory(ctx context.Context, req dto.PutMemoryRequest) error
    GetFact(ctx context.Context, factUUID string) (*dto.FactResponse, error)
    DeleteFact(ctx context.Context, factUUID string) error
    AddNode(ctx context.Context, req dto.AddNodeRequest) error
    AddGraphData(ctx context.Context, req dto.AddGraphDataRequest) error
    SetOntology(ctx context.Context, req dto.SetOntologyRequest) error
    DeleteGroup(ctx context.Context, groupID string) error
    
    // MCP-compatible graph queries
    GetUserNodes(ctx context.Context, userID string) ([]dto.NodeResponse, error)
    GetUserEdges(ctx context.Context, userID string) ([]dto.EdgeResponse, error)
    GetEpisodes(ctx context.Context, groupID string) ([]dto.EpisodeResponse, error)
    GetNode(ctx context.Context, nodeUUID string) (*dto.NodeResponse, error)
    GetEdge(ctx context.Context, edgeUUID string) (*dto.EdgeResponse, error)
    GetEpisode(ctx context.Context, episodeUUID string) (*dto.EpisodeResponse, error)
    GetNodeEdges(ctx context.Context, nodeUUID string) ([]dto.EdgeResponse, error)
    GetEpisodeMentions(ctx context.Context, episodeUUID string) (*dto.EpisodeMentionsResponse, error)
}

type GraphitiClient interface {
    PutMemory(ctx context.Context, groupID string, messages []dto.MessageInput, addPrefix bool) error
    GetMemory(ctx context.Context, groupID string, maxFacts int, queryMessages []string) ([]domain.EntityEdge, error)
    Search(ctx context.Context, req dto.GraphitiSearchRequest) ([]domain.EntityEdge, error)
    AddNode(ctx context.Context, node domain.EntityNode) error
    GetFact(ctx context.Context, uuid string) (*domain.EntityEdge, error)
    DeleteFact(ctx context.Context, uuid string) error
    DeleteGroup(ctx context.Context, groupID string) error
    DeleteEpisode(ctx context.Context, uuid string) error
}

type GraphRepository interface {
    GetNodesByGroupID(ctx context.Context, groupID string) ([]domain.EntityNode, error)
    GetEdgesByGroupID(ctx context.Context, groupID string) ([]domain.EntityEdge, error)
    GetEpisodesByGroupID(ctx context.Context, groupID string) ([]domain.Episode, error)
    GetNodeByUUID(ctx context.Context, uuid string) (*domain.EntityNode, error)
    GetEdgeByUUID(ctx context.Context, uuid string) (*domain.EntityEdge, error)
    GetEpisodeByUUID(ctx context.Context, uuid string) (*domain.Episode, error)
    GetEdgesForNode(ctx context.Context, nodeUUID string) ([]domain.EntityEdge, error)
    GetMentions(ctx context.Context, episodeUUID string) ([]domain.EntityNode, []domain.EntityEdge, error)
}
```

### 4.2 Extract Entities (NATS Consumer — Async 10-20s)

```go
func (uc *ExtractEntitiesUseCase) HandleMessagesIngested(ctx context.Context, event domain.MessagesIngested) error {
    // 1. Forward to Graphiti: PutMemory(sessionID, messages, addPrefix=true)
    err := uc.graphitiClient.PutMemory(ctx, event.SessionID, event.Messages, event.AddPrefix)
    if err != nil {
        return fmt.Errorf("graphiti PutMemory (session): %w", err)
    }
    
    // 2. If user linked, also extract to user's graph
    if event.UserID != nil {
        err := uc.graphitiClient.PutMemory(ctx, *event.UserID, event.Messages, event.AddPrefix)
        if err != nil {
            return fmt.Errorf("graphiti PutMemory (user): %w", err)
        }
    }
    
    // 3. Publish completion event
    uc.publisher.PublishExtractionCompleted(ctx, domain.ExtractionCompleted{
        SessionID:   event.SessionID,
        ProjectUUID: event.ProjectUUID,
        Timestamp:   time.Now(),
    })
    
    return nil
}
```

---

## 5. gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.graph.v1;

service GraphService {
  // Fact operations
  rpc GetFact(GetFactRequest) returns (FactResponse);
  rpc DeleteFact(DeleteFactRequest) returns (google.protobuf.Empty);
  
  // Graph data operations
  rpc AddGraphData(AddGraphDataRequest) returns (google.protobuf.Empty);
  rpc SetOntology(SetOntologyRequest) returns (google.protobuf.Empty);
  rpc DeleteGroup(DeleteGroupRequest) returns (google.protobuf.Empty);
  
  // MCP-compatible graph queries
  rpc GetUserNodes(GetUserNodesRequest) returns (NodeListResponse);
  rpc GetUserEdges(GetUserEdgesRequest) returns (EdgeListResponse);
  rpc GetEpisodes(GetEpisodesRequest) returns (EpisodeListResponse);
  rpc GetNode(GetNodeRequest) returns (NodeResponse);
  rpc GetEdge(GetEdgeRequest) returns (EdgeResponse);
  rpc GetEpisode(GetEpisodeRequest) returns (EpisodeResponse);
  rpc GetNodeEdges(GetNodeEdgesRequest) returns (EdgeListResponse);
  rpc GetEpisodeMentions(GetEpisodeMentionsRequest) returns (EpisodeMentionsResponse);
}

message NodeResponse {
  string uuid = 1;
  string name = 2;
  string node_type = 3;
  string group_id = 4;
  string summary = 5;
  repeated string labels = 6;
  google.protobuf.Struct properties = 7;
  google.protobuf.Timestamp created_at = 8;
}

message EdgeResponse {
  string uuid = 1;
  string name = 2;
  string fact = 3;
  string source_id = 4;
  string target_id = 5;
  string edge_type = 6;
  string group_id = 7;
  optional google.protobuf.Timestamp valid_at = 8;
  optional google.protobuf.Timestamp invalid_at = 9;
  optional google.protobuf.Timestamp expired_at = 10;
  google.protobuf.Timestamp created_at = 11;
}

message EpisodeResponse {
  string uuid = 1;
  string name = 2;
  string content = 3;
  string group_id = 4;
  string source_id = 5;
  google.protobuf.Timestamp created_at = 6;
}

message EpisodeMentionsResponse {
  repeated NodeResponse nodes = 1;
  repeated EdgeResponse edges = 2;
}

message AddGraphDataRequest {
  string user_id = 1;       // user scope
  string graph_id = 2;      // graph scope (alternative)
  string data = 3;          // text or JSON content
  string type = 4;          // "text" | "json"
}

message SetOntologyRequest {
  string graph_id = 1;
  google.protobuf.Struct entities = 2;
  google.protobuf.Struct edges = 3;
}
```

---

## 6. Graphiti HTTP Client

```go
type GraphitiHTTPClient struct {
    baseURL    string  // e.g., "http://graphiti:8003"
    httpClient *http.Client
}

// Endpoint mapping
var graphitiEndpoints = map[string]string{
    "PutMemory":     "POST /messages",
    "GetMemory":     "POST /get-memory",
    "Search":        "POST /search",
    "AddNode":       "POST /entity-node",
    "GetFact":       "GET  /entity-edge/{uuid}",
    "DeleteFact":    "DELETE /entity-edge/{uuid}",
    "DeleteGroup":   "DELETE /group/{id}",
    "DeleteEpisode": "DELETE /episode/{uuid}",
}
```

---

## 7. NATS Events

### Consumed

| Subject | Source | Action |
|---------|--------|--------|
| `zep.memory.messages.ingested` | zep-memory | Extract entities via Graphiti |
| `zep.user.deleted` | zep-user | Delete user's graph data |

### Published

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.graph.extraction.completed` | `{session_id, project_uuid}` | zep-search (reindex cache) |
| `zep.graph.fact.created` | `{fact_uuid, group_id, name, fact}` | zep-search (cache update) |
| `zep.graph.fact.invalidated` | `{fact_uuid, invalid_at}` | zep-search (cache invalidation) |

---

## 8. Configuration

```yaml
graph:
  grpc:
    port: 9044
  health:
    port: 9144
  graphiti:
    service_url: "http://graphiti:8003"
    timeout: 60s                    # longer timeout for LLM extraction
    max_retries: 3
  neo4j:
    uri: "bolt://neo4j:7687"
    username: "neo4j"
    password: "zepzepzep"
    max_connection_pool_size: 50
  nats:
    url: "nats://nats:4222"
    stream: "zep"
    consumer_group: "zep-graph"
    max_deliver: 3                 # retry failed extractions
    ack_wait: 120s                 # long ack wait for LLM processing
  telemetry:
    service_name: "zep-graph"
    otel_endpoint: "otel-collector:4317"
```
