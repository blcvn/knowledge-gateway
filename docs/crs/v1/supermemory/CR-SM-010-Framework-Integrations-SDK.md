# Change Request: CR-SM-010 — Framework Integrations & SDK

**CR ID:** CR-SM-010  
**Component:** `packages/sdk/` + `gateway/adapters/` [NEW PACKAGES]  
**Priority:** Medium  
**Status:** In Progress
**Reference:** Supermemory PRD §3.7, §4.4, SRS §3.6 (NFR-COMPAT-03)

---

## 1. Mô tả

Xây dựng hệ sinh thái SDK và Framework Integrations cho VNP Memory:

1. **Go SDK**: SDK native cho Golang (primary language của project).
2. **TypeScript SDK**: `npm install vnp-memory` — đơn giản hóa tích hợp cho JS/TS developers.
3. **Python SDK**: `pip install vnp-memory` — phục vụ ML/AI developers.
4. **Framework Integrations**: Vercel AI SDK, LangChain, LangGraph, OpenAI Agents SDK, Mastra.
5. **Content Hashing**: Hỗ trợ `customId` và `contentHash` để developer kiểm soát deduplication.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện chưa có SDK chính thức.
- Developers phải tự viết HTTP client wrapper → tăng thời gian onboarding.
- Thiếu integrations với các AI frameworks phổ biến → hạn chế adoption.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] Go SDK (`packages/sdk/go/`)

```go
// Quickstart (3 bước)
client := vnpmemory.New(os.Getenv("VNP_MEMORY_API_KEY"))

// 1. Thêm memory
doc, err := client.Add(ctx, vnpmemory.AddInput{
    Content:       "User prefers Go over Python",
    ContainerTags: []string{"user_123"},
})

// 2. Lấy profile
profile, err := client.Profile(ctx, vnpmemory.ProfileInput{
    ContainerTag: "user_123",
    Query:        "programming preferences",  // Optional search combo
})
// profile.Static, profile.Dynamic, profile.SearchResults

// 3. Search
results, err := client.Search(ctx, vnpmemory.SearchInput{
    Query:        "Go performance tips",
    ContainerTag: "user_123",
    Limit:        10,
})
```

### 3.2. [NEW] TypeScript SDK (`packages/sdk/typescript/`)

```typescript
import { VNPMemory } from 'vnp-memory';

const client = new VNPMemory(process.env.VNP_MEMORY_API_KEY);

// Add memory
const doc = await client.add({ content: "...", containerTags: ["user_123"] });

// Profile
const profile = await client.profile({ containerTag: "user_123", q: "optional query" });

// Search  
const results = await client.search.memories({ q: "Go tips", containerTag: "user_123" });
```

### 3.3. [NEW] Python SDK (`packages/sdk/python/`)

```python
from vnp_memory import VNPMemory

client = VNPMemory(api_key=os.environ["VNP_MEMORY_API_KEY"])

doc = client.add(content="...", container_tags=["user_123"])
profile = client.profile(container_tag="user_123")
results = client.search(q="Go tips", container_tag="user_123")
```

### 3.4. Framework Integrations

| Framework | Package | Cách tích hợp |
|-----------|---------|--------------|
| **Vercel AI SDK** | `@vnp-memory/tools/ai-sdk` | Tool definitions cho `useChat`, `generateText` |
| **LangChain** | Python class `VNPMemoryRetriever` | Custom retriever + memory saver |
| **LangGraph** | Python `MemoryCheckpointer` | Checkpointing với VNP Memory backend |
| **OpenAI Agents** | `@vnp-memory/tools/openai` | Tool definitions cho OpenAI function calling |
| **Mastra** | `@vnp-memory/tools/mastra` | Native Mastra memory provider |

**Vercel AI SDK example:**
```typescript
import { vnpMemoryTools } from '@vnp-memory/tools/ai-sdk';

const result = await generateText({
  model: openai('gpt-4o'),
  tools: vnpMemoryTools({ client }),
  prompt: "Help me with my Go project",
});
```

### 3.5. SDK Features

- **Retry logic**: 3 lần với linear delay, tự động retry cho 429/503.
- **Content hashing**: SDK tự tính SHA-256 nếu developer không cung cấp `customId`.
- **Type safety**: Strict TypeScript types, Go generics-friendly.
- **Streaming support**: SSE streaming cho search results.

---

## 4. Acceptance Criteria

- [ ] `npm install vnp-memory && node quickstart.js` hoạt động trong < 5 phút.
- [ ] Go SDK: `client.Add()`, `client.Profile()`, `client.Search()` hoạt động đúng.
- [ ] Python SDK: `pip install vnp-memory && python quickstart.py` hoạt động.
- [ ] Vercel AI SDK integration: `useChat` với VNP Memory tools tự động save/recall.
- [ ] SDK tự động retry 3 lần khi gặp lỗi 429 (rate limit).
- [ ] TypeScript SDK có đủ type definitions (không cần `any`).
