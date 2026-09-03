import os

PROTO_TEMPLATES = {
    'sm-analytics': """syntax = "proto3";

package vnp.memory.smanalytics.v1;
option go_package = "vnp-memory/services/sm-analytics/api/proto/v1;smanalyticsv1";

service SmAnalyticsService {
  rpc GetUsageAnalytics(GetUsageRequest) returns (GetUsageResponse);
  rpc GetMemoryAnalytics(GetMemoryRequest) returns (GetMemoryResponse);
}

message GetUsageRequest {
  string org_id = 1;
  string time_range = 2;
}

message GetUsageResponse {
  int32 total_tokens_saved = 1;
  float cost_saved_usd = 2;
  int32 total_requests = 3;
}

message GetMemoryRequest {
  string org_id = 1;
}

message GetMemoryResponse {
  int32 active_memories = 1;
}
""",

    'sm-auth': """syntax = "proto3";

package vnp.memory.smauth.v1;
option go_package = "vnp-memory/services/sm-auth/api/proto/v1;smauthv1";

service SmAuthService {
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
}

message CreateAPIKeyRequest {
  string org_id = 1;
  string name = 2;
}

message CreateAPIKeyResponse {
  string raw_key = 1;
  string key_id = 2;
}

message ValidateAPIKeyRequest {
  string api_key = 1;
}

message ValidateAPIKeyResponse {
  bool is_valid = 1;
  string org_id = 2;
  string role = 3;
}
""",

    'sm-connector': """syntax = "proto3";

package vnp.memory.smconnector.v1;
option go_package = "vnp-memory/services/sm-connector/api/proto/v1;smconnectorv1";

service SmConnectorService {
  rpc CreateConnection(CreateConnectionRequest) returns (CreateConnectionResponse);
  rpc SyncConnection(SyncConnectionRequest) returns (SyncConnectionResponse);
}

message CreateConnectionRequest {
  string provider = 1;
  string redirect_url = 2;
}

message CreateConnectionResponse {
  string auth_link = 1;
  string state_token = 2;
}

message SyncConnectionRequest {
  string connection_id = 1;
}

message SyncConnectionResponse {
  string sync_job_id = 1;
  string status = 2;
}
""",

    'sm-engine': """syntax = "proto3";

package vnp.memory.smengine.v1;
option go_package = "vnp-memory/services/sm-engine/api/proto/v1;smenginev1";

service SmEngineService {
  rpc ProcessDocument(ProcessDocumentRequest) returns (ProcessDocumentResponse);
  rpc SearchMemories(SearchMemoriesRequest) returns (SearchMemoriesResponse);
}

message ProcessDocumentRequest {
  string content = 1;
  string space_id = 2;
}

message ProcessDocumentResponse {
  string document_id = 1;
  int32 chunks_created = 2;
  int32 memories_extracted = 3;
}

message SearchMemoriesRequest {
  string query = 1;
  string space_id = 2;
}

message SearchMemoriesResponse {
  repeated string memory_snippets = 1;
}
""",

    'sm-search': """syntax = "proto3";

package vnp.memory.smsearch.v1;
option go_package = "vnp-memory/services/sm-search/api/proto/v1;smsearchv1";

service SmSearchService {
  rpc HybridSearch(HybridSearchRequest) returns (HybridSearchResponse);
}

message HybridSearchRequest {
  string query = 1;
  string space_id = 2;
  int32 limit = 3;
}

message HybridSearchResponse {
  repeated SearchResult results = 1;
}

message SearchResult {
  string id = 1;
  string content = 2;
  float score = 3;
}
""",

    'sm-project': """syntax = "proto3";

package vnp.memory.smproject.v1;
option go_package = "vnp-memory/services/sm-project/api/proto/v1;smprojectv1";

service SmProjectService {
  rpc CreateSpace(CreateSpaceRequest) returns (CreateSpaceResponse);
  rpc AddMember(AddMemberRequest) returns (AddMemberResponse);
}

message CreateSpaceRequest {
  string name = 1;
  string org_id = 2;
}

message CreateSpaceResponse {
  string space_id = 1;
}

message AddMemberRequest {
  string space_id = 1;
  string user_id = 2;
  string role = 3;
}

message AddMemberResponse {
  bool success = 1;
}
"""
}

BASE_DIR = '/Users/binhnt/Work/blockchain/vnp-memory/services'

for svc, content in PROTO_TEMPLATES.items():
    proto_dir = os.path.join(BASE_DIR, svc, 'api', 'proto', 'v1')
    os.makedirs(proto_dir, exist_ok=True)
    
    svc_name_stripped = svc.replace('sm-', '')
    proto_file = os.path.join(proto_dir, f"{svc_name_stripped}.proto")
    
    print(f"Creating proto definition for {svc}...")
    with open(proto_file, 'w') as f:
        f.write(content)
        
print("Proto files generated successfully.")
