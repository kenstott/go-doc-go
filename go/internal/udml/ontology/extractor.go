package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ContentResolver defines the interface for resolving content from content_location
type ContentResolver interface {
	ResolveContent(contentLocation map[string]interface{}, text bool) (string, error)
}

// RuleBasedExtractor applies ontology schema rules to extract entities and relationships
type RuleBasedExtractor struct {
	schema          *OntologySchema
	contentResolver ContentResolver
	idCounter       uint64
	// Performance optimization caches
	contentCache map[string]string      // Cache resolved content by element_id
	jsonDocCache map[string]interface{} // Cache reconstructed JSON documents by doc_id
	cacheMu      sync.RWMutex           // Mutex for thread-safe cache access
}

// Element represents a minimal UDML element for extraction
type Element struct {
	ElementID       string                 `json:"element_id"`
	ElementType     string                 `json:"element_type"`
	Content         string                 `json:"content"`          // May be empty - use ContentLocation instead
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"` // Pointer to actual content (HTML selector, etc.)
	ParentID        string                 `json:"parent_id"`
	ElementOrder    float64                `json:"element_order"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// NewRuleBasedExtractor creates a new rule-based extractor with the given schema and content resolver
func NewRuleBasedExtractor(schema *OntologySchema, contentResolver ContentResolver) *RuleBasedExtractor {
	return &RuleBasedExtractor{
		schema:          schema,
		contentResolver: contentResolver,
		contentCache:    make(map[string]string),
		jsonDocCache:    make(map[string]interface{}),
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
	entityMap := make(map[string]*Entity) // Deduplicate by <type>.<name> (case-insensitive)

	log.Printf("DEBUG: Starting entity extraction with %d entity mappings", len(e.schema.ElementEntityMappings))

	// Process each entity mapping in the schema
	for i, mapping := range e.schema.ElementEntityMappings {
		// Filter elements by type
		relevantElements := e.filterElementsByType(elements, mapping.ElementTypes)
		log.Printf("DEBUG: Mapping %d (%s) targeting %v: filtered %d/%d elements",
			i, mapping.EntityType, mapping.ElementTypes, len(relevantElements), len(elements))

		// Apply each extraction rule
		for j, rule := range mapping.ExtractionRules {
			log.Printf("DEBUG: Applying rule %d (type: %s) for entity %s", j, rule.Type, mapping.EntityType)
			extractedEntities, err := e.applyExtractionRule(ctx, mapping, rule, relevantElements)
			if err != nil {
				return nil, fmt.Errorf("failed to apply rule for %s: %w", mapping.EntityType, err)
			}
			log.Printf("DEBUG: Rule %d extracted %d entities", j, len(extractedEntities))

			// Deduplicate entities by <type>.<name> (case-insensitive) - keep highest confidence entity
			for _, entity := range extractedEntities {
				// Use <type>.<name> as deduplication key (case-insensitive)
				// Convert entity.Type from EntityType enum to string for the key
				typeStr := string(entity.Type)
				dedupKey := strings.ToLower(typeStr + "." + entity.Name)

				if existing, ok := entityMap[dedupKey]; ok {
					// Replace with higher confidence entity
					if entity.Confidence > existing.Confidence {
						// Keep all mentions from both
						entity.Mentions = append(entity.Mentions, existing.Mentions...)
						entityCopy := entity
						entityMap[dedupKey] = &entityCopy
					} else if entity.Confidence == existing.Confidence {
						// Same confidence - prefer the canonical form (title case or most common)
						if e.isMoreCanonical(entity.Name, existing.Name) {
							entity.Mentions = append(entity.Mentions, existing.Mentions...)
							entityCopy := entity
							entityMap[dedupKey] = &entityCopy
						} else {
							// Keep existing, merge mentions
							existing.Mentions = append(existing.Mentions, entity.Mentions...)
						}
					} else {
						// Keep existing, merge mentions
						existing.Mentions = append(existing.Mentions, entity.Mentions...)
					}
				} else {
					entityCopy := entity
					entityMap[dedupKey] = &entityCopy
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

		// Resolve content for instance_name extraction
		content := e.resolveContent(&elem)

		entity := Entity{
			ID:         e.generateID("ent"),
			Name:       e.extractInstanceName(content, rule, value),
			Type:       EntityType(mapping.EntityType),
			Domain:     mapping.Domain, // Domain ownership from mapping
			Confidence: mapping.Confidence,
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
		// Apply semantic filter at element level (before pattern matching to save compute)
		if rule.SemanticFilter != nil {
			if !e.checkSemanticFilter(&elem, rule.SemanticFilter) {
				continue // Element doesn't match semantic context
			}
		}

		// Resolve content before applying regex
		content := e.resolveContent(&elem)
		if content == "" {
			continue
		}

		// Find all matches in content
		matches := re.FindAllStringIndex(content, -1)
		matchStrings := re.FindAllString(content, -1)

		for i, match := range matches {
			entityName := matchStrings[i]

			entity := Entity{
				ID:         e.generateID("ent"),
				Name:       e.extractInstanceName(content, rule, entityName),
				Type:       EntityType(mapping.EntityType),
				Domain:     mapping.Domain, // Domain ownership from mapping
				Confidence: mapping.Confidence,
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
		// Apply semantic filter at element level
		if rule.SemanticFilter != nil {
			if !e.checkSemanticFilter(&elem, rule.SemanticFilter) {
				continue // Element doesn't match semantic context
			}
		}

		// Resolve content before applying keywords
		content := e.resolveContent(&elem)
		if content == "" {
			continue
		}

		for _, keyword := range rule.Keywords {
			// Use regex with word boundaries for exact word matching
			pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
			re, err := regexp.Compile("(?i)" + pattern) // Case-insensitive
			if err != nil {
				continue
			}

			// Find all matches
			matches := re.FindAllStringIndex(content, -1)
			if len(matches) == 0 {
				continue
			}

			// Extract the actual text for first match only (preserving case)
			match := matches[0]
			actualText := content[match[0]:match[1]]

			entity := Entity{
				ID:         e.generateID("ent"),
				Name:       e.extractInstanceName(content, rule, actualText),
				Type:       EntityType(mapping.EntityType),
				Domain:     mapping.Domain, // Domain ownership from mapping
				Confidence: mapping.Confidence,
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
		// Resolve content before applying similarity
		content := e.resolveContent(&elem)
		if content == "" {
			continue
		}

		// Calculate similarity between element content and reference text
		similarity := e.calculateTextSimilarity(content, rule.ReferenceText)

		// Check if similarity meets threshold
		if similarity >= rule.SimilarityThreshold {
			// Extract the whole content as entity when similar
			// Text similarity is binary: if similarity >= threshold -> TRUE, use mapping confidence
			entity := Entity{
				ID:         e.generateID("ent"),
				Name:       e.extractInstanceName(content, rule, elem.ContentPreview),
				Type:       EntityType(mapping.EntityType),
				Domain:     mapping.Domain, // Domain ownership from mapping
				Confidence: mapping.Confidence, // Use mapping confidence (context quality)
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
						Text:      content,
						StartPos:  0,
						EndPos:    len(content),
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
		// Apply semantic filter at element level
		if rule.SemanticFilter != nil {
			if !e.checkSemanticFilter(&elem, rule.SemanticFilter) {
				continue // Element doesn't match semantic context
			}
		}

		// Resolve content and try to parse as JSON
		content := e.resolveContent(&elem)

		var jsonData interface{}
		if content != "" {
			if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
				// If content is not JSON, try metadata
				jsonData = elem.Metadata
			}
		} else {
			// No content, try metadata
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
				Name:       e.extractInstanceName(content, rule, resultStr),
				Type:       EntityType(mapping.EntityType),
				Domain:     mapping.Domain, // Domain ownership from mapping
				Confidence: mapping.Confidence,
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
				// Use rule confidence (pattern reliability)
				// Entity confidence is separate and already set
				// Domain inherited from source entity (consumer domain owns enrichment relationship)
				// source = entity being enriched (consumer), target = entity providing data (producer)
				rel := Relationship{
					ID:         e.generateID("rel"),
					Type:       rule.RelationshipType,
					Domain:     source.Domain, // Consumer domain owns enrichment (source = entity being enriched)
					SourceID:   source.ID,
					TargetID:   target.ID,
					Confidence: rule.Confidence, // Pattern reliability
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

// resolveContent resolves element content from content_location or returns existing content
func (e *RuleBasedExtractor) resolveContent(elem *Element) string {
	// If content is already populated, use it
	if elem.Content != "" {
		return elem.Content
	}

	// Check element-level cache first (thread-safe read)
	e.cacheMu.RLock()
	if cached, ok := e.contentCache[elem.ElementID]; ok {
		e.cacheMu.RUnlock()
		return cached
	}
	e.cacheMu.RUnlock()

	// If no content_location, cache empty result and return
	if elem.ContentLocation == nil || len(elem.ContentLocation) == 0 {
		log.Printf("DEBUG: Element %s has no content_location", elem.ElementID)
		e.cacheMu.Lock()
		e.contentCache[elem.ElementID] = ""
		e.cacheMu.Unlock()
		return ""
	}

	// If no content resolver configured, cache empty result and return
	if e.contentResolver == nil {
		log.Printf("WARNING: No content resolver configured for element %s, cannot resolve content", elem.ElementID)
		e.cacheMu.Lock()
		e.contentCache[elem.ElementID] = ""
		e.cacheMu.Unlock()
		return ""
	}

	// Resolve content from content_location (ContentResolver has its own cache internally)
	log.Printf("DEBUG: Resolving content for element %s from content_location: %v", elem.ElementID, elem.ContentLocation)
	content, err := e.contentResolver.ResolveContent(elem.ContentLocation, true)
	if err != nil {
		log.Printf("WARNING: Failed to resolve content for element %s: %v", elem.ElementID, err)
		e.cacheMu.Lock()
		e.contentCache[elem.ElementID] = ""
		e.cacheMu.Unlock()
		return ""
	}

	// Cache the resolved content (thread-safe write)
	e.cacheMu.Lock()
	e.contentCache[elem.ElementID] = content
	e.cacheMu.Unlock()

	previewLen := 50
	if len(content) < previewLen {
		previewLen = len(content)
	}
	log.Printf("DEBUG: Resolved content for element %s (length: %d): %s", elem.ElementID, len(content), content[:previewLen])
	return content
}

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

// normalizeEntityType converts entity type to standard enum if recognized, otherwise uses custom type
func (e *RuleBasedExtractor) normalizeEntityType(typeStr string) EntityType {
	normalized := strings.ToLower(strings.TrimSpace(typeStr))

	// Check if it's a standard/universal entity type
	switch normalized {
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
		// Use the original string as-is for domain-specific types
		// This preserves types like "medical procedure", "disease", "treatment"
		return EntityType(typeStr)
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
// Supports: basic paths ($.key.subkey[0]), filters ($[?(@.price < 10)]), recursive descent ($..book)
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
			// Handle recursive descent (e.g., ..book)
			if strings.HasPrefix(segment, "..") {
				key := strings.TrimPrefix(segment, "..")
				next = append(next, e.recursiveDescent(item, key)...)
				continue
			}

			// Handle array filter (e.g., [?(@.price < 10)])
			if strings.HasPrefix(segment, "[?(") && strings.HasSuffix(segment, ")]") {
				filterExpr := strings.TrimSuffix(strings.TrimPrefix(segment, "[?("), ")]")
				if arr, ok := item.([]interface{}); ok {
					for _, arrItem := range arr {
						if e.evaluateFilter(arrItem, filterExpr) {
							next = append(next, arrItem)
						}
					}
				}
				continue
			}

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

// recursiveDescent recursively searches for a key in nested structures
func (e *RuleBasedExtractor) recursiveDescent(data interface{}, key string) []interface{} {
	var results []interface{}

	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this level has the key
		if val, exists := v[key]; exists {
			results = append(results, val)
		}
		// Recurse into all nested values
		for _, nestedVal := range v {
			results = append(results, e.recursiveDescent(nestedVal, key)...)
		}
	case []interface{}:
		// Recurse into array elements
		for _, item := range v {
			results = append(results, e.recursiveDescent(item, key)...)
		}
	}

	return results
}

// evaluateFilter evaluates a filter expression (e.g., "@.price < 10")
func (e *RuleBasedExtractor) evaluateFilter(item interface{}, filterExpr string) bool {
	// Parse filter expression like "@.price < 10" or "@.name == 'John'"
	filterExpr = strings.TrimSpace(filterExpr)

	// Handle comparison operators
	operators := []string{"==", "!=", ">=", "<=", ">", "<"}
	for _, op := range operators {
		if strings.Contains(filterExpr, op) {
			parts := strings.SplitN(filterExpr, op, 2)
			if len(parts) != 2 {
				return false
			}

			leftExpr := strings.TrimSpace(parts[0])
			rightExpr := strings.TrimSpace(parts[1])

			// Evaluate left side (should be @.field)
			leftValue := e.evaluateFilterExpression(item, leftExpr)

			// Evaluate right side (can be literal or @.field)
			var rightValue interface{}
			if strings.HasPrefix(rightExpr, "@.") {
				rightValue = e.evaluateFilterExpression(item, rightExpr)
			} else {
				// Parse literal value (number, string, bool)
				rightValue = e.parseLiteral(rightExpr)
			}

			// Compare values
			return e.compareValues(leftValue, rightValue, op)
		}
	}

	return false
}

// evaluateFilterExpression evaluates an expression like "@.price" against an item
func (e *RuleBasedExtractor) evaluateFilterExpression(item interface{}, expr string) interface{} {
	// Remove @ prefix
	expr = strings.TrimPrefix(expr, "@")
	expr = strings.TrimPrefix(expr, ".")

	if expr == "" {
		return item
	}

	// Navigate to the field
	parts := strings.Split(expr, ".")
	current := item

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

// parseLiteral parses a string literal into its actual type
func (e *RuleBasedExtractor) parseLiteral(s string) interface{} {
	s = strings.TrimSpace(s)

	// Try parsing as number
	if num, err := fmt.Sscanf(s, "%f", new(float64)); err == nil && num == 1 {
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}

	// Try parsing as bool
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Try parsing as string (remove quotes)
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) ||
		(strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) {
		return s[1 : len(s)-1]
	}

	return s
}

// compareValues compares two values using the given operator
func (e *RuleBasedExtractor) compareValues(left, right interface{}, op string) bool {
	if left == nil || right == nil {
		return false
	}

	// Try numeric comparison
	leftNum, leftIsNum := e.toFloat64(left)
	rightNum, rightIsNum := e.toFloat64(right)

	if leftIsNum && rightIsNum {
		switch op {
		case "==":
			return leftNum == rightNum
		case "!=":
			return leftNum != rightNum
		case ">":
			return leftNum > rightNum
		case "<":
			return leftNum < rightNum
		case ">=":
			return leftNum >= rightNum
		case "<=":
			return leftNum <= rightNum
		}
	}

	// Fall back to string comparison
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	switch op {
	case "==":
		return leftStr == rightStr
	case "!=":
		return leftStr != rightStr
	case ">":
		return leftStr > rightStr
	case "<":
		return leftStr < rightStr
	case ">=":
		return leftStr >= rightStr
	case "<=":
		return leftStr <= rightStr
	}

	return false
}

// toFloat64 attempts to convert a value to float64
func (e *RuleBasedExtractor) toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
			return f, true
		}
	}
	return 0, false
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

// isMoreCanonical determines if name1 is more canonical than name2
// Prefers title case, then capitalized words, then lowercase
func (e *RuleBasedExtractor) isMoreCanonical(name1, name2 string) bool {
	// If one is empty, the other is more canonical
	if name1 == "" {
		return false
	}
	if name2 == "" {
		return true
	}

	// Count uppercase letters at the start of each word
	words1 := strings.Fields(name1)
	words2 := strings.Fields(name2)

	// Count capitalized words (Title Case preference)
	capitalized1 := 0
	capitalized2 := 0
	allUpper1 := 0
	allUpper2 := 0
	allLower1 := 0
	allLower2 := 0

	for _, word := range words1 {
		if len(word) == 0 {
			continue
		}
		if word == strings.ToUpper(word) && word != strings.ToLower(word) {
			allUpper1++
		} else if word[0] >= 'A' && word[0] <= 'Z' {
			capitalized1++
		} else {
			allLower1++
		}
	}

	for _, word := range words2 {
		if len(word) == 0 {
			continue
		}
		if word == strings.ToUpper(word) && word != strings.ToLower(word) {
			allUpper2++
		} else if word[0] >= 'A' && word[0] <= 'Z' {
			capitalized2++
		} else {
			allLower2++
		}
	}

	// Preference order:
	// 1. Title Case (most words capitalized) over all lowercase
	// 2. Fewer ALL CAPS words (ALL CAPS is less canonical)
	// 3. More capitalized words over lowercase

	// Avoid all caps (all caps is least canonical)
	if allUpper1 < allUpper2 {
		return true
	}
	if allUpper1 > allUpper2 {
		return false
	}

	// Prefer more capitalized words (Title Case)
	if capitalized1 > capitalized2 {
		return true
	}
	if capitalized1 < capitalized2 {
		return false
	}

	// All else equal, prefer the one that appears first alphabetically
	// This provides deterministic behavior
	return name1 < name2
}

// extractInstanceName extracts entity instance name using optional instance_name regex or default
// If instance_name regex is specified and matches, uses the (?P<name>...) capture group
// Otherwise falls back to the provided defaultName
func (e *RuleBasedExtractor) extractInstanceName(content string, rule ExtractionRule, defaultName string) string {
	// If instance_name regex not specified, use default
	if rule.InstanceName == "" {
		return defaultName
	}

	// Compile instance_name regex
	re, err := regexp.Compile(rule.InstanceName)
	if err != nil {
		log.Printf("WARNING: Invalid instance_name regex '%s': %v - falling back to default", rule.InstanceName, err)
		return defaultName
	}

	// Execute regex on content
	match := re.FindStringSubmatch(content)
	if match == nil {
		log.Printf("DEBUG: instance_name regex did not match content - falling back to default")
		return defaultName
	}

	// Look for named capture group "name"
	for i, groupName := range re.SubexpNames() {
		if groupName == "name" && i < len(match) {
			extracted := strings.TrimSpace(match[i])
			if extracted != "" {
				return extracted
			}
			log.Printf("DEBUG: instance_name regex 'name' capture group is empty - falling back to default")
			return defaultName
		}
	}

	// No "name" capture group found
	log.Printf("DEBUG: instance_name regex matched but no 'name' capture group found - falling back to default")
	return defaultName
}

// Embedding cache for concept strings (global cache with mutex)
var conceptEmbeddingCache = make(map[string][]float64)
var conceptEmbeddingMutex sync.RWMutex

// checkSemanticFilter validates element against semantic filter using element-level embeddings
func (e *RuleBasedExtractor) checkSemanticFilter(elem *Element, filter *SemanticFilter) bool {
	// Get element embedding
	elementEmbedding := e.getElementEmbedding(elem)
	if elementEmbedding == nil {
		// Graceful degradation: no embeddings available, accept by default
		log.Printf("DEBUG: No embedding for element %s, skipping semantic filter (accepting)", elem.ElementID)
		return true
	}

	// Check similarity against each reference concept
	for _, concept := range filter.ReferenceConcepts {
		conceptEmbedding := e.getConceptEmbedding(concept)
		if conceptEmbedding == nil {
			continue
		}

		similarity := e.cosineSimilarity(elementEmbedding, conceptEmbedding)
		log.Printf("DEBUG: Semantic similarity for '%s': %.3f (threshold: %.3f)",
			concept, similarity, filter.SimilarityThreshold)

		if similarity >= filter.SimilarityThreshold {
			return true // Found matching semantic context
		}
	}

	return false // No concept matched threshold
}

// getElementEmbedding retrieves embedding vector for an element
func (e *RuleBasedExtractor) getElementEmbedding(elem *Element) []float64 {
	// Try to get embedding from content resolver
	if e.contentResolver == nil {
		return nil
	}

	// Check if resolver supports embeddings (type assertion)
	type EmbeddingResolver interface {
		GetEmbedding(contentLocation map[string]interface{}) ([]float64, error)
	}

	if embResolver, ok := e.contentResolver.(EmbeddingResolver); ok {
		embedding, err := embResolver.GetEmbedding(elem.ContentLocation)
		if err == nil && len(embedding) > 0 {
			return embedding
		}
	}

	return nil
}

// getConceptEmbedding computes/retrieves cached embedding for a concept string
func (e *RuleBasedExtractor) getConceptEmbedding(concept string) []float64 {
	// Check cache first
	conceptEmbeddingMutex.RLock()
	if cached, ok := conceptEmbeddingCache[concept]; ok {
		conceptEmbeddingMutex.RUnlock()
		return cached
	}
	conceptEmbeddingMutex.RUnlock()

	// Compute embedding for concept
	// TODO: Integrate with LLM client for embedding generation
	// For now, return nil (graceful degradation)
	log.Printf("WARNING: Concept embedding not yet implemented for: %s", concept)
	return nil
}
