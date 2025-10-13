package graph

import (
	"fmt"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
)

// OntologyConverter converts ontologies to generic graphs
type OntologyConverter struct {
	// Configuration options
	IncludeProvenance bool // Include UDML provenance in properties
	IncludeMentions   bool // Include mention details in properties
	LabelPrefix       string // Prefix for node labels (e.g., "Ont_")
}

// NewOntologyConverter creates a new ontology converter with default settings
func NewOntologyConverter() *OntologyConverter {
	return &OntologyConverter{
		IncludeProvenance: true,
		IncludeMentions:   true,
		LabelPrefix:       "",
	}
}

// ConvertToGraph converts an ontology to a generic graph
func (c *OntologyConverter) ConvertToGraph(ont *ontology.Ontology) (*Graph, error) {
	if ont == nil {
		return nil, fmt.Errorf("ontology cannot be nil")
	}

	// Create builder
	builder := NewBuilder(
		ont.ID+"_graph",
		ont.Name+" Knowledge Graph",
		ont.Version,
	)

	// Set metadata from ontology
	builder.SetMetadata("source_ontology_id", ont.ID)
	builder.SetMetadata("source_doc_id", ont.DocID)
	builder.SetMetadata("extractor_type", ont.Metadata.ExtractorType)
	builder.SetMetadata("extractor_version", ont.Metadata.ExtractorVersion)
	builder.SetMetadata("extraction_time", ont.Metadata.ExtractionTime)
	builder.SetMetadata("element_count", ont.Metadata.ElementCount)
	builder.SetMetadata("overall_confidence", ont.Metadata.Confidence)

	// Convert entities to nodes
	for _, entity := range ont.Entities {
		node := c.entityToNode(entity, ont.DocID)
		builder.AddNode(node)
	}

	// Convert relationships to edges
	for _, rel := range ont.Relationships {
		edge := c.relationshipToEdge(rel)
		builder.AddEdge(edge)
	}

	// Build and validate
	graph, err := builder.BuildAndValidate()
	if err != nil {
		return nil, fmt.Errorf("graph validation failed: %w", err)
	}

	return graph, nil
}

// entityToNode converts an ontology entity to a graph node
func (c *OntologyConverter) entityToNode(entity ontology.Entity, docID string) Node {
	// Create labels - include entity type and "Entity"
	labels := []string{
		c.LabelPrefix + "Entity",
		c.LabelPrefix + c.normalizeLabel(string(entity.Type)),
	}

	// Build properties bag - this is key for property graph export!
	properties := make(map[string]interface{})

	// Core properties
	properties["name"] = entity.Name
	properties["type"] = string(entity.Type)

	// Include all entity attributes in properties
	for key, value := range entity.Attributes {
		properties[key] = value
	}

	// Add mention information if enabled
	if c.IncludeMentions && len(entity.Mentions) > 0 {
		properties["mention_count"] = len(entity.Mentions)
		properties["first_mention"] = entity.Mentions[0].Text

		// Include all mention texts
		mentionTexts := make([]string, len(entity.Mentions))
		for i, mention := range entity.Mentions {
			mentionTexts[i] = mention.Text
		}
		properties["mentions"] = mentionTexts

		// Include mention element IDs
		mentionElementIDs := make([]string, len(entity.Mentions))
		for i, mention := range entity.Mentions {
			mentionElementIDs[i] = mention.ElementID
		}
		properties["mention_element_ids"] = mentionElementIDs
	}

	// Add provenance if enabled
	if c.IncludeProvenance {
		properties["extracted_at"] = entity.CreatedAt.Format(time.RFC3339)
		properties["updated_at"] = entity.UpdatedAt.Format(time.RFC3339)
	}

	return Node{
		ID:              entity.ID,
		Labels:          labels,
		Properties:      properties,
		SourceElementID: entity.ElementID,
		SourceDocID:     docID,
		Domain:          entity.Domain,
		Confidence:      entity.Confidence,
		CreatedAt:       entity.CreatedAt,
		UpdatedAt:       entity.UpdatedAt,
	}
}

// relationshipToEdge converts an ontology relationship to a graph edge
func (c *OntologyConverter) relationshipToEdge(rel ontology.Relationship) Edge {
	// Build properties bag for the relationship
	properties := make(map[string]interface{})

	// Core properties
	properties["type"] = string(rel.Type)

	// Include all relationship attributes in properties
	for key, value := range rel.Attributes {
		properties[key] = value
	}

	// Add evidence as property
	if rel.Evidence != "" {
		properties["evidence"] = rel.Evidence
	}

	// Add provenance
	if c.IncludeProvenance {
		properties["extracted_at"] = rel.CreatedAt.Format(time.RFC3339)
	}

	// Normalize relationship type for edge type (UPPER_CASE convention)
	edgeType := c.normalizeEdgeType(string(rel.Type))

	return Edge{
		ID:              rel.ID,
		Type:            edgeType,
		SourceID:        rel.SourceID,
		TargetID:        rel.TargetID,
		Properties:      properties,
		SourceElementID: rel.ElementID,
		Domain:          rel.Domain,
		Confidence:      rel.Confidence,
		Evidence:        rel.Evidence,
		CreatedAt:       rel.CreatedAt,
	}
}

// normalizeLabel normalizes an entity type to a label (PascalCase)
func (c *OntologyConverter) normalizeLabel(entityType string) string {
	// Convert "person" -> "Person", "technology" -> "Technology"
	if entityType == "" {
		return "Entity"
	}
	return strings.ToUpper(entityType[:1]) + entityType[1:]
}

// normalizeEdgeType normalizes a relationship type to edge type (UPPER_CASE)
func (c *OntologyConverter) normalizeEdgeType(relType string) string {
	// Convert "is_a" -> "IS_A", "part_of" -> "PART_OF"
	return strings.ToUpper(relType)
}

// ConvertMultipleToGraph converts multiple ontologies into a single merged graph
func (c *OntologyConverter) ConvertMultipleToGraph(ontologies []*ontology.Ontology) (*Graph, error) {
	if len(ontologies) == 0 {
		return nil, fmt.Errorf("no ontologies provided")
	}

	// Convert first ontology
	graph, err := c.ConvertToGraph(ontologies[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert ontology 0: %w", err)
	}

	// Merge remaining ontologies
	for i, ont := range ontologies[1:] {
		otherGraph, err := c.ConvertToGraph(ont)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ontology %d: %w", i+1, err)
		}
		graph = graph.Merge(otherGraph)
	}

	return graph, nil
}

// ConvertWithOptions converts an ontology to a graph with custom options
func ConvertWithOptions(ont *ontology.Ontology, opts ConversionOptions) (*Graph, error) {
	converter := &OntologyConverter{
		IncludeProvenance: opts.IncludeProvenance,
		IncludeMentions:   opts.IncludeMentions,
		LabelPrefix:       opts.LabelPrefix,
	}
	return converter.ConvertToGraph(ont)
}

// ConversionOptions configures ontology-to-graph conversion
type ConversionOptions struct {
	IncludeProvenance bool   // Include UDML provenance in properties
	IncludeMentions   bool   // Include mention details in properties
	LabelPrefix       string // Prefix for node labels
	MinConfidence     float64 // Minimum confidence threshold (0.0-1.0)
}

// DefaultConversionOptions returns default conversion options
func DefaultConversionOptions() ConversionOptions {
	return ConversionOptions{
		IncludeProvenance: true,
		IncludeMentions:   true,
		LabelPrefix:       "",
		MinConfidence:     0.0,
	}
}