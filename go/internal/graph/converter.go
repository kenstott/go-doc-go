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
	IncludeProvenance  bool   // Include UDML provenance in properties
	IncludeMentions    bool   // Include mention details in properties
	CreateMentionNodes bool   // Create separate mention nodes with MENTIONED_BY edges
	EnrichContext      bool   // Add element_type and section_title properties
	LabelPrefix        string // Prefix for node labels (e.g., "Ont_")
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

	// Create mention nodes and MENTIONED_BY edges if enabled
	if c.CreateMentionNodes {
		for _, entity := range ont.Entities {
			for _, mention := range entity.Mentions {
				// Create mention node
				mentionNode := c.mentionToNode(mention, entity, ont.DocID)
				builder.AddNode(mentionNode)

				// Create MENTIONED_BY edge from mention to canonical entity
				mentionEdge := Edge{
					ID:              "mentioned_by_" + mentionNode.ID,
					Type:            "MENTIONED_BY",
					SourceID:        mentionNode.ID,
					TargetID:        entity.ID,
					Properties:      make(map[string]interface{}),
					SourceElementID: mention.ElementID,
					Domain:          entity.Domain,
					Confidence:      entity.Confidence,
					CreatedAt:       entity.CreatedAt,
				}
				builder.AddEdge(mentionEdge)
			}
		}
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
		properties["element_id"] = entity.ElementID
		properties["doc_id"] = docID
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

// mentionToNode converts an ontology mention to a graph node
// Mention nodes share the same entity type labels as their canonical entity, plus "Mention"
func (c *OntologyConverter) mentionToNode(mention ontology.Mention, canonicalEntity ontology.Entity, docID string) Node {
	// Generate unique ID for mention node
	mentionID := "mention_" + canonicalEntity.ID + "_" + mention.ElementID

	// Create labels - same entity type as canonical, plus "Mention"
	labels := []string{
		c.LabelPrefix + "Mention",
		c.LabelPrefix + c.normalizeLabel(string(canonicalEntity.Type)),
	}

	// Build properties bag
	properties := make(map[string]interface{})

	// Core mention properties
	properties["mention_text"] = mention.Text
	properties["canonical_entity_id"] = canonicalEntity.ID
	properties["canonical_entity_name"] = canonicalEntity.Name
	properties["start_position"] = mention.StartPos
	properties["end_position"] = mention.EndPos

	// Add provenance
	if c.IncludeProvenance {
		properties["element_id"] = mention.ElementID
		properties["doc_id"] = docID
		properties["extracted_at"] = canonicalEntity.CreatedAt.Format(time.RFC3339)
	}

	return Node{
		ID:              mentionID,
		Labels:          labels,
		Properties:      properties,
		SourceElementID: mention.ElementID,
		SourceDocID:     docID,
		Domain:          canonicalEntity.Domain,
		Confidence:      canonicalEntity.Confidence, // Inherit from canonical
		CreatedAt:       canonicalEntity.CreatedAt,
		UpdatedAt:       canonicalEntity.UpdatedAt,
	}
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
		IncludeProvenance:  opts.IncludeProvenance,
		IncludeMentions:    opts.IncludeMentions,
		CreateMentionNodes: opts.CreateMentionNodes,
		EnrichContext:      opts.EnrichContext,
		LabelPrefix:        opts.LabelPrefix,
	}
	return converter.ConvertToGraph(ont)
}

// ConversionOptions configures ontology-to-graph conversion
type ConversionOptions struct {
	IncludeProvenance bool    // Include UDML provenance in properties
	IncludeMentions   bool    // Include mention details in properties
	CreateMentionNodes bool   // Create separate mention nodes with MENTIONED_BY edges
	EnrichContext     bool    // Add element_type and section_title properties
	LabelPrefix       string  // Prefix for node labels
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

// ConvertCommunityGraphToOntology converts community detection results to ontology entities and relationships
// This enables community structure to be stored in Parquet alongside extracted entities
func ConvertCommunityGraphToOntology(
	graph *Graph,
	communities *CommunityResult,
	sourceRunID string,
) ([]ontology.Entity, []ontology.Relationship, error) {
	if graph == nil || communities == nil {
		return nil, nil, fmt.Errorf("graph and communities cannot be nil")
	}

	entities := []ontology.Entity{}
	relationships := []ontology.Relationship{}
	now := time.Now()

	// Convert each community to an OntologyEntity
	for _, comm := range communities.Communities {
		// Create community entity
		entity := ontology.Entity{
			ID:         fmt.Sprintf("community_%s_%d", communities.Algorithm, comm.ID),
			Name:       fmt.Sprintf("Community %d (Level %d)", comm.ID, comm.Level),
			Type:       "community",
			Domain:     "graph_analysis",
			Confidence: 1.0, // Community detection is deterministic
			Attributes: map[string]interface{}{
				"community_id":  comm.ID,
				"algorithm":     string(communities.Algorithm),
				"level":         comm.Level,
				"size":          comm.Size,
				"member_count":  len(comm.NodeIDs),
				"modularity":    communities.Modularity,
				"max_level":     communities.MaxLevel,
			},
			ElementID: "", // Synthetic entity - no source element
			Mentions:  []ontology.Mention{},
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Add parent reference if exists
		if comm.ParentID != nil {
			entity.Attributes["parent_id"] = *comm.ParentID
		}

		// Add child references if exist
		if len(comm.ChildIDs) > 0 {
			entity.Attributes["child_ids"] = comm.ChildIDs
		}

		// Add member node IDs
		entity.Attributes["member_node_ids"] = comm.NodeIDs

		entities = append(entities, entity)
	}

	// Create BELONGS_TO relationships from original graph nodes to their Level 0 communities
	// Find all Level 0 communities
	level0Communities := map[string]int{} // nodeID -> communityID mapping
	for _, comm := range communities.Communities {
		if comm.Level == 0 {
			for _, nodeID := range comm.NodeIDs {
				level0Communities[nodeID] = comm.ID
			}
		}
	}

	// Create BELONGS_TO relationships for original graph nodes
	for _, node := range graph.Nodes {
		// Skip synthetic community nodes
		isCommunityNode := false
		for _, label := range node.Labels {
			if label == "Community" {
				isCommunityNode = true
				break
			}
		}
		if isCommunityNode {
			continue
		}

		// Find which Level 0 community this node belongs to
		if commID, found := level0Communities[node.ID]; found {
			rel := ontology.Relationship{
				ID:         fmt.Sprintf("belongs_to_%s_comm_%d", node.ID, commID),
				Type:       "BELONGS_TO",
				Domain:     "graph_analysis",
				SourceID:   node.ID,
				TargetID:   fmt.Sprintf("community_%s_%d", communities.Algorithm, commID),
				Confidence: 1.0,
				Attributes: map[string]interface{}{
					"algorithm":    string(communities.Algorithm),
					"community_id": commID,
					"level":        0,
				},
				ElementID: "", // Synthetic relationship
				Evidence:  fmt.Sprintf("Node %s assigned to community %d by %s algorithm", node.ID, commID, communities.Algorithm),
				CreatedAt: now,
			}
			relationships = append(relationships, rel)
		}
	}

	// Create PARENT_OF relationships for community hierarchy
	for _, comm := range communities.Communities {
		if comm.ParentID != nil {
			rel := ontology.Relationship{
				ID:       fmt.Sprintf("parent_of_%d_to_%d", *comm.ParentID, comm.ID),
				Type:     "PARENT_OF",
				Domain:   "graph_analysis",
				SourceID: fmt.Sprintf("community_%s_%d", communities.Algorithm, *comm.ParentID),
				TargetID: fmt.Sprintf("community_%s_%d", communities.Algorithm, comm.ID),
				Confidence: 1.0,
				Attributes: map[string]interface{}{
					"algorithm":    string(communities.Algorithm),
					"parent_id":    *comm.ParentID,
					"child_id":     comm.ID,
					"parent_level": comm.Level + 1,
					"child_level":  comm.Level,
				},
				ElementID: "",
				Evidence: fmt.Sprintf("Community %d (Level %d) is parent of community %d (Level %d)",
					*comm.ParentID, comm.Level+1, comm.ID, comm.Level),
				CreatedAt: now,
			}
			relationships = append(relationships, rel)
		}
	}

	return entities, relationships, nil
}