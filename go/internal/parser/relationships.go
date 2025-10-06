package parser

import (
	"github.com/google/uuid"
)

// CreateSequentialRelationships adds next_sibling and previous_sibling relationships
// between elements that share the same parent_id.
// This matches Python's StructuralRelationshipDetector behavior.
func CreateSequentialRelationships(elements []Element) []Relationship {
	var relationships []Relationship

	// Group elements by parent_id
	parentChildren := make(map[string][]string)
	for _, elem := range elements {
		if elem.ParentID != "" {
			parentChildren[elem.ParentID] = append(parentChildren[elem.ParentID], elem.ElementID)
		}
	}

	// Create sibling relationships for each parent with multiple children
	for _, childIDs := range parentChildren {
		// Skip if only one child
		if len(childIDs) <= 1 {
			continue
		}

		// Create relationships between consecutive siblings
		for i := 0; i < len(childIDs)-1; i++ {
			prevID := childIDs[i]
			nextID := childIDs[i+1]

			// Create next_sibling relationship
			nextRel := Relationship{
				RelationshipID:   generateRelID(),
				SourceElementID:  prevID,
				TargetElementID:  nextID,
				RelationshipType: "next_sibling",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			relationships = append(relationships, nextRel)

			// Create previous_sibling relationship
			prevRel := Relationship{
				RelationshipID:   generateRelID(),
				SourceElementID:  nextID,
				TargetElementID:  prevID,
				RelationshipType: "previous_sibling",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			relationships = append(relationships, prevRel)
		}
	}

	return relationships
}

func generateRelID() string {
	return "rel_" + uuid.New().String()[:8]
}
