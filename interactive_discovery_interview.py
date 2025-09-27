#!/usr/bin/env python3
"""
Interactive Ontology Discovery Interview System
Integrates with Go-Doc-Go to provide conversational ontology creation.
"""

import json
import os
import sys
import yaml
from typing import Dict, List, Any, Optional, Tuple
from pathlib import Path
import anthropic
import duckdb
import pandas as pd
from collections import Counter, defaultdict
from itertools import combinations
import re

# Add Go-Doc-Go to path
sys.path.insert(0, 'src')

try:
    from go_doc_go.domain.ontology import DomainOntology
except ImportError:
    print("⚠️  Go-Doc-Go modules not available - running in standalone mode")


class InteractiveDiscoveryInterview:
    """Interactive interview system for ontology discovery."""

    def __init__(self, analytics_path: str, output_dir: str = "."):
        self.analytics_path = analytics_path
        self.output_dir = Path(output_dir)
        self.client = anthropic.Anthropic() if 'ANTHROPIC_API_KEY' in os.environ else None

        # Interview state
        self.corpus_patterns = {}
        self.user_interests = []
        self.conversation_history = []
        self.generated_ontology = {}

    def start_interview(self) -> str:
        """Start the interactive discovery process."""
        print("🎤 GO-DOC-GO INTERACTIVE ONTOLOGY DISCOVERY")
        print("=" * 60)
        print("I'll help you create an ontology by analyzing your data and")
        print("understanding what you want to track.\n")

        # Step 1: Analyze corpus
        if not self._analyze_corpus():
            return "❌ Could not analyze corpus data"

        # Step 2: Present findings to user
        self._present_corpus_analysis()

        # Step 3: Conduct interactive interview
        self._conduct_interview()

        # Step 4: Generate ontology
        ontology_path = self._generate_final_ontology()

        print(f"\n✅ Interview complete! Ontology saved to: {ontology_path}")
        return ontology_path

    def _analyze_corpus(self) -> bool:
        """Analyze the corpus to understand data patterns."""
        print("🔍 Analyzing your document corpus...")

        try:
            # Load analytics data
            elements_df = self._load_analytics_data()
            if elements_df.empty:
                print("❌ No analytics data found")
                return False

            # Analyze patterns
            self.corpus_patterns = {
                'frequent_terms': self._find_frequent_terms(elements_df),
                'element_types': self._analyze_element_types(elements_df),
                'co_occurrences': self._find_co_occurrences(elements_df),
                'document_stats': self._get_document_stats(elements_df)
            }

            print(f"✅ Analyzed {self.corpus_patterns['document_stats']['total_docs']} documents")
            return True

        except Exception as e:
            print(f"❌ Error analyzing corpus: {e}")
            return False

    def _load_analytics_data(self) -> pd.DataFrame:
        """Load analytics data from parquet files."""
        conn = duckdb.connect(':memory:')

        # Find parquet files
        import glob
        parquet_files = []
        for file_path in glob.glob(f"{self.analytics_path}/elements/**/*.parquet", recursive=True):
            if not os.path.basename(file_path).startswith('._'):
                parquet_files.append(file_path)

        if not parquet_files:
            return pd.DataFrame()

        # Load data
        file_list = "', '".join(parquet_files)
        query = f"""
        SELECT
            doc_id,
            element_id,
            element_type,
            content_preview as content
        FROM read_parquet(['{file_list}'])
        WHERE content_preview IS NOT NULL
        AND content_preview != ''
        LIMIT 10000
        """

        return conn.execute(query).df()

    def _find_frequent_terms(self, df: pd.DataFrame) -> Dict[str, int]:
        """Find most frequent terms in the corpus."""
        all_content = ' '.join(df['content'].astype(str))

        # Extract words
        stop_words = {'the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by'}
        words = re.findall(r'\b[a-zA-Z]{3,}\b', all_content.lower())
        word_freq = Counter([w for w in words if w not in stop_words])

        return dict(word_freq.most_common(30))

    def _analyze_element_types(self, df: pd.DataFrame) -> Dict[str, int]:
        """Analyze element type distribution."""
        return dict(df['element_type'].value_counts())

    def _find_co_occurrences(self, df: pd.DataFrame) -> Dict[str, List[str]]:
        """Find terms that co-occur in documents."""
        frequent_terms = list(self._find_frequent_terms(df).keys())[:20]
        co_matrix = defaultdict(Counter)

        # Build co-occurrence by document
        for doc_id in df['doc_id'].unique():
            doc_content = ' '.join(df[df['doc_id'] == doc_id]['content'].astype(str)).lower()
            doc_words = set(re.findall(r'\b[a-zA-Z]{3,}\b', doc_content))

            # Count co-occurrences
            for word_pair in combinations([w for w in doc_words if w in frequent_terms], 2):
                co_matrix[word_pair[0]][word_pair[1]] += 1
                co_matrix[word_pair[1]][word_pair[0]] += 1

        # Convert to simple format
        result = {}
        for word, related in co_matrix.items():
            result[word] = list(related.most_common(5))

        return result

    def _get_document_stats(self, df: pd.DataFrame) -> Dict[str, Any]:
        """Get basic document statistics."""
        return {
            'total_docs': int(df['doc_id'].nunique()),
            'total_elements': int(len(df)),
            'avg_content_length': float(df['content'].str.len().mean()),
            'element_type_counts': {k: int(v) for k, v in df['element_type'].value_counts().items()}
        }

    def _present_corpus_analysis(self):
        """Present corpus analysis to user."""
        print("\n📊 CORPUS ANALYSIS RESULTS")
        print("-" * 40)

        # Show key statistics
        stats = self.corpus_patterns['document_stats']
        print(f"Documents analyzed: {stats['total_docs']}")
        print(f"Content elements: {stats['total_elements']}")

        # Show frequent terms
        print(f"\nMost frequent terms:")
        frequent = self.corpus_patterns['frequent_terms']
        for i, (term, count) in enumerate(list(frequent.items())[:10]):
            print(f"  {i+1}. {term} ({count} times)")

        # Show element types
        print(f"\nElement types found:")
        for elem_type, count in self.corpus_patterns['element_types'].items():
            print(f"  • {elem_type}: {count}")

        # Show co-occurrences
        print(f"\nTerms that appear together:")
        co_occur = self.corpus_patterns['co_occurrences']
        for word, related in list(co_occur.items())[:3]:
            related_words = [w for w, c in related[:3]]
            print(f"  • '{word}' often with: {related_words}")

        print()

    def _conduct_interview(self):
        """Conduct the interactive interview."""
        print("🎤 INTERACTIVE INTERVIEW")
        print("-" * 30)

        # Question 1: Domain and purpose
        domain_response = self._ask_question(
            "What domain or industry is this data from? What are you trying to analyze?",
            "domain_context"
        )

        # Question 2: Specific interests
        interests_response = self._ask_question(
            "Looking at the frequent terms I found, what specific concepts would you like to track? " +
            "For example, are you interested in people, events, organizations, or something else?",
            "specific_interests"
        )

        # Question 3: Relationships
        relationships_response = self._ask_question(
            "What relationships between these concepts matter to you? " +
            "For example: 'who did what', 'what belongs to whom', or 'what happened when'?",
            "relationships"
        )

        # Question 4: Variations and edge cases
        variations_response = self._ask_question(
            "What different ways might these concepts be expressed in your documents? " +
            "Are there synonyms, abbreviations, or alternative phrasings I should watch for?",
            "variations"
        )

        # Use LLM to enhance responses if available
        if self.client:
            self._enhance_with_llm()

    def _ask_question(self, question: str, category: str) -> str:
        """Ask a question and record the response."""
        print(f"\nQ: {question}")
        response = input("A: ").strip()

        # Record in conversation history
        self.conversation_history.append({
            'question': question,
            'response': response,
            'category': category
        })

        return response

    def _enhance_with_llm(self):
        """Use LLM to enhance user responses with domain knowledge."""
        if not self.client:
            return

        print("\n🤖 Analyzing your responses with AI assistance...")

        # Prepare conversation context
        conversation_text = ""
        for entry in self.conversation_history:
            conversation_text += f"Q: {entry['question']}\nA: {entry['response']}\n\n"

        # Prepare corpus context
        corpus_summary = f"""
        CORPUS PATTERNS:
        Frequent terms: {list(self.corpus_patterns['frequent_terms'].keys())[:15]}
        Element types: {list(self.corpus_patterns['element_types'].keys())}
        """

        # Ask LLM for enhancement
        prompt = f"""Based on this conversation with a domain expert and the corpus patterns, suggest an enhanced ontology structure.

CONVERSATION:
{conversation_text}

{corpus_summary}

Please suggest:
1. Refined concept definitions based on user interests
2. Semantic variations for each concept (synonyms, abbreviations, related terms)
3. Relationship types that match what the user described
4. Confidence levels for different rule types

Format as JSON with: concepts, semantic_variations, relationships, suggested_rules"""

        try:
            response = self.client.messages.create(
                model="claude-3-5-sonnet-20241022",
                max_tokens=1500,
                messages=[{"role": "user", "content": prompt}]
            )

            llm_suggestions = response.content[0].text
            print("✅ AI analysis complete")

            # Try to parse JSON suggestions
            try:
                self.llm_suggestions = json.loads(llm_suggestions)
            except:
                self.llm_suggestions = {"raw_response": llm_suggestions}

        except Exception as e:
            print(f"⚠️  AI analysis failed: {e}")
            self.llm_suggestions = {}

    def _generate_final_ontology(self) -> str:
        """Generate the final ontology YAML file."""
        print("\n🏗️  Generating ontology...")

        # Build ontology structure
        ontology = {
            'domain': {
                'name': self._generate_domain_name(),
                'version': '1.0.0',
                'description': self._generate_domain_description(),
                'discovery_method': 'interactive_interview',
                'settings': {
                    'default_confidence_threshold': 0.70,
                    'max_relationships_per_pair': 5,
                    'enable_transitive_inference': True,
                    'interview_driven': True
                }
            },
            'terms': self._generate_terms(),
            'element_mappings': self._generate_element_mappings(),
            'relationship_rules': self._generate_relationship_rules(),
            'metadata': self._generate_metadata()
        }

        # Save to file (convert numpy types to native Python first)
        output_path = self.output_dir / f"{ontology['domain']['name']}_ontology.yaml"
        clean_ontology = self._numpy_to_python(ontology)
        with open(output_path, 'w') as f:
            yaml.dump(clean_ontology, f, default_flow_style=False, sort_keys=False)

        # Also save conversation log
        log_path = self.output_dir / f"{ontology['domain']['name']}_interview_log.json"
        with open(log_path, 'w') as f:
            json.dump({
                'conversation_history': self.conversation_history,
                'corpus_patterns': self.corpus_patterns,
                'llm_suggestions': getattr(self, 'llm_suggestions', {}),
                'final_ontology': ontology
            }, f, indent=2, default=str)

        return str(output_path)

    def _numpy_to_python(self, data):
        """Convert numpy types to native Python types for YAML serialization."""
        import numpy as np
        if isinstance(data, np.integer):
            return int(data)
        elif isinstance(data, np.floating):
            return float(data)
        elif isinstance(data, np.ndarray):
            return data.tolist()
        elif isinstance(data, dict):
            return {k: self._numpy_to_python(v) for k, v in data.items()}
        elif isinstance(data, (list, tuple)):
            return [self._numpy_to_python(item) for item in data]
        elif isinstance(data, (int, float, str, bool)):
            return data  # Keep native Python types as-is
        else:
            return str(data)

    def _generate_domain_name(self) -> str:
        """Generate domain name from conversation."""
        # Look for domain context in responses
        domain_responses = [entry['response'] for entry in self.conversation_history
                          if entry['category'] == 'domain_context']

        if domain_responses:
            # Simple name generation - could be enhanced with LLM
            response = domain_responses[0].lower()
            if 'sec' in response or 'filing' in response:
                return 'sec_filing_analysis'
            elif 'financial' in response:
                return 'financial_analysis'
            elif 'legal' in response:
                return 'legal_document_analysis'
            elif 'medical' in response:
                return 'medical_analysis'
            else:
                return 'custom_domain_analysis'

        return 'discovered_domain'

    def _generate_domain_description(self) -> str:
        """Generate domain description from conversation."""
        domain_responses = [entry['response'] for entry in self.conversation_history
                          if entry['category'] == 'domain_context']

        if domain_responses:
            return f"Ontology for analyzing {domain_responses[0]} - created through interactive discovery"

        return "Ontology created through interactive discovery process"

    def _generate_terms(self) -> List[Dict[str, Any]]:
        """Generate terms from conversation and corpus analysis."""
        terms = []

        # Extract concepts from user interests
        interest_responses = [entry['response'] for entry in self.conversation_history
                            if entry['category'] == 'specific_interests']

        # Simple concept extraction - could be enhanced with NLP
        if interest_responses:
            response = interest_responses[0].lower()

            # Common concept patterns
            if 'people' in response or 'person' in response or 'executive' in response:
                terms.append({
                    'id': 'person',
                    'label': 'Person',
                    'description': 'Individuals mentioned in the documents',
                    'aliases': [],
                    'semantic_variations': self._get_person_variations()
                })

            if 'organization' in response or 'company' in response or 'business' in response:
                terms.append({
                    'id': 'organization',
                    'label': 'Organization',
                    'description': 'Companies, institutions, or other organizations',
                    'aliases': [],
                    'semantic_variations': self._get_organization_variations()
                })

            if 'event' in response or 'transaction' in response or 'activity' in response:
                terms.append({
                    'id': 'event',
                    'label': 'Event',
                    'description': 'Activities, transactions, or events described in documents',
                    'aliases': [],
                    'semantic_variations': self._get_event_variations()
                })

        # If no terms from interests, generate from corpus patterns
        if not terms:
            frequent_terms = list(self.corpus_patterns['frequent_terms'].keys())[:5]
            for term in frequent_terms:
                terms.append({
                    'id': term.lower(),
                    'label': term.title(),
                    'description': f'Concept related to {term}',
                    'aliases': [],
                    'semantic_variations': [term]
                })

        return terms

    def _get_person_variations(self) -> List[str]:
        """Get semantic variations for person concept."""
        # Check frequent terms for person-related words
        frequent = self.corpus_patterns['frequent_terms']
        person_words = []

        person_patterns = ['officer', 'director', 'ceo', 'president', 'executive', 'manager', 'employee']
        for word in frequent.keys():
            if any(pattern in word.lower() for pattern in person_patterns):
                person_words.append(word)

        # Add common variations
        person_words.extend(['CEO', 'CFO', 'President', 'Director', 'Officer', 'Executive'])

        return list(set(person_words))

    def _get_organization_variations(self) -> List[str]:
        """Get semantic variations for organization concept."""
        frequent = self.corpus_patterns['frequent_terms']
        org_words = []

        org_patterns = ['company', 'corporation', 'inc', 'llc', 'ltd', 'corp', 'organization', 'registrant']
        for word in frequent.keys():
            if any(pattern in word.lower() for pattern in org_patterns):
                org_words.append(word)

        org_words.extend(['Company', 'Corporation', 'Organization', 'Registrant', 'Issuer'])

        return list(set(org_words))

    def _get_event_variations(self) -> List[str]:
        """Get semantic variations for event concept."""
        frequent = self.corpus_patterns['frequent_terms']
        event_words = []

        event_patterns = ['transaction', 'sale', 'purchase', 'agreement', 'contract', 'deal', 'acquisition']
        for word in frequent.keys():
            if any(pattern in word.lower() for pattern in event_patterns):
                event_words.append(word)

        event_words.extend(['Transaction', 'Purchase', 'Sale', 'Agreement', 'Deal'])

        return list(set(event_words))

    def _generate_element_mappings(self) -> List[Dict[str, Any]]:
        """Generate element mappings for each term."""
        mappings = []

        for term in self._generate_terms():
            mapping = {
                'term_id': term['id'],
                'rules': []
            }

            # Add semantic rule using proper Go-Doc-Go format
            if term['semantic_variations']:
                # Use the most frequent term from corpus that matches this concept
                # Find the highest frequency variation from corpus analysis
                best_semantic_phrase = term['label'].lower()  # default fallback

                if hasattr(self, 'corpus_patterns') and 'frequent_terms' in self.corpus_patterns:
                    # Find the most frequent variation that appears in corpus
                    max_frequency = 0
                    for variation in term['semantic_variations']:
                        variation_lower = variation.lower()
                        if variation_lower in self.corpus_patterns['frequent_terms']:
                            freq = self.corpus_patterns['frequent_terms'][variation_lower]
                            if freq > max_frequency:
                                max_frequency = freq
                                best_semantic_phrase = variation_lower

                semantic_phrase = best_semantic_phrase

                semantic_rule = {
                    'type': 'semantic',
                    'semantic_phrase': semantic_phrase,
                    'confidence_threshold': 0.70,  # Correct field name for Go-Doc-Go
                    'element_types': ['paragraph', 'table_cell'],
                    'description': f"Semantic similarity matching for {term['label']} using embeddings"
                }
                mapping['rules'].append(semantic_rule)

            # Add regex rule for exact pattern matching
            if term['semantic_variations']:
                # Create regex from variations (case-insensitive)
                variations = [re.escape(v) for v in term['semantic_variations'][:5]]
                pattern = '\\b(' + '|'.join(variations) + ')\\b'

                regex_rule = {
                    'type': 'regex',
                    'pattern': pattern,
                    'element_types': ['paragraph', 'table_cell', 'table_row'],
                    'description': f"Exact pattern matching for {term['label']} concept"
                }
                mapping['rules'].append(regex_rule)

            mappings.append(mapping)

        return mappings

    def _generate_relationship_rules(self) -> List[Dict[str, Any]]:
        """Generate relationship rules from conversation."""
        relationships = []

        # Look for relationship responses
        rel_responses = [entry['response'] for entry in self.conversation_history
                        if entry['category'] == 'relationships']

        terms = self._generate_terms()
        if len(terms) >= 2 and rel_responses:
            # Generate basic relationships between first two terms
            source_term = terms[0]
            target_term = terms[1]

            relationships.append({
                'id': f"{source_term['id']}_relates_to_{target_term['id']}",
                'relationship_type': 'association',
                'description': f"{source_term['label']} relates to {target_term['label']}",
                'source': {'term_id': source_term['id']},
                'target': {'term_id': target_term['id']},
                'confidence': {'minimum_score': 0.75},
                'constraints': {'hierarchy_level': 'document'}
            })

        return relationships

    def _generate_metadata(self) -> Dict[str, Any]:
        """Generate metadata about the discovery process."""
        return {
            'interview_summary': {
                'total_questions': len(self.conversation_history),
                'corpus_analyzed': True,
                'llm_enhanced': hasattr(self, 'llm_suggestions'),
                'discovery_date': str(pd.Timestamp.now()),
            },
            'corpus_evidence': self.corpus_patterns,
            'conversation_log': self.conversation_history,
            'generation_notes': [
                'Ontology generated through interactive interview',
                'Terms derived from user interests and corpus analysis',
                'Semantic variations based on corpus patterns',
                'Relationships inferred from user descriptions'
            ]
        }


def main():
    """Main CLI interface for interactive discovery."""
    import argparse

    parser = argparse.ArgumentParser(description='Interactive Ontology Discovery')
    parser.add_argument('--analytics-path', required=True,
                       help='Path to analytics data directory')
    parser.add_argument('--output-dir', default='.',
                       help='Output directory for generated ontology')

    args = parser.parse_args()

    if not os.path.exists(args.analytics_path):
        print(f"❌ Analytics path not found: {args.analytics_path}")
        return

    # Create interviewer
    interviewer = InteractiveDiscoveryInterview(args.analytics_path, args.output_dir)

    # Start interview
    try:
        ontology_path = interviewer.start_interview()
        print(f"\n🎉 Success! Next steps:")
        print(f"1. Review the generated ontology: {ontology_path}")
        print(f"2. Test entity extraction with: PYTHONPATH=src python -m go_doc_go.cli.ontology_extract")
        print(f"3. Refine the ontology based on extraction results")

    except KeyboardInterrupt:
        print("\n❌ Interview cancelled by user")
    except Exception as e:
        print(f"\n❌ Interview failed: {e}")


if __name__ == "__main__":
    main()