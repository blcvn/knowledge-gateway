# Change Request: CR-ZEP-006 — Framework Integrations (AutoGen, CrewAI, ADK, LiveKit)

**CR ID:** CR-ZEP-006  
**Component:** `packages/integrations/` [NEW PACKAGES]  
**Priority:** High  
**Status:** In Progress
**Reference:** Zep PRD §6.2 F5-F6, SRS §5.3, URD §3.3  
**Frameworks:** Microsoft AutoGen, CrewAI, Google ADK, LiveKit

---

## 1. Mô tả

Xây dựng **Framework Integration Packages** cho VNP Memory — cho phép AI frameworks sử dụng VNP Memory như memory backend native:

1. **AutoGen Integration**: Implement `autogen_core.memory.Memory` interface.
2. **CrewAI Integration**: Dual storage — `ZepUserStorage` (per-user) + `ZepGraphStorage` (shared KG).
3. **Google ADK Integration**: ADK memory interface.
4. **LiveKit Integration**: LiveKit memory interface.
5. **Tool Factories**: `create_search_tool()` và `create_add_data_tool()` để tạo framework tools.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có Python integration packages cho các AI frameworks phổ biến.
- Developers phải tự implement memory interface cho mỗi framework → nhiều boilerplate.
- Thiếu standardized storage routing: messages → thread, data → graph.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] Package Structure

```
packages/integrations/python/
├── vnp-autogen/           # Microsoft AutoGen integration
│   ├── src/vnp_autogen/
│   │   ├── __init__.py
│   │   ├── memory.py      # ZepMemory(autogen_core.memory.Memory)
│   │   └── exceptions.py
│   ├── tests/
│   ├── pyproject.toml
│   └── Makefile
├── vnp-crewai/            # CrewAI integration
│   ├── src/vnp_crewai/
│   │   ├── user_storage.py   # ZepUserStorage
│   │   ├── graph_storage.py  # ZepGraphStorage
│   │   └── tools.py          # create_search_tool(), create_add_data_tool()
│   └── ...
├── vnp-adk/               # Google ADK integration
│   └── ...
└── vnp-livekit/           # LiveKit integration
    └── ...
```

### 3.2. AutoGen Integration (`vnp-autogen`)

```python
from vnp_autogen import ZepMemory
from autogen_core.memory import Memory

class ZepMemory(Memory):
    """VNP Memory implementation for Microsoft AutoGen."""
    
    def __init__(self, client: AsyncZep, user_id: str, thread_id: str | None = None):
        self.client = client
        self.user_id = user_id
        self.thread_id = thread_id
    
    async def add(self, content: MemoryContent, cancellation_token=None) -> None:
        """Route to thread or graph based on content type."""
        if content.metadata.get("type") == "message":
            await self.client.thread.add_messages(
                thread_id=self.thread_id,
                messages=[Message(role=content.role, content=str(content.content))]
            )
        else:
            await self.client.graph.add(
                user_id=self.user_id,
                data=str(content.content),
                type=content.metadata.get("type", "text")
            )
    
    async def query(self, query: str, cancellation_token=None) -> MemoryQueryResult:
        """Search graph + get context."""
        context = await self.client.thread.get_user_context(thread_id=self.thread_id)
        return MemoryQueryResult(results=[MemoryContent(content=context.context)])
    
    async def clear(self) -> None:
        if self.thread_id:
            await self.client.thread.delete_memory(thread_id=self.thread_id)
```

### 3.3. CrewAI Integration (`vnp-crewai`)

**Dual Storage Architecture:**

```python
# Per-user memory (conversations, personal facts)
class ZepUserStorage:
    def __init__(
        self,
        client: AsyncZep,
        user_id: str,
        thread_id: str | None = None,
        facts_limit: int = 20,
        entity_limit: int = 5,
        mode: Literal["summary", "raw_messages"] = "summary"
    ): ...

# Shared knowledge graph (team knowledge, product data)
class ZepGraphStorage:
    def __init__(self, client: AsyncZep, graph_id: str): ...

# Tool factories for CrewAI agents
def create_search_tool(client: AsyncZep, user_id: str) -> Tool:
    """Creates a CrewAI Tool for searching VNP Memory."""
    ...

def create_add_data_tool(client: AsyncZep, user_id: str) -> Tool:
    """Creates a CrewAI Tool for adding data to VNP Memory."""
    ...
```

**Usage example:**
```python
from vnp_crewai import ZepUserStorage, ZepGraphStorage, create_search_tool

user_storage = ZepUserStorage(client, user_id="alice", thread_id="session_1")
shared_kg = ZepGraphStorage(client, graph_id="company_knowledge")

search_tool = create_search_tool(client, user_id="alice")
research_agent = Agent(
    role="Research Analyst",
    memory=True,
    tools=[search_tool]
)
```

### 3.4. Storage Routing Convention

| Data Type | Metadata `type` | VNP Memory Destination | API Call |
|-----------|----------------|----------------------|---------|
| Chat messages | `"message"` | Thread | `thread.add_messages()` |
| Text data | `"text"` | User Graph | `graph.add(type="text")` |
| JSON data | `"json"` | User Graph | `graph.add(type="json")` |

### 3.5. Quality Requirements

| Requirement | Spec |
|-------------|------|
| Linting | ruff (Python) |
| Type checking | mypy strict |
| Test coverage | > 90% via pytest |
| Python versions | 3.10, 3.11, 3.12, 3.13 |
| CI | GitHub Actions matrix |
| Release naming | `vnp-{framework}-v{version}` |

---

## 4. Acceptance Criteria

- [ ] AutoGen: `ZepMemory` implements `autogen_core.memory.Memory` interface — mypy passes với no errors.
- [ ] CrewAI: `ZepUserStorage` per-user + `ZepGraphStorage` shared graph đều hoạt động.
- [ ] Tool factories: `create_search_tool()` → tạo CrewAI Tool gọi VNP Memory search.
- [ ] Storage routing: message với `type="message"` → vào thread; `type="json"` → vào graph.
- [ ] `make test` trên mỗi integration package → coverage > 90%.
- [ ] `make type-check` → mypy pass với 0 errors.
