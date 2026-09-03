---
id: FEAT-COG-002
title: Cognify Service — Adapter Layer (gRPC + NATS + Neo4j + Qdrant + Bifrost)
service: cognee-cognify
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_feat: FEAT-COG-001
---

## Mục Tiêu

Implement Layer 3 (Adapter) cho cognee-cognify — gRPC handlers, NATS subscriber/publisher, Neo4j graph repository, Qdrant vector repository, Bifrost LLM client, PostgreSQL job state repository.

## Scope

### In Scope
- gRPC handler: `CogneeCognifyServiceServer` (TriggerCognify, GetJobStatus, CancelJob)
- NATS subscriber: `cognee.data.ingested` → trigger cognify pipeline
- NATS publisher: `cognee.pipeline.completed` event
- Neo4j repository: GraphRepository impl (create nodes, edges, communities)
- Qdrant repository: VectorRepository impl (upsert embeddings, search)
- PostgreSQL repository: JobRepository impl (job state persistence)
- Bifrost LLM client: LLMClient impl (structured extraction)
- Embedding client: EmbedderClient impl (text-embedding-3-large)
- Proto ↔ Domain mapper

### Out of Scope
- Domain/Usecase (FEAT-COG-001)
- Config/Wire/Server (FEAT-COG-003)

## Thiết Kế Kỹ Thuật

### Directory Structure

```
internal/adapter/
├── grpc/
│   ├── handler.go               # CogneeCognifyServiceServer impl
│   └── mapper.go                # Proto ↔ Domain mapping
├── nats/
│   ├── subscriber.go            # cognee.data.ingested listener
│   └── publisher.go             # cognee.pipeline.completed publisher
├── repository/
│   ├── postgres/
│   │   └── job_repo.go          # JobRepository impl (CognifyJob state)
│   ├── neo4j/
│   │   ├── graph_repo.go        # GraphRepository impl
│   │   └── queries.go           # Cypher query templates
│   └── qdrant/
│       └── vector_repo.go       # VectorRepository impl
├── client/
│   ├── llm_client.go            # LLMClient impl → Bifrost
│   ├── embedder_client.go       # EmbedderClient impl → Bifrost
│   └── ingestion_client.go      # gRPC client → cognee-ingestion (fetch data items)
```

### Neo4j Graph Operations (Cypher)

```cypher
-- Create entity node with tenant namespace
MERGE (n:Entity:Tenant_{tenant_id} {name: $name, entity_type: $type})
SET n.description = $description, n.updated_at = datetime()
RETURN n

-- Create relationship edge
MATCH (s:Entity {name: $source}), (t:Entity {name: $target})
WHERE s:Tenant_{tenant_id} AND t:Tenant_{tenant_id}
CREATE (s)-[r:RELATES_TO {relation: $relation, weight: $weight, source_chunk_id: $chunk_id}]->(t)

-- Community detection (GDS)
CALL gds.louvain.stream('entity-graph', {nodeLabels: ['Tenant_{tenant_id}']})
YIELD nodeId, communityId
RETURN gds.util.asNode(nodeId).name AS name, communityId
```

### NATS Subscriber

```go
func (s *DataIngestedSubscriber) Handle(ctx context.Context, msg *nats.Msg) error {
    var event domain.DataIngestedEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        return fmt.Errorf("unmarshal event: %w", err)
    }
    
    // Fetch data items from cognee-ingestion via gRPC
    items, err := s.ingestionClient.GetDataItems(ctx, event.DatasetID)
    
    // Trigger cognify pipeline
    _, err = s.cognifyUseCase.Execute(ctx, dto.TriggerCognifyReq{
        DatasetID: event.DatasetID,
        TenantID:  event.TenantID,
    })
    
    return err
}
```

## Acceptance Criteria

- [ ] AC-1: Given TriggerCognify gRPC call, When handler processes, Then cognify pipeline starts and job is returned
- [ ] AC-2: Given `cognee.data.ingested` NATS event, When subscriber receives, Then cognify pipeline auto-triggers
- [ ] AC-3: Given entities and relationships, When Neo4j repo creates, Then nodes and edges are persisted with tenant namespace
- [ ] AC-4: Given chunks with embeddings, When Qdrant repo upserts, Then vectors are searchable with tenant_id filter
- [ ] AC-5: Given structured extraction prompt, When LLM client calls Bifrost, Then response is parsed into typed entities
- [ ] AC-6: Given job completion, When publisher fires, Then `cognee.pipeline.completed` NATS event is published with metrics

## Test Requirements

- **Unit tests**: Handler with mock usecase, repos with testcontainers (Neo4j, PostgreSQL, Qdrant)
- **Integration tests**: NATS subscriber → usecase → mock repos
- **Coverage**: ≥ 80%
