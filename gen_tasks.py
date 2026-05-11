import os

SERVICES = {
    'sm-document': {
        'prefix': 'DOC',
        'tasks': [
            ("domain", "Domain Models & Core Extraction", "Implement Document, Chunk, and ContentExtraction entities. Ensure format-aware chunking rules are defined."),
            ("usecase", "Document Usecases", "Implement Document CRUD and content extraction orchestrations (CreateDocument, GetChunks)."),
            ("repo", "PostgreSQL Repositories", "Implement repositories for Document and Chunk persistence. Include vector handling for chunks."),
            ("grpc", "gRPC Handlers & NATS", "Implement SmDocumentService gRPC endpoints. Publish events like `sm.document.created` to NATS."),
            ("infra", "Infrastructure & Telemetry", "Setup Wire dependency injection, configuration, structured logging, and Prometheus/OTel metrics.")
        ]
    },
    'sm-memory': {
        'prefix': 'MEM',
        'tasks': [
            ("domain", "Domain Models & Ebbinghaus Decay", "Implement Memory, Relation entities. Include the Ebbinghaus Forgetting Curve algorithms."),
            ("usecase", "Memory & Decay Usecases", "Implement Memory creation, fact extraction logic, and ForgetMemory (decay trigger)."),
            ("repo", "PostgreSQL Repositories", "Implement persistence for Memory and Relation models."),
            ("grpc", "gRPC Handlers & NATS", "Implement SmMemoryService gRPC endpoints. Implement NATS subscribers for document events and publishers for memory events."),
            ("infra", "Infrastructure & Bifrost LLM", "Setup Wire DI, Telemetry, and Bifrost LLM client for knowledge graph extraction.")
        ]
    },
    'sm-profile': {
        'prefix': 'PRF',
        'tasks': [
            ("domain", "Domain Models & Traits", "Implement Profile, StaticPreference, and DynamicTrait entities."),
            ("usecase", "Profile Usecases", "Implement Profile update logic and Dynamic Trait inference based on memory patterns."),
            ("repo", "PostgreSQL Repositories", "Implement repositories for Profiles and Traits."),
            ("grpc", "gRPC Handlers & NATS", "Implement SmProfileService gRPC endpoints. Subscribe to memory events for trait updates."),
            ("infra", "Infrastructure & Caching", "Setup Wire DI, Telemetry, and Redis caching for fast profile retrieval.")
        ]
    },
    'sm-mcp': {
        'prefix': 'MCP',
        'tasks': [
            ("domain", "Domain Models & MCP Schema", "Implement MCP Tool and Resource schemas mapping to Supermemory capabilities (add_memory, search, etc.)."),
            ("usecase", "MCP Orchestration Usecases", "Implement logic to translate MCP requests into downstream calls for document, memory, and search."),
            ("adapter", "SSE & JSON-RPC Transport", "Implement the SSE/JSON-RPC transport layer compliant with the Model Context Protocol."),
            ("grpc", "Downstream gRPC Clients", "Implement gRPC clients connecting to sm-document, sm-memory, and sm-search."),
            ("infra", "Infrastructure & Telemetry", "Setup Wire DI, configuration, and OTel tracing for MCP requests.")
        ]
    }
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
    print(f"Generated implementation tasks for {svc}")
