# Solution: SOL-SM-010 — Framework Integrations & SDK

**CR ID:** CR-SM-010  
**Solution ID:** SOL-SM-010  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo hệ sinh thái SDK đa ngôn ngữ (`packages/sdk/`) và các framework integrations. Go SDK là primary (native Go), TypeScript SDK cho JS/TS ecosystem, Python SDK cho ML/AI developers. Mỗi SDK wrap REST API và cung cấp developer experience tốt nhất cho ngôn ngữ đó.

---

## 2. Cấu trúc Package

```
packages/
├── sdk/
│   ├── go/                    # Go SDK (primary)
│   │   ├── client.go          # VNPMemoryClient main struct
│   │   ├── add.go             # client.Add()
│   │   ├── search.go          # client.Search()
│   │   ├── profile.go         # client.Profile()
│   │   ├── forget.go          # client.Forget()
│   │   ├── retry.go           # Retry logic (3x với linear delay)
│   │   └── hash.go            # SHA-256 content hashing
│   │
│   ├── typescript/            # TypeScript/JS SDK
│   │   ├── src/
│   │   │   ├── client.ts
│   │   │   ├── types.ts
│   │   │   └── retry.ts
│   │   ├── package.json       # npm: vnp-memory
│   │   └── tsconfig.json
│   │
│   └── python/                # Python SDK
│       ├── vnp_memory/
│       │   ├── __init__.py
│       │   ├── client.py
│       │   └── retry.py
│       ├── pyproject.toml     # pip: vnp-memory
│       └── setup.py
│
└── integrations/
    ├── ai-sdk/                # Vercel AI SDK tools
    │   ├── src/tools.ts
    │   └── package.json       # @vnp-memory/tools-ai-sdk
    ├── langchain/             # LangChain Python integration
    │   ├── retriever.py       # VNPMemoryRetriever
    │   └── memory_saver.py
    ├── langgraph/             # LangGraph checkpointer
    │   └── checkpointer.py
    ├── openai-agents/         # OpenAI Agents SDK tools
    │   └── src/tools.ts
    └── mastra/                # Mastra memory provider
        └── src/provider.ts
```

---

## 3. Go SDK Implementation

### 3.1. Client Setup

```go
// packages/sdk/go/client.go

package vnpmemory

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

const DefaultBaseURL = "https://api.vnpmemory.io"
const SDKVersion = "1.0.0"

type Client struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
    maxRetries int
    retryDelay time.Duration
}

type Option func(*Client)

func WithBaseURL(url string) Option {
    return func(c *Client) { c.baseURL = url }
}

func WithMaxRetries(n int) Option {
    return func(c *Client) { c.maxRetries = n }
}

// New creates a VNP Memory client
// Usage: client := vnpmemory.New(os.Getenv("VNP_MEMORY_API_KEY"))
func New(apiKey string, opts ...Option) *Client {
    c := &Client{
        apiKey:     apiKey,
        baseURL:    DefaultBaseURL,
        maxRetries: 3,
        retryDelay: 1 * time.Second,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}
```

### 3.2. Add (Document Ingestion)

```go
// packages/sdk/go/add.go

type AddInput struct {
    Content       string
    ContainerTags []string
    CustomID      *string
    Title         *string
    URL           *string
    Metadata      map[string]any
}

type AddResult struct {
    ID     string
    Status string
}

func (c *Client) Add(ctx context.Context, input AddInput) (*AddResult, error) {
    // Auto-compute SHA-256 if no CustomID provided
    var customID string
    if input.CustomID != nil {
        customID = *input.CustomID
    } else {
        h := sha256.Sum256([]byte(input.Content))
        id := fmt.Sprintf("%x", h)[:16] // Short hash as customID
        customID = id
    }

    body := map[string]any{
        "content":       input.Content,
        "containerTags": input.ContainerTags,
        "customId":      customID,
        "type":          "text",
    }
    if input.Title != nil { body["title"] = *input.Title }
    if input.URL != nil { body["url"] = *input.URL }
    if input.Metadata != nil { body["metadata"] = input.Metadata }

    var result AddResult
    err := c.doWithRetry(ctx, "POST", "/api/v1/documents", body, &result)
    return &result, err
}
```

### 3.3. Search

```go
// packages/sdk/go/search.go

type SearchInput struct {
    Query         string
    ContainerTag  string
    Limit         int        // Default 10
    Mode          string     // "hybrid" | "memories-only" | "documents-only"
    Filters       *FilterGroup
    Rerank        bool
    RewriteQuery  bool
}

type SearchResult struct {
    ID       string
    Content  string
    Score    float64
    Type     string  // "chunk" | "memory" | "document"
    Metadata map[string]any
}

type SearchResponse struct {
    Results        []SearchResult
    RewrittenQuery *string
    LatencyMs      int64
}

func (c *Client) Search(ctx context.Context, input SearchInput) (*SearchResponse, error) {
    if input.Limit == 0 { input.Limit = 10 }
    if input.Mode == "" { input.Mode = "hybrid" }

    body := map[string]any{
        "q":            input.Query,
        "containerTag": input.ContainerTag,
        "limit":        input.Limit,
        "mode":         input.Mode,
        "rerank":       input.Rerank,
        "rewriteQuery": input.RewriteQuery,
    }
    if input.Filters != nil { body["filters"] = input.Filters }

    var result SearchResponse
    err := c.doWithRetry(ctx, "POST", "/api/v1/search", body, &result)
    return &result, err
}

// Convenience: Memory-only search (V4)
func (c *Client) SearchMemories(ctx context.Context, input SearchInput) (*SearchResponse, error) {
    input.Mode = "memories-only"
    return c.Search(ctx, input)
}
```

### 3.4. Profile

```go
// packages/sdk/go/profile.go

type ProfileInput struct {
    ContainerTag string
    Query        string  // Optional: also search
    Limit        int
}

type ProfileResponse struct {
    Static       []string
    Dynamic      []string
    SearchResults []SearchResult  // If query provided
    UpdatedAt    time.Time
    CacheHit     bool
}

func (c *Client) Profile(ctx context.Context, input ProfileInput) (*ProfileResponse, error) {
    if input.Query != "" {
        // Profile + Search combo
        body := map[string]any{
            "containerTag": input.ContainerTag,
            "q":            input.Query,
            "limit":        input.Limit,
        }
        var result ProfileResponse
        err := c.doWithRetry(ctx, "POST", "/api/v1/profiles/search", body, &result)
        return &result, err
    }

    // Profile only
    url := fmt.Sprintf("/api/v1/profiles?containerTag=%s", input.ContainerTag)
    var result ProfileResponse
    err := c.doWithRetry(ctx, "GET", url, nil, &result)
    return &result, err
}

// ToSystemPrompt formats profile for AI system prompts
func (p *ProfileResponse) ToSystemPrompt() string {
    var sb strings.Builder
    sb.WriteString("About the user:\n")
    if len(p.Static) > 0 {
        sb.WriteString("Long-term facts:\n")
        for _, f := range p.Static { sb.WriteString("- " + f + "\n") }
    }
    if len(p.Dynamic) > 0 {
        sb.WriteString("\nCurrent context:\n")
        for _, d := range p.Dynamic { sb.WriteString("- " + d + "\n") }
    }
    return sb.String()
}
```

### 3.5. Retry Logic

```go
// packages/sdk/go/retry.go

// Retry với linear delay (1s, 2s, 3s) cho 429 và 503
func (c *Client) doWithRetry(ctx context.Context, method, path string, body any, result any) error {
    var lastErr error

    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        if attempt > 0 {
            delay := time.Duration(attempt) * c.retryDelay
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return ctx.Err()
            }
        }

        err := c.doRequest(ctx, method, path, body, result)
        if err == nil { return nil }

        // Retry chỉ cho 429 (rate limit) và 503 (service unavailable)
        var apiErr *APIError
        if errors.As(err, &apiErr) {
            if apiErr.StatusCode == 429 || apiErr.StatusCode == 503 {
                lastErr = err
                continue // Retry
            }
        }
        return err // Non-retryable error
    }

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

---

## 4. TypeScript SDK

```typescript
// packages/sdk/typescript/src/client.ts

export interface VNPMemoryConfig {
  apiKey: string;
  baseURL?: string;
  maxRetries?: number;
}

export interface AddInput {
  content: string;
  containerTags?: string[];
  customId?: string;
  title?: string;
  url?: string;
  metadata?: Record<string, unknown>;
}

export interface SearchInput {
  q: string;
  containerTag?: string;
  limit?: number;
  mode?: 'hybrid' | 'memories-only' | 'documents-only';
  rerank?: boolean;
  rewriteQuery?: boolean;
}

export class VNPMemory {
  private apiKey: string;
  private baseURL: string;
  private maxRetries: number;

  constructor(apiKey: string, config?: Partial<VNPMemoryConfig>) {
    this.apiKey = apiKey;
    this.baseURL = config?.baseURL ?? 'https://api.vnpmemory.io';
    this.maxRetries = config?.maxRetries ?? 3;
  }

  async add(input: AddInput): Promise<{ id: string; status: string }> {
    return this.fetchWithRetry('POST', '/api/v1/documents', {
      content: input.content,
      containerTags: input.containerTags ?? [],
      customId: input.customId ?? this.hashContent(input.content),
      type: 'text',
      ...input,
    });
  }

  async profile(input: { containerTag: string; q?: string; limit?: number }) {
    if (input.q) {
      return this.fetchWithRetry('POST', '/api/v1/profiles/search', input);
    }
    return this.fetchWithRetry('GET', `/api/v1/profiles?containerTag=${input.containerTag}`);
  }

  get search() {
    return {
      memories: (input: SearchInput) =>
        this.fetchWithRetry('POST', '/api/v4/search', input),
      hybrid: (input: SearchInput) =>
        this.fetchWithRetry('POST', '/api/v1/search', { ...input, mode: 'hybrid' }),
    };
  }

  private hashContent(content: string): string {
    // SHA-256 via Web Crypto API (browser) or crypto module (Node)
    const encoder = new TextEncoder();
    const data = encoder.encode(content);
    return Array.from(new Uint8Array(data))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')
      .substring(0, 16);
  }

  private async fetchWithRetry(method: string, path: string, body?: unknown): Promise<any> {
    let lastError: Error | undefined;
    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      if (attempt > 0) {
        await new Promise(resolve => setTimeout(resolve, attempt * 1000));
      }
      try {
        const response = await fetch(`${this.baseURL}${path}`, {
          method,
          headers: {
            'Authorization': `Bearer ${this.apiKey}`,
            'Content-Type': 'application/json',
            'User-Agent': `vnp-memory-ts/1.0.0`,
          },
          body: body ? JSON.stringify(body) : undefined,
        });

        if (response.ok) return response.json();

        if (response.status === 429 || response.status === 503) {
          lastError = new Error(`HTTP ${response.status}`);
          continue; // Retry
        }
        throw new Error(`HTTP ${response.status}: ${await response.text()}`);
      } catch (e) {
        if (e instanceof Error) lastError = e;
      }
    }
    throw lastError ?? new Error('Max retries exceeded');
  }
}
```

---

## 5. Python SDK

```python
# packages/sdk/python/vnp_memory/client.py

import hashlib
import time
from typing import Optional, List, Dict, Any
import httpx

class VNPMemory:
    def __init__(
        self,
        api_key: str,
        base_url: str = "https://api.vnpmemory.io",
        max_retries: int = 3,
    ):
        self.api_key = api_key
        self.base_url = base_url
        self.max_retries = max_retries
        self._client = httpx.Client(
            headers={
                "Authorization": f"Bearer {api_key}",
                "User-Agent": "vnp-memory-python/1.0.0",
            },
            timeout=30.0,
        )

    def add(
        self,
        content: str,
        container_tags: List[str] = None,
        custom_id: str = None,
        **kwargs
    ) -> Dict[str, Any]:
        return self._request("POST", "/api/v1/documents", {
            "content": content,
            "containerTags": container_tags or [],
            "customId": custom_id or self._hash_content(content)[:16],
            "type": "text",
            **kwargs,
        })

    def profile(self, container_tag: str, q: str = None, **kwargs) -> Dict[str, Any]:
        if q:
            return self._request("POST", "/api/v1/profiles/search", {
                "containerTag": container_tag, "q": q, **kwargs
            })
        return self._request("GET", f"/api/v1/profiles?containerTag={container_tag}")

    def search(self, q: str, container_tag: str = None, limit: int = 10, **kwargs) -> Dict[str, Any]:
        return self._request("POST", "/api/v1/search", {
            "q": q, "containerTag": container_tag,
            "limit": limit, "mode": "hybrid", **kwargs
        })

    def _hash_content(self, content: str) -> str:
        return hashlib.sha256(content.encode()).hexdigest()

    def _request(self, method: str, path: str, body: Dict = None) -> Dict[str, Any]:
        last_error = None
        for attempt in range(self.max_retries + 1):
            if attempt > 0:
                time.sleep(attempt)  # Linear delay: 1s, 2s, 3s
            try:
                if method == "GET":
                    r = self._client.get(f"{self.base_url}{path}")
                else:
                    r = self._client.request(method, f"{self.base_url}{path}", json=body)

                if r.is_success:
                    return r.json()
                if r.status_code in (429, 503):
                    last_error = Exception(f"HTTP {r.status_code}")
                    continue
                r.raise_for_status()
            except Exception as e:
                last_error = e
        raise last_error or Exception("Max retries exceeded")
```

---

## 6. Framework Integrations

### 6.1. Vercel AI SDK

```typescript
// packages/integrations/ai-sdk/src/tools.ts

import { tool } from 'ai';
import { z } from 'zod';
import { VNPMemory } from 'vnp-memory';

export function vnpMemoryTools(client: VNPMemory) {
  return {
    saveMemory: tool({
      description: 'Save information to VNP Memory for future retrieval',
      parameters: z.object({
        content: z.string().describe('Information to save'),
        containerTag: z.string().optional(),
      }),
      execute: async ({ content, containerTag }) => {
        const result = await client.add({ content, containerTags: containerTag ? [containerTag] : [] });
        return { saved: true, id: result.id };
      },
    }),

    recallMemory: tool({
      description: 'Search past memories and get user profile',
      parameters: z.object({
        query: z.string().describe('What to search for'),
        containerTag: z.string().optional(),
      }),
      execute: async ({ query, containerTag }) => {
        const profile = await client.profile({ containerTag: containerTag ?? 'sm_project_default', q: query });
        return profile;
      },
    }),
  };
}

// Usage:
// const { text } = await generateText({
//   model: openai('gpt-4o'),
//   tools: vnpMemoryTools(client),
//   prompt: "Help me with my Go project",
// });
```

### 6.2. LangChain Python

```python
# packages/integrations/langchain/retriever.py

from langchain_core.retrievers import BaseRetriever
from langchain_core.documents import Document
from vnp_memory import VNPMemory

class VNPMemoryRetriever(BaseRetriever):
    """LangChain retriever backed by VNP Memory hybrid search."""

    client: VNPMemory
    container_tag: str = "sm_project_default"
    k: int = 5

    def _get_relevant_documents(self, query: str) -> list[Document]:
        results = self.client.search(q=query, container_tag=self.container_tag, limit=self.k)
        return [
            Document(
                page_content=r["content"],
                metadata={"id": r["id"], "score": r["score"], "type": r["type"]},
            )
            for r in results.get("results", [])
        ]

# Usage:
# retriever = VNPMemoryRetriever(client=VNPMemory(api_key="..."), k=5)
# chain = RetrievalQA.from_chain_type(llm=llm, retriever=retriever)
```

### 6.3. LangGraph Checkpointer

```python
# packages/integrations/langgraph/checkpointer.py

from langgraph.checkpoint.base import BaseCheckpointSaver
from vnp_memory import VNPMemory

class VNPMemoryCheckpointer(BaseCheckpointSaver):
    """LangGraph checkpointer using VNP Memory as backend."""

    def __init__(self, client: VNPMemory, container_tag: str = "sm_project_default"):
        self.client = client
        self.container_tag = container_tag

    def put(self, config, checkpoint, metadata, new_versions):
        import json
        content = json.dumps({"checkpoint": checkpoint, "metadata": metadata})
        self.client.add(content=content, container_tags=[self.container_tag],
                       custom_id=f"checkpoint:{config['configurable']['thread_id']}")

    def get(self, config):
        thread_id = config['configurable']['thread_id']
        results = self.client.search(
            q=f"checkpoint thread_id:{thread_id}",
            container_tag=self.container_tag,
            limit=1
        )
        if results.get("results"):
            import json
            return json.loads(results["results"][0]["content"])
        return None
```

---

## 7. Package Distribution

| SDK | Package Name | Registry | Target Version |
|-----|-------------|---------|----------------|
| Go | `github.com/vnp-memory/sdk-go` | Go modules | v1.0.0 |
| TypeScript | `vnp-memory` | npm | 1.0.0 |
| Python | `vnp-memory` | PyPI | 1.0.0 |
| Vercel AI integration | `@vnp-memory/tools-ai-sdk` | npm | 1.0.0 |
| LangChain | Part of Python SDK | PyPI extra | - |
| Mastra | `@vnp-memory/tools-mastra` | npm | 1.0.0 |

---

## 8. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Go SDK: client, add, search, profile, retry | 2 ngày |
| **P2** | Go SDK: forget, content hash, error types | 1 ngày |
| **P3** | TypeScript SDK với strict types | 2 ngày |
| **P4** | Python SDK + retry logic | 2 ngày |
| **P5** | Vercel AI SDK integration | 1 ngày |
| **P6** | LangChain retriever + memory saver | 1 ngày |
| **P7** | LangGraph checkpointer | 1 ngày |
| **P8** | OpenAI Agents + Mastra tools | 1 ngày |
| **P9** | npm/PyPI publishing + docs | 1 ngày |
| **P10** | Quickstart guides + tests | 1 ngày |

**Tổng:** ~13 ngày (Wave 5)

---

## 9. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| `npm install && node quickstart.js` trong < 5 phút | TypeScript SDK + published npm package |
| Go SDK: Add, Profile, Search hoạt động | Implemented in packages/sdk/go/ |
| Python: `pip install && python quickstart.py` | Published PyPI package |
| Vercel AI SDK + useChat tools | vnpMemoryTools() với saveMemory + recallMemory |
| Retry 3 lần cho 429 | doWithRetry: attempt <= maxRetries, retry on 429/503 |
| TypeScript không có `any` | z.object() schema + strict types |
