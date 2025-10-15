# Domain Catalogs for Ontology Building

## Overview

This directory contains pre-built domain ontology catalogs that provide comprehensive entity types, terms, and relationship patterns for automatic ontology schema generation.

## Architecture

### Domain Catalogs (Industry/Subject Areas)
- **15 true domains** representing distinct industries or subject areas
- Each domain has specific terminology, entities, and relationships
- Domains can have subdomains (e.g., insurance → life, property, health, auto)

### Common Entity Templates (Cross-Domain)
- **~30 reusable entity templates** used across multiple domains
- Categories: Location, Person, Descriptive, Temporal, Contact, Numeric, Identifier
- NOT separate domains - these are shared building blocks

## Structure

```
catalogs/
├── types.go              # ✅ Core catalog types and registry
├── common.go             # ✅ Common entity templates (30 entities, 7 categories)
├── financial.go          # ✅ Financial domain
├── legal.go              # ✅ Legal domain (6 subdomains)
├── medical.go            # ✅ Medical domain (6 subdomains)
├── technical.go          # ✅ Technical domain (6 subdomains)
├── insurance.go          # ✅ Insurance domain (8 subdomains)
├── manufacturing.go      # ✅ Manufacturing domain (5 subdomains)
├── retail.go             # ✅ Retail domain (5 subdomains)
├── logistics.go          # ✅ Logistics/transportation domain (6 subdomains)
├── pharmaceutical.go     # 📋 Pharmaceutical domain (planned)
├── automotive.go         # 📋 Automotive domain (planned)
├── real_estate.go        # 📋 Real estate domain (planned)
├── education.go          # 📋 Education domain (planned)
├── government.go         # 📋 Government domain (planned)
├── energy.go             # 📋 Energy domain (planned)
└── telecommunications.go # 📋 Telecom domain (planned)
```

## Usage

### Registering a Domain Catalog

```go
var FinancialCatalog = &DomainCatalog{
    Domain: "financial",
    Description: "Financial reports, earnings, transactions",

    // Domain-specific entities
    EntityTypes: []EntityTemplate{
        {EntityType: "company", Description: "Business organization", ...},
        {EntityType: "stock_symbol", Description: "Ticker symbol", ...},
    },

    // Reference common entities
    CommonEntityRefs: []string{
        "person",      // For shareholders
        "executive",   // For CEO, CFO
        "date",        // For reporting dates
        "address",     // For headquarters
        "email",       // For investor relations
    },
}

func init() {
    RegisterCatalog(FinancialCatalog)
}
```

### Getting All Entities for a Domain

```go
catalog, _ := GetCatalog("financial")

// Get domain-specific + common entities
allEntities := catalog.GetAllEntityTemplates()

// Now allEntities includes:
// - company, stock_symbol, monetary_amount (domain-specific)
// - person, executive, date, address, email (common)
```

### Using in Ontology Builder

```go
// Phase 1: Compute domain similarities (cosine similarity)
similarities := builder.ComputeDomainSimilarities(ctx, samples)
// Returns: [{"financial": 0.92}, {"legal": 0.78}, ...]

// Phase 2: Load selected domain catalogs
catalog := catalogs.GetCatalog("financial")
entityTemplates := catalog.GetAllEntityTemplates()

// Phase 3: LLM uses templates + corpus to generate extraction rules
```

## Common Entity Templates

### Location Entities
- `city` - Municipality, town
- `street` - Street address
- `address` - Full mailing address
- `building` - Building or structure
- `postal_code` - ZIP/postal code
- `country` - Country or nation
- `region` - State, province, region

### Person Entities
- `person` - Individual person
- `public_figure` - Notable public figure
- `role` - Job title or position
- `executive` - Company executive
- `employee` - Employee or staff

### Descriptive Entities
- `color` - Color or hue
- `size` - Size or magnitude
- `dimension` - Physical dimensions
- `weight` - Weight or mass
- `volume` - Volume or capacity
- `material` - Material composition
- `texture` - Surface texture
- `shape` - Geometric shape

### Temporal Entities
- `date` - Calendar date
- `time` - Time of day
- `duration` - Time period

### Contact Entities
- `email` - Email address
- `phone` - Phone number
- `url` - Web URL

### Numeric Entities
- `percentage` - Percentage value
- `number` - Numeric value

### Identifier Entities
- `id_number` - Generic ID
- `code` - Alphanumeric code

## Domain Catalog Requirements

Each domain catalog MUST include:

1. **Domain Name** - Unique domain identifier
2. **Description** - Clear domain description
3. **Subdomains** - Optional subdomain list
4. **Terms** - Domain-specific terms with synonyms
5. **EntityTypes** - Domain-specific entity templates with:
   - EntityType name
   - Description
   - Aliases (synonyms)
   - ElementTypes (where they appear)
   - SampleRules (extraction patterns)
6. **CommonEntityRefs** - List of common entities to include
7. **Relationships** - Relationship templates

## Implementation Status

### ✅ Completed (Phase 1) - 8 Domain Catalogs
- [x] `types.go` - Core catalog system with CommonEntityRefs support
- [x] `common.go` - 30 common entity templates across 7 categories
- [x] `financial.go` - Financial domain (5 domain entities + 14 common refs)
- [x] `legal.go` - Legal domain with 6 subdomains (12 domain entities + 17 common refs)
- [x] `medical.go` - Medical domain with 6 subdomains (12 domain entities + 19 common refs)
- [x] `technical.go` - Technical domain with 6 subdomains (14 domain entities + 11 common refs)
- [x] `insurance.go` - Insurance domain with 8 subdomains (18 domain entities + 17 common refs)
- [x] `manufacturing.go` - Manufacturing domain with 5 subdomains (13 domain entities + 20 common refs)
- [x] `retail.go` - Retail domain with 5 subdomains (12 domain entities + 20 common refs)
- [x] `logistics.go` - Logistics/transportation domain with 6 subdomains (15 domain entities + 18 common refs)

**Total Coverage**: 8 domains, 42 subdomains, ~105 domain-specific entities, 30 common entities, ~70 relationship templates

### 📋 Planned (Phase 2) - Additional Domains
- [ ] 7+ additional domain catalogs (pharmaceutical, automotive, real_estate, education, government, energy, telecommunications)
- [ ] Cosine similarity analysis in builder
- [ ] MCP tools for LLM corpus exploration (6 tools)
- [ ] Interactive interview workflow

## Benefits

✅ **Comprehensive Coverage** - Pre-built catalogs cover common entities
✅ **DRY Principle** - Common entities defined once, reused
✅ **Semantic Analysis** - Cosine similarity for domain detection
✅ **LLM-Assisted** - LLM explores corpus via MCP tools
✅ **Human-in-Loop** - Interactive refinement
✅ **Extensible** - Easy to add new domains

## Next Steps

1. Update existing `financial.go` with `CommonEntityRefs`
2. Create `legal.go`, `medical.go`, `technical.go` catalogs
3. Create comprehensive `insurance.go` with 8 subdomains
4. Add 10+ more domain catalogs
5. Implement cosine similarity analysis
6. Create MCP server for LLM corpus exploration
7. Build interactive interview system
8. Test with real corpus data

## Related Documentation

- [Ontology Builder](../builder.go) - LLM-based ontology generation
- [Ontology CLI](../../cmd/ontology/README.md) - Command-line interface
- [UDML Specification](../../../../docs/UDML_SPECIFICATION.md) - Document model
