#!/usr/bin/env python3
"""
Generate ontology tables from the enhanced discovered ontology YAML configuration.
This script creates the actual ontology data structures used by the Go-Doc-Go system.
"""

import yaml
import json
import sys
from pathlib import Path
from typing import Dict, Any, List

# Add src to path to import Go-Doc-Go modules
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go.domain.ontology import DomainOntology


def load_ontology_config(yaml_path: str) -> Dict[str, Any]:
    """Load ontology configuration from YAML file."""
    with open(yaml_path, 'r') as f:
        return yaml.safe_load(f)


def create_ontology_tables(config: Dict[str, Any]) -> Dict[str, Any]:
    """
    Create structured ontology tables from the configuration.

    Returns:
        Dictionary containing organized ontology tables
    """

    # Load into Go-Doc-Go ontology structure
    ontology = DomainOntology.from_dict(config)

    # Validate the ontology
    validation_issues = ontology.validate()
    if validation_issues:
        print(f"⚠️  Ontology validation issues found:")
        for issue in validation_issues:
            print(f"   - {issue}")
        print()

    # Generate organized tables
    tables = {
        'domain_info': {
            'name': ontology.name,
            'version': ontology.version,
            'description': ontology.description,
            'settings': {
                'default_confidence_threshold': ontology.settings.default_confidence_threshold,
                'max_relationships_per_pair': ontology.settings.max_relationships_per_pair,
                'enable_transitive_inference': ontology.settings.enable_transitive_inference
            }
        },

        'terms_table': [],
        'mapping_rules_table': [],
        'relationship_rules_table': [],
        'statistics': {
            'total_terms': len(ontology.terms),
            'total_mapping_rules': sum(len(mapping.rules) for mapping in ontology.element_mappings),
            'total_relationship_rules': len(ontology.relationship_rules)
        }
    }

    # Terms table
    for term in ontology.terms:
        term_data = {
            'term_id': term.id,
            'label': term.label,
            'description': term.description,
            'aliases': term.aliases or [],
            'all_names': term.get_all_names()
        }
        tables['terms_table'].append(term_data)

    # Mapping rules table (flattened)
    for mapping in ontology.element_mappings:
        for rule in mapping.rules:
            rule_data = {
                'term_id': mapping.term_id,
                'rule_type': rule.type.value,
                'element_types': rule.element_types or ['*'],
                'pattern': rule.pattern,
                'semantic_phrase': rule.semantic_phrase,
                'keywords': rule.keywords,
                'confidence_threshold': rule.confidence_threshold,
                'case_sensitive': rule.case_sensitive,
                'word_boundary': rule.word_boundary
            }
            tables['mapping_rules_table'].append(rule_data)

    # Relationship rules table
    for rule in ontology.relationship_rules:
        rule_data = {
            'rule_id': rule.id,
            'relationship_type': rule.relationship_type,
            'description': rule.description,
            'source_term_id': rule.source.term_id,
            'source_semantic_phrase': rule.source.semantic_phrase,
            'source_confidence_threshold': rule.source.confidence_threshold,
            'target_term_id': rule.target.term_id,
            'target_semantic_phrase': rule.target.semantic_phrase,
            'target_confidence_threshold': rule.target.confidence_threshold,
            'confidence_minimum': rule.confidence.minimum,
            'confidence_calculation': rule.confidence.calculation.value,
            'hierarchy_level': rule.constraints.hierarchy_level if rule.constraints else None,
            'direction': rule.constraints.direction.value if rule.constraints else 'any',
            'bidirectional': rule.bidirectional
        }
        tables['relationship_rules_table'].append(rule_data)

    return tables


def print_ontology_summary(tables: Dict[str, Any]):
    """Print a summary of the generated ontology tables."""

    domain = tables['domain_info']
    stats = tables['statistics']

    print("🎯 ONTOLOGY TABLES GENERATED")
    print("=" * 60)
    print(f"Domain: {domain['name']} v{domain['version']}")
    print(f"Description: {domain['description']}")
    print()

    print("📊 STATISTICS")
    print("-" * 30)
    print(f"Terms: {stats['total_terms']}")
    print(f"Mapping Rules: {stats['total_mapping_rules']}")
    print(f"Relationship Rules: {stats['total_relationship_rules']}")
    print()

    print("🏷️  TERMS TABLE")
    print("-" * 30)
    for term in tables['terms_table']:
        aliases_str = f" (aliases: {', '.join(term['aliases'])})" if term['aliases'] else ""
        print(f"• {term['term_id']}: {term['label']}{aliases_str}")
    print()

    print("🔗 MAPPING RULES SUMMARY")
    print("-" * 30)
    rule_types = {}
    for rule in tables['mapping_rules_table']:
        rule_type = rule['rule_type']
        rule_types[rule_type] = rule_types.get(rule_type, 0) + 1

    for rule_type, count in rule_types.items():
        print(f"• {rule_type}: {count} rules")
    print()

    print("🔗 RELATIONSHIP RULES SUMMARY")
    print("-" * 30)
    rel_types = {}
    for rule in tables['relationship_rules_table']:
        rel_type = rule['relationship_type']
        rel_types[rel_type] = rel_types.get(rel_type, 0) + 1

    for rel_type, count in rel_types.items():
        print(f"• {rel_type}: {count} rules")


def save_tables_to_files(tables: Dict[str, Any], output_dir: str = "."):
    """Save ontology tables to separate files."""

    output_path = Path(output_dir)
    output_path.mkdir(exist_ok=True)

    # Save complete tables as JSON
    with open(output_path / "ontology_tables.json", 'w') as f:
        json.dump(tables, f, indent=2, default=str)

    # Save individual tables as JSON
    for table_name, table_data in tables.items():
        if table_name != 'statistics':  # Skip statistics for individual files
            with open(output_path / f"{table_name}.json", 'w') as f:
                json.dump(table_data, f, indent=2, default=str)

    # Save as YAML for readability
    with open(output_path / "ontology_tables.yaml", 'w') as f:
        yaml.dump(tables, f, default_flow_style=False, sort_keys=False)

    print("💾 FILES SAVED")
    print("-" * 30)
    print(f"Complete tables: {output_path}/ontology_tables.json")
    print(f"Complete tables: {output_path}/ontology_tables.yaml")
    print(f"Domain info: {output_path}/domain_info.json")
    print(f"Terms: {output_path}/terms_table.json")
    print(f"Mapping rules: {output_path}/mapping_rules_table.json")
    print(f"Relationship rules: {output_path}/relationship_rules_table.json")


def main():
    """Main function to generate ontology tables."""

    # Input ontology YAML file
    ontology_file = "enhanced_discovered_ontology.yaml"

    if not Path(ontology_file).exists():
        print(f"❌ Ontology file not found: {ontology_file}")
        sys.exit(1)

    print(f"📖 Loading ontology from: {ontology_file}")

    try:
        # Load configuration
        config = load_ontology_config(ontology_file)

        # Generate tables
        tables = create_ontology_tables(config)

        # Print summary
        print_ontology_summary(tables)

        # Save to files
        save_tables_to_files(tables)

        print("\n✅ Ontology tables generated successfully!")

        # Show sample data from each table
        print("\n📋 SAMPLE DATA")
        print("-" * 30)

        # Sample term
        if tables['terms_table']:
            sample_term = tables['terms_table'][0]
            print(f"Sample term: {sample_term}")

        # Sample mapping rule
        if tables['mapping_rules_table']:
            sample_rule = tables['mapping_rules_table'][0]
            print(f"Sample mapping rule: {sample_rule}")

        # Sample relationship rule
        if tables['relationship_rules_table']:
            sample_rel = tables['relationship_rules_table'][0]
            print(f"Sample relationship rule: {sample_rel}")

    except Exception as e:
        print(f"❌ Error generating ontology tables: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()