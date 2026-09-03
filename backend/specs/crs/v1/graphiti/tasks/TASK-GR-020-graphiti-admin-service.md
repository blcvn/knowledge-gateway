# TASK-GR-020 — graphiti-admin Service

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-020 |
| **Wave** | 3 |
| **Component** | `services/graphiti-admin/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-007 §2, §3 |
| **Priority** | Medium |
| **Depends On** | TASK-GR-010 |
| **Estimated** | 5h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-admin: 6 .go - admin service  
---

## Context

Tạo `services/graphiti-admin/` — service mới hoàn toàn. Cung cấp admin operations: community detection trigger, group data management, stats retrieval, index rebuild.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-admin/main.go` |
| CREATE | `services/graphiti-admin/internal/usecase/build_communities.go` |
| CREATE | `services/graphiti-admin/internal/usecase/manage_group.go` |
| CREATE | `services/graphiti-admin/internal/adapter/grpc/handler.go` |

---

## Implementation

### File 1: `services/graphiti-admin/main.go`

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/vnp-memory/services/graphiti-admin/internal/adapter/grpc"
    "github.com/vnp-memory/services/graphiti-admin/internal/usecase"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Initialize dependencies
    storeConn  := connectToGraphitiStore()
    knowledgeConn := connectToGraphitiKnowledge()
    natsConn   := connectToNATS()

    // Use cases
    buildCommunities := usecase.NewBuildCommunitiesAdminUseCase(storeConn, knowledgeConn)
    manageGroup      := usecase.NewManageGroupUseCase(storeConn, natsConn)

    // Start gRPC server
    server := grpc.NewAdminServer(buildCommunities, manageGroup)
    if err := server.Start(ctx, os.Getenv("ADMIN_GRPC_PORT")); err != nil {
        log.Fatalf("admin server: %v", err)
    }
}
```

### File 2: `services/graphiti-admin/internal/usecase/build_communities.go`

```go
package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/vnp-memory/services/graphiti-knowledge/internal/usecase"
    storepb "github.com/vnp-memory/api/proto/graphiti/store/v1"
    knowledgepb "github.com/vnp-memory/api/proto/graphiti/knowledge/v1"
)

type BuildCommunitiesAdminUseCase struct {
    storeClient     storepb.StoreServiceClient
    knowledgeClient knowledgepb.KnowledgeServiceClient
}

func NewBuildCommunitiesAdminUseCase(
    store storepb.StoreServiceClient,
    knowledge knowledgepb.KnowledgeServiceClient,
) *BuildCommunitiesAdminUseCase {
    return &BuildCommunitiesAdminUseCase{storeClient: store, knowledgeClient: knowledge}
}

type BuildCommunitiesAdminReq struct {
    GroupID    string
    DeleteMode bool  // true: remove existing communities first
}

type BuildCommunitiesAdminResult struct {
    CommunitiesBuilt int
    EntitiesGrouped  int
    ProcessingTimeMs int64
}

func (uc *BuildCommunitiesAdminUseCase) Execute(ctx context.Context, req BuildCommunitiesAdminReq) (*BuildCommunitiesAdminResult, error) {
    start := time.Now()

    // Step 1: Remove existing communities
    if req.DeleteMode {
        _, err := uc.storeClient.RemoveCommunities(ctx, &storepb.RemoveCommunitiesRequest{GroupId: req.GroupID})
        if err != nil { return nil, fmt.Errorf("remove existing communities: %w", err) }
    }

    // Step 2: Build communities via knowledge service
    resp, err := uc.knowledgeClient.BuildCommunities(ctx, &knowledgepb.BuildCommunitiesRequest{GroupId: req.GroupID})
    if err != nil { return nil, fmt.Errorf("build communities: %w", err) }

    return &BuildCommunitiesAdminResult{
        CommunitiesBuilt: int(resp.CommunitiesBuilt),
        EntitiesGrouped:  int(resp.EntitiesGrouped),
        ProcessingTimeMs: time.Since(start).Milliseconds(),
    }, nil
}
```

### File 3: `services/graphiti-admin/internal/usecase/manage_group.go`

```go
package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/nats-io/nats.go"
    "encoding/json"
    storepb "github.com/vnp-memory/api/proto/graphiti/store/v1"
)

type ManageGroupUseCase struct {
    storeClient storepb.StoreServiceClient
    natsConn    *nats.Conn
}

func NewManageGroupUseCase(store storepb.StoreServiceClient, nats *nats.Conn) *ManageGroupUseCase {
    return &ManageGroupUseCase{storeClient: store, natsConn: nats}
}

// GetStats returns entity/episode/edge counts for a group
func (uc *ManageGroupUseCase) GetStats(ctx context.Context, groupID string) (*GroupStats, error) {
    resp, err := uc.storeClient.GetGroupStats(ctx, &storepb.GetGroupStatsRequest{GroupId: groupID})
    if err != nil { return nil, err }
    return &GroupStats{
        GroupID:        groupID,
        EntityCount:    resp.EntityCount,
        EpisodeCount:   resp.EpisodeCount,
        EdgeCount:      resp.EdgeCount,
        CommunityCount: resp.CommunityCount,
    }, nil
}

type GroupStats struct {
    GroupID        string `json:"group_id"`
    EntityCount    int64  `json:"entity_count"`
    EpisodeCount   int64  `json:"episode_count"`
    EdgeCount      int64  `json:"edge_count"`
    CommunityCount int64  `json:"community_count"`
}

// ClearGroup deletes ALL data (entities + episodes + edges + communities) for a group
func (uc *ManageGroupUseCase) ClearGroup(ctx context.Context, groupID string) error {
    _, err := uc.storeClient.ClearData(ctx, &storepb.ClearDataRequest{GroupIds: []string{groupID}})
    if err != nil { return fmt.Errorf("clear group data: %w", err) }

    // Publish event so search cache is invalidated
    data, _ := json.Marshal(map[string]string{"group_id": groupID, "action": "clear"})
    uc.natsConn.Publish("graphiti.group.cleared", data)
    return nil
}

// RebuildIndices drops and recreates all Neo4j indices
func (uc *ManageGroupUseCase) RebuildIndices(ctx context.Context) error {
    _, err := uc.storeClient.BuildIndicesAndConstraints(ctx, &storepb.BuildIndicesRequest{DeleteExisting: true})
    return err
}
```

### File 4: `services/graphiti-admin/internal/adapter/grpc/handler.go`

```go
package grpc

import (
    "context"

    pb "github.com/vnp-memory/api/proto/graphiti/admin/v1"
    "github.com/vnp-memory/services/graphiti-admin/internal/usecase"
)

type AdminHandler struct {
    pb.UnimplementedAdminServiceServer
    communities *usecase.BuildCommunitiesAdminUseCase
    groups      *usecase.ManageGroupUseCase
}

func NewAdminServer(communities *usecase.BuildCommunitiesAdminUseCase, groups *usecase.ManageGroupUseCase) *AdminHandler {
    return &AdminHandler{communities: communities, groups: groups}
}

func (h *AdminHandler) BuildCommunities(ctx context.Context, req *pb.BuildCommunitiesRequest) (*pb.BuildCommunitiesResponse, error) {
    result, err := h.communities.Execute(ctx, usecase.BuildCommunitiesAdminReq{
        GroupID:    req.GroupId,
        DeleteMode: req.DeleteExisting,
    })
    if err != nil { return nil, err }
    return &pb.BuildCommunitiesResponse{
        CommunitiesBuilt: int32(result.CommunitiesBuilt),
        EntitiesGrouped:  int32(result.EntitiesGrouped),
        ProcessingTimeMs: result.ProcessingTimeMs,
    }, nil
}

func (h *AdminHandler) GetGroupStats(ctx context.Context, req *pb.GetGroupStatsRequest) (*pb.GetGroupStatsResponse, error) {
    stats, err := h.groups.GetStats(ctx, req.GroupId)
    if err != nil { return nil, err }
    return &pb.GetGroupStatsResponse{
        GroupId:        stats.GroupID,
        EntityCount:    stats.EntityCount,
        EpisodeCount:   stats.EpisodeCount,
        EdgeCount:      stats.EdgeCount,
        CommunityCount: stats.CommunityCount,
    }, nil
}

func (h *AdminHandler) ClearGroup(ctx context.Context, req *pb.ClearGroupRequest) (*pb.ClearGroupResponse, error) {
    return &pb.ClearGroupResponse{}, h.groups.ClearGroup(ctx, req.GroupId)
}

func (h *AdminHandler) RebuildIndices(ctx context.Context, req *pb.RebuildIndicesRequest) (*pb.RebuildIndicesResponse, error) {
    return &pb.RebuildIndicesResponse{}, h.groups.RebuildIndices(ctx)
}
```

---

## Proto Contract: `api/proto/graphiti/admin/v1/admin.proto`

```proto
syntax = "proto3";
package graphiti.admin.v1;
option go_package = "github.com/vnp-memory/api/proto/graphiti/admin/v1";

service AdminService {
    rpc BuildCommunities(BuildCommunitiesRequest) returns (BuildCommunitiesResponse);
    rpc GetGroupStats(GetGroupStatsRequest) returns (GetGroupStatsResponse);
    rpc ClearGroup(ClearGroupRequest) returns (ClearGroupResponse);
    rpc RebuildIndices(RebuildIndicesRequest) returns (RebuildIndicesResponse);
}

message BuildCommunitiesRequest { string group_id = 1; bool delete_existing = 2; }
message BuildCommunitiesResponse { int32 communities_built = 1; int32 entities_grouped = 2; int64 processing_time_ms = 3; }
message GetGroupStatsRequest { string group_id = 1; }
message GetGroupStatsResponse { string group_id = 1; int64 entity_count = 2; int64 episode_count = 3; int64 edge_count = 4; int64 community_count = 5; }
message ClearGroupRequest { string group_id = 1; }
message ClearGroupResponse {}
message RebuildIndicesRequest {}
message RebuildIndicesResponse {}
```

---

## Verification

```bash
cd services/graphiti-admin
go build ./...
```

**Acceptance tests (admin endpoints via gateway):**
1. `POST /v1/graphiti/admin/communities/build?group_id=test` → 200 + communities count
2. `GET /v1/graphiti/admin/groups/{id}/stats` → entity/episode counts
3. `DELETE /v1/graphiti/admin/groups/{id}` → 204, Neo4j data deleted
4. `POST /v1/graphiti/admin/indices/rebuild` → 200, indices rebuilt
