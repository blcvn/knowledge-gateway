# Product Requirements Document (PRD)

## Supermemory — Memory & Context Engine for AI

| Metadata          | Value                                         |
|-------------------|-----------------------------------------------|
| **Product Name**  | Supermemory                                   |
| **Version**       | 4.0.0                                         |
| **Date**          | 2026-05-09                                    |
| **Status**        | Production                                    |
| **License**       | MIT                                           |
| **Maintainer**    | Supermemory Team                               |

---

## 1. Tổng Quan Sản Phẩm

### 1.1. Tầm Nhìn (Vision)

Supermemory là **lớp bộ nhớ và ngữ cảnh (Memory & Context Layer)** dành cho AI. Hệ thống giải quyết vấn đề cốt lõi: **AI quên mọi thứ giữa các cuộc hội thoại**. Supermemory tự động học từ các cuộc hội thoại, trích xuất sự kiện, xây dựng hồ sơ người dùng, xử lý cập nhật và mâu thuẫn kiến thức, tự động quên thông tin hết hạn và cung cấp ngữ cảnh phù hợp đúng lúc.

### 1.2. Sứ Mệnh (Mission)

Xây dựng engine bộ nhớ AI state-of-the-art, đạt **#1 trên cả ba benchmark lớn** về AI memory:

| Benchmark | Đo Lường | Kết Quả |
|-----------|----------|---------|
| **LongMemEval** | Bộ nhớ dài hạn qua nhiều phiên với cập nhật kiến thức | **81.6% — #1** |
| **LoCoMo** | Nhớ lại sự kiện qua hội thoại mở rộng (single-hop, multi-hop, temporal, adversarial) | **#1** |
| **ConvoMem** | Cá nhân hóa và học sở thích | **#1** |

### 1.3. Đối Tượng Người Dùng (Target Users)

| Persona | Mô Tả | Nhu Cầu Chính |
|---------|--------|---------------|
| **AI Developer** | Nhà phát triển xây dựng AI agents/apps | API để thêm memory, RAG, user profiles, connectors vào ứng dụng |
| **AI Power User** | Người dùng sử dụng các công cụ AI hàng ngày | Bộ nhớ bền vững (persistent memory) cho Claude, Cursor, VS Code, v.v. |
| **Enterprise Team** | Đội phát triển sản phẩm AI quy mô lớn | Multi-tenant, scalable memory infrastructure |

---

## 2. Kiến Trúc Hệ Thống

### 2.1. Turbo Monorepo Architecture

```
supermemory/
├── apps/
│   ├── web/              # Next.js Web Application (Consumer & Console)
│   ├── mcp/              # Model Context Protocol Server (Cloudflare Workers)
│   ├── docs/             # Documentation Site (Mintlify)
│   ├── browser-extension/ # Chrome Extension
│   ├── memory-graph-playground/ # Graph Visualization Playground
│   └── raycast-extension/ # Raycast Extension
├── packages/
│   ├── lib/              # Shared library (API client, auth, queries)
│   ├── validation/       # Zod schemas & API validation
│   ├── memory-graph/     # Memory Graph visualization component
│   ├── tools/            # Framework integration tools (AI SDK, Mastra, OpenAI, etc.)
│   ├── ai-sdk/           # Vercel AI SDK integration
│   ├── hooks/            # Shared React hooks
│   ├── ui/               # Shared UI components
│   └── ...               # Python SDKs & other integrations
└── skills/               # Agent skills (MemoryBench)
```

### 2.2. Technology Stack

| Component | Technology |
|-----------|-----------|
| **Runtime** | Cloudflare Workers (API), Node.js (Web) |
| **Web Framework** | Next.js (Web), Hono (API & MCP) |
| **Language** | TypeScript |
| **Package Manager** | Bun |
| **Monorepo** | Turborepo |
| **Database** | PostgreSQL via Hyperdrive |
| **ORM** | Drizzle ORM |
| **Authentication** | Better Auth (Session + API Key) |
| **Schema Validation** | Zod + Zod OpenAPI |
| **AI/Embedding** | Cloudflare AI, OpenAI, Anthropic, Google GenAI, Cerebras |
| **Monitoring** | Sentry |
| **Analytics** | PostHog |
| **UI Components** | Radix UI + TanStack React Query |
| **Deployment** | Cloudflare Workers + Pages |

### 2.3. Core Architecture Flow

```
User App / AI Tool
       ↓
  Supermemory API (Hono on Cloudflare Workers)
       │
       ├── Memory Engine     → Trích xuất sự kiện, theo dõi cập nhật,
       │                       giải quyết mâu thuẫn, tự động quên
       ├── User Profiles     → Static facts + Dynamic context, luôn cập nhật
       ├── Hybrid Search     → RAG + Memory trong một truy vấn
       ├── Connectors        → Đồng bộ real-time từ Google Drive, Gmail, Notion, GitHub…
       └── File Processing   → PDFs, images, videos, code → searchable chunks
```

---

## 3. Tính Năng Sản Phẩm (Product Features)

### 3.1. Memory Engine (Lõi — Core)

#### 3.1.1. Memory Extraction
- Tự động trích xuất facts từ nội dung đầu vào (conversations, documents, URLs)
- Phân loại bộ nhớ: **Facts**, **Preferences**, **Episodes**
- Hỗ trợ entity context để hướng dẫn trích xuất memory

#### 3.1.2. Knowledge Graph
- Xây dựng đồ thị kiến thức sống (living knowledge graph)
- Ba loại quan hệ memory:
  - **Updates**: Khi thông tin mới mâu thuẫn/thay thế thông tin cũ
  - **Extends**: Khi thông tin mới bổ sung thêm chi tiết
  - **Derives**: Khi hệ thống suy luận ra liên kết mới từ patterns
- Version control cho memory entries (version, isLatest, parentMemoryId, rootMemoryId)
- Theo dõi nguồn (sourceCount, sourceRelevanceScore)

#### 3.1.3. Automatic Forgetting
- **Time-based forgetting**: Tự động quên thông tin tạm thời khi hết hạn (forgetAfter)
- **Contradiction resolution**: Khi facts mới mâu thuẫn, cập nhật isLatest
- **Noise filtering**: Nội dung không có ý nghĩa không trở thành memory vĩnh viễn

#### 3.1.4. Memory Types
- **Static memories** (isStatic): Sự kiện lâu dài, bền vững
- **Dynamic memories**: Ngữ cảnh gần đây, trạng thái tạm thời
- **Inferred memories** (isInference): Suy luận từ patterns

### 3.2. User Profiles

- Tự động duy trì hồ sơ người dùng từ tất cả tương tác
- **Static Profile**: Sự kiện ổn định dài hạn (role, preferences, expertise)
- **Dynamic Profile**: Ngữ cảnh gần đây (current projects, recent activities)
- Một lệnh gọi, ~50ms latency
- Inject trực tiếp vào system prompt cho AI agents

### 3.3. Hybrid Search

- **RAG + Memory** trong một truy vấn duy nhất
- Search modes:
  - `hybrid` (default): Kết hợp document chunks + personalized memories
  - `memories`: Chỉ tìm kiếm memories
  - `documents`: Chỉ tìm kiếm documents
- Semantic search với vector embeddings (Matryoshka embeddings support)
- Configurable thresholds (chunkThreshold, documentThreshold, 0–1)
- Query rewriting để tối ưu kết quả (tăng latency ~400ms)
- Reranking kết quả
- Metadata filtering (AND/OR/numeric operators)

### 3.4. Document Processing Pipeline

| Stage | Mô Tả |
|-------|--------|
| **Queued** | Document chờ xử lý |
| **Extracting** | Trích xuất nội dung (OCR, transcription, HTML parsing) |
| **Chunking** | Tạo memory chunks (AST-aware cho code) |
| **Embedding** | Tạo vector embeddings |
| **Indexing** | Xây dựng relationships, cập nhật knowledge graph |
| **Done** | Hoàn tất, searchable |

#### Supported Content Types
| Type | Mô Tả |
|------|--------|
| `text` | Plaintext |
| `pdf` | PDF documents |
| `tweet` | Twitter/X posts |
| `google_doc` | Google Docs |
| `google_slide` | Google Slides |
| `google_sheet` | Google Sheets |
| `image` | Images (OCR) |
| `video` | Videos (transcription) |
| `notion_doc` | Notion documents |
| `webpage` | Web pages |
| `onedrive` | OneDrive files |

### 3.5. Connectors (External Data Sync)

| Provider | Capabilities |
|----------|-------------|
| **Google Drive** | Auto-sync documents, real-time webhooks |
| **Gmail** | Email content sync |
| **Notion** | Workspace sync |
| **OneDrive** | Cloud storage sync |
| **GitHub** | Repository content |
| **Web Crawler** | Crawl & index websites |

- Cron triggers mỗi 4 giờ cho connection imports
- Hỗ trợ custom OAuth keys cho mỗi provider
- Document limit configurable per connection (max 10,000)
- Container tag scoping cho mỗi connection

### 3.6. MCP Server (Model Context Protocol)

- MCP v4.0.0 server chạy trên Cloudflare Workers với Durable Objects
- Registered tools:
  - `memory`: Save hoặc forget thông tin
  - `recall`: Tìm kiếm memories + user profile
  - `listProjects`: Liệt kê available projects
  - `whoAmI`: Lấy thông tin user đang đăng nhập
  - `memory-graph`: Trực quan hóa memory graph (MCP App UI)
  - `fetch-graph-data`: Pagination cho graph data
- Registered resources:
  - `supermemory://profile`: User profile resource
  - `supermemory://projects`: Projects list
  - `ui://memory-graph/mcp-app.html`: Interactive graph UI
- Registered prompts:
  - `context`: System context injection với user profile
- OAuth & API Key authentication

### 3.7. Framework Integrations

| Framework | Package |
|-----------|---------|
| **Vercel AI SDK** | `@supermemory/tools/ai-sdk` |
| **LangChain** | Integration module |
| **LangGraph** | Integration module |
| **OpenAI Agents SDK** | `@supermemory/tools/openai` |
| **Mastra** | `@supermemory/tools/mastra` |
| **Agno** | Integration module |
| **Claude Memory Tool** | `@supermemory/tools/claude-memory` |
| **n8n** | Workflow integration |
| **Voltagent** | `@supermemory/tools/voltagent` |
| **Pipecat** | Python SDK |
| **Cartesia** | Python SDK |

### 3.8. Web Console Application

- Next.js web application tại `app.supermemory.ai`
- **Tính năng chính**:
  - Dashboard quản lý memories, documents, connections
  - Memory Graph visualization (interactive force-directed graph)
  - Analytics & usage reporting (API usage, latency, tokens, cost savings)
  - Project management (CRUD với container tags)
  - API key management
  - Organization settings
  - Connection management (Google Drive, Notion, OneDrive)
  - Embedded AI agent (Nova)
- **Authentication**: Better Auth với session cookies + organization support
- **UI Stack**: Radix UI, TanStack React Query, Recharts, Sonner (toasts)

### 3.9. Browser Extension & Raycast

- Chrome extension cho web browsing context
- Raycast extension cho macOS productivity

---

## 4. API Design

### 4.1. API Versioning

| Version | Base URL | Status |
|---------|----------|--------|
| v3 | `/v3/*` | Production |
| v4 | `/v4/*` (search) | Production (Memory Search) |

### 4.2. Core API Endpoints

| Method | Endpoint | Mô Tả |
|--------|----------|--------|
| `POST` | `/v3/documents` | Thêm memory (text, URL, file) |
| `POST` | `/v3/documents/list` | Liệt kê memories với pagination |
| `POST` | `/v3/documents/documents` | Liệt kê documents với memory entries |
| `GET` | `/v3/documents/:id` | Lấy chi tiết memory |
| `DELETE` | `/v3/documents/:id` | Xóa memory |
| `DELETE` | `/v3/documents/bulk` | Xóa hàng loạt memories |
| `POST` | `/v3/search` | Semantic search (RAG + Memory) |
| `POST` | `/v3/connections/:provider` | Tạo connection mới |
| `GET` | `/v3/connections` | Liệt kê connections |
| `DELETE` | `/v3/connections/:connectionId` | Xóa connection |
| `GET` | `/v3/settings` | Lấy organization settings |
| `PATCH` | `/v3/settings` | Cập nhật settings |
| `POST` | `/v3/settings/reset` | Reset toàn bộ data |
| `GET` | `/v3/projects` | Liệt kê projects |
| `POST` | `/v3/projects` | Tạo project mới |
| `DELETE` | `/v3/projects/:projectId` | Xóa project |
| `GET` | `/v3/analytics/usage` | Usage analytics |
| `GET` | `/v3/analytics/chat` | Chat analytics |
| `GET` | `/v3/analytics/memory` | Memory analytics |
| `GET` | `/v3/container-tags/list` | Liệt kê container tags |
| `POST` | `/v3/documents/migrate-mcp` | Migrate MCP documents |

### 4.3. Authentication

| Method | Cách Sử Dụng |
|--------|-------------|
| **API Key** | `Authorization: Bearer sm_xxx` header |
| **Session Cookie** | Better Auth session (Web Console) |
| **OAuth** | MCP Server OAuth flow |
| **Organization** | RBAC với roles (owner, admin, editor, viewer) |

### 4.4. SDK Clients

| Language | Package | Install |
|----------|---------|---------|
| **TypeScript** | `supermemory` | `npm install supermemory` |
| **Python** | `supermemory` | `pip install supermemory` |

---

## 5. Data Model

### 5.1. Core Entities

```
Organization
  ├── Users (with roles)
  ├── Settings (LLM filtering, custom OAuth keys)
  ├── API Keys
  ├── Connections (Google Drive, Notion, OneDrive)
  └── Spaces (Projects/Container Tags)
       ├── Documents
       │    ├── Chunks (with embeddings)
       │    └── Memory Entries (extracted facts)
       │         ├── Version Chain (parent → root)
       │         ├── Memory Relations (updates/extends/derives)
       │         └── Document Sources (many-to-many)
       └── Members (with roles)
```

### 5.2. Key Schemas

| Entity | Fields Chính |
|--------|-------------|
| **Document** | id, customId, contentHash, orgId, userId, title, content, summary, type, status, metadata, tokenCount, chunkCount, summaryEmbedding |
| **Chunk** | id, documentId, content, embeddedContent, type, position, embedding, matryokshaEmbedding |
| **MemoryEntry** | id, memory, spaceId, version, isLatest, parentMemoryId, rootMemoryId, memoryRelations, isInference, isForgotten, isStatic, forgetAfter, memoryEmbedding |
| **Space** | id, name, orgId, containerTag, visibility, contentTextIndex, indexSize |
| **Connection** | id, provider, accessToken, refreshToken, documentLimit, containerTags |

---

## 6. Subscription & Pricing Tiers

| Tier | Slug | Mô Tả |
|------|------|--------|
| **Pro** | `api_pro` | Basic API access |
| **Scale** | `api_scale` | Higher limits & advanced features |
| **Enterprise** | `api_enterprise` | Custom limits, SLA, dedicated support |

---

## 7. Observability & Quality

### 7.1. Monitoring
- **Sentry**: Error tracking với user & organization context
- **PostHog**: Product analytics (memory added/forgot/search events)
- **Custom logging**: Filtered analytics noise

### 7.2. Benchmarking
- **MemoryBench**: Open-source framework tự đánh giá memory solutions
- Support so sánh head-to-head giữa Supermemory, Mem0, Zep, và các providers khác
- Agent skill cho tự động benchmark: `npx skills add supermemoryai/memorybench`

### 7.3. API Request Tracking
- Mỗi API request được log với: type, duration, statusCode, input/output
- Token usage tracking (originalTokens, finalTokens, tokensSaved)
- Cost tracking (costSavedUSD)

---

## 8. Deployment & Infrastructure

| Component | Platform |
|-----------|----------|
| **API Backend** | Cloudflare Workers |
| **MCP Server** | Cloudflare Workers + Durable Objects |
| **Web Console** | Cloudflare Pages (Open Next) |
| **Database** | PostgreSQL via Cloudflare Hyperdrive |
| **AI/Embeddings** | Cloudflare AI |
| **KV Storage** | Cloudflare KV |
| **Workflows** | Cloudflare Workflows (IngestContentWorkflow) |
| **Cron** | Cloudflare Cron Triggers (mỗi 4 giờ) |

---

## 9. Roadmap & Milestones

| Phase | Tính Năng | Status |
|-------|-----------|--------|
| **v3.x** | Core API, Hybrid Search, Connectors, User Profiles | ✅ Production |
| **v4.0** | Memory Search API, MCP v4.0, Graph Memory | ✅ Production |
| **v4.x** | Enhanced MCP Apps, Memory Graph UI, Slideshow mode | 🔄 In Progress |
| **v5.0** | Advanced inference, multi-modal memory, enterprise governance | 📋 Planned |
