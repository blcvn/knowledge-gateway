# Cognee Integration Tests

Integration test suite for the VNP Memory Cognee API (`v1.0.3`).

## Structure

```
test/cognee/
├── .env                          # Test config (gitignored)
├── package.json                  # Jest + dependencies
├── lib/
│   └── client.mjs                # Shared API client & helpers
└── __tests__/
    ├── 01-health/health.test.mjs # Health & connectivity
    ├── 02-auth/auth.test.mjs     # Auth: register/login/token/API keys
    ├── 03-datasets/datasets.test.mjs  # Dataset CRUD & data upload
    ├── 04-memory/memory.test.mjs # Memory pipeline: remember/cognify/recall/forget
    ├── 05-search/search.test.mjs # Search types & visualization
    ├── 06-settings/settings.test.mjs  # Settings & configuration
    └── 07-infra/infra.test.mjs   # Infrastructure: connections/sessions/activity
```

## Quick Start

```bash
cd test/cognee
cp .env.example .env   # adjust if needed
npm install
npm test               # run all tests
```

## Run Individual Suites

```bash
npm run test:health    # 01 — Health & connectivity
npm run test:auth      # 02 — Authentication
npm run test:datasets  # 03 — Dataset CRUD
npm run test:memory    # 04 — Memory pipeline (LLM-dependent, slow)
npm run test:search    # 05 — Search & retrieval
npm run test:settings  # 06 — Settings & config
npm run test:infra     # 07 — Infrastructure integration
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `COGNEE_API_URL` | `https://c6.openledger.vn/cognee` | Cognee API endpoint |
| `TEST_USER_EMAIL` | `test-cognee@vnpmemory.dev` | Test user email |
| `TEST_USER_PASSWORD` | `TestCognee2026!` | Test user password |
| `TEST_TIMEOUT` | `30` | Default timeout (seconds) |
| `TEST_LONG_TIMEOUT` | `120` | Timeout for LLM operations |
| `SKIP_DESTRUCTIVE` | `false` | Skip delete/forget tests |

## API Coverage

| Category | Endpoints | Tests |
|----------|-----------|-------|
| Health | `/health`, `/` | 5 |
| Auth | `/auth/login`, `/auth/register`, `/auth/me`, `/auth/logout`, `/auth/api-keys` | 10 |
| Datasets | `/datasets`, `/add`, `/datasets/:id/data`, `/datasets/status` | 8 |
| Memory | `/remember`, `/cognify`, `/recall`, `/forget`, `/improve` | 8 |
| Search | `/search`, `/visualize`, `/datasets/:id/graph` | 5 |
| Settings | `/settings`, `/ontologies`, `/configuration/*`, `/llm/custom-prompt` | 4 |
| Infra | `/checks/connection`, `/activity/*`, `/sessions/*`, `/users/me`, `/responses/` | 7 |
| **Total** | **~30 endpoints** | **~47 tests** |
