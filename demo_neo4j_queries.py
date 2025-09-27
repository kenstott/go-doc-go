#!/usr/bin/env python3
"""
Demonstrate Neo4j queries on the extracted SEC insider trading ontology data.
"""

import sys
from typing import List, Dict, Any

try:
    from neo4j import GraphDatabase
except ImportError:
    print("❌ Neo4j driver not installed. Run: pip install neo4j")
    sys.exit(1)


class Neo4jQueryDemo:
    """Demonstrate queries on extracted ontology data."""

    def __init__(self, uri: str, username: str, password: str, database: str = "neo4j"):
        self.driver = GraphDatabase.driver(uri, auth=(username, password))
        self.database = database

    def run_query(self, query: str, description: str) -> List[Dict[str, Any]]:
        """Run a query and return results."""
        print(f"\n🔍 {description}")
        print("-" * 50)
        print(f"Query: {query}")
        print()

        with self.driver.session(database=self.database) as session:
            result = session.run(query)
            records = [record.data() for record in result]

        print(f"Results ({len(records)} rows):")
        for i, record in enumerate(records[:10]):  # Limit to first 10
            print(f"  {i+1}. {record}")

        if len(records) > 10:
            print(f"  ... and {len(records) - 10} more")

        return records

    def demo_basic_queries(self):
        """Demonstrate basic graph queries."""
        print("🚀 Neo4j Ontology Query Demonstration")
        print("=" * 60)

        # 1. Total node count
        self.run_query(
            "MATCH (n) RETURN count(n) as total_nodes",
            "Total Nodes in Graph"
        )

        # 2. Node types
        self.run_query(
            """
            MATCH (n)
            RETURN labels(n) as node_types, count(n) as count
            ORDER BY count DESC
            """,
            "Node Types and Counts"
        )

        # 3. Entity types
        self.run_query(
            """
            MATCH (e:Entity)
            RETURN e.entity_type as entity_type, count(e) as count
            ORDER BY count DESC
            """,
            "Entity Types Extracted"
        )

        # 4. Documents with entities
        self.run_query(
            """
            MATCH (d:Document)<-[:EXTRACTED_FROM]-(e:Entity)
            RETURN d.doc_id as document, count(e) as entity_count
            ORDER BY entity_count DESC
            """,
            "Documents by Entity Count"
        )

    def demo_entity_queries(self):
        """Demonstrate entity-specific queries."""
        print("\n📊 ENTITY-SPECIFIC QUERIES")
        print("=" * 40)

        # 5. Company entities
        self.run_query(
            """
            MATCH (e:Entity)
            WHERE e.entity_type = 'company'
            RETURN e.content as company_mention, e.confidence as confidence
            ORDER BY e.confidence DESC
            LIMIT 10
            """,
            "Company Entity Mentions"
        )

        # 6. Revenue entities
        self.run_query(
            """
            MATCH (e:Entity)
            WHERE e.entity_type = 'revenue'
            RETURN e.content as revenue_mention, e.element_type as found_in
            LIMIT 10
            """,
            "Revenue Entity Mentions"
        )

        # 7. Executive entities
        self.run_query(
            """
            MATCH (e:Entity)
            WHERE e.entity_type = 'executive'
            RETURN e.content as executive_mention, e.extraction_rule as extracted_via
            LIMIT 10
            """,
            "Executive Entity Mentions"
        )

        # 8. Entities by extraction method
        self.run_query(
            """
            MATCH (e:Entity)
            RETURN e.extraction_rule as extraction_method, count(e) as count
            ORDER BY count DESC
            """,
            "Entities by Extraction Method"
        )

    def demo_document_queries(self):
        """Demonstrate document-centric queries."""
        print("\n📄 DOCUMENT-CENTRIC QUERIES")
        print("=" * 40)

        # 9. Document entity breakdown
        self.run_query(
            """
            MATCH (d:Document)<-[:EXTRACTED_FROM]-(e:Entity)
            WITH d, e.entity_type as entity_type, count(e) as entity_count
            RETURN d.doc_id as document,
                   collect([entity_type, entity_count]) as entity_breakdown
            ORDER BY size(entity_breakdown) DESC
            """,
            "Entity Breakdown by Document"
        )

        # 10. Cross-document entity analysis
        self.run_query(
            """
            MATCH (e:Entity)
            WITH e.entity_type as entity_type, e.content as content, count(DISTINCT e.doc_id) as doc_count
            WHERE doc_count > 1
            RETURN entity_type, content, doc_count
            ORDER BY doc_count DESC, entity_type
            LIMIT 10
            """,
            "Entities Appearing in Multiple Documents"
        )

    def demo_advanced_queries(self):
        """Demonstrate advanced analytical queries."""
        print("\n🔬 ADVANCED ANALYTICAL QUERIES")
        print("=" * 40)

        # 11. Entity co-occurrence within documents
        self.run_query(
            """
            MATCH (e1:Entity)-[:EXTRACTED_FROM]->(d:Document)<-[:EXTRACTED_FROM]-(e2:Entity)
            WHERE e1.entity_type <> e2.entity_type
            WITH e1.entity_type as type1, e2.entity_type as type2, count(*) as cooccurrence
            WHERE cooccurrence > 1
            RETURN type1, type2, cooccurrence
            ORDER BY cooccurrence DESC
            """,
            "Entity Type Co-occurrence in Documents"
        )

        # 12. Most mentioned company entities
        self.run_query(
            """
            MATCH (e:Entity)
            WHERE e.entity_type = 'company'
            WITH e.content as company_text, count(e) as mention_count
            ORDER BY mention_count DESC
            LIMIT 10
            RETURN company_text, mention_count
            """,
            "Most Frequently Mentioned Companies"
        )

        # 13. Document complexity analysis
        self.run_query(
            """
            MATCH (d:Document)<-[:EXTRACTED_FROM]-(e:Entity)
            WITH d,
                 count(DISTINCT e.entity_type) as entity_types,
                 count(e) as total_entities,
                 avg(e.confidence) as avg_confidence
            RETURN d.doc_id as document,
                   entity_types,
                   total_entities,
                   round(avg_confidence, 3) as avg_confidence
            ORDER BY entity_types DESC, total_entities DESC
            """,
            "Document Complexity Analysis"
        )

    def demo_insider_trading_queries(self):
        """Demonstrate insider trading specific queries."""
        print("\n📈 INSIDER TRADING ANALYSIS")
        print("=" * 40)

        # 14. Potential insider trading indicators
        self.run_query(
            """
            MATCH (d:Document)<-[:EXTRACTED_FROM]-(company:Entity),
                  (d)<-[:EXTRACTED_FROM]-(executive:Entity)
            WHERE company.entity_type = 'company'
              AND executive.entity_type = 'executive'
            RETURN d.doc_id as document,
                   company.content as company_mention,
                   executive.content as executive_mention
            LIMIT 10
            """,
            "Company-Executive Co-mentions (Potential Insider Context)"
        )

        # 15. Financial data context
        self.run_query(
            """
            MATCH (d:Document)<-[:EXTRACTED_FROM]-(company:Entity),
                  (d)<-[:EXTRACTED_FROM]-(revenue:Entity)
            WHERE company.entity_type = 'company'
              AND revenue.entity_type = 'revenue'
            RETURN d.doc_id as document,
                   company.content as company_mention,
                   revenue.content as revenue_mention
            LIMIT 10
            """,
            "Company-Revenue Co-mentions (Financial Context)"
        )

        # 16. Complete insider trading context
        self.run_query(
            """
            MATCH (d:Document)<-[:EXTRACTED_FROM]-(e:Entity)
            WITH d,
                 collect(CASE WHEN e.entity_type = 'company' THEN e.content END) as companies,
                 collect(CASE WHEN e.entity_type = 'executive' THEN e.content END) as executives,
                 collect(CASE WHEN e.entity_type = 'revenue' THEN e.content END) as revenues
            WHERE size([x IN companies WHERE x IS NOT NULL]) > 0
              AND size([x IN executives WHERE x IS NOT NULL]) > 0
            RETURN d.doc_id as document,
                   [x IN companies WHERE x IS NOT NULL] as companies,
                   [x IN executives WHERE x IS NOT NULL] as executives,
                   [x IN revenues WHERE x IS NOT NULL] as revenues
            LIMIT 5
            """,
            "Complete Insider Trading Context (Company + Executive + Revenue)"
        )

    def close(self):
        """Close database connection."""
        if self.driver:
            self.driver.close()
            print("\n🔌 Closed Neo4j connection")


def main():
    """Main demonstration function."""
    # Configuration
    NEO4J_URI = "bolt://localhost:7687"
    NEO4J_USERNAME = "neo4j"
    NEO4J_PASSWORD = "password"
    NEO4J_DATABASE = "neo4j"

    demo = Neo4jQueryDemo(NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD, NEO4J_DATABASE)

    try:
        # Run all demonstrations
        demo.demo_basic_queries()
        demo.demo_entity_queries()
        demo.demo_document_queries()
        demo.demo_advanced_queries()
        demo.demo_insider_trading_queries()

        print("\n" + "=" * 60)
        print("✅ Neo4j Query Demonstration Complete!")
        print("=" * 60)
        print("\n🌐 Access Neo4j Browser for visual exploration:")
        print("   URL: http://localhost:7474")
        print("   Username: neo4j")
        print("   Password: password")
        print("\n💡 Try these additional queries in the Neo4j Browser:")
        print("   - MATCH (n) RETURN n LIMIT 25  // Visualize all nodes")
        print("   - MATCH p=()-->() RETURN p LIMIT 25  // Show relationships")
        print("   - CALL db.schema.visualization()  // Schema overview")

    finally:
        demo.close()


if __name__ == "__main__":
    main()