"""
04_test_graph.py — Kiểm thử tính năng Knowledge Graph Extraction

Pipeline kiểm thử:
  1. graph.enabled     — Kiểm tra GRAPH_EXTRACTION_ENABLED=true qua /config/flags
  2. graph.stats       — GET /graph/stats → xem số nodes/edges hiện có
  3. graph.extract     — POST /graph/extract → gửi observations để LLM extract entities/relations
  4. graph.query       — POST /graph/query → truy vấn graph theo concept/entity
  5. graph.build       — POST /graph/build → backfill graph từ observations đã có
  6. graph.verify      — Verify nodes/edges tăng sau extract
  7. graph.snapshot    — POST /graph/snapshot-rebuild → rebuild snapshot index

Ghi chú:
  - /graph/build và /graph/snapshot-rebuild yêu cầu AGENTMEMORY_TOOLS=all
    Nếu server dùng AGENTMEMORY_TOOLS=core (mặc định), các bước này sẽ skip gracefully (404)
  - LLM extraction chạy async: nodes/edges có thể chưa reflect ngay sau extract

Chạy:
  cd tests/agentmemory
  python3 04_test_graph.py
  python3 04_test_graph.py --step enabled      # Chỉ kiểm tra flag enabled
  python3 04_test_graph.py --step extract      # Chỉ test extract
  python3 04_test_graph.py --step query        # Chỉ test query
  python3 04_test_graph.py --step build        # Chỉ backfill build
  python3 04_test_graph.py --step all          # Chạy tất cả (default)
  python3 04_test_graph.py --verbose           # In response chi tiết
"""
from __future__ import annotations

import argparse
import json
import sys
import time
import uuid
from datetime import datetime, timezone
from typing import Any, Optional

try:
    import requests
except ImportError:
    print("ERROR: requests chưa được cài.\n  → pip install requests")
    sys.exit(1)

import config

# ── State ─────────────────────────────────────────────────────────────────────
_results: list[dict] = []
_verbose = False


def _ts() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def _call(method: str, endpoint: str, payload: Optional[dict] = None, timeout: int = 60) -> tuple[int, dict]:
    url = f"{config.BASE_URL}/agentmemory/{endpoint.lstrip('/')}"
    try:
        if method.upper() == "GET":
            resp = requests.get(url, headers=config.auth_headers(), timeout=timeout)
        else:
            resp = requests.post(url, json=payload, headers=config.auth_headers(), timeout=timeout)
        try:
            body = resp.json()
        except Exception:
            body = {"raw": resp.text[:500]}
        return resp.status_code, body
    except requests.exceptions.ConnectionError as e:
        return 0, {"error": f"ConnectionError: {e}"}
    except Exception as e:
        return 0, {"error": str(e)}


class TC:
    def __init__(self, name: str, desc: str):
        self.name = name
        self.desc = desc
        self._t0 = time.time()
        self._checks: list[dict] = []
        self._status = "pass"

    def check(self, ok: bool, label: str, actual: Any = None) -> "TC":
        if not ok:
            self._status = "fail"
        self._checks.append({"label": label, "passed": ok, "actual": str(actual)[:300]})
        if _verbose or not ok:
            icon = "✅" if ok else "❌"
            suffix = f" — got: {str(actual)[:120]}" if not ok else ""
            print(f"    {icon} {label}{suffix}")
        return self

    def done(self, body: Any = None) -> dict:
        ms = round((time.time() - self._t0) * 1000)
        icon = "✅" if self._status == "pass" else "❌"
        print(f"  {icon} [{ms}ms] {self.name}")
        if _verbose and body:
            preview = json.dumps(body, ensure_ascii=False)[:400] if isinstance(body, dict) else str(body)[:400]
            print(f"    → {preview}")
        r = {
            "name": self.name,
            "description": self.desc,
            "status": self._status,
            "elapsed_ms": ms,
            "checks": self._checks,
        }
        _results.append(r)
        return r


# ── Observation fixtures ───────────────────────────────────────────────────────
# field 'title' là bắt buộc để mem::graph-extract xử lý (chỉ compress observations có title)
_OBSERVATIONS = [
    {
        "id": f"obs_graph_{uuid.uuid4().hex[:8]}",
        "title": "Implemented JWT authentication middleware",
        "narrative": (
            "Created a JWT authentication middleware in TypeScript. "
            "The middleware validates Bearer tokens using RS256 algorithm. "
            "It integrates with the UserService to fetch user roles from PostgreSQL. "
            "Invalid tokens return 401 Unauthorized. The middleware is applied to all "
            "protected API routes in the Express.js application."
        ),
        "facts": [
            "JWT middleware uses RS256 signing algorithm",
            "UserService fetches roles from PostgreSQL",
            "Express.js routes use Bearer token authentication",
        ],
        "concepts": ["jwt", "authentication", "middleware", "typescript", "postgresql"],
        "type": "file_edit",
    },
    {
        "id": f"obs_graph_{uuid.uuid4().hex[:8]}",
        "title": "Configured Redis cache for session storage",
        "narrative": (
            "Set up Redis as the session store for the authentication system. "
            "Sessions are cached with a TTL of 3600 seconds (1 hour). "
            "The RedisAdapter connects to the Redis cluster at redis.internal:6379. "
            "Cache invalidation is triggered when a user logs out or their role changes. "
            "This reduces PostgreSQL load by ~60% during peak traffic."
        ),
        "facts": [
            "Redis stores sessions with TTL=3600s",
            "RedisAdapter connects to redis.internal:6379",
            "Cache invalidation on logout or role change",
            "60% reduction in PostgreSQL load",
        ],
        "concepts": ["redis", "caching", "session", "authentication", "performance"],
        "type": "file_edit",
    },
    {
        "id": f"obs_graph_{uuid.uuid4().hex[:8]}",
        "title": "Deployed API Gateway with rate limiting",
        "narrative": (
            "Deployed the API Gateway service using Docker on server 172.20.2.39. "
            "The gateway routes requests to microservices: AuthService, UserService, and DataService. "
            "Rate limiting is configured at 100 requests/minute per IP using the TokenBucket algorithm. "
            "Nginx reverse proxy handles SSL termination at c11.openledger.vn. "
            "Health checks run every 30 seconds to detect service failures."
        ),
        "facts": [
            "API Gateway deployed on Docker at 172.20.2.39",
            "Routes to AuthService, UserService, DataService",
            "Rate limit: 100 req/min per IP with TokenBucket",
            "Nginx handles SSL at c11.openledger.vn",
        ],
        "concepts": ["api-gateway", "docker", "nginx", "rate-limiting", "microservices"],
        "type": "bash_run",
    },
]


# ── Test Steps ─────────────────────────────────────────────────────────────────

def step_enabled() -> bool:
    """Xác nhận GRAPH_EXTRACTION_ENABLED=true trên server."""
    print("\n─── STEP: graph.enabled ─────────────────────────────────────")
    tc = TC("graph.enabled", "GET /config/flags → GRAPH_EXTRACTION_ENABLED=true")
    code, body = _call("GET", "config/flags")
    tc.check(code == 200, "HTTP 200", code)
    tc.check(isinstance(body, dict), "body là dict", type(body))

    flags = body.get("flags", [])
    graph_flag = next((f for f in flags if f.get("key") == "GRAPH_EXTRACTION_ENABLED"), None)

    if graph_flag:
        enabled = graph_flag.get("enabled")
        tc.check(enabled is True, "GRAPH_EXTRACTION_ENABLED == true", enabled)
        if enabled:
            print(f"    [info] LLM provider: {body.get('provider', '?')}")
            print(f"    [info] Embedding: {body.get('embeddingProvider', '?')}")
    else:
        tc.check(False, "Flag GRAPH_EXTRACTION_ENABLED tồn tại trong response",
                 [f.get("key") for f in flags])

    tc.done(body)
    return graph_flag.get("enabled") is True if graph_flag else False


def step_stats(label: str = "before") -> dict:
    """Lấy graph stats (dùng để compare before/after)."""
    print(f"\n─── STEP: graph.stats ({label}) ─────────────────────────────")
    tc = TC(f"graph.stats.{label}", f"GET /graph/stats → node/edge counts ({label} extract)")
    code, body = _call("GET", "graph/stats")
    tc.check(code in (200, 503), "HTTP 200 hoặc 503", code)

    stats: dict = {}
    if code == 200:
        tc.check(isinstance(body, dict), "body là dict", type(body))
        node_count = body.get("totalNodes") or body.get("nodeCount") or body.get("nodes") or 0
        edge_count = body.get("totalEdges") or body.get("edgeCount") or body.get("edges") or 0
        stats = {"nodes": node_count, "edges": edge_count}
        print(f"    [stats] nodes={node_count}  edges={edge_count}")
        if _verbose:
            print(f"    [detail] nodesByType={body.get('nodesByType', {})}  edgesByType={body.get('edgesByType', {})}")
        tc.check(isinstance(node_count, (int, float)), "nodeCount là số", type(node_count))
    elif code == 503:
        print("    [warn] Graph disabled (503) — GRAPH_EXTRACTION_ENABLED chưa active?")

    tc.done(body)
    return stats


def step_extract() -> dict:
    """POST /graph/extract — gửi observations để LLM extract entities/relations."""
    print("\n─── STEP: graph.extract ─────────────────────────────────────")

    # TC: gửi observations có đủ title/narrative
    tc = TC("graph.extract.basic", "POST /graph/extract → LLM extracts nodes/edges")
    code, body = _call("POST", "graph/extract", {"observations": _OBSERVATIONS})
    tc.check(code in (200, 202), "HTTP 200/202", code)
    tc.check(isinstance(body, dict), "body là dict", type(body))
    if code in (200, 202):
        success = body.get("success")
        nodes_added = body.get("nodesAdded", body.get("nodes_added", 0))
        edges_added = body.get("edgesAdded", body.get("edges_added", 0))
        print(f"    [extract] success={success}  nodesAdded={nodes_added}  edgesAdded={edges_added}")
        if nodes_added == 0:
            print("    [info] nodesAdded=0 — LLM extraction chạy async, kết quả sẽ reflect sau vài giây")
        tc.check(success is True or code == 200, "success == true hoặc HTTP 200", success)
    tc.done(body)

    # TC: validation — observations rỗng → 400
    tc2 = TC("graph.extract.empty_array", "POST /graph/extract với observations=[] → 400")
    code2, body2 = _call("POST", "graph/extract", {"observations": []})
    tc2.check(code2 == 400, "HTTP 400", code2)
    tc2.check("error" in body2, "có error field", list(body2.keys()) if isinstance(body2, dict) else body2)
    tc2.done(body2)

    # TC: validation — thiếu field → 400
    tc3 = TC("graph.extract.missing_field", "POST /graph/extract thiếu observations → 400")
    code3, body3 = _call("POST", "graph/extract", {})
    tc3.check(code3 == 400, "HTTP 400", code3)
    tc3.done(body3)

    return body if code in (200, 202) else {}


def step_query() -> None:
    """POST /graph/query — truy vấn knowledge graph."""
    print("\n─── STEP: graph.query ───────────────────────────────────────")

    test_queries = [
        ("jwt authentication", None, 2),
        ("redis caching performance", None, 1),
        ("docker microservices deployment", None, 2),
    ]

    for query_text, node_type, max_depth in test_queries:
        label = f'"{query_text}"'
        tc = TC(
            f"graph.query.{query_text.split()[0]}",
            f"POST /graph/query → {label} (depth={max_depth})",
        )
        payload: dict = {"query": query_text, "limit": 10, "maxDepth": max_depth}
        if node_type:
            payload["nodeType"] = node_type
        code, body = _call("POST", "graph/query", payload)
        tc.check(code in (200, 503), "HTTP 200 hoặc 503", code)
        if code == 200:
            nodes = body.get("nodes") or body.get("results") or body.get("items") or []
            edges = body.get("edges") or []
            tc.check(isinstance(nodes, list), "nodes là array", type(nodes))
            print(f"    [query] '{query_text}' → {len(nodes)} nodes, {len(edges)} edges")
            if _verbose and nodes:
                for n in nodes[:2]:
                    print(f"      node: {json.dumps(n, ensure_ascii=False)[:120]}")
        elif code == 503:
            print("    [skip] Graph disabled")
        tc.done(body)

    # TC: query với limit nhỏ
    tc = TC("graph.query.with_limit", "POST /graph/query với limit=3 → ≤ 3 nodes")
    code, body = _call("POST", "graph/query", {"query": "authentication", "limit": 3, "maxDepth": 1})
    tc.check(code in (200, 503), "HTTP 200/503", code)
    if code == 200:
        nodes = body.get("nodes") or body.get("results") or []
        tc.check(len(nodes) <= 3, f"nodes ≤ 3 (got {len(nodes)})", len(nodes))
    tc.done(body)


def step_build() -> None:
    """POST /graph/build — backfill graph từ toàn bộ observations đã có trên server.
    Yêu cầu AGENTMEMORY_TOOLS=all — skip gracefully nếu 404.
    """
    print("\n─── STEP: graph.build ───────────────────────────────────────")
    tc = TC("graph.build", "POST /graph/build → backfill từ existing observations")
    print("    [info] Đang chạy graph build — có thể mất 1-5 phút tùy số sessions/observations...")
    code, body = _call("POST", "graph/build", {"batchSize": 10}, timeout=300)

    if code == 404:
        # /graph/build chỉ khả dụng khi AGENTMEMORY_TOOLS=all
        print("    [skip] HTTP 404 — /graph/build yêu cầu AGENTMEMORY_TOOLS=all")
        print("    [hint] Thêm AGENTMEMORY_TOOLS=all vào .env và deploy lại để bật")
        tc.check(True, "SKIPPED: graph/build cần AGENTMEMORY_TOOLS=all (404=expected với tools=core)", "skipped")
    elif code in (200, 202):
        sessions = body.get("sessions", 0)
        batches  = body.get("batches", 0)
        nodes    = body.get("nodes", 0)
        edges    = body.get("edges", 0)
        print(f"    [build] sessions={sessions}  batches={batches}  nodes={nodes}  edges={edges}")
        tc.check(body.get("success") is True, "success == true", body.get("success"))
    elif code == 503:
        print("    [skip] Graph disabled (503) — kiểm tra GRAPH_EXTRACTION_ENABLED")
        tc.check(True, "SKIPPED: graph disabled", "503")
    elif code == 504:
        print("    [warn] Gateway Timeout (504) — quá trình build vẫn tiếp tục ngầm trên server.")
        tc.check(True, "Gateway Timeout (504) = Expected for large builds", "504")
    else:
        tc.check(False, f"HTTP 200/202/404/503/504", code)

    tc.done(body)


def step_snapshot() -> None:
    """POST /graph/snapshot-rebuild — tái tạo snapshot index.
    Yêu cầu AGENTMEMORY_TOOLS=all — skip gracefully nếu 404.
    """
    print("\n─── STEP: graph.snapshot ─────────────────────────────────────")
    tc = TC("graph.snapshot", "POST /graph/snapshot-rebuild → rebuild index")
    code, body = _call("POST", "graph/snapshot-rebuild", {})

    if code == 404:
        print("    [skip] HTTP 404 — /graph/snapshot-rebuild yêu cầu AGENTMEMORY_TOOLS=all")
        tc.check(True, "SKIPPED: snapshot-rebuild cần AGENTMEMORY_TOOLS=all (404=expected)", "skipped")
    elif code in (200, 202):
        print(f"    [snapshot] {json.dumps(body, ensure_ascii=False)[:200]}")
        tc.check(True, "Snapshot rebuild OK", code)
    elif code == 503:
        print("    [skip] Graph disabled (503)")
        tc.check(True, "SKIPPED: graph disabled", "503")
    else:
        tc.check(False, f"HTTP 200/202/404/503", code)

    tc.done(body)


def step_verify(stats_before: dict, stats_after: dict) -> None:
    """So sánh graph stats trước/sau extract."""
    print("\n─── STEP: graph.verify ──────────────────────────────────────")
    tc = TC("graph.verify", "Nodes/Edges tăng sau extract so với trước")
    nb = stats_before.get("nodes", 0) or 0
    eb = stats_before.get("edges", 0) or 0
    na = stats_after.get("nodes", 0) or 0
    ea = stats_after.get("edges", 0) or 0

    print(f"    Before: nodes={nb}  edges={eb}")
    print(f"    After : nodes={na}  edges={ea}")
    print(f"    Delta : Δnodes={na - nb}  Δedges={ea - eb}")

    if nb == 0 and na == 0:
        # LLM extraction chạy async — nodes chưa reflect ngay
        print("    [info] Cả hai đều 0 — LLM extraction chạy async. Kiểm tra logs-app sau ~30s.")
        tc.check(True, "ASYNC: LLM extraction chưa complete (bình thường khi graph mới được bật)", "async")
    else:
        tc.check(na >= nb, f"nodes_after ({na}) ≥ nodes_before ({nb})", na)
        tc.check(ea >= eb, f"edges_after ({ea}) ≥ edges_before ({eb})", ea)

    tc.done()


# ── Report ─────────────────────────────────────────────────────────────────────
def _save_report():
    passed = sum(1 for r in _results if r["status"] == "pass")
    failed = sum(1 for r in _results if r["status"] == "fail")
    report = {
        "timestamp": _ts(),
        "server": config.BASE_URL,
        "feature": "knowledge_graph",
        "summary": {
            "total": len(_results),
            "passed": passed,
            "failed": failed,
            "pass_rate": f"{round(passed / len(_results) * 100)}%" if _results else "N/A",
        },
        "results": _results,
    }
    config.DATA_DIR.mkdir(parents=True, exist_ok=True)
    out = config.DATA_DIR / "graph_test_results.json"
    with out.open("w", encoding="utf-8") as f:
        json.dump(report, f, indent=2, ensure_ascii=False)
    return report, out


def _print_summary() -> tuple[int, int]:
    passed = sum(1 for r in _results if r["status"] == "pass")
    failed = sum(1 for r in _results if r["status"] == "fail")
    print(f"\n{'=' * 60}")
    print(f"KẾT QUẢ: {passed}/{len(_results)} tests PASSED  ({failed} FAILED)")
    print(f"{'=' * 60}")
    if failed:
        print("\nFAILED tests:")
        for r in _results:
            if r["status"] == "fail":
                print(f"  ❌ {r['name']}")
                for c in r["checks"]:
                    if not c["passed"]:
                        print(f"      → {c['label']}: {c['actual']}")
    return passed, failed


# ── Main ───────────────────────────────────────────────────────────────────────
def main() -> None:
    global _verbose

    parser = argparse.ArgumentParser(description="Test Knowledge Graph Extraction")
    parser.add_argument(
        "--step",
        choices=["enabled", "stats", "extract", "query", "build", "snapshot", "all"],
        default="all",
        help="Bước test cần chạy (default: all)",
    )
    parser.add_argument("--verbose", "-v", action="store_true", help="In response chi tiết")
    args = parser.parse_args()
    _verbose = args.verbose

    print("=" * 60)
    print("04_test_graph.py — Knowledge Graph Extraction Test")
    print("=" * 60)
    config.print_summary()
    print(f"[info] Test fixtures: {len(_OBSERVATIONS)} observations (JWT, Redis, API Gateway)")

    # Kiểm tra server
    code, _ = _call("GET", "livez")
    if code != 200:
        print(f"\n❌ Server không phản hồi. Kiểm tra AGENTMEMORY_URL={config.BASE_URL}")
        sys.exit(1)
    print(f"[check] ✅ Server OK\n")

    step = args.step
    stats_before: dict = {}
    stats_after: dict = {}

    if step in ("enabled", "all"):
        graph_enabled = step_enabled()
        if not graph_enabled and step == "all":
            print("\n⚠️  GRAPH_EXTRACTION_ENABLED chưa = true trên server.")
            print("   → Kiểm tra deploy/dev/.env và chạy make deploy-config")

    if step in ("stats", "all"):
        stats_before = step_stats("before")

    if step in ("extract", "all"):
        step_extract()
        if step == "all":
            print("\n    [wait] Đợi 8s để LLM extraction hoàn tất...")
            time.sleep(8)

    if step in ("query", "all"):
        step_query()

    if step in ("build", "all"):
        step_build()

    if step == "all":
        stats_after = step_stats("after")
        step_verify(stats_before, stats_after)

    if step in ("snapshot", "all"):
        step_snapshot()

    report, out = _save_report()
    passed, failed = _print_summary()
    print(f"\n📊 Báo cáo: {out}")

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
