package ontology

import (
	"fmt"
	"strings"
)

// PromptTemplate defines a template for LLM prompts
type PromptTemplate struct {
	Name        string
	Description string
	Template    string
	Variables   []string
}

// DefaultPrompts contains default extraction prompt templates
var DefaultPrompts = map[string]*PromptTemplate{
	"entity_extraction": {
		Name:        "entity_extraction",
		Description: "Extracts named entities from text",
		Template: `You are an expert at extracting named entities from documents.

Extract all named entities from the following text. For each entity, provide:
1. name: The entity name
2. type: One of [person, organization, location, date, event, concept, product, technology]
3. confidence: Your confidence level (0.0-1.0)
4. attributes: Any relevant attributes (optional)

Text:
{{.Text}}

Return the result as a JSON array of entities in this format:
{
  "entities": [
    {
      "name": "entity name",
      "type": "person",
      "confidence": 0.95,
      "attributes": {"role": "CEO"}
    }
  ]
}

Be precise and only extract entities that are clearly mentioned in the text.`,
		Variables: []string{"Text"},
	},

	"relationship_extraction": {
		Name:        "relationship_extraction",
		Description: "Extracts relationships between entities",
		Template: `You are an expert at identifying relationships between entities in documents.

Given the following entities and text, extract all relationships between them.

Entities:
{{.Entities}}

Text:
{{.Text}}

For each relationship, provide:
1. source: Source entity name
2. target: Target entity name
3. type: One of [is_a, part_of, related_to, located_in, occurred_at, created_by, mentions, depends_on, implements, extends, contains]
4. confidence: Your confidence level (0.0-1.0)
5. evidence: The text that supports this relationship

Return the result as a JSON array of relationships in this format:
{
  "relationships": [
    {
      "source": "entity1",
      "target": "entity2",
      "type": "created_by",
      "confidence": 0.90,
      "evidence": "text supporting this relationship"
    }
  ]
}

Only extract relationships that are explicitly stated or strongly implied in the text.`,
		Variables: []string{"Entities", "Text"},
	},

	"class_extraction": {
		Name:        "class_extraction",
		Description: "Extracts ontology classes/concepts",
		Template: `You are an expert at identifying conceptual classes and their relationships in documents.

Analyze the following text and extract key concepts that could form an ontology.

Text:
{{.Text}}

For each class/concept, provide:
1. name: The class name (singular, CamelCase)
2. description: A brief description
3. parent: Parent class name (if this is a specialization)
4. properties: List of properties with name, type, and description

Return the result as a JSON array of classes in this format:
{
  "classes": [
    {
      "name": "ClassName",
      "description": "Description of the concept",
      "parent": "ParentClassName",
      "properties": [
        {
          "name": "propertyName",
          "type": "string",
          "description": "Property description",
          "required": false
        }
      ]
    }
  ]
}

Focus on domain-specific concepts that appear in the text.`,
		Variables: []string{"Text"},
	},

	"combined_extraction": {
		Name:        "combined_extraction",
		Description: "Extracts entities, relationships, and classes in one pass",
		Template: `You are an expert knowledge engineer extracting structured information from documents.

Extract a complete ontology from the following text, including:
1. Entities (people, organizations, locations, concepts, etc.)
2. Relationships between entities
3. Key concepts/classes in the domain

Text:
{{.Text}}

Return the result in this JSON format:
{
  "entities": [
    {
      "name": "entity name",
      "type": "person|organization|location|date|event|concept|product|technology",
      "confidence": 0.95,
      "attributes": {}
    }
  ],
  "relationships": [
    {
      "source": "entity1",
      "target": "entity2",
      "type": "is_a|part_of|related_to|located_in|occurred_at|created_by|mentions|depends_on",
      "confidence": 0.90,
      "evidence": "supporting text"
    }
  ],
  "classes": [
    {
      "name": "ClassName",
      "description": "Description",
      "parent": "ParentClass",
      "properties": []
    }
  ]
}

Be thorough but precise. Only extract information that is clearly stated or strongly implied.`,
		Variables: []string{"Text"},
	},
}

// GetPrompt retrieves a prompt template by name
func GetPrompt(name string) (*PromptTemplate, error) {
	prompt, exists := DefaultPrompts[name]
	if !exists {
		return nil, fmt.Errorf("unknown prompt template: %s", name)
	}
	return prompt, nil
}

// RenderPrompt renders a prompt template with the given variables
func RenderPrompt(templateName string, vars map[string]string) (string, error) {
	template, err := GetPrompt(templateName)
	if err != nil {
		return "", err
	}

	result := template.Template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// BuildEntityExtractionPrompt builds a prompt for entity extraction
func BuildEntityExtractionPrompt(text string) string {
	prompt, _ := RenderPrompt("entity_extraction", map[string]string{
		"Text": text,
	})
	return prompt
}

// BuildRelationshipExtractionPrompt builds a prompt for relationship extraction
func BuildRelationshipExtractionPrompt(entities []Entity, text string) string {
	// Format entities as a list
	entityList := make([]string, len(entities))
	for i, entity := range entities {
		entityList[i] = fmt.Sprintf("- %s (%s)", entity.Name, entity.Type)
	}
	entitiesStr := strings.Join(entityList, "\n")

	prompt, _ := RenderPrompt("relationship_extraction", map[string]string{
		"Entities": entitiesStr,
		"Text":     text,
	})
	return prompt
}

// BuildClassExtractionPrompt builds a prompt for class extraction
func BuildClassExtractionPrompt(text string) string {
	prompt, _ := RenderPrompt("class_extraction", map[string]string{
		"Text": text,
	})
	return prompt
}

// BuildCombinedExtractionPrompt builds a prompt for combined extraction
func BuildCombinedExtractionPrompt(text string) string {
	prompt, _ := RenderPrompt("combined_extraction", map[string]string{
		"Text": text,
	})
	return prompt
}

// FormatElementsAsText converts UDML elements into a formatted text string
func FormatElementsAsText(elements []Element) string {
	var sb strings.Builder

	for _, elem := range elements {
		sb.WriteString(fmt.Sprintf("## %s (%s)\n", elem.ElementType, elem.ElementID))
		sb.WriteString(elem.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// FormatElementsWithContext converts UDML elements with context into formatted text
// Includes element metadata and parent-child relationships
func FormatElementsWithContext(elements []Element) string {
	var sb strings.Builder

	// Build parent-child map
	childrenMap := make(map[string][]Element)
	for _, elem := range elements {
		if elem.ParentID != "" {
			childrenMap[elem.ParentID] = append(childrenMap[elem.ParentID], elem)
		}
	}

	// Format elements hierarchically
	var formatElement func(elem Element, indent int)
	formatElement = func(elem Element, indent int) {
		indentStr := strings.Repeat("  ", indent)
		sb.WriteString(fmt.Sprintf("%s### %s\n", indentStr, elem.ElementType))
		sb.WriteString(fmt.Sprintf("%sID: %s\n", indentStr, elem.ElementID))
		if elem.ParentID != "" {
			sb.WriteString(fmt.Sprintf("%sParent: %s\n", indentStr, elem.ParentID))
		}
		sb.WriteString(fmt.Sprintf("%sContent:\n%s%s\n\n", indentStr, indentStr, elem.Content))

		// Process children
		children := childrenMap[elem.ElementID]
		for _, child := range children {
			formatElement(child, indent+1)
		}
	}

	// Find root elements (no parent)
	for _, elem := range elements {
		if elem.ParentID == "" {
			formatElement(elem, 0)
		}
	}

	return sb.String()
}

// TruncateText truncates text to a maximum length while trying to preserve sentence boundaries
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Try to find sentence boundary
	truncated := text[:maxLength]
	lastPeriod := strings.LastIndex(truncated, ".")
	lastNewline := strings.LastIndex(truncated, "\n")

	cutoff := lastPeriod
	if lastNewline > cutoff {
		cutoff = lastNewline
	}

	if cutoff > maxLength/2 {
		return text[:cutoff+1]
	}

	// No good sentence boundary, just cut at word boundary
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLength/2 {
		return text[:lastSpace] + "..."
	}

	return truncated + "..."
}

// ChunkText splits text into chunks of approximately equal size
// Tries to preserve paragraph boundaries
func ChunkText(text string, chunkSize int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	chunks := []string{}
	paragraphs := strings.Split(text, "\n\n")

	currentChunk := ""
	for _, para := range paragraphs {
		// If adding this paragraph would exceed chunk size, start new chunk
		if len(currentChunk)+len(para) > chunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, currentChunk)
			currentChunk = para
		} else {
			if len(currentChunk) > 0 {
				currentChunk += "\n\n"
			}
			currentChunk += para
		}
	}

	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}
