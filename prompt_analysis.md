# LLM Prompt Analysis and Improvement Suggestions

## Current Prompts Analysis

### Phase 1: Term Discovery
**Strengths:**
- Provides concrete corpus statistics showing document structure
- Includes actual content patterns from the data
- Lists user-provided candidate terms for context
- Clear output format specification

**Weaknesses:**
- Doesn't explain what the XML tags mean (e.g., `<footnoteId>`, `<directOrIndirectOwnership>`)
- Limited context about the domain (SEC filings)
- No guidance on how to interpret the patterns
- Doesn't suggest looking for related terms beyond the candidates

**Suggested Improvements:**
```
Add context: "This corpus contains SEC regulatory filings including Forms 10-K, 10-Q, 8-K, and insider trading forms (Form 4)."

Add interpretation hints: "The XML elements like <directOrIndirectOwnership> and <securityTitle> come from Form 4 insider trading reports. Elements like 'NAMED EXECUTIVE OFFICER COMPENSATION' are section headers from proxy statements."

Add discovery guidance: "In addition to defining the candidate terms, identify any other important domain terms you see in the patterns (e.g., 'Common Stock', 'Nasdaq', insider ownership terms)."
```

### Phase 2: Extraction Rule Creation
**Strengths:**
- Provides actual corpus samples for each term
- Shows element types where content was found
- Clear specification of rule structure

**Weaknesses:**
- Sample format is awkward: `"('Apple Inc.', 'paragraph')"` as strings
- No negative examples (what NOT to match)
- Doesn't provide context around the samples
- Missing guidance on confidence scoring

**Suggested Improvements:**
```
Better sample format:
"company": [
  {"content": "Apple Inc.", "element_type": "paragraph", "full_context": "Apple Inc. is a technology company..."},
  {"content": "MICROSOFT CORPORATION", "element_type": "table_cell", "full_context": "Row: Company Name | MICROSOFT CORPORATION"}
]

Add negative examples:
"not_company": [
  {"content": "Apple", "element_type": "paragraph", "note": "Product name, not company"},
  {"content": "company policy", "element_type": "paragraph", "note": "Generic use of word"}
]

Add confidence guidance: "Set confidence based on specificity: exact patterns (0.9-0.95), semantic matches (0.75-0.85), generic keywords (0.7-0.8)"
```

### Phase 3: Relationship Discovery
**Strengths:**
- Provides example relationships to model
- Clear structure specification
- Good use of semantic phrases

**Weaknesses:**
- No corpus evidence for relationships
- No co-occurrence statistics
- Doesn't explain hierarchy_level options
- Missing bidirectional relationship consideration

**Suggested Improvements:**
```
Add corpus evidence:
"Co-occurrence analysis shows:
- 'MICROSOFT CORP' appears with 'MSFT' in 289 documents
- Revenue figures appear near company names in 95% of financial tables
- Executive names co-occur with company names in proxy statements"

Explain constraints better:
"hierarchy_level options:
  -1: Same document anywhere
   0: Same parent element (e.g., same table)
   1: Same grandparent (e.g., same section)
   null: Cross-document relationships allowed"

Add relationship types: "Consider both directed (WORKS_FOR) and bidirectional (RELATED_TO) relationships"
```

## Improved Prompt Templates

### Enhanced Phase 1: Term Discovery with Domain Context
```python
prompt = f"""You are analyzing a corpus of SEC regulatory filings including annual reports (10-K),
quarterly reports (10-Q), current reports (8-K), proxy statements (DEF 14A), and insider trading forms (Form 4).

Corpus Profile:
- Total documents: {total_docs} spanning {unique_companies} companies
- Document types: HTML tables/paragraphs and XML structured data
- Element distribution: {element_stats}

Key Content Patterns Found:
{patterns_with_explanations}

Domain Context:
- XML tags like <issuerName>, <tradingSymbol> come from structured Form 4 filings
- Table cells contain financial data, company information, and executive compensation
- Headers like "NAMED EXECUTIVE OFFICER COMPENSATION" are from proxy statements

User-Identified Important Terms: {candidate_terms}

Task: Generate formal ontology term definitions for:
1. All user-identified candidate terms
2. Any additional critical domain terms you identify in the patterns

For each term, provide:
- id: snake_case identifier
- label: Human readable name
- description: Detailed explanation (2-3 sentences)
- aliases: List of synonyms and variations found in the data
- examples: 2-3 actual examples from the patterns (if found)

Return as JSON array. Include confidence score (0-1) for how well each term is supported by the data."""
```

### Enhanced Phase 2: Rule Creation with Context
```python
prompt = f"""Create extraction rules based on these term definitions and actual corpus evidence:

Terms to Extract:
{formatted_terms}

Actual Examples from Corpus (with surrounding context):
{formatted_samples_with_context}

Patterns to Avoid (false positives):
{negative_examples}

Rule Creation Guidelines:
- Regex patterns: Use for structured formats (dates, symbols, form numbers). Confidence: 0.85-0.95
- Semantic matching: Use for concepts with variation. Confidence: 0.75-0.85
- Keywords: Fallback for generic matching. Confidence: 0.70-0.80
- Element-specific: Target element_types where term typically appears

For each term, create 2-3 complementary rules that work together.
Include both precise rules (high confidence) and broad rules (lower confidence).

Output format:
[{
  "term_id": "...",
  "rules": [...],
  "expected_frequency": "common|moderate|rare",
  "validation_samples": ["example matches"]
}]"""
```

### Enhanced Phase 3: Evidence-Based Relationship Discovery
```python
prompt = f"""Discover relationships between entities based on corpus analysis:

Terms in Ontology:
{terms_with_descriptions}

Co-occurrence Evidence from Corpus:
{cooccurrence_statistics}

Document Structure Insights:
- Companies and tickers appear in same table rows {pct}% of the time
- Financial metrics follow company names in {pct}% of documents
- Executive names appear in dedicated sections near company info

Task: Generate relationship rules that reflect actual document patterns.

For each relationship:
1. Justify with corpus evidence
2. Set appropriate constraints based on document structure
3. Consider directionality (one-way vs bidirectional)
4. Set confidence based on co-occurrence strength

Include these relationship types:
- Ownership (HAS, OWNS, CONTAINS)
- Reporting (REPORTS, DISCLOSES, FILES)
- Association (WORKS_FOR, REPRESENTS, MANAGES)
- Temporal (DATED, FILED_ON, EFFECTIVE_DATE)

Output as JSON with corpus evidence noted in descriptions."""
```

## Key Improvements Summary

1. **Add Domain Context**: Explain what the data represents (SEC filings) and what patterns mean
2. **Provide Better Examples**: Use structured JSON with full context, not string tuples
3. **Include Negative Examples**: Show what NOT to match to improve precision
4. **Add Corpus Evidence**: Support decisions with statistics and co-occurrence data
5. **Explain Parameters**: Clarify what confidence scores, hierarchy levels, and other parameters mean
6. **Guide Discovery**: Encourage finding additional patterns beyond user-provided terms
7. **Validate with Data**: Ask LLM to provide example matches from the actual corpus
8. **Structure Output**: Request more structured, validatable output formats

These improvements would help the LLM:
- Better understand the domain and data
- Make evidence-based decisions
- Generate more precise extraction rules
- Discover relationships grounded in actual patterns
- Provide traceable, validatable ontologies