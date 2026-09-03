# TASK-SM-012 — packages/sdk: Go SDK & Framework Integrations

**Task ID:** TASK-SM-012  
**Wave:** 5 (Ecosystem)  
**Solution:** [SOL-SM-010](../solutions/SOL-SM-010-Framework-Integrations-SDK.md)  
**Depends on:** TASK-SM-007 (search API), TASK-SM-008 (profile API), TASK-SM-005 (document API)  
**Ước tính:** 5h  
**Priority:** Medium — developer ecosystem

---

## Mục tiêu

Tạo SDK và framework integrations cho Supermemory:
1. **Go SDK** (`packages/sdk/go/`) — type-safe client với sm_ auth
2. **LangChain Python** (`packages/integrations/python/vnp-langchain/`)
3. **Vercel AI TypeScript** (`packages/integrations/typescript/vnp-vercel-ai/`)

---

## Công việc cụ thể

### 1. Go SDK

**`packages/sdk/go/`**

```go
// packages/sdk/go/client.go

type SupramemoryClient struct {
    baseURL    string
    apiKey     string   // sm_xxx format
    httpClient *http.Client
    orgID      string
}

func NewClient(apiKey, baseURL string) *SupramemoryClient

// Document methods
func (c *SupramemoryClient) CreateDocument(ctx context.Context, req CreateDocumentRequest) (*Document, error)
func (c *SupramemoryClient) GetDocument(ctx context.Context, id string) (*Document, error)
func (c *SupramemoryClient) ListDocuments(ctx context.Context, req ListDocumentsRequest) ([]Document, error)
func (c *SupramemoryClient) DeleteDocument(ctx context.Context, id string) error
func (c *SupramemoryClient) BulkDeleteDocuments(ctx context.Context, ids []string) error

// Memory methods
func (c *SupramemoryClient) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error)
func (c *SupramemoryClient) ForgetMemory(ctx context.Context, req ForgetRequest) error
func (c *SupramemoryClient) ListMemories(ctx context.Context, req ListMemoriesRequest) ([]Memory, error)
func (c *SupramemoryClient) GetMemoryGraph(ctx context.Context, spaceID string) (*MemoryGraph, error)

// Search methods
func (c *SupramemoryClient) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error)
func (c *SupramemoryClient) SearchMemoriesV4(ctx context.Context, req SearchRequest) ([]MemorySearchV4Result, error)

// Profile methods
func (c *SupramemoryClient) GetProfile(ctx context.Context, spaceID string) (*UserProfile, error)
func (c *SupramemoryClient) GetProfileWithSearch(ctx context.Context, req ProfileSearchRequest) (*ProfileWithSearch, error)
func (c *SupramemoryClient) RebuildProfile(ctx context.Context, spaceID string) error

// UserProfile helper
func (p *UserProfile) ToSystemPrompt() string
```

**`packages/sdk/go/types.go`** — all request/response types:
```go
type CreateDocumentRequest struct {
    Content      string         `json:"content,omitempty"`
    URL          *string        `json:"url,omitempty"`
    CustomID     *string        `json:"customId,omitempty"`
    Type         DocumentType   `json:"type"`
    SpaceID      string         `json:"spaceId"`
    ContainerTag *string        `json:"containerTag,omitempty"`
    Metadata     map[string]any `json:"metadata,omitempty"`
}
// ... all other types
```

**`packages/sdk/go/pagination.go`**:
```go
// Page-based cursor pagination helper
type PageIterator[T any] struct { ... }
func (it *PageIterator[T]) Next(ctx context.Context) ([]T, error)
func (it *PageIterator[T]) HasMore() bool
```

### 2. LangChain Python Integration

**`packages/integrations/python/vnp-langchain/`**

```python
# src/vnp_langchain/memory.py

from langchain_core.memory import BaseMemory
from langchain_core.messages import BaseMessage
from vnp_supermemory import AsyncSupramemoryClient

class SupramemoryLangChainMemory(BaseMemory):
    """VNP Supermemory backend for LangChain.
    
    Supports:
    - chat_memory: stores conversation history
    - context: retrieves profile + search for system prompt injection
    """
    
    client: AsyncSupramemoryClient
    space_id: str
    human_prefix: str = "Human"
    ai_prefix: str = "AI"
    
    @property
    def memory_variables(self) -> list[str]:
        return ["context", "chat_history"]
    
    async def aload_memory_variables(self, inputs: dict) -> dict:
        """Load profile + search results into context."""
        query = inputs.get("input", "")
        profile = await self.client.get_profile_with_search(self.space_id, query=query)
        return {
            "context": profile.to_system_prompt(),
            "chat_history": self._format_messages(),
        }
    
    async def asave_context(self, inputs: dict, outputs: dict) -> None:
        """Save conversation turn to Supermemory."""
        human_msg = inputs.get("input", "")
        ai_msg = outputs.get("output", "")
        await self.client.create_document(
            content=f"{self.human_prefix}: {human_msg}\n{self.ai_prefix}: {ai_msg}",
            space_id=self.space_id,
            doc_type="text",
        )
    
    async def aclear(self) -> None:
        """No-op: Supermemory retains history by design."""
```

**`pyproject.toml`**:
```toml
[project]
name = "vnp-langchain"
requires-python = ">=3.10"
dependencies = ["langchain-core>=0.3.0", "vnp-supermemory>=1.0.0"]

[tool.mypy]
strict = true
```

**Tests**:
- `test_load_memory_variables`: mock client → returns context + chat_history
- `test_save_context_creates_document`: save → CreateDocument called with conversation text
- `test_mypy_strict`: `mypy src/ --strict` passes with 0 errors

### 3. Vercel AI TypeScript Integration

**`packages/integrations/typescript/vnp-vercel-ai/`**

```typescript
// src/index.ts

import { Message } from 'ai'
import { SupramemoryClient } from '@vnp/supermemory'

// VNP Supermemory DataStreamWriter for Vercel AI SDK
export interface SupramemoryVercelOptions {
    client: SupramemoryClient
    spaceId: string
    includeProfile?: boolean
}

/**
 * Get Vercel AI-compatible context from Supermemory
 * Returns profile + search results formatted for system prompt
 */
export async function getSupramemoryContext(
    options: SupramemoryVercelOptions,
    query: string,
): Promise<string> {
    const profile = await options.client.getProfileWithSearch(options.spaceId, { query })
    return profile.toSystemPrompt()
}

/**
 * Save Vercel AI conversation messages to Supermemory
 * Call after each response completes
 */
export async function saveConversation(
    options: SupramemoryVercelOptions,
    messages: Message[],
): Promise<void> {
    const lastTwo = messages.slice(-2)  // [user, assistant]
    const content = lastTwo.map(m => `${m.role}: ${m.content}`).join('\n')
    await options.client.createDocument({
        content,
        spaceId: options.spaceId,
        type: 'text',
    })
}
```

**`package.json`**:
```json
{
  "name": "@vnp/vercel-ai",
  "version": "1.0.0",
  "peerDependencies": { "ai": "^3.0.0" },
  "dependencies": { "@vnp/supermemory": "^1.0.0" }
}
```

**Tests** (`tests/index.test.ts`):
- `test_getContext_callsProfileWithSearch`: mock client → profile.toSystemPrompt called
- `test_saveConversation_last2Messages`: 5 messages → only last 2 saved
- TypeScript strict mode compiles without error

### 4. Shared Makefile Conventions

```makefile
# packages/sdk/go/Makefile
test:
	go test ./... -race -count=1

build:
	go build ./...

# packages/integrations/python/vnp-langchain/Makefile
ci: lint type-check test

# packages/integrations/typescript/vnp-vercel-ai/Makefile
test:
	npx jest --coverage --coverageThreshold='{"global":{"lines":90}}'
```

---

## Acceptance Criteria

### Go SDK
- [ ] `go build ./packages/sdk/go/...` không lỗi
- [ ] `go test ./packages/sdk/go/... -race` pass
- [ ] `SupramemoryClient.Search()` returns `[]SearchResult` with Score field
- [ ] `UserProfile.ToSystemPrompt()` returns formatted string
- [ ] PageIterator.HasMore() correct pagination

### LangChain
- [ ] `make ci` (lint + mypy + test) pass
- [ ] `mypy --strict src/` → 0 errors
- [ ] `aload_memory_variables` → dict with `context` and `chat_history` keys
- [ ] `asave_context` → CreateDocument called

### Vercel AI
- [ ] `tsc --strict` compiles without error
- [ ] `jest --coverage` > 90% coverage
- [ ] `getSupramemoryContext` returns string
- [ ] `saveConversation` with 5 messages → only last 2 saved

---

## Files tạo ra

```
packages/
├── sdk/go/
│   ├── client.go
│   ├── types.go
│   ├── pagination.go
│   ├── client_test.go
│   ├── go.mod
│   └── Makefile
└── integrations/
    ├── python/
    │   └── vnp-langchain/
    │       ├── src/vnp_langchain/
    │       │   ├── __init__.py
    │       │   └── memory.py
    │       ├── tests/
    │       │   └── test_memory.py
    │       ├── pyproject.toml
    │       └── Makefile
    └── typescript/
        └── vnp-vercel-ai/
            ├── src/
            │   └── index.ts
            ├── tests/
            │   └── index.test.ts
            ├── package.json
            ├── tsconfig.json
            └── Makefile
```

## Sau khi hoàn thành

```bash
# Go SDK
cd packages/sdk/go && go build ./... && go test ./...
# LangChain Python
cd packages/integrations/python/vnp-langchain && make ci
# Vercel AI TypeScript
cd packages/integrations/typescript/vnp-vercel-ai && npm test
```
