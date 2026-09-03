# TASK-PLAT-030 — Python SDK Implementation

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-030 |
| **Wave** | 4 |
| **Solution** | [SOL-PLAT-010](../solutions/SOL-PLAT-010-SDK-Client-Libraries.md) §2 |
| **Component** | `sdk/python/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 4h |

**Trạng thái:** ⏳ Pending  
**Ghi chú audit:** Python SDK not implemented — no sdk/ directory found
---

## Mục tiêu

Tạo Python SDK `vnp-memory` với async client, auto-retry, LangChain integration.

---

## Công việc cụ thể

### 1. Tạo `sdk/python/pyproject.toml` [NEW]

```toml
[project]
name = "vnp-memory"
version = "0.1.0"
requires-python = ">=3.10"
dependencies = ["httpx>=0.27", "pydantic>=2", "typing-extensions"]

[project.optional-dependencies]
langchain = ["langchain>=0.1"]
crewai = ["crewai>=0.1"]
```

### 2. Tạo `sdk/python/vnp_memory/_transport.py` [NEW]

```python
import asyncio, httpx

RETRYABLE = {429, 500, 502, 503, 504}
MAX_RETRIES = 3

class AsyncTransport:
    def __init__(self, api_key: str, base_url: str):
        self._client = httpx.AsyncClient(
            base_url=base_url,
            headers={"X-API-Key": api_key, "Content-Type": "application/json"},
            timeout=30.0
        )

    async def post(self, path: str, body: dict) -> dict:
        delay = 1.0
        for attempt in range(MAX_RETRIES + 1):
            resp = await self._client.post(path, json=body)
            if resp.status_code not in RETRYABLE:
                resp.raise_for_status()
                return resp.json()
            if attempt == MAX_RETRIES:
                raise VNPError(f"max retries exceeded ({resp.status_code})")
            if resp.status_code == 429:
                delay = float(resp.headers.get("Retry-After", delay))
            await asyncio.sleep(delay)
            delay = min(delay * 2, 30.0)

    async def get(self, path: str, params: dict = None) -> dict:
        resp = await self._client.get(path, params=params)
        resp.raise_for_status()
        return resp.json()
```

### 3. Tạo `sdk/python/vnp_memory/memory.py` [NEW]

```python
class MemoryAPI:
    def __init__(self, transport):
        self._t = transport

    async def store(self, content: str, type: str = "auto", user_id: str = "", **metadata) -> dict:
        return await self._t.post("/v1/memory/store", {"content": content, "type": type, "user_id": user_id, "metadata": metadata})

    async def recall(self, query: str, user_id: str = "", types: list = None, limit: int = 10) -> dict:
        return await self._t.post("/v1/memory/recall", {"query": query, "user_id": user_id, "types": types or [], "limit": limit})

    async def forget(self, user_id: str, reason: str = "") -> dict:
        return await self._t.post("/v1/memory/forget", {"user_id": user_id, "reason": reason})
```

### 4. Tạo `sdk/python/vnp_memory/integrations/langchain.py` [NEW]

```python
from langchain.memory.chat_memory import BaseChatMemory
import asyncio

class VNPMemoryLangChain(BaseChatMemory):
    client: object  # VNPClient
    user_id: str = ""

    def load_memory_variables(self, inputs: dict) -> dict:
        query = inputs.get("input", "")
        result = asyncio.run(self.client.memory.recall(query=query, user_id=self.user_id, limit=5))
        return {"history": self._format(result)}

    def save_context(self, inputs: dict, outputs: dict) -> None:
        text = f"Human: {inputs.get('input', '')}\nAI: {outputs.get('output', '')}"
        asyncio.run(self.client.memory.store(content=text, type="conversational", user_id=self.user_id))

    def clear(self) -> None:
        if self.user_id:
            asyncio.run(self.client.memory.forget(user_id=self.user_id))
```

### 5. Tests `sdk/python/tests/test_memory.py` [NEW]

```python
import pytest, httpx
from vnp_memory import VNPClient
from unittest.mock import AsyncMock, patch

@pytest.mark.asyncio
async def test_store_success():
    with patch("httpx.AsyncClient.post") as mock:
        mock.return_value = httpx.Response(200, json={"id": "mem-1", "engine": "graphiti"})
        client = VNPClient(api_key="vnp_test.test")
        resp = await client.memory.store("Hello", user_id="u1")
        assert resp["id"] == "mem-1"
```

---

## Acceptance Criteria

- [ ] `VNPClient(api_key=...).memory.store(...)` async works
- [ ] Auto-retry on 429: respects Retry-After header
- [ ] LangChain integration: `load_memory_variables` calls `recall`
- [ ] `go test` equivalent: `pytest sdk/python/tests/` passes

## Files

```
sdk/python/pyproject.toml
sdk/python/vnp_memory/__init__.py
sdk/python/vnp_memory/_transport.py
sdk/python/vnp_memory/memory.py
sdk/python/vnp_memory/observe.py
sdk/python/vnp_memory/client.py
sdk/python/vnp_memory/integrations/langchain.py
sdk/python/tests/test_memory.py
```
