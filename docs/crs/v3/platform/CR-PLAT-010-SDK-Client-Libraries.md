# Change Request: CR-PLAT-010 — SDK Client Libraries (Python & TypeScript)

**CR ID:** CR-PLAT-010
**Component:** `sdk/python/`, `sdk/typescript/`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F27](../../../features/27-organization-api-sdk-manager/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P6-02 | Framework Integrator | Phải viết raw HTTP calls |
| PP-P1-01 | Agent Developer | No SDK → slow integration |

---

## 2. Python SDK

### Installation
```bash
pip install vnp-memory
```

### Usage
```python
from vnp_memory import VNPClient

client = VNPClient(api_key="vnp_xxx.yyy")

# Store memory
resp = await client.memory.store(
    content="Meeting with Alice about Project X",
    type="episodic",
    user_id="user-123"
)
# resp: {"id": "...", "engine": "graphiti", "status": "processing"}

# Recall
results = await client.memory.recall(
    query="Alice meeting",
    user_id="user-123",
    limit=10
)

# MCP integration
mcp_server = client.mcp.get_server()
# Registers 37+ tools for use with LangChain, CrewAI, etc.

# Observe session
session = await client.observe.start_session(agent_id="my-agent")
await session.hook(hook_type="llm_call", payload={...})
await session.end()
```

### SDK Features
- Async/await native (asyncio)
- Auto-retry with exponential backoff
- Type hints for all methods
- LangChain integration: `VNPMemoryLangChain`
- CrewAI integration: `VNPMemoryCrewAI`
- OpenAI Agents SDK integration

---

## 3. TypeScript SDK

### Installation
```bash
npm install @vnp/memory-sdk
```

### Usage
```typescript
import { VNPClient } from '@vnp/memory-sdk';

const client = new VNPClient({ apiKey: 'vnp_xxx.yyy' });

// Store
const resp = await client.memory.store({
  content: 'Meeting about Project X',
  type: 'episodic',
  userId: 'user-123'
});

// WebSocket events
client.events.on('memory_stored', (event) => {
  console.log('New memory:', event.memoryId);
});
await client.events.connect();
```

---

## 4. SDK Structure

```
sdk/python/
  vnp_memory/
    client.py        # Main client
    memory.py        # Memory operations
    observe.py       # Session observation
    mcp.py           # MCP server integration
    models.py        # Pydantic models
    exceptions.py

sdk/typescript/
  src/
    client.ts
    memory.ts
    observe.ts
    events.ts        # WebSocket client
    types.ts
```

---

## 5. Acceptance Criteria

- [ ] Python SDK: store/recall/forget/observe work
- [ ] TypeScript SDK: store/recall/observe/websocket work
- [ ] Both SDKs: auto-retry on 429/5xx (max 3 retries)
- [ ] Both SDKs: type-safe API (mypy + TypeScript strict)
- [ ] LangChain integration: `VNPMemoryLangChain` works
- [ ] Published to PyPI + npm
