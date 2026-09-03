#!/usr/bin/env python3
"""
run_all.py — Run the full seed pipeline: generate → load → verify.

Usage:
    python run_all.py [--skip-generate] [--skip-load] [--skip-verify] [--verbose]
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).parent


def run_step(script: str, extra_args: list[str] = []) -> bool:
    cmd = [sys.executable, str(HERE / script)] + extra_args
    print(f"\n{'━' * 60}")
    print(f"  Running: {' '.join(cmd)}")
    print(f"{'━' * 60}")
    result = subprocess.run(cmd, cwd=str(HERE))
    return result.returncode == 0


def main() -> None:
    parser = argparse.ArgumentParser(description="Run full VNP Memory seed pipeline")
    parser.add_argument("--skip-generate", action="store_true")
    parser.add_argument("--skip-load", action="store_true")
    parser.add_argument("--skip-verify", action="store_true")
    parser.add_argument("--verbose", action="store_true", help="Pass --verbose to verify step")
    args = parser.parse_args()

    ok = True

    if not args.skip_generate:
        ok = run_step("01_generate_data.py") and ok

    if not args.skip_load:
        ok = run_step("02_load_data.py") and ok

    if not args.skip_verify:
        extra = ["--verbose"] if args.verbose else []
        ok = run_step("03_verify_data.py", extra) and ok

    print(f"\n{'━' * 60}")
    if ok:
        print(" ✅ Seed pipeline completed successfully!")
    else:
        print(" ⚠  Seed pipeline completed with errors.")
    print(f"{'━' * 60}\n")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
