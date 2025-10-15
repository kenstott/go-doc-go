# Domain Catalog

Complete list of available ontology domains organized by category.

## Business Functions (9 domains)

### Core Business Operations
- **compliance** - Regulatory compliance, risk management, audit, policy enforcement
- **financial** - Financial management, accounting, revenue, expenses, budgeting
- **human_resources** - Employee management, recruitment, compensation, benefits, performance
- **legal** - Contracts, litigation, intellectual property, legal entities
- **marketing** - Campaigns, leads, channels, brand management, analytics
- **sales** - Opportunities, quotes, customers, forecasts, CRM

### Operations & Supply Chain
- **logistics** - Transportation, warehousing, fleet management, freight
- **manufacturing** - Production, assembly, quality control, BOM
- **supply_chain** - Procurement, inventory, suppliers, demand planning

## Industry Sectors (13 domains)

### Financial Services
- **banking** - Investment banking, retail banking, commercial banking, wealth management
- **insurance** - Policies, claims, underwriting, actuarial, risk assessment

### Healthcare & Life Sciences
- **healthcare** - Patients, providers, diagnoses, procedures, medications, clinical care
- **pharmaceutical** - Drug development, clinical trials, regulatory affairs, manufacturing, pharmacovigilance

### Natural Resources
- **extractive_industries** - Mining, oil & gas extraction, quarrying, drilling, exploration, commodity production

### Retail & Consumer
- **retail** - Products, stores, inventory, point-of-sale, merchandising, consumer goods

### Services & Infrastructure
- **energy_utilities** - Power generation, transmission, distribution, renewable energy
- **leisure** - Travel, hotels, tourism, bookings, destinations
- **hospitality** - Hotels, reservations, guests, events, food & beverage
- **telecommunications** - Networks, subscribers, services, billing, infrastructure
- **real_estate** - Property management, leases, facilities, valuation

### Transportation & Manufacturing
- **automotive** - Vehicles, parts, service, sales, dealerships, manufacturing
- **aerospace_defense** - Aircraft, spacecraft, defense systems, maintenance, contracts

## Academic Disciplines (5 domains)

- **education** - Courses, students, teachers, curriculum, assessments
- **medical** - Medical research, clinical studies, treatments, pathology
- **technical** - Engineering, technology, specifications, documentation
- **science_research** - Scientific research, experiments, publications, grants, datasets
- **library_science** - Library cataloging, circulation, collections, archives

## Organizational Types (3 domains)

- **nonprofit** - Donors, grants, programs, volunteers, beneficiaries, foundations
- **religion** - Religious organizations, clergy, worship, sacraments, congregations
- **government** - Regulations, permits, licenses, public records, citizen services

## Cross-Cutting Concerns (6 domains)

- **security_access_control** - Authentication, authorization, permissions, roles, identity management
- **integration_api_management** - APIs, integrations, data exchange, service orchestration
- **quality_assurance** - Testing, defects, quality metrics, test automation, standards
- **sustainability_esg** - Environmental metrics, social responsibility, governance, carbon tracking
- **artificial_intelligence** - ML models, training data, inference, model governance, AI/ML operations
- **blockchain_crypto** - Blockchain transactions, wallets, smart contracts, tokens, cryptocurrency

---

## Summary

**Total Domains**: 36 domains across 5 categories

**Implementation Status**:
- ✅ Fully Implemented: 36 domains
- 📝 Catalog File Created: All domains have YAML files
- 🔄 Dynamically Loaded: System automatically discovers domains from catalog

**Recent Additions (Complete Set)**:
- Business Functions (9): compliance, financial, human_resources, legal, marketing, sales, logistics, manufacturing, supply_chain
- Industry Sectors (13): banking, insurance, healthcare, pharmaceutical, extractive_industries, retail, energy_utilities, leisure, hospitality, telecommunications, real_estate, automotive, aerospace_defense
- Academic Disciplines (5): education, medical, technical, science_research, library_science
- Organizational Types (3): nonprofit, religion, government
- Cross-Cutting Concerns (6): security_access_control, integration_api_management, quality_assurance, sustainability_esg, artificial_intelligence, blockchain_crypto

**Directory Structure**:
```
examples/ontologies/
├── academic/
│   ├── education.yaml
│   ├── library_science.yaml
│   ├── medical.yaml
│   ├── science_research.yaml
│   └── technical.yaml
├── business_functions/
│   ├── compliance.yaml
│   ├── financial.yaml
│   ├── human_resources.yaml
│   ├── legal.yaml
│   ├── logistics.yaml
│   ├── manufacturing.yaml
│   ├── marketing.yaml
│   ├── sales.yaml
│   └── supply_chain.yaml
├── industry_sectors/
│   ├── aerospace_defense.yaml
│   ├── automotive.yaml
│   ├── banking.yaml
│   ├── energy_utilities.yaml
│   ├── extractive_industries.yaml
│   ├── healthcare.yaml
│   ├── hospitality.yaml
│   ├── insurance.yaml
│   ├── leisure.yaml
│   ├── pharmaceutical.yaml
│   ├── real_estate.yaml
│   ├── retail.yaml
│   └── telecommunications.yaml
├── organizational_types/
│   ├── government.yaml
│   ├── nonprofit.yaml
│   └── religion.yaml
└── cross_cutting/
    ├── artificial_intelligence.yaml
    ├── blockchain_crypto.yaml
    ├── integration_api_management.yaml
    ├── quality_assurance.yaml
    ├── security_access_control.yaml
    └── sustainability_esg.yaml
```

## Adding New Domains

To add a new domain:

1. Create a YAML file in the appropriate subdirectory
2. Follow the schema format with `domain`, `description`, `subdomains`, `entity_types`, and `relationship_types`
3. The system will automatically discover it via `loadPredefinedDomains()` function
4. No code changes required - just add the YAML file!

## Domain Categories Explained

### Business Functions
Core operational domains that apply across most businesses and organizations. These represent fundamental business capabilities like finance, HR, sales, and logistics.

### Industry Sectors
Domains specific to particular industries or market sectors. Each has unique terminology, regulations, and workflows (e.g., healthcare, banking, automotive).

### Academic Disciplines
Domains focused on education, research, and scholarly activities. Includes both research institutions and educational organizations.

### Organizational Types
Domains that represent entire classes of organizations with unique structures and purposes (nonprofits, government agencies, religious institutions).

### Cross-Cutting Concerns
Horizontal capabilities that span across multiple domains and industries. These represent shared technical and operational concerns like security, quality, and integration.
