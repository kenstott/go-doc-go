#!/usr/bin/env python
"""
Script to run parser tests with proper imports.
"""

import sys
import os

# Add src directory to Python path
src_path = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..', 'src'))
sys.path.insert(0, src_path)

# Run pytest
import pytest

if __name__ == "__main__":
    # Run tests
    test_dir = os.path.dirname(__file__)
    result = pytest.main([test_dir, "-v", "--tb=short"])
    sys.exit(result)