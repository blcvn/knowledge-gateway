#!/usr/bin/env python3
"""
01_generate_data.py — Generate seed data for all VNP Memory engines.

Writes JSON fixtures to ./data/*.json.
No network calls are made — this is pure data generation.

Usage:
    python 01_generate_data.py
"""

from __future__ import annotations

import json
import random
import string
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

from client import cfg, save_json, print_section


# ── Helpers ───────────────────────────────────────────────────────────────────

def uid() -> str:
    return str(uuid.uuid4())


def now_iso(offset_days: int = 0) -> str:
    dt = datetime.now(timezone.utc) + timedelta(days=offset_days)
    return dt.isoformat()


def rand_str(n: int = 8) -> str:
    return "".join(random.choices(string.ascii_lowercase, k=n))


TECH_TOPICS = [
    "Python async patterns", "Go concurrency", "Microservices architecture",
    "PostgreSQL query optimization", "Redis caching strategies", "Kubernetes deployment",
    "GraphQL API design", "Event-driven systems", "Machine learning pipelines",
    "Vector database indexing", "Memory consolidation algorithms", "Knowledge graph construction",
    "Temporal reasoning in AI", "Context window management", "LLM prompt engineering",
]

USERS = [f"user_{i:03d}" for i in range(1, max(cfg.zep_users, cfg.memobase_users) + 1)]
AGENTS = ["claude-code-agent", "codex-agent", "opencode-agent"]
PROJECTS = ["vnp-memory", "osv-dev", "my-saas-app"]


# ── 1. Admin: Tenant & API Key ────────────────────────────────────────────────

def generate_admin() -> dict:
    print_section("1. Admin Data")
    tenant_id = cfg.tenant_id or uid()
    data = {
        "tenant": {
            "id": tenant_id,
            "name": "VNP Memory Test Tenant",
            "slug": "vnp-test",
            "plan": "pro",
            "status": "active",
        },
        "api_key": {
            "name": "seed-test-key",
            "tenant_id": tenant_id,
        },
    }
    path = save_json("admin.json", data)
    print(f"  ✓ Saved {path.name}")
    return data


# ── 2. Cognee Datasets ────────────────────────────────────────────────────────

def generate_cognee() -> dict:
    print_section("2. Cognee Engine Data")
    datasets = []
    for i in range(cfg.cognee_datasets):
        name = f"dataset-{rand_str(6)}"
        datasets.append({
            "name": name,
            "description": f"Test knowledge dataset #{i+1}: {random.choice(TECH_TOPICS)}",
            "data_items": [
                {
                    "content_type": "text",
                    "content": f"Technical notes on {random.choice(TECH_TOPICS)}. "
                               f"Key insight: {random.choice(TECH_TOPICS)} improves performance by "
                               f"{random.randint(20, 80)}% when combined with {random.choice(TECH_TOPICS)}.",
                    "metadata": {"source": "seed-generator", "topic": random.choice(TECH_TOPICS)},
                    "node_sets": [f"tenant:{cfg.tenant_id or 'test'}", f"domain:{rand_str(4)}"],
                }
                for _ in range(3)
            ],
            "cognify_config": {
                "template": "STANDARD",
                "chunk_size": 512,
                "skip_dedup": False,
                "skip_summarize": False,
            },
        })

    data = {"datasets": datasets}
    path = save_json("cognee.json", data)
    print(f"  ✓ {len(datasets)} datasets, {sum(len(d['data_items']) for d in datasets)} data items → {path.name}")
    return data


# ── 3. Graphiti Episodes ──────────────────────────────────────────────────────

def generate_graphiti() -> dict:
    print_section("3. Graphiti Knowledge Graph Data")
    events = [
        ("Alice met Bob at the VNP conference and discussed memory architecture.", "message"),
        ("The team deployed GraphQL API using graphiti-search for temporal queries.", "document"),
        ("Bob is the lead engineer at VNP Memory Ltd., based in Hanoi.", "message"),
        ("Alice completed the knowledge graph ontology for the medical domain.", "document"),
        ("The system processed 10,000 episodes per day in production.", "json"),
        ("VNP Memory integrates with Cognee for semantic knowledge extraction.", "message"),
        ("Neo4j database holds 2.3 million nodes and 8.7 million relationships.", "document"),
        ("The Graphiti temporal model supports bi-temporal fact validation.", "message"),
        ("Episode ingestion pipeline achieves sub-50ms p95 latency.", "document"),
        ("Community detection runs weekly using Louvain algorithm on the graph.", "message"),
    ]

    episodes = []
    for i, (content, source) in enumerate(events[:cfg.graphiti_episodes]):
        episodes.append({
            "name": f"episode-{i+1:03d}",
            "content": content,
            "source": source,
            "source_id": f"seed-{uid()[:8]}",
            "tenant_id": cfg.tenant_id or "test-tenant",
        })

    # Ontology schema
    ontology = {
        "entity_types": ["Person", "Organization", "Technology", "System", "Event", "Metric"],
        "relations": [
            {"name": "KNOWS", "source_type": "Person", "target_type": "Person"},
            {"name": "WORKS_AT", "source_type": "Person", "target_type": "Organization"},
            {"name": "USES", "source_type": "System", "target_type": "Technology"},
            {"name": "DEPLOYED_ON", "source_type": "System", "target_type": "System"},
            {"name": "PARTICIPATED_IN", "source_type": "Person", "target_type": "Event"},
        ],
    }

    data = {"episodes": episodes, "ontology": ontology}
    path = save_json("graphiti.json", data)
    print(f"  ✓ {len(episodes)} episodes + ontology → {path.name}")
    return data


# ── 4. Memobase Users & Blobs ─────────────────────────────────────────────────

def generate_memobase() -> dict:
    print_section("4. Memobase Working Memory Data")
    BLOB_TYPES = ["chat", "doc", "summary"]
    CONVERSATIONS = [
        [
            {"role": "user", "content": "What's the best way to handle async errors in Python?"},
            {"role": "assistant", "content": "Use asyncio.gather with return_exceptions=True for parallel tasks."},
        ],
        [
            {"role": "user", "content": "How should I structure my microservices?"},
            {"role": "assistant", "content": "Follow domain-driven design with bounded contexts."},
        ],
        [
            {"role": "user", "content": "What are the memory types in VNP Memory?"},
            {"role": "assistant", "content": "episodic, semantic, conversational, procedural, profile, adaptive."},
        ],
        [
            {"role": "user", "content": "How do I optimize PostgreSQL for vector search?"},
            {"role": "assistant", "content": "Use pgvector with HNSW index and tune m/ef_construction parameters."},
        ],
        [
            {"role": "user", "content": "What's the difference between Graphiti and Cognee?"},
            {"role": "assistant", "content": "Graphiti is temporal+episodic; Cognee is semantic knowledge graph construction."},
        ],
    ]

    users = []
    for i in range(cfg.memobase_users):
        uid_str = USERS[i % len(USERS)]
        project_id = f"proj-{rand_str(6)}"
        blobs = []
        for j in range(cfg.memobase_blobs_per_user):
            conv = CONVERSATIONS[j % len(CONVERSATIONS)]
            blobs.append({
                "type": BLOB_TYPES[j % len(BLOB_TYPES)],
                "content": json.dumps(conv) if j % 3 != 2 else f"Document summary for {uid_str}: {random.choice(TECH_TOPICS)}",
                "metadata": {"session": f"sess-{rand_str(4)}", "blob_index": j},
            })
        users.append({
            "user_id": uid_str,
            "project_id": project_id,
            "blobs": blobs,
        })

    data = {"users": users}
    path = save_json("memobase.json", data)
    print(f"  ✓ {len(users)} users, {sum(len(u['blobs']) for u in users)} blobs → {path.name}")
    return data


# ── 5. Zep Users, Threads & Messages ─────────────────────────────────────────

def generate_zep() -> dict:
    print_section("5. Zep Conversational Memory Data")
    MSG_PAIRS = [
        ("user", "Can you explain how knowledge graphs work?"),
        ("assistant", "Knowledge graphs store entities and relationships in a graph structure, enabling semantic reasoning."),
        ("user", "How does VNP Memory use knowledge graphs?"),
        ("assistant", "VNP Memory uses Graphiti for temporal episodic graphs and Cognee for semantic knowledge extraction."),
        ("user", "What's the performance target for memory recall?"),
        ("assistant", "Sub-200ms p95 for context assembly. The critical path is in zep-memory ContextAssembly."),
        ("user", "How are facts stored?"),
        ("assistant", "Facts are extracted by zep-graph from conversation context and stored as graph edges with validity timestamps."),
        ("user", "Can you show me an example?"),
        ("assistant", "Sure: Entity 'Alice' WORKS_AT 'VNP Memory Ltd.' with fact 'since 2024-01' valid from 2024-01-01."),
        ("user", "How does the forgetting mechanism work?"),
        ("assistant", "Ebbinghaus decay: strength = initial × e^(-t/half_life × ln(2)). Memories decay unless reinforced."),
    ]

    users = []
    for i in range(cfg.zep_users):
        user_id = USERS[i % len(USERS)]
        sessions = []
        for j in range(cfg.zep_sessions_per_user):
            msgs = []
            for k in range(cfg.zep_messages_per_session):
                role, content = MSG_PAIRS[k % len(MSG_PAIRS)]
                msgs.append({
                    "role": role,
                    "content": content,
                    "metadata": {"turn": k + 1},
                })
            sessions.append({
                "session_id": f"sess-{uid()[:8]}",
                "messages": msgs,
            })
        users.append({
            "user_id": user_id,
            "email": f"{user_id}@test.local",
            "first_name": user_id.replace("_", " ").title().split()[0],
            "last_name": "Tester",
            "metadata": {"source": "seed", "lang": "en"},
            "sessions": sessions,
        })

    data = {"users": users}
    path = save_json("zep.json", data)
    print(f"  ✓ {len(users)} users, {sum(len(u['sessions']) for u in users)} sessions → {path.name}")
    return data


# ── 6. Supermemory (sm) Memories & Documents ──────────────────────────────────

def generate_supermemory() -> dict:
    print_section("6. Supermemory Adaptive Memory Data")
    MEMORY_CONTENTS = [
        "User prefers concise technical answers without introductory fluff.",
        "User is working on a distributed memory platform called VNP Memory.",
        "User's primary language is Vietnamese but codes in English.",
        "User prefers Go for backend services and TypeScript for frontend.",
        "User dislikes verbose boilerplate — prefers clean architecture patterns.",
        "User has a MacOS development environment with Docker Desktop.",
        "The VNP Memory project uses PostgreSQL, Redis, Neo4j, and NATS.",
        "User is familiar with gRPC, REST, and WebSocket protocols.",
        "User follows clean architecture: domain → usecase → adapter layers.",
        "The system targets sub-200ms p95 for all memory recall operations.",
    ]

    DOC_CONTENTS = [
        {
            "title": "VNP Memory Architecture Overview",
            "content": "VNP Memory is a unified memory platform integrating 6 AI memory engines: "
                       "Cognee (semantic KG), Graphiti (temporal episodic), Memobase (working memory), "
                       "OpenViking (encrypted file system), Zep (conversational), and Supermemory (adaptive).",
            "content_type": "article",
        },
        {
            "title": "Memory Engine Comparison",
            "content": "Graphiti excels at temporal reasoning. Cognee at semantic knowledge extraction. "
                       "Memobase at user profiling. OpenViking at secure file-based memory. "
                       "Zep at conversation context. Supermemory at adaptive forgetting.",
            "content_type": "note",
        },
        {
            "title": "Performance Targets",
            "content": "Recall p95 < 200ms. Ingest throughput > 100 items/sec. "
                       "Graph query p99 < 500ms. Context assembly p95 < 200ms.",
            "content_type": "page",
        },
    ]

    memories = [
        {
            "content": MEMORY_CONTENTS[i % len(MEMORY_CONTENTS)],
            "tags": [f"pref-{i}", random.choice(["work", "personal", "technical"])],
            "metadata": {"source": "seed", "index": i},
        }
        for i in range(cfg.sm_memories)
    ]

    documents = DOC_CONTENTS[:cfg.sm_documents]

    data = {"memories": memories, "documents": documents}
    path = save_json("supermemory.json", data)
    print(f"  ✓ {len(memories)} memories + {len(documents)} documents → {path.name}")
    return data


# ── 7. Agent Memories (memory-service) ────────────────────────────────────────

def generate_agent_memories() -> dict:
    print_section("7. Agent Memory Data")
    MEMORY_TYPES = ["pattern", "preference", "architecture", "bug", "workflow", "fact"]
    MEMORY_DATA = [
        ("pattern", "Use retry with exponential backoff for all external API calls",
         ["retry", "resilience", "api"]),
        ("preference", "User prefers minimal code changes — avoid refactoring unless necessary",
         ["user-preference", "code-style"]),
        ("architecture", "All services follow clean architecture: domain → usecase → adapter",
         ["architecture", "design-pattern"]),
        ("bug", "Fixed: nil pointer in ov-crypto Envelope.Decrypt when EFK length is 0",
         ["bug", "crypto", "ov-crypto"]),
        ("workflow", "PR review workflow: lint → unit tests → integration → deploy to staging",
         ["workflow", "ci-cd"]),
        ("fact", "vnp-platform tenant_id is UUID v4, always propagated via X-Tenant-ID header",
         ["fact", "auth", "tenant"]),
        ("pattern", "Always use optimistic locking (version field) for concurrent FSNode updates",
         ["pattern", "concurrency", "ov-fs"]),
        ("preference", "User wants JSON responses with snake_case keys from all APIs",
         ["preference", "api-design"]),
        ("architecture", "Memory consolidation runs as background goroutine every 10 minutes",
         ["architecture", "consolidation"]),
        ("fact", "PutMemory in zep-memory must complete in sub-200ms p95 — critical path",
         ["fact", "performance", "zep"]),
    ]

    agent = random.choice(AGENTS)
    project = random.choice(PROJECTS)
    memories = []
    for i, (mem_type, content, concepts) in enumerate(MEMORY_DATA[:cfg.agent_memories]):
        memories.append({
            "type": mem_type,
            "title": f"[{mem_type.upper()}] {content[:50]}...",
            "content": content,
            "concepts": concepts,
            "strength": round(random.uniform(0.5, 1.0), 2),
            "project": project,
            "agent_id": agent,
            "metadata": {"seed_index": i, "generated_at": now_iso()},
        })

    slots = [
        {
            "scope": "project",
            "label": "architecture-notes",
            "content": "VNP Memory: 6 engines, unified gateway, multi-tenant PostgreSQL.",
            "description": "Current architecture summary",
            "size_limit": 4096,
            "pinned": True,
        },
        {
            "scope": "project",
            "label": "open-tasks",
            "content": "1. Implement CR-001 auth endpoints. 2. Add WebSocket metrics. 3. Fix zep-graph stubs.",
            "description": "Open tasks for current sprint",
            "size_limit": 2048,
            "pinned": False,
        },
    ]

    data = {"memories": memories, "slots": slots, "project": project, "agent": agent}
    path = save_json("agent_memory.json", data)
    print(f"  ✓ {len(memories)} agent memories + {len(slots)} slots → {path.name}")
    return data


# ── 8. Observe Sessions ───────────────────────────────────────────────────────

def generate_observe_sessions() -> dict:
    print_section("8. Observe Session Data")
    sessions = []
    for i in range(cfg.observe_sessions):
        agent_id = random.choice(AGENTS)
        project = random.choice(PROJECTS)
        observations = [
            {
                "hook_type": "tool_use",
                "tool_name": "read_file",
                "tool_input": {"path": f"/src/domain/entity{i}.go"},
                "tool_output": {"content": f"package domain\n\ntype Entity{i} struct {{...}}"},
                "user_prompt": "Read the entity file",
                "assistant_response": f"I've read entity{i}.go — it defines a domain struct.",
                "modality": "text",
            },
            {
                "hook_type": "tool_use",
                "tool_name": "write_file",
                "tool_input": {"path": f"/src/domain/entity{i}_updated.go", "content": "updated content"},
                "tool_output": {"success": True},
                "user_prompt": "Update the entity",
                "assistant_response": f"Updated entity{i} with new fields.",
                "modality": "text",
            },
        ]
        sessions.append({
            "project": project,
            "cwd": f"/Users/dev/{project}",
            "model": "claude-opus-4-5",
            "agent_id": agent_id,
            "observations": observations,
            "tags": [project, agent_id, "seed"],
        })

    data = {"sessions": sessions}
    path = save_json("observe.json", data)
    print(f"  ✓ {len(sessions)} observe sessions → {path.name}")
    return data


# ── 9. OpenViking Files ───────────────────────────────────────────────────────

def generate_openviking() -> dict:
    print_section("9. OpenViking Filesystem Data")
    files = [
        {
            "path": "/docs/architecture.md",
            "content": "# Architecture\n\nVNP Memory uses a unified gateway pattern...\n"
                       f"Generated at: {now_iso()}\n",
            "mime_type": "text/markdown",
        },
        {
            "path": "/docs/api-reference.md",
            "content": "# API Reference\n\n## Memory Store\n\n`POST /v1/memory/store`\n\n"
                       "Store memory with auto-routing by type.\n",
            "mime_type": "text/markdown",
        },
        {
            "path": "/src/config.json",
            "content": json.dumps({
                "engines": ["cognee", "graphiti", "memobase", "openviking", "zep", "supermemory"],
                "default_engine": "graphiti",
                "recall_timeout_ms": 200,
            }, indent=2),
            "mime_type": "application/json",
        },
    ]

    resource = {
        "uri": "http://example.com/vnp-memory-overview.html",
        "type": "web",
        "options": {"chunk_size": 512, "overlap": 64},
    }

    session_data = {
        "project": "vnp-memory",
        "cwd": "/Users/dev/vnp-memory",
        "model": "claude-opus-4-5",
        "agent_id": "claude-code-agent",
    }

    data = {"files": files, "resource": resource, "session": session_data}
    path = save_json("openviking.json", data)
    print(f"  ✓ {len(files)} files + 1 resource + 1 session → {path.name}")
    return data


# ── Main ─────────────────────────────────────────────────────────────────────

def main() -> None:
    print(f"\n{'━' * 60}")
    print(" VNP Memory — Seed Data Generator")
    print(f" Output dir: {cfg.data_dir}")
    print(f"{'━' * 60}")

    cfg.data_dir.mkdir(parents=True, exist_ok=True)

    results = {}
    results["admin"] = generate_admin()
    results["cognee"] = generate_cognee()
    results["graphiti"] = generate_graphiti()
    results["memobase"] = generate_memobase()
    results["zep"] = generate_zep()
    results["supermemory"] = generate_supermemory()
    results["agent_memory"] = generate_agent_memories()
    results["observe"] = generate_observe_sessions()
    results["openviking"] = generate_openviking()

    # Save manifest
    manifest = {
        "generated_at": now_iso(),
        "counts": {
            "admin_tenants": 1,
            "cognee_datasets": len(results["cognee"]["datasets"]),
            "graphiti_episodes": len(results["graphiti"]["episodes"]),
            "memobase_users": len(results["memobase"]["users"]),
            "zep_users": len(results["zep"]["users"]),
            "sm_memories": len(results["supermemory"]["memories"]),
            "agent_memories": len(results["agent_memory"]["memories"]),
            "observe_sessions": len(results["observe"]["sessions"]),
            "ov_files": len(results["openviking"]["files"]),
        },
    }
    save_json("manifest.json", manifest)

    print(f"\n{'━' * 60}")
    print(" ✅ Data generation complete!")
    print(f" Files in: {cfg.data_dir}")
    for f in sorted(cfg.data_dir.glob("*.json")):
        size_kb = f.stat().st_size / 1024
        print(f"   {f.name:<30} {size_kb:.1f} KB")
    print(f"{'━' * 60}\n")
    print("Next step: python 02_load_data.py")


if __name__ == "__main__":
    main()
