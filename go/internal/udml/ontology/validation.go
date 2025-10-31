package ontology

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// ValidationWarning represents a quality issue detected in the schema
type ValidationWarning struct {
	Severity          string `json:"severity"`            // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	Category          string `json:"category"`            // "duplicate_patterns", "confidence_mismatch", etc.
	Message           string `json:"message"`
	EntityType        string `json:"entity_type,omitempty"`
	Suggestion        string `json:"suggestion,omitempty"`
	Indices           []int  `json:"indices,omitempty"`            // Array indices of all occurrences (for duplicate detection)
	IndicesToRemove   []int  `json:"indices_to_remove,omitempty"`  // Indices to remove (in descending order)
	IndexToKeep       int    `json:"index_to_keep,omitempty"`      // Index of the occurrence to keep (-1 if not applicable)
	DomainToKeep      string `json:"domain_to_keep,omitempty"`     // Domain of the occurrence to keep
	ConfidenceToKeep  float64 `json:"confidence_to_keep,omitempty"` // Confidence of the occurrence to keep
}

// SchemaQualityReport contains quality metrics and warnings
type SchemaQualityReport struct {
	TotalEntityMappings       int                  `json:"total_entity_mappings"`
	TotalRelationships        int                  `json:"total_relationships"`
	SemanticFilterUsageCount  int                  `json:"semantic_filter_usage_count"`
	ProximityFilterUsageCount int                  `json:"proximity_filter_usage_count"`
	GenericPatternCount       int                  `json:"generic_pattern_count"`
	DuplicatePatternCount     int                  `json:"duplicate_pattern_count"`
	Warnings                  []ValidationWarning  `json:"warnings"`
}

// ValidateSchemaQuality performs comprehensive quality checks on the schema
func ValidateSchemaQuality(schema *OntologySchema) *SchemaQualityReport {
	report := &SchemaQualityReport{
		TotalEntityMappings: len(schema.ElementEntityMappings),
		TotalRelationships:  len(schema.EntityRelationshipRules),
		Warnings:            []ValidationWarning{},
	}

	// Run all validation checks - check duplicate entity types first (CRITICAL)
	report.Warnings = append(report.Warnings, checkDuplicateEntityTypes(schema)...)
	report.Warnings = append(report.Warnings, checkMissingWCategory(schema)...)
	report.Warnings = append(report.Warnings, checkDuplicatePatterns(schema)...)
	report.Warnings = append(report.Warnings, checkConfidenceSpecificityCorrelation(schema)...)
	report.Warnings = append(report.Warnings, checkMissingDisambiguation(schema)...)
	report.Warnings = append(report.Warnings, checkExcessiveProximityDistance(schema)...)
	report.Warnings = append(report.Warnings, checkGenericRelationshipTypes(schema)...)

	// Calculate statistics
	calculateQualityMetrics(schema, report)

	return report
}

// checkDuplicatePatterns detects patterns used by multiple entity types
func checkDuplicatePatterns(schema *OntologySchema) []ValidationWarning {
	warnings := []ValidationWarning{}

	// Track patterns by their instance_name regex
	patternMap := make(map[string][]string)

	for _, mapping := range schema.ElementEntityMappings {
		for _, rule := range mapping.ExtractionRules {
			if rule.InstanceName != "" {
				// Normalize pattern for comparison
				normalizedPattern := strings.TrimSpace(rule.InstanceName)
				patternMap[normalizedPattern] = append(patternMap[normalizedPattern], mapping.EntityType)
			}
		}
	}

	// Check for duplicates
	for pattern, entityTypes := range patternMap {
		if len(entityTypes) > 1 {
			// Check if any of these entities have disambiguation filters
			hasDisambiguation := false
			for _, entityType := range entityTypes {
				mapping := findEntityMapping(schema, entityType)
				if mapping != nil {
					for _, rule := range mapping.ExtractionRules {
						if strings.TrimSpace(rule.InstanceName) == pattern {
							if rule.Semantic != nil || rule.Proximity != nil {
								hasDisambiguation = true
								break
							}
						}
					}
				}
				if hasDisambiguation {
					break
				}
			}

			if !hasDisambiguation {
				warnings = append(warnings, ValidationWarning{
					Severity: "CRITICAL", // Duplicate patterns without disambiguation will cause extraction conflicts
					Category: "duplicate_patterns",
					Message: fmt.Sprintf("Pattern '%s' used by %d entity types without disambiguation: %v",
						truncatePattern(pattern), len(entityTypes), entityTypes),
					Suggestion: "Add semantic_filter or proximity_filter to disambiguate conflicting patterns",
				})
			} else {
				// Medium severity if some disambiguation exists
				warnings = append(warnings, ValidationWarning{
					Severity: "MEDIUM",
					Category: "duplicate_patterns",
					Message: fmt.Sprintf("Pattern '%s' used by %d entity types: %v (partial disambiguation found)",
						truncatePattern(pattern), len(entityTypes), entityTypes),
					Suggestion: "Verify disambiguation filters cover all conflict cases",
				})
			}
		}
	}

	return warnings
}

// checkConfidenceSpecificityCorrelation checks if confidence scores align with pattern specificity
func checkConfidenceSpecificityCorrelation(schema *OntologySchema) []ValidationWarning {
	warnings := []ValidationWarning{}

	// Generic pattern indicators
	genericPatterns := []string{
		`[A-Z][a-z]+(?:\\s+[A-Z][a-z]+)*`,  // Any capitalized words
		`[A-Z][A-Za-z\\s]+`,                 // Any text starting with capital
		`\\b[A-Z]\\w+\\b`,                   // Any capitalized word
	}

	for _, mapping := range schema.ElementEntityMappings {
		for _, rule := range mapping.ExtractionRules {
			if rule.InstanceName == "" {
				continue
			}

			// Check if pattern is generic
			isGeneric := false
			for _, genericPattern := range genericPatterns {
				if strings.Contains(rule.InstanceName, genericPattern) {
					isGeneric = true
					break
				}
			}

			// High confidence (>= 0.85) with generic pattern and no disambiguation
			if isGeneric && mapping.Confidence >= 0.85 {
				if rule.Semantic == nil && rule.Proximity == nil {
					warnings = append(warnings, ValidationWarning{
						Severity:   "HIGH",
						Category:   "confidence_mismatch",
						EntityType: mapping.EntityType,
						Message: fmt.Sprintf("Entity '%s' uses generic pattern '%s' with high confidence (%.2f) but no disambiguation filters",
							mapping.EntityType, truncatePattern(rule.InstanceName), mapping.Confidence),
						Suggestion: "Either lower confidence to 0.60-0.75 or add semantic/proximity filter",
					})
				}
			}

			// Specific patterns (DOI, URL, email, etc.) with low confidence
			specificIndicators := []string{"DOI:", "http://", "https://", "@", "\\d{4}-\\d{4}"}
			isSpecific := false
			for _, indicator := range specificIndicators {
				if strings.Contains(rule.InstanceName, indicator) {
					isSpecific = true
					break
				}
			}

			if isSpecific && mapping.Confidence < 0.80 {
				warnings = append(warnings, ValidationWarning{
					Severity:   "MEDIUM",
					Category:   "confidence_mismatch",
					EntityType: mapping.EntityType,
					Message: fmt.Sprintf("Entity '%s' uses specific pattern '%s' with low confidence (%.2f)",
						mapping.EntityType, truncatePattern(rule.InstanceName), mapping.Confidence),
					Suggestion: "Consider increasing confidence to 0.85-0.95 for highly specific patterns",
				})
			}
		}
	}

	return warnings
}

// checkMissingDisambiguation identifies entity mappings that need disambiguation
func checkMissingDisambiguation(schema *OntologySchema) []ValidationWarning {
	warnings := []ValidationWarning{}

	// Expanded generic pattern detection
	genericIndicators := []string{
		`\[A-Z\]\[a-z\]\+`,              // Capitalized word
		`\[A-Z\]\[A-Za-z\]\+`,           // Any capitalized text
		`\\b\[A-Z\]\\w\+\\b`,            // Word boundary capitalized
		`\[A-Z\]\[a-z\]\+\(\?:\\s\+\[A-Z\]\[a-z\]\+\)\*`, // Multiple capitalized words
	}

	// Check each entity mapping
	for i, mapping := range schema.ElementEntityMappings {
		hasGenericPattern := false
		hasDisambiguation := false

		for _, rule := range mapping.ExtractionRules {
			if rule.InstanceName == "" {
				continue
			}

			// Check if pattern matches ANY generic indicator
			for _, indicator := range genericIndicators {
				if strings.Contains(rule.InstanceName, indicator) {
					hasGenericPattern = true
					break
				}
			}

			// Check if disambiguation exists
			if rule.Semantic != nil || rule.Proximity != nil {
				hasDisambiguation = true
			}
		}

		// CRITICAL if generic pattern without disambiguation
		if hasGenericPattern && !hasDisambiguation && len(mapping.ExtractionRules) > 0 {
			warnings = append(warnings, ValidationWarning{
				Severity:   "CRITICAL",
				Category:   "missing_disambiguation",
				EntityType: mapping.EntityType,
				Indices:    []int{i},
				Message: fmt.Sprintf("Entity '%s' uses generic regex without semantic or proximity filters. Will produce high false positives.",
					mapping.EntityType),
				Suggestion: "Add EITHER: (1) semantic_filter with reference_terms and similarity_threshold>=0.70, OR (2) proximity_filter requiring nearby entities",
			})
		}
	}

	return warnings
}

// calculateQualityMetrics computes statistics for the report
func calculateQualityMetrics(schema *OntologySchema, report *SchemaQualityReport) {
	for _, mapping := range schema.ElementEntityMappings {
		for _, rule := range mapping.ExtractionRules {
			if rule.Semantic != nil {
				report.SemanticFilterUsageCount++
			}
			if rule.Proximity != nil {
				report.ProximityFilterUsageCount++
			}

			// Count generic patterns
			if rule.InstanceName != "" {
				if matched, _ := regexp.MatchString(`\[A-Z\]\[a-z\]\+`, rule.InstanceName); matched {
					report.GenericPatternCount++
				}
			}
		}
	}

	// Count duplicate patterns
	patternMap := make(map[string]int)
	for _, mapping := range schema.ElementEntityMappings {
		for _, rule := range mapping.ExtractionRules {
			if rule.InstanceName != "" {
				patternMap[strings.TrimSpace(rule.InstanceName)]++
			}
		}
	}
	for _, count := range patternMap {
		if count > 1 {
			report.DuplicatePatternCount++
		}
	}
}

// findEntityMapping finds an entity mapping by type
func findEntityMapping(schema *OntologySchema, entityType string) *ElementEntityMappingConfig {
	for i := range schema.ElementEntityMappings {
		if schema.ElementEntityMappings[i].EntityType == entityType {
			return &schema.ElementEntityMappings[i]
		}
	}
	return nil
}

// truncatePattern truncates long patterns for display
func truncatePattern(pattern string) string {
	maxLen := 60
	if len(pattern) > maxLen {
		return pattern[:maxLen] + "..."
	}
	return pattern
}

// CalculateSemanticFilterUsagePercent calculates percentage of mappings using semantic filters
func (report *SchemaQualityReport) CalculateSemanticFilterUsagePercent() float64 {
	if report.TotalEntityMappings == 0 {
		return 0.0
	}
	return (float64(report.SemanticFilterUsageCount) / float64(report.TotalEntityMappings)) * 100.0
}

// GetWarningsBySeverity returns warnings filtered by severity
func (report *SchemaQualityReport) GetWarningsBySeverity(severity string) []ValidationWarning {
	filtered := []ValidationWarning{}
	for _, w := range report.Warnings {
		if w.Severity == severity {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

// HasCriticalWarnings checks if there are any CRITICAL severity warnings
func (report *SchemaQualityReport) HasCriticalWarnings() bool {
	for _, w := range report.Warnings {
		if w.Severity == "CRITICAL" {
			return true
		}
	}
	return false
}

// HasHighSeverityWarnings checks if there are any HIGH severity warnings
func (report *SchemaQualityReport) HasHighSeverityWarnings() bool {
	for _, w := range report.Warnings {
		if w.Severity == "HIGH" {
			return true
		}
	}
	return false
}

// CalculateQualityScore computes an overall quality score (0-100)
func (report *SchemaQualityReport) CalculateQualityScore() float64 {
	score := 100.0

	// Deduct points for warnings
	highWarnings := len(report.GetWarningsBySeverity("HIGH"))
	mediumWarnings := len(report.GetWarningsBySeverity("MEDIUM"))
	lowWarnings := len(report.GetWarningsBySeverity("LOW"))

	score -= float64(highWarnings) * 10.0   // -10 points per HIGH warning
	score -= float64(mediumWarnings) * 5.0  // -5 points per MEDIUM warning
	score -= float64(lowWarnings) * 2.0     // -2 points per LOW warning

	// Bonus for good practices
	semanticUsagePercent := report.CalculateSemanticFilterUsagePercent()
	if semanticUsagePercent >= 30.0 {
		score += 5.0 // +5 bonus if >= 30% use semantic filters
	}

	// Ensure score is between 0 and 100
	return math.Max(0.0, math.Min(100.0, score))
}

// checkExcessiveProximityDistance validates proximity distances in relationship rules
func checkExcessiveProximityDistance(schema *OntologySchema) []ValidationWarning {
	warnings := []ValidationWarning{}

	for i, rule := range schema.EntityRelationshipRules {
		for _, pattern := range rule.ExtractionPatterns {
			// Only check patterns that use max_distance
			if pattern.MaxDistance == 0 {
				continue
			}

			// CRITICAL: distances > 75 tokens
			if pattern.MaxDistance > 75 {
				warnings = append(warnings, ValidationWarning{
					Severity: "CRITICAL",
					Category: "excessive_proximity_distance",
					Indices:  []int{i},
					Message: fmt.Sprintf("Relationship '%s' (%s → %s) uses max_distance=%d tokens (CRITICAL: >75)",
						rule.Name, rule.SourceEntityType, rule.TargetEntityType, pattern.MaxDistance),
					Suggestion: "Entities >75 tokens apart are essentially unrelated. Reduce to ≤30 tokens or use structural patterns",
				})
				continue
			}

			// HIGH: distances 51-75 tokens
			if pattern.MaxDistance > 50 && pattern.MaxDistance <= 75 {
				warnings = append(warnings, ValidationWarning{
					Severity: "HIGH",
					Category: "excessive_proximity_distance",
					Indices:  []int{i},
					Message: fmt.Sprintf("Relationship '%s' (%s → %s) uses max_distance=%d tokens (HIGH: 51-75)",
						rule.Name, rule.SourceEntityType, rule.TargetEntityType, pattern.MaxDistance),
					Suggestion: "Consider reducing to ≤30 tokens for better precision",
				})
				continue
			}

			// MEDIUM: distances 31-50 tokens
			if pattern.MaxDistance > 30 && pattern.MaxDistance <= 50 {
				warnings = append(warnings, ValidationWarning{
					Severity: "MEDIUM",
					Category: "high_proximity_distance",
					Indices:  []int{i},
					Message: fmt.Sprintf("Relationship '%s' (%s → %s) uses max_distance=%d tokens (above recommended 30)",
						rule.Name, rule.SourceEntityType, rule.TargetEntityType, pattern.MaxDistance),
					Suggestion: "Recommended maximum is 30 tokens. Consider reducing for better precision.",
				})
			}
		}
	}

	return warnings
}

// checkDuplicateEntityTypes detects multiple definitions of the same entity_type name
// This causes extraction conflicts where the same entity name is defined multiple times
func checkDuplicateEntityTypes(schema *OntologySchema) []ValidationWarning {
	warnings := []ValidationWarning{}

	// Track entity_type occurrences with index, domain, and confidence
	type entityOccurrence struct {
		index      int
		domain     string
		confidence float64
	}
	entityTypeMap := make(map[string][]entityOccurrence) // entity_type -> [occurrences]

	for i, mapping := range schema.ElementEntityMappings {
		entityTypeMap[mapping.EntityType] = append(entityTypeMap[mapping.EntityType], entityOccurrence{
			index:      i,
			domain:     mapping.Domain,
			confidence: mapping.Confidence,
		})
	}

	// Check for duplicates
	for entityType, occurrences := range entityTypeMap {
		if len(occurrences) > 1 {
			// Determine which occurrence to keep:
			// 1. Prefer global domain
			// 2. If no global, prefer highest confidence
			// 3. If tied, prefer first occurrence
			keepIndex := 0
			for i, occ := range occurrences {
				if occ.domain == "global" {
					keepIndex = i
					break
				}
				if occ.confidence > occurrences[keepIndex].confidence {
					keepIndex = i
				}
			}

			// Build list of indices to remove (in descending order to avoid index shifting)
			var indicesToRemove []int
			var allIndices []int
			var domains []string
			for i, occ := range occurrences {
				allIndices = append(allIndices, occ.index)
				domains = append(domains, occ.domain)
				if i != keepIndex {
					indicesToRemove = append(indicesToRemove, occ.index)
				}
			}
			// Sort indicesToRemove in descending order
			for i := 0; i < len(indicesToRemove); i++ {
				for j := i + 1; j < len(indicesToRemove); j++ {
					if indicesToRemove[i] < indicesToRemove[j] {
						indicesToRemove[i], indicesToRemove[j] = indicesToRemove[j], indicesToRemove[i]
					}
				}
			}

			// Build alternative names for domain-specific versions
			var renameAlternatives []string
			for i, occ := range occurrences {
				if i != keepIndex && occ.domain != "global" {
					qualifiedName := fmt.Sprintf("%s_%s", occ.domain, entityType)
					renameAlternatives = append(renameAlternatives, fmt.Sprintf("index %d → '%s'", occ.index, qualifiedName))
				}
			}

			// CRITICAL severity - this breaks extraction
			suggestion := fmt.Sprintf("Choose ONE approach:\n"+
				"  (1) REMOVE duplicates at indices %v and keep only index %d (domain: %s, confidence: %.2f)\n"+
				"  (2) RENAME domain-specific versions with qualified names: %v",
				indicesToRemove, occurrences[keepIndex].index, occurrences[keepIndex].domain, occurrences[keepIndex].confidence,
				renameAlternatives)

			warnings = append(warnings, ValidationWarning{
				Severity:         "CRITICAL",
				Category:         "duplicate_entity_type",
				EntityType:       entityType,
				Indices:          allIndices,
				IndicesToRemove:  indicesToRemove,
				IndexToKeep:      occurrences[keepIndex].index,
				DomainToKeep:     occurrences[keepIndex].domain,
				ConfidenceToKeep: occurrences[keepIndex].confidence,
				Message: fmt.Sprintf("Entity type '%s' defined %d times across domains: %v. This causes extraction conflicts.",
					entityType, len(domains), domains),
				Suggestion: suggestion,
			})
		}
	}

	return warnings
}

// checkMissingWCategory validates that all ROOT entities have w_category (REQUIRED field)
func checkMissingWCategory(schema *OntologySchema) []ValidationWarning {
	warnings := []ValidationWarning{}

	for i, mapping := range schema.ElementEntityMappings {
		// Only check ROOT entities (no parent_type)
		if mapping.ParentType != "" {
			continue // Child entities can inherit w_category
		}

		// Check if w_category is missing or empty
		if mapping.WCategory == "" {
			warnings = append(warnings, ValidationWarning{
				Severity:   "CRITICAL",
				Category:   "missing_w_category",
				EntityType: mapping.EntityType,
				Indices:    []int{i},
				Message: fmt.Sprintf("ROOT entity '%s' (domain: %s) missing REQUIRED w_category field. Cannot inherit (no parent_type).",
					mapping.EntityType, mapping.Domain),
				Suggestion: "Add w_category (who/what/where/when/why/how). Examples: person=who, location=where, date=when, process=how",
			})
		}
	}

	return warnings
}

// checkGenericRelationshipTypes warns about overuse of generic "related_to" type
func checkGenericRelationshipTypes(schema *OntologySchema) []ValidationWarning {
	warnings := []ValidationWarning{}

	totalRelationships := len(schema.EntityRelationshipRules)
	relatedToCount := 0
	relatedToIndices := []int{}

	// Count "related_to" usage
	for i, rule := range schema.EntityRelationshipRules {
		if rule.RelationshipType == "related_to" {
			relatedToCount++
			relatedToIndices = append(relatedToIndices, i)
		}
	}

	if totalRelationships == 0 {
		return warnings
	}

	relatedToPercent := float64(relatedToCount) / float64(totalRelationships) * 100.0

	// HIGH if >50% use "related_to"
	if relatedToPercent > 50.0 {
		warnings = append(warnings, ValidationWarning{
			Severity: "HIGH",
			Category: "generic_relationship_types",
			Indices:  relatedToIndices,
			Message: fmt.Sprintf("%d of %d relationships (%.1f%%) use generic 'related_to' type. Loses semantic precision.",
				relatedToCount, totalRelationships, relatedToPercent),
			Suggestion: "Use specific types: treats, manufactures, regulates, employs, contains, produces, enables, requires, etc.",
		})
	} else if relatedToPercent > 40.0 {
		// MEDIUM if 40-50%
		warnings = append(warnings, ValidationWarning{
			Severity: "MEDIUM",
			Category: "generic_relationship_types",
			Indices:  relatedToIndices,
			Message: fmt.Sprintf("%d of %d relationships (%.1f%%) use generic 'related_to' type.",
				relatedToCount, totalRelationships, relatedToPercent),
			Suggestion: "Consider using more specific relationship types where applicable (target: <40%)",
		})
	}

	return warnings
}
