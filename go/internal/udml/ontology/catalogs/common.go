package catalogs

import "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"

// CommonEntityTemplates defines reusable entity templates shared across all domains
// These are NOT separate domains - they are cross-domain entity types
var CommonEntityTemplates = map[string]EntityTemplate{
	// ========================================
	// LOCATION ENTITIES
	// ========================================
	"city": {
		EntityType:   "city",
		Description:  "Municipality, town, or urban area",
		Aliases:      []string{"town", "municipality", "metro area", "urban area", "city"},
		ElementTypes: []string{"paragraph", "table_cell", "list_item"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)?,\s*[A-Z]{2}\b`,
			},
		},
	},
	"street": {
		EntityType:   "street",
		Description:  "Street address or road name",
		Aliases:      []string{"road", "avenue", "boulevard", "lane", "drive", "way", "court", "place"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\d+\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:St|Ave|Rd|Dr|Blvd|Lane|Ct|Pl)`,
			},
		},
	},
	"address": {
		EntityType:   "address",
		Description:  "Full mailing or physical address",
		Aliases:      []string{"location", "street address", "mailing address", "physical address", "residence"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},
	"building": {
		EntityType:   "building",
		Description:  "Building, structure, or facility",
		Aliases:      []string{"structure", "facility", "property", "premises", "site", "complex"},
		ElementTypes: []string{"paragraph", "heading"},
	},
	"postal_code": {
		EntityType:   "postal_code",
		Description:  "ZIP code or postal code",
		Aliases:      []string{"zip code", "postcode", "zip", "postal"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b\d{5}(?:-\d{4})?\b`,
			},
		},
	},
	"country": {
		EntityType:   "country",
		Description:  "Country or nation",
		Aliases:      []string{"nation", "state"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},
	"region": {
		EntityType:   "region",
		Description:  "State, province, or geographic region",
		Aliases:      []string{"state", "province", "territory", "district"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},

	// ========================================
	// PERSON ENTITIES
	// ========================================
	"person": {
		EntityType:   "person",
		Description:  "Individual person or human",
		Aliases:      []string{"individual", "human", "name"},
		ElementTypes: []string{"paragraph", "heading", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b[A-Z][a-z]+\s+[A-Z][a-z]+\b`,
			},
		},
	},
	"public_figure": {
		EntityType:   "public_figure",
		Description:  "Notable public figure or celebrity",
		Aliases:      []string{"celebrity", "notable", "VIP", "prominent person", "public person"},
		ElementTypes: []string{"paragraph", "heading"},
	},
	"role": {
		EntityType:   "role",
		Description:  "Job title, position, or functional role",
		Aliases:      []string{"title", "position", "job title", "occupation", "function"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},
	"executive": {
		EntityType:   "executive",
		Description:  "Company executive or senior leader",
		Aliases:      []string{"CEO", "CFO", "CTO", "COO", "president", "director", "officer", "chairman", "vice president"},
		ElementTypes: []string{"paragraph", "heading"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type: ontology.RuleTypeKeyword,
				Keywords: []string{
					"CEO", "CFO", "CTO", "COO", "President", "Chairman", "Director",
					"Chief Executive Officer", "Chief Financial Officer",
				},
			},
		},
	},
	"employee": {
		EntityType:   "employee",
		Description:  "Employee or staff member",
		Aliases:      []string{"staff", "worker", "personnel", "team member", "associate"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},

	// ========================================
	// DESCRIPTIVE ENTITIES
	// ========================================
	"color": {
		EntityType:   "color",
		Description:  "Color or hue",
		Aliases:      []string{"hue", "shade", "tint", "tone", "pigment"},
		ElementTypes: []string{"paragraph", "table_cell", "list_item"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type: ontology.RuleTypeKeyword,
				Keywords: []string{
					"red", "blue", "green", "yellow", "black", "white", "orange", "purple",
					"pink", "brown", "gray", "silver", "gold",
				},
			},
		},
	},
	"size": {
		EntityType:   "size",
		Description:  "Size or magnitude",
		Aliases:      []string{"dimension", "measurement", "scale", "magnitude"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type: ontology.RuleTypeKeyword,
				Keywords: []string{
					"small", "medium", "large", "extra large", "XS", "S", "M", "L", "XL", "XXL",
				},
			},
		},
	},
	"dimension": {
		EntityType:   "dimension",
		Description:  "Physical dimensions (length, width, height)",
		Aliases:      []string{"measurements", "size", "proportions", "dimensions"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\d+(?:\.\d+)?\s*(?:inches|in|feet|ft|centimeters|cm|meters|m)\s*x\s*\d+`,
			},
		},
	},
	"weight": {
		EntityType:   "weight",
		Description:  "Weight or mass",
		Aliases:      []string{"mass", "heaviness", "weight"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\d+(?:\.\d+)?\s*(?:pounds|lbs|kilograms|kg|ounces|oz|grams|g)`,
			},
		},
	},
	"volume": {
		EntityType:   "volume",
		Description:  "Volume or capacity",
		Aliases:      []string{"capacity", "cubic measurement", "volume"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},
	"material": {
		EntityType:   "material",
		Description:  "Material or substance composition",
		Aliases:      []string{"substance", "composition", "fabric", "construction material"},
		ElementTypes: []string{"paragraph", "table_cell", "list_item"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type: ontology.RuleTypeKeyword,
				Keywords: []string{
					"wood", "metal", "plastic", "steel", "aluminum", "glass", "concrete",
					"fabric", "cotton", "polyester", "leather",
				},
			},
		},
	},
	"texture": {
		EntityType:   "texture",
		Description:  "Surface texture or finish",
		Aliases:      []string{"finish", "surface", "feel"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},
	"shape": {
		EntityType:   "shape",
		Description:  "Geometric shape or form",
		Aliases:      []string{"form", "geometry", "configuration"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type: ontology.RuleTypeKeyword,
				Keywords: []string{
					"round", "square", "rectangular", "circular", "oval", "triangular",
					"hexagonal", "cylindrical", "spherical",
				},
			},
		},
	},

	// ========================================
	// TEMPORAL ENTITIES
	// ========================================
	"date": {
		EntityType:   "date",
		Description:  "Calendar date",
		Aliases:      []string{"day", "calendar date", "date"},
		ElementTypes: []string{"paragraph", "table_cell", "heading"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b\d{1,2}/\d{1,2}/\d{2,4}\b`,
			},
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},?\s+\d{4}\b`,
			},
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b\d{4}-\d{2}-\d{2}\b`,
			},
		},
	},
	"time": {
		EntityType:   "time",
		Description:  "Time of day",
		Aliases:      []string{"hour", "timestamp", "clock time", "time"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b\d{1,2}:\d{2}(?::\d{2})?\s*(?:AM|PM|am|pm)?\b`,
			},
		},
	},
	"duration": {
		EntityType:   "duration",
		Description:  "Time duration or period",
		Aliases:      []string{"period", "timespan", "interval", "length"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\d+\s*(?:hours?|minutes?|seconds?|days?|weeks?|months?|years?)`,
			},
		},
	},

	// ========================================
	// CONTACT ENTITIES
	// ========================================
	"email": {
		EntityType:   "email",
		Description:  "Email address",
		Aliases:      []string{"email address", "electronic mail", "e-mail"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
			},
		},
	},
	"phone": {
		EntityType:   "phone",
		Description:  "Phone number",
		Aliases:      []string{"telephone", "phone number", "mobile", "cell", "contact number"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`,
			},
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\(\d{3}\)\s*\d{3}-\d{4}`,
			},
		},
	},
	"url": {
		EntityType:   "url",
		Description:  "Web URL or hyperlink",
		Aliases:      []string{"link", "website", "web address", "hyperlink"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\bhttps?://[^\s]+`,
			},
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\bwww\.[^\s]+`,
			},
		},
	},

	// ========================================
	// NUMERIC ENTITIES
	// ========================================
	"percentage": {
		EntityType:   "percentage",
		Description:  "Percentage value or rate",
		Aliases:      []string{"rate", "percent", "ratio", "proportion"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b\d+(?:\.\d+)?%`,
			},
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b\d+(?:\.\d+)?\s+percent\b`,
			},
		},
	},
	"number": {
		EntityType:   "number",
		Description:  "Numeric value or quantity",
		Aliases:      []string{"quantity", "count", "amount", "figure"},
		ElementTypes: []string{"paragraph", "table_cell"},
		SampleRules: []ontology.ExtractionRule{
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b\d+(?:,\d{3})*(?:\.\d+)?\b`,
			},
		},
	},

	// ========================================
	// IDENTIFIER ENTITIES
	// ========================================
	"id_number": {
		EntityType:   "id_number",
		Description:  "Generic identifier or ID number",
		Aliases:      []string{"ID", "identifier", "reference number", "ID number"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},
	"code": {
		EntityType:   "code",
		Description:  "Alphanumeric code or classification",
		Aliases:      []string{"classification code", "product code", "reference code"},
		ElementTypes: []string{"paragraph", "table_cell"},
	},
}

// GetCommonEntityTemplate retrieves a common entity template by name
func GetCommonEntityTemplate(entityType string) (EntityTemplate, bool) {
	template, exists := CommonEntityTemplates[entityType]
	return template, exists
}

// ListCommonEntityTypes returns all common entity type names
func ListCommonEntityTypes() []string {
	types := make([]string, 0, len(CommonEntityTemplates))
	for entityType := range CommonEntityTemplates {
		types = append(types, entityType)
	}
	return types
}
