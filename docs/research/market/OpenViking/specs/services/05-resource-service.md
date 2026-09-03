# 05 — OpenViking Resource Service

> **Service**: `openviking-resource`  
> **Port**: 9014 (gRPC) · 9094 (Health/Metrics)  
> **Origin**: L2 ResourceService + L4 ResourceProcessor + L4 SkillProcessor + L4 Parse Engine  
> **Role**: Resource ingestion pipeline, file parsing, skill loading, watch/auto-refresh

---

## 1. Responsibilities

| Capability | Description |
|-----------|-------------|
| **Resource Ingestion** | Full pipeline: detect → clone/download → parse → embed → store |
| **Source Detection** | Git repo, HTTP URL, local directory, single file |
| **File Parsing** | Tree-sitter (code), VLM (images/PDF), document parsers |
| **L0/L1 Generation** | Generate .abstract.md (L0) and .overview.md (L1) via VLM |
| **Skill Loading** | Load and index agent skills/tools |
| **Watch/Refresh** | Monitor resources for changes, auto re-process |
| **Task Tracking** | Track background task status (QUEUED→PROCESSING→DONE/FAILED) |

---

## 2. Clean Architecture Layout

```
services/openviking-resource/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── resource.go                 # Resource, ResourceStatus, SourceType
│   │   ├── skill.go                    # Skill, SkillMetadata
│   │   ├── task.go                     # BackgroundTask, TaskState
│   │   ├── chunk.go                    # ContentChunk, ParsedFile
│   │   ├── watch.go                    # WatchConfig, WatchState
│   │   └── errors.go
│   ├── usecase/
│   │   ├── add_resource.go             # Full ingestion pipeline orchestration
│   │   ├── detect_source.go            # Auto-detect source type
│   │   ├── clone_download.go           # Git clone / HTTP download
│   │   ├── parse_files.go              # File parsing orchestration
│   │   ├── generate_summaries.go       # L0/L1 via VLM
│   │   ├── refresh_resource.go         # Re-process existing resource
│   │   ├── load_skills.go              # Skill loading + indexing
│   │   ├── watch_resources.go          # Watch manager
│   │   ├── track_task.go               # Task tracking CRUD
│   │   ├── port/
│   │   │   ├── input.go               # ResourceUseCase, SkillUseCase interfaces
│   │   │   └── output.go             # FSClient, SearchClient, VLMClient, ParserRegistry
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── task_store/            # In-memory or Redis task store
│   │   │   └── git/                   # Git clone adapter
│   │   ├── client/
│   │   │   ├── fs_client.go           # Write parsed content to FS service
│   │   │   ├── search_client.go       # Trigger embedding + indexing
│   │   │   └── vlm_client.go          # VLM for L0/L1 generation
│   │   ├── parser/                    # Parser adapter implementations
│   │   │   ├── treesitter_adapter.go  #   tree-sitter Go bindings
│   │   │   ├── markdown_adapter.go
│   │   │   ├── document_adapter.go    #   PDF/DOCX
│   │   │   └── vlm_adapter.go         #   Images/PPTX via VLM
│   │   └── event/
│   │       ├── publisher.go            # NATS: ov.resource.ingested/parsed
│   │       └── subscriber.go          # NATS: admin.account.created
│   └── infra/
```

---

## 3. Ingestion Pipeline

```
add_resource(url_or_path, name, ctx):
  1. Detect source type:
     "https://github.com/..." → GitRepository
     "https://..."           → WebPage
     "/path/to/dir"          → LocalDirectory
     "/path/to/file.pdf"     → SingleFile

  2. Clone/download to viking://temp/{name}/ via FS service

  3. Scan directory → file tree
     Apply .gitignore filtering

  4. For each file:
     a. Detect parser from registry (50+ extensions)
     b. Parse → chunks (text segments with metadata)
     c. Compute metadata (line count, language)

  5. Build directory tree structure

  6. Background processing (Go goroutine pool):
     a. Write parsed content to FS service
        → FS emits ov.content.written → Search indexes it
     b. Generate L0 (.abstract.md) via VLM
     c. Generate L1 (.overview.md) via VLM
     d. Write L0/L1 to FS service

  7. Move from temp → viking://resources/{name}/ via FS service

  8. Update task status → DONE
  9. Publish ov.resource.ingested
```

---

## 4. Parser Registry

| Parser | File Types | Technology |
|--------|-----------|------------|
| Tree-sitter | `.py`, `.js`, `.ts`, `.go`, `.rs`, `.java`, `.c`, `.cpp` | Go tree-sitter bindings |
| VLM Parser | `.png`, `.jpg`, `.pptx` | Vision-Language Model via Bifrost |
| Document | `.pdf`, `.docx`, `.xlsx`, `.epub` | Go PDF/DOCX libraries |
| Markdown | `.md` | Native Go parser |
| Directory Scanner | directories | Recursive scan + .gitignore |

---

## 5. Watch Manager

```
WatchScheduler (periodic Go ticker):
  └── WatchManager.CheckAll()
      ├── For each watched resource:
      │   ├── Git: git fetch → check diff
      │   ├── HTTP: HEAD request → check ETag/Last-Modified
      │   ├── If changed: queue re-process
      │   └── Update watch metadata
      └── Configurable interval (default: 1 hour)
```

---

## 6. gRPC Service Definition

```protobuf
service ResourceService {
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
```

---

## 7. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Go tree-sitter bindings | `smacker/go-tree-sitter` — mature Go library for code parsing |
| Goroutine pool for parsing | Bounded concurrency for CPU-intensive parsing ops |
| VLM via Bifrost | Unified LLM gateway, no direct provider coupling |
| Task tracking in Redis | Shared state for multi-replica deployments |
