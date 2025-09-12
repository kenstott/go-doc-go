// Neo4j initialization script for testing
// This file can be imported manually or via auto-import

// Create constraints for unique IDs
CREATE CONSTRAINT doc_id_unique IF NOT EXISTS FOR (d:Document) REQUIRE d.doc_id IS UNIQUE;
CREATE CONSTRAINT element_id_unique IF NOT EXISTS FOR (e:Element) REQUIRE e.element_id IS UNIQUE;
CREATE CONSTRAINT entity_id_unique IF NOT EXISTS FOR (n:Entity) REQUIRE n.entity_id IS UNIQUE;

// Create indexes for common queries
CREATE INDEX doc_type_index IF NOT EXISTS FOR (d:Document) ON (d.doc_type);
CREATE INDEX element_type_index IF NOT EXISTS FOR (e:Element) ON (e.element_type);
CREATE INDEX entity_type_index IF NOT EXISTS FOR (n:Entity) ON (n.entity_type);
CREATE INDEX entity_name_index IF NOT EXISTS FOR (n:Entity) ON (n.name);

// Create full-text search indexes
CREATE FULLTEXT INDEX document_search IF NOT EXISTS FOR (d:Document) ON EACH [d.content, d.source];
CREATE FULLTEXT INDEX element_search IF NOT EXISTS FOR (e:Element) ON EACH [e.content_preview];

// Create sample test data
// Documents
CREATE (d1:Document {
    doc_id: 'test-doc-1',
    doc_type: 'pdf',
    source: 'test.pdf',
    created_at: datetime(),
    pages: 10
});

CREATE (d2:Document {
    doc_id: 'test-doc-2',
    doc_type: 'docx',
    source: 'test.docx',
    created_at: datetime(),
    author: 'Test User'
});

// Elements
CREATE (e1:Element {
    element_id: 'elem-1',
    element_type: 'paragraph',
    content_preview: 'This is a test paragraph',
    document_position: 1
});

CREATE (e2:Element {
    element_id: 'elem-2',
    element_type: 'heading',
    content_preview: 'Test Heading',
    document_position: 0,
    level: 1
});

// Entities
CREATE (entity1:Entity {
    entity_id: 'entity-1',
    entity_type: 'company',
    name: 'Test Corporation',
    ticker: 'TEST'
});

CREATE (entity2:Entity {
    entity_id: 'entity-2',
    entity_type: 'person',
    name: 'John Doe',
    role: 'CEO'
});

// Create relationships
MATCH (d1:Document {doc_id: 'test-doc-1'})
MATCH (e1:Element {element_id: 'elem-1'})
CREATE (d1)-[:CONTAINS]->(e1);

MATCH (d1:Document {doc_id: 'test-doc-1'})
MATCH (e2:Element {element_id: 'elem-2'})
CREATE (d1)-[:CONTAINS]->(e2);

MATCH (e1:Element {element_id: 'elem-1'})
MATCH (entity1:Entity {entity_id: 'entity-1'})
CREATE (e1)-[:MENTIONS]->(entity1);

MATCH (e2:Element {element_id: 'elem-2'})
MATCH (entity2:Entity {entity_id: 'entity-2'})
CREATE (e2)-[:MENTIONS]->(entity2);

// Create a semantic similarity relationship
MATCH (e1:Element {element_id: 'elem-1'})
MATCH (e2:Element {element_id: 'elem-2'})
CREATE (e1)-[:SIMILAR_TO {score: 0.85}]->(e2);

// Return summary
MATCH (d:Document) RETURN count(d) as document_count;
MATCH (e:Element) RETURN count(e) as element_count;
MATCH (n:Entity) RETURN count(n) as entity_count;
MATCH ()-[r]->() RETURN count(r) as relationship_count;