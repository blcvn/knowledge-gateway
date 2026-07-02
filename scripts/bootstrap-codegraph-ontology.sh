#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export CODEGRAPH_REPO_ROOT="${repo_root}"

python3 - "$@" <<'PY'
import json
import hashlib
import os
import subprocess
import time
import sys
import urllib.error
import urllib.request
from typing import Any, Optional

service_url = os.environ.get("KG_SERVICE_URL") or os.environ.get("KG_BASE_URL")
api_key = os.environ.get("KG_API_KEY")
tenant_id = os.environ.get("KG_TENANT_ID", "00000000-0000-0000-0000-000000000000")
tenant_slug = os.environ.get("KG_TENANT_SLUG", "codegraph-runtime")
tenant_name = os.environ.get("KG_TENANT_NAME", "CodeGraph Runtime Tenant")
tenant_tier = os.environ.get("KG_TENANT_TIER", "pro")
app_slug = os.environ.get("KG_APP_SLUG", "codegraph-runtime-admin")
app_name = os.environ.get("KG_APP_NAME", "CodeGraph Runtime Admin")
app_type = os.environ.get("KG_APP_TYPE", "admin_tool")
app_id = os.environ.get("KG_RUNTIME_APP_ID") or os.environ.get("KG_APP_ID")
domain_id = os.environ.get("KG_DOMAIN_ID", "code-graph")
domain_name = os.environ.get("KG_DOMAIN_NAME", "CodeGraph")
domain_description = os.environ.get(
    "KG_DOMAIN_DESCRIPTION",
    "Bootstrap ontology for source-code symbols and traversal templates.",
)
domain_visibility = os.environ.get("KG_DOMAIN_VISIBILITY", "private")
strategy_key = os.environ.get("KG_QUERY_STRATEGY_KEY", "code-graph-traversal")

if not service_url:
    print("KG_SERVICE_URL or KG_BASE_URL is required", file=sys.stderr)
    raise SystemExit(1)
if not api_key:
    print("KG_API_KEY is required", file=sys.stderr)
    raise SystemExit(1)
if not app_id:
    print("KG_RUNTIME_APP_ID or KG_APP_ID is required", file=sys.stderr)
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
        "validation_rules": [],
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
        "validation_rules": [],
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
        "validation_rules": [],
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
        "validation_rules": [],
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
        "validation_rules": [],
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
        "validation_rules": [],
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
    "version": 1,
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

templates = [
    {
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
    {
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
    {
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
    {
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
]


def log(message: str) -> None:
    print(message)


def request(method: str, path: str, body: Optional[Any] = None) -> tuple[int, dict[str, str], bytes]:
    data = None if body is None else json.dumps(body).encode("utf-8")
    req = urllib.request.Request(
        service_url + path,
        data=data,
        method=method,
    )
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, dict(resp.headers.items()), resp.read()
    except urllib.error.HTTPError as exc:
        return exc.code, dict(exc.headers.items()), exc.read()


def request_with_retry(method: str, path: str, body: Optional[Any] = None) -> tuple[int, dict[str, str], bytes]:
    attempts = 0
    while True:
        status, headers, raw = request(method, path, body)
        if status != 429 or attempts >= max_retries:
            return status, headers, raw
        retry_after = headers.get("Retry-After", "")
        delay = retry_delay_s * (2 ** attempts)
        try:
            if retry_after:
                delay = max(delay, float(retry_after))
        except ValueError:
            pass
        time.sleep(delay)
        attempts += 1


def get_json(path: str) -> Any:
    status, _, raw = request_with_retry("GET", path)
    if status == 404:
        return None
    if status < 200 or status >= 300:
        raise RuntimeError(f"GET {path} failed with status {status}: {raw.decode('utf-8', 'replace').strip()}")
    if not raw:
        return None
    return json.loads(raw.decode("utf-8"))


def post_json(path: str, body: Any) -> Any:
    status, _, raw = request_with_retry("POST", path, body)
    if status < 200 or status >= 300:
        raise RuntimeError(f"POST {path} failed with status {status}: {raw.decode('utf-8', 'replace').strip()}")
    return json.loads(raw.decode("utf-8"))


def put_json(path: str, body: Any) -> Any:
    status, _, raw = request_with_retry("PUT", path, body)
    if status < 200 or status >= 300:
        raise RuntimeError(f"PUT {path} failed with status {status}: {raw.decode('utf-8', 'replace').strip()}")
    return json.loads(raw.decode("utf-8"))


def find_by(items: list[dict[str, Any]], key: str, value: str) -> Optional[dict[str, Any]]:
    for item in items:
        if str(item.get(key, "")) == value:
            return item
    return None


def list_or_empty(value: Optional[Any]) -> list[dict[str, Any]]:
    if isinstance(value, list):
        return value
    return []


def ensure_domain() -> dict[str, Any]:
    log(f"==> ensuring domain {domain_id}")
    details = get_json(f"/v1/ontology/domains/{domain_id}")
    if details is None:
        post_json(
            f"/v1/tenants/{tenant_id}/ontology/domains",
            {
                "id": domain_id,
                "name": domain_name,
                "description": domain_description,
                "status": "active",
                "visibility": domain_visibility,
            },
        )
        details = get_json(f"/v1/ontology/domains/{domain_id}")
    if details is None:
        raise RuntimeError(f"domain {domain_id} could not be read back after creation")
    ensure_runtime_identity_rows()
    ensure_domain_row(details)
    return details


def ensure_runtime_identity_rows() -> None:
    repo_root = os.environ.get("CODEGRAPH_REPO_ROOT")
    if not repo_root:
        raise RuntimeError("CODEGRAPH_REPO_ROOT is required to seed the Postgres identity rows")

    compose_file = os.path.join(repo_root, "deploy/compose/codegraph-runtime/docker-compose.yml")
    clear_runtime_namespace(compose_file)
    api_key_hash = hashlib.sha256(api_key.encode("utf-8")).hexdigest()
    api_key_prefix = api_key[:8] if len(api_key) >= 8 else api_key

    def sql_quote(value: str) -> str:
        return "'" + value.replace("'", "''") + "'"

    sql = "\n".join(
        [
            "INSERT INTO tenants (id, slug, name, status, tier)",
            "VALUES ("
            + ", ".join(
                [
                    sql_quote(tenant_id),
                    sql_quote(tenant_slug),
                    sql_quote(tenant_name),
                    sql_quote("active"),
                    sql_quote(tenant_tier),
                ]
            )
            + ")",
            "ON CONFLICT (slug) DO UPDATE SET "
            "name = EXCLUDED.name, "
            "status = EXCLUDED.status, "
            "tier = EXCLUDED.tier, "
            "updated_at = now();",
            "INSERT INTO apps (id, tenant_id, slug, name, type, api_key_hash, api_key_prefix, status)",
            "VALUES ("
            + ", ".join(
                [
                    sql_quote(app_id),
                    sql_quote(tenant_id),
                    sql_quote(app_slug),
                    sql_quote(app_name),
                    sql_quote(app_type),
                    sql_quote(api_key_hash),
                    sql_quote(api_key_prefix),
                    sql_quote("active"),
                ]
            )
            + ")",
            "ON CONFLICT (tenant_id, slug) DO UPDATE SET "
            "name = EXCLUDED.name, "
            "type = EXCLUDED.type, "
            "api_key_hash = EXCLUDED.api_key_hash, "
            "api_key_prefix = EXCLUDED.api_key_prefix, "
            "status = EXCLUDED.status, "
            "revoked_at = NULL;",
        ]
    )

    log("==> ensuring Postgres tenant and app rows for the runtime identity")
    result = subprocess.run(
        [
            "docker",
            "compose",
            "-f",
            compose_file,
            "exec",
            "-T",
            "postgres",
            "psql",
            "-U",
            "postgres",
            "-d",
            "kg_service",
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            sql,
        ],
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            "failed to seed runtime identity rows in Postgres: "
            + (result.stderr.strip() or result.stdout.strip() or f"exit {result.returncode}")
        )


def clear_runtime_namespace(compose_file: str) -> None:
    log("==> clearing stale CodeGraph runtime rows")

    sql = "\n".join(
        [
            "DELETE FROM kg_vector_documents WHERE domain_id = 'code-graph';",
            "DELETE FROM kg_relationships WHERE domain_id = 'code-graph';",
            "DELETE FROM kg_nodes WHERE domain_id = 'code-graph';",
            "DELETE FROM kg_graph_version_entities WHERE version_id IN (SELECT version_id FROM kg_graph_versions WHERE identifier_id IN (SELECT identifier_id FROM kg_graph_identifiers WHERE owner_tenant_id IN (SELECT id FROM tenants WHERE slug = 'codegraph-runtime') OR owner_app_id IN (SELECT id FROM apps WHERE slug = 'codegraph-runtime-admin')));",
            "DELETE FROM kg_graph_versions WHERE identifier_id IN (SELECT identifier_id FROM kg_graph_identifiers WHERE owner_tenant_id IN (SELECT id FROM tenants WHERE slug = 'codegraph-runtime') OR owner_app_id IN (SELECT id FROM apps WHERE slug = 'codegraph-runtime-admin'));",
            "DELETE FROM kg_graph_projection_heads WHERE identifier_id IN (SELECT identifier_id FROM kg_graph_identifiers WHERE owner_tenant_id IN (SELECT id FROM tenants WHERE slug = 'codegraph-runtime') OR owner_app_id IN (SELECT id FROM apps WHERE slug = 'codegraph-runtime-admin'));",
            "DELETE FROM kg_graph_identifiers WHERE owner_tenant_id IN (SELECT id FROM tenants WHERE slug = 'codegraph-runtime') OR owner_app_id IN (SELECT id FROM apps WHERE slug = 'codegraph-runtime-admin');",
            "DELETE FROM domain_query_templates WHERE domain_id = 'code-graph';",
            "DELETE FROM domain_status_field_configs WHERE domain_id = 'code-graph';",
            "DELETE FROM node_type_schemas WHERE domain_id = 'code-graph';",
            "DELETE FROM rel_type_schemas WHERE domain_id = 'code-graph';",
            "DELETE FROM cross_domain_rel_rules WHERE from_domain_id = 'code-graph' OR to_domain_id = 'code-graph';",
            "DELETE FROM ontology_versions WHERE domain_id = 'code-graph';",
            "DELETE FROM domains WHERE id = 'code-graph';",
            "DELETE FROM access_grants WHERE grantor_tenant_id IN (SELECT id FROM tenants WHERE slug = 'codegraph-runtime') "
            "OR grantee_tenant_id IN (SELECT id FROM tenants WHERE slug = 'codegraph-runtime');",
            "DELETE FROM apps WHERE tenant_id IN (SELECT id FROM tenants WHERE slug = 'codegraph-runtime') "
            "AND slug = 'codegraph-runtime-admin';",
            "DELETE FROM tenants WHERE slug = 'codegraph-runtime';",
        ]
    )

    result = subprocess.run(
        [
            "docker",
            "compose",
            "-f",
            compose_file,
            "exec",
            "-T",
            "postgres",
            "psql",
            "-U",
            "postgres",
            "-d",
            "kg_service",
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            sql,
        ],
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            "failed to clear stale runtime rows from Postgres: "
            + (result.stderr.strip() or result.stdout.strip() or f"exit {result.returncode}")
        )


def ensure_domain_row(details: dict[str, Any]) -> None:
    repo_root = os.environ.get("CODEGRAPH_REPO_ROOT")
    if not repo_root:
        raise RuntimeError("CODEGRAPH_REPO_ROOT is required to seed the Postgres domain row")

    compose_file = os.path.join(repo_root, "deploy/compose/codegraph-runtime/docker-compose.yml")
    domain = details.get("domain") or {}
    desired_name = str(domain.get("name") or domain_name)
    desired_description = str(domain.get("description") or domain_description)
    desired_visibility = str(domain.get("visibility") or domain_visibility)

    def sql_quote(value: str) -> str:
        return "'" + value.replace("'", "''") + "'"

    sql = (
        "INSERT INTO domains (id, name, description, owner_tenant_id, status, version, visibility) "
        "VALUES ("
        + ", ".join(
            [
                sql_quote(domain_id),
                sql_quote(desired_name),
                sql_quote(desired_description),
                sql_quote(tenant_id),
                sql_quote("active"),
                "1",
                sql_quote(desired_visibility),
            ]
        )
        + ") "
        "ON CONFLICT (id) DO UPDATE SET "
        "name = EXCLUDED.name, "
        "description = EXCLUDED.description, "
        "owner_tenant_id = EXCLUDED.owner_tenant_id, "
        "status = EXCLUDED.status, "
        "version = EXCLUDED.version, "
        "visibility = EXCLUDED.visibility, "
        "updated_at = now();"
    )

    log("==> ensuring Postgres domain row for code-graph")
    result = subprocess.run(
        [
            "docker",
            "compose",
            "-f",
            compose_file,
            "exec",
            "-T",
            "postgres",
            "psql",
            "-U",
            "postgres",
            "-d",
            "kg_service",
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            sql,
        ],
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            "failed to seed code-graph domain row in Postgres: "
            + (result.stderr.strip() or result.stdout.strip() or f"exit {result.returncode}")
        )


def ensure_query_strategy() -> dict[str, Any]:
    log(f"==> ensuring query strategy {strategy_key}")
    strategies = get_json("/v1/ontology/query-strategies")
    if not isinstance(strategies, list):
        raise RuntimeError("query strategy list response is not an array")
    existing = find_by(strategies, "key", strategy_key)
    desired = {
        "key": query_strategy["key"],
        "version": query_strategy["version"],
        "max_depth": query_strategy["max_depth"],
        "params": query_strategy["params"],
    }
    if existing is None:
        log(f"   creating query strategy {strategy_key}")
        existing = post_json(
            f"/v1/tenants/{tenant_id}/ontology/query-strategies",
            query_strategy,
        )
    else:
        observed = {
            "key": existing.get("key"),
            "max_depth": existing.get("max_depth"),
            "params": existing.get("params", {}),
        }
        if observed != {k: desired[k] for k in ("key", "max_depth", "params")}:
            log(f"   updating query strategy {strategy_key}")
            existing = put_json(
                f"/v1/tenants/{tenant_id}/ontology/query-strategies/{strategy_key}",
                query_strategy,
            )
    return existing


def ensure_node_types(details: dict[str, Any]) -> dict[str, Any]:
    log("==> ensuring node type schemas")
    existing = {item.get("node_type_name"): item for item in list_or_empty(details.get("node_types"))}
    for schema in node_types:
        current = existing.get(schema["node_type_name"])
        if current is None:
            log(f"   creating node type {schema['node_type_name']}")
            post_json(
                f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types",
                schema,
            )
            continue
        expected = {
            "node_type_name": schema["node_type_name"],
            "graph_label": schema["graph_label"],
            "required_props": schema["required_props"],
            "optional_props": schema["optional_props"],
            "validation_rules": schema["validation_rules"],
        }
        observed = {
            "node_type_name": current.get("node_type_name"),
            "graph_label": current.get("graph_label"),
            "required_props": current.get("required_props", []),
            "optional_props": current.get("optional_props", []),
            "validation_rules": current.get("validation_rules", []),
        }
        if observed != expected:
            raise RuntimeError(
                f"node type {schema['node_type_name']} already exists but does not match the bootstrap spec"
            )
        log(f"   node type {schema['node_type_name']} already present")
    return get_json(f"/v1/ontology/domains/{domain_id}") or details


def ensure_rel_types(details: dict[str, Any]) -> dict[str, Any]:
    log("==> ensuring relationship type schemas")
    existing = list_or_empty(details.get("rel_types"))
    for rel_type_name, pairs in rel_pairs.items():
        for from_node_type, to_node_type in pairs:
            current = next(
                (
                    item
                    for item in existing
                    if item.get("rel_type_name") == rel_type_name
                    and item.get("from_node_type") == from_node_type
                    and item.get("to_node_type") == to_node_type
                ),
                None,
            )
            payload = {
                "rel_type_name": rel_type_name,
                "from_node_type": from_node_type,
                "to_node_type": to_node_type,
                "same_domain": True,
                "required_props": [],
                "optional_props": [
                    {"name": "codegraph_edge_kind", "type": "string"},
                    {"name": "line", "type": "number"},
                    {"name": "col", "type": "number"},
                    {"name": "provenance", "type": "string"},
                ],
            }
            if current is None:
                log(f"   creating rel type {rel_type_name} {from_node_type}->{to_node_type}")
                post_json(
                    f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types",
                    payload,
                )
                continue
            observed = {
                "rel_type_name": current.get("rel_type_name"),
                "from_node_type": current.get("from_node_type"),
                "to_node_type": current.get("to_node_type"),
                "same_domain": current.get("same_domain"),
                "required_props": current.get("required_props", []),
                "optional_props": current.get("optional_props", []),
            }
            if observed != payload:
                raise RuntimeError(
                    f"relationship type {rel_type_name} {from_node_type}->{to_node_type} already exists but does not match the bootstrap spec"
                )
            log(f"   rel type {rel_type_name} {from_node_type}->{to_node_type} already present")
    return get_json(f"/v1/ontology/domains/{domain_id}") or details


def ensure_search_profile(details: dict[str, Any]) -> dict[str, Any]:
    log("==> ensuring search profile")
    domain = details.get("domain") or {}
    current = domain.get("search_profile")
    desired = {
        "semantic_fields": search_profile["semantic_fields"],
        "fts_language": search_profile["fts_language"],
        "query_strategy_ref": search_profile["query_strategy_ref"],
    }
    if current is None:
        log("   creating search profile")
        put_json(
            f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/search-profile",
            search_profile,
        )
        details = get_json(f"/v1/ontology/domains/{domain_id}")
        current = (details or {}).get("domain", {}).get("search_profile")
    if current is None:
        raise RuntimeError("search profile could not be read back after upsert")
    observed = {
        "semantic_fields": current.get("semantic_fields", []),
        "fts_language": current.get("fts_language"),
        "query_strategy_ref": current.get("query_strategy_ref"),
    }
    if observed != desired:
        log("   updating search profile")
        put_json(
            f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/search-profile",
            search_profile,
        )
        details = get_json(f"/v1/ontology/domains/{domain_id}") or details
        current = (details or {}).get("domain", {}).get("search_profile")
        observed = {
            "semantic_fields": current.get("semantic_fields", []),
            "fts_language": current.get("fts_language"),
            "query_strategy_ref": current.get("query_strategy_ref"),
        }
    if observed != desired:
        raise RuntimeError("search profile does not match the bootstrap spec")
    log("   search profile already present")
    return details or {}


def ensure_template(details: dict[str, Any], template: dict[str, Any]) -> dict[str, Any]:
    templates_in_domain = list_or_empty(details.get("query_templates"))
    current = find_by(templates_in_domain, "template_name", template["template_name"])
    if current is None:
        log(f"   creating template {template['template_name']}")
        post_json(
            f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates",
            template,
        )
        current = get_json(f"/v1/ontology/domains/{domain_id}")
        templates_in_domain = list_or_empty((current or {}).get("query_templates"))
        current = find_by(templates_in_domain, "template_name", template["template_name"])
        if current is None:
            raise RuntimeError(f"template {template['template_name']} could not be read back after creation")
    expected = {
        "template_name": template["template_name"],
        "pattern_spec": template["pattern_spec"],
        "param_schema": template["param_schema"],
        "return_fields": template["return_fields"],
        "description": template["description"],
        "status": "active",
    }
    observed = {
        "template_name": current.get("template_name"),
        "pattern_spec": current.get("pattern_spec", {}),
        "param_schema": current.get("param_schema", []),
        "return_fields": current.get("return_fields", []),
        "description": current.get("description", ""),
        "status": current.get("status"),
    }
    if observed != expected:
        if current.get("status") != "active":
            log(f"   activating template {template['template_name']}")
            put_json(
                f"/v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{template['template_name']}/activate",
                {},
            )
            details = get_json(f"/v1/ontology/domains/{domain_id}") or details
            templates_in_domain = list_or_empty(details.get("query_templates"))
            current = find_by(templates_in_domain, "template_name", template["template_name"])
            observed["status"] = current.get("status") if current else None
            if observed != expected:
                raise RuntimeError(f"template {template['template_name']} does not match the bootstrap spec")
        else:
            raise RuntimeError(f"template {template['template_name']} already exists but does not match the bootstrap spec")
    else:
        log(f"   template {template['template_name']} already present and active")
    return details


def verify_domain(details: dict[str, Any]) -> None:
    log("==> verifying bootstrapped entities")
    domain = details.get("domain") or {}
    if domain.get("id") != domain_id:
        raise RuntimeError("domain verification failed: wrong domain id")
    if domain.get("name") != domain_name:
        raise RuntimeError("domain verification failed: wrong domain name")
    if domain.get("status") != "active":
        raise RuntimeError("domain verification failed: wrong status")
    if domain.get("visibility") != domain_visibility:
        raise RuntimeError("domain verification failed: wrong visibility")

    node_types_by_name = {item.get("node_type_name"): item for item in list_or_empty(details.get("node_types"))}
    for schema in node_types:
        current = node_types_by_name.get(schema["node_type_name"])
        if current is None:
            raise RuntimeError(f"missing node type {schema['node_type_name']}")

    rel_types_by_key = {
        (item.get("rel_type_name"), item.get("from_node_type"), item.get("to_node_type")): item
        for item in list_or_empty(details.get("rel_types"))
    }
    for rel_type_name, pairs in rel_pairs.items():
        for from_node_type, to_node_type in pairs:
            if (rel_type_name, from_node_type, to_node_type) not in rel_types_by_key:
                raise RuntimeError(f"missing relationship type {rel_type_name} {from_node_type}->{to_node_type}")

    profile = domain.get("search_profile")
    if profile is None:
        raise RuntimeError("missing search profile")
    profile_subset = {
        "semantic_fields": profile.get("semantic_fields", []),
        "fts_language": profile.get("fts_language"),
        "query_strategy_ref": profile.get("query_strategy_ref"),
    }
    desired_profile = {
        "semantic_fields": search_profile["semantic_fields"],
        "fts_language": search_profile["fts_language"],
        "query_strategy_ref": search_profile["query_strategy_ref"],
    }
    if profile_subset != desired_profile:
        raise RuntimeError("search profile verification failed")

    strategies = get_json("/v1/ontology/query-strategies")
    if not isinstance(strategies, list):
        raise RuntimeError("query strategy verification failed: response is not an array")
    strategy = find_by(strategies, "key", strategy_key)
    if strategy is None:
        raise RuntimeError("query strategy verification failed: missing strategy")
    strategy_subset = {
        "key": strategy.get("key"),
        "max_depth": strategy.get("max_depth"),
        "params": strategy.get("params", {}),
    }
    desired_strategy = {
        "key": query_strategy["key"],
        "max_depth": query_strategy["max_depth"],
        "params": query_strategy["params"],
    }
    if strategy_subset != desired_strategy:
        raise RuntimeError("query strategy verification failed")

    templates_by_name = {item.get("template_name"): item for item in list_or_empty(details.get("query_templates"))}
    for template in templates:
        current = templates_by_name.get(template["template_name"])
        if current is None:
            raise RuntimeError(f"missing query template {template['template_name']}")
        if current.get("status") != "active":
            raise RuntimeError(f"template {template['template_name']} is not active")
        current_subset = {
            "template_name": current.get("template_name"),
            "pattern_spec": current.get("pattern_spec", {}),
            "param_schema": current.get("param_schema", []),
            "return_fields": current.get("return_fields", []),
            "description": current.get("description", ""),
        }
        desired_subset = {
            "template_name": template["template_name"],
            "pattern_spec": template["pattern_spec"],
            "param_schema": template["param_schema"],
            "return_fields": template["return_fields"],
            "description": template["description"],
        }
        if current_subset != desired_subset:
            raise RuntimeError(f"template {template['template_name']} verification failed")

    log("verification passed for domain, node types, relationship types, search profile, and query templates")


def main() -> None:
    log(f"bootstrapping ontology domain {domain_id} at {service_url}")
    details = ensure_domain()
    details = ensure_query_strategy()
    details = get_json(f"/v1/ontology/domains/{domain_id}") or details
    details = ensure_node_types(details)
    details = ensure_rel_types(details)
    details = get_json(f"/v1/ontology/domains/{domain_id}") or details
    details = ensure_search_profile(details)
    details = get_json(f"/v1/ontology/domains/{domain_id}") or details
    for template in templates:
        details = ensure_template(details, template)
    details = get_json(f"/v1/ontology/domains/{domain_id}") or details
    verify_domain(details)


if __name__ == "__main__":
    main()
PY
