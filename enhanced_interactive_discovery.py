#!/usr/bin/env python3
"""
Enhanced interactive ontology discovery system with improved prompts.
Incorporates all suggestions from prompt_analysis.md for better domain discovery.
"""

import os
import json
import duckdb
import anthropic
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass, field

# Initialize Anthropic client
client = anthropic.Anthropic()

@dataclass
class EnhancedDiscoveryContext:
    """Enhanced context with better corpus evidence tracking."""
    # Discovered through analysis
    hypothesized_domain: Optional[str] = None
    domain_confidence: float = 0.0
    domain_evidence: List[str] = field(default_factory=list)
    alternative_domains: List[str] = field(default_factory=list)

    # Confirmed through user interaction
    confirmed_domain: Optional[str] = None
    user_context: Optional[str] = None
    user_corrections: List[str] = field(default_factory=list)

    # Enhanced corpus understanding
    corpus_statistics: Dict[str, Any] = field(default_factory=dict)
    cooccurrence_data: Dict[str, Any] = field(default_factory=dict)
    negative_examples: Dict[str, List] = field(default_factory=dict)

    # Built progressively
    discovered_terms: List[Dict] = field(default_factory=list)
    extraction_rules: List[Dict] = field(default_factory=list)
    relationships: List[Dict] = field(default_factory=list)


class EnhancedInteractiveOntologyDiscovery:
    """
    Enhanced progressive ontology discovery with improved prompts and corpus analysis.
    """

    def __init__(self, analytics_path: str):
        self.analytics_path = analytics_path
        self.context = EnhancedDiscoveryContext()

    def discover_ontology(self, candidate_terms: List[str]) -> Dict[str, Any]:
        """
        Enhanced discovery flow with better prompts and evidence gathering.
        """
        print("🚀 Enhanced Interactive Ontology Discovery System")
        print("=" * 60)
        print("This system will analyze your corpus and work with you to")
        print("discover the appropriate ontology for your documents.")
        print("=" * 60)

        # Phase 1: Enhanced corpus analysis and domain hypothesis
        print("\n📊 PHASE 1: Enhanced Corpus Analysis")
        print("-" * 40)
        self._enhanced_corpus_analysis()

        # Phase 2: Domain confirmation with alternatives
        print("\n🤝 PHASE 2: Domain Confirmation")
        print("-" * 40)
        self._confirm_domain_with_alternatives()

        # Phase 3: Enhanced term discovery with domain context
        print("\n📚 PHASE 3: Enhanced Term Discovery")
        print("-" * 40)
        self._enhanced_term_discovery(candidate_terms)

        # Phase 4: Create extraction rules with context and examples
        print("\n🎯 PHASE 4: Enhanced Extraction Rules")
        print("-" * 40)
        self._enhanced_extraction_rules()

        # Phase 5: Evidence-based relationship discovery
        print("\n🔗 PHASE 5: Evidence-Based Relationship Discovery")
        print("-" * 40)
        self._enhanced_relationship_discovery()

        # Build final ontology
        return self._build_enhanced_ontology()

    def _enhanced_corpus_analysis(self):
        """
        Enhanced corpus analysis with better statistics and pattern interpretation.
        """
        conn = duckdb.connect(':memory:')

        # Get comprehensive corpus statistics
        stats = conn.execute(f"""
            SELECT
                COUNT(DISTINCT doc_id) as total_docs,
                COUNT(*) as total_elements,
                element_type,
                COUNT(*) as count,
                COUNT(DISTINCT doc_id) as doc_count,
                AVG(LENGTH(content_preview)) as avg_content_length
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            GROUP BY element_type
            ORDER BY count DESC
        """).df()

        # Get distinctive patterns with better context
        patterns = conn.execute(f"""
            SELECT
                content_preview,
                element_type,
                COUNT(*) as frequency,
                COUNT(DISTINCT doc_id) as document_spread,
                AVG(LENGTH(content_preview)) as avg_length
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            WHERE content_preview IS NOT NULL
            AND LENGTH(content_preview) > 10
            GROUP BY content_preview, element_type
            HAVING COUNT(*) > 10
            ORDER BY frequency DESC
            LIMIT 30
        """).df()

        # Get XML structure insights
        xml_analysis = conn.execute(f"""
            SELECT
                content_preview as xml_tag,
                COUNT(*) as frequency,
                COUNT(DISTINCT doc_id) as doc_coverage
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            WHERE element_type = 'xml_element'
            AND content_preview LIKE '<%>'
            AND LENGTH(content_preview) < 50
            GROUP BY content_preview
            ORDER BY frequency DESC
            LIMIT 20
        """).fetchall()

        # Analyze potential co-occurrences for relationship hints
        cooccurrence_sample = conn.execute(f"""
            SELECT DISTINCT
                doc_id,
                STRING_AGG(content_preview, ' | ') as content_sample
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            WHERE content_preview IS NOT NULL
            AND LENGTH(content_preview) BETWEEN 10 AND 100
            GROUP BY doc_id
            LIMIT 10
        """).fetchall()

        conn.close()

        # Store enhanced corpus data
        self.context.corpus_statistics = {
            'comprehensive_stats': stats.to_dict('records'),
            'patterns_with_context': patterns.to_dict('records'),
            'xml_structure': [{'tag': tag[0], 'frequency': tag[1], 'coverage': tag[2]} for tag in xml_analysis],
            'cooccurrence_samples': [{'doc_id': co[0], 'content': co[1]} for co in cooccurrence_sample]
        }

        # Enhanced LLM prompt with domain interpretation guidance
        prompt = f"""You are analyzing a document corpus to discover its domain. Analyze the evidence systematically.

CORPUS PROFILE:
================

Document Statistics:
{json.dumps(self.context.corpus_statistics['comprehensive_stats'][:10], indent=2)}

Content Patterns (with element context):
{json.dumps(self.context.corpus_statistics['patterns_with_context'][:15], indent=2)}

XML Structure Found:
{json.dumps(self.context.corpus_statistics['xml_structure'], indent=2)}

Sample Document Content Co-occurrences:
{json.dumps([s['content'][:200] + '...' for s in self.context.corpus_statistics['cooccurrence_samples'][:5]], indent=2)}

ANALYSIS TASK:
==============

Based on this evidence, determine what domain these documents represent. Look for:

1. **Regulatory Patterns**: XML tags like <issuerName>, <tradingSymbol> suggest regulatory filings
2. **Financial Patterns**: Terms like "revenue", "income", monetary amounts
3. **Corporate Patterns**: Company names, executive titles, organizational structure
4. **Legal Patterns**: Section headers, form numbers, compliance terminology
5. **Industry-Specific Terms**: Domain vocabularies (medical, financial, legal, etc.)

INTERPRETATION HINTS:
- XML elements with precise naming suggest structured data from formal systems
- Table structures with numerical data suggest systematic reporting
- Form numbers and codes suggest regulatory or compliance documentation
- Compensation headers suggest organizational management reporting

Provide your analysis as JSON:
{{
  "domain": "specific domain name",
  "document_types": ["list of document types identified"],
  "evidence": [
    "XML tag <issuerName> indicates SEC Form 4 insider trading reports",
    "Pattern 'NAMED EXECUTIVE OFFICER COMPENSATION' from proxy statements",
    "Financial tables with revenue/income data from annual reports"
  ],
  "confidence": 0.85,
  "alternative_domains": ["other possibilities if confidence < 0.9"],
  "key_indicators": {{
    "xml_tags": ["most significant XML elements"],
    "content_patterns": ["most distinctive content patterns"],
    "document_structure": "description of how documents are organized"
  }}
}}"""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=1500,
            temperature=0.2,
            messages=[{"role": "user", "content": prompt}]
        )

        # Parse enhanced response
        import re
        content = response.content[0].text
        json_match = re.search(r'\{.*\}', content, re.DOTALL)
        if json_match:
            hypothesis = json.loads(json_match.group())
        else:
            hypothesis = {"domain": "unknown", "confidence": 0}

        self.context.hypothesized_domain = hypothesis.get('domain', 'unknown')
        self.context.domain_confidence = hypothesis.get('confidence', 0)
        self.context.domain_evidence = hypothesis.get('evidence', [])
        self.context.alternative_domains = hypothesis.get('alternative_domains', [])

        print(f"Enhanced analysis complete!")
        print(f"  • Hypothesized domain: {self.context.hypothesized_domain}")
        print(f"  • Confidence: {self.context.domain_confidence:.0%}")
        print(f"  • Key evidence:")
        for evidence in self.context.domain_evidence[:3]:
            print(f"    - {evidence}")
        if self.context.alternative_domains:
            print(f"  • Alternative possibilities: {', '.join(self.context.alternative_domains)}")

    def _confirm_domain_with_alternatives(self):
        """
        Present hypothesis with alternatives and get user confirmation.
        """
        print(f"\nBased on enhanced analysis, this corpus appears to contain:")
        print(f"  📁 {self.context.hypothesized_domain} (confidence: {self.context.domain_confidence:.0%})")

        if self.context.alternative_domains:
            print(f"\nAlternative possibilities:")
            for alt in self.context.alternative_domains:
                print(f"  📄 {alt}")

        # Simulate enhanced user interaction
        print("\n[Enhanced simulation - user provides detailed context]")
        print("User: Yes, these are SEC regulatory filings. Specifically:")
        print("      • 10-K annual reports with comprehensive financial data")
        print("      • 10-Q quarterly reports with interim financials")
        print("      • 8-K current reports for material events")
        print("      • Form 4 insider trading reports (XML structured)")
        print("      • DEF 14A proxy statements with executive compensation")
        print("      The corpus contains both HTML (human-readable) and XML (machine-readable) versions.")

        self.context.confirmed_domain = "SEC regulatory filings"
        self.context.user_context = """SEC regulatory filings including:
        • 10-K annual reports with comprehensive financial statements
        • 10-Q quarterly reports with interim financial updates
        • 8-K current reports for material corporate events
        • Form 4 insider trading reports in structured XML format
        • DEF 14A proxy statements with executive compensation details
        The corpus contains both HTML (human-readable) and XML (structured) versions."""

        print("\n✓ Domain confirmed with detailed context")

    def _enhanced_term_discovery(self, candidate_terms: List[str]):
        """
        Enhanced term discovery with domain expertise and corpus evidence.
        """
        # Get actual examples for each candidate term
        conn = duckdb.connect(':memory:')
        term_evidence = {}

        for term in candidate_terms:
            examples = conn.execute(f"""
                SELECT DISTINCT
                    content_preview,
                    element_type,
                    COUNT(*) as frequency
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                WHERE content_preview ILIKE '%{term.split()[0]}%'
                AND LENGTH(content_preview) > 5
                GROUP BY content_preview, element_type
                ORDER BY frequency DESC
                LIMIT 8
            """).fetchall()

            term_evidence[term] = [
                {
                    "content": ex[0],
                    "element_type": ex[1],
                    "frequency": ex[2],
                    "confidence": min(0.95, 0.7 + (ex[2] / 100))
                }
                for ex in examples
            ]

        conn.close()

        # Domain-agnostic prompt that uses only user-provided context
        prompt = f"""You are analyzing documents to create ontology terms based on confirmed domain context.

USER-CONFIRMED DOMAIN:
{self.context.confirmed_domain}

USER-PROVIDED CONTEXT:
{self.context.user_context}

CANDIDATE TERMS PROVIDED BY USER:
{json.dumps(candidate_terms, indent=2)}

ACTUAL CORPUS EVIDENCE FOR EACH TERM:
{json.dumps(term_evidence, indent=2)}

TASK: Generate comprehensive term definitions using ONLY the user-provided context and corpus evidence.

For EACH candidate term provided:
1. Create a formal definition based on the user's domain context
2. Include ALL relevant aliases found in the corpus evidence
3. Identify additional important terms from the patterns (but only interpret them based on user context)

Do NOT assume domain knowledge beyond what the user has provided. Base all interpretations on:
- The user's explicit domain context
- Patterns observed in the actual corpus evidence
- Relationships suggested by the document structure

Return as JSON array with enhanced term definitions:
[{{
  "id": "snake_case_id",
  "label": "Human Readable Label",
  "description": "Detailed 2-3 sentence description explaining what this represents in SEC filings context",
  "aliases": ["all variations found in corpus"],
  "domain_context": "how this term specifically applies to SEC regulatory filings",
  "examples": ["2-3 actual examples from the corpus evidence"],
  "confidence": 0.85
}}]"""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=2000,
            temperature=0.3,
            messages=[{"role": "user", "content": prompt}]
        )

        # Parse response
        content = response.content[0].text
        import re
        json_match = re.search(r'\[.*\]', content, re.DOTALL)
        if json_match:
            self.context.discovered_terms = json.loads(json_match.group())

        print(f"✓ Discovered {len(self.context.discovered_terms)} enhanced domain-specific terms")
        print("\nSample discovered terms:")
        for term in self.context.discovered_terms[:4]:
            print(f"  • {term['label']}: {term['description'][:80]}...")

    def _enhanced_extraction_rules(self):
        """
        Create extraction rules with full context, examples, and negative patterns.
        """
        # Gather comprehensive examples with context
        conn = duckdb.connect(':memory:')
        enhanced_samples = {}
        negative_samples = {}

        for term in self.context.discovered_terms[:8]:  # Process key terms
            # Positive examples with surrounding context
            positive_examples = conn.execute(f"""
                SELECT DISTINCT
                    content_preview,
                    element_type,
                    COUNT(*) as frequency
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet') e
                WHERE content_preview ILIKE '%{term['label'].split()[0]}%'
                AND LENGTH(content_preview) BETWEEN 10 AND 200
                GROUP BY content_preview, element_type
                ORDER BY frequency DESC
                LIMIT 6
            """).fetchall()

            # Potential negative examples (common words that might false-positive)
            negative_examples = conn.execute(f"""
                SELECT DISTINCT content_preview, element_type
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                WHERE content_preview ILIKE '%{term['label'].split()[0]}%'
                AND (
                    content_preview ILIKE '%company policy%' OR
                    content_preview ILIKE '%the company%' OR
                    content_preview ILIKE '%company name%' OR
                    LENGTH(content_preview) < 8
                )
                LIMIT 3
            """).fetchall()

            enhanced_samples[term['id']] = [
                {
                    "content": ex[0],
                    "element_type": ex[1],
                    "frequency": ex[2],
                    "note": "Strong match - specific terminology"
                }
                for ex in positive_examples
            ]

            negative_samples[term['id']] = [
                {
                    "content": ex[0],
                    "element_type": ex[1],
                    "note": "Avoid generic usage"
                }
                for ex in negative_examples
            ]

        conn.close()

        # Domain-agnostic rule creation prompt
        prompt = f"""Create precise extraction rules based on user-provided domain context and corpus evidence.

USER-CONFIRMED DOMAIN: {self.context.confirmed_domain}
USER-PROVIDED CONTEXT: {self.context.user_context}

TERMS TO EXTRACT:
{json.dumps(self.context.discovered_terms[:8], indent=2)}

POSITIVE EXAMPLES WITH CONTEXT:
{json.dumps(enhanced_samples, indent=2)}

NEGATIVE EXAMPLES TO AVOID:
{json.dumps(negative_samples, indent=2)}

RULE CREATION GUIDELINES:
========================

Base confidence scoring on pattern specificity observed in corpus:
• Exact structured patterns (consistent XML tags, precise formats): 0.90-0.95
• Semantic patterns for concepts with clear corpus definition: 0.80-0.88
• Keyword matching with context constraints: 0.75-0.85
• Generic keyword fallbacks: 0.70-0.80

Analyze corpus patterns to determine:
• What structured data formats exist (XML tags, form patterns, etc.)
• Where different types of content typically appear (element types)
• What formatting conventions are used (dates, numbers, identifiers)
• What terminology variations exist in the data

Element type targeting based on observed corpus patterns:
• Use element types where terms actually appear in the examples
• Consider document structure patterns from the corpus
• Target specific contexts based on the user's domain description

Create 2-3 complementary rules per term:
1. High-precision rule (specific pattern, higher confidence)
2. Medium-precision rule (semantic matching with constraints)
3. Broad-coverage rule (keyword-based with element type filtering)

Return as JSON:
[{{
  "term_id": "...",
  "rules": [
    {{
      "type": "regex|semantic|keywords",
      "pattern": "for regex" OR "semantic_phrase": "for semantic" OR "keywords": ["for keywords"],
      "confidence": 0.85,
      "element_types": ["target_element_types"],
      "description": "what this rule matches"
    }}
  ],
  "expected_frequency": "common|moderate|rare",
  "validation_examples": ["examples this should match"]
}}]"""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=2500,
            temperature=0.3,
            messages=[{"role": "user", "content": prompt}]
        )

        content = response.content[0].text
        import re
        json_match = re.search(r'\[.*\]', content, re.DOTALL)
        if json_match:
            self.context.extraction_rules = json.loads(json_match.group())

        print(f"✓ Created {len(self.context.extraction_rules)} enhanced extraction rule sets")

    def _enhanced_relationship_discovery(self):
        """
        Discover relationships with corpus evidence and co-occurrence analysis.
        """
        # Analyze actual co-occurrences in the corpus
        conn = duckdb.connect(':memory:')

        # Get document-level co-occurrences between terms
        cooccurrence_analysis = conn.execute(f"""
            WITH term_mentions AS (
                SELECT DISTINCT
                    doc_id,
                    CASE
                        WHEN content_preview ILIKE '%company%' OR content_preview ILIKE '%corp%' THEN 'company'
                        WHEN content_preview ILIKE '%ticker%' OR content_preview ILIKE '%symbol%' THEN 'ticker'
                        WHEN content_preview ILIKE '%revenue%' OR content_preview ILIKE '%income%' THEN 'revenue'
                        WHEN content_preview ILIKE '%executive%' OR content_preview ILIKE '%officer%' THEN 'executive'
                        WHEN content_preview ILIKE '%filing%' OR content_preview ILIKE '%date%' THEN 'filing_date'
                        WHEN content_preview ILIKE '%form%' OR content_preview ILIKE '%10-K%' THEN 'form_type'
                    END as term_type
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                WHERE content_preview IS NOT NULL
            )
            SELECT
                a.term_type as term_a,
                b.term_type as term_b,
                COUNT(DISTINCT a.doc_id) as cooccurrence_count,
                COUNT(DISTINCT a.doc_id) * 100.0 / (SELECT COUNT(DISTINCT doc_id) FROM term_mentions) as percentage
            FROM term_mentions a
            JOIN term_mentions b ON a.doc_id = b.doc_id AND a.term_type != b.term_type
            WHERE a.term_type IS NOT NULL AND b.term_type IS NOT NULL
            GROUP BY a.term_type, b.term_type
            HAVING COUNT(DISTINCT a.doc_id) > 5
            ORDER BY cooccurrence_count DESC
        """).fetchall()

        conn.close()

        cooccurrence_data = [
            {
                "source": co[0],
                "target": co[1],
                "doc_count": co[2],
                "percentage": round(co[3], 1)
            }
            for co in cooccurrence_analysis
        ]

        self.context.cooccurrence_data = cooccurrence_data

        terms_for_prompt = [{'id': t['id'], 'label': t['label'], 'description': t['description']} for t in self.context.discovered_terms[:8]]

        prompt = f"""Discover relationships between entities based on corpus analysis and user-provided domain context.

USER-CONFIRMED DOMAIN: {self.context.confirmed_domain}
USER-PROVIDED CONTEXT: {self.context.user_context}

TERMS IN ONTOLOGY:
{json.dumps(terms_for_prompt, indent=2)}

CO-OCCURRENCE EVIDENCE FROM CORPUS:
{json.dumps(cooccurrence_data, indent=2)}

TASK: Generate relationship rules using ONLY:
1. Co-occurrence patterns observed in the corpus
2. Document structure insights from the data
3. User-provided domain context (above)

RELATIONSHIP CONSTRAINTS:
========================

hierarchy_level guidance (based on document structure):
• -1: Same document anywhere (for high-level associations)
• 0: Same parent element (e.g., same table or section)
• 1: Same grandparent (e.g., same major section)
• null: Cross-document relationships allowed

For each relationship, justify with:
1. Specific co-occurrence evidence from the corpus data
2. Document structure patterns observed in the corpus
3. How this fits the user's domain context (without assuming additional knowledge)

Generate relationship rules with corpus evidence:
[{{
  "id": "descriptive_relationship_id",
  "description": "What this relationship represents in SEC context with corpus evidence",
  "relationship_type": "HAS|OWNS|REPORTS|FILES|CONTAINS|WORKS_FOR|DATED",
  "source": {{
    "term_id": "source_term",
    "semantic_phrase": "how source appears in text",
    "confidence_threshold": 0.80
  }},
  "target": {{
    "term_id": "target_term",
    "semantic_phrase": "how target appears in text",
    "confidence_threshold": 0.80
  }},
  "constraints": {{
    "hierarchy_level": -1,
    "direction": "any"
  }},
  "confidence": {{
    "minimum": 0.80,
    "calculation": "average"
  }},
  "corpus_evidence": "X% of documents show these terms co-occurring",
  "domain_justification": "Why this relationship makes sense for the user's confirmed domain"
}}]

IMPORTANT: Return ONLY the JSON array, no explanatory text, no markdown formatting, no code blocks. Start your response with [ and end with ]."""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=2000,
            temperature=0.3,
            messages=[{"role": "user", "content": prompt}]
        )

        content = response.content[0].text
        import re
        json_match = re.search(r'\[.*\]', content, re.DOTALL)
        if json_match:
            try:
                self.context.relationships = json.loads(json_match.group())
                print(f"✓ Successfully parsed {len(self.context.relationships)} relationships")
            except json.JSONDecodeError as e:
                print(f"JSON parsing error: {e}")
                print(f"Extracted content: {json_match.group()[:200]}...")
                self.context.relationships = []
        else:
            print("No JSON array found in LLM response")
            self.context.relationships = []

        print(f"✓ Discovered {len(self.context.relationships)} evidence-based relationships")

    def _build_enhanced_ontology(self) -> Dict[str, Any]:
        """
        Build the final ontology with enhanced metadata and evidence.
        """
        return {
            'domain': {
                'name': self.context.confirmed_domain.replace(' ', '_').lower(),
                'version': '1.0.0',
                'description': f'Enhanced ontology for {self.context.confirmed_domain}',
                'discovery_method': 'enhanced_interactive_progressive',
                'user_context': self.context.user_context,
                'settings': {
                    'default_confidence_threshold': 0.75,
                    'max_relationships_per_pair': 5,
                    'enable_transitive_inference': True
                }
            },
            'terms': self.context.discovered_terms,
            'element_mappings': self.context.extraction_rules,
            'relationship_rules': self.context.relationships,
            'metadata': {
                'hypothesized_domain': self.context.hypothesized_domain,
                'domain_confidence': self.context.domain_confidence,
                'domain_evidence': self.context.domain_evidence,
                'alternative_domains': self.context.alternative_domains,
                'user_corrections': self.context.user_corrections,
                'corpus_statistics': self.context.corpus_statistics,
                'cooccurrence_evidence': self.context.cooccurrence_data
            }
        }


def main():
    """
    Demonstrate enhanced interactive discovery process.
    """
    # Configuration
    analytics_path = '/Volumes/T9/sec_analytics'

    # Candidate terms (user provides without assuming domain)
    candidate_terms = [
        'company',
        'ticker',
        'revenue',
        'executive',
        'filing date',
        'form type'
    ]

    # Create enhanced discovery system
    discovery = EnhancedInteractiveOntologyDiscovery(analytics_path)

    # Execute enhanced discovery
    ontology = discovery.discover_ontology(candidate_terms)

    # Save result
    import yaml
    output_file = 'enhanced_discovered_ontology.yaml'
    with open(output_file, 'w') as f:
        yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)

    print("\n" + "=" * 60)
    print("✅ ENHANCED DISCOVERY COMPLETE")
    print("=" * 60)
    print(f"Domain: {ontology['domain']['name']}")
    print(f"Terms: {len(ontology['terms'])}")
    print(f"Extraction Rules: {len(ontology['element_mappings'])}")
    print(f"Relationships: {len(ontology['relationship_rules'])}")
    print(f"Co-occurrence Evidence: {len(ontology['metadata']['cooccurrence_evidence'])} patterns")
    print(f"\nEnhanced ontology saved to: {output_file}")


if __name__ == '__main__':
    main()