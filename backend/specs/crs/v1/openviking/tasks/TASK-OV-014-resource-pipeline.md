# TASK-OV-014 — `services/openviking-resource` Ingestion Pipeline

**Wave:** 5 (Context — song song với Session)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-009 (fs client), TASK-OV-004 (pkg/parse)  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-OV-005 §3, §4](../solutions/SOL-OV-005-Resource-Service.md)  
**Port gRPC:** 9014

---

## Mục tiêu

Tạo phần cốt lõi của `services/openviking-resource/` — 7-step ingestion pipeline: source detection, git clone / HTTP download, directory scanning (.gitignore aware), file parsing (goroutine pool), VikingFS building, L0/L1 VLM summary generation, finalize.

---

## Cấu trúc thư mục

```
services/openviking-resource/
├── cmd/server/main.go
├── api/proto/resource/v1/resource.proto
├── internal/
│   ├── domain/
│   │   ├── resource.go        # Resource, SourceType, ResourceStatus, WatchConfig
│   │   ├── skill.go           # Skill
│   │   └── task.go            # BackgroundTask, TaskState
│   ├── usecase/
│   │   ├── add_resource.go    # Orchestrate 7-step pipeline
│   │   ├── refresh_resource.go
│   │   ├── delete_resource.go
│   │   ├── load_skills.go
│   │   ├── pipeline/
│   │   │   ├── detect_source.go
│   │   │   ├── clone_download.go
│   │   │   ├── scan_directory.go
│   │   │   ├── parse_files.go
│   │   │   ├── build_fs.go
│   │   │   └── generate_summaries.go
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go      # FSClient, VLMClient, TaskStore, ResourceRepo, EventPublisher
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   ├── source/
│   │   │   ├── git_adapter.go
│   │   │   └── http_adapter.go
│   │   ├── store/
│   │   │   ├── task_redis.go
│   │   │   └── resource_postgres.go
│   │   └── event/publisher.go
│   └── infra/
```

---

## 1. Domain Models

**File: `internal/domain/resource.go`**

```go
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
    Name        string          // Directory name in VikingFS (must be DNS-safe)
    SourceURL   string
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
    LastCommit string  // Git HEAD hash
    LastETag   string  // HTTP ETag
}

// Parsed file during ingestion (internal to pipeline)
type ParsedFile struct {
    LocalPath string
    RelPath   string          // Relative to repo root
    Extension string
    Size      int64
    VikingURI string          // Target: viking://resources/{name}/{relPath}
    Content   string          // Decoded text content
    Chunks    []parsekg.Chunk // From pkg/parse
    Language  string          // Detected language (for code files)
}
```

---

## 2. Pipeline Steps

### Step 1: Detect Source

**File: `internal/usecase/pipeline/detect_source.go`**

```go
var gitPattern = regexp.MustCompile(`(github\.com|gitlab\.com|gitee\.com|bitbucket\.org)(/[\w\-\.]+){2,}(\.git)?$`)

func DetectSource(urlOrPath string) domain.SourceType {
    if gitPattern.MatchString(urlOrPath) { return domain.SourceTypeGit }
    if strings.HasPrefix(urlOrPath, "http://") || strings.HasPrefix(urlOrPath, "https://") {
        return domain.SourceTypeHTTP
    }
    if info, err := os.Stat(urlOrPath); err == nil {
        if info.IsDir() { return domain.SourceTypeLocalDir }
        return domain.SourceTypeLocalFile
    }
    return domain.SourceTypeHTTP  // Fallback
}
```

### Step 2: Clone/Download

**File: `internal/adapter/source/git_adapter.go`**

```go
type GitAdapter struct {
    tempDir string         // e.g., ~/.openviking/temp
    timeout time.Duration  // default: 300s
}

func (a *GitAdapter) Clone(ctx context.Context, repoURL, name string) (localPath string, err error) {
    destPath := filepath.Join(a.tempDir, name)
    if err := os.RemoveAll(destPath); err != nil { return "", err }  // Clean slate
    
    cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", "--single-branch", repoURL, destPath)
    if out, err := cmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("git clone: %s: %w", out, err)
    }
    return destPath, nil
}

func (a *GitAdapter) GetHEAD(localPath string) string {
    out, err := exec.Command("git", "-C", localPath, "rev-parse", "HEAD").Output()
    if err != nil { return "" }
    return strings.TrimSpace(string(out))
}

func (a *GitAdapter) HasNewCommits(repoURL, lastCommit string) (bool, string, error) {
    out, err := exec.Command("git", "ls-remote", repoURL, "HEAD").Output()
    if err != nil { return false, "", err }
    parts := strings.Fields(string(out))
    if len(parts) == 0 { return false, "", fmt.Errorf("no output") }
    currentHEAD := parts[0]
    return currentHEAD != lastCommit, currentHEAD, nil
}
```

**File: `internal/adapter/source/http_adapter.go`**

```go
type HTTPAdapter struct {
    client    *http.Client
    tempDir   string
    maxSizeMB int  // default: 100MB
    userAgent string
}

func (a *HTTPAdapter) Download(ctx context.Context, url, name string) (string, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("User-Agent", a.userAgent)
    
    resp, err := a.client.Do(req)
    if err != nil { return "", err }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(io.LimitReader(resp.Body, int64(a.maxSizeMB)*1024*1024))
    if err != nil { return "", err }
    
    contentType := resp.Header.Get("Content-Type")
    content := string(body)
    if strings.Contains(contentType, "text/html") {
        content = htmlToMarkdown(body)
    }
    
    destDir := filepath.Join(a.tempDir, name)
    os.MkdirAll(destDir, 0755)
    destFile := filepath.Join(destDir, "index.md")
    if err := os.WriteFile(destFile, []byte(content), 0644); err != nil { return "", err }
    return destDir, nil
}

func (a *HTTPAdapter) HasChanged(url, lastETag string) (bool, string, error) {
    req, _ := http.NewRequest("HEAD", url, nil)
    resp, err := a.client.Do(req)
    if err != nil { return false, "", err }
    defer resp.Body.Close()
    
    newETag := resp.Header.Get("ETag")
    if newETag == "" { newETag = resp.Header.Get("Last-Modified") }
    return newETag != lastETag, newETag, nil
}

func htmlToMarkdown(html []byte) string {
    // github.com/JohannesKaufmann/html-to-markdown
    converter := md.NewConverter("", true, nil)
    result, _ := converter.ConvertBytes(html)
    return string(result)
}
```

### Step 3: Scan Directory

**File: `internal/usecase/pipeline/scan_directory.go`**

```go
var defaultIgnorePatterns = []string{
    "*.pyc", "__pycache__", ".git", "node_modules", ".venv", "venv",
    "*.min.js", "*.min.css", "vendor", "dist", "build", "target",
    ".DS_Store", "*.lock", "go.sum", "package-lock.json", "yarn.lock",
    ".idea", ".vscode", "*.log", "*.tmp",
}

func ScanDirectory(ctx context.Context, rootPath string, sourceType domain.SourceType) ([]domain.ParsedFile, error) {
    // Load .gitignore if git repo
    var matcher gitignore.GitIgnore
    if sourceType == domain.SourceTypeGit {
        matcher, _ = gitignore.NewFromFile(filepath.Join(rootPath, ".gitignore"))
    }
    
    var files []domain.ParsedFile
    err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil { return nil }
        
        relPath, _ := filepath.Rel(rootPath, path)
        if relPath == "." { return nil }
        
        // Check gitignore
        if matcher != nil && matcher.MatchesPath(relPath) {
            if d.IsDir() { return filepath.SkipDir }
            return nil
        }
        
        // Check default patterns
        for _, pattern := range defaultIgnorePatterns {
            if matched, _ := filepath.Match(pattern, d.Name()); matched {
                if d.IsDir() { return filepath.SkipDir }
                return nil
            }
        }
        
        if !d.IsDir() {
            info, _ := d.Info()
            if info.Size() > 10*1024*1024 { return nil }  // Skip files > 10MB
            files = append(files, domain.ParsedFile{
                LocalPath: path,
                RelPath:   filepath.ToSlash(relPath),
                Extension: strings.ToLower(filepath.Ext(path)),
                Size:      info.Size(),
            })
        }
        return nil
    })
    return files, err
}
```

### Step 4: Parse Files

**File: `internal/usecase/pipeline/parse_files.go`**

```go
const parseWorkers = 5  // goroutine pool size

func ParseFiles(ctx context.Context, files []domain.ParsedFile, registry *parsekg.Registry) ([]*domain.ParsedFile, error) {
    type job struct{ file *domain.ParsedFile }
    type result struct {
        parsed *domain.ParsedFile
        err    error
    }
    
    jobs    := make(chan job, len(files))
    results := make(chan result, len(files))
    
    for i := 0; i < parseWorkers; i++ {
        go func() {
            for j := range jobs {
                content, err := os.ReadFile(j.file.LocalPath)
                if err != nil { results <- result{err: err}; continue }
                
                ext := j.file.Extension
                parser := registry.ParserFor(ext)
                if parser == nil { parser = registry.TextParser() }
                
                chunks, _ := parser.Parse(ctx, j.file.LocalPath, content)
                j.file.Content   = string(content)
                j.file.Chunks    = chunks
                j.file.Language  = detectLanguage(ext)
                results <- result{parsed: j.file}
            }
        }()
    }
    
    for i := range files { jobs <- job{&files[i]} }
    close(jobs)
    
    var parsed []*domain.ParsedFile
    for range files {
        r := <-results
        if r.err != nil || r.parsed == nil { continue }  // Skip on error
        parsed = append(parsed, r.parsed)
    }
    return parsed, nil
}
```

### Step 5: Build FS

**File: `internal/usecase/pipeline/build_fs.go`**

```go
func BuildFS(ctx context.Context, resourceName string, files []*domain.ParsedFile, accountID string, fsClient port.FSClient) error {
    // Temp URI: viking://temp/{accountID}/{resourceName}/
    tempBaseURI := fmt.Sprintf("viking://temp/%s/%s/", accountID, resourceName)
    
    // Create base directory
    fsClient.Mkdir(ctx, tempBaseURI, true)
    
    g, gCtx := errgroup.WithContext(ctx)
    sem := make(chan struct{}, 5)
    
    for _, file := range files {
        file := file
        g.Go(func() error {
            sem <- struct{}{}
            defer func() { <-sem }()
            
            targetURI := tempBaseURI + file.RelPath
            // Set VikingURI on file for later reference
            file.VikingURI = fmt.Sprintf("viking://resources/%s/%s", resourceName, file.RelPath)
            
            return fsClient.Write(gCtx, targetURI, []byte(file.Content), accountID)
        })
    }
    return g.Wait()
}
```

### Step 6: Generate L0/L1 Summaries (Background)

**File: `internal/usecase/pipeline/generate_summaries.go`**

```go
const summaryWorkers = 2  // VLM concurrent limit

func GenerateSummaries(ctx context.Context, files []*domain.ParsedFile, accountID string, fsClient port.FSClient, vlmClient port.VLMClient) {
    sem := make(chan struct{}, summaryWorkers)
    var wg sync.WaitGroup
    
    for _, file := range files {
        if len(file.Content) == 0 || !shouldGenerateSummary(file.Extension) { continue }
        
        wg.Add(1)
        file := file
        go func() {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            
            // L0: Abstract (~100 tokens)
            abstractPrompt := fmt.Sprintf(
                "Generate a one-sentence summary (max 100 tokens) for:\n\nFile: %s\n\n%s",
                file.RelPath, truncate(file.Content, 2000))
            
            abstract, err := vlmClient.Generate(ctx, abstractPrompt, vlm.WithVLMMaxTokens(100))
            if err == nil {
                abstractURI := file.VikingURI + ".abstract.md"
                fsClient.Write(ctx, abstractURI, []byte(abstract), accountID)
                // FS emits ov.content.written → Search indexes L0
            }
            
            // L1: Overview (~2K tokens)
            overviewPrompt := fmt.Sprintf(
                "Generate a detailed technical overview (max 2000 tokens) covering purpose, key concepts, usage for:\n\nFile: %s\n\n%s",
                file.RelPath, truncate(file.Content, 8000))
            
            overview, err := vlmClient.Generate(ctx, overviewPrompt, vlm.WithVLMMaxTokens(2000))
            if err == nil {
                overviewURI := file.VikingURI + ".overview.md"
                fsClient.Write(ctx, overviewURI, []byte(overview), accountID)
            }
        }()
    }
    wg.Wait()
}

func shouldGenerateSummary(ext string) bool {
    summaryExtensions := map[string]bool{
        ".go": true, ".py": true, ".ts": true, ".js": true, ".rs": true,
        ".java": true, ".md": true, ".txt": true, ".yaml": true, ".json": true,
    }
    return summaryExtensions[ext]
}

func truncate(s string, maxBytes int) string {
    if len(s) <= maxBytes { return s }
    return s[:maxBytes] + "... [truncated]"
}
```

### Step 7: Finalize

**File: `internal/usecase/pipeline/finalize.go`**

```go
func Finalize(ctx context.Context, tempURI, finalURI string, fsClient port.FSClient) error {
    // Move temp → final location
    // viking://temp/{account}/{name}/ → viking://resources/{name}/
    return fsClient.Mv(ctx, tempURI, finalURI)
}
```

---

## 3. AddResourceUseCase — Orchestrator

**File: `internal/usecase/add_resource.go`**

```go
type AddResourceUseCase struct {
    resourceRepo port.ResourceRepo
    taskStore    port.TaskStore
    gitAdapter   *source.GitAdapter
    httpAdapter  *source.HTTPAdapter
    parseRegistry *parsekg.Registry
    fsClient     port.FSClient
    vlmClient    port.VLMClient
    publisher    port.EventPublisher
    config       *Config
}

func (uc *AddResourceUseCase) Execute(ctx context.Context, req AddResourceRequest) (*AddResourceResponse, error) {
    // 1. Create Resource record (status=queued)
    resource := &domain.Resource{
        ID:        uuid.New().String(),
        AccountID: req.AccountID, UserID: req.UserID,
        Name: req.Name, SourceURL: req.URLOrPath,
        Status: domain.ResourceStatusQueued,
        VikingURI: fmt.Sprintf("viking://resources/%s/", req.Name),
    }
    uc.resourceRepo.Save(ctx, resource)
    
    // 2. Create BackgroundTask
    task := &domain.BackgroundTask{
        ID: uuid.New().String(), AccountID: req.AccountID,
        TaskType: "resource_ingest", ResourceID: resource.ID,
        State: domain.TaskStateQueued,
    }
    uc.taskStore.Save(ctx, task)
    
    // 3. Run ingestion async
    go uc.runIngestion(context.Background(), resource, task, req)
    
    return &AddResourceResponse{
        ResourceID: resource.ID, TaskID: task.ID, VikingURI: resource.VikingURI,
    }, nil
}

func (uc *AddResourceUseCase) runIngestion(ctx context.Context, res *domain.Resource, task *domain.BackgroundTask, req AddResourceRequest) {
    // Panic recovery
    defer func() {
        if r := recover(); r != nil {
            uc.updateStatus(ctx, res, task, domain.ResourceStatusFailed, fmt.Sprintf("panic: %v", r), 0)
        }
    }()
    
    uc.updateStatus(ctx, res, task, domain.ResourceStatusProcessing, "", 0.0)
    
    // Step 1: Detect source
    res.SourceType = pipeline.DetectSource(req.URLOrPath)
    uc.updateStatus(ctx, res, task, domain.ResourceStatusProcessing, "", 0.05)
    
    // Step 2: Clone/Download
    var tempPath string
    var err error
    switch res.SourceType {
    case domain.SourceTypeGit:
        tempPath, err = uc.gitAdapter.Clone(ctx, req.URLOrPath, req.Name)
    case domain.SourceTypeHTTP:
        tempPath, err = uc.httpAdapter.Download(ctx, req.URLOrPath, req.Name)
    case domain.SourceTypeLocalDir:
        tempPath = req.URLOrPath  // Use as-is
    case domain.SourceTypeLocalFile:
        tempPath, err = copyLocalFile(req.URLOrPath, uc.config.TempDir, req.Name)
    }
    if err != nil {
        uc.updateStatus(ctx, res, task, domain.ResourceStatusFailed, err.Error(), 0.05)
        return
    }
    uc.updateStatus(ctx, res, task, domain.ResourceStatusProcessing, "", 0.15)
    
    // Step 3: Scan directory
    files, err := pipeline.ScanDirectory(ctx, tempPath, res.SourceType)
    if err != nil {
        uc.updateStatus(ctx, res, task, domain.ResourceStatusFailed, err.Error(), 0.15)
        return
    }
    uc.updateStatus(ctx, res, task, domain.ResourceStatusProcessing, "", 0.25)
    
    // Step 4: Parse files
    parsedFiles, err := pipeline.ParseFiles(ctx, files, uc.parseRegistry)
    if err != nil {
        uc.updateStatus(ctx, res, task, domain.ResourceStatusFailed, err.Error(), 0.25)
        return
    }
    uc.updateStatus(ctx, res, task, domain.ResourceStatusProcessing, "", 0.50)
    
    // Step 5: Build FS (to temp location)
    if err := pipeline.BuildFS(ctx, req.Name, parsedFiles, req.AccountID, uc.fsClient); err != nil {
        uc.updateStatus(ctx, res, task, domain.ResourceStatusFailed, err.Error(), 0.50)
        return
    }
    uc.updateStatus(ctx, res, task, domain.ResourceStatusProcessing, "", 0.75)
    
    // Step 6: Background L0/L1 generation
    if req.GenerateSummaries {
        go pipeline.GenerateSummaries(context.Background(), parsedFiles, req.AccountID, uc.fsClient, uc.vlmClient)
    }
    
    // Step 7: Finalize (temp → resources/)
    tempURI := fmt.Sprintf("viking://temp/%s/%s/", req.AccountID, req.Name)
    finalURI := fmt.Sprintf("viking://resources/%s/", req.Name)
    if err := pipeline.Finalize(ctx, tempURI, finalURI, uc.fsClient); err != nil {
        uc.updateStatus(ctx, res, task, domain.ResourceStatusFailed, err.Error(), 0.75)
        return
    }
    
    // Watch setup
    if req.Watch {
        res.WatchConfig = &domain.WatchConfig{
            Enabled:  true,
            Interval: req.WatchInterval,
        }
        if res.SourceType == domain.SourceTypeGit {
            res.WatchConfig.LastCommit = uc.gitAdapter.GetHEAD(tempPath)
        }
    }
    
    res.FileCount = len(parsedFiles)
    uc.updateStatus(ctx, res, task, domain.ResourceStatusDone, "", 1.0)
    
    // Publish NATS event
    uc.publisher.PublishResourceIngested(ctx, port.ResourceIngestedPayload{
        ResourceID: res.ID, Name: res.Name, URI: finalURI, FileCount: len(parsedFiles),
    })
    
    // Cleanup temp (non-blocking)
    go os.RemoveAll(tempPath)
}
```

---

## Unit Tests

```
TestDetectSource_GitHub         → github.com URL → SourceTypeGit
TestDetectSource_HTTP           → https:// URL → SourceTypeHTTP
TestDetectSource_LocalDir       → existing directory path → SourceTypeLocalDir
TestDetectSource_LocalFile      → existing file path → SourceTypeLocalFile
TestScanDirectory_GitignoreApplied → .gitignore node_modules → skipped
TestScanDirectory_DefaultIgnore → *.pyc files → skipped
TestScanDirectory_SkipsLargeFiles → file > 10MB → skipped
TestParseFiles_GoFile           → .go file → chunks with ChunkType=function
TestParseFiles_UnknownExtension → .xyz → TextParser fallback
TestParseFiles_WorkerPool       → 10 files → parseWorkers goroutines max
TestBuildFS_WritesFiles         → 5 files → 5 WriteRaw calls to FS
TestGenerateSummaries_VLMFails  → VLM error → resource not failed
TestShouldGenerateSummary_Go    → .go → true
TestShouldGenerateSummary_Binary → .exe → false
TestAddResource_CreatesTask     → call → task in store with state=queued
TestAddResource_AsyncIngestion  → call returns → goroutine running
TestUpdateStatus_Progress       → progress stored in task store
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

# Deps
go get github.com/JohannesKaufmann/html-to-markdown
go get github.com/go-git/go-gitignore

buf generate services/openviking-resource/
go build ./services/openviking-resource/...
go test ./services/openviking-resource/... -v -count=1
```

---

## Ghi chú triển khai

- `tempDir` default: `~/.openviking/temp` (configurable)
- `go-gitignore` hoặc tự implement gitignore matching với `github.com/sabhiram/go-gitignore`
- Task progress stored in Redis (`task_redis.go`): key=`ov_task:{account_id}:{task_id}`, TTL=7 ngày
- `LocalDir` và `LocalFile` source types: hardlink hoặc copy vào tempDir để xử lý đồng nhất
- Background cleanup: `os.RemoveAll(tempPath)` sau Finalize (Mv đã move files, temp có thể xóa)
