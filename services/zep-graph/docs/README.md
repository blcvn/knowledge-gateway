---
id: DOC-S01
service: zep-graph
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Zep Team
---

# zep-graph

> **Group**: Zep (Context Engineering) | **gRPC Port**: 9064 | **Health Port**: 12064 | **Origin**: Zep

## Purpose

Temporal Knowledge Graph management — extraction, storage, and retrieval of relationship-aware facts. Cung cấp async entity extraction via LLM-powered Graphiti pipeline, ontology management (9-type node hierarchy), fact CRUD with temporal annotations, và graph data operations.

### Business Capability

- **Async Entity Extraction**: Consume messages từ NATS → Graphiti LLM extraction → Neo4j upsert (10-20s)
- **Graphiti Integration**: HTTP client wrapping Graphiti service endpoints
- **Ontology Management**: 9-type node hierarchy (User, Assistant, Preference, Organization, Event, Location, Document, Topic, Object) + edge type mapping
- **Fact CRUD**: Temporal facts with `valid_at`/`invalid_at`/`expired_at`
- **MCP-Compatible Queries**: Graph exploration APIs (nodes, edges, episodes, mentions)
- **Group Data Operations**: Add nodes, manage episodes, delete groups

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream consumer/publisher
- **Database**: Neo4j (graph storage), PostgreSQL (metadata)
- **External**: Graphiti HTTP service (LLM entity extraction)
- **Architecture**: 4-layer Clean Architecture

## Quick Start

```bash
make build-zep-graph
make run-zep-graph
docker compose up zep-graph
```

## API Surface

### gRPC Service

```protobuf
service GraphService {
  rpc GetFact(GetFactRequest) returns (FactResponse);
  rpc DeleteFact(DeleteFactRequest) returns (google.protobuf.Empty);
  rpc AddGraphData(AddGraphDataRequest) returns (google.protobuf.Empty);
  rpc SetOntology(SetOntologyRequest) returns (google.protobuf.Empty);
  rpc DeleteGroup(DeleteGroupRequest) returns (google.protobuf.Empty);
  rpc GetUserNodes(GetUserNodesRequest) returns (NodeListResponse);
  rpc GetUserEdges(GetUserEdgesRequest) returns (EdgeListResponse);
  rpc GetEpisodes(GetEpisodesRequest) returns (EpisodeListResponse);
  rpc GetNode(GetNodeRequest) returns (NodeResponse);
  rpc GetEdge(GetEdgeRequest) returns (EdgeResponse);
  rpc GetEpisode(GetEpisodeRequest) returns (EpisodeResponse);
  rpc GetNodeEdges(GetNodeEdgesRequest) returns (EdgeListResponse);
  rpc GetEpisodeMentions(GetEpisodeMentionsRequest) returns (EpisodeMentionsResponse);
}
```

## NATS Events

### Consumed

| Subject | Source | Action |
|---------|--------|--------|
| `zep.memory.messages.ingested` | zep-memory | Extract entities via Graphiti (10-20s) |
| `zep.user.deleted` | zep-user | Delete user's graph data |

### Published

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.graph.extraction.completed` | `{session_id, project_uuid}` | zep-search (reindex cache) |
| `zep.graph.fact.created` | `{fact_uuid, group_id, name, fact}` | zep-search (cache update) |
| `zep.graph.fact.invalidated` | `{fact_uuid, invalid_at}` | zep-search (cache invalidation) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| Graphiti | HTTP | LLM entity extraction, graph operations |
| Neo4j | Bolt | Graph storage (nodes, edges, episodes) |
| NATS JetStream | Sub/Pub | Consume messages.ingested, publish extraction events |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Reference Design](../../../references/zep/specs/services/05-graph-service.md)

## Owner

- **Team**: VNP Memory — Zep
