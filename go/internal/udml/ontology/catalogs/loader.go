package catalogs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"
	"gopkg.in/yaml.v3"
)

// DomainCatalogConfig represents a domain catalog loaded from YAML/JSON
type DomainCatalogConfig struct {
	Domain           string                       `yaml:"domain" json:"domain"`
	Description      string                       `yaml:"description" json:"description"`
	Subdomains       []string                     `yaml:"subdomains,omitempty" json:"subdomains,omitempty"`
	Terms            []TermConfig                 `yaml:"terms,omitempty" json:"terms,omitempty"`
	EntityTypes      []EntityTemplateConfig       `yaml:"entity_types" json:"entity_types"`
	CommonEntityRefs []string                     `yaml:"common_entity_refs,omitempty" json:"common_entity_refs,omitempty"`
	Relationships    []RelationshipTemplateConfig `yaml:"relationships,omitempty" json:"relationships,omitempty"`
}

// TermConfig represents a term in the config file
type TermConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Synonyms    []string `yaml:"synonyms,omitempty" json:"synonyms,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
}

// EntityTemplateConfig represents an entity template in the config file
type EntityTemplateConfig struct {
	EntityType   string                 `yaml:"entity_type" json:"entity_type"`
	ParentType   string                 `yaml:"parent_type,omitempty" json:"parent_type,omitempty"`
	WCategory    string                 `yaml:"w_category,omitempty" json:"w_category,omitempty"`
	Domain       string                 `yaml:"domain,omitempty" json:"domain,omitempty"`
	Description  string                 `yaml:"description" json:"description"`
	Aliases      []string               `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Subdomain    string                 `yaml:"subdomain,omitempty" json:"subdomain,omitempty"`
	ElementTypes []string               `yaml:"element_types,omitempty" json:"element_types,omitempty"`
	SampleRules  []ExtractionRuleConfig `yaml:"sample_rules,omitempty" json:"sample_rules,omitempty"`
}

// ExtractionRuleConfig represents an extraction rule in the config file
type ExtractionRuleConfig struct {
	Type                string   `yaml:"type" json:"type"`
	FieldPath           string   `yaml:"field_path,omitempty" json:"field_path,omitempty"`
	Pattern             string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Keywords            []string `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	ReferenceText       string   `yaml:"reference_text,omitempty" json:"reference_text,omitempty"`
	SimilarityThreshold float64  `yaml:"similarity_threshold,omitempty" json:"similarity_threshold,omitempty"`
	JSONPathExpr        string   `yaml:"jsonpath_expr,omitempty" json:"jsonpath_expr,omitempty"`
}

// RelationshipTemplateConfig represents a relationship template in the config file
type RelationshipTemplateConfig struct {
	Name             string   `yaml:"name" json:"name"`
	SourceType       string   `yaml:"source_type" json:"source_type"`
	TargetType       string   `yaml:"target_type" json:"target_type"`
	RelationshipType string   `yaml:"relationship_type" json:"relationship_type"`
	Description      string   `yaml:"description,omitempty" json:"description,omitempty"`
	Subdomain        string   `yaml:"subdomain,omitempty" json:"subdomain,omitempty"`
	SamplePatterns   []string `yaml:"sample_patterns,omitempty" json:"sample_patterns,omitempty"`
}

// LoadFromFile loads a domain catalog from a YAML or JSON file
func LoadFromFile(path string) (*DomainCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var config DomainCatalogConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML/JSON: %w", err)
	}

	return convertConfigToCatalog(&config), nil
}

// LoadFromDirectory loads all domain catalogs from a directory
func LoadFromDirectory(dirPath string) ([]*DomainCatalog, error) {
	files, err := filepath.Glob(filepath.Join(dirPath, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob YAML files: %w", err)
	}

	// Also check for .yml extension
	ymlFiles, err := filepath.Glob(filepath.Join(dirPath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob YML files: %w", err)
	}
	files = append(files, ymlFiles...)

	catalogs := make([]*DomainCatalog, 0, len(files))
	for _, file := range files {
		catalog, err := LoadFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
		catalogs = append(catalogs, catalog)
	}

	return catalogs, nil
}

// RegisterFromDirectory loads and registers all catalogs from a directory
func RegisterFromDirectory(dirPath string) error {
	catalogs, err := LoadFromDirectory(dirPath)
	if err != nil {
		return err
	}

	for _, catalog := range catalogs {
		RegisterCatalog(catalog)
	}

	return nil
}

// convertConfigToCatalog converts config format to internal catalog format
func convertConfigToCatalog(config *DomainCatalogConfig) *DomainCatalog {
	catalog := &DomainCatalog{
		Domain:           config.Domain,
		Description:      config.Description,
		Subdomains:       config.Subdomains,
		CommonEntityRefs: config.CommonEntityRefs,
	}

	// Convert terms
	catalog.Terms = make([]Term, len(config.Terms))
	for i, t := range config.Terms {
		catalog.Terms[i] = Term{
			Name:        t.Name,
			Synonyms:    t.Synonyms,
			Description: t.Description,
		}
	}

	// Convert entity types
	catalog.EntityTypes = make([]EntityTemplate, len(config.EntityTypes))
	for i, e := range config.EntityTypes {
		catalog.EntityTypes[i] = EntityTemplate{
			EntityType:   e.EntityType,
			ParentType:   e.ParentType,
			WCategory:    e.WCategory,
			Domain:       e.Domain,
			Description:  e.Description,
			Aliases:      e.Aliases,
			Subdomain:    e.Subdomain,
			ElementTypes: e.ElementTypes,
			SampleRules:  convertRuleConfigs(e.SampleRules),
		}
	}

	// Convert relationships
	catalog.Relationships = make([]RelationshipTemplate, len(config.Relationships))
	for i, r := range config.Relationships {
		catalog.Relationships[i] = RelationshipTemplate{
			Name:             r.Name,
			SourceType:       r.SourceType,
			TargetType:       r.TargetType,
			RelationshipType: ontology.RelationshipType(r.RelationshipType),
			Description:      r.Description,
			Subdomain:        r.Subdomain,
			SamplePatterns:   r.SamplePatterns,
		}
	}

	return catalog
}

// convertRuleConfigs converts rule configs to extraction rules
func convertRuleConfigs(configs []ExtractionRuleConfig) []ontology.ExtractionRule {
	rules := make([]ontology.ExtractionRule, len(configs))
	for i, c := range configs {
		rules[i] = ontology.ExtractionRule{
			JSONPath:     c.JSONPathExpr, // Old JSONPathExpr → new JSONPath
			Pattern:      c.Pattern,
			InstanceName: c.FieldPath, // Old FieldPath can map to InstanceName for now
			// TODO: Map other fields from config to new ExtractionRule structure
			// Keywords, ReferenceText, SimilarityThreshold moved to filter structures
		}
	}
	return rules
}

// SaveToFile saves a domain catalog to a YAML file
func SaveToFile(catalog *DomainCatalog, path string) error {
	config := convertCatalogToConfig(catalog)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// convertCatalogToConfig converts internal catalog to config format
func convertCatalogToConfig(catalog *DomainCatalog) *DomainCatalogConfig {
	config := &DomainCatalogConfig{
		Domain:           catalog.Domain,
		Description:      catalog.Description,
		Subdomains:       catalog.Subdomains,
		CommonEntityRefs: catalog.CommonEntityRefs,
	}

	// Convert terms
	config.Terms = make([]TermConfig, len(catalog.Terms))
	for i, t := range catalog.Terms {
		config.Terms[i] = TermConfig{
			Name:        t.Name,
			Synonyms:    t.Synonyms,
			Description: t.Description,
		}
	}

	// Convert entity types
	config.EntityTypes = make([]EntityTemplateConfig, len(catalog.EntityTypes))
	for i, e := range catalog.EntityTypes {
		config.EntityTypes[i] = EntityTemplateConfig{
			EntityType:   e.EntityType,
			Description:  e.Description,
			Aliases:      e.Aliases,
			Subdomain:    e.Subdomain,
			ElementTypes: e.ElementTypes,
			SampleRules:  convertRulesToConfigs(e.SampleRules),
		}
	}

	// Convert relationships
	config.Relationships = make([]RelationshipTemplateConfig, len(catalog.Relationships))
	for i, r := range catalog.Relationships {
		config.Relationships[i] = RelationshipTemplateConfig{
			Name:             r.Name,
			SourceType:       r.SourceType,
			TargetType:       r.TargetType,
			RelationshipType: string(r.RelationshipType),
			Description:      r.Description,
			Subdomain:        r.Subdomain,
			SamplePatterns:   r.SamplePatterns,
		}
	}

	return config
}

// convertRulesToConfigs converts extraction rules to rule configs
func convertRulesToConfigs(rules []ontology.ExtractionRule) []ExtractionRuleConfig {
	configs := make([]ExtractionRuleConfig, len(rules))
	for i, r := range rules {
		configs[i] = ExtractionRuleConfig{
			FieldPath:    r.InstanceName, // New InstanceName → old FieldPath
			Pattern:      r.Pattern,
			JSONPathExpr: r.JSONPath, // New JSONPath → old JSONPathExpr
			// TODO: Extract from filter structures to config format
		}
	}
	return configs
}
