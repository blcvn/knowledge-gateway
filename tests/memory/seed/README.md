# VNP Memory — Seed Scripts

Python scripts for seeding test data into a VNP Memory server.

## Structure

```
seed/
├── .env.example          ← Copy to .env and fill in your values
├── requirements.txt      ← pip install -r requirements.txt
├── client.py             ← Shared HTTP client (reads .env)
│
├── 01_generate_data.py   ← Step 1: Generate JSON fixtures (no network)
├── 02_load_data.py       ← Step 2: Push fixtures to server via backend API
├── 03_verify_data.py     ← Step 3: Query server via console API and verify
│
├── run_all.py            ← Run all 3 steps in sequence
│
└── data/                 ← Generated and created ID files (git-ignored)
    ├── admin.json
    ├── cognee.json
    ├── graphiti.json
    ├── memobase.json
    ├── zep.json
    ├── supermemory.json
    ├── agent_memory.json
    ├── observe.json
    ├── openviking.json
    ├── manifest.json
    └── created_*.json    ← IDs returned by server after load
```

## Quick Start

```bash
# 1. Install dependencies
pip install -r requirements.txt

# 2. Configure
cp .env.example .env
# Edit .env — set VNP_BASE_URL and VNP_API_KEY

# 3. Run full pipeline
python run_all.py

# Or run steps individually:
python 01_generate_data.py         # generate fixtures
python 02_load_data.py             # push to server
python 03_verify_data.py           # verify responses
python 03_verify_data.py --verbose # with full JSON output
```

## Configuration (`.env`)

| Variable | Default | Description |
|----------|---------|-------------|
| `VNP_BASE_URL` | `http://localhost:8080` | Gateway base URL |
| `VNP_API_KEY` | — | API key (`vnp_...`). Takes priority over token |
| `VNP_ACCESS_TOKEN` | — | JWT bearer token. Used if no API key |
| `VNP_EMAIL` | `admin@vnp-memory.local` | Login email (used when no key/token) |
| `VNP_PASSWORD` | `changeme` | Login password |
| `VNP_TENANT_ID` | — | Tenant UUID (injected as `X-Tenant-ID`) |
| `SEED_DATA_DIR` | `./data` | Directory for fixture files |
| `SEED_COGNEE_DATASETS` | `2` | Number of Cognee datasets |
| `SEED_GRAPHITI_EPISODES` | `10` | Number of Graphiti episodes |
| `SEED_MEMOBASE_USERS` | `3` | Number of Memobase users |
| `SEED_ZEP_USERS` | `3` | Number of Zep users |
| `SEED_SM_MEMORIES` | `10` | Number of Supermemory memories |
| `SEED_AGENT_MEMORIES` | `10` | Number of agent memories |

## Step Details

### `01_generate_data.py` — Data Generation

Generates realistic fixtures for all engines:
- **Admin**: tenant + API key payload
- **Cognee**: datasets with text data items + cognify config
- **Graphiti**: episodes (10 realistic knowledge facts) + ontology schema
- **Memobase**: users with chat/doc/summary blobs
- **Zep**: users with sessions and conversation messages
- **Supermemory**: adaptive memories + documents
- **Agent Memory**: typed memories (pattern/preference/architecture/bug) + slots
- **Observe**: agent observation sessions with tool calls
- **OpenViking**: markdown/JSON files + resource ingestion + OV session

### `02_load_data.py` — API Load (Backend API)

Pushes fixtures via `specs/backend-api-specs.md` endpoints:

| Engine | Endpoints Called |
|--------|-----------------|
| Admin | `POST /v1/admin/tenants`, `POST /v1/admin/tenants/{id}/keys` |
| Cognee | `POST /v1/cognee/datasets`, `POST /v1/cognee/datasets/{id}/data`, `POST /v1/cognee/datasets/{id}/cognify` |
| Graphiti | `POST /v1/graphiti/episodes`, `POST /v1/graphiti/ontology` |
| Memobase | `POST /v1/memobase/users/{uid}/blobs`, `POST /v1/memobase/users/{uid}/flush` |
| Zep | `POST /v1/zep/users`, `POST /v1/zep/sessions/{id}/memory` |
| Supermemory | `POST /v1/sm/memories`, `POST /v1/sm/documents` |
| Agent Memory | `POST /v1/memory/agent/remember`, `POST /v1/memory/slots/{scope}/{label}` |
| Observe | `POST /v1/observe/sessions`, `POST /v1/observe/sessions/{id}/observe`, `POST /v1/observe/sessions/{id}/end` |
| OpenViking | `PUT /v1/ov/files/{path}`, `POST /v1/ov/resources/ingest`, `POST /v1/ov/sessions` |

Saves returned IDs to `data/created_*.json` for use by the verify step.

### `03_verify_data.py` — API Verification (Frontend Console API)

Queries via `ui/specs/frontend-backend-api-specs.md` console endpoints:

| Domain | Key Checks |
|--------|-----------|
| Auth | `GET /v1/auth/me` → user object shape |
| Dashboard | Health, metrics, throughput, heatmap |
| Memory Explorer | `POST /v1/console/memory/search` → results/total/latencyMs |
| Graph Studio | Ontology, subgraph, Cypher query |
| User Profiles | Profile list, detail, context assembly, events |
| Adaptive Memory | Memory list, connectors, analytics, forget-rules |
| Sessions | Paginated list, detail, timeline, working-memory |
| Governance | Tenants, policies, audit log, GDPR preview |
| Pipelines | Status, queues, workers, per-engine jobs |
| Infrastructure | Topology, services, databases, resources |
| Observability | Metrics, traces, errors, costs |

Exits with code `0` if all checks pass, `1` if any fail.

## Selective Engine Loading

```bash
# Load only Cognee data
python 02_load_data.py --engine cognee

# Verify only Dashboard
python 03_verify_data.py --engine dashboard

# Run all but skip generate (reuse existing fixtures)
python run_all.py --skip-generate
```

## Known Gaps

These endpoints are not yet implemented in the backend (CR-001):
- `POST /v1/auth/login` / `GET /v1/auth/me` / `POST /v1/auth/refresh`
- `GET /v1/console/org/settings` and all `/v1/console/sdk/*`

The verify script handles these gracefully — marking them as expected failures.
