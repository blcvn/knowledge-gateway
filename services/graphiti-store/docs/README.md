---
id: DOC-S01
service: graphiti-store
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Graphiti Team
---

# graphiti-store

> **Group**: Graphiti (Episodic KG) | **gRPC Port**: 9024 | **Health Port**: 9097 | **Origin**: Graphiti

## Purpose

Graph database abstraction layer providing pluggable backends. All graph CRUD operations, transactions, index management, and search primitives (cosine similarity, full-text BM25, BFS traversal) are implemented here behind the `GraphDriver` interface.

### Business Capability

- **Multi-Backend Support**: Pluggable graph database drivers (Neo4j, FalkorDB, Kuzu, Neptune)
- **Graph CRUD**: Save/Get/Delete for EntityNode, EpisodicNode, CommunityNode, SagaNode
- **Edge CRUD**: Save/Get/Delete for EntityEdge, EpisodicEdge, CommunityEdge, HasEpisodeEdge, NextEpisodeEdge
- **Search Primitives**: CosineSimilaritySearch, FulltextSearch (BM25), BFSSearch (graph traversal)
- **Bulk Operations**: SaveBulk for efficient batch graph writes with transaction support
- **Index Management**: Create/drop indices and constraints per backend

## Tech Stack

- **Language**: Go 1.23+
- **Primary Backend**: Neo4j 5.x
- **Pluggable Backends**: FalkorDB, Kuzu, Neptune
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## GraphDriver Interface

```go
type GraphDriver interface {
    SaveNode(ctx context.Context, node *graph.EntityNode) error
    SaveEdge(ctx context.Context, edge *graph.EntityEdge) error
    GetNode(ctx context.Context, id string, groupID string) (*graph.EntityNode, error)
    DeleteNode(ctx context.Context, id string) error
    CosineSimilarity(ctx context.Context, embedding []float64, groupID string, limit int) ([]*graph.EntityNode, error)
    FulltextSearch(ctx context.Context, query string, groupID string, limit int) ([]*graph.EntityNode, error)
    BFSTraversal(ctx context.Context, startNodeID string, depth int) ([]*graph.EntityNode, error)
    BuildIndicesAndConstraints(deleteExisting bool) error
    Close() error
}
```

## Backend Implementations

| Backend | Status | Isolation Mechanism | Index Support |
|---------|--------|-------------------|---------------|
| Neo4j 5.x | Primary | Property filter `group_id` | Vector + Fulltext + B-tree |
| FalkorDB | Pluggable | Separate graph per `group_id` | RediSearch |
| Kuzu | Pluggable | Separate database per `group_id` | Built-in |
| Neptune | Pluggable | Property filter `group_id` | OpenSearch |

## Quick Start

```bash
make build-graphiti-store
make run-graphiti-store
docker compose up graphiti-store neo4j
```

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — Graphiti
