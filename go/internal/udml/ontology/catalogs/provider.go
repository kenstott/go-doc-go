package catalogs

// CatalogProvider implements the DomainCatalogProvider interface for the ontology package
// This avoids import cycle issues
type CatalogProvider struct{}

// NewCatalogProvider creates a new catalog provider
func NewCatalogProvider() *CatalogProvider {
	return &CatalogProvider{}
}

// ListDomains returns all available domain names
func (p *CatalogProvider) ListDomains() []string {
	return ListDomains()
}

// GetDomainDescription returns domain information for similarity computation
func (p *CatalogProvider) GetDomainDescription(domain string) (description string, subdomains []string, terms []string, entities []string, exists bool) {
	catalog, exists := GetCatalog(domain)
	if !exists {
		return "", nil, nil, nil, false
	}

	// Extract term names and synonyms
	termList := []string{}
	for _, term := range catalog.Terms {
		termList = append(termList, term.Name)
		termList = append(termList, term.Synonyms...)
	}

	// Extract entity type names and aliases
	entityList := []string{}
	for _, entity := range catalog.EntityTypes {
		entityList = append(entityList, entity.EntityType)
		entityList = append(entityList, entity.Aliases...)
	}

	return catalog.Description, catalog.Subdomains, termList, entityList, true
}
