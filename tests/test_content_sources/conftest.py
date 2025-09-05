"""
Shared fixtures for content source tests.
Imports fixtures from test_adapters/conftest.py
"""

import os
import sys

# Add test_adapters to path to access its conftest
test_adapters_path = os.path.join(os.path.dirname(__file__), '..', 'test_adapters')
sys.path.insert(0, test_adapters_path)

# Import all fixtures from test_adapters conftest
from conftest import *