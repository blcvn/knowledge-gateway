# SRS — Memobase: Software Requirements Specification

| Field | Value |
|-------|-------|
| **Product** | Memobase |
| **Version** | 1.0 |
| **Date** | 2026-05-09 |
| **Standard** | IEEE 830-1998 |

---

## 1. Giới thiệu

### 1.1 Mục đích
Tài liệu SRS mô tả chi tiết các yêu cầu kỹ thuật (functional & non-functional) cho hệ thống Memobase — user profile-based memory system cho ứng dụng LLM.

### 1.2 Phạm vi hệ thống
Memobase bao gồm:
- **Server**: FastAPI application xử lý memory pipeline
- **Client SDKs**: Python, TypeScript, Go
- **MCP Server**: Model Context Protocol integration
- **Infrastructure**: PostgreSQL (pgvector), Redis

### 1.3 Thuật ngữ

| Thuật ngữ | Định nghĩa |
|-----------|------------|
| Blob | Đơn vị dữ liệu đầu vào (ChatBlob, DocBlob, SummaryBlob) |
| Buffer Zone | Vùng đệm chứa blobs chưa xử lý |
| Flush | Quá trình xử lý buffer thành profiles và events |
| Profile | Thông tin có cấu trúc về người dùng (topic/sub_topic/content) |
| Event | Sự kiện trên timeline từ conversations |
| Event Gist | Tóm tắt chi tiết của một event |
| Context | Chuỗi text đóng gói profiles + events, sẵn sàng inject vào prompt |
| YOLO Merge | Thuật toán merge profiles trong 1 LLM call duy nhất |

---

## 2. Mô tả tổng quan hệ thống

### 2.1 Kiến trúc tổng thể

```
┌────────────────────────────────────────────────────────────┐
│                    CLIENT LAYER                            │
│  Python SDK │ TypeScript SDK │ Go SDK │ MCP Server │ REST  │
└──────────────────────┬─────────────────────────────────────┘
                       │ HTTP/REST (Bearer Auth)
┌──────────────────────▼─────────────────────────────────────┐
│                    API LAYER (FastAPI)                      │
│  AuthMiddleware → Router → API Handlers                    │
│  Telemetry (OpenTelemetry + Prometheus)                    │
├────────────────────────────────────────────────────────────┤
│                    CONTROLLER LAYER                        │
│  UserCtrl │ BlobCtrl │ BufferCtrl │ ProfileCtrl │ EventCtrl│
│  ContextCtrl │ ProjectCtrl │ BillingCtrl │ RoleplayCtrl   │
├────────────────────────────────────────────────────────────┤
│                    LLM LAYER                               │
│  OpenAI LLM │ Doubao Cache LLM                            │
│  Embedding: OpenAI │ Jina │ Ollama                        │
│  Prompt Templates: EN │ ZH                                │
├────────────────────────────────────────────────────────────┤
│                    DATA LAYER                              │
│  SQLAlchemy ORM │ Redis Cache │ pgvector                  │
├────────────────────────────────────────────────────────────┤
│                    INFRASTRUCTURE                          │
│  PostgreSQL 17 + pgvector │ Redis 7.4 │ Docker Compose    │
└────────────────────────────────────────────────────────────┘
```

### 2.2 Interfaces

| Interface | Protocol | Format |
|-----------|----------|--------|
| Client → Server | HTTP/1.1 REST | JSON |
| Server → PostgreSQL | TCP | SQL (SQLAlchemy) |
| Server → Redis | TCP | RESP protocol |
| Server → LLM Provider | HTTPS | OpenAI Chat Completions API |
| Server → Embedding Provider | HTTPS | OpenAI Embeddings API |
| MCP Server → Memobase | HTTP REST | JSON |

---

## 3. Functional Requirements

### 3.1 Authentication & Authorization Module

#### FR-AUTH-001: Bearer Token Validation
- **Input**: HTTP Header `Authorization: Bearer <token>`
- **Processing**: Middleware intercept mọi request tới `/api/*` (trừ `/api/v1/healthcheck`)
- **Output**: Request cho phép đi tiếp hoặc 401 Unauthorized
- **Rules**:
  - Root token: so sánh trực tiếp với `ACCESS_TOKEN` env var
  - Project token: parse `project_id` → verify `project_secret` trong DB → check `project_status ≠ suspended`

#### FR-AUTH-002: Multi-Project Isolation
- Mỗi request gắn `project_id` vào `request.state.memobase_project_id`
- Tất cả DB queries phải filter theo `project_id`
- Composite primary key: `(id, project_id)` trên tất cả tables

### 3.2 User Management Module

#### FR-USER-001: Create User
- **Endpoint**: `POST /api/v1/users`
- **Input**: `{ "data": { "any_key": "any_value" } }` (optional)
- **Processing**: Insert User record với UUID v4, gắn `project_id`
- **Output**: `{ "data": { "id": "<uuid>" }, "errno": 0 }`

#### FR-USER-002: Get User
- **Endpoint**: `GET /api/v1/users/{user_id}`
- **Input**: UUID path parameter
- **Validation**: UUID format check
- **Output**: `{ "data": { "id", "data", "created_at", "updated_at" } }`

#### FR-USER-003: Update User
- **Endpoint**: `PUT /api/v1/users/{user_id}`
- **Processing**: Update `additional_fields` JSONB

#### FR-USER-004: Delete User
- **Endpoint**: `DELETE /api/v1/users/{user_id}`
- **Processing**: CASCADE delete tất cả related records (blobs, buffers, profiles, events, statuses)

### 3.3 Blob (Data Ingestion) Module

#### FR-BLOB-001: Insert Blob
- **Endpoint**: `POST /api/v1/blobs/insert/{user_id}`
- **Input**: Blob data theo type (chat, doc, summary)
- **Processing**:
  1. Validate blob format
  2. Store blob vào `general_blobs` table
  3. Calculate token_size bằng tiktoken
  4. Insert entry vào `buffer_zones` với status=`idle`
  5. Kiểm tra buffer capacity:
     - Nếu tổng token_size > `max_chat_blob_buffer_token_size` (default 1024) → auto background flush
- **Output**: `{ "data": { "id": "<blob_id>", "chat_results": [...] } }`

#### FR-BLOB-002: ChatBlob Format
```json
{
  "blob_type": "chat",
  "blob_data": {
    "messages": [
      { "role": "user", "content": "..." },
      { "role": "assistant", "content": "..." }
    ]
  }
}
```
- `role` phải là: `user`, `assistant`, hoặc `system`

#### FR-BLOB-003: Non-persistent Blobs
- Mặc định `persistent_chat_blobs = false`
- Sau khi flush thành công, raw blobs bị xóa từ `general_blobs`
- Chỉ profiles và events được lưu trữ

### 3.4 Buffer Zone Module

#### FR-BUF-001: Buffer Flush
- **Endpoint**: `POST /api/v1/users/buffer/{user_id}/{buffer_type}`
- **Processing**:
  1. Query tất cả buffers với status=`idle` cho user
  2. Update status → `processing`
  3. Join với `general_blobs` để lấy blob_data
  4. Gọi Memory Processing Pipeline (Section 3.5)
  5. Thành công → status=`done`, xóa blobs (nếu non-persistent)
  6. Thất bại → status=`failed`
- **Concurrency**: Status-based locking ngăn parallel flush duplicate

#### FR-BUF-002: Buffer Capacity Query
- **Endpoint**: `GET /api/v1/users/buffer/capacity/{user_id}/{buffer_type}`
- **Output**: Số lượng buffer items đang ở status=`idle`

### 3.5 Memory Processing Pipeline

#### FR-MEM-001: Profile Extraction
- **Trigger**: Buffer flush
- **Processing** (cố định 3 LLM calls):
  1. **Call 1 — Extract**: Gọi LLM với prompt `extract_profile` để trích xuất profile facts từ conversations
  2. **Call 2 — Merge (YOLO)**: Gọi LLM với prompt `merge_profile_yolo` để merge profiles mới với existing profiles (add/update/delete)
  3. **Call 3 — Summarize Events**: Gọi LLM với prompt `summary_entry_chats` để tạo event summary và gists
- **Multi-language**: Sử dụng prompt templates EN hoặc ZH tùy config

#### FR-MEM-002: Profile Merge Logic
- **Add**: Profile mới không trùng topic/sub_topic → insert
- **Update**: Profile trùng topic/sub_topic → update content
- **Delete**: Profile cũ bị phủ nhận bởi thông tin mới → delete
- **Token limit**: Mỗi profile entry ≤ `max_pre_profile_token_size` (default 128)
- **Subtopic limit**: Mỗi topic ≤ `max_profile_subtopics` (default 15)

#### FR-MEM-003: Event Processing
- Extract event_tip (summary) và event_tags từ conversations
- Event gists: Split event_tip thành từng dòng bắt đầu bằng `-`
- Nếu `enable_event_embedding = true`:
  - Tạo embeddings cho event và từng gist
  - Store vào `user_events.embedding` và `user_event_gists.embedding`

### 3.6 Profile Module

#### FR-PROF-001: Get User Profiles
- **Endpoint**: `GET /api/v1/users/profile/{user_id}`
- **Caching**: Check Redis key `user_profiles::{project_id}::{user_id}`
  - Cache hit → return deserialized data
  - Cache miss → query DB → cache với TTL=`cache_user_profiles_ttl` (default 1200s)
- **Output**: List of `{ id, content, attributes: { topic, sub_topic }, created_at, updated_at }`

#### FR-PROF-002: Profile Truncation
- Support parameters: `prefer_topics`, `only_topics`, `max_token_size`, `topk`, `max_subtopic_size`, `topic_limits`
- Sort by `updated_at DESC` (most recent first)
- Priority ordering khi có `prefer_topics`
- Token counting bằng tiktoken (gpt-4o encoder)

#### FR-PROF-003: Manual Profile CRUD
- Add: `POST /api/v1/users/profile/{user_id}`
- Update: `PUT /api/v1/users/profile/{user_id}/{profile_id}`
- Delete: `DELETE /api/v1/users/profile/{user_id}/{profile_id}`
- Mỗi operation invalidate Redis cache

### 3.7 Event Module

#### FR-EVT-001: Get User Events
- **Endpoint**: `GET /api/v1/users/event/{user_id}`
- **Parameters**: `topk` (default 10), `time_range_in_days` (default 21)
- **Output**: Events sorted by `created_at DESC`

#### FR-EVT-002: Semantic Search Events
- **Endpoint**: `GET /api/v1/users/event/search/{user_id}`
- **Requires**: `enable_event_embedding = true`
- **Processing**:
  1. Embed query text
  2. Cosine similarity search qua pgvector
  3. Filter by `similarity > threshold` (default 0.2)
  4. Filter by `time_range_in_days`
  5. Order by similarity DESC, limit topk

#### FR-EVT-003: Event Gist Search
- **Endpoint**: `GET /api/v1/users/event_gist/search/{user_id}`
- Tương tự FR-EVT-002 nhưng trên `user_event_gists` table
- Fine-grained results so với event-level search

#### FR-EVT-004: Event Tag Filtering
- **Endpoint**: `GET /api/v1/users/event_tags/search/{user_id}`
- **Parameters**: `has_event_tag` (list), `event_tag_equal` (dict)
- **Processing**: JSONB containment query `@>` trên `event_data.event_tags`

### 3.8 Context Module

#### FR-CTX-001: Get User Context
- **Endpoint**: `GET /api/v1/users/context/{user_id}`
- **Parameters**:
  - `max_token_size`: Tổng token budget
  - `prefer_topics`: Danh sách topics ưu tiên
  - `only_topics`: Filter chỉ certain topics
  - `profile_event_ratio`: Tỷ lệ token cho profiles (0-1, default ~0.7)
  - `chats`: Latest messages để search relevant events
  - `customize_context_prompt`: Custom template
  - `event_similarity_threshold`, `time_range_in_days`
- **Processing**:
  1. Parallel fetch: profiles + event gists (asyncio.gather)
  2. Truncate profiles theo token budget
  3. Calculate remaining tokens cho events
  4. Truncate events
  5. Assemble context string qua prompt template

#### FR-CTX-002: Context Output Format
```
# Memory
Unless the user has relevant queries, do not actively mention those memories.
## User Background:
- topic::sub_topic: content
...
## Latest Events:
- event gist content
...
```

### 3.9 Project Configuration Module

#### FR-PROJ-001: Profile Config Update
- **Endpoint**: `POST /api/v1/project/profile_config`
- **Input**: YAML string chứa profile configuration
- **Stored**: `projects.profile_config` column
- **Overridable fields**: `language`, `profile_strict_mode`, `profile_validate_mode`, `additional_user_profiles`, `overwrite_user_profiles`, `event_tags`

#### FR-PROJ-002: Usage & Billing
- **Billing**: `GET /api/v1/project/billing` → `token_left`, `next_refill_at`
- **Usage**: `GET /api/v1/project/usage` → Daily aggregation: `total_insert`, `total_input_token`, `total_output_token`

---

## 4. Non-Functional Requirements

### 4.1 Performance

| ID | Requirement | Specification |
|----|------------|---------------|
| NFR-P01 | Context API latency | P99 < 100ms (excluding external API calls) |
| NFR-P02 | DB connection pool | pool_size=75, max_overflow=50, pool_timeout=45s |
| NFR-P03 | Pool recycle | pool_recycle=300s, pool_pre_ping=True |
| NFR-P04 | Redis connection | Connection pool with decode_responses=True |
| NFR-P05 | Profile cache TTL | Default 1200s (20 minutes) |
| NFR-P06 | Parallel execution | Profile + Event fetch dùng asyncio.gather |
| NFR-P07 | Token counting | tiktoken (gpt-4o encoder) cho accurate counting |

### 4.2 Reliability

| ID | Requirement | Specification |
|----|------------|---------------|
| NFR-R01 | Buffer failure recovery | Status-based: `idle → processing → done/failed` |
| NFR-R02 | Transaction rollback | SQLAlchemy session rollback on exception |
| NFR-R03 | Embedding fallback | Nếu embedding API fail → store `None`, skip search |
| NFR-R04 | Embedding dimension validation | Runtime check tại startup, raise error nếu mismatch |
| NFR-R05 | Health checks | DB health: `pg_isready`, Redis: `PING`, API: `/healthcheck` |
| NFR-R06 | Docker restart policy | `unless-stopped` cho DB và Redis services |

### 4.3 Security

| ID | Requirement | Specification |
|----|------------|---------------|
| NFR-S01 | Authentication | Bearer token trên mọi API endpoint (trừ healthcheck) |
| NFR-S02 | Project isolation | Composite PK `(id, project_id)`, FK constraints với CASCADE |
| NFR-S03 | Data minimization | Raw blobs xóa sau processing (default) |
| NFR-S04 | CORS | Configurable via `USE_CORS` env var, whitelist `API_HOSTS` |
| NFR-S05 | Project table immutability | SQLAlchemy event listeners ngăn unauthorized insert/update/delete |
| NFR-S06 | UUID validation | UUID format check trước khi query DB |

### 4.4 Scalability

| ID | Requirement | Specification |
|----|------------|---------------|
| NFR-SC01 | Horizontal scaling | Stateless API server, shared PG + Redis |
| NFR-SC02 | DB indexes | Composite indexes trên (user_id, project_id) cho tất cả tables |
| NFR-SC03 | Vector search | pgvector extension cho efficient ANN search |
| NFR-SC04 | Buffer batching | Batch process thay vì per-message processing |

### 4.5 Observability

| ID | Requirement | Specification |
|----|------------|---------------|
| NFR-O01 | Logging | Structured logging via structlog (JSON format option) |
| NFR-O02 | Tracing | Request ID tracking (X-Request-ID header) |
| NFR-O03 | Metrics | OpenTelemetry counters: request count, healthcheck count |
| NFR-O04 | Histograms | Request latency (ms) per path/method/project |
| NFR-O05 | Pool monitoring | DB pool utilization logging (warning at > 80%) |
| NFR-O06 | Response timing | X-Process-Time header trên mọi response |

### 4.6 Compatibility

| ID | Requirement | Specification |
|----|------------|---------------|
| NFR-C01 | Python | ≥ 3.12 (server), ≥ 3.11 (client SDK) |
| NFR-C02 | PostgreSQL | 17 với pgvector extension |
| NFR-C03 | Redis | ≥ 7.4 |
| NFR-C04 | LLM Providers | OpenAI SDK Compatible (OpenAI, vLLM, Ollama, Doubao) |
| NFR-C05 | Embedding Providers | OpenAI, Jina, Ollama |
| NFR-C06 | Docker | linux/amd64 platform |
| NFR-C07 | DB Migration | Alembic autogenerate support |

---

## 5. Database Schema

### 5.1 Tables

| Table | Primary Key | Description |
|-------|------------|-------------|
| `projects` | `project_id` | Multi-tenant project registry |
| `billings` | `id` | Billing records with usage_left |
| `project_billings` | `(project_id, billing_id)` | Project-billing association |
| `users` | `(id, project_id)` | User records with custom metadata |
| `general_blobs` | `(id, project_id)` | Raw input data (chat, doc, summary) |
| `buffer_zones` | `(id, project_id)` | Processing queue entries |
| `user_profiles` | `(id, project_id)` | Structured user profile entries |
| `user_events` | `(id, project_id)` | Timeline events with vector embeddings |
| `user_event_gists` | `(id, project_id)` | Fine-grained event descriptions |
| `user_statuses` | `(id, project_id)` | User state tracking |

### 5.2 Key Indexes

| Table | Index | Columns |
|-------|-------|---------|
| users | idx_users_id_project_id | (id, project_id) |
| general_blobs | idx_general_blobs_user_id_project_id | (user_id, project_id) |
| general_blobs | idx_general_blobs_user_id_blob_type | (user_id, project_id, blob_type) |
| buffer_zones | idx_buffer_zones_user_id_blob_type | (user_id, project_id, blob_type, status) |
| user_profiles | idx_user_profiles_user_id_project_id | (user_id, project_id) |
| user_events | idx_user_events_user_id_project_id | (user_id, project_id) |
| user_event_gists | idx_user_event_gists_user_id_project_id | (user_id, project_id) |

### 5.3 Foreign Key Constraints
- Tất cả tables có FK tới `users(id, project_id)` với `ON DELETE CASCADE`
- `buffer_zones` có FK tới cả `users` và `general_blobs`
- `user_event_gists` có FK tới cả `users` và `user_events`
- `project_billings` có FK tới `projects` và `billings`

---

## 6. Error Handling

### 6.1 Error Code System

| Code | Name | HTTP Status | Mô tả |
|------|------|-------------|-------|
| 0 | SUCCESS | 200 | Thành công |
| 400 | BAD_REQUEST | 400 | Request không hợp lệ |
| 401 | UNAUTHORIZED | 401 | Thiếu hoặc sai token |
| 403 | FORBIDDEN | 403 | Project bị suspended |
| 404 | NOT_FOUND | 404 | Resource không tồn tại |
| 500 | INTERNAL_SERVER_ERROR | 500 | Lỗi server |
| 501 | NOT_IMPLEMENTED | 501 | Feature chưa enable |
| 520 | SERVER_PARSE_ERROR | 520 | Lỗi parse dữ liệu |

### 6.2 Response Format
```json
{
  "data": { ... } | null,
  "errno": 0,
  "errmsg": ""
}
```

---

## 7. Deployment Specifications

### 7.1 Docker Compose Services

| Service | Image | Internal Port | External Port | Health Check |
|---------|-------|--------------|---------------|-------------|
| memobase-server-db | pgvector/pgvector:pg17 | 5432 | ${DATABASE_EXPORT_PORT} | pg_isready |
| memobase-server-redis | redis:7.4 | 6379 | ${REDIS_EXPORT_PORT} | redis-cli ping |
| memobase-server-api | Custom build | 8000 | ${API_EXPORT_PORT} | depends_on healthy |

### 7.2 Environment Variables

| Variable | Required | Default | Mô tả |
|----------|---------|---------|-------|
| DATABASE_URL | Yes | - | PostgreSQL connection string |
| REDIS_URL | Yes | - | Redis connection string |
| ACCESS_TOKEN | Yes | - | Root authentication token |
| PROJECT_ID | No | "default" | Default project identifier |
| API_HOSTS | No | memobase.dev | Swagger server URLs |
| USE_CORS | No | false | Enable CORS middleware |
| LOG_FORMAT | No | plain | "plain" or "json" |

### 7.3 Configuration File (config.yaml)

```yaml
# Required
llm_api_key: <OpenAI API Key>

# Optional
llm_base_url: null                    # Custom LLM endpoint
best_llm_model: gpt-4o-mini          # Primary model
language: en                          # en | zh
enable_event_embedding: true          # Toggle semantic search
embedding_provider: openai            # openai | jina | ollama
embedding_model: text-embedding-3-small
embedding_dim: 1536
max_chat_blob_buffer_token_size: 1024 # Buffer flush threshold
max_profile_subtopics: 15            # Max subtopics per topic
profile_strict_mode: false           # Only collect defined profiles
persistent_chat_blobs: false         # Keep raw chats after processing
```

---

## 8. Traceability Matrix

| User Story | Functional Req | Non-Functional Req |
|-----------|---------------|-------------------|
| US-001 | FR-USER-001 | NFR-S02 |
| US-010 | FR-BLOB-001, FR-BLOB-002 | NFR-P04 |
| US-012 | FR-BUF-001 | NFR-R01 |
| US-020 | FR-MEM-001, FR-MEM-002 | NFR-P01, NFR-P06 |
| US-021 | FR-PROJ-001 | NFR-C04 |
| US-031 | FR-EVT-002 | NFR-SC03 |
| US-040 | FR-CTX-001, FR-CTX-002 | NFR-P01, NFR-P05 |
| US-050 | FR-PROJ-001 | NFR-C07 |
| US-053 | FR-AUTH-001 | NFR-R05, NFR-O03 |
| US-060 | - (Client SDK) | NFR-C01 |
| US-070 | - (Deployment) | NFR-R06, NFR-SC01 |
