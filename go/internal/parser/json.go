package parser

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kennethstott/go-doc-go/internal/temporal"
	"github.com/oliveagle/jsonpath"
)

// ElementType represents the type of JSON element
type JSONElementType string

const (
	JSONElementTypeRoot      JSONElementType = "root"
	JSONElementTypeObject    JSONElementType = "json_object"
	JSONElementTypeArray     JSONElementType = "json_array"
	JSONElementTypeField     JSONElementType = "json_field"
	JSONElementTypeItem      JSONElementType = "json_item"
)

// JSONElement represents a parsed JSON element
type JSONElement struct {
	ElementID       string                 `json:"element_id"`
	DocID          string                 `json:"doc_id"`
	ElementType     JSONElementType        `json:"element_type"`
	ParentID        string                 `json:"parent_id,omitempty"`
	ContentPreview  string                 `json:"content_preview"`
	ContentLocation map[string]interface{} `json:"content_location"`
	ContentHash     string                 `json:"content_hash"`
	ElementOrder    int                    `json:"element_order"`
	DocumentOrder   int                    `json:"document_position"`
	Metadata        map[string]interface{} `json:"metadata"`
	Text            string                 `json:"text,omitempty"`
	Content         string                 `json:"content,omitempty"`
	TemporalValue   interface{}            `json:"temporal_value,omitempty"`
}

// JSONLink represents an extracted link
type JSONLink struct {
	SourceID   string `json:"source_id"`
	LinkText   string `json:"link_text"`
	LinkTarget string `json:"link_target"`
	LinkType   string `json:"link_type"`
}

// JSONRelationship represents a relationship between elements
type JSONRelationship struct {
	RelationshipID   string                 `json:"relationship_id"`
	SourceElementID  string                 `json:"source_element_id"`
	TargetElementID  string                 `json:"target_element_id"`
	RelationshipType string                 `json:"relationship_type"`
	Confidence       float64                `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// JSONParseRequest represents the input for JSON parsing
type JSONParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// JSONParseResponse represents the output of JSON parsing
type JSONParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []JSONElement          `json:"elements"`
	Links         []JSONLink             `json:"links"`
	Relationships []JSONRelationship     `json:"relationships"`
	Dates         map[string]interface{} `json:"dates,omitempty"`
}

// JSONParser handles JSON document parsing
type JSONParser struct {
	MaxContentPreview int
	IncludeFieldNames bool
	FlattenArrays     bool
	MaxDepth          int
	ExtractDates      bool
	EnableCaching     bool
}

// NewJSONParser creates a new JSON parser instance
func NewJSONParser() *JSONParser {
	return &JSONParser{
		MaxContentPreview: 100,
		IncludeFieldNames: true,
		FlattenArrays:     false,
		MaxDepth:          10,
		ExtractDates:      true,
		EnableCaching:     true,
	}
}

// Parse is the universal interface that converts JSON content to ParseResult
func (p *JSONParser) Parse(docID string, content interface{}) (*ParseResult, error) {
	// Handle different input types
	var jsonContent string
	switch v := content.(type) {
	case string:
		jsonContent = v
	case []byte:
		jsonContent = string(v)
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	// Create request structure for compatibility
	request := JSONParseRequest{
		ID:       docID,
		Content:  jsonContent,
		Metadata: make(map[string]interface{}),
	}

	// Parse using existing implementation
	response, err := p.parseJSON(request)
	if err != nil {
		return nil, err
	}

	// Convert to universal ParseResult
	return p.convertToParseResult(response), nil
}

// parseJSON parses a JSON document into structured elements (internal implementation)
func (p *JSONParser) parseJSON(request JSONParseRequest) (*JSONParseResponse, error) {
	// Parse JSON content
	var jsonData interface{}
	if err := json.Unmarshal([]byte(request.Content), &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Initialize response
	response := &JSONParseResponse{
		Document: map[string]interface{}{
			"doc_id":       request.ID,
			"doc_type":     "json",
			"source":       request.ID,
			"metadata":     request.Metadata,
			"content_hash": p.generateHash(request.Content),
		},
		Elements:      []JSONElement{},
		Links:         []JSONLink{},
		Relationships: []JSONRelationship{},
	}

	// Create root element
	rootElement := JSONElement{
		ElementID:       p.generateID("root_"),
		DocID:          request.ID,
		ElementType:     JSONElementTypeRoot,
		ContentPreview:  p.truncateContent(request.Content),
		ContentLocation: p.createContentLocation(request.ID, JSONElementTypeRoot, "$"),
		ContentHash:     p.generateHash(request.Content),
		ElementOrder:    0,
		DocumentOrder:   0,
		Metadata:        make(map[string]interface{}),
	}
	response.Elements = append(response.Elements, rootElement)

	// Parse JSON elements
	elementCounter := 1
	p.parseJSONElement(jsonData, request.ID, rootElement.ElementID, request.ID,
		&response.Elements, &response.Relationships, "$", 0, &elementCounter)

	// Extract links from the JSON content
	p.extractLinks(jsonData, &response.Links, rootElement.ElementID)

	return response, nil
}

// parseJSONElement recursively parses JSON elements
func (p *JSONParser) parseJSONElement(data interface{}, docID, parentID, sourceID string,
	elements *[]JSONElement, relationships *[]JSONRelationship,
	jsonPath string, depth int, counter *int) {

	// Prevent infinite recursion
	if depth > p.MaxDepth {
		return
	}

	switch v := data.(type) {
	case map[string]interface{}:
		// Handle JSON object
		objectID := p.generateID("obj_")
		objectPreview := p.getObjectPreview(v)

		objectElement := JSONElement{
			ElementID:       objectID,
			DocID:          docID,
			ElementType:     JSONElementTypeObject,
			ParentID:        parentID,
			ContentPreview:  objectPreview,
			ContentLocation: p.createContentLocation(sourceID, JSONElementTypeObject, jsonPath),
			ContentHash:     p.generateHash(p.serializeForHash(v)),
			ElementOrder:    *counter,
			DocumentOrder:   *counter,
			Metadata: map[string]interface{}{
				"fields":     p.getObjectKeys(v),
				"item_count": len(v),
				"json_path":  jsonPath,
			},
		}

		*elements = append(*elements, objectElement)
		*counter++

		// Create bidirectional parent-child relationships
		if parentID != "" {
			containsRel := JSONRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  parentID,
				TargetElementID:  objectID,
				RelationshipType: "contains",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			containedByRel := JSONRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  objectID,
				TargetElementID:  parentID,
				RelationshipType: "contained_by",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			*relationships = append(*relationships, containsRel, containedByRel)
		}

		// Parse object fields
		for key, value := range v {
			fieldPath := fmt.Sprintf("%s.%s", jsonPath, key)
			fieldID := p.generateID("field_")
			fieldPreview := p.getFieldPreview(key, value)

			// Check for temporal values if ExtractDates is enabled
			var temporalValue interface{}
			var temporalMetadata map[string]interface{}

			if p.ExtractDates {
				if strValue, ok := value.(string); ok {
					// Process field value for temporal content
					normalizedValue, tempMeta := temporal.ProcessFieldValue(key, strValue)
					if tempMeta != nil {
						temporalValue = tempMeta
						temporalMetadata = temporal.GenerateTemporalMetadata(strValue)
						// Update the preview with normalized value if temporal
						fieldPreview = p.truncateContent(fmt.Sprintf("%s: \"%s\"", key, normalizedValue))
					}
				}
			}

			fieldElement := JSONElement{
				ElementID:       fieldID,
				DocID:          docID,
				ElementType:     JSONElementTypeField,
				ParentID:        objectID,
				ContentPreview:  fieldPreview,
				ContentLocation: p.createContentLocation(sourceID, JSONElementTypeField, fieldPath),
				ContentHash:     p.generateHash(fmt.Sprintf("%s:%s", key, p.serializeForHash(value))),
				ElementOrder:    *counter,
				DocumentOrder:   *counter,
				Metadata: map[string]interface{}{
					"field_name": key,
					"json_path":  fieldPath,
					"value_type": p.getValueType(value),
				},
				TemporalValue:   temporalValue,
			}

			// Add temporal metadata if found
			if temporalMetadata != nil {
				for k, v := range temporalMetadata {
					fieldElement.Metadata[k] = v
				}
			}

			*elements = append(*elements, fieldElement)
			*counter++

			// Create bidirectional parent-child relationships
			containsRel := JSONRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  objectID,
				TargetElementID:  fieldID,
				RelationshipType: "contains",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			containedByRel := JSONRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  fieldID,
				TargetElementID:  objectID,
				RelationshipType: "contained_by",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			*relationships = append(*relationships, containsRel, containedByRel)

			// Recursively parse field value
			if p.isComplexType(value) {
				p.parseJSONElement(value, docID, fieldID, sourceID, elements, relationships,
					fieldPath, depth+1, counter)
			}
		}

	case []interface{}:
		// Handle JSON array
		arrayID := p.generateID("arr_")
		arrayPreview := p.getArrayPreview(v)

		arrayElement := JSONElement{
			ElementID:       arrayID,
			DocID:          docID,
			ElementType:     JSONElementTypeArray,
			ParentID:        parentID,
			ContentPreview:  arrayPreview,
			ContentLocation: p.createContentLocation(sourceID, JSONElementTypeArray, jsonPath),
			ContentHash:     p.generateHash(p.serializeForHash(v)),
			ElementOrder:    *counter,
			DocumentOrder:   *counter,
			Metadata: map[string]interface{}{
				"item_count": len(v),
				"json_path":  jsonPath,
			},
		}

		*elements = append(*elements, arrayElement)
		*counter++

		// Create bidirectional parent-child relationships
		if parentID != "" {
			containsRel := JSONRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  parentID,
				TargetElementID:  arrayID,
				RelationshipType: "contains",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			containedByRel := JSONRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  arrayID,
				TargetElementID:  parentID,
				RelationshipType: "contained_by",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			*relationships = append(*relationships, containsRel, containedByRel)
		}

		// Parse array items
		for i, item := range v {
			itemPath := fmt.Sprintf("%s[%d]", jsonPath, i)
			itemID := p.generateID("item_")
			itemPreview := p.getItemPreview(item)

			// Check for temporal values in array items if ExtractDates is enabled
			var itemTemporalValue interface{}
			var itemTemporalMetadata map[string]interface{}

			if p.ExtractDates {
				if strItem, ok := item.(string); ok {
					// Process item value for temporal content
					normalizedValue, tempMeta := temporal.ProcessFieldValue("item", strItem)
					if tempMeta != nil {
						itemTemporalValue = tempMeta
						itemTemporalMetadata = temporal.GenerateTemporalMetadata(strItem)
						// Update the preview with normalized value if temporal
						itemPreview = p.truncateContent(fmt.Sprintf("\"%s\"", normalizedValue))
					}
				}
			}

			itemElement := JSONElement{
				ElementID:       itemID,
				DocID:          docID,
				ElementType:     JSONElementTypeItem,
				ParentID:        arrayID,
				ContentPreview:  itemPreview,
				ContentLocation: p.createContentLocation(sourceID, JSONElementTypeItem, itemPath),
				ContentHash:     p.generateHash(p.serializeForHash(item)),
				ElementOrder:    *counter,
				DocumentOrder:   *counter,
				Metadata: map[string]interface{}{
					"array_index": i,
					"json_path":   itemPath,
					"value_type":  p.getValueType(item),
				},
				TemporalValue:   itemTemporalValue,
			}

			// Add temporal metadata if found
			if itemTemporalMetadata != nil {
				for k, v := range itemTemporalMetadata {
					itemElement.Metadata[k] = v
				}
			}

			*elements = append(*elements, itemElement)
			*counter++

			// Create bidirectional parent-child relationships
			containsRel := JSONRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  arrayID,
				TargetElementID:  itemID,
				RelationshipType: "contains_array_item",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			containedByRel := JSONRelationship{
				RelationshipID:   p.generateID("rel_"),
				SourceElementID:  itemID,
				TargetElementID:  arrayID,
				RelationshipType: "contained_by",
				Confidence:       1.0,
				Metadata:         make(map[string]interface{}),
			}
			*relationships = append(*relationships, containsRel, containedByRel)

			// Recursively parse item value
			if p.isComplexType(item) {
				p.parseJSONElement(item, docID, itemID, sourceID, elements, relationships,
					itemPath, depth+1, counter)
			}
		}
	}
}

// Helper functions for content generation
func (p *JSONParser) getObjectPreview(obj map[string]interface{}) string {
	keys := p.getObjectKeys(obj)
	if len(keys) == 0 {
		return "{}"
	}

	preview := "{"
	for i, key := range keys {
		if i >= 3 {
			preview += ", ..."
			break
		}
		if i > 0 {
			preview += ", "
		}
		preview += fmt.Sprintf("%s: ...", key)
	}
	preview += "}"

	return p.truncateContent(preview)
}

func (p *JSONParser) getArrayPreview(arr []interface{}) string {
	if len(arr) == 0 {
		return "[]"
	}

	preview := "["
	for i := 0; i < len(arr) && i < 3; i++ {
		if i > 0 {
			preview += ", "
		}
		preview += "..."
	}
	if len(arr) > 3 {
		preview += ", ..."
	}
	preview += "]"

	return p.truncateContent(preview)
}

func (p *JSONParser) getFieldPreview(key string, value interface{}) string {
	var valueStr string
	switch v := value.(type) {
	case string:
		valueStr = fmt.Sprintf("\"%s\"", v)
	case map[string]interface{}:
		valueStr = "{...}"
	case []interface{}:
		valueStr = "[...]"
	default:
		valueStr = fmt.Sprintf("%v", v)
	}

	preview := fmt.Sprintf("%s: %s", key, valueStr)
	return p.truncateContent(preview)
}

func (p *JSONParser) getItemPreview(item interface{}) string {
	switch v := item.(type) {
	case string:
		return p.truncateContent(fmt.Sprintf("\"%s\"", v))
	case map[string]interface{}:
		return p.truncateContent("{...}")
	case []interface{}:
		return p.truncateContent("[...]")
	default:
		return p.truncateContent(fmt.Sprintf("%v", v))
	}
}

func (p *JSONParser) getValueType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func (p *JSONParser) isComplexType(value interface{}) bool {
	switch value.(type) {
	case map[string]interface{}, []interface{}:
		return true
	default:
		return false
	}
}

func (p *JSONParser) getObjectKeys(obj map[string]interface{}) []string {
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	return keys
}

// extractLinks extracts URLs and email addresses from JSON string values
func (p *JSONParser) extractLinks(data interface{}, links *[]JSONLink, sourceID string) {
	// URL regex pattern
	urlRegex := regexp.MustCompile(`https?://[^\s"'<>]+`)
	// Email regex pattern
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	p.extractLinksRecursive(data, links, sourceID, urlRegex, emailRegex)
}

func (p *JSONParser) extractLinksRecursive(data interface{}, links *[]JSONLink, sourceID string, urlRegex *regexp.Regexp, emailRegex *regexp.Regexp) {
	switch v := data.(type) {
	case string:
		// Look for URLs in string values
		urlMatches := urlRegex.FindAllString(v, -1)
		for _, match := range urlMatches {
			link := JSONLink{
				SourceID:   sourceID,
				LinkText:   match,
				LinkTarget: match,
				LinkType:   "url",
			}
			*links = append(*links, link)
		}

		// Look for email addresses in string values
		emailMatches := emailRegex.FindAllString(v, -1)
		for _, match := range emailMatches {
			link := JSONLink{
				SourceID:   sourceID,
				LinkText:   match,
				LinkTarget: "mailto:" + match,
				LinkType:   "url",
			}
			*links = append(*links, link)
		}
	case map[string]interface{}:
		for _, value := range v {
			p.extractLinksRecursive(value, links, sourceID, urlRegex, emailRegex)
		}
	case []interface{}:
		for _, item := range v {
			p.extractLinksRecursive(item, links, sourceID, urlRegex, emailRegex)
		}
	}
}

// createContentLocation creates a content location object
func (p *JSONParser) createContentLocation(source string, elementType JSONElementType, path string) map[string]interface{} {
	return map[string]interface{}{
		"source": source,
		"type":   string(elementType),
		"path":   path,
	}
}

// Helper functions
func (p *JSONParser) generateID(prefix string) string {
	return generateID(prefix)
}

func (p *JSONParser) generateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func (p *JSONParser) truncateContent(content string) string {
	if len(content) <= p.MaxContentPreview {
		return content
	}
	return content[:p.MaxContentPreview-3] + "..."
}

func (p *JSONParser) serializeForHash(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(bytes)
}

// ToJSON converts the parse response to JSON
func (r *JSONParseResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a ParseRequest from JSON
func (r *JSONParseRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}

// convertToParseResult converts JSONParseResponse to universal ParseResult format
func (p *JSONParser) convertToParseResult(response *JSONParseResponse) *ParseResult {
	result := &ParseResult{
		Document: Document{
			ID:      response.Document["doc_id"].(string),
			DocType: response.Document["doc_type"].(string),
		},
		Elements:      make([]Element, 0, len(response.Elements)),
		Relationships: make([]Relationship, 0, len(response.Relationships)),
		Links:         make([]Link, 0, len(response.Links)),
	}

	// Convert metadata if present
	if meta, ok := response.Document["metadata"]; ok {
		result.Document.Metadata = meta.(map[string]interface{})
	}

	// Convert elements
	for i, jsonElem := range response.Elements {
		element := Element{
			ElementID:       jsonElem.ElementID,
			ElementType:     string(jsonElem.ElementType),
			Content:         jsonElem.Content,
			ContentPreview:  jsonElem.ContentPreview,
			ParentID:        jsonElem.ParentID,
			Position:        i,
			Depth:           calculateJSONDepth(jsonElem.ElementType),
			ContentLocation: jsonElem.ContentLocation,
			Metadata:        jsonElem.Metadata,
		}
		result.Elements = append(result.Elements, element)
	}

	// Convert relationships
	for _, jsonRel := range response.Relationships {
		relationship := Relationship{
			RelationshipID:   jsonRel.RelationshipID,
			RelationshipType: jsonRel.RelationshipType,
			SourceElementID:  jsonRel.SourceElementID,
			TargetElementID:  jsonRel.TargetElementID,
			Confidence:       jsonRel.Confidence,
			Metadata:         jsonRel.Metadata,
		}
		result.Relationships = append(result.Relationships, relationship)
	}

	// Convert links
	for _, jsonLink := range response.Links {
		link := Link{
			LinkID:          generateID("link"),
			SourceElementID: jsonLink.SourceID,
			LinkType:        jsonLink.LinkType,
			LinkTarget:      jsonLink.LinkTarget,
			LinkText:        jsonLink.LinkText,
		}
		result.Links = append(result.Links, link)
	}

	return result
}

// calculateJSONDepth calculates element depth based on JSON element type
func calculateJSONDepth(elementType JSONElementType) int {
	switch elementType {
	case JSONElementTypeRoot:
		return 0
	case JSONElementTypeObject, JSONElementTypeArray:
		return 1
	case JSONElementTypeField, JSONElementTypeItem:
		return 2
	default:
		return 1 // Default depth for unknown elements
	}
}

// SupportsLocation checks if this parser can resolve the given content location
func (p *JSONParser) SupportsLocation(contentLocation map[string]interface{}) bool {
	if contentLocation == nil {
		return false
	}

	// Check if source file exists and is a JSON file
	source, ok := contentLocation["source"].(string)
	if !ok || source == "" {
		return false
	}

	if _, err := os.Stat(source); os.IsNotExist(err) {
		return false
	}

	ext := strings.ToLower(filepath.Ext(source))
	return ext == ".json"
}

// ResolveElementText extracts plain text for a specific element using JSONPath
func (p *JSONParser) ResolveElementText(contentLocation map[string]interface{}, sourceContent string) (string, error) {
	value, err := p.resolveElementValue(contentLocation)
	if err != nil {
		return "", err
	}

	// Convert value to text representation
	return p.valueToText(value), nil
}

// ResolveElementContent extracts raw JSON content for a specific element using JSONPath
func (p *JSONParser) ResolveElementContent(contentLocation map[string]interface{}, sourceContent string) (string, error) {
	value, err := p.resolveElementValue(contentLocation)
	if err != nil {
		return "", err
	}

	// Serialize value to JSON string
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to serialize JSON value: %w", err)
	}

	return string(bytes), nil
}

// resolveElementValue resolves a JSON element using JSONPath
func (p *JSONParser) resolveElementValue(contentLocation map[string]interface{}) (interface{}, error) {
	source, _ := contentLocation["source"].(string)
	path, ok := contentLocation["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("missing or invalid 'path' in content_location")
	}

	// Read JSON file
	fileContent, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	// Parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(fileContent, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Handle root path specially
	if path == "$" {
		return jsonData, nil
	}

	// Compile and execute JSONPath
	compiled, err := jsonpath.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSONPath '%s': %w", path, err)
	}

	result, err := compiled.Lookup(jsonData)
	if err != nil {
		return nil, fmt.Errorf("JSONPath '%s' failed: %w", path, err)
	}

	return result, nil
}

// valueToText converts a JSON value to plain text
func (p *JSONParser) valueToText(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64, int, int64, bool:
		return fmt.Sprintf("%v", v)
	case nil:
		return ""
	case map[string]interface{}:
		// For objects, return JSON representation
		bytes, _ := json.Marshal(v)
		return string(bytes)
	case []interface{}:
		// For arrays, return JSON representation
		bytes, _ := json.Marshal(v)
		return string(bytes)
	default:
		return fmt.Sprintf("%v", v)
	}
}