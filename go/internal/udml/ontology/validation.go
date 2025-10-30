package ontology

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// ValidationWarning represents a quality issue detected in the schema
type ValidationWarning struct {
	Severity    string `json:"severity"` // "HIGH", "MEDIUM", "LOW"
	Category    string `json:"category"` // "duplicate_patterns", "confidence_mismatch", etc.
	Message     string `json:"message"`
	EntityType  string `json:"entity_type,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
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
	report.Warnings = append(report.Warnings, checkDuplicatePatterns(schema)...)
	report.Warnings = append(report.Warnings, checkConfidenceSpecificityCorrelation(schema)...)
	report.Warnings = append(report.Warnings, checkMissingDisambiguation(schema)...)
	report.Warnings = append(report.Warnings, checkExcessiveProximityDistance(schema)...)

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
					Severity: "HIGH",
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

	// Check each entity mapping
	for _, mapping := range schema.ElementEntityMappings {
		hasGenericPattern := false
		hasDisambiguation := false

		for _, rule := range mapping.ExtractionRules {
			// Check if pattern is generic (matches common capitalization patterns)
			if rule.InstanceName != "" {
				// Very generic patterns that match lots of text
				if matched, _ := regexp.MatchString(`\[A-Z\]\[a-z\]\+`, rule.InstanceName); matched {
					hasGenericPattern = true
				}
			}

			// Check if disambiguation exists
			if rule.Semantic != nil || rule.Proximity != nil {
				hasDisambiguation = true
			}
		}

		// Warn if generic pattern without disambiguation
		if hasGenericPattern && !hasDisambiguation && len(mapping.ExtractionRules) > 0 {
			warnings = append(warnings, ValidationWarning{
				Severity:   "MEDIUM",
				Category:   "missing_disambiguation",
				EntityType: mapping.EntityType,
				Message: fmt.Sprintf("Entity '%s' uses generic capitalization patterns without semantic or proximity filters",
					mapping.EntityType),
				Suggestion: "Add semantic_filter (recommended) or proximity_filter to reduce false positives",
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

	for _, rule := range schema.EntityRelationshipRules {
		for _, pattern := range rule.ExtractionPatterns {
			// Only check patterns that use max_distance
			if pattern.MaxDistance == 0 {
				continue
			}

			// HIGH severity for distances > 75 tokens
			if pattern.MaxDistance > 75 {
				warnings = append(warnings, ValidationWarning{
					Severity: "HIGH",
					Category: "excessive_proximity_distance",
					Message: fmt.Sprintf("Relationship rule '%s' (%s -> %s) uses max_distance=%d tokens (type: %s)",
						rule.Name, rule.SourceEntityType, rule.TargetEntityType, pattern.MaxDistance, pattern.Type),
					Suggestion: "Entities >75 tokens apart are essentially unrelated. Reduce to ≤30 tokens or use structural patterns for long-range relationships",
				})
				continue
			}

			// MEDIUM severity for distances > 50 tokens
			if pattern.MaxDistance > 50 {
				warnings = append(warnings, ValidationWarning{
					Severity: "MEDIUM",
					Category: "excessive_proximity_distance",
					Message: fmt.Sprintf("Relationship rule '%s' (%s -> %s) uses max_distance=%d tokens (type: %s)",
						rule.Name, rule.SourceEntityType, rule.TargetEntityType, pattern.MaxDistance, pattern.Type),
					Suggestion: "Most proximity rules should use ≤30 tokens. Consider reducing distance or using structural patterns",
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

	// Track entity_type names and their domains
	entityTypeMap := make(map[string][]string) // entity_type -> [domains]

	for _, mapping := range schema.ElementEntityMappings {
		entityTypeMap[mapping.EntityType] = append(entityTypeMap[mapping.EntityType], mapping.Domain)
	}

	// Check for duplicates
	for entityType, domains := range entityTypeMap {
		if len(domains) > 1 {
			// CRITICAL severity - this breaks extraction
			warnings = append(warnings, ValidationWarning{
				Severity:   "CRITICAL",
				Category:   "duplicate_entity_type",
				EntityType: entityType,
				Message: fmt.Sprintf("Entity type '%s' defined %d times across domains: %v. This causes extraction conflicts.",
					entityType, len(domains), domains),
				Suggestion: fmt.Sprintf("Choose ONE approach: (1) Keep only global.%s and remove domain-specific versions, OR (2) Qualify domain versions as '{domain}_%s' (e.g., '%s_%s') with parent_type: 'global.%s'",
					entityType, entityType, domains[1], entityType, entityType),
			})
		}
	}

	return warnings
}
