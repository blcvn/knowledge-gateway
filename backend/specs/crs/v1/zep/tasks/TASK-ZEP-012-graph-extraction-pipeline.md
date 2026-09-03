# TASK-ZEP-012 — services/zep-graph: Entity Extraction Pipeline (NATS → Graphiti → Neo4j)

**Task ID:** TASK-ZEP-012  
**Wave:** 4 (Graph Intelligence)  
**Solution:** [SOL-ZEP-003](../solutions/SOL-ZEP-003-Temporal-Knowledge-Graph.md)  
**Depends on:** TASK-ZEP-011 (graph domain + Neo4j repo), TASK-ZEP-010 (Graphiti client)  
**Ước tính:** 4h  
**Priority:** Critical — core Temporal KG pipeline

**Trạng thái:** ✅ Implemented  
**Ghi chú:** zep-graph: 6 .go - graph extraction pipeline  
---

## Mục tiêu

Implement async entity extraction pipeline:
- NATS consumer: lắng nghe `zep.memory.messages.ingested`
- Gọi Graphiti để extract entities (10-20s, không sync với PutMemory)
- Upsert kết quả (nodes + temporal edges + episodes) vào Neo4j
- Publish `zep.graph.extraction.completed` → Search Service invalidate cache
- Implement `AddGraphData` (non-message data) + `SetOntology`

---

## Công việc cụ thể

### 1. Tạo NATS Event Subscriber

**`services/zep-graph/internal/adapter/subscriber/message_events.go`**

```go
// MessageEventSubscriber lắng nghe "zep.memory.messages.ingested"
// và trigger async entity extraction
type MessageEventSubscriber struct {
    nats      NATSClient
    extractor *ExtractEntitiesUseCase
}

// Start bắt đầu consume NATS events
// NATS JetStream config: MaxDeliver=3 (retry 3 lần nếu fail)
func (s *MessageEventSubscriber) Start(ctx context.Context) error {
    return s.nats.Subscribe(ctx, "zep.memory.messages.ingested", func(msg []byte) error {
        var event MessagesIngestedEvent
        if err := json.Unmarshal(msg, &event); err != nil { return err }
        return s.extractor.Execute(ctx, event)
    })
}
```

### 2. Tạo `ExtractEntities` Use Case

**`services/zep-graph/internal/usecase/extract_entities.go`**

```go
// ExtractEntitiesUseCase: NATS event → Graphiti → Neo4j → publish completion
type ExtractEntitiesUseCase struct {
    graphitiClient *graphiti.GraphitiClient
    nodeRepo       NodeRepository
    edgeRepo       EdgeRepository
    episodeRepo    EpisodeRepository
    publisher      EventPublisher
    ontologyRepo   OntologyRepository
}

// Execute được gọi async (sau khi PutMemory đã trả về 200)
// Expected latency: 10-20 seconds (OK — async)
func (uc *ExtractEntitiesUseCase) Execute(ctx context.Context, event MessagesIngestedEvent) error {
    // 1. Load custom ontology (nếu có)
    ontology, _ := uc.ontologyRepo.GetForGroup(ctx, event.SessionID)

    // 2. Call Graphiti cho session graph
    sessionResp, err := uc.graphitiClient.PutMemory(ctx, graphiti.PutMemoryRequest{
        GroupID:  event.SessionID,
        Messages: convertMessages(event.Messages),
        Ontology: ontology,
    })
    if err != nil { return err }  // NATS JetStream sẽ retry (MaxDeliver=3)

    // 3. Call Graphiti cho user graph (nếu user linked)
    if event.UserID != nil {
        userResp, err := uc.graphitiClient.PutMemory(ctx, graphiti.PutMemoryRequest{
            GroupID:  *event.UserID,
            Messages: convertMessages(event.Messages),
            Ontology: ontology,
        })
        if err == nil {
            uc.upsertGraphData(ctx, *event.UserID, userResp)
        }
    }

    // 4. Upsert vào Neo4j
    uc.upsertGraphData(ctx, event.SessionID, sessionResp)

    // 5. Publish completion event → Search Service invalidate cache
    uc.publisher.Publish(ctx, "zep.graph.extraction.completed", GraphExtractionCompletedEvent{
        GroupID:   event.SessionID,
        EdgeCount: len(sessionResp.Edges),
        NodeCount: len(sessionResp.Nodes),
    })
    return nil
}

// upsertGraphData writes extracted data to Neo4j
func (uc *ExtractEntitiesUseCase) upsertGraphData(ctx context.Context, groupID string, resp *graphiti.GraphitiResponse) {
    for _, n := range resp.Nodes {
        uc.nodeRepo.Upsert(ctx, &EntityNode{
            UUID:     n.UUID,
            GroupID:  groupID,
            Name:     n.Name,
            NodeType: classifyNodeType(n.Labels),  // map Graphiti labels → NodeType enum
            Summary:  n.Summary,
        })
    }
    for _, e := range resp.Edges {
        uc.edgeRepo.Upsert(ctx, &TemporalEdge{
            UUID:       e.UUID,
            GroupID:    groupID,
            Name:       e.Name,
            Fact:       e.Fact,
            FactRating: e.FactRating,
            ValidAt:    e.ValidAt,   // from Graphiti LLM extraction
            InvalidAt:  e.InvalidAt,
        })
    }
    for _, ep := range resp.Episodes {
        uc.episodeRepo.Upsert(ctx, &Episode{
            UUID:    ep.UUID,
            GroupID: groupID,
            Name:    fmt.Sprintf("%s-%s", groupID, ep.UUID),  // avoid collision
            Content: ep.Content,
            Source:  "message",
        })
    }
}
```

### 3. Tạo `AddGraphData` Use Case (non-message ingestion)

**`services/zep-graph/internal/usecase/add_graph_data.go`**

```go
// AddGraphDataRequest — for CRM data, product catalogs, telemetry
type AddGraphDataRequest struct {
    UserID  string `json:"user_id"`   // target user's graph
    GraphID string `json:"graph_id"`  // or specific graph scope
    Data    string `json:"data"`      // raw text or JSON
    Type    string `json:"type"`      // "text" | "json"
}

// AddGraphDataUseCase: POST /api/v2/graph/data
// Treat non-message data as synthetic message for Graphiti extraction
func (uc *AddGraphDataUseCase) Execute(ctx context.Context, req AddGraphDataRequest) error {
    groupID := req.GraphID
    if req.UserID != "" { groupID = req.UserID }
    
    // Synthetic message với role="system" cho non-message data
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

### 4. Tạo `SetOntology`, `GetFact`, `InvalidateFact` Use Cases

```go
// SetOntology: POST /api/v2/graph/ontology
func (uc *SetOntologyUseCase) Execute(ctx context.Context, ontology *GraphOntology) error {
    return uc.ontologyRepo.Set(ctx, ontology)
}

// GetFact: GET /api/v2/facts/:uuid
func (uc *GetFactUseCase) Execute(ctx context.Context, uuid string) (*TemporalEdge, error) {
    return uc.edgeRepo.GetByUUID(ctx, uuid)
}

// InvalidateFact: DELETE /api/v2/facts/:uuid (soft invalidate)
func (uc *InvalidateFactUseCase) Execute(ctx context.Context, uuid string) error {
    if err := uc.edgeRepo.Invalidate(ctx, uuid); err != nil { return err }
    // Publish event → Search Service invalidate cache
    uc.publisher.Publish(ctx, "zep.graph.fact.invalidated", FactInvalidatedEvent{FactUUID: uuid})
    return nil
}
```

### 5. Tests

- `TestExtractEntities_UpsertNodes`: after execution → nodes in Neo4j
- `TestExtractEntities_TemporalEdges`: valid_at/invalid_at preserved from Graphiti
- `TestExtractEntities_UserAndSession`: both user and session graphs updated
- `TestAddGraphData_SyntheticMessage`: text data → synthetic "system" message to Graphiti
- `TestInvalidateFact_SetsInvalidAt`: → edge.invalid_at is set in Neo4j
- `TestSetOntology_PersistsAndRetrieves`: set + get → same definition

---

## Acceptance Criteria

- [ ] `go build ./services/zep-graph/...` không có lỗi
- [ ] NATS consumer nhận event `zep.memory.messages.ingested` → trigger extraction
- [ ] TemporalEdge trong Neo4j có `valid_at` từ Graphiti (không null nếu LLM detects)
- [ ] InvalidateFact → Neo4j edge có `invalid_at` set
- [ ] AddGraphData với type="json" → synthetic "system" message
- [ ] After extraction → NATS event `zep.graph.extraction.completed` published
- [ ] `go test ./services/zep-graph/...` pass

---

## Files tạo ra

```
services/zep-graph/
├── internal/
│   ├── usecase/
│   │   ├── extract_entities.go
│   │   ├── extract_entities_test.go
│   │   ├── add_graph_data.go
│   │   ├── set_ontology.go
│   │   ├── get_fact.go
│   │   └── invalidate_fact.go
│   └── adapter/
│       └── subscriber/
│           └── message_events.go
```

## Sau khi hoàn thành

Chạy: `go build ./services/zep-graph/... && go test ./services/zep-graph/...`
