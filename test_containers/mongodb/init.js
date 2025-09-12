// MongoDB initialization script for testing
// This runs automatically when the container starts

// Switch to the test database
use go_doc_go_test;

// Create test user with full permissions on test database
db.createUser({
  user: "testuser",
  pwd: "testpass",
  roles: [
    { role: "readWrite", db: "go_doc_go_test" },
    { role: "dbAdmin", db: "go_doc_go_test" }
  ]
});

// Create collections with indexes
db.createCollection("documents");
db.documents.createIndex({ "doc_id": 1 }, { unique: true });
db.documents.createIndex({ "doc_type": 1 });
db.documents.createIndex({ "created_at": -1 });
db.documents.createIndex({ "metadata.source": 1 });

db.createCollection("elements");
db.elements.createIndex({ "element_id": 1 }, { unique: true });
db.elements.createIndex({ "doc_id": 1 });
db.elements.createIndex({ "element_type": 1 });
db.elements.createIndex({ "parent_id": 1 });
db.elements.createIndex({ "$**": "text" }); // Text index for search

db.createCollection("relationships");
db.relationships.createIndex({ "source_id": 1, "target_id": 1, "relationship_type": 1 }, { unique: true });
db.relationships.createIndex({ "source_id": 1 });
db.relationships.createIndex({ "target_id": 1 });

db.createCollection("entities");
db.entities.createIndex({ "entity_id": 1 }, { unique: true });
db.entities.createIndex({ "entity_type": 1 });
db.entities.createIndex({ "name": "text" });

// Create collections for content sources
db.createCollection("mongodb_documents");
db.createCollection("test_collection");

// Insert sample test data
db.documents.insertMany([
  {
    _id: ObjectId(),
    doc_id: "test-doc-1",
    doc_type: "pdf",
    source: "test.pdf",
    metadata: {
      pages: 10,
      author: "Test Author"
    },
    created_at: new Date(),
    updated_at: new Date()
  },
  {
    _id: ObjectId(),
    doc_id: "test-doc-2",
    doc_type: "docx",
    source: "test.docx",
    metadata: {
      words: 5000,
      language: "en"
    },
    created_at: new Date(),
    updated_at: new Date()
  }
]);

db.elements.insertMany([
  {
    _id: ObjectId(),
    element_id: "elem-1",
    doc_id: "test-doc-1",
    element_type: "paragraph",
    content_preview: "This is a test paragraph",
    document_position: 1,
    metadata: {
      page: 1
    }
  },
  {
    _id: ObjectId(),
    element_id: "elem-2",
    doc_id: "test-doc-1",
    element_type: "heading",
    content_preview: "Test Heading",
    document_position: 0,
    metadata: {
      level: 1
    }
  }
]);

// Sample MongoDB source documents for content source testing
db.mongodb_documents.insertMany([
  {
    _id: ObjectId(),
    title: "MongoDB Test Document 1",
    content: "This is test content from MongoDB",
    type: "article",
    tags: ["test", "mongodb"],
    created_at: new Date()
  },
  {
    _id: ObjectId(),
    title: "MongoDB Test Document 2",
    content: "Another test document",
    type: "report",
    tags: ["sample", "data"],
    created_at: new Date()
  }
]);

// Create a capped collection for logs (optional, for testing)
db.createCollection("test_logs", {
  capped: true,
  size: 1048576,  // 1MB
  max: 1000       // Max 1000 documents
});

print("MongoDB test database initialized successfully");
print("Collections created: documents, elements, relationships, entities, mongodb_documents, test_collection, test_logs");
print("Test user created: testuser/testpass");
print("Sample data inserted");