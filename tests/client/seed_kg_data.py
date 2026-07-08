#!/usr/bin/env python3
"""
seed_kg_data.py — Seed sample-policy domain data vào KG Service
Tạo nodes dùng node types đã được seed sẵn lúc startup:
  Topic (required: topic_key), ActionGuide (required: guide_key),
  Obligation (required: obligation_key), ReferenceDoc (required: reference_key)

Usage:
    python3 seed_kg_data.py [--base-url URL] [--api-key KEY]

Default target: https://c14.openledger.vn/api
"""

import argparse
import json
import sys
import time
import requests

# ── Config ────────────────────────────────────────────────────────────────────
DEFAULT_BASE_URL = "https://c14.openledger.vn/api"
DEFAULT_API_KEY  = "kgsk_seed_writer"  # ingestion_producer on platform tenant — valid UUID app_id

# Node types seeded at startup — safe to use without bridge_reference_ids:
#   ActionGuide  (required: guide_key,       optional: title)
#   Obligation   (required: obligation_key,  optional: summary)
#   ReferenceDoc (required: reference_key,   optional: title)
#   Schedule     (required: schedule_key,    optional: effective_on)
#   Record       (required: record_key,      optional: title)
# NOTE: Topic requires ATTACHES bridge_reference_ids (cross-domain rule) — skip for simple seeding
SAMPLE_NODES = [
    # ── ActionGuides ──────────────────────────────────────────────────────────
    {
        "domain_id":    "sample-policy",
        "node_type":    "ActionGuide",
        "external_ref": "guide-firewall-config",
        "properties": {
            "guide_key": "firewall-traffic-routing-config",
            "title":     "Firewall Traffic Routing Configuration — rules for routing network traffic through enterprise firewall, allowed ports, protocols, IP ranges inbound outbound connections",
        },
    },
    {
        "domain_id":    "sample-policy",
        "node_type":    "ActionGuide",
        "external_ref": "guide-encryption-at-rest",
        "properties": {
            "guide_key": "data-encryption-at-rest",
            "title":     "Data Encryption at Rest Implementation — AES-256 encryption for sensitive data on disk, databases, file shares, backup media, portable storage, PII and confidential data protection",
        },
    },
    {
        "domain_id":    "sample-policy",
        "node_type":    "ActionGuide",
        "external_ref": "guide-zero-trust",
        "properties": {
            "guide_key": "zero-trust-network-access",
            "title":     "Zero Trust Network Access Implementation — no implicit trust for any user or device, all access requests authenticated authorized continuously validated micro-segmentation least-privilege",
        },
    },
    {
        "domain_id":    "sample-policy",
        "node_type":    "ActionGuide",
        "external_ref": "guide-incident-response",
        "properties": {
            "guide_key": "incident-response-escalation",
            "title":     "Incident Response and Escalation Runbook — detecting reporting triaging security incidents, escalation paths, SLA targets per severity, communication templates, post-incident review",
        },
    },
    {
        "domain_id":    "sample-policy",
        "node_type":    "ActionGuide",
        "external_ref": "guide-vuln-patching",
        "properties": {
            "guide_key": "vulnerability-management-patching",
            "title":     "Vulnerability Management and Patching Guide — weekly vulnerability scanning, critical CVE CVSS patch within 48 hours, high severity 7 days, monthly compliance reports security committee",
        },
    },
    {
        "domain_id":    "sample-policy",
        "node_type":    "ActionGuide",
        "external_ref": "guide-iam-provisioning",
        "properties": {
            "guide_key": "iam-provisioning-runbook",
            "title":     "IAM Provisioning Runbook — user provisioning deprovisioning RBAC role assignment MFA enrollment privileged access management PAM periodic access reviews corporate systems",
        },
    },
    {
        "domain_id":    "sample-policy",
        "node_type":    "ActionGuide",
        "external_ref": "guide-cloud-hardening",
        "properties": {
            "guide_key": "cloud-security-hardening",
            "title":     "Cloud Security Hardening Guide — S3 bucket policies VM security groups container registry scanning managed service configuration CIS Benchmarks v2.0 for AWS GCP Azure cloud infrastructure",
        },
    },
    # ── Obligations ───────────────────────────────────────────────────────────
    {
        "domain_id":    "sample-policy",
        "node_type":    "Obligation",
        "external_ref": "obligation-mfa-admin",
        "properties": {
            "obligation_key": "mfa-enforcement-admin-console",
            "summary":        "All administrative console logins must complete MFA using TOTP or hardware security key. Non-compliance triggers automated account lock. Applies to AWS GCP Azure on-prem admin portals.",
        },
    },
    {
        "domain_id":    "sample-policy",
        "node_type":    "Obligation",
        "external_ref": "obligation-aup",
        "properties": {
            "obligation_key": "acceptable-use-compliance",
            "summary":        "All employees must comply with Acceptable Use Policy. Prohibited: unauthorized software installation, personal data on corporate systems, social media misuse during work hours, data exfiltration.",
        },
    },
    # ── ReferenceDocs ─────────────────────────────────────────────────────────
    {
        "domain_id":    "sample-policy",
        "node_type":    "ReferenceDoc",
        "external_ref": "ref-nist-800-53",
        "properties": {
            "reference_key": "nist-sp-800-53-rev5",
            "title":         "NIST SP 800-53 Rev 5 — Security and Privacy Controls for Information Systems, authoritative control framework for security policy mappings and compliance assessments",
        },
    },
]


def make_headers(api_key: str) -> dict:
    return {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    }


def post(base_url: str, path: str, body: dict, headers: dict):
    r = requests.post(f"{base_url}{path}", headers=headers, json=body, timeout=15)
    try:
        return r.status_code, r.json()
    except Exception:
        return r.status_code, {"raw": r.text}


def get(base_url: str, path: str, headers: dict):
    r = requests.get(f"{base_url}{path}", headers=headers, timeout=15)
    try:
        return r.status_code, r.json()
    except Exception:
        return r.status_code, {"raw": r.text}


def ok(msg):   print(f"  \033[32m✅\033[0m {msg}")
def warn(msg): print(f"  \033[33m⚠️\033[0m  {msg}")
def err(msg):  print(f"  \033[31m❌\033[0m {msg}")
def info(msg): print(f"\033[36m▶\033[0m  {msg}")


def step_health(base_url: str):
    info("[0] Health check")
    r = requests.get(f"{base_url}/healthz", timeout=10)
    if r.status_code == 200:
        ok(f"Service healthy: {list(r.json().keys())}")
    else:
        err(f"Service unhealthy: HTTP {r.status_code}")
        sys.exit(1)


def step_verify_domain(base_url: str, api_key: str):
    """sample-policy được tự động seed lúc startup — chỉ kiểm tra."""
    info("[1] Verify domain 'sample-policy' (auto-seeded at startup)")
    h = make_headers(api_key)
    code, body = get(base_url, f"/v1/kg/read/templates?domain_id=sample-policy", h)
    if code == 200:
        ok(f"Domain accessible — {len(body.get('data', []))} query templates found")
    else:
        warn(f"Domain check HTTP {code}: {body}")


def step_seed_nodes(base_url: str, api_key: str) -> list:
    info(f"[2] Seeding {len(SAMPLE_NODES)} nodes via POST /v1/kg/write/nodes")
    h = make_headers(api_key)
    node_ids = []
    for i, node in enumerate(SAMPLE_NODES, 1):
        props = node.get("properties", {})
        label_val = next(iter(props.values()), "?")[:55]
        code, body = post(base_url, "/v1/kg/write/nodes", node, h)
        if code in (200, 201, 202):  # 202 = accepted/processing
            nid = body.get("node_id", body.get("id", "?"))
            node_ids.append(nid)
            ok(f"[{i:2d}/{len(SAMPLE_NODES)}] {node['node_type']}: {label_val}")
        elif "already exists" in str(body).lower() or "external_ref" in str(body).lower():
            warn(f"[{i:2d}] Already exists — skipping ({node['node_type']})")
        else:
            warn(f"[{i:2d}] FAILED HTTP {code}: {body}")
        time.sleep(0.1)
    return node_ids


def step_wait_for_projection(base_url: str, api_key: str, timeout: int = 90):
    info("[3] Waiting for outbox worker → vector store projection")
    h = make_headers(api_key)
    deadline = time.time() + timeout
    last_backlog = None
    while time.time() < deadline:
        code, metrics = get(base_url, "/v1/kg/metrics", h)
        if code == 200:
            backlog = metrics.get("kg_outbox_backlog", -1)
            vec_lag = metrics.get("kg_vector_lag_seconds", -1)
            if backlog != last_backlog:
                print(f"     outbox_backlog={backlog}  vector_lag={vec_lag}s")
                last_backlog = backlog
            if backlog == 0:
                ok("Outbox fully drained — vectors projected!")
                return True
        time.sleep(3)
    warn(f"Timeout after {timeout}s — outbox may still be processing")
    return False


def step_verify_search(base_url: str, api_key: str):
    info("[4] Verifying search endpoints")
    h = make_headers(api_key)

    semantic_queries = [
        "Firewall routing rules and network traffic",
        "data encryption compliance AES",
        "zero trust authentication micro-segmentation",
        "incident response security operations",
    ]
    print()
    for query in semantic_queries:
        code, body = post(base_url, "/v1/kg/search/semantic", {
            "query":      query,
            "domain_ids": ["sample-policy"],
            "top_k":      3,
        }, h)
        if code == 200:
            hits = body.get("data", [])
            ok(f"Semantic: '{query[:45]}' → {len(hits)} result(s)")
            for hit in hits[:2]:
                props = hit.get("properties", {})
                title = props.get("title") or props.get("summary") or props.get("topic_key", "?")
                score = hit.get("score", 0)
                print(f"        • [{score:.3f}] {title[:65]}")
        else:
            err(f"Semantic '{query[:40]}' HTTP {code}: {body.get('error', body)}")

    print()
    # Hybrid
    code, body = post(base_url, "/v1/kg/search/hybrid", {
        "query":      "encryption compliance policy",
        "domain_ids": ["sample-policy"],
        "top_k":      3,
    }, h)
    if code == 200:
        ok(f"Hybrid search → {len(body.get('data', []))} result(s)")
    elif code == 404:
        warn("Hybrid endpoint not available on this build")
    else:
        err(f"Hybrid HTTP {code}: {body.get('error', body)}")


def main():
    parser = argparse.ArgumentParser(description="Seed KG Service with sample-policy data")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL)
    parser.add_argument("--api-key",  default=DEFAULT_API_KEY)
    parser.add_argument("--no-wait",  action="store_true", help="Skip waiting for vector projection")
    args = parser.parse_args()

    print()
    print("╔══════════════════════════════════════════════════════════════╗")
    print("║  KG Service — seed_kg_data.py                              ║")
    print(f"║  Target : {args.base_url:<50} ║")
    print(f"║  Domain : sample-policy (10 nodes)                         ║")
    print("╚══════════════════════════════════════════════════════════════╝")
    print()

    step_health(args.base_url)
    print()
    step_verify_domain(args.base_url, args.api_key)
    print()
    node_ids = step_seed_nodes(args.base_url, args.api_key)
    print()

    if not args.no_wait:
        projected = step_wait_for_projection(args.base_url, args.api_key)
    else:
        projected = False
        warn("Skipped projection wait (--no-wait)")

    print()
    step_verify_search(args.base_url, args.api_key)

    print()
    print("══════════════════════════════════════════════════════════════")
    print(f"  Nodes seeded : {len(node_ids)}")
    print(f"  Projected    : {'yes' if projected else 'pending (check metrics)'}")
    print("══════════════════════════════════════════════════════════════")
    print()


if __name__ == "__main__":
    main()
