#!/usr/bin/env python3
"""
Generate ontology using Anthropic LLM with actual analytics data.
This is a focused implementation that properly uses the LLM.
"""

import os
import json
import duckdb
import anthropic
from typing import Dict, Any, List

# Initialize Anthropic client
client = anthropic.Anthropic()

def analyze_corpus(analytics_path: str) -> Dict[str, Any]:
    """Analyze the corpus to understand content."""
    conn = duckdb.connect(':memory:')

    # Get corpus statistics
    stats = conn.execute(f"""
        SELECT
            COUNT(DISTINCT doc_id) as total_docs,
            COUNT(*) as total_elements,
            element_type,
            COUNT(*) as count
        FROM read_parquet('{analytics_path}/elements/**/*.parquet')
        GROUP BY element_type
        ORDER BY count DESC
    """).df()

    # Get frequent patterns
    patterns = conn.execute(f"""
        SELECT
            content_preview,
            COUNT(*) as frequency
        FROM read_parquet('{analytics_path}/elements/**/*.parquet')
        WHERE content_preview IS NOT NULL
        AND LENGTH(content_preview) > 10
        GROUP BY content_preview
        HAVING COUNT(*) > 5
        ORDER BY frequency DESC
        LIMIT 50
    """).df()

    conn.close()

    return {
        'stats': stats.to_dict('records'),
        'patterns': patterns.to_dict('records')
    }

def discover_terms_for_domain(corpus_data: Dict, candidate_terms: List[str]) -> List[Dict]:
    """Use LLM to discover and define terms."""

    prompt = f"""Based on this corpus analysis of regulatory filing documents:

Element Statistics:
{json.dumps(corpus_data['stats'][:10], indent=2)}

Frequent Content Patterns:
{json.dumps(corpus_data['patterns'][:20], indent=2)}

The user has indicated these candidate terms are relevant to the domain:
{', '.join(candidate_terms)}

Generate formal ontology term definitions for these candidate terms.
Each term should have:
- id: snake_case identifier
- label: Human readable name
- description: What it represents
- aliases: List of alternative names

Return ONLY a JSON array of term definitions, no other text."""

    response = client.messages.create(
        model="claude-3-haiku-20240307",
        max_tokens=1000,
        temperature=0.3,
        messages=[{
            "role": "user",
            "content": prompt
        }]
    )

    # Parse response
    content = response.content[0].text
    # Extract JSON from response
    import re
    json_match = re.search(r'\[.*\]', content, re.DOTALL)
    if json_match:
        return json.loads(json_match.group())
    return json.loads(content)

def create_extraction_rules(terms: List[Dict], corpus_data: Dict) -> List[Dict]:
    """Generate extraction rules for each term."""

    # Sample actual content for each term
    conn = duckdb.connect(':memory:')
    term_samples = {}

    for term in terms:
        # Search for term variations in corpus
        term_name = term.get('label', term['id'])
        samples = conn.execute(f"""
            SELECT DISTINCT content_preview, element_type
            FROM read_parquet('/Volumes/T9/sec_analytics/elements/**/*.parquet')
            WHERE content_preview ILIKE '%{term_name.split()[0]}%'
            LIMIT 10
        """).fetchall()
        term_samples[term['id']] = samples

    conn.close()

    prompt = f"""Based on these term definitions and actual corpus samples:

Terms:
{json.dumps(terms, indent=2)}

Sample occurrences from the corpus:
{json.dumps({k: [str(v) for v in vals] for k, vals in term_samples.items()}, indent=2)}

Generate extraction rules for each term. Each should have:
- term_id: matching the term id
- rules: array of extraction rules, each with:
  - type: "regex", "semantic", or "keywords"
  - pattern/semantic_phrase/keywords: the matching pattern
  - confidence: confidence score (0.7-0.95)
  - element_types (optional): specific element types to match

Return ONLY a JSON array of element mappings, no other text."""

    response = client.messages.create(
        model="claude-3-haiku-20240307",
        max_tokens=1500,
        temperature=0.3,
        messages=[{
            "role": "user",
            "content": prompt
        }]
    )

    content = response.content[0].text
    import re
    json_match = re.search(r'\[.*\]', content, re.DOTALL)
    if json_match:
        return json.loads(json_match.group())
    return json.loads(content)

def discover_relationships(terms: List[Dict]) -> List[Dict]:
    """Discover relationships between terms."""

    prompt = f"""Based on these domain terms for regulatory filings:

{json.dumps(terms, indent=2)}

Generate relationship rules between these terms. Focus on natural relationships like:
- Companies have stock symbols
- Companies report financial metrics
- Executives work for companies
- Filings contain dates

Each relationship should follow this structure:
- id: unique identifier
- description: what the relationship represents
- relationship_type: type name (e.g., HAS_SYMBOL, REPORTS, WORKS_FOR)
- source: object with term_id, semantic_phrase, confidence_threshold
- target: object with term_id, semantic_phrase, confidence_threshold
- constraints: object with hierarchy_level (-1 for same doc), direction ("any")
- confidence: object with minimum (0.7-0.9), calculation ("average")

Return ONLY a JSON array of relationship rules, no other text."""

    response = client.messages.create(
        model="claude-3-haiku-20240307",
        max_tokens=1500,
        temperature=0.3,
        messages=[{
            "role": "user",
            "content": prompt
        }]
    )

    content = response.content[0].text
    import re
    json_match = re.search(r'\[.*\]', content, re.DOTALL)
    if json_match:
        return json.loads(json_match.group())
    return json.loads(content)

def main():
    """Generate ontology using LLM."""

    print("🚀 Generating Ontology with Anthropic LLM")
    print("=" * 60)

    # Configuration
    analytics_path = '/Volumes/T9/sec_analytics'
    domain_name = 'corporate_regulatory_filings'
    candidate_terms = [
        'company',
        'ticker',
        'revenue',
        'executive',
        'filing date',
        'form type'
    ]

    # Step 1: Analyze corpus
    print("\n📊 Step 1: Analyzing corpus...")
    corpus_data = analyze_corpus(analytics_path)
    print(f"  ✓ Analyzed {corpus_data['stats'][0]['total_docs']} documents")

    # Step 2: Discover terms
    print("\n📚 Step 2: Discovering terms with LLM...")
    terms = discover_terms_for_domain(corpus_data, candidate_terms)
    print(f"  ✓ Defined {len(terms)} terms")

    # Step 3: Create extraction rules
    print("\n🎯 Step 3: Creating extraction rules...")
    extraction_rules = create_extraction_rules(terms, corpus_data)
    print(f"  ✓ Created {len(extraction_rules)} extraction mappings")

    # Step 4: Discover relationships
    print("\n🔗 Step 4: Discovering relationships...")
    relationships = discover_relationships(terms)
    print(f"  ✓ Discovered {len(relationships)} relationships")

    # Build final ontology
    ontology = {
        'domain': {
            'name': domain_name,
            'version': '1.0.0',
            'description': f'LLM-generated ontology for {domain_name}',
            'settings': {
                'default_confidence_threshold': 0.70,
                'max_relationships_per_pair': 5,
                'enable_transitive_inference': True
            }
        },
        'terms': terms,
        'element_mappings': extraction_rules,
        'relationship_rules': relationships
    }

    # Save to file
    output_file = 'llm_generated_ontology.yaml'
    import yaml
    with open(output_file, 'w') as f:
        yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)

    print(f"\n✅ Ontology saved to: {output_file}")

    # Display summary
    print("\n" + "=" * 60)
    print("📋 ONTOLOGY SUMMARY")
    print("=" * 60)
    for term in terms:
        print(f"  • {term['label']} ({term['id']})")

    print(f"\nTotal: {len(terms)} terms, {len(extraction_rules)} rules, {len(relationships)} relationships")

if __name__ == '__main__':
    main()