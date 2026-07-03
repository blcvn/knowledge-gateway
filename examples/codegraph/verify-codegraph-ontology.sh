#!/usr/bin/env bash

set -euo pipefail

python3 - "$@" <<'PY'
import json
import os
import time
import sys
import urllib.error
import urllib.request
from typing import Any, Dict, List, Optional, Tuple

service_url = os.environ.get("KG_SERVICE_URL") or os.environ.get("KG_BASE_URL")
api_key = os.environ.get("KG_API_KEY")
domain_id = os.environ.get("KG_DOMAIN_ID", "code-graph")
strategy_key = os.environ.get("KG_QUERY_STRATEGY_KEY", "code-graph-traversal")

if not service_url:
    print("KG_SERVICE_URL or KG_BASE_URL is required", file=sys.stderr)
    raise SystemExit(1)
if not api_key:
    print("KG_API_KEY is required", file=sys.stderr)
    raise SystemExit(1)

service_url = service_url.rstrip("/")
max_retries = int(os.environ.get("KG_BOOTSTRAP_MAX_RETRIES", "6"))
retry_delay_s = float(os.environ.get("KG_BOOTSTRAP_RETRY_DELAY_S", "1.0"))

node_types = [
    {
        "node_type_name": "Function",
        "graph_label": "Function",
        "required_props": [
            {"name": "name", "type": "string"},
            {"name": "kind", "type": "string"},
            {"name": "file", "type": "string"},
            {"name": "line", "type": "number"},
            {"name": "language", "type": "string"},
            {"name": "project_id", "type": "string"},
        ],
        "optional_props": [
            {"name": "signature", "type": "string"},
            {"name": "docstring", "type": "string"},
            {"name": "package", "type": "string"},
            {"name": "commit_sha", "type": "string"},
            {"name": "external_ref", "type": "string"},
        ],
    },
    {
        "node_type_name": "Method",
        "graph_label": "Method",
        "required_props": [
            {"name": "name", "type": "string"},
            {"name": "kind", "type": "string"},
            {"name": "file", "type": "string"},
            {"name": "line", "type": "number"},
            {"name": "language", "type": "string"},
            {"name": "project_id", "type": "string"},
        ],
        "optional_props": [
            {"name": "signature", "type": "string"},
            {"name": "docstring", "type": "string"},
            {"name": "package", "type": "string"},
            {"name": "commit_sha", "type": "string"},
            {"name": "external_ref", "type": "string"},
        ],
    },
    {
        "node_type_name": "Struct",
        "graph_label": "Struct",
        "required_props": [
            {"name": "name", "type": "string"},
            {"name": "kind", "type": "string"},
            {"name": "file", "type": "string"},
            {"name": "line", "type": "number"},
            {"name": "language", "type": "string"},
            {"name": "project_id", "type": "string"},
        ],
        "optional_props": [
            {"name": "signature", "type": "string"},
            {"name": "docstring", "type": "string"},
            {"name": "package", "type": "string"},
            {"name": "commit_sha", "type": "string"},
            {"name": "external_ref", "type": "string"},
        ],
    },
    {
        "node_type_name": "Interface",
        "graph_label": "Interface",
        "required_props": [
            {"name": "name", "type": "string"},
            {"name": "kind", "type": "string"},
            {"name": "file", "type": "string"},
            {"name": "line", "type": "number"},
            {"name": "language", "type": "string"},
            {"name": "project_id", "type": "string"},
        ],
        "optional_props": [
            {"name": "signature", "type": "string"},
            {"name": "docstring", "type": "string"},
            {"name": "package", "type": "string"},
            {"name": "commit_sha", "type": "string"},
            {"name": "external_ref", "type": "string"},
        ],
    },
    {
        "node_type_name": "Package",
        "graph_label": "Package",
        "required_props": [
            {"name": "name", "type": "string"},
            {"name": "kind", "type": "string"},
            {"name": "file", "type": "string"},
            {"name": "line", "type": "number"},
            {"name": "language", "type": "string"},
            {"name": "project_id", "type": "string"},
        ],
        "optional_props": [
            {"name": "signature", "type": "string"},
            {"name": "docstring", "type": "string"},
            {"name": "package", "type": "string"},
            {"name": "commit_sha", "type": "string"},
            {"name": "external_ref", "type": "string"},
        ],
    },
    {
        "node_type_name": "File",
        "graph_label": "File",
        "required_props": [
            {"name": "name", "type": "string"},
            {"name": "kind", "type": "string"},
            {"name": "file", "type": "string"},
            {"name": "line", "type": "number"},
            {"name": "language", "type": "string"},
            {"name": "project_id", "type": "string"},
        ],
        "optional_props": [
            {"name": "signature", "type": "string"},
            {"name": "docstring", "type": "string"},
            {"name": "package", "type": "string"},
            {"name": "commit_sha", "type": "string"},
            {"name": "external_ref", "type": "string"},
        ],
    },
]

rel_pairs = {
    "CALLS": [
        ("Function", "Function"),
        ("Function", "Method"),
        ("Function", "Interface"),
        ("Method", "Function"),
        ("Method", "Method"),
    ],
    "IMPLEMENTS": [
        ("Struct", "Interface"),
    ],
    "CONTAINS": [
        ("File", "Function"),
        ("File", "Method"),
        ("File", "Struct"),
        ("File", "Interface"),
        ("File", "Package"),
        ("Package", "File"),
        ("Struct", "Method"),
        ("Interface", "Method"),
    ],
    "REFERENCES": [
        (from_type, to_type)
        for from_type in ("Function", "Method", "Struct", "Interface", "Package", "File")
        for to_type in ("Function", "Method", "Struct", "Interface", "Package", "File")
    ],
    "IMPORTS": [
        ("File", "Package"),
        ("Package", "Package"),
    ],
}

query_strategy = {
    "key": strategy_key,
    "max_depth": 5,
    "params": {
        "direction": "out",
        "depth_mode": "fixed",
        "acl_predicate": "any_hop",
    },
}

search_profile = {
    "semantic_fields": [
        {"field_name": "name", "weight": 2.0},
        {"field_name": "signature", "weight": 1.5},
        {"field_name": "docstring", "weight": 1.0},
        {"field_name": "file", "weight": 1.0},
        {"field_name": "package", "weight": 1.0},
        {"field_name": "language", "weight": 0.75},
        {"field_name": "project_id", "weight": 0.75},
        {"field_name": "kind", "weight": 0.75},
        {"field_name": "external_ref", "weight": 0.75},
    ],
    "fts_language": "simple",
    "query_strategy_ref": strategy_key,
}

templates = {
    "code_callers": {
        "template_name": "code_callers",
        "pattern_spec": {
            "start": {
                "node_type": "Function",
                "match": {
                    "project_id": "$project_id",
                    "file": "$file",
                    "line": "$line",
                    "name": "$name",
                },
            },
            "hops": [
                {
                    "rel_type": "CALLS",
                    "direction": "in",
                    "to_node_type": "Function",
                }
            ],
        },
        "param_schema": [
            {"name": "project_id", "type": "string", "required": True},
            {"name": "file", "type": "string", "required": True},
            {"name": "line", "type": "number", "required": True},
            {"name": "name", "type": "string", "required": True},
        ],
        "return_fields": [
            "name",
            "kind",
            "file",
            "line",
            "language",
            "package",
            "signature",
            "docstring",
            "project_id",
            "commit_sha",
        ],
        "description": "Return direct callers of a code symbol.",
    },
    "code_callees": {
        "template_name": "code_callees",
        "pattern_spec": {
            "start": {
                "node_type": "Function",
                "match": {
                    "project_id": "$project_id",
                    "file": "$file",
                    "line": "$line",
                    "name": "$name",
                },
            },
            "hops": [
                {
                    "rel_type": "CALLS",
                    "direction": "out",
                    "to_node_type": "Function",
                }
            ],
        },
        "param_schema": [
            {"name": "project_id", "type": "string", "required": True},
            {"name": "file", "type": "string", "required": True},
            {"name": "line", "type": "number", "required": True},
            {"name": "name", "type": "string", "required": True},
        ],
        "return_fields": [
            "name",
            "kind",
            "file",
            "line",
            "language",
            "package",
            "signature",
            "docstring",
            "project_id",
            "commit_sha",
        ],
        "description": "Return direct callees of a code symbol.",
    },
    "code_impact": {
        "template_name": "code_impact",
        "pattern_spec": {
            "start": {
                "node_type": "Function",
                "match": {
                    "project_id": "$project_id",
                    "file": "$file",
                    "line": "$line",
                    "name": "$name",
                },
            },
            "hops": [
                {
                    "rel_type": "CALLS",
                    "direction": "out",
                    "to_node_type": "Function",
                },
                {
                    "rel_type": "CALLS",
                    "direction": "out",
                    "to_node_type": "Function",
                },
            ],
        },
        "param_schema": [
            {"name": "project_id", "type": "string", "required": True},
            {"name": "file", "type": "string", "required": True},
            {"name": "line", "type": "number", "required": True},
            {"name": "name", "type": "string", "required": True},
        ],
        "return_fields": [
            "name",
            "kind",
            "file",
            "line",
            "language",
            "package",
            "signature",
            "docstring",
            "project_id",
            "commit_sha",
        ],
        "description": "Return a two-hop downstream impact chain for a code symbol.",
    },
    "code_implements": {
        "template_name": "code_implements",
        "pattern_spec": {
            "start": {
                "node_type": "Interface",
                "match": {
                    "project_id": "$project_id",
                    "file": "$file",
                    "line": "$line",
                    "name": "$name",
                },
            },
            "hops": [
                {
                    "rel_type": "IMPLEMENTS",
                    "direction": "in",
                    "to_node_type": "Struct",
                }
            ],
        },
        "param_schema": [
            {"name": "project_id", "type": "string", "required": True},
            {"name": "file", "type": "string", "required": True},
            {"name": "line", "type": "number", "required": True},
            {"name": "name", "type": "string", "required": True},
        ],
        "return_fields": [
            "name",
            "kind",
            "file",
            "line",
            "language",
            "package",
            "signature",
            "docstring",
            "project_id",
            "commit_sha",
        ],
        "description": "Return structs that implement a code interface.",
    },
}


def log(message: str) -> None:
    print(message)


def request(method: str, path: str, body: Optional[Any] = None) -> Tuple[int, bytes]:
    data = None if body is None else json.dumps(body).encode("utf-8")
    req = urllib.request.Request(service_url + path, data=data, method=method)
    req.add_header("Authorization", "Bearer " + api_key)
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, resp.read()
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read()


def request_with_retry(method: str, path: str, body: Optional[Any] = None) -> Tuple[int, bytes]:
    attempts = 0
    while True:
        status, raw = request(method, path, body)
        if status != 429 or attempts >= max_retries:
            return status, raw
        time.sleep(retry_delay_s * (2 ** attempts))
        attempts += 1


def get_json(path: str) -> Any:
    status, raw = request_with_retry("GET", path)
    if status < 200 or status >= 300:
        raise RuntimeError("GET %s failed with status %s: %s" % (path, status, raw.decode("utf-8", "replace").strip()))
    if not raw:
        return None
    return json.loads(raw.decode("utf-8"))


def list_to_map(items: List[Dict[str, Any]], key: str) -> Dict[str, Dict[str, Any]]:
    out: Dict[str, Dict[str, Any]] = {}
    for item in items:
        value = str(item.get(key, ""))
        if value:
            out[value] = item
    return out


def list_or_empty(value: Optional[Any]) -> List[Dict[str, Any]]:
    if isinstance(value, list):
        return value
    return []


def assert_equal(label: str, got: Any, want: Any) -> None:
    if got != want:
        raise RuntimeError("%s mismatch: got=%r want=%r" % (label, got, want))


def verify_domain(details: Dict[str, Any]) -> None:
    domain = details.get("domain") or {}
    assert_equal("domain.id", domain.get("id"), domain_id)
    assert_equal("domain.status", domain.get("status"), "active")
    assert_equal("domain.visibility", domain.get("visibility"), "private")

    node_map = list_to_map(list_or_empty(details.get("node_types")), "node_type_name")
    for schema in node_types:
        current = node_map.get(schema["node_type_name"])
        if current is None:
            raise RuntimeError("missing node type %s" % schema["node_type_name"])
        assert_equal(
            "node type %s" % schema["node_type_name"],
            {
                "node_type_name": current.get("node_type_name"),
                "graph_label": current.get("graph_label"),
                "required_props": current.get("required_props", []),
                "optional_props": current.get("optional_props", []),
            },
            schema,
        )

    rel_map = {
        (item.get("rel_type_name"), item.get("from_node_type"), item.get("to_node_type")): item
        for item in list_or_empty(details.get("rel_types"))
    }
    expected_rel_count = sum(len(pairs) for pairs in rel_pairs.values())
    assert_equal("relationship type count", len(rel_map), expected_rel_count)
    for rel_type_name, pairs in rel_pairs.items():
        for from_node_type, to_node_type in pairs:
            if (rel_type_name, from_node_type, to_node_type) not in rel_map:
                raise RuntimeError("missing relationship type %s %s->%s" % (rel_type_name, from_node_type, to_node_type))

    profile = domain.get("search_profile")
    if profile is None:
        raise RuntimeError("missing search profile")
    assert_equal(
        "search profile",
        {
            "semantic_fields": profile.get("semantic_fields", []),
            "fts_language": profile.get("fts_language"),
            "query_strategy_ref": profile.get("query_strategy_ref"),
        },
        search_profile,
    )

    strategies = get_json("/v1/ontology/query-strategies")
    if not isinstance(strategies, list):
        raise RuntimeError("query strategies response is not an array")
    strategy = next((item for item in strategies if item.get("key") == strategy_key), None)
    if strategy is None:
        raise RuntimeError("missing query strategy %s" % strategy_key)
    assert_equal(
        "query strategy",
        {
            "key": strategy.get("key"),
            "max_depth": strategy.get("max_depth"),
            "params": strategy.get("params", {}),
        },
        query_strategy,
    )

    query_templates = list_to_map(list_or_empty(details.get("query_templates")), "template_name")
    template_list = get_json("/v1/kg/read/templates?domain_id=%s" % domain_id)
    if not isinstance(template_list, dict):
        raise RuntimeError("template list response is not an object")
    active_templates = list_to_map(template_list.get("data", []), "template_name")
    assert_equal("active template count", set(active_templates.keys()), set(templates.keys()))

    for name, template in templates.items():
        current = query_templates.get(name)
        if current is None:
            raise RuntimeError("missing query template %s" % name)
        assert_equal(
            "template %s" % name,
            {
                "template_name": current.get("template_name"),
                "pattern_spec": current.get("pattern_spec", {}),
                "param_schema": current.get("param_schema", []),
                "return_fields": current.get("return_fields", []),
                "description": current.get("description", ""),
                "status": current.get("status"),
            },
            {
                "template_name": template["template_name"],
                "pattern_spec": template["pattern_spec"],
                "param_schema": template["param_schema"],
                "return_fields": template["return_fields"],
                "description": template["description"],
                "status": "active",
            },
        )


def main() -> None:
    log("verifying ontology domain %s at %s" % (domain_id, service_url))
    details = get_json("/v1/ontology/domains/%s" % domain_id)
    if details is None:
        raise RuntimeError("domain %s was not found" % domain_id)
    verify_domain(details)
    log("verification passed for domain, node types, relationship types, search profile, query strategy, and active templates")


if __name__ == "__main__":
    main()
PY
