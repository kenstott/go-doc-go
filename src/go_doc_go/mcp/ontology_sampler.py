"""
MCP component for intelligent document and element sampling for ontology generation.
"""

import json
import logging
from typing import Dict, List, Any, Optional, Tuple
from dataclasses import dataclass
from collections import defaultdict

logger = logging.getLogger(__name__)


@dataclass
class SamplingConfig:
    """Configuration for document/element sampling."""
    max_elements: int = 200
    max_documents: int = 50
    min_elements_per_type: int = 5
    max_elements_per_type: int = 50
    include_temporal: bool = True
    include_rare_elements: bool = True
    diversity_factor: float = 0.3  # Balance between frequency and diversity


@dataclass
class OntologyContext:
    """Domain context for targeted sampling."""
    domain_name: str
    keywords: List[str]
    document_types: List[str]
    entity_hints: List[str]
    relationship_hints: List[str]


class OntologySampler:
    """Intelligent sampler for ontology generation data."""

    def __init__(self, db_connection, config: Optional[SamplingConfig] = None):
        """Initialize the sampler with database connection."""
        self.db = db_connection
        self.config = config or SamplingConfig()

    def sample_for_domain(self, context: OntologyContext) -> Dict[str, Any]:
        """
        Sample documents and elements optimized for a specific domain.

        Args:
            context: Domain-specific context for targeted sampling

        Returns:
            Comprehensive sampling data for ontology generation
        """
        logger.info(f"Sampling data for domain: {context.domain_name}")

        # Get corpus statistics
        stats = self._get_corpus_statistics(context)

        # Sample documents strategically
        documents = self._sample_documents(context, stats)

        # Sample elements with diversity and coverage
        elements = self._sample_elements(context, documents, stats)

        # Analyze patterns
        patterns = self._analyze_patterns(elements)

        # Generate metadata analysis
        metadata_analysis = self._analyze_metadata(elements)

        return {
            "domain_context": {
                "name": context.domain_name,
                "keywords": context.keywords,
                "document_types": context.document_types,
                "entity_hints": context.entity_hints,
                "relationship_hints": context.relationship_hints
            },
            "corpus_statistics": stats,
            "sampled_documents": documents,
            "sampled_elements": elements,
            "pattern_analysis": patterns,
            "metadata_analysis": metadata_analysis,
            "sampling_summary": {
                "total_elements": len(elements),
                "total_documents": len(documents),
                "element_types": len(set(e["element_type"] for e in elements)),
                "format_types": len(set(e["format_type"] for e in elements)),
                "has_temporal": sum(1 for e in elements if e.get("has_temporal_value"))
            }
        }

    def _get_corpus_statistics(self, context: OntologyContext) -> Dict[str, Any]:
        """Get comprehensive corpus statistics."""
        query = """
        SELECT
            format_type,
            document_category,
            element_type,
            structural_name,
            element_count,
            document_count,
            avg_content_length,
            avg_hierarchy_depth,
            temporal_element_count,
            sample_paths
        FROM element_sampling_stats
        WHERE element_count >= %s
        ORDER BY element_count DESC
        LIMIT 100
        """

        cursor = self.db.cursor()
        cursor.execute(query, (self.config.min_elements_per_type,))
        stats_rows = cursor.fetchall()

        # Convert to structured format
        stats = {
            "total_elements": 0,
            "total_documents": 0,
            "format_distribution": defaultdict(int),
            "category_distribution": defaultdict(int),
            "element_type_distribution": defaultdict(int),
            "top_structural_names": [],
            "temporal_coverage": 0
        }

        for row in stats_rows:
            stats["total_elements"] += row["element_count"]
            stats["total_documents"] += row["document_count"]
            stats["format_distribution"][row["format_type"]] += row["element_count"]
            stats["category_distribution"][row["document_category"]] += row["element_count"]
            stats["element_type_distribution"][row["element_type"]] += row["element_count"]
            stats["temporal_coverage"] += row["temporal_element_count"]

            stats["top_structural_names"].append({
                "name": row["structural_name"],
                "count": row["element_count"],
                "avg_depth": row["avg_hierarchy_depth"],
                "sample_paths": row["sample_paths"][:3] if row["sample_paths"] else []
            })

        return stats

    def _sample_documents(self, context: OntologyContext, stats: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Sample documents with strategic diversity."""

        # Build document sampling query with domain targeting
        domain_filters = []
        params = []

        # Target document types mentioned in context
        if context.document_types:
            type_conditions = []
            for doc_type in context.document_types:
                type_conditions.append("document_category ILIKE %s OR source ILIKE %s")
                params.extend([f"%{doc_type}%", f"%{doc_type}%"])
            domain_filters.append(f"({' OR '.join(type_conditions)})")

        # Target documents with domain keywords
        if context.keywords:
            keyword_conditions = []
            for keyword in context.keywords:
                keyword_conditions.append("source ILIKE %s OR document_metadata::text ILIKE %s")
                params.extend([f"%{keyword}%", f"%{keyword}%"])
            domain_filters.append(f"({' OR '.join(keyword_conditions)})")

        where_clause = " AND ".join(domain_filters) if domain_filters else "TRUE"

        query = f"""
        WITH document_samples AS (
            SELECT DISTINCT
                doc_id,
                source,
                doc_type,
                document_category,
                format_type,
                document_metadata,
                COUNT(*) as element_count
            FROM element_document_enriched
            WHERE {where_clause}
            GROUP BY doc_id, source, doc_type, document_category, format_type, document_metadata
            HAVING COUNT(*) >= 5
        ),
        stratified_sample AS (
            SELECT *,
                ROW_NUMBER() OVER (PARTITION BY format_type ORDER BY element_count DESC) as rn_format,
                ROW_NUMBER() OVER (PARTITION BY document_category ORDER BY element_count DESC) as rn_category
            FROM document_samples
        )
        SELECT *
        FROM stratified_sample
        WHERE rn_format <= 10 AND rn_category <= 10
        ORDER BY element_count DESC
        LIMIT %s
        """

        params.append(self.config.max_documents)

        cursor = self.db.cursor()
        cursor.execute(query, params)
        return [dict(row) for row in cursor.fetchall()]

    def _sample_elements(self, context: OntologyContext, documents: List[Dict], stats: Dict) -> List[Dict[str, Any]]:
        """Sample elements with intelligent diversity and coverage."""

        doc_ids = [doc["doc_id"] for doc in documents]
        if not doc_ids:
            return []

        # Build element sampling with multiple strategies

        # Strategy 1: High-frequency structural names (common patterns)
        high_freq_query = """
        SELECT *
        FROM element_document_enriched
        WHERE doc_id = ANY(%s)
        AND structural_name IN (
            SELECT structural_name
            FROM element_sampling_stats
            WHERE element_count >= 100
            ORDER BY element_count DESC
            LIMIT 20
        )
        ORDER BY RANDOM()
        LIMIT %s
        """

        # Strategy 2: Domain-relevant elements (keyword matching)
        domain_conditions = []
        domain_params = [doc_ids]
        if context.keywords + context.entity_hints:
            all_keywords = context.keywords + context.entity_hints
            keyword_conditions = []
            for keyword in all_keywords:
                keyword_conditions.append(
                    "structural_name ILIKE %s OR content_preview ILIKE %s OR structural_path ILIKE %s"
                )
                domain_params.extend([f"%{keyword}%", f"%{keyword}%", f"%{keyword}%"])
            domain_conditions.append(f"({' OR '.join(keyword_conditions)})")

        domain_where = " AND " + " AND ".join(domain_conditions) if domain_conditions else ""

        domain_query = f"""
        SELECT *
        FROM element_document_enriched
        WHERE doc_id = ANY(%s)
        {domain_where}
        ORDER BY RANDOM()
        LIMIT %s
        """

        # Strategy 3: Diverse element types
        diversity_query = """
        WITH type_samples AS (
            SELECT *,
                ROW_NUMBER() OVER (PARTITION BY element_type ORDER BY RANDOM()) as rn
            FROM element_document_enriched
            WHERE doc_id = ANY(%s)
        )
        SELECT *
        FROM type_samples
        WHERE rn <= %s
        LIMIT %s
        """

        # Strategy 4: Temporal elements (always valuable)
        temporal_query = """
        SELECT *
        FROM element_document_enriched
        WHERE doc_id = ANY(%s)
        AND has_temporal_value = TRUE
        ORDER BY RANDOM()
        LIMIT %s
        """

        cursor = self.db.cursor()
        all_elements = []

        # Execute sampling strategies
        try:
            # High-frequency elements
            cursor.execute(high_freq_query, (doc_ids, self.config.max_elements // 4))
            all_elements.extend([dict(row) for row in cursor.fetchall()])

            # Domain-relevant elements
            if domain_conditions:
                cursor.execute(domain_query, domain_params + [self.config.max_elements // 4])
                all_elements.extend([dict(row) for row in cursor.fetchall()])

            # Diverse element types
            cursor.execute(diversity_query, (doc_ids, 10, self.config.max_elements // 4))
            all_elements.extend([dict(row) for row in cursor.fetchall()])

            # Temporal elements
            if self.config.include_temporal:
                cursor.execute(temporal_query, (doc_ids, self.config.max_elements // 4))
                all_elements.extend([dict(row) for row in cursor.fetchall()])

        except Exception as e:
            logger.error(f"Error sampling elements: {e}")
            # Fallback to simple sampling
            cursor.execute("""
                SELECT * FROM element_document_enriched
                WHERE doc_id = ANY(%s)
                ORDER BY RANDOM()
                LIMIT %s
            """, (doc_ids, self.config.max_elements))
            all_elements = [dict(row) for row in cursor.fetchall()]

        # Deduplicate by element_id
        seen_ids = set()
        unique_elements = []
        for element in all_elements:
            if element["element_id"] not in seen_ids:
                seen_ids.add(element["element_id"])
                unique_elements.append(element)

        return unique_elements[:self.config.max_elements]

    def _analyze_patterns(self, elements: List[Dict]) -> Dict[str, Any]:
        """Analyze patterns in sampled elements."""
        patterns = {
            "structural_name_frequency": defaultdict(int),
            "element_type_frequency": defaultdict(int),
            "path_patterns": defaultdict(int),
            "attribute_patterns": defaultdict(int),
            "temporal_patterns": [],
            "hierarchy_patterns": defaultdict(list)
        }

        for element in elements:
            # Structural name frequency
            patterns["structural_name_frequency"][element.get("structural_name", "unknown")] += 1

            # Element type frequency
            patterns["element_type_frequency"][element["element_type"]] += 1

            # Path patterns
            path = element.get("structural_path")
            if path:
                # Extract path patterns
                path_parts = path.split("/")
                for i in range(len(path_parts) - 1):
                    pattern = "/".join(path_parts[:i+2])
                    patterns["path_patterns"][pattern] += 1

            # Temporal patterns
            if element.get("has_temporal_value"):
                patterns["temporal_patterns"].append({
                    "structural_name": element.get("structural_name"),
                    "path": element.get("structural_path"),
                    "content": element.get("content_preview")
                })

            # Hierarchy patterns
            depth = element.get("hierarchy_depth", 0)
            patterns["hierarchy_patterns"][depth].append(element.get("structural_name"))

        # Convert to sorted lists for readability
        patterns["structural_name_frequency"] = dict(sorted(
            patterns["structural_name_frequency"].items(),
            key=lambda x: x[1], reverse=True
        )[:20])

        patterns["element_type_frequency"] = dict(sorted(
            patterns["element_type_frequency"].items(),
            key=lambda x: x[1], reverse=True
        ))

        patterns["path_patterns"] = dict(sorted(
            patterns["path_patterns"].items(),
            key=lambda x: x[1], reverse=True
        )[:30])

        return patterns

    def _analyze_metadata(self, elements: List[Dict]) -> Dict[str, Any]:
        """Analyze metadata patterns for ontology rules."""
        analysis = {
            "element_name_coverage": {},
            "path_coverage": {},
            "attribute_usage": {},
            "content_patterns": {},
            "format_specific_patterns": defaultdict(dict)
        }

        # Group by format type for specific analysis
        by_format = defaultdict(list)
        for element in elements:
            by_format[element.get("format_type", "unknown")].append(element)

        for format_type, format_elements in by_format.items():
            format_analysis = {
                "common_names": defaultdict(int),
                "common_paths": defaultdict(int),
                "sample_elements": format_elements[:5]  # Sample for inspection
            }

            for element in format_elements:
                if element.get("structural_name"):
                    format_analysis["common_names"][element["structural_name"]] += 1
                if element.get("structural_path"):
                    format_analysis["common_paths"][element["structural_path"]] += 1

            # Sort by frequency
            format_analysis["common_names"] = dict(sorted(
                format_analysis["common_names"].items(),
                key=lambda x: x[1], reverse=True
            )[:10])

            format_analysis["common_paths"] = dict(sorted(
                format_analysis["common_paths"].items(),
                key=lambda x: x[1], reverse=True
            )[:10])

            analysis["format_specific_patterns"][format_type] = format_analysis

        return analysis


def create_mcp_tools():
    """Create MCP tools for ontology sampling."""

    def sample_for_ontology(domain_name: str, domain_description: str,
                           keywords: str = "", document_types: str = "",
                           entity_hints: str = "", relationship_hints: str = "",
                           max_elements: int = 200) -> str:
        """
        Sample documents and elements for ontology generation.

        Args:
            domain_name: Name of the domain (e.g., "insider_trading")
            domain_description: Detailed description of the domain
            keywords: Comma-separated keywords to target
            document_types: Comma-separated document types to focus on
            entity_hints: Comma-separated entity types to look for
            relationship_hints: Comma-separated relationship types to consider
            max_elements: Maximum number of elements to sample

        Returns:
            JSON string with comprehensive sampling data
        """
        from ..storage.factory import StorageFactory

        # Get database connection
        config = {}  # Load from config
        storage = StorageFactory.create_storage("postgresql", config)

        # Create context
        context = OntologyContext(
            domain_name=domain_name,
            keywords=[k.strip() for k in keywords.split(",") if k.strip()],
            document_types=[t.strip() for t in document_types.split(",") if t.strip()],
            entity_hints=[e.strip() for e in entity_hints.split(",") if e.strip()],
            relationship_hints=[r.strip() for r in relationship_hints.split(",") if r.strip()]
        )

        # Create sampler
        sampler_config = SamplingConfig(max_elements=max_elements)
        sampler = OntologySampler(storage.connection, sampler_config)

        # Sample data
        sampling_data = sampler.sample_for_domain(context)

        return json.dumps(sampling_data, indent=2, default=str)

    return {
        "sample_for_ontology": sample_for_ontology
    }