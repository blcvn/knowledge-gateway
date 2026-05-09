# TDD — Memobase: Technical Design Document

| Field | Value |
|-------|-------|
| **Product** | Memobase v0.0.40 |
| **Date** | 2026-05-09 |
| **Status** | Active |

---

## 1. Overview

Tài liệu TDD mô tả chi tiết thiết kế kỹ thuật của từng module trong Memobase, bao gồm data flow, algorithm design, API contracts, error handling, và configuration patterns.

---

## 2. Module Design: Authentication

### 2.1 Auth Middleware (`api_layer/middleware.py`)

**Class**: `AuthMiddleware(BaseHTTPMiddleware)`

**Flow**:
```
dispatch(request, call_next)
  ├── Skip non-API paths
  ├── Skip /healthcheck (increment HEALTHCHECK counter)
  ├── Extract Bearer token from Authorization header
  ├── is_valid_root(token)?
  │   ├── Yes → project_id = "__root__", is_root = True
  │   └── No → parse_project_token(token)
  │       ├── parse_project_id(token) — extract project_id from token format
  │       ├── check_project_secret(project_id, token) — DB verify
  │       └── get_project_status(project_id) — check ≠ "suspended"
  ├── Normalize path for metrics
  ├── Record REQUEST counter + REQUEST_LATENCY histogram
  └── call_next(request)
```

**Token Module** (`auth/token.py`):
- `parse_project_id(token)` — extract project_id from `sk-proj-{project_id}-{secret}` format
- `check_project_secret(project_id, token)` — verify against DB
- `get_project_status(project_id)` — return project status enum

### 2.2 Global Wrapper Middleware

**Responsibilities**:
- Inject `X-Request-ID` (from header or UUID v4)
- Bind structlog context vars: `request_id`, `project_id`, `memobase_version`
- Measure `process_time` (perf_counter_ns)
- Catch unhandled exceptions → 500 JSON response with error report URL
- Set `X-Process-Time` response header

---

## 3. Module Design: Blob Ingestion

### 3.1 Blob Types (`models/blob.py`)

```python
class BlobType(str, Enum):
    chat = "chat"
    doc = "doc"
    summary = "summary"
```

**ChatBlob Format**:
```json
{
  "messages": [
    {"role": "user|assistant|system", "content": "..."}
  ]
}
```

### 3.2 Insert Flow (`api_layer/blob.py` → `controllers/blob.py`)

```
insert_blob(user_id, blob_data)
  │
  ├── Validate blob_type ∈ {chat, doc, summary}
  ├── Store GeneralBlob in DB
  │   └── blob_data = JSONB, blob_type = VARCHAR
  │
  ├── insert_blob_to_buffer(user_id, blob_id, blob_data)
  │   └── Create BufferZone entry:
  │       token_size = tiktoken.encode(blob_str).length
  │       status = "idle"
  │
  ├── detect_buffer_full_or_not()
  │   └── SUM(token_size) WHERE status="idle" > max_chat_blob_buffer_token_size?
  │       ├── Yes → Background task: flush_buffer_by_ids()
  │       └── No → Return blob_id
  │
  └── Return {id: blob_id, chat_results: [...]}
```

### 3.3 Token Size Calculation

```python
# utils.py
ENCODER = tiktoken.encoding_for_model("gpt-4o")

def get_blob_token_size(blob: Blob) -> int:
    return len(ENCODER.encode(get_blob_str(blob)))
```

---

## 4. Module Design: Buffer Zone

### 4.1 State Machine

```
        insert_blob
            │
            ▼
    ┌──────────────┐
    │     idle     │ ← Initial state
    └──────┬───────┘
           │ flush triggered
           ▼
    ┌──────────────┐
    │  processing  │ ← LLM calls in progress
    └──┬───────┬───┘
       │       │
       ▼       ▼
  ┌────────┐ ┌────────┐
  │  done  │ │ failed │ ← Retry-able
  └────────┘ └────────┘
```

### 4.2 Flush Algorithm (`controllers/buffer.py`)

```python
async def flush_buffer_by_ids(user_id, project_id, blob_type, buffer_ids):
    # 1. Join BufferZone + GeneralBlob in one query
    buffer_blob_data = session.query(
        BufferZone.id, BufferZone.blob_id, BufferZone.token_size,
        GeneralBlob.blob_data
    ).join(GeneralBlob).filter(status=idle, id.in_(buffer_ids))

    # 2. Update status to "processing"
    session.update(BufferZone).set(status="processing")

    # 3. Process blobs (LLM pipeline)
    result = await BLOBS_PROCESS[blob_type](user_id, project_id, blobs)

    # 4a. Success → status="done", delete blobs if non-persistent
    # 4b. Failure → status="failed"
```

### 4.3 Concurrency Control

- **Optimistic**: Status-based locking thay vì DB locks
- **Risk**: Parallel flush có thể tạo duplicate processing (documented FIXME)
- **Mitigation**: Status check `WHERE status=idle` trong query ngăn re-processing

---

## 5. Module Design: Memory Processing Pipeline (Chat Modal)

### 5.1 Pipeline Entry (`controllers/modal/chat/__init__.py`)

```python
async def process_blobs(user_id, project_id, blobs):
    # Truncate blobs to max_chat_blob_buffer_process_token_size (16384)
    blobs = truncate_chat_blobs(blobs, CONFIG.max_...)

    # Get project config + existing profiles
    project_profiles = get_project_profile_config(project_id)
    current_user_profiles = get_user_profiles(user_id, project_id)

    # Step 1: Entry summary (LLM call #1)
    user_memo_str = entry_chat_summary(blobs, project_profiles, current_profiles)

    # Step 2: Parallel processing
    asyncio.gather(
        process_profile_res(user_memo_str),   # LLM calls #2 + #3
        process_event_res(user_memo_str),      # Conditional
    )

    # Step 3: Persist results
    handle_session_event(event + tags + profile_delta)
    handle_user_profile_db(add + update + delete)
```

### 5.2 Profile Processing Sub-pipeline

```
process_profile_res(user_memo_str)
  │
  ├── extract_topics() ── LLM Call #2
  │   Input: user_memo_str + profile schema
  │   Output: {fact_contents: [...], fact_attributes: [...]}
  │   Prompt: extract_profile.py (EN) | zh_extract_profile.py (ZH)
  │
  ├── merge_yolo() ── LLM Call #3
  │   Input: extracted facts + existing profiles
  │   Output: {add: [...], update: [...], delete: [...]}
  │   Prompt: merge_profile_yolo.py (EN) | zh_merge_profile_yolo.py (ZH)
  │   Algorithm: Single LLM call determines all add/update/delete decisions
  │
  ├── organize_profiles() ── No LLM
  │   Logic: If any topic has > max_profile_subtopics → merge similar subtopics
  │
  └── re_summary() ── Conditional LLM
      Logic: If any profile content > max_pre_profile_token_size → summarize
```

### 5.3 YOLO Merge Algorithm

**Design Decision**: Trước v0.0.40 dùng multi-step merge (3-10 LLM calls), YOLO merge gộp thành 1 call.

**Input cho LLM**:
```
Existing Profiles:
[0] basic_info::name: "Gus"
[1] interest::food: "Mexican cuisine"

New Facts:
- basic_info::age: "25"
- interest::food: "Also likes Thai food"
```

**Expected Output** (JSON):
```json
{
  "add": [{"topic": "basic_info", "sub_topic": "age", "memo": "25"}],
  "update": [{"index": 1, "memo": "Mexican cuisine, Thai food"}],
  "delete": []
}
```

### 5.4 Event Processing Sub-pipeline

```
process_event_res(memo_str)
  │
  ├── tag_event() ── Conditional LLM
  │   Only if event_tags configured in project
  │   Input: memo_str + tag definitions
  │   Output: [{tag: "emotion", value: "happy"}, ...]
  │
  └── append_user_event()
      ├── Store event_data (JSONB)
      ├── Generate embedding for event (if enabled)
      ├── Split event_tip into gists (lines starting with "-")
      ├── Generate embeddings for each gist
      └── Store UserEventGist records
```

---

## 6. Module Design: Profile Management

### 6.1 Caching Strategy

```python
# Read path
async def get_user_profiles(user_id, project_id):
    # 1. Check Redis
    cached = redis.get(f"user_profiles::{project_id}::{user_id}")
    if cached:
        return UserProfilesData.model_validate_json(cached)

    # 2. Cache miss → DB query
    profiles = session.query(UserProfile).filter_by(user_id, project_id)

    # 3. Cache result (TTL = 1200s)
    redis.set(key, profiles.model_dump_json(), ex=1200)
    return profiles

# Write path (any mutation)
async def refresh_user_profile_cache(user_id, project_id):
    redis.delete(f"user_profiles::{project_id}::{user_id}")
```

### 6.2 Profile Truncation Algorithm

**Purpose**: Fit profiles into token budget for context API.

```python
async def truncate_profiles(profiles, prefer_topics, only_topics,
                           max_token_size, max_subtopic_size, topic_limits):
    # 1. Sort by updated_at DESC (most recent first)
    profiles.sort(key=lambda p: p.updated_at, reverse=True)

    # 2. Priority ordering (if prefer_topics)
    #    Move preferred topics to front, maintain internal order

    # 3. Topic filter (if only_topics)
    #    Keep only specified topics

    # 4. Subtopic limit (if max_subtopic_size or topic_limits)
    #    Cap subtopics per topic

    # 5. Token budget (if max_token_size)
    #    Accumulate tokens until budget exceeded
    for p in profiles:
        tokens += tiktoken.encode(f"{topic}::{sub_topic}: {content}")
        if tokens > max_token_size:
            break
```

### 6.3 Profile Filtering with Chat Context

**Purpose**: Nếu user đang chat, chỉ trả về profiles relevant.

```python
# controllers/post_process/profile.py
async def filter_profiles_with_chats(profiles, chats):
    # 1. Prepare profile index với truncated content
    topics_index = [{index, topic, sub_topic, content[:10_tokens]}]

    # 2. LLM call: "Which profiles are relevant to this chat?"
    #    Prompt: pick_related_profiles.py
    #    Temperature: 0.2 (precise)

    # 3. Parse LLM response → list of profile indices
    found_ids = find_list_int_or_none(response)

    return filtered_profiles
```

---

## 7. Module Design: Event & Semantic Search

### 7.1 Embedding Architecture

```
┌──────────────────────────────────────────────────┐
│              Embedding Engine                     │
│                                                  │
│  get_embedding(project_id, texts, phase, model)  │
│      │                                           │
│      ├── OpenAI: openai.embeddings.create()      │
│      ├── Jina: HTTP POST api.jina.ai/v1          │
│      ├── Ollama: HTTP POST /api/embed            │
│      └── LMStudio: openai-compatible endpoint    │
│                                                  │
│  Output: numpy.ndarray (dim = CONFIG.embedding_dim)│
│  Validation: dim check at startup + per-response │
└──────────────────────────────────────────────────┘
```

### 7.2 Vector Search Implementation

```python
# controllers/event.py
async def search_user_events(user_id, project_id, query, topk, threshold):
    # 1. Embed query
    query_embedding = await get_embedding(project_id, [query], phase="query")

    # 2. pgvector cosine similarity search
    stmt = select(
        UserEvent,
        (1 - UserEvent.embedding.cosine_distance(query_embedding)).label("similarity")
    ).where(
        user_id == user_id,
        similarity > threshold,          # default 0.2
        created_at > now() - days(21)     # time range filter
    ).order_by(desc("similarity")).limit(topk)
```

### 7.3 Event Gist Search

Same pattern as event search nhưng trên `user_event_gists` table. Event gists là fine-grained descriptions (mỗi line bắt đầu bằng `-` trong event_tip), cho kết quả chính xác hơn event-level search.

---

## 8. Module Design: Context Assembly

### 8.1 Context API Algorithm

```python
async def get_user_context(user_id, project_id, max_token_size, ...):
    # Token budget allocation
    max_profile_tokens = max_token_size * profile_event_ratio

    # Parallel fetch
    profile_result, event_result = await asyncio.gather(
        get_user_profiles_data(...),      # Profiles (cache → DB)
        get_user_event_gists_data(...),   # Events (semantic search or recent)
    )

    # Profile section
    profile_section = "- " + "\n- ".join([
        f"{topic}::{sub_topic}: {content}" for p in profiles
    ])

    # Event section: fill remaining token budget
    max_event_tokens = max_token_size - profile_section_tokens
    event_section = truncated_event_gists

    # Assemble via prompt template
    return context_prompt_func(profile_section, event_section)
```

### 8.2 Prompt Templates

**English** (`prompts/chat_context_pack.py`):
```
# Memory
Unless the user has relevant queries, do not actively mention those memories.
## User Background:
{profile_section}

## Latest Events:
{event_section}
```

**Chinese**: Equivalent ZH prompt template.

**Custom**: User provides template with `{profile_section}` and `{event_section}` placeholders.

---

## 9. Module Design: LLM Integration

### 9.1 LLM Abstraction Layer

```python
# llms/__init__.py
FACTORIES = {
    "openai": openai_complete,       # Standard OpenAI Chat Completions
    "doubao_cache": doubao_cache_complete  # Volcengine Doubao with cache
}

async def llm_complete(project_id, prompt, system_prompt=None,
                       json_mode=False, model=None, max_tokens=1024):
    result = await FACTORIES[CONFIG.llm_style](model, prompt, ...)

    # Token accounting (async fire-and-forget)
    asyncio.create_task(project_cost_token_billing(project_id, in_tokens, out_tokens))

    # Telemetry
    telemetry_manager.increment_counter(LLM_INVOCATIONS)
    telemetry_manager.increment_counter(LLM_TOKENS_INPUT, in_tokens)
    telemetry_manager.increment_counter(LLM_TOKENS_OUTPUT, out_tokens)
    telemetry_manager.record_histogram(LLM_LATENCY_MS, latency)

    if json_mode:
        return convert_response_to_json(result)
    return result
```

### 9.2 Model Configuration

| Config Key | Default | Purpose |
|-----------|---------|---------|
| `best_llm_model` | gpt-4o-mini | Primary model cho extract, merge, summary |
| `thinking_llm_model` | o4-mini | Reasoning-heavy tasks |
| `summary_llm_model` | None (= best) | Summary/filter tasks |

### 9.3 Sanity Check

Tại startup, `llm_sanity_check()` gửi test request "Test" với max_tokens=16 để verify LLM connectivity.

---

## 10. Module Design: Configuration System

### 10.1 Config Loading Priority

```
1. config.yaml (file)
   ↓ overridden by
2. Environment variables (MEMOBASE_* prefix)
   ↓ overridden by
3. Per-project API config (POST /project/profile_config)
```

### 10.2 Environment Variable Processing

```python
# env.py: Config._process_env_vars()
for field in dataclasses.fields(Config):
    env_var = f"MEMOBASE_{field.name.upper()}"
    if env_var in os.environ:
        # Try JSON parse first (for complex types: lists, dicts)
        # Then try raw string
        # Type-check with typeguard.check_type()
```

### 10.3 Per-Project Config Scope

`ProfileConfig` (stored in `projects.profile_config` as YAML string):
- `language` — Override prompt language (en/zh)
- `profile_strict_mode` — Only collect defined profiles
- `profile_validate_mode` — Validate extracted profiles
- `additional_user_profiles` — Extra profile topics
- `overwrite_user_profiles` — Replace default profile topics
- `event_tags` — Custom event tag definitions
- `event_theme_requirement` — Event extraction guidance

---

## 11. Module Design: Telemetry & Observability

### 11.1 TelemetryManager Design

**Singleton**: `telemetry_manager` created at module import.

```python
class TelemetryManager:
    def __init__(self, service_name, prometheus_port=9464):
        self.setup_telemetry()   # OpenTelemetry + Prometheus reader
        self.setup_metrics()     # Create all metric instruments

    @no_raise_exception    # Decorator: swallow telemetry errors
    def increment_counter_metric(metric, value, attributes): ...
    def record_histogram_metric(metric, value, attributes): ...
    def set_gauge_metric(metric, value, attributes): ...
```

**Attributes injected per metric**:
- `deployment_environment` (from config)
- `memobase_server_ip` (from POD_IP env or hostname)
- Custom labels per call (project_id, path, method)

### 11.2 Structured Logging

**Production (JSON)**:
```python
configure_logger()  # structlog config
LOG = structlog.get_logger().bind(app_name="memobase_server")
TRACE_LOG = ProjectStructLogger(LOG)  # Adds project_id, user_id context
```

**Development (Plain)**:
```python
LOG = logging.getLogger("memobase_server")
# Format: "memobase_server | INFO - 2026-05-09 - message"
TRACE_LOG = ProjectLogger(LOG)  # JSON-encodes project_id, user_id
```

---

## 12. Module Design: Database Connection Management

### 12.1 Connection Pool Configuration

```python
DB_ENGINE = create_engine(DATABASE_URL,
    pool_size=75,               # Base persistent connections
    max_overflow=50,            # Burst capacity
    pool_recycle=300,           # Recycle every 5 min
    pool_pre_ping=True,         # Validate before use
    pool_timeout=45,            # Wait timeout
    pool_reset_on_return="commit"  # Clean state
)
```

**Total capacity**: 75 + 50 = 125 connections per instance.

### 12.2 Pool Monitoring

```python
def log_pool_status(operation: str):
    status = {
        size, checked_in, checked_out,
        overflow, total_capacity, utilization_percent
    }
    if utilization_percent > 80:
        LOG.warning(f"High DB pool utilization: {status}")
```

Called at: `flush_buffer_by_ids_start`, `flush_buffer_by_ids_db_error`, `flush_buffer_by_ids_exception`.

### 12.3 Startup Initialization

```python
# connectors.py (module-level)
create_pgvector_extension()          # CREATE EXTENSION IF NOT EXISTS vector
REG.metadata.create_all(DB_ENGINE)   # Create all ORM tables
Project.initialize_root_project()     # Ensure __root__ project exists
UserEvent.check_legal_embedding_dim() # Validate vector dimensions
UserEventGist.check_legal_embedding_dim()
```

---

## 13. Error Handling Patterns

### 13.1 Promise Pattern

Memobase uses a custom `Promise` pattern cho tất cả controller operations:

```python
class Promise:
    @staticmethod
    def resolve(data) -> Promise      # Success with data
    @staticmethod
    def reject(code, msg) -> Promise  # Failure with error code

    def ok() -> bool                  # Check success
    def data() -> T                   # Get data (if ok)
    def msg() -> str                  # Get error message (if not ok)
```

**Usage pattern**:
```python
p = await some_operation()
if not p.ok():
    return p  # Propagate error
result = p.data()
```

### 13.2 Error Propagation

```
Controller → Promise.reject(CODE, msg)
    ↓
API Handler → BaseResponse(errno=CODE, errmsg=msg)
    ↓
Middleware → JSONResponse(status_code=200, content=BaseResponse)
    (Note: HTTP status is always 200, error in errno field)

Exception → Middleware catches → 500 JSON with traceback
```

### 13.3 Buffer Failure Recovery

```python
try:
    result = await BLOBS_PROCESS[blob_type](blobs)
    if not result.ok():
        buffer.status = "failed"  # Recoverable
    else:
        buffer.status = "done"
        delete_blobs()  # If non-persistent
except Exception:
    buffer.status = "failed"  # Recoverable
    raise  # Re-raise for logging
```

---

## 14. API Contract Details

### 14.1 Standard Response Envelope

```json
{
  "data": { ... } | null,
  "errno": 0,
  "errmsg": ""
}
```

**Note**: HTTP status code is always 200 for business errors. `errno` field carries the actual error code.

### 14.2 Key Request/Response Examples

**Insert Blob**:
```
POST /api/v1/blobs/insert/{user_id}
Body: {"blob_type": "chat", "blob_data": {"messages": [...]}}
Response: {"data": {"id": "uuid", "chat_results": [...]}, "errno": 0}
```

**Get Context**:
```
GET /api/v1/users/context/{user_id}?max_token_size=500&prefer_topics=basic_info
Response: {"data": {"context": "# Memory\n..."}, "errno": 0}
```

**Flush Buffer**:
```
POST /api/v1/users/buffer/{user_id}/chat
Response: {"data": [{"event_id": "uuid", "add_profiles": [...], ...}], "errno": 0}
```

---

## 15. Cross-Cutting Concerns

### 15.1 Multi-Language Support

| Component | EN Files | ZH Files |
|-----------|---------|---------|
| Profile Extract | `extract_profile.py` | `zh_extract_profile.py` |
| Profile Merge | `merge_profile_yolo.py` | `zh_merge_profile_yolo.py` |
| Entry Summary | `summary_entry_chats.py` | `zh_summary_entry_chats.py` |
| Profile Topics | `user_profile_topics.py` | `zh_user_profile_topics.py` |
| Context Pack | `chat_context_pack.py` | `chat_context_pack.py` (shared) |

Language selection: `project_config.language || CONFIG.language || "en"`

### 15.2 Background Task Pattern

```python
# Async fire-and-forget for non-critical operations
asyncio.create_task(project_cost_token_billing(project_id, in_tokens, out_tokens))
```

Used in: billing token tracking, buffer auto-flush after insert.

### 15.3 UUID Validation

All UUID path parameters are validated before DB queries to prevent SQL injection and invalid lookups.
