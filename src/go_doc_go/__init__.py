"""Automatically generated __init__.py"""
__all__ = ['Config', 'SearchEngine', 'SearchRequest', 'SearchResponse', 'SearchHit',
           'config', 'configure_logging', 'crawl', 'crawler', 'get_vendor_path',
           'ingest_documents', 'main', 'search_unified', 'vendor']

from . import config
from . import crawler
from . import main
from . import vendor
from .config import Config
from .configure_logging import configure_logging
from .crawler import crawl
from .main import ingest_documents
# Unified search module exports
from .search_module import SearchEngine
from .search_module import SearchRequest
from .search_module import SearchResponse
from .search_module import SearchHit
from .search_module import search as search_unified
# Server imports removed - API server deprecated
# Use CLI tools instead:
#   - Search: python -m go_doc_go.cli.search
#   - Process: python -m go_doc_go.cli.process
#   - Status: python -m go_doc_go.cli.status
#   - Analytics: python -m go_doc_go.cli.analytics
from .vendor import get_vendor_path

configure_logging()

__version__ = "0.41.0"
