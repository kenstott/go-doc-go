# Go-Doc-Go Discovery Demo

## Interactive Ontology Discovery Demo

The demo provides a live, automated walkthrough of the Go-Doc-Go ontology discovery process. Perfect for presentations and showcasing the system's capabilities.

### Quick Start

```bash
## Run financial domain demo (default)
PYTHONPATH=src python -m go_doc_go demo-discovery

## Run healthcare domain demo
PYTHONPATH=src python -m go_doc_go demo-discovery --domain healthcare

## Fast mode (shorter delays)
PYTHONPATH=src python -m go_doc_go demo-discovery --fast

## Specify analytics path for display
PYTHONPATH=src python -m go_doc_go demo-discovery --analytics /Volumes/T9/sec_analytics
```python

### Demo Flow

The demo walks through 8 steps of the discovery process:

1. **Corpus Analysis** - Shows document statistics and analysis
2. **Domain Detection** - Demonstrates automatic domain identification
3. **Entity Discovery** - Shows AI discovering entities with confidence scores
4. **Confidence Assessment** - Displays high/low confidence entities
5. **Relationship Discovery** - Shows semantic relationship detection
6. **Pattern Generation** - Demonstrates extraction rule creation
7. **Ontology Generation** - Shows YAML ontology compilation
8. **Final Results** - Displays generated ontology and summary

### Interactive Features

- **User-Controlled Pacing**: Press Enter to advance through each step
- **Realistic Data**: Uses domain-specific demo data based on actual patterns
- **Color-Coded Output**: Different colors for different discovery phases
- **Processing Simulation**: Animated dots show "thinking" process
- **Confidence Visualization**: Bar charts show confidence levels
- **Progress Tracking**: Step counter shows progress through demo

### Demo Domains

#### Financial Domain (`--domain financial`)
- Uses SEC filing data patterns
- Shows executive discovery (Tim Cook, Luca Maestri)
- Demonstrates financial instrument detection
- Includes regulatory compliance patterns

#### Healthcare Domain (`--domain healthcare`)
- Uses medical terminology patterns
- Shows doctor/patient entity discovery
- Demonstrates disease and treatment detection
- Includes clinical relationship patterns

#### General Domain (`--domain general`)
- Uses generic entity patterns
- Shows basic person/organization discovery
- Demonstrates general relationship detection

### Output

The demo generates a real ontology file:
- `demo_ontology_financial.yaml`
- `demo_ontology_healthcare.yaml`
- `demo_ontology_general.yaml`

These files can be used for actual entity extraction:

```bash
## Use demo-generated ontology for real extraction
python extract_semantic_entities.py --ontology demo_ontology_financial.yaml --analytics /Volumes/T9/sec_analytics
```bash

### Demo Features

- **Realistic Discovery**: Shows actual entity/relationship discovery patterns
- **Foundation Model Integration**: Demonstrates AI-powered entity recognition
- **Confidence Scoring**: Shows how the system assesses discovery confidence
- **Ambiguity Handling**: Demonstrates entities that need human review
- **Pattern Generation**: Shows extraction rule creation process
- **Final Ontology**: Generates real, usable ontology files

### Perfect For

- **Product Demonstrations**: Show Go-Doc-Go capabilities to stakeholders
- **Training Sessions**: Teach users about the discovery process
- **Conference Presentations**: Live demo at conferences/meetups
- **Customer Onboarding**: Help customers understand the system
- **Development Testing**: Test discovery flow without real analytics data

### Tips for Presentations

1. **Prepare Your Audience**: Explain that they'll see an automated discovery process
2. **Control Pacing**: Use Enter key to pause at interesting discoveries
3. **Highlight Key Features**: Point out confidence scores and AI reasoning
4. **Show Real Output**: The generated ontology is actually usable
5. **Use Fast Mode**: For time-constrained presentations, use `--fast` flag

The demo provides a polished, professional showcase of Go-Doc-Go's intelligent discovery capabilities while maintaining full presenter control over timing and flow.
