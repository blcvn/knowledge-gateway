---
id: TDD-zep-graph
title: Technical Design — zep-graph
service: zep-graph
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Zep
---

# Technical Design — zep-graph

> **Group**: Zep | **gRPC Port**: 9064 | **Health Port**: 12064

## 1. Service Overview

Temporal Knowledge Graph management. Async entity extraction via Graphiti LLM pipeline. Neo4j storage for nodes, edges (facts), episodes. 9-type node ontology with temporal annotations.

## 2. Domain Model

- **EntityNode**: UUID, Name, NodeType (9 types), GroupID, Summary, Labels[], Properties{}, CreatedAt
- **EntityEdge**: UUID, Name, Fact, SourceID, TargetID, EdgeType, GroupID, Temporal{ValidAt, InvalidAt, ExpiredAt}
- **Episode**: UUID, Name, Content, GroupID, SourceID (prefixed UUID)
- **NodeOntology**: Priority 1-6 hierarchy (User → Object)
- **EdgeOntology**: LOCATED_AT, OCCURRED_AT relationship types

## 3. Critical Data Flows

### Async Entity Extraction (NATS Consumer)
1. Consume `zep.memory.messages.ingested` from NATS
2. Forward to Graphiti: PutMemory(sessionID, messages, addPrefix=true)
3. If user linked: also PutMemory(userID, messages, addPrefix=true)
4. Publish `zep.graph.extraction.completed`

### Graphiti HTTP Integration
- PutMemory → POST /messages
- GetMemory → POST /get-memory
- Search → POST /search
- CRUD → GET/DELETE /entity-edge/{uuid}

## 4. NATS Events

### Consumed
| Subject | Action |
|---------|--------|
| `zep.memory.messages.ingested` | Extract entities via Graphiti |
| `zep.user.deleted` | Delete user graph data |

### Published
| Subject | Subscribers |
|---------|-------------|
| `zep.graph.extraction.completed` | zep-search |
| `zep.graph.fact.created` | zep-search |
| `zep.graph.fact.invalidated` | zep-search |

## 5. Storage

Neo4j: EntityNodes, EntityEdges (with temporal annotations), Episodes. Graph queries via Cypher. Max pool: 50 connections.

## 6. Multi-Tenancy

GroupID-based isolation. GroupID = user_id | session_id.

---

> **Next Steps**: Decompose into FEAT specs in `specs/features/`.
