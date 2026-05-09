# 08 — Data Models & Protobuf Definitions

---

## 1. Database Schema (PostgreSQL + pgvector)

### 1.1 Tables

```sql
-- Projects (multi-tenant root)
CREATE TABLE projects (
    project_id    VARCHAR(64) PRIMARY KEY,
    project_secret VARCHAR(256) NOT NULL,
    profile_config TEXT,                    -- YAML string
    status         VARCHAR(16) DEFAULT 'active',
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);

-- Users
CREATE TABLE users (
    id             UUID DEFAULT gen_random_uuid(),
    project_id     VARCHAR(64) NOT NULL REFERENCES projects(project_id),
    additional_fields JSONB DEFAULT '{}',
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id)
);

-- General Blobs
CREATE TABLE general_blobs (
    id             UUID DEFAULT gen_random_uuid(),
    project_id     VARCHAR(64) NOT NULL,
    user_id        UUID NOT NULL,
    blob_type      VARCHAR(16) NOT NULL,
    blob_data      JSONB NOT NULL,
    additional_fields JSONB DEFAULT '{}',
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);

-- Buffer Zones
CREATE TABLE buffer_zones (
    id             UUID DEFAULT gen_random_uuid(),
    project_id     VARCHAR(64) NOT NULL,
    user_id        UUID NOT NULL,
    blob_id        UUID NOT NULL,
    blob_type      VARCHAR(16) NOT NULL,
    token_size     INT NOT NULL DEFAULT 0,
    status         VARCHAR(16) NOT NULL DEFAULT 'idle',
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (blob_id, project_id) REFERENCES general_blobs(id, project_id) ON DELETE CASCADE
);

-- User Profiles
CREATE TABLE user_profiles (
    id             UUID DEFAULT gen_random_uuid(),
    project_id     VARCHAR(64) NOT NULL,
    user_id        UUID NOT NULL,
    content        TEXT NOT NULL,
    attributes     JSONB NOT NULL DEFAULT '{}',  -- {topic, sub_topic}
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);

-- User Events
CREATE TABLE user_events (
    id             UUID DEFAULT gen_random_uuid(),
    project_id     VARCHAR(64) NOT NULL,
    user_id        UUID NOT NULL,
    event_data     JSONB NOT NULL,
    embedding      vector,                        -- pgvector
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);

-- User Event Gists
CREATE TABLE user_event_gists (
    id             UUID DEFAULT gen_random_uuid(),
    project_id     VARCHAR(64) NOT NULL,
    user_id        UUID NOT NULL,
    event_id       UUID NOT NULL,
    gist_data      JSONB NOT NULL,
    embedding      vector,                        -- pgvector
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (event_id, project_id) REFERENCES user_events(id, project_id) ON DELETE CASCADE
);

-- Billings
CREATE TABLE billings (
    id             UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    usage_left     BIGINT DEFAULT 0,
    next_refill_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE project_billings (
    project_id     VARCHAR(64) REFERENCES projects(project_id),
    billing_id     UUID REFERENCES billings(id),
    PRIMARY KEY (project_id, billing_id)
);

-- User Statuses
CREATE TABLE user_statuses (
    id             UUID DEFAULT gen_random_uuid(),
    project_id     VARCHAR(64) NOT NULL,
    user_id        UUID NOT NULL,
    status_type    VARCHAR(32) NOT NULL,
    attributes     JSONB DEFAULT '{}',
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
```

### 1.2 Indexes

```sql
CREATE INDEX idx_users_project ON users(project_id);
CREATE INDEX idx_blobs_user_type ON general_blobs(user_id, project_id, blob_type);
CREATE INDEX idx_buffer_user_status ON buffer_zones(user_id, project_id, blob_type, status);
CREATE INDEX idx_profiles_user ON user_profiles(user_id, project_id);
CREATE INDEX idx_events_user ON user_events(user_id, project_id);
CREATE INDEX idx_events_user_time ON user_events(user_id, project_id, created_at DESC);
CREATE INDEX idx_gists_user_event ON user_event_gists(user_id, project_id, event_id);

-- Vector indexes (HNSW)
CREATE INDEX idx_events_embedding ON user_events
    USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 200);
CREATE INDEX idx_gists_embedding ON user_event_gists
    USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 200);
```

---

## 2. Protobuf Definitions

### 2.1 Common Types

```protobuf
// api/proto/common/v1/common.proto
syntax = "proto3";
package common.v1;

message Pagination {
    int32 page = 1;
    int32 page_size = 2;
    int32 total = 3;
}

message Timestamp {
    google.protobuf.Timestamp created_at = 1;
    google.protobuf.Timestamp updated_at = 2;
}
```

### 2.2 Ingestion Service

```protobuf
// api/proto/memobase/ingestion/v1/ingestion.proto
service IngestionService {
    rpc InsertBlob(InsertBlobRequest) returns (InsertBlobResponse);
    rpc GetBlob(GetBlobRequest) returns (GetBlobResponse);
    rpc DeleteBlob(DeleteBlobRequest) returns (DeleteBlobResponse);
    rpc FlushBuffer(FlushBufferRequest) returns (FlushBufferResponse);
    rpc GetBufferCapacity(GetBufferCapacityRequest) returns (GetBufferCapacityResponse);
    rpc GetBlobsByBufferIds(GetBlobsByBufferIdsRequest) returns (GetBlobsByBufferIdsResponse);
}

message InsertBlobRequest {
    string user_id = 1;
    string blob_type = 2;        // chat, doc, summary
    bytes blob_data = 3;         // JSON
    bytes additional_fields = 4; // JSON
}
```

### 2.3 Engine Service

```protobuf
// api/proto/memobase/engine/v1/engine.proto
service EngineService {
    rpc ProcessBlobs(ProcessBlobsRequest) returns (ProcessBlobsResponse);
}

message ProcessBlobsRequest {
    string user_id = 1;
    string project_id = 2;
    repeated string buffer_ids = 3;
    string blob_type = 4;
}

message ProcessBlobsResponse {
    string event_id = 1;
    repeated string added_profile_ids = 2;
    repeated string updated_profile_ids = 3;
    repeated string deleted_profile_ids = 4;
}
```

### 2.4 Context Service

```protobuf
// api/proto/memobase/context/v1/context.proto
service ContextService {
    rpc GetContext(GetContextRequest) returns (GetContextResponse);
    rpc GetProfiles(GetProfilesRequest) returns (GetProfilesResponse);
    rpc AddProfile(AddProfileRequest) returns (AddProfileResponse);
    rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileResponse);
    rpc DeleteProfile(DeleteProfileRequest) returns (DeleteProfileResponse);
}

message GetContextRequest {
    string user_id = 1;
    int32 max_token_size = 2;
    repeated string prefer_topics = 3;
    repeated string only_topics = 4;
    double profile_event_ratio = 5;
    repeated ChatMessage chats = 6;
    string custom_prompt = 7;
    double event_similarity_threshold = 8;
    int32 time_range_in_days = 9;
}
```

### 2.5 Event Service

```protobuf
// api/proto/memobase/event/v1/event.proto
service EventService {
    rpc GetEvents(GetEventsRequest) returns (GetEventsResponse);
    rpc UpdateEvent(UpdateEventRequest) returns (UpdateEventResponse);
    rpc DeleteEvent(DeleteEventRequest) returns (DeleteEventResponse);
    rpc SearchEvents(SearchEventsRequest) returns (SearchEventsResponse);
    rpc SearchEventGists(SearchGistsRequest) returns (SearchGistsResponse);
    rpc FilterByTags(FilterByTagsRequest) returns (FilterByTagsResponse);
    rpc AppendEvent(AppendEventRequest) returns (AppendEventResponse);
}
```

### 2.6 Admin Service

```protobuf
// api/proto/memobase/admin/v1/admin.proto
service AdminService {
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
    rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
    rpc ListProjectUsers(ListProjectUsersRequest) returns (ListProjectUsersResponse);
    rpc UpdateProfileConfig(UpdateProfileConfigRequest) returns (UpdateProfileConfigResponse);
    rpc GetProfileConfig(GetProfileConfigRequest) returns (GetProfileConfigResponse);
    rpc GetBilling(GetBillingRequest) returns (GetBillingResponse);
    rpc GetUsage(GetUsageRequest) returns (GetUsageResponse);
    rpc VerifyProject(VerifyProjectRequest) returns (VerifyProjectResponse);
    rpc StatusCheck(StatusCheckRequest) returns (StatusCheckResponse);
}
```
