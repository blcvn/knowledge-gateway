---
id: TASK-GRA-001
title: Domain Models & Core Algorithms
service: zep-graph
status: Done
priority: P0
created: 2026-05-11
---

# Domain Models & Core Algorithms

## Objective
Implement the core domain entities, value objects, and algorithms.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
# Technical Design Document: Zep Graph Service

## 1. System Architecture

`zep-graph` operates mostly asynchronously from the critical user path. It uses LLMs to extract a temporal knowledge graph from ingested messages.

```text
zep-graph/
├── internal/
│   ├── domain/
│   │   ├── fact/        # Nodes, Edges, Episodes
│   │   └── ontology/    # Schema definitions for extraction
│   ├── usecase/
│   │   ├── extractor/   # LLM interaction, Prompt building
│   │   ├── graph/       # Graphiti pipeline logic
│   │   └── ontology/    # Ontology management
│   ├── adapter/
│   │   ├── grpc/        # gRPC Handlers for manual fact overrides
│   │   ├── broker/      # NATS Subscriber & Publisher
│   │   └── llm/         # LLM Client integration
│   └── infra/
│       ├── neo4j/       # Core Graph Database (Nodes/Edges)
│       └── postgres/    # Fact Metadata fallback
```

## 2. Component Design

### 2.1 Domain Layer
- **Node**: Represents entities. Extracted based on strict **Node Priority Hierarchy**:
  1. `User` (singleton)
  2. `Assistant` (singleton)
  3. `Preference` (low threshold for classification)
  4. `Organization`
  5. `Event`
  6. `Location`
  7. `Document`
  8. `Topic`
  9. `Object` (last resort)
- **Edge**: Represents relationships, annotated with temporal bounds (`valid_at`, `invalid_at`). Standard types include `LOCATED_AT` (Entity→Location) and `OCCURRED_AT` (Event→Entity/Location).
- **Episode**: Represents a distinct chunk of grouped interactions temporally.

### 2.2 Usecase Layer
- **Graphiti Extractor**: Receives new messages, clusters them, prompts the LLM to identify updates, additions, or invalidations to existing facts.
- **Group ID Strategy**: Messages are grouped by session ID. When `addGroupIDPrefix=true`, episode UUIDs are prefixed with `{groupID}-{messageUUID}` to namespace episodes across groups.
- **Temporal Resolver**: Ensures facts that contradict each other are correctly timestamped using `invalid_at` instead of hard deletion.

### 2.3 Adapter Layer
- **NATS Subscriber**: Binds to `zep.memory.messages.ingested` queue group.
- **LLM Client**: Standard interface to generic LLM APIs (OpenAI, Anthropic, or Local).

### 2.4 Infrastructure Layer
- **Neo4j 5.x**: Chosen for native temporal and vector search capabilities in graph traversals.
- **Resilience**: Redis rate-limiting/backoff for LLM API limits.

```

## Acceptance Criteria
- [x] Domain models compile and have no external dependencies.
- [x] Core algorithms are fully implemented and unit tested.
