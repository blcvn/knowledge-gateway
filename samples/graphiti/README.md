# Graphiti Samples

Sample scripts and integration tests for the **Graphiti** temporal knowledge graph service — part of the VNP Memory platform.

## What is Graphiti?

[Graphiti](https://github.com/getzep/graphiti) (by Zep) is a framework for building temporal knowledge graphs that power AI agent memory. It:

- Extracts **entities** and **relationships** from conversations
- Maintains **temporal facts** (facts that evolve over time)
- Enables **semantic search** across the knowledge graph
- Supports **group-based isolation** for multi-session/multi-user memory

## Structure

```
samples/graphiti/
├── .env                                    # Config (gitignored)
├── .env.example                            # Config template
├── package.json                            # Jest + dependencies
├── lib/
│   └── client.mjs                          # Shared API client & helpers
├── __tests__/
│   ├── 01-health/health.test.mjs           # Health & connectivity
│   ├── 02-ingest/ingest.test.mjs           # Message ingestion & entities
│   ├── 03-search/search.test.mjs           # Search & retrieval
│   ├── 04-memory/memory.test.mjs           # Temporal memory features
│   └── 05-lifecycle/lifecycle.test.mjs     # Data lifecycle management
└── scripts/
    ├── demo-conversation.mjs               # Interactive conversation demo
    ├── demo-entity-graph.mjs               # Entity graph construction demo
    └── demo-search.mjs                     # Advanced search patterns demo
```

## Quick Start

```bash
cd samples/graphiti
cp .env.example .env   # adjust URL if needed
npm install
npm test               # run all tests
```

## Run Individual Test Suites

```bash
npm run test:health     # 01 — Health & connectivity
npm run test:ingest     # 02 — Message ingestion
npm run test:search     # 03 — Search & retrieval (seeds own data, ~30s)
npm run test:memory     # 04 — Temporal memory (seeds own data, ~45s)
npm run test:lifecycle  # 05 — Data lifecycle management
```

## Run Demo Scripts

```bash
npm run demo            # Full conversation workflow
npm run demo:entity     # Entity graph construction
npm run demo:search     # Advanced search patterns
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GRAPHITI_API_URL` | `https://c6.openledger.vn/graphiti` | Graphiti API endpoint |
| `REQUEST_TIMEOUT` | `30` | Default timeout (seconds) |
| `LONG_TIMEOUT` | `120` | Timeout for LLM operations |
| `SAMPLE_GROUP_ID` | `sample-graphiti-demo` | Default group for samples |

## API Coverage

| Category | Endpoints | Tests |
|----------|-----------|-------|
| Health | `GET /healthcheck` | 4 |
| Ingest | `POST /messages`, `POST /entity-node`, `POST /clear` | 6 |
| Search | `POST /search`, `POST /get-memory`, `GET /episodes/:id` | 6 |
| Lifecycle | `DELETE /group/:id`, `DELETE /episode/:id`, `DELETE /entity-edge/:id` | 4 |
| **Total** | **~10 endpoints** | **~20 tests** |

## Architecture

```
┌─────────────────┐     HTTPS      ┌──────────────┐     bolt://     ┌─────────┐
│  Sample Scripts  │  ──────────▶  │   Graphiti    │  ──────────▶   │  Neo4j  │
│  & Jest Tests    │               │  (FastAPI)    │                │  (Graph │
│                  │               │  Port 8001    │                │   DB)   │
└─────────────────┘               └──────┬───────┘                └─────────┘
                                          │
                                          │  OpenAI-compatible
                                          ▼
                                   ┌──────────────┐
                                   │   Bifrost    │
                                   │  AI Gateway  │
                                   └──────────────┘
```

## Key Concepts

### Groups
Messages are organized into **groups** (e.g., conversation sessions). Each group isolates its knowledge graph, allowing multi-tenant or multi-session usage.

### Episodes
Each ingested message becomes an **episode** in the knowledge graph. Episodes are timestamped and linked to entities and relationships.

### Facts
Graphiti extracts **facts** (entity relationships) from episodes. Facts have temporal validity (`valid_at`, `invalid_at`) — enabling the graph to track how information changes over time.

### Temporal Memory
When facts change (e.g., "Alice lives in HCMC" → "Alice moved to Hanoi"), Graphiti maintains both the old and new facts with appropriate timestamps, allowing queries to understand the evolution of knowledge.
