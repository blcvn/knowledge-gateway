# TASK-ZEP-015 — packages/integrations: Python Framework Packages

**Task ID:** TASK-ZEP-015  
**Wave:** 5 (Integration)  
**Solution:** [SOL-ZEP-006](../solutions/SOL-ZEP-006-Framework-Integrations.md)  
**Depends on:** TASK-ZEP-009 (GetUserContext REST endpoint available)  
**Ước tính:** 6h  
**Priority:** High — developer ecosystem

---

## Mục tiêu

Tạo 4 Python integration packages:
1. `vnp-autogen` — Microsoft AutoGen (`autogen_core.memory.Memory`)
2. `vnp-crewai` — CrewAI (`ZepUserStorage` + `ZepGraphStorage` + tool factories)
3. `vnp-adk` — Google ADK
4. `vnp-livekit` — LiveKit voice agents

---

## Quy tắc chất lượng (áp dụng cho TẤT CẢ packages)

- Python 3.10, 3.11, 3.12, 3.13
- `mypy --strict` pass với 0 errors
- `ruff check` pass
- `pytest` coverage > 90%
- Async-first design (`async def`)

---

## Công việc cụ thể

### Package 1: `packages/integrations/python/vnp-autogen/`

**`src/vnp_autogen/memory.py`**

```python
from autogen_core.memory import Memory, MemoryContent, MemoryQueryResult, MemoryMimeType
from autogen_core.model_context import ChatCompletionContext
from vnp_memory import AsyncVNPMemory, Message

class ZepMemory(Memory):
    """VNP Memory backend for Microsoft AutoGen.
    
    Storage routing:
      content.metadata["type"] == "message" → thread.add_messages()
      content.metadata["type"] == "text"|"json" → graph.add()
    """
    
    def __init__(self, client: AsyncVNPMemory, user_id: str, thread_id: str | None = None) -> None: ...
    
    async def add(self, content: MemoryContent, cancellation_token: Any = None) -> None:
        """Route content to Thread or Graph based on type metadata."""
        ...
    
    async def query(self, query: str | MemoryContent, cancellation_token: Any = None) -> MemoryQueryResult:
        """Get user context + graph search results."""
        ...
    
    async def update_context(self, model_context: ChatCompletionContext) -> None:
        """Inject VNP Memory context into AutoGen conversation context."""
        ...
    
    async def clear(self) -> None:
        """Delete session memory."""
        ...
```

**`pyproject.toml`**
```toml
[project]
name = "vnp-autogen"
version = "1.0.0"
requires-python = ">=3.10"
dependencies = ["autogen-core>=0.4.0", "vnp-memory>=1.0.0"]

[tool.mypy]
strict = true
python_version = "3.10"
```

**Tests `tests/test_memory.py`:**
- `test_add_message_routes_to_thread`: type="message" → thread.add_messages called
- `test_add_text_routes_to_graph`: type="text" → graph.add called
- `test_query_returns_context`: mock client → MemoryQueryResult with context
- `test_update_context_injects_system_message`: context injected as system role
- `test_clear_deletes_session`: clear() → thread.delete_memory called
- `test_mypy_strict`: mypy check passes

---

### Package 2: `packages/integrations/python/vnp-crewai/`

**`src/vnp_crewai/user_storage.py`** — `ZepUserStorage(Storage)`:
```python
# Per-user memory: conversations + personal facts
# save(): route message/text/json via metadata["type"]
# search(): get_user_context() + graph.search() combined results
```

**`src/vnp_crewai/graph_storage.py`** — `ZepGraphStorage(Storage)`:
```python
# Shared team knowledge graph
# save(): graph.add(graph_id=...) for shared knowledge
# search(): graph.search(graph_id=...) for team knowledge
```

**`src/vnp_crewai/tools.py`** — Tool factories:
```python
def create_search_tool(client: AsyncVNPMemory, user_id: str) -> Tool:
    """Factory: CrewAI Tool that searches VNP Memory graph."""
    ...

def create_add_data_tool(client: AsyncVNPMemory, user_id: str) -> Tool:
    """Factory: CrewAI Tool that adds data to VNP Memory graph."""
    ...
```

**Tests `tests/test_user_storage.py`:**
- `test_save_message_goes_to_thread`
- `test_save_json_goes_to_graph`
- `test_search_combines_context_and_graph`
- `test_create_search_tool_is_crewai_tool`
- `test_create_add_data_tool_calls_graph_add`

---

### Package 3: `packages/integrations/python/vnp-adk/`

**`src/vnp_adk/memory.py`** — `ZepADKMemory(BaseMemory)`:
```python
class ZepADKMemory(BaseMemory):
    async def add_session(self, session: Any) -> None:
        """Store ADK session history as thread messages."""
    
    async def search_memory(self, query: str, config: dict | None = None) -> MemoryResult:
        """Search via get_user_context()."""
```

---

### Package 4: `packages/integrations/python/vnp-livekit/`

**`src/vnp_livekit/memory.py`** — `ZepLiveKitMemory`:
```python
class ZepLiveKitMemory:
    async def get_context_for_prompt(self) -> str:
        """Get formatted context for voice agent system prompt."""
    
    async def save_conversation_turn(self, role: str, content: str) -> None:
        """Save voice turn to thread."""
    
    @classmethod
    def attach_to_agent(cls, agent: PipelineAgent, client: AsyncVNPMemory, user_id: str) -> "ZepLiveKitMemory":
        """Factory: auto-hook into agent events for save_conversation_turn."""
```

---

### Shared Makefile Template (mỗi package đều có)

```makefile
# packages/integrations/python/vnp-{framework}/Makefile

.PHONY: test type-check lint format ci

test:
	pytest tests/ -v --asyncio-mode=auto \
		--cov=vnp_{framework} \
		--cov-report=term-missing \
		--cov-fail-under=90

type-check:
	mypy src/ --strict --python-version 3.10

lint:
	ruff check src/ tests/

format:
	ruff format src/ tests/

ci: lint type-check test
	@echo "✅ All checks passed"
```

### GitHub Actions Matrix

**`.github/workflows/python-integrations.yml`**:
```yaml
strategy:
  matrix:
    python-version: ["3.10", "3.11", "3.12", "3.13"]
    package: ["vnp-autogen", "vnp-crewai", "vnp-adk", "vnp-livekit"]
```

---

## Storage Routing Convention (phải nhất quán trong tất cả packages)

| Data Type | metadata `type` | VNP Memory Destination |
|-----------|-----------------|------------------------|
| Chat message | `"message"` | Thread (`thread.add_messages()`) |
| Plain text | `"text"` | User Graph (`graph.add(type="text")`) |
| JSON data | `"json"` | User Graph (`graph.add(type="json")`) |
| Team knowledge | N/A (graph_id) | Graph Service (`graph.add(graph_id=...)`) |

---

## Acceptance Criteria

- [ ] `make ci` pass cho TẤT CẢ 4 packages (lint + mypy + test)
- [ ] `mypy --strict` 0 errors (không có `Any` untyped)
- [ ] pytest coverage > 90% trong mỗi package
- [ ] `ZepMemory(Memory)` implements tất cả abstract methods của autogen_core
- [ ] `ZepUserStorage(Storage)` implements tất cả abstract methods của crewai
- [ ] Storage routing: type="message" → thread ALWAYS; type="json" → graph ALWAYS
- [ ] `attach_to_agent()` tự động hook vào LiveKit agent events

---

## Files tạo ra

```
packages/integrations/python/
├── vnp-autogen/
│   ├── src/vnp_autogen/__init__.py
│   ├── src/vnp_autogen/memory.py
│   ├── src/vnp_autogen/exceptions.py
│   ├── tests/test_memory.py
│   ├── pyproject.toml
│   └── Makefile
├── vnp-crewai/
│   ├── src/vnp_crewai/__init__.py
│   ├── src/vnp_crewai/user_storage.py
│   ├── src/vnp_crewai/graph_storage.py
│   ├── src/vnp_crewai/tools.py
│   ├── tests/
│   ├── pyproject.toml
│   └── Makefile
├── vnp-adk/
│   ├── src/vnp_adk/memory.py
│   ├── tests/
│   ├── pyproject.toml
│   └── Makefile
└── vnp-livekit/
    ├── src/vnp_livekit/memory.py
    ├── tests/
    ├── pyproject.toml
    └── Makefile
.github/workflows/python-integrations.yml
```

## Sau khi hoàn thành

Với mỗi package:
```bash
cd packages/integrations/python/vnp-{framework}
make ci
```
