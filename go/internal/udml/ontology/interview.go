package ontology

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/sampler"
)

// InterviewMode defines the type of interview workflow
type InterviewMode int

const (
	ModeNewOntology InterviewMode = iota // Create new ontology from scratch
	ModeRefineOntology                   // Refine existing ontology
)

// InterviewBuilderV2 provides interactive ontology creation/refinement
type InterviewBuilderV2 struct {
	builder        *OntologyBuilder
	reader         *bufio.Reader
	schema         *OntologySchema
	samples        *sampler.SamplingResult
	mode           InterviewMode
	existingSchema *OntologySchema // For refinement mode
	llmCalls       int
	tokens         int
	nonInteractive bool // Auto-approve all prompts when true
}

// NewInterviewBuilderV2 creates a new interactive ontology builder
func NewInterviewBuilderV2(builder *OntologyBuilder, mode InterviewMode) *InterviewBuilderV2 {
	return &InterviewBuilderV2{
		builder: builder,
		reader:  bufio.NewReader(os.Stdin),
		mode:    mode,
	}
}

// SetNonInteractive enables non-interactive mode (auto-approve all prompts)
func (ib *InterviewBuilderV2) SetNonInteractive(enabled bool) {
	ib.nonInteractive = enabled
}

// StartInterview begins the interactive ontology creation workflow (3 phases with mandatory confirmation)
func (ib *InterviewBuilderV2) StartInterview(ctx context.Context) (*OntologySchema, error) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  INTERACTIVE ONTOLOGY BUILDER - NEW ONTOLOGY")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("This interview creates a custom ontology for your corpus through 3 phases:\n")
	fmt.Println("  Phase 1: Domain Selection - LLM analyzes corpus and suggests domains")
	fmt.Println("           └─ You confirm or adjust domain selection")
	fmt.Println()
	fmt.Println("  Phase 2: Draft Generation - LLM creates entity/relationship schema")
	fmt.Println("           └─ Automated generation based on corpus analysis")
	fmt.Println()
	fmt.Println("  Phase 3: Review & Confirmation - **MANDATORY USER APPROVAL**")
	fmt.Println("           └─ Review complete schema and approve/refine/reject")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80) + "\n")

	// Sample corpus
	if err := ib.sampleCorpus(ctx); err != nil {
		return nil, err
	}

	// Phase 1: Domain Selection (with user confirmation)
	if err := ib.phaseDomainSelection(ctx); err != nil {
		return nil, err
	}

	// Phase 2: Draft Generation (entities + relationships)
	if err := ib.phaseDraftGeneration(ctx); err != nil {
		return nil, err
	}

	// Phase 3: Review & Confirmation (MANDATORY)
	approved, err := ib.phaseReviewAndConfirmation(ctx)
	if err != nil {
		return nil, err
	}

	if !approved {
		fmt.Println("\n❌ Ontology rejected by user. No schema generated.")
		return nil, fmt.Errorf("user rejected ontology schema")
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  ✅ ONTOLOGY APPROVED AND FINALIZED")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("• Schema name: %s\n", ib.schema.Name)
	fmt.Printf("• Domains: %v\n", formatDomainNames(ib.schema.Domains))
	fmt.Printf("• Entity mappings: %d\n", len(ib.schema.ElementEntityMappings))
	fmt.Printf("• Relationship rules: %d\n", len(ib.schema.EntityRelationshipRules))
	fmt.Printf("• LLM calls: %d\n", ib.llmCalls)
	fmt.Printf("• Tokens used: %d\n\n", ib.tokens)

	return ib.schema, nil
}

// StartRefinementInterview begins refinement workflow for existing ontology
func (ib *InterviewBuilderV2) StartRefinementInterview(ctx context.Context, existingSchema *OntologySchema) (*OntologySchema, error) {
	ib.existingSchema = existingSchema
	ib.schema = existingSchema // Start with existing

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  INTERACTIVE ONTOLOGY BUILDER - REFINEMENT MODE")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("This interview refines an existing ontology based on corpus analysis:\n")
	fmt.Println("  Phase 1: Corpus Analysis - LLM analyzes current schema vs corpus")
	fmt.Println("           └─ Identifies gaps, issues, and improvement opportunities")
	fmt.Println()
	fmt.Println("  Phase 2: Suggested Changes - LLM proposes specific modifications")
	fmt.Println("           └─ You review and approve/reject each suggestion")
	fmt.Println()
	fmt.Println("  Phase 3: Apply Changes - Apply approved changes")
	fmt.Println("           └─ Iterative refinement until satisfied")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80) + "\n")

	// Sample corpus
	if err := ib.sampleCorpus(ctx); err != nil {
		return nil, err
	}

	// Phase 1: Analyze existing schema vs corpus
	if err := ib.phaseAnalyzeExisting(ctx); err != nil {
		return nil, err
	}

	// Phase 2: Suggest and apply changes iteratively
	if err := ib.phaseSuggestChanges(ctx); err != nil {
		return nil, err
	}

	// Phase 3: Final confirmation
	approved, err := ib.phaseReviewAndConfirmation(ctx)
	if err != nil {
		return nil, err
	}

	if !approved {
		fmt.Println("\n❌ Changes rejected. Returning original schema.")
		return existingSchema, nil
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  ✅ REFINED ONTOLOGY APPROVED")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("• Schema name: %s\n", ib.schema.Name)
	fmt.Printf("• LLM calls: %d\n", ib.llmCalls)
	fmt.Printf("• Tokens used: %d\n\n", ib.tokens)

	return ib.schema, nil
}

// sampleCorpus samples the document corpus
func (ib *InterviewBuilderV2) sampleCorpus(ctx context.Context) error {
	fmt.Println("📊 Sampling corpus...")
	samples, err := ib.builder.sampler.Sample(ctx)
	if err != nil {
		return fmt.Errorf("sampling failed: %w", err)
	}

	ib.samples = samples
	samples.EntityFrequencies = ib.builder.sampler.AnalyzeEntityFrequencies(samples.Samples)

	fmt.Printf("✓ Sampled %d elements from %d total\n", samples.SampledCount, samples.TotalElements)
	fmt.Printf("✓ Found %d unique entities\n\n", len(samples.EntityFrequencies))

	return nil
}

// formatCorpusStatistics creates a summary of corpus statistics from samples
func (ib *InterviewBuilderV2) formatCorpusStatistics() string {
	if ib.samples == nil || len(ib.samples.Samples) == 0 {
		return "No corpus statistics available"
	}

	// Count element types
	elementTypeCounts := make(map[string]int)
	uniqueDocs := make(map[string]bool)
	for _, sample := range ib.samples.Samples {
		elementTypeCounts[sample.ElementType]++
		uniqueDocs[sample.DocID] = true
	}

	// Format element type distribution (sorted by count descending)
	type typeCount struct {
		Type  string
		Count int
	}
	var typeCounts []typeCount
	for t, c := range elementTypeCounts {
		typeCounts = append(typeCounts, typeCount{t, c})
	}
	sort.Slice(typeCounts, func(i, j int) bool {
		return typeCounts[i].Count > typeCounts[j].Count
	})

	var stats strings.Builder
	stats.WriteString(fmt.Sprintf("## Corpus Statistics (from %d sampled elements)\n\n", len(ib.samples.Samples)))
	stats.WriteString(fmt.Sprintf("- **Total elements sampled**: %d\n", len(ib.samples.Samples)))
	stats.WriteString(fmt.Sprintf("- **Unique documents**: %d\n", len(uniqueDocs)))
	stats.WriteString(fmt.Sprintf("- **Element type distribution**:\n"))
	for _, tc := range typeCounts {
		pct := float64(tc.Count) / float64(len(ib.samples.Samples)) * 100
		stats.WriteString(fmt.Sprintf("  - %s: %d (%.1f%%)\n", tc.Type, tc.Count, pct))
	}
	stats.WriteString("\n")

	return stats.String()
}

// phaseDomainSelection - Hybrid approach: LLM selects from catalog OR proposes new domains if needed
func (ib *InterviewBuilderV2) phaseDomainSelection(ctx context.Context) error {
	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("  PHASE 1: DOMAIN SELECTION (HYBRID MODE)")
	fmt.Println(strings.Repeat("-", 80) + "\n")
	fmt.Println("This phase tries to match corpus to predefined domains.")
	fmt.Println("If catalog coverage is insufficient, LLM may propose new domains.\n")
	fmt.Printf("DEBUG: nonInteractive flag = %v\n\n", ib.nonInteractive)

	sampleTexts := ib.builder.prepareSampleTexts(ib.samples.Samples, 20)

	// Get corpus statistics from samples
	corpusStats := ib.formatCorpusStatistics()

	// Load predefined domains from catalog
	predefinedDomains, err := loadPredefinedDomains()
	if err != nil {
		return fmt.Errorf("failed to load domain catalog: %w", err)
	}

	// Format predefined domains for LLM
	var domainListFormatted strings.Builder
	for name, desc := range predefinedDomains {
		domainListFormatted.WriteString(fmt.Sprintf("  • **%s**: %s\n", name, desc))
	}

	prompt := fmt.Sprintf(`Your task is to identify 3-5 domains that best describe this corpus.

## PREFERRED APPROACH: Use Predefined Catalog Domains

**PREDEFINED CATALOG DOMAINS:**

%s

**IF catalog domains are a good match:**
- Use the EXACT domain names from the catalog above
- Do NOT modify or combine catalog names (e.g., "medical_healthcare" is WRONG - use "medical" OR "healthcare")
- Prefer using multiple separate catalog domains over creating new ones

**IF catalog domains are inadequate:**
- You MAY propose NEW domain names to better represent the corpus
- New domains must follow naming rules:
  * Single word, lowercase (e.g., "genomics", "aviation")
  * OR snake_case for multi-word (e.g., "climate_science", "quantum_computing")
  * Must be specific and well-defined (no generic names like "business", "general")
  * Must NOT be variations of existing catalog domains
  * Must NOT combine existing catalog domains (e.g., "medical_healthcare" is prohibited if catalog has "medical" and "healthcare")

## CRITICAL: Distinguishing Metadata/Format from Domain Topics

**Metadata Format ≠ Domain Content**
**Code Format ≠ "Technical" Domain**

Many documents contain bibliographic metadata or code syntax. The **presence of these formats does NOT determine the domain**. Classify by BUSINESS FUNCTION or SUBJECT MATTER, not by format.

**Common Misclassifications:**

**1. Bibliographic Metadata → library_science (WRONG)**
- ❌ Seeing "ISBN: 978-0-123456-78-9" → identifying as "library_science"
- ❌ Seeing "DOI: 10.1000/xyz123" → identifying as "library_science"
- ❌ Seeing "PMID: 12345678" → identifying as "library_science"
- ❌ Seeing citation templates or Handle System URIs → identifying as "library_science"
- ✓ CORRECT: Medical article with PMID → "medical" domain
- ✓ CORRECT: Economic policy paper with ISBN → "government" domain

**library_science domain specifically requires:**
- Discussion of cataloging processes, workflows, or standards
- Content about circulation systems, lending policies
- Topics on collection development, archival procedures

**2. Code Files → "technical" Domain (WRONG)**
- ❌ Seeing payment processing code → identifying as "technical"
- ❌ Seeing customer relationship code → identifying as "technical"
- ❌ Seeing inventory management code → identifying as "technical"
- ❌ ANY business logic code → identifying as "technical"
- ✓ CORRECT: Payment processing code → "financial" domain
- ✓ CORRECT: CRM code → "sales" or "marketing" domain
- ✓ CORRECT: Inventory code → "supply_chain" or "manufacturing" domain

**technical domain specifically requires:**
- Documentation ABOUT software infrastructure (API specs, architecture docs)
- IT infrastructure documentation (networking, databases, deployment)
- System architecture or technical manuals

**Domain Classification Rules:**
1. Ignore format indicators (metadata, code syntax, file types)
2. Read the actual content to determine WHAT IT'S ABOUT
3. Classify by business function or subject matter
4. Ask: "What business problem does this solve?" or "What topic does this discuss?"

## CORPUS STATISTICS

%s

## YOUR TASK

Analyze these document samples and identify the TOP 3-5 MOST PROMINENT domains:

Sample texts:
%s

For each domain you select or propose:
1. **name**: Exact catalog name OR new proposed name (following rules above)
2. **source**: "catalog" or "proposed_new"
3. **reasoning**: Why this domain? Cite specific evidence from samples
4. **confidence**: 0.0-1.0 based on evidence strength
5. **suggested_owner**: Team/department who owns this domain
6. **key_concepts**: List of key terms/concepts
7. **estimated_coverage**: What %% of corpus belongs to this domain (0.0-1.0)

Ask 2-3 clarifying questions to better understand the corpus and use case.

Return JSON:
{
  "detected_domains": [
    {
      "name": "healthcare",
      "source": "catalog",
      "reasoning": "I see medical procedures, patient care, hospital operations in samples 2, 5, 7",
      "confidence": 0.95,
      "suggested_owner": "Medical Affairs Department",
      "key_concepts": ["surgery", "patient care", "medical procedures"],
      "estimated_coverage": 0.45
    },
    {
      "name": "genomics",
      "source": "proposed_new",
      "reasoning": "Samples 4, 8, 12 contain DNA sequencing, CRISPR, gene editing - not covered by existing catalog domains",
      "confidence": 0.85,
      "suggested_owner": "Genomics Research Team",
      "key_concepts": ["DNA sequencing", "CRISPR", "gene editing"],
      "estimated_coverage": 0.30
    }
  ],
  "clarifying_questions": [
    "What is the primary purpose of these documents?",
    "Who are the main consumers of this data?"
  ],
  "catalog_adequacy": "partial",
  "catalog_adequacy_explanation": "Catalog covers healthcare well, but genomics content requires a new domain"
}

**DECISION CRITERIA:**
- If catalog domains cover >80%% of content → use catalog domains only
- If catalog domains cover 50-80%% → use catalog + propose new domains for gaps
- If catalog domains cover <50%% → propose new domains as primary

IMPORTANT: Order domains by estimated coverage (highest first). Return 3-5 domains total.`, domainListFormatted.String(), corpusStats, sampleTexts)

	// Build system prompt
	systemPrompt := "You are an expert at domain analysis and data mesh principles. Identify distinct domains based on ownership boundaries and vocabulary."

	response, err := ib.builder.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:    2000,
		Temperature:  0.3,
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(response)

	// Parse response
	var result struct {
		DetectedDomains []struct {
			Name              string   `json:"name"`
			Source            string   `json:"source"` // "catalog" or "proposed_new"
			Reasoning         string   `json:"reasoning"`
			Confidence        float64  `json:"confidence"`
			SuggestedOwner    string   `json:"suggested_owner"`
			KeyConcepts       []string `json:"key_concepts"`
			EstimatedCoverage float64  `json:"estimated_coverage"`
		} `json:"detected_domains"`
		ClarifyingQuestions      []string `json:"clarifying_questions"`
		CatalogAdequacy          string   `json:"catalog_adequacy"`
		CatalogAdequacyExplanation string `json:"catalog_adequacy_explanation"`
	}

	if err := ib.builder.extractJSON(response, &result); err != nil {
		return err
	}

	// Validate proposed new domains
	catalogDomains := make([]string, 0)
	proposedDomains := make([]string, 0)

	for _, domain := range result.DetectedDomains {
		if domain.Source == "proposed_new" {
			// Validate the proposed domain name
			if err := validateProposedDomainName(domain.Name, predefinedDomains); err != nil {
				fmt.Printf("⚠️  WARNING: Rejected proposed domain '%s': %v\n", domain.Name, err)
				fmt.Println("    This domain will be excluded from the final selection.\n")
				continue
			}
			proposedDomains = append(proposedDomains, domain.Name)
		} else {
			// Verify catalog domain exists
			if _, exists := predefinedDomains[domain.Name]; !exists {
				fmt.Printf("⚠️  WARNING: Domain '%s' marked as 'catalog' but not found in catalog\n", domain.Name)
				fmt.Println("    This domain will be excluded from the final selection.\n")
				continue
			}
			catalogDomains = append(catalogDomains, domain.Name)
		}
	}

	// Show LLM's analysis
	fmt.Println("🤖 LLM Analysis - Domain Selection:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Catalog Adequacy: %s\n", result.CatalogAdequacy)
	fmt.Printf("Explanation: %s\n\n", result.CatalogAdequacyExplanation)

	if len(catalogDomains) > 0 {
		fmt.Printf("CATALOG DOMAINS (%d):\n", len(catalogDomains))
		for i, domain := range result.DetectedDomains {
			if domain.Source == "catalog" {
				if _, exists := predefinedDomains[domain.Name]; exists {
					fmt.Printf("  %d. %s (confidence: %.2f, coverage: %.0f%%)\n", i+1, domain.Name, domain.Confidence, domain.EstimatedCoverage*100)
					fmt.Printf("     Reasoning: %s\n", domain.Reasoning)
					fmt.Printf("     Suggested Owner: %s\n", domain.SuggestedOwner)
					fmt.Printf("     Key Concepts: %v\n\n", domain.KeyConcepts)
				}
			}
		}
	}

	if len(proposedDomains) > 0 {
		fmt.Printf("PROPOSED NEW DOMAINS (%d):\n", len(proposedDomains))
		for i, domain := range result.DetectedDomains {
			if domain.Source == "proposed_new" {
				if validateProposedDomainName(domain.Name, predefinedDomains) == nil {
					fmt.Printf("  %d. %s (confidence: %.2f, coverage: %.0f%%)\n", i+1, domain.Name, domain.Confidence, domain.EstimatedCoverage*100)
					fmt.Printf("     Reasoning: %s\n", domain.Reasoning)
					fmt.Printf("     Suggested Owner: %s\n", domain.SuggestedOwner)
					fmt.Printf("     Key Concepts: %v\n\n", domain.KeyConcepts)
				}
			}
		}
	}
	fmt.Println()

	// Ask clarifying questions
	userAnswers := []string{}
	if ib.nonInteractive {
		fmt.Println("DEBUG: Non-interactive mode enabled - auto-skipping clarifying questions")
	}
	for _, question := range result.ClarifyingQuestions {
		fmt.Printf("🤖 LLM: %s\n", question)
		var answer string
		if ib.nonInteractive {
			answer = "" // Auto-skip clarifying questions in non-interactive mode
			fmt.Print("   You: [auto-skipped]\n")
		} else {
			fmt.Print("   You: ")
			var err error
			answer, err = ib.reader.ReadString('\n')
			if err != nil {
				return err
			}
		}
		userAnswers = append(userAnswers, strings.TrimSpace(answer))
		fmt.Println()
	}

	// LLM finalizes domains based on user answers
	// Extract exact detected domain names to enforce in refinement
	detectedNames := make([]string, len(result.DetectedDomains))
	for i, d := range result.DetectedDomains {
		detectedNames[i] = d.Name
	}

	refinementPrompt := fmt.Sprintf(`Based on user answers, finalize domain selection by providing descriptions and owners.

**CRITICAL CONSTRAINT - DO NOT MODIFY DOMAIN NAMES:**
You MUST use ONLY these exact domain names: %v

Do NOT create new names. Do NOT combine domains. Do NOT modify these names in ANY way.
You may ONLY add descriptions, owners, and key concepts for these exact names.

Initial analysis:
%s

User answers:
%s

Return final domains as JSON array using ONLY the exact names from the constraint above:
[
  {
    "name": "financial",
    "description": "Financial performance and reporting",
    "owner": "CFO Office",
    "key_concepts": ["revenue", "profit", "EBITDA"]
  }
]

VALIDATION REQUIREMENT: Before responding, verify EVERY "name" field is EXACTLY one of: %v
If you create ANY other name, the response will be rejected.`, detectedNames, formatDetectedDomainsV2(result.DetectedDomains), formatUserAnswersV2(result.ClarifyingQuestions, userAnswers), detectedNames)

	finalResponse, err := ib.builder.llmClient.Complete(ctx, refinementPrompt, LLMOptions{
		MaxTokens:    1500,
		Temperature:  0.2,
		SystemPrompt: "Finalize domain selection based on analysis and user input.",
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

	// Show final domains and ask for confirmation
	fmt.Println("📋 Final Domain Selection:")
	fmt.Println(strings.Repeat("-", 80))
	for _, d := range domains {
		fmt.Printf("  ✓ %s - %s\n", d.Name, d.Description)
		fmt.Printf("    Owner: %s\n", d.Owner)
	}
	fmt.Println()

	var approval string
	if ib.nonInteractive {
		approval = "yes"
		fmt.Print("❓ Approve these domains? (yes/no): yes [auto-approved]\n")
	} else {
		fmt.Print("❓ Approve these domains? (yes/no): ")
		var err error
		approval, err = ib.reader.ReadString('\n')
		if err != nil {
			return err
		}
	}

	if strings.ToLower(strings.TrimSpace(approval)) != "yes" {
		fmt.Print("\n   What changes would you like? (describe): ")
		changes, err := ib.reader.ReadString('\n')
		if err != nil {
			return err
		}

		// Apply user changes
		adjustmentPrompt := fmt.Sprintf(`User wants these changes to domains: %s

Current domains:
%s

Apply changes and return updated domain list as JSON array.`, strings.TrimSpace(changes), formatDomainsJSON(domains))

		adjustedResponse, err := ib.builder.llmClient.Complete(ctx, adjustmentPrompt, LLMOptions{
			MaxTokens:   1500,
			Temperature: 0.2,
		})
		if err != nil {
			return err
		}
		ib.llmCalls++
		ib.tokens += len(adjustedResponse)

		if err := ib.builder.extractJSON(adjustedResponse, &domains); err != nil {
			return err
		}

		fmt.Println("\n  ✓ Domains adjusted")
	}

	// Initialize schema with confirmed domains
	ib.schema = &OntologySchema{
		Name:                    ib.builder.config.SchemaName,
		Version:                 ib.builder.config.SchemaVersion,
		LLMModel:                ib.builder.config.LLMModel, // Preserve LLM model configuration
		Domains:                 domains,
		ElementEntityMappings:   []ElementEntityMapping{},
		EntityRelationshipRules: []EntityRelationshipRule{},
		CreatedAt:               time.Now(),
	}

	fmt.Println("\n  ✅ Domains confirmed\n")

	return nil
}

// phaseDraftGeneration - LLM generates complete entity/relationship schema (automated)
func (ib *InterviewBuilderV2) phaseDraftGeneration(ctx context.Context) error {
	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("  PHASE 2: DRAFT GENERATION")
	fmt.Println(strings.Repeat("-", 80) + "\n")

	fmt.Println("🤖 LLM is generating entity types and relationships...")
	fmt.Println("   (this may take 30-60 seconds)\n")

	topEntities := ib.samples.GetTopEntities(ib.builder.config.TopEntityCount)

	// Generate entities using builder's existing logic
	entityMappings, calls1, tokens1, err := ib.builder.defineEntityTypes(ctx, ib.samples, topEntities, ib.schema.Domains)
	if err != nil {
		return err
	}
	ib.llmCalls += calls1
	ib.tokens += tokens1

	ib.schema.ElementEntityMappings = entityMappings

	fmt.Printf("  ✓ Generated %d entity mappings\n", len(entityMappings))

	// Generate relationships using builder's existing logic
	relationshipRules, calls2, tokens2, err := ib.builder.defineRelationshipTypes(ctx, ib.samples, entityMappings)
	if err != nil {
		return err
	}
	ib.llmCalls += calls2
	ib.tokens += tokens2

	ib.schema.EntityRelationshipRules = relationshipRules

	fmt.Printf("  ✓ Generated %d relationship rules\n", len(relationshipRules))
	fmt.Println("\n  ✅ Draft schema complete\n")

	return nil
}

// phaseReviewAndConfirmation - MANDATORY user approval of schema (returns approved bool)
func (ib *InterviewBuilderV2) phaseReviewAndConfirmation(ctx context.Context) (bool, error) {
	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("  PHASE 3: REVIEW & CONFIRMATION (MANDATORY)")
	fmt.Println(strings.Repeat("-", 80) + "\n")

	// Display complete schema summary
	fmt.Println("📋 COMPLETE ONTOLOGY SCHEMA")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("\nDOMAINS (%d):\n", len(ib.schema.Domains))
	fmt.Println(strings.Repeat("-", 80))
	for _, d := range ib.schema.Domains {
		fmt.Printf("  • %s - %s\n", d.Name, d.Description)
		fmt.Printf("    Owner: %s\n", d.Owner)
	}

	fmt.Printf("\nENTITY TYPES (%d):\n", len(ib.schema.ElementEntityMappings))
	fmt.Println(strings.Repeat("-", 80))
	// Group by domain
	byDomain := make(map[string][]ElementEntityMapping)
	for _, m := range ib.schema.ElementEntityMappings {
		byDomain[m.Domain] = append(byDomain[m.Domain], m)
	}
	for domain, mappings := range byDomain {
		fmt.Printf("  [%s domain]\n", domain)
		for _, m := range mappings {
			fmt.Printf("    • %s - %s (confidence: %.2f)\n", m.EntityType, m.Description, m.Confidence)
			fmt.Printf("      Elements: %v\n", m.ElementTypes)
			fmt.Printf("      Rules: %d extraction patterns\n", len(m.ExtractionRules))
		}
	}

	fmt.Printf("\nRELATIONSHIPS (%d):\n", len(ib.schema.EntityRelationshipRules))
	fmt.Println(strings.Repeat("-", 80))
	for _, r := range ib.schema.EntityRelationshipRules {
		fmt.Printf("  • %s: %s → %s (%s)\n", r.Name, r.SourceEntityType, r.TargetEntityType, r.RelationshipType)
		fmt.Printf("    %s\n", r.Description)
		fmt.Printf("    Confidence: %.2f | Patterns: %d\n", r.Confidence, len(r.ExtractionPatterns))
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("\n❓ APPROVAL REQUIRED")
	fmt.Println("   Your options:")
	fmt.Println("     1. 'approve' - Accept schema as-is and finalize")
	fmt.Println("     2. 'refine'  - Make iterative changes before approving")
	fmt.Println("     3. 'reject'  - Reject schema entirely")
	fmt.Println()

	for {
		var decision string
		if ib.nonInteractive {
			decision = "approve"
			fmt.Print("   Your decision (approve/refine/reject): approve [auto-approved]\n")
		} else {
			fmt.Print("   Your decision (approve/refine/reject): ")
			var err error
			decision, err = ib.reader.ReadString('\n')
			if err != nil {
				return false, err
			}
		}
		decision = strings.ToLower(strings.TrimSpace(decision))

		switch decision {
		case "approve":
			fmt.Println("\n  ✅ Schema APPROVED")
			return true, nil

		case "reject":
			fmt.Println("\n  ❌ Schema REJECTED")
			return false, nil

		case "refine":
			if err := ib.iterativeRefinement(ctx); err != nil {
				return false, err
			}
			// After refinement, ask again
			fmt.Println("\n❓ Review refined schema - approve/refine again/reject?")
			continue

		default:
			fmt.Println("   Invalid choice. Please enter 'approve', 'refine', or 'reject'.")
			continue
		}
	}
}

// iterativeRefinement - User describes changes, LLM applies them
func (ib *InterviewBuilderV2) iterativeRefinement(ctx context.Context) error {
	fmt.Println("\n  🔧 REFINEMENT MODE")
	fmt.Println("     Describe the changes you want (be specific):")
	fmt.Print("     > ")

	changes, err := ib.reader.ReadString('\n')
	if err != nil {
		return err
	}

	schemaJSON, _ := json.MarshalIndent(ib.schema, "", "  ")

	prompt := fmt.Sprintf(`User requests these changes to the ontology schema:

%s

Current schema:
%s

Apply the requested changes and return the COMPLETE updated schema as JSON.
Preserve all fields and structure.`, strings.TrimSpace(changes), string(schemaJSON))

	response, err := ib.builder.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:   8000,
		Temperature: 0.2,
		SystemPrompt: "Apply user-requested changes to ontology schema while preserving structure and quality.",
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(response)

	var updatedSchema OntologySchema
	if err := ib.builder.extractJSON(response, &updatedSchema); err != nil {
		return fmt.Errorf("failed to parse updated schema: %w", err)
	}

	ib.schema = &updatedSchema

	fmt.Println("\n  ✓ Changes applied")

	return nil
}

// phaseAnalyzeExisting - Analyze existing schema against corpus
func (ib *InterviewBuilderV2) phaseAnalyzeExisting(ctx context.Context) error {
	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("  PHASE 1: ANALYZE EXISTING SCHEMA")
	fmt.Println(strings.Repeat("-", 80) + "\n")

	sampleTexts := ib.builder.prepareSampleTexts(ib.samples.Samples, 20)
	schemaJSON, _ := json.MarshalIndent(ib.existingSchema, "", "  ")

	prompt := fmt.Sprintf(`Analyze this existing ontology schema against the corpus samples.

Existing schema:
%s

Corpus samples:
%s

Identify:
1. **Gaps** - Entity types or relationships missing from schema but present in corpus
2. **Issues** - Rules that may not work well (bad regex, wrong confidence, etc.)
3. **Improvements** - Ways to enhance extraction quality

Return JSON:
{
  "gaps": [
    {
      "type": "missing_entity",
      "description": "Email addresses appear frequently but aren't extracted",
      "evidence": "Seen in samples 3, 7, 12"
    }
  ],
  "issues": [
    {
      "type": "bad_rule",
      "location": "entity_type: company, rule 2",
      "description": "Regex pattern too broad, will match false positives",
      "severity": "high"
    }
  ],
  "improvements": [
    {
      "description": "Add text_similarity rule for contextual extraction of executives",
      "expected_benefit": "Better recall for person entities in narrative text"
    }
  ]
}`, string(schemaJSON), sampleTexts)

	response, err := ib.builder.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:    3000,
		Temperature:  0.3,
		SystemPrompt: "You are an expert at ontology quality analysis. Identify gaps and issues systematically.",
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(response)

	var analysis struct {
		Gaps         []map[string]string `json:"gaps"`
		Issues       []map[string]string `json:"issues"`
		Improvements []map[string]string `json:"improvements"`
	}

	if err := ib.builder.extractJSON(response, &analysis); err != nil {
		return err
	}

	// Display analysis
	fmt.Println("🤖 LLM Analysis Results:")
	fmt.Println(strings.Repeat("-", 80))

	if len(analysis.Gaps) > 0 {
		fmt.Printf("\n  GAPS FOUND (%d):\n", len(analysis.Gaps))
		for i, gap := range analysis.Gaps {
			fmt.Printf("    %d. %s\n", i+1, gap["description"])
			fmt.Printf("       Evidence: %s\n", gap["evidence"])
		}
	}

	if len(analysis.Issues) > 0 {
		fmt.Printf("\n  ISSUES FOUND (%d):\n", len(analysis.Issues))
		for i, issue := range analysis.Issues {
			fmt.Printf("    %d. [%s] %s\n", i+1, issue["severity"], issue["description"])
			fmt.Printf("       Location: %s\n", issue["location"])
		}
	}

	if len(analysis.Improvements) > 0 {
		fmt.Printf("\n  IMPROVEMENTS SUGGESTED (%d):\n", len(analysis.Improvements))
		for i, imp := range analysis.Improvements {
			fmt.Printf("    %d. %s\n", i+1, imp["description"])
			fmt.Printf("       Benefit: %s\n", imp["expected_benefit"])
		}
	}

	fmt.Println()

	return nil
}

// phaseSuggestChanges - LLM suggests specific changes, user approves/rejects each
func (ib *InterviewBuilderV2) phaseSuggestChanges(ctx context.Context) error {
	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("  PHASE 2: SUGGESTED CHANGES")
	fmt.Println(strings.Repeat("-", 80) + "\n")

	schemaJSON, _ := json.MarshalIndent(ib.schema, "", "  ")
	sampleTexts := ib.builder.prepareSampleTexts(ib.samples.Samples, 15)

	prompt := fmt.Sprintf(`Based on the analysis, propose specific changes to fix issues and fill gaps.

Current schema:
%s

Corpus samples:
%s

Return JSON array of specific changes:
[
  {
    "change_type": "add_entity_mapping",
    "reasoning": "Email addresses appear frequently but not extracted",
    "proposed_change": {
      "entity_type": "email",
      "domain": "legal",
      "description": "Email addresses",
      "element_types": ["paragraph", "table_cell"],
      "confidence": 0.95,
      "extraction_rules": [
        {
          "type": "regex_pattern",
          "pattern": "\\b[A-Za-z0-9._%%-+]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b"
        }
      ]
    }
  },
  {
    "change_type": "fix_rule",
    "reasoning": "Company regex too broad, matches false positives",
    "location": "entity_type: company, domain: financial",
    "proposed_change": {
      "remove_rule_index": 1,
      "add_rule": {
        "type": "regex_pattern",
        "pattern": "\\b[A-Z][a-z]+(?:\\s+[A-Z][a-z]+)*\\s+(?:Inc|Corp|LLC|Ltd)\\.?\\b"
      }
    }
  }
]`, string(schemaJSON), sampleTexts)

	response, err := ib.builder.llmClient.Complete(ctx, prompt, LLMOptions{
		MaxTokens:    4000,
		Temperature:  0.3,
		SystemPrompt: "Propose specific, actionable changes to improve ontology quality.",
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(response)

	var suggestedChanges []map[string]interface{}
	if err := ib.builder.extractJSON(response, &suggestedChanges); err != nil {
		return err
	}

	// Present changes one by one for approval
	fmt.Printf("🤖 LLM suggests %d changes:\n\n", len(suggestedChanges))

	approvedChanges := []map[string]interface{}{}

	for i, change := range suggestedChanges {
		fmt.Printf("  Change %d/%d:\n", i+1, len(suggestedChanges))
		fmt.Printf("    Type: %v\n", change["change_type"])
		fmt.Printf("    Reason: %v\n", change["reasoning"])

		changeJSON, _ := json.MarshalIndent(change["proposed_change"], "    ", "  ")
		fmt.Printf("    Proposed:\n%s\n", string(changeJSON))

		fmt.Print("\n    Approve this change? (yes/no/skip): ")
		approval, err := ib.reader.ReadString('\n')
		if err != nil {
			return err
		}

		switch strings.ToLower(strings.TrimSpace(approval)) {
		case "yes":
			approvedChanges = append(approvedChanges, change)
			fmt.Println("    ✓ Approved")
		case "skip":
			fmt.Println("    ⊘ Skipped (continue to next change)")
			continue
		default:
			fmt.Println("    ✗ Rejected")
		}

		fmt.Println()
	}

	if len(approvedChanges) == 0 {
		fmt.Println("  No changes approved. Schema unchanged.\n")
		return nil
	}

	// Apply approved changes
	fmt.Printf("\n  Applying %d approved changes...\n", len(approvedChanges))

	changesJSON, _ := json.MarshalIndent(approvedChanges, "", "  ")
	applyPrompt := fmt.Sprintf(`Apply these approved changes to the schema:

Changes to apply:
%s

Current schema:
%s

Return the COMPLETE updated schema as JSON.`, string(changesJSON), string(schemaJSON))

	applyResponse, err := ib.builder.llmClient.Complete(ctx, applyPrompt, LLMOptions{
		MaxTokens:   8000,
		Temperature: 0.2,
	})
	if err != nil {
		return err
	}
	ib.llmCalls++
	ib.tokens += len(applyResponse)

	var updatedSchema OntologySchema
	if err := ib.builder.extractJSON(applyResponse, &updatedSchema); err != nil {
		return fmt.Errorf("failed to parse updated schema: %w", err)
	}

	ib.schema = &updatedSchema

	fmt.Println("  ✅ Changes applied successfully\n")

	return nil
}

// Helper functions

func formatDetectedDomainsV2(domains []struct {
	Name              string   `json:"name"`
	Source            string   `json:"source"`
	Reasoning         string   `json:"reasoning"`
	Confidence        float64  `json:"confidence"`
	SuggestedOwner    string   `json:"suggested_owner"`
	KeyConcepts       []string `json:"key_concepts"`
	EstimatedCoverage float64  `json:"estimated_coverage"`
}) string {
	var result []string
	for _, d := range domains {
		sourceLabel := ""
		if d.Source == "proposed_new" {
			sourceLabel = " [NEW]"
		}
		result = append(result, fmt.Sprintf("- %s%s (confidence: %.2f, coverage: %.0f%%): %s | Owner: %s",
			d.Name, sourceLabel, d.Confidence, d.EstimatedCoverage*100, d.Reasoning, d.SuggestedOwner))
	}
	return strings.Join(result, "\n")
}

func formatUserAnswersV2(questions []string, answers []string) string {
	var result []string
	for i, q := range questions {
		if i < len(answers) {
			result = append(result, fmt.Sprintf("Q: %s\nA: %s", q, answers[i]))
		}
	}
	return strings.Join(result, "\n\n")
}

func formatDomainsJSON(domains []Domain) string {
	data, _ := json.MarshalIndent(domains, "", "  ")
	return string(data)
}

func formatDomainNames(domains []Domain) []string {
	names := make([]string, len(domains))
	for i, d := range domains {
		names[i] = d.Name
	}
	return names
}

func formatPredefinedDomainsForValidation(predefinedDomains map[string]string) string {
	var result []string
	for name := range predefinedDomains {
		result = append(result, name)
	}
	return strings.Join(result, ", ")
}

func validateProposedDomainName(name string, predefinedDomains map[string]string) error {
	// Check if already in catalog
	if _, exists := predefinedDomains[name]; exists {
		return fmt.Errorf("domain '%s' already exists in catalog", name)
	}

	// Check format: lowercase letters, numbers, and underscores only
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	// Must be lowercase alphanumeric with optional underscores
	validFormat := true
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_') {
			validFormat = false
			break
		}
	}
	if !validFormat {
		return fmt.Errorf("invalid format - must be lowercase alphanumeric with optional underscores (e.g., 'genomics' or 'climate_science')")
	}

	// Check for prohibited generic names
	genericNames := []string{"business", "general", "other", "misc", "miscellaneous", "data", "information"}
	for _, generic := range genericNames {
		if name == generic {
			return fmt.Errorf("generic name '%s' not allowed - be more specific", name)
		}
	}

	// Check for compound names combining existing catalog domains
	parts := strings.Split(name, "_")
	if len(parts) > 1 {
		// Check if parts match catalog domains
		for _, part := range parts {
			if _, exists := predefinedDomains[part]; exists {
				return fmt.Errorf("compound name '%s' combines existing catalog domain '%s' - choose a single specific domain instead", name, part)
			}
		}
	}

	return nil
}
