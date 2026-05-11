# Zep Graph API

## Overview
The Zep Graph Service handles Knowledge Graph (KG) extraction and maintenance using the Graphiti framework logic. It processes memories asynchronously via LLM pipelines and stores temporal facts in Neo4j and PostgreSQL.

## gRPC Services (Port 9064)

### ZepGraphService

#### Fact Management
```protobuf
message Fact {
  string uuid = 1;
  string content = 2;
  string source_node = 3;
  string target_node = 4;
  string relation = 5;
  int64 valid_at = 6;
  int64 invalid_at = 7;
}

rpc AddFact(AddFactRequest) returns (Fact);
rpc DeleteFact(DeleteFactRequest) returns (Empty);
```

#### Graph & Ontology
```protobuf
rpc GetEpisodes(GetEpisodesRequest) returns (EpisodesResponse);
rpc SetOntology(SetOntologyRequest) returns (Empty);
rpc GetOntology(GetOntologyRequest) returns (Ontology);
```

## Async Extraction Pipeline (NATS)

The primary entry point for `zep-graph` is event-driven:
1. Listens for `zep.memory.messages.ingested`.
2. Runs Graphiti's Entity and Relation Extraction using an LLM.
3. Computes Temporal Annotations (`valid_at`, `invalid_at`).
4. Upserts Nodes and Edges into Neo4j 5.x.
5. Emits `zep.graph.extraction.completed` when processing finishes.
