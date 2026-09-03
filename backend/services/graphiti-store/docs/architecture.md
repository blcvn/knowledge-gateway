---
id: DOC-S03
service: graphiti-store
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-store — Service Architecture

> **Group**: Graphiti | **Pattern**: 4-layer Clean Architecture + Strategy Pattern for DB Backends

## Layer Structure

```
services/graphiti-store/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go              # EntityNode, EpisodicNode, CommunityNode, SagaNode
│   │   ├── edge.go                # EntityEdge, EpisodicEdge, CommunityEdge, etc.
│   │   ├── value_object.go        # NodeLabel, EdgeType, GraphProvider
│   │   └── errors.go              # ErrNodeNotFound, ErrEdgeNotFound
│   ├── usecase/
│   │   ├── node_crud.go           # Node save/get/delete operations
│   │   ├── edge_crud.go           # Edge save/get/delete operations
│   │   ├── bulk_save.go           # Transactional bulk write
│   │   ├── search.go              # Vector/fulltext/BFS search coordination
│   │   └── port/
│   │       ├── input.go           # NodeUseCase, EdgeUseCase, SearchUseCase
│   │       └── output.go          # GraphDriver interface (strategy pattern)
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go         # gRPC server implementation
│   │   │   └── mapper.go          # Proto ↔ Domain ↔ Graph driver mapping
│   │   └── repository/
│   │       ├── neo4j/             # Neo4j 5.x driver implementation
│   │       │   ├── driver.go      # Connection, session, transaction mgmt
│   │       │   ├── node_ops.go    # Entity/Episodic/Community node queries
│   │       │   ├── edge_ops.go    # Edge Cypher queries
│   │       │   ├── search_ops.go  # Vector index + fulltext + BFS queries
│   │       │   └── index_ops.go   # Index/constraint management
│   │       ├── falkordb/          # FalkorDB driver implementation
│   │       └── kuzu/              # Kuzu driver implementation
│   └── infra/
│       ├── config/config.go       # Viper + graph backend selection
│       ├── server/grpc.go
│       └── wire/wire.go           # Wire providers per backend
```

## Design Decisions

- **Strategy pattern**: GraphDriver interface allows runtime backend selection via config
- **Cypher abstraction**: Provider-specific query builders handle Neo4j vs FalkorDB vs Kuzu syntax differences
- **Transaction support**: Real transactions for Neo4j/Neptune, session-based fallback for FalkorDB/Kuzu
- **Embedding storage**: Neo4j vector index for native vector search; pgvector fallback for non-native backends
- **Multi-tenant isolation**: Neo4j uses property filter (group_id), FalkorDB uses separate graphs, Kuzu uses separate databases

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| Neo4j 5.x | Primary graph database backend |
| FalkorDB | Alternative lightweight graph backend |
| Kuzu | Alternative embedded graph backend |
