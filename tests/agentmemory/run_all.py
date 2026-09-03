"""
run_all.py — Runner tổng hợp: chạy toàn bộ pipeline test

Pipeline:
  Step 1: Sinh dữ liệu  (01_generate_data.py)
  Step 2: Push lên server (02_push_data.py)
  Step 3: Kiểm thử     (03_test_memory.py)
  Step 4: In báo cáo tổng kết

Chạy:
  cd tests/agentmemory
  python run_all.py
  python run_all.py --skip-generate   # Dùng data đã có
  python run_all.py --skip-push       # Không push lại
  python run_all.py --dry-push        # Push nhưng dry-run
  python run_all.py --suite health    # Chỉ test health sau khi generate+push
"""
from __future__ import annotations

import argparse
import subprocess
import sys
import time
from pathlib import Path

import config

HERE = Path(__file__).parent


def _run_step(step_name: str, script: str, extra_args: list[str] | None = None) -> int:
    """Chạy một step, trả về exit code."""
    cmd = [sys.executable, str(HERE / script)] + (extra_args or [])
    print(f"\n{'─' * 60}")
    print(f"STEP: {step_name}")
    print(f"CMD : {' '.join(cmd)}")
    print(f"{'─' * 60}")

    start = time.time()
    result = subprocess.run(cmd, cwd=str(HERE))
    elapsed = round(time.time() - start, 1)

    status = "✅ OK" if result.returncode == 0 else f"❌ FAILED (exit {result.returncode})"
    print(f"\n[{step_name}] {status} — {elapsed}s")
    return result.returncode


def main() -> None:
    parser = argparse.ArgumentParser(description="Chạy toàn bộ pipeline test agentmemory")
    parser.add_argument("--skip-generate", action="store_true", help="Bỏ qua bước sinh dữ liệu")
    parser.add_argument("--skip-push", action="store_true", help="Bỏ qua bước push data")
    parser.add_argument("--dry-push", action="store_true", help="Push ở chế độ dry-run")
    parser.add_argument("--skip-graph", action="store_true", help="Bỏ qua graph extraction test")
    parser.add_argument(
        "--suite",
        default="all",
        choices=["health", "search", "remember", "lifecycle", "replay", "validation", "all"],
        help="Test suite cần chạy (default: all)",
    )
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    args = parser.parse_args()

    print("=" * 60)
    print("run_all.py — agentmemory Test Pipeline")
    print("=" * 60)
    config.print_summary()

    results = {}

    # Step 1: Generate
    if not args.skip_generate:
        rc = _run_step("1. Sinh dữ liệu", "01_generate_data.py")
        results["generate"] = rc
        if rc != 0:
            print("\n❌ Sinh dữ liệu thất bại. Dừng pipeline.")
            sys.exit(rc)
    else:
        print("\n[skip] Bỏ qua bước sinh dữ liệu")

    # Step 2: Push
    if not args.skip_push:
        push_args = ["--batch-delay-ms", "30"]
        if args.dry_push:
            push_args.append("--dry-run")
        rc = _run_step("2. Push data lên server", "02_push_data.py", push_args)
        results["push"] = rc
        if rc != 0:
            print("\n⚠️  Push có lỗi — tiếp tục test với data hiện có trên server")
    else:
        print("\n[skip] Bỏ qua bước push data")

    # Step 3: Test
    test_args = ["--suite", args.suite]
    if args.verbose:
        test_args.append("--verbose")
    rc = _run_step("3. Kiểm thử memory", "03_test_memory.py", test_args)
    results["test"] = rc

    # Step 4: Graph extraction test
    if not args.skip_graph:
        graph_args = ["--step", "all"]
        if args.verbose:
            graph_args.append("--verbose")
        rc = _run_step("4. Kiểm thử Knowledge Graph", "04_test_graph.py", graph_args)
        results["graph"] = rc
        if rc != 0:
            print("\n⚠️  Graph tests có lỗi — kiểm tra GRAPH_EXTRACTION_ENABLED và LLM config")
    else:
        print("\n[skip] Bỏ qua graph test")

    # Tổng kết
    print("\n" + "=" * 60)
    print("PIPELINE KẾT THÚC")
    print("=" * 60)
    for step, code in results.items():
        icon = "✅" if code == 0 else "❌"
        print(f"  {icon} {step}: exit {code}")

    # Báo cáo file
    report_file = config.DATA_DIR / "test_results.json"
    if report_file.exists():
        print(f"\n📊 Báo cáo test: {report_file}")

    # Exit code = test result
    sys.exit(results.get("test", 0))


if __name__ == "__main__":
    main()
