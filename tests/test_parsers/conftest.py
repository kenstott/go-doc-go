"""
Pytest configuration to bypass server initialization issues.
"""

import sys
import os
import unittest.mock as mock

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'src'))

# Mock the server module to prevent database initialization
sys.modules['go_doc_go.server'] = mock.MagicMock()

# Set dummy environment variable
os.environ['DOCUMENTS_URI'] = 'file://./test_storage'