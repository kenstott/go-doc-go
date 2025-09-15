"""Automatically generated __init__.py"""
__all__ = ['Config', 'SearchEngine', 'SearchRequest', 'SearchResponse', 'SearchHit',
           'api_info', 'bad_request', 'check_api_key', 'config', 'configure_logging', 'crawl', 'crawler',
           'extract_topic_parameters', 'get_vendor_path', 'health_check', 'ingest_documents', 'internal_error', 
           'load_openapi_spec', 'main', 'not_found', 'openapi_spec', 'print_startup_info', 'search_unified', 
           'server', 'vendor']

from . import config
from . import crawler
from . import main
from . import server
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
# Search endpoints removed - no longer importing
# from .server import advanced_search_endpoint
# from .server import document_sources_endpoint
# from .server import search_endpoint
# from .server import simple_structured_search_endpoint
# from .server import structured_search_endpoint
from .server import api_info
from .server import bad_request
from .server import check_api_key
from .server import extract_topic_parameters
from .server import health_check
from .server import internal_error
from .server import load_openapi_spec
from .server import not_found
from .server import openapi_spec
from .server import print_startup_info
# from .server import root  # Commented out - root route handled differently
from .vendor import get_vendor_path

configure_logging()

__version__ = "0.41.0"
