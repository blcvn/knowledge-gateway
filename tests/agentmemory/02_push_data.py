"""
02_push_data.py — Push dữ liệu từ data/ lên agentmemory server

Quy trình:
  1. Đọc sessions.json → POST /agentmemory/session/start mỗi session
  2. Đọc observations.jsonl → POST /agentmemory/observe mỗi observation
  3. Đọc memories.json → POST /agentmemory/remember mỗi memory
  4. In báo cáo kết quả

Requires:
  pip install requests

Chạy:
  cd tests/agentmemory
  python 02_push_data.py
  python 02_push_data.py --dry-run          # Không gửi request thực
  python 02_push_data.py --sessions-only    # Chỉ push sessions
  python 02_push_data.py --obs-only         # Chỉ push observations
  python 02_push_data.py --memories-only    # Chỉ push memories
"""
from __future__ import annotations

import argparse
import json
import sys
import time
from pathlib import Path
from typing import Optional

try:
    import requests
except ImportError:
    print("ERROR: requests chưa được cài.\n  → pip install requests")
    sys.exit(1)

import config

# ── HTTP client helpers ────────────────────────────────────────────────────────
def _post(
    endpoint: str,
    payload: dict,
    dry_run: bool = False,
    retries: int = 3,
) -> tuple[int, dict]:
    """POST đến server. Trả về (status_code, body_dict)."""
    url = f"{config.BASE_URL}/agentmemory/{endpoint.lstrip('/')}"

    if dry_run:
        print(f"  [DRY] POST {url}")
        print(f"        {json.dumps(payload, ensure_ascii=False)[:120]}...")
        return (200, {"dry_run": True})

    for attempt in range(1, retries + 1):
        try:
            resp = requests.post(
                url,
                json=payload,
                headers=config.auth_headers(),
                timeout=30,
            )
            try:
                body = resp.json()
            except Exception:
                body = {"raw": resp.text[:500]}
            return resp.status_code, body
        except requests.exceptions.ConnectionError as e:
            if attempt < retries:
                wait = attempt * 2
                print(f"  [WARN] Kết nối thất bại (attempt {attempt}/{retries}), chờ {wait}s... {e}")
                time.sleep(wait)
            else:
                return (0, {"error": f"ConnectionError: {e}"})
        except Exception as e:
            return (0, {"error": str(e)})

    return (0, {"error": "max retries exceeded"})


def _get(endpoint: str, dry_run: bool = False) -> tuple[int, dict]:
    """GET từ server. Trả về (status_code, body_dict)."""
    url = f"{config.BASE_URL}/agentmemory/{endpoint.lstrip('/')}"
    if dry_run:
        print(f"  [DRY] GET {url}")
        return (200, {"dry_run": True})
    try:
        resp = requests.get(url, headers=config.auth_headers(), timeout=10)
        body = resp.json() if resp.content else {}
        return resp.status_code, body
    except Exception as e:
        return (0, {"error": str(e)})


def _check_server(dry_run: bool = False) -> bool:
    """Kiểm tra server sẵn sàng."""
    if dry_run:
        print("[check] DRY RUN — bỏ qua kiểm tra server")
        return True
    code, body = _get("livez")
    if code == 200:
        print(f"[check] ✅ Server sẵn sàng: {config.BASE_URL}")
        return True
    print(f"[check] ❌ Server không phản hồi (HTTP {code}): {body}")
    return False


# ── Loaders ────────────────────────────────────────────────────────────────────
def _load_sessions() -> list[dict]:
    f = config.DATA_DIR / "sessions.json"
    if not f.exists():
        print(f"[load] ⚠️  {f} không tìm thấy — chạy 01_generate_data.py trước")
        return []
    with f.open(encoding="utf-8") as fp:
        return json.load(fp)


def _load_observations() -> list[dict]:
    f = config.DATA_DIR / "observations.jsonl"
    if not f.exists():
        print(f"[load] ⚠️  {f} không tìm thấy — chạy 01_generate_data.py trước")
        return []
    lines = []
    with f.open(encoding="utf-8") as fp:
        for line in fp:
            line = line.strip()
            if line:
                try:
                    lines.append(json.loads(line))
                except json.JSONDecodeError as e:
                    print(f"  [WARN] Bỏ qua dòng JSONL không hợp lệ: {e}")
    return lines


def _load_memories() -> list[dict]:
    f = config.DATA_DIR / "memories.json"
    if not f.exists():
        print(f"[load] ⚠️  {f} không tìm thấy — chạy 01_generate_data.py trước")
        return []
    with f.open(encoding="utf-8") as fp:
        return json.load(fp)


# ── Push logic ─────────────────────────────────────────────────────────────────
def push_sessions(sessions: list[dict], dry_run: bool = False) -> dict[str, int]:
    print(f"\n[sessions] Push {len(sessions)} sessions...")
    ok = fail = skip = 0

    for i, session in enumerate(sessions, 1):
        payload = {
            "sessionId": session["id"],
            "project": session.get("project", config.TEST_PROJECT),
            "cwd": session.get("cwd", config.TEST_CWD),
            "agentId": session.get("agentId", config.AGENT_ID),
            "title": session.get("summary", ""),
        }
        code, body = _post("session/start", payload, dry_run=dry_run)
        if code in (200, 201):
            ok += 1
            if i % 5 == 0 or i == len(sessions):
                print(f"  [{i}/{len(sessions)}] ✅ {session['id']}")
        elif code == 0:
            fail += 1
            print(f"  [{i}/{len(sessions)}] ❌ {session['id']}: {body.get('error', '?')}")
        else:
            fail += 1
            print(f"  [{i}/{len(sessions)}] ❌ HTTP {code}: {body}")

    print(f"[sessions] Kết quả: ✅ {ok}  ❌ {fail}")
    return {"ok": ok, "fail": fail, "skip": skip}


def push_observations(
    observations: list[dict],
    dry_run: bool = False,
    batch_delay_ms: int = 100,
) -> dict[str, int]:
    print(f"\n[observations] Push {len(observations)} observations...")
    ok = fail = 0

    for i, obs in enumerate(observations, 1):
        # Build HookPayload theo đúng schema API
        payload = {
            "hookType": obs.get("hookType", "post_tool_use"),
            "sessionId": obs["sessionId"],
            "project": obs.get("project", config.TEST_PROJECT),
            "cwd": obs.get("cwd", config.TEST_CWD),
            "timestamp": obs.get("timestamp", ""),
            "data": obs.get("data", {}),
        }
        code, body = _post("observe", payload, dry_run=dry_run)
        if code in (200, 201):
            ok += 1
        else:
            fail += 1
            if fail <= 5:  # Chỉ in 5 lỗi đầu
                print(f"  [{i}] ❌ HTTP {code}: {json.dumps(body)[:200]}")

        # Progress mỗi 50 obs
        if i % 50 == 0 or i == len(observations):
            print(f"  [{i}/{len(observations)}] ✅ {ok}  ❌ {fail}")

        # Giảm tải server
        if batch_delay_ms > 0 and not dry_run:
            time.sleep(batch_delay_ms / 1000)

    print(f"[observations] Kết quả: ✅ {ok}  ❌ {fail}")
    return {"ok": ok, "fail": fail}


def push_memories(memories: list[dict], dry_run: bool = False) -> dict[str, int]:
    print(f"\n[memories] Push {len(memories)} memories...")
    ok = fail = 0

    for i, mem in enumerate(memories, 1):
        payload = {
            "content": mem["content"],
            "type": mem.get("type", "fact"),
            "concepts": mem.get("concepts", []),
            "files": mem.get("files", []),
            "project": mem.get("project", config.TEST_PROJECT),
            "agentId": mem.get("agentId", config.AGENT_ID),
            # sessionId là optional
            **({
                "sessionId": mem["sessionIds"][0]
            } if mem.get("sessionIds") else {}),
        }
        code, body = _post("remember", payload, dry_run=dry_run)
        if code in (200, 201):
            ok += 1
            if i % 5 == 0 or i == len(memories):
                print(f"  [{i}/{len(memories)}] ✅")
        else:
            fail += 1
            print(f"  [{i}/{len(memories)}] ❌ HTTP {code}: {json.dumps(body)[:200]}")

    print(f"[memories] Kết quả: ✅ {ok}  ❌ {fail}")
    return {"ok": ok, "fail": fail}


# ── Report ─────────────────────────────────────────────────────────────────────
def _save_push_report(results: dict, dry_run: bool) -> None:
    report = {
        "timestamp": __import__("datetime").datetime.utcnow().isoformat() + "Z",
        "server": config.BASE_URL,
        "dry_run": dry_run,
        "results": results,
    }
    report_file = config.DATA_DIR / "push_report.json"
    with report_file.open("w") as f:
        json.dump(report, f, indent=2)
    print(f"\n[report] Lưu tại: {report_file}")


# ── CLI ────────────────────────────────────────────────────────────────────────
def main() -> None:
    parser = argparse.ArgumentParser(
        description="Push test data lên agentmemory server"
    )
    parser.add_argument("--dry-run", action="store_true", help="Không gửi request thực")
    parser.add_argument("--sessions-only", action="store_true")
    parser.add_argument("--obs-only", action="store_true")
    parser.add_argument("--memories-only", action="store_true")
    parser.add_argument(
        "--batch-delay-ms",
        type=int,
        default=50,
        help="Delay giữa mỗi observation request (ms, default=50)",
    )
    args = parser.parse_args()

    print("=" * 60)
    print("02_push_data.py — Push dữ liệu lên server")
    print("=" * 60)
    config.print_summary()

    if not _check_server(args.dry_run):
        print("\n❌ Không thể kết nối server. Kiểm tra AGENTMEMORY_URL trong .env")
        sys.exit(1)

    all_ok = args.sessions_only or args.obs_only or args.memories_only
    results: dict[str, dict] = {}

    # --- Sessions ---
    if not args.obs_only and not args.memories_only:
        sessions = _load_sessions()
        if sessions:
            results["sessions"] = push_sessions(sessions, dry_run=args.dry_run)

    # --- Observations ---
    if not args.sessions_only and not args.memories_only:
        observations = _load_observations()
        if observations:
            results["observations"] = push_observations(
                observations,
                dry_run=args.dry_run,
                batch_delay_ms=args.batch_delay_ms,
            )

    # --- Memories ---
    if not args.sessions_only and not args.obs_only:
        memories = _load_memories()
        if memories:
            results["memories"] = push_memories(memories, dry_run=args.dry_run)

    _save_push_report(results, args.dry_run)

    # Tổng kết
    total_ok = sum(r.get("ok", 0) for r in results.values())
    total_fail = sum(r.get("fail", 0) for r in results.values())
    print(f"\n{'=' * 60}")
    print(f"TỔNG KẾT: ✅ {total_ok} thành công  ❌ {total_fail} thất bại")
    if total_fail > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
