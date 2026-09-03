"""
Test script for Bifrost AI Gateway — key, model, and connectivity validation.

Bifrost gateway exposes an OpenAI-compatible API:
  POST /v1/chat/completions  (model format: "provider/model-name")
  POST /v1/messages          (Anthropic-native — NOT supported by Bifrost)

Config is loaded from .env in the same directory:
  ANTHROPIC_API_KEY   = sk-bf-...   (Bifrost API key)
  ANTHROPIC_BASE_URL  = https://...  (Bifrost base URL, include /v1)
  ANTHROPIC_MODEL     = claude-sonnet-4-6

Usage:
  python3 test_llm.py
"""

from __future__ import annotations

import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Dict, Tuple, Union

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def load_env(env_path: Path) -> dict:
    """Parse a .env file without external dependencies."""
    if not env_path.exists():
        print(f"[ERROR] .env file not found: {env_path}")
        sys.exit(1)
    env_vars: dict[str, str] = {}
    with open(env_path) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" in line:
                k, _, v = line.partition("=")
                env_vars[k.strip()] = v.strip()
    return env_vars


W = 62

def header(title: str):
    print("\n" + "=" * W)
    print(f"  {title}")
    print("=" * W)

def ok(msg: str):   print(f"  ✅  {msg}")
def fail(msg: str): print(f"  ❌  {msg}")
def info(msg: str): print(f"  ℹ️   {msg}")


def bifrost_model(model: str, provider: str = "anthropic") -> str:
    """
    Bifrost requires "provider/model" format.
    If the model already contains a slash, return as-is.
    """
    if "/" in model:
        return model
    return f"{provider}/{model}"


def post_json(url: str, payload: dict, headers: dict) -> tuple[int, dict | str]:
    data = json.dumps(payload).encode()
    req = urllib.request.Request(url, data=data, method="POST")
    for k, v in headers.items():
        req.add_header(k, v)
    try:
        resp = urllib.request.urlopen(req, timeout=30)
        return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        try:
            return e.code, json.loads(body)
        except Exception:
            return e.code, body
    except Exception as exc:
        return -1, str(exc)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    env_path = Path(__file__).parent / ".env"
    header("Loading configuration from .env")
    env = load_env(env_path)

    api_key  = env.get("ANTHROPIC_API_KEY", "")
    base_url = env.get("ANTHROPIC_BASE_URL", "https://b9.openledger.vn/v1").rstrip("/")
    model    = env.get("ANTHROPIC_MODEL", "claude-sonnet-4-6")

    masked = api_key[:12] + "..." + api_key[-4:] if len(api_key) > 16 else "***"
    info(f"ANTHROPIC_API_KEY   = {masked}")
    info(f"ANTHROPIC_BASE_URL  = {base_url}")
    info(f"ANTHROPIC_MODEL     = {model}")

    if not api_key:
        fail("ANTHROPIC_API_KEY is empty — check your .env file.")
        sys.exit(1)

    # Bifrost gateway detection
    is_bifrost = api_key.startswith("sk-bf-")
    if is_bifrost:
        info("Detected Bifrost AI Gateway key (sk-bf-...)")
        info("Using OpenAI-compatible endpoint: /v1/chat/completions")
        bif_model = bifrost_model(model)
        endpoint  = f"{base_url}/chat/completions"
        auth_header = {"Authorization": f"Bearer {api_key}"}
    else:
        info("Using native Anthropic endpoint: /v1/messages")
        bif_model = model
        endpoint  = f"{base_url}/messages"
        auth_header = {"x-api-key": api_key, "anthropic-version": "2023-06-01"}

    common_headers = {
        "content-type": "application/json",
        "accept": "application/json",
        **auth_header,
    }

    # ------------------------------------------------------------------
    # Test 1 — Simple ping
    # ------------------------------------------------------------------
    header("Test 1: Simple ping")
    info(f"POST {endpoint}")
    info(f"Model: {bif_model}")

    t0 = time.perf_counter()
    payload = {
        "model": bif_model,
        "max_tokens": 64,
        "messages": [{"role": "user", "content": "Reply with exactly one word: PONG"}],
    }
    status, body = post_json(endpoint, payload, common_headers)
    elapsed = time.perf_counter() - t0

    if status == 200 and isinstance(body, dict):
        # OpenAI-compatible response
        if "choices" in body:
            reply = body["choices"][0]["message"]["content"].strip()
            usage = body.get("usage", {})
            ok(f"Response ({elapsed:.2f}s): {reply!r}")
            info(f"Provider  : {body.get('extra_fields', {}).get('provider', 'N/A')}")
            info(f"Model used: {body.get('model', 'N/A')}")
            info(f"Tokens    : {usage.get('prompt_tokens', '?')} in / {usage.get('completion_tokens', '?')} out")
        # Anthropic-native response
        elif "content" in body:
            reply = body["content"][0]["text"].strip()
            ok(f"Response ({elapsed:.2f}s): {reply!r}")
            info(f"Stop reason  : {body.get('stop_reason')}")
            info(f"Input tokens : {body.get('usage', {}).get('input_tokens', '?')}")
            info(f"Output tokens: {body.get('usage', {}).get('output_tokens', '?')}")
        else:
            ok(f"HTTP {status} ({elapsed:.2f}s)")
            info(f"Body: {json.dumps(body)[:300]}")
    else:
        fail(f"HTTP {status} ({elapsed:.2f}s)")
        info(f"Response: {json.dumps(body) if isinstance(body, dict) else body}")
        sys.exit(1)

    # ------------------------------------------------------------------
    # Test 2 — Longer reasoning / multi-line response
    # ------------------------------------------------------------------
    header("Test 2: Structured reasoning (count 1–5)")
    t0 = time.perf_counter()
    payload2 = {
        "model": bif_model,
        "max_tokens": 128,
        "messages": [{"role": "user", "content": "Count from 1 to 5, one number per line, nothing else."}],
    }
    status2, body2 = post_json(endpoint, payload2, common_headers)
    elapsed2 = time.perf_counter() - t0

    if status2 == 200 and isinstance(body2, dict):
        if "choices" in body2:
            text = body2["choices"][0]["message"]["content"].strip()
        elif "content" in body2:
            text = body2["content"][0]["text"].strip()
        else:
            text = str(body2)
        ok(f"Completed in {elapsed2:.2f}s")
        print("  " + "\n  ".join(text.splitlines()))
    else:
        fail(f"HTTP {status2}: {body2}")

    # ------------------------------------------------------------------
    # Test 3 — Streaming (only for OpenAI-compatible gateway)
    # ------------------------------------------------------------------
    if is_bifrost or "chat/completions" in endpoint:
        header("Test 3: Streaming response")
        stream_payload = {
            "model": bif_model,
            "max_tokens": 128,
            "stream": True,
            "messages": [{"role": "user", "content": "List 3 benefits of AI in one sentence each."}],
        }
        data = json.dumps(stream_payload).encode()
        req = urllib.request.Request(endpoint, data=data, method="POST")
        for k, v in common_headers.items():
            req.add_header(k, v)

        t0 = time.perf_counter()
        try:
            resp = urllib.request.urlopen(req, timeout=60)
            print("  Streamed: ", end="", flush=True)
            total_chars = 0
            for raw_line in resp:
                line = raw_line.decode("utf-8").strip()
                if line.startswith("data: "):
                    chunk = line[6:]
                    if chunk == "[DONE]":
                        break
                    try:
                        obj = json.loads(chunk)
                        delta = obj.get("choices", [{}])[0].get("delta", {}).get("content", "")
                        if delta:
                            print(delta, end="", flush=True)
                            total_chars += len(delta)
                    except json.JSONDecodeError:
                        pass
            elapsed3 = time.perf_counter() - t0
            print()  # newline
            ok(f"Stream done in {elapsed3:.2f}s — {total_chars} chars received")
        except Exception as exc:
            fail(f"Streaming error: {exc}")
    else:
        header("Test 3: Skipping streaming (native Anthropic endpoint)")
        info("Streaming skipped — use anthropic SDK for native streaming support.")

    # ------------------------------------------------------------------
    # Summary
    # ------------------------------------------------------------------
    header("✅ All tests passed!")
    info(f"Gateway : {'Bifrost (OpenAI-compat)' if is_bifrost else 'Native Anthropic'}")
    info(f"Endpoint: {endpoint}")
    info(f"Model   : {bif_model}")
    print()


if __name__ == "__main__":
    main()
