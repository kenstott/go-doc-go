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
	Dependencies     []string                     `yaml:"dependencies,omitempty" json:"dependencies,omitempty"` // Domain dependencies (imports)
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
	EntityType      string                 `yaml:"entity_type" json:"entity_type"`
	ParentType      string                 `yaml:"parent_type,omitempty" json:"parent_type,omitempty"`
	WCategory       string                 `yaml:"w_category,omitempty" json:"w_category,omitempty"`
	Domain          string                 `yaml:"domain,omitempty" json:"domain,omitempty"`
	Description     string                 `yaml:"description" json:"description"`
	Aliases         []string               `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Subdomain       string                 `yaml:"subdomain,omitempty" json:"subdomain,omitempty"`
	ElementTypes    []string               `yaml:"element_types,omitempty" json:"element_types,omitempty"`
	ExtractionRules []ExtractionRuleConfig `yaml:"extraction_rules,omitempty" json:"extraction_rules,omitempty"`
}

// ExtractionRuleConfig represents an extraction rule in the config file
// Supports both old domain catalog format and new global catalog format
type ExtractionRuleConfig struct {
	// New format fields (used by global catalogs)
	Name          string                            `yaml:"name,omitempty" json:"name,omitempty"`
	Description   string                            `yaml:"description,omitempty" json:"description,omitempty"`
	InheritanceOp string                            `yaml:"inheritance_op,omitempty" json:"inheritance_op,omitempty"`
	JSONPath      string                            `yaml:"jsonpath,omitempty" json:"jsonpath,omitempty"`
	PhraseList    []string                          `yaml:"phrase_list,omitempty" json:"phrase_list,omitempty"`
	InstanceName  string                            `yaml:"instance_name,omitempty" json:"instance_name,omitempty"`
	Pattern       string                            `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Proximity     *ontology.ProximityFilter         `yaml:"proximity,omitempty" json:"proximity,omitempty"`
	Dictionary    *ontology.DictionaryFilter        `yaml:"dictionary,omitempty" json:"dictionary,omitempty"`
	Semantic      *ontology.SemanticFilter          `yaml:"semantic,omitempty" json:"semantic,omitempty"`
	LLMValidation *ontology.LLMValidationPrompt     `yaml:"llm_validation,omitempty" json:"llm_validation,omitempty"`
	Attributes    []ontology.AttributeExtractionRule `yaml:"attributes,omitempty" json:"attributes,omitempty"`

	// Old format fields (used by domain catalogs for backward compatibility)
	Type                string   `yaml:"type,omitempty" json:"type,omitempty"`
	FieldPath           string   `yaml:"field_path,omitempty" json:"field_path,omitempty"`
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

	return ConvertConfigToCatalog(&config), nil
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

// ConvertConfigToCatalog converts config format to internal catalog format
// Exported for use by other packages (e.g., builder.go)
func ConvertConfigToCatalog(config *DomainCatalogConfig) *DomainCatalog {
	catalog := &DomainCatalog{
		Domain:           config.Domain,
		Description:      config.Description,
		Dependencies:     config.Dependencies,
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
			SampleRules:  convertRuleConfigs(e.ExtractionRules),
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
// Supports both new format (global catalogs) and old format (domain catalogs)
func convertRuleConfigs(configs []ExtractionRuleConfig) []ontology.ExtractionRule {
	rules := make([]ontology.ExtractionRule, len(configs))
	for i, c := range configs {
		rule := ontology.ExtractionRule{
			// New format fields (prefer these if present)
			Name:          c.Name,
			Description:   c.Description,
			InheritanceOp: c.InheritanceOp,
			PhraseList:    c.PhraseList,
			InstanceName:  c.InstanceName,
			Pattern:       c.Pattern,
			Proximity:     c.Proximity,
			Dictionary:    c.Dictionary,
			Semantic:      c.Semantic,
			LLMValidation: c.LLMValidation,
			Attributes:    c.Attributes,
		}

		// Map JSONPath field (supports both old and new field names)
		if c.JSONPath != "" {
			rule.JSONPath = c.JSONPath
		} else if c.JSONPathExpr != "" {
			rule.JSONPath = c.JSONPathExpr // Old format fallback
		}

		// Old format fallbacks (if new format fields are empty)
		if rule.InstanceName == "" && c.FieldPath != "" {
			rule.InstanceName = c.FieldPath
		}

		// Map old 'keywords' field to new 'phrase_list' field
		if len(rule.PhraseList) == 0 && len(c.Keywords) > 0 {
			rule.PhraseList = c.Keywords
		}

		rules[i] = rule
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
		Dependencies:     catalog.Dependencies,
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
			EntityType:      e.EntityType,
			Description:     e.Description,
			Aliases:         e.Aliases,
			Subdomain:       e.Subdomain,
			ElementTypes:    e.ElementTypes,
			ExtractionRules: convertRulesToConfigs(e.SampleRules),
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
