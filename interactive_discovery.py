#!/usr/bin/env python3
"""
Interactive ontology discovery system with progressive understanding.
This system discovers the domain through analysis and user interaction,
rather than making assumptions.
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


class InteractiveOntologyDiscovery:
    """
    Progressive ontology discovery through corpus analysis and user interaction.
    """

    def __init__(self, analytics_path: str):
        self.analytics_path = analytics_path
        self.context = DiscoveryContext()
        self.corpus_profile = None

    def discover_ontology(self, candidate_terms: List[str]) -> Dict[str, Any]:
        """
        Main discovery flow with progressive understanding.
        """
        print("🚀 Interactive Ontology Discovery System")
        print("=" * 60)
        print("This system will analyze your corpus and work with you to")
        print("discover the appropriate ontology for your documents.")
        print("=" * 60)

        # Phase 1: Analyze corpus and hypothesize domain
        print("\n📊 PHASE 1: Corpus Analysis")
        print("-" * 40)
        self._analyze_and_hypothesize_domain()

        # Phase 2: Confirm domain with user
        print("\n🤝 PHASE 2: Domain Confirmation")
        print("-" * 40)
        self._confirm_domain_with_user()

        # Phase 3: Discover terms with domain context
        print("\n📚 PHASE 3: Term Discovery")
        print("-" * 40)
        self._discover_terms_with_context(candidate_terms)

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
        Analyze corpus and hypothesize what domain it represents.
        """
        # Get corpus profile
        conn = duckdb.connect(':memory:')

        # Get element statistics
        stats = conn.execute(f"""
            SELECT
                element_type,
                COUNT(*) as count,
                COUNT(DISTINCT doc_id) as doc_count
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            GROUP BY element_type
            ORDER BY count DESC
            LIMIT 10
        """).df()

        # Get distinctive patterns
        patterns = conn.execute(f"""
            SELECT
                content_preview,
                COUNT(*) as frequency
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            WHERE content_preview IS NOT NULL
            AND LENGTH(content_preview) > 10
            GROUP BY content_preview
            HAVING COUNT(*) > 10
            ORDER BY frequency DESC
            LIMIT 30
        """).df()

        # Get unique XML tags if present
        xml_tags = conn.execute(f"""
            SELECT DISTINCT content_preview
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            WHERE element_type = 'xml_element'
            AND content_preview LIKE '<%>'
            LIMIT 20
        """).fetchall()

        conn.close()

        self.corpus_profile = {
            'stats': stats.to_dict('records'),
            'patterns': patterns.to_dict('records'),
            'xml_tags': [tag[0] for tag in xml_tags] if xml_tags else []
        }

        # Use LLM to hypothesize domain
        prompt = f"""Analyze this corpus profile to determine what type of documents it likely contains.
Do not assume any particular domain - discover it from the evidence.

Element Type Distribution:
{json.dumps(self.corpus_profile['stats'], indent=2)}

Most Frequent Content Patterns:
{json.dumps(self.corpus_profile['patterns'][:15], indent=2)}

XML Tags Found (if any):
{json.dumps(self.corpus_profile['xml_tags'], indent=2)}

Based on this evidence:
1. What domain/industry do these documents likely represent?
2. What types of documents are these?
3. What evidence supports your hypothesis?
4. How confident are you (0-1)?

Return as JSON: {{
  "domain": "hypothesized domain",
  "document_types": ["list", "of", "types"],
  "evidence": ["specific patterns that indicate this domain"],
  "confidence": 0.85,
  "alternative_domains": ["other possibilities if unsure"]
}}"""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=1000,
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
        print(f"  • Evidence found:")
        for evidence in self.context.domain_evidence[:3]:
            print(f"    - {evidence}")

    def _confirm_domain_with_user(self):
        """
        Present hypothesis to user and get confirmation/corrections.
        """
        print(f"\nBased on my analysis, this corpus appears to contain:")
        print(f"  📁 {self.context.hypothesized_domain}")

        # In a real system, this would be interactive
        # For demo, we'll simulate user confirmation
        print("\n[Simulating user interaction]")
        print("User: Yes, these are SEC regulatory filings including")
        print("      10-K annual reports, 10-Q quarterly reports,")
        print("      8-K current reports, and Form 4 insider trading reports.")

        self.context.confirmed_domain = "SEC regulatory filings"
        self.context.user_context = """These are SEC regulatory filings including:
        - 10-K annual reports
        - 10-Q quarterly reports
        - 8-K current reports
        - Form 4 insider trading reports
        The corpus contains both HTML (human-readable) and XML (structured) versions."""

        print("\n✓ Domain confirmed and context noted")

    def _discover_terms_with_context(self, candidate_terms: List[str]):
        """
        Discover terms using confirmed domain context.
        """
        # Now we can use the domain context in our prompt!
        prompt = f"""You are analyzing a corpus of {self.context.confirmed_domain}.

User Context: {self.context.user_context}

The user has suggested these terms might be relevant:
{json.dumps(candidate_terms, indent=2)}

Based on the confirmed domain and these patterns from the corpus:
{json.dumps(self.corpus_profile['patterns'][:20], indent=2)}

Define formal ontology terms that are appropriate for {self.context.confirmed_domain}.
Include both the user's suggested terms (if they apply) and any additional critical domain terms.

For example, since these are SEC filings:
- XML tags like <issuerName> and <issuerTradingSymbol> are from Form 4
- "NAMED EXECUTIVE OFFICER COMPENSATION" is from proxy statements
- Financial terms like "Net income" are from financial statements

Return a JSON array of term definitions with id, label, description, and aliases."""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=1500,
            temperature=0.3,
            messages=[{"role": "user", "content": prompt}]
        )

        # Parse response
        content = response.content[0].text
        import re
        json_match = re.search(r'\[.*\]', content, re.DOTALL)
        if json_match:
            self.context.discovered_terms = json.loads(json_match.group())

        print(f"✓ Discovered {len(self.context.discovered_terms)} domain-specific terms")

        # Show terms to user for confirmation
        print("\nDiscovered terms:")
        for term in self.context.discovered_terms[:5]:
            print(f"  • {term['label']} - {term['description'][:60]}...")

    def _create_extraction_rules(self):
        """
        Create extraction rules with full context.
        """
        # Sample actual content for discovered terms
        conn = duckdb.connect(':memory:')
        term_samples = {}

        for term in self.context.discovered_terms[:6]:  # Limit for demo
            samples = conn.execute(f"""
                SELECT DISTINCT
                    content_preview,
                    element_type
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                WHERE content_preview ILIKE '%{term['label'].split()[0]}%'
                LIMIT 5
            """).fetchall()
            term_samples[term['id']] = [(s[0], s[1]) for s in samples]

        conn.close()

        prompt = f"""Create extraction rules for {self.context.confirmed_domain} terms.

Domain Context: {self.context.user_context}

Terms to Extract:
{json.dumps(self.context.discovered_terms[:6], indent=2)}

Actual Examples from Corpus:
{json.dumps(term_samples, indent=2)}

Understanding the domain helps create better rules:
- For SEC filings, company names often appear in specific XML tags or table headers
- Stock symbols are typically 1-5 uppercase letters
- Financial metrics appear in tables with specific formatting
- Dates follow regulatory format requirements

Generate extraction rules appropriate for {self.context.confirmed_domain}.
Return a JSON array of element mappings with term_id and rules."""

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
            self.context.extraction_rules = json.loads(json_match.group())

        print(f"✓ Created {len(self.context.extraction_rules)} extraction rule sets")

    def _discover_relationships(self):
        """
        Discover relationships specific to the confirmed domain.
        """
        prompt = f"""Discover relationships for {self.context.confirmed_domain}.

Domain Context: {self.context.user_context}

Terms in Ontology:
{json.dumps([{'id': t['id'], 'label': t['label']} for t in self.context.discovered_terms[:6]], indent=2)}

For {self.context.confirmed_domain}, typical relationships include:
- Companies file reports (company -> filing)
- Companies have trading symbols (company -> ticker)
- Executives work for companies (executive -> company)
- Reports contain financial metrics (report -> metrics)
- Insiders own securities (insider -> ownership)

Generate relationship rules appropriate for this specific domain.
Return a JSON array following the standard relationship structure."""

        response = client.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=1500,
            temperature=0.3,
            messages=[{"role": "user", "content": prompt}]
        )

        content = response.content[0].text
        import re
        json_match = re.search(r'\[.*\]', content, re.DOTALL)
        if json_match:
            self.context.relationships = json.loads(json_match.group())

        print(f"✓ Discovered {len(self.context.relationships)} domain-specific relationships")

    def _build_final_ontology(self) -> Dict[str, Any]:
        """
        Build the final ontology with all discovered components.
        """
        return {
            'domain': {
                'name': self.context.confirmed_domain.replace(' ', '_').lower(),
                'version': '1.0.0',
                'description': f'Ontology for {self.context.confirmed_domain}',
                'discovery_method': 'interactive_progressive',
                'user_context': self.context.user_context,
                'settings': {
                    'default_confidence_threshold': 0.70,
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
                'user_corrections': self.context.user_corrections
            }
        }


def main():
    """
    Demonstrate interactive discovery process.
    """
    # Configuration
    analytics_path = '/Volumes/T9/sec_analytics'

    # User provides candidate terms without assuming domain
    candidate_terms = [
        'company',        # Could be any type of organization
        'ticker',         # Might or might not apply
        'revenue',        # Could be financial or other
        'executive',      # Could be corporate or other
        'filing date',    # Generic date concept
        'form type'       # Generic document type
    ]

    # Create discovery system
    discovery = InteractiveOntologyDiscovery(analytics_path)

    # Execute interactive discovery
    ontology = discovery.discover_ontology(candidate_terms)

    # Save result
    import yaml
    output_file = 'interactive_discovered_ontology.yaml'
    with open(output_file, 'w') as f:
        yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)

    print("\n" + "=" * 60)
    print("✅ DISCOVERY COMPLETE")
    print("=" * 60)
    print(f"Domain: {ontology['domain']['name']}")
    print(f"Terms: {len(ontology['terms'])}")
    print(f"Rules: {len(ontology['element_mappings'])}")
    print(f"Relationships: {len(ontology['relationship_rules'])}")
    print(f"\nOntology saved to: {output_file}")


if __name__ == '__main__':
    main()