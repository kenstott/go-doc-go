package ontology

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/sampler"
)

// InterviewBuilder provides interactive ontology refinement through LLM-guided interview
type InterviewBuilder struct {
	builder  *OntologyBuilder
	reader   *bufio.Reader
	schema   *OntologySchema
	samples  *sampler.SamplingResult
	phase    int
	llmCalls int
	tokens   int
}

// InterviewPhase represents a phase in the interview process
type InterviewPhase int

const (
	PhaseDomainSelection InterviewPhase = iota
	PhaseEntityDiscovery
	PhaseRelationshipDiscovery
	PhaseRefinement
	PhaseComplete
)

// NewInterviewBuilder creates a new interactive ontology builder
func NewInterviewBuilder(builder *OntologyBuilder) *InterviewBuilder {
	return &InterviewBuilder{
		builder: builder,
		reader:  bufio.NewReader(os.Stdin),
		phase:   int(PhaseDomainSelection),
	}
}

// StartInterview begins the interactive ontology building interview
func (ib *InterviewBuilder) StartInterview(ctx context.Context) (*OntologySchema, error) {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║   Interactive Ontology Builder - Guided Interview        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This interactive process will help you build a high-quality")
	fmt.Println("ontology schema for your document corpus through 4 phases:")
	fmt.Println()
	fmt.Println("  1. Domain Selection - Identify relevant domains")
	fmt.Println("  2. Entity Discovery - Define entity types")
	fmt.Println("  3. Relationship Discovery - Define relationships")
	fmt.Println("  4. Refinement - Review and refine the schema")
	fmt.Println()
	fmt.Println("The LLM will ask clarifying questions. You can answer freely.")
	fmt.Println()

	// Sample the corpus first
	if err := ib.sampleCorpus(ctx); err != nil {
		return nil, err
	}

	// Phase 1: Domain Selection
	if err := ib.phaseDomainSelection(ctx); err != nil {
		return nil, err
	}

	// Phase 2: Entity Discovery
	if err := ib.phaseEntityDiscovery(ctx); err != nil {
		return nil, err
	}

	// Phase 3: Relationship Discovery
	if err := ib.phaseRelationshipDiscovery(ctx); err != nil {
		return nil, err
	}

	// Phase 4: Refinement
	if err := ib.phaseRefinement(ctx); err != nil {
		return nil, err
	}

	ib.phase = int(PhaseComplete)

	fmt.Println("\n✓ Ontology generation complete!")
	fmt.Printf("  - Domains: %d\n", len(ib.schema.Domains))
	fmt.Printf("  - Entity types: %d\n", len(ib.schema.ElementEntityMappings))
	fmt.Printf("  - Relationships: %d\n", len(ib.schema.EntityRelationshipRules))
	fmt.Printf("  - LLM calls: %d (tokens: %d)\n", ib.llmCalls, ib.tokens)
	fmt.Println()

	return ib.schema, nil
}

// sampleCorpus samples the document corpus
func (ib *InterviewBuilder) sampleCorpus(ctx context.Context) error {
	fmt.Println("→ Sampling document corpus...")

	samples, err := ib.builder.sampler.Sample(ctx)
	if err != nil {
		return fmt.Errorf("sampling failed: %w", err)
	}

	ib.samples = samples
	samples.EntityFrequencies = ib.builder.sampler.AnalyzeEntityFrequencies(samples.Samples)

	fmt.Printf("  ✓ Sampled %d elements from %d total documents\n", samples.SampledCount, samples.TotalElements)
	fmt.Printf("  ✓ Found %d unique entities\n\n", len(samples.EntityFrequencies))

	return nil
}

// phaseDomainSelection guides user through domain identification
func (ib *InterviewBuilder) phaseDomainSelection(ctx context.Context) error {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║  Phase 1: Domain Selection                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// List available domains (will be populated by calling code)
	// For now, show generic message
	fmt.Println("The system has access to built-in domain templates including:")
	fmt.Println("  - Financial, Legal, Medical, Technical")
	fmt.Println("  - Insurance, Manufacturing, Retail, Logistics")
	fmt.Println("  - Education, and more...")
	fmt.Println()

	// Ask LLM to analyze corpus and suggest domains
	sampleTexts := ib.builder.prepareSampleTexts(ib.samples.Samples, 20)

	prompt := fmt.Sprintf(`Analyze these document samples and suggest which domains are present in the corpus.

Available built-in domains:
- financial: Financial reports, earnings, transactions
- legal: Contracts, court documents, regulations
- medical: Clinical notes, prescriptions, lab reports
- technical: Software docs, API specs, manuals
- insurance: Policies, claims, underwriting
- manufacturing: Specs, production reports, QC docs
- retail: Sales reports, inventory, pricing
- logistics: Shipping docs, freight, warehouse
- education: Curricula, student records, programs

Sample texts:
%s

For each domain you detect:
1. Name the domain (from the list above or propose a new one)
2. Explain WHY you think this domain is present (cite specific evidence)
3. Rate confidence (0.0-1.0)

Ask the user 2-3 clarifying questions to better understand the corpus.

Return JSON:
{
  "detected_domains": [
    {
      "name": "financial",
      "reasoning": "I see references to revenue, profit, EBITDA in samples",
      "confidence": 0.9
    }
  ],
  "clarifying_questions": [
    "What is the primary business purpose of these documents?",
    "Are these internal reports or external filings?"
  ]
}`, sampleTexts)

	response, err := ib.builder.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:    2000,
		Temperature:  0.3,
		SystemPrompt: "You are an expert at document analysis and domain identification. Ask thoughtful questions to understand the corpus better.",
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(response)

	// Parse response
	var result struct {
		DetectedDomains      []struct {
			Name       string  `json:"name"`
			Reasoning  string  `json:"reasoning"`
			Confidence float64 `json:"confidence"`
		} `json:"detected_domains"`
		ClarifyingQuestions []string `json:"clarifying_questions"`
	}

	if err := ib.builder.extractJSON(response, &result); err != nil {
		return err
	}

	// Show LLM's analysis
	fmt.Println("LLM Analysis:")
	fmt.Println("─────────────")
	for _, domain := range result.DetectedDomains {
		fmt.Printf("  • %s (confidence: %.2f)\n", domain.Name, domain.Confidence)
		fmt.Printf("    %s\n", domain.Reasoning)
	}
	fmt.Println()

	// Ask clarifying questions
	userAnswers := []string{}
	for _, question := range result.ClarifyingQuestions {
		fmt.Printf("LLM: %s\n", question)
		fmt.Print("You: ")
		answer, err := ib.reader.ReadString('\n')
		if err != nil {
			return err
		}
		userAnswers = append(userAnswers, strings.TrimSpace(answer))
		fmt.Println()
	}

	// LLM refines domain selection based on answers
	refinementPrompt := fmt.Sprintf(`Based on the user's answers, finalize the domain selection.

Initial analysis:
%s

User's answers:
%s

Return final domain selection as JSON array:
[
  {
    "name": "financial",
    "description": "Financial performance, reporting, and metrics",
    "owner": "CFO Office",
    "key_concepts": ["revenue", "profit", "EBITDA", "cash flow"]
  }
]`, formatDetectedDomains(result.DetectedDomains), formatUserAnswers(result.ClarifyingQuestions, userAnswers))

	finalResponse, err := ib.builder.llmClient.Complete(ctx, refinementPrompt, LLMOptions{
		MaxTokens:    1500,
		Temperature:  0.2,
		SystemPrompt: "Finalize domain selection based on document analysis and user input.",
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(finalResponse)

	var domains []Domain
	if err := ib.builder.extractJSON(finalResponse, &domains); err != nil {
		return err
	}

	// Initialize schema with domains
	ib.schema = &OntologySchema{
		Name:                    ib.builder.config.SchemaName,
		Version:                 ib.builder.config.SchemaVersion,
		Domains:                 domains,
		ElementEntityMappings:   []ElementEntityMapping{},
		EntityRelationshipRules: []EntityRelationshipRule{},
	}

	fmt.Println("Selected Domains:")
	for _, d := range domains {
		fmt.Printf("  ✓ %s - %s (Owner: %s)\n", d.Name, d.Description, d.Owner)
	}
	fmt.Println()

	return nil
}

// phaseEntityDiscovery guides user through entity type definition
func (ib *InterviewBuilder) phaseEntityDiscovery(ctx context.Context) error {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║  Phase 2: Entity Discovery                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get top entities from frequency analysis
	topEntities := ib.samples.GetTopEntities(20)

	fmt.Println("Most frequent entities found:")
	for i, e := range topEntities[:10] {
		fmt.Printf("  %d. %s (%d occurrences)\n", i+1, e.Entity, e.Count)
	}
	fmt.Println()

	// LLM proposes entity types and asks questions
	sampleTexts := ib.builder.prepareSampleTexts(ib.samples.Samples, 15)

	prompt := fmt.Sprintf(`Based on the document samples and detected domains, propose entity types to extract.

Detected domains:
%s

Top entities:
%s

Sample texts:
%s

For each entity type:
1. Name the entity type
2. Describe what it represents
3. Suggest extraction patterns (regex, keywords, etc.)
4. Assign to appropriate domain

Ask the user 2-3 questions about entity types they care about or ambiguities you see.

Return JSON:
{
  "proposed_entities": [
    {
      "entity_type": "company",
      "description": "Business organization",
      "domain": "financial",
      "sample_patterns": ["regex: \\b[A-Z][a-z]+ (Inc|Corp|LLC)\\b"]
    }
  ],
  "questions": [
    "Should we distinguish between subsidiaries and parent companies?",
    "Are there specific company identifiers (like ticker symbols) you want to extract?"
  ]
}`, formatDomains(ib.schema.Domains), formatTopEntities(topEntities[:20]), sampleTexts)

	response, err := ib.builder.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:    3000,
		Temperature:  0.3,
		SystemPrompt: "You are an expert at entity extraction and ontology design. Propose entity types and ask clarifying questions.",
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(response)

	var result struct {
		ProposedEntities []struct {
			EntityType     string   `json:"entity_type"`
			Description    string   `json:"description"`
			Domain         string   `json:"domain"`
			SamplePatterns []string `json:"sample_patterns"`
		} `json:"proposed_entities"`
		Questions []string `json:"questions"`
	}

	if err := ib.builder.extractJSON(response, &result); err != nil {
		return err
	}

	// Show proposals
	fmt.Println("Proposed Entity Types:")
	fmt.Println("──────────────────────")
	for _, e := range result.ProposedEntities {
		fmt.Printf("  • %s (%s domain)\n", e.EntityType, e.Domain)
		fmt.Printf("    %s\n", e.Description)
	}
	fmt.Println()

	// Ask questions
	userAnswers := []string{}
	for _, question := range result.Questions {
		fmt.Printf("LLM: %s\n", question)
		fmt.Print("You: ")
		answer, err := ib.reader.ReadString('\n')
		if err != nil {
			return err
		}
		userAnswers = append(userAnswers, strings.TrimSpace(answer))
		fmt.Println()
	}

	// Finalize entity definitions
	finalizationPrompt := fmt.Sprintf(`Based on user feedback, finalize entity type definitions with complete extraction rules.

User's feedback:
%s

Create complete ElementEntityMapping objects with:
- entity_type, description, domain
- element_types where they appear
- confidence level (0.65-0.95)
- extraction_rules (keyword_match, regex_pattern, etc.)

Return as JSON array matching ElementEntityMapping structure.`, formatUserAnswers(result.Questions, userAnswers))

	finalResponse, err := ib.builder.llmClient.Complete(ctx, finalizationPrompt, LLMOptions{
		MaxTokens:   4000,
		Temperature: 0.2,
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(finalResponse)

	var entityMappings []ElementEntityMapping
	if err := ib.builder.extractJSON(finalResponse, &entityMappings); err != nil {
		return err
	}

	ib.schema.ElementEntityMappings = entityMappings

	fmt.Printf("✓ Defined %d entity types\n\n", len(entityMappings))

	return nil
}

// phaseRelationshipDiscovery guides user through relationship definition
func (ib *InterviewBuilder) phaseRelationshipDiscovery(ctx context.Context) error {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║  Phase 3: Relationship Discovery                         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Extract entity types for relationship discovery
	entityTypes := make([]string, len(ib.schema.ElementEntityMappings))
	for i, m := range ib.schema.ElementEntityMappings {
		entityTypes[i] = m.EntityType
	}

	fmt.Println("Entity types defined:")
	for i, et := range entityTypes {
		fmt.Printf("  %d. %s\n", i+1, et)
	}
	fmt.Println()

	// LLM proposes relationships
	sampleTexts := ib.builder.prepareSampleTexts(ib.samples.Samples, 10)

	prompt := fmt.Sprintf(`Given these entity types: %s

Analyze sample texts to discover relationships between entities.

Sample texts:
%s

Propose relationships with:
1. Relationship name
2. Source and target entity types
3. Relationship type (is_a, part_of, related_to, etc.)
4. Sample patterns showing how the relationship appears in text

Ask 1-2 questions about important relationships to capture.

Return JSON:
{
  "proposed_relationships": [
    {
      "name": "company_reports_metric",
      "source_type": "company",
      "target_type": "financial_metric",
      "relationship_type": "related_to",
      "sample_patterns": ["{company} reported {metric}"]
    }
  ],
  "questions": [
    "Are there specific ownership or hierarchical relationships we should capture?"
  ]
}`, strings.Join(entityTypes, ", "), sampleTexts)

	response, err := ib.builder.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:   3000,
		Temperature: 0.3,
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(response)

	var result struct {
		ProposedRelationships []struct {
			Name             string   `json:"name"`
			SourceType       string   `json:"source_type"`
			TargetType       string   `json:"target_type"`
			RelationshipType string   `json:"relationship_type"`
			SamplePatterns   []string `json:"sample_patterns"`
		} `json:"proposed_relationships"`
		Questions []string `json:"questions"`
	}

	if err := ib.builder.extractJSON(response, &result); err != nil {
		return err
	}

	// Show proposals
	fmt.Println("Proposed Relationships:")
	fmt.Println("───────────────────────")
	for _, r := range result.ProposedRelationships {
		fmt.Printf("  • %s: %s → %s (%s)\n", r.Name, r.SourceType, r.TargetType, r.RelationshipType)
	}
	fmt.Println()

	// Ask questions
	userAnswers := []string{}
	for _, question := range result.Questions {
		fmt.Printf("LLM: %s\n", question)
		fmt.Print("You: ")
		answer, err := ib.reader.ReadString('\n')
		if err != nil {
			return err
		}
		userAnswers = append(userAnswers, strings.TrimSpace(answer))
		fmt.Println()
	}

	// Finalize relationships
	finalizationPrompt := fmt.Sprintf(`Based on user feedback, finalize relationship definitions.

User's feedback:
%s

Create complete EntityRelationshipRule objects with extraction patterns.
Return as JSON array matching EntityRelationshipRule structure.`, formatUserAnswers(result.Questions, userAnswers))

	finalResponse, err := ib.builder.llmClient.Complete(ctx, finalizationPrompt, LLMOptions{
		MaxTokens:   4000,
		Temperature: 0.2,
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(finalResponse)

	var relationshipRules []EntityRelationshipRule
	if err := ib.builder.extractJSON(finalResponse, &relationshipRules); err != nil {
		return err
	}

	ib.schema.EntityRelationshipRules = relationshipRules

	fmt.Printf("✓ Defined %d relationships\n\n", len(relationshipRules))

	return nil
}

// phaseRefinement allows user to review and refine the schema
func (ib *InterviewBuilder) phaseRefinement(ctx context.Context) error {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║  Phase 4: Refinement                                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Show summary
	fmt.Println("Current Schema Summary:")
	fmt.Println("───────────────────────")
	fmt.Printf("  Domains: %d\n", len(ib.schema.Domains))
	for _, d := range ib.schema.Domains {
		fmt.Printf("    • %s\n", d.Name)
	}
	fmt.Printf("  Entity types: %d\n", len(ib.schema.ElementEntityMappings))
	fmt.Printf("  Relationships: %d\n", len(ib.schema.EntityRelationshipRules))
	fmt.Println()

	fmt.Print("Would you like to make any changes? (yes/no): ")
	response, err := ib.reader.ReadString('\n')
	if err != nil {
		return err
	}

	if strings.ToLower(strings.TrimSpace(response)) != "yes" {
		fmt.Println("✓ Schema finalized\n")
		return nil
	}

	fmt.Print("What would you like to change? (describe in detail): ")
	changes, err := ib.reader.ReadString('\n')
	if err != nil {
		return err
	}

	// LLM applies changes
	schemaJSON, _ := json.MarshalIndent(ib.schema, "", "  ")

	refinementPrompt := fmt.Sprintf(`User wants to make these changes to the ontology schema:

%s

Current schema:
%s

Apply the requested changes and return the complete updated schema as JSON.`, strings.TrimSpace(changes), string(schemaJSON))

	llmResponse, err := ib.builder.llmClient.Complete(ctx, refinementPrompt, LLMOptions{
		MaxTokens:   8000,
		Temperature: 0.2,
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(llmResponse)

	var updatedSchema OntologySchema
	if err := ib.builder.extractJSON(llmResponse, &updatedSchema); err != nil {
		return fmt.Errorf("failed to parse updated schema: %w", err)
	}

	ib.schema = &updatedSchema

	fmt.Println("✓ Changes applied\n")

	return nil
}

// Helper formatting functions

// formatDomainList removed - domains are hardcoded in prompt to avoid import cycle

func formatDetectedDomains(domains []struct {
	Name       string  `json:"name"`
	Reasoning  string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
}) string {
	var result []string
	for _, d := range domains {
		result = append(result, fmt.Sprintf("- %s (%.2f): %s", d.Name, d.Confidence, d.Reasoning))
	}
	return strings.Join(result, "\n")
}

func formatDomains(domains []Domain) string {
	var result []string
	for _, d := range domains {
		result = append(result, fmt.Sprintf("- %s: %s (Owner: %s)", d.Name, d.Description, d.Owner))
	}
	return strings.Join(result, "\n")
}

func formatTopEntities(entities []sampler.EntityFrequency) string {
	var result []string
	for i, e := range entities {
		result = append(result, fmt.Sprintf("%d. %s (%d occurrences)", i+1, e.Entity, e.Count))
	}
	return strings.Join(result, "\n")
}

func formatUserAnswers(questions []string, answers []string) string {
	var result []string
	for i, q := range questions {
		if i < len(answers) {
			result = append(result, fmt.Sprintf("Q: %s\nA: %s", q, answers[i]))
		}
	}
	return strings.Join(result, "\n\n")
}
