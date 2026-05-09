# Architecture Document

## Supermemory — Memory & Context Engine for AI

| Metadata | Value |
|----------|-------|
| **Version** | 4.0.0 |
| **Date** | 2026-05-09 |
| **Architecture Style** | Serverless Monorepo · Edge-First · Event-Driven |

---

## 1. Tổng Quan Kiến Trúc

### 1.1. Architecture Vision

Supermemory là hệ thống **Edge-First Serverless** chạy hoàn toàn trên Cloudflare infrastructure. Kiến trúc được thiết kế theo mô hình **Turbo Monorepo** với shared packages, tối ưu cho latency thấp (< 100ms profile retrieval) và auto-scaling không giới hạn.

### 1.2. High-Level Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                                    │
├──────────┬──────────┬──────────┬──────────┬──────────┬─────────────────┤
│ AI SDKs  │ MCP      │ Browser  │ Web      │ Raycast  │ Framework       │
│ (TS/Py)  │ Clients  │ Extension│ Console  │ Ext      │ Integrations    │
└────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬────────────┘
     │          │          │          │          │          │
     ▼          ▼          │          ▼          │          ▼
┌─────────────────────┐    │   ┌──────────────┐  │  ┌──────────────────┐
│  API Backend        │    │   │  Web Console │  │  │  Framework Tools │
│  (Cloudflare        │◄───┘   │  (Cloudflare │  │  │  (AI SDK/Mastra/ │
│   Workers + Hono)   │◄──────►│   Pages)     │  │  │   OpenAI/Claude) │
└────────┬────────────┘        └──────┬───────┘  │  └────────┬─────────┘
         │                            │          │           │
         ▼                            │          │           │
┌─────────────────────┐               │          │           │
│  MCP Server         │◄──────────────┘          │           │
│  (Workers +         │◄─────────────────────────┘           │
│   Durable Objects)  │◄─────────────────────────────────────┘
└────────┬────────────┘
         │
         ▼
┌────────────────────────────────────────────────────────────────────────┐
│                     PLATFORM LAYER (Cloudflare)                        │
├──────────────┬──────────────┬──────────────┬──────────────┬────────────┤
│  Hyperdrive  │  AI Gateway  │  KV Storage  │  Workflows   │  Cron      │
│  (DB Pool)   │  (Embed/LLM) │  (Cache)     │  (Ingest)    │  Triggers  │
└──────┬───────┴──────────────┴──────────────┴──────────────┴────────────┘
       │
       ▼
┌────────────────────┐
│   PostgreSQL       │
│   (Primary DB)     │
└────────────────────┘
```

---

## 2. Monorepo Topology

### 2.1. Package Dependency Graph

```
                      ┌──────────────────┐
                      │    turbo.json     │
                      │  (Build System)   │
                      └────────┬─────────┘
                               │
            ┌──────────────────┼──────────────────┐
            │                  │                  │
     ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
     │   apps/web  │   │  apps/mcp   │   │  apps/docs  │
     │  (Next.js)  │   │ (CF Worker) │   │ (Mintlify)  │
     └──────┬──────┘   └──────┬──────┘   └─────────────┘
            │                 │
            │      ┌──────────┤
            │      │          │
     ┌──────▼──────▼──┐  ┌───▼────────────┐
     │  packages/lib  │  │ packages/tools │
     │ (API, Auth,    │  │ (AI SDK, Mastra│
     │  Queries)      │  │  OpenAI, etc.) │
     └──────┬─────────┘  └───┬────────────┘
            │                │
     ┌──────▼────────────────▼──┐
     │   packages/validation    │
     │  (Zod Schemas, API Spec) │
     └──────┬───────────────────┘
            │
     ┌──────▼──────────────┐    ┌────────────────────┐
     │ packages/memory-    │    │   packages/ui       │
     │ graph (Visualization│    │  (Shared Components)│
     └────────────────────┘    └────────────────────┘
```

### 2.2. Package Inventory

| Package | Type | Runtime | Dependencies | Purpose |
|---------|------|---------|-------------|---------|
| `apps/web` | Application | Cloudflare Pages (Open Next) | lib, validation, memory-graph, ui | Web Console UI |
| `apps/mcp` | Application | Cloudflare Workers + DO | supermemory SDK | MCP Protocol Server |
| `apps/docs` | Application | Mintlify | — | Documentation site |
| `apps/browser-extension` | Application | Chrome | — | Browser context capture |
| `packages/validation` | Shared | Universal | zod, zod-openapi | Schema definitions, API types |
| `packages/lib` | Shared | Browser | better-auth, @tanstack/react-query | Auth, API client, queries |
| `packages/tools` | Shared | Node.js | supermemory, ai (Vercel) | Framework integration tools |
| `packages/memory-graph` | Shared | Browser | d3-force | Interactive graph visualization |
| `packages/ui` | Shared | Browser | radix-ui | Shared UI components |
| `packages/hooks` | Shared | Browser | react | Shared React hooks |

---

## 3. Component Architecture

### 3.1. API Backend (Cloudflare Workers + Hono)

```
┌─ API Worker (Hono) ─────────────────────────────────────────────┐
│                                                                  │
│  ┌─ Middleware Stack ─────────────────────────────────────────┐  │
│  │  CORS → Auth (Better Auth / API Key) → Sentry → Logger    │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─ Route Groups ────────────────────────────────────────────┐  │
│  │  /v3/documents   → Memory CRUD, Processing, Bulk Ops     │  │
│  │  /v3/search      → Semantic Search (v3 RAG)               │  │
│  │  /v4/search      → Memory Search (v4 Facts)               │  │
│  │  /v3/connections → External Data Connectors                │  │
│  │  /v3/settings    → Organization Config                     │  │
│  │  /v3/analytics   → Usage Reporting                         │  │
│  │  /v3/projects    → Project/Space Management                │  │
│  │  /api/auth/*     → Better Auth Endpoints                   │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─ Cloudflare Bindings ─────────────────────────────────────┐  │
│  │  • Hyperdrive (PostgreSQL connection pooling)              │  │
│  │  • AI (Embedding generation)                               │  │
│  │  • KV (Cache, session storage)                             │  │
│  │  • Workflows (IngestContentWorkflow)                       │  │
│  │  • Cron Triggers (Connection import every 4h)              │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 3.2. MCP Server Architecture

```
┌─ MCP Worker (Hono + Agents Framework) ──────────────────────────┐
│                                                                  │
│  ┌─ Edge Router ─────────────────────────────────────────────┐  │
│  │  GET  /                    → Server info (v4.0.0)          │  │
│  │  GET  /.well-known/oauth-* → OAuth discovery               │  │
│  │  ALL  /mcp                 → Auth middleware → DO dispatch  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─ Auth Layer ──────────────────────────────────────────────┐  │
│  │  isApiKey(token) → sm_ prefix check                        │  │
│  │  validateApiKey() → GET /v3/session                        │  │
│  │  validateOAuthToken() → GET /v3/mcp/session-with-key       │  │
│  │  WWW-Authenticate with resource_metadata for 401s          │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─ Durable Object: SupermemoryMCP ──────────────────────────┐  │
│  │  State: clientInfo, cachedContainerTags (TTL 5min)          │  │
│  │  Storage: SQLite (via Durable Objects)                      │  │
│  │                                                              │  │
│  │  Tools:  memory, recall, listProjects, whoAmI, memory-graph │  │
│  │  Resources: supermemory://profile, supermemory://projects   │  │
│  │  Prompts: context (system prompt injection)                 │  │
│  │                                                              │  │
│  │  → Delegates to SupermemoryClient (wraps supermemory SDK)   │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 3.3. Web Console Architecture (Next.js)

```
┌─ Next.js App (Cloudflare Pages via Open Next) ──────────────────┐
│                                                                  │
│  middleware.ts                                                    │
│  ├── Session cookie check (Better Auth)                          │
│  ├── Public paths: /login, /login/new, ?view=mcp                │
│  ├── API routes: 401 if no session                               │
│  └── Redirect to /login for unauthenticated                     │
│                                                                  │
│  app/                                                            │
│  ├── (auth)/        → Login, signup flows                        │
│  ├── (app)/         → Main dashboard (memories, graph, settings) │
│  │   ├── page.tsx   → Dashboard with memory list & management    │
│  │   ├── settings/  → Organization settings                      │
│  │   └── onboarding/→ First-time user setup                      │
│  ├── api/           → Server-side API routes                     │
│  ├── auth/          → Better Auth callback handlers              │
│  └── ref/           → Reference/sharing pages                    │
│                                                                  │
│  Instrumentation: Sentry client-side error tracking              │
│  Analytics: PostHog (proxied through /ingest/* rewrite)          │
└──────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Architecture

### 4.1. Entity Relationship Model

```
Organization (1)
  │
  ├── OrganizationSettings (1:1) ── LLM filter config, custom OAuth keys
  ├── Users (1:N via org membership)
  │
  ├── Spaces/Projects (1:N)
  │     │
  │     ├── containerTag: "sm_project_{name}" (unique identifier)
  │     ├── visibility: public | private | unlisted
  │     ├── contentTextIndex: KnowledgeBase (JSON)
  │     │
  │     ├── Documents (1:N via DocumentsToSpaces M:M)
  │     │     ├── Chunks (1:N) ── position-ordered segments
  │     │     │     ├── embedding: float[] (primary)
  │     │     │     ├── matryokshaEmbedding: float[] (Matryoshka)
  │     │     │     └── embeddingNew: float[] (migration support)
  │     │     │
  │     │     └── MemoryDocumentSource (M:M to MemoryEntry)
  │     │           ├── relevanceScore: 0-100
  │     │           └── addedAt: timestamp
  │     │
  │     ├── MemoryEntries (1:N)
  │     │     ├── Version Chain: rootMemoryId → parentMemoryId → current
  │     │     ├── memoryRelations: {memoryId: updates|extends|derives}
  │     │     ├── Lifecycle: isLatest, isForgotten, forgetAfter
  │     │     ├── Classification: isStatic, isInference
  │     │     └── memoryEmbedding: float[]
  │     │
  │     └── SpaceToMembers (M:M with roles)
  │
  └── Connections (1:N) ── Google Drive, Notion, OneDrive
        ├── OAuth tokens (access, refresh, expiry)
        ├── documentLimit: max 10,000
        └── containerTags: scope imported docs
```

### 4.2. Embedding Strategy

| Field | Model | Dimension | Usage |
|-------|-------|-----------|-------|
| `chunk.embedding` | Primary model | Variable | Main search |
| `chunk.matryokshaEmbedding` | Matryoshka model | Variable | Efficient truncated search |
| `chunk.embeddingNew` | Migration model | Variable | Model upgrade support |
| `document.summaryEmbedding` | Primary model | Variable | Document-level similarity |
| `memoryEntry.memoryEmbedding` | Primary model | Variable | Memory search |
| `memoryEntry.memoryEmbeddingNew` | Migration model | Variable | Model upgrade support |

**Similarity Algorithm:** Cosine similarity via dot product on normalized (unit) vectors. Fallback to relevance score (0-100 scale normalized to 0-1) when embeddings unavailable.

---

## 5. Data Flow Pipelines

### 5.1. Content Ingestion Pipeline

```
Input (text/URL/file)
  │
  ▼
┌─ IngestContentWorkflow (Cloudflare Workflow) ───────────────────┐
│                                                                  │
│  1. QUEUED → Content type detection                              │
│     ├── URL → fetch & parse HTML/PDF/media                       │
│     ├── File → binary extraction (PDF, image, video)             │
│     └── Text → direct processing                                 │
│                                                                  │
│  2. EXTRACTING → Content extraction                              │
│     ├── Images → OCR                                             │
│     ├── Videos → Transcription                                   │
│     ├── PDFs → Text extraction                                   │
│     └── HTML → Clean text + metadata                             │
│                                                                  │
│  3. CHUNKING → Semantic segmentation                             │
│     ├── Text → Semantic boundary detection                       │
│     ├── Code → AST-aware chunking                                │
│     └── Content hashing (dedup check)                            │
│                                                                  │
│  4. EMBEDDING → Vector generation                                │
│     ├── Chunk embeddings (Cloudflare AI)                         │
│     ├── Summary embedding                                        │
│     └── Matryoshka embeddings (optional)                         │
│                                                                  │
│  5. INDEXING → Knowledge graph integration                       │
│     ├── AI-powered fact extraction → MemoryEntries               │
│     ├── Relationship detection (updates/extends/derives)         │
│     ├── Auto-tagging and summarization                           │
│     ├── Space relationship management                            │
│     └── forgetAfter scheduling for temporal facts                │
│                                                                  │
│  6. DONE → Fully searchable                                      │
│                                                                  │
│  Processing metadata tracked at each step:                       │
│  { startTime, endTime, status, error, finalStatus }              │
└──────────────────────────────────────────────────────────────────┘
```

### 5.2. Search Pipeline

```
Query
  │
  ▼
┌─ Search Flow ───────────────────────────────────────────────────┐
│                                                                  │
│  1. Optional: Query Rewriting (AI) → +~400ms                    │
│                                                                  │
│  2. Embedding Generation → Query vector                          │
│                                                                  │
│  3. Parallel Search:                                             │
│     ├── Vector similarity on chunks (chunkThreshold filter)      │
│     ├── Vector similarity on memories (v4 memory search)         │
│     └── Document-level similarity (documentThreshold filter)     │
│                                                                  │
│  4. Metadata Filtering:                                          │
│     └── AND/OR logic with string/numeric/boolean operators       │
│                                                                  │
│  5. Container Tag Scoping → orgId + containerTags filter         │
│                                                                  │
│  6. Optional: Reranking (AI) → Re-score by query relevance      │
│                                                                  │
│  7. Context Assembly:                                            │
│     ├── Adjacent chunks (prev/next) unless onlyMatchingChunks    │
│     ├── Related memories (parents/children chain)                │
│     ├── Document summaries (if includeSummary)                   │
│     └── Full documents (if includeFullDocs)                      │
│                                                                  │
│  8. Response: { results[], timing, total }                       │
└──────────────────────────────────────────────────────────────────┘
```

### 5.3. Memory Forgetting Pipeline

```
Forget Request
  │
  ├── By Content (exact match) → memories.forget({content})
  │     │
  │     ├── Found → Mark isForgotten=true, set forgetReason
  │     │
  │     └── Not Found (404) → Fallback to semantic search
  │           │
  │           ├── Similarity > 0.85 → Forget best match
  │           └── No match → Return "not found"
  │
  ├── By ID → memories.forget({id})
  │
  └── Auto-Forget (Cron) → Check forgetAfter < now()
        └── Mark isForgotten=true for expired memories
```

---

## 6. Authentication Architecture

```
┌─ Authentication Flows ──────────────────────────────────────────┐
│                                                                  │
│  1. WEB CONSOLE (Session-Based)                                  │
│     ├── Better Auth with plugins:                                │
│     │   username, magicLink, emailOTP, apiKey, admin,            │
│     │   organization, anonymous                                  │
│     ├── Session cookie (better-auth.session_token)               │
│     └── Middleware enforces auth on all non-public paths         │
│                                                                  │
│  2. API (Bearer Token)                                           │
│     ├── API Key: Authorization: Bearer sm_xxx                    │
│     │   └── Validated via GET /v3/session                        │
│     └── Returns: { user.id, user.email, user.name }              │
│                                                                  │
│  3. MCP SERVER (OAuth + API Key)                                 │
│     ├── API Key: sm_ prefix → validateApiKey()                   │
│     │   └── GET /v3/session with Bearer header                   │
│     ├── OAuth Token: → validateOAuthToken()                      │
│     │   └── GET /v3/mcp/session-with-key with Bearer header      │
│     ├── OAuth Discovery:                                         │
│     │   ├── /.well-known/oauth-protected-resource                │
│     │   └── /.well-known/oauth-authorization-server (proxied)    │
│     └── 401 → WWW-Authenticate with resource_metadata URL        │
│                                                                  │
│  4. ORGANIZATION RBAC                                            │
│     └── Roles: owner > admin > editor > viewer                   │
│         └── Space-level membership with SpaceRole                │
└──────────────────────────────────────────────────────────────────┘
```

---

## 7. Framework Integration Architecture

```
┌─ @supermemory/tools ─────────────────────────────────────────────┐
│                                                                   │
│  ┌─ tools-shared.ts ──────────────────────────────────────────┐  │
│  │  TOOL_DESCRIPTIONS, PARAMETER_DESCRIPTIONS, DEFAULT_VALUES  │  │
│  │  getContainerTags(), deduplicateMemories()                  │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌─ ai-sdk.ts (Vercel AI SDK) ────────────────────────────────┐  │
│  │  searchMemoriesTool, addMemoryTool, getProfileTool,         │  │
│  │  documentListTool, documentDeleteTool, documentAddTool,     │  │
│  │  memoryForgetTool                                           │  │
│  │  → supermemoryTools(apiKey, config) → all-in-one export     │  │
│  │  → withSupermemory() → model wrapper                        │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌─ Other Integrations ───────────────────────────────────────┐  │
│  │  mastra.ts         → Mastra agent wrapper                   │  │
│  │  openai/            → OpenAI Agents SDK tools               │  │
│  │  claude-memory.ts   → Claude Memory Tool interface          │  │
│  │  voltagent/         → Voltagent integration                 │  │
│  │  conversations-client.ts → Conversation-level tools         │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                   │
│  All tools: init Supermemory SDK → wrap as framework tool         │
│  Config: { apiKey, projectId?, containerTags?, baseUrl?, strict } │
└───────────────────────────────────────────────────────────────────┘
```

---

## 8. Deployment Architecture

### 8.1. Infrastructure Map

| Component | Platform | Domain | Config |
|-----------|----------|--------|--------|
| API Backend | Cloudflare Workers | `api.supermemory.ai` | wrangler.jsonc (staging/prod) |
| MCP Server | Cloudflare Workers + DO | `mcp.supermemory.ai` | wrangler.jsonc + migrations |
| Web Console | Cloudflare Pages (Open Next) | `app.supermemory.ai` | open-next.config.ts |
| Documentation | Mintlify | `supermemory.ai/docs` | docs.json |
| Database | PostgreSQL | Managed | via Hyperdrive binding |

### 8.2. Cloudflare Bindings Matrix

| Binding | Type | Usage |
|---------|------|-------|
| `Hyperdrive` | Database | PostgreSQL connection pooling |
| `AI` | AI Gateway | Embedding generation, LLM inference |
| `KV` | Key-Value | Session cache, API key validation cache |
| `Workflows` | Durable Workflows | IngestContentWorkflow (async processing) |
| `Cron Triggers` | Scheduled | Connection imports every 4 hours |
| `MCP_SERVER` | Durable Object | Per-user MCP session state |

### 8.3. Observability Stack

```
┌─ Observability ─────────────────────────────────────────────────┐
│                                                                  │
│  Sentry (Error Tracking)                                         │
│  ├── org: supermemory, project: consumer-app                     │
│  ├── Source maps uploaded post-build                              │
│  ├── Tunnel route: /monitoring (bypass ad-blockers)               │
│  ├── User & organization context attached to events               │
│  └── Custom logging that filters analytics noise                  │
│                                                                  │
│  PostHog (Product Analytics)                                      │
│  ├── Events: memoryAdded, memoryForgot, memorySearch              │
│  ├── Properties: userId, source, mcp_client_name/version,        │
│  │   sessionId, containerTag, content_length, results_count       │
│  ├── Web: proxied through /ingest/* Next.js rewrite               │
│  └── MCP: server-side via initPosthog()                           │
│                                                                  │
│  Cloudflare Observability                                         │
│  └── enabled: true (Wrangler config)                              │
│                                                                  │
│  Internal Analytics                                               │
│  ├── ApiRequest logs: type, duration, statusCode, tokens, cost    │
│  └── Per-key, hourly, and period aggregations                     │
└──────────────────────────────────────────────────────────────────┘
```

---

## 9. Cross-Cutting Concerns

### 9.1. Schema Validation

Single source of truth: `packages/validation/`
- `schemas.ts` — Database entity schemas (Document, Chunk, MemoryEntry, Space, Connection, etc.)
- `api.ts` — API request/response schemas with OpenAPI extensions
- All schemas are Zod-based with `zod-openapi` extensions for auto-generated docs
- Shared across all apps and packages

### 9.2. Error Handling

| Layer | Strategy |
|-------|----------|
| API Backend | HTTPException → `{ error, details }` JSON response |
| MCP Server | Try/catch → `{ isError: true, content: [{ text }] }` MCP response |
| SDK Client | Status-aware error mapping (400→422→401→402→403→404→429→5xx) |
| Web Console | TanStack Query error states + Sonner toast notifications |
| Global | Sentry capture with user/org context |

### 9.3. Content Size Limits

| Parameter | Limit |
|-----------|-------|
| Memory content | 200,000 characters (~50k tokens) |
| Search query | 1,000 characters |
| Container tag | 128 characters |
| Search results | 100 items max |
| Memory list | 1,100 items max |
| Connection documents | 10,000 per connection |
| Filter items | 20 items per include/exclude list |
| Filter prompt | 750 characters |
| Project name | 100 characters |

### 9.4. Caching Strategy

| What | Where | TTL | Invalidation |
|------|-------|-----|-------------|
| Container tags (MCP) | In-memory (DO) | 5 minutes | On memory save to new tag, force refresh |
| MCP client info | Durable Object Storage | Persistent | On MCP re-initialization |
| API responses | SDK retry | — | 3 attempts, linear delay |
| PostHog analytics | Proxied | — | N/A |
