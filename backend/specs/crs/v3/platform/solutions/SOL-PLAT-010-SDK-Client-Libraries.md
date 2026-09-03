# Solution: SOL-PLAT-010 — SDK Client Libraries (Python & TypeScript)

**CR:** CR-PLAT-010
**TDD refs:** `backend-api-specs.md`, `architecture/01-gateway.md §4`
**Version:** v3/platform

---

## 1. Architecture: SDK Design Pattern

SDK → direct HTTP calls to Gateway REST API (không gRPC) → standard auth via API key.

```
SDK Client
  ├── Transport layer (httpx/fetch)
  │   ├── Auto-retry: 429/5xx → exponential backoff (1s, 2s, 4s, max 3)
  │   ├── Auth injection: X-API-Key header
  │   └── Error parsing: map HTTP → SDK exception types
  ├── Memory module: store/recall/forget/timeline
  ├── Observe module: start_session/hook/end_session
  ├── Events module: WebSocket connection (TypeScript only)
  └── MCP module: SSE MCP client wrapper
```

---

## 2. Python SDK Structure

```
sdk/python/
  setup.py / pyproject.toml
  vnp_memory/
    __init__.py           # export VNPClient
    client.py             # main client, config
    _transport.py         # httpx async transport + retry
    memory.py             # MemoryAPI: store/recall/forget/timeline
    observe.py            # ObserveAPI: session + hooks
    mcp.py                # MCPClient wrapper (SSE)
    models.py             # pydantic models for requests/responses
    exceptions.py         # VNPError, RateLimitError, AuthError
    integrations/
      langchain.py        # VNPMemoryLangChain
      crewai.py           # VNPMemoryCrewAI
      openai_agents.py    # VNPMemoryOpenAI
  tests/
    test_memory.py
    test_observe.py
    test_retry.py
```

### Key Python classes

```python
# vnp_memory/client.py
class VNPClient:
    def __init__(self, api_key: str, base_url: str = "https://api.vnp.memory"):
        self._transport = AsyncTransport(api_key=api_key, base_url=base_url)
        self.memory  = MemoryAPI(self._transport)
        self.observe = ObserveAPI(self._transport)
        self.mcp     = MCPClient(self._transport)

# vnp_memory/memory.py
class MemoryAPI:
    async def store(self, content: str, type: str = "auto",
                    user_id: str = "", **metadata) -> StoreResponse:
        return await self._t.post("/v1/memory/store", {
            "content": content, "type": type,
            "user_id": user_id, "metadata": metadata
        })

    async def recall(self, query: str, user_id: str = "",
                     types: list[str] = None, limit: int = 10) -> RecallResponse:
        return await self._t.post("/v1/memory/recall", {
            "query": query, "user_id": user_id,
            "types": types or [], "limit": limit
        })

    async def forget(self, user_id: str, reason: str = "") -> ForgetResponse: ...
    async def timeline(self, user_id: str, from_: str = "", to: str = "") -> TimelineResponse: ...

# vnp_memory/observe.py
class Session:
    async def hook(self, hook_type: str, payload: dict = None) -> ObservationResponse: ...
    async def end(self) -> SessionResponse: ...

class ObserveAPI:
    async def start_session(self, agent_id: str, project: str = "",
                            model: str = "") -> Session: ...

# vnp_memory/integrations/langchain.py
class VNPMemoryLangChain(BaseMemory):
    """LangChain-compatible memory using VNP Memory backend."""
    client: VNPClient
    user_id: str

    def load_memory_variables(self, inputs: dict) -> dict:
        query = inputs.get("input", "")
        results = asyncio.run(self.client.memory.recall(query=query, user_id=self.user_id))
        return {"history": self._format_results(results)}

    def save_context(self, inputs: dict, outputs: dict) -> None:
        combined = f"Human: {inputs['input']}\nAI: {outputs['output']}"
        asyncio.run(self.client.memory.store(content=combined, type="conversational", user_id=self.user_id))
```

---

## 3. TypeScript SDK Structure

```
sdk/typescript/
  package.json
  tsconfig.json
  src/
    index.ts             # export VNPClient
    client.ts            # main client
    transport.ts         # fetch + retry
    memory.ts
    observe.ts
    events.ts            # WebSocket real-time events
    types.ts             # TypeScript interfaces
  tests/
    memory.test.ts
    events.test.ts
```

### Key TypeScript classes

```typescript
// src/client.ts
export class VNPClient {
  readonly memory: MemoryAPI;
  readonly observe: ObserveAPI;
  readonly events: EventsAPI;

  constructor(config: { apiKey: string; baseUrl?: string }) {
    const transport = new Transport(config);
    this.memory  = new MemoryAPI(transport);
    this.observe = new ObserveAPI(transport);
    this.events  = new EventsAPI(config);
  }
}

// src/memory.ts
export class MemoryAPI {
  async store(req: StoreRequest): Promise<StoreResponse> { ... }
  async recall(req: RecallRequest): Promise<RecallResponse> { ... }
  async forget(userId: string, reason?: string): Promise<ForgetResponse> { ... }
}

// src/events.ts — WebSocket client
export class EventsAPI extends EventEmitter {
  private ws: WebSocket | null = null;
  private reconnectDelay = 1000;

  async connect(token: string): Promise<void> {
    const url = `${this.baseWsUrl}/v1/console/ws?token=${token}`;
    this.ws = new WebSocket(url);
    this.ws.onmessage = (e) => {
      const event = JSON.parse(e.data);
      this.emit(event.event, event.data);
    };
    this.ws.onclose = () => this._scheduleReconnect(token);
  }

  private _scheduleReconnect(token: string) {
    setTimeout(() => {
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
      this.connect(token);
    }, this.reconnectDelay);
  }
  // on('memory_stored', handler), on('session_end', handler) etc.
}
```

---

## 4. Retry Logic (both SDKs)

```python
# Python: _transport.py
MAX_RETRIES = 3
RETRYABLE_STATUS = {429, 500, 502, 503, 504}

async def _request_with_retry(self, method, path, body):
    delay = 1.0
    for attempt in range(MAX_RETRIES + 1):
        resp = await self._client.request(method, path, json=body)
        if resp.status_code not in RETRYABLE_STATUS:
            return resp
        if attempt == MAX_RETRIES:
            raise VNPError(f"max retries exceeded: {resp.status_code}")
        if resp.status_code == 429:
            delay = float(resp.headers.get("Retry-After", delay))
        await asyncio.sleep(delay)
        delay = min(delay * 2, 30)
```

---

## 5. File Changes

| File | Action |
|---|---|
| `sdk/python/vnp_memory/client.py` | **[NEW]** |
| `sdk/python/vnp_memory/memory.py` | **[NEW]** |
| `sdk/python/vnp_memory/observe.py` | **[NEW]** |
| `sdk/python/vnp_memory/integrations/langchain.py` | **[NEW]** |
| `sdk/python/pyproject.toml` | **[NEW]** |
| `sdk/typescript/src/client.ts` | **[NEW]** |
| `sdk/typescript/src/events.ts` | **[NEW]** |
| `sdk/typescript/package.json` | **[NEW]** |
