#!/usr/bin/env python3
"""
Test runner for PostgreSQL queue tests using Docker.
This manages the PostgreSQL instance completely, making it a unit test.
"""

import os
import subprocess
import sys
import time
from pathlib import Path

# Add src to path
src_path = Path(__file__).parent.parent.parent / "src"
sys.path.insert(0, str(src_path))


def wait_for_postgres(max_attempts=30):
    """Wait for PostgreSQL to be ready."""
    compose_path = Path(__file__).parent.parent.parent / "test_containers" / "postgres"
    for i in range(max_attempts):
        try:
            result = subprocess.run([
                "docker-compose", "-f", "compose.yaml", "exec", "-T", "postgres-test",
                "pg_isready", "-U", "testuser", "-d", "go_doc_go_test"
            ], capture_output=True, cwd=compose_path)
            
            if result.returncode == 0:
                print(f"✅ PostgreSQL ready after {i+1} attempts")
                return True
                
        except Exception as e:
            pass
        
        print(f"⏳ Waiting for PostgreSQL... ({i+1}/{max_attempts})")
        time.sleep(2)
    
    return False


def main():
    """Run PostgreSQL queue tests with Docker."""
    test_dir = Path(__file__).parent
    compose_path = Path(__file__).parent.parent.parent / "test_containers" / "postgres"
    
    print("🐳 Starting Docker PostgreSQL for testing...")
    
    # Start PostgreSQL
    subprocess.run([
        "docker-compose", "-f", "compose.yaml", "up", "-d"
    ], cwd=compose_path, check=True)
    
    try:
        # Wait for PostgreSQL to be ready
        if not wait_for_postgres():
            print("❌ PostgreSQL failed to start")
            return False
        
        # Set test environment
        env = os.environ.copy()
        env.update({
            "TEST_PG_HOST": "localhost",
            "TEST_PG_PORT": "15432", 
            "TEST_PG_DB": "go_doc_go_test",
            "TEST_PG_USER": "testuser",
            "TEST_PG_PASSWORD": "testpass",
            "PYTHONPATH": str(Path(__file__).parent.parent.parent / "src")
        })
        
        # Run atomic claiming test
        print("🔧 Running atomic claiming test...")
        result = subprocess.run([
            sys.executable, "-m", "pytest", 
            "test_work_queue.py::TestWorkQueueIntegration::test_atomic_claiming",
            "-xvs"
        ], env=env, cwd=test_dir)
        
        if result.returncode == 0:
            print("✅ Atomic claiming test PASSED")
            atomic_passed = True
        else:
            print("❌ Atomic claiming test FAILED")
            atomic_passed = False
        
        # Run concurrent throughput test
        print("🔧 Running concurrent throughput test...")
        result = subprocess.run([
            sys.executable, "-m", "pytest",
            "test_work_queue.py::TestWorkQueuePerformance::test_concurrent_throughput", 
            "-xvs"
        ], env=env, cwd=test_dir)
        
        if result.returncode == 0:
            print("✅ Concurrent throughput test PASSED")
            throughput_passed = True
        else:
            print("❌ Concurrent throughput test FAILED")
            throughput_passed = False
        
        # Summary
        if atomic_passed and throughput_passed:
            print("🎉 All PostgreSQL queue tests PASSED!")
            return True
        else:
            print("💥 Some PostgreSQL queue tests FAILED!")
            return False
            
    finally:
        # Cleanup
        print("🧹 Cleaning up Docker PostgreSQL...")
        subprocess.run([
            "docker-compose", "-f", "compose.yaml", "down", "-v"
        ], cwd=compose_path)


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)