# SOL-SM-010 — Solution: Framework Integrations & SDK

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-010 |
| **CR** | CR-SM-010 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `backend/apps` |

---

## 1. Giải pháp

SDK wrappers for LangChain, AutoGen, LlamaIndex, CrewAI.

```python
# Python SDK: pip install vnp-memory
from vnp_memory import MemoryClient

client = MemoryClient(api_key="...", base_url="https://api.vnp-memory.io")

# LangChain integration
from vnp_memory.integrations.langchain import VNPMemory
memory = VNPMemory(client=client, user_id="u_123")

# AutoGen integration  
from vnp_memory.integrations.autogen import VNPMemoryPlugin
```

Go SDK: simple REST wrapper in `shared/pkg/client/`.

## 2. Acceptance Criteria

- [ ] Python SDK: pip installable, works with LangChain + AutoGen
- [ ] Go SDK: module publishable
- [ ] SDK docs with examples for each framework

