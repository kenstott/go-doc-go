# UDML Ontology System - Workflows Guide

## Overview

The UDML Ontology System provides **two distinct workflows** for ontology creation and refinement, both requiring **mandatory user approval** before finalizing any schema. This ensures you maintain complete control over your ontology rules.

## Key Principle: No Auto-Apply

**CRITICAL**: The system NEVER automatically applies ontology rules without user confirmation. All workflows require explicit approval.

---

## Workflow 1: Create New Ontology (Interactive Interview)

Creates a new ontology from scratch through a guided 3-phase interview process.

### When to Use
- Starting from scratch with a new document corpus
- No existing ontology schema
- Need LLM guidance to discover domains, entities, and relationships

### Process Overview

```
Phase 1: Domain Selection
├─ LLM analyzes corpus samples
├─ Suggests relevant domains with reasoning
├─ Asks clarifying questions
└─ ✓ USER CONFIRMS domains before proceeding

Phase 2: Draft Generation (Automated)
├─ LLM generates entity types with extraction rules
├─ LLM generates relationships with patterns
└─ Complete draft schema created

Phase 3: Review & Confirmation (MANDATORY)
├─ User reviews complete schema
├─ Options:
│   ├─ approve  → Finalize and save
│   ├─ refine   → Make iterative changes
│   └─ reject   → Discard entirely
└─ No schema saved without approval
```

### Command

```bash
export ANTHROPIC_API_KEY=your_key_here

# Basic usage (output: ontology.json)
ontology_interview ./output/udml.parquet

# With custom output path
ontology_interview ./corpus.parquet ./my-ontology.json
```

### Phase Details

#### Phase 1: Domain Selection (Interactive)

**LLM Actions:**
1. Analyzes sample texts from corpus
2. Identifies distinct domains (financial, legal, technical, etc.)
3. Provides reasoning with evidence from samples
4. Suggests domain ownership (CFO Office, Legal Department, etc.)
5. Asks 2-3 clarifying questions about use case

**User Actions:**
- Answer clarifying questions
- Review suggested domains
- Approve or request changes

**Example Interaction:**
```
🤖 LLM Analysis:
──────────────────────────────────────────────────────────────

  Domain: financial (confidence: 0.95)
  Reasoning: I see revenue, EBITDA, profit margins in samples 2, 5, 7
  Suggested Owner: CFO Office

  Domain: legal (confidence: 0.88)
  Reasoning: References to entities, jurisdictions, compliance in samples 3, 9
  Suggested Owner: Legal Department

🤖 LLM: What is the primary purpose of these documents?
   You: These are annual reports for public companies

🤖 LLM: Who are the main consumers of this data?
   You: Investors, analysts, and regulators

📋 Final Domain Selection:
──────────────────────────────────────────────────────────────
  ✓ financial - Financial performance and reporting
    Owner: CFO Office
  ✓ legal - Legal entities and compliance
    Owner: Legal Department

❓ Approve these domains? (yes/no): yes

  ✅ Domains confirmed
```

#### Phase 2: Draft Generation (Automated)

**LLM Actions (No User Input Required):**
1. Generates entity types based on:
   - Confirmed domains
   - Top entity frequencies from corpus
   - Sample texts
2. Creates extraction rules for each entity:
   - Keyword matches
   - Regex patterns
   - Text similarity patterns
3. Generates relationships between entities:
   - Text templates
   - Proximity patterns
   - Cooccurrence patterns
4. Assigns confidence levels based on context quality

**Progress Display:**
```
🤖 LLM is generating entity types and relationships...
   (this may take 30-60 seconds)

  ✓ Generated 15 entity mappings
  ✓ Generated 12 relationship rules

  ✅ Draft schema complete
```

#### Phase 3: Review & Confirmation (MANDATORY)

**User Actions (Required):**
- Review complete schema summary
- Choose one of three options:
  - **approve**: Accept schema as-is and save
  - **refine**: Make iterative changes before approving
  - **reject**: Discard schema entirely

**Schema Display:**
```
📋 COMPLETE ONTOLOGY SCHEMA
════════════════════════════════════════════════════════════════════════════════

DOMAINS (2):
────────────────────────────────────────────────────────────────────────────────
  • financial - Financial performance and reporting
    Owner: CFO Office
  • legal - Legal entities and compliance
    Owner: Legal Department

ENTITY TYPES (15):
────────────────────────────────────────────────────────────────────────────────
  [financial domain]
    • revenue - Financial revenue metric (confidence: 0.95)
      Elements: [table_cell, paragraph]
      Rules: 3 extraction patterns
    • company - Business organization (confidence: 0.85)
      Elements: [paragraph, heading]
      Rules: 2 extraction patterns
    ...

  [legal domain]
    • legal_entity - Legal entity registration (confidence: 0.90)
      Elements: [paragraph, table_cell]
      Rules: 4 extraction patterns
    ...

RELATIONSHIPS (12):
────────────────────────────────────────────────────────────────────────────────
  • company_reports_revenue: company → revenue (related_to)
    Company reports financial revenue metric
    Confidence: 0.95 | Patterns: 3
  • company_registered_in_jurisdiction: company → jurisdiction (part_of)
    Company registered in legal jurisdiction
    Confidence: 0.88 | Patterns: 2
  ...

════════════════════════════════════════════════════════════════════════════════

❓ APPROVAL REQUIRED
   Your options:
     1. 'approve' - Accept schema as-is and finalize
     2. 'refine'  - Make iterative changes before approving
     3. 'reject'  - Reject schema entirely

   Your decision (approve/refine/reject):
```

**If User Chooses "refine":**
```
  🔧 REFINEMENT MODE
     Describe the changes you want (be specific):
     > Add extraction rule for stock ticker symbols (e.g., AAPL, MSFT)

  ✓ Changes applied

❓ Review refined schema - approve/refine again/reject?
   Your decision (approve/refine/reject): approve

  ✅ Schema APPROVED

✓ Ontology schema saved to: ontology.json
```

### Time Estimate
- **Total**: 5-10 minutes
- Phase 1: 2-3 minutes (interactive)
- Phase 2: 1-2 minutes (automated)
- Phase 3: 2-5 minutes (review + optional refinement)

### Output

JSON file containing:
- Domain definitions with ownership
- Entity extraction rules (keyword, regex, similarity patterns)
- Relationship patterns
- Confidence levels for each rule
- Metadata (schema name, version, creation timestamp)

---

## Workflow 2: Refine Existing Ontology

Improves an existing ontology by analyzing it against the corpus to find gaps and issues.

### When to Use
- You have an existing ontology that needs improvement
- Corpus has changed/evolved since ontology was created
- Want to identify missing entities or relationships
- Need to fix overly broad or narrow extraction rules

### Process Overview

```
Phase 1: Analysis
├─ LLM compares existing schema vs corpus
├─ Identifies gaps (missing entities/relationships)
├─ Identifies issues (bad rules, wrong confidence)
└─ Suggests improvements

Phase 2: Suggested Changes (Interactive)
├─ LLM proposes specific changes one by one
├─ For each change:
│   ├─ Shows reasoning and proposed modification
│   └─ USER APPROVES/REJECTS individually
└─ Apply only approved changes

Phase 3: Review & Confirmation (MANDATORY)
├─ User reviews refined schema
├─ Options:
│   ├─ approve  → Save refined schema
│   ├─ refine   → Make additional changes
│   └─ reject   → Revert to original schema
└─ No changes saved without approval
```

### Command

```bash
export ANTHROPIC_API_KEY=your_key_here

# Refine existing ontology
ontology_interview --refine ./current-ontology.json ./corpus.parquet ./improved-ontology.json
```

### Phase Details

#### Phase 1: Analysis

**LLM Actions:**
1. Loads existing ontology schema
2. Samples corpus
3. Identifies gaps, issues, and improvements

**Analysis Output:**
```
🤖 LLM Analysis Results:
────────────────────────────────────────────────────────────────────────────────

  GAPS FOUND (3):
    1. Email addresses appear frequently but aren't extracted
       Evidence: Seen in samples 3, 7, 12, 18
    2. Phone numbers present but no extraction rule
       Evidence: Samples 5, 11, 14
    3. Missing relationship: person_works_at_company
       Evidence: Pattern "{person}, {title} at {company}" in samples 8, 15

  ISSUES FOUND (2):
    1. [high] Regex pattern too broad, will match false positives
       Location: entity_type: company, domain: financial, rule 2
    2. [medium] Confidence too high for narrative text extraction
       Location: entity_type: executive, confidence: 0.95

  IMPROVEMENTS SUGGESTED (2):
    1. Add text_similarity rule for contextual extraction of roles
       Benefit: Better recall for person entities in unstructured text
    2. Split company entity into two mappings (structured vs unstructured)
       Benefit: More accurate confidence modeling
```

#### Phase 2: Suggested Changes (Interactive)

**LLM Actions:**
- Proposes specific, actionable changes
- One change at a time for user review

**User Actions:**
- Review each proposed change
- Approve, reject, or skip each one

**Example Interaction:**
```
🤖 LLM suggests 5 changes:

  Change 1/5:
    Type: add_entity_mapping
    Reason: Email addresses appear frequently but not extracted
    Proposed:
    {
      "entity_type": "email",
      "domain": "legal",
      "description": "Email addresses",
      "element_types": ["paragraph", "table_cell"],
      "confidence": 0.95,
      "extraction_rules": [
        {
          "type": "regex_pattern",
          "pattern": "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b"
        }
      ]
    }

    Approve this change? (yes/no/skip): yes
    ✓ Approved

  Change 2/5:
    Type: fix_rule
    Reason: Company regex too broad, matches false positives like "New York"
    Location: entity_type: company, domain: financial
    Proposed:
    {
      "remove_rule_index": 1,
      "add_rule": {
        "type": "regex_pattern",
        "pattern": "\\b[A-Z][a-z]+(?:\\s+[A-Z][a-z]+)*\\s+(?:Inc|Corp|LLC|Ltd)\\.?\\b"
      }
    }

    Approve this change? (yes/no/skip): yes
    ✓ Approved

  Change 3/5:
    ...

  Applying 3 approved changes...
  ✅ Changes applied successfully
```

#### Phase 3: Review & Confirmation (MANDATORY)

Same as Workflow 1 Phase 3 - user reviews the refined schema and must approve before saving.

### Time Estimate
- **Total**: 7-12 minutes
- Phase 1: 2-3 minutes (automated analysis)
- Phase 2: 3-6 minutes (review and approve changes)
- Phase 3: 2-3 minutes (final review)

### Output

JSON file containing refined ontology with:
- Original domains (preserved unless changed)
- Updated entity extraction rules
- New entities (approved additions)
- Fixed rules (approved fixes)
- New relationships (approved additions)
- Updated confidence levels

---

## Comparison: When to Use Which Workflow

| Criterion | New Ontology | Refine Existing |
|-----------|-------------|-----------------|
| **Starting Point** | No ontology | Existing ontology |
| **Use Case** | New corpus analysis | Improve existing schema |
| **Domain Selection** | Interactive with LLM | Preserved from existing |
| **Entity Discovery** | From scratch | Gap analysis |
| **User Involvement** | Domain confirmation + final approval | Change-by-change approval |
| **Time** | 5-10 minutes | 7-12 minutes |
| **Output** | Complete new schema | Refined schema |

---

## Best Practices

### 1. Always Review Before Approval

**Why**: The LLM is sophisticated but not perfect. Always review:
- Entity types: Are they relevant to your use case?
- Extraction rules: Will they match what you need?
- Confidence levels: Do they reflect extraction context quality?
- Relationships: Do they capture important connections?

### 2. Use Refinement for Iterative Improvement

**Strategy**:
1. Create initial ontology with Workflow 1
2. Test extraction on real documents
3. Refine with Workflow 2 when you identify issues
4. Repeat refinement as corpus evolves

### 3. Domain-Driven Thinking

**Important**: Every entity and relationship belongs to a domain with an owner:
- Helps organize large ontologies
- Clarifies data ownership
- Enables data mesh architecture

**Example**:
```
Domain: financial (Owner: CFO Office)
  ├─ revenue
  ├─ profit
  └─ EBITDA

Domain: legal (Owner: Legal Department)
  ├─ legal_entity
  ├─ jurisdiction
  └─ compliance_status
```

### 4. Confidence Represents Context Quality

**Remember**: Confidence is NOT about pattern matching certainty (patterns are binary: match or don't match).

Confidence represents **extraction context reliability**:
- **0.95**: Structured context (tables, forms) - highly predictable
- **0.85**: Semi-structured (lists, headings) - reliable
- **0.75**: Narrative context (paragraphs) - less predictable
- **0.65**: Unstructured/mixed - least reliable

**Example**:
```json
{
  "entity_type": "company",
  "description": "Company names from table cells",
  "element_types": ["table_cell"],
  "confidence": 0.95,
  ...
}
```
vs.
```json
{
  "entity_type": "company",
  "description": "Company names from paragraphs",
  "element_types": ["paragraph"],
  "confidence": 0.75,
  ...
}
```

### 5. Multiple Mappings for Same Entity Type

**Strategy**: Create multiple mappings for the same entity type with different confidence levels:

```json
[
  {
    "entity_type": "person",
    "description": "Person names from structured contacts",
    "element_types": ["table_cell"],
    "confidence": 0.95,
    "extraction_rules": [...]
  },
  {
    "entity_type": "person",
    "description": "Person names from narrative text",
    "element_types": ["paragraph"],
    "confidence": 0.75,
    "extraction_rules": [...]
  }
]
```

**Benefit**: When extraction runs, the highest confidence match wins.

### 6. Iterative Refinement Process

**Workflow**:
```
1. Create initial ontology (Workflow 1)
   ↓
2. Run extraction on test documents
   ↓
3. Review extraction results
   ↓
4. Identify issues (false positives, false negatives)
   ↓
5. Refine ontology (Workflow 2)
   ↓
6. Repeat from step 2 until satisfied
```

---

## Technical Details

### Schema Structure

```json
{
  "name": "MyOntology",
  "version": "1.0.0",
  "description": "Automatically generated ontology for financial/legal domains",
  "domains": [
    {
      "name": "financial",
      "description": "Financial performance and reporting",
      "owner": "CFO Office"
    }
  ],
  "element_entity_mappings": [
    {
      "entity_type": "company",
      "domain": "financial",
      "description": "Business organization",
      "element_types": ["paragraph", "table_cell"],
      "confidence": 0.85,
      "extraction_rules": [
        {
          "type": "keyword_match",
          "keywords": ["Microsoft", "Apple", "Google"]
        },
        {
          "type": "regex_pattern",
          "pattern": "\\b[A-Z][a-z]+ (Inc|Corp|LLC)\\b"
        }
      ]
    }
  ],
  "entity_relationship_rules": [
    {
      "name": "company_reports_revenue",
      "source_entity_type": "company",
      "target_entity_type": "revenue",
      "relationship_type": "related_to",
      "description": "Company reports revenue metric",
      "confidence": 0.90,
      "extraction_patterns": [
        {
          "type": "text_template",
          "template": "{company} reported {revenue}",
          "examples": ["Microsoft reported $50B revenue"]
        },
        {
          "type": "proximity",
          "signal_words": ["reported", "revenue", "earnings"],
          "max_distance": 10,
          "direction": "forward"
        }
      ]
    }
  ],
  "created_at": "2025-10-14T10:30:00Z"
}
```

### Extraction Pattern Types

#### For Entity Extraction

1. **keyword_match**: Exact keyword matching
   ```json
   {
     "type": "keyword_match",
     "keywords": ["Microsoft", "MSFT", "MS"]
   }
   ```

2. **regex_pattern**: Regular expression matching
   ```json
   {
     "type": "regex_pattern",
     "pattern": "\\b[A-Z]{2,5}\\b"
   }
   ```

3. **text_similarity**: Semantic similarity matching
   ```json
   {
     "type": "text_similarity",
     "reference_text": "the CEO stated that",
     "similarity_threshold": 0.7
   }
   ```

4. **metadata_field**: Extract from document metadata
   ```json
   {
     "type": "metadata_field",
     "field_path": "author.name"
   }
   ```

#### For Relationship Extraction

1. **text_template**: Pattern with entity placeholders
   ```json
   {
     "type": "text_template",
     "template": "{person} is CEO of {company}",
     "examples": ["John Smith is CEO of Acme Corp"]
   }
   ```

2. **proximity**: Entities near each other with signal words
   ```json
   {
     "type": "proximity",
     "signal_words": ["CEO", "president"],
     "max_distance": 10,
     "direction": "bidirectional"
   }
   ```

3. **regex**: Complex pattern with named groups
   ```json
   {
     "type": "regex",
     "pattern": "(?P<person>[A-Z][a-z]+ [A-Z][a-z]+), CEO of (?P<company>[A-Z][^,]+)"
   }
   ```

4. **cooccurrence**: Statistical co-occurrence
   ```json
   {
     "type": "cooccurrence",
     "context_window": "paragraph",
     "required_keywords": ["works", "employed"]
   }
   ```

---

## Troubleshooting

### Issue: LLM suggests irrelevant entities

**Solution**: Use the refinement workflow to:
1. Reject irrelevant suggestions in Phase 2
2. Or refine after creation and remove unwanted entities

### Issue: Extraction rules too broad (false positives)

**Solution**:
1. Use refinement workflow
2. LLM will identify broad patterns in Phase 1 (Analysis)
3. Approve rule fixes in Phase 2

### Issue: Missing important entities

**Solution**:
1. Use refinement workflow with existing schema
2. LLM will identify gaps in Phase 1
3. Approve additions in Phase 2

### Issue: Confidence levels seem wrong

**Solution**:
- Review context: Is this entity extracted from structured (tables) or unstructured (paragraphs) context?
- Use refinement to adjust confidence
- Remember: Confidence = context quality, not matching accuracy

---

## Next Steps

1. **Create your first ontology**:
   ```bash
   export ANTHROPIC_API_KEY=your_key
   ontology_interview ./corpus.parquet ./my-ontology.json
   ```

2. **Test extraction** (see extraction documentation)

3. **Refine based on results**:
   ```bash
   ontology_interview --refine ./my-ontology.json ./corpus.parquet ./v2-ontology.json
   ```

4. **Iterate** until extraction quality meets your needs

---

## Related Documentation

- [UDML Ontology System Overview](./UDML_ONTOLOGY_SYSTEM.md)
- [Domain Catalog Guide](../examples/ontologies/README.md)
- [Quick Start Guide](./ONTOLOGY_QUICK_START.md)
