"""
Search functionality extracted from the API server for CLI use.

This module provides comprehensive search capabilities against parquet data lakes
with contextual information retrieval, including parent/child relationships,
semantic relationships, and document reconstruction.
"""

import logging
import json
import duckdb
import pandas as pd
import numpy as np
from typing import Dict, List, Any, Optional, Union
from pathlib import Path
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.metrics.pairwise import cosine_similarity

logger = logging.getLogger(__name__)


class SearchFilters:
    """Container for search filter parameters."""

    def __init__(self,
                 regex_pattern: Optional[str] = None,
                 element_types: Optional[List[str]] = None,
                 cosine_threshold: float = 0.0,
                 limit: int = 10):
        self.regex_pattern = regex_pattern
        self.element_types = element_types or []
        self.cosine_threshold = cosine_threshold
        self.limit = limit


class ContextConfig:
    """Container for context retrieval parameters."""

    def __init__(self,
                 parents: int = 2,
                 siblings: int = 3,
                 semantic_relationships: int = 5,
                 include_document_metadata: bool = True):
        self.parents = parents
        self.siblings = siblings
        self.semantic_relationships = semantic_relationships
        self.include_document_metadata = include_document_metadata


class ReconstructionConfig:
    """Container for document reconstruction parameters."""

    def __init__(self,
                 format: str = "markdown",
                 include_metadata: bool = True,
                 max_depth: int = 10):
        self.format = format
        self.include_metadata = include_metadata
        self.max_depth = max_depth


class SearchResult:
    """Container for a single search result with context."""

    def __init__(self, element_data: Dict[str, Any], similarity: Optional[float] = None):
        self.element_data = element_data
        self.similarity = similarity
        self.context = {}
        self.relationships = []
        self.siblings = []

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            'element': self.element_data,
            'similarity': self.similarity,
            'context': self.context,
            'relationships': self.relationships,
            'siblings': self.siblings
        }


class SearchResults:
    """Container for search results with metadata."""

    def __init__(self, results: List[SearchResult], total: int, query: str):
        self.results = results
        self.total = total
        self.query = query
        self.materialized_documents = {}

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            'query': self.query,
            'total': self.total,
            'results': [r.to_dict() for r in self.results],
            'materialized_documents': self.materialized_documents
        }


class ParquetSearchEngine:
    """Search engine for parquet-based data lakes."""

    def __init__(self, base_path: str = './data-lake'):
        self.base_path = base_path
        self.conn = duckdb.connect(':memory:')

        # Paths for data
        self.elements_path = f'{self.base_path}/elements/**/*.parquet'
        self.relationships_path = f'{self.base_path}/relationships/**/*.parquet'
        self.documents_path = f'{self.base_path}/documents/**/*.parquet'

    def search(self,
               query: str,
               filters: Optional[SearchFilters] = None,
               context_config: Optional[ContextConfig] = None,
               reconstruction_config: Optional[ReconstructionConfig] = None) -> SearchResults:
        """
        Execute comprehensive search with context retrieval.

        Args:
            query: Search query text
            filters: Filter parameters for elements
            context_config: Context retrieval configuration
            reconstruction_config: Document reconstruction configuration

        Returns:
            SearchResults object with results and metadata
        """
        if filters is None:
            filters = SearchFilters()
        if context_config is None:
            context_config = ContextConfig()
        if reconstruction_config is None:
            reconstruction_config = ReconstructionConfig()

        logger.info(f"Searching parquet data lake at {self.base_path} for query: '{query}'")

        # Build filter conditions
        where_clauses = self._build_filter_conditions(query, filters)
        where_clause = f"WHERE {' AND '.join(where_clauses)}" if where_clauses else ""

        # Execute comprehensive query
        comprehensive_query = self._build_comprehensive_query(where_clause, context_config)

        try:
            result_df = self.conn.execute(comprehensive_query).df()

            if result_df.empty:
                return SearchResults([], 0, query)

            # Apply cosine similarity if query provided
            if query and not result_df.empty:
                result_df = self._apply_cosine_similarity(result_df, query, filters.cosine_threshold)

            # Convert to SearchResult objects
            results = self._convert_to_search_results(result_df, filters.limit)

            # Handle document reconstruction if requested
            materialized_docs = {}
            if reconstruction_config.format in ['markdown', 'html', 'json']:
                materialized_docs = self._reconstruct_documents(results, reconstruction_config)

            search_results = SearchResults(results, len(results), query)
            search_results.materialized_documents = materialized_docs

            return search_results

        except Exception as e:
            logger.error(f"Search failed: {str(e)}", exc_info=True)
            raise

    def _build_filter_conditions(self, query: str, filters: SearchFilters) -> List[str]:
        """Build WHERE clause conditions from filters."""
        where_clauses = []

        # Parse element type filters with +/- convention
        include_types, exclude_types = self._parse_element_types(filters.element_types)

        # Element type filters
        if include_types:
            types_list = ', '.join([f"'{t}'" for t in include_types])
            where_clauses.append(f"e.element_type IN ({types_list})")
        if exclude_types:
            exclude_list = ', '.join([f"'{t}'" for t in exclude_types])
            where_clauses.append(f"e.element_type NOT IN ({exclude_list})")

        # Regex pattern filter
        if filters.regex_pattern:
            where_clauses.append(f"regexp_matches(e.content_preview, '{filters.regex_pattern}')")

        # Text search (basic keyword matching)
        if query:
            query_terms = query.lower().split()
            term_conditions = []
            for term in query_terms:
                term_conditions.append(f"lower(e.content_preview) LIKE '%{term}%'")
            if term_conditions:
                where_clauses.append(f"({' OR '.join(term_conditions)})")

        return where_clauses

    def _parse_element_types(self, element_types: List[str]) -> tuple[List[str], List[str]]:
        """Parse element types with +/- include/exclude convention."""
        include_types = []
        exclude_types = []

        if element_types:
            try:
                from go_doc_go.storage.element_element import ElementType
                valid_element_types = {e.value for e in ElementType}

                for elem_type in element_types:
                    if elem_type.startswith('-'):
                        base_type = elem_type[1:]
                        if base_type in valid_element_types:
                            exclude_types.append(base_type)
                    elif elem_type.startswith('+'):
                        base_type = elem_type[1:]
                        if base_type in valid_element_types:
                            include_types.append(base_type)
                    else:
                        if elem_type in valid_element_types:
                            include_types.append(elem_type)
            except ImportError:
                logger.warning("ElementType enum not available, skipping element type validation")

        return include_types, exclude_types

    def _build_comprehensive_query(self, where_clause: str, context_config: ContextConfig) -> str:
        """Build the comprehensive DuckDB query for search with context."""
        return f"""
        WITH
        -- Load all elements with filters applied
        filtered_elements AS (
            SELECT * FROM read_parquet('{self.elements_path}', hive_partitioning=true) e
            {where_clause}
        ),

        -- Load all elements (for context lookup)
        all_elements AS (
            SELECT * FROM read_parquet('{self.elements_path}', hive_partitioning=true)
        ),

        -- Load all relationships
        all_relationships AS (
            SELECT * FROM read_parquet('{self.relationships_path}', hive_partitioning=true)
        ),

        -- Load all documents
        all_documents AS (
            SELECT * FROM read_parquet('{self.documents_path}', hive_partitioning=true)
        ),

        -- Get relationships for filtered elements
        element_relationships AS (
            SELECT
                CASE
                    WHEN r.source_id IN (SELECT element_id FROM filtered_elements) THEN r.source_id
                    ELSE r.target_id
                END as element_id,
                r.relationship_id,
                r.relationship_type,
                r.source_id,
                r.target_id,
                r.metadata as rel_metadata
            FROM all_relationships r
            WHERE r.source_id IN (SELECT element_id FROM filtered_elements)
               OR r.target_id IN (SELECT element_id FROM filtered_elements)
        ),

        -- Get parent hierarchy
        parent_hierarchy AS (
            SELECT
                f.element_id,
                f.parent_id as parent1_id,
                p1.element_type as parent1_type,
                p1.content_preview as parent1_content,
                p1.parent_id as parent2_id,
                p2.element_type as parent2_type,
                p2.content_preview as parent2_content,
                p2.parent_id as parent3_id,
                p3.element_type as parent3_type,
                p3.content_preview as parent3_content
            FROM filtered_elements f
            LEFT JOIN all_elements p1 ON f.parent_id = p1.element_id
            LEFT JOIN all_elements p2 ON p1.parent_id = p2.element_id
            LEFT JOIN all_elements p3 ON p2.parent_id = p3.element_id
        ),

        -- Get siblings
        element_siblings AS (
            SELECT
                f.element_id,
                s.element_id as sibling_id,
                s.element_type as sibling_type,
                s.content_preview as sibling_content
            FROM filtered_elements f
            INNER JOIN all_elements s ON f.parent_id = s.parent_id
            WHERE f.element_id != s.element_id
        )

        -- Main query combining everything
        SELECT
            f.*,
            d.metadata as doc_metadata,
            d.source as doc_source,
            ph.parent1_id, ph.parent1_type, ph.parent1_content,
            ph.parent2_id, ph.parent2_type, ph.parent2_content,
            ph.parent3_id, ph.parent3_type, ph.parent3_content,
            r.relationship_id, r.relationship_type, r.source_id, r.target_id, r.rel_metadata,
            s.sibling_id, s.sibling_type, s.sibling_content
        FROM filtered_elements f
        LEFT JOIN all_documents d ON f.doc_id = d.doc_id
        LEFT JOIN parent_hierarchy ph ON f.element_id = ph.element_id
        LEFT JOIN element_relationships r ON f.element_id = r.element_id
        LEFT JOIN element_siblings s ON f.element_id = s.element_id
        """

    def _apply_cosine_similarity(self, df: pd.DataFrame, query: str, threshold: float) -> pd.DataFrame:
        """Apply cosine similarity filtering and ranking."""
        try:
            # Create corpus from content_preview
            corpus = [query] + df['content_preview'].fillna('').tolist()

            # Use TF-IDF vectorization
            vectorizer = TfidfVectorizer(
                stop_words='english',
                max_features=1000,
                ngram_range=(1, 2),
                lowercase=True
            )

            tfidf_matrix = vectorizer.fit_transform(corpus)

            # Calculate similarities (query is first document)
            similarities = cosine_similarity(tfidf_matrix[0:1], tfidf_matrix[1:]).flatten()

            # Add similarity scores to dataframe
            df = df.copy()
            df['similarity'] = similarities

            # Filter by threshold and sort
            df = df[df['similarity'] >= threshold]
            df = df.sort_values('similarity', ascending=False)

            return df

        except Exception as e:
            logger.warning(f"Cosine similarity calculation failed: {e}")
            # Return original dataframe if similarity fails
            df = df.copy()
            df['similarity'] = 0.0
            return df

    def _convert_to_search_results(self, df: pd.DataFrame, limit: int) -> List[SearchResult]:
        """Convert DataFrame rows to SearchResult objects."""
        results = []

        # Group by element_id to aggregate context
        grouped = df.groupby('element_id')

        count = 0
        for element_id, group in grouped:
            if count >= limit:
                break

            # Get the main element data (first row)
            row = group.iloc[0]

            element_data = {
                'element_id': row['element_id'],
                'element_type': row.get('element_type'),
                'content_preview': row.get('content_preview'),
                'doc_id': row.get('doc_id'),
                'parent_id': row.get('parent_id'),
                'metadata': json.loads(row.get('metadata', '{}')) if row.get('metadata') else {},
            }

            similarity = row.get('similarity', 0.0)
            result = SearchResult(element_data, similarity)

            # Add context information
            result.context = {
                'document': {
                    'source': row.get('doc_source'),
                    'metadata': json.loads(row.get('doc_metadata', '{}')) if row.get('doc_metadata') else {}
                },
                'parents': []
            }

            # Add parent hierarchy
            for i in range(1, 4):
                parent_id = row.get(f'parent{i}_id')
                if parent_id:
                    result.context['parents'].append({
                        'element_id': parent_id,
                        'element_type': row.get(f'parent{i}_type'),
                        'content_preview': row.get(f'parent{i}_content')
                    })

            # Add relationships
            for _, rel_row in group.iterrows():
                if rel_row.get('relationship_id'):
                    result.relationships.append({
                        'relationship_id': rel_row['relationship_id'],
                        'relationship_type': rel_row['relationship_type'],
                        'source_id': rel_row['source_id'],
                        'target_id': rel_row['target_id'],
                        'metadata': json.loads(rel_row.get('rel_metadata', '{}')) if rel_row.get('rel_metadata') else {}
                    })

            # Add siblings
            for _, sib_row in group.iterrows():
                if sib_row.get('sibling_id'):
                    result.siblings.append({
                        'element_id': sib_row['sibling_id'],
                        'element_type': sib_row['sibling_type'],
                        'content_preview': sib_row['sibling_content']
                    })

            results.append(result)
            count += 1

        return results

    def _reconstruct_documents(self, results: List[SearchResult], config: ReconstructionConfig) -> Dict[str, Any]:
        """Reconstruct full documents from search results."""
        materialized_docs = {}

        # Get unique document IDs from results
        doc_ids = set()
        for result in results:
            doc_id = result.element_data.get('doc_id')
            if doc_id:
                doc_ids.add(doc_id)

        # For each document, reconstruct the full structure
        for doc_id in doc_ids:
            try:
                doc_structure = self._get_document_structure(doc_id)
                if config.format == 'markdown':
                    materialized_docs[doc_id] = self._to_markdown(doc_structure)
                elif config.format == 'html':
                    materialized_docs[doc_id] = self._to_html(doc_structure)
                elif config.format == 'json':
                    materialized_docs[doc_id] = doc_structure
            except Exception as e:
                logger.warning(f"Failed to reconstruct document {doc_id}: {e}")

        return materialized_docs

    def _get_document_structure(self, doc_id: str) -> Dict[str, Any]:
        """Get full document structure from parquet files."""
        query = f"""
        SELECT * FROM read_parquet('{self.elements_path}', hive_partitioning=true)
        WHERE doc_id = '{doc_id}'
        ORDER BY parent_id NULLS FIRST, element_id
        """

        elements_df = self.conn.execute(query).df()

        # Build hierarchical structure
        # This is a simplified version - full implementation would need proper tree building
        return {
            'doc_id': doc_id,
            'elements': elements_df.to_dict('records'),
            'total_elements': len(elements_df)
        }

    def _to_markdown(self, doc_structure: Dict[str, Any]) -> str:
        """Convert document structure to markdown."""
        lines = [f"# Document: {doc_structure['doc_id']}\n"]

        for element in doc_structure['elements']:
            elem_type = element.get('element_type', 'unknown')
            content = element.get('content_preview', '')

            if elem_type == 'heading':
                lines.append(f"## {content}")
            elif elem_type == 'paragraph':
                lines.append(f"{content}\n")
            elif elem_type == 'list_item':
                lines.append(f"- {content}")
            else:
                lines.append(f"**{elem_type}**: {content}")

        return '\n'.join(lines)

    def _to_html(self, doc_structure: Dict[str, Any]) -> str:
        """Convert document structure to HTML."""
        lines = [f"<h1>Document: {doc_structure['doc_id']}</h1>"]

        for element in doc_structure['elements']:
            elem_type = element.get('element_type', 'unknown')
            content = element.get('content_preview', '')

            if elem_type == 'heading':
                lines.append(f"<h2>{content}</h2>")
            elif elem_type == 'paragraph':
                lines.append(f"<p>{content}</p>")
            elif elem_type == 'list_item':
                lines.append(f"<li>{content}</li>")
            else:
                lines.append(f"<div class='{elem_type}'>{content}</div>")

        return '\n'.join(lines)


def create_search_engine(config: Dict[str, Any]) -> ParquetSearchEngine:
    """
    Factory function to create search engine from configuration.

    Args:
        config: Configuration dictionary with storage backend info

    Returns:
        Appropriate search engine instance
    """
    storage_config = config.get('storage', {})
    analytics_config = config.get('analytics', {})

    # For now, only support parquet-based search from analytics outputs
    if analytics_config.get('enabled', False):
        outputs = analytics_config.get('outputs', [])
        for output in outputs:
            if output.get('type') == 'parquet':
                base_path = output.get('path', './analytics-output')
                return ParquetSearchEngine(base_path)

    # Fallback to default path
    return ParquetSearchEngine('./data-lake')