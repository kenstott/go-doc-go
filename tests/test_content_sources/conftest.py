"""
Shared fixtures for content source tests.
Imports fixtures from test_adapters/conftest.py
"""

import os
import sys

# Add test_adapters to path to access its conftest
test_adapters_path = os.path.join(os.path.dirname(__file__), '..', 'test_adapters')
sys.path.insert(0, test_adapters_path)

# Import everything from the test_adapters conftest module
import importlib.util
conftest_path = os.path.join(test_adapters_path, 'conftest.py')
spec = importlib.util.spec_from_file_location("test_adapters_conftest", conftest_path)
test_adapters_conftest = importlib.util.module_from_spec(spec)
spec.loader.exec_module(test_adapters_conftest)

# Re-export all public items
for name in dir(test_adapters_conftest):
    if not name.startswith('_'):
        globals()[name] = getattr(test_adapters_conftest, name)