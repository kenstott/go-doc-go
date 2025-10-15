package catalogs

import "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"

// DomainCatalog represents a complete domain ontology catalog
type DomainCatalog struct {
	Domain           string
	Description      string
	Subdomains       []string
	Terms            []Term
	EntityTypes      []EntityTemplate       // Domain-specific entities
	CommonEntityRefs []string               // References to common entity templates
	Relationships    []RelationshipTemplate
}

// Term represents a domain-specific term with synonyms
type Term struct {
	Name        string
	Synonyms    []string
	Description string
}

// EntityTemplate represents a template for an entity type
type EntityTemplate struct {
	EntityType   string
	Description  string
	Aliases      []string
	Subdomain    string   // Optional: for domain-specific entities
	ElementTypes []string // UDML element types this entity appears in
	SampleRules  []ontology.ExtractionRule
}

// RelationshipTemplate represents a template for a relationship type
type RelationshipTemplate struct {
	Name             string
	SourceType       string
	TargetType       string
	RelationshipType ontology.RelationshipType
	Description      string
	Subdomain        string // Optional: for domain-specific relationships
	SamplePatterns   []string
}

// DomainCatalogs is the global registry of all domain catalogs
var DomainCatalogs = make(map[string]*DomainCatalog)

// RegisterCatalog registers a domain catalog
func RegisterCatalog(catalog *DomainCatalog) {
	DomainCatalogs[catalog.Domain] = catalog
}

// GetCatalog retrieves a domain catalog by name
func GetCatalog(domain string) (*DomainCatalog, bool) {
	catalog, exists := DomainCatalogs[domain]
	return catalog, exists
}

// ListDomains returns all available domain names
func ListDomains() []string {
	domains := make([]string, 0, len(DomainCatalogs))
	for domain := range DomainCatalogs {
		domains = append(domains, domain)
	}
	return domains
}

// GetAllEntityTemplates returns both domain-specific and common entity templates
func (dc *DomainCatalog) GetAllEntityTemplates() []EntityTemplate {
	// Estimate capacity: domain entities + common entity references
	entities := make([]EntityTemplate, 0, len(dc.EntityTypes)+len(dc.CommonEntityRefs))

	// Add domain-specific entities
	entities = append(entities, dc.EntityTypes...)

	// Add referenced common entities
	for _, ref := range dc.CommonEntityRefs {
		if commonEntity, exists := GetCommonEntityTemplate(ref); exists {
			entities = append(entities, commonEntity)
		}
	}

	return entities
}

// init loads all built-in domain catalogs from YAML configuration files
// This replaces the previous hardcoded catalog registrations
func init() {
	// Auto-load built-in catalogs from embedded directory or default location
	// Users can also manually load custom catalogs via RegisterFromDirectory()

	// Try multiple possible paths for the examples/ontologies directory
	catalogPaths := []string{
		"./examples/ontologies",
		"../examples/ontologies",
		"../../examples/ontologies",
		"../../../examples/ontologies",
	}

	for _, catalogPath := range catalogPaths {
		err := RegisterFromDirectory(catalogPath)
		if err == nil {
			// Successfully loaded from this path
			return
		}
	}

	// Catalogs directory not found - user must manually load
	// This is OK for custom deployments where users provide their own catalogs
	// Users can call catalogs.RegisterFromDirectory("path/to/catalogs") explicitly
}
