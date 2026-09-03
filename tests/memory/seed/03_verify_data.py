#!/usr/bin/env python3
"""
03_verify_data.py — Query the VNP Memory server and verify seeded data.

Calls frontend-facing console APIs (ui/specs/frontend-backend-api-specs.md)
and validates responses against expected counts and shapes.

Usage:
    python 03_verify_data.py [--engine ENGINE] [--verbose]

Options:
    --engine   Only verify the specified domain:
               dashboard | memory | graph | profiles | adaptive | sessions |
               governance | pipelines | infra | observability | all
    --verbose  Print full JSON responses
"""

from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass, field
from typing import Any

from client import api, cfg, load_json, print_section


# ── Result tracking ───────────────────────────────────────────────────────────

@dataclass
class CheckResult:
    name: str
    passed: bool
    detail: str = ""


@dataclass
class VerifyReport:
    checks: list[CheckResult] = field(default_factory=list)

    def add(self, name: str, condition: bool, detail: str = "") -> None:
        status = "✓" if condition else "✗"
        print(f"    {status} {name}" + (f" — {detail}" if detail else ""))
        self.checks.append(CheckResult(name, condition, detail))

    def summary(self) -> tuple[int, int]:
        passed = sum(1 for c in self.checks if c.passed)
        return passed, len(self.checks)


report = VerifyReport()
VERBOSE = False


def dump(label: str, data: Any) -> None:
    if VERBOSE:
        print(f"\n  [{label}]")
        print(json.dumps(data, indent=2, default=str)[:2000])


# ── Helpers ───────────────────────────────────────────────────────────────────

def check_list_not_empty(path: str, label: str, **kwargs) -> list | None:
    data = api.safe_get(path, **kwargs)
    dump(label, data)
    if data is None:
        report.add(f"{label} responds", False, "HTTP error")
        return None
    # Handle paginated wrapper
    items = data if isinstance(data, list) else data.get("data", data.get("results", []))
    report.add(f"{label} returns data", bool(items), f"{len(items)} items")
    return items


def check_field(obj: dict | None, field: str, label: str) -> bool:
    if obj is None:
        report.add(label, False, "response is None")
        return False
    val = obj.get(field)
    ok = val is not None and val != "" and val != [] and val != {}
    report.add(label, ok, f"{field}={repr(val)[:60]}")
    return ok


# ── 1. Auth ───────────────────────────────────────────────────────────────────

def verify_auth() -> None:
    print_section("1. Auth API Verification")

    # GET /v1/auth/me
    me = api.safe_get("/v1/auth/me")
    dump("me", me)
    if me:
        # Frontend expects {"user": {...}} or direct user object
        user = me.get("user", me)
        report.add("GET /v1/auth/me responds", True)
        check_field(user, "id", "me.id present")
        check_field(user, "role", "me.role present")
        check_field(user, "email", "me.email present")
    else:
        report.add("GET /v1/auth/me responds", False, "Known gap: CR-001 — endpoint not yet implemented")


# ── 2. Dashboard ──────────────────────────────────────────────────────────────

def verify_dashboard() -> None:
    print_section("2. Dashboard API Verification")

    # GET /v1/console/dashboard/health
    health = api.safe_get("/v1/console/dashboard/health")
    dump("health", health)
    if health:
        report.add("GET /v1/console/dashboard/health responds", True)
        engines = health if isinstance(health, list) else health.get("engines", [])
        report.add("health: engine list present", len(engines) > 0, f"{len(engines)} engines")
        if engines and isinstance(engines[0], dict):
            check_field(engines[0], "name", "engine[0].name present")
            check_field(engines[0], "status", "engine[0].status present")
    else:
        report.add("GET /v1/console/dashboard/health responds", False)

    # GET /v1/console/dashboard/metrics
    metrics = api.safe_get("/v1/console/dashboard/metrics")
    dump("metrics", metrics)
    if metrics:
        report.add("GET /v1/console/dashboard/metrics responds", True)
        # KPIData fields
        for fld in ["activeAgents", "recallLatencyP50Ms", "errorRatePct"]:
            check_field(metrics, fld, f"metrics.{fld}")
    else:
        report.add("GET /v1/console/dashboard/metrics responds", False)

    # GET /v1/console/dashboard/throughput
    tp = api.safe_get("/v1/console/dashboard/throughput", params={"window": "1h"})
    dump("throughput", tp)
    report.add("GET /v1/console/dashboard/throughput responds", tp is not None)

    # GET /v1/console/dashboard/heatmap
    hm = api.safe_get("/v1/console/dashboard/heatmap")
    dump("heatmap", hm)
    report.add("GET /v1/console/dashboard/heatmap responds", hm is not None)


# ── 3. Memory Explorer ────────────────────────────────────────────────────────

def verify_memory() -> None:
    print_section("3. Memory Explorer API Verification")

    # POST /v1/console/memory/search
    search_body = {
        "query": "knowledge graph temporal reasoning",
        "mode": "hybrid",
        "engines": ["graphiti", "cognee"],
        "filters": {},
        "limit": 10,
        "offset": 0,
        "reranking": "rrf",
    }
    result = api.safe_post("/v1/console/memory/search", search_body)
    dump("memory/search", result)
    if result:
        report.add("POST /v1/console/memory/search responds", True)
        results = result.get("results", [])
        report.add("search: results array present", isinstance(results, list),
                   f"{len(results)} results")
        check_field(result, "total", "search.total present")
        check_field(result, "latencyMs", "search.latencyMs present")
        if results:
            first = results[0]
            check_field(first, "id", "result[0].id")
            check_field(first, "engine", "result[0].engine")
            check_field(first, "content", "result[0].content")
    else:
        report.add("POST /v1/console/memory/search responds", False)

    # Verify a seeded graphiti episode if we have its ID
    try:
        graphiti_data = load_json("created_graphiti.json")
        if graphiti_data:
            ep_id = graphiti_data[0].get("id", "")
            if ep_id and ep_id != "?":
                import urllib.parse
                encoded_id = urllib.parse.quote(f"graphiti:{ep_id}", safe="")
                detail = api.safe_get(f"/v1/console/memory/{encoded_id}")
                report.add("GET /v1/console/memory/{id} (graphiti ep)", detail is not None,
                           f"id=graphiti:{ep_id[:12]}")
    except FileNotFoundError:
        report.add("GET /v1/console/memory/{id}", False, "created_graphiti.json missing — run 02_load_data.py first")


# ── 4. Graph Studio ───────────────────────────────────────────────────────────

def verify_graph() -> None:
    print_section("4. Graph Studio API Verification")

    # GET /v1/console/graph/ontology
    ontology = api.safe_get("/v1/console/graph/ontology")
    dump("ontology", ontology)
    if ontology:
        report.add("GET /v1/console/graph/ontology responds", True)
        check_field(ontology, "classes", "ontology.classes present")
        check_field(ontology, "relationships", "ontology.relationships present")
    else:
        report.add("GET /v1/console/graph/ontology responds", False)

    # POST /v1/console/graph/subgraph
    sg = api.safe_post("/v1/console/graph/subgraph", {"entity": "Alice", "depth": 1})
    dump("subgraph", sg)
    if sg:
        report.add("POST /v1/console/graph/subgraph responds", True)
        check_field(sg, "nodes", "subgraph.nodes present")
        check_field(sg, "edges", "subgraph.edges present")
    else:
        report.add("POST /v1/console/graph/subgraph responds", False)

    # POST /v1/console/graph/query
    query_result = api.safe_post("/v1/console/graph/query", {"query": "MATCH (n) RETURN n LIMIT 5"})
    report.add("POST /v1/console/graph/query responds", query_result is not None)


# ── 5. User Profiles ──────────────────────────────────────────────────────────

def verify_profiles() -> None:
    print_section("5. User Profiles API Verification")

    # GET /v1/console/profiles
    profiles = api.safe_get("/v1/console/profiles")
    dump("profiles", profiles)
    if profiles is not None:
        plist = profiles if isinstance(profiles, list) else profiles.get("data", [])
        report.add("GET /v1/console/profiles responds", True, f"{len(plist)} profiles")

        if plist:
            uid = plist[0].get("user_id", "")
            if uid:
                # GET /v1/console/profiles/{user_id}
                detail = api.safe_get(f"/v1/console/profiles/{uid}")
                report.add(f"GET /v1/console/profiles/{uid}", detail is not None)
                if detail:
                    check_field(detail, "user_id", "profile.user_id")
                    check_field(detail, "profiles", "profile.profiles (topics)")

                # GET /v1/console/profiles/{user_id}/context
                ctx = api.safe_get(f"/v1/console/profiles/{uid}/context")
                report.add(f"GET /v1/console/profiles/{uid}/context", ctx is not None)
                if ctx:
                    check_field(ctx, "context_string", "context.context_string")
                    check_field(ctx, "token_count", "context.token_count")

                # GET /v1/console/profiles/{user_id}/events
                events = api.safe_get(f"/v1/console/profiles/{uid}/events")
                report.add(f"GET /v1/console/profiles/{uid}/events", events is not None)
    else:
        report.add("GET /v1/console/profiles responds", False)

    # GET /v1/console/profiles/config
    config = api.safe_get("/v1/console/profiles/config")
    if config:
        report.add("GET /v1/console/profiles/config responds", True)
        check_field(config, "profiles", "config.profiles (schema)")


# ── 6. Adaptive Memory ────────────────────────────────────────────────────────

def verify_adaptive() -> None:
    print_section("6. Adaptive Memory API Verification")

    # GET /v1/console/adaptive/memories
    mems = check_list_not_empty("/v1/console/adaptive/memories", "adaptive/memories")
    if mems:
        first_id = mems[0].get("id", "")
        check_field(mems[0], "content", "adaptive[0].content")
        check_field(mems[0], "memory_type", "adaptive[0].memory_type")

    # GET /v1/console/adaptive/connectors
    connectors = api.safe_get("/v1/console/adaptive/connectors")
    report.add("GET /v1/console/adaptive/connectors responds", connectors is not None)

    # GET /v1/console/adaptive/analytics
    analytics = api.safe_get("/v1/console/adaptive/analytics")
    if analytics:
        report.add("GET /v1/console/adaptive/analytics responds", True)
        check_field(analytics, "creation_rate", "analytics.creation_rate")

    # GET /v1/console/adaptive/forget-rules
    rules = api.safe_get("/v1/console/adaptive/forget-rules")
    report.add("GET /v1/console/adaptive/forget-rules responds", rules is not None)


# ── 7. Sessions ───────────────────────────────────────────────────────────────

def verify_sessions() -> None:
    print_section("7. Sessions API Verification")

    # GET /v1/console/sessions
    sessions = api.safe_get("/v1/console/sessions", params={"page": 1, "page_size": 10})
    dump("sessions", sessions)
    if sessions:
        slist = sessions if isinstance(sessions, list) else sessions.get("data", [])
        report.add("GET /v1/console/sessions responds", True, f"{len(slist)} sessions")

        if slist:
            sid = slist[0].get("id", "")
            if sid:
                # GET /v1/console/sessions/{id}
                detail = api.safe_get(f"/v1/console/sessions/{sid}")
                report.add(f"GET /v1/console/sessions/{sid}", detail is not None)
                if detail:
                    check_field(detail, "messages", "session.messages")

                # GET /v1/console/sessions/{id}/timeline
                tl = api.safe_get(f"/v1/console/sessions/{sid}/timeline")
                report.add(f"GET /v1/console/sessions/{sid}/timeline", tl is not None)

                # GET /v1/console/sessions/{id}/working-memory
                wm = api.safe_get(f"/v1/console/sessions/{sid}/working-memory")
                report.add(f"GET /v1/console/sessions/{sid}/working-memory", wm is not None)
    else:
        report.add("GET /v1/console/sessions responds", False)

    # GET /v1/console/sessions/live
    live = api.safe_get("/v1/console/sessions/live")
    report.add("GET /v1/console/sessions/live responds", live is not None)


# ── 8. Governance ─────────────────────────────────────────────────────────────

def verify_governance() -> None:
    print_section("8. Governance API Verification")

    # GET /v1/console/governance/tenants
    tenants = api.safe_get("/v1/console/governance/tenants")
    if tenants is not None:
        tlist = tenants if isinstance(tenants, list) else tenants.get("data", [])
        report.add("GET /v1/console/governance/tenants responds", True, f"{len(tlist)} tenants")
        if tlist:
            check_field(tlist[0], "id", "tenant[0].id")
            check_field(tlist[0], "name", "tenant[0].name")
    else:
        report.add("GET /v1/console/governance/tenants responds", False)

    # GET /v1/console/governance/policies
    policies = api.safe_get("/v1/console/governance/policies")
    report.add("GET /v1/console/governance/policies responds", policies is not None)

    # GET /v1/console/governance/audit
    audit = api.safe_get("/v1/console/governance/audit")
    report.add("GET /v1/console/governance/audit responds", audit is not None)

    # GDPR preview (dry-run): POST /v1/console/governance/gdpr/forget/preview
    preview = api.safe_post("/v1/console/governance/gdpr/forget/preview",
                            {"user_id": "user_001"})
    if preview:
        report.add("POST governance/gdpr/forget/preview responds", True)
        check_field(preview, "estimated_items", "preview.estimated_items")
        check_field(preview, "breakdown_by_engine", "preview.breakdown_by_engine")
    else:
        report.add("POST governance/gdpr/forget/preview responds", False)


# ── 9. Pipelines ──────────────────────────────────────────────────────────────

def verify_pipelines() -> None:
    print_section("9. Pipelines API Verification")

    # GET /v1/console/pipelines/status
    status = api.safe_get("/v1/console/pipelines/status")
    report.add("GET /v1/console/pipelines/status responds", status is not None)

    # GET /v1/console/pipelines/queues
    queues = api.safe_get("/v1/console/pipelines/queues")
    report.add("GET /v1/console/pipelines/queues responds", queues is not None)

    # GET /v1/console/pipelines/workers
    workers = api.safe_get("/v1/console/pipelines/workers")
    report.add("GET /v1/console/pipelines/workers responds", workers is not None)

    # Per-engine
    for engine in ["graphiti", "cognee"]:
        jobs = api.safe_get(f"/v1/console/pipelines/{engine}/jobs")
        report.add(f"GET /v1/console/pipelines/{engine}/jobs responds", jobs is not None)


# ── 10. Infrastructure ────────────────────────────────────────────────────────

def verify_infra() -> None:
    print_section("10. Infrastructure API Verification")

    # GET /v1/console/infra/topology
    topo = api.safe_get("/v1/console/infra/topology")
    if topo:
        report.add("GET /v1/console/infra/topology responds", True)
        check_field(topo, "services", "topology.services")
    else:
        report.add("GET /v1/console/infra/topology responds", False)

    # GET /v1/console/infra/services
    services = api.safe_get("/v1/console/infra/services")
    if services:
        slist = services if isinstance(services, list) else services.get("data", [])
        report.add("GET /v1/console/infra/services responds", True, f"{len(slist)} services")
        if slist:
            check_field(slist[0], "name", "service[0].name")
            check_field(slist[0], "status", "service[0].status")
    else:
        report.add("GET /v1/console/infra/services responds", False)

    # GET /v1/console/infra/databases
    dbs = api.safe_get("/v1/console/infra/databases")
    if dbs:
        dblist = dbs if isinstance(dbs, list) else dbs.get("data", [])
        report.add("GET /v1/console/infra/databases responds", True, f"{len(dblist)} databases")
    else:
        report.add("GET /v1/console/infra/databases responds", False)

    # GET /v1/console/infra/resources
    resources = api.safe_get("/v1/console/infra/resources")
    report.add("GET /v1/console/infra/resources responds", resources is not None)


# ── 11. Observability ─────────────────────────────────────────────────────────

def verify_observability() -> None:
    print_section("11. Observability API Verification")

    # GET /v1/console/observability/metrics
    metrics = api.safe_get("/v1/console/observability/metrics")
    if metrics:
        report.add("GET /v1/console/observability/metrics responds", True)
        check_field(metrics, "latency", "metrics.latency time-series")
        check_field(metrics, "error_rate", "metrics.error_rate time-series")
    else:
        report.add("GET /v1/console/observability/metrics responds", False)

    # GET /v1/console/observability/traces
    traces = api.safe_get("/v1/console/observability/traces")
    if traces:
        tlist = traces if isinstance(traces, list) else traces.get("data", [])
        report.add("GET /v1/console/observability/traces responds", True, f"{len(tlist)} traces")
        if tlist:
            check_field(tlist[0], "trace_id", "trace[0].trace_id")
            check_field(tlist[0], "service", "trace[0].service")
    else:
        report.add("GET /v1/console/observability/traces responds", False)

    # GET /v1/console/observability/errors
    errors = api.safe_get("/v1/console/observability/errors")
    report.add("GET /v1/console/observability/errors responds", errors is not None)

    # GET /v1/console/observability/costs
    costs = api.safe_get("/v1/console/observability/costs")
    report.add("GET /v1/console/observability/costs responds", costs is not None)


# ── Main ──────────────────────────────────────────────────────────────────────

VERIFIERS = {
    "auth": verify_auth,
    "dashboard": verify_dashboard,
    "memory": verify_memory,
    "graph": verify_graph,
    "profiles": verify_profiles,
    "adaptive": verify_adaptive,
    "sessions": verify_sessions,
    "governance": verify_governance,
    "pipelines": verify_pipelines,
    "infra": verify_infra,
    "observability": verify_observability,
}

VERIFY_ORDER = list(VERIFIERS.keys())


def main() -> None:
    global VERBOSE

    parser = argparse.ArgumentParser(description="Verify VNP Memory seeded data via frontend API")
    parser.add_argument(
        "--engine",
        choices=list(VERIFIERS.keys()) + ["all"],
        default="all",
        help="Which domain to verify (default: all)",
    )
    parser.add_argument("--verbose", action="store_true", help="Print full JSON responses")
    args = parser.parse_args()
    VERBOSE = args.verbose

    print(f"\n{'━' * 60}")
    print(" VNP Memory — Seed Data Verifier")
    print(f" Server: {cfg.base_url}")
    print(f" Domain: {args.engine}")
    print(f"{'━' * 60}")

    domains = VERIFY_ORDER if args.engine == "all" else [args.engine]

    for domain in domains:
        try:
            VERIFIERS[domain]()
        except Exception as exc:
            print(f"\n  ❌ Error in '{domain}': {exc}")
            report.add(f"{domain} (exception)", False, str(exc)[:120])

    # ── Final report ─────────────────────────────────────────────────────────
    passed, total = report.summary()
    pct = 100 * passed // total if total else 0

    print(f"\n{'━' * 60}")
    print(f" RESULTS: {passed}/{total} checks passed ({pct}%)")
    print(f"{'━' * 60}")

    failed = [c for c in report.checks if not c.passed]
    if failed:
        print("\n Failed checks:")
        for c in failed:
            print(f"   ✗ {c.name}" + (f" — {c.detail}" if c.detail else ""))

    # Exit code: 0 = all pass, 1 = some fail
    sys.exit(0 if not failed else 1)


if __name__ == "__main__":
    main()
