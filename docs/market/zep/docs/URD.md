# Zep — User Requirements Document (URD)

> **Product**: Zep — End-to-End Context Engineering Platform  
> **Version**: 1.0  
> **Generated**: 2026-05-07  
> **Status**: Active

---

## 1. Document Purpose

This User Requirements Document defines the interaction models, user personas, standard operating procedures (SOPs), and use-case scenarios for the Zep context engineering platform. It serves as the bridge between business requirements (PRD) and technical specifications (SRS).

---

## 2. User Personas

### 2.1 Persona 1: AI Application Developer (Primary)

| Attribute | Description |
|---|---|
| **Role** | Full-stack or backend developer building AI-powered applications |
| **Technical Level** | Intermediate to Advanced |
| **Primary Languages** | Python, TypeScript, Go |
| **Goals** | Add persistent, intelligent memory to AI agents with minimal integration effort |
| **Pain Points** | Managing conversation state, assembling relevant context, handling temporal data changes |
| **Interaction Model** | SDK-first — uses `zep-cloud` package to interact with Zep API programmatically |

**Typical Workflow:**
```
1. Install SDK (pip/npm/go get)
2. Initialize Zep client with API key
3. Create users and threads
4. Add messages during conversations
5. Retrieve context for LLM prompt assembly
6. Search graph for specific facts
```

### 2.2 Persona 2: ML/AI Engineer

| Attribute | Description |
|---|---|
| **Role** | Designs and optimizes AI agent pipelines |
| **Technical Level** | Advanced |
| **Primary Languages** | Python |
| **Goals** | Maximize context retrieval accuracy, optimize ontology for domain-specific use cases |
| **Pain Points** | Generic ontologies miss domain entities, difficulty measuring retrieval quality |
| **Interaction Model** | Eval harness + custom ontology configuration |

**Typical Workflow:**
```
1. Define custom ontology (entity types + edge types)
2. Configure custom extraction instructions
3. Ingest test data via eval harness
4. Run evaluation pipeline
5. Analyze per-category/per-user metrics
6. Iterate on ontology and search parameters
```

### 2.3 Persona 3: Platform/DevOps Engineer

| Attribute | Description |
|---|---|
| **Role** | Deploys and operates AI infrastructure |
| **Technical Level** | Advanced (infrastructure focus) |
| **Goals** | Reliable deployment, monitoring, and scaling of Zep services |
| **Pain Points** | Managing multi-service dependencies, monitoring graph processing latency |
| **Interaction Model** | Docker Compose, environment configuration, health check monitoring |

**Typical Workflow:**
```
1. Configure environment variables (API keys, connection strings)
2. Deploy via Docker Compose or Cloud dashboard
3. Monitor health endpoints (/healthz)
4. Review structured logs and OpenTelemetry traces
5. Scale services based on load patterns
```

### 2.4 Persona 4: AI Framework Integrator

| Attribute | Description |
|---|---|
| **Role** | Builds memory integrations for AI frameworks (AutoGen, CrewAI, etc.) |
| **Technical Level** | Advanced |
| **Primary Languages** | Python |
| **Goals** | Implement framework-native memory interfaces backed by Zep |
| **Pain Points** | Interface compliance, async lifecycle management, type safety across frameworks |
| **Interaction Model** | Integration SDK development following standardized patterns |

**Typical Workflow:**
```
1. Create integration package (integrations/python/{framework}/)
2. Implement framework's memory interface (Memory, Storage, etc.)
3. Route data: messages → Thread, data/facts → Graph
4. Test with mock clients + integration tests
5. Publish to PyPI with zep-{framework} naming
```

---

## 3. User Stories

### 3.1 Context Ingestion Stories

| ID | Story | Priority | Acceptance Criteria |
|---|---|---|---|
| **US-01** | As a developer, I want to add chat messages to a session so that the conversation history is persisted | P0 | Messages stored in PostgreSQL; graph extraction triggered asynchronously |
| **US-02** | As a developer, I want to add business data (JSON/text) to a user's graph so that agents can access it | P0 | Data ingested via `graph.add()` with `type="json"` or `type="text"` |
| **US-03** | As a developer, I want to create and manage users with metadata so that context is personalized | P0 | User CRUD with JSONB metadata support |
| **US-04** | As a developer, I want to create conversation threads linked to users so that context is session-aware | P0 | Session lifecycle with user association via `user_id` |
| **US-05** | As an ML engineer, I want to define custom ontologies so that graph extraction is domain-specific | P1 | Custom Pydantic models applied via `graph.set_ontology()` |

### 3.2 Context Retrieval Stories

| ID | Story | Priority | Acceptance Criteria |
|---|---|---|---|
| **US-10** | As a developer, I want to retrieve memory (messages + facts) for a session so that I can build LLM prompts | P0 | Combined response with last-N messages and relevant Graphiti facts |
| **US-11** | As a developer, I want to search the knowledge graph semantically so that I can find specific facts | P0 | Scoped search (edges/nodes/episodes) with reranking options |
| **US-12** | As a developer, I want to get pre-formatted context blocks so that I don't need to format them myself | P0 | `get_user_context()` returns formatted context string |
| **US-13** | As a developer, I want to filter search results by node labels and edge types so that I get precise results | P1 | `SearchFilters` with `node_labels` and `edge_types` arrays |
| **US-14** | As an AI assistant, I want to access Zep's graph via MCP tools so that I can provide informed responses | P1 | 13 read-only MCP tools available via stdio/HTTP transport |

### 3.3 Integration Stories

| ID | Story | Priority | Acceptance Criteria |
|---|---|---|---|
| **US-20** | As an AutoGen developer, I want Zep memory to implement `autogen_core.memory.Memory` so that it's a drop-in replacement | P1 | Interface compliance verified by mypy; async operations with CancellationToken |
| **US-21** | As a CrewAI developer, I want separate user storage and graph storage backends so that I can manage personal and shared knowledge | P1 | `ZepUserStorage` for per-user memory; `ZepGraphStorage` for shared KGs |
| **US-22** | As a framework integrator, I want standardized tool factories so that search and data addition are consistent | P1 | `create_search_tool()` and `create_add_data_tool()` factories |

### 3.4 Evaluation Stories

| ID | Story | Priority | Acceptance Criteria |
|---|---|---|---|
| **US-30** | As an ML engineer, I want to run end-to-end evaluations so that I can measure context retrieval quality | P2 | Context completeness (COMPLETE/PARTIAL/INSUFFICIENT) and answer accuracy metrics |
| **US-31** | As an ML engineer, I want to compare different ontology/instruction configurations so that I can find the optimal setup | P2 | Decoupled ingestion + combinatorial evaluation via `--user-run N --doc-run M` |
| **US-32** | As an ML engineer, I want reproducible experiment runs so that I can share and compare results | P2 | Config snapshotting in each run directory |

---

## 4. Standard Operating Procedures (SOPs)

### 4.1 SOP-01: Basic Agent Memory Setup

**Objective**: Integrate Zep memory into a new AI agent application.

**Prerequisites**:
- Zep Cloud API key
- Python 3.10+ / Node.js 18+ / Go 1.21+

**Procedure**:

```python
# Step 1: Install SDK
# pip install zep-cloud

# Step 2: Initialize client
from zep_cloud.client import AsyncZep
import os

zep = AsyncZep(api_key=os.getenv("ZEP_API_KEY"))

# Step 3: Create user
await zep.user.add(user_id="alice", first_name="Alice", last_name="Smith")

# Step 4: Create conversation thread
await zep.thread.create(user_id="alice", thread_id="thread_001")

# Step 5: Add messages during conversation
from zep_cloud.types import Message
await zep.thread.add_messages(
    thread_id="thread_001",
    messages=[
        Message(role="user", content="I prefer dark roast coffee"),
        Message(role="assistant", content="Noted! I'll remember your preference for dark roast coffee."),
    ]
)

# Step 6: Retrieve context for next turn
context = await zep.thread.get_user_context(thread_id="thread_001")

# Step 7: Search for specific facts
results = await zep.graph.search(user_id="alice", query="coffee preferences")
```

### 4.2 SOP-02: Custom Ontology Configuration

**Objective**: Define domain-specific entity and edge types for improved graph extraction.

**Procedure**:

```python
# Step 1: Define custom entity types using Pydantic
from pydantic import BaseModel, Field

class Product(BaseModel):
    """Represents a product or service being discussed."""
    name: str = Field(..., description="Product name")
    category: str = Field(..., description="Product category")

class Customer(BaseModel):
    """Represents a customer or client."""
    customer_id: str = Field(..., description="Customer identifier")

# Step 2: Define custom edge types
class Purchased(BaseModel):
    """Represents a purchase relationship between customer and product."""
    ...

# Step 3: Compose ontology
custom_nodes = {"Product": Product, "Customer": Customer}
custom_edges = {"PURCHASED": Purchased}

# Step 4: Apply ontology via API
await zep.graph.set_ontology(
    graph_id="my_knowledge_graph",
    entities=custom_nodes,
    edges=custom_edges
)
```

### 4.3 SOP-03: Framework Integration Development

**Objective**: Build a new Zep integration for an AI framework.

**Procedure**:

1. **Create package structure**:
   ```
   integrations/python/{framework}/
   ├── src/zep_{framework}/
   │   ├── __init__.py
   │   ├── memory.py
   │   └── exceptions.py
   ├── tests/
   ├── examples/
   ├── pyproject.toml
   └── Makefile
   ```

2. **Implement framework's memory interface** in `memory.py`:
   - Route `type="message"` metadata to `thread.add_messages()`
   - Route `type="text"` or `type="json"` metadata to `graph.add()`
   - Retrieve context via `thread.get_user_context()` and `graph.search()`

3. **Handle async lifecycle**:
   - Don't close externally-provided clients
   - Support `CancellationToken` where applicable
   - Guard against `None` for optional `thread_id`

4. **Quality gates** (all must pass):
   - `make lint` — ruff linting
   - `make type-check` — mypy validation
   - `make test` — pytest with >90% coverage
   - `make ci` — full CI validation

5. **Release**: Tag as `zep-{framework}-v{version}`, GitHub Actions handles PyPI publishing

### 4.4 SOP-04: Running Evaluation Pipeline

**Objective**: Measure and optimize context retrieval quality using the eval harness.

**Procedure**:

```bash
# Step 1: Prepare data
# Place files in data/users.json, data/conversations/, data/test_cases/

# Step 2: Ingest users (with custom config if desired)
uv run zep_ingest_users.py --custom-ontology --custom-instructions

# Step 3: Chunk documents (if using document graphs)
uv run zep_chunk_documents.py --chunk-size 500 --concurrency 10

# Step 4: Ingest documents
uv run zep_ingest_documents.py --chunk-set 1 --custom-ontology

# Step 5: Run evaluation
uv run zep_evaluate.py --user-run 1 --doc-run 1 --concurrency 30

# Step 6: Analyze results
# Check runs/evaluations/{N}/results.json for:
# - aggregate_scores.completeness.complete_rate (PRIMARY)
# - aggregate_scores.accuracy.accuracy_rate (SECONDARY)
# - category_scores for per-category breakdown
# - user_scores for per-user breakdown
```

### 4.5 SOP-05: MCP Server Deployment

**Objective**: Deploy the Zep MCP Server for AI assistant access.

**Procedure**:

**Option A — Claude Desktop (stdio mode)**:
```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "zep": {
      "command": "/path/to/zep-mcp-server",
      "args": ["--stdio"],
      "env": { "ZEP_API_KEY": "your-api-key" }
    }
  }
}
```

**Option B — Claude Code (HTTP mode)**:
```bash
# Start server
export ZEP_API_KEY=your-api-key
./zep-mcp-server --port 8080

# Register with Claude Code
claude mcp add --transport http zep http://localhost:8080
```

**Option C — Docker deployment**:
```bash
docker run -e ZEP_API_KEY=your-key -p 8080:8080 zep-mcp-server:latest
```

---

## 5. Interaction Models

### 5.1 SDK Interaction (Programmatic)

```
Developer Code  →  Zep SDK  →  Zep Cloud API  →  Graphiti + PostgreSQL
                                                        ↓
                                                    Knowledge Graph
                                                    (Neo4j)
```

**Key Patterns**:
- All SDKs support async operations
- Thread-based message storage with session management
- Graph-based fact/entity storage with ontology support
- Unified search across threads and graphs

### 5.2 MCP Interaction (AI Assistants)

```
AI Assistant  →  MCP Client  →  MCP Server  →  Zep Cloud API
(Claude)         (stdio/HTTP)    (Go service)    (Read-only)
```

**Available Operations** (13 tools, all read-only):

| Phase | Tools | Description |
|---|---|---|
| **Core Search** | `search_graph`, `get_user_context`, `get_user`, `list_threads` | Primary context retrieval |
| **Graph Query** | `get_user_nodes`, `get_user_edges`, `get_episodes` | Graph exploration |
| **Detail Retrieval** | `get_thread_messages`, `get_node`, `get_edge`, `get_episode`, `get_node_edges`, `get_episode_mentions` | Specific entity lookup |

### 5.3 Framework Integration Interaction

```
Framework Agent  →  Framework Memory Interface  →  Zep Integration  →  Zep SDK  →  Zep API
(AutoGen/CrewAI)    (Memory/Storage class)          (zep_{framework})
```

**Storage Routing**:

| Data Type | Metadata `type` | Zep Destination | API Call |
|---|---|---|---|
| Chat messages | `"message"` | Thread | `thread.add_messages()` |
| Text data | `"text"` | User Graph | `graph.add(type="text")` |
| JSON data | `"json"` | User Graph | `graph.add(type="json")` |

### 5.4 Eval Harness Interaction

```
Config Files  →  Ingestion Scripts  →  Zep API  →  Evaluation Script  →  Results
(ontology,       (users, documents)               (search + LLM judge)    (JSON)
 instructions)
```

**Pipeline Steps**:
1. **Ingest** — Create users, add conversations, add telemetry/documents
2. **Search** — Query knowledge graph using configured parameters
3. **Evaluate Context** — LLM judges context completeness
4. **Generate Response** — LLM answers using retrieved context
5. **Grade Answer** — LLM judges answer against golden answer

---

## 6. Data Flow Diagrams

### 6.1 Message Ingestion Flow

```
Client sends messages
        │
        ▼
  Upsert Session (create if new)
        │
        ▼
  Check session.EndedAt
  (reject if session ended)
        │
        ▼
  INSERT messages into PostgreSQL
        │
        ▼
  Publish to Graphiti service
  (async graph extraction)
        │
        ▼
  Graphiti extracts entities/edges
  and updates Neo4j knowledge graph
```

### 6.2 Memory Retrieval Flow

```
Client requests memory (sessionID, lastN)
        │
        ▼
  Fetch last N messages from PostgreSQL
        │
        ▼
  Call Graphiti.GetMemory(groupID)
  with last 4 messages for context
        │
        ▼
  Graphiti returns relevant Facts
  (with valid_at/invalid_at metadata)
        │
        ▼
  Assemble Memory response:
  { messages: [...], relevant_facts: [...] }
```

---

## 7. Error Handling & Edge Cases

### 7.1 User-Facing Error Scenarios

| Scenario | Expected Behavior | Error Response |
|---|---|---|
| Invalid API key | 401 Unauthorized | Clear error message directing to API key setup |
| Session already ended | 400 Bad Request | `SessionEndedError` — cannot add messages |
| User/Thread not found | 404 Not Found | Descriptive error with creation guidance |
| Request too large (>5MB) | 413 Payload Too Large | Middleware-level rejection |
| Server timeout (>30s) | 504 Gateway Timeout | Context cancellation with cleanup |
| Concurrent metadata update | Retry with backoff | PostgreSQL advisory lock with exponential backoff (200ms→30s) |

### 7.2 Async Processing Caveats

> [!IMPORTANT]
> Graph data ingestion via Graphiti is **asynchronous** — allow 10-20 seconds after `graph.add()` or message ingestion before search results reflect the new data.

---

## 8. Acceptance Criteria Summary

| Category | Criteria | Metric |
|---|---|---|
| **Latency** | Context retrieval completes within SLA | < 200ms (p95) |
| **Completeness** | Retrieved context contains relevant facts | > 90% completeness rate |
| **Accuracy** | Agent answers match golden answers | > 80% accuracy rate |
| **Integration** | Framework memory interfaces are fully compliant | mypy + tests pass |
| **Safety** | MCP server exposes only read operations | 0 write tools |
| **Reproducibility** | Eval runs can be reproduced from config snapshots | Config snapshot in every run |

---

*Document generated from source analysis of `github.com/getzep/zep` repository.*
