#!/usr/bin/env python3
"""Isolated Python-only test with dedicated output directory."""

import subprocess
import sys
import os
import time

# Kill any existing Python processes
os.system("pkill -9 -f python 2>/dev/null")
os.system("pkill -9 -f pytest 2>/dev/null")
time.sleep(3)

# Clean setup
os.system("rm -rf tests/test_output/python_clean")
os.system("mkdir -p tests/test_output/python_clean")

print("=== PYTHON ONLY TEST (CLEAN) ===")
start_time = time.time()

env = os.environ.copy()
env["USE_GO_MODULES"] = "false"

cmd = [
    sys.executable, "-m", "pytest",
    "tests/test_worker_cli.py::TestWorkerCLI::test_worker_cli_process_documents",
    "-v", "--tb=short"
]

result = subprocess.run(cmd, env=env, capture_output=True, text=True)
end_time = time.time()

print(f"Exit code: {result.returncode}")
print(f"Time: {end_time - start_time:.2f}s")
print("STDOUT:", result.stdout)
if result.stderr:
    print("STDERR:", result.stderr)

# Count parquet files if test succeeded
if result.returncode == 0:
    parquet_count = subprocess.run(
        ["find", "tests/test_output", "-name", "*.parquet"],
        capture_output=True, text=True
    )
    count = len([line for line in parquet_count.stdout.strip().split('\n') if line])
    print(f"Parquet files generated: {count}")