#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
base_url="${KG_BASE_URL:-http://127.0.0.1:8082}"
platform_api_key="${KG_PLATFORM_API_KEY:-kgsk_platform_admin}"
env_file="${repo_root}/deploy/compose/codegraph-runtime/.env"

if [[ -f "${env_file}" ]]; then
  # shellcheck disable=SC1090
  set -a
  source "${env_file}"
  set +a
fi

skip_compose=0
skip_tenant_bootstrap=0
skip_ontology_bootstrap=0
skip_sync=0
skip_verify=0
fast_fail=0

usage() {
  cat <<'EOF'
Usage: validate-codegraph-runtime.sh [options]

Options:
  --skip-compose             Reuse an already-running Compose stack
  --skip-tenant-bootstrap    Reuse an existing tenant/app key instead of creating one
  --skip-ontology-bootstrap  Reuse an existing code-graph ontology instead of bootstrapping it
  --skip-sync                Skip the CodeGraph upsert step
  --skip-verify              Skip the post-bootstrap and post-sync verification steps
  --fast-fail                Reduce sync wait timeouts for quicker failure on projection issues
  --help                     Show this message

Environment:
  EMBEDDING_PROVIDER         Must be http
  EMBEDDING_URL              Required
  EMBEDDING_MODEL            Required
  EMBEDDING_API_KEY          Required
  KG_PLATFORM_API_KEY        Platform admin key used to create tenant/app on first run
  KG_API_KEY                 Existing tenant admin key to reuse instead of creating a new app
  KG_TENANT_ID               Existing tenant id to reuse with KG_API_KEY
  KG_RUNTIME_STATE_FILE      Local state file for tenant/app reuse
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-compose)
      skip_compose=1
      ;;
    --skip-tenant-bootstrap)
      skip_tenant_bootstrap=1
      ;;
    --skip-ontology-bootstrap)
      skip_ontology_bootstrap=1
      ;;
    --skip-sync)
      skip_sync=1
      ;;
    --skip-verify)
      skip_verify=1
      ;;
    --fast-fail)
      fast_fail=1
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

require_var() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "${name} is required for CodeGraph runtime validation" >&2
    exit 1
  fi
}

step() {
  echo "==> $1"
}

if [[ "${EMBEDDING_PROVIDER:-}" != "http" ]]; then
  echo "EMBEDDING_PROVIDER must be http for CodeGraph runtime validation" >&2
  exit 1
fi

require_var EMBEDDING_URL
require_var EMBEDDING_MODEL
require_var EMBEDDING_API_KEY

export KG_BASE_URL="${base_url%/}"
export KG_SERVICE_URL="${KG_SERVICE_URL:-${KG_BASE_URL}}"
export KG_RUNTIME_PROFILE="qdrant-memgraph"
export KG_DOMAIN_ID="${KG_DOMAIN_ID:-code-graph}"
export KG_TEMPLATE_DOMAIN_ID="${KG_TEMPLATE_DOMAIN_ID:-${KG_DOMAIN_ID}}"
export KG_DOMAIN_VISIBILITY="${KG_DOMAIN_VISIBILITY:-private}"
export PROJECT_PATH="${PROJECT_PATH:-${repo_root}}"
export KG_TENANT_SLUG="${KG_TENANT_SLUG:-codegraph-runtime}"
export KG_TENANT_NAME="${KG_TENANT_NAME:-CodeGraph Runtime Tenant}"
export KG_TENANT_TIER="${KG_TENANT_TIER:-pro}"
export KG_APP_SLUG="${KG_APP_SLUG:-codegraph-runtime-admin}"
export KG_APP_NAME="${KG_APP_NAME:-CodeGraph Runtime Admin}"
export KG_APP_TYPE="${KG_APP_TYPE:-admin_tool}"
export KG_RUNTIME_STATE_FILE="${KG_RUNTIME_STATE_FILE:-${repo_root}/examples/codegraph/.state/codegraph-runtime-bootstrap.json}"
export KG_PLATFORM_API_KEY="${platform_api_key}"
export KG_SKIP_TENANT_BOOTSTRAP="${skip_tenant_bootstrap}"
export KG_CODEGRAPH_PROBE_FILE="${repo_root}/internal/codegraphprobe/probe.go"
export KG_CODEGRAPH_PROBE_NAME="CodeGraphValidationProbe"
export KG_CODEGRAPH_PROBE_DOCSTRING_OLD="CodeGraphValidationProbe is a stable symbol used by the CodeGraph validation flow."
export KG_CODEGRAPH_PROBE_DOCSTRING_NEW="CodeGraphValidationProbe is a stable symbol used by the updated CodeGraph validation flow."
export KG_FAST_FAIL="${fast_fail}"

if [[ "${skip_compose}" -eq 0 ]]; then
  step "boot CodeGraph Compose stack"
  "${repo_root}/examples/codegraph/deploy-compose-codegraph-runtime.sh"
else
  step "reuse existing CodeGraph Compose stack"
fi

step "verify Memgraph vm.max_map_count"
memgraph_vm_max_map_count="$(
  docker compose -f "${repo_root}/deploy/compose/codegraph-runtime/docker-compose.yml" exec -T memgraph cat /proc/sys/vm/max_map_count
)"
memgraph_vm_max_map_count="${memgraph_vm_max_map_count//$'\r'/}"
if [[ "${memgraph_vm_max_map_count}" -lt 524288 ]]; then
  echo "Memgraph vm.max_map_count=${memgraph_vm_max_map_count} is too low; expected at least 524288. On Docker Desktop, rerun the stack after applying the sysctl or updating the Compose service override." >&2
  exit 1
fi

step "wait for healthz"
python3 - <<'PY'
import os
import sys
import time
import urllib.request

base_url = os.environ["KG_BASE_URL"].rstrip("/")
deadline = time.time() + 180
last_error = None

while time.time() < deadline:
    try:
        with urllib.request.urlopen(base_url + "/healthz", timeout=5) as resp:
            if resp.status == 200:
                sys.exit(0)
            last_error = f"unexpected status {resp.status}"
    except Exception as exc:  # noqa: BLE001
        last_error = str(exc)
    time.sleep(2)

print(f"health check did not succeed within timeout: {last_error}", file=sys.stderr)
raise SystemExit(1)
PY

step "ensure CodeGraph tenant and app identity"
eval "$(
python3 - <<'PY'
import json
import os
import shlex
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Dict, Optional, Tuple

base_url = os.environ["KG_BASE_URL"].rstrip("/")
platform_api_key = os.environ["KG_PLATFORM_API_KEY"]
runtime_api_key = os.environ.get("KG_API_KEY", "").strip()
tenant_id_hint = os.environ.get("KG_TENANT_ID", "").strip()
runtime_state_file = Path(os.environ["KG_RUNTIME_STATE_FILE"])
skip_tenant_bootstrap = os.environ.get("KG_SKIP_TENANT_BOOTSTRAP", "0") == "1"


def request(method: str, path: str, api_key: str, payload: Optional[Dict[str, Any]] = None) -> Tuple[int, object]:
    body = None if payload is None else json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(base_url + path, data=body, method=method)
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read()
            return resp.status, json.loads(raw.decode("utf-8")) if raw else None
    except urllib.error.HTTPError as exc:
        raw = exc.read()
        try:
            payload = json.loads(raw.decode("utf-8")) if raw else None
        except Exception:  # noqa: BLE001
            payload = raw.decode("utf-8", "replace")
        return exc.code, payload


def fail(message: str) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(1)


def resolve_identity(api_key: str) -> Dict[str, Any]:
    status, payload = request("GET", "/v1/access/resolve", api_key)
    if status < 200 or status >= 300 or not isinstance(payload, dict):
        fail(f"could not resolve API key identity: status={status} payload={payload!r}")
    return payload


def try_resolve_identity(api_key: str) -> Optional[Dict[str, Any]]:
    status, payload = request("GET", "/v1/access/resolve", api_key)
    if status < 200 or status >= 300 or not isinstance(payload, dict):
        return None
    return payload


def save_state(state: Dict[str, Any]) -> None:
    runtime_state_file.parent.mkdir(parents=True, exist_ok=True)
    runtime_state_file.write_text(json.dumps(state, indent=2) + "\n", encoding="utf-8")


def emit_exports(state: Dict[str, Any], source: str) -> None:
    exports = {
        "KG_API_KEY": state["api_key"],
        "KG_TENANT_ID": state["tenant_id"],
        "KG_RUNTIME_APP_ID": state["app_id"],
        "KG_RUNTIME_APP_SLUG": state.get("app_slug", ""),
    }
    for key, value in exports.items():
        print(f"export {key}={shlex.quote(str(value))}")
    print(f"echo using CodeGraph runtime identity from {shlex.quote(source)} >&2")


if runtime_api_key:
    identity = resolve_identity(runtime_api_key)
    tenant_id = identity.get("tenant_id", "")
    app_id = identity.get("app_id", "")
    if tenant_id_hint and tenant_id_hint != tenant_id:
        fail(f"KG_TENANT_ID={tenant_id_hint} does not match the supplied KG_API_KEY tenant {tenant_id}")
    state = {
        "tenant_id": tenant_id,
        "app_id": app_id,
        "api_key": runtime_api_key,
        "tenant_slug": os.environ.get("KG_TENANT_SLUG", ""),
        "app_slug": os.environ.get("KG_APP_SLUG", ""),
    }
    save_state(state)
    emit_exports(state, "KG_API_KEY")
    raise SystemExit(0)

if runtime_state_file.exists():
    try:
        state = json.loads(runtime_state_file.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        fail(f"runtime state file is invalid JSON: {exc}")
    api_key = str(state.get("api_key", "")).strip()
    tenant_id = str(state.get("tenant_id", "")).strip()
    if api_key and tenant_id:
        identity = try_resolve_identity(api_key)
        if identity is not None and identity.get("tenant_id") == tenant_id:
            emit_exports(state, str(runtime_state_file))
            raise SystemExit(0)

if skip_tenant_bootstrap:
    fail("no reusable tenant/app identity found; rerun without --skip-tenant-bootstrap or provide KG_API_KEY")

status, tenant = request(
    "POST",
    "/v1/tenants",
    platform_api_key,
    {
        "slug": os.environ["KG_TENANT_SLUG"],
        "name": os.environ["KG_TENANT_NAME"],
        "tier": os.environ["KG_TENANT_TIER"],
    },
)
if status < 200 or status >= 300 or not isinstance(tenant, dict):
    fail(f"tenant bootstrap failed: status={status} payload={tenant!r}")

tenant_id = str(tenant.get("id", "")).strip()
if not tenant_id:
    fail("tenant bootstrap did not return an id")

status, app = request(
    "POST",
    f"/v1/tenants/{tenant_id}/apps",
    platform_api_key,
    {
        "slug": os.environ["KG_APP_SLUG"],
        "name": os.environ["KG_APP_NAME"],
        "type": os.environ["KG_APP_TYPE"],
    },
)
if status < 200 or status >= 300 or not isinstance(app, dict):
    fail(f"app bootstrap failed: status={status} payload={app!r}")

api_key = str(app.get("api_key", "")).strip()
app_id = str(app.get("id", "")).strip()
if not api_key or not app_id:
    fail("app bootstrap did not return app id and api key")

state = {
    "tenant_id": tenant_id,
    "app_id": app_id,
    "api_key": api_key,
    "tenant_slug": os.environ["KG_TENANT_SLUG"],
    "app_slug": os.environ["KG_APP_SLUG"],
}
save_state(state)
emit_exports(state, "bootstrap")
PY
)"

step "verify runtime identity"
python3 - <<'PY'
import json
import os
import sys
import urllib.request

base_url = os.environ["KG_BASE_URL"].rstrip("/")
api_key = os.environ["KG_API_KEY"]
tenant_id = os.environ["KG_TENANT_ID"]

req = urllib.request.Request(base_url + "/v1/access/resolve")
req.add_header("Authorization", f"Bearer {api_key}")
with urllib.request.urlopen(req, timeout=30) as resp:
    payload = json.loads(resp.read().decode("utf-8"))

if payload.get("tenant_id") != tenant_id:
    print(f"runtime identity mismatch: {payload!r}", file=sys.stderr)
    raise SystemExit(1)

print(f"resolved tenant={payload.get('tenant_id')} app={payload.get('app_id')}")
PY

if [[ "${skip_ontology_bootstrap}" -eq 0 ]]; then
  step "bootstrap code-graph ontology"
  "${repo_root}/examples/codegraph/bootstrap-codegraph-ontology.sh"
else
  step "reuse existing code-graph ontology"
fi

if [[ "${skip_verify}" -eq 0 ]]; then
  step "verify code-graph ontology"
  "${repo_root}/examples/codegraph/verify-codegraph-ontology.sh"
else
  step "skip code-graph ontology verification"
fi

if [[ "${skip_sync}" -eq 0 ]]; then
  step "dry-run CodeGraph sync"
  "${repo_root}/examples/codegraph/codegraph-example-sync-dry"

  step "upsert CodeGraph index"
  "${repo_root}/examples/codegraph/codegraph-example-sync"
else
  step "skip CodeGraph sync"
fi

if [[ "${skip_verify}" -eq 0 ]]; then
  step "verify CodeGraph create and update sync path"
  if command -v codegraph >/dev/null 2>&1; then
    probe_backup="$(mktemp "${TMPDIR:-/tmp}/codegraph-probe.XXXXXX")"
    cleanup_probe() {
      if [[ -f "${probe_backup}" ]]; then
        mv "${probe_backup}" "${KG_CODEGRAPH_PROBE_FILE}"
      fi
    }
    cp "${KG_CODEGRAPH_PROBE_FILE}" "${probe_backup}"
    trap cleanup_probe EXIT

    eval "$(
      python3 - <<'PY'
import json
import os
import shlex
import sys
import time
import urllib.error
import urllib.request
from typing import Any, Dict, Optional

base_url = os.environ["KG_BASE_URL"].rstrip("/")
api_key = os.environ["KG_API_KEY"]
app_id = os.environ["KG_RUNTIME_APP_ID"]
domain_id = os.environ["KG_DOMAIN_ID"]
probe_name = os.environ["KG_CODEGRAPH_PROBE_NAME"]
probe_file = os.environ["KG_CODEGRAPH_PROBE_FILE"]


def request(
    method: str,
    path: str,
    payload: Optional[Dict[str, Any]] = None,
    *,
    allow_not_found: bool = False,
) -> Dict[str, Any]:
    body = None
    req = urllib.request.Request(base_url + path, method=method)
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        req.data = body
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read()
            return json.loads(raw.decode("utf-8")) if raw else {}
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", "replace")
        if allow_not_found and exc.code == 404:
            return {}
        print(f"{method} {path} failed: {exc.code} {raw}", file=sys.stderr)
        raise SystemExit(1)


def read_probe() -> Dict[str, Any]:
    deadline = time.time() + 600
    last_error = None
    while time.time() < deadline:
        result = request(
            "POST",
            "/v1/kg/search/fulltext",
            {
                "query": probe_name,
                "domain_ids": [domain_id],
                "top_k": 10,
                "mode": "all_tokens",
                "fields": ["name", "docstring", "file"],
            },
        )
        for item in result.get("results", []):
            if item.get("node_type") != "Function":
                continue
            props = item.get("domain_props") or {}
            if props.get("name") != probe_name:
                continue
            if not str(props.get("file", "")).endswith("internal/codegraphprobe/probe.go"):
                continue
            node_id = str(item.get("node_id", "")).strip()
            if not node_id:
                continue
            node = request("GET", f"/v1/kg/read/nodes/{node_id}?app_id={app_id}&mode=non-realtime")
            if node.get("id") != node_id:
                continue
            props = node.get("properties") or {}
            docstring = str(props.get("docstring", "")).strip()
            sync_version = int(node.get("_kg_sync_version") or 0)
            if sync_version <= 0:
                continue
            return {
                "node_id": node_id,
                "sync_version": sync_version,
                "docstring": docstring,
            }
        last_error = "probe symbol not yet queryable"
        time.sleep(2)
    print(f"could not observe CodeGraph validation probe after sync: {last_error}", file=sys.stderr)
    raise SystemExit(1)


probe = read_probe()
print(f"export KG_CODEGRAPH_PROBE_NODE_ID={shlex.quote(probe['node_id'])}")
print(f"export KG_CODEGRAPH_PROBE_SYNC_VERSION={probe['sync_version']}")
print(f"export KG_CODEGRAPH_PROBE_DOCSTRING={shlex.quote(probe['docstring'])}")
PY
    )"

    python3 - <<'PY'
import os
import sys
from pathlib import Path

probe_file = Path(os.environ["KG_CODEGRAPH_PROBE_FILE"])
old = os.environ["KG_CODEGRAPH_PROBE_DOCSTRING_OLD"]
new = os.environ["KG_CODEGRAPH_PROBE_DOCSTRING_NEW"]
text = probe_file.read_text(encoding="utf-8")
if old not in text:
    print(f"probe docstring marker not found in {probe_file}", file=sys.stderr)
    raise SystemExit(1)
probe_file.write_text(text.replace(old, new, 1), encoding="utf-8")
PY

    "${repo_root}/examples/codegraph/codegraph-refresh.sh"
    "${repo_root}/examples/codegraph/codegraph-example-sync"

    eval "$(
      python3 - <<'PY'
import json
import os
import sys
import time
import urllib.error
import urllib.request
from typing import Any, Dict, Optional

base_url = os.environ["KG_BASE_URL"].rstrip("/")
api_key = os.environ["KG_API_KEY"]
app_id = os.environ["KG_RUNTIME_APP_ID"]
probe_id = os.environ["KG_CODEGRAPH_PROBE_NODE_ID"]
before_version = int(os.environ["KG_CODEGRAPH_PROBE_SYNC_VERSION"])
before_docstring = os.environ["KG_CODEGRAPH_PROBE_DOCSTRING"]
expected_docstring = os.environ["KG_CODEGRAPH_PROBE_DOCSTRING_NEW"]


def request(
    method: str,
    path: str,
    payload: Optional[Dict[str, Any]] = None,
    *,
    allow_not_found: bool = False,
) -> Dict[str, Any]:
    body = None
    req = urllib.request.Request(base_url + path, method=method)
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        req.data = body
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read()
            return json.loads(raw.decode("utf-8")) if raw else {}
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", "replace")
        if allow_not_found and exc.code == 404:
            return {}
        print(f"{method} {path} failed: {exc.code} {raw}", file=sys.stderr)
        raise SystemExit(1)


deadline = time.time() + 600
last_payload = None
while time.time() < deadline:
    node = request("GET", f"/v1/kg/read/nodes/{probe_id}?app_id={app_id}&mode=non-realtime")
    if node.get("id") != probe_id:
        time.sleep(2)
        continue
    props = node.get("properties") or {}
    docstring = str(props.get("docstring", "")).strip()
    sync_version = int(node.get("_kg_sync_version") or 0)
    last_payload = {
        "docstring": docstring,
        "sync_version": sync_version,
    }
    if sync_version > before_version and docstring == expected_docstring:
        print(f"export KG_CODEGRAPH_PROBE_SYNC_VERSION_AFTER={sync_version}")
        print(f"export KG_CODEGRAPH_PROBE_DOCSTRING_AFTER={json.dumps(docstring)}")
        raise SystemExit(0)
    time.sleep(2)

print(
    "probe update did not advance version or docstring within 10m: "
    f"before_version={before_version} before_docstring={before_docstring!r} "
    f"last={last_payload!r}",
    file=sys.stderr,
)
raise SystemExit(1)
PY
    )"
    if [[ "${KG_CODEGRAPH_PROBE_DOCSTRING_AFTER:-}" != "${KG_CODEGRAPH_PROBE_DOCSTRING_NEW}" ]]; then
      echo "probe docstring did not update to the expected value" >&2
      exit 1
    fi

    cleanup_probe
    trap - EXIT
    step "verified CodeGraph create and update sync path"
  else
    step "skip CodeGraph update validation (codegraph CLI unavailable)"
  fi

  step "verify relationshipdb write and projection sync timing"
python3 - <<'PY'
import json
import os
import sys
import time
import urllib.error
import urllib.request
import uuid
from typing import Any, Dict, Optional

base_url = os.environ["KG_BASE_URL"].rstrip("/")
api_key = os.environ["KG_API_KEY"]
app_id = os.environ["KG_RUNTIME_APP_ID"]
domain_id = os.environ["KG_DOMAIN_ID"]
project_id = os.environ["PROJECT_PATH"]
fast_fail = os.environ.get("KG_FAST_FAIL", "0") == "1"
marker = uuid.uuid4().hex
node_name = f"syncprobe{marker}"
node_docstring = f"syncprobe{marker}"
external_ref = f"syncprobe-{marker}"


def request(method: str, path: str, payload: Optional[Dict[str, Any]] = None, *, allow_not_found: bool = False) -> Optional[Dict[str, Any]]:
    data = None
    req = urllib.request.Request(base_url + path, method=method)
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        req.data = data
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read()
            return json.loads(raw.decode("utf-8")) if raw else {}
    except urllib.error.HTTPError as exc:
        if allow_not_found and exc.code in (404, 409):
            raw = exc.read().decode("utf-8", "replace")
            try:
                return json.loads(raw) if raw else None
            except Exception:  # noqa: BLE001
                return {"error_text": raw, "http_status": exc.code}
        raw = exc.read().decode("utf-8", "replace")
        print(f"{method} {path} failed: {exc.code} {raw}", file=sys.stderr)
        raise SystemExit(1)


def open_mcp_session() -> str:
    req = urllib.request.Request(base_url + "/v1/mcp/connect", method="GET")
    req.add_header("Authorization", f"Bearer {api_key}")
    with urllib.request.urlopen(req, timeout=30) as resp:
        for raw in resp:
            line = raw.decode("utf-8", "replace").strip()
            if not line.startswith("data:"):
                continue
            payload = json.loads(line[len("data:"):].strip())
            session_id = str(payload.get("session_id", "")).strip()
            if session_id:
                return session_id
    print("could not open MCP session", file=sys.stderr)
    raise SystemExit(1)


def mcp_call(session_id: str, tool_name: str, arguments: Dict[str, Any]) -> Optional[Dict[str, Any]]:
    body = {
        "jsonrpc": "2.0",
        "id": int(time.time() * 1000),
        "method": "tools/call",
        "params": {
            "name": tool_name,
            "arguments": arguments,
        },
    }
    req = urllib.request.Request(base_url + f"/v1/mcp/messages/{session_id}", data=json.dumps(body).encode("utf-8"), method="POST")
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            payload = json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", "replace")
        print(f"MCP {tool_name} failed: {exc.code} {raw}", file=sys.stderr)
        return None
    if payload.get("error"):
        return None
    result = payload.get("result")
    return result if isinstance(result, dict) else None


def has_node(result: Dict[str, Any], node_id: str, marker_value: str) -> bool:
    if not isinstance(result, dict):
        return False
    if result.get("id") != node_id:
        return False
    props = result.get("properties") or {}
    if marker_value not in json.dumps(result, sort_keys=True):
        return False
    return isinstance(props, dict) and props.get("name") == node_name


def search_fulltext(node_id: str, query: str) -> bool:
    result = request(
        "POST",
        "/v1/kg/search/fulltext",
        {
            "query": query,
            "domain_ids": [domain_id],
            "top_k": 10,
            "mode": "all_tokens",
            "fields": ["name", "docstring"],
        },
    )
    if not result:
        return False
    for item in result.get("results", []):
        if item.get("node_id") == node_id:
            return True
    return False


def fmt_duration(seconds: int) -> str:
    if seconds % 60 == 0:
        return f"{seconds // 60}m"
    if seconds < 60:
        return f"{seconds}s"
    return f"{seconds // 60}m{seconds % 60:02d}s"


created = request(
    "POST",
    "/v1/kg/write/nodes",
    {
        "domain_id": domain_id,
        "node_type": "Function",
        "properties": {
            "name": node_name,
            "kind": "function",
            "file": "examples/codegraph/validate-codegraph-runtime.sh",
            "line": 1,
            "language": "bash",
            "project_id": project_id,
            "docstring": node_docstring,
        },
        "visibility": "private",
        "external_ref": external_ref,
        "reference_id": external_ref,
    },
)
if not created or not created.get("node_id"):
    print(f"node write did not return a node_id: {created!r}", file=sys.stderr)
    raise SystemExit(1)

node_id = str(created["node_id"])
realtime = request("GET", f"/v1/kg/read/nodes/{node_id}?app_id={app_id}&mode=realtime")
if not realtime or not has_node(realtime, node_id, marker):
    print(f"realtime read did not return the freshly written node: {realtime!r}", file=sys.stderr)
    raise SystemExit(1)

session_id = open_mcp_session()
start = time.monotonic()
backend_timeout = 90 if fast_fail else 180
deadline = start + (300 if fast_fail else 600)
graph_deadline = start + backend_timeout
fts_deadline = start + backend_timeout
vector_deadline = start + backend_timeout
graph_sync_time = None
fts_sync_time = None
vector_sync_time = None
last_graph_status = None
last_vector_status = None
last_graph_read = None
last_progress_log = start

while time.monotonic() < deadline and (graph_sync_time is None or fts_sync_time is None or vector_sync_time is None):
    now = time.monotonic()
    pending = []
    if graph_sync_time is None:
        pending.append(f"graphdb({max(0, int(graph_deadline - now))}s left)")
    if fts_sync_time is None:
        pending.append(f"fts db({max(0, int(fts_deadline - now))}s left)")
    if vector_sync_time is None:
        pending.append(f"vector db({max(0, int(vector_deadline - now))}s left)")
    if pending and now - last_progress_log >= 15:
        print(
            "waiting for projection sync: "
            + ", ".join(pending)
            + (
                f"; graph_status={json.dumps(last_graph_status, sort_keys=True)}"
                if last_graph_status is not None
                else ""
            )
            + (
                f"; vector_status={json.dumps(last_vector_status, sort_keys=True)}"
                if last_vector_status is not None
                else ""
            ),
            file=sys.stderr,
        )
        last_progress_log = now

    if graph_sync_time is None and now >= graph_deadline:
        print(
            f"graph projection did not become queryable within {fmt_duration(backend_timeout)}",
            file=sys.stderr,
        )
        if last_graph_status is not None:
            print(f"last observed graph sync status: {json.dumps(last_graph_status, sort_keys=True)}", file=sys.stderr)
        if last_graph_read is not None:
            print(
                f"last observed non-realtime graph read: {json.dumps(last_graph_read, sort_keys=True) if last_graph_read is not None else 'null'}",
                file=sys.stderr,
            )
        raise SystemExit(1)

    if fts_sync_time is None and now >= fts_deadline:
        print(
            f"fulltext projection did not become queryable within {fmt_duration(backend_timeout)}",
            file=sys.stderr,
        )
        raise SystemExit(1)

    if vector_sync_time is None and now >= vector_deadline:
        print(
            f"vector projection did not report SYNCED within {fmt_duration(backend_timeout)}",
            file=sys.stderr,
        )
        if last_vector_status is not None:
            print(f"last observed vector sync status: {json.dumps(last_vector_status, sort_keys=True)}", file=sys.stderr)
        raise SystemExit(1)

    if graph_sync_time is None:
        graph = request("GET", f"/v1/kg/read/nodes/{node_id}?app_id={app_id}&mode=non-realtime", allow_not_found=True)
        if graph and has_node(graph, node_id, marker):
            graph_sync_time = time.monotonic() - start
        else:
            last_graph_read = graph
            status = mcp_call(session_id, "kg_entity_sync_status", {"entity_id": node_id, "entity_kind": "kg_node"})
            if status:
                last_graph_status = status
                if status.get("graph_lag_class") == "SYNCED":
                    print(
                        "graph sync status reported SYNCED but non-realtime read was not queryable; "
                        "this indicates the graph projection backend did not persist the node",
                        file=sys.stderr,
                    )
                    print(f"last_graph_status={json.dumps(status, sort_keys=True)}", file=sys.stderr)
                    print(f"last_graph_read={json.dumps(graph, sort_keys=True) if graph is not None else 'null'}", file=sys.stderr)
                    raise SystemExit(1)

    if fts_sync_time is None and search_fulltext(node_id, node_name):
        fts_sync_time = time.monotonic() - start

    if vector_sync_time is None:
        status = mcp_call(session_id, "kg_entity_sync_status", {"entity_id": node_id, "entity_kind": "kg_node"})
        if status and status.get("vector_lag_class") == "SYNCED":
            vector_sync_time = time.monotonic() - start
        if status:
            last_vector_status = status

    if graph_sync_time is None or fts_sync_time is None or vector_sync_time is None:
        time.sleep(2)

missing = [
    name
    for name, value in (
        ("graphdb", graph_sync_time),
        ("fts db", fts_sync_time),
        ("vector db", vector_sync_time),
    )
    if value is None
]
if missing:
    if graph_sync_time is None and last_graph_status is not None:
        print(
            "last observed graph sync status: "
            f"{json.dumps(last_graph_status, sort_keys=True)}",
            file=sys.stderr,
        )
    if vector_sync_time is None and last_vector_status is not None:
        print(
            "last observed vector sync status: "
            f"{json.dumps(last_vector_status, sort_keys=True)}",
            file=sys.stderr,
        )
    if graph_sync_time is None and last_graph_read is not None:
        print(
            "last observed non-realtime graph read: "
            f"{json.dumps(last_graph_read, sort_keys=True)}",
            file=sys.stderr,
        )
    print(
        f"projection sync did not complete within {fmt_duration(int(deadline - start))} for: " + ", ".join(missing),
        file=sys.stderr,
    )
    raise SystemExit(1)

print(
    "projection sync timing: "
    f"graphdb={graph_sync_time:.2f}s "
    f"fts_db={fts_sync_time:.2f}s "
    f"vector_db={vector_sync_time:.2f}s"
)
PY
  step "verify runtime get/list/search/template behavior"
python3 - <<'PY'
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Dict, Optional

base_url = os.environ["KG_BASE_URL"].rstrip("/")
api_key = os.environ["KG_API_KEY"]
tenant_id = os.environ["KG_TENANT_ID"]
domain_id = os.environ["KG_DOMAIN_ID"]
runtime_app_id = os.environ.get("KG_RUNTIME_APP_ID", "")


def request(
    method: str,
    path: str,
    payload: Optional[Dict[str, Any]] = None,
    *,
    allow_not_found: bool = False,
) -> Dict[str, Any]:
    body = None
    req = urllib.request.Request(base_url + path, method=method)
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        req.data = body
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read()
            return json.loads(raw.decode("utf-8")) if raw else {}
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", "replace")
        if allow_not_found and exc.code == 404:
            return {}
        print(f"{method} {path} failed: {exc.code} {raw}", file=sys.stderr)
        raise SystemExit(1)


def request_retry(method: str, path: str, payload: Optional[Dict[str, Any]] = None, *, expect_results: bool = False) -> Dict[str, Any]:
    deadline = time.time() + 60
    last_payload: Dict[str, Any] = {}
    while True:
        result = request(method, path, payload)
        last_payload = result
        if not expect_results or result.get("results"):
            return result
        if time.time() >= deadline:
            return last_payload
        time.sleep(2)


apps = request("GET", f"/v1/tenants/{tenant_id}/apps")
app_rows = apps.get("data", [])
if not any(item.get("id") == runtime_app_id for item in app_rows):
    print("runtime app was not visible in app listing", file=sys.stderr)
    raise SystemExit(1)

domain = request("GET", f"/v1/ontology/domains/{domain_id}")
if (domain.get("domain") or {}).get("id") != domain_id:
    print("domain lookup did not return the expected domain", file=sys.stderr)
    raise SystemExit(1)

templates = request("GET", f"/v1/kg/read/templates?domain_id={domain_id}")
template_rows = templates.get("data", [])
if not template_rows:
    print("template list is empty for the code-graph domain", file=sys.stderr)
    raise SystemExit(1)

semantic = request_retry(
    "POST",
    "/v1/kg/search/hybrid",
    {
        "query": "codegraph sync bridge",
        "domain_ids": [domain_id],
        "top_k": 5,
        "semantic_weight": 0.7,
        "fts_operator": "all_tokens",
    },
    expect_results=True,
)
semantic_results = semantic.get("results", [])
if not semantic_results:
    print("semantic search returned no results", file=sys.stderr)
    raise SystemExit(1)

fulltext = request(
    "POST",
    "/v1/kg/search/fulltext",
    {
        "query": "function",
        "domain_ids": [domain_id],
        "top_k": 20,
        "mode": "all_tokens",
        "fields": ["kind", "name", "docstring"],
    },
)
fulltext_results = fulltext.get("results", [])

probe_name = os.environ.get("KG_CODEGRAPH_PROBE_NAME", "").strip()
probe_file = os.environ.get("KG_CODEGRAPH_PROBE_FILE", "").strip()
probe_file_suffix = "/".join(Path(probe_file).parts[-3:]) if probe_file else ""
probe_search = None
if probe_name:
    probe_search = request(
        "POST",
        "/v1/kg/search/fulltext",
        {
            "query": probe_name,
            "domain_ids": [domain_id],
            "top_k": 10,
            "mode": "all_tokens",
            "fields": ["name", "docstring", "file"],
        },
    )

candidate = None
probe_node_id = os.environ.get("KG_CODEGRAPH_PROBE_NODE_ID", "").strip()
probe_owner_app_id = os.environ.get("KG_RUNTIME_APP_ID", "").strip()
probe_docstring_candidates = [
    os.environ.get("KG_CODEGRAPH_PROBE_DOCSTRING_AFTER", "").strip(),
    os.environ.get("KG_CODEGRAPH_PROBE_DOCSTRING_NEW", "").strip(),
    os.environ.get("KG_CODEGRAPH_PROBE_DOCSTRING", "").strip(),
]
if probe_node_id and probe_owner_app_id:
    node = request("GET", f"/v1/kg/read/nodes/{probe_node_id}?app_id={probe_owner_app_id}&mode=non-realtime", allow_not_found=True)
    if node.get("id") == probe_node_id:
        props = node.get("properties") or {}
        docstring = str(props.get("docstring", "")).strip()
        if not probe_docstring_candidates or docstring in probe_docstring_candidates or not docstring:
            candidate = {
                "node_id": probe_node_id,
                "owner_app_id": probe_owner_app_id,
                "project_id": props.get("project_id", ""),
                "file": props.get("file", ""),
                "line": props.get("line", ""),
                "name": props.get("name", ""),
            }

if candidate is None and probe_search is not None:
    probe_items = []
    for item in probe_search.get("results", []):
        if item.get("node_type") != "Function":
            continue
        props = item.get("domain_props") or {}
        if probe_name and str(props.get("name", "")).strip() != probe_name:
            continue
        if probe_file_suffix and not str(props.get("file", "")).endswith(probe_file_suffix):
            continue
        if item.get("node_id") and item.get("owner_app_id") and all(key in props for key in ("project_id", "file", "line", "name")):
            probe_items.append({
                "node_id": str(item["node_id"]).strip(),
                "owner_app_id": str(item["owner_app_id"]).strip(),
                "project_id": str(props["project_id"]).strip(),
                "file": str(props["file"]).strip(),
                "line": str(props["line"]).strip(),
                "name": str(props["name"]).strip(),
            })
    probe_items.sort(key=lambda item: (item["file"], item["line"], item["name"], item["node_id"]))
    for item in probe_items:
        node = request("GET", f"/v1/kg/read/nodes/{item['node_id']}?app_id={item['owner_app_id']}&mode=non-realtime", allow_not_found=True)
        if node.get("id") != item["node_id"]:
            continue
        candidate = item
        break

if candidate is None:
    ranked_items = []
    for item in fulltext_results:
        if item.get("node_type") != "Function":
            continue
        props = item.get("domain_props") or {}
        if item.get("node_id") and item.get("owner_app_id") and all(key in props for key in ("project_id", "file", "line", "name")):
            ranked_items.append({
                "node_id": str(item["node_id"]).strip(),
                "owner_app_id": str(item["owner_app_id"]).strip(),
                "project_id": str(props["project_id"]).strip(),
                "file": str(props["file"]).strip(),
                "line": str(props["line"]).strip(),
                "name": str(props["name"]).strip(),
            })
    ranked_items.sort(key=lambda item: (item["file"], item["line"], item["name"], item["node_id"]))
    for item in ranked_items:
        node = request("GET", f"/v1/kg/read/nodes/{item['node_id']}?app_id={item['owner_app_id']}", allow_not_found=True)
        if node.get("id") != item["node_id"]:
            continue
        candidate = item
        break

if candidate is None:
    print("could not find a Function node suitable for get/template verification", file=sys.stderr)
    raise SystemExit(1)

if not re.fullmatch(r"[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}", candidate["node_id"]):
    print(f"candidate node_id is not a UUID: {candidate['node_id']}", file=sys.stderr)
    raise SystemExit(1)

print(
    "codegraph runtime validation passed "
    f"(apps={len(app_rows)} templates={len(template_rows)} semantic_results={len(semantic_results)} "
    f"fulltext_results={len(fulltext_results)})"
)
PY
else
  step "skip post-sync verification"
fi
