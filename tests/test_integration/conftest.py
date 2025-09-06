"""
Shared fixtures for integration tests.
Imports fixtures from test_adapters/conftest.py
"""

import os
import sys

# Add test_adapters to path to access its conftest
test_adapters_path = os.path.join(os.path.dirname(__file__), '..', 'test_adapters')
sys.path.insert(0, test_adapters_path)

# Import all fixtures and markers from test_adapters conftest
from test_adapters.conftest import *

# Explicitly re-export the markers
__all__ = ['requires_boto3', 'requires_docker', 'requires_minio', 'requires_pymongo']