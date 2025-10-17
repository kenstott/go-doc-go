package ontology

import (
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
)

// ConvertToAnalyticsEntity converts an ontology.Entity to analytics.OntologyEntity
func ConvertToAnalyticsEntity(entity Entity, docID string, sourceName string, runID string) analytics.OntologyEntity {
	// Store run_id in attributes for filtering
	attributes := make(map[string]interface{})
	for k, v := range entity.Attributes {
		attributes[k] = v
	}
	attributes["run_id"] = runID

	return analytics.OntologyEntity{
		EntityID:    entity.ID,
		DocID:       docID,
		SourceName:  sourceName,
		EntityName:  entity.Name,
		EntityType:  string(entity.Type),
		Domain:      entity.Domain,
		Confidence:  entity.Confidence,
		Attributes:  attributes,
		ElementID:   entity.ElementID,
		ExtractedAt: entity.CreatedAt,
	}
}

// ConvertToAnalyticsRelationship converts an ontology.Relationship to analytics.OntologyRelationship
func ConvertToAnalyticsRelationship(rel Relationship, docID string, sourceName string, runID string) analytics.OntologyRelationship {
	// Store run_id in attributes for filtering
	attributes := make(map[string]interface{})
	for k, v := range rel.Attributes {
		attributes[k] = v
	}
	attributes["run_id"] = runID

	return analytics.OntologyRelationship{
		RelationshipID:   rel.ID,
		DocID:            docID,
		SourceName:       sourceName,
		SourceEntityID:   rel.SourceID,
		TargetEntityID:   rel.TargetID,
		RelationshipType: string(rel.Type),
		Domain:           rel.Domain,
		Confidence:       rel.Confidence,
		Evidence:         rel.Evidence,
		Attributes:       attributes,
		ElementID:        rel.ElementID,
		ExtractedAt:      rel.CreatedAt,
	}
}

// ConvertToAnalyticsMention converts an ontology.Mention to analytics.OntologyMention
func ConvertToAnalyticsMention(mention Mention, entityID string, docID string, sourceName string) analytics.OntologyMention {
	return analytics.OntologyMention{
		MentionID:     entityID + "_mention_" + mention.ElementID,
		EntityID:      entityID,
		DocID:         docID,
		SourceName:    sourceName,
		ElementID:     mention.ElementID,
		MentionText:   mention.Text,
		StartPosition: mention.StartPos,
		EndPosition:   mention.EndPos,
		ExtractedAt:   time.Now(),
	}
}

// ConvertFromAnalyticsEntity converts analytics.OntologyEntity back to ontology.Entity
func ConvertFromAnalyticsEntity(analyticsEntity analytics.OntologyEntity) Entity {
	// Extract mentions if stored in attributes (not implemented in this direction yet)
	// For now, just convert the core fields

	return Entity{
		ID:         analyticsEntity.EntityID,
		Name:       analyticsEntity.EntityName,
		Type:       EntityType(analyticsEntity.EntityType),
		Domain:     analyticsEntity.Domain,
		Confidence: analyticsEntity.Confidence,
		Attributes: analyticsEntity.Attributes,
		ElementID:  analyticsEntity.ElementID,
		Mentions:   []Mention{}, // Mentions stored separately in analytics
		CreatedAt:  analyticsEntity.ExtractedAt,
		UpdatedAt:  analyticsEntity.ExtractedAt,
	}
}

// ConvertFromAnalyticsRelationship converts analytics.OntologyRelationship back to ontology.Relationship
func ConvertFromAnalyticsRelationship(analyticsRel analytics.OntologyRelationship) Relationship {
	return Relationship{
		ID:         analyticsRel.RelationshipID,
		Type:       RelationshipType(analyticsRel.RelationshipType),
		Domain:     analyticsRel.Domain,
		SourceID:   analyticsRel.SourceEntityID,
		TargetID:   analyticsRel.TargetEntityID,
		Confidence: analyticsRel.Confidence,
		Attributes: analyticsRel.Attributes,
		ElementID:  analyticsRel.ElementID,
		Evidence:   analyticsRel.Evidence,
		CreatedAt:  analyticsRel.ExtractedAt,
	}
}

// ConvertEntitiesToAnalytics converts a slice of entities to analytics format
func ConvertEntitiesToAnalytics(entities []Entity, docID string, sourceName string, runID string) []analytics.OntologyEntity {
	result := make([]analytics.OntologyEntity, len(entities))
	for i, entity := range entities {
		result[i] = ConvertToAnalyticsEntity(entity, docID, sourceName, runID)
	}
	return result
}

// ConvertRelationshipsToAnalytics converts a slice of relationships to analytics format
func ConvertRelationshipsToAnalytics(relationships []Relationship, docID string, sourceName string, runID string) []analytics.OntologyRelationship {
	result := make([]analytics.OntologyRelationship, len(relationships))
	for i, rel := range relationships {
		result[i] = ConvertToAnalyticsRelationship(rel, docID, sourceName, runID)
	}
	return result
}

// ConvertEntitiesFromAnalytics converts a slice of analytics entities to ontology format
func ConvertEntitiesFromAnalytics(analyticsEntities []analytics.OntologyEntity) []Entity {
	result := make([]Entity, len(analyticsEntities))
	for i, entity := range analyticsEntities {
		result[i] = ConvertFromAnalyticsEntity(entity)
	}
	return result
}

// ConvertRelationshipsFromAnalytics converts a slice of analytics relationships to ontology format
func ConvertRelationshipsFromAnalytics(analyticsRels []analytics.OntologyRelationship) []Relationship {
	result := make([]Relationship, len(analyticsRels))
	for i, rel := range analyticsRels {
		result[i] = ConvertFromAnalyticsRelationship(rel)
	}
	return result
}
