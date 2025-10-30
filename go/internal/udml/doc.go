// Package udml implements the Universal Document Markup Language specification.
//
// UDML (Universal Document Markup Language) is a layered data model for representing
// structured information extracted from documents. It provides a unified schema for
// elements, relationships, embeddings, and ontology instances across all document types.
//
// # UDML Layers
//
// **UDML-D (Documents)**:
//   - Document metadata (ID, title, source, processing timestamps)
//   - Document-level statistics
//
// **UDML-E (Elements)**:
//   - Universal elements (headings, paragraphs, tables, code, etc.)
//   - Hierarchical structure via parent-child relationships
//   - Query-optimized promoted fields (page_number, section_level, etc.)
//   - Code-specific fields (function_name, class_name, line_number)
//
// **UDML-R (Relationships)**:
//   - Structural relationships (contains, part_of)
//   - Semantic relationships (similar_to, references, depends_on)
//   - Cross-document relationships
//
// **UDML-V (Vector Embeddings)**:
//   - Element embeddings for semantic search
//   - Contextual text used for embedding generation
//
// **UDML-L (Links)**:
//   - Hyperlinks and cross-references
//   - Code dependencies and imports
//
// **UDML-O (Ontology Instances)**:
//   - Extracted entities (people, organizations, locations, concepts)
//   - Entity relationships (works_at, located_in, mentions)
//   - Canonical entities (deduplicated from raw extractions)
//   - LLM-based validation results
//
// # Schema System
//
// The SchemaRegistry provides Arrow/Parquet schemas for UDML elements:
//
//	registry := udml.NewSchemaRegistry()
//	schema := registry.GetSchema("paragraph")
//
// All element types share a universal schema with promoted fields for fast queries.
//
// # Ontology Extraction
//
// The ontology subsystem extracts structured entities and relationships from documents
// using LLM-generated extraction schemas.
//
// **Workflow**:
//	1. Interview (LLM generates ontology schema from requirements)
//	2. Schema (EntityType definitions with extraction rules)
//	3. Extract (SQL-based extraction from UDML-E elements)
//	4. Store (Results in UDML-O layer)
//	5. Consolidate (Deduplicate entities into canonical form)
//
// See go/internal/udml/ontology/ for details.
//
// # Builder
//
// The builder subsystem provides LLM-based ontology schema generation.
//
// See go/internal/udml/builder/ for details.
//
// # Query
//
// The query subsystem provides corpus exploration and analysis tools.
//
// See go/internal/udml/query/ for details.
//
// # Sampler
//
// The sampler subsystem provides data sampling strategies for ontology schema generation.
//
// See go/internal/udml/sampler/ for details.
//
// # Versioning
//
// The versioning subsystem supports temporal queries and corpus evolution tracking.
//
// See go/internal/udml/versioning/ for details.
//
// # See Also
//
//   - UDML Specification: docs/features/udml/
//   - Ontology System: docs/features/ontology/
//   - Analytics Storage: go/internal/analytics/ (UDML persistence)
//   - Parser: go/internal/parser/ (UDML element generation)
package udml
