package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kennethstott/go-doc-go/internal/udml/ontology"
)

func main() {
	// Command line flags
	var (
		parquetPath = flag.String("parquet", "", "Path to UDML Parquet storage (required)")
		schemaName  = flag.String("name", "", "Schema name (required)")
		domain      = flag.String("domain", "", "Domain name (optional, LLM will auto-detect if not provided)")
		outputPath  = flag.String("output", "", "Output path for schema JSON (default: <name>.ontology.json)")
		sampleSize  = flag.Int("sample-size", 100, "Number of elements to sample for analysis")
		topEntities = flag.Int("top-entities", 50, "Number of top entities to analyze for aliases")
		llmProvider = flag.String("llm-provider", "anthropic", "LLM provider (anthropic)")
		llmModel    = flag.String("llm-model", "", "LLM model (default: claude-3-5-sonnet-20241022)")
		apiKey      = flag.String("api-key", "", "LLM API key (or set ANTHROPIC_API_KEY env var)")
		nonInteractive = flag.Bool("non-interactive", false, "Skip interactive refinement, just generate and save")
	)

	flag.Parse()

	// Validate required flags
	if *parquetPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --parquet is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *schemaName == "" {
		fmt.Fprintf(os.Stderr, "Error: --name is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Get API key from env if not provided
	if *apiKey == "" {
		*apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if *apiKey == "" {
			*apiKey = os.Getenv("CLAUDE_API_KEY")
		}
		if *apiKey == "" {
			fmt.Fprintf(os.Stderr, "Error: --api-key flag or ANTHROPIC_API_KEY/CLAUDE_API_KEY env var required\n")
			os.Exit(1)
		}
	}

	// Set default output path
	if *outputPath == "" {
		*outputPath = fmt.Sprintf("%s.ontology.json", *schemaName)
	}

	// Check if Parquet path exists
	if _, err := os.Stat(*parquetPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Parquet path not found: %s\n", *parquetPath)
		os.Exit(1)
	}

	log.Println("========================================")
	log.Println("UDML Ontology Builder")
	log.Println("========================================")
	log.Printf("Parquet path: %s", *parquetPath)
	log.Printf("Schema name: %s", *schemaName)
	log.Printf("Sample size: %d", *sampleSize)
	log.Printf("Top entities: %d", *topEntities)
	log.Printf("LLM provider: %s", *llmProvider)
	if *llmModel != "" {
		log.Printf("LLM model: %s", *llmModel)
	}
	log.Println("========================================")

	// Create builder config
	config := ontology.BuilderConfig{
		ParquetPath:         *parquetPath,
		SampleSize:          *sampleSize,
		LLMProvider:         *llmProvider,
		LLMModel:            *llmModel,
		LLMAPIKey:           *apiKey,
		TopEntityCount:      *topEntities,
		MinEntityFrequency:  5,
		ConfidenceThreshold: 0.7,
		SchemaName:          *schemaName,
		SchemaVersion:       "1.0.0",
		Domain:              *domain,
	}

	// Create builder
	builder, err := ontology.NewOntologyBuilder(config)
	if err != nil {
		log.Fatalf("Failed to create ontology builder: %v", err)
	}
	defer builder.Close()

	// Run build process
	ctx := context.Background()
	log.Println("\nStarting ontology build process...")
	result, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	// Display build results
	log.Println("\n========================================")
	log.Println("BUILD COMPLETE")
	log.Println("========================================")
	log.Printf("Build time: %v", result.BuildTime)
	log.Printf("LLM calls: %d", result.LLMCallCount)
	log.Printf("Total tokens: %d", result.TotalLLMTokens)
	log.Println("\nAnalysis log:")
	for _, entry := range result.AnalysisLog {
		log.Printf("  %s", entry)
	}

	// Display draft schema summary
	schema := result.DraftSchema
	log.Println("\n========================================")
	log.Println("DRAFT ONTOLOGY SCHEMA")
	log.Println("========================================")
	log.Printf("Domain: %s", schema.Domain)
	log.Printf("Key concepts: %s", strings.Join(schema.KeyConcepts, ", "))
	log.Printf("Entity types: %d", len(schema.ElementEntityMappings))
	log.Printf("Relationship types: %d", len(schema.EntityRelationshipRules))

	log.Println("\nEntity Types:")
	for i, mapping := range schema.ElementEntityMappings {
		log.Printf("  %d. %s - %s", i+1, mapping.EntityType, mapping.Description)
		log.Printf("     Element types: %s", strings.Join(mapping.ElementTypes, ", "))
		log.Printf("     Extraction rules: %d", len(mapping.ExtractionRules))

		// Show sample of extraction rules
		for j, rule := range mapping.ExtractionRules {
			if j >= 2 { // Show first 2 rules
				log.Printf("       ... (%d more rules)", len(mapping.ExtractionRules)-2)
				break
			}
			switch rule.Type {
			case ontology.RuleTypeKeyword:
				keywords := rule.Keywords
				if len(keywords) > 3 {
					keywords = append(keywords[:3], "...")
				}
				log.Printf("       - keyword_match: %s (confidence: %.2f)", strings.Join(keywords, ", "), rule.Confidence)
			case ontology.RuleTypeRegex:
				log.Printf("       - regex_pattern: %s (confidence: %.2f)", rule.Pattern, rule.Confidence)
			case ontology.RuleTypeMetadata:
				log.Printf("       - metadata_field: %s (confidence: %.2f)", rule.FieldPath, rule.Confidence)
			}
		}
	}

	log.Println("\nRelationship Types:")
	for i, rule := range schema.EntityRelationshipRules {
		log.Printf("  %d. %s: %s -> %s (%s)", i+1, rule.Name, rule.SourceEntityType, rule.TargetEntityType, rule.RelationshipType)
		log.Printf("     %s", rule.Description)
	}

	// Interactive refinement
	if !*nonInteractive {
		log.Println("\n========================================")
		log.Println("INTERACTIVE REFINEMENT")
		log.Println("========================================")

		if !confirm("Would you like to review and refine the schema interactively?") {
			log.Println("Skipping interactive refinement.")
		} else {
			// Interactive refinement loop
			refineSchema(schema, result)
		}
	}

	// Save schema
	if err := saveSchema(schema, *outputPath); err != nil {
		log.Fatalf("Failed to save schema: %v", err)
	}

	log.Println("\n========================================")
	log.Println("SUCCESS")
	log.Println("========================================")
	log.Printf("Ontology schema saved to: %s", *outputPath)
	log.Println("\nYou can now use this schema with the rule-based extractor:")
	log.Printf("  extractor := ontology.NewRuleBasedExtractor(schema)")
	log.Printf("  ontology, err := extractor.ExtractFromElements(ctx, docID, elements)")
}

// refineSchema provides interactive refinement of the schema
func refineSchema(schema *ontology.OntologySchema, result *ontology.BuildResult) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\nRefinement options:")
		fmt.Println("  1. Edit entity type")
		fmt.Println("  2. Add entity type")
		fmt.Println("  3. Remove entity type")
		fmt.Println("  4. Edit relationship rule")
		fmt.Println("  5. Add relationship rule")
		fmt.Println("  6. Remove relationship rule")
		fmt.Println("  7. Edit domain/metadata")
		fmt.Println("  8. Review top entities")
		fmt.Println("  9. Done (save and exit)")
		fmt.Print("\nChoice: ")

		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)

		switch choice {
		case "1":
			editEntityType(schema, reader, result)
		case "2":
			addEntityType(schema, reader, result)
		case "3":
			removeEntityType(schema, reader, result)
		case "4":
			editRelationshipRule(schema, reader, result)
		case "5":
			addRelationshipRule(schema, reader, result)
		case "6":
			removeRelationshipRule(schema, reader, result)
		case "7":
			editMetadata(schema, reader, result)
		case "8":
			reviewTopEntities(result)
		case "9":
			return
		default:
			fmt.Println("Invalid choice, please try again.")
		}
	}
}

// editEntityType allows editing an existing entity type
func editEntityType(schema *ontology.OntologySchema, reader *bufio.Reader, result *ontology.BuildResult) {
	if len(schema.ElementEntityMappings) == 0 {
		fmt.Println("No entity types to edit.")
		return
	}

	fmt.Println("\nEntity types:")
	for i, mapping := range schema.ElementEntityMappings {
		fmt.Printf("  %d. %s - %s\n", i+1, mapping.EntityType, mapping.Description)
	}

	fmt.Print("Select entity type number: ")
	line, _ := reader.ReadString('\n')
	idx, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || idx < 1 || idx > len(schema.ElementEntityMappings) {
		fmt.Println("Invalid selection.")
		return
	}

	mapping := &schema.ElementEntityMappings[idx-1]

	fmt.Printf("\nEditing: %s\n", mapping.EntityType)
	fmt.Printf("Current description: %s\n", mapping.Description)
	fmt.Print("New description (or press Enter to keep): ")
	line, _ = reader.ReadString('\n')
	newDesc := strings.TrimSpace(line)
	if newDesc != "" {
		mapping.Description = newDesc
		result.UserRefinements = append(result.UserRefinements,
			fmt.Sprintf("Updated description for %s", mapping.EntityType))
		fmt.Println("Description updated.")
	}

	// Edit extraction rules
	if confirm("Would you like to edit extraction rules?") {
		editExtractionRules(mapping, reader, result)
	}
}

// editExtractionRules allows editing extraction rules for an entity type
func editExtractionRules(mapping *ontology.ElementEntityMapping, reader *bufio.Reader, result *ontology.BuildResult) {
	for {
		fmt.Printf("\nExtraction rules for %s:\n", mapping.EntityType)
		for i, rule := range mapping.ExtractionRules {
			fmt.Printf("  %d. %s ", i+1, rule.Type)
			switch rule.Type {
			case ontology.RuleTypeKeyword:
				fmt.Printf("(keywords: %s)\n", strings.Join(rule.Keywords, ", "))
			case ontology.RuleTypeRegex:
				fmt.Printf("(pattern: %s)\n", rule.Pattern)
			case ontology.RuleTypeMetadata:
				fmt.Printf("(field: %s)\n", rule.FieldPath)
			}
		}

		fmt.Println("\nOptions:")
		fmt.Println("  1. Add keyword rule")
		fmt.Println("  2. Add regex rule")
		fmt.Println("  3. Remove rule")
		fmt.Println("  4. Done")
		fmt.Print("Choice: ")

		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)

		switch choice {
		case "1":
			addKeywordRule(mapping, reader, result)
		case "2":
			addRegexRule(mapping, reader, result)
		case "3":
			removeRule(mapping, reader, result)
		case "4":
			return
		default:
			fmt.Println("Invalid choice.")
		}
	}
}

// addKeywordRule adds a keyword extraction rule
func addKeywordRule(mapping *ontology.ElementEntityMapping, reader *bufio.Reader, result *ontology.BuildResult) {
	fmt.Print("Enter keywords (comma-separated): ")
	line, _ := reader.ReadString('\n')
	keywordsStr := strings.TrimSpace(line)
	if keywordsStr == "" {
		return
	}

	keywords := strings.Split(keywordsStr, ",")
	for i := range keywords {
		keywords[i] = strings.TrimSpace(keywords[i])
	}

	fmt.Print("Enter confidence (0.0-1.0, default 0.8): ")
	line, _ = reader.ReadString('\n')
	confStr := strings.TrimSpace(line)
	conf := 0.8
	if confStr != "" {
		if parsed, err := strconv.ParseFloat(confStr, 64); err == nil {
			conf = parsed
		}
	}

	rule := ontology.ExtractionRule{
		Type:       ontology.RuleTypeKeyword,
		Keywords:   keywords,
		Confidence: conf,
	}

	mapping.ExtractionRules = append(mapping.ExtractionRules, rule)
	result.UserRefinements = append(result.UserRefinements,
		fmt.Sprintf("Added keyword rule to %s: %s", mapping.EntityType, strings.Join(keywords, ", ")))
	fmt.Println("Keyword rule added.")
}

// addRegexRule adds a regex extraction rule
func addRegexRule(mapping *ontology.ElementEntityMapping, reader *bufio.Reader, result *ontology.BuildResult) {
	fmt.Print("Enter regex pattern: ")
	line, _ := reader.ReadString('\n')
	pattern := strings.TrimSpace(line)
	if pattern == "" {
		return
	}

	fmt.Print("Enter confidence (0.0-1.0, default 0.8): ")
	line, _ = reader.ReadString('\n')
	confStr := strings.TrimSpace(line)
	conf := 0.8
	if confStr != "" {
		if parsed, err := strconv.ParseFloat(confStr, 64); err == nil {
			conf = parsed
		}
	}

	rule := ontology.ExtractionRule{
		Type:       ontology.RuleTypeRegex,
		Pattern:    pattern,
		Confidence: conf,
	}

	mapping.ExtractionRules = append(mapping.ExtractionRules, rule)
	result.UserRefinements = append(result.UserRefinements,
		fmt.Sprintf("Added regex rule to %s: %s", mapping.EntityType, pattern))
	fmt.Println("Regex rule added.")
}

// removeRule removes an extraction rule
func removeRule(mapping *ontology.ElementEntityMapping, reader *bufio.Reader, result *ontology.BuildResult) {
	if len(mapping.ExtractionRules) == 0 {
		fmt.Println("No rules to remove.")
		return
	}

	fmt.Print("Enter rule number to remove: ")
	line, _ := reader.ReadString('\n')
	idx, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || idx < 1 || idx > len(mapping.ExtractionRules) {
		fmt.Println("Invalid selection.")
		return
	}

	removedRule := mapping.ExtractionRules[idx-1]
	mapping.ExtractionRules = append(mapping.ExtractionRules[:idx-1], mapping.ExtractionRules[idx:]...)
	result.UserRefinements = append(result.UserRefinements,
		fmt.Sprintf("Removed %s rule from %s", removedRule.Type, mapping.EntityType))
	fmt.Println("Rule removed.")
}

// addEntityType adds a new entity type
func addEntityType(schema *ontology.OntologySchema, reader *bufio.Reader, result *ontology.BuildResult) {
	fmt.Print("Entity type name: ")
	line, _ := reader.ReadString('\n')
	entityType := strings.TrimSpace(line)
	if entityType == "" {
		return
	}

	fmt.Print("Description: ")
	line, _ = reader.ReadString('\n')
	description := strings.TrimSpace(line)

	fmt.Print("Element types (comma-separated, e.g., paragraph,heading): ")
	line, _ = reader.ReadString('\n')
	elementTypes := strings.Split(strings.TrimSpace(line), ",")
	for i := range elementTypes {
		elementTypes[i] = strings.TrimSpace(elementTypes[i])
	}

	mapping := ontology.ElementEntityMapping{
		EntityType:      entityType,
		Description:     description,
		ElementTypes:    elementTypes,
		ExtractionRules: []ontology.ExtractionRule{},
	}

	schema.ElementEntityMappings = append(schema.ElementEntityMappings, mapping)
	result.UserRefinements = append(result.UserRefinements,
		fmt.Sprintf("Added entity type: %s", entityType))
	fmt.Println("Entity type added.")

	if confirm("Would you like to add extraction rules now?") {
		editExtractionRules(&schema.ElementEntityMappings[len(schema.ElementEntityMappings)-1], reader, result)
	}
}

// removeEntityType removes an entity type
func removeEntityType(schema *ontology.OntologySchema, reader *bufio.Reader, result *ontology.BuildResult) {
	if len(schema.ElementEntityMappings) == 0 {
		fmt.Println("No entity types to remove.")
		return
	}

	fmt.Println("\nEntity types:")
	for i, mapping := range schema.ElementEntityMappings {
		fmt.Printf("  %d. %s\n", i+1, mapping.EntityType)
	}

	fmt.Print("Select entity type number to remove: ")
	line, _ := reader.ReadString('\n')
	idx, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || idx < 1 || idx > len(schema.ElementEntityMappings) {
		fmt.Println("Invalid selection.")
		return
	}

	removed := schema.ElementEntityMappings[idx-1]
	schema.ElementEntityMappings = append(schema.ElementEntityMappings[:idx-1], schema.ElementEntityMappings[idx:]...)
	result.UserRefinements = append(result.UserRefinements,
		fmt.Sprintf("Removed entity type: %s", removed.EntityType))
	fmt.Println("Entity type removed.")
}

// editRelationshipRule allows editing a relationship rule
func editRelationshipRule(schema *ontology.OntologySchema, reader *bufio.Reader, result *ontology.BuildResult) {
	if len(schema.EntityRelationshipRules) == 0 {
		fmt.Println("No relationship rules to edit.")
		return
	}

	fmt.Println("\nRelationship rules:")
	for i, rule := range schema.EntityRelationshipRules {
		fmt.Printf("  %d. %s: %s -> %s\n", i+1, rule.Name, rule.SourceEntityType, rule.TargetEntityType)
	}

	fmt.Print("Select rule number: ")
	line, _ := reader.ReadString('\n')
	idx, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || idx < 1 || idx > len(schema.EntityRelationshipRules) {
		fmt.Println("Invalid selection.")
		return
	}

	rule := &schema.EntityRelationshipRules[idx-1]

	fmt.Printf("\nEditing: %s\n", rule.Name)
	fmt.Printf("Current description: %s\n", rule.Description)
	fmt.Print("New description (or press Enter to keep): ")
	line, _ = reader.ReadString('\n')
	newDesc := strings.TrimSpace(line)
	if newDesc != "" {
		rule.Description = newDesc
		result.UserRefinements = append(result.UserRefinements,
			fmt.Sprintf("Updated description for relationship %s", rule.Name))
		fmt.Println("Description updated.")
	}
}

// addRelationshipRule adds a new relationship rule
func addRelationshipRule(schema *ontology.OntologySchema, reader *bufio.Reader, result *ontology.BuildResult) {
	fmt.Print("Relationship name: ")
	line, _ := reader.ReadString('\n')
	name := strings.TrimSpace(line)
	if name == "" {
		return
	}

	fmt.Print("Source entity type: ")
	line, _ = reader.ReadString('\n')
	sourceType := strings.TrimSpace(line)

	fmt.Print("Target entity type: ")
	line, _ = reader.ReadString('\n')
	targetType := strings.TrimSpace(line)

	fmt.Println("Relationship type:")
	fmt.Println("  1. is_a")
	fmt.Println("  2. part_of")
	fmt.Println("  3. related_to")
	fmt.Println("  4. located_in")
	fmt.Println("  5. mentions")
	fmt.Print("Choice: ")
	line, _ = reader.ReadString('\n')
	choice := strings.TrimSpace(line)

	relTypeMap := map[string]ontology.RelationshipType{
		"1": ontology.RelationshipIsA,
		"2": ontology.RelationshipPartOf,
		"3": ontology.RelationshipRelatedTo,
		"4": ontology.RelationshipLocatedIn,
		"5": ontology.RelationshipMentions,
	}

	relType, ok := relTypeMap[choice]
	if !ok {
		fmt.Println("Invalid choice.")
		return
	}

	fmt.Print("Description: ")
	line, _ = reader.ReadString('\n')
	description := strings.TrimSpace(line)

	rule := ontology.EntityRelationshipRule{
		Name:                name,
		SourceEntityType:    sourceType,
		TargetEntityType:    targetType,
		RelationshipType:    relType,
		Description:         description,
		ConfidenceThreshold: 0.7,
	}

	schema.EntityRelationshipRules = append(schema.EntityRelationshipRules, rule)
	result.UserRefinements = append(result.UserRefinements,
		fmt.Sprintf("Added relationship rule: %s", name))
	fmt.Println("Relationship rule added.")
}

// removeRelationshipRule removes a relationship rule
func removeRelationshipRule(schema *ontology.OntologySchema, reader *bufio.Reader, result *ontology.BuildResult) {
	if len(schema.EntityRelationshipRules) == 0 {
		fmt.Println("No relationship rules to remove.")
		return
	}

	fmt.Println("\nRelationship rules:")
	for i, rule := range schema.EntityRelationshipRules {
		fmt.Printf("  %d. %s\n", i+1, rule.Name)
	}

	fmt.Print("Select rule number to remove: ")
	line, _ := reader.ReadString('\n')
	idx, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || idx < 1 || idx > len(schema.EntityRelationshipRules) {
		fmt.Println("Invalid selection.")
		return
	}

	removed := schema.EntityRelationshipRules[idx-1]
	schema.EntityRelationshipRules = append(schema.EntityRelationshipRules[:idx-1], schema.EntityRelationshipRules[idx:]...)
	result.UserRefinements = append(result.UserRefinements,
		fmt.Sprintf("Removed relationship rule: %s", removed.Name))
	fmt.Println("Relationship rule removed.")
}

// editMetadata allows editing domain and metadata
func editMetadata(schema *ontology.OntologySchema, reader *bufio.Reader, result *ontology.BuildResult) {
	fmt.Printf("\nCurrent domain: %s\n", schema.Domain)
	fmt.Print("New domain (or press Enter to keep): ")
	line, _ := reader.ReadString('\n')
	newDomain := strings.TrimSpace(line)
	if newDomain != "" {
		schema.Domain = newDomain
		result.UserRefinements = append(result.UserRefinements,
			fmt.Sprintf("Updated domain to: %s", newDomain))
		fmt.Println("Domain updated.")
	}

	fmt.Printf("\nCurrent key concepts: %s\n", strings.Join(schema.KeyConcepts, ", "))
	fmt.Print("New key concepts (comma-separated, or press Enter to keep): ")
	line, _ = reader.ReadString('\n')
	newConcepts := strings.TrimSpace(line)
	if newConcepts != "" {
		concepts := strings.Split(newConcepts, ",")
		for i := range concepts {
			concepts[i] = strings.TrimSpace(concepts[i])
		}
		schema.KeyConcepts = concepts
		result.UserRefinements = append(result.UserRefinements,
			fmt.Sprintf("Updated key concepts"))
		fmt.Println("Key concepts updated.")
	}
}

// reviewTopEntities shows the top entities found during analysis
func reviewTopEntities(result *ontology.BuildResult) {
	fmt.Println("\nTop entities found in corpus:")
	for i, entity := range result.TopEntities {
		if i >= 20 { // Show top 20
			fmt.Printf("  ... (%d more)\n", len(result.TopEntities)-20)
			break
		}
		fmt.Printf("  %d. %s (%d occurrences)\n", i+1, entity.Entity, entity.Count)
	}
}

// saveSchema saves the ontology schema to JSON
func saveSchema(schema *ontology.OntologySchema, outputPath string) error {
	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// confirm prompts for yes/no confirmation
func confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/n): ", prompt)
	line, _ := reader.ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
