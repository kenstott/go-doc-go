#!/usr/bin/env python3
"""
Debug Go Parser Issues - Test each document type individually
"""

import os
import sys
import subprocess
import tempfile
import yaml
import time
from pathlib import Path

def test_single_document(doc_path, use_go_modules=True):
    """Test parsing a single document with Go or Python parsers."""

    with tempfile.TemporaryDirectory() as temp_dir:
        temp_path = Path(temp_dir)

        # Create test config for single file
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
            'content_sources': [{
                'name': 'test-file',
                'type': 'file',
                'base_path': str(doc_path.parent),
                'file_pattern': doc_path.name,
                'include_extensions': [doc_path.suffix[1:]],
                'watch_for_changes': False,
                'max_link_depth': 0
            }],
            'relationship_detection': {
                'enabled': True,
                'structural': True,
                'semantic': False
            },
            'processing': {
                'batch_size': 1,
                'max_workers': 1,
                'timeout_seconds': 60,
                'job_control': {
                    'backend': 'sqlite',
                    'path': str(temp_path / 'job_queue.db'),
                    'claim_timeout': 60,
                    'heartbeat_interval': 10,
                    'max_retries': 1
                }
            },
            'analytics': {
                'enabled': True,
                'outputs': [{
                    'type': 'parquet',
                    'path': str(temp_path / 'analytics-output'),
                    'partitioning': ['date', 'source']
                }]
            },
            'logging': {
                'level': 'DEBUG',
                'file': str(temp_path / 'debug.log')
            }
        }

        # Create directories
        (temp_path / 'logs').mkdir(exist_ok=True)

        # Write config
        config_path = temp_path / 'test_config.yaml'
        with open(config_path, 'w') as f:
            yaml.dump(config, f, default_flow_style=False)

        # Run worker
        env = os.environ.copy()
        env['USE_GO_MODULES'] = 'true' if use_go_modules else 'false'
        env['PYTHONPATH'] = 'src'

        cmd = [
            'python', '-m', 'go_doc_go.cli.worker',
            '--config', str(config_path),
            '--max-documents', '1',
            '--worker-id', f'debug-{doc_path.stem}'
        ]

        try:
            result = subprocess.run(
                cmd,
                cwd=Path(__file__).parent,
                env=env,
                capture_output=True,
                text=True,
                timeout=60
            )

            # Count output files
            analytics_path = temp_path / 'analytics-output'
            file_counts = {
                'documents': len(list(analytics_path.glob('**/documents/**/*.parquet'))),
                'elements': len(list(analytics_path.glob('**/elements/**/*.parquet'))),
                'relationships': len(list(analytics_path.glob('**/relationships/**/*.parquet'))),
                'embeddings': len(list(analytics_path.glob('**/embeddings/**/*.parquet')))
            }

            # Read debug log
            debug_log = ""
            log_path = temp_path / 'debug.log'
            if log_path.exists():
                with open(log_path, 'r') as f:
                    debug_log = f.read()

            return {
                'success': result.returncode == 0,
                'file_counts': file_counts,
                'stdout': result.stdout,
                'stderr': result.stderr,
                'debug_log': debug_log,
                'duration': None
            }

        except subprocess.TimeoutExpired as e:
            return {
                'success': False,
                'file_counts': {'documents': 0, 'elements': 0, 'relationships': 0, 'embeddings': 0},
                'stdout': (e.stdout or b'').decode('utf-8'),
                'stderr': (e.stderr or b'').decode('utf-8'),
                'debug_log': '',
                'timeout': True
            }

def main():
    """Test all document types with Go modules to identify failures."""

    project_root = Path(__file__).parent
    assets_dir = project_root / 'tests' / 'assets'

    # Get all test files
    test_files = []
    for pattern in ['*.md', '*.txt', '*.xml', '*.csv', '*.docx', '*.xlsx', '*.pdf', '*.htm', '*.pptx']:
        test_files.extend(assets_dir.glob(pattern))

    test_files = sorted(test_files)

    print("Go Parser Individual Document Test")
    print("=" * 50)
    print(f"Testing {len(test_files)} documents...")
    print()

    failed_files = []
    successful_files = []

    for doc_path in test_files:
        print(f"Testing: {doc_path.name} ({doc_path.suffix})")

        result = test_single_document(doc_path, use_go_modules=True)

        counts = result['file_counts']
        total_files = sum(counts.values())

        # Expected: 1 file each for documents, elements, relationships, embeddings = 4 total
        expected_total = 4

        if total_files == expected_total and counts['documents'] == 1 and counts['elements'] == 1:
            print(f"  ✅ SUCCESS: {total_files}/4 files generated")
            successful_files.append(doc_path.name)
        else:
            print(f"  ❌ FAILED: {total_files}/4 files generated")
            print(f"     Documents: {counts['documents']}, Elements: {counts['elements']}")
            print(f"     Relationships: {counts['relationships']}, Embeddings: {counts['embeddings']}")

            if result.get('timeout'):
                print(f"     ERROR: Timeout")
            elif not result['success']:
                print(f"     ERROR: {result['stderr'][:100]}...")
            elif result['debug_log']:
                # Look for errors in debug log
                errors = [line for line in result['debug_log'].split('\n') if 'ERROR' in line.upper()]
                if errors:
                    print(f"     LOG ERRORS: {errors[0][:100]}...")

            failed_files.append(doc_path.name)

        print()

    print("=" * 50)
    print("SUMMARY")
    print("=" * 50)
    print(f"✅ Successful: {len(successful_files)}")
    print(f"❌ Failed: {len(failed_files)}")
    print()

    if failed_files:
        print("FAILED FILES:")
        for filename in failed_files:
            print(f"  - {filename}")
        print()

        print("FILES BY EXTENSION:")
        failed_extensions = {}
        for filename in failed_files:
            ext = Path(filename).suffix
            failed_extensions[ext] = failed_extensions.get(ext, 0) + 1

        for ext, count in sorted(failed_extensions.items()):
            print(f"  {ext}: {count} files")

    print()
    if successful_files:
        print("SUCCESSFUL FILES:")
        for filename in successful_files:
            print(f"  - {filename}")

if __name__ == '__main__':
    main()