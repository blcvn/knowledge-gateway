"""
VNP Memory Seed — Shared client & config loader.

All seed scripts import from this module:
    from client import cfg, api
"""

from __future__ import annotations

import json
import os
import time
from pathlib import Path
from typing import Any

import requests
from dotenv import load_dotenv

# ── Load .env ─────────────────────────────────────────────────────────────────
_HERE = Path(__file__).parent
load_dotenv(_HERE / ".env")


class Config:
    """Reads all seed settings from environment variables."""

    base_url: str = os.getenv("VNP_BASE_URL", "http://localhost:8080").rstrip("/")
    api_key: str = os.getenv("VNP_API_KEY", "")
    access_token: str = os.getenv("VNP_ACCESS_TOKEN", "")
    email: str = os.getenv("VNP_EMAIL", "admin@vnp-memory.local")
    password: str = os.getenv("VNP_PASSWORD", "changeme")
    tenant_id: str = os.getenv("VNP_TENANT_ID", "")
    data_dir: Path = _HERE / os.getenv("SEED_DATA_DIR", "data")

    # Counts
    cognee_datasets: int = int(os.getenv("SEED_COGNEE_DATASETS", "2"))
    graphiti_episodes: int = int(os.getenv("SEED_GRAPHITI_EPISODES", "10"))
    memobase_users: int = int(os.getenv("SEED_MEMOBASE_USERS", "3"))
    memobase_blobs_per_user: int = int(os.getenv("SEED_MEMOBASE_BLOBS_PER_USER", "5"))
    zep_users: int = int(os.getenv("SEED_ZEP_USERS", "3"))
    zep_sessions_per_user: int = int(os.getenv("SEED_ZEP_SESSIONS_PER_USER", "2"))
    zep_messages_per_session: int = int(os.getenv("SEED_ZEP_MESSAGES_PER_SESSION", "6"))
    sm_memories: int = int(os.getenv("SEED_SM_MEMORIES", "10"))
    sm_documents: int = int(os.getenv("SEED_SM_DOCUMENTS", "3"))
    agent_memories: int = int(os.getenv("SEED_AGENT_MEMORIES", "10"))
    observe_sessions: int = int(os.getenv("SEED_OBSERVE_SESSIONS", "2"))

    # HTTP
    timeout: int = int(os.getenv("HTTP_TIMEOUT", "30"))
    retries: int = int(os.getenv("HTTP_RETRIES", "3"))
    retry_delay: float = float(os.getenv("HTTP_RETRY_DELAY", "2"))


cfg = Config()


class VNPClient:
    """HTTP client for VNP Memory API.

    Auth precedence: API Key > Access Token > Login (auto).
    Auth is configured lazily on the first request so importing client.py
    in scripts that don't need network access (e.g. 01_generate_data.py)
    does NOT trigger a connection attempt.
    """

    def __init__(self, config: Config):
        self._cfg = config
        self._session = requests.Session()
        self._session.timeout = config.timeout
        self._token: str = ""
        self._auth_ready: bool = False

    # ── Auth ─────────────────────────────────────────────────────────────────

    def _ensure_auth(self) -> None:
        """Configure auth headers on the first network call (lazy init)."""
        if self._auth_ready:
            return
        c = self._cfg
        if c.api_key:
            self._session.headers["X-API-Key"] = c.api_key
            print(f"[auth] Using API key: {c.api_key[:12]}...")
        elif c.access_token:
            self._token = c.access_token
            self._session.headers["Authorization"] = f"Bearer {self._token}"
            print("[auth] Using pre-configured access token.")
        else:
            print("[auth] No API key / token — attempting login...")
            self._login()

        if c.tenant_id:
            self._session.headers["X-Tenant-ID"] = c.tenant_id
        self._auth_ready = True

    def _login(self) -> None:
        resp = self._session.post(
            f"{self._cfg.base_url}/v1/auth/login",
            json={"email": self._cfg.email, "password": self._cfg.password},
            timeout=self._cfg.timeout,
        )
        resp.raise_for_status()
        data = resp.json()
        self._token = data["access_token"]
        self._session.headers["Authorization"] = f"Bearer {self._token}"
        # Extract tenant_id from login response if not set
        if not self._cfg.tenant_id:
            user = data.get("user", {})
            if tid := user.get("tenant_id"):
                self._cfg.tenant_id = tid
                self._session.headers["X-Tenant-ID"] = tid
        print(f"[auth] Logged in as {self._cfg.email}")

    # ── HTTP helpers ─────────────────────────────────────────────────────────

    def _request(self, method: str, path: str, **kwargs) -> requests.Response:
        self._ensure_auth()
        url = f"{self._cfg.base_url}{path}"
        last_exc: Exception | None = None
        for attempt in range(1, self._cfg.retries + 1):
            try:
                resp = self._session.request(method, url, **kwargs)
                if resp.status_code == 429:
                    retry_after = int(resp.headers.get("Retry-After", self._cfg.retry_delay))
                    print(f"  [rate-limit] Sleeping {retry_after}s (attempt {attempt})")
                    time.sleep(retry_after)
                    continue
                return resp
            except requests.RequestException as exc:
                last_exc = exc
                if attempt < self._cfg.retries:
                    print(f"  [retry {attempt}] {exc} — waiting {self._cfg.retry_delay}s")
                    time.sleep(self._cfg.retry_delay)
        raise RuntimeError(f"Request failed after {self._cfg.retries} attempts") from last_exc

    def get(self, path: str, **kwargs) -> dict[str, Any]:
        resp = self._request("GET", path, **kwargs)
        resp.raise_for_status()
        return resp.json()

    def post(self, path: str, body: Any = None, **kwargs) -> dict[str, Any]:
        resp = self._request("POST", path, json=body, **kwargs)
        resp.raise_for_status()
        return resp.json() if resp.content else {}

    def put(self, path: str, body: Any = None, **kwargs) -> dict[str, Any]:
        resp = self._request("PUT", path, json=body, **kwargs)
        resp.raise_for_status()
        return resp.json() if resp.content else {}

    def patch(self, path: str, body: Any = None, **kwargs) -> dict[str, Any]:
        resp = self._request("PATCH", path, json=body, **kwargs)
        resp.raise_for_status()
        return resp.json() if resp.content else {}

    def delete(self, path: str, **kwargs) -> dict[str, Any]:
        resp = self._request("DELETE", path, **kwargs)
        resp.raise_for_status()
        return resp.json() if resp.content else {}

    def safe_post(self, path: str, body: Any = None, **kwargs) -> dict[str, Any] | None:
        """POST that returns None on error instead of raising."""
        try:
            return self.post(path, body, **kwargs)
        except requests.HTTPError as exc:
            print(f"  [warn] POST {path} → {exc.response.status_code}: {exc.response.text[:200]}")
            return None

    def safe_get(self, path: str, **kwargs) -> dict[str, Any] | None:
        """GET that returns None on error instead of raising."""
        try:
            return self.get(path, **kwargs)
        except requests.HTTPError as exc:
            print(f"  [warn] GET {path} → {exc.response.status_code}: {exc.response.text[:200]}")
            return None


# Shared singleton
api = VNPClient(cfg)


# ── Data helpers ─────────────────────────────────────────────────────────────

def save_json(filename: str, data: Any) -> Path:
    """Save data to SEED_DATA_DIR/{filename}.json and return the path."""
    cfg.data_dir.mkdir(parents=True, exist_ok=True)
    path = cfg.data_dir / filename
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False))
    return path


def load_json(filename: str) -> Any:
    """Load data from SEED_DATA_DIR/{filename}.json."""
    path = cfg.data_dir / filename
    if not path.exists():
        raise FileNotFoundError(f"Seed data not found: {path}. Run 01_generate_data.py first.")
    return json.loads(path.read_text())


def print_section(title: str) -> None:
    print(f"\n{'═' * 60}")
    print(f"  {title}")
    print("═" * 60)
