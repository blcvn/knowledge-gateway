---
id: DOC-S05
service: memobase-engine
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-engine — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | 9032 | Yes | gRPC server port |
| `HEALTH_PORT` | int | 9099 | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `BIFROST_URL` | string | — | Yes | Bifrost LLM gateway endpoint |
| `BEST_LLM_MODEL` | string | `gpt-4o-mini` | No | Primary LLM model for extract/merge/summary |
| `THINKING_LLM_MODEL` | string | `o4-mini` | No | Reasoning-heavy LLM model |
| `SUMMARY_LLM_MODEL` | string | — | No | Summary/filter model (defaults to BEST) |
| `LLM_MAX_TOKENS` | int | 1024 | No | Max output tokens per LLM call |
| `EMBEDDING_PROVIDER` | string | `openai` | No | Embedding provider (openai/jina/ollama/lmstudio) |
| `EMBEDDING_MODEL` | string | `text-embedding-3-large` | No | Embedding model name |
| `EMBEDDING_DIM` | int | 1536 | No | Embedding vector dimension |
| `MAX_PROCESS_TOKEN_SIZE` | int | 16384 | No | Max input tokens for pipeline processing |
| `MAX_PROFILE_SUBTOPICS` | int | 10 | No | Max subtopics per topic before organize |
| `MAX_PRE_PROFILE_TOKEN_SIZE` | int | 256 | No | Max tokens per profile before re_summary |
| `PROMPT_LANGUAGE` | string | `en` | No | Default prompt language (en/zh) |

## Example .env

```env
GRPC_PORT=9032
HEALTH_PORT=9099
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://user:pass@postgresql:5432/memobase?sslmode=disable
BIFROST_URL=http://bifrost:8443
BEST_LLM_MODEL=gpt-4o-mini
EMBEDDING_PROVIDER=openai
EMBEDDING_DIM=1536
PROMPT_LANGUAGE=en
```
