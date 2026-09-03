# Zep Search API

## Overview
The Zep Search Service provides high-performance semantic and hybrid search capabilities across both message histories (PostgreSQL pgvector) and the Knowledge Graph (Neo4j).

## gRPC Services (Port 9065)

### ZepSearchService

#### Search Capabilities
```protobuf
message SearchRequest {
  string query = 1;
  string project_uuid = 2;
  string user_uuid = 3;
  string thread_uuid = 4;
  int32 limit = 5;
  string mode = 6; // 'graph', 'session', 'hybrid'
}

message SearchResult {
  string id = 1;
  string content = 2;
  float score = 3;
  map<string, string> metadata = 4;
}

rpc GraphSearch(GraphSearchRequest) returns (SearchResponse);
rpc SessionSearch(SessionSearchRequest) returns (SearchResponse);
```

## Search Modes & Strategies

1. **Session Search (Vector)**
   - Queries `pgvector` index in PostgreSQL.
   - Used for fetching direct contextual quotes from recent chat history.

2. **Graph Search (Traversal + Vector)**
   - Queries Neo4j for entities matching the user's intent.
   - Extracts 1-hop and 2-hop subgraphs containing relevant facts.

3. **Hybrid Search**
   - Fuses results using Reciprocal Rank Fusion (RRF) and Maximal Marginal Relevance (MMR).
   - Applies Temporal Decay to downweight older facts.
