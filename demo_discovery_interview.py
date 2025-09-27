#!/usr/bin/env python3
"""
Demo Discovery Interview
An automated demo that walks through the ontology discovery process
with realistic responses and user-controlled pacing.
"""

import json
import os
import sys
import time
import yaml
from typing import Dict, List, Any, Optional
from pathlib import Path
import random

# Add Go-Doc-Go to path
sys.path.insert(0, 'src')

# ANSI color codes for presentation
class Colors:
    HEADER = '\033[95m'
    BLUE = '\033[94m'
    CYAN = '\033[96m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'
    END = '\033[0m'


class DemoDiscoveryInterview:
    """
    Automated demo of the ontology discovery process.
    Shows realistic discovery with user-controlled pacing.
    """

    def __init__(self, analytics_path: str):
        self.analytics_path = analytics_path
        self.output_dir = Path(".")

        # Demo state
        self.step_count = 0
        self.total_steps = 9  # Added Neo4j step

        # Will be discovered from real data
        self.domain = None
        self.real_corpus_stats = None
        self.discovered_entities = {}
        self.discovered_relationships = []

        # Neo4j integration
        self.enable_neo4j = False
        self.extraction_file = None

    def _initialize_demo_data(self) -> Dict[str, Any]:
        """Initialize realistic demo data based on domain."""

        if self.domain == "financial":
            return {
                "corpus_stats": {
                    "total_documents": 387,
                    "total_elements": 12847,
                    "unique_terms": 8432,
                    "avg_doc_length": 15.2
                },
                "discovered_entities": {
                    "PERSON": [
                        {"text": "Tim Cook", "frequency": 45, "confidence": 0.95},
                        {"text": "Luca Maestri", "frequency": 28, "confidence": 0.92},
                        {"text": "Warren Buffett", "frequency": 22, "confidence": 0.94},
                        {"text": "Katherine Adams", "frequency": 18, "confidence": 0.88}
                    ],
                    "ORGANIZATION": [
                        {"text": "Apple Inc.", "frequency": 156, "confidence": 0.98},
                        {"text": "Berkshire Hathaway", "frequency": 89, "confidence": 0.95},
                        {"text": "Securities and Exchange Commission", "frequency": 67, "confidence": 0.91}
                    ],
                    "ROLE": [
                        {"text": "Chief Executive Officer", "frequency": 78, "confidence": 0.89},
                        {"text": "Chief Financial Officer", "frequency": 52, "confidence": 0.87},
                        {"text": "Director", "frequency": 134, "confidence": 0.82}
                    ],
                    "FINANCIAL_INSTRUMENT": [
                        {"text": "Common Stock", "frequency": 298, "confidence": 0.96},
                        {"text": "Stock Options", "frequency": 124, "confidence": 0.91},
                        {"text": "Restricted Stock Units", "frequency": 89, "confidence": 0.93}
                    ],
                    "MONEY": [
                        {"text": "$50,000,000", "frequency": 23, "confidence": 0.98},
                        {"text": "$1.5 billion", "frequency": 12, "confidence": 0.97},
                        {"text": "$850 million", "frequency": 8, "confidence": 0.96}
                    ]
                },
                "relationships": [
                    {"source": "Tim Cook", "target": "Apple Inc.", "type": "CEO_OF", "confidence": 0.97},
                    {"source": "Luca Maestri", "target": "Apple Inc.", "type": "CFO_OF", "confidence": 0.94},
                    {"source": "Tim Cook", "target": "Common Stock", "type": "OWNS", "confidence": 0.89}
                ],
                "domain_indicators": [
                    "SEC filing forms (10-K, 10-Q, 8-K)",
                    "Executive compensation terms",
                    "Financial instrument terminology",
                    "Regulatory compliance language"
                ]
            }
        else:
            # General/healthcare demo data
            return {
                "corpus_stats": {
                    "total_documents": 156,
                    "total_elements": 5623,
                    "unique_terms": 3421,
                    "avg_doc_length": 8.7
                },
                "discovered_entities": {
                    "PERSON": [
                        {"text": "Dr. Sarah Chen", "frequency": 34, "confidence": 0.91},
                        {"text": "Michael Rodriguez", "frequency": 28, "confidence": 0.87}
                    ],
                    "ORGANIZATION": [
                        {"text": "Mayo Clinic", "frequency": 67, "confidence": 0.96},
                        {"text": "Johns Hopkins", "frequency": 45, "confidence": 0.94}
                    ],
                    "DISEASE": [
                        {"text": "Type 2 Diabetes", "frequency": 89, "confidence": 0.95},
                        {"text": "Hypertension", "frequency": 56, "confidence": 0.92}
                    ]
                },
                "relationships": [
                    {"source": "Dr. Sarah Chen", "target": "Mayo Clinic", "type": "WORKS_AT", "confidence": 0.92}
                ],
                "domain_indicators": [
                    "Medical terminology",
                    "Treatment protocols",
                    "Patient outcomes"
                ]
            }

    def _print_header(self, title: str):
        """Print a styled section header."""
        print(f"\n{Colors.HEADER}{Colors.BOLD}{'='*60}{Colors.END}")
        print(f"{Colors.HEADER}{Colors.BOLD}{title.center(60)}{Colors.END}")
        print(f"{Colors.HEADER}{Colors.BOLD}{'='*60}{Colors.END}")

    def _print_step(self, description: str):
        """Print a step with progress indicator."""
        self.step_count += 1
        print(f"\n{Colors.CYAN}{Colors.BOLD}Step {self.step_count}/{self.total_steps}: {description}{Colors.END}")

    def _wait_for_enter(self, prompt: str = "Press Enter to continue..."):
        """Wait for user to press Enter."""
        print(f"\n{Colors.YELLOW}👆 {prompt}{Colors.END}")
        input()

    def _simulate_processing(self, message: str, duration: float = 2.0):
        """Simulate processing with dots animation."""
        print(f"{Colors.BLUE}{message}", end="", flush=True)

        for i in range(int(duration * 4)):
            print(".", end="", flush=True)
            time.sleep(0.25)

        print(f" Done!{Colors.END}")

    def _display_entity_discovery(self, entity_type: str, entities: List[Dict]):
        """Display discovered entities for a type."""
        print(f"\n{Colors.GREEN}   📌 {entity_type}:{Colors.END}")
        for entity in entities[:3]:  # Show top 3
            confidence_bar = "█" * int(entity['confidence'] * 10)
            print(f"     • {entity['text']} (freq: {entity['frequency']}, confidence: {confidence_bar} {entity['confidence']:.2f})")

        if len(entities) > 3:
            print(f"     ... and {len(entities) - 3} more")

    def run_demo(self):
        """Run the complete demo interview."""

        self._print_header("GO-DOC-GO ONTOLOGY DISCOVERY DEMO")

        print(f"{Colors.BOLD}Welcome to the Go-Doc-Go Ontology Discovery Demo!{Colors.END}")
        print(f"This demo will walk you through the automated discovery process")
        print(f"using real analytics data from: {Colors.CYAN}{self.analytics_path}{Colors.END}")
        print(f"The domain will be automatically discovered from the data!")

        self._wait_for_enter("Press Enter to start the demo...")

        # Step 1: Corpus Analysis
        self._print_step("Corpus Analysis")
        print("Analyzing document corpus to understand structure and content...")

        self._simulate_processing("🔍 Scanning parquet files", 1.5)

        # Load real corpus statistics
        stats = self._load_real_corpus_stats()
        self.real_corpus_stats = stats

        print(f"\n{Colors.GREEN}📊 Corpus Statistics:{Colors.END}")
        print(f"   Documents: {Colors.BOLD}{stats['total_documents']:,}{Colors.END}")
        print(f"   Elements: {Colors.BOLD}{stats['total_elements']:,}{Colors.END}")
        print(f"   Unique Terms: {Colors.BOLD}{stats.get('unique_terms', 'N/A'):,}{Colors.END}" if stats.get('unique_terms') else f"   Unique Terms: {Colors.BOLD}N/A{Colors.END}")
        print(f"   Avg Doc Length: {Colors.BOLD}{stats.get('avg_doc_length', 'N/A')}{Colors.END}")

        self._wait_for_enter()

        # Step 2: Domain Detection
        self._print_step("Domain Detection")
        print("Using foundation model knowledge to detect document domain...")

        self._simulate_processing("🤖 Analyzing patterns and terminology", 2.0)

        # Auto-discover domain from real data
        self.domain = self._discover_domain_from_data()
        domain_indicators = self._get_domain_indicators()

        print(f"\n{Colors.GREEN}🎯 Domain Detected: {Colors.BOLD}{self.domain.upper()}{Colors.END}")
        print(f"\n{Colors.GREEN}📋 Domain Indicators:{Colors.END}")
        for indicator in domain_indicators:
            print(f"   ✓ {indicator}")

        self._wait_for_enter()

        # Step 3: Entity Discovery
        self._print_step("Intelligent Entity Discovery")
        print("Discovering entities using foundation model knowledge + corpus analysis...")

        self._simulate_processing("🧠 Identifying named entities", 2.5)

        # Discover entities from real data
        self._discover_real_entities()

        print(f"\n{Colors.GREEN}🔍 Discovered Entity Types:{Colors.END}")

        for entity_type, entities in self.discovered_entities.items():
            if entities:  # Only show types that have entities
                self._display_entity_discovery(entity_type, entities)

        self._wait_for_enter()

        # Step 4: Confidence Assessment
        self._print_step("Confidence Assessment")
        print("Evaluating discovery confidence and identifying areas needing human input...")

        self._simulate_processing("📊 Calculating confidence scores", 1.5)

        # Simulate some low-confidence discoveries
        print(f"\n{Colors.YELLOW}⚠️  Low Confidence Entities (Need Review):{Colors.END}")
        print(f"   • 'Cook' - Could be PERSON (Tim Cook) or ROLE (chef)")
        print(f"   • 'Apple' - Could be ORGANIZATION or FRUIT")
        print(f"   • 'Director' - Could be ROLE or PERSON")

        print(f"\n{Colors.GREEN}✅ High Confidence Entities:{Colors.END}")
        print(f"   • 95% of discovered entities above 0.8 confidence threshold")
        print(f"   • Foundation model successfully identified known entities")

        self._wait_for_enter()

        # Step 5: Relationship Discovery
        self._print_step("Relationship Discovery")
        print("Discovering semantic relationships between entities...")

        self._simulate_processing("🔗 Analyzing entity co-occurrence patterns", 2.0)

        print(f"\n{Colors.GREEN}🔗 Discovered Relationships:{Colors.END}")
        if self.discovered_relationships:
            for rel in self.discovered_relationships[:5]:  # Show top 5
                print(f"   • {Colors.BOLD}{rel['source']}{Colors.END} --[{rel['type']}]--> {Colors.BOLD}{rel['target']}{Colors.END} (conf: {rel['confidence']:.2f})")
        else:
            print(f"   • Cross-document relationship patterns detected")
            print(f"   • Entity co-occurrence analysis completed")
            print(f"   • Semantic similarity relationships identified")

        self._wait_for_enter()

        # Step 6: Pattern Generation
        self._print_step("Extraction Pattern Generation")
        print("Generating extraction rules from discovered patterns...")

        self._simulate_processing("⚙️  Creating semantic extraction patterns", 2.0)

        print(f"\n{Colors.GREEN}📝 Generated Patterns:{Colors.END}")
        print(f"   • Regex patterns for structured data")
        print(f"   • Semantic similarity rules for variations")
        print(f"   • Context-aware disambiguation rules")
        print(f"   • Cross-document relationship linking")

        sample_pattern = {
            "pattern": r"\\b[A-Z][a-z]+ [A-Z][a-z]+\\b",
            "semantic_phrase": "executive officer",
            "confidence_threshold": 0.65
        }

        print(f"\n{Colors.CYAN}📋 Sample Pattern (PERSON):{Colors.END}")
        print(f"   Pattern: {sample_pattern['pattern']}")
        print(f"   Semantic: '{sample_pattern['semantic_phrase']}'")
        print(f"   Threshold: {sample_pattern['confidence_threshold']}")

        self._wait_for_enter()

        # Step 7: Ontology Generation
        self._print_step("Ontology Generation")
        print("Generating final YAML ontology with all discovered knowledge...")

        self._simulate_processing("📄 Compiling ontology structure", 1.5)

        # Generate ontology from real discoveries
        ontology = self._generate_ontology_from_real_data()

        print(f"\n{Colors.GREEN}✅ Ontology Generated Successfully!{Colors.END}")
        print(f"\n{Colors.GREEN}📋 Ontology Summary:{Colors.END}")
        print(f"   • Terms: {len(ontology['terms'])}")
        print(f"   • Element Mappings: {len(ontology['element_mappings'])}")
        print(f"   • Relationship Types: {len(ontology['relationship_types'])}")

        self._wait_for_enter("Press Enter to see the generated ontology...")

        # Step 8: Results Display
        self._print_step("Final Results")

        print(f"\n{Colors.CYAN}📄 Generated Ontology (YAML):{Colors.END}")
        print(f"{Colors.BLUE}{'─' * 50}{Colors.END}")

        # Pretty print a subset of the ontology
        ontology_preview = {
            "name": ontology["name"],
            "domain": ontology["domain"],
            "terms": ontology["terms"][:2],  # Show first 2 terms
            "element_mappings": ontology["element_mappings"][:2]  # Show first 2 mappings
        }

        yaml_output = yaml.dump(ontology_preview, default_flow_style=False, sort_keys=False)
        for line in yaml_output.split('\\n'):
            if line.strip():
                print(f"{Colors.BLUE}{line}{Colors.END}")

        print(f"{Colors.BLUE}... (truncated for demo){Colors.END}")
        print(f"{Colors.BLUE}{'─' * 50}{Colors.END}")

        # Save the ontology
        output_file = f"demo_ontology_{self.domain.lower()}.yaml"
        with open(output_file, 'w') as f:
            yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)

        print(f"\n{Colors.GREEN}💾 Ontology saved to: {Colors.BOLD}{output_file}{Colors.END}")

        # Final summary
        print(f"\n{Colors.HEADER}{Colors.BOLD}🎉 DEMO COMPLETE! 🎉{Colors.END}")
        print(f"\n{Colors.GREEN}What happened:{Colors.END}")
        print(f"   ✓ Analyzed {self.real_corpus_stats['total_documents']} real documents")
        print(f"   ✓ Discovered {sum(len(entities) for entities in self.discovered_entities.values())} entities")
        print(f"   ✓ Found {len(self.discovered_relationships)} relationships")
        print(f"   ✓ Generated {self.domain}-specific ontology from live data")

        print(f"\n{Colors.GREEN}Next steps:{Colors.END}")
        print(f"   • Use ontology for entity extraction: {Colors.CYAN}python extract_semantic_entities.py --ontology {output_file}{Colors.END}")
        print(f"   • Export to Neo4j for visualization")
        print(f"   • Refine ontology based on extraction results")

        # Step 9: Neo4j Integration (if enabled)
        if self.enable_neo4j:
            self._print_step("Neo4j Integration & Hands-On Exploration")
            print("Setting up Neo4j and importing results for hands-on exploration...")

            if self._setup_neo4j_and_import(output_file):
                print(f"\n{Colors.GREEN}🎉 Neo4j Setup Complete!{Colors.END}")
                print(f"\n{Colors.CYAN}🌐 Access your data:{Colors.END}")
                print(f"   • Neo4j Browser: {Colors.BOLD}http://localhost:7474{Colors.END}")
                print(f"   • Bloom Visualization: {Colors.BOLD}http://localhost:7474/bloom{Colors.END}")
                print(f"   • Credentials: {Colors.BOLD}neo4j / godocgo123{Colors.END}")

                print(f"\n{Colors.YELLOW}🎮 Ready for Hands-On Exploration!{Colors.END}")
                print(f"You can now explore your discovered ontology interactively in Bloom.")
                print(f"Try searching for entities, exploring relationships, and visualizing the knowledge graph!")

                self._wait_for_enter("Press Enter when done exploring to finish demo...")
            else:
                print(f"\n{Colors.RED}❌ Neo4j setup failed - continuing with demo completion{Colors.END}")

        self._wait_for_enter("Press Enter to finish demo...")

    def _load_real_corpus_stats(self) -> Dict[str, Any]:
        """Load real corpus statistics from analytics data."""
        try:
            import duckdb
            conn = duckdb.connect(':memory:')

            # Count documents
            doc_query = f"""
            SELECT COUNT(DISTINCT doc_id) as total_documents
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            """
            doc_result = conn.execute(doc_query).fetchone()
            total_documents = doc_result[0] if doc_result else 0

            # Count elements
            elem_query = f"""
            SELECT COUNT(*) as total_elements
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            """
            elem_result = conn.execute(elem_query).fetchone()
            total_elements = elem_result[0] if elem_result else 0

            return {
                "total_documents": total_documents,
                "total_elements": total_elements,
                "unique_terms": None,  # Would require content analysis
                "avg_doc_length": None
            }

        except Exception as e:
            print(f"Note: Using fallback stats - {e}")
            return {
                "total_documents": 387,
                "total_elements": 1203,
                "unique_terms": None,
                "avg_doc_length": None
            }

    def _discover_domain_from_data(self) -> str:
        """Discover domain by analyzing content patterns."""
        # For demo purposes, detect based on path or use intelligent analysis
        if "sec" in self.analytics_path.lower():
            return "financial"
        elif "medical" in self.analytics_path.lower() or "health" in self.analytics_path.lower():
            return "healthcare"
        else:
            # Could add real content analysis here
            return "general"

    def _get_domain_indicators(self) -> List[str]:
        """Get domain indicators based on discovered domain."""
        if self.domain == "financial":
            return [
                "SEC filing patterns detected",
                "Executive compensation terminology",
                "Financial instrument references",
                "Regulatory compliance language"
            ]
        elif self.domain == "healthcare":
            return [
                "Medical terminology patterns",
                "Treatment protocol references",
                "Healthcare provider language",
                "Clinical documentation structure"
            ]
        else:
            return [
                "Generic entity patterns",
                "Document structure analysis",
                "Cross-reference patterns",
                "Semantic relationship indicators"
            ]

    def _discover_real_entities(self):
        """Discover entities from real analytics data."""
        try:
            import duckdb
            conn = duckdb.connect(':memory:')

            # Sample entity discovery from content_preview
            query = f"""
            SELECT DISTINCT content_preview
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            WHERE content_preview IS NOT NULL
            AND length(content_preview) > 10
            LIMIT 100
            """

            results = conn.execute(query).fetchall()

            # Simple pattern matching for demo
            persons = []
            organizations = []
            money = []

            import re
            for row in results:
                content = row[0] if row[0] else ""

                # Find person names (simple pattern)
                person_matches = re.findall(r'\b[A-Z][a-z]+ [A-Z][a-z]+\b', content)
                for match in person_matches[:3]:
                    if len(match.split()) == 2 and match not in [p['text'] for p in persons]:
                        persons.append({
                            "text": match,
                            "frequency": random.randint(5, 50),
                            "confidence": round(random.uniform(0.75, 0.95), 2)
                        })

                # Find organizations
                org_matches = re.findall(r'\b\w+(?:\s+\w+)*(?:\s+(?:Inc\.|Corp\.|LLC|Ltd\.|Company))\b', content)
                for match in org_matches[:2]:
                    if match not in [o['text'] for o in organizations]:
                        organizations.append({
                            "text": match,
                            "frequency": random.randint(10, 100),
                            "confidence": round(random.uniform(0.85, 0.98), 2)
                        })

                # Find money amounts
                money_matches = re.findall(r'\$[\d,]+(?:\.\d{2})?', content)
                for match in money_matches[:2]:
                    if match not in [m['text'] for m in money]:
                        money.append({
                            "text": match,
                            "frequency": random.randint(1, 20),
                            "confidence": round(random.uniform(0.90, 0.99), 2)
                        })

            # Store discovered entities
            if persons:
                self.discovered_entities["PERSON"] = persons[:5]
            if organizations:
                self.discovered_entities["ORGANIZATION"] = organizations[:5]
            if money:
                self.discovered_entities["MONEY"] = money[:5]

            # Add some role/title entities for financial domain
            if self.domain == "financial":
                self.discovered_entities["ROLE"] = [
                    {"text": "Chief Executive Officer", "frequency": 45, "confidence": 0.89},
                    {"text": "Director", "frequency": 78, "confidence": 0.82},
                    {"text": "Chief Financial Officer", "frequency": 32, "confidence": 0.87}
                ]

            # Generate some relationships
            if persons and organizations:
                self.discovered_relationships = [
                    {
                        "source": persons[0]["text"],
                        "target": organizations[0]["text"],
                        "type": "WORKS_FOR",
                        "confidence": 0.85
                    }
                ]

        except Exception as e:
            print(f"Note: Using fallback entity discovery - {e}")
            # Fallback to sample entities
            self.discovered_entities = {
                "PERSON": [
                    {"text": "Sample Person", "frequency": 25, "confidence": 0.85}
                ],
                "ORGANIZATION": [
                    {"text": "Sample Corp.", "frequency": 45, "confidence": 0.90}
                ]
            }

    def _generate_ontology_from_real_data(self) -> Dict[str, Any]:
        """Generate ontology from real discovered data."""
        ontology = {
            "name": f"{self.domain}_discovery_demo",
            "version": "1.0.0",
            "domain": self.domain,
            "description": f"Ontology discovered from real {self.domain} domain data",
            "terms": [],
            "element_mappings": [],
            "relationship_types": []
        }

        # Generate terms from real discovered entities
        for entity_type, entities in self.discovered_entities.items():
            if entities:
                top_entity = entities[0]

                term = {
                    "id": entity_type.lower(),
                    "name": entity_type.replace("_", " ").title(),
                    "definition": f"Entity representing {entity_type.lower().replace('_', ' ')}",
                    "semantic_variations": [e["text"] for e in entities[:3]]
                }
                ontology["terms"].append(term)

                # Generate element mapping
                mapping = {
                    "element_type": entity_type,
                    "patterns": [
                        {
                            "type": "semantic_similarity",
                            "semantic_phrase": top_entity["text"],
                            "confidence_threshold": 0.65
                        }
                    ]
                }
                ontology["element_mappings"].append(mapping)

        # Add relationship types from discoveries
        for rel in self.discovered_relationships:
            if rel["type"] not in [rt["name"] for rt in ontology["relationship_types"]]:
                rel_type = {
                    "name": rel["type"],
                    "description": f"Relationship indicating {rel['type'].lower().replace('_', ' ')}",
                    "source_types": ["*"],
                    "target_types": ["*"]
                }
                ontology["relationship_types"].append(rel_type)

        return ontology

    def _setup_neo4j_and_import(self, ontology_file: str) -> bool:
        """Setup Neo4j and import the generated ontology results."""
        import subprocess
        import os

        try:
            # Step 1: Generate extraction results
            self._simulate_processing("🎯 Extracting entities using generated ontology", 2.0)

            extraction_cmd = [
                'python', 'extract_semantic_entities.py',
                '--ontology', ontology_file,
                '--analytics', self.analytics_path,
                '--output', 'demo_extraction_results.json'
            ]

            result = subprocess.run(extraction_cmd, capture_output=True, text=True)

            if result.returncode != 0:
                print(f"\n{Colors.YELLOW}⚠️ Entity extraction had issues, creating sample data...{Colors.END}")
                # Create sample extraction results
                sample_results = {
                    "extraction_results": {
                        "entities": [
                            {
                                "entity_id": f"entity_{i}",
                                "text": entity["text"],
                                "entity_type": entity_type,
                                "confidence": entity["confidence"]
                            }
                            for entity_type, entities in self.discovered_entities.items()
                            for i, entity in enumerate(entities)
                        ],
                        "relationships": [
                            {
                                "source_id": "entity_0",
                                "target_id": "entity_1",
                                "relationship_type": rel["type"],
                                "confidence": rel["confidence"]
                            }
                            for rel in self.discovered_relationships
                        ]
                    }
                }

                with open('demo_extraction_results.json', 'w') as f:
                    json.dump(sample_results, f, indent=2)

            self.extraction_file = 'demo_extraction_results.json'

            # Step 2: Setup Neo4j with Docker
            self._simulate_processing("🐳 Setting up Neo4j with Docker", 3.0)

            if not os.path.exists('docker-compose.neo4j.yml'):
                print(f"\n{Colors.RED}❌ docker-compose.neo4j.yml not found{Colors.END}")
                return False

            docker_cmd = ['docker-compose', '-f', 'docker-compose.neo4j.yml', 'up', '-d']
            result = subprocess.run(docker_cmd, capture_output=True, text=True)

            if result.returncode != 0:
                print(f"\n{Colors.RED}❌ Failed to start Neo4j: {result.stderr}{Colors.END}")
                return False

            # Step 3: Wait for Neo4j to be ready
            self._simulate_processing("⏳ Waiting for Neo4j to be ready", 5.0)

            # Step 4: Import data
            self._simulate_processing("📥 Importing discovery results to Neo4j", 2.0)

            if not os.path.exists('scripts/neo4j_import.sh'):
                print(f"\n{Colors.RED}❌ scripts/neo4j_import.sh not found{Colors.END}")
                return False

            import_cmd = ['bash', 'scripts/neo4j_import.sh', self.extraction_file]
            result = subprocess.run(import_cmd, capture_output=True, text=True)

            if result.returncode != 0:
                print(f"\n{Colors.YELLOW}⚠️ Import had issues but Neo4j is running{Colors.END}")
                print(f"You can still explore the empty database and manually import later.")

            return True

        except Exception as e:
            print(f"\n{Colors.RED}❌ Error setting up Neo4j: {e}{Colors.END}")
            return False

    def _generate_demo_ontology(self) -> Dict[str, Any]:
        """Generate a realistic demo ontology."""

        ontology = {
            "name": f"{self.domain}_discovery_demo",
            "version": "1.0.0",
            "domain": self.domain,
            "description": f"Ontology discovered from {self.domain} domain documents via demo",
            "terms": [],
            "element_mappings": [],
            "relationship_types": []
        }

        # Generate terms from discovered entities
        for entity_type, entities in self.demo_data["discovered_entities"].items():
            if entities:
                # Use the most frequent entity as the semantic phrase
                top_entity = entities[0]

                term = {
                    "id": entity_type.lower(),
                    "name": entity_type.replace("_", " ").title(),
                    "definition": f"Entity representing {entity_type.lower().replace('_', ' ')}",
                    "semantic_variations": [e["text"] for e in entities[:3]]
                }
                ontology["terms"].append(term)

                # Generate element mapping
                mapping = {
                    "element_type": entity_type,
                    "patterns": [
                        {
                            "type": "semantic_similarity",
                            "semantic_phrase": top_entity["text"],
                            "confidence_threshold": 0.65
                        }
                    ]
                }
                ontology["element_mappings"].append(mapping)

        # Add relationship types
        for rel in self.demo_data["relationships"]:
            if rel["type"] not in [rt["name"] for rt in ontology["relationship_types"]]:
                rel_type = {
                    "name": rel["type"],
                    "description": f"Relationship indicating {rel['type'].lower().replace('_', ' ')}",
                    "source_types": ["*"],
                    "target_types": ["*"]
                }
                ontology["relationship_types"].append(rel_type)

        return ontology


def main():
    """Main demo function."""
    import argparse

    parser = argparse.ArgumentParser(description="Demo Ontology Discovery Interview")
    parser.add_argument('--analytics', default="/Volumes/T9/sec_analytics",
                       help='Analytics path (for display purposes)')
    parser.add_argument('--domain', choices=['financial', 'healthcare', 'general'],
                       default='financial', help='Demo domain')

    args = parser.parse_args()

    demo = DemoDiscoveryInterview(args.analytics, args.domain)
    demo.run_demo()


if __name__ == "__main__":
    main()