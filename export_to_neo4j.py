#!/usr/bin/env python3
"""
Manual export of extracted ontology entities and relationships to Neo4j.
This bypasses the incomplete Neo4j storage adapter in Go-Doc-Go.
"""

import json
import sys
from typing import Dict, List, Any

# Import neo4j driver
try:
    from neo4j import GraphDatabase
except ImportError:
    print("❌ Neo4j driver not installed. Run: pip install neo4j")
    sys.exit(1)


class Neo4jOntologyExporter:
    """Export extracted entities and relationships to Neo4j."""

    def __init__(self, uri: str, username: str, password: str, database: str = "neo4j"):
        self.uri = uri
        self.username = username
        self.password = password
        self.database = database
        self.driver = None

    def connect(self):
        """Connect to Neo4j database."""
        try:
            self.driver = GraphDatabase.driver(
                self.uri,
                auth=(self.username, self.password)
            )
            # Test connection
            with self.driver.session(database=self.database) as session:
                result = session.run("RETURN 1 as test")
                test_value = result.single()["test"]
                if test_value == 1:
                    print("✅ Connected to Neo4j successfully")
                    return True
        except Exception as e:
            print(f"❌ Failed to connect to Neo4j: {e}")
            return False

    def clear_database(self):
        """Clear all nodes and relationships (for clean demo)."""
        with self.driver.session(database=self.database) as session:
            session.run("MATCH (n) DETACH DELETE n")
            print("🧹 Cleared existing data")

    def create_constraints(self):
        """Create constraints and indexes for better performance."""
        constraints = [
            "CREATE CONSTRAINT entity_id IF NOT EXISTS FOR (e:Entity) REQUIRE e.entity_id IS UNIQUE",
            "CREATE CONSTRAINT document_id IF NOT EXISTS FOR (d:Document) REQUIRE d.doc_id IS UNIQUE",
        ]

        with self.driver.session(database=self.database) as session:
            for constraint in constraints:
                try:
                    session.run(constraint)
                except Exception as e:
                    # Constraint might already exist
                    pass
        print("📋 Created constraints and indexes")

    def export_entities(self, entities: List[Dict[str, Any]]) -> int:
        """Export entities as nodes to Neo4j."""

        entity_query = """
        MERGE (e:Entity {entity_id: $entity_id})
        SET e.entity_type = $entity_type,
            e.term_id = $term_id,
            e.content = $content,
            e.confidence = $confidence,
            e.source_element_id = $source_element_id,
            e.doc_id = $doc_id,
            e.source_name = $source_name,
            e.extracted_at = $extracted_at,
            e.element_type = $element_type,
            e.extraction_rule = $extraction_rule

        // Create document node if it doesn't exist
        MERGE (d:Document {doc_id: $doc_id})
        SET d.source_name = $source_name

        // Create relationship to document
        MERGE (e)-[:EXTRACTED_FROM]->(d)
        """

        count = 0
        with self.driver.session(database=self.database) as session:
            for entity in entities:
                # Create a valid Neo4j label from entity type
                entity_type_label = entity['entity_type'].title().replace('_', '')

                session.run(entity_query, {
                    'entity_id': entity['entity_id'],
                    'entity_type': entity['entity_type'],
                    'entity_type_label': entity_type_label,
                    'term_id': entity['term_id'],
                    'content': entity['content'][:500],  # Limit content length
                    'confidence': entity['confidence'],
                    'source_element_id': entity['source_element_id'],
                    'doc_id': entity['doc_id'],
                    'source_name': entity['source_name'],
                    'extracted_at': entity['extracted_at'],
                    'element_type': entity.get('element_type'),
                    'extraction_rule': entity['metadata'].get('extraction_method')
                })
                count += 1

        print(f"📊 Exported {count} entities as nodes")
        return count

    def export_relationships(self, relationships: List[Dict[str, Any]]) -> int:
        """Export relationships as edges to Neo4j."""

        relationship_query = """
        MATCH (source:Entity {entity_id: $source_entity_id})
        MATCH (target:Entity {entity_id: $target_entity_id})
        MERGE (source)-[r:RELATED {relationship_id: $relationship_id}]->(target)
        SET r.relationship_type = $relationship_type,
            r.confidence = $confidence,
            r.doc_id = $doc_id,
            r.source_name = $source_name,
            r.extracted_at = $extracted_at,
            r.extraction_method = $extraction_method
        """

        count = 0
        with self.driver.session(database=self.database) as session:
            for relationship in relationships:
                session.run(relationship_query, {
                    'relationship_id': relationship['relationship_id'],
                    'source_entity_id': relationship['source_entity_id'],
                    'target_entity_id': relationship['target_entity_id'],
                    'relationship_type': relationship['relationship_type'],
                    'confidence': relationship['confidence'],
                    'doc_id': relationship['doc_id'],
                    'source_name': relationship['source_name'],
                    'extracted_at': relationship['extracted_at'],
                    'extraction_method': relationship['metadata'].get('extraction_method')
                })
                count += 1

        print(f"🔗 Exported {count} relationships as edges")
        return count

    def get_summary(self) -> Dict[str, int]:
        """Get summary of exported data."""
        with self.driver.session(database=self.database) as session:
            # Count nodes by type
            result = session.run("""
                MATCH (n)
                RETURN labels(n) as labels, count(n) as count
                ORDER BY count DESC
            """)

            node_counts = {}
            for record in result:
                labels = record["labels"]
                count = record["count"]
                # Use the most specific label (not 'Entity')
                main_label = [l for l in labels if l != 'Entity'][0] if len(labels) > 1 else labels[0]
                node_counts[main_label] = count

            # Count relationships
            rel_result = session.run("MATCH ()-[r]->() RETURN count(r) as rel_count")
            rel_count = rel_result.single()["rel_count"]

            return {
                'nodes': node_counts,
                'relationships': rel_count
            }

    def close(self):
        """Close database connection."""
        if self.driver:
            self.driver.close()
            print("🔌 Closed Neo4j connection")


def main():
    """Main function to export extraction results to Neo4j."""

    # Configuration
    NEO4J_URI = "bolt://localhost:7687"
    NEO4J_USERNAME = "neo4j"
    NEO4J_PASSWORD = "password"
    NEO4J_DATABASE = "neo4j"

    # Load extraction results
    results_file = "simple_semantic_extraction_results.json"

    try:
        with open(results_file, 'r') as f:
            data = json.load(f)
    except FileNotFoundError:
        print(f"❌ Results file not found: {results_file}")
        sys.exit(1)

    extraction_results = data['extraction_results']
    entities = extraction_results['entities']
    relationships = extraction_results['relationships']

    print("🚀 Neo4j Ontology Export")
    print("=" * 50)
    print(f"📁 Loading: {results_file}")
    print(f"📊 Found: {len(entities)} entities, {len(relationships)} relationships")
    print()

    # Initialize exporter
    exporter = Neo4jOntologyExporter(NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD, NEO4J_DATABASE)

    if not exporter.connect():
        sys.exit(1)

    try:
        # Prepare database
        exporter.clear_database()
        exporter.create_constraints()

        # Export data
        entity_count = exporter.export_entities(entities)
        relationship_count = exporter.export_relationships(relationships)

        # Show summary
        print("\n📈 EXPORT SUMMARY")
        print("-" * 30)
        summary = exporter.get_summary()

        print("Node counts by type:")
        for node_type, count in summary['nodes'].items():
            print(f"  {node_type}: {count}")

        print(f"Relationships: {summary['relationships']}")

        print("\n✅ Export completed successfully!")
        print("\n🌐 Access Neo4j Browser at: http://localhost:7474")
        print("   Username: neo4j")
        print("   Password: password")

        print("\n📋 Sample Queries:")
        print("   MATCH (n) RETURN count(n)  // Total nodes")
        print("   MATCH (c:Company) RETURN c.content LIMIT 10  // Company entities")
        print("   MATCH (r:Revenue) RETURN r.content LIMIT 10  // Revenue entities")
        print("   MATCH ()-[rel]->() RETURN type(rel), count(rel)  // Relationships by type")
        print("   MATCH (c:Company)-[r]->(rev:Revenue) RETURN c, r, rev LIMIT 5  // Company-Revenue links")

    finally:
        exporter.close()


if __name__ == "__main__":
    main()