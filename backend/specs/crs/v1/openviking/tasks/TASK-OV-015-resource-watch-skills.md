# TASK-OV-015 — `services/openviking-resource` WatchManager, Skills & gRPC Server

**Wave:** 5 (Context)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-014 (resource pipeline)  
**Ước tính:** 3 giờ  
**Solution tham chiếu:** [SOL-OV-005 §5, §6, §7, §8](../solutions/SOL-OV-005-Resource-Service.md)

---

## Mục tiêu

Hoàn thiện `services/openviking-resource/` với: WatchManager (periodic check git/HTTP changes → auto-refresh), Skill loading từ VikingFS, Task tracking Redis store, gRPC handler, NATS events, config và main.go.

---

## Các file cần tạo

### 1. `internal/usecase/watch_resources.go` — WatchManager

```go
type WatchManager struct {
    resourceRepo  port.ResourceRepo
    gitAdapter    *source.GitAdapter
    httpAdapter   *source.HTTPAdapter
    refreshUC     *RefreshResourceUseCase
    config        *Config
    ticker        *time.Ticker
    stop          chan struct{}
}

func NewWatchManager(
    repo port.ResourceRepo,
    git *source.GitAdapter,
    http *source.HTTPAdapter,
    refresh *RefreshResourceUseCase,
    config *Config,
) *WatchManager

func (wm *WatchManager) Start(ctx context.Context) {
    wm.ticker = time.NewTicker(wm.config.WatchCheckInterval)  // default: 5min
    for {
        select {
        case <-wm.ticker.C:
            wm.checkAll(ctx)
        case <-ctx.Done():
            wm.ticker.Stop()
            return
        }
    }
}

func (wm *WatchManager) checkAll(ctx context.Context) {
    resources, err := wm.resourceRepo.ListWatched(ctx)
    if err != nil { slog.Warn("watch check failed", "error", err); return }
    
    for _, resource := range resources {
        if !resource.WatchConfig.Enabled { continue }
        
        var hasChanges bool
        var newCheckpoint string
        
        switch resource.SourceType {
        case domain.SourceTypeGit:
            hasChanges, newCheckpoint, _ = wm.gitAdapter.HasNewCommits(
                resource.SourceURL, resource.WatchConfig.LastCommit)
        case domain.SourceTypeHTTP:
            hasChanges, newCheckpoint, _ = wm.httpAdapter.HasChanged(
                resource.SourceURL, resource.WatchConfig.LastETag)
        default:
            continue  // Local files: no auto-watch
        }
        
        if hasChanges {
            slog.Info("resource has updates", "resource", resource.Name)
            resource.WatchConfig.LastCommit = newCheckpoint
            resource.WatchConfig.LastETag   = newCheckpoint
            wm.resourceRepo.Update(ctx, resource)
            
            // Async refresh
            go func(r *domain.Resource) {
                wm.refreshUC.Execute(context.Background(), RefreshResourceRequest{
                    ResourceID: r.ID, AccountID: r.AccountID,
                })
            }(resource)
        }
    }
}

func (wm *WatchManager) Stop() {
    close(wm.stop)
}
```

### 2. `internal/usecase/refresh_resource.go`

```go
func (uc *RefreshResourceUseCase) Execute(ctx context.Context, req RefreshResourceRequest) (*RefreshResponse, error) {
    resource, err := uc.resourceRepo.GetByID(ctx, req.ResourceID)
    if err != nil { return nil, err }
    
    // Re-run full ingestion pipeline
    return uc.addResourceUC.Execute(ctx, AddResourceRequest{
        AccountID:         resource.AccountID,
        UserID:            resource.UserID,
        URLOrPath:         resource.SourceURL,
        Name:              resource.Name,
        GenerateSummaries: true,
        Watch:             resource.WatchConfig != nil && resource.WatchConfig.Enabled,
        WatchInterval:     func() time.Duration {
            if resource.WatchConfig != nil { return resource.WatchConfig.Interval }
            return 0
        }(),
    })
}
```

### 3. `internal/usecase/delete_resource.go`

```go
func (uc *DeleteResourceUseCase) Execute(ctx context.Context, req DeleteResourceRequest) error {
    resource, err := uc.resourceRepo.GetByID(ctx, req.ResourceID)
    if err != nil { return err }
    
    // 1. Delete from VikingFS (triggers ov.content.deleted NATS per file → Search cleanup)
    uc.fsClient.Rm(ctx, resource.VikingURI, true)
    
    // 2. Disable watch
    if resource.WatchConfig != nil {
        resource.WatchConfig.Enabled = false
        uc.resourceRepo.Update(ctx, resource)
    }
    
    // 3. Delete from DB
    return uc.resourceRepo.Delete(ctx, req.ResourceID)
}
```

### 4. `internal/usecase/load_skills.go` — Load Agent Skills

```go
// Skill structure in VikingFS:
// viking://agent/{accountID}/{agentID}/skills/
//   ├── web_search/
//   │   ├── .abstract.md      (L0: one-sentence description)
//   │   ├── .overview.md      (L1: full description + examples)
//   │   └── instructions.md   (L2: implementation guide)
//   └── code_executor/
//       └── ...

type LoadSkillsUseCase struct {
    fsClient port.FSClient
}

func (uc *LoadSkillsUseCase) Execute(ctx context.Context, req LoadSkillsRequest) (*LoadSkillsResponse, error) {
    skillsURI := fmt.Sprintf("viking://agent/%s/%s/skills/", req.AccountID, req.AgentID)
    
    entries, err := uc.fsClient.Ls(ctx, skillsURI)
    if err != nil {
        if errors.Is(err, fs.ErrNotExist) {
            return &LoadSkillsResponse{Skills: []*domain.Skill{}}, nil
        }
        return nil, err
    }
    
    var skills []*domain.Skill
    for _, entry := range entries {
        if !entry.IsDirectory { continue }
        
        // Read L0 abstract (skill description)
        abstractBytes, _ := uc.fsClient.Read(ctx, entry.URI+".abstract.md", 0)
        
        // Read L2 instructions
        instrBytes, _ := uc.fsClient.Read(ctx, entry.URI+"instructions.md", 2)
        
        skills = append(skills, &domain.Skill{
            ID:           uuid.New().String(),
            AgentID:      req.AgentID,
            AccountID:    req.AccountID,
            Name:         entry.Name,
            Description:  string(abstractBytes),
            Instructions: string(instrBytes),
            VikingURI:    entry.URI,
        })
    }
    
    return &LoadSkillsResponse{Skills: skills, AgentID: req.AgentID}, nil
}
```

### 5. Task Redis Store

**File: `internal/adapter/store/task_redis.go`**

```go
type TaskRedisStore struct {
    redis  *redis.Client
    keyTTL time.Duration  // default: 7 days
}

const taskKeyPrefix = "ov_task"

func taskKey(accountID, taskID string) string {
    return fmt.Sprintf("%s:%s:%s", taskKeyPrefix, accountID, taskID)
}

func (s *TaskRedisStore) Save(ctx context.Context, task *domain.BackgroundTask) error {
    task.CreatedAt = time.Now(); task.UpdatedAt = time.Now()
    data, _ := json.Marshal(task)
    return s.redis.Set(ctx, taskKey(task.AccountID, task.ID), data, s.keyTTL).Err()
}

func (s *TaskRedisStore) GetByID(ctx context.Context, accountID, taskID string) (*domain.BackgroundTask, error) {
    data, err := s.redis.Get(ctx, taskKey(accountID, taskID)).Bytes()
    if err != nil { return nil, err }
    var task domain.BackgroundTask
    json.Unmarshal(data, &task)
    return &task, nil
}

func (s *TaskRedisStore) UpdateProgress(ctx context.Context, accountID, taskID string, progress float64, state domain.TaskState, errMsg string) error {
    task, err := s.GetByID(ctx, accountID, taskID)
    if err != nil { return err }
    task.Progress = progress; task.State = state; task.ErrorMsg = errMsg
    task.UpdatedAt = time.Now()
    data, _ := json.Marshal(task)
    return s.redis.Set(ctx, taskKey(accountID, taskID), data, s.keyTTL).Err()
}

func (s *TaskRedisStore) ListByAccount(ctx context.Context, accountID string) ([]*domain.BackgroundTask, error) {
    pattern := fmt.Sprintf("%s:%s:*", taskKeyPrefix, accountID)
    keys, err := s.redis.Keys(ctx, pattern).Result()
    if err != nil { return nil, err }
    
    var tasks []*domain.BackgroundTask
    for _, key := range keys {
        data, _ := s.redis.Get(ctx, key).Bytes()
        var task domain.BackgroundTask
        json.Unmarshal(data, &task)
        tasks = append(tasks, &task)
    }
    return tasks, nil
}
```

### 6. Resource PostgreSQL Repo

**File: `internal/adapter/store/resource_postgres.go`**

```go
// Schema:
// CREATE TABLE resources (
//   id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
//   account_id   VARCHAR(63) NOT NULL,
//   user_id      UUID NOT NULL,
//   name         VARCHAR(255) NOT NULL,
//   source_url   TEXT NOT NULL,
//   source_type  VARCHAR(20) NOT NULL,
//   status       VARCHAR(20) NOT NULL DEFAULT 'queued',
//   viking_uri   TEXT NOT NULL,
//   watch_config JSONB,
//   file_count   INT NOT NULL DEFAULT 0,
//   error_msg    TEXT,
//   created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
//   updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
//   UNIQUE(account_id, name)
// );

type PostgresResourceRepo struct {
    db *pgx.Conn
}

func (r *PostgresResourceRepo) Save(ctx context.Context, resource *domain.Resource) error
func (r *PostgresResourceRepo) GetByID(ctx context.Context, id string) (*domain.Resource, error)
func (r *PostgresResourceRepo) ListByAccount(ctx context.Context, accountID string) ([]*domain.Resource, error)
func (r *PostgresResourceRepo) ListWatched(ctx context.Context) ([]*domain.Resource, error)
// WHERE watch_config->>'enabled' = 'true' AND status = 'done'
func (r *PostgresResourceRepo) Update(ctx context.Context, resource *domain.Resource) error
func (r *PostgresResourceRepo) Delete(ctx context.Context, id string) error
```

### 7. gRPC Handler

**File: `internal/adapter/grpc/handler.go`**

```go
type Handler struct {
    resourcev1.UnimplementedResourceServiceServer
    addUC      *usecase.AddResourceUseCase
    refreshUC  *usecase.RefreshResourceUseCase
    deleteUC   *usecase.DeleteResourceUseCase
    listUC     *usecase.ListResourcesUseCase
    skillsUC   *usecase.LoadSkillsUseCase
    taskStore  port.TaskStore
    resourceRepo port.ResourceRepo
}

// All 7 RPC methods:
// AddResource, GetResource, ListResources, DeleteResource, RefreshResource
// GetResourceStatus (= get task by ID)
// LoadSkills, ListSkills
// GetTask, ListTasks
```

### 8. NATS Events

```go
// Published:
// Subject: "ov.resource.ingested"
// Payload: {resource_id, name, uri, file_count}
// Subscriber: Search service can trigger special directory-level indexing

// Subscribed:
// Subject: "admin.account.created"
// Action: Pre-create resources namespace for account
//   → fsClient.Mkdir("viking://resources/", existOK=true)
//   (resources namespace is global, not per account)
```

---

## 9. Config

```yaml
resource:
  grpc:
    port: 9014
  health:
    port: 9094
  ingestion:
    worker_pool_size: 5
    vlm_worker_pool_size: 2
    temp_dir: "~/.openviking/temp"
    generate_summaries: true
    summary_max_tokens_l0: 100
    summary_max_tokens_l1: 2000
  watch:
    enabled: true
    check_interval: 5m
    max_watched: 100
  git:
    clone_depth: 1
    timeout: 300s
  http:
    download_timeout: 60s
    max_size_mb: 100
    user_agent: "OpenViking/1.0"
  clients:
    fs: "openviking-fs:9011"
    search: "openviking-search:9012"
    vlm: "bifrost:4000"
  redis:
    url: "redis://redis:6379/3"
    task_ttl: 168h
  database:
    url: "${DATABASE_URL}"
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
```

---

## Unit Tests

```
TestWatchManager_GitNewCommits       → new HEAD → refresh triggered
TestWatchManager_HTTPETagChanged     → new ETag → refresh triggered
TestWatchManager_NoChanges           → same HEAD → no refresh
TestWatchManager_LocalSkipped        → SourceTypeLocalDir → no check
TestWatchManager_DisabledSkipped     → WatchConfig.Enabled=false → no check
TestRefreshResource_RerunsIngestion  → calls addResourceUC with same URL
TestDeleteResource_RemovesFromFS     → fsClient.Rm called
TestDeleteResource_DisablesWatch     → WatchConfig.Enabled=false after delete
TestLoadSkills_ReadsAbstract         → .abstract.md → skill.Description set
TestLoadSkills_ReadsInstructions     → instructions.md → skill.Instructions set
TestLoadSkills_EmptyDir              → no skill dirs → empty list, no error
TestTaskRedisStore_SaveAndGet        → save → get → same task
TestTaskRedisStore_UpdateProgress    → save → update 0.5/processing → get returns updated
TestTaskRedisStore_ListByAccount     → save 3 tasks → list → 3 returned
TestGRPCHandler_AddResource          → request → task ID returned
TestGRPCHandler_GetTask_Progress     → task in store → returns progress
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
buf generate services/openviking-resource/
go build ./services/openviking-resource/...
go test ./services/openviking-resource/... -v -count=1

# Integration test (requires git binary)
go test ./services/openviking-resource/... -v -run "TestGit" -tags integration
```

---

## Ghi chú triển khai

- WatchManager chạy trong goroutine sau khi service start, không block gRPC listener
- `ListWatched` query: PostgreSQL `WHERE (watch_config->'enabled')::boolean = true AND status = 'done'`
- Resources namespace `viking://resources/` là GLOBAL (không phân theo account) — mọi account đều share namespace này; access control qua RBAC (account-level visibility future feature)
- Skill instructions: không encrypt (agent instructions là công khai trong account scope)
