# Solution: SOL-OV-005 — Resource Service (Ingestion Pipeline)

**CR:** [CR-OV-005](../CR-OV-005-Resource-Service.md)  
**Wave:** 5 (Context — song song với Session service)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/openviking-resource` — ingestion pipeline xử lý external resources (git repos, web pages, local files, documents) và ingest vào VikingFS dưới `viking://resources/`.

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Không có multi-source ingestion | Source detector + 4 adapters (git/HTTP/local/doc) |
| Không có code parsing | `pkg/parse/` Registry với tree-sitter (50+ langs) |
| Không có L0/L1 auto-generation | VLM goroutine pool (max 2) sau main parse |
| Không có watch/refresh | WatchManager với periodic ticker + ETag/git-diff |
| Không có task tracking | Background task state machine in Redis |
| Large repo → OOM | Goroutine pool (max 5) + streaming file processing |

---

## 2. Codebase Structure

```
services/openviking-resource/
├── cmd/server/main.go
├── api/proto/resource/v1/resource.proto
├── internal/
│   ├── domain/
│   │   ├── resource.go      # Resource, SourceType, ResourceStatus
│   │   ├── skill.go         # Skill, SkillDirectory
│   │   ├── task.go          # BackgroundTask, TaskState
│   │   ├── parsed_file.go   # ParsedFile, Chunk
│   │   └── errors.go
│   ├── usecase/
│   │   ├── add_resource.go      # Orchestrate 7-step ingestion pipeline
│   │   ├── refresh_resource.go  # Re-run ingestion on existing resource
│   │   ├── delete_resource.go   # Delete resource + cleanup Search
│   │   ├── load_skills.go       # Load agent skills from directory
│   │   ├── watch_resources.go   # WatchManager background ticker
│   │   ├── pipeline/
│   │   │   ├── detect_source.go   # Source type detection
│   │   │   ├── clone_download.go  # Git clone / HTTP download
│   │   │   ├── scan_directory.go  # Build file tree, apply .gitignore
│   │   │   ├── parse_files.go     # Goroutine pool parser
│   │   │   ├── build_fs.go        # Write parsed content to FS
│   │   │   ├── generate_summaries.go # VLM L0/L1 generation (background)
│   │   │   └── finalize.go        # Move temp → resources/; update status
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go  # FSClient, SearchClient, VLMClient, TaskStore, EventPublisher
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   ├── source/
│   │   │   ├── git_adapter.go     # git clone, diff detection
│   │   │   ├── http_adapter.go    # Web scraping + HTML→Markdown
│   │   │   └── local_adapter.go   # Hardlink/copy local files
│   │   ├── store/
│   │   │   ├── task_redis.go      # Background task state in Redis
│   │   │   └── resource_postgres.go # Resource metadata in PostgreSQL
│   │   ├── client/
│   │   │   ├── fs_client.go
│   │   │   ├── search_client.go
│   │   │   └── vlm_client.go
│   │   └── event/
│   │       ├── publisher.go       # ov.resource.ingested
│   │       └── subscriber.go     # admin.account.created
│   └── infra/
```

---

## 3. Domain Model

```go
// internal/domain/resource.go

type SourceType string
const (
    SourceTypeGit       SourceType = "git"
    SourceTypeHTTP      SourceType = "http"
    SourceTypeLocalDir  SourceType = "local_dir"
    SourceTypeLocalFile SourceType = "local_file"
)

type ResourceStatus string
const (
    ResourceStatusQueued     ResourceStatus = "queued"
    ResourceStatusProcessing ResourceStatus = "processing"
    ResourceStatusDone       ResourceStatus = "done"
    ResourceStatusFailed     ResourceStatus = "failed"
)

type Resource struct {
    ID          string
    AccountID   string
    UserID      string
    Name        string          // Directory name in VikingFS
    SourceURL   string          // Source git URL, HTTP URL, or local path
    SourceType  SourceType
    Status      ResourceStatus
    VikingURI   string          // viking://resources/{name}/
    WatchConfig *WatchConfig
    FileCount   int
    ErrorMsg    string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type WatchConfig struct {
    Enabled    bool
    Interval   time.Duration
    LastCommit string  // For git: last processed commit hash
    LastETag   string  // For HTTP: ETag header value
}

// internal/domain/task.go
type TaskState string
const (
    TaskStateQueued     TaskState = "queued"
    TaskStateProcessing TaskState = "processing"
    TaskStateDone       TaskState = "done"
    TaskStateFailed     TaskState = "failed"
)

type BackgroundTask struct {
    ID         string
    AccountID  string
    TaskType   string    // "resource_ingest" | "resource_refresh" | "skill_load"
    ResourceID string
    State      TaskState
    Progress   float64   // 0.0 - 1.0
    ErrorMsg   string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

---

## 4. Ingestion Pipeline — 7 Steps

### 4.1 AddResourceUseCase — Orchestrator

```go
// internal/usecase/add_resource.go

func (uc *AddResourceUseCase) Execute(ctx context.Context, req dto.AddResourceRequest) (*dto.AddResourceResponse, error) {
    // Create resource record
    resource := &domain.Resource{
        ID:        uuid.New().String(),
        AccountID: req.AccountID,
        UserID:    req.UserID,
        Name:      req.Name,
        SourceURL: req.URLOrPath,
        Status:    domain.ResourceStatusQueued,
        VikingURI: fmt.Sprintf("viking://resources/%s/", req.Name),
    }
    uc.resourceRepo.Save(ctx, resource)
    
    // Create background task
    task := &domain.BackgroundTask{
        ID:         uuid.New().String(),
        AccountID:  req.AccountID,
        TaskType:   "resource_ingest",
        ResourceID: resource.ID,
        State:      domain.TaskStateQueued,
    }
    uc.taskStore.Save(ctx, task)
    
    // Run ingestion in background
    go uc.runIngestion(context.Background(), resource, task, req)
    
    return &dto.AddResourceResponse{
        ResourceID: resource.ID,
        TaskID:     task.ID,
        VikingURI:  resource.VikingURI,
    }, nil
}

func (uc *AddResourceUseCase) runIngestion(ctx context.Context, resource *domain.Resource, task *domain.BackgroundTask, req dto.AddResourceRequest) {
    defer func() {
        if r := recover(); r != nil {
            uc.updateStatus(ctx, resource, task, domain.ResourceStatusFailed, fmt.Sprintf("panic: %v", r), 0)
        }
    }()
    
    uc.updateStatus(ctx, resource, task, domain.ResourceStatusProcessing, "", 0.0)
    
    // Step 1: Source Detection
    sourceType := uc.pipeline.DetectSource(req.URLOrPath)
    resource.SourceType = sourceType
    
    // Step 2: Clone/Download
    tempURI, err := uc.pipeline.CloneOrDownload(ctx, req.URLOrPath, sourceType, resource.Name)
    if err != nil {
        uc.updateStatus(ctx, resource, task, domain.ResourceStatusFailed, err.Error(), 0.0)
        return
    }
    uc.updateStatus(ctx, resource, task, domain.ResourceStatusProcessing, "", 0.15)
    
    // Step 3: Scan Directory
    files, err := uc.pipeline.ScanDirectory(ctx, tempURI, sourceType)
    if err != nil {
        uc.updateStatus(ctx, resource, task, domain.ResourceStatusFailed, err.Error(), 0.15)
        return
    }
    uc.updateStatus(ctx, resource, task, domain.ResourceStatusProcessing, "", 0.25)
    
    // Step 4: Parse files (goroutine pool)
    parsedFiles, err := uc.pipeline.ParseFiles(ctx, files, uc.parseRegistry)
    if err != nil {
        uc.updateStatus(ctx, resource, task, domain.ResourceStatusFailed, err.Error(), 0.25)
        return
    }
    uc.updateStatus(ctx, resource, task, domain.ResourceStatusProcessing, "", 0.50)
    
    // Step 5: Build FS structure (write to VikingFS temp location)
    if err := uc.pipeline.BuildFS(ctx, tempURI, parsedFiles, req.AccountID); err != nil {
        uc.updateStatus(ctx, resource, task, domain.ResourceStatusFailed, err.Error(), 0.50)
        return
    }
    uc.updateStatus(ctx, resource, task, domain.ResourceStatusProcessing, "", 0.75)
    
    // Step 6: Background L0/L1 Generation (if enabled)
    if req.GenerateSummaries {
        // Non-blocking: failures here don't fail the resource
        go uc.pipeline.GenerateSummaries(context.Background(), parsedFiles, req.AccountID, uc.vlmClient)
    }
    
    // Step 7: Finalize — Move temp → resources/
    finalURI := fmt.Sprintf("viking://resources/%s/", resource.Name)
    if err := uc.pipeline.Finalize(ctx, tempURI, finalURI); err != nil {
        uc.updateStatus(ctx, resource, task, domain.ResourceStatusFailed, err.Error(), 0.75)
        return
    }
    
    // Update resource watch config if requested
    if req.Watch {
        resource.WatchConfig = &domain.WatchConfig{
            Enabled:  true,
            Interval: req.WatchInterval,
        }
        // For git: record current HEAD commit hash
        if sourceType == domain.SourceTypeGit {
            resource.WatchConfig.LastCommit = uc.gitAdapter.GetHEAD(req.URLOrPath)
        }
    }
    
    resource.FileCount = len(parsedFiles)
    uc.updateStatus(ctx, resource, task, domain.ResourceStatusDone, "", 1.0)
    
    // Publish NATS event
    uc.publisher.PublishResourceIngested(ctx, port.ResourceIngestedPayload{
        ResourceID: resource.ID,
        Name:       resource.Name,
        URI:        finalURI,
        FileCount:  len(parsedFiles),
    })
}
```

### 4.2 Source Detection

```go
// internal/usecase/pipeline/detect_source.go

func (p *Pipeline) DetectSource(urlOrPath string) domain.SourceType {
    // Git: github.com, gitlab.com, gitee.com, or ends in .git
    if gitPattern.MatchString(urlOrPath) {
        return domain.SourceTypeGit
    }
    
    // HTTP: starts with http:// or https://
    if strings.HasPrefix(urlOrPath, "http://") || strings.HasPrefix(urlOrPath, "https://") {
        return domain.SourceTypeHTTP
    }
    
    // Local: check if it's a file or directory
    info, err := os.Stat(urlOrPath)
    if err == nil {
        if info.IsDir() {
            return domain.SourceTypeLocalDir
        }
        return domain.SourceTypeLocalFile
    }
    
    return domain.SourceTypeHTTP  // Default fallback
}

var gitPattern = regexp.MustCompile(`(github\.com|gitlab\.com|gitee\.com|bitbucket\.org)(/[\w\-]+){2,}(\.git)?$`)
```

### 4.3 Git Clone Adapter

```go
// adapter/source/git_adapter.go

type GitAdapter struct {
    tempDir string  // ~/.openviking/temp
    timeout time.Duration
}

func (a *GitAdapter) Clone(ctx context.Context, repoURL, name string) (localPath string, err error) {
    destPath := filepath.Join(a.tempDir, name)
    
    // Shallow clone (depth=1) for faster download
    cmd := exec.CommandContext(ctx, "git", "clone",
        "--depth=1",
        "--single-branch",
        repoURL,
        destPath,
    )
    cmd.Timeout = a.timeout  // default: 300s
    
    if output, err := cmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("git clone failed: %s: %w", output, err)
    }
    return destPath, nil
}

func (a *GitAdapter) HasNewCommits(repoURL, lastCommit string) (bool, string, error) {
    // git ls-remote <url> HEAD
    cmd := exec.Command("git", "ls-remote", repoURL, "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return false, "", err
    }
    currentHEAD := strings.Fields(string(output))[0]
    return currentHEAD != lastCommit, currentHEAD, nil
}
```

### 4.4 HTTP Scraper Adapter

```go
// adapter/source/http_adapter.go

type HTTPAdapter struct {
    client *http.Client
    maxSizeMB int
}

func (a *HTTPAdapter) Download(ctx context.Context, url, name string) (string, error) {
    resp, err := a.client.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    // Read body (with size limit)
    body, err := io.ReadAll(io.LimitReader(resp.Body, int64(a.maxSizeMB)*1024*1024))
    
    contentType := resp.Header.Get("Content-Type")
    
    var content string
    if strings.Contains(contentType, "text/html") {
        // HTML → Markdown conversion
        content = htmlToMarkdown(body)
    } else {
        content = string(body)
    }
    
    // Write to temp directory
    destPath := filepath.Join(a.tempDir, name, "index.md")
    os.MkdirAll(filepath.Dir(destPath), 0755)
    os.WriteFile(destPath, []byte(content), 0644)
    return filepath.Dir(destPath), nil
}

func (a *HTTPAdapter) HasChanged(url, lastETag string) (bool, string, error) {
    req, _ := http.NewRequest("HEAD", url, nil)
    resp, err := a.client.Do(req)
    if err != nil {
        return false, "", err
    }
    defer resp.Body.Close()
    
    currentETag := resp.Header.Get("ETag")
    if currentETag == "" {
        // Fallback: Last-Modified
        currentETag = resp.Header.Get("Last-Modified")
    }
    return currentETag != lastETag, currentETag, nil
}

func htmlToMarkdown(html []byte) string {
    // Use html-to-markdown library
    // github.com/JohannesKaufmann/html-to-markdown
    converter := md.NewConverter("", true, nil)
    markdown, _ := converter.ConvertBytes(html)
    return string(markdown)
}
```

### 4.5 Directory Scanner (.gitignore aware)

```go
// internal/usecase/pipeline/scan_directory.go

var defaultIgnorePatterns = []string{
    "*.pyc", "__pycache__", ".git", "node_modules", ".venv",
    "*.min.js", "*.min.css", "vendor", "dist", "build",
    ".DS_Store", "*.lock", "go.sum",
}

func (p *Pipeline) ScanDirectory(ctx context.Context, rootPath string, sourceType domain.SourceType) ([]domain.FileEntry, error) {
    // Load .gitignore rules if git repo
    var ignoreRules []gitignore.Pattern
    if sourceType == domain.SourceTypeGit {
        gitignorePath := filepath.Join(rootPath, ".gitignore")
        if content, err := os.ReadFile(gitignorePath); err == nil {
            ignoreRules = gitignore.Parse(content)
        }
    }
    
    // Add default ignore patterns
    for _, pattern := range defaultIgnorePatterns {
        ignoreRules = append(ignoreRules, gitignore.ParsePattern(pattern, nil))
    }
    
    var files []domain.FileEntry
    err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return nil  // Skip inaccessible files
        }
        
        relPath := strings.TrimPrefix(path, rootPath)
        
        // Check ignore rules
        for _, rule := range ignoreRules {
            if rule.Match(relPath, d.IsDir()) {
                if d.IsDir() {
                    return filepath.SkipDir
                }
                return nil
            }
        }
        
        if !d.IsDir() {
            info, _ := d.Info()
            files = append(files, domain.FileEntry{
                LocalPath: path,
                RelPath:   relPath,
                Extension: strings.ToLower(filepath.Ext(path)),
                Size:      info.Size(),
            })
        }
        return nil
    })
    
    return files, err
}
```

### 4.6 Parse Files — Goroutine Pool

```go
// internal/usecase/pipeline/parse_files.go

func (p *Pipeline) ParseFiles(ctx context.Context, files []domain.FileEntry, registry *parse.Registry) ([]*domain.ParsedFile, error) {
    type job struct{ file domain.FileEntry }
    type result struct {
        parsed *domain.ParsedFile
        err    error
    }
    
    jobs    := make(chan job, len(files))
    results := make(chan result, len(files))
    
    // Start worker pool (max 5 workers)
    for i := 0; i < 5; i++ {
        go func() {
            for j := range jobs {
                content, err := os.ReadFile(j.file.LocalPath)
                if err != nil {
                    results <- result{err: err}
                    continue
                }
                
                parser := registry.ParserFor(j.file.Extension)
                if parser == nil {
                    // Unknown extension: treat as raw text
                    parser = registry.TextParser()
                }
                
                chunks, err := parser.Parse(ctx, j.file.LocalPath, content)
                results <- result{
                    parsed: &domain.ParsedFile{
                        FileEntry: j.file,
                        Content:   string(content),
                        Chunks:    chunks,
                        Language:  detectLanguage(j.file.Extension),
                    },
                    err: err,
                }
            }
        }()
    }
    
    // Feed jobs
    for _, file := range files {
        jobs <- job{file}
    }
    close(jobs)
    
    // Collect results
    var parsedFiles []*domain.ParsedFile
    for range files {
        r := <-results
        if r.err != nil {
            slog.Warn("parse failed", "path", r.parsed.LocalPath, "error", r.err)
            // Don't fail ingestion on parse error — continue
            continue
        }
        parsedFiles = append(parsedFiles, r.parsed)
    }
    
    return parsedFiles, nil
}
```

### 4.7 L0/L1 Summary Generation

```go
// internal/usecase/pipeline/generate_summaries.go

func (p *Pipeline) GenerateSummaries(ctx context.Context, parsedFiles []*domain.ParsedFile, accountID string, vlm port.VLMClient) {
    sem := make(chan struct{}, 2)  // Max 2 VLM calls concurrently
    
    for _, file := range parsedFiles {
        if len(file.Content) == 0 {
            continue
        }
        
        sem <- struct{}{}
        go func(f *domain.ParsedFile) {
            defer func() { <-sem }()
            
            // L0: Abstract (~100 tokens)
            abstractPrompt := fmt.Sprintf(
                "Generate a one-sentence summary (max 100 tokens) of this file:\n\nFile: %s\n\nContent:\n%s",
                f.RelPath, truncate(f.Content, 2000))
            
            abstract, err := vlm.Generate(ctx, abstractPrompt, adapters.WithVLMMaxTokens(100))
            if err == nil {
                abstractURI := f.VikingURI + ".abstract.md"
                p.fsClient.WriteRaw(ctx, abstractURI, []byte(abstract))
                // FS auto-emits ov.content.written → L0 indexed in Search
            }
            
            // L1: Overview (~2K tokens)
            overviewPrompt := fmt.Sprintf(
                "Generate a detailed overview (max 2000 tokens) covering the purpose, key concepts, and usage of:\n\nFile: %s\n\nContent:\n%s",
                f.RelPath, truncate(f.Content, 8000))
            
            overview, err := vlm.Generate(ctx, overviewPrompt, adapters.WithVLMMaxTokens(2000))
            if err == nil {
                overviewURI := f.VikingURI + ".overview.md"
                p.fsClient.WriteRaw(ctx, overviewURI, []byte(overview))
                // FS auto-emits ov.content.written → L1 indexed in Search
            }
        }(file)
    }
    
    // Wait for all VLM goroutines to complete
    for i := 0; i < cap(sem); i++ {
        sem <- struct{}{}
    }
}

// Directory-level abstract generation
func (p *Pipeline) GenerateDirectoryAbstract(ctx context.Context, dirURI string, treeStr string, vlm port.VLMClient) {
    prompt := fmt.Sprintf(
        "Generate a one-sentence description of this directory based on its structure:\n\n%s",
        treeStr)
    abstract, err := vlm.Generate(ctx, prompt, adapters.WithVLMMaxTokens(100))
    if err == nil {
        p.fsClient.WriteRaw(ctx, dirURI+".abstract.md", []byte(abstract))
    }
}
```

---

## 5. Watch Manager

```go
// internal/usecase/watch_resources.go

type WatchManager struct {
    ticker      *time.Ticker
    resourceRepo port.ResourceRepo
    gitAdapter  *source.GitAdapter
    httpAdapter *source.HTTPAdapter
    taskQueue   chan *domain.BackgroundTask
    refreshUC   *RefreshResourceUseCase
}

func (wm *WatchManager) Start(ctx context.Context) {
    wm.ticker = time.NewTicker(5 * time.Minute)  // Check interval: 5 min
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
    resources, _ := wm.resourceRepo.ListWatched(ctx)
    
    for _, resource := range resources {
        if !resource.WatchConfig.Enabled {
            continue
        }
        
        var hasChanges bool
        var newCheckpoint string
        
        switch resource.SourceType {
        case domain.SourceTypeGit:
            hasChanges, newCheckpoint, _ = wm.gitAdapter.HasNewCommits(
                resource.SourceURL, resource.WatchConfig.LastCommit)
        case domain.SourceTypeHTTP:
            hasChanges, newCheckpoint, _ = wm.httpAdapter.HasChanged(
                resource.SourceURL, resource.WatchConfig.LastETag)
        }
        
        if hasChanges {
            slog.Info("resource has updates", "resource", resource.Name, "new_checkpoint", newCheckpoint)
            go wm.refreshUC.Execute(context.Background(), dto.RefreshResourceRequest{
                ResourceID: resource.ID,
                AccountID:  resource.AccountID,
            })
            // Update checkpoint
            resource.WatchConfig.LastCommit = newCheckpoint
            resource.WatchConfig.LastETag   = newCheckpoint
            wm.resourceRepo.Update(context.Background(), resource)
        }
    }
}
```

---

## 6. Skill Loading

```go
// internal/usecase/load_skills.go

// Skill directory structure in VikingFS:
// viking://agent/{account_id}/{agent_id}/skills/
//   ├── web_search/
//   │   ├── .abstract.md      (L0: "Search the web for information")
//   │   ├── .overview.md      (L1: full description + examples)
//   │   └── instructions.md   (L2: detailed implementation guide)
//   └── code_executor/
//       └── ...

func (uc *LoadSkillsUseCase) Execute(ctx context.Context, req dto.LoadSkillsRequest) (*dto.LoadSkillsResponse, error) {
    skillsURI := fmt.Sprintf("viking://agent/%s/%s/skills/", req.AccountID, req.AgentID)
    
    // List skill directories
    entries, err := uc.fsClient.Ls(ctx, skillsURI)
    if err != nil {
        return nil, err
    }
    
    var skills []*domain.Skill
    for _, entry := range entries {
        if !entry.IsDirectory {
            continue
        }
        
        // Read L0 abstract (skill description)
        abstract, _ := uc.fsClient.ReadRaw(ctx, entry.URI+".abstract.md")
        
        // Read L2 instructions
        instructions, _ := uc.fsClient.ReadRaw(ctx, entry.URI+"instructions.md")
        
        skill := &domain.Skill{
            ID:           uuid.New().String(),
            AgentID:      req.AgentID,
            Name:         entry.Name,
            Description:  string(abstract),
            Instructions: string(instructions),
            VikingURI:    entry.URI,
        }
        skills = append(skills, skill)
        
        // Ensure L0/L1 are indexed in Search
        // (FS already emits ov.content.written when files written)
    }
    
    return &dto.LoadSkillsResponse{
        Skills:  skills,
        AgentID: req.AgentID,
    }, nil
}
```

---

## 7. Task Store (Redis)

```go
// adapter/store/task_redis.go

// Key: "ov_task:{account_id}:{task_id}"
// TTL: 7 days

func (s *TaskRedisStore) Save(ctx context.Context, task *domain.BackgroundTask) error {
    data, _ := json.Marshal(task)
    key := fmt.Sprintf("ov_task:%s:%s", task.AccountID, task.ID)
    return s.redis.Set(ctx, key, data, 7*24*time.Hour).Err()
}

func (s *TaskRedisStore) UpdateProgress(ctx context.Context, taskID, accountID string, progress float64, state domain.TaskState) error {
    key := fmt.Sprintf("ov_task:%s:%s", accountID, taskID)
    data, err := s.redis.Get(ctx, key).Bytes()
    if err != nil {
        return err
    }
    var task domain.BackgroundTask
    json.Unmarshal(data, &task)
    task.Progress  = progress
    task.State     = state
    task.UpdatedAt = time.Now()
    newData, _ := json.Marshal(task)
    return s.redis.Set(ctx, key, newData, 7*24*time.Hour).Err()
}
```

---

## 8. gRPC Service

```protobuf
syntax = "proto3";
package openviking.resource.v1;

service ResourceService {
  // Resources
  rpc AddResource(AddResourceRequest) returns (AddResourceResponse);
  rpc GetResource(GetResourceRequest) returns (GetResourceResponse);
  rpc ListResources(ListResourcesRequest) returns (ListResourcesResponse);
  rpc DeleteResource(DeleteResourceRequest) returns (DeleteResourceResponse);
  rpc RefreshResource(RefreshResourceRequest) returns (RefreshResourceResponse);
  rpc GetResourceStatus(GetResourceStatusRequest) returns (GetResourceStatusResponse);

  // Skills
  rpc LoadSkills(LoadSkillsRequest) returns (LoadSkillsResponse);
  rpc ListSkills(ListSkillsRequest) returns (ListSkillsResponse);

  // Tasks
  rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
}

message AddResourceRequest {
  string account_id        = 1;
  string user_id           = 2;
  string url_or_path       = 3;
  string name              = 4;
  bool   generate_summaries = 5;  // default: true
  bool   watch              = 6;
  string watch_interval     = 7;  // "1h" | "24h"
}

message AddResourceResponse {
  string resource_id = 1;
  string task_id     = 2;
  string viking_uri  = 3;
}

message GetTaskResponse {
  string    id         = 1;
  string    task_type  = 2;
  string    state      = 3;
  double    progress   = 4;  // 0.0 - 1.0
  string    error_msg  = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
}
```

---

## 9. Configuration

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
    task_ttl: 168h  # 7 days

  nats:
    url: "nats://nats:4222"
    stream: "openviking"

  database:
    url: "${DATABASE_URL}"
```

---

## 10. Testing Strategy

### Unit Tests
- `TestDetectSource_GitHub` — github.com URL → SourceTypeGit
- `TestDetectSource_HTTP` — https:// URL → SourceTypeHTTP
- `TestDetectSource_LocalDir` — existing directory → SourceTypeLocalDir
- `TestScanDirectory_GitignoreApplied` — .gitignore `node_modules/` → excluded
- `TestScanDirectory_DefaultIgnore` — `*.pyc` files → excluded
- `TestParseFiles_GoFile` — tree-sitter splits at function boundaries
- `TestParseFiles_IgnoresUnknownExtension` — uses TextParser as fallback
- `TestParseFiles_PoolBounded` — 100 files, max 5 goroutines
- `TestWatchManager_GitNewCommits` — new commit → refresh triggered
- `TestWatchManager_HTTPETagChanged` — ETag differs → refresh triggered
- `TestGenerateSummaries_VLMFails_ResourceContinues` — VLM error → status=DONE (not FAILED)
- `TestLoadSkills_ReadsAbstractAndInstructions`

### Integration Tests
- `TestAddResourceGitE2E` — real git clone → files in VikingFS → task=DONE
- `TestAddResourceHTTPE2E` — real HTTP download → markdown written
- `TestDeleteResourceCleansFS` — delete → viking://resources/{name}/ removed; NATS event fired

---

## 11. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| Large repo (100K files) → goroutine pool insufficient | Trung bình | Pool size 5; each file processed sequentially in goroutine → memory O(pool_size × file_content) |
| VLM L0/L1 generation cost ($) | Trung bình | `generate_summaries=false` default; only enable explicitly |
| Git clone timeout (large repo) | Thấp | `--depth=1` + 300s timeout; report failure in task |
| HTML scraping blocked by anti-bot | Thấp | Custom User-Agent; playwright fallback (future enhancement) |
| Temp directory disk space | Trung bình | Clean temp after successful finalize; `os.RemoveAll(tempDir)` |
| Watch interval too frequent → flood ingestion | Thấp | Min interval enforced: 5min; max_watched=100 |
