#!/usr/bin/env python3
"""
Compare Python vs Go parser implementations by running tests with separate output directories.
"""

import os
import sys
import subprocess
import json
import shutil
from pathlib import Path
import pyarrow.parquet as pq
import pandas as pd
from typing import Dict, Any

def setup_test_environment(test_name: str, use_go_modules: bool) -> Dict[str, str]:
    """Setup isolated test environment for Python or Go test."""

    # Create unique paths for this test run in tests directory
    test_base = Path(__file__).parent / "tests" / "test_output" / f"comparison_{test_name}"
    if test_base.exists():
        shutil.rmtree(test_base)
    test_base.mkdir(parents=True, exist_ok=True)
    
    # Setup paths
    paths = {
        "test_dir": str(test_base),
        "db_path": str(test_base / "job_queue.db"),
        "analytics_path": str(test_base / "analytics-output"),
        "logs_path": str(test_base / "logs"),
        "config_path": str(test_base / "worker_config.yaml")
    }
    
    # Create directories
    Path(paths["analytics_path"]).mkdir(parents=True)
    Path(paths["logs_path"]).mkdir(parents=True)
    
    # Create worker config
    config = {
        "processing": {
            "job_control": {
                "type": "sqlite",
                "path": paths["db_path"],
                "max_retries": 3,
                "claim_timeout": 300,
                "heartbeat_interval": 30,
                "stale_claim_timeout": 600
            }
        },
        "analytics": {
            "outputs": [{
                "type": "parquet",
                "path": paths["analytics_path"]
            }]
        },
        "embeddings": {
            "generator": {
                "type": "fastembed",
                "model": "sentence-transformers/all-MiniLM-L6-v2",
                "dimensions": 384
            }
        },
        "logging": {
            "level": "INFO",
            "file": str(Path(paths["logs_path"]) / "worker.log")
        },
        "content_sources": [
            {
                "name": "file-docs",
                "type": "file",
                "path": str(Path(__file__).parent / "tests" / "assets"),
                "extensions": [".txt", ".csv", ".json", ".xml", ".html", ".md", ".docx", ".xlsx"]
            }
        ]
    }
    
    # Write config file
    import yaml
    with open(paths["config_path"], "w") as f:
        yaml.dump(config, f)
    
    # Set environment variable
    os.environ["USE_GO_MODULES"] = "true" if use_go_modules else "false"
    
    return paths

def run_worker_test(paths: Dict[str, str], max_docs: int = None) -> bool:
    """Run worker test with given configuration."""

    try:
        # Add src directory to Python path
        import sys
        src_path = Path(__file__).parent / "src"
        if str(src_path) not in sys.path:
            sys.path.insert(0, str(src_path))

        # Initialize database directly using SQLite module
        print(f"Initializing database at {paths['db_path']}")
        from go_doc_go.shared.simple_job_control.sqlite import SimpleSQLiteJobControlDB

        # Create a mock config object with the required method
        class MockConfig:
            def __init__(self, db_path):
                self.db_path = db_path

            def get_job_control_config(self):
                return {
                    'path': self.db_path,
                    'claim_timeout': 300,
                    'heartbeat_interval': 30,
                    'max_retries': 3,
                    'init_retries': 5,
                    'init_retry_delay': 0.1
                }

        # Create job control and initialize
        config = MockConfig(paths["db_path"])
        job_control = SimpleSQLiteJobControlDB(config)
        job_control.initialize_schema()

        # Queue documents using job control
        print(f"Queuing documents...")
        from go_doc_go.content_source.file import FileContentSource

        # Queue file documents from test assets
        file_source = FileContentSource(
            config={"base_path": str(Path(__file__).parent / "tests" / "assets")}
        )
        docs_queued = 0
        for doc in file_source.list_documents():
            # Use enqueue_document method with correct parameters
            job_control.enqueue_document(doc["id"], "file-docs", doc)
            docs_queued += 1
            if max_docs and docs_queued >= max_docs:
                break

        print(f"Queued {docs_queued} documents")

        # Run worker
        print(f"Running worker to process documents...")
        worker_cmd = [
            sys.executable, "-m", "go_doc_go.cli.worker",
            "--config", paths["config_path"]
        ]

        # Only add max-documents if specified
        if max_docs is not None:
            worker_cmd.extend(["--max-documents", str(max_docs)])

        result = subprocess.run(worker_cmd, capture_output=True, text=True, timeout=300,
                              env={**os.environ, "PYTHONPATH": str(src_path)})
        if result.returncode != 0:
            print(f"Worker failed: {result.stderr}")
            return False

        return True

    except subprocess.TimeoutExpired:
        print("Worker test timed out")
        return False
    except Exception as e:
        print(f"Error running test: {e}")
        import traceback
        traceback.print_exc()
        return False

def analyze_results(paths: Dict[str, str]) -> Dict[str, Any]:
    """Analyze parquet output from test run."""
    
    analytics_dir = Path(paths["analytics_path"])
    
    if not analytics_dir.exists():
        return {"error": "No analytics output found"}
    
    results = {
        "documents": 0,
        "elements": 0,
        "embeddings": 0,
        "relationships": 0,
        "doc_types": {},
        "sources": {}
    }
    
    # Count documents
    for parquet_file in analytics_dir.glob("**/documents/**/*.parquet"):
        try:
            table = pq.read_table(str(parquet_file))
            df = table.to_pandas()
            results["documents"] += len(df)
            
            # Count by source
            if "source" in df.columns:
                for source in df["source"].value_counts().items():
                    results["sources"][source[0]] = results["sources"].get(source[0], 0) + source[1]
                    
            # Parse doc_types from metadata
            if "metadata" in df.columns:
                for metadata in df["metadata"]:
                    try:
                        if isinstance(metadata, str):
                            meta_dict = json.loads(metadata)
                        else:
                            meta_dict = metadata
                        doc_type = meta_dict.get("doc_type", "unknown")
                        results["doc_types"][doc_type] = results["doc_types"].get(doc_type, 0) + 1
                    except:
                        pass
        except Exception as e:
            print(f"Error reading documents: {e}")
    
    # Count elements
    for parquet_file in analytics_dir.glob("**/elements/**/*.parquet"):
        try:
            table = pq.read_table(str(parquet_file))
            results["elements"] += len(table)
        except:
            pass
            
    # Count embeddings
    for parquet_file in analytics_dir.glob("**/embeddings/**/*.parquet"):
        try:
            table = pq.read_table(str(parquet_file))
            results["embeddings"] += len(table)
        except:
            pass
            
    # Count relationships
    for parquet_file in analytics_dir.glob("**/relationships/**/*.parquet"):
        try:
            table = pq.read_table(str(parquet_file))
            results["relationships"] += len(table)
        except:
            pass
            
    return results

def main():
    """Run comparison test."""
    
    print("=" * 60)
    print("PYTHON VS GO PARSER COMPARISON TEST")
    print("=" * 60)
    
    # Test configuration - remove limit to test all documents
    max_docs = None
    
    # Run Python test
    print("\n1. Running Python Parser Test...")
    print("-" * 40)
    python_paths = setup_test_environment("python", use_go_modules=False)
    python_success = run_worker_test(python_paths, max_docs)
    
    if not python_success:
        print("Python test failed!")
        return 1
        
    python_results = analyze_results(python_paths)
    
    # Run Go test
    print("\n2. Running Go Parser Test...")
    print("-" * 40)
    go_paths = setup_test_environment("go", use_go_modules=True)
    go_success = run_worker_test(go_paths, max_docs)
    
    if not go_success:
        print("Go test failed!")
        return 1
        
    go_results = analyze_results(go_paths)
    
    # Compare results
    print("\n" + "=" * 60)
    print("RESULTS COMPARISON")
    print("=" * 60)
    
    print("\nDocument Processing Counts:")
    print("-" * 40)
    print(f"{'Metric':<20} {'Python':>10} {'Go':>10} {'Match':>10}")
    print("-" * 40)
    
    for metric in ["documents", "elements", "embeddings", "relationships"]:
        py_val = python_results.get(metric, 0)
        go_val = go_results.get(metric, 0)
        match = "✓" if py_val == go_val else f"Δ={go_val-py_val:+d}"
        print(f"{metric:<20} {py_val:>10} {go_val:>10} {match:>10}")
    
    print("\nDocuments by Source:")
    print("-" * 40)
    all_sources = set(python_results.get("sources", {}).keys()) | set(go_results.get("sources", {}).keys())
    for source in sorted(all_sources):
        py_val = python_results.get("sources", {}).get(source, 0)
        go_val = go_results.get("sources", {}).get(source, 0)
        match = "✓" if py_val == go_val else f"Δ={go_val-py_val:+d}"
        print(f"{source:<20} {py_val:>10} {go_val:>10} {match:>10}")
    
    print("\nDocument Types:")
    print("-" * 40)
    all_types = set(python_results.get("doc_types", {}).keys()) | set(go_results.get("doc_types", {}).keys())
    for doc_type in sorted(all_types):
        py_val = python_results.get("doc_types", {}).get(doc_type, 0)
        go_val = go_results.get("doc_types", {}).get(doc_type, 0)
        match = "✓" if py_val == go_val else f"Δ={go_val-py_val:+d}"
        print(f"{doc_type:<20} {py_val:>10} {go_val:>10} {match:>10}")
    
    # Functional equivalence assessment
    print("\n" + "=" * 60)
    print("FUNCTIONAL EQUIVALENCE ASSESSMENT")
    print("=" * 60)
    
    doc_match = python_results["documents"] == go_results["documents"]
    elem_variance = abs(python_results["elements"] - go_results["elements"]) / max(python_results["elements"], 1)
    emb_variance = abs(python_results["embeddings"] - go_results["embeddings"]) / max(python_results["embeddings"], 1)
    rel_variance = abs(python_results["relationships"] - go_results["relationships"]) / max(python_results["relationships"], 1)
    
    if doc_match and elem_variance < 0.05 and emb_variance < 0.05 and rel_variance < 0.05:
        print("✓ FUNCTIONALLY EQUIVALENT - Implementations produce similar results")
        print(f"  Element variance: {elem_variance:.1%}")
        print(f"  Embedding variance: {emb_variance:.1%}")
        print(f"  Relationship variance: {rel_variance:.1%}")
    else:
        print("✗ SIGNIFICANT DIFFERENCES DETECTED")
        if not doc_match:
            print(f"  - Document count mismatch: {python_results['documents']} vs {go_results['documents']}")
        if elem_variance >= 0.05:
            print(f"  - Element variance too high: {elem_variance:.1%}")
        if emb_variance >= 0.05:
            print(f"  - Embedding variance too high: {emb_variance:.1%}")
        if rel_variance >= 0.05:
            print(f"  - Relationship variance too high: {rel_variance:.1%}")
    
    print("\nTest data preserved at:")
    print(f"  Python: {python_paths['test_dir']}")
    print(f"    SQLite DB: {python_paths['db_path']}")
    print(f"    Analytics: {python_paths['analytics_path']}")
    print(f"  Go: {go_paths['test_dir']}")
    print(f"    SQLite DB: {go_paths['db_path']}")
    print(f"    Analytics: {go_paths['analytics_path']}")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())
