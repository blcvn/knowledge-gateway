# Change Request: CR-OV-005 — Resource Service (Ingestion Pipeline)

**CR ID:** CR-OV-005  
**Component:** `services/openviking-resource` [NEW SERVICE]  
**Priority:** High  
**Status:** Implemented
**Reference:** OpenViking PRD §4.5, SRS §2.5, specs/services/05-resource-service.md  
**Maps from Python:** `utils/resource_processor.py`, `parse/` (9 files), `resource/watch_manager.py`

---

## 1. Mô tả

Xây dựng **openviking-resource** — pipeline xử lý và ingest external resources vào VikingFS:

1. **Multi-Source Ingestion**: Git repos (clone + tree-sitter), HTTP URLs (scrape + markdown), local files/dirs, documents (PDF, DOCX, PPTX, XLSX, EPUB).
2. **Source Detection**: Auto-detect source type từ URL/path pattern.
3. **File Parser Registry**: 50+ extensions → đúng parser (tree-sitter cho code, VLM cho images/PDF, native Go cho markdown).
4. **L0/L1 Summary Generation**: Tự động tạo `.abstract.md` (L0, ~100 tokens) và `.overview.md` (L1, ~2K tokens) via VLM/Bifrost.
5. **Skill Loading**: Load agent skills/tools từ directories, index vào Search.
6. **Watch/Auto-Refresh**: Monitor resources for changes (git diff, HTTP ETag), auto re-process.
7. **Task Tracking**: Background task lifecycle (QUEUED→PROCESSING→DONE/FAILED) với Redis state.
8. **Goroutine Pool**: Bounded concurrency cho CPU-intensive parsing operations.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có resource ingestion pipeline cho git repos và web URLs.
- Thiếu tree-sitter code parsing cho semantic code understanding.
- Không có L0/L1 auto-generation → thiếu tiered context system.
- Chưa có watch/auto-refresh cho resources.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/openviking-resource/` (Port gRPC: 9014)

### 3.2. Domain Model

```go
// domain/resource.go
type Resource struct {
    ID          string
    Name        string
    SourceURL   string
    SourceType  SourceType   // Git | HTTP | LocalDir | SingleFile
    Status      ResourceStatus
    VikingURI   string       // viking://resources/{name}/
    WatchConfig *WatchConfig
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

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

// domain/skill.go
type Skill struct {
    ID           string
    AgentID      string
    Name         string
    Description  string   // L0 summary
    Instructions string   // L2 full content
    VikingURI    string   // viking://agent/{id}/skills/{name}/
}

// domain/task.go
type BackgroundTask struct {
    ID        string
    TaskType  string    // "resource_ingest" | "resource_refresh" | "skill_load"
    State     TaskState
    Progress  float64   // 0.0 - 1.0
    Error     string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 3.3. Ingestion Pipeline (7 Steps)

```
add_resource(url_or_path, name, context):

Step 1: Source Detection
  "https://github.com/user/repo" → SourceType = GitRepository
  "https://example.com/doc"     → SourceType = WebPage
  "/path/to/directory"          → SourceType = LocalDirectory
  "/path/to/file.pdf"           → SourceType = SingleFile

Step 2: Clone/Download → temp location
  Git:   git clone --depth=1 url → viking://temp/{name}/
  HTTP:  fetch HTML → markdown conversion → viking://temp/{name}/index.md
  Local: hardlink/copy → viking://temp/{name}/

Step 3: Scan Directory → file tree
  Apply .gitignore rules (git repos)
  Apply default ignore patterns: *.pyc, __pycache__, .git/, node_modules/
  Build ParsedFileList

Step 4: Parse each file (Goroutine pool — N workers)
  Detect parser from registry by file extension
  → tree-sitter: .py, .js, .ts, .go, .rs, .java, .c, .cpp, .rb, .php (50+ langs)
  → markdown: .md, .mdx
  → document: .pdf, .docx, .xlsx, .epub (Go libraries)
  → vlm: .png, .jpg, .pptx (Vision-Language Model)
  → text: .txt, .yaml, .toml, .json (native)
  Output: ParsedFile{path, content, language, lineCount, chunks[]}

Step 5: Build directory tree structure
  Create FSService Mkdir calls for each directory
  Write parsed content to FS service (auto-emits ov.content.written → Search indexes)

Step 6: Background — L0/L1 Generation (Goroutine pool)
  For each file:
    a. VLM.GenerateAbstract(content, max_tokens=100)
       → Write file.abstract.md to FS service
    b. VLM.GenerateOverview(content, max_tokens=2000)
       → Write file.overview.md to FS service
  For each directory:
    a. VLM.GenerateDirectoryAbstract(tree_structure)
       → Write .abstract.md for directory node

Step 7: Finalize
  Move viking://temp/{name}/ → viking://resources/{name}/
  Update resource status → DONE
  Publish NATS: ov.resource.ingested{name, uri, file_count}
```

### 3.4. Parser Registry (50+ Extensions)

| Parser | Extensions | Technology |
|--------|-----------|------------|
| Tree-sitter | `.go`, `.py`, `.js`, `.ts`, `.rs`, `.java`, `.c`, `.cpp`, `.rb`, `.php`, `.swift`, `.kt` | `smacker/go-tree-sitter` |
| Markdown | `.md`, `.mdx`, `.markdown` | Native Go + goldmark |
| Document | `.pdf`, `.docx`, `.xlsx`, `.epub` | `unidoc/unipdf`, `gooxml` |
| VLM | `.png`, `.jpg`, `.jpeg`, `.pptx` | Vision model via Bifrost |
| Text | `.txt`, `.yaml`, `.toml`, `.json`, `.csv` | Native Go |
| Directory | directories | Recursive scanner + .gitignore |

### 3.5. Watch Manager

```go
// usecase/watch_resources.go
// Periodic ticker (default: 1 hour, configurable)
type WatchManager struct {
    ticker  *time.Ticker
    watched []WatchConfig
}

func (wm *WatchManager) CheckAll(ctx context.Context) {
    for _, config := range wm.watched {
        switch config.SourceType {
        case SourceTypeGit:
            // git fetch && git diff origin/main → has_changes
            changed, _ := wm.gitAdapter.HasNewCommits(config.SourceURL, config.LastCommit)
        case SourceTypeHTTP:
            // HEAD request → check ETag or Last-Modified header
            changed, _ := wm.httpAdapter.HasChanged(config.URL, config.LastETag)
        }
        if changed {
            wm.taskQueue.Enqueue(RefreshTask{ResourceID: config.ResourceID})
        }
    }
}
```

### 3.6. Skill Loading

```go
// usecase/load_skills.go
// Load skills from directory structure:
// viking://agent/{agent_id}/skills/
//   ├── web_search/
//   │   ├── .abstract.md   (L0: "Search the web for information")
//   │   ├── .overview.md   (L1: full skill description with examples)
//   │   └── instructions.md (L2: detailed implementation guide)
//   └── code_executor/
//       └── ...

func (uc *LoadSkillsUseCase) Execute(ctx context.Context, agentID string, skillsDir string) error {
    // 1. FS.Ls(skillsDir) → list skill directories
    // 2. For each skill: read .abstract.md + instructions.md
    // 3. Store Skill entities
    // 4. Search.IndexContent for semantic search
}
```

### 3.7. gRPC Service Definition

```protobuf
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

  // Background Tasks
  rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
}

message AddResourceRequest {
  string account_id = 1;
  string user_id = 2;
  string url_or_path = 3;           // Source URL or local path
  string name = 4;                  // Resource name (directory name in VikingFS)
  bool generate_summaries = 5;      // Generate L0/L1 (default: true)
  bool watch = 6;                   // Enable auto-refresh watching
  string watch_interval = 7;        // "1h" | "24h" | etc.
}

message AddResourceResponse {
  string resource_id = 1;
  string task_id = 2;               // Background task ID to track progress
  string viking_uri = 3;            // viking://resources/{name}/
}
```

### 3.8. NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `ov.resource.ingested` | `{resource_id, name, uri, file_count}` | Search (collection warm-up) |
| `ov.resource.parsed` | `{resource_id, file_uri, content_hash}` | FS (write content — done inline) |

**Consumed:**
| Subject | Source | Action |
|---------|--------|--------|
| `admin.account.created` | Admin | Initialize `viking://resources/` and `viking://agent/` dirs for account |

### 3.9. Configuration

```yaml
resource:
  grpc:
    port: 9014
  health:
    port: 9094
  ingestion:
    worker_pool_size: 5              # Goroutine pool for parsing
    vlm_worker_pool_size: 2          # Goroutine pool for VLM calls (rate-limited)
    temp_dir: "~/.openviking/temp"   # Staging area during ingestion
    generate_summaries: true
    summary_max_tokens_l0: 100
    summary_max_tokens_l1: 2000
  watch:
    enabled: true
    default_interval: "1h"
    max_watched: 100
  git:
    clone_depth: 1
    timeout: 300s
  http:
    download_timeout: 60s
    max_size_mb: 100
  clients:
    fs: "openviking-fs:9011"
    search: "openviking-search:9012"
    vlm: "bifrost:4000"
  redis:
    url: "redis://redis:6379/3"     # Task state storage
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
```

---

## 4. Acceptance Criteria

- [ ] `AddResource(url="https://github.com/golang/go", name="golang-stdlib")` → clone, parse, index; `status=DONE` trong < 5 phút.
- [ ] Parser registry: `.go` files parsed với tree-sitter → chunked thành function-level segments.
- [ ] L0 generation: `viking://resources/golang-stdlib/net/http/server.go.abstract.md` được tạo với ~100 token summary.
- [ ] `AddResource(url="https://docs.example.com")` → web page scraped, converted to markdown, indexed.
- [ ] Watch enabled: resource có git commit mới → auto-refresh triggered trong vòng interval configured.
- [ ] Task tracking: `GetTask(task_id)` → trả về progress 0.0→1.0 trong quá trình ingestion.
- [ ] `DeleteResource(resource_id)` → toàn bộ `viking://resources/{name}/` bị xóa; Search index được cleanup.
- [ ] Goroutine pool: ingest large repo (10,000 files) → memory usage bounded, không OOM.
- [ ] VLM summary generation fails → resource vẫn được ingested (L0/L1 là optional), status = DONE (partial).
