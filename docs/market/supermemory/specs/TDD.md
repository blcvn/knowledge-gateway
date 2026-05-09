# Technical Design Document (TDD)

## Supermemory — Memory & Context Engine for AI

| Metadata | Value |
|----------|-------|
| **Version** | 4.0.0 |
| **Date** | 2026-05-09 |
| **Stack** | TypeScript · Cloudflare Workers · Hono · PostgreSQL · Drizzle ORM |

---

## 1. Mục Đích

Tài liệu thiết kế kỹ thuật chi tiết cho Supermemory, mô tả các quyết định thiết kế, implementation patterns, data models, interfaces, và algorithms cụ thể của từng subsystem.

---

## 2. Subsystem: Validation Layer

### 2.1. Design Decisions

**Single Source of Truth Pattern**: Tất cả schemas được định nghĩa tại `packages/validation/` và shared across toàn bộ monorepo. Đây là package có zero runtime dependencies ngoài `zod` và `zod-openapi`.

**Schema Architecture**:
```
packages/validation/
├── schemas.ts    → Database entity schemas (ORM-aligned)
└── api.ts        → API request/response schemas (HTTP-aligned)
```

### 2.2. Database Entity Schemas (`schemas.ts`)

**Document Schema** — Core content entity:
```typescript
DocumentSchema = z.object({
  id: z.string(),                          // nanoid generated
  customId: z.string().nullable(),         // User-provided dedup ID
  contentHash: z.string().nullable(),      // SHA-256 for dedup
  orgId: z.string(),                       // Organization scope
  userId: z.string(),                      // Creator
  connectionId: z.string().nullable(),     // Source connection
  type: DocumentTypeEnum,                  // text|pdf|tweet|google_doc|...
  status: DocumentStatusEnum,              // queued→extracting→chunking→embedding→indexing→done|failed
  summaryEmbedding: z.array(z.number()),   // Document-level vector
  summaryEmbeddingNew: z.array(z.number()),// Migration-ready secondary embedding
  // ... content fields, timestamps
})
```

**MemoryEntry Schema** — Extracted knowledge unit:
```typescript
MemoryEntrySchema = z.object({
  id: z.string(),
  memory: z.string(),                      // The fact text
  spaceId: z.string(),                     // Space/project FK
  
  // Version chain
  version: z.number().default(1),
  isLatest: z.boolean().default(true),
  parentMemoryId: z.string().nullable(),   // Direct parent
  rootMemoryId: z.string().nullable(),     // Chain root
  
  // Relationships to other memories
  memoryRelations: z.record(MemoryRelationEnum), // {memoryId: "updates"|"extends"|"derives"}
  
  // Lifecycle
  isInference: z.boolean().default(false), // AI-derived fact
  isForgotten: z.boolean().default(false), // Soft-deleted
  isStatic: z.boolean().default(false),    // Long-term stable fact
  forgetAfter: z.coerce.date().nullable(), // Auto-forget timestamp
  forgetReason: z.string().nullable(),
  
  // Dual embedding support
  memoryEmbedding: z.array(z.number()),
  memoryEmbeddingNew: z.array(z.number()),
})
```

**Chunk Schema** — Document segment with embeddings:
```typescript
ChunkSchema = z.object({
  id: z.string(),
  documentId: z.string(),                   // Parent document FK
  content: z.string(),                      // Chunk text
  embeddedContent: z.string().nullable(),   // Optimized embedding text
  type: ChunkTypeEnum,                      // text | image
  position: z.number(),                     // Order within document
  embedding: z.array(z.number()),           // Primary vector
  embeddingNew: z.array(z.number()),        // Migration vector
  matryokshaEmbedding: z.array(z.number()), // Matryoshka truncatable vector
})
```

### 2.3. API Schemas (`api.ts`)

**Search Request Schema** — V3 Hybrid Search:
```typescript
SearchRequestSchema = z.object({
  q: z.string().min(1),                    // Required query
  containerTags: z.array(z.string()),      // Scope filter
  limit: z.number().int().positive().max(100).default(10),
  chunkThreshold: z.number().min(0).max(1).default(0),    // Chunk sensitivity
  documentThreshold: z.number().min(0).max(1).default(0), // Doc sensitivity
  filters: SearchFiltersSchema,            // AND/OR metadata filters
  includeFullDocs: z.boolean().default(false),
  includeSummary: z.boolean().default(false),
  onlyMatchingChunks: z.boolean().default(true),
  rerank: z.boolean().default(false),       // AI reranking
  rewriteQuery: z.boolean().default(false), // AI query optimization
})
```

**Memory Search Schema** — V4 Fact-Level Search:
```typescript
Searchv4RequestSchema = z.object({
  q: z.string().min(1),
  containerTag: z.string().optional(),     // Single tag (v4 simplified)
  threshold: z.number().min(0).max(1).default(0.6),
  include: z.object({
    documents: z.boolean().default(false),  // Include source docs
    summaries: z.boolean().default(false),
    relatedMemories: z.boolean().default(false), // Parent/child chain
  }),
  // ... limit, filters, rerank, rewriteQuery
})
```

**Memory Search Result** — Rich result with version context:
```typescript
MemorySearchResult = z.object({
  id: z.string(),
  memory: z.string(),                       // The fact
  similarity: z.number().min(0).max(1),     // Cosine similarity score
  version: z.number().nullable(),
  metadata: z.record(z.unknown()),
  context: z.object({                       // Version chain context
    parents: z.array(z.object({
      relation: z.enum(["updates", "extends", "derives"]),
      version: z.number(),                  // Relative: -1, -2, ...
      memory: z.string(),
    })),
    children: z.array(z.object({
      relation: z.enum(["updates", "extends", "derives"]),
      version: z.number(),                  // Relative: +1, +2, ...
      memory: z.string(),
    })),
  }),
  documents: z.array(MemorySearchDocumentSchema), // Source documents
})
```

---

## 3. Subsystem: MCP Server

### 3.1. Runtime Architecture

**Platform**: Cloudflare Workers + Durable Objects  
**Framework**: Hono (HTTP router) + `agents/mcp` (MCP protocol handler)  
**Session Model**: One Durable Object per authenticated user (keyed by userId)

### 3.2. Authentication Flow

```typescript
// 1. Token extraction
const token = authHeader?.replace(/^Bearer\s+/i, "")

// 2. Token classification
if (isApiKey(token))   // token.startsWith("sm_")
  authUser = await validateApiKey(token, apiUrl)   // GET /v3/session
else
  authUser = await validateOAuthToken(token, apiUrl) // GET /v3/mcp/session-with-key

// 3. Props injection into Durable Object
const props: Props = {
  userId: authUser.userId,
  apiKey: authUser.apiKey,
  containerTag: req.header("x-sm-project"),  // Optional per-request scope
  email: authUser.email,
  name: authUser.name,
}
```

### 3.3. Tool Implementation Pattern

Mỗi MCP tool tuân theo pattern:
1. Resolve effective containerTag: `args.containerTag || this.props?.containerTag`
2. Get SupermemoryClient instance
3. Execute operation via SDK
4. Track event via PostHog (fire-and-forget `.catch()`)
5. Return structured MCP response `{ content: [{ type: "text", text }] }`

**Memory Tool** — Save/Forget decision:
```typescript
// Save flow
const result = await client.createMemory(content)
// → POST /v3/documents { content, containerTag, metadata: { sm_source: "mcp" } }

// Forget flow (2-phase)
// Phase 1: Exact content match
await client.memories.forget({ content, containerTag })
// Phase 2 (fallback): Semantic search with threshold 0.85
const searchResult = await client.search(content, 5, 0.85)
// Filter to memory results only (not chunks)
const memoryToDelete = searchResult.results.find(r => "memory" in r)
await client.memories.forget({ id: memoryToDelete.id })
```

### 3.4. Caching Design

```typescript
// Container tags caching with TTL
private cachedContainerTags: string[] = []
private containerTagsLastFetchedAt: number | null = null
const CONTAINER_TAGS_TTL_MS = 5 * 60 * 1000  // 5 minutes

// Auto-refresh: check staleness before every list/recall operation
private async ensureContainerTagsFresh(): Promise<void> {
  const needsRefresh = this.containerTagsLastFetchedAt === null 
    || Date.now() - this.containerTagsLastFetchedAt > CONTAINER_TAGS_TTL_MS
  if (needsRefresh) await this.refreshContainerTags()
}

// Force-refresh: after creating memory in unknown containerTag
if (!this.cachedContainerTags.includes(result.containerTag)) {
  await this.refreshContainerTags()
}
```

### 3.5. MCP App UI (Memory Graph)

```typescript
// Register HTML resource for embedded UI
registerAppResource(this.server, "Memory Graph UI", 
  "ui://memory-graph/mcp-app.html",
  { mimeType: RESOURCE_MIME_TYPE },
  async () => ({
    contents: [{ uri, mimeType: RESOURCE_MIME_TYPE, text: mcpAppHtml }]
  })
)

// Tool returns structuredContent for the UI to consume
return {
  content: [{ type: "text", text: "Memory Graph: X docs, Y memories" }],
  structuredContent: {
    containerTag: effectiveContainerTag,
    documents: result.documents,      // Full document+memory data
    totalCount: result.pagination.totalItems,
  },
}
```

---

## 4. Subsystem: Framework Integration Tools

### 4.1. Tool Factory Pattern

Tất cả framework integrations sử dụng chung pattern:

```typescript
// 1. Shared constants (tools-shared.ts)
TOOL_DESCRIPTIONS = { searchMemories, addMemory, getProfile, ... }
PARAMETER_DESCRIPTIONS = { informationToGet, memory, containerTag, ... }
DEFAULT_VALUES = { includeFullDocs: true, limit: 10, chunkThreshold: 0.6 }

// 2. Container tag resolution
function getContainerTags(config?: { projectId?, containerTags? }): string[] {
  if (config?.projectId) return [`sm_project_${config.projectId}`]
  return config?.containerTags ?? ["sm_project_default"]
}

// 3. Tool creation per framework
export const searchMemoriesTool = (apiKey, config?) => {
  const client = new Supermemory({ apiKey, baseURL: config?.baseUrl })
  const containerTags = getContainerTags(config)
  return tool({
    description: TOOL_DESCRIPTIONS.searchMemories,
    inputSchema: z.object({ ... }),
    execute: async (args) => { /* SDK call */ },
  })
}
```

### 4.2. Memory Deduplication Algorithm

```typescript
function deduplicateMemories(data: ProfileWithMemories): DeduplicatedMemories {
  // Priority: Static > Dynamic > Search Results
  const seenMemories = new Set<string>()
  
  // Phase 1: Collect static memories (highest priority)
  for (const item of data.static) {
    const memory = normalize(item)  // Handle both string and {memory} object
    if (memory) { staticMemories.push(memory); seenMemories.add(memory) }
  }
  
  // Phase 2: Collect dynamic (skip duplicates from static)
  for (const item of data.dynamic) {
    const memory = normalize(item)
    if (memory && !seenMemories.has(memory)) {
      dynamicMemories.push(memory); seenMemories.add(memory)
    }
  }
  
  // Phase 3: Collect search results (skip all seen)
  // Same pattern...
}
```

### 4.3. AI SDK `withSupermemory` Wrapper

Wraps any AI model to automatically inject Supermemory tools:
```typescript
// Usage
const model = withSupermemory(openai("gpt-4o"), {
  containerTag: "user_123",
  customId: "conv-1"
})
// → Model now has: searchMemories, addMemory, getProfile tools
```

---

## 5. Subsystem: Memory Graph Visualization

### 5.1. Component Architecture

```
packages/memory-graph/src/
├── index.tsx          → MemoryGraph (main exported component)
├── types.ts           → GraphNode, GraphEdge, GraphThemeColors, etc.
├── api-types.ts       → API response types
├── constants.ts       → Visual constants
├── canvas/            → Canvas rendering (2D context)
│   ├── simulation.ts  → D3-force simulation wrapper
│   └── viewport.ts    → Pan/zoom state management
├── components/        → React UI overlays
├── hooks/             → Custom hooks for data fetching & interaction
└── __tests__/         → Unit tests
```

### 5.2. Graph Data Model

```typescript
interface GraphNode {
  id: string
  type: "document" | "memory"
  x: number; y: number         // Position (d3-force managed)
  vx?: number; vy?: number     // Velocity
  fx?: number | null; fy?: number | null  // Fixed position (dragging)
  size: number
  borderColor: string           // Lifecycle-based coloring
  data: DocumentNodeData | MemoryNodeData
}

interface GraphEdge {
  id: string
  source: string | GraphNode    // D3-force link reference
  target: string | GraphNode
  edgeType: "updates" | "extends" | "derives"
  visualProps: { opacity: number; thickness: number }
}
```

### 5.3. Visual Encoding

| Memory State | Border Color |
|-------------|-------------|
| Forgotten (`isForgotten`) | `memBorderForgotten` |
| Expiring (`forgetAfter` set) | `memBorderExpiring` |
| Recent (< 24h) | `memBorderRecent` |
| Default | `memStrokeDefault` |

| Edge Type | Color |
|-----------|-------|
| `derives` | `edgeDerives` |
| `updates` | `edgeUpdates` |
| `extends` | `edgeExtends` |

### 5.4. Similarity Visualization

```typescript
// Opacity: 0 → 1 (direct mapping)
opacity: Math.max(0, normalizedSimilarity)

// Edge thickness: 1px → 4px
thickness: Math.max(1, normalizedSimilarity * 4)

// Glow intensity: 0 → 0.6
glow: normalizedSimilarity * 0.6

// Color: HSL with dynamic saturation/lightness
// saturation: 60% → 100%, lightness: 40% → 70%
color: `hsl(${hue}, ${60 + sim * 40}%, ${40 + sim * 30}%)`
```

---

## 6. Subsystem: Web Console

### 6.1. Authentication & Middleware

```typescript
// middleware.ts — Request gate
export default async function proxy(request: Request) {
  const sessionCookie = getSessionCookie(request)  // better-auth
  
  // Public paths: /login, /login/new, ?view=mcp
  if (publicPaths.includes(url.pathname)) return NextResponse.next()
  
  // API routes: 401 JSON response if no session
  if (url.pathname.startsWith("/api/") && !sessionCookie)
    return new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 })
  
  // All other routes: redirect to /login with ?redirect= param
  if (!sessionCookie)
    return NextResponse.redirect(new URL("/login", request.url))
  
  // Set tracking cookie
  response.cookies.set({ name: "last-site-visited", value: "https://app.supermemory.ai" })
}
```

### 6.2. API Client Architecture

```typescript
// packages/lib/api.ts
export const $fetch = createFetch({
  baseURL: `${NEXT_PUBLIC_BACKEND_URL ?? "https://api.supermemory.ai"}/v3`,
  credentials: "include",           // Session cookie forwarding
  retry: { attempts: 3, delay: 100, type: "linear" },
  schema: apiSchema,                // Typed API schema (createSchema)
})

// Usage: fully typed, auto-validated
const { data, error } = await $fetch("@post/documents", {
  body: { content: "...", containerTags: ["user_123"] }
})
```

### 6.3. Query Layer

```typescript
// Optimistic delete with cache invalidation
const useDeleteDocument = (selectedProject) => useMutation({
  mutationFn: (docId) => $fetch(`@delete/documents/${docId}`),
  onMutate: (docId) => {
    // Cancel in-flight queries
    await queryClient.cancelQueries({ queryKey: ["documents-with-memories", selectedProject] })
    // Optimistically remove from both infinite and standard query caches
    queryClient.setQueryData([...key], old => filterOut(old, docId))
    return { previousData }
  },
  onError: (_, __, ctx) => queryClient.setQueryData([...key], ctx.previousData),
  onSettled: () => queryClient.invalidateQueries({ queryKey: [...key] }),
})
```

### 6.4. Subscription Logic

```typescript
const PLAN_TIERS = ["api_pro", "api_scale", "api_enterprise"] as const

function isAllowedFrom(status: SubscriptionStatusMap, minimumTier: PlanTier): boolean {
  const minIndex = PLAN_TIERS.indexOf(minimumTier)
  return PLAN_TIERS.slice(minIndex).some(tier => status[tier]?.status === "active")
}

function hasActivePlan(subscriptions, minimumTier): boolean {
  return isAllowedFrom(getSubscriptionStatus(subscriptions), minimumTier)
}
```

---

## 7. Subsystem: Content Processing

### 7.1. IngestContentWorkflow Design

**Platform**: Cloudflare Workflows (durable, retryable)  
**Trigger**: `POST /v3/documents` → enqueue workflow

**State Machine**:
```
UNKNOWN → QUEUED → EXTRACTING → CHUNKING → EMBEDDING → INDEXING → DONE
                                                                  ↓ (on error)
                                                                FAILED
```

**Processing Metadata** tracked per step:
```typescript
ProcessingMetadataSchema = z.object({
  startTime: z.number(),
  endTime: z.number().optional(),
  duration: z.number().optional(),
  error: z.string().optional(),
  finalStatus: z.enum(["completed", "failed", "done"]),
  chunkingStrategy: z.string().optional(),
  tokenCount: z.number().optional(),
  steps: z.array(ProcessingStepSchema), // name, startTime, endTime, status, error
})
```

### 7.2. Content Type Detection

```typescript
DocumentTypeEnum = z.enum([
  "text",          // Plaintext
  "pdf",           // PDF → text extraction
  "tweet",         // Twitter content
  "google_doc",    // Google Docs → API fetch
  "google_slide",  // Google Slides → API fetch
  "google_sheet",  // Google Sheets → API fetch
  "image",         // Image → OCR
  "video",         // Video → transcription
  "notion_doc",    // Notion → API fetch
  "webpage",       // URL → HTML parse
  "onedrive",      // OneDrive → API fetch
])
```

Auto-detection flow: URL response `Content-Type` header → map to DocumentType

---

## 8. Subsystem: Connectors

### 8.1. Connection Lifecycle

```
1. Client: POST /v3/connections/:provider
   Body: { redirectUrl, containerTags, documentLimit, metadata }
   
2. Server: Create ConnectionState with stateToken (CSRF)
   Response: { authLink: "https://provider.oauth.url?state=...", id, expiresIn }

3. User: Authorize at provider

4. Callback: Provider redirects with auth code
   Server: Exchange code → store accessToken + refreshToken

5. Sync: Cron (4h) or webhook → fetch documents → enqueue IngestContentWorkflow
```

### 8.2. Custom OAuth Keys

Enterprise organizations can bring their own OAuth credentials:
```typescript
OrganizationSettingsSchema = z.object({
  // Google Drive
  googleDriveCustomKeyEnabled: z.boolean().default(false),
  googleDriveClientId: z.string().nullable(),
  googleDriveClientSecret: z.string().nullable(),
  // Notion
  notionCustomKeyEnabled: z.boolean().default(false),
  notionClientId: z.string().nullable(),
  notionClientSecret: z.string().nullable(),
  // OneDrive
  onedriveCustomKeyEnabled: z.boolean().default(false),
  onedriveClientId: z.string().nullable(),
  onedriveClientSecret: z.string().nullable(),
})
```

---

## 9. Subsystem: Analytics

### 9.1. Request Tracking Schema

```typescript
ApiRequestSchema = z.object({
  id: z.string(),
  type: RequestTypeEnum,    // add|search|fast_search|request|update|delete|chat|search_v4
  orgId: z.string(),
  userId: z.string(),
  keyId: z.string(),        // Which API key was used
  statusCode: z.number(),
  duration: z.number(),     // ms
  
  // Token economics
  originalTokens: z.number(),   // Before optimization
  finalTokens: z.number(),      // After optimization
  tokensSaved: z.number(),      // Computed: original - final
  costSavedUSD: z.number(),     // Dollar value of savings
  
  // Chat-specific
  model: z.string(),
  provider: z.string(),
  conversationId: z.string(),
  contextModified: z.boolean(), // Was context injected?
  
  origin: z.string().default("api"), // api | mcp | console
})
```

### 9.2. Analytics Aggregations

| Endpoint | Aggregation | Dimensions |
|----------|-------------|-----------|
| `/analytics/usage` | Count, avgDuration | By type, by API key, hourly |
| `/analytics/memory` | Growth metrics | totalMemories, searchQueries, connections, tokens |
| `/analytics/chat` | Token economics | tokensByDay, latency trends, amount saved (7d/30d/90d/lifetime) |

---

## 10. Build & Development

### 10.1. Turborepo Pipeline

```json
{
  "tasks": {
    "build": { "dependsOn": ["^build"], "outputs": [".next/**"] },
    "dev": { "cache": false, "persistent": true },
    "check-types": { "dependsOn": ["^check-types"] },
    "lint": { "dependsOn": ["^lint"] }
  }
}
```

### 10.2. Code Quality

| Tool | Config | Scope |
|------|--------|-------|
| **Biome** | `biome.json` (root) | Lint + format entire monorepo |
| **TypeScript** | `@total-typescript/tsconfig` | Strict mode, per-package tsconfig |
| **Drizzle Kit** | `drizzle-kit` | Database migrations |
| **Wrangler** | Per-app `wrangler.jsonc` | Cloudflare Workers config |

### 10.3. Development Commands

```bash
bun run dev          # Start all apps (Turbo)
bun run build        # Build all apps
bun run check-types  # TypeScript check across monorepo
bun run format-lint  # Biome lint + format
```

### 10.4. Runtime Requirements

| Requirement | Version |
|-------------|---------|
| Node.js | >= 20 |
| Bun | >= 1.3.6 |
| TypeScript | 5.8.3 |
| Wrangler | >= 4.42 |
| Compatibility Date | 2025-01-01 |
| Compatibility Flags | `nodejs_compat` |
