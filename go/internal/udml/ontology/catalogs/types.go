package catalogs

import "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"

// DomainCatalog represents a complete domain ontology catalog
type DomainCatalog struct {
	Domain           string
	Description      string
	Dependencies     []string                // Domain dependencies (imports)
	Subdomains       []string
	Terms            []Term
	EntityTypes      []EntityTemplate        // Domain-specific entities
	CommonEntityRefs []string                // References to common entity templates
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
	ParentType   string // Optional: qualified parent reference (e.g., "global.person")
	WCategory    string // 5 W's category: who, what, where, when, why
	Domain       string // Domain this template belongs to (e.g., "global", "medical")
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

// GetAllEntityTemplates returns domain-specific entity templates
// Note: Global entity references are now handled via parent_type in OntologySchema
func (dc *DomainCatalog) GetAllEntityTemplates() []EntityTemplate {
	return dc.EntityTypes
}

// init loads all built-in global domain catalogs from YAML configuration files
// Global catalogs are located at: <project-root>/catalogs/global/
// This replaces the previous hardcoded catalog registrations
func init() {
	// Load global catalogs from fixed location relative to project root
	// The catalog path is relative to where binaries are executed (typically project root)
	catalogPath := "catalogs/global"

	err := RegisterFromDirectory(catalogPath)
	if err != nil {
		// Failed to load - likely running from wrong directory or catalogs missing
		// This is OK - users can manually load via catalogs.RegisterFromDirectory()
		// or the system will work without global catalogs (domain-specific only)
		return
	}
}
