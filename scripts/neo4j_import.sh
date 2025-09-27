#!/bin/bash
# Domain-agnostic Neo4j import script for Go-Doc-Go extracted data

set -e

# Configuration
NEO4J_URL="${NEO4J_URL:-bolt://localhost:7687}"
NEO4J_USER="${NEO4J_USER:-neo4j}"
NEO4J_PASSWORD="${NEO4J_PASSWORD:-godocgo123}"
EXTRACTION_FILE="${1:-extraction_results.json}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Go-Doc-Go Neo4j Import Script${NC}"
echo "=================================="

# Check if extraction file exists
if [ ! -f "$EXTRACTION_FILE" ]; then
    echo -e "${RED}❌ Extraction file not found: $EXTRACTION_FILE${NC}"
    echo "Please run entity extraction first."
    exit 1
fi

echo -e "${YELLOW}📊 Loading extraction results...${NC}"

# Create Python script for import
cat > /tmp/neo4j_import.py << 'EOF'
import json
import sys
from neo4j import GraphDatabase
import os

# Configuration from environment
neo4j_url = os.getenv("NEO4J_URL", "bolt://localhost:7687")
neo4j_user = os.getenv("NEO4J_USER", "neo4j")
neo4j_password = os.getenv("NEO4J_PASSWORD", "godocgo123")
extraction_file = sys.argv[1] if len(sys.argv) > 1 else "extraction_results.json"

# Load extraction results
with open(extraction_file, 'r') as f:
    data = json.load(f)

# Connect to Neo4j
driver = GraphDatabase.driver(neo4j_url, auth=(neo4j_user, neo4j_password))

def import_entities(tx, entities):
    """Import entities as nodes."""
    for entity in entities:
        # Create generic entity node with dynamic labels
        entity_type = entity.get('entity_type', 'Entity')
        properties = {
            'id': entity['entity_id'],
            'text': entity['text'],
            'normalized': entity.get('normalized_text', entity['text']),
            'confidence': entity.get('confidence', 1.0)
        }

        # Add any additional metadata
        for key, value in entity.get('metadata', {}).items():
            if value is not None and key not in properties:
                properties[key] = value

        # Create node with dynamic label
        query = f"""
        MERGE (e:{entity_type} {{id: $id}})
        SET e += $properties
        """

        tx.run(query, id=properties['id'], properties=properties)

    return len(entities)

def import_relationships(tx, relationships):
    """Import relationships between entities."""
    for rel in relationships:
        # Create generic relationship
        rel_type = rel.get('relationship_type', 'RELATES_TO').replace(' ', '_').upper()

        query = f"""
        MATCH (source {{id: $source_id}})
        MATCH (target {{id: $target_id}})
        MERGE (source)-[r:{rel_type}]->(target)
        SET r.confidence = $confidence
        """

        tx.run(query,
               source_id=rel['source_id'],
               target_id=rel['target_id'],
               confidence=rel.get('confidence', 1.0))

    return len(relationships)

def create_indexes(tx):
    """Create indexes for better query performance."""
    # Create index on entity id (generic)
    tx.run("CREATE INDEX IF NOT EXISTS FOR (n:Entity) ON (n.id)")
    tx.run("CREATE INDEX IF NOT EXISTS FOR (n:Entity) ON (n.text)")

    # Create fulltext index for searching
    try:
        tx.run("""
        CREATE FULLTEXT INDEX entity_text IF NOT EXISTS
        FOR (n:Entity) ON EACH [n.text, n.normalized]
        """)
    except:
        pass  # May not be supported in all versions

# Import data
with driver.session() as session:
    # Clear existing data (optional - comment out to append)
    print("🧹 Clearing existing data...")
    session.run("MATCH (n) DETACH DELETE n")

    # Create indexes
    print("📇 Creating indexes...")
    session.execute_write(create_indexes)

    # Import entities
    if 'extraction_results' in data:
        entities = data['extraction_results'].get('entities', [])
    else:
        entities = data.get('entities', [])

    if entities:
        print(f"📌 Importing {len(entities)} entities...")
        count = session.execute_write(import_entities, entities)
        print(f"✅ Imported {count} entities")
    else:
        print("⚠️  No entities found in extraction results")

    # Import relationships
    if 'extraction_results' in data:
        relationships = data['extraction_results'].get('relationships', [])
    else:
        relationships = data.get('relationships', [])

    if relationships:
        print(f"🔗 Importing {len(relationships)} relationships...")
        count = session.execute_write(import_relationships, relationships)
        print(f"✅ Imported {count} relationships")
    else:
        print("⚠️  No relationships found in extraction results")

# Generate statistics
with driver.session() as session:
    # Count nodes by label
    result = session.run("""
    CALL db.labels() YIELD label
    CALL apoc.cypher.run('MATCH (n:' + label + ') RETURN count(n) as count', {})
    YIELD value
    RETURN label, value.count as count
    ORDER BY value.count DESC
    """)

    print("\n📊 Import Statistics:")
    print("-" * 40)
    for record in result:
        if record['count'] > 0:
            print(f"  {record['label']}: {record['count']} nodes")

    # Count relationships
    result = session.run("""
    MATCH ()-[r]->()
    RETURN type(r) as type, count(r) as count
    ORDER BY count DESC
    """)

    print("\n🔗 Relationship Types:")
    for record in result:
        print(f"  {record['type']}: {record['count']} relationships")

driver.close()
print("\n✨ Import completed successfully!")
EOF

# Run the import script
echo -e "${YELLOW}🔄 Importing to Neo4j...${NC}"
python3 /tmp/neo4j_import.py "$EXTRACTION_FILE"

# Clean up
rm -f /tmp/neo4j_import.py

echo -e "${GREEN}✅ Neo4j import completed!${NC}"
echo ""
echo "Access your graph at:"
echo "  - Neo4j Browser: http://localhost:7474"
echo "  - Bloom: http://localhost:7474/bloom"
echo "  - Credentials: neo4j / godocgo123"