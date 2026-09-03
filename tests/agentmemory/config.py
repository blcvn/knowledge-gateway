"""
config.py — Đọc cấu hình từ .env

Tìm .env theo thứ tự:
  1. Đường dẫn trong biến môi trường ENV_FILE
  2. tests/agentmemory/.env  (thư mục chứa script này)
  3. .env.example            (fallback)
"""
from __future__ import annotations

import os
from pathlib import Path
from typing import Optional

# ── Auto-locate .env ───────────────────────────────────────────────────────────
_HERE = Path(__file__).parent          # tests/agentmemory/

def _find_env_file() -> Optional[Path]:
    """Tìm file .env theo thứ tự ưu tiên."""
    # 1. Biến môi trường
    if env_path := os.getenv("ENV_FILE"):
        p = Path(env_path)
        if p.exists():
            return p

    # 2. Cùng thư mục với script
    candidates = [
        _HERE / ".env",
        _HERE / ".env.example",
    ]
    for c in candidates:
        if c.exists():
            return c

    return None


def _load_dotenv(path: Optional[Path]) -> None:
    """Parse và nạp file .env vào os.environ (không ghi đè giá trị có sẵn)."""
    if path is None:
        return
    with path.open(encoding="utf-8") as f:
        for raw in f:
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            if "=" not in line:
                continue
            key, _, value = line.partition("=")
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            # Không ghi đè biến đã có trong shell
            if key and key not in os.environ:
                os.environ[key] = value


# Nạp .env ngay khi import module
_env_file = _find_env_file()
_load_dotenv(_env_file)


# ── Typed config accessors ─────────────────────────────────────────────────────
def get(key: str, default: str = "") -> str:
    return os.environ.get(key, default)


def require(key: str) -> str:
    """Lấy biến bắt buộc; raise nếu không có."""
    val = os.environ.get(key)
    if not val:
        raise EnvironmentError(
            f"Biến môi trường bắt buộc '{key}' chưa được set.\n"
            f"  → Tạo file {_HERE / '.env'} từ mẫu {_HERE / '.env.example'}"
        )
    return val


# ── Giá trị cấu hình cụ thể ───────────────────────────────────────────────────
BASE_URL:       str = get("AGENTMEMORY_URL", "http://localhost:3111")
SECRET:         str = get("AGENTMEMORY_SECRET", "")
TEST_PROJECT:   str = get("TEST_PROJECT", "vnp-test-project")
TEST_CWD:       str = get("TEST_CWD", "/tmp/vnp-test")
AGENT_ID:       str = get("AGENT_ID", "test-agent-python")

# Thư mục output (tương đối so với tests/agentmemory/)
_data_dir_raw:  str = get("TEST_DATA_DIR", "data")
DATA_DIR:       Path = (_HERE / _data_dir_raw).resolve()

# Tham số sinh dữ liệu
GEN_SESSION_COUNT:   int = int(get("GEN_SESSION_COUNT", "5"))
GEN_OBS_PER_SESSION: int = int(get("GEN_OBS_PER_SESSION", "20"))
GEN_MEMORY_COUNT:    int = int(get("GEN_MEMORY_COUNT", "10"))

# HTTP headers
def auth_headers() -> dict[str, str]:
    """Trả về dict headers kèm Authorization nếu SECRET được set."""
    headers = {"Content-Type": "application/json", "Accept": "application/json"}
    if SECRET:
        headers["Authorization"] = f"Bearer {SECRET}"
    return headers


def print_summary() -> None:
    print(f"[config] Server : {BASE_URL}")
    print(f"[config] Secret : {'***' if SECRET else '(không set — local mode)'}")
    print(f"[config] Project: {TEST_PROJECT}")
    print(f"[config] DataDir: {DATA_DIR}")
    print(f"[config] EnvFile: {_env_file or '(không tìm thấy)'}")


if __name__ == "__main__":
    print_summary()
