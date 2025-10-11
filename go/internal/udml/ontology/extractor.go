package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

// RuleBasedExtractor applies ontology schema rules to extract entities and relationships
type RuleBasedExtractor struct {
	schema   *OntologySchema
	idCounter uint64
}

// Element represents a minimal UDML element for extraction
type Element struct {
	ElementID      string                 `json:"element_id"`
	ElementType    string                 `json:"element_type"`
	Content        string                 `json:"content"`
	ContentPreview string                 `json:"content_preview"`
	ParentID       string                 `json:"parent_id"`
	ElementOrder   float64                `json:"element_order"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewRuleBasedExtractor creates a new rule-based extractor with the given schema
func NewRuleBasedExtractor(schema *OntologySchema) *RuleBasedExtractor {
	return &RuleBasedExtractor{
		schema: schema,
	}
}

// ExtractFromElements extracts entities and relationships from UDML elements
func (e *RuleBasedExtractor) ExtractFromElements(ctx context.Context, docID string, elements []Element) (*Ontology, error) {
	startTime := time.Now()

	ontology := &Ontology{
		ID:            e.generateID("ont"),
		DocID:         docID,
		Name:          e.schema.Name,
		Description:   fmt.Sprintf("Extracted from %s using %s schema", docID, e.schema.Name),
		Version:       e.schema.Version,
		Entities:      []Entity{},
		Relationships: []Relationship{},
		Classes:       []Class{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Extract entities using schema rules
	entities, err := e.extractEntities(ctx, elements)
	if err != nil {
		return nil, fmt.Errorf("entity extraction failed: %w", err)
	}
	ontology.Entities = entities

	// Extract relationships using schema rules
	relationships, err := e.extractRelationships(ctx, elements, entities)
	if err != nil {
		return nil, fmt.Errorf("relationship extraction failed: %w", err)
	}
	ontology.Relationships = relationships

	// Set metadata
	ontology.Metadata = OntologyMeta{
		ExtractorType:    "rule_based",
		ExtractorVersion: "1.0.0",
		ExtractionTime:   time.Since(startTime),
		ElementCount:     len(elements),
		Confidence:       e.calculateOverallConfidence(ontology),
		CustomFields:     make(map[string]interface{}),
	}

	// Validate ontology
	if err := ontology.Validate(); err != nil {
		return nil, fmt.Errorf("ontology validation failed: %w", err)
	}

	return ontology, nil
}

// extractEntities applies entity extraction rules to elements
func (e *RuleBasedExtractor) extractEntities(ctx context.Context, elements []Element) ([]Entity, error) {
	var entities []Entity
	entityMap := make(map[string]*Entity) // Deduplicate by name

	// Process each entity mapping in the schema
	for _, mapping := range e.schema.ElementEntityMappings {
		// Filter elements by type
		relevantElements := e.filterElementsByType(elements, mapping.ElementTypes)

		// Apply each extraction rule
		for _, rule := range mapping.ExtractionRules {
			extractedEntities, err := e.applyExtractionRule(ctx, mapping, rule, relevantElements)
			if err != nil {
				return nil, fmt.Errorf("failed to apply rule for %s: %w", mapping.EntityType, err)
			}

			// Deduplicate entities by name
			for _, entity := range extractedEntities {
				if existing, ok := entityMap[entity.Name]; ok {
					// Update confidence if higher
					if entity.Confidence > existing.Confidence {
						existing.Confidence = entity.Confidence
					}
					// Merge mentions
					existing.Mentions = append(existing.Mentions, entity.Mentions...)
				} else {
					entityMap[entity.Name] = &entity
				}
			}
		}
	}

	// Convert map to slice
	for _, entity := range entityMap {
		entities = append(entities, *entity)
	}

	return entities, nil
}

// applyExtractionRule applies a single extraction rule to elements
func (e *RuleBasedExtractor) applyExtractionRule(ctx context.Context, mapping ElementEntityMapping, rule ExtractionRule, elements []Element) ([]Entity, error) {
	var entities []Entity

	switch rule.Type {
	case RuleTypeMetadata:
		entities = e.extractFromMetadata(mapping, rule, elements)
	case RuleTypeRegex:
		entities = e.extractFromRegex(mapping, rule, elements)
	case RuleTypeKeyword:
		entities = e.extractFromKeywords(mapping, rule, elements)
	case RuleTypeSimilarity:
		entities = e.extractFromSimilarity(mapping, rule, elements)
	case RuleTypeJSONPath:
		entities = e.extractFromJSONPath(mapping, rule, elements)
	default:
		return nil, fmt.Errorf("unknown rule type: %s", rule.Type)
	}

	return entities, nil
}

// extractFromMetadata extracts entities from element metadata
func (e *RuleBasedExtractor) extractFromMetadata(mapping ElementEntityMapping, rule ExtractionRule, elements []Element) []Entity {
	var entities []Entity

	for _, elem := range elements {
		// Navigate metadata path (e.g., "metadata.speaker")
		value := e.getMetadataValue(elem.Metadata, rule.FieldPath)
		if value == "" {
			continue
		}

		entity := Entity{
			ID:         e.generateID("ent"),
			Name:       value,
			Type:       e.parseEntityType(mapping.EntityType),
			Confidence: rule.Confidence,
			Attributes: map[string]interface{}{
				"source":     "metadata",
				"field_path": rule.FieldPath,
			},
			ElementID: elem.ElementID,
			Mentions: []Mention{
				{
					ElementID: elem.ElementID,
					Text:      value,
					StartPos:  0,
					EndPos:    len(value),
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		entities = append(entities, entity)
	}

	return entities
}

// extractFromRegex extracts entities using regex patterns
func (e *RuleBasedExtractor) extractFromRegex(mapping ElementEntityMapping, rule ExtractionRule, elements []Element) []Entity {
	var entities []Entity

	// Compile regex
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return entities // Skip invalid regex
	}

	for _, elem := range elements {
		// Find all matches in content
		matches := re.FindAllStringIndex(elem.Content, -1)
		matchStrings := re.FindAllString(elem.Content, -1)

		for i, match := range matches {
			entityName := matchStrings[i]

			entity := Entity{
				ID:         e.generateID("ent"),
				Name:       entityName,
				Type:       e.parseEntityType(mapping.EntityType),
				Confidence: rule.Confidence,
				Attributes: map[string]interface{}{
					"source":  "regex",
					"pattern": rule.Pattern,
				},
				ElementID: elem.ElementID,
				Mentions: []Mention{
					{
						ElementID: elem.ElementID,
						Text:      entityName,
						StartPos:  match[0],
						EndPos:    match[1],
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			entities = append(entities, entity)
		}
	}

	return entities
}

// extractFromKeywords extracts entities by keyword matching
func (e *RuleBasedExtractor) extractFromKeywords(mapping ElementEntityMapping, rule ExtractionRule, elements []Element) []Entity {
	var entities []Entity

	for _, elem := range elements {
		for _, keyword := range rule.Keywords {
			// Use regex with word boundaries for exact word matching
			pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
			re, err := regexp.Compile("(?i)" + pattern) // Case-insensitive
			if err != nil {
				continue
			}

			// Find all matches
			matches := re.FindAllStringIndex(elem.Content, -1)
			if len(matches) == 0 {
				continue
			}

			// Extract the actual text for first match only (preserving case)
			match := matches[0]
			actualText := elem.Content[match[0]:match[1]]

			entity := Entity{
				ID:         e.generateID("ent"),
				Name:       actualText,
				Type:       e.parseEntityType(mapping.EntityType),
				Confidence: rule.Confidence,
				Attributes: map[string]interface{}{
					"source":   "keyword",
					"keywords": rule.Keywords,
				},
				ElementID: elem.ElementID,
				Mentions: []Mention{
					{
						ElementID: elem.ElementID,
						Text:      actualText,
						StartPos:  match[0],
						EndPos:    match[1],
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			entities = append(entities, entity)
		}
	}

	return entities
}

// extractFromSimilarity extracts entities based on text similarity to a reference
func (e *RuleBasedExtractor) extractFromSimilarity(mapping ElementEntityMapping, rule ExtractionRule, elements []Element) []Entity {
	var entities []Entity

	if rule.ReferenceText == "" {
		return entities // No reference text provided
	}

	for _, elem := range elements {
		// Calculate similarity between element content and reference text
		similarity := e.calculateTextSimilarity(elem.Content, rule.ReferenceText)

		// Check if similarity meets threshold
		if similarity >= rule.SimilarityThreshold {
			// Extract the whole content as entity when similar
			entity := Entity{
				ID:         e.generateID("ent"),
				Name:       elem.ContentPreview,
				Type:       e.parseEntityType(mapping.EntityType),
				Confidence: similarity * rule.Confidence, // Combine similarity with rule confidence
				Attributes: map[string]interface{}{
					"source":               "similarity",
					"reference_text":       rule.ReferenceText,
					"similarity_score":     similarity,
					"similarity_threshold": rule.SimilarityThreshold,
				},
				ElementID: elem.ElementID,
				Mentions: []Mention{
					{
						ElementID: elem.ElementID,
						Text:      elem.Content,
						StartPos:  0,
						EndPos:    len(elem.Content),
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			entities = append(entities, entity)
		}
	}

	return entities
}

// extractFromJSONPath extracts entities using JSONPath expressions
func (e *RuleBasedExtractor) extractFromJSONPath(mapping ElementEntityMapping, rule ExtractionRule, elements []Element) []Entity {
	var entities []Entity

	if rule.JSONPathExpr == "" {
		return entities // No JSONPath expression provided
	}

	for _, elem := range elements {
		// Try to parse element content as JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(elem.Content), &jsonData); err != nil {
			// If content is not JSON, try metadata
			jsonData = elem.Metadata
		}

		// Evaluate JSONPath expression
		results := e.evaluateJSONPath(jsonData, rule.JSONPathExpr)

		// Create entity for each result
		for _, result := range results {
			// Convert result to string
			var resultStr string
			if str, ok := result.(string); ok {
				resultStr = str
			} else {
				resultStr = fmt.Sprintf("%v", result)
			}

			if resultStr == "" {
				continue
			}

			entity := Entity{
				ID:         e.generateID("ent"),
				Name:       resultStr,
				Type:       e.parseEntityType(mapping.EntityType),
				Confidence: rule.Confidence,
				Attributes: map[string]interface{}{
					"source":        "jsonpath",
					"jsonpath_expr": rule.JSONPathExpr,
				},
				ElementID: elem.ElementID,
				Mentions: []Mention{
					{
						ElementID: elem.ElementID,
						Text:      resultStr,
						StartPos:  0,
						EndPos:    len(resultStr),
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			entities = append(entities, entity)
		}
	}

	return entities
}

// extractRelationships extracts relationships based on schema rules and extracted entities
func (e *RuleBasedExtractor) extractRelationships(ctx context.Context, elements []Element, entities []Entity) ([]Relationship, error) {
	var relationships []Relationship

	// Build entity index by type and element
	entityIndex := e.buildEntityIndex(entities)

	// Apply each relationship rule
	for _, rule := range e.schema.EntityRelationshipRules {
		rels := e.applyRelationshipRule(rule, entityIndex, elements)
		relationships = append(relationships, rels...)
	}

	return relationships, nil
}

// applyRelationshipRule applies a relationship extraction rule
func (e *RuleBasedExtractor) applyRelationshipRule(rule EntityRelationshipRule, entityIndex map[string][]Entity, elements []Element) []Relationship {
	var relationships []Relationship

	// Get source and target entities
	sourceEntities := entityIndex[rule.SourceEntityType]
	targetEntities := entityIndex[rule.TargetEntityType]

	// For each source-target pair in the same or nearby elements, create a relationship
	for _, source := range sourceEntities {
		for _, target := range targetEntities {
			// Check if entities are in the same element or adjacent elements
			if e.areEntitiesRelated(source, target) {
				// Check confidence threshold
				avgConfidence := (source.Confidence + target.Confidence) / 2
				if avgConfidence < rule.ConfidenceThreshold {
					continue
				}

				rel := Relationship{
					ID:         e.generateID("rel"),
					Type:       rule.RelationshipType,
					SourceID:   source.ID,
					TargetID:   target.ID,
					Confidence: avgConfidence,
					Attributes: map[string]interface{}{
						"rule": rule.Name,
					},
					ElementID: source.ElementID,
					Evidence:  fmt.Sprintf("%s and %s appear in element %s", source.Name, target.Name, source.ElementID),
					CreatedAt: time.Now(),
				}

				relationships = append(relationships, rel)
			}
		}
	}

	return relationships
}

// Helper functions

func (e *RuleBasedExtractor) filterElementsByType(elements []Element, elementTypes []string) []Element {
	if len(elementTypes) == 0 {
		return elements
	}

	var filtered []Element
	typeSet := make(map[string]bool)
	for _, t := range elementTypes {
		typeSet[t] = true
	}

	for _, elem := range elements {
		if typeSet[elem.ElementType] {
			filtered = append(filtered, elem)
		}
	}

	return filtered
}

func (e *RuleBasedExtractor) getMetadataValue(metadata map[string]interface{}, path string) string {
	// Handle simple paths like "speaker" and nested paths like "metadata.speaker"
	parts := strings.Split(path, ".")

	current := metadata
	for i, part := range parts {
		// Skip "metadata" prefix if present
		if i == 0 && part == "metadata" {
			continue
		}

		value, ok := current[part]
		if !ok {
			return ""
		}

		// If last part, convert to string
		if i == len(parts)-1 {
			if str, ok := value.(string); ok {
				return str
			}
			return fmt.Sprintf("%v", value)
		}

		// Otherwise, should be a map
		if nested, ok := value.(map[string]interface{}); ok {
			current = nested
		} else {
			return ""
		}
	}

	return ""
}

func (e *RuleBasedExtractor) parseEntityType(typeStr string) EntityType {
	typeStr = strings.ToLower(strings.TrimSpace(typeStr))
	switch typeStr {
	case "person":
		return EntityTypePerson
	case "organization":
		return EntityTypeOrganization
	case "location":
		return EntityTypeLocation
	case "date":
		return EntityTypeDate
	case "event":
		return EntityTypeEvent
	case "concept":
		return EntityTypeConcept
	case "product":
		return EntityTypeProduct
	case "technology":
		return EntityTypeTechnology
	default:
		return EntityTypeCustom
	}
}

func (e *RuleBasedExtractor) buildEntityIndex(entities []Entity) map[string][]Entity {
	index := make(map[string][]Entity)
	for _, entity := range entities {
		typeStr := string(entity.Type)
		index[typeStr] = append(index[typeStr], entity)
	}
	return index
}

func (e *RuleBasedExtractor) areEntitiesRelated(source, target Entity) bool {
	// Same element
	if source.ElementID == target.ElementID {
		return true
	}

	// TODO: Could enhance to check adjacent elements, document proximity, etc.
	return false
}

func (e *RuleBasedExtractor) calculateOverallConfidence(ontology *Ontology) float64 {
	if len(ontology.Entities) == 0 {
		return 0.0
	}

	total := 0.0
	count := 0

	for _, entity := range ontology.Entities {
		total += entity.Confidence
		count++
	}

	for _, rel := range ontology.Relationships {
		total += rel.Confidence
		count++
	}

	if count == 0 {
		return 0.0
	}

	return total / float64(count)
}

func (e *RuleBasedExtractor) generateID(prefix string) string {
	count := atomic.AddUint64(&e.idCounter, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano()/1000000, count)
}

// calculateTextSimilarity calculates Jaccard similarity between two texts
// Returns a value between 0.0 (no similarity) and 1.0 (identical)
func (e *RuleBasedExtractor) calculateTextSimilarity(text1, text2 string) float64 {
	// Normalize and tokenize
	words1 := e.tokenizeText(strings.ToLower(text1))
	words2 := e.tokenizeText(strings.ToLower(text2))

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0 // Both empty = identical
	}

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0 // One empty = no similarity
	}

	// Convert to sets
	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}

	set2 := make(map[string]bool)
	for _, word := range words2 {
		set2[word] = true
	}

	// Calculate intersection
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}

	// Calculate union
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	// Jaccard similarity = intersection / union
	return float64(intersection) / float64(union)
}

// tokenizeText splits text into words
func (e *RuleBasedExtractor) tokenizeText(text string) []string {
	// Simple word-based tokenization
	// Remove punctuation and split on whitespace
	re := regexp.MustCompile(`[^\w\s]+`)
	cleaned := re.ReplaceAllString(text, " ")

	words := strings.Fields(cleaned)

	// Filter out very short words
	var filtered []string
	for _, word := range words {
		if len(word) > 1 {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// evaluateJSONPath evaluates a JSONPath expression against JSON data
// This is a simplified JSONPath implementation supporting basic paths like "$.key.subkey[0]"
func (e *RuleBasedExtractor) evaluateJSONPath(data interface{}, path string) []interface{} {
	// Remove leading $. if present
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	if path == "" {
		return []interface{}{data}
	}

	// Split path into segments
	segments := e.parseJSONPathSegments(path)

	// Navigate through the path
	current := []interface{}{data}

	for _, segment := range segments {
		var next []interface{}

		for _, item := range current {
			// Handle array index (e.g., [0])
			if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
				indexStr := strings.Trim(segment, "[]")
				if indexStr == "*" {
					// Wildcard - return all array elements
					if arr, ok := item.([]interface{}); ok {
						next = append(next, arr...)
					}
				} else {
					// Specific index
					var index int
					fmt.Sscanf(indexStr, "%d", &index)
					if arr, ok := item.([]interface{}); ok && index >= 0 && index < len(arr) {
						next = append(next, arr[index])
					}
				}
				continue
			}

			// Handle object key
			if m, ok := item.(map[string]interface{}); ok {
				if val, exists := m[segment]; exists {
					next = append(next, val)
				}
			}
		}

		current = next
	}

	return current
}

// parseJSONPathSegments parses a JSONPath expression into segments
func (e *RuleBasedExtractor) parseJSONPathSegments(path string) []string {
	var segments []string
	var currentSegment strings.Builder
	inBracket := false

	for i := 0; i < len(path); i++ {
		ch := path[i]

		if ch == '[' {
			if currentSegment.Len() > 0 {
				segments = append(segments, currentSegment.String())
				currentSegment.Reset()
			}
			inBracket = true
			currentSegment.WriteByte(ch)
		} else if ch == ']' {
			currentSegment.WriteByte(ch)
			segments = append(segments, currentSegment.String())
			currentSegment.Reset()
			inBracket = false
		} else if ch == '.' && !inBracket {
			if currentSegment.Len() > 0 {
				segments = append(segments, currentSegment.String())
				currentSegment.Reset()
			}
		} else {
			currentSegment.WriteByte(ch)
		}
	}

	if currentSegment.Len() > 0 {
		segments = append(segments, currentSegment.String())
	}

	return segments
}

// cosineSimilarity calculates cosine similarity between two texts
// This could be used for more advanced similarity matching with embeddings
func (e *RuleBasedExtractor) cosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64
	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}
