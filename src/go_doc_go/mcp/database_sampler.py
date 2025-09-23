"""
Generic MCP component for sampling document and element records from analytics database.
"""

import json
import logging
from typing import Dict, List, Any, Optional, Union
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class SamplingParams:
    """Parameters for database sampling."""
    limit: int = 100
    random_seed: Optional[int] = None
    stratify_by: Optional[str] = None
    where_clause: Optional[str] = None
    order_by: Optional[str] = None


class DatabaseSampler:
    """Generic database sampler for MCP tools."""

    def __init__(self, db_connection):
        """Initialize with database connection."""
        self.db = db_connection

    def sample_elements(self,
                       filters: Optional[Dict[str, Any]] = None,
                       limit: int = 100,
                       stratify_by: Optional[str] = None,
                       include_document_attrs: bool = True,
                       random_seed: Optional[int] = None) -> List[Dict[str, Any]]:
        """
        Sample elements from the database with flexible filtering.

        Args:
            filters: Dictionary of column filters (e.g., {"element_type": "xml_element"})
            limit: Maximum number of elements to return
            stratify_by: Column to stratify sampling by (e.g., "element_type")
            include_document_attrs: Whether to join document attributes
            random_seed: Random seed for reproducible sampling

        Returns:
            List of element records as dictionaries
        """

        # Choose base table/view
        base_table = "element_document_enriched" if include_document_attrs else "elements"

        # Build WHERE clause from filters
        where_conditions = []
        params = []

        if filters:
            for column, value in filters.items():
                if isinstance(value, list):
                    placeholders = ','.join(['%s'] * len(value))
                    where_conditions.append(f"{column} IN ({placeholders})")
                    params.extend(value)
                elif isinstance(value, str) and '*' in value:
                    # Support wildcard matching
                    where_conditions.append(f"{column} ILIKE %s")
                    params.append(value.replace('*', '%'))
                else:
                    where_conditions.append(f"{column} = %s")
                    params.append(value)

        where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

        # Build query with optional stratification
        if stratify_by and limit > 10:
            # Stratified sampling - get roughly equal samples from each stratum
            query = f"""
            WITH stratified_sample AS (
                SELECT *,
                    ROW_NUMBER() OVER (
                        PARTITION BY {stratify_by}
                        ORDER BY {'RANDOM()' if random_seed is None else f'setseed({random_seed/1000000.0}), RANDOM()'}
                    ) as rn
                FROM {base_table}
                WHERE {where_clause}
            ),
            stratum_limits AS (
                SELECT {stratify_by},
                       COUNT(*) as total_count,
                       GREATEST(1, %s / COUNT(DISTINCT {stratify_by}) OVER ()) as per_stratum_limit
                FROM {base_table}
                WHERE {where_clause}
                GROUP BY {stratify_by}
            )
            SELECT s.*
            FROM stratified_sample s
            JOIN stratum_limits l ON s.{stratify_by} = l.{stratify_by}
            WHERE s.rn <= l.per_stratum_limit
            ORDER BY s.{stratify_by}, s.rn
            LIMIT %s
            """
            params.extend([limit, limit])
        else:
            # Simple random sampling
            if random_seed is not None:
                seed_clause = f"(SELECT setseed({random_seed/1000000.0})),"
            else:
                seed_clause = ""

            query = f"""
            SELECT *
            FROM (
                {seed_clause}
                SELECT *
                FROM {base_table}
                WHERE {where_clause}
                ORDER BY RANDOM()
                LIMIT %s
            ) sample
            ORDER BY element_pk
            """
            params.append(limit)

        cursor = self.db.cursor()
        cursor.execute(query, params)

        # Convert to list of dictionaries
        columns = [desc[0] for desc in cursor.description]
        results = []
        for row in cursor.fetchall():
            row_dict = dict(zip(columns, row))
            # Convert any JSON columns to proper objects
            for key, value in row_dict.items():
                if key.endswith('_metadata') and isinstance(value, str):
                    try:
                        row_dict[key] = json.loads(value)
                    except (json.JSONDecodeError, TypeError):
                        pass
            results.append(row_dict)

        return results

    def get_corpus_stats(self,
                        filters: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Get statistics about the corpus for sampling decisions.

        Args:
            filters: Optional filters to apply to the corpus

        Returns:
            Dictionary with corpus statistics
        """

        # Build WHERE clause from filters
        where_conditions = []
        params = []

        if filters:
            for column, value in filters.items():
                if isinstance(value, list):
                    placeholders = ','.join(['%s'] * len(value))
                    where_conditions.append(f"{column} IN ({placeholders})")
                    params.extend(value)
                elif isinstance(value, str) and '*' in value:
                    where_conditions.append(f"{column} ILIKE %s")
                    params.append(value.replace('*', '%'))
                else:
                    where_conditions.append(f"{column} = %s")
                    params.append(value)

        where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

        # Get comprehensive statistics
        stats_query = f"""
        SELECT
            COUNT(*) as total_elements,
            COUNT(DISTINCT doc_id) as total_documents,
            COUNT(DISTINCT element_type) as distinct_element_types,
            COUNT(DISTINCT COALESCE(structural_name, 'unknown')) as distinct_structural_names,
            COUNT(*) FILTER (WHERE has_temporal_value = TRUE) as temporal_elements,

            -- Top element types
            json_object_agg(
                'element_type_' || element_type,
                COUNT(*)
            ) FILTER (WHERE element_type IS NOT NULL) as element_type_distribution,

            -- Top structural names
            json_object_agg(
                'struct_name_' || COALESCE(structural_name, 'unknown'),
                COUNT(*)
            ) as structural_name_distribution,

            -- Format distribution
            json_object_agg(
                'format_' || COALESCE(format_type, 'unknown'),
                COUNT(*)
            ) as format_distribution

        FROM element_document_enriched
        WHERE {where_clause}
        """

        cursor = self.db.cursor()
        cursor.execute(stats_query, params)
        row = cursor.fetchone()

        if row:
            return dict(zip([desc[0] for desc in cursor.description], row))
        else:
            return {}

    def sample_documents(self,
                        filters: Optional[Dict[str, Any]] = None,
                        limit: int = 50,
                        random_seed: Optional[int] = None) -> List[Dict[str, Any]]:
        """
        Sample documents from the database.

        Args:
            filters: Dictionary of document filters
            limit: Maximum number of documents
            random_seed: Random seed for reproducible sampling

        Returns:
            List of document records
        """

        # Build WHERE clause
        where_conditions = []
        params = []

        if filters:
            for column, value in filters.items():
                if isinstance(value, list):
                    placeholders = ','.join(['%s'] * len(value))
                    where_conditions.append(f"{column} IN ({placeholders})")
                    params.extend(value)
                elif isinstance(value, str) and '*' in value:
                    where_conditions.append(f"{column} ILIKE %s")
                    params.append(value.replace('*', '%'))
                else:
                    where_conditions.append(f"{column} = %s")
                    params.append(value)

        where_clause = " AND ".join(where_conditions) if where_conditions else "TRUE"

        # Sample documents
        if random_seed is not None:
            seed_clause = f"(SELECT setseed({random_seed/1000000.0})),"
        else:
            seed_clause = ""

        query = f"""
        SELECT DISTINCT doc_id, source, doc_type, metadata, created_at
        FROM (
            {seed_clause}
            SELECT DISTINCT doc_id, source, doc_type, metadata, created_at
            FROM element_document_enriched
            WHERE {where_clause}
            ORDER BY RANDOM()
            LIMIT %s
        ) sample
        ORDER BY doc_id
        """
        params.append(limit)

        cursor = self.db.cursor()
        cursor.execute(query, params)

        columns = [desc[0] for desc in cursor.description]
        results = []
        for row in cursor.fetchall():
            row_dict = dict(zip(columns, row))
            # Convert metadata JSON
            if 'metadata' in row_dict and isinstance(row_dict['metadata'], str):
                try:
                    row_dict['metadata'] = json.loads(row_dict['metadata'])
                except (json.JSONDecodeError, TypeError):
                    pass
            results.append(row_dict)

        return results

    def execute_custom_query(self,
                           query: str,
                           params: Optional[List] = None) -> List[Dict[str, Any]]:
        """
        Execute a custom SQL query for advanced sampling.

        Args:
            query: SQL query string
            params: Query parameters

        Returns:
            Query results as list of dictionaries
        """

        cursor = self.db.cursor()
        cursor.execute(query, params or [])

        columns = [desc[0] for desc in cursor.description]
        results = []
        for row in cursor.fetchall():
            results.append(dict(zip(columns, row)))

        return results


def create_mcp_tools(db_connection):
    """Create MCP tools for database sampling."""

    sampler = DatabaseSampler(db_connection)

    def sample_elements_tool(filters: str = "",
                           limit: int = 100,
                           stratify_by: str = "",
                           include_document_attrs: bool = True,
                           random_seed: int = None) -> str:
        """
        Sample elements from the analytics database.

        Args:
            filters: JSON string of column filters (e.g., '{"element_type": "xml_element", "format_type": "xml"}')
            limit: Maximum number of elements to return (default 100)
            stratify_by: Column to stratify sampling by (e.g., "element_type")
            include_document_attrs: Include document attributes (default True)
            random_seed: Random seed for reproducible sampling (optional)

        Returns:
            JSON string with sampled elements
        """

        # Parse filters
        filter_dict = None
        if filters.strip():
            try:
                filter_dict = json.loads(filters)
            except json.JSONDecodeError:
                return json.dumps({"error": f"Invalid JSON in filters: {filters}"})

        try:
            results = sampler.sample_elements(
                filters=filter_dict,
                limit=limit,
                stratify_by=stratify_by if stratify_by else None,
                include_document_attrs=include_document_attrs,
                random_seed=random_seed
            )

            return json.dumps({
                "elements": results,
                "count": len(results),
                "filters_applied": filter_dict,
                "stratified_by": stratify_by if stratify_by else None
            }, default=str, indent=2)

        except Exception as e:
            return json.dumps({"error": str(e)})

    def get_corpus_stats_tool(filters: str = "") -> str:
        """
        Get statistics about the document corpus.

        Args:
            filters: JSON string of filters to apply (optional)

        Returns:
            JSON string with corpus statistics
        """

        filter_dict = None
        if filters.strip():
            try:
                filter_dict = json.loads(filters)
            except json.JSONDecodeError:
                return json.dumps({"error": f"Invalid JSON in filters: {filters}"})

        try:
            stats = sampler.get_corpus_stats(filters=filter_dict)
            return json.dumps(stats, default=str, indent=2)
        except Exception as e:
            return json.dumps({"error": str(e)})

    def sample_documents_tool(filters: str = "",
                            limit: int = 50,
                            random_seed: int = None) -> str:
        """
        Sample documents from the analytics database.

        Args:
            filters: JSON string of document filters (optional)
            limit: Maximum number of documents (default 50)
            random_seed: Random seed for reproducible sampling (optional)

        Returns:
            JSON string with sampled documents
        """

        filter_dict = None
        if filters.strip():
            try:
                filter_dict = json.loads(filters)
            except json.JSONDecodeError:
                return json.dumps({"error": f"Invalid JSON in filters: {filters}"})

        try:
            results = sampler.sample_documents(
                filters=filter_dict,
                limit=limit,
                random_seed=random_seed
            )

            return json.dumps({
                "documents": results,
                "count": len(results),
                "filters_applied": filter_dict
            }, default=str, indent=2)

        except Exception as e:
            return json.dumps({"error": str(e)})

    def custom_query_tool(query: str, params: str = "") -> str:
        """
        Execute a custom SQL query against the analytics database.

        Args:
            query: SQL query string
            params: JSON array of query parameters (optional)

        Returns:
            JSON string with query results
        """

        param_list = None
        if params.strip():
            try:
                param_list = json.loads(params)
            except json.JSONDecodeError:
                return json.dumps({"error": f"Invalid JSON in params: {params}"})

        try:
            results = sampler.execute_custom_query(query, param_list)
            return json.dumps({
                "results": results,
                "count": len(results),
                "query": query
            }, default=str, indent=2)
        except Exception as e:
            return json.dumps({"error": str(e)})

    return {
        "sample_elements": sample_elements_tool,
        "get_corpus_stats": get_corpus_stats_tool,
        "sample_documents": sample_documents_tool,
        "custom_query": custom_query_tool
    }