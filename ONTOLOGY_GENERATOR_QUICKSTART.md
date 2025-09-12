# 🎯 Ontology Generator Quick Start

The ontology generator CLI is now ready to use! It will automatically use Anthropic Claude since your `ANTHROPIC_API_KEY` is configured.

## Running the Generator

### Method 1: Direct Python Module (Recommended)
```bash
cd /Users/kennethstott/PycharmProjects/doculyzer
PYTHONPATH=src python -m go_doc_go.cli.ontology_generator
```

### Method 2: Using the Shell Script
```bash
cd /Users/kennethstott/PycharmProjects/doculyzer
./scripts/generate-ontology.sh
```

## Quick Examples

### 1. Interactive Mode (Default)
Start an interactive interview to create your ontology:
```bash
PYTHONPATH=src python -m go_doc_go.cli.ontology_generator
```

### 2. Start from a Template
Use a pre-built template as your starting point:
```bash
PYTHONPATH=src python -m go_doc_go.cli.ontology_generator --template financial
```

Available templates:
- `financial` - For financial documents (earnings calls, reports)
- `legal` - For legal documents (contracts, cases)
- `medical` - For medical documents (patient records, studies)
- `technical` - For technical documentation (APIs, code docs)

### 3. Preview Without Saving
Test the generator without creating files:
```bash
PYTHONPATH=src python -m go_doc_go.cli.ontology_generator --dry-run
```

### 4. Save to Specific File
```bash
PYTHONPATH=src python -m go_doc_go.cli.ontology_generator --output my_domain_ontology.yaml
```

## What Happens During the Interview

The tool will guide you through 5 phases:

1. **Domain Understanding** - What type of documents are you analyzing?
2. **Term Definition** - What are the key terms and their synonyms?
3. **Entity Extraction** - What entities should be extracted?
4. **Relationship Mapping** - How do entities relate to each other?
5. **Refinement** - Review and adjust your ontology

The AI will provide intelligent suggestions based on your domain!

## Example Session

```bash
$ PYTHONPATH=src python -m go_doc_go.cli.ontology_generator --template financial

🎯 Welcome to the Go-Doc-Go Ontology Generator!
============================================================
I'll help you create a domain ontology for document analysis.
Let's start with some questions about your domain...

📚 Phase 1: Domain Understanding
----------------------------------------
What domain or industry are you working with? [financial]: 
What specific document types will you analyze? earnings_calls, 10-K
...
```

## Output

The generator creates a YAML file that can be used with Go-Doc-Go's domain extraction system. Example:

```yaml
name: financial_ontology
version: 1.0.0
terms:
  - term: revenue
    synonyms: [income, sales]
element_entity_mappings:
  - entity_type: company
    extraction_rules:
      - type: regex_pattern
        pattern: '\b[A-Z]\w+\s+(Inc|Corp|LLC)\b'
```

## Notes

- ✅ Your `ANTHROPIC_API_KEY` is configured - Claude will be used automatically
- The tool uses Claude to provide intelligent suggestions
- All generated ontologies are compatible with Go-Doc-Go's extraction system
- You can validate ontologies against sample documents using `--validate-with`

## Need Help?

- Run with `--help` to see all options
- Check `docs/ontology-generator.md` for detailed documentation
- Test with mock provider: `--llm-provider mock`