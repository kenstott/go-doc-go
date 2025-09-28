#!/usr/bin/env python3
"""
Enhanced interactive ontology discovery with specific insider trading context.
This simulates a more detailed user interview that mentions insider trading specifically.
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


class InsiderTradingOntologyDiscovery:
    """
    Enhanced ontology discovery with specific insider trading focus.
    """

    def __init__(self, analytics_path: str):
        self.analytics_path = analytics_path
        self.context = DiscoveryContext()
        self.corpus_profile = None

    def discover_ontology(self, candidate_terms: List[str]) -> Dict[str, Any]:
        """
        Main discovery flow with enhanced user interview.
        """
        print("🚀 Enhanced Insider Trading Ontology Discovery")
        print("=" * 60)
        print("This system will analyze your SEC insider trading corpus")
        print("and work with you to discover the appropriate ontology.")
        print("=" * 60)

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

        # Phase 4: Create extraction rules
        print("\n🎯 PHASE 4: Extraction Rules")
        print("-" * 40)
        self._create_extraction_rules()

        # Phase 5: Discover relationships
        print("\n🔗 PHASE 5: Relationship Discovery")
        print("-" * 40)
        self._discover_relationships()

        # Build final ontology
        return self._build_final_ontology()

    def _analyze_and_hypothesize_domain(self):
        """
        Analyze corpus with enhanced focus on XML structure.
        """
        # Get corpus profile with enhanced XML analysis
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

        # Enhanced XML tag analysis with better pattern recognition
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

        # Enhanced LLM analysis with XML focus
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

The goal is to track insider trading patterns, identify key corporate insiders,
monitor ownership changes, and correlate with company financial performance."""

        print("\n✓ Enhanced domain context established with insider trading focus")

    def _discover_terms_with_enhanced_context(self, candidate_terms: List[str]):
        """
        Discover terms with enhanced insider trading context.
        """
        prompt = f"""You are analyzing a corpus of {self.context.confirmed_domain}.

Enhanced User Context: {self.context.user_context}

The user has suggested these general terms, but we need to enhance them for insider trading analysis:
{json.dumps(candidate_terms, indent=2)}

XML Structure and Patterns from Corpus:
{json.dumps(self.corpus_profile['xml_structure'][:15], indent=2)}

Content Patterns:
{json.dumps(self.corpus_profile['patterns_with_context'][:15], indent=2)}

Based on the confirmed insider trading focus and XML patterns like <transactionShares>, <ownershipNature>, <transactionDate>, etc.,
define formal ontology terms for insider trading analysis.

Required term categories:
1. INSIDERS: People who trade (executives, directors, officers)
2. TRANSACTIONS: Trading activities (buy, sell, amounts, prices, dates)
3. OWNERSHIP: Share ownership details (direct/indirect, amounts)
4. COMPANIES: Issuers being traded (names, tickers, CIK numbers)
5. FILINGS: Documents and dates (Form 4, filing dates, periods)
6. FINANCIAL: Revenue, earnings, financial metrics from reports

For each term, provide:
- id: snake_case identifier
- label: Human readable name
- description: Detailed explanation (2-3 sentences)
- aliases: List of variations found in XML tags and content
- domain_context: How this term relates to insider trading analysis

Include both user-suggested terms (if applicable) and new insider trading specific terms.
Focus on terms that will help analyze trading patterns and insider behavior.

Return ONLY a JSON array of term definitions."""

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

        print(f"✓ Discovered {len(self.context.discovered_terms)} insider trading terms")

        # Show enhanced terms to user
        print("\nDiscovered insider trading terms:")
        for term in self.context.discovered_terms[:8]:
            print(f"  • {term['label']} - {term['description'][:70]}...")

    def _create_extraction_rules(self):
        """
        Create extraction rules optimized for insider trading data.
        """
        # Sample actual content for enhanced rule creation
        conn = duckdb.connect(':memory:')
        term_samples = {}

        for term in self.context.discovered_terms[:8]:  # Enhanced coverage
            # Search for term-related content in XML and text
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

        prompt = f"""Create enhanced extraction rules for insider trading analysis.

Domain Context: {self.context.user_context}

Terms to Extract (Insider Trading Focus):
{json.dumps(self.context.discovered_terms[:8], indent=2)}

Actual Examples from SEC Corpus:
{json.dumps(term_samples, indent=2)}

XML Structure Patterns:
{json.dumps(self.corpus_profile['xml_structure'][:10], indent=2)}

Create extraction rules optimized for insider trading data:

FOR XML ELEMENTS (Form 4 data):
- Use precise XML tag patterns: <transactionShares>, <ownershipNature>, etc.
- Target specific element types: xml_element, xml_object, xml_list
- High confidence (0.85-0.95) for structured XML matches

FOR FINANCIAL DATA (10-K/10-Q):
- Use financial table patterns for revenue, earnings
- Target table_cell, paragraph elements
- Medium confidence (0.75-0.85) for financial metrics

FOR TEXT CONTENT:
- Semantic matching for executive names, company names
- Keyword matching for roles, transaction types
- Lower confidence (0.70-0.80) for text extraction

Each rule should specify:
- type: "regex", "semantic", or "keywords"
- pattern/semantic_phrase/keywords: appropriate for the rule type
- confidence: based on specificity and domain
- element_types: specific to where this data appears
- description: why this rule works for insider trading analysis

Return ONLY a JSON array of element mappings with term_id and rules."""

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
        Build enhanced ontology for insider trading analysis.
        """
        return {
            'domain': {
                'name': 'sec_insider_trading_analysis',
                'version': '1.0.0',
                'description': 'Enhanced ontology for SEC insider trading and financial reporting analysis',
                'discovery_method': 'enhanced_interactive_insider_trading',
                'user_context': self.context.user_context,
                'settings': {
                    'default_confidence_threshold': 0.75,
                    'max_relationships_per_pair': 5,
                    'enable_transitive_inference': True,
                    'insider_trading_focus': True
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
                'corpus_statistics': self.corpus_profile
            }
        }


def main():
    """
    Enhanced insider trading ontology discovery.
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
    discovery = InsiderTradingOntologyDiscovery(analytics_path)

    # Execute enhanced discovery
    ontology = discovery.discover_ontology(candidate_terms)

    # Save result
    output_file = 'insider_trading_discovered_ontology.yaml'
    with open(output_file, 'w') as f:
        yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)

    print("\n" + "=" * 60)
    print("✅ ENHANCED INSIDER TRADING DISCOVERY COMPLETE")
    print("=" * 60)
    print(f"Domain: {ontology['domain']['name']}")
    print(f"Terms: {len(ontology['terms'])}")
    print(f"Rules: {len(ontology['element_mappings'])}")
    print(f"Relationships: {len(ontology['relationship_rules'])}")
    print(f"\nOntology saved to: {output_file}")

    # Show key terms discovered
    print("\n🏷️  KEY INSIDER TRADING TERMS:")
    for term in ontology['terms'][:6]:
        print(f"  • {term['label']}: {term['description'][:60]}...")


if __name__ == '__main__':
    main()