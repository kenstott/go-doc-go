"""
Neo4j exporter for domain entity extraction results.
Exports entities, relationships, and documents to Neo4j for visualization and analysis.
"""
import logging
from typing import Dict, List, Any, Optional, Tuple
from dataclasses import dataclass
from datetime import datetime
import json

try:
    from neo4j import GraphDatabase, Transaction
    NEO4J_AVAILABLE = True
except ImportError:
    NEO4J_AVAILABLE = False
    
logger = logging.getLogger(__name__)


@dataclass
class Neo4jConfig:
    """Configuration for Neo4j connection."""
    uri: str = "bolt://localhost:7687"
    username: str = "neo4j"
    password: str = "go-doc-go123"
    database: str = "neo4j"
    max_connection_lifetime: int = 3600
    max_connection_pool_size: int = 50


class Neo4jExporter:
    """
    Exports domain extraction results to Neo4j knowledge graph.
    
    Creates a graph structure with:
    - Document nodes
    - Element nodes
    - Term (Entity) nodes
    - Relationships between them
    """
    
    def __init__(self, config: Optional[Neo4jConfig] = None):
        """
        Initialize Neo4j exporter.
        
        Args:
            config: Neo4j connection configuration
        """
        if not NEO4J_AVAILABLE:
            raise ImportError("neo4j-driver package is required. Install with: pip install neo4j")
            
        self.config = config or Neo4jConfig()
        self.driver = None
        self._connect()
        
    def _connect(self):
        """Establish connection to Neo4j."""
        try:
            self.driver = GraphDatabase.driver(
                self.config.uri,
                auth=(self.config.username, self.config.password),
                max_connection_lifetime=self.config.max_connection_lifetime,
                max_connection_pool_size=self.config.max_connection_pool_size
            )
            # Verify connectivity
            self.driver.verify_connectivity()
            logger.info(f"Connected to Neo4j at {self.config.uri}")
        except Exception as e:
            logger.error(f"Failed to connect to Neo4j: {e}")
            raise
            
    def close(self):
        """Close Neo4j connection."""
        if self.driver:
            self.driver.close()
            logger.info("Closed Neo4j connection")
            
    def __enter__(self):
        """Context manager entry."""
        return self
        
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()
        
    def create_indexes(self):
        """Create indexes and constraints for better performance."""
        with self.driver.session(database=self.config.database) as session:
            # Create unique constraints (which also create indexes)
            constraints = [
                "CREATE CONSTRAINT IF NOT EXISTS FOR (d:Document) REQUIRE d.doc_id IS UNIQUE",
                "CREATE CONSTRAINT IF NOT EXISTS FOR (e:Element) REQUIRE e.element_id IS UNIQUE",
                "CREATE CONSTRAINT IF NOT EXISTS FOR (t:Term) REQUIRE t.term_id IS UNIQUE",
                "CREATE CONSTRAINT IF NOT EXISTS FOR (entity:Entity) REQUIRE entity.entity_id IS UNIQUE",
            ]
            
            # Create additional indexes for frequently queried properties
            indexes = [
                "CREATE INDEX IF NOT EXISTS FOR (d:Document) ON (d.doc_type)",
                "CREATE INDEX IF NOT EXISTS FOR (d:Document) ON (d.source)",
                "CREATE INDEX IF NOT EXISTS FOR (e:Element) ON (e.element_type)",
                "CREATE INDEX IF NOT EXISTS FOR (t:Term) ON (t.domain)",
                "CREATE INDEX IF NOT EXISTS FOR (t:Term) ON (t.label)",
                "CREATE INDEX IF NOT EXISTS FOR (entity:Entity) ON (entity.entity_type)",
                "CREATE INDEX IF NOT EXISTS FOR (entity:Entity) ON (entity.name)",
                "CREATE INDEX IF NOT EXISTS FOR (entity:Entity) ON (entity.domain)",
                "CREATE INDEX IF NOT EXISTS FOR ()-[r:MAPPED_TO]-() ON (r.confidence)",
                "CREATE INDEX IF NOT EXISTS FOR ()-[r:DOMAIN_RELATIONSHIP]-() ON (r.relationship_type)",
                "CREATE INDEX IF NOT EXISTS FOR ()-[r:DERIVED_FROM]-() ON (r.extraction_rule)",
            ]
            
            for constraint in constraints:
                try:
                    session.run(constraint)
                    logger.debug(f"Created constraint: {constraint[:50]}...")
                except Exception as e:
                    logger.warning(f"Constraint may already exist: {e}")
                    
            for index in indexes:
                try:
                    session.run(index)
                    logger.debug(f"Created index: {index[:50]}...")
                except Exception as e:
                    logger.warning(f"Index may already exist: {e}")
                    
            logger.info("Created Neo4j indexes and constraints")
            
    def clear_graph(self):
        """Clear all nodes and relationships from the graph."""
        with self.driver.session(database=self.config.database) as session:
            session.run("MATCH (n) DETACH DELETE n")
            logger.info("Cleared Neo4j graph")
            
    def export_documents(self, documents: List[Dict[str, Any]], domain_config: Dict[str, Any] = None) -> int:
        """
        Export documents to Neo4j with optional domain configuration for attribute flattening.
        
        Args:
            documents: List of document dictionaries
            domain_config: Optional domain configuration with neo4j_export settings
            
        Returns:
            Number of documents created
        """
        # Extract export config if provided
        export_config = domain_config.get('neo4j_export', {}) if domain_config else {}
        doc_attributes = export_config.get('document_attributes', [])
        title_template = export_config.get('title_templates', {}).get('document', '')
        
        with self.driver.session(database=self.config.database) as session:
            # Prepare documents for Neo4j first to know all fields
            neo4j_docs = []
            all_fields = set()  # Track all fields for dynamic query
            
            for doc in documents:
                metadata = doc.get('metadata', {})
                
                # Generate title
                title = self._generate_title(doc, metadata, title_template)
                if not title:
                    title = metadata.get('title', f"{doc.get('doc_type', 'Document')} - {doc.get('doc_id', 'unknown')}")
                
                neo4j_doc = {
                    'doc_id': doc.get('doc_id'),
                    'doc_type': doc.get('doc_type', 'unknown'),
                    'source': doc.get('source', ''),
                    'title': title,
                    'created_at': doc.get('created_at', datetime.now().isoformat())
                }
                
                # Flatten specified attributes
                for attr in doc_attributes:
                    if attr in metadata:
                        neo4j_doc[attr] = metadata[attr]
                
                # Flatten ALL remaining metadata as top-level properties with metadata_ prefix
                remaining_metadata = {k: v for k, v in metadata.items() if k not in doc_attributes}
                flattened_remaining = self._flatten_metadata(remaining_metadata)
                for key, value in flattened_remaining.items():
                    # Add with metadata_ prefix to distinguish from configured attributes
                    field_name = f'metadata_{key}'
                    neo4j_doc[field_name] = value
                    all_fields.add(field_name)
                
                # Track all fields
                all_fields.update(neo4j_doc.keys())
                neo4j_docs.append(neo4j_doc)
            
            # Build dynamic query with all fields
            set_clauses = [f'd.{field} = doc.{field}' for field in sorted(all_fields)]
            
            query = f"""
            UNWIND $documents AS doc
            MERGE (d:Document {{doc_id: doc.doc_id}})
            SET {', '.join(set_clauses)}
            RETURN count(d) as count
            """
            
            result = session.run(query, documents=neo4j_docs)
            count = result.single()['count']
            logger.info(f"Exported {count} documents to Neo4j")
            return count
            
    def _generate_title(self, data: Dict[str, Any], metadata: Dict[str, Any], template: str) -> str:
        """Generate title from template."""
        if not template:
            return ""
        
        title = template
        # Replace placeholders
        for key, value in metadata.items():
            title = title.replace(f"{{{key}}}", str(value) if value else "")
        for key, value in data.items():
            if key != 'metadata':
                title = title.replace(f"{{{key}}}", str(value) if value else "")
        
        # Clean up any remaining placeholders
        import re
        title = re.sub(r'\{[^}]+\}', '', title)
        title = ' '.join(title.split())  # Clean up multiple spaces
        return title.strip()
    
    def _flatten_metadata(self, metadata: Dict[str, Any]) -> Dict[str, Any]:
        """Flatten nested metadata structures for Neo4j compatibility."""
        flattened = {}
        for key, value in metadata.items():
            if isinstance(value, dict):
                # Flatten nested dictionaries with prefix
                for nested_key, nested_value in value.items():
                    if not isinstance(nested_value, (dict, list)):
                        flattened[f"{key}_{nested_key}"] = nested_value
            elif isinstance(value, list):
                # Only store lists of primitives
                if value and not isinstance(value[0], (dict, list)):
                    flattened[key] = value
            else:
                # Store primitive values directly
                flattened[key] = value
        return flattened
    
    def export_elements(self, elements: List[Dict[str, Any]], domain_config: Dict[str, Any] = None) -> int:
        """
        Export elements to Neo4j with optional domain configuration for attribute flattening.
        
        Args:
            elements: List of element dictionaries
            domain_config: Optional domain configuration with neo4j_export settings
            
        Returns:
            Number of elements created
        """
        # Extract export config if provided
        export_config = domain_config.get('neo4j_export', {}) if domain_config else {}
        elem_attributes = export_config.get('element_attributes', [])
        title_templates = export_config.get('title_templates', {}).get('element', {})
        
        with self.driver.session(database=self.config.database) as session:
            # Prepare elements for Neo4j first to know all fields
            neo4j_elements = []
            all_fields = set()  # Track all fields for dynamic query
            
            for elem in elements:
                metadata = elem.get('metadata', {})
                elem_type = elem.get('element_type', 'unknown')
                
                # Generate title based on element type
                if isinstance(title_templates, dict):
                    template = title_templates.get(elem_type, title_templates.get('default', ''))
                else:
                    template = title_templates
                
                title = self._generate_title(elem, metadata, template)
                if not title:
                    # Fallback title generation
                    content_preview = elem.get('content_preview', '')[:30]
                    if metadata.get('speaker'):
                        title = f"{metadata['speaker']} - {content_preview}"
                    elif metadata.get('section'):
                        title = f"{metadata['section']} - {content_preview}"
                    else:
                        title = f"{elem_type} {elem.get('document_position', 0)}"
                
                neo4j_elem = {
                    'element_id': elem.get('element_id'),
                    'doc_id': elem.get('doc_id'),
                    'element_type': elem_type,
                    'content_preview': elem.get('content_preview', '')[:500],
                    'position': elem.get('document_position', 0),
                    'title': title
                }
                
                # Flatten specified attributes
                for attr in elem_attributes:
                    if attr in metadata:
                        neo4j_elem[attr] = metadata[attr]
                
                # Flatten ALL remaining metadata as top-level properties with metadata_ prefix
                remaining_metadata = {k: v for k, v in metadata.items() if k not in elem_attributes}
                flattened_remaining = self._flatten_metadata(remaining_metadata)
                for key, value in flattened_remaining.items():
                    # Add with metadata_ prefix to distinguish from configured attributes
                    field_name = f'metadata_{key}'
                    neo4j_elem[field_name] = value
                    all_fields.add(field_name)
                
                # Track all fields
                all_fields.update(neo4j_elem.keys())
                neo4j_elements.append(neo4j_elem)
            
            # Build dynamic query with all fields
            set_clauses = [f'e.{field} = elem.{field}' for field in sorted(all_fields)]
            
            element_query = f"""
            UNWIND $elements AS elem
            MERGE (e:Element {{element_id: elem.element_id}})
            SET {', '.join(set_clauses)}
            WITH e, elem
            MATCH (d:Document {{doc_id: elem.doc_id}})
            MERGE (e)-[:BELONGS_TO]->(d)
            RETURN count(e) as count
            """
            
            result = session.run(element_query, elements=neo4j_elements)
            count = result.single()['count']
            logger.info(f"Exported {count} elements to Neo4j")
            return count
            
    def export_terms(self, terms: List[Any], domain: str) -> int:
        """
        Export domain terms to Neo4j.
        
        Args:
            terms: List of term objects or dictionaries from ontology
            domain: Domain name
            
        Returns:
            Number of terms created
        """
        with self.driver.session(database=self.config.database) as session:
            query = """
            UNWIND $terms AS term
            MERGE (t:Term {term_id: term.term_id})
            SET t.label = term.label,
                t.domain = term.domain,
                t.description = term.description,
                t.aliases = term.aliases
            RETURN count(t) as count
            """
            
            # Prepare terms for Neo4j
            neo4j_terms = []
            for term in terms:
                # Handle both Term objects and dictionaries
                if hasattr(term, 'id'):  # Term object
                    neo4j_term = {
                        'term_id': f"{domain}:{term.id}",
                        'label': term.label,
                        'domain': domain,
                        'description': term.description,
                        'aliases': term.aliases if term.aliases else []
                    }
                else:  # Dictionary
                    neo4j_term = {
                        'term_id': f"{domain}:{term.get('id', '')}",
                        'label': term.get('label', ''),
                        'domain': domain,
                        'description': term.get('description', ''),
                        'aliases': term.get('aliases', [])
                    }
                neo4j_terms.append(neo4j_term)
                
            result = session.run(query, terms=neo4j_terms)
            count = result.single()['count']
            logger.info(f"Exported {count} terms to Neo4j")
            return count
            
    def export_entities(self, entities: List[Dict[str, Any]], domain_config: Dict[str, Any] = None) -> int:
        """
        Export entities to Neo4j with multiple labels and configured attributes.
        
        Args:
            entities: List of entity dictionaries
            domain_config: Optional domain configuration with neo4j_export settings
            
        Returns:
            Number of entities created
        """
        # Extract export config if provided
        export_config = domain_config.get('neo4j_export', {}) if domain_config else {}
        entity_attributes = export_config.get('entity_attributes', [])
        
        with self.driver.session(database=self.config.database) as session:
            # Prepare entities for Neo4j first to know all fields
            neo4j_entities = []
            all_fields = set()  # Track all fields for dynamic query
            
            for entity in entities:
                entity_type = entity.get('entity_type', 'unknown')
                name = entity.get('name', '')
                domain = entity.get('domain', 'unknown')
                
                # Handle attributes - could be dict or JSON string
                attributes = entity.get('attributes', {})
                if isinstance(attributes, str):
                    try:
                        attributes = json.loads(attributes)
                    except (json.JSONDecodeError, ValueError):
                        logger.warning(f"Could not parse attributes JSON: {attributes}")
                        attributes = {}
                elif not isinstance(attributes, dict):
                    attributes = {}
                
                # Create base entity data
                neo4j_entity = {
                    'entity_id': entity.get('entity_id'),
                    'entity_type': entity_type,
                    'name': name,
                    'domain': domain,
                    'created_at': entity.get('created_at', datetime.now().isoformat())
                }
                
                # Create labels for Neo4j (multiple labels)
                # Always include Entity as base label, plus specific type label
                labels = ['Entity']
                if entity_type and entity_type != 'unknown':
                    # Capitalize first letter for Neo4j label convention
                    type_label = entity_type.replace('_', '').title()
                    labels.append(type_label)
                
                neo4j_entity['labels'] = labels
                
                # Flatten specified attributes
                for attr in entity_attributes:
                    if attr in attributes:
                        neo4j_entity[attr] = attributes[attr]
                
                # Flatten ALL remaining attributes as top-level properties
                remaining_attributes = {k: v for k, v in attributes.items() if k not in entity_attributes}
                flattened_remaining = self._flatten_metadata(remaining_attributes)
                for key, value in flattened_remaining.items():
                    neo4j_entity[key] = value
                    all_fields.add(key)
                
                # Track all fields
                all_fields.update(neo4j_entity.keys())
                neo4j_entities.append(neo4j_entity)
            
            # Build dynamic query with all fields
            set_clauses = [f'entity.{field} = ent.{field}' for field in sorted(all_fields) if field != 'labels']
            
            # For now, we'll use a simple approach without APOC dependency
            # We'll create separate queries for different entity types
            
            # Group entities by type for efficient querying
            entities_by_type = {}
            for entity in neo4j_entities:
                entity_type = entity.get('entity_type', 'unknown')
                if entity_type not in entities_by_type:
                    entities_by_type[entity_type] = []
                entities_by_type[entity_type].append(entity)
            
            total_count = 0
            for entity_type, type_entities in entities_by_type.items():
                # Create appropriate label based on entity type
                type_label = entity_type.replace('_', '').title() if entity_type != 'unknown' else 'Unknown'
                
                query = f"""
                UNWIND $entities AS ent
                MERGE (entity:Entity:{type_label} {{entity_id: ent.entity_id}})
                SET {', '.join(set_clauses) if set_clauses else 'entity.entity_id = ent.entity_id'}
                RETURN count(entity) as count
                """
                
                result = session.run(query, entities=type_entities)
                type_count = result.single()['count']
                total_count += type_count
                logger.debug(f"Created {type_count} {type_label} entities")
            
            count = total_count            
            logger.info(f"Exported {count} entities to Neo4j with multiple labels")
            return count
            
    def export_element_entity_mappings(self, mappings: List[Dict[str, Any]]) -> int:
        """
        Export element-to-entity mappings with DERIVED_FROM relationships.
        
        Args:
            mappings: List of element-entity mapping dictionaries
            
        Returns:
            Number of mappings created
        """
        with self.driver.session(database=self.config.database) as session:
            query = """
            UNWIND $mappings AS mapping
            MATCH (e:Element {element_id: mapping.element_id})
            MATCH (entity:Entity {entity_id: mapping.entity_id})
            MERGE (entity)-[r:DERIVED_FROM]->(e)
            SET r.extraction_rule = mapping.extraction_rule,
                r.confidence = mapping.confidence,
                r.extracted_at = mapping.extracted_at
            RETURN count(r) as count
            """
            
            # Prepare mappings for Neo4j
            neo4j_mappings = []
            for mapping in mappings:
                neo4j_mapping = {
                    'element_id': mapping.get('element_id'),
                    'entity_id': mapping.get('entity_id'),
                    'extraction_rule': mapping.get('extraction_rule', 'unknown'),
                    'confidence': mapping.get('confidence', 1.0),
                    'extracted_at': mapping.get('extracted_at', datetime.now().isoformat())
                }
                neo4j_mappings.append(neo4j_mapping)
                
            result = session.run(query, mappings=neo4j_mappings)
            count = result.single()['count']
            logger.info(f"Exported {count} element-entity DERIVED_FROM mappings to Neo4j")
            return count
            
    def export_entity_relationships(self, relationships: List[Dict[str, Any]]) -> int:
        """
        Export entity-to-entity relationships.
        
        Args:
            relationships: List of entity relationship dictionaries
            
        Returns:
            Number of relationships created
        """
        with self.driver.session(database=self.config.database) as session:
            # Group relationships by type to create with proper labels
            relationships_by_type = {}
            for rel in relationships:
                rel_type = rel.get('relationship_type', 'RELATED_TO')
                if rel_type not in relationships_by_type:
                    relationships_by_type[rel_type] = []
                relationships_by_type[rel_type].append(rel)
            
            total_count = 0
            for rel_type, rels in relationships_by_type.items():
                # Use the actual relationship type as the Neo4j relationship label
                # Cypher doesn't allow parameterized relationship types, so we build the query dynamically
                query = f"""
                UNWIND $relationships AS rel
                MATCH (source:Entity {{entity_id: rel.source_entity_id}})
                MATCH (target:Entity {{entity_id: rel.target_entity_id}})
                MERGE (source)-[r:{rel_type}]->(target)
                SET r.confidence = rel.confidence,
                    r.rule_id = rel.rule_id,
                    r.discovered_at = rel.discovered_at,
                    r.metadata = rel.metadata
                RETURN count(r) as count
                """
                
                # Prepare relationships for Neo4j
                neo4j_rels = []
                for rel in rels:
                    neo4j_rel = {
                        'source_entity_id': rel.get('source_entity_id'),
                        'target_entity_id': rel.get('target_entity_id'),
                        'confidence': rel.get('confidence', 0.0),
                        'rule_id': rel.get('rule_id', ''),
                        'discovered_at': rel.get('discovered_at', datetime.now().isoformat()),
                        'metadata': json.dumps(rel.get('metadata', {}))
                    }
                    neo4j_rels.append(neo4j_rel)
                    
                result = session.run(query, relationships=neo4j_rels)
                count = result.single()['count']
                total_count += count
                logger.info(f"Exported {count} {rel_type} relationships to Neo4j")
            
            logger.info(f"Exported {total_count} total entity relationships to Neo4j")
            return total_count
            
    def export_element_term_mappings(self, mappings: List[Dict[str, Any]]) -> int:
        """
        Export element-to-term mappings.
        
        Args:
            mappings: List of mapping dictionaries
            
        Returns:
            Number of mappings created
        """
        with self.driver.session(database=self.config.database) as session:
            query = """
            UNWIND $mappings AS mapping
            MATCH (e:Element {element_id: mapping.element_id})
            MATCH (t:Term {term_id: mapping.term_id})
            MERGE (e)-[r:MAPPED_TO]->(t)
            SET r.confidence = mapping.confidence,
                r.rule_type = mapping.rule_type,
                r.extracted_at = mapping.extracted_at
            RETURN count(r) as count
            """
            
            # Prepare mappings for Neo4j
            neo4j_mappings = []
            for mapping in mappings:
                neo4j_mapping = {
                    'element_id': mapping.get('element_id'),
                    'term_id': mapping.get('term_id'),
                    'confidence': mapping.get('confidence', 0.0),
                    'rule_type': mapping.get('rule_type', 'unknown'),
                    'extracted_at': mapping.get('extracted_at', datetime.now().isoformat())
                }
                neo4j_mappings.append(neo4j_mapping)
                
            result = session.run(query, mappings=neo4j_mappings)
            count = result.single()['count']
            logger.info(f"Exported {count} element-term mappings to Neo4j")
            return count
            
    def export_domain_relationships(self, relationships: List[Dict[str, Any]]) -> int:
        """
        Export domain relationships between elements.
        
        Args:
            relationships: List of relationship dictionaries
            
        Returns:
            Number of relationships created
        """
        with self.driver.session(database=self.config.database) as session:
            query = """
            UNWIND $relationships AS rel
            MATCH (source:Element {element_id: rel.source_id})
            MATCH (target:Element {element_id: rel.target_id})
            MERGE (source)-[r:DOMAIN_RELATIONSHIP]->(target)
            SET r.relationship_type = rel.relationship_type,
                r.confidence = rel.confidence,
                r.rule_id = rel.rule_id,
                r.discovered_at = rel.discovered_at,
                r.metadata = rel.metadata
            RETURN count(r) as count
            """
            
            # Prepare relationships for Neo4j
            neo4j_rels = []
            for rel in relationships:
                neo4j_rel = {
                    'source_id': rel.get('source_id'),
                    'target_id': rel.get('target_reference'),  # Note: field name difference
                    'relationship_type': rel.get('relationship_type'),
                    'confidence': rel.get('confidence', 0.0),
                    'rule_id': rel.get('rule_id', ''),
                    'discovered_at': rel.get('discovered_at', datetime.now().isoformat()),
                    'metadata': json.dumps(rel.get('metadata', {}))
                }
                neo4j_rels.append(neo4j_rel)
                
            result = session.run(query, relationships=neo4j_rels)
            count = result.single()['count']
            logger.info(f"Exported {count} domain relationships to Neo4j")
            return count
            
    def create_term_hierarchy(self, ontology: Dict[str, Any]) -> int:
        """
        Create hierarchical relationships between terms based on ontology.
        
        Args:
            ontology: Ontology dictionary with relationship rules
            
        Returns:
            Number of term relationships created
        """
        with self.driver.session(database=self.config.database) as session:
            # Extract term relationships from ontology rules
            term_relationships = []
            domain = ontology.get('domain', {}).get('name', 'unknown')
            
            for rule in ontology.get('relationship_rules', []):
                # Create relationships between terms themselves
                # Handle both dict and RelationshipEndpoint objects
                if hasattr(rule, 'source'):  # RelationshipRule object
                    source_term = f"{domain}:{rule.source.term_id}"
                    target_term = f"{domain}:{rule.target.term_id}"
                    rel_type = rule.relationship_type.upper()
                    rule_id = rule.id
                else:  # Dictionary
                    source_term = f"{domain}:{rule['source']['term_id']}"
                    target_term = f"{domain}:{rule['target']['term_id']}"
                    rel_type = rule['relationship_type'].upper()
                    rule_id = rule['id']
                
                term_relationships.append({
                    'source': source_term,
                    'target': target_term,
                    'type': rel_type,
                    'rule_id': rule_id
                })
                
            query = """
            UNWIND $relationships AS rel
            MATCH (source:Term {term_id: rel.source})
            MATCH (target:Term {term_id: rel.target})
            MERGE (source)-[r:TERM_RELATIONSHIP {type: rel.type}]->(target)
            SET r.rule_id = rel.rule_id
            RETURN count(r) as count
            """
            
            result = session.run(query, relationships=term_relationships)
            count = result.single()['count']
            logger.info(f"Created {count} term relationships in ontology hierarchy")
            return count
            
    def get_graph_statistics(self) -> Dict[str, Any]:
        """
        Get statistics about the graph.
        
        Returns:
            Dictionary with graph statistics
        """
        with self.driver.session(database=self.config.database) as session:
            stats = {}
            
            # Node counts
            node_queries = {
                'documents': "MATCH (d:Document) RETURN count(d) as count",
                'elements': "MATCH (e:Element) RETURN count(e) as count",
                'terms': "MATCH (t:Term) RETURN count(t) as count",
                'entities': "MATCH (entity:Entity) RETURN count(entity) as count",
            }
            
            for key, query in node_queries.items():
                result = session.run(query)
                stats[key] = result.single()['count']
                
            # Relationship counts
            rel_queries = {
                'element_to_doc': "MATCH ()-[r:BELONGS_TO]->() RETURN count(r) as count",
                'element_to_term': "MATCH ()-[r:MAPPED_TO]->() RETURN count(r) as count",
                'domain_relationships': "MATCH ()-[r:DOMAIN_RELATIONSHIP]->() RETURN count(r) as count",
                'term_relationships': "MATCH ()-[r:TERM_RELATIONSHIP]->() RETURN count(r) as count",
                'entity_derived_from': "MATCH ()-[r:DERIVED_FROM]->() RETURN count(r) as count",
                'entity_relationships': "MATCH ()-[r:ENTITY_RELATIONSHIP]->() RETURN count(r) as count",
            }
            
            for key, query in rel_queries.items():
                result = session.run(query)
                stats[key] = result.single()['count']
                
            # Additional metrics
            # Average mappings per element
            result = session.run("""
                MATCH (e:Element)
                OPTIONAL MATCH (e)-[r:MAPPED_TO]->()
                WITH e, count(r) as mappings
                RETURN avg(mappings) as avg_mappings
            """)
            stats['avg_mappings_per_element'] = result.single()['avg_mappings']
            
            # Most common terms
            result = session.run("""
                MATCH (t:Term)<-[r:MAPPED_TO]-()
                WITH t, count(r) as usage_count
                RETURN t.label as term, usage_count
                ORDER BY usage_count DESC
                LIMIT 10
            """)
            stats['top_terms'] = [(r['term'], r['usage_count']) for r in result]
            
            # Connected components
            result = session.run("""
                CALL gds.graph.exists('temp_graph')
                YIELD exists
                WITH exists
                CALL apoc.when(
                    exists,
                    'CALL gds.graph.drop("temp_graph") YIELD graphName RETURN graphName',
                    'RETURN null as graphName',
                    {exists: exists}
                ) YIELD value
                WITH value
                CALL gds.graph.project(
                    'temp_graph',
                    ['Document', 'Element', 'Term', 'Entity'],
                    ['BELONGS_TO', 'MAPPED_TO', 'DOMAIN_RELATIONSHIP', 'DERIVED_FROM', 'ENTITY_RELATIONSHIP']
                )
                YIELD nodeCount, relationshipCount
                RETURN nodeCount, relationshipCount
            """)
            graph_info = result.single()
            if graph_info:
                stats['total_nodes'] = graph_info['nodeCount']
                stats['total_relationships'] = graph_info['relationshipCount']
                
            return stats
            
    def find_paths_between_terms(self, term1: str, term2: str, max_hops: int = 5) -> List[Dict[str, Any]]:
        """
        Find paths between two terms through elements and relationships.
        
        Args:
            term1: First term ID
            term2: Second term ID
            max_hops: Maximum number of hops in path
            
        Returns:
            List of paths
        """
        with self.driver.session(database=self.config.database) as session:
            query = f"""
            MATCH path = shortestPath(
                (t1:Term {{term_id: $term1}})-[*..{max_hops}]-(t2:Term {{term_id: $term2}})
            )
            RETURN path
            LIMIT 10
            """
            
            result = session.run(query, term1=term1, term2=term2)
            paths = []
            
            for record in result:
                path = record['path']
                path_info = {
                    'nodes': [],
                    'relationships': [],
                    'length': len(path.relationships)
                }
                
                for node in path.nodes:
                    node_info = dict(node)
                    node_info['labels'] = list(node.labels)
                    path_info['nodes'].append(node_info)
                    
                for rel in path.relationships:
                    rel_info = dict(rel)
                    rel_info['type'] = rel.type
                    path_info['relationships'].append(rel_info)
                    
                paths.append(path_info)
                
            return paths
            
    def export_to_graphml(self, filepath: str):
        """
        Export the graph to GraphML format for visualization in other tools.
        
        Args:
            filepath: Path to save the GraphML file
        """
        with self.driver.session(database=self.config.database) as session:
            query = """
            CALL apoc.export.graphml.all($filepath, {
                useTypes: true,
                storeNodeIds: true
            })
            YIELD file, nodes, relationships
            RETURN file, nodes, relationships
            """
            
            result = session.run(query, filepath=filepath)
            record = result.single()
            
            if record:
                logger.info(f"Exported {record['nodes']} nodes and {record['relationships']} "
                          f"relationships to {record['file']}")
                return record
            return None