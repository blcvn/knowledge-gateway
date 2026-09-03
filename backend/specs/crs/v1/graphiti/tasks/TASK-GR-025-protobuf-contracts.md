# TASK-GR-025 — Protobuf Contracts (All Services)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-025 |
| **Wave** | 4 (Admin & Observability) |
| **Component** | `api/proto/graphiti/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-001 §3 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-001 |
| **Estimated** | 5h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** shared/proto/graphiti + api/proto/graphiti generated  
---

## Context

Định nghĩa đầy đủ 6 protobuf files cho tất cả graphiti services. Phải được compile trước khi implement handlers. Đây là source of truth cho tất cả inter-service communication.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `api/proto/graphiti/store/v1/store.proto` |
| CREATE | `api/proto/graphiti/knowledge/v1/knowledge.proto` |
| CREATE | `api/proto/graphiti/ingestion/v1/ingestion.proto` |
| CREATE | `api/proto/graphiti/search/v1/search.proto` |
| CREATE | `api/proto/graphiti/admin/v1/admin.proto` |
| MODIFY | `Makefile` |

---

## Implementation

### File 1: `api/proto/graphiti/store/v1/store.proto`

```protobuf
syntax = "proto3";
package graphiti.store.v1;
option go_package = "github.com/vnp-memory/api/proto/graphiti/store/v1";

import "google/protobuf/timestamp.proto";

service StoreService {
    // Bulk operations
    rpc SaveBulk(SaveBulkRequest) returns (SaveBulkResponse);

    // Episode operations
    rpc RetrieveEpisodes(RetrieveEpisodesRequest) returns (RetrieveEpisodesResponse);
    rpc DeleteEpisode(DeleteEpisodeRequest) returns (DeleteEpisodeResponse);
    rpc ListEpisodes(ListEpisodesRequest) returns (ListEpisodesResponse);

    // Entity search
    rpc NodeSimilaritySearch(NodeSimilaritySearchRequest) returns (NodeSimilaritySearchResponse);
    rpc NodeFulltextSearch(NodeFulltextSearchRequest) returns (NodeFulltextSearchResponse);
    rpc EdgeSimilaritySearch(EdgeSimilaritySearchRequest) returns (EdgeSimilaritySearchResponse);
    rpc EdgeFulltextSearch(EdgeFulltextSearchRequest) returns (EdgeFulltextSearchResponse);
    rpc NodeBFSSearch(NodeBFSSearchRequest) returns (NodeBFSSearchResponse);
    rpc EdgeBFSSearch(EdgeBFSSearchRequest) returns (EdgeBFSSearchResponse);
    rpc NodeDistanceReranker(NodeDistanceRerankerRequest) returns (NodeDistanceRerankerResponse);
    rpc EpisodeMentionsReranker(EpisodeMentionsRerankerRequest) returns (EpisodeMentionsRerankerResponse);
    rpc EpisodeSimilaritySearch(EpisodeSimilaritySearchRequest) returns (EpisodeSimilaritySearchResponse);
    rpc EpisodeFulltextSearch(EpisodeFulltextSearchRequest) returns (EpisodeFulltextSearchResponse);
    rpc CommunitySimilaritySearch(CommunitySimilaritySearchRequest) returns (CommunitySimilaritySearchResponse);
    rpc CommunityFulltextSearch(CommunityFulltextSearchRequest) returns (CommunityFulltextSearchResponse);

    // Admin
    rpc GetGroupStats(GetGroupStatsRequest) returns (GetGroupStatsResponse);
    rpc ClearData(ClearDataRequest) returns (ClearDataResponse);
    rpc BuildIndicesAndConstraints(BuildIndicesRequest) returns (BuildIndicesResponse);
    rpc GetCommunityClusters(GetCommunityClustersRequest) returns (GetCommunityClustersResponse);
    rpc RemoveCommunities(RemoveCommunitiesRequest) returns (RemoveCommunitiesResponse);
    rpc SaveCommunityNode(SaveCommunityNodeRequest) returns (SaveCommunityNodeResponse);
    rpc SaveCommunityEdge(SaveCommunityEdgeRequest) returns (SaveCommunityEdgeResponse);

    // Saga
    rpc GetOrCreateSaga(GetOrCreateSagaRequest) returns (GetOrCreateSagaResponse);
    rpc GetLastEpisodeInSaga(GetLastEpisodeInSagaRequest) returns (GetLastEpisodeInSagaResponse);
}

message EntityNode {
    string uuid = 1;
    string name = 2;
    repeated string labels = 3;
    string summary = 4;
    string group_id = 5;
    repeated float name_embedding = 6;
    google.protobuf.Timestamp created_at = 7;
    google.protobuf.Timestamp updated_at = 8;
}

message EntityEdge {
    string uuid = 1;
    string source_node_uuid = 2;
    string target_node_uuid = 3;
    string name = 4;
    string fact = 5;
    repeated float fact_embedding = 6;
    repeated string episodes = 7;
    string group_id = 8;
    google.protobuf.Timestamp valid_at = 9;
    google.protobuf.Timestamp invalid_at = 10;
    google.protobuf.Timestamp expired_at = 11;
    google.protobuf.Timestamp created_at = 12;
    google.protobuf.Timestamp updated_at = 13;
}

message EpisodicNode {
    string uuid = 1;
    string name = 2;
    string content = 3;
    string source = 4;
    string source_description = 5;
    string group_id = 6;
    google.protobuf.Timestamp valid_at = 7;
    google.protobuf.Timestamp created_at = 8;
}

message CommunityNode {
    string uuid = 1;
    string name = 2;
    string summary = 3;
    string group_id = 4;
    repeated float name_embedding = 5;
    google.protobuf.Timestamp created_at = 6;
}

message SagaNode {
    string uuid = 1;
    string name = 2;
    string summary = 3;
    string group_id = 4;
    google.protobuf.Timestamp created_at = 5;
    google.protobuf.Timestamp last_summarized_at = 6;
}

message EpisodicEdge { string uuid = 1; string source_uuid = 2; string target_uuid = 3; string group_id = 4; }
message CommunityEdge { string uuid = 1; string source_uuid = 2; string target_uuid = 3; string group_id = 4; }
message HasEpisodeEdge { string uuid = 1; string source_uuid = 2; string target_uuid = 3; string group_id = 4; }
message NextEpisodeEdge { string uuid = 1; string source_uuid = 2; string target_uuid = 3; string group_id = 4; }

message SaveBulkRequest {
    EpisodicNode episode = 1;
    repeated EntityNode entity_nodes = 2;
    repeated EntityEdge entity_edges = 3;
    repeated EpisodicEdge episodic_edges = 4;
    SagaNode saga_node = 5;
    repeated HasEpisodeEdge has_episode_edges = 6;
    repeated NextEpisodeEdge next_episode_edges = 7;
    repeated string invalidated_edge_ids = 8;
    string group_id = 9;
}
message SaveBulkResponse {}

message RetrieveEpisodesRequest { string group_id = 1; int32 last_n = 2; string source = 3; string saga_id = 4; }
message RetrieveEpisodesResponse { repeated EpisodicNode episodes = 1; }

message NodeSimilaritySearchRequest { repeated float vector = 1; repeated string group_ids = 2; int32 limit = 3; double min_score = 4; }
message NodeSimilaritySearchResponse { repeated EntityNode nodes = 1; }

message NodeFulltextSearchRequest { string query = 1; repeated string group_ids = 2; int32 limit = 3; }
message NodeFulltextSearchResponse { repeated EntityNode nodes = 1; }

message EdgeSimilaritySearchRequest {
    repeated float vector = 1;
    repeated string group_ids = 2;
    int32 limit = 3;
    double min_score = 4;
    string source_node_uuid = 5;
    string target_node_uuid = 6;
    string valid_at = 7;
}
message EdgeSimilaritySearchResponse { repeated EntityEdge edges = 1; }

message EdgeFulltextSearchRequest { string query = 1; repeated string group_ids = 2; int32 limit = 3; string valid_at = 4; }
message EdgeFulltextSearchResponse { repeated EntityEdge edges = 1; }

message NodeBFSSearchRequest { repeated string origin_uuids = 1; int32 max_depth = 2; repeated string group_ids = 3; int32 limit = 4; }
message NodeBFSSearchResponse { repeated EntityNode nodes = 1; }

message EdgeBFSSearchRequest { repeated string origin_uuids = 1; int32 max_depth = 2; repeated string group_ids = 3; int32 limit = 4; }
message EdgeBFSSearchResponse { repeated EntityEdge edges = 1; }

message NodeDistanceRerankerRequest { repeated string node_uuids = 1; string center_uuid = 2; }
message NodeDistanceRerankerResponse { map<string, double> scores = 1; }

message EpisodeMentionsRerankerRequest { repeated string node_uuids = 1; }
message EpisodeMentionsRerankerResponse { map<string, int32> counts = 1; }

message GetGroupStatsRequest { string group_id = 1; }
message GetGroupStatsResponse { string group_id = 1; int64 entity_count = 2; int64 episode_count = 3; int64 edge_count = 4; int64 community_count = 5; }

message ClearDataRequest { repeated string group_ids = 1; }
message ClearDataResponse {}

message BuildIndicesRequest { bool delete_existing = 1; }
message BuildIndicesResponse {}

message GetCommunityClustersRequest { repeated string group_ids = 1; }
message GetCommunityClustersResponse { repeated Cluster clusters = 1; }
message Cluster { repeated string node_uuids = 1; }

message RemoveCommunitiesRequest { string group_id = 1; }
message RemoveCommunitiesResponse {}

message SaveCommunityNodeRequest { CommunityNode node = 1; }
message SaveCommunityNodeResponse {}
message SaveCommunityEdgeRequest { CommunityEdge edge = 1; }
message SaveCommunityEdgeResponse {}

message GetOrCreateSagaRequest { string saga_id = 1; string group_id = 2; }
message GetOrCreateSagaResponse { SagaNode saga = 1; bool created = 2; }

message GetLastEpisodeInSagaRequest { string saga_id = 1; string group_id = 2; }
message GetLastEpisodeInSagaResponse { EpisodicNode episode = 1; }

message DeleteEpisodeRequest { string episode_uuid = 1; }
message DeleteEpisodeResponse {}
message ListEpisodesRequest { string group_id = 1; int32 last_n = 2; string source = 3; string saga_id = 4; }
message ListEpisodesResponse { repeated EpisodicNode episodes = 1; }

message EpisodeSimilaritySearchRequest { repeated float vector = 1; repeated string group_ids = 2; int32 limit = 3; double min_score = 4; }
message EpisodeSimilaritySearchResponse { repeated EpisodicNode episodes = 1; }
message EpisodeFulltextSearchRequest { string query = 1; repeated string group_ids = 2; int32 limit = 3; }
message EpisodeFulltextSearchResponse { repeated EpisodicNode episodes = 1; }
message CommunitySimilaritySearchRequest { repeated float vector = 1; repeated string group_ids = 2; int32 limit = 3; double min_score = 4; }
message CommunitySimilaritySearchResponse { repeated CommunityNode communities = 1; }
message CommunityFulltextSearchRequest { string query = 1; repeated string group_ids = 2; int32 limit = 3; }
message CommunityFulltextSearchResponse { repeated CommunityNode communities = 1; }
```

### File 2: `api/proto/graphiti/ingestion/v1/ingestion.proto`

```protobuf
syntax = "proto3";
package graphiti.ingestion.v1;
option go_package = "github.com/vnp-memory/api/proto/graphiti/ingestion/v1";

service IngestionService {
    rpc IngestEpisode(IngestEpisodeRequest) returns (IngestEpisodeResponse);
    rpc RemoveEpisode(RemoveEpisodeRequest) returns (RemoveEpisodeResponse);
    rpc ListEpisodes(ListEpisodesRequest) returns (ListEpisodesResponse);
    rpc AddTriplet(AddTripletRequest) returns (AddTripletResponse);
    rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse);
}

message IngestEpisodeRequest {
    string name = 1;
    string body = 2;
    string source = 3;
    string source_description = 4;
    string group_id = 5;
    string saga_id = 6;
    string prev_episode_uuid = 7;
    string reference_time = 8;
}

message IngestStats {
    int32 entities_extracted = 1;
    int32 entities_new = 2;
    int32 edges_extracted = 3;
    int32 edges_new = 4;
    int64 processing_time_ms = 5;
}

message IngestEpisodeResponse { string episode_uuid = 1; IngestStats stats = 2; }
message RemoveEpisodeRequest { string episode_uuid = 1; }
message RemoveEpisodeResponse {}
message ListEpisodesRequest { string group_id = 1; int32 last_n = 2; string source = 3; string saga_id = 4; }
message EpisodeItem { string uuid = 1; string name = 2; string content = 3; string source = 4; string valid_at = 5; }
message ListEpisodesResponse { repeated EpisodeItem episodes = 1; }
message AddTripletRequest { string source_entity = 1; string relation = 2; string target_entity = 3; string fact = 4; string valid_at = 5; }
message AddTripletResponse { string episode_uuid = 1; }
message GetPipelineStatusRequest {}
message GetPipelineStatusResponse { string status = 1; map<string, int32> worker_queue_depths = 2; }
```

### File 3: `api/proto/graphiti/search/v1/search.proto`

```protobuf
syntax = "proto3";
package graphiti.search.v1;
option go_package = "github.com/vnp-memory/api/proto/graphiti/search/v1";

service SearchService {
    rpc Search(SearchRequest) returns (SearchResponse);
    rpc SearchAdvanced(SearchAdvancedRequest) returns (SearchResponse);
    rpc SearchNodes(SearchNodesRequest) returns (SearchNodesResponse);
    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

message EdgeResult { string uuid = 1; string source_uuid = 2; string target_uuid = 3; string fact = 4; string name = 5; double score = 6; string valid_at = 7; string invalid_at = 8; }
message NodeResult  { string uuid = 1; string name = 2; repeated string labels = 3; string summary = 4; double score = 5; }
message EpisodeResult { string uuid = 1; string content = 2; string source = 3; string valid_at = 4; }
message CommunityResult { string uuid = 1; string name = 2; string summary = 3; double score = 4; }

message SearchRequest {
    string query = 1;
    repeated string group_ids = 2;
    int32 num_results = 3;
    string center_node_uuid = 4;
}

message SearchAdvancedRequest {
    string query = 1;
    repeated string group_ids = 2;
    int32 num_results = 3;
    string center_node_uuid = 4;
    string search_config_name = 5;
    string valid_at = 6;
    string created_at_start = 7;
    string created_at_end = 8;
}

message SearchResponse {
    repeated EdgeResult edges = 1;
    repeated NodeResult nodes = 2;
    repeated EpisodeResult episodes = 3;
    repeated CommunityResult communities = 4;
    int64 latency_ms = 5;
}

message SearchNodesRequest { string query = 1; repeated string group_ids = 2; int32 num_results = 3; }
message SearchNodesResponse { repeated NodeResult nodes = 1; }

message HealthCheckRequest {}
message HealthCheckResponse { string status = 1; }
```

### MODIFY: `Makefile`

```makefile
# Add graphiti proto compilation target
proto-graphiti:
	@echo "Compiling graphiti protobufs..."
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       api/proto/graphiti/store/v1/store.proto \
	       api/proto/graphiti/knowledge/v1/knowledge.proto \
	       api/proto/graphiti/ingestion/v1/ingestion.proto \
	       api/proto/graphiti/search/v1/search.proto \
	       api/proto/graphiti/admin/v1/admin.proto

.PHONY: proto-graphiti
```

---

## Verification

```bash
make proto-graphiti

# Verify generated files
ls api/proto/graphiti/*/v1/*.pb.go
ls api/proto/graphiti/*/v1/*_grpc.pb.go

# Verify all services can find their generated types
go build ./...
```

**Expected:** All `.pb.go` and `_grpc.pb.go` files generated. `go build ./...` passes.
