import os

SERVICES = {
    'sm-engine': {
        'prefix': 'ENG',
        'tasks': [
            ("domain", "Domain Models & Core Algorithms", "Implement Document, Memory, Profile entities. Include Ebbinghaus decay function and core trait structures."),
            ("usecase", "Usecases & Orchestration", "Implement DocumentUseCase, MemoryUseCase, ProfileUseCase. Handle the local orchestration of document -> memory -> profile."),
            ("decay-worker", "Ebbinghaus Decay Worker", "Implement the background worker to recalculate forgetting curves for memories and soft-delete forgotten ones."),
            ("repo", "Repositories (pgvector)", "Implement PostgreSQL repositories for documents, chunks (pgvector), memories, relations, and profiles."),
            ("grpc", "gRPC Handlers & NATS Publisher", "Implement SmDocumentService, SmMemoryService, SmProfileService gRPC handlers and NATS event publisher for sm-search."),
            ("infra", "Infrastructure & Telemetry", "Setup Wire dependency injection, configuration, Zap logging, Prometheus metrics, and Bifrost LLM client.")
        ]
    },
    'sm-search': {
        'prefix': 'SEA',
        'tasks': [
            ("domain", "Domain Models & Filter Logic", "Implement SearchRequest, SearchResult, FilterExpression. Include logic for threshold-based filtering and RRF merge strategy."),
            ("usecase", "Search Usecases", "Implement HybridSearchUseCase (parallel vector + fulltext), MemorySearchUseCase, RAGCompleteUseCase, and QueryRewriteUseCase."),
            ("repo", "PostgreSQL Search Repositories", "Implement ChunkSearchRepository (HNSW) and MemorySearchRepository. Querying replica tables from engine."),
            ("grpc", "gRPC Handlers & NATS Subscriber", "Implement SmSearchService gRPC endpoints. Implement NATS subscriber for real-time indexing of engine events."),
            ("infra", "Infrastructure & Telemetry", "Setup Wire DI, config, Telemetry (OTel tracing for pipeline steps), and Bifrost client for reranking.")
        ]
    },
    'sm-analytics': {
        'prefix': 'ANA',
        'tasks': [
            ("domain", "Domain Models & Token Economics", "Implement ApiRequest, UsageMetrics. Include logic for calculating tokens_saved and cost_saved_usd."),
            ("usecase", "Analytics Aggregation Usecases", "Implement usecases for querying analytics periods (24h, 7d, 30d) and aggregating data."),
            ("repo", "Repositories & Materialized Views", "Implement PostgreSQL repo for api_requests and daily_aggregates materialized views."),
            ("grpc", "gRPC Handlers & NATS Subscriber", "Implement SmAnalyticsService gRPC endpoints. Subscribe to `sm.auth.api_key.used` to track usage."),
            ("infra", "Infrastructure & Telemetry", "Setup Wire DI, configuration, structured logging, and OTel metrics.")
        ]
    },
    'sm-auth': {
        'prefix': 'AUT',
        'tasks': [
            ("domain", "Domain Models & Crypto Algorithms", "Implement AuthContext, APIKey. Implement Base58, SHA-256/Argon2id hashing, and RS256 JWT validation algorithms."),
            ("usecase", "Auth & RBAC Usecases", "Implement API Key lifecycle management and Organization/Waitlist logic."),
            ("repo", "PostgreSQL Repositories", "Implement repos for api_keys, organizations, and org_members."),
            ("grpc", "gRPC Handlers & NATS Publisher", "Implement SmAuthService gRPC endpoints. Publish to `sm.auth.api_key.used`."),
            ("infra", "Infrastructure & Telemetry", "Setup Wire DI, configuration, JWKS fetching for JWT, and Telemetry.")
        ]
    },
    'sm-connector': {
        'prefix': 'CON',
        'tasks': [
            ("domain", "Domain Models & Sync Algorithms", "Implement Connection, ConnectionState. Implement OAuth2 state management and Incremental Sync cursor logic."),
            ("usecase", "OAuth & Sync Usecases", "Implement usecases for managing OAuth connections and orchestrating the incremental sync batches."),
            ("repo", "Repositories", "Implement PostgreSQL repos for connections, states, and sync_history."),
            ("grpc", "gRPC Handlers & NATS Publisher", "Implement SmConnectorService. Publish document batches to `sm.connection.synced`."),
            ("infra", "Infrastructure & External Clients", "Setup Wire DI, config, Telemetry, and external HTTP clients for Google Drive, Notion, OneDrive.")
        ]
    },
    'sm-project': {
        'prefix': 'PRO',
        'tasks': [
            ("domain", "Domain Models & RBAC Algorithm", "Implement Space, SpaceMember. Implement the RBAC Resolution Algorithm based on visibility and roles."),
            ("usecase", "Project Management Usecases", "Implement usecases for Space CRUD, Container Tags, and Member Management."),
            ("repo", "PostgreSQL Repositories", "Implement repos for spaces and spaces_to_members."),
            ("grpc", "gRPC Handlers", "Implement SmProjectService gRPC endpoints."),
            ("infra", "Infrastructure & Telemetry", "Setup Wire DI, configuration, structured logging, and OTel metrics.")
        ]
    }
}

DEPRECATED_SERVICES = {
    'sm-document': 'DOC',
    'sm-memory': 'MEM',
    'sm-profile': 'PRF',
    'sm-mcp': 'MCP'
}

for svc, data in SERVICES.items():
    tasks_dir = f"services/{svc}/specs/tasks"
    os.makedirs(tasks_dir, exist_ok=True)
    prefix = data['prefix']
    
    for idx, (task_id, title, desc) in enumerate(data['tasks']):
        filename = f"{tasks_dir}/TASK-{prefix}-{idx+1:03d}-{task_id}.md"
        content = f"""---
id: TASK-{prefix}-{idx+1:03d}
title: {title}
service: {svc}
status: Todo
priority: P0
created: 2026-05-11
---

# {title}

## Objective
{desc}

## Requirements
- Strictly follow the Clean Architecture definitions from `specs/tdd.md` and `docs/architecture.md`.
- No new features or architectures are to be created; only execute the documented design.
- Token-efficient execution: keep implementations focused entirely on the `{task_id}` layer/component.

## Acceptance Criteria
- [ ] Code compiles without errors.
- [ ] Unit tests written and passing (if applicable).
- [ ] 100% alignment with the `specs/tdd.md` document for `{svc}`.
"""
        with open(filename, 'w') as f:
            f.write(content)
    print(f"Generated tasks for {svc}")

for svc, prefix in DEPRECATED_SERVICES.items():
    tasks_dir = f"services/{svc}/specs/tasks"
    os.makedirs(tasks_dir, exist_ok=True)
    filename = f"{tasks_dir}/TASK-{prefix}-001-decommission.md"
    content = f"""---
id: TASK-{prefix}-001
title: Decommission Service
service: {svc}
status: Todo
priority: P1
created: 2026-05-11
---

# Decommission {svc}

## Objective
This service has been deprecated and merged into another service per ARCH-007 / ARCH-008. This task is for cleaning up any remaining deployment configs.

## Requirements
- Ensure `{svc}` is removed from docker-compose, CI/CD, and routing.
"""
    with open(filename, 'w') as f:
        f.write(content)
    print(f"Generated decommission task for {svc}")

