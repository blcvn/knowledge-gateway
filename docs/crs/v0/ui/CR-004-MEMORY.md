# CR-004 — Memory Explorer: Mock Search/Detail → Real API

| Field | Value |
|---|---|
| **CR ID** | CR-004 |
| **Title** | Memory Explorer: Kết nối memory search, detail, neighbors, versions với backend API |
| **Type** | Feature Implementation |
| **Priority** | P0 — Critical |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Memory Explorer |
| **Files thay đổi** | `ui/src/mock/memory.mock.ts`, `ui/src/hooks/useMemory.ts`, `ui/src/services/memory.service.ts` |

---

## 1. Hiện trạng

### Mock data ([`memory.mock.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/mock/memory.mock.ts))

```typescript
const mockItem: MemoryItem = {
  id: 'mem_123',
  engine: 'memobase',
  memoryType: 'profile',
  title: 'Mock Profile Memory',
  summary: 'This is a mock memory item.',
  content: 'Full content of the mock memory.',
  score: 0.95,
  entities: ['Mock', 'Data'],
  sourceSessions: ['sess_1'],
  temporalValidity: { from: '2026-01-01', to: null },
  policyTags: ['public'],
  versionChain: null,
  metadata: {},
};

export const memoryMock = {
  searchResult: {
    results: [mockItem],
    total: 1,
    facets: { byEngine: { memobase: 1 }, byType: { profile: 1 } },
    latencyMs: 45,
  },
  detail: mockItem,
};
```

### Hooks ([`useMemory.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/hooks/useMemory.ts))

```typescript
export function useMemorySearch(query: Record<string, unknown>) {
  return useQuery({
    queryFn: useMock
      ? () => Promise.resolve(memoryMock.searchResult)  // ← 1 kết quả hardcoded
      : () => memoryService.search(query),
  });
}
```

---

## 2. Backend API cần implement

Base path: `/v1/console/memory`

### 2.1 POST /v1/console/memory/search

Tìm kiếm cross-engine unified search.

**Request schema** (khớp [`MemorySearchQuery`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/memory.ts)):
```json
{
  "query": "temporal episodic memory graphiti",
  "mode": "hybrid",
  "engines": ["graphiti", "cognee", "memobase"],
  "filters": {
    "memory_type": "episodic",
    "date_from": "2026-01-01",
    "date_to": "2026-06-16",
    "policy_tags": ["public"]
  },
  "limit": 20,
  "offset": 0,
  "reranking": "cross_encoder"
}
```

**Response schema** (khớp [`MemorySearchResult`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/types/memory.ts)):
```json
{
  "results": [
    {
      "id": "ep_abc123",
      "engine": "graphiti",
      "memoryType": "episodic",
      "title": "Memory Architecture Discussion — Episode",
      "summary": "User explored Graphiti temporal reasoning vs Memobase.",
      "content": "Full episode content from knowledge graph...",
      "score": 0.94,
      "entities": ["Graphiti", "Memobase", "temporal reasoning"],
      "sourceSessions": ["sess_1"],
      "temporalValidity": { "from": "2026-06-01", "to": null },
      "policyTags": ["public"],
      "versionChain": null,
      "metadata": { "episode_id": "ep_abc123", "facts_count": 5 }
    }
  ],
  "total": 45,
  "facets": {
    "byEngine": { "graphiti": 20, "cognee": 15, "memobase": 10 },
    "byType": { "episodic": 20, "semantic": 15, "profile": 10 }
  },
  "latencyMs": 127
}
```

**Luồng xử lý**:
```
POST /v1/console/memory/search
  → vnp-search-hub (gRPC fan-out)
    ├── graphiti-search (Neo4j + BM25 hybrid)
    ├── cognee-search (vector search)
    ├── memobase-context (profile retrieval)
    ├── zep-search (session search)
    └── sm-search (adaptive KG)
  → Merge + rerank results
  → Return unified MemorySearchResult
```

**Search modes**:
| Mode | Mô tả |
|---|---|
| `semantic` | Vector similarity search |
| `bm25` | Keyword full-text search |
| `hybrid` | Semantic + BM25 + reranking |
| `graph` | Graph traversal (Graphiti/Cognee) |

### 2.2 GET /v1/console/memory/{id}

Chi tiết một memory item.

**Path format**: `{id}` = `{engine}:{memory_id}` (ví dụ: `graphiti:ep_abc123`, `memobase:prof_xyz`)

**Response**: `MemoryItem` đầy đủ bao gồm `content` chi tiết và `metadata`.

**Routing theo engine prefix**:
```
graphiti:* → graphiti-store service
cognee:*   → cognee-search service
memobase:* → memobase-engine service
zep:*      → zep-memory service
sm:*       → sm-memory service
ov:*       → ov-fs service
```

### 2.3 GET /v1/console/memory/{id}/neighbors

Lấy các memory items liên quan (semantic + graph neighbors).

**Query params**: `?limit=10&strategy=semantic` (strategies: `semantic`, `graph`, `temporal`)

**Response**: `MemorySearchResult` — tương tự search result nhưng filter theo similarity/adjacency.

**Luồng xử lý**:
- `semantic`: vector similarity search từ cùng embedding space
- `graph`: graph traversal từ Neo4j (1-2 hop neighbors)
- `temporal`: memories trong cùng time window

### 2.4 GET /v1/console/memory/{id}/versions

Lịch sử phiên bản của một memory item (Supermemory adaptive memory).

**Response schema**:
```json
[
  {
    "id": "mem_v3",
    "memory_id": "sm_abc",
    "content": "Latest version of the memory",
    "version_number": 3,
    "is_latest": true,
    "diff": "Changed 'beginner' to 'intermediate' in expertise.graphiti",
    "created_at": "2026-06-15T10:00:00Z"
  },
  {
    "id": "mem_v2",
    "memory_id": "sm_abc",
    "content": "Previous version",
    "version_number": 2,
    "is_latest": false,
    "diff": "Added graphiti expertise field",
    "created_at": "2026-06-01T10:00:00Z"
  }
]
```

**Nguồn**: Supermemory `sm-memory` service — version chain via `parent_id` / `root_id`.

---

## 3. Database schema

### Memory index (PostgreSQL + pgvector)

Console search cần một index layer tổng hợp từ tất cả engines:

```sql
-- Unified memory index cho console search
CREATE TABLE memory_index (
  id              TEXT PRIMARY KEY,          -- '{engine}:{local_id}'
  engine          TEXT NOT NULL,             -- 'graphiti', 'cognee', etc.
  memory_type     TEXT NOT NULL,
  title           TEXT,
  summary         TEXT,
  content         TEXT,
  embedding       vector(1536),              -- OpenAI text-embedding-3-small
  score           FLOAT DEFAULT 0,
  entities        TEXT[],
  source_sessions TEXT[],
  valid_from      TIMESTAMPTZ,
  valid_to        TIMESTAMPTZ,
  policy_tags     TEXT[],
  version_chain   TEXT,
  metadata        JSONB DEFAULT '{}',
  tenant_id       UUID NOT NULL,
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_memory_index_engine ON memory_index(engine);
CREATE INDEX idx_memory_index_type ON memory_index(memory_type);
CREATE INDEX idx_memory_index_tenant ON memory_index(tenant_id);
CREATE INDEX idx_memory_index_embedding ON memory_index USING ivfflat (embedding vector_cosine_ops);
```

> **Note**: Index này được cập nhật async thông qua NATS events khi memory được thêm/sửa/xóa ở bất kỳ engine nào.

---

## 4. Frontend thay đổi

### 4.1 Xóa mock dependency trong `useMemory.ts`

```typescript
// SAU — không còn mock
import { useQuery } from '@tanstack/react-query';
import { memoryService } from '../services/memory.service';
import type { MemorySearchQuery } from '../types/memory';

export function useMemorySearch(query: Partial<MemorySearchQuery>) {
  return useQuery({
    queryKey: ['memories', 'search', query],
    queryFn: () => memoryService.search({
      query: '',
      mode: 'hybrid',
      engines: ['graphiti', 'cognee', 'memobase', 'zep', 'supermemory', 'openviking'],
      filters: {},
      limit: 20,
      offset: 0,
      reranking: 'cross_encoder',
      ...query,
    }),
    enabled: !!query.query,    // Chỉ search khi có query text
    staleTime: 2 * 60 * 1000,
  });
}

export function useMemoryDetail(id: string) {
  return useQuery({
    queryKey: ['memories', 'detail', id],
    queryFn: () => memoryService.getById(id),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
  });
}

export function useMemoryNeighbors(id: string, strategy = 'semantic') {
  return useQuery({
    queryKey: ['memories', 'neighbors', id, strategy],
    queryFn: () => memoryService.getNeighbors(id, strategy),
    enabled: !!id,
  });
}

export function useMemoryVersions(id: string) {
  return useQuery({
    queryKey: ['memories', 'versions', id],
    queryFn: () => memoryService.getVersions(id),
    enabled: !!id && id.startsWith('sm:'),  // Chỉ có versioning cho Supermemory
  });
}
```

### 4.2 Cập nhật `memory.service.ts`

```typescript
export const memoryService = {
  search: (query: MemorySearchQuery) =>
    apiClient.post<MemorySearchResult>(`${BASE}/search`, query),

  getById: (id: string) =>
    apiClient.get<MemoryItem>(`${BASE}/${encodeURIComponent(id)}`),

  getNeighbors: (id: string, strategy = 'semantic') =>
    apiClient.get<MemorySearchResult>(`${BASE}/${encodeURIComponent(id)}/neighbors?strategy=${strategy}`),

  getVersions: (id: string) =>
    apiClient.get<MemoryVersion[]>(`${BASE}/${encodeURIComponent(id)}/versions`),
};
```

---

## 5. Điều kiện hoàn thành

- [ ] `POST /v1/console/memory/search` fan-out đến tất cả engines và merge kết quả
- [ ] `GET /v1/console/memory/{id}` route đúng đến engine dựa trên prefix
- [ ] `GET /v1/console/memory/{id}/neighbors` trả về related memories
- [ ] `GET /v1/console/memory/{id}/versions` trả về version chain từ Supermemory
- [ ] Memory Explorer không còn import từ `memory.mock.ts`
- [ ] Facets (byEngine, byType) được tính đúng từ kết quả thực
- [ ] `latencyMs` phản ánh thời gian thực của backend search
- [ ] Empty state hiển thị khi search không có kết quả

---

## 6. Notes

> **Performance**: Cross-engine search có thể mất 200-500ms. Frontend cần hiển thị loading skeleton. Backend nên timeout từng engine sau 2s và merge kết quả từ những engine đã respond.

> **ID encoding**: Memory ID có dạng `engine:local_id` — cần `encodeURIComponent` khi dùng trong URL path.
