#!/usr/bin/env python3
"""
Controlled Performance Test: Go vs Python Parsers

This test processes the exact same set of local documents to provide
a fair comparison between Go and Python parser implementations.
"""

import os
import sys
import time
import subprocess
import tempfile
import shutil
import yaml
from pathlib import Path
from typing import Dict, List, Any

class ControlledPerformanceTest:
    def __init__(self):
        self.project_root = Path(__file__).parent
        self.assets_dir = self.project_root / "tests" / "assets"
        self.test_files = self._get_test_files()

    def _get_test_files(self) -> List[Path]:
        """Get list of test files to process."""
        extensions = [".md", ".txt", ".xml", ".pptx", ".csv", ".docx", ".xlsx", ".pdf", ".htm"]
        files = []
        for ext in extensions:
            files.extend(self.assets_dir.glob(f"*{ext}"))
        return sorted(files)

    def create_test_config(self, test_dir: Path, include_web: bool = False) -> Path:
        """Create test configuration for controlled processing."""
        config = {
            'embedding': {
                'enabled': True,
                'provider': 'fastembed',
                'model': 'sentence-transformers/all-MiniLM-L6-v2',
                'dimensions': 384,
                'chunk_size': 512,
                'overlap': 128,
                'contextual': True,
                'predecessor_count': 1,
                'successor_count': 1
            },
            'content_sources': [
                {
                    'name': 'file-docs',
                    'type': 'file',
                    'base_path': str(self.assets_dir),
                    'file_pattern': '**/*',
                    'include_extensions': ['md', 'txt', 'xml', 'pptx', 'csv', 'docx', 'xlsx', 'pdf', 'htm'],
                    'watch_for_changes': False,
                    'max_link_depth': 0  # No link following for pure local test
                }
            ],
            'relationship_detection': {
                'enabled': True,
                'structural': True,
                'semantic': False,
                'cross_document_semantic': {
                    'similarity_threshold': 0.65
                }
            },
            'processing': {
                'batch_size': 100,
                'max_workers': 1,  # Single worker for controlled test
                'timeout_seconds': 600,  # 10 minute timeout
                'job_control': {
                    'backend': 'sqlite',
                    'path': str(test_dir / 'job_queue.db'),
                    'claim_timeout': 300,
                    'heartbeat_interval': 30,
                    'max_retries': 3
                }
            },
            'analytics': {
                'enabled': True,
                'outputs': [{
                    'type': 'parquet',
                    'path': str(test_dir / 'analytics-output'),
                    'partitioning': ['date', 'source']
                }]
            },
            'logging': {
                'level': 'INFO',
                'file': str(test_dir / 'logs' / 'worker.log')
            }
        }

        # Create log directory
        (test_dir / 'logs').mkdir(parents=True, exist_ok=True)

        # Write config
        config_path = test_dir / 'test_config.yaml'
        with open(config_path, 'w') as f:
            yaml.dump(config, f, default_flow_style=False)

        return config_path

    def run_worker_test(self, config_path: Path, use_go_modules: bool, test_name: str) -> Dict[str, Any]:
        """Run worker with specified configuration."""
        print(f"\n=== Running {test_name} ===")
        print(f"Files to process: {len(self.test_files)}")
        print(f"Go modules: {'enabled' if use_go_modules else 'disabled'}")

        # Set up environment
        env = os.environ.copy()
        env['USE_GO_MODULES'] = 'true' if use_go_modules else 'false'
        env['PYTHONPATH'] = 'src'

        # Run worker command
        cmd = [
            'python', '-m', 'go_doc_go.cli.worker',
            '--config', str(config_path),
            '--max-documents', str(len(self.test_files) + 5),  # Process all files plus some buffer
            '--worker-id', f'controlled-test-{test_name}'
        ]

        start_time = time.time()

        try:
            result = subprocess.run(
                cmd,
                cwd=self.project_root,
                env=env,
                capture_output=True,
                text=True,
                timeout=600  # 10 minute timeout
            )

            end_time = time.time()
            duration = end_time - start_time

            return {
                'success': result.returncode == 0,
                'duration': duration,
                'stdout': result.stdout,
                'stderr': result.stderr,
                'returncode': result.returncode
            }

        except subprocess.TimeoutExpired as e:
            end_time = time.time()
            duration = end_time - start_time

            return {
                'success': False,
                'duration': duration,
                'stdout': (e.stdout or b'').decode('utf-8'),
                'stderr': (e.stderr or b'').decode('utf-8'),
                'returncode': None,
                'timeout': True
            }

    def count_output_files(self, test_dir: Path) -> Dict[str, int]:
        """Count generated Parquet files by type."""
        analytics_dir = test_dir / 'analytics-output'
        if not analytics_dir.exists():
            return {'documents': 0, 'elements': 0, 'relationships': 0, 'embeddings': 0, 'total': 0}

        counts = {
            'documents': len(list(analytics_dir.glob('**/documents/**/*.parquet'))),
            'elements': len(list(analytics_dir.glob('**/elements/**/*.parquet'))),
            'relationships': len(list(analytics_dir.glob('**/relationships/**/*.parquet'))),
            'embeddings': len(list(analytics_dir.glob('**/embeddings/**/*.parquet')))
        }
        counts['total'] = sum(counts.values())

        return counts

    def run_comparison_test(self):
        """Run controlled comparison between Go and Python parsers."""
        print("Controlled Performance Test: Go vs Python Parsers")
        print("=" * 60)
        print(f"Test files: {len(self.test_files)}")
        for f in self.test_files:
            print(f"  - {f.name}")

        results = {}

        # Test 1: Go modules
        with tempfile.TemporaryDirectory(prefix="go_test_") as go_dir:
            go_test_dir = Path(go_dir)
            go_config = self.create_test_config(go_test_dir)

            go_result = self.run_worker_test(go_config, True, "Go-Modules")
            go_counts = self.count_output_files(go_test_dir)

            results['go'] = {
                'result': go_result,
                'counts': go_counts,
                'test_dir': str(go_test_dir)
            }

        # Test 2: Python parsers
        with tempfile.TemporaryDirectory(prefix="python_test_") as python_dir:
            python_test_dir = Path(python_dir)
            python_config = self.create_test_config(python_test_dir)

            python_result = self.run_worker_test(python_config, False, "Python-Parsers")
            python_counts = self.count_output_files(python_test_dir)

            results['python'] = {
                'result': python_result,
                'counts': python_counts,
                'test_dir': str(python_test_dir)
            }

        # Print comparison results
        self.print_comparison(results)

        return results

    def print_comparison(self, results: Dict[str, Any]):
        """Print detailed comparison results."""
        print("\n" + "="*80)
        print("CONTROLLED PERFORMANCE COMPARISON RESULTS")
        print("="*80)

        go_res = results['go']
        py_res = results['python']

        print(f"\n{'Metric':<30} {'Go Modules':<20} {'Python Parsers':<20} {'Ratio':<15}")
        print("-" * 85)

        # Duration comparison
        go_duration = go_res['result']['duration']
        py_duration = py_res['result']['duration']
        duration_ratio = go_duration / py_duration if py_duration > 0 else float('inf')

        print(f"{'Processing Time':<30} {go_duration:.2f}s{'':<12} {py_duration:.2f}s{'':<12} {duration_ratio:.2f}x{'':<10}")

        # Success status
        go_success = "✓" if go_res['result']['success'] else "✗"
        py_success = "✓" if py_res['result']['success'] else "✗"
        print(f"{'Success Status':<30} {go_success:<20} {py_success:<20} {'-':<15}")

        # File counts
        for file_type in ['documents', 'elements', 'relationships', 'embeddings', 'total']:
            go_count = go_res['counts'][file_type]
            py_count = py_res['counts'][file_type]
            ratio = go_count / py_count if py_count > 0 else float('inf')

            print(f"{file_type.title() + ' Files':<30} {go_count:<20} {py_count:<20} {ratio:.2f}x{'':<10}")

        # Processing rate (files per second)
        if go_duration > 0 and py_duration > 0:
            go_rate = go_res['counts']['documents'] / go_duration
            py_rate = py_res['counts']['documents'] / py_duration
            rate_ratio = py_rate / go_rate if go_rate > 0 else float('inf')

            print(f"{'Processing Rate (docs/sec)':<30} {go_rate:.2f}{'':<16} {py_rate:.2f}{'':<16} {rate_ratio:.2f}x{'':<10}")

        # Error analysis
        print(f"\n{'Error Analysis':<30}")
        print("-" * 30)

        if not go_res['result']['success']:
            print(f"Go Modules Error: {go_res['result'].get('stderr', 'Unknown error')[:100]}...")

        if not py_res['result']['success']:
            print(f"Python Parsers Error: {py_res['result'].get('stderr', 'Unknown error')[:100]}...")

if __name__ == '__main__':
    test = ControlledPerformanceTest()
    results = test.run_comparison_test()