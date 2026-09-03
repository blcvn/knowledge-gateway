# L7 — External Services & Storage Layer

> **Layer**: 7 (External Dependencies)  
> **Responsibility**: External third-party services consumed by L6 adapters  
> **Dependencies**: None (leaf layer)

---

## 1. Tổng Quan

Layer 7 là tầng **external services** — các hệ thống bên ngoài mà Cognee tích hợp qua L6 adapters. Cognee không kiểm soát code của các service này, chỉ tương tác qua APIs/protocols.

---

## 2. LLM Providers

| Provider | Package | Model Examples | Extra |
|----------|---------|---------------|-------|
| **OpenAI** | `openai` (via litellm) | `gpt-4o-mini`, `gpt-4o`, `gpt-4-turbo` | Default |
| **Azure OpenAI** | `openai` | Same as OpenAI, Azure-hosted | — |
| **Anthropic** | `anthropic` | `claude-3-5-sonnet`, `claude-3-opus` | `anthropic` |
| **Google Gemini** | `google-generativeai` | `gemini-2.0-flash-exp` | `gemini` |
| **Ollama** | `ollama` | `llama3.1:8b`, any local model | `ollama` |
| **AWS Bedrock** | `boto3` | `anthropic.claude-3-sonnet` | `aws` |
| **Mistral** | `mistralai` | `mistral-large-latest` | `mistral` |
| **Groq** | `groq` | `llama-3.1-70b-versatile` | `groq` |
| **LM Studio** | OpenAI-compatible | Any local model | — |
| **Custom** | OpenAI-compatible | Any model via `LLM_ENDPOINT` | — |

### 2.1 LLM Middleware

- **LiteLLM** — unified API wrapper cho tất cả providers
- **Instructor** — structured output extraction (default)
- **BAML** — alternative structured output DSL

---

## 3. Graph Databases

| Service | Protocol | Port | Deployment | Extra |
|---------|----------|------|------------|-------|
| **Kuzu** | Embedded (C++ lib) | — | In-process | Default |
| **Kuzu Remote** | HTTP/REST | Custom | Standalone server | — |
| **Neo4j** | Bolt | 7474/7687 | Docker/Cloud | `neo4j` |
| **AWS Neptune** | WebSocket/HTTP | 8182 | AWS managed | `neptune` |
| **PostgreSQL** | TCP (asyncpg) | 5432 | Docker/Cloud | `postgres` |

### 3.1 Docker Compose Profiles

```yaml
# Neo4j
docker compose --profile neo4j up

# Ports: 7474 (HTTP), 7687 (Bolt)
```

---

## 4. Vector Databases

| Service | Protocol | Port | Deployment | Extra |
|---------|----------|------|------------|-------|
| **LanceDB** | Embedded (Rust lib) | — | In-process | Default |
| **ChromaDB** | HTTP/REST | 3002 | Docker/Self-hosted | `chromadb` |
| **PGVector** | TCP (asyncpg) | 5432 | PostgreSQL extension | `postgres` |
| **Qdrant** | HTTP/gRPC | 6333/6334 | Docker/Cloud | — |
| **Weaviate** | HTTP/REST | 8080 | Docker/Cloud | — |
| **Milvus** | gRPC | 19530 | Docker/Cloud | — |

---

## 5. Relational Databases

| Service | Protocol | Port | Deployment |
|---------|----------|------|------------|
| **SQLite** | Embedded (C lib) | — | In-process, default |
| **PostgreSQL** | TCP (asyncpg + SQLAlchemy) | 5432 | Docker/Cloud |

---

## 6. Embedding Services

| Service | API | Model Examples |
|---------|-----|---------------|
| **OpenAI Embeddings** | HTTP/REST | `text-embedding-3-small` |
| **Ollama Embeddings** | HTTP/REST | `nomic-embed-text:latest` |
| **HuggingFace** | Local (transformers) | `sentence-transformers/*` |
| **Cohere** | HTTP/REST | `embed-english-v3.0` |

---

## 7. Storage Services

| Service | Protocol | Deployment |
|---------|----------|------------|
| **Local Filesystem** | OS calls | Default |
| **AWS S3** | HTTP/REST (boto3) | AWS Cloud |

---

## 8. Caching Services

| Service | Protocol | Port | Extra |
|---------|----------|------|-------|
| **Redis** | TCP | 6379 | `redis` |

---

## 9. Observability Services

| Service | Protocol | Chức năng | Extra |
|---------|----------|-----------|-------|
| **Langfuse** | HTTP/REST | LLM observability | `monitoring` |
| **Sentry** | HTTP/REST | Error tracking | `monitoring` |
| **PostHog** | HTTP/REST | Product analytics | `posthog` |

---

## 10. Data Integration Services

| Service | Protocol | Chức năng | Extra |
|---------|----------|-----------|-------|
| **Tavily** | HTTP/REST | Web scraping API | `scraping` |
| **DLT** | Python SDK | Data load tool | `dlt` |
| **Docling** | Python SDK | Document processing | `docling` |
| **Unstructured** | Python SDK | Document parsing | `docs` |

---

## 11. Deployment Platforms

| Platform | Cách triển khai |
|----------|----------------|
| **Docker Compose** | `docker compose up` — full stack local |
| **Modal** | `bash distributed/deploy/modal-deploy.sh` — serverless |
| **Railway** | `railway init && railway up` — PaaS |
| **Fly.io** | `bash distributed/deploy/fly-deploy.sh` — edge |
| **Render** | Deploy button — simple PaaS |
| **Daytona** | `distributed/deploy/daytona_sandbox.py` — cloud sandbox |
| **Cognee Cloud** | `await cognee.serve(url=..., api_key=...)` — managed |

---

## 12. Default Stack (Zero Config)

Khi không cấu hình gì, Cognee sử dụng:

| Component | Default | Storage Location |
|-----------|---------|-----------------|
| Relational DB | SQLite | `.venv/` |
| Vector DB | LanceDB | `.venv/` |
| Graph DB | Kuzu | `.venv/` |
| File Storage | Local FS | `DATA_ROOT_DIRECTORY` |
| LLM | OpenAI `gpt-4o-mini` | API call |
| Embeddings | OpenAI `text-embedding-3-small` | API call |

**Chỉ cần 1 biến**: `LLM_API_KEY` — OpenAI API key.

---

## 13. Docker Compose Services Map

| Service | Profile | Port(s) | Description |
|---------|---------|---------|-------------|
| `cognee` | default | 8000 | FastAPI backend |
| `cognee-mcp` | `mcp` | 8000 | MCP server |
| `frontend` | `ui` | 3000 | Next.js UI |
| `neo4j` | `neo4j` | 7474/7687 | Graph database |
| `chromadb` | `chromadb` | 3002 | Vector database |
| `postgres` | `postgres` | 5432 | PostgreSQL + pgvector |
| `redis` | `redis` | 6379 | Session cache |

---

## 14. Python Package Extras

| Extra | Packages |
|-------|----------|
| `postgres` | asyncpg, psycopg2 |
| `neo4j` | neo4j |
| `chromadb` | chromadb |
| `docs` | unstructured |
| `scraping` | tavily-python, beautifulsoup4, playwright |
| `anthropic` | anthropic |
| `gemini` | google-generativeai |
| `ollama` | ollama |
| `aws` | boto3 |
| `redis` | redis |
| `baml` | baml |
| `dlt` | dlt |
| `monitoring` | sentry-sdk, langfuse |
| `posthog` | posthog |
| `distributed` | modal |
| `dev` | pytest, ruff, ty, etc. |
