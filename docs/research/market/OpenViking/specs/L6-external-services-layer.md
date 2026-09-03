# L6 — External Services Layer

> **Layer**: 6 (External)  
> **Responsibility**: Third-party services and runtimes  
> **Dependencies**: None (leaf layer)

---

## 1. Tổng Quan

Layer 6 đại diện cho tất cả external services mà OpenViking tương tác. Tầng này không chứa code của OpenViking — chỉ là các APIs và runtimes bên ngoài.

OpenViking chỉ tương tác với L6 thông qua L5 adapters.

---

## 2. Service Catalog

### 2.1 Embedding Services

| Provider | API | Models | Dense | Sparse |
|----------|-----|--------|-------|--------|
| **OpenAI** | `api.openai.com` | text-embedding-3-small/large | ✅ | ❌ |
| **Volcengine** | `api.volcengine.com` | doubao-embedding | ✅ | ✅ |
| **Gemini** | `generativelanguage.googleapis.com` | text-embedding-004 | ✅ | ❌ |
| **DashScope** | `dashscope.aliyuncs.com` | text-embedding-v3 | ✅ | ✅ |
| **VikingDB** | `api.volcengine.com` | vikingdb-embedding | ✅ | ✅ |
| **Jina** | `api.jina.ai` | jina-embeddings-v3 | ✅ | ❌ |
| **Cohere** | `api.cohere.com` | embed-v3 | ✅ | ❌ |
| **MiniMax** | `api.minimax.chat` | embo-01 | ✅ | ❌ |
| **Voyage** | `api.voyageai.com` | voyage-3 | ✅ | ❌ |
| **LiteLLM** | Proxy | Various | ✅ | ❌ |
| **Local (ONNX)** | Local runtime | Custom ONNX | ✅ | ❌ |

### 2.2 Vision-Language Model Services

| Provider | API | Models |
|----------|-----|--------|
| **OpenAI** | `api.openai.com` | gpt-4o, gpt-4o-mini |
| **Volcengine** | `api.volcengine.com` | doubao-pro, doubao-lite |
| **Gemini** | `generativelanguage.googleapis.com` | gemini-2.0-flash |
| **Kimi** | `api.moonshot.cn` | moonshot-v1 |
| **GLM** | `open.bigmodel.cn` | glm-4v |
| **LiteLLM** | Proxy | Various |

### 2.3 Reranking Services

| Provider | API | Models |
|----------|-----|--------|
| **Volcengine** | `api.volcengine.com` | doubao-reranker |
| **OpenAI** | `api.openai.com` | gpt-based rerank |
| **Cohere** | `api.cohere.com` | rerank-v3 |
| **Jina** | `api.jina.ai` | jina-reranker-v2 |
| **Local** | Local runtime | Custom models |

### 2.4 Key Management Services

| Provider | Protocol | Mô tả |
|----------|----------|--------|
| **Local File** | File I/O | Root key stored in local file |
| **HashiCorp Vault** | HTTP API | KV secrets engine |
| **Volcengine KMS** | HTTPS API | Cloud-managed key service |

### 2.5 Observability Services

| Service | Protocol | Mô tả |
|---------|----------|--------|
| **Prometheus** | HTTP scrape | `/metrics` endpoint |
| **OTLP Backend** | gRPC/HTTP | OpenTelemetry trace export |
| **Grafana** | Dashboard | Visualization (optional) |

### 2.6 Bot Channel Services

| Channel | Protocol | Library |
|---------|----------|---------|
| **Telegram** | Telegram Bot API | python-telegram-bot |
| **Feishu/Lark** | Feishu Open API | — |
| **DingTalk** | DingTalk Open API | — |
| **Slack** | Slack Web API | — |
| **QQ** | QQ Bot API | — |
| **Discord** | Discord Bot API | — |

### 2.7 Code Hosting Services

| Service | Protocol | Mô tả |
|---------|----------|--------|
| **GitHub** | HTTPS/SSH | Git clone for resource ingestion |
| **GitLab** | HTTPS/SSH | Git clone |
| **Bitbucket** | HTTPS/SSH | Git clone |

---

## 3. Default Stack (Zero External Dependencies)

| Component | Default | Config Required |
|-----------|---------|-----------------|
| Storage | RAGFS (embedded Rust) | Workspace path only |
| Vector Index | Embedded | None |
| Embedding | *Must configure* | Provider + API key |
| VLM | *Must configure* | Provider + API key |
| Auth | DEV mode | None |
| Encryption | Disabled | None |
| Bot | Disabled | None |

**Minimal `ov.conf`** cho development:

```json
{
  "storage": {"workspace": "~/.openviking/data"},
  "embedding": {
    "dense": {
      "provider": "openai",
      "model": "text-embedding-3-small",
      "api_key": "sk-..."
    }
  },
  "vlm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "api_key": "sk-..."
  }
}
```

---

## 4. Deployment Runtimes

### 4.1 Standalone

```
Python 3.10+ → openviking-server → :1933
```

### 4.2 Docker

```
Multi-stage build:
  Stage 1: Rust toolchain → compile RAGFS + CLI
  Stage 2: Python → uv pip install
  Stage 3: Runtime → slim image
```

### 4.3 Kubernetes

```
Helm chart at examples/k8s-helm/
  → Deployment (N replicas)
  → Service (:1933)
  → PersistentVolumeClaim (shared storage)
  → ConfigMap (ov.conf)
  → Secret (API keys)
```

---

## 5. Integration Patterns

### 5.1 AI IDE Integration (MCP)

```
Claude Code / OpenCode / Codex
  └── MCP Protocol (Streamable HTTP)
       └── http://localhost:1933/mcp
            └── 9 tools (search, read, list, store, ...)
```

Config: `X-OpenViking-Account` + `X-OpenViking-User` headers.

### 5.2 Application Integration (SDK)

```python
from openviking import AsyncOpenViking

client = AsyncOpenViking(url="http://localhost:1933", api_key="...")
results = await client.find("query", limit=10)
```

### 5.3 Bot Integration (Multi-channel)

```
User → Telegram/Feishu/Slack
  → VikingBot Gateway (:18790)
    → OpenViking API (:1933)
      → Context retrieval + memory persistence
```
