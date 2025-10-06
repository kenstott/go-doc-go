#!/usr/bin/env python3
"""
Embedding Parity Test

Tests that Go and Python workers generate identical (or nearly identical) embeddings
for the same documents. This verifies:

1. Contextual text generation matches between implementations
2. Embedding vectors are numerically equivalent
3. Element filtering logic is consistent
4. Token budgeting produces same results
"""

import os
import sys
import json
import sqlite3
import subprocess
import tempfile
import shutil
from pathlib import Path
from typing import Dict, List, Tuple, Optional
import numpy as np
import pyarrow.parquet as pq
import pytest

# Add src to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from go_doc_go.config import Config


class EmbeddingParityTest:
    """Test embedding parity between Go and Python implementations"""

    def __init__(self, test_dir: Path):
        self.test_dir = test_dir
        self.python_output_dir = test_dir / "python_output"
        self.go_output_dir = test_dir / "go_output"
        self.db_path = test_dir / "test_queue.db"

        # Create directories
        self.python_output_dir.mkdir(parents=True, exist_ok=True)
        self.go_output_dir.mkdir(parents=True, exist_ok=True)

    def create_test_config(self, output_dir: Path, use_go_modules: bool = False) -> Path:
        """Create a test configuration file"""
        config_path = self.test_dir / f"config_{'go' if use_go_modules else 'python'}.yaml"

        config_content = f"""
analytics_storage:
  type: parquet
  output_dir: {output_dir}

content_sources:
  - name: file-docs
    type: file
    source_id: file-docs
    base_path: tests/test_documents
    patterns:
      - "*.pdf"
      - "*.docx"

job_control:
  type: sqlite
  db_path: {self.db_path}

embeddings:
  enabled: true
  model_name: nomic-embed-text
  model_path: models/nomic-embed-text-v1.5.f16.gguf
  chunk_size: 16384
  predecessor_count: 1
  successor_count: 1
  batch_size: 32

# Go-specific settings
{"use_go_modules: true" if use_go_modules else "use_go_modules: false"}
"""
        config_path.write_text(config_content)
        return config_path

    def run_python_worker(self, config_path: Path, max_docs: int = 5) -> subprocess.CompletedProcess:
        """Run Python worker"""
        env = os.environ.copy()
        env['USE_GO_MODULES'] = 'false'
        env['PYTHONPATH'] = 'src'

        cmd = [
            sys.executable, '-m', 'go_doc_go.cli.worker',
            '--config', str(config_path),
            '--max-documents', str(max_docs)
        ]

        print(f"Running Python worker: {' '.join(cmd)}")
        result = subprocess.run(cmd, capture_output=True, text=True, env=env)

        if result.returncode != 0:
            print(f"Python worker stderr: {result.stderr}")
            print(f"Python worker stdout: {result.stdout}")

        return result

    def run_go_worker(self, config_path: Path, max_docs: int = 5) -> subprocess.CompletedProcess:
        """Run Go worker"""
        go_binary = Path("bin/goworker")
        if not go_binary.exists():
            raise FileNotFoundError(f"Go worker binary not found at {go_binary}")

        cmd = [
            str(go_binary),
            '--config', str(config_path),
            '--max-documents', str(max_docs)
        ]

        print(f"Running Go worker: {' '.join(cmd)}")
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)

        if result.returncode != 0:
            print(f"Go worker stderr: {result.stderr}")
            print(f"Go worker stdout: {result.stdout}")

        return result

    def load_embeddings_from_parquet(self, output_dir: Path) -> Dict[str, Dict]:
        """Load all embeddings from parquet files in output directory"""
        embeddings = {}

        # Find all parquet files
        parquet_files = list(output_dir.rglob("*.parquet"))
        print(f"Found {len(parquet_files)} parquet files in {output_dir}")

        for parquet_file in parquet_files:
            try:
                table = pq.read_table(parquet_file)
                df = table.to_pandas()

                # Each row has: element_id, embedding (as list), embedding_text, doc_id, etc.
                for _, row in df.iterrows():
                    element_id = row['element_id']
                    embeddings[element_id] = {
                        'embedding': np.array(row['embedding']),
                        'embedding_text': row.get('embedding_text', ''),
                        'doc_id': row.get('doc_id', ''),
                        'source_id': row.get('source_id', '')
                    }

            except Exception as e:
                print(f"Error loading {parquet_file}: {e}")
                continue

        return embeddings

    def compare_embeddings(
        self,
        python_embeddings: Dict[str, Dict],
        go_embeddings: Dict[str, Dict],
        cosine_threshold: float = 0.999,
        text_diff_threshold: int = 10
    ) -> Tuple[bool, List[str]]:
        """
        Compare embeddings between Python and Go implementations

        Returns:
            (passed, error_messages)
        """
        errors = []

        # Check counts
        if len(python_embeddings) != len(go_embeddings):
            errors.append(
                f"Element count mismatch: Python={len(python_embeddings)}, "
                f"Go={len(go_embeddings)}"
            )

        # Compare common elements
        common_ids = set(python_embeddings.keys()) & set(go_embeddings.keys())
        python_only = set(python_embeddings.keys()) - set(go_embeddings.keys())
        go_only = set(go_embeddings.keys()) - set(python_embeddings.keys())

        if python_only:
            errors.append(f"Python-only elements ({len(python_only)}): {list(python_only)[:5]}")
        if go_only:
            errors.append(f"Go-only elements ({len(go_only)}): {list(go_only)[:5]}")

        print(f"\nComparing {len(common_ids)} common elements...")

        # Compare each common element
        text_mismatches = []
        vector_mismatches = []

        for element_id in common_ids:
            py_data = python_embeddings[element_id]
            go_data = go_embeddings[element_id]

            # Compare embedding text
            py_text = py_data['embedding_text']
            go_text = go_data['embedding_text']

            if py_text != go_text:
                text_diff_len = abs(len(py_text) - len(go_text))
                if text_diff_len > text_diff_threshold:
                    text_mismatches.append({
                        'element_id': element_id,
                        'py_len': len(py_text),
                        'go_len': len(go_text),
                        'diff_len': text_diff_len,
                        'py_preview': py_text[:100],
                        'go_preview': go_text[:100]
                    })

            # Compare embedding vectors using cosine similarity
            py_vec = py_data['embedding']
            go_vec = go_data['embedding']

            if len(py_vec) != len(go_vec):
                errors.append(
                    f"Vector dimension mismatch for {element_id}: "
                    f"Python={len(py_vec)}, Go={len(go_vec)}"
                )
                continue

            # Cosine similarity
            cosine_sim = np.dot(py_vec, go_vec) / (
                np.linalg.norm(py_vec) * np.linalg.norm(go_vec)
            )

            if cosine_sim < cosine_threshold:
                vector_mismatches.append({
                    'element_id': element_id,
                    'cosine_similarity': cosine_sim,
                    'l2_distance': np.linalg.norm(py_vec - go_vec)
                })

        # Report mismatches
        if text_mismatches:
            errors.append(
                f"\n{len(text_mismatches)} significant text mismatches (>{text_diff_threshold} chars):"
            )
            for mismatch in text_mismatches[:3]:  # Show first 3
                errors.append(
                    f"  {mismatch['element_id']}: "
                    f"Python={mismatch['py_len']} chars, Go={mismatch['go_len']} chars\n"
                    f"    Python preview: {mismatch['py_preview']}\n"
                    f"    Go preview: {mismatch['go_preview']}"
                )

        if vector_mismatches:
            errors.append(
                f"\n{len(vector_mismatches)} vector mismatches (cosine < {cosine_threshold}):"
            )
            for mismatch in vector_mismatches[:3]:  # Show first 3
                errors.append(
                    f"  {mismatch['element_id']}: "
                    f"cosine_similarity={mismatch['cosine_similarity']:.6f}, "
                    f"L2_distance={mismatch['l2_distance']:.6f}"
                )

        passed = len(errors) == 0
        return passed, errors

    def run_parity_test(self, max_docs: int = 5) -> bool:
        """Run full parity test"""
        print("\n" + "="*80)
        print("EMBEDDING PARITY TEST")
        print("="*80)

        # Clean database
        if self.db_path.exists():
            self.db_path.unlink()

        # Create configs
        python_config = self.create_test_config(self.python_output_dir, use_go_modules=False)
        go_config = self.create_test_config(self.go_output_dir, use_go_modules=True)

        # Run Python worker
        print("\n1. Running Python worker...")
        py_result = self.run_python_worker(python_config, max_docs)
        if py_result.returncode != 0:
            print(f"Python worker failed with exit code {py_result.returncode}")
            return False
        print("   ✓ Python worker completed")

        # Reset database for Go worker
        if self.db_path.exists():
            self.db_path.unlink()

        # Run Go worker
        print("\n2. Running Go worker...")
        go_result = self.run_go_worker(go_config, max_docs)
        if go_result.returncode != 0:
            print(f"Go worker failed with exit code {go_result.returncode}")
            return False
        print("   ✓ Go worker completed")

        # Load embeddings
        print("\n3. Loading embeddings...")
        python_embeddings = self.load_embeddings_from_parquet(self.python_output_dir)
        go_embeddings = self.load_embeddings_from_parquet(self.go_output_dir)

        print(f"   Python embeddings: {len(python_embeddings)}")
        print(f"   Go embeddings: {len(go_embeddings)}")

        if not python_embeddings or not go_embeddings:
            print("   ✗ No embeddings found")
            return False

        # Compare
        print("\n4. Comparing embeddings...")
        passed, errors = self.compare_embeddings(python_embeddings, go_embeddings)

        if passed:
            print("   ✓ All embeddings match!")
            return True
        else:
            print("   ✗ Embedding mismatches detected:")
            for error in errors:
                print(f"     {error}")
            return False


@pytest.fixture
def test_env():
    """Create test environment"""
    test_dir = Path(tempfile.mkdtemp(prefix="embedding_parity_"))
    yield test_dir
    # Cleanup
    shutil.rmtree(test_dir, ignore_errors=True)


def test_embedding_parity(test_env):
    """Test that Python and Go workers generate identical embeddings"""
    tester = EmbeddingParityTest(test_env)
    assert tester.run_parity_test(max_docs=3), "Embedding parity test failed"


if __name__ == "__main__":
    # Run standalone
    test_dir = Path(tempfile.mkdtemp(prefix="embedding_parity_"))
    try:
        tester = EmbeddingParityTest(test_dir)
        success = tester.run_parity_test(max_docs=3)
        sys.exit(0 if success else 1)
    finally:
        shutil.rmtree(test_dir, ignore_errors=True)
