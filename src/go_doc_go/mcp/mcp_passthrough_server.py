#!/usr/bin/env python3
"""
Simple MCP passthrough server that forwards requests to primary server's sampling endpoints.
This allows MCP clients to use the database sampling functionality.
"""

import json
import logging
import requests
from typing import Dict, Any, Optional, List

logger = logging.getLogger(__name__)


class MCPPassthroughServer:
    """
    MCP server that acts as a passthrough to the primary server's sampling endpoints.
    """

    def __init__(self, primary_server_url: str = "http://localhost:5002"):
        """
        Initialize MCP passthrough server.

        Args:
            primary_server_url: URL of the primary server with sampling endpoints
        """
        self.primary_server_url = primary_server_url.rstrip('/')
        self.api_base = f"{self.primary_server_url}/api/sampling"

    def sample_elements(self,
                       filters: Optional[Dict[str, Any]] = None,
                       limit: int = 100,
                       stratify_by: Optional[str] = None,
                       include_document_attrs: bool = True,
                       random_seed: Optional[int] = None) -> Dict[str, Any]:
        """
        Sample elements from the database via primary server.

        Args:
            filters: Dictionary of column filters
            limit: Maximum number of elements to return
            stratify_by: Column to stratify sampling by
            include_document_attrs: Whether to join document attributes
            random_seed: Random seed for reproducible sampling

        Returns:
            Dictionary with sampled elements and metadata
        """
        try:
            payload = {
                "filters": filters,
                "limit": limit,
                "stratify_by": stratify_by,
                "include_document_attrs": include_document_attrs,
                "random_seed": random_seed
            }

            response = requests.post(
                f"{self.api_base}/elements",
                json=payload,
                timeout=30
            )
            response.raise_for_status()

            return response.json()

        except requests.RequestException as e:
            logger.error(f"Error sampling elements: {e}")
            return {"error": str(e), "success": False}

    def get_corpus_stats(self, filters: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Get corpus statistics from the primary server.

        Args:
            filters: Optional filters to apply to the corpus

        Returns:
            Dictionary with corpus statistics
        """
        try:
            payload = {"filters": filters} if filters else {}

            response = requests.post(
                f"{self.api_base}/corpus-stats",
                json=payload,
                timeout=30
            )
            response.raise_for_status()

            return response.json()

        except requests.RequestException as e:
            logger.error(f"Error getting corpus stats: {e}")
            return {"error": str(e), "success": False}

    def sample_documents(self,
                        filters: Optional[Dict[str, Any]] = None,
                        limit: int = 50,
                        random_seed: Optional[int] = None) -> Dict[str, Any]:
        """
        Sample documents from the database via primary server.

        Args:
            filters: Dictionary of document filters
            limit: Maximum number of documents
            random_seed: Random seed for reproducible sampling

        Returns:
            Dictionary with sampled documents
        """
        try:
            payload = {
                "filters": filters,
                "limit": limit,
                "random_seed": random_seed
            }

            response = requests.post(
                f"{self.api_base}/documents",
                json=payload,
                timeout=30
            )
            response.raise_for_status()

            return response.json()

        except requests.RequestException as e:
            logger.error(f"Error sampling documents: {e}")
            return {"error": str(e), "success": False}

    def execute_custom_query(self,
                           query: str,
                           params: Optional[List] = None) -> Dict[str, Any]:
        """
        Execute a custom SQL query via primary server.

        Args:
            query: SQL query string
            params: Query parameters

        Returns:
            Dictionary with query results
        """
        try:
            payload = {
                "query": query,
                "params": params
            }

            response = requests.post(
                f"{self.api_base}/custom-query",
                json=payload,
                timeout=60
            )
            response.raise_for_status()

            return response.json()

        except requests.RequestException as e:
            logger.error(f"Error executing custom query: {e}")
            return {"error": str(e), "success": False}

    def get_schema_info(self) -> Dict[str, Any]:
        """
        Get schema information from the primary server.

        Returns:
            Dictionary with schema information and examples
        """
        try:
            response = requests.get(
                f"{self.api_base}/schema",
                timeout=30
            )
            response.raise_for_status()

            return response.json()

        except requests.RequestException as e:
            logger.error(f"Error getting schema info: {e}")
            return {"error": str(e), "success": False}

    def sample_for_ontology(self,
                          domain_keywords: List[str],
                          max_elements: int = 200,
                          include_stats: bool = True) -> Dict[str, Any]:
        """
        Comprehensive sampling for ontology generation via primary server.

        Args:
            domain_keywords: Keywords for the domain
            max_elements: Maximum number of elements to sample
            include_stats: Whether to include corpus statistics

        Returns:
            Dictionary with comprehensive sampling results
        """
        try:
            payload = {
                "domain_keywords": domain_keywords,
                "max_elements": max_elements,
                "include_stats": include_stats
            }

            response = requests.post(
                f"{self.api_base}/ontology-sample",
                json=payload,
                timeout=60
            )
            response.raise_for_status()

            return response.json()

        except requests.RequestException as e:
            logger.error(f"Error in ontology sampling: {e}")
            return {"error": str(e), "success": False}


def create_mcp_tools(primary_server_url: str = "http://localhost:5002"):
    """
    Create MCP tools that use the passthrough server.

    Args:
        primary_server_url: URL of the primary server

    Returns:
        Dictionary of MCP tool functions
    """

    server = MCPPassthroughServer(primary_server_url)

    def sample_elements_tool(filters: str = "",
                           limit: int = 100,
                           stratify_by: str = "",
                           include_document_attrs: bool = True,
                           random_seed: int = None) -> str:
        """
        Sample elements from the analytics database.

        Args:
            filters: JSON string of column filters
            limit: Maximum number of elements to return
            stratify_by: Column to stratify sampling by
            include_document_attrs: Include document attributes
            random_seed: Random seed for reproducible sampling

        Returns:
            JSON string with sampled elements
        """

        filter_dict = None
        if filters.strip():
            try:
                filter_dict = json.loads(filters)
            except json.JSONDecodeError:
                return json.dumps({"error": f"Invalid JSON in filters: {filters}"})

        result = server.sample_elements(
            filters=filter_dict,
            limit=limit,
            stratify_by=stratify_by if stratify_by else None,
            include_document_attrs=include_document_attrs,
            random_seed=random_seed
        )

        return json.dumps(result, default=str, indent=2)

    def get_corpus_stats_tool(filters: str = "") -> str:
        """
        Get statistics about the document corpus.

        Args:
            filters: JSON string of filters to apply

        Returns:
            JSON string with corpus statistics
        """

        filter_dict = None
        if filters.strip():
            try:
                filter_dict = json.loads(filters)
            except json.JSONDecodeError:
                return json.dumps({"error": f"Invalid JSON in filters: {filters}"})

        result = server.get_corpus_stats(filters=filter_dict)
        return json.dumps(result, default=str, indent=2)

    def sample_documents_tool(filters: str = "",
                            limit: int = 50,
                            random_seed: int = None) -> str:
        """
        Sample documents from the analytics database.

        Args:
            filters: JSON string of document filters
            limit: Maximum number of documents
            random_seed: Random seed for reproducible sampling

        Returns:
            JSON string with sampled documents
        """

        filter_dict = None
        if filters.strip():
            try:
                filter_dict = json.loads(filters)
            except json.JSONDecodeError:
                return json.dumps({"error": f"Invalid JSON in filters: {filters}"})

        result = server.sample_documents(
            filters=filter_dict,
            limit=limit,
            random_seed=random_seed
        )

        return json.dumps(result, default=str, indent=2)

    def custom_query_tool(query: str, params: str = "") -> str:
        """
        Execute a custom SQL query against the analytics database.

        Args:
            query: SQL query string
            params: JSON array of query parameters

        Returns:
            JSON string with query results
        """

        param_list = None
        if params.strip():
            try:
                param_list = json.loads(params)
            except json.JSONDecodeError:
                return json.dumps({"error": f"Invalid JSON in params: {params}"})

        result = server.execute_custom_query(query, param_list)
        return json.dumps(result, default=str, indent=2)

    return {
        "sample_elements": sample_elements_tool,
        "get_corpus_stats": get_corpus_stats_tool,
        "sample_documents": sample_documents_tool,
        "custom_query": custom_query_tool
    }


if __name__ == "__main__":
    # Example usage
    import sys

    server_url = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:5002"

    print(f"MCP Passthrough Server connecting to: {server_url}")

    # Create MCP tools
    tools = create_mcp_tools(server_url)

    # Test corpus stats
    print("\nTesting corpus stats...")
    result = tools["get_corpus_stats"]()
    print(result)

    # Test element sampling
    print("\nTesting element sampling...")
    result = tools["sample_elements"](
        filters='{"element_type": "xml_element"}',
        limit=10
    )
    print(result[:500] + "..." if len(result) > 500 else result)