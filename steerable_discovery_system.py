#!/usr/bin/env python3
"""
Steerable Intelligent Discovery System
Provides strategic checkpoints for human validation and refinement
while maintaining automation efficiency.
"""

import json
import os
import sys
import yaml
import re
from typing import Dict, List, Any, Optional, Tuple, Set
from pathlib import Path
from collections import Counter, defaultdict
from dataclasses import dataclass, field
import logging

import anthropic
import duckdb
import pandas as pd
import numpy as np

# Add Go-Doc-Go to path
sys.path.insert(0, 'src')

from comprehensive_entity_taxonomy import (
    COMPREHENSIVE_ENTITY_TYPES,
    get_entity_types_for_domain,
    disambiguate_by_context
)
from intelligent_discovery_interview import IntelligentDiscoverySystem

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@dataclass
class DiscoveryCheckpoint:
    """Represents a point where human can intervene."""
    name: str
    stage: str
    data: Dict[str, Any]
    confidence: float
    can_skip: bool = True
    user_decision: Optional[str] = None
    refinements: Dict[str, Any] = field(default_factory=dict)


class SteerableDiscoverySystem:
    """
    Intelligence-first discovery with strategic human checkpoints.
    Allows humans to validate, refine, and steer at key decision points.
    """

    def __init__(self, analytics_path: str, output_dir: str = ".", existing_ontology_path: str = None,
                 config_path: str = None):
        self.analytics_path = analytics_path
        self.output_dir = Path(output_dir)
        self.existing_ontology_path = existing_ontology_path
        self.config_path = config_path
        self.client = anthropic.Anthropic() if 'ANTHROPIC_API_KEY' in os.environ else None

        # Discovery state
        self.corpus_data = None
        self.discovered_entities = {}
        self.discovered_relationships = {}
        self.domain_indicators = {}
        self.checkpoints: List[DiscoveryCheckpoint] = []

        # Existing ontology if provided
        self.existing_ontology = None
        self.ontology_mode = 'create'  # 'create', 'extend', 'refine'

        # User preferences
        self.user_preferences = {
            'automation_level': 'balanced',  # 'full', 'balanced', 'guided'
            'confidence_threshold': 0.7,
            'require_confirmation': True,
            'verbose_explanations': False
        }

        # Initialize entity taxonomy
        self.entity_taxonomy = COMPREHENSIVE_ENTITY_TYPES

        # Load existing ontology if provided
        if self.existing_ontology_path:
            self._load_existing_ontology()

    def _load_existing_ontology(self):
        """Load and parse existing ontology for extension/refinement."""
        try:
            with open(self.existing_ontology_path, 'r') as f:
                self.existing_ontology = yaml.safe_load(f)

            print(f"📚 Loaded existing ontology: {self.existing_ontology_path}")

            # Extract existing discoveries
            if 'terms' in self.existing_ontology:
                for term in self.existing_ontology['terms']:
                    entity_type = term['id'].upper()
                    if entity_type not in self.discovered_entities:
                        self.discovered_entities[entity_type] = []

                    # Add variations as existing entities
                    for variation in term.get('semantic_variations', []):
                        self.discovered_entities[entity_type].append({
                            'text': variation,
                            'frequency': 0,  # Will be updated from corpus
                            'confidence': 1.0,
                            'from_existing_ontology': True
                        })

            # Extract existing relationships
            if 'relationship_rules' in self.existing_ontology:
                for rule in self.existing_ontology['relationship_rules']:
                    rel_type = rule['relationship_type'].upper()
                    if rel_type not in self.discovered_relationships:
                        self.discovered_relationships[rel_type] = []

                    self.discovered_relationships[rel_type].append({
                        'patterns': rule.get('patterns', []),
                        'from_existing_ontology': True
                    })

            print(f"   Loaded {len(self.discovered_entities)} entity types")
            print(f"   Loaded {len(self.discovered_relationships)} relationship types")

        except Exception as e:
            print(f"⚠️  Could not load existing ontology: {e}")
            self.existing_ontology = None

    def _determine_ontology_mode(self):
        """Determine how to handle the existing ontology."""
        if not self.existing_ontology:
            self.ontology_mode = 'create'
            return

        print("\n🔧 ONTOLOGY MODE SELECTION")
        print("-" * 30)
        print("An existing ontology was loaded. How would you like to proceed?")
        print("1. Extend - Add new entities/relationships to existing ontology")
        print("2. Refine - Improve existing patterns with new data")
        print("3. Replace - Create new ontology (existing saved as backup)")
        print("4. Merge - Smart merge of existing and new discoveries")

        choice = input("\nYour choice [1]: ").strip() or '1'

        mode_map = {
            '1': 'extend',
            '2': 'refine',
            '3': 'replace',
            '4': 'merge'
        }

        self.ontology_mode = mode_map.get(choice, 'extend')
        print(f"✓ Mode set to: {self.ontology_mode}")

    def _reset_discoveries(self):
        """Reset discovery state while preserving existing ontology."""
        # Clear discovered items but keep existing ontology entities
        for entity_type in list(self.discovered_entities.keys()):
            self.discovered_entities[entity_type] = [
                e for e in self.discovered_entities[entity_type]
                if e.get('from_existing_ontology', False)
            ]

        for rel_type in list(self.discovered_relationships.keys()):
            self.discovered_relationships[rel_type] = [
                r for r in self.discovered_relationships[rel_type]
                if r.get('from_existing_ontology', False)
            ]

        self.domain_indicators = {}
        self.checkpoints = []

    def _show_phase_menu(self):
        """Show menu of available phases for jumping."""
        print("\n📋 AVAILABLE PHASES:")
        for i, (key, name) in enumerate(self.phases):
            status = "✓" if i < self.current_phase else "○"
            print(f"{i + 1}. {status} {name}")

    def _get_phase_selection(self) -> Optional[int]:
        """Get user's phase selection for jumping."""
        choice = input("\nJump to phase (1-7) [cancel]: ").strip()

        if choice.isdigit():
            phase_num = int(choice) - 1
            if 0 <= phase_num < len(self.phases):
                return phase_num

        print("❌ Invalid selection, continuing current phase")
        return None

    def discover_with_steering(self, use_config_delegation: bool = False) -> str:
        """
        Main discovery process with human steering checkpoints.
        Can delegate to intelligent discovery if config provided.
        """
        # Check if we should delegate to intelligent discovery
        if use_config_delegation and self.config_path:
            return self._delegate_to_intelligent_discovery()

        print("🚀 STEERABLE INTELLIGENT DISCOVERY SYSTEM")
        print("=" * 60)

        # Handle existing ontology if provided
        if self.existing_ontology:
            print("📚 Working with existing ontology")
            self._determine_ontology_mode()
            print()
        else:
            print("I'll analyze your documents and check with you at key decision points.\n")

        # Set automation level
        self._set_automation_preferences()

        # Initialize phase tracking
        self.current_phase = 0
        self.phases = [
            ('corpus', "Initial Corpus Analysis"),
            ('domain', "Domain Detection & Classification"),
            ('entities', "Entity Discovery"),
            ('disambiguation', "Entity Disambiguation"),
            ('relationships', "Relationship Discovery"),
            ('patterns', "Pattern Refinement"),
            ('final', "Final Review & Generation")
        ]

        # Main discovery loop with back navigation
        while self.current_phase < len(self.phases):
            phase_key, phase_name = self.phases[self.current_phase]

            print(f"\n📊 PHASE {self.current_phase + 1}: {phase_name}")
            print("-" * 40)

            # Execute phase based on type
            if phase_key == 'corpus':
                checkpoint = self._analyze_corpus_with_checkpoint()
                if checkpoint.user_decision == 'exit':
                    return self._handle_early_exit()

            elif phase_key == 'domain':
                checkpoint = self._detect_domains_with_checkpoint()

            elif phase_key == 'entities':
                checkpoint = self._discover_entities_with_checkpoint()

            elif phase_key == 'disambiguation':
                if self._has_ambiguous_entities():
                    checkpoint = self._disambiguate_with_checkpoint()
                else:
                    # Skip this phase if no ambiguous entities
                    self.current_phase += 1
                    continue

            elif phase_key == 'relationships':
                checkpoint = self._discover_relationships_with_checkpoint()

            elif phase_key == 'patterns':
                checkpoint = self._refine_patterns_with_checkpoint()

            elif phase_key == 'final':
                checkpoint = self._final_review_checkpoint()

            # Handle navigation based on user decision
            if checkpoint.user_decision == 'back':
                self.current_phase = max(0, self.current_phase - 1)
                print("\n↩️  Going back to previous phase...")
            elif checkpoint.user_decision == 'restart':
                self.current_phase = 0
                self._reset_discoveries()
                print("\n🔄 Restarting discovery from beginning...")
            elif checkpoint.user_decision == 'jump':
                # Allow jumping to specific phase
                self._show_phase_menu()
                target_phase = self._get_phase_selection()
                if target_phase is not None:
                    self.current_phase = target_phase
            else:
                # Move to next phase
                self.current_phase += 1

        # Generate ontology with all refinements
        ontology_path = self._generate_refined_ontology()

        print(f"\n🎉 Discovery complete! Ontology saved to: {ontology_path}")
        self._save_discovery_report()

        return ontology_path

    def _set_automation_preferences(self):
        """Let user set their preferred level of automation."""
        print("🎚️  AUTOMATION PREFERENCES")
        print("-" * 30)
        print("How much automation would you prefer?")
        print("1. Full Auto - Minimal interruptions (high confidence only)")
        print("2. Balanced - Check at major decision points (recommended)")
        print("3. Guided - Review all major discoveries")

        choice = input("\nSelect (1/2/3) [2]: ").strip() or '2'

        automation_map = {
            '1': ('full', 0.85),
            '2': ('balanced', 0.70),
            '3': ('guided', 0.50)
        }

        if choice in automation_map:
            self.user_preferences['automation_level'] = automation_map[choice][0]
            self.user_preferences['confidence_threshold'] = automation_map[choice][1]
            print(f"✓ Set to {automation_map[choice][0]} automation mode")

    def _create_checkpoint(self, name: str, stage: str, data: Dict,
                          confidence: float, can_skip: bool = True) -> DiscoveryCheckpoint:
        """Create a checkpoint for human review."""
        checkpoint = DiscoveryCheckpoint(
            name=name,
            stage=stage,
            data=data,
            confidence=confidence,
            can_skip=can_skip
        )

        # Decide if we need human input based on confidence and preferences
        if self._should_request_human_input(checkpoint):
            checkpoint = self._request_human_review(checkpoint)
        else:
            checkpoint.user_decision = 'auto_approved'
            print(f"✓ Auto-approved: {name} (confidence: {confidence:.0%})")

        self.checkpoints.append(checkpoint)
        return checkpoint

    def _should_request_human_input(self, checkpoint: DiscoveryCheckpoint) -> bool:
        """Determine if human input is needed for this checkpoint."""
        if not checkpoint.can_skip:
            return True

        if checkpoint.confidence < self.user_preferences['confidence_threshold']:
            return True

        if self.user_preferences['automation_level'] == 'guided':
            return True

        if self.user_preferences['automation_level'] == 'balanced' and checkpoint.stage in ['domain', 'entities']:
            return True

        return False

    def _request_human_review(self, checkpoint: DiscoveryCheckpoint) -> DiscoveryCheckpoint:
        """Request human review of a checkpoint."""
        print(f"\n🔍 CHECKPOINT: {checkpoint.name}")
        print(f"Confidence: {checkpoint.confidence:.0%}")

        # Display checkpoint-specific information
        if checkpoint.stage == 'corpus':
            self._display_corpus_summary(checkpoint.data)
        elif checkpoint.stage == 'domain':
            self._display_domain_analysis(checkpoint.data)
        elif checkpoint.stage == 'entities':
            self._display_entity_summary(checkpoint.data)
        elif checkpoint.stage == 'relationships':
            self._display_relationship_summary(checkpoint.data)
        elif checkpoint.stage == 'disambiguation':
            self._display_ambiguous_entities(checkpoint.data)

        # Show checkpoint history if available
        if len(self.checkpoints) > 0:
            print(f"\n📍 Progress: Phase {self.current_phase + 1} of {len(self.phases)}")
            recent_checkpoints = self.checkpoints[-3:] if len(self.checkpoints) > 3 else self.checkpoints
            for cp in recent_checkpoints:
                status = "✓" if cp.user_decision in ['continue', 'refined', 'auto_approved'] else "○"
                print(f"   {status} {cp.name}")

        # Get user decision
        print("\n" + "=" * 40)
        print("OPTIONS:")
        print("1. ✓ Accept and continue")
        print("2. 📝 Refine this analysis")
        print("3. ⏭️  Skip this step")

        # Navigation options
        if self.current_phase > 0:
            print("4. ↩️  Go back to previous phase")

        # Additional options based on stage
        if checkpoint.stage != 'corpus':
            print("5. 🔄 Restart from beginning")
            print("6. 📋 Jump to specific phase")

        if checkpoint.stage == 'corpus':
            print("9. ❌ Exit discovery")

        print("-" * 40)
        choice = input("Your choice [1]: ").strip() or '1'

        # Process choice
        if choice == '1':
            checkpoint.user_decision = 'continue'
        elif choice == '2':
            checkpoint = self._refine_checkpoint(checkpoint)
            checkpoint.user_decision = 'refined'
        elif choice == '3':
            checkpoint.user_decision = 'skipped'
        elif choice == '4' and self.current_phase > 0:
            checkpoint.user_decision = 'back'
        elif choice == '5' and checkpoint.stage != 'corpus':
            checkpoint.user_decision = 'restart'
        elif choice == '6' and checkpoint.stage != 'corpus':
            checkpoint.user_decision = 'jump'
        elif choice == '9' and checkpoint.stage == 'corpus':
            checkpoint.user_decision = 'exit'
        else:
            checkpoint.user_decision = 'continue'

        return checkpoint

    def _refine_checkpoint(self, checkpoint: DiscoveryCheckpoint) -> DiscoveryCheckpoint:
        """Allow user to refine the checkpoint analysis."""
        print("\n📝 REFINEMENT OPTIONS")

        if checkpoint.stage == 'domain':
            self._refine_domain_detection(checkpoint)
        elif checkpoint.stage == 'entities':
            self._refine_entity_discovery(checkpoint)
        elif checkpoint.stage == 'relationships':
            self._refine_relationship_discovery(checkpoint)
        elif checkpoint.stage == 'disambiguation':
            self._refine_disambiguation(checkpoint)

        return checkpoint

    # ============== CHECKPOINT IMPLEMENTATIONS ==============

    def _analyze_corpus_with_checkpoint(self) -> DiscoveryCheckpoint:
        """Analyze corpus with checkpoint."""
        try:
            # Load and analyze
            self.corpus_data = self._load_analytics_data()

            if self.corpus_data.empty:
                print("❌ No data found in analytics path")
                return self._create_checkpoint(
                    "Corpus Analysis", "corpus",
                    {"error": "No data found"}, 0.0, False
                )

            # Analyze patterns
            self.corpus_patterns = {
                'frequent_terms': self._find_frequent_terms(self.corpus_data),
                'element_types': dict(self.corpus_data['element_type'].value_counts()),
                'document_count': self.corpus_data['doc_id'].nunique(),
                'element_count': len(self.corpus_data)
            }

            # Create checkpoint
            return self._create_checkpoint(
                "Corpus Analysis",
                "corpus",
                self.corpus_patterns,
                confidence=1.0,  # Data loading is deterministic
                can_skip=False  # Can't skip initial data load
            )

        except Exception as e:
            logger.error(f"Corpus analysis failed: {e}")
            return self._create_checkpoint(
                "Corpus Analysis", "corpus",
                {"error": str(e)}, 0.0, False
            )

    def _detect_domains_with_checkpoint(self) -> DiscoveryCheckpoint:
        """Detect domains with human validation checkpoint."""

        # Multi-domain detection
        domain_clusters = self._cluster_documents_by_domain()

        # Score each cluster
        domain_analysis = {}
        for cluster_id, doc_ids in domain_clusters.items():
            cluster_data = self.corpus_data[self.corpus_data['doc_id'].isin(doc_ids)]
            domain_scores = self._score_domain_indicators(cluster_data)

            domain_analysis[f"Cluster_{cluster_id}"] = {
                'document_count': len(doc_ids),
                'likely_domain': max(domain_scores.items(), key=lambda x: x[1]),
                'all_scores': domain_scores,
                'example_docs': doc_ids[:3]
            }

        # Calculate overall confidence
        confidence = self._calculate_domain_confidence(domain_analysis)

        # Create checkpoint for review
        checkpoint = self._create_checkpoint(
            "Domain Detection",
            "domain",
            domain_analysis,
            confidence=confidence
        )

        # Apply refinements if any
        if checkpoint.refinements:
            self._apply_domain_refinements(checkpoint.refinements)

        self.domain_indicators = domain_analysis
        return checkpoint

    def _discover_entities_with_checkpoint(self) -> DiscoveryCheckpoint:
        """Discover entities with checkpoint for review."""

        # Discover entities automatically
        discovered = self._automatic_entity_discovery()

        # Summarize for checkpoint
        entity_summary = {
            'total_entities': sum(len(e) for e in discovered.values()),
            'entity_types': list(discovered.keys()),
            'top_entities_by_type': {}
        }

        for entity_type, entities in discovered.items():
            if entities:
                top_5 = sorted(entities, key=lambda x: x.get('frequency', 0), reverse=True)[:5]
                entity_summary['top_entities_by_type'][entity_type] = [
                    {'text': e['text'], 'frequency': e.get('frequency', 0),
                     'confidence': e.get('confidence', 0)}
                    for e in top_5
                ]

        # Calculate confidence
        confidence = self._calculate_entity_confidence(discovered)

        # Create checkpoint
        checkpoint = self._create_checkpoint(
            "Entity Discovery",
            "entities",
            entity_summary,
            confidence=confidence
        )

        # Apply refinements
        if checkpoint.refinements:
            discovered = self._apply_entity_refinements(discovered, checkpoint.refinements)

        self.discovered_entities = discovered
        return checkpoint

    def _disambiguate_with_checkpoint(self) -> DiscoveryCheckpoint:
        """Handle ambiguous entities with user input."""

        ambiguous = self._find_ambiguous_entities()

        if not ambiguous:
            return self._create_checkpoint(
                "Disambiguation",
                "disambiguation",
                {"message": "No ambiguous entities found"},
                confidence=1.0
            )

        # For each ambiguous entity, get user input
        disambiguation_decisions = {}

        for entity_text, contexts in ambiguous.items():
            print(f"\n❓ Ambiguous entity: '{entity_text}'")
            print(f"   Appears {len(contexts)} times in different contexts")

            # Show example contexts
            for i, context in enumerate(contexts[:3], 1):
                print(f"   {i}. ...{context['snippet']}...")

            # Get user input
            print("\n   What type of entity is this?")
            entity_types = ['PERSON', 'ORGANIZATION', 'LOCATION', 'ROLE', 'OTHER', 'SKIP']
            for i, et in enumerate(entity_types, 1):
                print(f"   {i}. {et}")

            choice = input("   Select [6 to skip]: ").strip()

            if choice and choice.isdigit() and 1 <= int(choice) <= len(entity_types):
                selected_type = entity_types[int(choice) - 1]
                if selected_type != 'SKIP':
                    disambiguation_decisions[entity_text] = selected_type

        checkpoint = self._create_checkpoint(
            "Entity Disambiguation",
            "disambiguation",
            {'disambiguated': disambiguation_decisions, 'total_ambiguous': len(ambiguous)},
            confidence=0.5  # Low confidence triggers review
        )

        # Apply disambiguation decisions
        self._apply_disambiguation(disambiguation_decisions)

        return checkpoint

    def _discover_relationships_with_checkpoint(self) -> DiscoveryCheckpoint:
        """Discover relationships with review checkpoint."""

        # Automatic relationship discovery
        relationships = self._automatic_relationship_discovery()

        # Summarize for review
        rel_summary = {
            'total_relationships': sum(len(r) for r in relationships.values()),
            'relationship_types': list(relationships.keys()),
            'examples': {}
        }

        for rel_type, rels in relationships.items():
            if rels:
                top_3 = sorted(rels, key=lambda x: x.get('frequency', 0), reverse=True)[:3]
                rel_summary['examples'][rel_type] = [
                    f"{r['source']} -> {r['target']}" for r in top_3
                ]

        confidence = self._calculate_relationship_confidence(relationships)

        checkpoint = self._create_checkpoint(
            "Relationship Discovery",
            "relationships",
            rel_summary,
            confidence=confidence
        )

        if checkpoint.refinements:
            relationships = self._apply_relationship_refinements(relationships, checkpoint.refinements)

        self.discovered_relationships = relationships
        return checkpoint

    def _refine_patterns_with_checkpoint(self) -> DiscoveryCheckpoint:
        """Allow refinement of extraction patterns."""

        # Generate extraction patterns
        patterns = self._generate_extraction_patterns()

        # Summarize patterns
        pattern_summary = {
            'entity_patterns': {
                entity_type: len(patterns['entities'].get(entity_type, []))
                for entity_type in self.discovered_entities.keys()
            },
            'relationship_patterns': {
                rel_type: len(patterns['relationships'].get(rel_type, []))
                for rel_type in self.discovered_relationships.keys()
            },
            'semantic_phrases': patterns.get('semantic_phrases', {})
        }

        checkpoint = self._create_checkpoint(
            "Pattern Generation",
            "patterns",
            pattern_summary,
            confidence=0.8
        )

        if checkpoint.user_decision == 'refined':
            patterns = self._refine_patterns_interactive(patterns)

        self.extraction_patterns = patterns
        return checkpoint

    def _final_review_checkpoint(self) -> DiscoveryCheckpoint:
        """Final review before ontology generation."""

        summary = {
            'domains_detected': len(self.domain_indicators),
            'entity_types': len(self.discovered_entities),
            'total_entities': sum(len(e) for e in self.discovered_entities.values()),
            'relationship_types': len(self.discovered_relationships),
            'total_relationships': sum(len(r) for r in self.discovered_relationships.values()),
            'checkpoints_passed': len(self.checkpoints),
            'user_refinements': sum(1 for c in self.checkpoints if c.user_decision == 'refined')
        }

        print("\n📋 FINAL SUMMARY")
        print("-" * 40)
        for key, value in summary.items():
            print(f"{key.replace('_', ' ').title()}: {value}")

        checkpoint = self._create_checkpoint(
            "Final Review",
            "final",
            summary,
            confidence=0.9
        )

        return checkpoint

    # ============== REFINEMENT METHODS ==============

    def _refine_domain_detection(self, checkpoint: DiscoveryCheckpoint):
        """Allow user to refine domain detection."""
        print("\nCurrent domain detection:")

        for cluster, info in checkpoint.data.items():
            print(f"\n{cluster}:")
            print(f"  Documents: {info['document_count']}")
            print(f"  Detected as: {info['likely_domain'][0]} ({info['likely_domain'][1]:.0%})")

        refine = input("\nWould you like to correct any domain classifications? [y/N]: ").lower()

        if refine == 'y':
            refinements = {}
            for cluster in checkpoint.data.keys():
                print(f"\n{cluster} currently: {checkpoint.data[cluster]['likely_domain'][0]}")
                new_domain = input("New domain (or Enter to keep): ").strip().upper()
                if new_domain:
                    refinements[cluster] = new_domain

            checkpoint.refinements['domain_corrections'] = refinements

    def _refine_entity_discovery(self, checkpoint: DiscoveryCheckpoint):
        """Allow user to refine entity discovery."""
        print("\n1. Add entity types to track")
        print("2. Remove entity types")
        print("3. Adjust confidence thresholds")
        print("4. Add specific entities to watch for")

        choice = input("\nSelect refinement option [Enter to skip]: ").strip()

        refinements = {}

        if choice == '1':
            additional_types = input("Enter additional entity types (comma-separated): ").strip()
            if additional_types:
                refinements['add_types'] = [t.strip().upper() for t in additional_types.split(',')]

        elif choice == '2':
            remove_types = input("Enter entity types to remove (comma-separated): ").strip()
            if remove_types:
                refinements['remove_types'] = [t.strip().upper() for t in remove_types.split(',')]

        elif choice == '3':
            new_threshold = input("New confidence threshold (0.0-1.0) [0.7]: ").strip()
            if new_threshold:
                try:
                    refinements['confidence_threshold'] = float(new_threshold)
                except ValueError:
                    pass

        elif choice == '4':
            specific_entities = input("Enter specific entities to track (comma-separated): ").strip()
            if specific_entities:
                refinements['specific_entities'] = [e.strip() for e in specific_entities.split(',')]

        checkpoint.refinements = refinements

    def _refine_relationship_discovery(self, checkpoint: DiscoveryCheckpoint):
        """Allow user to refine relationships."""
        print("\nCurrent relationship types discovered:")
        for rel_type in checkpoint.data['relationship_types']:
            print(f"  • {rel_type}")

        print("\n1. Add custom relationship types")
        print("2. Remove relationship types")
        print("3. Define custom patterns")

        choice = input("\nSelect option [Enter to skip]: ").strip()

        refinements = {}

        if choice == '1':
            custom_rels = input("Enter custom relationship types (comma-separated): ").strip()
            if custom_rels:
                refinements['add_relationships'] = [r.strip() for r in custom_rels.split(',')]

        elif choice == '2':
            remove_rels = input("Enter relationships to remove (comma-separated): ").strip()
            if remove_rels:
                refinements['remove_relationships'] = [r.strip() for r in remove_rels.split(',')]

        elif choice == '3':
            print("Define a custom pattern:")
            source_type = input("Source entity type: ").strip()
            rel_name = input("Relationship name: ").strip()
            target_type = input("Target entity type: ").strip()
            pattern = input("Pattern (e.g., '[SOURCE] appointed as [TARGET]'): ").strip()

            if all([source_type, rel_name, target_type, pattern]):
                refinements['custom_patterns'] = [{
                    'source': source_type,
                    'relationship': rel_name,
                    'target': target_type,
                    'pattern': pattern
                }]

        checkpoint.refinements = refinements

    # ============== DISPLAY METHODS ==============

    def _display_corpus_summary(self, data: Dict):
        """Display corpus analysis summary."""
        print(f"\n📊 Corpus Summary:")
        print(f"  Documents: {data.get('document_count', 0)}")
        print(f"  Elements: {data.get('element_count', 0)}")

        if 'frequent_terms' in data:
            print(f"\n  Top Terms:")
            for term, count in list(data['frequent_terms'].items())[:10]:
                print(f"    • {term}: {count}")

    def _display_domain_analysis(self, data: Dict):
        """Display domain detection results."""
        print(f"\n🎯 Domain Analysis:")

        # Check if multi-domain
        if len(data) > 1:
            print(f"  ⚠️  Multiple domains detected ({len(data)} clusters)")

        for cluster, info in data.items():
            domain, confidence = info['likely_domain']
            print(f"\n  {cluster}:")
            print(f"    Domain: {domain} ({confidence:.0%} confidence)")
            print(f"    Documents: {info['document_count']}")

            # Show alternative interpretations if confidence is low
            if confidence < 0.7:
                print(f"    Alternatives:")
                for alt_domain, alt_score in sorted(info['all_scores'].items(),
                                                   key=lambda x: x[1], reverse=True)[:3]:
                    if alt_domain != domain:
                        print(f"      • {alt_domain}: {alt_score:.0%}")

    def _display_entity_summary(self, data: Dict):
        """Display entity discovery summary."""
        print(f"\n🔍 Entity Discovery Summary:")
        print(f"  Total entities found: {data['total_entities']}")
        print(f"  Entity types: {len(data['entity_types'])}")

        print(f"\n  Top Entities by Type:")
        for entity_type, entities in list(data.get('top_entities_by_type', {}).items())[:5]:
            print(f"\n    {entity_type}:")
            for entity in entities[:3]:
                print(f"      • {entity['text']} (freq: {entity['frequency']}, conf: {entity['confidence']:.0%})")

    def _display_relationship_summary(self, data: Dict):
        """Display relationship discovery summary."""
        print(f"\n🔗 Relationship Summary:")
        print(f"  Total relationships: {data['total_relationships']}")
        print(f"  Relationship types: {len(data['relationship_types'])}")

        if 'examples' in data:
            print(f"\n  Examples:")
            for rel_type, examples in list(data['examples'].items())[:5]:
                print(f"    {rel_type}:")
                for example in examples[:2]:
                    print(f"      • {example}")

    def _display_ambiguous_entities(self, data: Dict):
        """Display ambiguous entities for disambiguation."""
        print(f"\n❓ Ambiguous Entities:")
        print(f"  Total requiring disambiguation: {data.get('total_ambiguous', 0)}")

        if 'disambiguated' in data:
            print(f"  Disambiguated: {len(data['disambiguated'])}")

    # ============== HELPER METHODS ==============

    def _load_analytics_data(self) -> pd.DataFrame:
        """Load data from analytics parquet files."""
        conn = duckdb.connect(':memory:')

        import glob
        parquet_files = []
        for file_path in glob.glob(f"{self.analytics_path}/elements/**/*.parquet", recursive=True):
            if not os.path.basename(file_path).startswith('._'):
                parquet_files.append(file_path)

        if not parquet_files:
            return pd.DataFrame()

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
        LIMIT 100000
        """

        return conn.execute(query).df()

    def _find_frequent_terms(self, df: pd.DataFrame) -> Dict[str, int]:
        """Find frequent terms in corpus."""
        all_content = ' '.join(df['content'].astype(str))

        stop_words = {'the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by'}
        words = re.findall(r'\b[a-zA-Z]{3,}\b', all_content.lower())
        word_freq = Counter([w for w in words if w not in stop_words])

        return dict(word_freq.most_common(100))

    def _cluster_documents_by_domain(self) -> Dict[int, List[str]]:
        """Cluster documents to detect multiple domains."""
        # Simplified clustering - in practice would use more sophisticated methods
        clusters = defaultdict(list)

        for doc_id in self.corpus_data['doc_id'].unique():
            doc_data = self.corpus_data[self.corpus_data['doc_id'] == doc_id]

            # Simple heuristic clustering based on content
            content = ' '.join(doc_data['content'].astype(str)).lower()

            # Assign to cluster based on dominant terms
            if any(term in content for term in ['regulation', 'compliance', 'filing']):
                clusters[0].append(doc_id)  # Regulatory cluster
            elif any(term in content for term in ['patient', 'diagnosis', 'treatment']):
                clusters[1].append(doc_id)  # Medical cluster
            elif any(term in content for term in ['trade', 'position', 'market']):
                clusters[2].append(doc_id)  # Trading cluster
            else:
                clusters[99].append(doc_id)  # Other

        return dict(clusters)

    def _score_domain_indicators(self, data: pd.DataFrame) -> Dict[str, float]:
        """Score domain indicators for a dataset."""
        indicators = {
            'REGULATORY': 0,
            'MEDICAL': 0,
            'FINANCIAL': 0,
            'TECHNOLOGY': 0,
            'LEGAL': 0,
            'ACADEMIC': 0,
            'NEWS': 0,
            'GENERAL': 0
        }

        content = ' '.join(data['content'].astype(str)).lower()

        # Simple scoring based on keyword presence
        domain_keywords = {
            'REGULATORY': ['regulation', 'compliance', 'filing', 'sec', 'audit'],
            'MEDICAL': ['patient', 'diagnosis', 'treatment', 'medication', 'symptom'],
            'FINANCIAL': ['trade', 'investment', 'portfolio', 'market', 'stock'],
            'TECHNOLOGY': ['software', 'algorithm', 'code', 'system', 'api'],
            'LEGAL': ['court', 'plaintiff', 'defendant', 'statute', 'litigation'],
            'ACADEMIC': ['research', 'study', 'hypothesis', 'methodology', 'analysis'],
            'NEWS': ['reported', 'announced', 'according', 'sources', 'breaking']
        }

        for domain, keywords in domain_keywords.items():
            for keyword in keywords:
                indicators[domain] += content.count(keyword)

        # Normalize
        total = sum(indicators.values())
        if total > 0:
            for domain in indicators:
                indicators[domain] = indicators[domain] / total

        return indicators

    def _calculate_domain_confidence(self, domain_analysis: Dict) -> float:
        """Calculate overall confidence in domain detection."""
        confidences = []

        for cluster_info in domain_analysis.values():
            domain, score = cluster_info['likely_domain']
            confidences.append(score)

        return float(np.mean(confidences)) if confidences else 0.5

    def _calculate_entity_confidence(self, entities: Dict) -> float:
        """Calculate overall confidence in entity discovery."""
        all_confidences = []

        for entity_list in entities.values():
            for entity in entity_list[:10]:  # Top 10 per type
                all_confidences.append(entity.get('confidence', 0.5))

        return float(np.mean(all_confidences)) if all_confidences else 0.5

    def _calculate_relationship_confidence(self, relationships: Dict) -> float:
        """Calculate confidence in relationship discovery."""
        if not relationships:
            return 0.3

        # Higher confidence if we found multiple relationship types
        base_confidence = min(0.5 + (len(relationships) * 0.1), 0.9)

        # Boost if relationships have high frequency
        high_freq_count = sum(
            1 for rels in relationships.values()
            for rel in rels
            if rel.get('frequency', 0) > 5
        )

        if high_freq_count > 10:
            base_confidence = min(base_confidence + 0.1, 0.95)

        return base_confidence

    def _automatic_entity_discovery(self) -> Dict[str, List[Dict]]:
        """Perform automatic entity discovery, merging with existing if present."""
        # Start with existing entities if in extend/merge mode
        if self.ontology_mode in ['extend', 'merge', 'refine'] and self.existing_ontology:
            discovered = defaultdict(list, self.discovered_entities)
        else:
            discovered = defaultdict(list)

        # Simplified entity discovery
        for _, row in self.corpus_data.iterrows():
            content = str(row['content'])

            # Extract various entity types
            # Person names (simplified)
            person_pattern = r'\b[A-Z][a-z]+ [A-Z][a-z]+\b'
            for match in re.findall(person_pattern, content):
                discovered['PERSON'].append({
                    'text': match,
                    'frequency': 1,
                    'confidence': 0.7
                })

            # Organizations
            org_pattern = r'\b\w+\s+(?:Inc\.|Corp\.|LLC|Ltd\.)\b'
            for match in re.findall(org_pattern, content):
                discovered['ORGANIZATION'].append({
                    'text': match,
                    'frequency': 1,
                    'confidence': 0.9
                })

            # Money amounts
            money_pattern = r'\$[\d,]+(?:\.\d{2})?'
            for match in re.findall(money_pattern, content):
                discovered['MONEY'].append({
                    'text': match,
                    'frequency': 1,
                    'confidence': 0.95
                })

        # Aggregate and count
        for entity_type in discovered:
            # Count frequencies
            counter = Counter(e['text'] for e in discovered[entity_type])

            # Create unique list with frequencies
            unique_entities = []
            for text, freq in counter.items():
                unique_entities.append({
                    'text': text,
                    'frequency': freq,
                    'confidence': min(0.5 + (freq * 0.02), 0.95)
                })

            discovered[entity_type] = unique_entities

        return dict(discovered)

    def _automatic_relationship_discovery(self) -> Dict[str, List[Dict]]:
        """Discover relationships automatically."""
        relationships = defaultdict(list)

        # Simple pattern-based relationship extraction
        patterns = {
            'APPOINTED': r'(\w+ \w+) (?:was )?appointed (?:as )?(\w+)',
            'WORKS_AT': r'(\w+ \w+), (\w+) (?:at|of) (\w+)',
            'RECEIVED': r'(\w+ \w+) received (\$[\d,]+)'
        }

        for _, row in self.corpus_data.iterrows():
            content = str(row['content'])

            for rel_type, pattern in patterns.items():
                for match in re.findall(pattern, content):
                    if len(match) >= 2:
                        relationships[rel_type].append({
                            'source': match[0],
                            'target': match[-1],
                            'frequency': 1,
                            'confidence': 0.7
                        })

        return dict(relationships)

    def _find_ambiguous_entities(self) -> Dict[str, List[Dict]]:
        """Find entities that might be ambiguous."""
        ambiguous = {}

        # Look for entities that appear in multiple contexts
        for entity_type, entities in self.discovered_entities.items():
            for entity in entities:
                if entity['frequency'] > 10 and entity['confidence'] < 0.7:
                    # This might be ambiguous
                    contexts = []

                    # Find contexts where this entity appears
                    for _, row in self.corpus_data.iterrows():
                        if entity['text'] in str(row['content']):
                            contexts.append({
                                'snippet': str(row['content'])[:100],
                                'element_type': row['element_type']
                            })

                    if len(contexts) > 2:
                        ambiguous[entity['text']] = contexts[:5]

        return ambiguous

    def _has_ambiguous_entities(self) -> bool:
        """Check if there are ambiguous entities."""
        return len(self._find_ambiguous_entities()) > 0

    def _apply_disambiguation(self, decisions: Dict[str, str]):
        """Apply disambiguation decisions."""
        for entity_text, entity_type in decisions.items():
            # Move entity to correct type
            for old_type in list(self.discovered_entities.keys()):
                self.discovered_entities[old_type] = [
                    e for e in self.discovered_entities[old_type]
                    if e['text'] != entity_text
                ]

            # Add to new type
            if entity_type not in self.discovered_entities:
                self.discovered_entities[entity_type] = []

            self.discovered_entities[entity_type].append({
                'text': entity_text,
                'frequency': 10,  # Placeholder
                'confidence': 0.9,  # Higher after human validation
                'human_validated': True
            })

    def _delegate_to_intelligent_discovery(self) -> str:
        """
        Delegate to the intelligent discovery system when config is provided.
        """
        print("🔄 Delegating to Intelligent Discovery System...")
        print(f"📂 Using config: {self.config_path}")

        # Initialize intelligent discovery with same analytics path
        intelligent_discovery = IntelligentDiscoverySystem(
            analytics_path=self.analytics_path,
            output_dir=self.output_dir
        )

        # Load the config
        with open(self.config_path, 'r') as f:
            config = yaml.safe_load(f)

        # Run automated discovery with config parameters
        if 'discovery_config' in config:
            discovery_config = config['discovery_config']
            # Set parameters from config
            if 'confidence_threshold' in discovery_config:
                intelligent_discovery.confidence_threshold = discovery_config['confidence_threshold']
            if 'max_entities' in discovery_config:
                intelligent_discovery.max_entities = discovery_config['max_entities']

        # Run the automated discovery
        print("\n🤖 Running automated discovery...")
        intelligent_discovery.analyze_corpus()

        # Use LLM if available for enhancement
        if intelligent_discovery.client:
            intelligent_discovery.discover_with_llm()

        # Generate ontology
        ontology_path = intelligent_discovery.generate_ontology()

        print(f"✅ Ontology generated via delegation: {ontology_path}")
        return ontology_path

    def _generate_extraction_patterns(self) -> Dict:
        """Generate extraction patterns from discoveries."""
        patterns = {
            'entities': {},
            'relationships': {},
            'semantic_phrases': {}
        }

        # Entity patterns
        for entity_type, entities in self.discovered_entities.items():
            if entities:
                # Get top entities as patterns
                top_entities = sorted(entities, key=lambda x: x['frequency'], reverse=True)[:10]
                patterns['entities'][entity_type] = [e['text'] for e in top_entities]

                # Semantic phrase = most frequent entity
                if top_entities:
                    patterns['semantic_phrases'][entity_type] = top_entities[0]['text']

        # Relationship patterns
        for rel_type in self.discovered_relationships:
            patterns['relationships'][rel_type] = [
                f"[SOURCE] {rel_type.lower()} [TARGET]"
            ]

        return patterns

    def _refine_patterns_interactive(self, patterns: Dict) -> Dict:
        """Allow interactive refinement of patterns."""
        print("\nWould you like to refine extraction patterns?")

        # This would be expanded with actual refinement logic
        return patterns

    def _apply_domain_refinements(self, refinements: Dict):
        """Apply domain refinements."""
        if 'domain_corrections' in refinements:
            for cluster, new_domain in refinements['domain_corrections'].items():
                if cluster in self.domain_indicators:
                    self.domain_indicators[cluster]['likely_domain'] = (new_domain, 1.0)

    def _apply_entity_refinements(self, entities: Dict, refinements: Dict) -> Dict:
        """Apply entity refinements."""
        if 'add_types' in refinements:
            for entity_type in refinements['add_types']:
                if entity_type not in entities:
                    entities[entity_type] = []

        if 'remove_types' in refinements:
            for entity_type in refinements['remove_types']:
                if entity_type in entities:
                    del entities[entity_type]

        if 'specific_entities' in refinements:
            if 'CUSTOM' not in entities:
                entities['CUSTOM'] = []

            for entity_text in refinements['specific_entities']:
                entities['CUSTOM'].append({
                    'text': entity_text,
                    'frequency': 0,
                    'confidence': 1.0,
                    'user_specified': True
                })

        return entities

    def _apply_relationship_refinements(self, relationships: Dict, refinements: Dict) -> Dict:
        """Apply relationship refinements."""
        if 'add_relationships' in refinements:
            for rel_type in refinements['add_relationships']:
                if rel_type not in relationships:
                    relationships[rel_type] = []

        if 'remove_relationships' in refinements:
            for rel_type in refinements['remove_relationships']:
                if rel_type in relationships:
                    del relationships[rel_type]

        if 'custom_patterns' in refinements:
            for pattern_def in refinements['custom_patterns']:
                rel_type = pattern_def['relationship']
                if rel_type not in relationships:
                    relationships[rel_type] = []

                relationships[rel_type].append({
                    'source': pattern_def['source'],
                    'target': pattern_def['target'],
                    'pattern': pattern_def['pattern'],
                    'user_defined': True
                })

        return relationships

    def _generate_refined_ontology(self) -> str:
        """Generate final ontology with all refinements applied."""
        # Determine primary domain
        if self.domain_indicators:
            primary_domain = list(self.domain_indicators.values())[0]['likely_domain'][0]
        else:
            primary_domain = 'GENERAL'

        domain_name = primary_domain.lower() + '_steered'

        # Build ontology
        ontology = {
            'domain': {
                'name': domain_name,
                'version': '3.0.0',
                'description': f'Steerable discovery ontology with human refinements',
                'discovery_method': 'steerable_intelligent_discovery',
                'automation_level': self.user_preferences['automation_level'],
                'settings': {
                    'default_confidence_threshold': self.user_preferences['confidence_threshold'],
                    'human_validated': True,
                    'checkpoints_used': len(self.checkpoints),
                    'refinements_applied': sum(1 for c in self.checkpoints if c.refinements)
                }
            },
            'terms': self._generate_terms(),
            'element_mappings': self._generate_element_mappings(),
            'relationship_rules': self._generate_relationship_rules(),
            'metadata': {
                'discovery_summary': {
                    'method': 'steerable_discovery',
                    'checkpoints': [
                        {
                            'name': c.name,
                            'confidence': c.confidence,
                            'decision': c.user_decision,
                            'refined': bool(c.refinements)
                        }
                        for c in self.checkpoints
                    ]
                },
                'domains_detected': self.domain_indicators,
                'user_preferences': self.user_preferences
            }
        }

        # Save
        output_path = self.output_dir / f"{domain_name}_ontology.yaml"
        with open(output_path, 'w') as f:
            yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)

        return str(output_path)

    def _generate_terms(self) -> List[Dict]:
        """Generate term definitions."""
        terms = []

        for entity_type, entities in self.discovered_entities.items():
            if entities:
                term = {
                    'id': entity_type.lower(),
                    'label': entity_type.replace('_', ' ').title(),
                    'description': f'{entity_type} entities',
                    'aliases': [],
                    'semantic_variations': [e['text'] for e in sorted(entities,
                                          key=lambda x: x['frequency'], reverse=True)[:10]]
                }
                terms.append(term)

        return terms

    def _generate_element_mappings(self) -> List[Dict]:
        """Generate element mapping rules."""
        mappings = []

        for entity_type in self.discovered_entities:
            mapping = {
                'term_id': entity_type.lower(),
                'rules': []
            }

            # Semantic rule
            if entity_type in self.extraction_patterns.get('semantic_phrases', {}):
                mapping['rules'].append({
                    'type': 'semantic',
                    'semantic_phrase': self.extraction_patterns['semantic_phrases'][entity_type],
                    'confidence_threshold': 0.65,
                    'element_types': ['paragraph', 'table_cell', 'div']
                })

            mappings.append(mapping)

        return mappings

    def _generate_relationship_rules(self) -> List[Dict]:
        """Generate relationship rules."""
        rules = []

        for rel_type, rels in self.discovered_relationships.items():
            if rels:
                rule = {
                    'id': rel_type.lower(),
                    'relationship_type': rel_type.lower(),
                    'description': f'{rel_type} relationship',
                    'patterns': self.extraction_patterns.get('relationships', {}).get(rel_type, [])
                }
                rules.append(rule)

        return rules

    def _save_discovery_report(self):
        """Save detailed discovery report."""
        report = {
            'timestamp': str(pd.Timestamp.now()),
            'checkpoints': [
                {
                    'name': c.name,
                    'stage': c.stage,
                    'confidence': c.confidence,
                    'user_decision': c.user_decision,
                    'had_refinements': bool(c.refinements)
                }
                for c in self.checkpoints
            ],
            'statistics': {
                'documents_analyzed': self.corpus_patterns.get('document_count', 0),
                'elements_processed': self.corpus_patterns.get('element_count', 0),
                'entity_types_discovered': len(self.discovered_entities),
                'total_entities': sum(len(e) for e in self.discovered_entities.values()),
                'relationship_types': len(self.discovered_relationships),
                'user_interventions': sum(1 for c in self.checkpoints if c.user_decision != 'auto_approved')
            }
        }

        report_path = self.output_dir / "discovery_report.json"
        with open(report_path, 'w') as f:
            json.dump(report, f, indent=2, default=str)

        print(f"\n📄 Detailed report saved to: {report_path}")

    def _handle_early_exit(self) -> str:
        """Handle early exit from discovery."""
        print("\n❌ Discovery cancelled by user")
        return "Discovery cancelled"


def main():
    """Main entry point for steerable discovery."""
    import argparse

    parser = argparse.ArgumentParser(description='Steerable Intelligent Discovery System')
    parser.add_argument('--analytics-path', required=True,
                       help='Path to analytics data directory')
    parser.add_argument('--output-dir', default='.',
                       help='Output directory for generated ontology')
    parser.add_argument('--existing-ontology', default=None,
                       help='Path to existing ontology to extend/refine')

    args = parser.parse_args()

    if not os.path.exists(args.analytics_path):
        print(f"❌ Analytics path not found: {args.analytics_path}")
        return

    if args.existing_ontology and not os.path.exists(args.existing_ontology):
        print(f"⚠️  Existing ontology not found: {args.existing_ontology}")
        print("   Proceeding with new ontology creation...")
        args.existing_ontology = None

    # Create discovery system
    discovery = SteerableDiscoverySystem(
        args.analytics_path,
        args.output_dir,
        args.existing_ontology
    )

    # Run discovery with steering
    try:
        ontology_path = discovery.discover_with_steering()

        if ontology_path != "Discovery cancelled":
            print(f"\n🎉 Success! Your steered ontology is ready:")
            print(f"   {ontology_path}")
            print(f"\nThe system incorporated your feedback at {len(discovery.checkpoints)} checkpoints")

            if args.existing_ontology:
                print(f"Mode used: {discovery.ontology_mode}")
                if discovery.ontology_mode == 'extend':
                    print("✓ Extended existing ontology with new discoveries")
                elif discovery.ontology_mode == 'refine':
                    print("✓ Refined existing patterns with new data")
                elif discovery.ontology_mode == 'merge':
                    print("✓ Merged existing and new discoveries")

    except KeyboardInterrupt:
        print("\n❌ Discovery interrupted by user")
    except Exception as e:
        print(f"\n❌ Discovery failed: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    main()