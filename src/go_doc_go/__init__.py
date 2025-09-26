"""Automatically generated __init__.py"""
__all__ = ['Config', 'config', 'configure_logging', 'get_vendor_path', 'vendor']

from . import config
from . import vendor
from .config import Config
from .configure_logging import configure_logging
# Search functionality moved to CLI tools
# Use CLI tools instead:
#   - Search: python -m go_doc_go.cli.search
#   - Worker: python -m go_doc_go.cli.worker
#   - Status: python -m go_doc_go.cli.status
#   - Analytics: python -m go_doc_go.cli.analytics
from .vendor import get_vendor_path

configure_logging()

__version__ = "0.41.0"
