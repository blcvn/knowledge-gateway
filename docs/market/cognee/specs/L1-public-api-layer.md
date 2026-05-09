# L1 — Public API Layer

> **Layer**: 1 (Presentation)  
> **Responsibility**: Expose Cognee's capabilities to external consumers via multiple interfaces  
> **Dependencies**: L2 (Core Operations)

---

## 1. Tổng Quan

Layer 1 là tầng tiếp xúc trực tiếp với người dùng và hệ thống bên ngoài. Nó cung cấp 4 interface song song:

| Interface | Path | Protocol | Mô tả |
|-----------|------|----------|--------|
| **Python SDK** | `cognee/__init__.py` | Python import | `import cognee; await cognee.add(...)` |
| **REST API** | `cognee/api/v1/` | HTTP/JSON (FastAPI) | `/add`, `/cognify`, `/search`, etc. |
| **CLI** | `cognee/cli/` | Terminal | `cognee-cli add "text"` |
| **MCP Server** | `cognee-mcp/` | Model Context Protocol (SSE/stdio) | IDE integration (Cursor, VS Code) |

---

## 2. Python SDK (`cognee/__init__.py`)

### 2.1 V1 API Exports

```python
from cognee import add, cognify, search, memify, delete, update, prune
from cognee import config, datasets, session
from cognee import SearchType
from cognee import visualize_graph, start_visualization_server, start_ui
from cognee import run_custom_pipeline, pipelines, Drop
```

### 2.2 V2 Memory API Exports

```python
from cognee import remember, RememberResult, recall, improve, forget
from cognee import serve, disconnect, visualize
from cognee import MemoryEntry, QAEntry, TraceEntry, FeedbackEntry
```

### 2.3 Observability Exports

```python
from cognee import enable_tracing, disable_tracing, get_last_trace, get_all_traces, clear_traces
```

### 2.4 Agent Memory Export

```python
from cognee import agent_memory  # decorator cho async agent functions
```

### 2.5 Session Models

```python
from cognee import SessionModelUsage, SessionRecord
```

**Tất cả các hàm đều là `async`** — sử dụng `await` hoặc `asyncio.run()`.

---

## 3. REST API (`cognee/api/v1/`)

FastAPI application với versioned routes.

### 3.1 Route Directory Map

| Route Dir | Endpoint | Method | Chức năng |
|-----------|----------|--------|-----------|
| `add/` | `/add` | POST | Ingest data vào dataset |
| `cognify/` | `/cognify` | POST | Build knowledge graph |
| `search/` | `/search` | POST | Query knowledge |
| `memify/` | `/memify` | POST | Enrich graph |
| `remember/` | `/remember` | POST | V2: add + cognify atomic |
| `recall/` | `/recall` | POST | V2: session-aware search |
| `improve/` | `/improve` | POST | V2: self-improvement |
| `forget/` | `/forget` | POST | V2: delete memory |
| `delete/` | `/delete` | DELETE | Delete data |
| `update/` | `/update` | PUT | Update data |
| `prune/` | `/prune` | POST | Prune obsolete data |
| `datasets/` | `/datasets` | GET/POST/DELETE | Dataset CRUD |
| `users/` | `/users` | GET/POST | User management |
| `permissions/` | `/permissions` | GET/POST | ACL management |
| `ontologies/` | `/ontologies` | GET/POST | Ontology management |
| `health/` | `/health` | GET | Health check |
| `config/` | `/config` | GET/POST | Configuration |
| `settings/` | `/settings` | GET/POST | Runtime settings |
| `activity/` | `/activity` | GET | Pipeline run history |
| `api_keys/` | `/api-keys` | GET/POST | API key management |
| `serve/` | `/serve` | POST | Connect to Cognee Cloud |
| `session/` | `/session` | GET/POST | Session management |
| `sessions/` | `/sessions` | GET | List sessions |
| `visualize/` | `/visualize` | GET | Graph visualization server |
| `llm/` | `/llm` | POST | Direct LLM interaction |
| `sync/` | `/sync` | POST | Data synchronization |
| `cloud/` | `/cloud` | POST | Cloud operations |
| `ui/` | `/ui` | GET | Launch frontend UI |
| `notebooks/` | `/notebooks` | GET | Notebook endpoints |

### 3.2 Authentication

- Built on **FastAPI Users** (`modules/users/get_fastapi_users.py`)
- JWT bearer token khi `REQUIRE_AUTHENTICATION=True`
- Default unauthenticated mode: single user `default_user@example.com`
- API key support: `cognee/api/v1/api_keys/`

### 3.3 Client SDK (`cognee/api/client.py`)

HTTP client wrapper cho remote Cognee instance:

```python
from cognee.api.client import CogneeClient

client = CogneeClient(url="https://instance.cognee.ai", api_key="ck_...")
await client.add(data, dataset_name)
await client.cognify(datasets)
results = await client.search(query_text, search_type)
```

### 3.4 DTO Layer (`cognee/api/DTO.py`)

Data Transfer Objects cho request/response serialization.

---

## 4. CLI (`cognee/cli/`)

Terminal interface cho Cognee operations:

```bash
# Data operations
cognee-cli add "Your text here"
cognee-cli cognify
cognee-cli search "Your query"
cognee-cli delete --all

# V2 Memory operations
cognee-cli remember "Important context"
cognee-cli recall "What happened?"
cognee-cli forget --all

# UI
cognee-cli -ui    # Launch full stack at http://localhost:3000
```

---

## 5. MCP Server (`cognee-mcp/`)

Standalone **Model Context Protocol** server cho IDE integration:

| Thuộc tính | Giá trị |
|------------|---------|
| Transport | SSE (`TRANSPORT_MODE=sse`) hoặc stdio |
| Port | `8000` (MCP), `5678` (debugger) |
| Docker profile | `--profile mcp` |
| Target IDEs | Cursor, Claude Desktop, VS Code (Cline/Roo) |

Expose tất cả Cognee memory operations dưới dạng MCP tools.

---

## 6. Frontend UI (`cognee-frontend/`)

| Thuộc tính | Giá trị |
|------------|---------|
| Framework | Next.js |
| Port | `3000` |
| Docker profile | `--profile ui` |
| Chức năng | Graph visualization, data management |

---

## 7. Key Files

| File | Chức năng |
|------|-----------|
| `cognee/__init__.py` | SDK entry points, tất cả public exports |
| `cognee/__main__.py` | CLI entry point |
| `cognee/api/v1/__init__.py` | V2 Memory API re-exports |
| `cognee/api/client.py` | Remote HTTP client |
| `cognee/api/DTO.py` | Data Transfer Objects |
