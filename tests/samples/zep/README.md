# Zep Samples

Sample scripts and integration tests for the **Zep** context engineering platform — part of the VNP Memory stack.

## What is Zep?

[Zep](https://github.com/getzep/zep) is a context engineering platform that provides:

- **Agent Memory** — stores and retrieves conversation history with automatic fact extraction
- **Temporal Knowledge Graph** — builds a user-centric knowledge graph from conversations (powered by Graphiti)
- **Context Assembly** — assembles personalized context (facts, summaries, relevant history) for LLM prompts
- **User-Centric Memory** — memory persists across sessions for each user, creating a unified profile

## Structure

```
samples/zep/
├── .env                                     # Config (gitignored)
├── .env.example                             # Config template
├── package.json                             # Jest + dependencies
├── lib/
│   └── client.mjs                           # Shared API client & helpers
├── __tests__/
│   ├── 01-health/health.test.mjs            # Health & connectivity
│   ├── 02-users/users.test.mjs              # User CRUD
│   ├── 03-sessions/sessions.test.mjs        # Session management
│   ├── 04-memory/memory.test.mjs            # Memory pipeline
│   ├── 05-search/search.test.mjs            # Graph search
│   └── 06-lifecycle/lifecycle.test.mjs      # Data lifecycle
└── scripts/
    ├── demo-conversation.mjs                # Full conversation workflow
    ├── demo-memory-retrieval.mjs            # Memory retrieval patterns
    └── demo-graph-search.mjs                # Graph search capabilities
```

## Quick Start

```bash
cd samples/zep
cp .env.example .env   # adjust URL if needed
npm install
npm test               # run all tests
```

## Run Individual Test Suites

```bash
npm run test:health     # 01 — Health & connectivity
npm run test:users      # 02 — User CRUD
npm run test:sessions   # 03 — Session management
npm run test:memory     # 04 — Memory pipeline (~15s processing wait)
npm run test:search     # 05 — Graph search (~25s processing wait)
npm run test:lifecycle  # 06 — Data lifecycle
```

## Run Demo Scripts

```bash
npm run demo            # Full conversation workflow
npm run demo:memory     # Memory retrieval patterns (multi-session)
npm run demo:search     # Graph search capabilities
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ZEP_API_URL` | `https://c6.openledger.vn/zep` | Zep API endpoint |
| `ZEP_API_KEY` | *(empty)* | API key (empty if auth disabled) |
| `REQUEST_TIMEOUT` | `30` | Default timeout (seconds) |
| `LONG_TIMEOUT` | `120` | Timeout for LLM operations |
| `SAMPLE_USER_PREFIX` | `sample-zep` | User ID prefix for samples |
| `SAMPLE_SESSION_PREFIX` | `sample-session` | Session ID prefix for samples |

## API Coverage

| Category | Endpoints | Tests |
|----------|-----------|-------|
| Health | `GET /healthz` | 4 |
| Users | `POST/GET/DEL /api/v2/users` | 5 |
| Sessions | `POST/GET/DEL /api/v2/sessions` | 5 |
| Memory | `POST/GET /api/v2/sessions/{id}/memory`, `GET /messages` | 6 |
| Graph | `POST /api/v2/graph/search` | 4 |
| Lifecycle | Session + User deletion | 4 |
| **Total** | **~15 endpoints** | **~28 tests** |

## Architecture

```
┌─────────────────┐     HTTPS      ┌──────────────┐     bolt://     ┌─────────┐
│  Sample Scripts  │  ──────────▶  │     Zep      │  ──────────▶   │  Neo4j  │
│  & Jest Tests    │               │   (Go)       │                │  (Graph │
│                  │               │  Port 8002   │                │   DB)   │
└─────────────────┘               └──────┬───────┘                └─────────┘
                                          │                        ┌─────────┐
                                          ├──────────────────────▶ │PostgreSQL│
                                          │   pgvector             │(Relational│
                                          │                        │ + Vector) │
                                          │                        └─────────┘
                                          │  OpenAI-compatible
                                          ▼
                                   ┌──────────────┐
                                   │  Graphiti    │ ◄── dedicated instance
                                   │  (for Zep)   │     (port 8003)
                                   │              │
                                   └──────┬───────┘
                                          │
                                          ▼
                                   ┌──────────────┐
                                   │   Bifrost    │
                                   │  AI Gateway  │
                                   └──────────────┘
```

## Key Concepts

### Users
The primary identity entity. All memory is user-centric — facts and knowledge graph nodes persist across all sessions for a given user.

### Sessions
A conversation thread linked to a user. Each session stores chat messages and contributes facts to the user's knowledge graph.

### Memory
The assembled context for an LLM prompt, including:
- **context** — Pre-formatted string combining facts, summaries, and recent messages
- **messages** — Recent conversation messages
- **relevant_facts** — Temporal facts extracted from conversations
- **summary** — High-level summary of the conversation

### Knowledge Graph
Zep builds a temporal knowledge graph per user, extracting entities, relationships, and facts from conversation messages. The graph supports semantic search across all of a user's sessions.

### Facts vs. Summary
- **Facts**: Precise, timestamped information stored as graph edges (e.g., "An works at OpenLedger")
- **Summary**: A condensed overview of the conversation, updated as new messages arrive
