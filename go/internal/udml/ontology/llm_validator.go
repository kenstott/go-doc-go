package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// LLMValidationTemplate defines LLM validation prompts for entity types
type LLMValidationTemplate struct {
	EntityType           string   // Entity type this validation applies to
	QuestionTemplate     string   // Template for validation question (use {entity} placeholder)
	RejectExamples       []string // Examples of things that should be rejected
	AcceptExamples       []string // Examples of things that should be accepted
	AdditionalGuidance   string   // Optional additional guidance for the LLM
	RequireVeryConfident bool     // If true, emphasizes "VERY confident" language
}

// llmValidationTemplates defines LLM validation prompts for different entity types
var llmValidationTemplates = map[string]LLMValidationTemplate{
	"person": {
		EntityType:           "person",
		QuestionTemplate:     "Is '{entity}' an individual human person's name with BOTH first and last name (not an organization, concept, place, or single-word name)?",
		RequireVeryConfident: true,
		RejectExamples: []string{
			"Medical Association",
			"Medicine",
			"Medical Science",
			"Department of Health",
			"World Health",
			"Public Health",
			"Harvard Medical",
			"",        // Empty string
			"Abraham", // Single-word names
			"Apollo",  // Deity
			"Adam",
			"Antoine",
			"Anon",
			"Amon", // Deity
			"Abbate",
			"Achillini",
		},
		AcceptExamples: []string{
			"John Smith",
			"Dr. Jane Doe",
			"Professor Albert Einstein",
			"Mary Johnson",
			"Dr. Robert Koch",
		},
		AdditionalGuidance: "Answer 'yes' ONLY if you are VERY confident it is an individual human person's name with BOTH first and last name. Answer 'no' for: organizations, departments, concepts, academic fields, single-word names, deities, empty strings, incomplete names, or anything uncertain.",
	},
	"organization": {
		EntityType:           "organization",
		QuestionTemplate:     "Is '{entity}' a specific, real organization name (not a sentence fragment, generic term, or concatenated list)?",
		RequireVeryConfident: true,
		RejectExamples: []string{
			"Endemol Shine Group Takeover Approved By European Commission",
			"Agriculture Banking Communications Companies Energy Insurance Manufacturing",
			"Jesus Christ Ministry Crucifixion Resurrection Great Commission",
			"American Revolution War Second Continental Congress",
			"Advanced Manufacturing",
			"American Council",
			"The Institute",
		},
		AcceptExamples: []string{
			"Harvard University",
			"Microsoft Corporation",
			"World Health Organization",
			"Stanford University",
			"General Electric",
			"United Nations",
		},
		AdditionalGuidance: "Answer 'yes' ONLY for specific, real organization names. Answer 'no' for sentence fragments, generic terms (like 'Manufacturing', 'Council'), incomplete names, or concatenated lists of words.",
	},
}

// getLLMValidationTemplate retrieves an LLM validation template by entity type
func getLLMValidationTemplate(entityType string) (LLMValidationTemplate, bool) {
	template, exists := llmValidationTemplates[entityType]
	return template, exists
}

// LLMValidator provides LLM-based validation for filtering false positives
type LLMValidator struct {
	llmClient LLMClient
}

// NewLLMValidator creates a new LLM validator
func NewLLMValidator(llmClient LLMClient) *LLMValidator {
	return &LLMValidator{
		llmClient: llmClient,
	}
}

// HasValidationTemplate checks if a validation template exists for the given entity type
func (v *LLMValidator) HasValidationTemplate(entityType string) bool {
	_, exists := llmValidationTemplates[entityType]
	return exists
}

// GetAvailableEntityTypes returns all entity types that have validation templates
func (v *LLMValidator) GetAvailableEntityTypes() []string {
	types := make([]string, 0, len(llmValidationTemplates))
	for entityType := range llmValidationTemplates {
		types = append(types, entityType)
	}
	return types
}

// EntityToValidate represents an entity that needs LLM validation
type EntityToValidate struct {
	EntityName string // The entity name to validate
	Prompt     string // The validation prompt (deprecated - use entity type templates instead)
	EntityType string // The entity type for template lookup
}

// ValidationResponse represents the LLM's response format
type ValidationResponse []string // Array of "yes"/"no" for each entity

// BatchValidate validates multiple entities using LLM in batches
// Returns []bool in same order as input entities
// On error, returns all false (strict - reject uncertain entities)
func (v *LLMValidator) BatchValidate(ctx context.Context, entities []EntityToValidate) ([]bool, error) {
	if len(entities) == 0 {
		return []bool{}, nil
	}

	log.Printf("LLM Validation: Validating %d entities...", len(entities))

	// Determine entity type - all entities in a batch are of the same type
	entityType := entities[0].EntityType
	if entityType == "" {
		// Fallback: default to person if not specified (backward compatibility)
		entityType = "person"
	}

	// Look up validation template for this entity type
	template, exists := getLLMValidationTemplate(entityType)
	if !exists {
		log.Printf("WARNING: No LLM validation template found for entity type '%s' - using generic validation", entityType)
		// Use generic validation without specific examples
		template = LLMValidationTemplate{
			EntityType:           entityType,
			QuestionTemplate:     "Is '{entity}' a valid " + entityType + "?",
			RequireVeryConfident: true,
			RejectExamples:       []string{},
			AcceptExamples:       []string{},
			AdditionalGuidance:   "Answer 'yes' ONLY if you are VERY confident. Answer 'no' for anything uncertain.",
		}
	}

	// Build prompt with template
	promptBuilder := strings.Builder{}
	promptBuilder.WriteString("You are an entity validation system. For each question below, answer ONLY 'yes' or 'no' with NO explanation.\n\n")
	promptBuilder.WriteString("IMPORTANT: You MUST respond with ONLY a JSON array in this EXACT format:\n")
	promptBuilder.WriteString(`["yes", "no", "yes", ...]` + "\n\n")
	promptBuilder.WriteString("The array must have exactly " + fmt.Sprintf("%d", len(entities)) + " answers (only 'yes' or 'no'), one for each entity below.\n\n")

	// Add entity-specific guidance from template
	if template.AdditionalGuidance != "" {
		promptBuilder.WriteString(template.AdditionalGuidance + "\n\n")
	}

	// Add examples from template
	if len(template.AcceptExamples) > 0 || len(template.RejectExamples) > 0 {
		promptBuilder.WriteString("Examples:\n")
		for _, example := range template.AcceptExamples {
			promptBuilder.WriteString(fmt.Sprintf("- \"%s\" → yes\n", example))
		}
		for _, example := range template.RejectExamples {
			promptBuilder.WriteString(fmt.Sprintf("- \"%s\" → no\n", example))
		}
		promptBuilder.WriteString("\n")
	}

	promptBuilder.WriteString("Questions:\n\n")

	// Build questions using template's question format
	questionTemplate := template.QuestionTemplate
	for i, entity := range entities {
		// Replace {entity} placeholder in question template
		question := strings.ReplaceAll(questionTemplate, "{entity}", entity.EntityName)
		promptBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, question))
		promptBuilder.WriteString(fmt.Sprintf("   Value: \"%s\"\n\n", entity.EntityName))
	}

	promptBuilder.WriteString("\nRespond with ONLY the JSON array, no additional text:")

	// Call LLM
	response, err := v.llmClient.Complete(ctx, promptBuilder.String(), LLMOptions{
		MaxTokens:    2000,
		Temperature:  0.1, // Low temperature for consistent validation
		SystemPrompt: "You are a JSON-only API. You respond exclusively with valid JSON arrays, nothing else. No explanations, no preambles, no markdown code fences - only pure JSON.",
		Prefill:      "[", // Force JSON array format, avoid preamble
	})

	if err != nil {
		// Strict error handling - on failure, reject all entities
		log.Printf("WARNING: LLM validation failed: %v - rejecting all entities (strict)", err)
		results := make([]bool, len(entities))
		for i := range results {
			results[i] = false // Reject all on error
		}
		return results, nil
	}

	// Parse JSON response
	var validationResp ValidationResponse

	// Try to extract JSON from response (LLM might add extra text)
	jsonStart := strings.Index(response, "[")
	jsonEnd := strings.LastIndex(response, "]")

	if jsonStart == -1 || jsonEnd == -1 {
		log.Printf("WARNING: LLM response does not contain JSON array - rejecting all entities (strict)")
		log.Printf("Response was: %s", response)
		results := make([]bool, len(entities))
		for i := range results {
			results[i] = false
		}
		return results, nil
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	if err := json.Unmarshal([]byte(jsonStr), &validationResp); err != nil {
		log.Printf("WARNING: Failed to parse LLM validation response: %v - rejecting all entities (strict)", err)
		log.Printf("Response was: %s", response)
		results := make([]bool, len(entities))
		for i := range results {
			results[i] = false
		}
		return results, nil
	}

	// Verify array length matches
	if len(validationResp) != len(entities) {
		log.Printf("WARNING: LLM returned %d results but expected %d - rejecting all entities (strict)",
			len(validationResp), len(entities))
		results := make([]bool, len(entities))
		for i := range results {
			results[i] = false
		}
		return results, nil
	}

	// Convert string responses to bool - ONLY "yes" (case-insensitive) is accepted
	results := make([]bool, len(entities))
	accepted := 0
	rejected := 0
	for i, answer := range validationResp {
		// Only accept clear "yes" responses
		if strings.ToLower(strings.TrimSpace(answer)) == "yes" {
			results[i] = true
			accepted++
		} else {
			results[i] = false
			rejected++
		}
	}

	log.Printf("LLM Validation complete: %d accepted, %d rejected", accepted, rejected)

	return results, nil
}

// ValidateInBatches validates entities in configurable batch sizes
// Returns []bool in same order as input entities
func (v *LLMValidator) ValidateInBatches(ctx context.Context, entities []EntityToValidate, batchSize int) ([]bool, error) {
	if batchSize <= 0 {
		batchSize = 50 // Default batch size
	}

	results := make([]bool, len(entities))

	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}

		batch := entities[i:end]
		batchResults, err := v.BatchValidate(ctx, batch)

		// On error, BatchValidate already returns permissive results (all true)
		if err != nil {
			// Copy permissive results
			copy(results[i:end], batchResults)
			continue
		}

		copy(results[i:end], batchResults)
	}

	return results, nil
}
