package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jStorage implements analytics storage using Neo4j graph database
type Neo4jStorage struct {
	driver   neo4j.DriverWithContext
	database string
	runID    string
	ctx      context.Context
}

// NewNeo4jStorage creates a new Neo4j analytics storage
func NewNeo4jStorage(config map[string]interface{}) (*Neo4jStorage, error) {
	// Extract configuration
	uri, ok := config["uri"].(string)
	if !ok {
		uri = "bolt://localhost:7687"
	}

	username, ok := config["username"].(string)
	if !ok {
		username = "neo4j"
	}

	password, ok := config["password"].(string)
	if !ok {
		password = "password"
	}

	database, ok := config["database"].(string)
	if !ok {
		database = "neo4j"
	}

	// Generate run ID
	runID := fmt.Sprintf("export_%s", time.Now().Format("20060102_150405"))

	// Create Neo4j driver
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	ctx := context.Background()

	storage := &Neo4jStorage{
		driver:   driver,
		database: database,
		runID:    runID,
		ctx:      ctx,
	}

	// Initialize (create indexes)
	if err := storage.initialize(); err != nil {
		driver.Close(ctx)
		return nil, err
	}

	log.Printf("Neo4j analytics storage initialized: %s", uri)
	return storage, nil
}

// initialize creates indexes for efficient queries
func (s *Neo4jStorage) initialize() error {
	session := s.driver.NewSession(s.ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(s.ctx)

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS FOR (d:Document) ON (d.doc_id)",
		"CREATE INDEX IF NOT EXISTS FOR (d:Document) ON (d.run_id)",
		"CREATE INDEX IF NOT EXISTS FOR (e:Element) ON (e.element_id)",
		"CREATE INDEX IF NOT EXISTS FOR (e:Element) ON (e.run_id)",
		"CREATE INDEX IF NOT EXISTS FOR (r:Run) ON (r.run_id)",
	}

	for _, indexQuery := range indexes {
		_, err := session.Run(s.ctx, indexQuery, nil)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// AppendDocuments appends documents as nodes to Neo4j
func (s *Neo4jStorage) AppendDocuments(documents []Document) error {
	if len(documents) == 0 {
		return nil
	}

	session := s.driver.NewSession(s.ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(s.ctx)

	for _, doc := range documents {
		// Create Document node
		_, err := session.Run(s.ctx, `
			MERGE (d:Document {doc_id: $doc_id, run_id: $run_id})
			SET d.source_name = $source_name,
				d.title = $title,
				d.url = $url,
				d.content_type = $content_type,
				d.processed_at = $processed_at,
				d.element_count = $element_count,
				d.relationship_count = $relationship_count,
				d.written_at = $written_at
			RETURN d
		`, map[string]interface{}{
			"doc_id":             doc.DocID,
			"run_id":             s.runID,
			"source_name":        doc.SourceName,
			"title":              doc.Title,
			"url":                doc.URL,
			"content_type":       doc.ContentType,
			"processed_at":       doc.ProcessedAt.Format(time.RFC3339),
			"element_count":      doc.ElementCount,
			"relationship_count": doc.RelationshipCount,
			"written_at":         time.Now().Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("failed to append document %s: %w", doc.DocID, err)
		}

		// Link to Run node
		_, err = session.Run(s.ctx, `
			MERGE (r:Run {run_id: $run_id})
			MERGE (d:Document {doc_id: $doc_id, run_id: $run_id})
			MERGE (r)-[:CONTAINS]->(d)
		`, map[string]interface{}{
			"run_id": s.runID,
			"doc_id": doc.DocID,
		})
		if err != nil {
			return fmt.Errorf("failed to link document %s to run: %w", doc.DocID, err)
		}
	}

	log.Printf("Appended %d documents to Neo4j", len(documents))
	return nil
}

// AppendElements appends elements as nodes to Neo4j
func (s *Neo4jStorage) AppendElements(elements []Element) error {
	if len(elements) == 0 {
		return nil
	}

	session := s.driver.NewSession(s.ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(s.ctx)

	for _, elem := range elements {
		// Convert metadata to JSON string
		metadataJSON, err := json.Marshal(elem.Metadata)
		if err != nil {
			log.Printf("Warning: failed to marshal metadata for element %s: %v", elem.ElementID, err)
			metadataJSON = []byte("{}")
		}

		temporalJSON, err := json.Marshal(elem.TemporalMetadata)
		if err != nil {
			log.Printf("Warning: failed to marshal temporal metadata for element %s: %v", elem.ElementID, err)
			temporalJSON = []byte("{}")
		}

		contentLocationJSON, err := json.Marshal(elem.ContentLocation)
		if err != nil {
			log.Printf("Warning: failed to marshal content location for element %s: %v", elem.ElementID, err)
			contentLocationJSON = []byte("{}")
		}

		// Create Element node
		_, err = session.Run(s.ctx, `
			MERGE (e:Element {element_id: $element_id, run_id: $run_id})
			SET e.doc_id = $doc_id,
				e.source_name = $source_name,
				e.element_type = $element_type,
				e.element_category = $element_category,
				e.content_preview = $content_preview,
				e.content_hash = $content_hash,
				e.parent_id = $parent_id,
				e.metadata_json = $metadata_json,
				e.temporal_metadata_json = $temporal_metadata_json,
				e.content_location_json = $content_location_json,
				e.element_order = $element_order,
				e.document_position = $document_position,
				e.written_at = $written_at
			RETURN e
		`, map[string]interface{}{
			"element_id":               elem.ElementID,
			"run_id":                   s.runID,
			"doc_id":                   elem.DocID,
			"source_name":              elem.SourceName,
			"element_type":             elem.ElementType,
			"element_category":         elem.ElementCategory,
			"content_preview":          elem.ContentPreview,
			"content_hash":             elem.ContentHash,
			"parent_id":                elem.ParentID,
			"metadata_json":            string(metadataJSON),
			"temporal_metadata_json":   string(temporalJSON),
			"content_location_json":    string(contentLocationJSON),
			"element_order":            elem.ElementOrder,
			"document_position":        elem.DocumentPosition,
			"written_at":               time.Now().Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("failed to append element %s: %w", elem.ElementID, err)
		}

		// Link to Document
		_, err = session.Run(s.ctx, `
			MERGE (d:Document {doc_id: $doc_id, run_id: $run_id})
			MERGE (e:Element {element_id: $element_id, run_id: $run_id})
			MERGE (d)-[:HAS_ELEMENT]->(e)
		`, map[string]interface{}{
			"doc_id":     elem.DocID,
			"element_id": elem.ElementID,
			"run_id":     s.runID,
		})
		if err != nil {
			return fmt.Errorf("failed to link element %s to document: %w", elem.ElementID, err)
		}

		// Create parent-child relationships
		if elem.ParentID != "" {
			_, err = session.Run(s.ctx, `
				MERGE (p:Element {element_id: $parent_id, run_id: $run_id})
				MERGE (c:Element {element_id: $element_id, run_id: $run_id})
				MERGE (p)-[:CONTAINS]->(c)
			`, map[string]interface{}{
				"parent_id":  elem.ParentID,
				"element_id": elem.ElementID,
				"run_id":     s.runID,
			})
			if err != nil {
				return fmt.Errorf("failed to create parent-child relationship for element %s: %w", elem.ElementID, err)
			}
		}
	}

	log.Printf("Appended %d elements to Neo4j", len(elements))
	return nil
}

// AppendRelationships appends relationships as edges in Neo4j
func (s *Neo4jStorage) AppendRelationships(relationships []Relationship) error {
	if len(relationships) == 0 {
		return nil
	}

	session := s.driver.NewSession(s.ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(s.ctx)

	for _, rel := range relationships {
		// Convert metadata to JSON string
		metadataJSON, err := json.Marshal(rel.Metadata)
		if err != nil {
			log.Printf("Warning: failed to marshal metadata for relationship: %v", err)
			metadataJSON = []byte("{}")
		}

		// Sanitize relationship type (Neo4j requires uppercase, no spaces)
		relType := sanitizeRelationshipType(rel.RelationshipType)

		// Create relationship - use dynamic relationship type
		query := fmt.Sprintf(`
			MATCH (s:Element {element_id: $source_id, run_id: $run_id})
			MATCH (t:Element {element_id: $target_id, run_id: $run_id})
			MERGE (s)-[r:%s]->(t)
			SET r.created_at = $created_at,
				r.metadata_json = $metadata_json,
				r.doc_id = $doc_id,
				r.source_name = $source_name
			RETURN r
		`, relType)

		_, err = session.Run(s.ctx, query, map[string]interface{}{
			"source_id":     rel.SourceElementID,
			"target_id":     rel.TargetElementID,
			"run_id":        s.runID,
			"created_at":    time.Now().Format(time.RFC3339),
			"metadata_json": string(metadataJSON),
			"doc_id":        rel.DocID,
			"source_name":   rel.SourceName,
		})
		if err != nil {
			return fmt.Errorf("failed to append relationship %s->%s: %w", rel.SourceElementID, rel.TargetElementID, err)
		}
	}

	log.Printf("Appended %d relationships to Neo4j", len(relationships))
	return nil
}

// AppendEmbeddings appends embeddings as properties on Element nodes
func (s *Neo4jStorage) AppendEmbeddings(embeddings []Embedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	session := s.driver.NewSession(s.ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(s.ctx)

	for _, emb := range embeddings {
		// Update Element node with embedding
		_, err := session.Run(s.ctx, `
			MATCH (e:Element {element_id: $element_id, run_id: $run_id})
			SET e.embedding = $embedding,
				e.embedding_text = $text,
				e.embedding_dims = $dims
			RETURN e
		`, map[string]interface{}{
			"element_id": emb.ElementID,
			"run_id":     s.runID,
			"embedding":  emb.Embedding,
			"text":       emb.Text,
			"dims":       len(emb.Embedding),
		})
		if err != nil {
			return fmt.Errorf("failed to append embedding for element %s: %w", emb.ElementID, err)
		}
	}

	log.Printf("Appended %d embeddings to Neo4j", len(embeddings))
	return nil
}

// AppendLinks appends links (not implemented for Neo4j yet)
func (s *Neo4jStorage) AppendLinks(links []Link) error {
	if len(links) == 0 {
		return nil
	}

	session := s.driver.NewSession(s.ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(s.ctx)

	for _, link := range links {
		// Create Link node
		_, err := session.Run(s.ctx, `
			MERGE (l:Link {link_id: $link_id, run_id: $run_id})
			SET l.source_element_id = $source_element_id,
				l.doc_id = $doc_id,
				l.source_name = $source_name,
				l.link_type = $link_type,
				l.link_target = $link_target,
				l.link_text = $link_text,
				l.written_at = $written_at
			RETURN l
		`, map[string]interface{}{
			"link_id":           link.LinkID,
			"run_id":            s.runID,
			"source_element_id": link.SourceElementID,
			"doc_id":            link.DocID,
			"source_name":       link.SourceName,
			"link_type":         link.LinkType,
			"link_target":       link.LinkTarget,
			"link_text":         link.LinkText,
			"written_at":        time.Now().Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("failed to append link %s: %w", link.LinkID, err)
		}

		// Link to source element
		_, err = session.Run(s.ctx, `
			MATCH (e:Element {element_id: $source_element_id, run_id: $run_id})
			MATCH (l:Link {link_id: $link_id, run_id: $run_id})
			MERGE (e)-[:HAS_LINK]->(l)
		`, map[string]interface{}{
			"source_element_id": link.SourceElementID,
			"link_id":           link.LinkID,
			"run_id":            s.runID,
		})
		if err != nil {
			return fmt.Errorf("failed to link %s to element: %w", link.LinkID, err)
		}
	}

	log.Printf("Appended %d links to Neo4j", len(links))
	return nil
}

// Close closes the Neo4j connection
func (s *Neo4jStorage) Close() error {
	return s.driver.Close(s.ctx)
}

// GetRunStats returns statistics for the export run
func (s *Neo4jStorage) GetRunStats() (map[string]interface{}, error) {
	session := s.driver.NewSession(s.ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(s.ctx)

	stats := make(map[string]interface{})
	stats["run_id"] = s.runID

	// Count documents
	result, err := session.Run(s.ctx, `
		MATCH (d:Document {run_id: $run_id})
		RETURN count(d) as count
	`, map[string]interface{}{"run_id": s.runID})
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}
	if result.Next(s.ctx) {
		stats["document_count"] = result.Record().Values[0]
	}

	// Count elements
	result, err = session.Run(s.ctx, `
		MATCH (e:Element {run_id: $run_id})
		RETURN count(e) as count
	`, map[string]interface{}{"run_id": s.runID})
	if err != nil {
		return nil, fmt.Errorf("failed to count elements: %w", err)
	}
	if result.Next(s.ctx) {
		stats["element_count"] = result.Record().Values[0]
	}

	// Count relationships
	result, err = session.Run(s.ctx, `
		MATCH (s:Element {run_id: $run_id})-[r]->(t:Element {run_id: $run_id})
		RETURN count(r) as count
	`, map[string]interface{}{"run_id": s.runID})
	if err != nil {
		return nil, fmt.Errorf("failed to count relationships: %w", err)
	}
	if result.Next(s.ctx) {
		stats["relationship_count"] = result.Record().Values[0]
	}

	return stats, nil
}

// sanitizeRelationshipType converts relationship type to Neo4j-compatible format
func sanitizeRelationshipType(relType string) string {
	// Replace spaces with underscores, convert to uppercase
	result := ""
	for _, c := range relType {
		if c == ' ' || c == '-' {
			result += "_"
		} else if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			if c >= 'a' && c <= 'z' {
				result += string(c - 32) // Convert to uppercase
			} else {
				result += string(c)
			}
		}
	}
	if result == "" {
		result = "RELATED_TO"
	}
	return result
}
