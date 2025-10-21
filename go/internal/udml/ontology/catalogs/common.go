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
		ElementTypes: []string{"paragraph", "div", "list_item", "table_cell"}, // Added div, removed heading
		SampleRules: []ontology.ExtractionRule{
			// HIGH CONFIDENCE: Person with title prefix (Dr., Prof., Mr., Mrs., Ms.)
			// Pattern excludes whitespace characters (tabs, newlines) via [^\s\n\r\t] requirement
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b(?:Dr|Prof|Professor|Mr|Mrs|Ms|Miss)\.?\s+[A-Z][a-z]{1,20}(?:\s+[A-Z]\.?)?\s+[A-Z][a-z]{1,20}\b`,
				// Dictionary-based linguistic validation
				DictionaryFilter: &ontology.DictionaryFilter{
					RequireUnknownWords:   true,             // At least one word not in dictionary (proper name)
					MaxKnownWordsRatio:    0.5,              // Max 50% of words can be common words
					RejectIfAllPOS:        []string{"noun"}, // Reject if ALL words are common nouns
					RejectIfAllCategories: []string{"place", "ui_action", "temporal"}, // Reject if all words are places/UI/temporal
					RejectCategoryCombinations: [][]string{
						{"place", "common_noun"},   // "Capitol Reef", "Bloomsbury Academic"
						{"common_noun", "common_noun"}, // "World Health"
						{"ui_action", "ui_action"}, // "Donate Create"
					},
				},
			},
			// MEDIUM CONFIDENCE: Full names with optional middle initial or suffix
			// Pattern excludes whitespace characters (tabs, newlines)
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b[A-Z][a-z]{1,20}(?:\s+[A-Z]\.)?\s+[A-Z][a-z]{1,20}(?:\s+(?:Jr|Sr|II|III|IV)\.?)?\b`,
				// Dictionary-based linguistic validation - requires proper names, not common nouns
				DictionaryFilter: &ontology.DictionaryFilter{
					RequireUnknownWords:   true,             // At least one word NOT in dictionary (proper name signal)
					MaxKnownWordsRatio:    0.5,              // Max 50% can be known words (allows "John" if rare)
					RejectIfAllPOS:        []string{"noun"}, // Reject if ALL words are common nouns
					RejectIfAllCategories: []string{"place", "ui_action", "temporal"}, // Reject if all words are places/UI/temporal
					RejectCategoryCombinations: [][]string{
						{"place", "common_noun"},   // "Capitol Reef", "Bloomsbury Academic" (publisher)
						{"common_noun", "common_noun"}, // "World Health"
						{"ui_action", "ui_action"}, // "Donate Create"
					},
				},
				// Semantic filter for additional context validation
				SemanticFilter: &ontology.SemanticFilter{
					ReferenceConcepts: []string{
						"individual person with biography or credentials",
						"author or creator attribution to individual",
						"personal pronouns (he, she, his, her) referencing the name",
					},
					SimilarityThreshold: 0.65,
				},
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
		SampleRules: []ontology.ExtractionRule{
			{
				Type: ontology.RuleTypeKeyword,
				Keywords: []string{
					// C-Suite and Executive Roles
					"CEO", "CFO", "CTO", "COO", "CIO", "CMO", "CHRO", "CPO", "CLO",
					"Chief Executive Officer", "Chief Financial Officer", "Chief Technology Officer",
					"Chief Operating Officer", "Chief Information Officer", "Chief Marketing Officer",
					"Chief Human Resources Officer", "Chief Product Officer", "Chief Legal Officer",
					"Chief Data Officer", "Chief Security Officer", "Chief Strategy Officer",

					// Corporate Leadership
					"President", "Vice President", "Senior Vice President", "Executive Vice President",
					"VP", "SVP", "EVP", "Managing Director", "General Manager",
					"Chairman", "Chairwoman", "Chairperson", "Board Member", "Director",
					"Executive Director", "Senior Director", "Associate Director",

					// Management Roles
					"Manager", "Senior Manager", "Department Manager", "Project Manager",
					"Program Manager", "Product Manager", "Account Manager", "Operations Manager",
					"Regional Manager", "District Manager", "Branch Manager",
					"Team Leader", "Team Lead", "Supervisor", "Coordinator",

					// Professional/Technical Roles
					"Engineer", "Senior Engineer", "Lead Engineer", "Principal Engineer", "Staff Engineer",
					"Architect", "Senior Architect", "Technical Architect", "Solutions Architect",
					"Developer", "Senior Developer", "Software Developer", "Lead Developer",
					"Analyst", "Senior Analyst", "Data Analyst", "Business Analyst", "Systems Analyst",
					"Scientist", "Senior Scientist", "Research Scientist", "Principal Scientist",
					"Researcher", "Research Associate", "Research Fellow",
					"Consultant", "Senior Consultant", "Principal Consultant",
					"Specialist", "Senior Specialist", "Technical Specialist",
					"Designer", "Senior Designer", "Lead Designer", "UX Designer", "UI Designer",

					// Academic Roles
					"Professor", "Associate Professor", "Assistant Professor", "Adjunct Professor",
					"Lecturer", "Senior Lecturer", "Instructor", "Teaching Assistant",
					"Dean", "Associate Dean", "Department Chair", "Department Head",
					"Researcher", "Postdoctoral Researcher", "Research Assistant",
					"Chancellor", "Provost", "Registrar",

					// Education/Training
					"Teacher", "Head Teacher", "Lead Teacher", "Educator",
					"Tutor", "Mentor", "Coach", "Trainer", "Facilitator",
					"Student", "Graduate Student", "Undergraduate", "PhD Candidate",
					"Principal", "Vice Principal", "Headmaster", "Superintendent",

					// Political/Government Roles
					"President", "Vice President", "Governor", "Lieutenant Governor",
					"Senator", "Congressman", "Congresswoman", "Representative",
					"Mayor", "Deputy Mayor", "Councilmember", "Alderman",
					"Minister", "Prime Minister", "Secretary", "Undersecretary",
					"Ambassador", "Diplomat", "Consul", "Attaché",
					"Judge", "Justice", "Chief Justice", "Magistrate",
					"Commissioner", "Director General", "Administrator",

					// Medical/Healthcare Roles
					"Doctor", "Physician", "Surgeon", "Cardiologist", "Oncologist",
					"Nurse", "Registered Nurse", "Nurse Practitioner", "Head Nurse",
					"Physician Assistant", "Medical Assistant", "Paramedic",
					"Pharmacist", "Therapist", "Psychologist", "Psychiatrist",
					"Dentist", "Veterinarian", "Medical Director", "Chief Medical Officer",

					// Legal Roles
					"Lawyer", "Attorney", "Counsel", "General Counsel", "Legal Counsel",
					"Partner", "Senior Partner", "Associate", "Senior Associate",
					"Paralegal", "Legal Assistant",

					// Creative/Media Roles
					"Author", "Writer", "Journalist", "Reporter", "Editor",
					"Senior Editor", "Managing Editor", "Copy Editor",
					"Producer", "Director", "Cinematographer", "Photographer",
					"Artist", "Illustrator", "Graphic Designer", "Creative Director",

					// Sales/Marketing
					"Sales Representative", "Account Executive", "Sales Director",
					"Marketing Manager", "Marketing Director", "Brand Manager",

					// Operations/Support
					"Administrator", "Office Manager", "Executive Assistant",
					"Officer", "Senior Officer", "Operations Officer",
					"Technician", "Senior Technician", "Support Specialist",

					// Family/Social Roles
					"Parent", "Mother", "Father", "Guardian",
					"Child", "Son", "Daughter", "Sibling",
					"Spouse", "Partner", "Husband", "Wife",
				},
			},
		},
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
	"organization": {
		EntityType:   "organization",
		Description:  "Company, institution, agency, or group",
		Aliases:      []string{"company", "institution", "agency", "corporation", "firm", "business"},
		ElementTypes: []string{"paragraph", "div", "list_item", "table_cell"}, // Removed heading
		SampleRules: []ontology.ExtractionRule{
			// HIGH CONFIDENCE: Legal entity with suffix
			// Restricts to max 5 total words to avoid sentence fragments
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,3}\s+(?:Inc|Corp|Corporation|LLC|Ltd|Limited|Co|Company|Group|Partners|LP|LLP)\.?\b`,
				// No semantic filter needed - legal suffix is strong signal
			},
			// HIGH CONFIDENCE: "The [Name] [OrgType]" pattern
			// Restricts to max 5 total words (The + 3 name words + type word)
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\bThe\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,2}\s+(?:Foundation|Institute|Organization|Association|Society|Council|Commission|Agency|Department|Ministry|Bureau|Office)\b`,
				// No semantic filter needed - structure is strong signal
			},
			// HIGH CONFIDENCE: Specific institutional patterns
			// Universities, colleges, institutes with strong contextual indicators
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b(?:[A-Z][a-z]+\s+){0,3}(?:University|Institute|College|Laboratory|Center|Centre)(?:\s+of\s+[A-Z][a-z]+)?\b`,
			},
			// MEDIUM CONFIDENCE: Acronyms (2-5 letters)
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b[A-Z]{2,5}\b`,
				SemanticFilter: &ontology.SemanticFilter{
					ReferenceConcepts: []string{
						"corporate actions (announced, reported, filed, acquired)",
						"business operations (earnings, revenue, products, services)",
						"company or institution as collective entity",
					},
					SimilarityThreshold: 0.70, // Higher threshold for acronyms
				},
			},
			// MEDIUM CONFIDENCE: Multi-word capitalized (3-4 words = likely org)
			// Avoids 5+ word captures that are likely sentence fragments
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+){2,3}\b`,
				SemanticFilter: &ontology.SemanticFilter{
					ReferenceConcepts: []string{
						"organizational structure (headquarters, subsidiary, division)",
						"company or institution as collective entity",
						"specific named organization",
					},
					SimilarityThreshold: 0.70, // Raised from 0.65 to reduce false positives
				},
			},
			// LOW CONFIDENCE: 2-word capitalized (ambiguous - could be person)
			{
				Type:    ontology.RuleTypeRegex,
				Pattern: `\b[A-Z][a-z]+\s+[A-Z][a-z]+\b`,
				SemanticFilter: &ontology.SemanticFilter{
					ReferenceConcepts: []string{
						"corporate actions (announced, reported, filed, acquired)",
						"business operations (earnings, revenue, products, services)",
						"organizational structure (headquarters, subsidiary, division)",
					},
					SimilarityThreshold: 0.75, // Raised from 0.70 due to high ambiguity
				},
			},
		},
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

// LLMValidationTemplate defines LLM validation prompts for entity types
type LLMValidationTemplate struct {
	EntityType          string   // Entity type this validation applies to
	QuestionTemplate    string   // Template for validation question (use {entity} placeholder)
	RejectExamples      []string // Examples of things that should be rejected
	AcceptExamples      []string // Examples of things that should be accepted
	AdditionalGuidance  string   // Optional additional guidance for the LLM
	RequireVeryConfident bool    // If true, emphasizes "VERY confident" language
}

// CommonLLMValidationTemplates defines reusable LLM validation prompts
var CommonLLMValidationTemplates = map[string]LLMValidationTemplate{
	"person": {
		EntityType:          "person",
		QuestionTemplate:    "Is '{entity}' an individual human person's name (not an organization, concept, or place)?",
		RequireVeryConfident: true,
		RejectExamples: []string{
			"Medical Association", // Organization
			"Medicine",            // Concept/field
			"Medical Science",     // Academic field
			"Department of Health", // Government org
			"World Health",        // Org fragment
			"Public Health",       // Field/concept
			"Harvard Medical",     // Institution fragment
		},
		AcceptExamples: []string{
			"John Smith",
			"Dr. Jane Doe",
			"Professor Albert Einstein",
			"Mary Johnson",
			"Dr. Robert Koch",
		},
		AdditionalGuidance: "Answer 'yes' ONLY if you are VERY confident it is an individual human person's name. Answer 'no' for organizations, departments, concepts, academic fields, or anything uncertain.",
	},
	"organization": {
		EntityType:       "organization",
		QuestionTemplate: "Is '{entity}' a specific, real organization name (not a sentence fragment, generic term, or concatenated list)?",
		RequireVeryConfident: true,
		RejectExamples: []string{
			"Endemol Shine Group Takeover Approved By European Commission", // Sentence fragment
			"Agriculture Banking Communications Companies Energy Insurance Manufacturing", // Concatenated list
			"Jesus Christ Ministry Crucifixion Resurrection Great Commission", // Religious text fragment
			"American Revolution War Second Continental Congress", // Historical event description
			"Advanced Manufacturing", // Too generic
			"American Council",       // Incomplete/generic
			"The Institute",          // Too generic
		},
		AcceptExamples: []string{
			"Harvard University",
			"Microsoft Corporation",
			"World Health Organization",
			"Stanford University",
			"General Electric",
			"United Nations",
		},
		AdditionalGuidance: "Answer 'yes' ONLY for specific, real organization names. Answer 'no' for sentence fragments, generic terms (like 'Manufacturing', 'Council'), incomplete names, or concatenated lists of words.",
	},
}

// GetLLMValidationTemplate retrieves an LLM validation template by entity type
func GetLLMValidationTemplate(entityType string) (LLMValidationTemplate, bool) {
	template, exists := CommonLLMValidationTemplates[entityType]
	return template, exists
}

// CommonRelationshipTemplates defines common relationship patterns
var CommonRelationshipTemplates = []RelationshipTemplate{
	// Role-Person relationships
	{
		Name:             "person_has_role",
		SourceType:       "person",
		TargetType:       "role",
		RelationshipType: ontology.RelationshipCustom,
		Description:      "A person holds or performs a role/position",
		SamplePatterns: []string{
			"[Person] is the [Role]",
			"[Person] serves as [Role]",
			"[Person], [Role]",
			"[Role] [Person]",
			"[Person] was appointed [Role]",
			"[Person] worked as [Role]",
			"[Person], who is [Role]",
			"[Person] holds the position of [Role]",
		},
	},
	{
		Name:             "role_held_by",
		SourceType:       "role",
		TargetType:       "person",
		RelationshipType: ontology.RelationshipCustom,
		Description:      "A role is held/performed by a person (inverse of person_has_role)",
		SamplePatterns: []string{
			"[Role] [Person]",
			"The [Role], [Person]",
		},
	},
	// Role-Organization relationships
	{
		Name:             "role_at_organization",
		SourceType:       "role",
		TargetType:       "organization",
		RelationshipType: ontology.RelationshipLocatedIn,
		Description:      "A role exists within an organization",
		SamplePatterns: []string{
			"[Role] at [Organization]",
			"[Role] of [Organization]",
			"[Organization]'s [Role]",
			"[Role] for [Organization]",
		},
	},
	{
		Name:             "person_works_at",
		SourceType:       "person",
		TargetType:       "organization",
		RelationshipType: ontology.RelationshipRelatedTo,
		Description:      "A person is employed by or affiliated with an organization",
		SamplePatterns: []string{
			"[Person] at [Organization]",
			"[Person] works for [Organization]",
			"[Person] of [Organization]",
			"[Person], [Organization]",
			"[Organization] employee [Person]",
		},
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

// GetCommonRelationshipTemplates returns all common relationship templates
func GetCommonRelationshipTemplates() []RelationshipTemplate {
	return CommonRelationshipTemplates
}

// GetRelationshipTemplatesBySourceType returns relationship templates for a given source entity type
func GetRelationshipTemplatesBySourceType(sourceType string) []RelationshipTemplate {
	var templates []RelationshipTemplate
	for _, template := range CommonRelationshipTemplates {
		if template.SourceType == sourceType {
			templates = append(templates, template)
		}
	}
	return templates
}

// GetRelationshipTemplatesByTargetType returns relationship templates for a given target entity type
func GetRelationshipTemplatesByTargetType(targetType string) []RelationshipTemplate {
	var templates []RelationshipTemplate
	for _, template := range CommonRelationshipTemplates {
		if template.TargetType == targetType {
			templates = append(templates, template)
		}
	}
	return templates
}
