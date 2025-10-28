#!/usr/bin/env python3
"""
Migrate 36 example catalog files from examples/ontologies/ to catalogs/
Updates format to remove 'type' field, add parent_type and w_category fields, rename filter fields
"""

import os
import yaml
import sys
from pathlib import Path

# Mapping of domain entity types to global parent types
PARENT_TYPE_MAPPINGS = {
    # Product/Service entities
    'drug_compound': 'global.product',
    'drug_product': 'global.product',
    'software_product': 'global.product',
    'financial_product': 'global.product',
    'service': 'global.product',
    'component': 'global.product',

    # Document entities
    'clinical_trial': 'global.document',
    'regulatory_submission': 'global.document',
    'patent': 'global.document',
    'research_paper': 'global.document',
    'contract': 'global.document',
    'policy': 'global.document',
    'specification': 'global.document',
    'report': 'global.document',

    # Organization entities
    'regulatory_agency': 'global.organization',
    'manufacturer': 'global.organization',
    'sponsor': 'global.organization',
    'university': 'global.organization',
    'department': 'global.organization',
    'vendor': 'global.organization',
    'supplier': 'global.organization',
    'customer': 'global.organization',
    'competitor': 'global.organization',

    # Person entities
    'investigator': 'global.person',
    'researcher': 'global.person',
    'employee': 'global.person',
    'student': 'global.person',
    'faculty': 'global.person',
    'executive': 'global.person',
    'manager': 'global.person',
    'developer': 'global.person',
    'analyst': 'global.person',

    # Concept entities
    'indication': 'global.concept',
    'adverse_event': 'global.concept',
    'biomarker': 'global.concept',
    'methodology': 'global.concept',
    'algorithm': 'global.concept',
    'framework': 'global.concept',
    'theory': 'global.concept',
    'principle': 'global.concept',
    'standard': 'global.concept',
    'metric': 'global.concept',

    # Event entities
    'conference': 'global.event',
    'meeting': 'global.event',
    'milestone': 'global.event',
    'deadline': 'global.event',
    'release': 'global.event',

    # Process entities
    'pharmacokinetics': 'global.process',
    'pharmacodynamics': 'global.process',
    'formulation': 'global.process',
    'methodology': 'global.process',
    'workflow': 'global.process',
    'procedure': 'global.process',

    # Identifier entities
    'quality_specification': 'global.identifier',
    'batch_lot': 'global.identifier',
    'clinical_endpoint': 'global.identifier',
    'requirement': 'global.identifier',
    'objective': 'global.identifier',
    'deliverable': 'global.identifier',

    # Location entities
    'facility': 'global.location',
    'campus': 'global.location',
    'office': 'global.location',
    'laboratory': 'global.location',
    'site': 'global.location',
}

# W-Category mappings based on entity semantics
W_CATEGORY_MAPPINGS = {
    # WHO - persons and organizations
    'investigator': 'who',
    'researcher': 'who',
    'employee': 'who',
    'student': 'who',
    'faculty': 'who',
    'executive': 'who',
    'manager': 'who',
    'developer': 'who',
    'analyst': 'who',
    'regulatory_agency': 'who',
    'manufacturer': 'who',
    'sponsor': 'who',
    'university': 'who',
    'department': 'who',
    'vendor': 'who',
    'supplier': 'who',
    'customer': 'who',
    'competitor': 'who',

    # WHAT - products, documents, concepts, identifiers, roles
    'drug_compound': 'what',
    'drug_product': 'what',
    'software_product': 'what',
    'financial_product': 'what',
    'service': 'what',
    'component': 'what',
    'clinical_trial': 'what',
    'regulatory_submission': 'what',
    'patent': 'what',
    'research_paper': 'what',
    'contract': 'what',
    'policy': 'what',
    'specification': 'what',
    'report': 'what',
    'indication': 'what',
    'adverse_event': 'what',
    'biomarker': 'what',
    'methodology': 'what',
    'algorithm': 'what',
    'framework': 'what',
    'theory': 'what',
    'principle': 'what',
    'standard': 'what',
    'metric': 'what',
    'quality_specification': 'what',
    'batch_lot': 'what',
    'clinical_endpoint': 'what',
    'requirement': 'what',
    'objective': 'what',
    'deliverable': 'what',

    # WHERE - locations and facilities
    'facility': 'where',
    'campus': 'where',
    'office': 'where',
    'laboratory': 'where',
    'site': 'where',

    # WHEN - events and time-related
    'conference': 'when',
    'meeting': 'when',
    'milestone': 'when',
    'deadline': 'when',
    'release': 'when',

    # HOW - processes and methods
    'pharmacokinetics': 'how',
    'pharmacodynamics': 'how',
    'formulation': 'how',
    'workflow': 'how',
    'procedure': 'how',
}

def migrate_extraction_rules(rules):
    """Remove 'type' field and rename filter fields in extraction rules"""
    if not rules:
        return []

    migrated = []
    for rule in rules:
        new_rule = {}

        # Copy all fields except 'type'
        for key, value in rule.items():
            if key == 'type':
                continue  # Skip 'type' field
            elif key == 'proximity_filter':
                new_rule['proximity'] = value
            elif key == 'semantic_filter':
                new_rule['semantic'] = value
            elif key == 'dictionary_filter':
                new_rule['dictionary'] = value
            else:
                new_rule[key] = value

        if new_rule:  # Only add non-empty rules
            migrated.append(new_rule)

    return migrated

def migrate_entity_type(entity, domain):
    """Add parent_type and w_category to entity type"""
    entity_type = entity.get('entity_type', '')

    # Add parent_type
    if 'parent_type' not in entity:
        parent = PARENT_TYPE_MAPPINGS.get(entity_type, '')
        if parent:
            entity['parent_type'] = parent
        else:
            # If no mapping found, leave empty for manual review
            entity['parent_type'] = ''

    # Add w_category
    if 'w_category' not in entity:
        w_cat = W_CATEGORY_MAPPINGS.get(entity_type, 'what')  # Default to 'what'
        entity['w_category'] = w_cat

    # Add domain if missing
    if 'domain' not in entity:
        entity['domain'] = domain

    # Migrate extraction rules
    if 'sample_rules' in entity:
        entity['sample_rules'] = migrate_extraction_rules(entity['sample_rules'])

    return entity

def migrate_relationship_rules(rules):
    """Remove 'type' field from relationship extraction patterns"""
    if not rules:
        return []

    migrated = []
    for rule in rules:
        new_rule = rule.copy()

        # Migrate extraction patterns
        if 'sample_rules' in new_rule:
            new_rule['sample_rules'] = migrate_extraction_rules(new_rule['sample_rules'])

        # Rename fields for consistency
        if 'source_entity' in new_rule:
            new_rule['source_type'] = new_rule.pop('source_entity')
        if 'target_entity' in new_rule:
            new_rule['target_type'] = new_rule.pop('target_entity')

        migrated.append(new_rule)

    return migrated

def migrate_catalog_file(input_path, output_path):
    """Migrate a single catalog file"""
    print(f"Migrating: {input_path}")

    with open(input_path, 'r') as f:
        catalog = yaml.safe_load(f)

    domain = catalog.get('domain', '')

    # Migrate entity types
    if 'entity_types' in catalog:
        catalog['entity_types'] = [
            migrate_entity_type(entity, domain)
            for entity in catalog['entity_types']
        ]

    # Migrate relationship types
    if 'relationship_types' in catalog:
        catalog['relationship_types'] = migrate_relationship_rules(catalog['relationship_types'])

    # Write migrated catalog
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, 'w') as f:
        yaml.dump(catalog, f, default_flow_style=False, sort_keys=False, width=120)

    print(f"  -> {output_path}")

def main():
    project_root = Path(__file__).parent
    examples_dir = project_root / 'examples' / 'ontologies'
    catalogs_dir = project_root / 'catalogs'

    if not examples_dir.exists():
        print(f"ERROR: {examples_dir} not found")
        sys.exit(1)

    # Find all YAML files in examples/ontologies/
    yaml_files = list(examples_dir.rglob('*.yaml')) + list(examples_dir.rglob('*.yml'))

    print(f"Found {len(yaml_files)} catalog files to migrate\n")

    for input_file in yaml_files:
        # Calculate relative path from examples/ontologies/
        rel_path = input_file.relative_to(examples_dir)

        # Output to catalogs/ maintaining directory structure
        output_file = catalogs_dir / rel_path

        migrate_catalog_file(input_file, output_file)

    print(f"\n✓ Migrated {len(yaml_files)} catalog files")
    print(f"  From: {examples_dir}")
    print(f"  To:   {catalogs_dir}")

if __name__ == '__main__':
    main()
