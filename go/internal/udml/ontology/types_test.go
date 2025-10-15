package ontology

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEntity_Creation(t *testing.T) {
	entity := Entity{
		ID:         "ent1",
		Name:       "John Doe",
		Type:       EntityTypePerson,
		Confidence: 0.95,
		Attributes: map[string]interface{}{
			"role": "CEO",
		},
		ElementID: "elem1",
		Mentions: []Mention{
			{
				ElementID: "elem1",
				Text:      "John Doe",
				StartPos:  0,
				EndPos:    8,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if entity.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", entity.Name)
	}

	if entity.Type != EntityTypePerson {
		t.Errorf("Expected type 'person', got '%s'", entity.Type)
	}

	if len(entity.Mentions) != 1 {
		t.Errorf("Expected 1 mention, got %d", len(entity.Mentions))
	}
}

func TestRelationship_Creation(t *testing.T) {
	rel := Relationship{
		ID:         "rel1",
		Type:       RelationshipCreatedBy,
		SourceID:   "ent1",
		TargetID:   "ent2",
		Confidence: 0.9,
		Attributes: map[string]interface{}{},
		ElementID:  "elem1",
		Evidence:   "John Doe created this document",
		CreatedAt:  time.Now(),
	}

	if rel.SourceID != "ent1" {
		t.Errorf("Expected source 'ent1', got '%s'", rel.SourceID)
	}

	if rel.Type != RelationshipCreatedBy {
		t.Errorf("Expected type 'created_by', got '%s'", rel.Type)
	}
}

func TestClass_Creation(t *testing.T) {
	class := Class{
		ID:          "cls1",
		Name:        "Person",
		Description: "A human being",
		Properties: []Property{
			{
				Name:        "name",
				Type:        "string",
				Description: "Person's full name",
				Required:    true,
			},
			{
				Name:        "age",
				Type:        "int",
				Description: "Person's age",
				Required:    false,
			},
		},
		Attributes: map[string]interface{}{},
		CreatedAt:  time.Now(),
	}

	if class.Name != "Person" {
		t.Errorf("Expected name 'Person', got '%s'", class.Name)
	}

	if len(class.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(class.Properties))
	}
}

func TestOntology_Creation(t *testing.T) {
	ont := Ontology{
		ID:          "ont1",
		DocID:       "doc1",
		Name:        "Test Ontology",
		Description: "A test ontology",
		Version:     "1.0.0",
		Entities: []Entity{
			{
				ID:         "ent1",
				Name:       "John Doe",
				Type:       EntityTypePerson,
				Confidence: 0.95,
			},
		},
		Relationships: []Relationship{},
		Classes:       []Class{},
		Metadata: OntologyMeta{
			ExtractorType: "mock",
			ModelName:     "test-model",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if ont.DocID != "doc1" {
		t.Errorf("Expected doc_id 'doc1', got '%s'", ont.DocID)
	}

	if len(ont.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(ont.Entities))
	}
}

func TestOntology_ToJSON(t *testing.T) {
	ont := Ontology{
		ID:            "ont1",
		DocID:         "doc1",
		Name:          "Test",
		Entities:      []Entity{},
		Relationships: []Relationship{},
		Classes:       []Class{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	jsonData, err := ont.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["doc_id"] != "doc1" {
		t.Errorf("Expected doc_id 'doc1' in JSON, got %v", parsed["doc_id"])
	}
}

func TestOntology_ToJSONCompact(t *testing.T) {
	ont := Ontology{
		ID:            "ont1",
		DocID:         "doc1",
		Entities:      []Entity{},
		Relationships: []Relationship{},
		Classes:       []Class{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	prettyJSON, _ := ont.ToJSON()
	compactJSON, _ := ont.ToJSONCompact()

	if len(compactJSON) >= len(prettyJSON) {
		t.Error("Expected compact JSON to be smaller than pretty JSON")
	}
}

func TestOntology_GetEntityByID(t *testing.T) {
	ont := Ontology{
		Entities: []Entity{
			{ID: "ent1", Name: "Entity1"},
			{ID: "ent2", Name: "Entity2"},
		},
	}

	entity := ont.GetEntityByID("ent2")
	if entity == nil {
		t.Fatal("GetEntityByID() returned nil")
	}

	if entity.Name != "Entity2" {
		t.Errorf("Expected 'Entity2', got '%s'", entity.Name)
	}

	notFound := ont.GetEntityByID("ent999")
	if notFound != nil {
		t.Error("Expected nil for non-existent entity")
	}
}

func TestOntology_GetEntityByName(t *testing.T) {
	ont := Ontology{
		Entities: []Entity{
			{ID: "ent1", Name: "John Doe"},
			{ID: "ent2", Name: "Jane Smith"},
		},
	}

	entity := ont.GetEntityByName("Jane Smith")
	if entity == nil {
		t.Fatal("GetEntityByName() returned nil")
	}

	if entity.ID != "ent2" {
		t.Errorf("Expected ID 'ent2', got '%s'", entity.ID)
	}
}

func TestOntology_GetEntitiesByType(t *testing.T) {
	ont := Ontology{
		Entities: []Entity{
			{ID: "ent1", Name: "John", Type: EntityTypePerson},
			{ID: "ent2", Name: "Acme Corp", Type: EntityTypeOrganization},
			{ID: "ent3", Name: "Jane", Type: EntityTypePerson},
		},
	}

	people := ont.GetEntitiesByType(EntityTypePerson)
	if len(people) != 2 {
		t.Errorf("Expected 2 people, got %d", len(people))
	}

	orgs := ont.GetEntitiesByType(EntityTypeOrganization)
	if len(orgs) != 1 {
		t.Errorf("Expected 1 organization, got %d", len(orgs))
	}
}

func TestOntology_GetRelationshipsBySource(t *testing.T) {
	ont := Ontology{
		Relationships: []Relationship{
			{ID: "rel1", SourceID: "ent1", TargetID: "ent2"},
			{ID: "rel2", SourceID: "ent1", TargetID: "ent3"},
			{ID: "rel3", SourceID: "ent2", TargetID: "ent3"},
		},
	}

	rels := ont.GetRelationshipsBySource("ent1")
	if len(rels) != 2 {
		t.Errorf("Expected 2 relationships from ent1, got %d", len(rels))
	}
}

func TestOntology_GetRelationshipsByTarget(t *testing.T) {
	ont := Ontology{
		Relationships: []Relationship{
			{ID: "rel1", SourceID: "ent1", TargetID: "ent3"},
			{ID: "rel2", SourceID: "ent2", TargetID: "ent3"},
		},
	}

	rels := ont.GetRelationshipsByTarget("ent3")
	if len(rels) != 2 {
		t.Errorf("Expected 2 relationships to ent3, got %d", len(rels))
	}
}

func TestOntology_GetRelationshipsByType(t *testing.T) {
	ont := Ontology{
		Relationships: []Relationship{
			{ID: "rel1", Type: RelationshipIsA},
			{ID: "rel2", Type: RelationshipPartOf},
			{ID: "rel3", Type: RelationshipIsA},
		},
	}

	isaRels := ont.GetRelationshipsByType(RelationshipIsA)
	if len(isaRels) != 2 {
		t.Errorf("Expected 2 is_a relationships, got %d", len(isaRels))
	}
}

func TestOntology_GetClassByID(t *testing.T) {
	ont := Ontology{
		Classes: []Class{
			{ID: "cls1", Name: "Person"},
			{ID: "cls2", Name: "Organization"},
		},
	}

	class := ont.GetClassByID("cls2")
	if class == nil {
		t.Fatal("GetClassByID() returned nil")
	}

	if class.Name != "Organization" {
		t.Errorf("Expected 'Organization', got '%s'", class.Name)
	}
}

func TestOntology_GetStats(t *testing.T) {
	ont := Ontology{
		Entities: []Entity{
			{Type: EntityTypePerson},
			{Type: EntityTypePerson},
			{Type: EntityTypeOrganization},
		},
		Relationships: []Relationship{
			{Type: RelationshipIsA},
			{Type: RelationshipPartOf},
		},
		Classes: []Class{
			{ID: "cls1"},
		},
	}

	stats := ont.GetStats()

	if stats.EntityCount != 3 {
		t.Errorf("Expected 3 entities, got %d", stats.EntityCount)
	}

	if stats.RelationshipCount != 2 {
		t.Errorf("Expected 2 relationships, got %d", stats.RelationshipCount)
	}

	if stats.ClassCount != 1 {
		t.Errorf("Expected 1 class, got %d", stats.ClassCount)
	}

	if stats.EntityTypes[EntityTypePerson] != 2 {
		t.Errorf("Expected 2 person entities, got %d", stats.EntityTypes[EntityTypePerson])
	}

	if stats.RelationshipTypes[RelationshipIsA] != 1 {
		t.Errorf("Expected 1 is_a relationship, got %d", stats.RelationshipTypes[RelationshipIsA])
	}
}

func TestOntology_Validate(t *testing.T) {
	t.Run("valid ontology", func(t *testing.T) {
		ont := Ontology{
			Entities: []Entity{
				{ID: "ent1", Name: "Entity1"},
				{ID: "ent2", Name: "Entity2"},
			},
			Relationships: []Relationship{
				{ID: "rel1", SourceID: "ent1", TargetID: "ent2"},
			},
			Classes: []Class{
				{ID: "cls1", Name: "Class1"},
			},
		}

		err := ont.Validate()
		if err != nil {
			t.Errorf("Validate() returned error for valid ontology: %v", err)
		}
	})

	t.Run("empty entity ID", func(t *testing.T) {
		ont := Ontology{
			Entities: []Entity{
				{ID: "", Name: "Entity1"},
			},
		}

		err := ont.Validate()
		if err == nil {
			t.Error("Expected validation error for empty entity ID")
		}
	})

	t.Run("duplicate entity ID", func(t *testing.T) {
		ont := Ontology{
			Entities: []Entity{
				{ID: "ent1", Name: "Entity1"},
				{ID: "ent1", Name: "Entity2"},
			},
		}

		err := ont.Validate()
		if err == nil {
			t.Error("Expected validation error for duplicate entity ID")
		}
	})

	t.Run("relationship with unknown source", func(t *testing.T) {
		ont := Ontology{
			Entities: []Entity{
				{ID: "ent1", Name: "Entity1"},
			},
			Relationships: []Relationship{
				{ID: "rel1", SourceID: "ent999", TargetID: "ent1"},
			},
		}

		err := ont.Validate()
		if err == nil {
			t.Error("Expected validation error for unknown source entity")
		}
	})

	t.Run("relationship with unknown target", func(t *testing.T) {
		ont := Ontology{
			Entities: []Entity{
				{ID: "ent1", Name: "Entity1"},
			},
			Relationships: []Relationship{
				{ID: "rel1", SourceID: "ent1", TargetID: "ent999"},
			},
		}

		err := ont.Validate()
		if err == nil {
			t.Error("Expected validation error for unknown target entity")
		}
	})

	t.Run("duplicate class ID", func(t *testing.T) {
		ont := Ontology{
			Classes: []Class{
				{ID: "cls1", Name: "Class1"},
				{ID: "cls1", Name: "Class2"},
			},
		}

		err := ont.Validate()
		if err == nil {
			t.Error("Expected validation error for duplicate class ID")
		}
	})
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("test error message")

	if err == nil {
		t.Fatal("NewValidationError() returned nil")
	}

	expectedMsg := "ontology validation error: test error message"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestEntityTypes(t *testing.T) {
	types := []EntityType{
		EntityTypePerson,
		EntityTypeOrganization,
		EntityTypeLocation,
		EntityTypeDate,
		EntityTypeEvent,
		EntityTypeConcept,
		EntityTypeProduct,
		EntityTypeTechnology,
		EntityTypeCustom,
	}

	if len(types) != 9 {
		t.Errorf("Expected 9 entity types, got %d", len(types))
	}

	// Verify they're all distinct
	seen := make(map[EntityType]bool)
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("Duplicate entity type: %s", typ)
		}
		seen[typ] = true
	}
}

func TestRelationshipTypes(t *testing.T) {
	types := []RelationshipType{
		RelationshipIsA,
		RelationshipPartOf,
		RelationshipRelatedTo,
		RelationshipLocatedIn,
		RelationshipOccurredAt,
		RelationshipCreatedBy,
		RelationshipMentions,
		RelationshipDependsOn,
		RelationshipImplements,
		RelationshipExtends,
		RelationshipContains,
		RelationshipReferencedBy,
		RelationshipCustom,
	}

	if len(types) != 13 {
		t.Errorf("Expected 13 relationship types, got %d", len(types))
	}

	// Verify they're all distinct
	seen := make(map[RelationshipType]bool)
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("Duplicate relationship type: %s", typ)
		}
		seen[typ] = true
	}
}
