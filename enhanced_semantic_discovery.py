#!/usr/bin/env python3
"""
Enhanced ontology discovery with explicit semantic similarity rule generation.
This version specifically encourages the LLM to create semantic rules using embeddings.
"""

import os
import json
import duckdb
import anthropic
import yaml
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass, field

# Initialize Anthropic client
client = anthropic.Anthropic()

@dataclass
class DiscoveryContext:
    """Maintains context throughout the discovery process."""
    # Discovered through analysis
    hypothesized_domain: Optional[str] = None
    domain_confidence: float = 0.0
    domain_evidence: List[str] = field(default_factory=list)

    # Confirmed through user interaction
    confirmed_domain: Optional[str] = None
    user_context: Optional[str] = None
    user_corrections: List[str] = field(default_factory=list)

    # Built progressively
    discovered_terms: List[Dict] = field(default_factory=list)
    extraction_rules: List[Dict] = field(default_factory=list)
    relationships: List[Dict] = field(default_factory=list)


class SemanticEnhancedOntologyDiscovery:
    """
    Ontology discovery with enhanced semantic similarity rule generation.
    """

    def __init__(self, analytics_path: str):
        self.analytics_path = analytics_path
        self.context = DiscoveryContext()
        self.corpus_profile = None

    def discover_ontology(self, candidate_terms: List[str]) -> Dict[str, Any]:
        """
        Main discovery flow with semantic enhancement.
        """
        print("🚀 Semantic-Enhanced Insider Trading Ontology Discovery")
        print("=" * 65)
        print("This system will discover domain terms AND generate semantic")
        print("similarity rules for robust entity extraction using embeddings.")
        print("=" * 65)

        # Phase 1: Analyze corpus and hypothesize domain
        print("\n📊 PHASE 1: Corpus Analysis")
        print("-" * 40)
        self._analyze_and_hypothesize_domain()

        # Phase 2: Enhanced user interview with insider trading context
        print("\n🤝 PHASE 2: Enhanced Domain Confirmation")
        print("-" * 40)
        self._enhanced_user_interview()

        # Phase 3: Discover terms with enhanced context
        print("\n📚 PHASE 3: Enhanced Term Discovery")
        print("-" * 40)
        self._discover_terms_with_enhanced_context(candidate_terms)

        # Phase 4: Create semantic-enhanced extraction rules
        print("\n🎯 PHASE 4: Semantic-Enhanced Extraction Rules")
        print("-" * 40)
        self._create_semantic_enhanced_rules()

        # Phase 5: Discover relationships
        print("\n🔗 PHASE 5: Relationship Discovery")
        print("-" * 40)
        self._discover_relationships()

        # Build final ontology
        return self._build_final_ontology()

    def _analyze_and_hypothesize_domain(self):
        """
        Analyze corpus and hypothesize domain.
        """
        # Get corpus profile
        conn = duckdb.connect(':memory:')

        # Enhanced element statistics
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

        # Enhanced XML tag analysis
        xml_analysis = conn.execute(f"""
            SELECT
                content_preview,
                element_type,
                COUNT(*) as frequency,
                COUNT(DISTINCT doc_id) as document_spread,
                AVG(LENGTH(content_preview)) as avg_length
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            WHERE element_type = 'xml_element'
            AND content_preview LIKE '<%>'
            GROUP BY content_preview, element_type
            ORDER BY frequency DESC
            LIMIT 30
        """).df()

        # Get distinctive content patterns
        patterns = conn.execute(f"""
            SELECT
                content_preview,
                element_type,
                COUNT(*) as frequency,
                COUNT(DISTINCT doc_id) as document_spread,
                AVG(LENGTH(content_preview)) as avg_length
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            WHERE content_preview IS NOT NULL
            AND LENGTH(content_preview) > 5
            GROUP BY content_preview, element_type
            HAVING COUNT(*) > 10
            ORDER BY frequency DESC
            LIMIT 50
        """).df()

        conn.close()

        self.corpus_profile = {
            'comprehensive_stats': stats.to_dict('records'),
            'xml_structure': xml_analysis.to_dict('records'),
            'patterns_with_context': patterns.to_dict('records')
        }

        # Enhanced LLM analysis
        prompt = f"""Analyze this document corpus to determine what type of documents it contains.
Do not assume any domain - discover it from the patterns and structure.

Comprehensive Element Statistics:
{json.dumps(self.corpus_profile['comprehensive_stats'], indent=2)}

XML Structure Analysis (Top Tags):
{json.dumps(self.corpus_profile['xml_structure'], indent=2)}

Content Patterns with Context:
{json.dumps(self.corpus_profile['patterns_with_context'][:20], indent=2)}

Based on this evidence, particularly the XML tags and structured content:
1. What domain/industry do these documents represent?
2. What specific types of regulatory documents are these?
3. What structured data formats are present?
4. What evidence supports your hypothesis?
5. How confident are you (0-1)?

Pay special attention to XML tags that suggest transaction data, ownership information, or financial reporting.

Return as JSON: {{
  "domain": "specific domain name",
  "document_types": ["specific", "document", "types"],
  "structured_formats": ["XML formats identified"],
  "evidence": ["specific XML tags and patterns that indicate this domain"],
  "confidence": 0.85,
  "key_entities_suggested": ["entities that this domain typically tracks"]
}}"""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=1500,
            temperature=0.3,
            messages=[{"role": "user", "content": prompt}]
        )

        # Parse response
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

        print(f"Analysis complete!")
        print(f"  • Hypothesized domain: {self.context.hypothesized_domain}")
        print(f"  • Confidence: {self.context.domain_confidence:.0%}")
        print(f"  • Key evidence:")
        for evidence in self.context.domain_evidence[:3]:
            print(f"    - {evidence}")

    def _enhanced_user_interview(self):
        """
        Enhanced user interview with specific insider trading context.
        """
        print(f"\nBased on my analysis, this corpus appears to contain:")
        print(f"  📁 {self.context.hypothesized_domain}")

        # Enhanced simulated user interaction with insider trading focus
        print("\n[Enhanced User Interview Simulation]")
        print("User: Yes, these are SEC regulatory filings, but I should be more specific.")
        print("      The corpus primarily contains insider trading reports, specifically:")
        print("      • Form 4 - Insider trading transaction reports")
        print("      • 10-K annual reports with financial data")
        print("      • 10-Q quarterly reports")
        print("      • 8-K current reports")
        print()
        print("      I'm particularly interested in tracking:")
        print("      • Insider transactions (buy/sell activities)")
        print("      • Share ownership changes")
        print("      • Transaction amounts and prices")
        print("      • Who the insiders are (executives, directors)")
        print("      • What companies they're trading")
        print("      • When transactions occurred")
        print()
        print("      The XML files contain structured insider transaction data")
        print("      while HTML files contain human-readable financial reports.")
        print()
        print("      IMPORTANT: I want semantic similarity matching for:")
        print("      • Executive titles (CEO, President, Chief Executive Officer)")
        print("      • Company name variations (MSFT, Microsoft, Microsoft Corp)")
        print("      • Transaction terms (buy, purchase, acquire, disposition)")

        self.context.confirmed_domain = "SEC insider trading and financial reporting"
        self.context.user_context = """SEC regulatory filings focused on insider trading analysis:

Primary Documents:
• Form 4 - Insider trading transaction reports (XML structured data)
• 10-K annual reports with comprehensive financial statements
• 10-Q quarterly reports with interim financial updates
• 8-K current reports for material corporate events

Key Analysis Focus:
• Insider transactions: buy/sell activities, transaction amounts, prices
• Share ownership: ownership changes, direct/indirect ownership
• Insider identification: executives, directors, officers, their roles
• Company identification: issuer names, ticker symbols, CIK numbers
• Temporal tracking: transaction dates, filing dates, reporting periods
• Financial metrics: revenue, earnings, financial performance data

Data Formats:
• XML: Structured transaction data from Form 4 filings
• HTML: Human-readable financial reports and statements

SEMANTIC MATCHING REQUIREMENTS:
• Executive titles: Need to match CEO ≈ Chief Executive Officer ≈ President ≈ Chief Operating Officer
• Company variations: Need to match Microsoft ≈ MSFT ≈ Microsoft Corporation ≈ Microsoft Corp
• Transaction types: Need to match buy ≈ purchase ≈ acquisition ≈ acquired ≈ disposition ≈ sold
• Financial terms: Need to match revenue ≈ sales ≈ income ≈ earnings ≈ profit

The goal is to track insider trading patterns with robust semantic matching that handles
natural language variations, abbreviations, and synonyms commonly found in SEC filings."""

        print("\n✓ Enhanced domain context established with semantic matching requirements")

    def _discover_terms_with_enhanced_context(self, candidate_terms: List[str]):
        """
        Discover terms with enhanced semantic context.
        """
        prompt = f"""You are analyzing a corpus of {self.context.confirmed_domain}.

Enhanced User Context with Semantic Requirements: {self.context.user_context}

The user has suggested these general terms:
{json.dumps(candidate_terms, indent=2)}

XML Structure and Patterns from Corpus:
{json.dumps(self.corpus_profile['xml_structure'][:15], indent=2)}

Content Patterns:
{json.dumps(self.corpus_profile['patterns_with_context'][:15], indent=2)}

Based on the confirmed insider trading focus and SEMANTIC MATCHING REQUIREMENTS,
define formal ontology terms for insider trading analysis.

CRITICAL: Each term should include SEMANTIC VARIATIONS that will be used for similarity matching:

Required term categories:
1. INSIDERS: People who trade (executives, directors, officers)
   - Must include title variations: CEO, Chief Executive Officer, President, etc.
2. TRANSACTIONS: Trading activities (buy, sell, amounts, prices, dates)
   - Must include action variations: buy, purchase, acquire, disposition, sold, etc.
3. OWNERSHIP: Share ownership details (direct/indirect, amounts)
4. COMPANIES: Issuers being traded (names, tickers, CIK numbers)
   - Must include name variations: Microsoft, MSFT, Microsoft Corp, etc.
5. FILINGS: Documents and dates (Form 4, filing dates, periods)
6. FINANCIAL: Revenue, earnings, financial metrics from reports
   - Must include metric variations: revenue, sales, income, earnings, profit

For each term, provide:
- id: snake_case identifier
- label: Human readable name
- description: Detailed explanation (2-3 sentences)
- aliases: List of variations found in XML tags and content
- semantic_variations: NEW FIELD - List of semantically similar terms for embedding matching
- domain_context: How this term relates to insider trading analysis

Focus on terms that will benefit from semantic similarity matching to handle
natural language variations in SEC filings.

Return ONLY a JSON array of term definitions."""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=2500,
            temperature=0.3,
            messages=[{"role": "user", "content": prompt}]
        )

        # Parse response
        content = response.content[0].text
        import re
        json_match = re.search(r'\[.*\]', content, re.DOTALL)
        if json_match:
            self.context.discovered_terms = json.loads(json_match.group())

        print(f"✓ Discovered {len(self.context.discovered_terms)} insider trading terms with semantic variations")

        # Show enhanced terms to user
        print("\nDiscovered terms with semantic variations:")
        for term in self.context.discovered_terms[:6]:
            semantic_vars = term.get('semantic_variations', [])[:3]
            print(f"  • {term['label']} (semantic: {', '.join(semantic_vars)})")

    def _create_semantic_enhanced_rules(self):
        """
        Create extraction rules with explicit semantic similarity rules.
        """
        # Sample actual content for enhanced rule creation
        conn = duckdb.connect(':memory:')
        term_samples = {}

        for term in self.context.discovered_terms[:8]:
            # Search for term-related content
            samples = conn.execute(f"""
                SELECT DISTINCT
                    content_preview,
                    element_type
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                WHERE content_preview ILIKE '%{term['label'].split()[0]}%'
                OR content_preview ILIKE '%{term['id']}%'
                LIMIT 8
            """).fetchall()
            term_samples[term['id']] = [(s[0], s[1]) for s in samples]

        conn.close()

        prompt = f"""Create SEMANTIC-ENHANCED extraction rules for insider trading analysis.

CRITICAL INSTRUCTION: You MUST create semantic similarity rules using the "semantic" rule type.
The Go-Doc-Go system supports these rule types:
- "regex": For exact pattern matching (XML tags, structured data)
- "semantic": For semantic similarity using embeddings (REQUIRED for natural language variations)
- "keywords": For exact keyword matching

Domain Context: {self.context.user_context}

Terms with Semantic Variations:
{json.dumps(self.context.discovered_terms[:8], indent=2)}

Actual Examples from SEC Corpus:
{json.dumps(term_samples, indent=2)}

XML Structure Patterns:
{json.dumps(self.corpus_profile['xml_structure'][:10], indent=2)}

RULE CREATION STRATEGY:

1. FOR STRUCTURED XML DATA (Form 4):
   - Use "regex" rules for exact XML tag matching
   - Pattern: <transactionShares>, <ownershipNature>, etc.
   - Target element_type: xml_element
   - High confidence: 0.85-0.95

2. FOR NATURAL LANGUAGE VARIATIONS (Critical - MUST implement):
   - Use "semantic" rules for synonyms and variations
   - semantic_phrase: Space-separated terms for embedding similarity
   - Include all semantic_variations from term definitions
   - Target element_type: paragraph, table_cell
   - Medium confidence: 0.70-0.80

3. FOR EXACT KEYWORD MATCHING:
   - Use "keywords" rules for specific terms
   - Target precise matches
   - Lower confidence: 0.65-0.75

SEMANTIC RULE EXAMPLES (REQUIRED):
- For "insider" term:
  semantic_phrase: "executive director officer CEO president chief executive officer chief operating officer chief financial officer"
- For "transaction" term:
  semantic_phrase: "buy purchase acquire acquisition disposition sold sale trading transaction"
- For "company" term:
  semantic_phrase: "company corporation issuer enterprise business firm organization"

Each rule should specify:
- type: "regex", "semantic", or "keywords"
- pattern (for regex), semantic_phrase (for semantic), or keywords (for keywords)
- confidence: appropriate confidence level
- element_types: where this data appears
- description: why this rule works for insider trading

IMPORTANT: EVERY term should have at least ONE semantic rule to handle natural language variations.

Return ONLY a JSON array of element mappings with term_id and rules."""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=3000,
            temperature=0.3,
            messages=[{"role": "user", "content": prompt}]
        )

        content = response.content[0].text
        import re
        json_match = re.search(r'\[.*\]', content, re.DOTALL)
        if json_match:
            self.context.extraction_rules = json.loads(json_match.group())

        # Count semantic rules
        semantic_rules = 0
        for mapping in self.context.extraction_rules:
            for rule in mapping.get('rules', []):
                if rule.get('type') == 'semantic':
                    semantic_rules += 1

        print(f"✓ Created {len(self.context.extraction_rules)} extraction rule sets")
        print(f"  Including {semantic_rules} semantic similarity rules for robust matching")

    def _discover_relationships(self):
        """
        Discover insider trading specific relationships.
        """
        prompt = f"""Discover relationships for insider trading analysis.

Domain Context: {self.context.user_context}

Terms in Ontology:
{json.dumps([{'id': t['id'], 'label': t['label']} for t in self.context.discovered_terms[:8]], indent=2)}

For insider trading analysis, key relationships include:
• INSIDER ACTIONS: insider → executes → transaction
• OWNERSHIP: insider → owns → shares
• CORPORATE STRUCTURE: insider → works_for → company
• TRANSACTIONS: transaction → involves → company
• TEMPORAL: transaction → occurred_on → date
• FINANCIAL: company → reports → financial_metrics

Focus on relationships that help analyze:
1. Who is trading (insider identification)
2. What they're trading (company stocks)
3. How much they're trading (transaction amounts)
4. When they're trading (dates and timing)
5. Their relationship to the company (roles)

Each relationship should specify:
- id: descriptive identifier
- relationship_type: action/connection type
- description: what this relationship means for analysis
- source/target: term_id and semantic_phrase
- confidence settings: minimum thresholds
- constraints: hierarchy_level for document structure

Return ONLY a JSON array of relationship rules optimized for insider trading patterns."""

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
            self.context.relationships = json.loads(json_match.group())

        print(f"✓ Discovered {len(self.context.relationships)} insider trading relationships")

    def _build_final_ontology(self) -> Dict[str, Any]:
        """
        Build enhanced ontology with semantic similarity rules.
        """
        return {
            'domain': {
                'name': 'sec_insider_trading_semantic_analysis',
                'version': '1.0.0',
                'description': 'Semantic-enhanced ontology for SEC insider trading analysis with robust similarity matching',
                'discovery_method': 'semantic_enhanced_interactive_insider_trading',
                'user_context': self.context.user_context,
                'settings': {
                    'default_confidence_threshold': 0.70,
                    'max_relationships_per_pair': 5,
                    'enable_transitive_inference': True,
                    'insider_trading_focus': True,
                    'semantic_similarity_enabled': True,
                    'embedding_model_suggested': 'sentence-transformers/all-MiniLM-L6-v2'
                }
            },
            'terms': self.context.discovered_terms,
            'element_mappings': self.context.extraction_rules,
            'relationship_rules': self.context.relationships,
            'metadata': {
                'hypothesized_domain': self.context.hypothesized_domain,
                'domain_confidence': self.context.domain_confidence,
                'domain_evidence': self.context.domain_evidence,
                'user_corrections': self.context.user_corrections,
                'corpus_statistics': self.corpus_profile,
                'semantic_enhancement': True,
                'embedding_requirements': {
                    'model_type': 'sentence_transformer',
                    'dimensions': 384,
                    'use_cases': ['executive_title_matching', 'company_name_variations', 'transaction_synonyms']
                }
            }
        }


def main():
    """
    Enhanced semantic insider trading ontology discovery.
    """
    # Configuration
    analytics_path = '/Volumes/T9/sec_analytics'

    # Enhanced candidate terms with insider trading hints
    candidate_terms = [
        'company',           # Issuer companies
        'insider',           # Corporate insiders
        'transaction',       # Trading transactions
        'shares',           # Share amounts
        'ownership',        # Ownership details
        'executive',        # Company executives
        'filing_date',      # When reported
        'transaction_date', # When traded
        'ticker',           # Stock symbols
        'revenue'           # Financial metrics
    ]

    # Create enhanced discovery system
    discovery = SemanticEnhancedOntologyDiscovery(analytics_path)

    # Execute enhanced discovery
    ontology = discovery.discover_ontology(candidate_terms)

    # Save result
    output_file = 'semantic_enhanced_insider_trading_ontology.yaml'
    with open(output_file, 'w') as f:
        yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)

    print("\n" + "=" * 65)
    print("✅ SEMANTIC-ENHANCED INSIDER TRADING DISCOVERY COMPLETE")
    print("=" * 65)
    print(f"Domain: {ontology['domain']['name']}")
    print(f"Terms: {len(ontology['terms'])}")
    print(f"Rules: {len(ontology['element_mappings'])}")
    print(f"Relationships: {len(ontology['relationship_rules'])}")
    print(f"Semantic Enhancement: {ontology['metadata']['semantic_enhancement']}")
    print(f"\nOntology saved to: {output_file}")

    # Count semantic rules
    semantic_rules = 0
    for mapping in ontology['element_mappings']:
        for rule in mapping.get('rules', []):
            if rule.get('type') == 'semantic':
                semantic_rules += 1

    print(f"\n🧠 SEMANTIC SIMILARITY FEATURES:")
    print(f"  • {semantic_rules} semantic similarity rules generated")
    print(f"  • Handles natural language variations and synonyms")
    print(f"  • Embedding model suggested: {ontology['domain']['settings']['embedding_model_suggested']}")

    # Show key terms with semantic variations
    print("\n🏷️  KEY TERMS WITH SEMANTIC MATCHING:")
    for term in ontology['terms'][:4]:
        semantic_vars = term.get('semantic_variations', [])[:3]
        if semantic_vars:
            print(f"  • {term['label']}: {', '.join(semantic_vars)}")


if __name__ == '__main__':
    main()