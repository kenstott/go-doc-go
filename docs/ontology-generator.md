# Ontology Generator CLI

The Ontology Generator is an interactive command-line tool that helps you create domain ontologies for document analysis through an LLM-guided interview process.

## Features

- **Interactive Interview**: Step-by-step guided process to define your ontology
- **LLM Integration**: Intelligent suggestions based on your domain and use case
- **Multiple LLM Providers**: Support for Anthropic Claude, OpenAI GPT, and local models via Ollama
- **Template Support**: Start from pre-built templates for common domains
- **Validation**: Test your ontology against sample documents
- **Export Formats**: Generate YAML or JSON output

## Installation

### Required Dependencies

```bash
# For Anthropic Claude (recommended if you have ANTHROPIC_API_KEY)
pip install anthropic

# For OpenAI GPT
pip install openai

# Core dependencies are already included in requirements.txt
```

## Usage

### Basic Usage

Start an interactive ontology creation session:

```bash
python -m go_doc_go.cli.ontology_generator
```

### Using Templates

Start from a pre-built template:

```bash
# Available templates: financial, legal, medical, technical
python -m go_doc_go.cli.ontology_generator --template financial
```

### Specify LLM Provider

Choose a specific LLM provider:

```bash
# Use Anthropic Claude (default if ANTHROPIC_API_KEY is set)
python -m go_doc_go.cli.ontology_generator --llm-provider anthropic

# Use OpenAI GPT
python -m go_doc_go.cli.ontology_generator --llm-provider openai --model gpt-4

# Use local Ollama
python -m go_doc_go.cli.ontology_generator --llm-provider ollama --model llama2
```

### Validate with Sample Documents

Test your ontology against real documents:

```bash
python -m go_doc_go.cli.ontology_generator \
  --validate-with samples/ \
  --output my_ontology.yaml
```

### Non-Interactive Mode

Generate from a configuration file:

```bash
python -m go_doc_go.cli.ontology_generator \
  --non-interactive \
  --config ontology_config.yaml \
  --output generated_ontology.yaml
```

## Interview Process

The interactive interview guides you through five phases:

### Phase 1: Domain Understanding
- Identify your domain/industry
- Specify document types you'll analyze
- Define key concepts to extract

### Phase 2: Term Definition
- Define important terms and their synonyms
- Get AI suggestions based on your domain
- Group related terms

### Phase 3: Entity Extraction
- Define entity types to extract
- Specify extraction rules (metadata, regex, keywords)
- Map entities to document elements

### Phase 4: Relationship Mapping
- Define how entities relate to each other
- Specify relationship types
- Set confidence thresholds

### Phase 5: Refinement
- Review and refine your ontology
- Add derived entities
- Validate the structure

## Configuration

### Environment Variables

```bash
# For Anthropic Claude (prioritized if available)
export ANTHROPIC_API_KEY="your-api-key"

# For OpenAI GPT
export OPENAI_API_KEY="your-api-key"
```

### Available Templates

- **financial**: Companies, people, monetary amounts, dates
- **legal**: Parties, case numbers, statutes, obligations
- **medical**: Patients, conditions, medications, providers
- **technical**: Functions, APIs, error codes, configurations

## Example Session

```bash
$ python -m go_doc_go.cli.ontology_generator --template financial

🎯 Welcome to the Go-Doc-Go Ontology Generator!
============================================================
I'll help you create a domain ontology for document analysis.
Let's start with some questions about your domain...

📚 Phase 1: Domain Understanding
----------------------------------------
What domain or industry are you working with? financial

🤖 Assistant: Financial documents typically include earnings calls, 
annual reports, financial statements, and analyst reports...

What specific document types will you analyze? earnings_calls, 10-K

🎯 What are the key concepts you want to extract?
List key concepts (comma-separated): revenue, profit, growth, guidance

📝 Phase 2: Term Definition
----------------------------------------
🤖 Suggested terms:
[
  {"term": "revenue", "synonyms": ["income", "sales", "turnover"], 
   "description": "Money generated from business operations"},
  {"term": "profit", "synonyms": ["earnings", "net income"], 
   "description": "Revenue minus expenses"}
]

(a)dd term, (m)odify term, (d)elete term, or (c)ontinue: c

[... continues through all phases ...]

✅ Ontology saved to: financial_ontology.yaml
📊 Summary:
  - Terms: 15
  - Element Mappings: 8
  - Entity Relationships: 12
  - Derived Entities: 2
```

## Output Format

The generator creates a YAML ontology file compatible with Go-Doc-Go's domain extraction system:

```yaml
name: financial_ontology
version: 1.0.0
description: Ontology for financial domain
metadata:
  domain: financial
  document_types: [earnings_call, 10-K]
  key_concepts: [revenue, profit, growth]

terms:
  - term: revenue
    synonyms: [income, sales, turnover]
    description: Money generated from operations

element_entity_mappings:
  - entity_type: company
    element_types: [paragraph, heading]
    extraction_rules:
      - type: regex_pattern
        pattern: '\b[A-Z]\w+\s+(Inc|Corp|LLC)\b'
        confidence: 0.8

entity_relationship_rules:
  - name: company_revenue
    source_entity_type: company
    target_entity_type: monetary_amount
    relationship_type: HAS_REVENUE
    confidence_threshold: 0.7
```

## Tips for Creating Effective Ontologies

1. **Start Simple**: Begin with core entities and relationships, then add complexity
2. **Use Examples**: Provide sample content during the interview for better suggestions
3. **Test Incrementally**: Validate against sample documents as you build
4. **Leverage Templates**: Start from a template and customize for your needs
5. **Be Specific**: Use precise regex patterns and metadata paths
6. **Set Appropriate Confidence**: Higher confidence for exact matches, lower for fuzzy matches

## Troubleshooting

### No LLM Provider Available

If you see "No LLM provider available, using mock provider", ensure:
1. You have installed the provider library (`pip install anthropic` or `pip install openai`)
2. You have set the API key environment variable
3. The API key is valid and has credits

### Validation Failures

If validation against sample documents fails:
1. Check that your extraction rules match the document structure
2. Ensure element types align with your parser output
3. Verify regex patterns with a regex tester
4. Review metadata paths against actual document metadata

### Template Not Found

Built-in templates: financial, legal, medical, technical
Custom templates: Place in `examples/ontology_templates/`

## Advanced Usage

### Programmatic Usage

```python
from go_doc_go.llm import create_chat_provider
from go_doc_go.cli.ontology_interview import OntologyInterviewer
from go_doc_go.domain.ontology_builder import OntologyBuilder

# Initialize components
chat_provider = create_chat_provider("anthropic")
builder = OntologyBuilder()
interviewer = OntologyInterviewer(chat_provider, builder)

# Conduct interview
ontology = interviewer.conduct_interview()

# Save ontology
with open("my_ontology.yaml", "w") as f:
    f.write(builder.to_yaml(ontology))
```

### Extending Templates

```python
from go_doc_go.domain.templates import ONTOLOGY_TEMPLATES
from go_doc_go.domain.ontology_builder import OntologyBuilder

# Start from template
template = ONTOLOGY_TEMPLATES["financial"]

# Extend with custom entities
template["element_entity_mappings"].append({
    "entity_type": "risk_factor",
    "element_types": ["paragraph"],
    "extraction_rules": [{
        "type": "keyword_match",
        "keywords": ["risk", "uncertainty", "volatile"],
        "confidence": 0.75
    }]
})

# Build ontology
builder = OntologyBuilder(template)
ontology = builder.build_from_config({"name": "extended_financial"})
```

## See Also

- [Domain Ontology System](./domain-ontology.md) - Understanding ontology structure
- [Content Sources](./sources.md) - Configuring document sources
- [Document Parsers](./parsers.md) - Understanding element types