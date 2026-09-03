# Solution: SOL-ZEP-006 — Framework Integrations (AutoGen, CrewAI, ADK, LiveKit)

**CR ID:** CR-ZEP-006  
**Solution ID:** SOL-ZEP-006  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo 4 Python integration packages trong `packages/integrations/python/` với kiến trúc Storage Routing Convention rõ ràng: messages → Thread Service, data → Graph Service. Mỗi package implement đúng interface của framework tương ứng với mypy strict typing và coverage > 90%.

---

## 2. Cấu trúc Package

```
packages/integrations/python/
├── vnp-autogen/
│   ├── src/vnp_autogen/
│   │   ├── __init__.py        # export ZepMemory
│   │   ├── memory.py          # ZepMemory(autogen_core.memory.Memory)
│   │   ├── types.py           # Internal type aliases
│   │   └── exceptions.py      # VNPAutogenError
│   ├── tests/
│   │   └── test_memory.py
│   ├── pyproject.toml
│   └── Makefile               # test, type-check, lint
│
├── vnp-crewai/
│   ├── src/vnp_crewai/
│   │   ├── __init__.py        # export ZepUserStorage, ZepGraphStorage, create_*_tool
│   │   ├── user_storage.py    # ZepUserStorage
│   │   ├── graph_storage.py   # ZepGraphStorage
│   │   └── tools.py           # create_search_tool(), create_add_data_tool()
│   ├── tests/
│   ├── pyproject.toml
│   └── Makefile
│
├── vnp-adk/
│   ├── src/vnp_adk/
│   │   ├── __init__.py
│   │   └── memory.py          # ZepADKMemory(BaseMemory)
│   ├── tests/
│   ├── pyproject.toml
│   └── Makefile
│
└── vnp-livekit/
    ├── src/vnp_livekit/
    │   ├── __init__.py
    │   └── memory.py          # ZepLiveKitMemory(BaseMemory)
    ├── tests/
    ├── pyproject.toml
    └── Makefile
```

---

## 3. AutoGen Integration (`vnp-autogen`)

### 3.1. pyproject.toml

```toml
[project]
name = "vnp-autogen"
version = "1.0.0"
description = "VNP Memory integration for Microsoft AutoGen"
requires-python = ">=3.10"
dependencies = [
    "autogen-core>=0.4.0",
    "vnp-memory>=1.0.0",  # Python SDK
]

[project.optional-dependencies]
dev = ["pytest", "pytest-asyncio", "mypy", "ruff"]
```

### 3.2. ZepMemory Implementation

```python
# packages/integrations/python/vnp-autogen/src/vnp_autogen/memory.py

from __future__ import annotations

from typing import Literal
import asyncio

from autogen_core.memory import (
    Memory,
    MemoryContent,
    MemoryQueryResult,
    MemoryMimeType,
)
from autogen_core.model_context import ChatCompletionContext

from vnp_memory import AsyncVNPMemory, Message


class ZepMemory(Memory):
    """VNP Memory implementation for Microsoft AutoGen.
    
    Routes content based on type:
    - "message" → Thread Service (conversation messages)
    - "text" | "json" → Graph Service (knowledge data)
    """

    def __init__(
        self,
        client: AsyncVNPMemory,
        user_id: str,
        thread_id: str | None = None,
    ) -> None:
        self._client = client
        self._user_id = user_id
        self._thread_id = thread_id

    async def add(self, content: MemoryContent, cancellation_token=None) -> None:
        """Route content to appropriate VNP Memory service based on type."""
        content_type = content.metadata.get("type", "message") if content.metadata else "message"
        
        if content_type == "message":
            # Conversation messages → Thread Service
            if not self._thread_id:
                raise ValueError("thread_id required for message routing")
            role = content.metadata.get("role", "user") if content.metadata else "user"
            await self._client.thread.add_messages(
                thread_id=self._thread_id,
                messages=[Message(
                    role=role,
                    role_type="user" if role == "user" else "assistant",
                    content=str(content.content),
                )]
            )
        else:
            # Knowledge data → Graph Service
            await self._client.graph.add(
                user_id=self._user_id,
                data=str(content.content),
                type=content_type,  # "text" | "json"
            )

    async def query(
        self,
        query: str | MemoryContent,
        cancellation_token=None,
    ) -> MemoryQueryResult:
        """Search VNP Memory: get user context + relevant facts."""
        query_str = query if isinstance(query, str) else str(query.content)
        
        # Get pre-formatted context (includes facts + recent messages)
        if self._thread_id:
            context_resp = await self._client.thread.get_user_context(
                thread_id=self._thread_id,
            )
            context_text = context_resp.context
        else:
            # Fallback: search graph
            search_resp = await self._client.graph.search(
                user_id=self._user_id,
                query=query_str,
                limit=5,
            )
            context_text = "\n".join(r.fact for r in search_resp.items if r.fact)

        return MemoryQueryResult(
            results=[MemoryContent(
                content=context_text,
                mime_type=MemoryMimeType.TEXT,
            )]
        )

    async def update_context(self, model_context: ChatCompletionContext) -> None:
        """Inject VNP Memory context into the conversation context."""
        if not self._thread_id:
            return
        context_resp = await self._client.thread.get_user_context(
            thread_id=self._thread_id
        )
        if context_resp.context:
            await model_context.add_message({
                "role": "system",
                "content": context_resp.context,
            })

    async def clear(self) -> None:
        """Delete session memory."""
        if self._thread_id:
            await self._client.thread.delete_memory(thread_id=self._thread_id)
```

---

## 4. CrewAI Integration (`vnp-crewai`)

### 4.1. pyproject.toml

```toml
[project]
name = "vnp-crewai"
version = "1.0.0"
requires-python = ">=3.10"
dependencies = [
    "crewai>=0.67.0",
    "vnp-memory>=1.0.0",
]
```

### 4.2. ZepUserStorage (Per-User Memory)

```python
# packages/integrations/python/vnp-crewai/src/vnp_crewai/user_storage.py

from __future__ import annotations

from typing import Any, Literal
from crewai.memory.storage.interface import Storage

from vnp_memory import AsyncVNPMemory, Message, SearchInput


class ZepUserStorage(Storage):
    """Per-user memory storage for CrewAI agents.
    
    Stores conversation messages and retrieves personalized context.
    Use for individual user interactions.
    """

    def __init__(
        self,
        client: AsyncVNPMemory,
        user_id: str,
        thread_id: str | None = None,
        facts_limit: int = 20,
        entity_limit: int = 5,
        mode: Literal["summary", "raw_messages"] = "summary",
    ) -> None:
        self._client = client
        self._user_id = user_id
        self._thread_id = thread_id or f"crewai_{user_id}"
        self._facts_limit = facts_limit
        self._entity_limit = entity_limit
        self._mode = mode

    def save(self, value: Any, metadata: dict[str, Any]) -> None:
        """Save message or data to user memory."""
        import asyncio
        content_type = metadata.get("type", "message")
        
        if content_type == "message":
            asyncio.get_event_loop().run_until_complete(
                self._client.thread.add_messages(
                    thread_id=self._thread_id,
                    messages=[Message(
                        role=metadata.get("role", "user"),
                        content=str(value),
                    )]
                )
            )
        else:
            asyncio.get_event_loop().run_until_complete(
                self._client.graph.add(
                    user_id=self._user_id,
                    data=str(value),
                    type=content_type,
                )
            )

    def search(self, query: str, limit: int = 3, score_threshold: float = 0.0) -> list[Any]:
        """Search user memory for relevant context."""
        import asyncio

        async def _search() -> list[Any]:
            # Get pre-formatted context combining facts + messages
            context = await self._client.thread.get_user_context(
                thread_id=self._thread_id,
            )
            # Also do graph search for additional facts
            graph_results = await self._client.graph.search(
                user_id=self._user_id,
                query=query,
                scope="edges",
                limit=self._facts_limit,
            )
            return [
                {"value": context.context, "score": 1.0, "source": "context"},
                *[{"value": r.fact.fact, "score": r.score, "source": "graph"}
                  for r in graph_results.items if r.fact],
            ]

        results = asyncio.get_event_loop().run_until_complete(_search())
        return [r for r in results if r.get("score", 0) >= score_threshold][:limit]
```

### 4.3. ZepGraphStorage (Shared Knowledge Graph)

```python
# packages/integrations/python/vnp-crewai/src/vnp_crewai/graph_storage.py

from __future__ import annotations

from typing import Any
from crewai.memory.storage.interface import Storage

from vnp_memory import AsyncVNPMemory


class ZepGraphStorage(Storage):
    """Shared knowledge graph storage for CrewAI teams.
    
    Stores team-wide knowledge (product data, company info, research).
    All agents on the team share the same knowledge graph.
    Use alongside ZepUserStorage for dual-storage architecture.
    """

    def __init__(self, client: AsyncVNPMemory, graph_id: str) -> None:
        self._client = client
        self._graph_id = graph_id

    def save(self, value: Any, metadata: dict[str, Any]) -> None:
        """Save data to shared knowledge graph."""
        import asyncio
        content_type = metadata.get("type", "text")
        asyncio.get_event_loop().run_until_complete(
            self._client.graph.add(
                graph_id=self._graph_id,
                data=str(value),
                type=content_type,
            )
        )

    def search(self, query: str, limit: int = 5, score_threshold: float = 0.0) -> list[Any]:
        """Search shared knowledge graph."""
        import asyncio

        async def _search() -> list[Any]:
            results = await self._client.graph.search(
                graph_id=self._graph_id,
                query=query,
                scope="edges",
                reranker="rrf",
                limit=limit,
            )
            return [
                {"value": r.fact.fact, "score": r.score}
                for r in results.items
                if r.fact and r.score >= score_threshold
            ]

        return asyncio.get_event_loop().run_until_complete(_search())
```

### 4.4. Tool Factories

```python
# packages/integrations/python/vnp-crewai/src/vnp_crewai/tools.py

from __future__ import annotations

from crewai.tools import Tool
from vnp_memory import AsyncVNPMemory


def create_search_tool(client: AsyncVNPMemory, user_id: str) -> Tool:
    """Create a CrewAI Tool for searching VNP Memory knowledge graph."""
    
    def search_memory(query: str) -> str:
        """Search VNP Memory for relevant facts about the user."""
        import asyncio
        results = asyncio.get_event_loop().run_until_complete(
            client.graph.search(
                user_id=user_id,
                query=query,
                scope="edges",
                reranker="rrf",
                limit=10,
            )
        )
        if not results.items:
            return "No relevant information found."
        return "\n".join(
            f"- {r.fact.fact}" + (
                f" (valid: {r.fact.valid_at.strftime('%Y-%m')})" 
                if r.fact and r.fact.valid_at else ""
            )
            for r in results.items if r.fact
        )

    return Tool(
        name="search_vnp_memory",
        description="Search VNP Memory for relevant facts, entities, and context about the current user.",
        func=search_memory,
    )


def create_add_data_tool(client: AsyncVNPMemory, user_id: str) -> Tool:
    """Create a CrewAI Tool for adding data to VNP Memory."""
    
    def add_to_memory(data: str, data_type: str = "text") -> str:
        """Add information to VNP Memory knowledge graph."""
        import asyncio
        asyncio.get_event_loop().run_until_complete(
            client.graph.add(
                user_id=user_id,
                data=data,
                type=data_type,
            )
        )
        return f"Successfully added {data_type} data to VNP Memory."

    return Tool(
        name="add_to_vnp_memory",
        description="Add information (text or JSON) to VNP Memory knowledge graph for future retrieval.",
        func=add_to_memory,
    )
```

---

## 5. Google ADK Integration (`vnp-adk`)

```python
# packages/integrations/python/vnp-adk/src/vnp_adk/memory.py

from __future__ import annotations

from google.adk.memory import BaseMemory, MemoryResult
from vnp_memory import AsyncVNPMemory


class ZepADKMemory(BaseMemory):
    """VNP Memory implementation for Google ADK agents."""

    def __init__(self, client: AsyncVNPMemory, user_id: str, session_id: str) -> None:
        self._client = client
        self._user_id = user_id
        self._session_id = session_id

    async def add_session(self, session) -> None:
        """Store ADK session as messages in VNP Memory thread."""
        messages = [
            Message(role=m.role, content=m.content)
            for m in session.history
        ]
        await self._client.thread.add_messages(
            thread_id=self._session_id,
            messages=messages,
        )

    async def search_memory(self, query: str, config: dict | None = None) -> MemoryResult:
        """Search VNP Memory graph for relevant context."""
        context = await self._client.thread.get_user_context(
            thread_id=self._session_id,
        )
        return MemoryResult(memories=[{"text": context.context}])
```

---

## 6. LiveKit Integration (`vnp-livekit`)

```python
# packages/integrations/python/vnp-livekit/src/vnp_livekit/memory.py

from __future__ import annotations

from livekit.agents.pipeline.pipeline_agent import PipelineAgent
from vnp_memory import AsyncVNPMemory


class ZepLiveKitMemory:
    """VNP Memory integration for LiveKit voice agents."""

    def __init__(self, client: AsyncVNPMemory, user_id: str, session_id: str) -> None:
        self._client = client
        self._user_id = user_id
        self._session_id = session_id

    async def get_context_for_prompt(self) -> str:
        """Get user context for injection into voice agent system prompt."""
        resp = await self._client.thread.get_user_context(
            thread_id=self._session_id,
        )
        return resp.context

    async def save_conversation_turn(self, role: str, content: str) -> None:
        """Save a conversation turn to VNP Memory."""
        await self._client.thread.add_messages(
            thread_id=self._session_id,
            messages=[Message(role=role, content=content)]
        )

    @classmethod
    def attach_to_agent(cls, agent: PipelineAgent, client: AsyncVNPMemory, user_id: str) -> "ZepLiveKitMemory":
        """Factory: attach VNP Memory to a LiveKit pipeline agent."""
        session_id = f"livekit_{user_id}_{agent.room.name}"
        memory = cls(client, user_id, session_id)

        @agent.on("user_speech_committed")
        async def on_user_speech(message: str):
            await memory.save_conversation_turn("user", message)

        @agent.on("agent_speech_committed")
        async def on_agent_speech(message: str):
            await memory.save_conversation_turn("assistant", message)

        return memory
```

---

## 7. Quality Standards

```makefile
# packages/integrations/python/vnp-autogen/Makefile (pattern cho tất cả packages)

.PHONY: test type-check lint format

test:
	pytest tests/ -v --asyncio-mode=auto --cov=vnp_autogen --cov-report=term-missing
	@coverage report --fail-under=90

type-check:
	mypy src/ --strict --python-version 3.10

lint:
	ruff check src/ tests/

format:
	ruff format src/ tests/

ci: lint type-check test
```

### Testing Matrix

```yaml
# .github/workflows/python-integrations.yml
matrix:
  python-version: ["3.10", "3.11", "3.12", "3.13"]
  package: [vnp-autogen, vnp-crewai, vnp-adk, vnp-livekit]
```

---

## 8. Storage Routing Convention

| Data Type | Metadata `type` | VNP Memory Destination | API Call |
|-----------|----------------|----------------------|---------|
| Chat messages | `"message"` | Thread | `thread.add_messages()` |
| Plain text | `"text"` | User Graph | `graph.add(type="text")` |
| JSON data | `"json"` | User Graph | `graph.add(type="json")` |
| Shared knowledge | N/A | Graph Service with graph_id | `graph.add(graph_id=...)` |

---

## 9. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | vnp-autogen: ZepMemory với mypy strict | 2 ngày |
| **P2** | vnp-crewai: ZepUserStorage + ZepGraphStorage | 2 ngày |
| **P3** | vnp-crewai: Tool factories (search + add_data) | 1 ngày |
| **P4** | vnp-adk: ZepADKMemory | 1.5 ngày |
| **P5** | vnp-livekit: ZepLiveKitMemory + attach_to_agent | 1.5 ngày |
| **P6** | Tests (coverage > 90% cho mỗi package) | 3 ngày |
| **P7** | CI setup (GitHub Actions matrix) | 0.5 ngày |
| **P8** | PyPI publishing setup | 0.5 ngày |

**Tổng:** ~12 ngày (Wave 5)

---

## 10. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| AutoGen ZepMemory implements Memory interface | `class ZepMemory(Memory)` với tất cả abstract methods |
| mypy pass với 0 errors | `mypy src/ --strict` trong Makefile |
| CrewAI: ZepUserStorage per-user + ZepGraphStorage shared | Separate classes với routing convention |
| create_search_tool() → CrewAI Tool gọi VNP Memory search | Tool(func=search_memory) đúng chuẩn |
| type="message" → thread; type="json" → graph | Storage Routing Convention trong tất cả packages |
| `make test` coverage > 90% | `--cov-report --fail-under=90` |
| `make type-check` → mypy 0 errors | `--strict` mode, no `Any` |
