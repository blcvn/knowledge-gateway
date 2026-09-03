"""
01_generate_data.py — Sinh dữ liệu test và lưu vào thư mục data/

Tạo ra:
  data/sessions.json        — danh sách session objects
  data/observations.jsonl   — observations (JSONL, 1 dòng/obs)
  data/memories.json        — memories
  data/search_queries.json  — query mẫu để dùng trong test search

Chạy:
  cd tests/agentmemory
  python 01_generate_data.py
"""
from __future__ import annotations

import json
import random
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

import config  # noqa: F401 — nạp .env

# ── Template data ──────────────────────────────────────────────────────────────
_PROJECTS = [config.TEST_PROJECT, "auth-service", "api-gateway", "frontend-app"]
_CWD_ROOTS = ["/home/dev/projects", "/Users/dev/work", config.TEST_CWD]
_HOOK_TYPES = ["post_tool_use", "pre_tool_use", "stop", "notification"]
_TOOL_NAMES = [
    "edit_file", "read_file", "bash", "search_files",
    "list_directory", "write_to_file", "run_command",
]
_MEMORY_TYPES = ["pattern", "preference", "architecture", "bug", "workflow", "fact"]
_CONCEPTS: list[str] = [
    "jwt", "authentication", "authorization", "middleware", "database",
    "api", "typescript", "nodejs", "performance", "security",
    "refactor", "testing", "deployment", "docker", "redis",
    "postgres", "graphql", "rest", "websocket", "caching",
]
_FILES = [
    "src/auth.ts", "src/middleware/jwt.ts", "src/config.ts",
    "src/services/user.ts", "src/api/routes.ts", "tests/auth.test.ts",
    "src/database/schema.ts", "src/utils/logger.ts", "package.json",
]
_OBSERVATION_TEMPLATES = [
    "Implemented {concept} module with {tool} — modified {file}",
    "Fixed bug in {concept} handler — root cause was missing validation",
    "Refactored {file} to use {concept} pattern for better performance",
    "Added {concept} middleware to protect API endpoints",
    "Reviewed {file}: found {concept} issue, applying fix",
    "Created unit tests for {concept} functionality in {file}",
    "Deployed {concept} changes to staging environment via {tool}",
    "Optimized {concept} query reducing latency by ~40%",
    "Added {concept} configuration to {file}",
    "Debugged {concept} error — stack overflow in recursive call",
]
_MEMORY_TEMPLATES = [
    "The {concept} system uses {tech} for {purpose} to ensure {benefit}",
    "Best practice: always validate {concept} before processing to prevent {issue}",
    "Architecture decision: {concept} is implemented as a singleton to {benefit}",
    "Known bug: {concept} fails when {condition} — workaround is {fix}",
    "Workflow: when changing {concept}, always update {file} and run tests",
    "Pattern: use {concept} interceptor pattern for cross-cutting concerns",
    "Performance note: {concept} should be cached with TTL=300s to reduce DB load",
    "Security: {concept} tokens must be validated with timing-safe comparison",
]
_TECH_WORDS = ["JWT", "Redis", "PostgreSQL", "GraphQL", "REST", "WebSocket", "Docker"]
_PURPOSES = ["session management", "request routing", "data caching", "auth flow"]
_BENEFITS = ["security", "scalability", "maintainability", "performance"]
_ISSUES = ["injection attacks", "race conditions", "memory leaks", "auth bypass"]

_SEARCH_QUERIES = [
    "authentication jwt",
    "middleware authorization",
    "database performance",
    "api error handling",
    "typescript refactor",
    "docker deployment",
    "security vulnerability",
    "caching strategy redis",
    "testing best practices",
    "websocket connection",
]


# ── Helpers ────────────────────────────────────────────────────────────────────
def _ts(offset_minutes: int = 0) -> str:
    """ISO timestamp với optional offset (minutes)."""
    t = datetime.now(timezone.utc) - timedelta(days=3) + timedelta(minutes=offset_minutes)
    return t.isoformat().replace("+00:00", "Z")


def _pick(*lst: list) -> object:
    return random.choice(lst[0])


def _session_id() -> str:
    return f"sess_{uuid.uuid4().hex[:12]}"


def _obs_id() -> str:
    return f"obs_{uuid.uuid4().hex[:16]}"


def _mem_id() -> str:
    return f"mem_{int(datetime.now(timezone.utc).timestamp() * 1000)}_{uuid.uuid4().hex[:8]}"


# ── Generators ─────────────────────────────────────────────────────────────────
def generate_sessions(count: int) -> list[dict]:
    sessions = []
    for i in range(count):
        sid = _session_id()
        project = _pick(_PROJECTS)
        start_offset = i * 60 * 24  # mỗi session cách 1 ngày
        sessions.append({
            "id": sid,
            "project": project,
            "cwd": f"{_pick(_CWD_ROOTS)}/{project}",
            "startedAt": _ts(-start_offset),
            "endedAt": _ts(-start_offset + random.randint(30, 480)),
            "status": "completed" if i < count - 1 else "active",
            "observationCount": config.GEN_OBS_PER_SESSION,
            "agentId": config.AGENT_ID,
            "summary": f"Session {i+1}: working on {project} features",
        })
    return sessions


def generate_observations(sessions: list[dict]) -> list[dict]:
    observations = []
    for session in sessions:
        sid = session["id"]
        project = session["project"]
        cwd = session["cwd"]

        for j in range(config.GEN_OBS_PER_SESSION):
            concept = _pick(_CONCEPTS)
            tool = _pick(_TOOL_NAMES)
            file_ = _pick(_FILES)
            template = _pick(_OBSERVATION_TEMPLATES)
            title = template.format(concept=concept, tool=tool, file=file_)

            obs_concepts = [concept] + random.sample(_CONCEPTS, k=random.randint(0, 3))
            obs_files = [file_] if random.random() > 0.3 else []

            # Tạo dữ liệu phù hợp với hook payload
            hook_type = "post_tool_use" if j % 5 != 4 else _pick(["stop", "notification"])
            data: dict = {}
            if hook_type == "post_tool_use":
                data = {
                    "tool_name": tool,
                    "tool_input": {"path": file_, "operation": "write"},
                    "tool_output": f"Successfully {tool.replace('_', ' ')} on {file_}",
                }

            obs = {
                "id": _obs_id(),
                "sessionId": sid,
                "project": project,
                "cwd": cwd,
                "hookType": hook_type,
                "timestamp": _ts(-session.get("_start_offset", 0) + j * 3),
                "agentId": config.AGENT_ID,
                "type": "file_edit" if "file" in tool else "bash_run",
                "title": title[:200],
                "narrative": f"[{session['project']}] {title}. Tool '{tool}' was used on {file_}.",
                "facts": [
                    f"{concept} was modified in {file_}",
                    f"Tool {tool} completed successfully",
                ],
                "concepts": list(set(obs_concepts)),
                "files": obs_files,
                "importance": random.randint(3, 9),
                "data": data,
            }
            observations.append(obs)

    return observations


def generate_memories(sessions: list[dict]) -> list[dict]:
    memories = []
    session_ids = [s["id"] for s in sessions]

    for i in range(config.GEN_MEMORY_COUNT):
        concept = _pick(_CONCEPTS)
        tech = _pick(_TECH_WORDS)
        template = _pick(_MEMORY_TEMPLATES)
        content = template.format(
            concept=concept,
            tech=tech,
            purpose=_pick(_PURPOSES),
            benefit=_pick(_BENEFITS),
            issue=_pick(_ISSUES),
            condition=f"{concept} is null",
            fix=f"check for null before accessing {concept}",
            file=_pick(_FILES),
        )
        created_at = _ts(-random.randint(0, 72 * 60))
        mem = {
            "id": _mem_id(),
            "content": content,
            "title": content[:80],
            "type": _pick(_MEMORY_TYPES),
            "concepts": [concept] + random.sample(_CONCEPTS, k=random.randint(1, 3)),
            "files": [_pick(_FILES)] if random.random() > 0.5 else [],
            "sessionIds": random.sample(session_ids, k=min(2, len(session_ids))),
            "project": config.TEST_PROJECT,
            "agentId": config.AGENT_ID,
            "strength": random.randint(5, 10),
            "version": 1,
            "isLatest": True,
            "supersedes": [],
            "createdAt": created_at,
            "updatedAt": created_at,
        }
        memories.append(mem)

    return memories


def generate_search_queries() -> list[dict]:
    queries = []
    for q in _SEARCH_QUERIES:
        queries.append({
            "query": q,
            "expected_concepts": q.split(),
            "limit": random.choice([5, 10, 20]),
        })
    return queries


# ── Main ───────────────────────────────────────────────────────────────────────
def main() -> None:
    print("=" * 60)
    print("01_generate_data.py — Sinh dữ liệu test")
    print("=" * 60)
    config.print_summary()
    print()

    # Đảm bảo thư mục data/ tồn tại
    config.DATA_DIR.mkdir(parents=True, exist_ok=True)
    print(f"[gen] DataDir: {config.DATA_DIR}")

    # Sinh sessions
    print(f"[gen] Đang sinh {config.GEN_SESSION_COUNT} sessions...")
    sessions = generate_sessions(config.GEN_SESSION_COUNT)
    sessions_file = config.DATA_DIR / "sessions.json"
    with sessions_file.open("w", encoding="utf-8") as f:
        json.dump(sessions, f, indent=2, ensure_ascii=False)
    print(f"      → {sessions_file} ({len(sessions)} sessions)")

    # Sinh observations
    total_obs = config.GEN_SESSION_COUNT * config.GEN_OBS_PER_SESSION
    print(f"[gen] Đang sinh {total_obs} observations...")
    observations = generate_observations(sessions)
    obs_file = config.DATA_DIR / "observations.jsonl"
    with obs_file.open("w", encoding="utf-8") as f:
        for obs in observations:
            f.write(json.dumps(obs, ensure_ascii=False) + "\n")
    print(f"      → {obs_file} ({len(observations)} records)")

    # Sinh memories
    print(f"[gen] Đang sinh {config.GEN_MEMORY_COUNT} memories...")
    memories = generate_memories(sessions)
    mem_file = config.DATA_DIR / "memories.json"
    with mem_file.open("w", encoding="utf-8") as f:
        json.dump(memories, f, indent=2, ensure_ascii=False)
    print(f"      → {mem_file} ({len(memories)} memories)")

    # Sinh search queries
    print("[gen] Đang sinh search queries...")
    queries = generate_search_queries()
    q_file = config.DATA_DIR / "search_queries.json"
    with q_file.open("w", encoding="utf-8") as f:
        json.dump(queries, f, indent=2, ensure_ascii=False)
    print(f"      → {q_file} ({len(queries)} queries)")

    # Tạo file manifest
    manifest = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "config": {
            "server": config.BASE_URL,
            "project": config.TEST_PROJECT,
            "agent_id": config.AGENT_ID,
        },
        "counts": {
            "sessions": len(sessions),
            "observations": len(observations),
            "memories": len(memories),
            "search_queries": len(queries),
        },
        "files": {
            "sessions": str(sessions_file),
            "observations": str(obs_file),
            "memories": str(mem_file),
            "search_queries": str(q_file),
        },
    }
    manifest_file = config.DATA_DIR / "manifest.json"
    with manifest_file.open("w", encoding="utf-8") as f:
        json.dump(manifest, f, indent=2)
    print(f"\n[gen] ✅ Manifest: {manifest_file}")
    print(f"[gen] ✅ Hoàn thành! Tổng: {len(sessions)} sessions, "
          f"{len(observations)} obs, {len(memories)} memories.")


if __name__ == "__main__":
    main()
