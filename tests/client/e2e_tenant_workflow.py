#!/usr/bin/env python3
"""
e2e_tenant_workflow.py — End-to-end workflow test cho KG Service

Kịch bản:
  1. Health check
  2. Tạo tenant mới  (POST /v1/access/tenants)
  3. Tạo app cho tenant  (POST /v1/access/tenants/{id}/apps)
  4. Tạo knowledge domain cho tenant  (POST /v1/ontology/{tenant_id}/domains)
  5. Định nghĩa node types (Concept, Article, FAQ)
  6. Định nghĩa rel type (RELATES_TO)
  7. Upsert search profile
  8. Seed 8 knowledge nodes
  9. Tạo 2 relations giữa nodes
  10. Chờ vector projection
  11. Semantic search
  12. Hybrid search
  13. FTS search (keyword)
  14. Cross-domain search (platform domain + tenant domain)
  15. Kiểm tra audit log

Usage:
    python3 e2e_tenant_workflow.py [--base-url URL] [--admin-key KEY] [--no-wait]

Yêu cầu:
    pip install requests python-dotenv
"""

import argparse
import json
import sys
import time
import uuid
from typing import Any, Optional

import requests

# ── Default config ─────────────────────────────────────────────────────────────
DEFAULT_BASE_URL  = "https://c14.openledger.vn/api"
PLATFORM_ADMIN_KEY = "kgsk_platform_admin"   # platform admin — quản lý tenant/app

# Tên tenant test — thêm suffix ngẫu nhiên để tránh trùng khi chạy nhiều lần
RUN_ID = str(uuid.uuid4())[:8]
TENANT_SLUG   = f"test-corp-{RUN_ID}"
TENANT_NAME   = f"Test Corp {RUN_ID}"
APP_SLUG      = f"kg-writer-{RUN_ID}"
APP_NAME      = f"KG Writer App {RUN_ID}"
DOMAIN_ID     = f"kb-{RUN_ID}"
DOMAIN_NAME   = f"Knowledge Base {RUN_ID}"

# ── ANSI colors ────────────────────────────────────────────────────────────────
GRN = "\033[32m"
YLW = "\033[33m"
RED = "\033[31m"
CYN = "\033[36m"
BLD = "\033[1m"
RST = "\033[0m"


def ok(msg: str):   print(f"  {GRN}✅{RST} {msg}")
def warn(msg: str): print(f"  {YLW}⚠️ {RST} {msg}")
def err(msg: str):  print(f"  {RED}❌{RST} {msg}")
def info(msg: str): print(f"{CYN}▶{RST}  {BLD}{msg}{RST}")
def sub(msg: str):  print(f"     {msg}")
def sep():          print(f"  {'-'*60}")


# ── HTTP helpers ───────────────────────────────────────────────────────────────

def headers(api_key: str) -> dict:
    return {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}


def req(method: str, base_url: str, path: str, body: Optional[dict], api_key: str,
        timeout: int = 20) -> tuple[int, dict]:
    url = f"{base_url}{path}"
    fn = getattr(requests, method)
    kw = {"headers": headers(api_key), "timeout": timeout}
    if body is not None:
        kw["json"] = body
    r = fn(url, **kw)
    try:
        return r.status_code, r.json()
    except Exception:
        return r.status_code, {"raw": r.text}


def GET(base_url, path, key, **kw):  return req("get",    base_url, path, None, key, **kw)
def POST(base_url, path, body, key, **kw): return req("post", base_url, path, body, key, **kw)


def assert_ok(code, body, label: str, expected=(200, 201, 202)) -> dict:
    if code in expected:
        return body
    err(f"{label} → HTTP {code}: {json.dumps(body, indent=2)[:200]}")
    sys.exit(1)


# ── Steps ──────────────────────────────────────────────────────────────────────

def step_health(base_url: str):
    info("[0] Health check")
    code, body = GET(base_url, "/healthz", PLATFORM_ADMIN_KEY)
    if code == 200:
        ok(f"Service healthy: {list(body.keys())}")
    else:
        err(f"Service unhealthy: HTTP {code}")
        sys.exit(1)
    print()


def step_create_tenant(base_url: str) -> dict:
    info("[1] Tạo tenant mới")
    code, body = POST(base_url, "/v1/tenants", {
        "slug": TENANT_SLUG,
        "name": TENANT_NAME,
        "tier": "pro",
    }, PLATFORM_ADMIN_KEY)
    tenant = assert_ok(code, body, "CreateTenant")
    ok(f"Tenant: {tenant.get('id')} | slug={tenant.get('slug')} | status={tenant.get('status')}")
    print()
    return tenant


def step_create_app(base_url: str, tenant_id: str) -> dict:
    info("[2] Tạo app cho tenant")
    code, body = POST(base_url, f"/v1/tenants/{tenant_id}/apps", {
        "slug": APP_SLUG,
        "name": APP_NAME,
        "type": "hybrid",
    }, PLATFORM_ADMIN_KEY)
    app = assert_ok(code, body, "CreateApp", expected=(200, 201))
    app_id  = app.get("id", "?")
    api_key = app.get("api_key", "")
    ok(f"App: {app_id} | slug={app.get('slug')} | type={app.get('type')}")
    if not api_key:
        warn("api_key trống — thử rotate-key")
        code2, body2 = POST(base_url, f"/v1/tenants/{tenant_id}/apps/{app_id}/rotate-key",
                            {}, PLATFORM_ADMIN_KEY)
        if code2 == 200:
            api_key = body2.get("api_key", "")
            ok(f"Rotated key: {api_key[:20]}…")
        else:
            warn(f"rotate-key HTTP {code2}: {body2}")
    else:
        ok(f"API key: {api_key[:20]}…")
    print()
    return {**app, "api_key": api_key}


def step_create_domain(base_url: str, tenant_id: str, app_key: str) -> dict:
    info("[3] Tạo knowledge domain")
    code, body = POST(base_url, f"/v1/tenants/{tenant_id}/ontology/domains", {
        "id":          DOMAIN_ID,
        "name":        DOMAIN_NAME,
        "description": "Domain kiến thức tự động tạo bởi e2e_tenant_workflow.py",
        "status":      "active",
        "visibility":  "private",
    }, PLATFORM_ADMIN_KEY)
    domain = assert_ok(code, body, "CreateDomain", expected=(200, 201))
    ok(f"Domain: {domain.get('id')} | name={domain.get('name')} | visibility={domain.get('visibility')}")
    print()
    return domain


def step_create_node_types(base_url: str, tenant_id: str, domain_id: str) -> list:
    info("[4] Định nghĩa node types")
    schemas = [
        {
            "node_type_name": "Concept",
            "graph_label":    "Concept",
            "required_props": [{"name": "concept_key", "type": "string"}],
            "optional_props": [
                {"name": "title",       "type": "string"},
                {"name": "summary",     "type": "string"},
                {"name": "tags",        "type": "string"},
                {"name": "difficulty",  "type": "string"},
            ],
        },
        {
            "node_type_name": "Article",
            "graph_label":    "Article",
            "required_props": [{"name": "article_key", "type": "string"}],
            "optional_props": [
                {"name": "title",       "type": "string"},
                {"name": "body",        "type": "string"},
                {"name": "author",      "type": "string"},
                {"name": "category",    "type": "string"},
            ],
        },
        {
            "node_type_name": "FAQ",
            "graph_label":    "FAQ",
            "required_props": [{"name": "faq_key", "type": "string"}],
            "optional_props": [
                {"name": "question",    "type": "string"},
                {"name": "answer",      "type": "string"},
                {"name": "category",    "type": "string"},
            ],
        },
    ]
    created = []
    for s in schemas:
        code, body = POST(base_url, f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types", s,
                          PLATFORM_ADMIN_KEY)
        if code in (200, 201):
            ok(f"NodeType: {body.get('node_type_name')} (id={body.get('id')})")
            created.append(body)
        elif "already exists" in str(body).lower():
            warn(f"NodeType {s['node_type_name']} already exists — OK")
        else:
            warn(f"NodeType {s['node_type_name']} HTTP {code}: {body}")
    print()
    return created


def step_create_rel_types(base_url: str, tenant_id: str, domain_id: str):
    info("[5] Định nghĩa rel types")
    rels = [
        {
            "rel_type_name": "RELATES_TO",
            "from_node_type": "Concept",
            "to_node_type":   "Concept",
            "same_domain":    True,
            "required_props": [],
            "optional_props": [{"name": "weight", "type": "float"}],
        },
        {
            "rel_type_name": "REFERENCES",
            "from_node_type": "Article",
            "to_node_type":   "Concept",
            "same_domain":    True,
            "required_props": [],
            "optional_props": [],
        },
        {
            "rel_type_name": "ANSWERS",
            "from_node_type": "FAQ",
            "to_node_type":   "Concept",
            "same_domain":    True,
            "required_props": [],
            "optional_props": [],
        },
    ]
    for r in rels:
        code, body = POST(base_url, f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types", r,
                          PLATFORM_ADMIN_KEY)
        if code in (200, 201):
            ok(f"RelType: {r['from_node_type']} --[{r['rel_type_name']}]--> {r['to_node_type']}")
        elif "already exists" in str(body).lower():
            warn(f"RelType {r['rel_type_name']} already exists — OK")
        else:
            warn(f"RelType HTTP {code}: {body}")
    print()


def step_upsert_search_profile(base_url: str, tenant_id: str, domain_id: str):
    info("[6] Upsert search profile cho domain")
    # Search profile uses PUT per router registration
    url = f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/search-profile"
    code, body = req("put", base_url, url, {
        "semantic_fields": [
            {"field_name": "title",       "weight": 1.5},
            {"field_name": "summary",     "weight": 1.2},
            {"field_name": "body",        "weight": 1.0},
            {"field_name": "question",    "weight": 1.4},
            {"field_name": "answer",      "weight": 1.2},
            {"field_name": "concept_key", "weight": 0.8, "prefix": "concept:"},
        ],
        "fts_language":       "english",
        "query_strategy_ref": "default",
    }, PLATFORM_ADMIN_KEY)
    if code in (200, 201):
        ok(f"Search profile upserted — {len(body.get('semantic_fields', []))} fields")
    else:
        warn(f"SearchProfile HTTP {code}: {body}")
    print()


def step_seed_nodes(base_url: str, tenant_id: str, domain_id: str, app_key: str) -> list:
    info("[7] Seed 8 knowledge nodes")
    nodes = [
        # Concepts
        {
            "domain_id":    domain_id,
            "node_type":    "Concept",
            "external_ref": f"concept-ml-{RUN_ID}",
            "properties": {
                "concept_key": f"machine-learning-{RUN_ID}",
                "title":    "Machine Learning — supervised unsupervised reinforcement learning neural networks training inference",
                "summary":  "Nhánh của AI cho phép máy tính học từ dữ liệu mà không lập trình tường minh",
                "tags":     "AI, ML, deep-learning",
                "difficulty": "intermediate",
            },
        },
        {
            "domain_id":    domain_id,
            "node_type":    "Concept",
            "external_ref": f"concept-nlp-{RUN_ID}",
            "properties": {
                "concept_key": f"natural-language-processing-{RUN_ID}",
                "title":    "Natural Language Processing — text classification sentiment analysis NER tokenization embedding",
                "summary":  "Xử lý ngôn ngữ tự nhiên — phân tích, hiểu và sinh văn bản bằng máy tính",
                "tags":     "NLP, text-mining, LLM",
                "difficulty": "advanced",
            },
        },
        {
            "domain_id":    domain_id,
            "node_type":    "Concept",
            "external_ref": f"concept-rag-{RUN_ID}",
            "properties": {
                "concept_key": f"retrieval-augmented-generation-{RUN_ID}",
                "title":    "Retrieval Augmented Generation — vector search knowledge base LLM grounding hallucination reduction",
                "summary":  "Kết hợp tìm kiếm vector với LLM để sinh câu trả lời có căn cứ từ knowledge base",
                "tags":     "RAG, LLM, vector-search",
                "difficulty": "advanced",
            },
        },
        {
            "domain_id":    domain_id,
            "node_type":    "Concept",
            "external_ref": f"concept-kg-{RUN_ID}",
            "properties": {
                "concept_key": f"knowledge-graph-{RUN_ID}",
                "title":    "Knowledge Graph — entities relations ontology Cypher graph database Neo4j Memgraph",
                "summary":  "Đồ thị tri thức biểu diễn entities và relationships, hỗ trợ suy luận và tìm kiếm đa chiều",
                "tags":     "graph-db, ontology, KG",
                "difficulty": "intermediate",
            },
        },
        # Articles
        {
            "domain_id":    domain_id,
            "node_type":    "Article",
            "external_ref": f"article-rag-guide-{RUN_ID}",
            "properties": {
                "article_key": f"rag-implementation-guide-{RUN_ID}",
                "title":    "RAG Implementation Guide — từng bước xây dựng hệ thống RAG với pgvector và LLM",
                "body":     "Hướng dẫn chi tiết triển khai RAG: chunking documents, embedding, vector store indexing, retrieval ranking, LLM prompting, answer grounding",
                "author":   "KG Team",
                "category": "implementation-guide",
            },
        },
        {
            "domain_id":    domain_id,
            "node_type":    "Article",
            "external_ref": f"article-kg-design-{RUN_ID}",
            "properties": {
                "article_key": f"knowledge-graph-design-patterns-{RUN_ID}",
                "title":    "Knowledge Graph Design Patterns — ontology modeling, node types, relationship schemas, validation rules",
                "body":     "Các pattern thiết kế KG: hierarchical taxonomy, property graph model, bi-temporal modeling, polyglot persistence với graph + vector store",
                "author":   "Architecture Team",
                "category": "design-patterns",
            },
        },
        # FAQs
        {
            "domain_id":    domain_id,
            "node_type":    "FAQ",
            "external_ref": f"faq-vector-search-{RUN_ID}",
            "properties": {
                "faq_key":  f"what-is-vector-search-{RUN_ID}",
                "question": "Vector search là gì và khác gì với full-text search?",
                "answer":   "Vector search tìm kiếm theo nghĩa ngữ nghĩa bằng embedding similarity (cosine, dot product). FTS tìm theo keyword match. Vector search hiệu quả với câu hỏi ngữ nghĩa, FTS hiệu quả với từ chính xác.",
                "category": "search-fundamentals",
            },
        },
        {
            "domain_id":    domain_id,
            "node_type":    "FAQ",
            "external_ref": f"faq-rag-vs-finetuning-{RUN_ID}",
            "properties": {
                "faq_key":  f"rag-vs-finetuning-{RUN_ID}",
                "question": "Khi nào dùng RAG thay vì fine-tuning LLM?",
                "answer":   "RAG phù hợp khi knowledge base thường xuyên cập nhật, cần traceability, chi phí thấp. Fine-tuning phù hợp khi cần thay đổi hành vi model, format output đặc thù, hoặc domain-specific language.",
                "category": "architecture-decisions",
            },
        },
    ]

    node_ids = []
    for i, node in enumerate(nodes, 1):
        first_prop = next(iter(node["properties"].values()), "?")
        label = first_prop[:50]
        code, body = POST(base_url, "/v1/kg/write/nodes", node, app_key)
        if code in (200, 201, 202):
            nid = body.get("node_id", body.get("id", "?"))
            node_ids.append(nid)
            ok(f"[{i:2d}/8] {node['node_type']}: {label}")
        elif "already exists" in str(body).lower():
            warn(f"[{i:2d}] Already exists — skip")
        else:
            warn(f"[{i:2d}] HTTP {code}: {body}")
        time.sleep(0.1)
    print()
    return node_ids


def step_create_relations(base_url: str, tenant_id: str, domain_id: str, node_ids: list, app_key: str):
    info("[8] Tạo relations giữa nodes")
    if len(node_ids) < 4:
        warn("Không đủ node_ids để tạo relations")
        print()
        return

    # RAG RELATES_TO NLP, RAG RELATES_TO KG
    relations = [
        {
            "domain_id":     domain_id,
            "from_node_id":  node_ids[2],  # RAG concept
            "to_node_id":    node_ids[1],  # NLP concept
            "rel_type":      "RELATES_TO",
            "properties":    {"weight": 0.85},
        },
        {
            "domain_id":     domain_id,
            "from_node_id":  node_ids[2],  # RAG concept
            "to_node_id":    node_ids[3],  # KG concept
            "rel_type":      "RELATES_TO",
            "properties":    {"weight": 0.90},
        },
    ]
    for i, rel in enumerate(relations, 1):
        code, body = POST(base_url, "/v1/kg/write/relations", rel, app_key)
        if code in (200, 201, 202):
            ok(f"[{i}] {rel['from_node_id'][:8]}… --[{rel['rel_type']}]--> {rel['to_node_id'][:8]}…")
        else:
            warn(f"[{i}] Relation HTTP {code}: {body}")
    print()


def step_wait_projection(base_url: str, app_key: str, timeout: int = 90) -> bool:
    info("[9] Waiting for vector projection")
    deadline = time.time() + timeout
    prev = None
    while time.time() < deadline:
        code, metrics = GET(base_url, "/v1/kg/metrics", app_key)
        if code == 200:
            backlog = metrics.get("kg_outbox_backlog", -1)
            vec_lag = metrics.get("kg_vector_lag_seconds", -1)
            line = f"     backlog={backlog}  vector_lag={vec_lag}s"
            if line != prev:
                print(line)
                prev = line
            if backlog == 0:
                ok("Outbox fully drained — vectors projected!")
                print()
                return True
        time.sleep(5)
    warn(f"Timeout {timeout}s — projections may still be processing")
    print()
    return False


def step_search(base_url: str, domain_id: str, app_key: str):
    info("[10] Semantic search")
    queries = [
        ("machine learning AI neural network training", 3),
        ("vector search embedding similarity retrieval", 3),
        ("RAG knowledge base LLM hallucination grounding", 3),
        ("knowledge graph ontology entities relations", 3),
        ("Khi nào dùng RAG thay vì fine-tuning", 2),
    ]
    for query, top_k in queries:
        code, body = POST(base_url, "/v1/kg/search/semantic", {
            "query":      query,
            "domain_ids": [domain_id],
            "top_k":      top_k,
        }, app_key)
        if code == 200:
            results = body.get("results", [])
            hit_info = []
            for r in results[:3]:
                props = r.get("domain_props", {})
                title = (props.get("title") or props.get("question") or
                         props.get("concept_key") or props.get("article_key") or
                         props.get("faq_key") or "?")[:55]
                hit_info.append(f"[{r.get('score', 0):.3f}] {r.get('node_type')}: {title}")
            ok(f"'{query[:42]}…' → {len(results)} hit(s)")
            for h in hit_info:
                sub(f"• {h}")
        else:
            err(f"Semantic HTTP {code}: {body.get('error', body)}")
    print()


def step_hybrid_search(base_url: str, domain_id: str, app_key: str):
    info("[11] Hybrid search")
    code, body = POST(base_url, "/v1/kg/search/hybrid", {
        "query":          "retrieval augmented generation vector embedding knowledge base",
        "domain_ids":     [domain_id],
        "top_k":          5,
        "semantic_weight": 0.7,
    }, app_key)
    if code == 200:
        results = body.get("results", [])
        ok(f"Hybrid search → {len(results)} result(s) in {body.get('search_time_ms', '?')}ms")
        for r in results:
            props = r.get("domain_props", {})
            title = (props.get("title") or props.get("question") or
                     props.get("concept_key") or "?")[:60]
            sub(f"  [{r.get('score', 0):.4f}] {r.get('node_type')}: {title}")
    elif code == 404:
        warn("Hybrid endpoint not available")
    else:
        err(f"Hybrid HTTP {code}: {body.get('error', body)}")
    print()


def step_cross_domain_search(base_url: str, domain_id: str, app_key: str):
    info("[12] Cross-domain search (platform sample-policy + tenant domain)")
    code, body = POST(base_url, "/v1/kg/search/semantic", {
        "query":      "authentication security policy compliance",
        "domain_ids": ["sample-policy", domain_id],
        "top_k":      4,
    }, PLATFORM_ADMIN_KEY)
    if code == 200:
        results = body.get("results", [])
        ok(f"Cross-domain → {len(results)} result(s)")
        for r in results:
            props = r.get("domain_props", {})
            title = (props.get("title") or props.get("question") or
                     props.get("concept_key") or props.get("guide_key") or "?")[:55]
            sub(f"  [{r.get('score', 0):.3f}] [{r.get('domain_id')}] {r.get('node_type')}: {title}")
    else:
        err(f"Cross-domain HTTP {code}: {body.get('error', body)}")
    print()


def step_read_graph(base_url: str, tenant_id: str, domain_id: str, node_ids: list, app_key: str):
    info("[13] Kiểm tra graph (read node + relations)")
    if not node_ids:
        warn("Không có node_id để đọc")
        return

    # Read first node
    nid = node_ids[0]
    code, body = GET(base_url, f"/v1/kg/read/nodes/{nid}", app_key)
    if code == 200:
        props = body.get("domain_props", {})
        ok(f"Node {nid[:8]}… → {body.get('node_type')}: {list(props.keys())[:4]}")
    else:
        warn(f"Read node HTTP {code}: {body}")

    # List node types of domain
    code2, body2 = GET(base_url, f"/v1/kg/read/node-types?domain_id={domain_id}", app_key)
    if code2 == 200:
        types = body2.get("data", body2) if isinstance(body2, dict) else body2
        ok(f"Node types in domain: {[t.get('node_type_name') for t in types] if isinstance(types, list) else types}")
    else:
        warn(f"List node-types HTTP {code2}: {body2}")
    print()


def step_audit_log(base_url: str, tenant_id: str):
    info("[14] Kiểm tra audit log")
    code, body = GET(base_url, f"/v1/access/audit?resource_owner_tenant_id={tenant_id}", PLATFORM_ADMIN_KEY)
    if code == 200:
        entries = body.get("data", [])
        ok(f"Audit entries: {len(entries)}")
        for e in entries[:5]:
            sub(f"  [{e.get('outcome')}] {e.get('action')} {e.get('resource_type')}/{e.get('resource_id', '')[:12]}")
    else:
        warn(f"Audit HTTP {code}: {body}")
    print()


# ── Main ───────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(description="KG Service E2E: tenant → domain → search")
    parser.add_argument("--base-url",  default=DEFAULT_BASE_URL)
    parser.add_argument("--no-wait",   action="store_true", help="Bỏ qua chờ vector projection")
    parser.add_argument("--no-search", action="store_true", help="Bỏ qua bước search")
    args = parser.parse_args()

    B = args.base_url

    print()
    print(f"{'═'*64}")
    print(f"  {BLD}KG Service — E2E Tenant Workflow Test{RST}")
    print(f"  Target : {B}")
    print(f"  Run ID : {RUN_ID}")
    print(f"{'═'*64}")
    print()

    step_health(B)
    tenant = step_create_tenant(B)
    tenant_id = tenant["id"]

    app = step_create_app(B, tenant_id)
    app_id  = app["id"]
    app_key = app.get("api_key") or PLATFORM_ADMIN_KEY

    step_create_domain(B, tenant_id, app_key)
    step_create_node_types(B, tenant_id, DOMAIN_ID)
    step_create_rel_types(B, tenant_id, DOMAIN_ID)
    step_upsert_search_profile(B, tenant_id, DOMAIN_ID)

    node_ids = step_seed_nodes(B, tenant_id, DOMAIN_ID, app_key)
    step_create_relations(B, tenant_id, DOMAIN_ID, node_ids, app_key)

    if not args.no_wait:
        step_wait_projection(B, app_key)

    if not args.no_search:
        step_search(B, DOMAIN_ID, app_key)
        step_hybrid_search(B, DOMAIN_ID, app_key)
        step_cross_domain_search(B, DOMAIN_ID, app_key)

    step_read_graph(B, tenant_id, DOMAIN_ID, node_ids, app_key)
    step_audit_log(B, tenant_id)

    print(f"{'═'*64}")
    print(f"  {GRN}{BLD}E2E hoàn thành!{RST}")
    print(f"  Tenant ID  : {tenant_id}")
    print(f"  App ID     : {app_id}")
    print(f"  Domain ID  : {DOMAIN_ID}")
    print(f"  Nodes seed : {len(node_ids)}/8")
    print(f"{'═'*64}")
    print()


if __name__ == "__main__":
    main()
