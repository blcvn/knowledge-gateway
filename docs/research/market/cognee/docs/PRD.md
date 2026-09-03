# Product Requirements Document (PRD)

**Product:** Cognee — AI Knowledge Engine  
**Version:** 1.0.2  
**Status:** Production (Beta)  
**Last Updated:** 2026-04-24  
**Authors:** Vasilije Markovic, Boris Arzentar  
**License:** Apache-2.0

---

## 1. Executive Summary

Cognee là một **Knowledge Engine mã nguồn mở** cho phép nạp dữ liệu ở bất kỳ định dạng nào và liên tục học để cung cấp đúng ngữ cảnh cho AI Agent. Nó kết hợp vector search, graph database, và các tiếp cận khoa học nhận thức để làm cho tài liệu vừa có thể tìm kiếm theo ngữ nghĩa vừa được kết nối bằng các mối quan hệ khi chúng thay đổi và phát triển.

### Giá trị cốt lõi

| Pillar | Mô tả |
|--------|--------|
| **Knowledge Infrastructure** | Nhập dữ liệu thống nhất, tìm kiếm graph/vector, chạy local, ontology grounding, multimodal |
| **Persistent & Learning Agents** | Học từ phản hồi, quản lý ngữ cảnh, chia sẻ kiến thức cross-agent |
| **Reliable & Trustworthy Agents** | Agentic user/tenant isolation, traceability, OTEL collector, audit traits |

---

## 2. Problem Statement

### 2.1 Vấn đề hiện tại

AI Agent hiện đại đối mặt với những thách thức căn bản:

1. **Mất ngữ cảnh giữa các phiên**: Agent không nhớ những gì đã xảy ra ở phiên trước.
2. **RAG truyền thống không đủ thông minh**: Chỉ tìm kiếm theo vector similarity, không hiểu mối quan hệ giữa các khái niệm.
3. **Dữ liệu phân tán**: Thông tin nằm rải rác ở nhiều nguồn (files, URLs, databases) mà không có cách thống nhất để truy vấn.
4. **Agent không tự cải thiện**: Không có cơ chế để học từ kết quả và phản hồi trước đó.
5. **Thiếu isolation**: Nhiều user/tenant dùng chung bộ nhớ gây rò rỉ thông tin.

### 2.2 Giải pháp của Cognee

Cognee giải quyết bằng kiến trúc **3-layer memory**:
- **Session memory** (Redis/cache) — ngắn hạn, nhanh
- **Knowledge Graph** (Neo4j/Kuzu) — dài hạn, có quan hệ
- **Vector store** (LanceDB/pgvector/ChromaDB) — semantic search

---

## 3. Target Users

### 3.1 Primary Users

| User Segment | Nhu cầu | Use Case |
|---|---|---|
| **AI/ML Engineers** | Thêm memory vào AI Agent | Customer support bots, coding assistants |
| **Backend Developers** | Tích hợp knowledge base vào ứng dụng | Enterprise search, knowledge management |
| **Data Scientists** | Phân tích và truy vấn knowledge graph | Research analytics, pattern discovery |
| **DevOps/Platform Teams** | Self-host và scale hệ thống memory | Infrastructure management |

### 3.2 Secondary Users

- **Product Teams** dùng Cognee Cloud (managed service)
- **Researchers** nghiên cứu Graph RAG và knowledge representation
- **Claude Code / Hermes users** qua plugin memory

---

## 4. Core Product Features

### 4.1 V1 API — Core Knowledge Operations

#### 4.1.1 `cognee.add()` — Data Ingestion

Nhập dữ liệu vào hệ thống từ nhiều nguồn:

```python
await cognee.add("notes.md", dataset_name="research")
await cognee.add("https://example.com", dataset_name="research")
await cognee.add([file1, file2], dataset_name="research", node_set=["tag1"])
```

**Supported input types:**
- Text strings
- File paths: PDF, TXT, CSV, Audio, Image
- URLs (web scraping)
- Mixed lists

**Optional loaders (plugins):**
- `UnstructuredLoader` — DOC, DOCX, PPTX, EPUB, RTF, RST, XLSX, ORG, etc.
- `AdvancedPdfLoader` (Docling)
- `BeautifulSoupLoader` — HTML parsing
- `DoclingLoader` — advanced document processing

#### 4.1.2 `cognee.cognify()` — Knowledge Graph Construction

Pipeline xử lý dữ liệu thô thành knowledge graph có cấu trúc:

**Processing steps:**
1. Document classification
2. Permission validation
3. Text chunking (configurable chunk_size)
4. Entity extraction (LLM-powered)
5. Relationship detection
6. Graph construction + embeddings
7. Content summarization

```python
await cognee.cognify(
    datasets="research",
    temporal_cognify=True,  # Temporal event extraction
    chunk_size=1024,
    custom_prompt="Extract companies, products, and partnerships."
)
```

#### 4.1.3 `cognee.search()` — Multi-mode Search

| SearchType | Mô tả | Dùng khi nào |
|---|---|---|
| `GRAPH_COMPLETION` | Graph-aware Q&A với LLM | Câu hỏi phức tạp, phân tích |
| `RAG_COMPLETION` | Traditional RAG over chunks | Tìm kiếm tài liệu cụ thể |
| `CHUNKS` | Raw semantic chunk retrieval | Fast retrieval, citations |
| `CHUNKS_LEXICAL` | Keyword / exact-term matching | Tìm kiếm từ khóa chính xác |
| `SUMMARIES` | Pre-generated hierarchical summaries | Overview, abstracts |
| `TRIPLET_COMPLETION` | Subject-predicate-object graph Q&A | Structured knowledge |
| `GRAPH_SUMMARY_COMPLETION` | Graph + summary answers | Comprehensive answers |
| `GRAPH_COMPLETION_COT` | Chain-of-thought reasoning | Deep analysis |
| `GRAPH_COMPLETION_CONTEXT_EXTENSION` | Broader graph context | Extended context |
| `CYPHER` | Raw Cypher graph queries | Advanced graph traversal |
| `NATURAL_LANGUAGE` | NL to graph query | User-friendly graph search |
| `TEMPORAL` | Time-aware graph search | Event timelines |
| `CODING_RULES` | Code patterns & rules | Code assistance |
| `FEELING_LUCKY` | Auto-select best strategy | General purpose |
| `FEEDBACK` | Reinforce retrieval behavior | Improvement loops |

#### 4.1.4 V2 Memory-Oriented API

High-level API cho agent memory patterns:

```python
# Store permanently
await cognee.remember("Cognee turns documents into AI memory.")

# Store in session (syncs to graph in background)
await cognee.remember("User prefers detailed explanations.", session_id="chat_1")

# Auto-routing recall
results = await cognee.recall("What does Cognee do?")

# Session-scoped recall
results = await cognee.recall("What does the user prefer?", session_id="chat_1")

# Delete data
await cognee.forget(dataset="main_dataset")

# Enrich existing graph
await cognee.improve(dataset="research")
```

#### 4.1.5 `memify()` — Graph Enrichment

Làm giàu graph hiện có với derived facts, rules, patterns mà không cần rebuild từ đầu.

```python
await cognee.memify(dataset="research")
```

### 4.2 NodeSets — Memory Scoping

NodeSets cho phép tag và phân vùng memory theo nhiều chiều:

```python
await cognee.add(
    "Customer prefers weekly reports via Slack.",
    dataset_name="crm",
    node_set=["customer_123", "preferences", "support_agent"]
)

# Search only within specific scope
results = await cognee.search(
    query_text="How to respond to this customer?",
    datasets="crm",
    node_name=["customer_123", "preferences"]
)
```

**NodeSet patterns:**
- Per customer: `["customer_123"]`
- Per workflow: `["support_bot", "refund_flow"]`
- Per topic: `["contracts", "vendor_risk"]`
- Per environment: `["prod", "staging"]`
- Per user memory: `["user_42", "preferences"]`

### 4.3 DataPoints — Structured Knowledge Units

`DataPoint` là atomic unit of knowledge, cho phép định nghĩa schema-shaped memory:

```python
from cognee.infrastructure.engine import DataPoint
from cognee.tasks.storage import add_data_points

class ScientificPaper(DataPoint):
    title: str
    authors: list[str]
    methodology: str
    findings: list[str]
    metadata: dict = {"index_fields": ["title", "findings"]}

await add_data_points([paper])
```

### 4.4 Custom Pipelines

```python
from cognee.modules.pipelines.tasks.task import Task

async def my_task(data):
    return processed_data

await cognee.run_custom_pipeline(
    tasks=[Task(my_task)],
    data="input",
    dataset="research"
)
```

### 4.5 Feedback & Self-improvement Loop

```python
# Perform search with interaction saving
results = await cognee.search(
    query_text="What are the main themes?",
    query_type=SearchType.GRAPH_COMPLETION,
    save_interaction=True,
)

# Apply feedback to reinforce useful behavior
await cognee.search(
    query_text="Helpful — captured key technical themes.",
    query_type=SearchType.FEEDBACK,
    last_k=1,
)
```

---

## 5. Infrastructure & Technical Architecture

### 5.1 Database Backends

#### Graph Databases

| Provider | Status | Use Case |
|---|---|---|
| **Kuzu** (default) | Built-in, no setup | Local dev, embedded |
| **Neo4j** | `pip install cognee[neo4j]` | Production, large graphs |
| **Amazon Neptune** | `pip install cognee[neptune]` | AWS cloud deployment |
| **PostgreSQL** (via Apache AGE) | `pip install cognee[postgres]` | Unified DB stack |

#### Vector Databases

| Provider | Status |
|---|---|
| **LanceDB** (default) | Built-in |
| **pgvector** | `pip install cognee[postgres]` |
| **ChromaDB** | `pip install cognee[chromadb]` |

#### Relational Databases

| Provider | Default |
|---|---|
| **SQLite** | Yes (local) |
| **PostgreSQL** | Production |

### 5.2 LLM Provider Support

Cognee hỗ trợ hầu hết LLM providers qua **LiteLLM**:

| Provider | Extra |
|---|---|
| OpenAI (default) | built-in |
| Anthropic | `pip install cognee[anthropic]` |
| Azure OpenAI | `pip install cognee[azure]` |
| Ollama (local) | `pip install cognee[ollama]` |
| Groq | `pip install cognee[groq]` |
| Mistral | `pip install cognee[mistral]` |
| HuggingFace | `pip install cognee[huggingface]` |
| llama.cpp (local) | `pip install cognee[llama-cpp]` |
| BAML (structured output) | `pip install cognee[baml]` |

**Structured Output Frameworks:**
- `instructor` (default) — via Instructor library
- `baml` — via BAML DSL for complex schemas

### 5.3 Embedding Providers

- OpenAI embeddings (default)
- HuggingFace / FastEmbed (local, `pip install cognee[fastembed]`)
- Configurable dimensions và model

### 5.4 Storage Backends

- **Local filesystem** (default)
- **AWS S3** (`pip install cognee[aws]`)
- Auto-configured cache directory for S3

### 5.5 Rate Limiting

```
LLM_RATE_LIMIT_ENABLED=true
LLM_RATE_LIMIT_REQUESTS=60
LLM_RATE_LIMIT_INTERVAL=60
LLM_RATE_LIMIT_TOKENS=0  # 0 = disabled
EMBEDDING_RATE_LIMIT_ENABLED=true
```

---

## 6. API Layer

### 6.1 REST API (FastAPI)

Server chạy tại `http://localhost:8000`, bao gồm các endpoints:

| Prefix | Tag | Mô tả |
|---|---|---|
| `/api/v1/auth` | auth | Login, register, reset password, verify, API keys |
| `/api/v1/add` | add | Ingest dữ liệu |
| `/api/v1/cognify` | cognify | Xây dựng knowledge graph |
| `/api/v1/memify` | memify | Enrich graph |
| `/api/v1/search` | search | Multi-mode search |
| `/api/v1/remember` | remember | V2 memory store |
| `/api/v1/recall` | recall | V2 memory retrieve |
| `/api/v1/improve` | improve | V2 graph improvement |
| `/api/v1/forget` | forget | V2 memory deletion |
| `/api/v1/datasets` | datasets | Dataset management |
| `/api/v1/permissions` | permissions | Access control |
| `/api/v1/ontologies` | ontologies | Ontology management |
| `/api/v1/settings` | settings | System configuration |
| `/api/v1/users` | users | User management |
| `/api/v1/sync` | sync | Data synchronization |
| `/api/v1/activity` | activity | Observability/audit |
| `/api/v1/responses` | responses | LLM responses |
| `/api/v1/llm` | llm | LLM configuration |
| `/api/v1/update` | update | Data updates |
| `/api/v1/delete` | delete | Data deletion |
| `/api/v1/checks` | checks | Cloud health checks |
| `/api/v1/notebooks` | notebooks | Jupyter integration |
| `/health` | health | Health check |

**Authentication schemes:**
- Bearer token (JWT via fastapi-users)
- API key header (`X-Api-Key`)
- Cookie-based session

**CORS:** Configurable via `CORS_ALLOWED_ORIGINS` env var.

### 6.2 Authentication & Multi-tenancy

**User model:** Users, Roles, Tenants, ACLs, Permissions  
**Permission types:** read, write, delete, admin  
**Isolation:** Per-user dataset scoping, per-tenant knowledge separation  
**API Keys:** Programmatic access without session management

### 6.3 MCP Server (Model Context Protocol)

`cognee-mcp/` — Exposes Cognee as MCP tools cho AI coding assistants:

**Transport modes:**
- `stdio` (default — Claude Desktop, Cline)
- `sse` — Server-Sent Events
- `http` — Streamable HTTP

**MCP Tools exposed:**

| Tool | Mô tả |
|---|---|
| `cognify` | Transform data into knowledge graph |
| `search` | Query knowledge graph |
| `save_interaction` | Log user-agent interactions |
| `list_data` | List datasets and data items |
| `delete_dataset` | Delete a dataset |
| `cognify_status` | Check background task status |

**Deployment:**
```bash
# Direct mode (local cognee)
uv run python src/server.py --transport sse

# API mode (connect to remote Cognee API)
uv run python src/server.py --transport sse --api-url http://localhost:8000 --api-token TOKEN
```

---

## 7. Observability & Monitoring

### 7.1 OpenTelemetry Tracing

```python
from cognee.modules.observability.trace_context import (
    enable_tracing, disable_tracing,
    get_last_trace, get_all_traces
)

enable_tracing()
# ... run operations ...
trace = get_last_trace()
print(trace.summary())
```

**Span attributes tracked:**
- `cognee.db.system`, `cognee.db.query`, `cognee.db.row_count`
- `cognee.llm.model`, `cognee.llm.provider`
- `cognee.search.type`, `cognee.search.query`
- `cognee.pipeline.task_name`, `cognee.pipeline.name`
- `cognee.vector.collection`, `cognee.vector.result_count`
- `cognee.dataset.name`, `cognee.session.id`
- `cognee.operation.mode` (session, permanent, cloud)

**OTLP export:** Tương thích với Dash0, Grafana, Datadog, Jaeger, etc.

### 7.2 Monitoring Integrations

| Tool | Extra |
|---|---|
| Langfuse | `pip install cognee[monitoring]` + `LANGFUSE_PUBLIC_KEY` |
| Sentry | `pip install cognee[monitoring]` + `SENTRY_REPORTING_URL` |
| OpenTelemetry | `pip install cognee[tracing]` |

### 7.3 Structured Logging

- Library: `structlog`
- Log level: Configurable via `LOG_LEVEL` env var
- Secrets redaction: Auto-redact API keys, bearer tokens trong traces
- Log directory: `~/.cognee/logs/`

### 7.4 Activity Tracking

`/api/v1/activity` endpoint cung cấp audit trail cho tất cả operations.

---

## 8. Frontend (cognee-frontend)

**Tech stack:** Next.js, TypeScript, ESLint, Prettier  
**Purpose:** Local UI cho development và demos  

**Khởi chạy:**
```bash
cd cognee-frontend && npm install && npm run dev
```

---

## 9. Deployment Options

### 9.1 Local Development

```bash
# Python library
uv pip install cognee
export LLM_API_KEY="sk-..."
```

### 9.2 Self-hosted (Docker)

```bash
docker compose up
```

**Env vars required:**
```
LLM_API_KEY=sk-...
ENV=prod
DB_PROVIDER=postgresql  # optional
GRAPH_DATABASE_PROVIDER=neo4j  # optional
```

### 9.3 Cloud Deployment

| Platform | Command |
|---|---|
| **Cognee Cloud** | `await cognee.serve(url=..., api_key=...)` |
| **Modal** | `bash distributed/deploy/modal-deploy.sh` |
| **Railway** | `railway init && railway up` |
| **Fly.io** | `bash distributed/deploy/fly-deploy.sh` |
| **Render** | Deploy button |
| **Daytona** | `distributed/deploy/daytona_sandbox.py` |

### 9.4 Distributed Mode (Modal)

`distributed/` — Workers và queues cho distributed processing:
- Modal integration cho serverless, auto-scaling
- Worker pools cho heavy cognify tasks
- Queue-based processing

---

## 10. Integrations & Ecosystem

### 10.1 AI Agent Integrations

| Platform | Setup |
|---|---|
| **Claude Code** | Plugin via hooks (SessionStart, PostToolUse, SessionEnd) |
| **Hermes Agent** | `memory.provider: cognee` trong config |
| **OpenClaw** | `npm install @cognee/cognee-openclaw` |
| **LangChain** | `pip install cognee[langchain]` |
| **LlamaIndex** | `pip install cognee[llama-index]` |

### 10.2 Data Pipeline Integrations

- **dlt** (data load tool): `pip install cognee[dlt]`
- **Graphiti**: `pip install cognee[graphiti]`
- **Tavily** (web scraping): `pip install cognee[scraping]`

### 10.3 Evaluation Framework

`cognee/eval_framework/` + `evals/` — Tools để đánh giá chất lượng retrieval:
- DeepEval: `pip install cognee[deepeval]`
- Plotly, pandas, matplotlib, scikit-learn cho visualization

---

## 11. Configuration Reference

### 11.1 Key Environment Variables

```bash
# LLM
LLM_API_KEY=sk-...
LLM_PROVIDER=openai          # openai, anthropic, ollama, groq, mistral...
LLM_MODEL=openai/gpt-4o-mini
LLM_ENDPOINT=                # Custom endpoint (Ollama, Azure...)
LLM_TEMPERATURE=0.0
LLM_MAX_COMPLETION_TOKENS=16384

# Graph DB
GRAPH_DATABASE_PROVIDER=kuzu  # kuzu, neo4j, neptune, postgres
GRAPH_DATABASE_URL=bolt://localhost:7687
GRAPH_DATABASE_USERNAME=neo4j
GRAPH_DATABASE_PASSWORD=...

# Vector DB
VECTOR_DB_PROVIDER=lancedb  # lancedb, pgvector, chromadb

# Relational DB
DB_PROVIDER=sqlite           # sqlite, postgresql
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=cognee
DB_PASSWORD=...
DB_NAME=cognee_db

# Embeddings
EMBEDDING_PROVIDER=openai
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_DIMENSIONS=1536

# Storage
STORAGE_BACKEND=local        # local, s3
STORAGE_BUCKET_NAME=...

# Auth
REQUIRE_AUTHENTICATION=false
DEFAULT_USER_EMAIL=...
DEFAULT_USER_PASSWORD=...

# Observability
COGNEE_TRACING_ENABLED=false
OTEL_EXPORTER_OTLP_ENDPOINT=...
LANGFUSE_PUBLIC_KEY=...
LANGFUSE_SECRET_KEY=...
SENTRY_REPORTING_URL=...

# Rate Limiting
LLM_RATE_LIMIT_ENABLED=false
LLM_RATE_LIMIT_REQUESTS=60
LLM_RATE_LIMIT_INTERVAL=60

# MCP Server
MCP_DISABLE_DNS_REBINDING_PROTECTION=false
MCP_ALLOWED_HOSTS=192.168.1.50:*
MCP_CORS_ALLOW_ORIGINS=http://localhost:3000
```

### 11.2 Python Config API

```python
cognee.config.set_llm_provider("openai")
cognee.config.set_llm_model("gpt-4o-mini")
cognee.config.set_llm_api_key("sk-...")
```

---

## 12. Security

### 12.1 Authentication

- JWT-based sessions (fastapi-users)
- API key authentication (`X-Api-Key` header)
- Optional: Disable auth với `REQUIRE_AUTHENTICATION=false`
- Password reset và email verification flows

### 12.2 Multi-tenancy & Isolation

- Tenant model: User → UserTenant → Tenant
- Role-based access: User → UserRole → Role
- ACL model: Principal → Permission → Resource
- Dataset-level permission enforcement
- NodeSet scoping cho per-user memory isolation

### 12.3 Secret Redaction

Automatic redaction trong traces/logs:
- OpenAI API keys (`sk-*`)
- Bearer tokens
- `api_key=...` patterns
- `password=...` patterns

### 12.4 MCP Transport Security

- DNS rebinding protection (default on)
- Configurable allowed hosts whitelist
- CORS middleware cho SSE/HTTP transports

---

## 13. Developer Experience

### 13.1 CLI

```bash
cognee-cli remember "Cognee turns documents into AI memory."
cognee-cli recall "What does Cognee do?"
cognee-cli forget --all
cognee-cli -ui   # Launch full UI stack
```

### 13.2 Python SDK (async-first)

```python
import cognee
import asyncio

async def main():
    await cognee.add("data.pdf", dataset_name="docs")
    await cognee.cognify(datasets="docs")
    results = await cognee.search("What is in the document?")

asyncio.run(main())
```

### 13.3 Jupyter Notebooks

- `cognee/modules/notebooks/` — Tutorial notebooks
- `notebooks/` — Demo notebooks
- `pip install cognee[notebook]`

### 13.4 Examples

`examples/` chứa:
- `demos/` — Feature demonstrations
- `guides/` — Step-by-step guides
- `configurations/` — Backend configuration examples
- `database_examples/` — DB-specific examples
- `custom_pipelines/` — Custom task pipeline examples
- `pocs/` — Proof of concept scripts

---

## 14. Non-Functional Requirements

### 14.1 Performance

- **LanceDB** làm default vector DB cho zero-setup local performance
- **Kuzu** làm default graph DB — embedded, no server needed
- **Rate limiting** tích hợp sẵn cho LLM và embedding calls
- **Background cognify** trong MCP server để không block UI
- **LRU cache** cho config objects

### 14.2 Scalability

- Distributed workers via Modal (`distributed/`)
- Queue-based task processing
- PostgreSQL + pgvector cho production scale
- Neo4j/Neptune cho large-scale graph operations

### 14.3 Reliability

- **Fallback LLM**: `FALLBACK_API_KEY`, `FALLBACK_MODEL`, `FALLBACK_ENDPOINT`
- **Tenacity** cho retry logic
- **Alembic** migrations với backward compatibility
- **Sentry** error monitoring

### 14.4 Compatibility

- **Python**: 3.10 – 3.13
- **OS**: macOS, Linux, Windows
- **Database**: SQLite (dev), PostgreSQL (prod)
- **Architecture**: ARM64, x86_64

---

## 15. Roadmap & Current Status

### v1.0.2 (Current — Beta)

✅ Core pipeline: add → cognify → search  
✅ V2 API: remember, recall, improve, forget  
✅ Multi-modal ingestion (PDF, text, audio, image, CSV, web)  
✅ 15+ SearchTypes bao gồm GRAPH_COMPLETION, RAG, TEMPORAL, CYPHER  
✅ NodeSets memory scoping  
✅ DataPoint custom schema  
✅ Multi-tenant RBAC (Users, Roles, Tenants, ACLs)  
✅ MCP server (stdio, SSE, HTTP)  
✅ OpenTelemetry tracing  
✅ LLM rate limiting  
✅ Distributed processing (Modal)  
✅ 6 deployment options  
✅ Claude Code & Hermes Agent plugins  
✅ Feedback loops & session memory  
✅ Ontology grounding (RDF/XML)  
✅ Code graph analysis  

### Planned / Community

- Extended community plugins & add-ons (`cognee-community` repo)
- Evaluation benchmarks (eval_framework)
- Additional graph database connectors
- Enhanced temporal knowledge graph features

---

## 16. Research Foundation

Cognee được hỗ trợ bởi nghiên cứu học thuật:

```bibtex
@misc{markovic2025optimizinginterfaceknowledgegraphs,
  title={Optimizing the Interface Between Knowledge Graphs and LLMs for Complex Reasoning},
  author={Vasilije Markovic and Lazar Obradovic and Laszlo Hajdu and Jovan Pavlovic},
  year={2025},
  eprint={2505.24478},
  archivePrefix={arXiv},
  primaryClass={cs.AI},
  url={https://arxiv.org/abs/2505.24478},
}
```

---

## 17. Links & Resources

| Resource | URL |
|---|---|
| Documentation | https://docs.cognee.ai |
| Homepage | https://www.cognee.ai |
| GitHub | https://github.com/topoteretes/cognee |
| Discord | https://discord.gg/NQPKmU5CCg |
| Reddit | https://www.reddit.com/r/AIMemory/ |
| Community Plugins | https://github.com/topoteretes/cognee-community |
| Demo Video | https://www.youtube.com/watch?v=8hmqS2Y5RVQ |
| Colab Quickstart | https://colab.research.google.com/drive/12Vi9zID-M3fpKpKiaqDBvkk98ElkRPWy |
