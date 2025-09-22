"""
Neo4j analytics storage adapter for graph-based OLAP operations.
"""

import json
import logging
from datetime import datetime
from typing import Dict, Any, List, Optional

from .base import AnalyticsStorage

logger = logging.getLogger(__name__)

try:
    from neo4j import GraphDatabase
    NEO4J_AVAILABLE = True
except ImportError:
    NEO4J_AVAILABLE = False
    logger.warning("Neo4j not available. Install with: pip install neo4j")


class Neo4jAnalyticsStorage(AnalyticsStorage):
    """Neo4j implementation of analytics storage for graph analytics."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Neo4j analytics storage.
        
        Args:
            config: Configuration with uri, username, password, database
        """
        super().__init__(config)
        
        if not NEO4J_AVAILABLE:
            raise ImportError("neo4j required for Neo4j analytics storage")
        
        self.uri = config.get('uri', 'bolt://localhost:7687')
        self.username = config.get('username', 'neo4j')
        self.password = config.get('password', 'password')
        self.database = config.get('database', 'neo4j')
        
        self.driver = None
        self.initialize()
    
    def initialize(self) -> None:
        """Initialize Neo4j connection."""
        try:
            self.driver = GraphDatabase.driver(
                self.uri,
                auth=(self.username, self.password)
            )
            
            # Test connection and create indexes
            with self.driver.session(database=self.database) as session:
                # Create indexes for efficient queries
                session.run("CREATE INDEX IF NOT EXISTS FOR (d:Document) ON (d.doc_id)")
                session.run("CREATE INDEX IF NOT EXISTS FOR (d:Document) ON (d.run_id)")
                session.run("CREATE INDEX IF NOT EXISTS FOR (e:Element) ON (e.element_id)")
                session.run("CREATE INDEX IF NOT EXISTS FOR (e:Element) ON (e.run_id)")
                session.run("CREATE INDEX IF NOT EXISTS FOR (r:Run) ON (r.run_id)")
                
                # Test connection
                result = session.run("RETURN 1 as test")
                result.single()
            
            logger.info(f"Neo4j analytics storage initialized: {self.uri}")
            
        except Exception as e:
            logger.error(f"Failed to initialize Neo4j analytics storage: {e}")
            raise
    
    def _add_partition_properties(self, properties: Dict[str, Any], run_id: str) -> Dict[str, Any]:
        """Add partition properties for Neo4j node."""
        now = datetime.now()
        properties['run_id'] = run_id
        properties['written_at'] = now.isoformat()
        
        if 'year' in self.partitioning:
            properties['year'] = now.year
        if 'month' in self.partitioning:
            properties['month'] = now.month
        if 'day' in self.partitioning:
            properties['day'] = now.day
        
        # Convert complex types to JSON strings
        for key, value in list(properties.items()):
            if isinstance(value, (dict, list)):
                properties[f"{key}_json"] = json.dumps(value)
                del properties[key]
        
        return properties
    
    def append_documents(self, documents: List[Dict[str, Any]], 
                        run_id: str) -> int:
        """Append documents as nodes to Neo4j."""
        if not documents:
            return 0
        
        with self.driver.session(database=self.database) as session:
            count = 0
            
            for doc in documents:
                properties = self._add_partition_properties(doc.copy(), run_id)
                
                # Create Document node
                result = session.run("""
                    MERGE (d:Document {doc_id: $doc_id, run_id: $run_id})
                    SET d += $properties
                    RETURN d
                """, doc_id=doc.get('doc_id'), run_id=run_id, properties=properties)
                
                if result.single():
                    count += 1
                
                # Link to Run node
                session.run("""
                    MERGE (r:Run {run_id: $run_id})
                    MERGE (d:Document {doc_id: $doc_id, run_id: $run_id})
                    MERGE (r)-[:CONTAINS]->(d)
                """, run_id=run_id, doc_id=doc.get('doc_id'))
        
        logger.info(f"Appended {count} documents to Neo4j")
        return count
    
    def append_elements(self, elements: List[Dict[str, Any]], 
                       run_id: str) -> int:
        """Append elements as nodes to Neo4j."""
        if not elements:
            return 0
        
        with self.driver.session(database=self.database) as session:
            count = 0
            
            for elem in elements:
                properties = self._add_partition_properties(elem.copy(), run_id)
                
                # Create Element node
                result = session.run("""
                    MERGE (e:Element {element_id: $element_id, run_id: $run_id})
                    SET e += $properties
                    RETURN e
                """, element_id=elem.get('element_id'), run_id=run_id, properties=properties)
                
                if result.single():
                    count += 1
                
                # Link to Document if doc_id present
                if 'doc_id' in elem:
                    session.run("""
                        MERGE (d:Document {doc_id: $doc_id, run_id: $run_id})
                        MERGE (e:Element {element_id: $element_id, run_id: $run_id})
                        MERGE (d)-[:HAS_ELEMENT]->(e)
                    """, doc_id=elem['doc_id'], element_id=elem.get('element_id'), run_id=run_id)
                
                # Create parent-child relationships if parent_id present
                if 'parent_id' in elem and elem['parent_id']:
                    session.run("""
                        MERGE (p:Element {element_id: $parent_id, run_id: $run_id})
                        MERGE (c:Element {element_id: $element_id, run_id: $run_id})
                        MERGE (p)-[:CONTAINS]->(c)
                    """, parent_id=elem['parent_id'], element_id=elem.get('element_id'), run_id=run_id)
        
        logger.info(f"Appended {count} elements to Neo4j")
        return count
    
    def append_embeddings(self, embeddings: List[Dict[str, Any]], 
                         run_id: str) -> int:
        """Append embeddings as properties on Element nodes."""
        if not embeddings:
            return 0
        
        with self.driver.session(database=self.database) as session:
            count = 0
            
            for emb in embeddings:
                # Convert embedding vector to list if needed
                vector = emb.get('embedding', [])
                if not isinstance(vector, list):
                    vector = list(vector)
                
                # Update Element node with embedding
                result = session.run("""
                    MATCH (e:Element {element_id: $element_id, run_id: $run_id})
                    SET e.embedding = $embedding,
                        e.embedding_model = $model,
                        e.embedding_dims = $dims
                    RETURN e
                """, 
                    element_id=emb.get('element_id'),
                    run_id=run_id,
                    embedding=vector,
                    model=emb.get('model', 'unknown'),
                    dims=len(vector)
                )
                
                if result.single():
                    count += 1
        
        logger.info(f"Appended {count} embeddings to Neo4j")
        return count
    
    def append_relationships(self, relationships: List[Dict[str, Any]], 
                           run_id: str) -> int:
        """Append relationships as edges in Neo4j."""
        if not relationships:
            return 0
        
        with self.driver.session(database=self.database) as session:
            count = 0
            
            for rel in relationships:
                rel_type = rel.get('relationship_type', 'RELATED_TO').upper()
                
                # Create relationship between elements
                result = session.run(f"""
                    MATCH (s:Element {{element_id: $source_id, run_id: $run_id}})
                    MATCH (t:Element {{element_id: $target_id, run_id: $run_id}})
                    MERGE (s)-[r:{rel_type}]->(t)
                    SET r.created_at = $created_at
                    SET r += $metadata
                    RETURN r
                """,
                    source_id=rel.get('source_id'),
                    target_id=rel.get('target_id'),
                    run_id=run_id,
                    created_at=datetime.now().isoformat(),
                    metadata=rel.get('metadata', {})
                )
                
                if result.single():
                    count += 1
        
        logger.info(f"Appended {count} relationships to Neo4j")
        return count
    
    def append_metrics(self, metrics: Dict[str, Any], 
                      run_id: str) -> bool:
        """Append processing metrics to Run node."""
        with self.driver.session(database=self.database) as session:
            properties = self._add_partition_properties(metrics.copy(), run_id)
            
            result = session.run("""
                MERGE (r:Run {run_id: $run_id})
                SET r += $properties
                RETURN r
            """, run_id=run_id, properties=properties)
            
            return result.single() is not None
    
    def get_partition_path(self, run_id: str, data_type: str) -> str:
        """Get logical partition path (Neo4j doesn't use file paths)."""
        return f"neo4j://{self.database}/{data_type}/{run_id}"
    
    def list_runs(self, start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None) -> List[str]:
        """List processing runs in date range."""
        with self.driver.session(database=self.database) as session:
            # Build query
            where_clause = ""
            params = {}
            
            if start_date or end_date:
                conditions = []
                if start_date:
                    conditions.append("r.written_at >= $start_date")
                    params['start_date'] = start_date.isoformat()
                if end_date:
                    conditions.append("r.written_at <= $end_date")
                    params['end_date'] = end_date.isoformat()
                
                where_clause = f"WHERE {' AND '.join(conditions)}"
            
            result = session.run(f"""
                MATCH (r:Run)
                {where_clause}
                RETURN r.run_id as run_id
                ORDER BY r.written_at DESC
            """, **params)
            
            return [record['run_id'] for record in result]
    
    def get_run_stats(self, run_id: str) -> Optional[Dict[str, Any]]:
        """Get statistics for a processing run."""
        with self.driver.session(database=self.database) as session:
            # Get Run node
            run_result = session.run("""
                MATCH (r:Run {run_id: $run_id})
                RETURN r
            """, run_id=run_id)
            
            run_record = run_result.single()
            if not run_record:
                return None
            
            stats = dict(run_record['r'])
            
            # Count documents
            doc_result = session.run("""
                MATCH (d:Document {run_id: $run_id})
                RETURN count(d) as count
            """, run_id=run_id)
            stats['document_count'] = doc_result.single()['count']
            
            # Count elements
            elem_result = session.run("""
                MATCH (e:Element {run_id: $run_id})
                RETURN count(e) as count
            """, run_id=run_id)
            stats['element_count'] = elem_result.single()['count']
            
            # Count relationships
            rel_result = session.run("""
                MATCH (s:Element {run_id: $run_id})-[r]->(t:Element {run_id: $run_id})
                RETURN count(r) as count
            """, run_id=run_id)
            stats['relationship_count'] = rel_result.single()['count']
            
            # Count embeddings
            emb_result = session.run("""
                MATCH (e:Element {run_id: $run_id})
                WHERE e.embedding IS NOT NULL
                RETURN count(e) as count
            """, run_id=run_id)
            stats['embedding_count'] = emb_result.single()['count']
            
            return stats
    
    def close(self) -> None:
        """Close Neo4j connection."""
        if self.driver:
            self.driver.close()
            self.driver = None

    # TODO: Implement sampling methods for MCP database sampling
    def sample_elements(self, filters: Optional[Dict] = None, limit: int = 100,
                       stratify_by: Optional[str] = None,
                       random_seed: Optional[int] = None) -> List[Dict[str, Any]]:
        """
        Sample elements from Neo4j graph with flexible filtering.

        TODO: Implement Cypher queries for sampling
        - Use RAND() function for random sampling
        - Implement stratified sampling with COLLECT and UNWIND
        - Apply filters with WHERE clauses
        """
        raise NotImplementedError("Neo4j sampling methods not yet implemented")

    def get_corpus_stats(self, filters: Optional[Dict] = None) -> Dict[str, Any]:
        """
        Get comprehensive statistics about the graph corpus.

        TODO: Implement Cypher aggregations for statistics
        - Count nodes by label
        - Count relationships by type
        - Calculate graph metrics (density, components)
        """
        raise NotImplementedError("Neo4j corpus stats not yet implemented")

    def sample_documents(self, filters: Optional[Dict] = None, limit: int = 50,
                        random_seed: Optional[int] = None) -> List[Dict[str, Any]]:
        """
        Sample document nodes from Neo4j.

        TODO: Implement document node sampling
        - Query Document nodes with filters
        - Use RAND() for random selection
        """
        raise NotImplementedError("Neo4j document sampling not yet implemented")

    def execute_custom_query(self, query: str,
                            params: Optional[List] = None) -> List[Dict[str, Any]]:
        """
        Execute custom Cypher query.

        TODO: Implement safe execution of Cypher queries
        - Validate read-only query (no CREATE, SET, DELETE, REMOVE)
        - Execute with parameters
        - Return results as list of dicts
        """
        raise NotImplementedError("Neo4j custom query execution not yet implemented")