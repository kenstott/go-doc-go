# Global Domain Structure (37 Entity Types)

## Overview

The global domain provides **37 reusable entity types** organized by the **6 W's framework** (Who, What, Where, When, Why, hoW):
- **12 top-level types** (2 WHO, 3 WHAT, 1 WHERE, 2 WHEN, 2 WHY, 2 HOW)
- **25 subtypes** (organized under top-level parents)

These types serve as **baseline templates** that can be:
1. **Used directly** as extraction targets (leaves)
2. **Extended** by domain-specific children
3. **Inherited from** for extraction rule patterns

---

## WHO (12 total: 2 top-level + 10 subtypes)

### person (top-level, w_category: who)

**Description**: Individual human being

**Baseline Extraction Rules**:
- Pattern: `(?P<name>[A-Z][a-z]+ [A-Z][a-z]+)` (capitalized two-word names)
- Dictionary filter: Require unknown words (proper nouns)
- Semantic filter: Personal pronouns, biographical context

**Children** (3 subtypes):

#### public_figure
- **parent_type**: `global.person`
- **Description**: Notable public figure or celebrity
- **Additional patterns**: Title prefixes, media mentions

#### executive
- **parent_type**: `global.person`
- **Description**: Company executive or senior leader
- **Additional patterns**: C-suite titles, leadership roles

#### employee
- **parent_type**: `global.person`
- **Description**: Employee or staff member
- **Additional patterns**: Employment context, role mentions

---

### organization (top-level, w_category: who)

**Description**: Company, institution, agency, or group

**Baseline Extraction Rules**:
- Pattern: `(?P<name>[A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,3}\s+(?:Inc|Corp|LLC|Ltd))` (legal entities)
- Pattern: `(?P<name>The\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:Foundation|Institute))` (institutions)
- Semantic filter: Corporate actions, business operations

**Children** (7 subtypes):

#### business
- **parent_type**: `global.organization`
- **Description**: For-profit companies, corporations
- **Additional patterns**: Revenue, earnings, products, services

#### nonprofit
- **parent_type**: `global.organization`
- **Description**: Charities, foundations, NGOs
- **Additional patterns**: Donations, grants, mission statements

#### government
- **parent_type**: `global.organization`
- **Description**: Government agencies, departments, bureaus
- **Additional patterns**: Regulations, policies, public sector

#### educational
- **parent_type**: `global.organization`
- **Description**: Universities, schools, research institutions
- **Additional patterns**: Degrees, courses, faculty, students

#### healthcare
- **parent_type**: `global.organization`
- **Description**: Hospitals, clinics, medical centers
- **Additional patterns**: Patients, treatments, medical services

#### religious
- **parent_type**: `global.organization`
- **Description**: Churches, temples, mosques, religious institutions
- **Additional patterns**: Worship, faith, clergy, congregation

#### media
- **parent_type**: `global.organization`
- **Description**: News outlets, publishers, broadcasters
- **Additional patterns**: Reporting, journalism, publication

---

## WHAT (9 total: 3 top-level + 6 subtypes)

### document (top-level, w_category: what)

**Description**: Referenced documents, reports, papers, publications

**Baseline Extraction Rules**:
- Pattern: `(?P<name>\"[^\"]+\")` (quoted titles)
- Pattern: `(?P<name>[A-Z][^.!?]+(?:Report|Paper|Study|Document))` (document types)
- Proximity: Published, authored, cited

**Children**: None (standalone)

---

### identifier (top-level, w_category: what)

**Description**: Generic identifier or reference code

**Baseline Extraction Rules**:
- Pattern: `(?P<name>[A-Z0-9-]+)` (alphanumeric codes)

**Children** (5 subtypes):

#### email
- **parent_type**: `global.identifier`
- **Description**: Email address
- **Pattern**: `(?P<name>[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,})`

#### phone
- **parent_type**: `global.identifier`
- **Description**: Phone number
- **Pattern**: `(?P<name>\d{3}[-.]?\d{3}[-.]?\d{4})`
- **Pattern**: `(?P<name>\(\d{3}\)\s*\d{3}-\d{4})`

#### url
- **parent_type**: `global.identifier`
- **Description**: Web URL or hyperlink
- **Pattern**: `(?P<name>https?://[^\s]+)`
- **Pattern**: `(?P<name>www\.[^\s]+)`

#### code
- **parent_type**: `global.identifier`
- **Description**: Alphanumeric code or classification
- **Pattern**: `(?P<name>[A-Z]{2,5}-\d{3,6})`

#### id_number
- **parent_type**: `global.identifier`
- **Description**: Generic ID number
- **Pattern**: `(?P<name>ID[:\s]*[A-Z0-9-]+)`

---

### role (top-level, w_category: what)

**Description**: Job title, position, or functional role

**Note**: Role is WHAT (a position/function), not WHO (a person)

**Baseline Extraction Rules**:
- Phrase list: CEO, CFO, CTO, COO, President, Manager, Director, Professor, etc.
- Pattern: `(?P<name>Chief\s+[A-Z][a-z]+\s+Officer)`
- Proximity: Appointed, serves as, position

**Children**: None (standalone)

---

## WHERE (6 total: 1 top-level + 5 subtypes)

### location (top-level, w_category: where)

**Description**: Geographic place or position

**Baseline Extraction Rules**:
- Pattern: `(?P<name>[A-Z][a-z]+(?:,\s*[A-Z]{2})?)`
- Semantic filter: Geographic context, spatial references

**Children** (5 subtypes):

#### city
- **parent_type**: `global.location`
- **Description**: Municipality, town, or urban area
- **Pattern**: `(?P<name>[A-Z][a-z]+(?:\s+[A-Z][a-z]+)?,\s*[A-Z]{2})`

#### country
- **parent_type**: `global.location`
- **Description**: Country or nation
- **Phrase list**: United States, Canada, Mexico, United Kingdom, etc.

#### region
- **parent_type**: `global.location`
- **Description**: State, province, or geographic region
- **Phrase list**: California, Texas, Ontario, etc.

#### address
- **parent_type**: `global.location`
- **Description**: Full mailing or physical address
- **Pattern**: `(?P<name>\d+\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:St|Ave|Rd|Dr|Blvd))`

#### building
- **parent_type**: `global.location`
- **Description**: Building, structure, or facility
- **Proximity**: Located, situated, facility

---

## WHEN (4 total: 2 top-level + 2 subtypes)

### date (top-level, w_category: when)

**Description**: Calendar date or temporal reference

**Baseline Extraction Rules**:
- Pattern: `(?P<name>\d{1,2}/\d{1,2}/\d{2,4})`
- Pattern: `(?P<name>(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},?\s+\d{4})`
- Pattern: `(?P<name>\d{4}-\d{2}-\d{2})`

**Children** (2 subtypes):

#### time
- **parent_type**: `global.date`
- **Description**: Time of day
- **Pattern**: `(?P<name>\d{1,2}:\d{2}(?::\d{2})?\s*(?:AM|PM|am|pm)?)`

#### duration
- **parent_type**: `global.date`
- **Description**: Time duration or period
- **Pattern**: `(?P<name>\d+\s*(?:hours?|minutes?|seconds?|days?|weeks?|months?|years?))`

---

### event (top-level, w_category: when)

**Description**: Occurrence with start/end times, significant happening

**Baseline Extraction Rules**:
- Proximity: Occurred, happened, took place, started, ended
- Semantic filter: Temporal boundaries, significant occurrences

**Children**: None (standalone)

---

## WHY (2 total: 2 top-level + 0 subtypes)

### assertion (top-level, w_category: why)

**Description**: Claims, requirements, declarations, statements presented as fact

**Baseline Extraction Rules**:
- Proximity: Must, shall, requires, declares, states
- Pattern: `(?P<name>.+(?:must|shall|requires).+)`
- Semantic filter: Normative language, declarative statements

**Children**: None (standalone)

---

### hypothesis (top-level, w_category: why)

**Description**: Testable, provisional explanations, theories

**Baseline Extraction Rules**:
- Proximity: If, then, hypothesis, theory, proposes, suggests
- Pattern: `(?P<name>If\s+.+,?\s+then\s+.+)`
- Semantic filter: Conditional statements, scientific reasoning

**Children**: None (standalone)

---

## HOW (5 total: 2 top-level + 3 subtypes)

### process (top-level, w_category: how)

**Description**: Multi-step procedure with ordered actions

**Baseline Extraction Rules**:
- Pattern: `(?P<name>[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:Process|Procedure|Protocol))` (procedural names)
- Proximity: Step, steps, procedure, perform, conduct, execute
- Semantic filter: Sequential actions, workflow context

**Children** (3 subtypes):

#### algorithm
- **parent_type**: `global.process`
- **Description**: Computational procedure with defined steps and logic
- **Additional patterns**: Algorithm suffix, computational keywords

#### protocol
- **parent_type**: `global.process`
- **Description**: Standardized, formal procedure with defined rules
- **Additional patterns**: Protocol suffix, standard/guideline keywords

#### workflow
- **parent_type**: `global.process`
- **Description**: Business or operational process flow with stages
- **Additional patterns**: Workflow/pipeline suffix, stage/phase keywords

---

### method (top-level, w_category: how)

**Description**: Technique or approach for accomplishing something

**Baseline Extraction Rules**:
- Pattern: `(?P<name>[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:Method|Technique|Approach))` (methodological names)
- Proximity: Method, technique, approach, using, via, by means of
- Semantic filter: Technique for achieving result

**Children**: None (standalone)

---

## File Organization

The 37 types are split into 6 files by W category:

### global_who.go (~300 lines)
- person (+ 3 children: public_figure, executive, employee)
- organization (+ 7 children: business, nonprofit, government, educational, healthcare, religious, media)

### global_what.go (~250 lines)
- document (standalone)
- identifier (+ 5 children: email, phone, url, code, id_number)
- role (standalone)

### global_where.go (~150 lines)
- location (+ 5 children: city, country, region, address, building)

### global_when.go (~150 lines)
- date (+ 2 children: time, duration)
- event (standalone)

### global_why.go (~100 lines)
- assertion (standalone)
- hypothesis (standalone)

### global_how.yaml (~180 lines)
- process (+ 3 children: algorithm, protocol, workflow)
- method (standalone)

**Total**: ~1,130 lines across 6 files (avg ~188 lines/file)

---

## Usage Patterns

### Pattern 1: Use Global Type Directly
```yaml
# Medical domain uses global person as-is
- entity_type: person
  domain: medical
  parent_type: ""  # No parent (using global directly)
  w_category: who
  confidence: 0.80
  extraction_rules:
    - instance_name: (?P<name>[A-Z][a-z]+ [A-Z][a-z]+)
      proximity:
        keywords: [patient, doctor]
```

### Pattern 2: Extend Global Type with Domain-Specific Child
```yaml
# Medical domain creates physician as child of global person
- entity_type: physician
  domain: medical
  parent_type: global.person
  w_category: who
  confidence: 0.90
  extraction_rules:
    - instance_name: (?P<name>Dr\. [A-Z][a-z]+ [A-Z][a-z]+)
      proximity:
        keywords: [patient, diagnosis]
```

### Pattern 3: Create Domain-Specific Type (No Global Parent)
```yaml
# Medical domain creates condition (no global equivalent)
- entity_type: condition
  domain: medical
  parent_type: ""  # No parent
  w_category: what
  confidence: 0.85
  extraction_rules:
    - phrase_list: [diabetes, hypertension, asthma]
      semantic:
        reference_concepts: [medical diagnosis]
```

---

## Global Relationship Rules

### Overview

Global domain catalogs define **common relationship patterns** that apply when global entities are extracted (not synthesized). These rules can be **inherited and refined** by domain catalogs using the `parent_relationship` field.

### Design Principles for Global Relationships

1. **Low confidence** (0.60-0.70) - General patterns applicable across domains
2. **Common vocabulary** - Universal signal words like "works at", "located in"
3. **Broad applicability** - Patterns that work in multiple domains
4. **Inheritance-ready** - Designed to be extended by domain-specific rules

### WHO Category Relationships

#### person_works_at_organization
```yaml
entity_relationship_rules:
  - name: person_works_at_organization
    source_entity_type: person
    target_entity_type: organization
    relationship_type: part_of
    description: Person employed by or affiliated with organization
    confidence: 0.70
    extraction_patterns:
      - type: proximity
        signal_words: [works at, employed by, employee of, works for]
        max_distance: 50
        direction: forward
```

**Domain inheritance example**:
```yaml
# Medical domain extends this rule
- name: physician_works_at_hospital
  parent_relationship: global.person_works_at_organization
  source_entity_type: physician  # More specific
  target_entity_type: hospital   # More specific
  confidence: 0.85  # Higher confidence
  extraction_patterns:
    - type: proximity
      signal_words: [practicing at, on staff at, attending physician at]
      max_distance: 50
```

#### person_located_in_location
```yaml
  - name: person_located_in_location
    source_entity_type: person
    target_entity_type: location
    relationship_type: located_in
    description: Person resides in or is from location
    confidence: 0.65
    extraction_patterns:
      - type: proximity
        signal_words: [lives in, from, resides in, based in, citizen of]
        max_distance: 50
        direction: forward
```

#### organization_located_in_location
```yaml
  - name: organization_located_in_location
    source_entity_type: organization
    target_entity_type: location
    relationship_type: located_in
    description: Organization headquartered in or operates in location
    confidence: 0.70
    extraction_patterns:
      - type: proximity
        signal_words: [headquartered in, based in, located in, operates in]
        max_distance: 50
        direction: forward
```

### WHAT Category Relationships

#### document_authored_by_person
```yaml
  - name: document_authored_by_person
    source_entity_type: document
    target_entity_type: person
    relationship_type: created_by
    description: Document created or authored by person
    confidence: 0.75
    extraction_patterns:
      - type: proximity
        signal_words: [authored by, written by, created by, by]
        max_distance: 30
        direction: forward
      - type: regex
        pattern: '(?P<document>.+?)\s+by\s+(?P<person>[A-Z][a-z]+ [A-Z][a-z]+)'
```

#### document_published_by_organization
```yaml
  - name: document_published_by_organization
    source_entity_type: document
    target_entity_type: organization
    relationship_type: created_by
    description: Document published or released by organization
    confidence: 0.70
    extraction_patterns:
      - type: proximity
        signal_words: [published by, released by, issued by]
        max_distance: 40
        direction: forward
```

### WHEN Category Relationships

#### event_occurred_at_date
```yaml
  - name: event_occurred_at_date
    source_entity_type: event
    target_entity_type: date
    relationship_type: occurred_at
    description: Event occurred on or at date
    confidence: 0.75
    extraction_patterns:
      - type: proximity
        signal_words: [on, at, during, in]
        max_distance: 20
        direction: forward
```

#### event_located_in_location
```yaml
  - name: event_located_in_location
    source_entity_type: event
    target_entity_type: location
    relationship_type: located_in
    description: Event took place at location
    confidence: 0.70
    extraction_patterns:
      - type: proximity
        signal_words: [at, in, took place at, held in]
        max_distance: 30
        direction: forward
```

### HOW Category Relationships

#### entity_uses_method
```yaml
  - name: entity_uses_method
    source_entity_type: ""  # Any entity type
    target_entity_type: method
    relationship_type: uses
    description: Entity uses method to accomplish goal or perform action
    confidence: 0.65
    extraction_patterns:
      - type: proximity
        signal_words: [uses, using, via, by means of, through, with]
        max_distance: 30
        direction: forward
```

#### entity_follows_process
```yaml
  - name: entity_follows_process
    source_entity_type: ""  # Any entity type
    target_entity_type: process
    relationship_type: follows
    description: Entity follows multi-step process or procedure
    confidence: 0.70
    extraction_patterns:
      - type: proximity
        signal_words: [follows, according to, as per, per, implements]
        max_distance: 40
        direction: forward
```

#### action_step_of_process
```yaml
  - name: action_step_of_process
    source_entity_type: ""  # Any action/verb phrase
    target_entity_type: process
    relationship_type: part_of
    description: Action or step is part of larger process (with ordering)
    confidence: 0.75
    extraction_patterns:
      - type: proximity
        signal_words: [step, stage, phase, first, second, next, then, finally]
        max_distance: 20
        direction: bidirectional
```

### Total Global Relationship Rules

**11 relationship rules** across 5 W categories:
- WHO: 3 rules (person-organization, person-location, organization-location)
- WHAT: 2 rules (document-person, document-organization)
- WHEN: 2 rules (event-date, event-location)
- WHY: 1 rule (assertion-related_to-assertion)
- HOW: 3 rules (entity-uses-method, entity-follows-process, action-step-of-process)

### Inheritance Benefits

**Example: Medical domain extends 3 global rules**
```yaml
# Inherits from global.person_works_at_organization
- physician_works_at_hospital (confidence: 0.85)
- nurse_works_at_hospital (confidence: 0.80)
- technician_works_at_facility (confidence: 0.75)

# Inherits from global.document_authored_by_person
- clinical_trial_led_by_investigator (confidence: 0.90)

# Inherits from global.event_occurred_at_date
- treatment_administered_at_date (confidence: 0.85)
```

**Result**: Medical domain gets 3 global patterns + domain-specific refinements with just 5 rules, but effectively has 8+ relationship patterns after inheritance resolution.

---

## Design Principles

1. **High-value, reusable types only** - No low-value types like "color", "size", "texture"
2. **Domain-agnostic** - Global types work across all domains
3. **Baseline patterns** - Provide starting point, domains customize
4. **Organized by W category** - Clear mental model
5. **Hierarchical where appropriate** - Parent-child for natural groupings
6. **Standalone where not** - Don't force hierarchies

---

## Summary Statistics

### Entity Types

| W Category | Top-Level | Subtypes | Total |
|------------|-----------|----------|-------|
| WHO        | 2         | 10       | 12    |
| WHAT       | 3         | 5        | 8     |
| WHERE      | 1         | 5        | 6     |
| WHEN       | 2         | 2        | 4     |
| WHY        | 2         | 0        | 2     |
| HOW        | 2         | 3        | 5     |
| **TOTAL**  | **12**    | **25**   | **37** |

### Relationship Rules

| W Category | Relationship Rules | Example |
|------------|--------------------|---------|
| WHO        | 3                  | person_works_at_organization |
| WHAT       | 2                  | document_authored_by_person |
| WHERE      | 0                  | (covered by located_in rules) |
| WHEN       | 2                  | event_occurred_at_date |
| WHY        | 1                  | assertion_supports_assertion |
| HOW        | 3                  | entity_uses_method |
| **TOTAL**  | **11**             | |

---

## Next Steps

Proceed to:
- [03_UNIFIED_EXTRACTION.md](03_UNIFIED_EXTRACTION.md) - Learn the extraction rule structure
- [04_HIERARCHY_SYSTEM.md](04_HIERARCHY_SYSTEM.md) - Understand parent-child relationships
- [06_IMPLEMENTATION_STEPS.md](06_IMPLEMENTATION_STEPS.md) - Start implementing