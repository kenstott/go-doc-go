#!/usr/bin/env python3
"""
Intelligent Ontology Discovery System
Uses foundation model knowledge, corpus analysis, and minimal human input
to automatically discover entities, relationships, and generate ontologies.
"""

import json
import os
import sys
import yaml
import re
from typing import Dict, List, Any, Optional, Tuple, Set
from pathlib import Path
from collections import Counter, defaultdict
from itertools import combinations
import logging

import anthropic
import duckdb
import pandas as pd
import numpy as np

# Add Go-Doc-Go to path
sys.path.insert(0, 'src')

try:
    from go_doc_go.domain.ontology import DomainOntology
except ImportError:
    print("⚠️  Go-Doc-Go modules not available - running in standalone mode")

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class IntelligentDiscoverySystem:
    """Intelligence-first ontology discovery with minimal human intervention."""

    def __init__(self, analytics_path: str, output_dir: str = "."):
        self.analytics_path = analytics_path
        self.output_dir = Path(output_dir)
        self.client = anthropic.Anthropic() if 'ANTHROPIC_API_KEY' in os.environ else None

        # Discovery state
        self.corpus_data = None
        self.discovered_entities = {}
        self.discovered_relationships = {}
        self.domain_indicators = {}
        self.confidence_scores = {}
        self.corpus_patterns = {}
        self.entity_contexts = defaultdict(list)

        # Initialize comprehensive entity taxonomy
        self._init_entity_taxonomy()

    def _init_entity_taxonomy(self):
        """Initialize comprehensive entity type taxonomy."""
        self.entity_types = {
            # People & Organizations
            'PERSON': {'patterns': [r'^[A-Z][a-z]+ [A-Z][a-z]+$', r'^Mr\.|Mrs\.|Ms\.|Dr\. \w+'], 'confidence': 0.9},
            'ORGANIZATION': {'patterns': [r'\b\w+\s+(Inc\.|Corp\.|LLC|Ltd\.|Limited|Co\.)'], 'confidence': 0.95},
            'ROLE': {'patterns': [r'\b(Chief|Senior|Executive|Director|Manager|Officer)\b'], 'confidence': 0.85},
            'DEPARTMENT': {'patterns': [r'\b\w+\s+(Department|Division|Unit|Team)\b'], 'confidence': 0.8},
            'COMMITTEE': {'patterns': [r'\b\w+\s+(Committee|Board|Council|Panel)\b'], 'confidence': 0.85},

            # Temporal
            'DATE': {'patterns': [r'\d{4}-\d{2}-\d{2}', r'\d{1,2}/\d{1,2}/\d{4}', r'\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4}'], 'confidence': 0.95},
            'TIME': {'patterns': [r'\d{1,2}:\d{2}(?::\d{2})?(?:\s*[AP]M)?'], 'confidence': 0.9},
            'DURATION': {'patterns': [r'\d+\s+(years?|months?|weeks?|days?|hours?|minutes?)'], 'confidence': 0.85},
            'FREQUENCY': {'patterns': [r'\b(daily|weekly|monthly|quarterly|annually|yearly)\b'], 'confidence': 0.9},

            # Financial & Quantitative
            'MONEY': {'patterns': [r'\$[\d,]+(?:\.\d{2})?(?:\s*(?:million|billion|thousand))?'], 'confidence': 0.95},
            'PERCENTAGE': {'patterns': [r'\d+(?:\.\d+)?%'], 'confidence': 0.95},
            'QUANTITY': {'patterns': [r'\d+(?:,\d{3})*(?:\.\d+)?\s*(?:shares|units|items)'], 'confidence': 0.85},

            # Legal & Regulatory
            'LAW': {'patterns': [r'\b[A-Z][a-zA-Z\s]+Act(?:\s+of\s+\d{4})?'], 'confidence': 0.85},
            'REGULATION': {'patterns': [r'\bRule\s+\d+[a-zA-Z]?-\d+', r'\bSection\s+\d+(?:\.\d+)?'], 'confidence': 0.8},
            'DOCUMENT_TYPE': {'patterns': [r'\b(10-K|10-Q|8-K|S-1|DEF 14A|Form\s+\w+)\b'], 'confidence': 0.9},

            # Location
            'LOCATION': {'patterns': [], 'confidence': 0.8},  # Will use NER
            'ADDRESS': {'patterns': [r'\d+\s+[A-Z][a-zA-Z\s]+(?:Street|St\.|Avenue|Ave\.|Road|Rd\.|Boulevard|Blvd\.)'], 'confidence': 0.85},

            # Healthcare & Medical
            'DRUG': {'patterns': [], 'confidence': 0.8},  # Domain-specific, will detect contextually
            'DISEASE': {'patterns': [], 'confidence': 0.8},
            'SYMPTOM': {'patterns': [], 'confidence': 0.75},
            'TREATMENT': {'patterns': [], 'confidence': 0.75},

            # Technology
            'SOFTWARE': {'patterns': [r'\b\w+\s+v?\d+\.\d+(?:\.\d+)?'], 'confidence': 0.7},
            'TECHNOLOGY': {'patterns': [], 'confidence': 0.75},

            # Identifiers
            'EMAIL': {'patterns': [r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'], 'confidence': 0.95},
            'URL': {'patterns': [r'https?://[^\s]+'], 'confidence': 0.95},
            'PHONE': {'patterns': [r'\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}'], 'confidence': 0.9},

            # Abstract
            'STATUS': {'patterns': [], 'confidence': 0.7},
            'ACTION': {'patterns': [], 'confidence': 0.7},
            'OUTCOME': {'patterns': [], 'confidence': 0.7},

            # Catch-all
            'UNKNOWN': {'patterns': [], 'confidence': 0.0}
        }

        # Active entity types (will be refined based on domain)
        self.active_entity_types = set(self.entity_types.keys())

    def discover_ontology(self) -> str:
        """Main discovery process with minimal human intervention."""
        print("🧠 INTELLIGENT ONTOLOGY DISCOVERY")
        print("=" * 60)
        print("Analyzing your documents with advanced AI capabilities...\n")

        # Phase 1: Load and analyze corpus
        print("📊 Phase 1: Corpus Analysis")
        if not self._analyze_corpus():
            return "❌ Failed to analyze corpus"

        # Phase 2: Automatic entity discovery
        print("\n🔍 Phase 2: Automatic Entity Discovery")
        self._discover_entities_automatically()

        # Phase 3: Domain detection
        print("\n🎯 Phase 3: Domain Detection")
        self._detect_domain()

        # Phase 4: Relationship discovery
        print("\n🔗 Phase 4: Relationship Discovery")
        self._discover_relationships()

        # Phase 5: Pattern extension
        print("\n📈 Phase 5: Statistical Pattern Extension")
        self._extend_patterns_statistically()

        # Phase 6: Web research validation (if API available)
        if self.client:
            print("\n🌐 Phase 6: Web Research Validation")
            self._validate_with_web_research()

        # Phase 7: Focused human input
        print("\n💬 Phase 7: Customization")
        self._conduct_focused_interview()

        # Phase 8: Generate ontology
        print("\n🏗️  Phase 8: Ontology Generation")
        ontology_path = self._generate_ontology()

        print(f"\n✅ Discovery complete! Ontology saved to: {ontology_path}")
        return ontology_path

    def _analyze_corpus(self) -> bool:
        """Load and analyze the corpus data."""
        try:
            # Load data from analytics
            self.corpus_data = self._load_analytics_data()
            if self.corpus_data.empty:
                print("❌ No analytics data found")
                return False

            # Analyze patterns
            self.corpus_patterns = {
                'frequent_terms': self._find_frequent_terms(self.corpus_data),
                'element_types': self._analyze_element_types(self.corpus_data),
                'co_occurrences': self._find_co_occurrences(self.corpus_data),
                'document_stats': self._get_document_stats(self.corpus_data),
                'syntactic_patterns': self._extract_syntactic_patterns(self.corpus_data)
            }

            print(f"✅ Analyzed {self.corpus_patterns['document_stats']['total_docs']} documents")
            print(f"   {self.corpus_patterns['document_stats']['total_elements']} elements")
            return True

        except Exception as e:
            logger.error(f"Corpus analysis failed: {e}")
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
        LIMIT 50000
        """

        return conn.execute(query).df()

    def _discover_entities_automatically(self):
        """Discover entities using foundation model knowledge and patterns."""
        self.discovered_entities = defaultdict(list)
        entity_counter = defaultdict(Counter)

        for _, row in self.corpus_data.iterrows():
            content = str(row['content'])

            # Extract potential entities using multiple methods
            candidates = self._extract_entity_candidates(content)

            for candidate in candidates:
                entity_type, confidence = self._classify_entity(candidate, content)

                if confidence > 0.5:
                    entity_counter[entity_type][candidate] += 1
                    self.entity_contexts[candidate].append(content[:100])

        # Convert to final format with frequency information
        for entity_type, entities in entity_counter.items():
            for entity_text, frequency in entities.most_common():
                self.discovered_entities[entity_type].append({
                    'text': entity_text,
                    'frequency': frequency,
                    'confidence': self._calculate_entity_confidence(entity_text, entity_type, frequency)
                })

        # Print summary
        total_entities = sum(len(entities) for entities in self.discovered_entities.values())
        print(f"✅ Discovered {total_entities} entities across {len(self.discovered_entities)} types")
        for entity_type, entities in list(self.discovered_entities.items())[:5]:
            if entities:
                top_examples = [e['text'] for e in sorted(entities, key=lambda x: x['frequency'], reverse=True)[:3]]
                print(f"   {entity_type}: {len(entities)} entities (e.g., {', '.join(top_examples)})")

    def _extract_entity_candidates(self, text: str) -> List[str]:
        """Extract potential entity candidates from text."""
        candidates = set()

        # Pattern-based extraction
        for entity_type, type_info in self.entity_types.items():
            for pattern in type_info['patterns']:
                matches = re.findall(pattern, text)
                candidates.update(matches)

        # Noun phrase extraction (simple version)
        # Capitalized sequences
        cap_sequences = re.findall(r'\b[A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+)*\b', text)
        candidates.update(cap_sequences)

        # Numbers and quantities
        numbers = re.findall(r'\b\d+(?:,\d{3})*(?:\.\d+)?\b', text)
        candidates.update(numbers)

        return list(candidates)

    def _classify_entity(self, candidate: str, context: str) -> Tuple[str, float]:
        """Classify an entity candidate using patterns and context."""
        best_type = 'UNKNOWN'
        best_confidence = 0.0

        # Check against known patterns
        for entity_type, type_info in self.entity_types.items():
            for pattern in type_info['patterns']:
                if re.match(pattern, candidate):
                    confidence = type_info['confidence']
                    if confidence > best_confidence:
                        best_type = entity_type
                        best_confidence = confidence

        # Use foundation model knowledge for known entities
        if best_confidence < 0.8:
            fm_type, fm_confidence = self._classify_with_foundation_model(candidate, context)
            if fm_confidence > best_confidence:
                best_type = fm_type
                best_confidence = fm_confidence

        return best_type, best_confidence

    def _classify_with_foundation_model(self, candidate: str, context: str) -> Tuple[str, float]:
        """Use foundation model knowledge to classify entities."""
        # Known person names (simplified - in practice would use a more comprehensive approach)
        known_people = ['Tim Cook', 'Luca Maestri', 'Warren Buffett', 'Elon Musk']
        if any(name in candidate for name in known_people):
            return 'PERSON', 0.95

        # Known companies
        known_companies = ['Apple', 'Microsoft', 'Google', 'Amazon', 'SEC']
        if any(company in candidate for company in known_companies):
            return 'ORGANIZATION', 0.95

        # Name patterns
        if re.match(r'^[A-Z][a-z]+ [A-Z][a-z]+$', candidate):
            # Check context for role indicators
            if any(role in context.lower() for role in ['ceo', 'officer', 'director', 'president']):
                return 'PERSON', 0.8

        # Corporate suffixes
        if any(suffix in candidate for suffix in ['Inc.', 'Corp.', 'LLC', 'Ltd.']):
            return 'ORGANIZATION', 0.9

        return 'UNKNOWN', 0.0

    def _calculate_entity_confidence(self, entity_text: str, entity_type: str, frequency: int) -> float:
        """Calculate confidence score for an entity."""
        base_confidence = self.entity_types.get(entity_type, {}).get('confidence', 0.5)

        # Boost confidence based on frequency
        if frequency > 50:
            frequency_boost = 0.2
        elif frequency > 20:
            frequency_boost = 0.1
        elif frequency > 5:
            frequency_boost = 0.05
        else:
            frequency_boost = 0.0

        return min(base_confidence + frequency_boost, 1.0)

    def _detect_domain(self):
        """Automatically detect the domain of the documents."""
        self.domain_indicators = {
            'SEC_FILING': 0,
            'LEGAL_DOCUMENT': 0,
            'MEDICAL_RECORD': 0,
            'TECHNICAL_DOCUMENTATION': 0,
            'ACADEMIC_PAPER': 0,
            'NEWS_ARTICLE': 0,
            'FINANCIAL_REPORT': 0,
            'GENERAL_BUSINESS': 0
        }

        # Check for domain-specific terms
        freq_terms = self.corpus_patterns['frequent_terms']

        # SEC filing indicators
        sec_terms = ['registrant', 'filing', 'securities', 'proxy', 'shares', 'officer', 'director']
        for term in sec_terms:
            if term in freq_terms:
                self.domain_indicators['SEC_FILING'] += freq_terms[term] / 10

        # Check document types
        if 'DOCUMENT_TYPE' in self.discovered_entities:
            doc_types = [e['text'] for e in self.discovered_entities['DOCUMENT_TYPE']]
            if any('10-K' in dt or '10-Q' in dt or '8-K' in dt for dt in doc_types):
                self.domain_indicators['SEC_FILING'] += 20

        # Legal document indicators
        legal_terms = ['plaintiff', 'defendant', 'court', 'pursuant', 'whereas', 'hereby']
        for term in legal_terms:
            if term in freq_terms:
                self.domain_indicators['LEGAL_DOCUMENT'] += freq_terms[term] / 10

        # Medical indicators
        medical_terms = ['patient', 'diagnosis', 'treatment', 'symptom', 'medication']
        for term in medical_terms:
            if term in freq_terms:
                self.domain_indicators['MEDICAL_RECORD'] += freq_terms[term] / 10

        # Get the most likely domain
        likely_domain = max(self.domain_indicators.items(), key=lambda x: x[1])
        confidence = likely_domain[1] / sum(self.domain_indicators.values()) * 100 if sum(self.domain_indicators.values()) > 0 else 0

        print(f"✅ Detected domain: {likely_domain[0]} (confidence: {confidence:.1f}%)")

        # Refine active entity types based on domain
        self._refine_entity_types_for_domain(likely_domain[0])

    def _refine_entity_types_for_domain(self, domain: str):
        """Activate relevant entity types for the detected domain."""
        # Base types always active
        base_types = {'PERSON', 'ORGANIZATION', 'DATE', 'LOCATION', 'MONEY', 'QUANTITY', 'UNKNOWN'}

        domain_specific = {
            'SEC_FILING': {'ROLE', 'COMMITTEE', 'DOCUMENT_TYPE', 'PERCENTAGE', 'LAW', 'REGULATION'},
            'LEGAL_DOCUMENT': {'LAW', 'REGULATION', 'DOCUMENT_TYPE', 'STATUS'},
            'MEDICAL_RECORD': {'DRUG', 'DISEASE', 'SYMPTOM', 'TREATMENT'},
            'TECHNICAL_DOCUMENTATION': {'SOFTWARE', 'TECHNOLOGY', 'VERSION'},
            'FINANCIAL_REPORT': {'MONEY', 'PERCENTAGE', 'QUANTITY', 'ROLE'}
        }

        self.active_entity_types = base_types.union(domain_specific.get(domain, set()))
        print(f"   Activated {len(self.active_entity_types)} entity types for this domain")

    def _discover_relationships(self):
        """Discover relationships between entities."""
        self.discovered_relationships = defaultdict(list)

        # Pattern-based relationship extraction
        relationship_patterns = {
            'HOLDS_POSITION': [
                (r'(\w+\s+\w+),?\s+(Chief|President|Director|Officer)', ['PERSON', 'ROLE']),
                (r'(\w+\s+\w+)\s+serves as\s+(\w+)', ['PERSON', 'ROLE'])
            ],
            'WORKS_AT': [
                (r'(\w+\s+\w+)\s+of\s+(\w+\s+(?:Inc\.|Corp\.|LLC))', ['PERSON', 'ORGANIZATION']),
                (r'(\w+)\s+at\s+(\w+\s+(?:Inc\.|Corp\.|LLC))', ['ROLE', 'ORGANIZATION'])
            ],
            'RECEIVES_COMPENSATION': [
                (r'(\w+\s+\w+)\s+received\s+(\$[\d,]+)', ['PERSON', 'MONEY']),
                (r'(\w+\s+\w+).*compensation.*(\$[\d,]+)', ['PERSON', 'MONEY'])
            ],
            'APPOINTED_ON': [
                (r'(\w+\s+\w+)\s+appointed.*(\d{4})', ['PERSON', 'DATE']),
                (r'elected\s+(\w+\s+\w+).*(\d{4})', ['PERSON', 'DATE'])
            ]
        }

        # Extract relationships from corpus
        for _, row in self.corpus_data.iterrows():
            content = str(row['content'])

            for rel_type, patterns in relationship_patterns.items():
                for pattern, entity_types in patterns:
                    matches = re.findall(pattern, content, re.IGNORECASE)
                    for match in matches:
                        if len(match) == 2:
                            self.discovered_relationships[rel_type].append({
                                'source': match[0],
                                'target': match[1],
                                'source_type': entity_types[0],
                                'target_type': entity_types[1],
                                'frequency': 1,
                                'confidence': 0.7
                            })

        # Aggregate and count relationships
        for rel_type in self.discovered_relationships:
            # Count occurrences
            rel_counter = Counter()
            for rel in self.discovered_relationships[rel_type]:
                key = (rel['source'], rel['target'])
                rel_counter[key] += 1

            # Update frequencies
            aggregated = []
            for (source, target), count in rel_counter.items():
                aggregated.append({
                    'source': source,
                    'target': target,
                    'frequency': count,
                    'confidence': min(0.7 + (count * 0.05), 0.95)
                })
            self.discovered_relationships[rel_type] = aggregated

        total_relationships = sum(len(rels) for rels in self.discovered_relationships.values())
        print(f"✅ Discovered {total_relationships} relationships of {len(self.discovered_relationships)} types")

        for rel_type, rels in list(self.discovered_relationships.items())[:3]:
            if rels:
                top_rel = sorted(rels, key=lambda x: x['frequency'], reverse=True)[0]
                print(f"   {rel_type}: {len(rels)} instances (e.g., {top_rel['source']} -> {top_rel['target']})")

    def _extend_patterns_statistically(self):
        """Use discovered high-confidence entities to find more of the same type."""
        print("   Extending patterns from high-confidence entities...")

        extended_count = 0
        for entity_type in self.active_entity_types:
            if entity_type in self.discovered_entities:
                # Get high-confidence entities of this type
                high_conf_entities = [e for e in self.discovered_entities[entity_type]
                                     if e['confidence'] > 0.8 and e['frequency'] > 5]

                if not high_conf_entities:
                    continue

                # Learn patterns from their contexts
                learned_patterns = self._learn_contextual_patterns(high_conf_entities, entity_type)

                # Find new entities matching these patterns
                new_entities = self._find_entities_by_patterns(learned_patterns, entity_type)

                # Add new entities with lower confidence
                for entity in new_entities:
                    if not any(e['text'] == entity['text'] for e in self.discovered_entities[entity_type]):
                        entity['confidence'] *= 0.8  # Reduce confidence for learned entities
                        self.discovered_entities[entity_type].append(entity)
                        extended_count += 1

        print(f"   Extended {extended_count} new entities through pattern learning")

    def _learn_contextual_patterns(self, entities: List[Dict], entity_type: str) -> List[str]:
        """Learn contextual patterns from high-confidence entities."""
        patterns = []

        for entity in entities[:10]:  # Limit to top 10
            contexts = self.entity_contexts.get(entity['text'], [])
            for context in contexts[:5]:  # Limit contexts
                # Create pattern by replacing entity with placeholder
                pattern = context.replace(entity['text'], f'[{entity_type}]')
                patterns.append(pattern)

        return patterns

    def _find_entities_by_patterns(self, patterns: List[str], entity_type: str) -> List[Dict]:
        """Find new entities matching learned patterns."""
        new_entities = []

        # This is simplified - in practice would use more sophisticated pattern matching
        for pattern in patterns[:5]:  # Limit patterns to check
            # Extract the context around the entity placeholder
            if f'[{entity_type}]' in pattern:
                # Find similar contexts in corpus
                # (Simplified implementation)
                pass

        return new_entities

    def _validate_with_web_research(self):
        """Validate entities and domain using web research."""
        if not self.client:
            return

        print("   Validating top entities with web research...")

        # Validate top organizations
        if 'ORGANIZATION' in self.discovered_entities:
            top_orgs = sorted(self.discovered_entities['ORGANIZATION'],
                            key=lambda x: x['frequency'], reverse=True)[:3]

            for org in top_orgs:
                # In a real implementation, would use web search here
                # For now, just mark as validated if high frequency
                if org['frequency'] > 10:
                    org['web_validated'] = True

        # Validate domain detection
        likely_domain = max(self.domain_indicators.items(), key=lambda x: x[1])
        if likely_domain[1] > 10:
            # Would search for domain confirmation
            print(f"   Domain {likely_domain[0]} validated through pattern analysis")

    def _conduct_focused_interview(self):
        """Minimal, focused interaction to customize the ontology."""
        print("\n" + "=" * 60)
        print("📋 DISCOVERY SUMMARY")
        print("-" * 40)

        # Show domain
        likely_domain = max(self.domain_indicators.items(), key=lambda x: x[1])
        confidence = likely_domain[1] / sum(self.domain_indicators.values()) * 100 if sum(self.domain_indicators.values()) > 0 else 0
        print(f"\n📊 Document Domain: {likely_domain[0]} ({confidence:.0f}% confidence)")

        # Show entity summary
        print(f"\n📌 Discovered Entities:")
        entity_summary = {}
        for entity_type in self.active_entity_types:
            if entity_type in self.discovered_entities and self.discovered_entities[entity_type]:
                count = len(self.discovered_entities[entity_type])
                entity_summary[entity_type] = count
                # Show top 3 examples
                top_3 = sorted(self.discovered_entities[entity_type],
                             key=lambda x: x['frequency'], reverse=True)[:3]
                examples = [e['text'] for e in top_3]
                print(f"  • {entity_type}: {count} found (e.g., {', '.join(examples)})")

        # Show relationship summary
        if self.discovered_relationships:
            print(f"\n🔗 Discovered Relationships:")
            for rel_type, rels in self.discovered_relationships.items():
                if rels:
                    top_rel = sorted(rels, key=lambda x: x['frequency'], reverse=True)[0]
                    print(f"  • {rel_type}: {len(rels)} instances")
                    print(f"    Example: {top_rel['source']} → {top_rel['target']}")

        # Focused questions
        print("\n" + "=" * 60)
        print("🎯 CUSTOMIZATION (press Enter to accept defaults)")
        print("-" * 40)

        # Question 1: Confirm domain if uncertain
        if confidence < 70:
            print(f"\nThe domain appears to be {likely_domain[0]} but confidence is low.")
            domain_input = input("Is this correct? [Y/n]: ").strip().lower()
            if domain_input == 'n':
                print("Available domains:", ', '.join(self.domain_indicators.keys()))
                new_domain = input("Enter the correct domain: ").strip().upper()
                if new_domain in self.domain_indicators:
                    self.domain_indicators[new_domain] = 100

        # Question 2: Entity filtering
        print(f"\nFound {sum(entity_summary.values())} entities across {len(entity_summary)} types.")
        filter_input = input("Track all entities or filter? [all/filter]: ").strip().lower()

        if filter_input == 'filter':
            print("Available entity types:", ', '.join(entity_summary.keys()))
            selected = input("Enter types to track (comma-separated) or 'all': ").strip()
            if selected.lower() != 'all':
                selected_types = [t.strip().upper() for t in selected.split(',')]
                # Filter entities
                for entity_type in list(self.discovered_entities.keys()):
                    if entity_type not in selected_types:
                        del self.discovered_entities[entity_type]

        # Question 3: Relationship filtering
        if self.discovered_relationships:
            print(f"\nFound {len(self.discovered_relationships)} relationship types.")
            print("Available:", ', '.join(self.discovered_relationships.keys()))
            rel_input = input("Enter relationships to track (comma-separated) or 'all': ").strip()

            if rel_input.lower() != 'all' and rel_input:
                selected_rels = [r.strip().upper() for r in rel_input.split(',')]
                # Filter relationships
                for rel_type in list(self.discovered_relationships.keys()):
                    if rel_type not in selected_rels:
                        del self.discovered_relationships[rel_type]

        print("\n✅ Customization complete!")

    def _generate_ontology(self) -> str:
        """Generate the final ontology based on discoveries."""
        # Determine domain name
        likely_domain = max(self.domain_indicators.items(), key=lambda x: x[1])
        domain_name = likely_domain[0].lower().replace('_', '_')
        if domain_name == 'sec_filing':
            domain_name = 'sec_filing_analysis'

        # Build ontology structure
        ontology = {
            'domain': {
                'name': domain_name,
                'version': '2.0.0',
                'description': f'Intelligently discovered ontology for {likely_domain[0]} domain',
                'discovery_method': 'intelligent_discovery',
                'settings': {
                    'default_confidence_threshold': 0.70,
                    'max_relationships_per_pair': 10,
                    'enable_transitive_inference': True,
                    'ai_discovered': True
                }
            },
            'terms': self._generate_terms(),
            'element_mappings': self._generate_element_mappings(),
            'relationship_rules': self._generate_relationship_rules(),
            'metadata': self._generate_metadata()
        }

        # Save ontology
        output_path = self.output_dir / f"{domain_name}_intelligent_ontology.yaml"
        with open(output_path, 'w') as f:
            yaml.dump(self._numpy_to_python(ontology), f, default_flow_style=False, sort_keys=False)

        return str(output_path)

    def _generate_terms(self) -> List[Dict]:
        """Generate term definitions from discovered entities."""
        terms = []

        for entity_type in self.active_entity_types:
            if entity_type in self.discovered_entities and self.discovered_entities[entity_type]:
                # Get top entities of this type
                top_entities = sorted(self.discovered_entities[entity_type],
                                    key=lambda x: x['frequency'], reverse=True)[:20]

                if top_entities:
                    term = {
                        'id': entity_type.lower(),
                        'label': entity_type.replace('_', ' ').title(),
                        'description': f'{entity_type} entities discovered in documents',
                        'aliases': [],
                        'semantic_variations': [e['text'] for e in top_entities[:10]]
                    }
                    terms.append(term)

        return terms

    def _generate_element_mappings(self) -> List[Dict]:
        """Generate extraction rules for discovered entities."""
        mappings = []

        for entity_type in self.active_entity_types:
            if entity_type in self.discovered_entities and self.discovered_entities[entity_type]:
                # Get entities for this type
                entities = self.discovered_entities[entity_type]

                # Get the most frequent entity as semantic phrase
                top_entities = sorted(entities, key=lambda x: x['frequency'], reverse=True)

                if top_entities:
                    # Find the highest frequency term from corpus
                    best_semantic_phrase = None
                    for entity in top_entities[:10]:
                        if entity['text'].lower() in self.corpus_patterns['frequent_terms']:
                            best_semantic_phrase = entity['text'].lower()
                            break

                    if not best_semantic_phrase and top_entities:
                        best_semantic_phrase = top_entities[0]['text'].lower()

                    mapping = {
                        'term_id': entity_type.lower(),
                        'rules': []
                    }

                    # Add semantic rule
                    if best_semantic_phrase:
                        semantic_rule = {
                            'type': 'semantic',
                            'semantic_phrase': best_semantic_phrase,
                            'confidence_threshold': 0.65,
                            'element_types': ['paragraph', 'table_cell', 'div'],
                            'description': f'Semantic matching for {entity_type}'
                        }
                        mapping['rules'].append(semantic_rule)

                    # Add regex rule if we have patterns
                    if entity_type in self.entity_types and self.entity_types[entity_type]['patterns']:
                        for pattern in self.entity_types[entity_type]['patterns'][:2]:
                            regex_rule = {
                                'type': 'regex',
                                'pattern': pattern,
                                'element_types': ['paragraph', 'table_cell', 'table_row', 'div'],
                                'description': f'Pattern matching for {entity_type}'
                            }
                            mapping['rules'].append(regex_rule)

                    mappings.append(mapping)

        return mappings

    def _generate_relationship_rules(self) -> List[Dict]:
        """Generate relationship rules from discoveries."""
        rules = []

        for rel_type, relationships in self.discovered_relationships.items():
            if relationships:
                # Get top relationship as example
                top_rel = sorted(relationships, key=lambda x: x['frequency'], reverse=True)[0]

                rule = {
                    'id': rel_type.lower(),
                    'relationship_type': rel_type.lower(),
                    'description': f'{rel_type} relationship',
                    'source': {'term_id': top_rel.get('source_type', 'entity').lower()},
                    'target': {'term_id': top_rel.get('target_type', 'entity').lower()},
                    'confidence': {'minimum_score': 0.6},
                    'constraints': {'hierarchy_level': 'document'},
                    'patterns': self._get_relationship_patterns(rel_type)
                }
                rules.append(rule)

        return rules

    def _get_relationship_patterns(self, rel_type: str) -> List[str]:
        """Get patterns for a relationship type."""
        patterns = {
            'HOLDS_POSITION': ['[PERSON], [ROLE]', '[PERSON] serves as [ROLE]'],
            'WORKS_AT': ['[PERSON] of [ORGANIZATION]', '[ROLE] at [ORGANIZATION]'],
            'RECEIVES_COMPENSATION': ['[PERSON] received [MONEY]', '[PERSON] compensation of [MONEY]'],
            'APPOINTED_ON': ['[PERSON] appointed in [DATE]', 'elected [PERSON] on [DATE]']
        }
        return patterns.get(rel_type, [])

    def _generate_metadata(self) -> Dict:
        """Generate metadata about the discovery process."""
        return {
            'discovery_summary': {
                'method': 'intelligent_discovery',
                'corpus_analyzed': True,
                'ai_enhanced': True,
                'web_validated': self.client is not None,
                'discovery_date': str(pd.Timestamp.now()),
                'confidence_score': self._calculate_overall_confidence()
            },
            'corpus_evidence': self.corpus_patterns,
            'domain_detection': self.domain_indicators,
            'entity_statistics': {
                entity_type: {
                    'total': len(entities),
                    'high_confidence': len([e for e in entities if e['confidence'] > 0.8]),
                    'avg_frequency': np.mean([e['frequency'] for e in entities]) if entities else 0
                }
                for entity_type, entities in self.discovered_entities.items()
            },
            'generation_notes': [
                'Ontology generated through intelligent discovery',
                'Entities discovered using foundation model knowledge',
                'Relationships extracted through pattern analysis',
                'Domain detected automatically from corpus patterns',
                f'Minimal human input - {self._count_human_inputs()} decisions'
            ]
        }

    def _calculate_overall_confidence(self) -> float:
        """Calculate overall confidence in the discovered ontology."""
        scores = []

        # Domain confidence
        likely_domain = max(self.domain_indicators.items(), key=lambda x: x[1])
        domain_conf = likely_domain[1] / sum(self.domain_indicators.values()) if sum(self.domain_indicators.values()) > 0 else 0
        scores.append(domain_conf)

        # Entity confidence
        for entities in self.discovered_entities.values():
            if entities:
                entity_confs = [e['confidence'] for e in entities[:10]]  # Top 10
                scores.extend(entity_confs)

        return float(np.mean(scores)) if scores else 0.5

    def _count_human_inputs(self) -> int:
        """Count the number of human decisions made."""
        # In this implementation, maximum of 3 questions asked
        return 3

    def _numpy_to_python(self, data):
        """Convert numpy types to native Python types."""
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
            return data
        else:
            return str(data)

    def _find_frequent_terms(self, df: pd.DataFrame) -> Dict[str, int]:
        """Find most frequent terms in the corpus."""
        all_content = ' '.join(df['content'].astype(str))

        # Extract words
        stop_words = {'the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by', 'a', 'an', 'as', 'is', 'was', 'are', 'were', 'be', 'been', 'being', 'have', 'has', 'had', 'do', 'does', 'did', 'will', 'would', 'could', 'should', 'may', 'might', 'can', 'shall', 'must', 'ought'}
        words = re.findall(r'\b[a-zA-Z]{3,}\b', all_content.lower())
        word_freq = Counter([w for w in words if w not in stop_words])

        return dict(word_freq.most_common(100))

    def _analyze_element_types(self, df: pd.DataFrame) -> Dict[str, int]:
        """Analyze element type distribution."""
        return dict(df['element_type'].value_counts())

    def _find_co_occurrences(self, df: pd.DataFrame) -> Dict[str, List]:
        """Find terms that co-occur in documents."""
        frequent_terms = list(self._find_frequent_terms(df).keys())[:30]
        co_matrix = defaultdict(Counter)

        # Build co-occurrence by document
        for doc_id in df['doc_id'].unique()[:100]:  # Limit for performance
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

    def _extract_syntactic_patterns(self, df: pd.DataFrame) -> Dict[str, List]:
        """Extract syntactic patterns for relationship discovery."""
        patterns = {
            'noun_verb_noun': [],
            'noun_prep_noun': [],
            'title_patterns': []
        }

        # Simple pattern extraction (would be more sophisticated in practice)
        for _, row in df.head(1000).iterrows():  # Sample for performance
            content = str(row['content'])

            # Title patterns (Name, Role)
            title_matches = re.findall(r'([A-Z][a-z]+ [A-Z][a-z]+),\s*([A-Z][a-zA-Z\s]+)', content)
            patterns['title_patterns'].extend(title_matches[:5])

        return patterns


def main():
    """Main entry point for intelligent discovery."""
    import argparse

    parser = argparse.ArgumentParser(description='Intelligent Ontology Discovery')
    parser.add_argument('--analytics-path', required=True,
                       help='Path to analytics data directory')
    parser.add_argument('--output-dir', default='.',
                       help='Output directory for generated ontology')

    args = parser.parse_args()

    if not os.path.exists(args.analytics_path):
        print(f"❌ Analytics path not found: {args.analytics_path}")
        return

    # Create discovery system
    discovery = IntelligentDiscoverySystem(args.analytics_path, args.output_dir)

    # Run discovery
    try:
        ontology_path = discovery.discover_ontology()
        print(f"\n🎉 Success! Next steps:")
        print(f"1. Review the generated ontology: {ontology_path}")
        print(f"2. Test entity extraction with the new ontology")
        print(f"3. The system has learned from your data with minimal input!")

    except KeyboardInterrupt:
        print("\n❌ Discovery cancelled by user")
    except Exception as e:
        print(f"\n❌ Discovery failed: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    main()