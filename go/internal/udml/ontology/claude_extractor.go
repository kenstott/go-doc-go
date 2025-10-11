package ontology

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClaudeExtractor uses Anthropic's Claude API for ontology extraction
type ClaudeExtractor struct {
	apiKey      string
	modelName   string
	baseURL     string
	timeout     time.Duration
	maxRetries  int
	httpClient  *http.Client
	version     string
}

// ClaudeRequest represents a request to the Claude API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []ClaudeMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	System      string          `json:"system,omitempty"`
}

// ClaudeMessage represents a message in the conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	Usage        ClaudeUsage `json:"usage"`
}

// ClaudeUsage represents token usage information
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeExtractionResult represents the parsed JSON response from Claude
type ClaudeExtractionResult struct {
	Entities      []ClaudeEntity       `json:"entities"`
	Relationships []ClaudeRelationship `json:"relationships"`
	Classes       []ClaudeClass        `json:"classes"`
}

// ClaudeEntity represents an entity in the extraction result
type ClaudeEntity struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Confidence float64                `json:"confidence"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ClaudeRelationship represents a relationship in the extraction result
type ClaudeRelationship struct {
	Source     string  `json:"source"`
	Target     string  `json:"target"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// ClaudeClass represents a class in the extraction result
type ClaudeClass struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parent      string          `json:"parent,omitempty"`
	Properties  []ClaudeProperty `json:"properties,omitempty"`
}

// ClaudeProperty represents a property of a class
type ClaudeProperty struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// init registers the Claude extractor
func init() {
	Register("claude", NewClaudeExtractor)
}

// NewClaudeExtractor creates a new Claude extractor
func NewClaudeExtractor(config ExtractorConfig) (OntologyExtractor, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("claude extractor requires API key")
	}

	modelName := config.ModelName
	if modelName == "" {
		modelName = "claude-3-5-sonnet-20241022" // Default to Claude 3.5 Sonnet
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	return &ClaudeExtractor{
		apiKey:     apiKey,
		modelName:  modelName,
		baseURL:    baseURL,
		timeout:    timeout,
		maxRetries: maxRetries,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		version: "1.0.0",
	}, nil
}

// GetName returns the extractor identifier
func (c *ClaudeExtractor) GetName() string {
	return "claude"
}

// GetVersion returns the extractor version
func (c *ClaudeExtractor) GetVersion() string {
	return c.version
}

// ExtractFromText extracts ontology from text using Claude API
func (c *ClaudeExtractor) ExtractFromText(ctx context.Context, text string, options ExtractionOptions) (*Ontology, error) {
	// Build the extraction prompt
	prompt := c.buildPrompt(text, options)

	// Call Claude API
	response, err := c.callClaude(ctx, prompt, options)
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Parse the response
	extractionResult, err := c.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	// Convert to ontology
	ontology := c.buildOntology(extractionResult, options, response.Usage)

	// Link entity names to IDs
	c.linkRelationshipsToEntities(ontology)

	// Apply filters
	if options.MinConfidence > 0 {
		filter := OntologyFilter{
			MinConfidence: options.MinConfidence,
		}
		ontology = FilterOntology(ontology, filter)
	}

	// Validate
	if err := ontology.Validate(); err != nil {
		return nil, fmt.Errorf("ontology validation failed: %w", err)
	}

	return ontology, nil
}

// ExtractFromElements extracts ontology from structured elements
func (c *ClaudeExtractor) ExtractFromElements(ctx context.Context, elements []Element, options ExtractionOptions) (*Ontology, error) {
	// Format elements as text with context
	text := FormatElementsWithContext(elements)

	// Use text extraction
	ontology, err := c.ExtractFromText(ctx, text, options)
	if err != nil {
		return nil, err
	}

	// Link entities to elements (basic implementation - link first entity to first element)
	elementIdx := 0
	for i := range ontology.Entities {
		if elementIdx < len(elements) {
			ontology.Entities[i].ElementID = elements[elementIdx].ElementID
			elementIdx++
		}
	}

	ontology.Metadata.ElementCount = len(elements)
	return ontology, nil
}

// SupportsFeature checks if the extractor supports a specific feature
func (c *ClaudeExtractor) SupportsFeature(feature string) bool {
	supported := map[string]bool{
		"entity_extraction":       true,
		"relationship_extraction": true,
		"class_extraction":        true,
		"batch_processing":        true,
		"streaming":               false,
	}
	return supported[feature]
}

// Close cleans up resources
func (c *ClaudeExtractor) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// buildPrompt builds the extraction prompt
func (c *ClaudeExtractor) buildPrompt(text string, options ExtractionOptions) string {
	if options.CustomPrompt != "" {
		return strings.ReplaceAll(options.CustomPrompt, "{{.Text}}", text)
	}

	// Use combined extraction prompt if classes are requested
	if options.ExtractClasses {
		return BuildCombinedExtractionPrompt(text)
	}

	// Otherwise, use entity and relationship only (simpler, faster)
	entityPrompt := BuildEntityExtractionPrompt(text)

	// Add relationship extraction to the prompt
	prompt := entityPrompt + "\n\nAlso extract relationships between the entities you identified. " +
		"For each relationship, provide source entity name, target entity name, type " +
		"(is_a, part_of, related_to, located_in, occurred_at, created_by, mentions, depends_on), " +
		"confidence (0.0-1.0), and evidence text.\n\n" +
		"Return the result as JSON with this structure:\n" +
		"{\n" +
		"  \"entities\": [...],\n" +
		"  \"relationships\": [...],\n" +
		"  \"classes\": []\n" +
		"}"

	return prompt
}

// callClaude makes a request to the Claude API
func (c *ClaudeExtractor) callClaude(ctx context.Context, prompt string, options ExtractionOptions) (*ClaudeResponse, error) {
	request := ClaudeRequest{
		Model:     c.modelName,
		MaxTokens: 4096,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: options.Temperature,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(requestBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
			continue
		}

		var response ClaudeResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return &response, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.maxRetries, lastErr)
}

// parseResponse parses the Claude API response into extraction result
func (c *ClaudeExtractor) parseResponse(response *ClaudeResponse) (*ClaudeExtractionResult, error) {
	if len(response.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	// Get the text content
	text := response.Content[0].Text

	// Try to extract JSON from the response (Claude might wrap it in markdown)
	jsonText := c.extractJSON(text)

	// First parse into a flexible structure
	var rawResult map[string]interface{}
	if err := json.Unmarshal([]byte(jsonText), &rawResult); err != nil {
		return nil, fmt.Errorf("failed to parse extraction result: %w (response: %s)", err, jsonText)
	}

	// Now manually build the result to handle flexible properties format
	result := &ClaudeExtractionResult{
		Entities:      []ClaudeEntity{},
		Relationships: []ClaudeRelationship{},
		Classes:       []ClaudeClass{},
	}

	// Parse entities
	if entitiesRaw, ok := rawResult["entities"]; ok {
		entitiesJSON, _ := json.Marshal(entitiesRaw)
		json.Unmarshal(entitiesJSON, &result.Entities)
	}

	// Parse relationships
	if relationshipsRaw, ok := rawResult["relationships"]; ok {
		relationshipsJSON, _ := json.Marshal(relationshipsRaw)
		json.Unmarshal(relationshipsJSON, &result.Relationships)
	}

	// Parse classes (with flexible properties handling)
	if classesRaw, ok := rawResult["classes"]; ok {
		if classesArray, ok := classesRaw.([]interface{}); ok {
			for _, classRaw := range classesArray {
				if classMap, ok := classRaw.(map[string]interface{}); ok {
					class := ClaudeClass{
						Name:        getString(classMap, "name"),
						Description: getString(classMap, "description"),
						Parent:      getString(classMap, "parent"),
						Properties:  []ClaudeProperty{},
					}

					// Handle properties flexibly - could be array of strings or objects
					if propsRaw, ok := classMap["properties"]; ok {
						switch props := propsRaw.(type) {
						case []interface{}:
							for _, propRaw := range props {
								switch p := propRaw.(type) {
								case string:
									// Simple string property name
									class.Properties = append(class.Properties, ClaudeProperty{
										Name:     p,
										Type:     "string",
										Required: false,
									})
								case map[string]interface{}:
									// Full property object
									prop := ClaudeProperty{
										Name:        getString(p, "name"),
										Type:        getString(p, "type"),
										Description: getString(p, "description"),
										Required:    getBool(p, "required"),
									}
									class.Properties = append(class.Properties, prop)
								}
							}
						}
					}

					result.Classes = append(result.Classes, class)
				}
			}
		}
	}

	return result, nil
}

// Helper functions for parsing
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// extractJSON attempts to extract JSON from text that might be wrapped in markdown
func (c *ClaudeExtractor) extractJSON(text string) string {
	// Try to find JSON block in markdown
	if strings.Contains(text, "```json") {
		start := strings.Index(text, "```json") + 7
		end := strings.Index(text[start:], "```")
		if end > 0 {
			return strings.TrimSpace(text[start : start+end])
		}
	}

	// Try to find JSON block without language specifier
	if strings.Contains(text, "```") {
		start := strings.Index(text, "```") + 3
		end := strings.Index(text[start:], "```")
		if end > 0 {
			candidate := strings.TrimSpace(text[start : start+end])
			if strings.HasPrefix(candidate, "{") {
				return candidate
			}
		}
	}

	// Assume the entire text is JSON
	return strings.TrimSpace(text)
}

// buildOntology converts extraction result to ontology
func (c *ClaudeExtractor) buildOntology(result *ClaudeExtractionResult, options ExtractionOptions, usage ClaudeUsage) *Ontology {
	ontology := &Ontology{
		ID:            generateID("ont"),
		DocID:         options.DocID,
		Name:          options.DocName,
		Description:   fmt.Sprintf("Ontology extracted from %s using Claude", options.DocName),
		Version:       "1.0.0",
		Entities:      make([]Entity, 0, len(result.Entities)),
		Relationships: make([]Relationship, 0, len(result.Relationships)),
		Classes:       make([]Class, 0, len(result.Classes)),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Convert entities
	for _, ce := range result.Entities {
		entity := Entity{
			ID:         generateID("ent"),
			Name:       ce.Name,
			Type:       c.parseEntityType(ce.Type),
			Confidence: ce.Confidence,
			Attributes: ce.Attributes,
			Mentions:   []Mention{},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if entity.Attributes == nil {
			entity.Attributes = make(map[string]interface{})
		}
		ontology.Entities = append(ontology.Entities, entity)
	}

	// Convert relationships (we'll link them to entity IDs later)
	for _, cr := range result.Relationships {
		rel := Relationship{
			ID:         generateID("rel"),
			Type:       c.parseRelationshipType(cr.Type),
			SourceID:   cr.Source, // Temporary - will be replaced with entity ID
			TargetID:   cr.Target, // Temporary - will be replaced with entity ID
			Confidence: cr.Confidence,
			Attributes: make(map[string]interface{}),
			Evidence:   cr.Evidence,
			CreatedAt:  time.Now(),
		}
		ontology.Relationships = append(ontology.Relationships, rel)
	}

	// Convert classes
	for _, cc := range result.Classes {
		class := Class{
			ID:          generateID("cls"),
			Name:        cc.Name,
			Description: cc.Description,
			ParentID:    cc.Parent,
			Properties:  make([]Property, 0, len(cc.Properties)),
			Attributes:  make(map[string]interface{}),
			CreatedAt:   time.Now(),
		}

		for _, cp := range cc.Properties {
			prop := Property{
				Name:        cp.Name,
				Type:        cp.Type,
				Description: cp.Description,
				Required:    cp.Required,
			}
			class.Properties = append(class.Properties, prop)
		}

		ontology.Classes = append(ontology.Classes, class)
	}

	// Set metadata
	ontology.Metadata = OntologyMeta{
		ExtractorType:    "claude",
		ExtractorVersion: c.version,
		ElementCount:     1,
		Confidence:       c.calculateOverallConfidence(ontology),
		ModelName:        c.modelName,
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		CustomFields:     make(map[string]interface{}),
	}

	return ontology
}

// linkRelationshipsToEntities replaces entity names with entity IDs in relationships
func (c *ClaudeExtractor) linkRelationshipsToEntities(ontology *Ontology) {
	// Build name-to-ID map
	nameToID := make(map[string]string)
	for _, entity := range ontology.Entities {
		nameToID[entity.Name] = entity.ID
	}

	// Update relationships
	validRelationships := make([]Relationship, 0, len(ontology.Relationships))
	for i := range ontology.Relationships {
		rel := &ontology.Relationships[i]

		// Try to find source entity ID
		if sourceID, ok := nameToID[rel.SourceID]; ok {
			rel.SourceID = sourceID
		}

		// Try to find target entity ID
		if targetID, ok := nameToID[rel.TargetID]; ok {
			rel.TargetID = targetID
		}

		// Only keep relationships where both entities exist
		if _, sourceExists := nameToID[rel.SourceID]; sourceExists {
			if _, targetExists := nameToID[rel.TargetID]; targetExists {
				validRelationships = append(validRelationships, *rel)
			}
		} else {
			// Check if IDs are already set
			sourceFound := false
			targetFound := false
			for _, e := range ontology.Entities {
				if e.ID == rel.SourceID {
					sourceFound = true
				}
				if e.ID == rel.TargetID {
					targetFound = true
				}
			}
			if sourceFound && targetFound {
				validRelationships = append(validRelationships, *rel)
			}
		}
	}

	ontology.Relationships = validRelationships
}

// parseEntityType converts string to EntityType
func (c *ClaudeExtractor) parseEntityType(typeStr string) EntityType {
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

// parseRelationshipType converts string to RelationshipType
func (c *ClaudeExtractor) parseRelationshipType(typeStr string) RelationshipType {
	typeStr = strings.ToLower(strings.TrimSpace(typeStr))
	switch typeStr {
	case "is_a", "isa", "is a":
		return RelationshipIsA
	case "part_of", "partof", "part of":
		return RelationshipPartOf
	case "related_to", "relatedto", "related to":
		return RelationshipRelatedTo
	case "located_in", "locatedin", "located in":
		return RelationshipLocatedIn
	case "occurred_at", "occurredat", "occurred at":
		return RelationshipOccurredAt
	case "created_by", "createdby", "created by":
		return RelationshipCreatedBy
	case "mentions":
		return RelationshipMentions
	case "depends_on", "dependson", "depends on":
		return RelationshipDependsOn
	case "implements":
		return RelationshipImplements
	case "extends":
		return RelationshipExtends
	case "contains":
		return RelationshipContains
	case "referenced_by", "referencedby", "referenced by":
		return RelationshipReferencedBy
	default:
		return RelationshipCustom
	}
}

// calculateOverallConfidence calculates the overall confidence score
func (c *ClaudeExtractor) calculateOverallConfidence(ontology *Ontology) float64 {
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
