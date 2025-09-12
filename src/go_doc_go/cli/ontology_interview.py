"""
Interactive interview system for ontology generation.
"""

import json
import logging
from typing import Dict, Any, List, Optional
from dataclasses import dataclass, field

from go_doc_go.llm.chat import ChatProvider

logger = logging.getLogger(__name__)


@dataclass
class InterviewContext:
    """Context maintained throughout the interview."""
    domain: str = ""
    document_types: List[str] = field(default_factory=list)
    key_concepts: List[str] = field(default_factory=list)
    terms: List[Dict[str, Any]] = field(default_factory=list)
    entities: List[Dict[str, Any]] = field(default_factory=list)
    relationships: List[Dict[str, Any]] = field(default_factory=list)
    sample_content: Optional[str] = None
    metadata_fields: List[str] = field(default_factory=list)
    messages: List[Dict[str, str]] = field(default_factory=list)


class OntologyInterviewer:
    """Conducts interactive interviews for ontology creation."""
    
    def __init__(
        self, 
        chat_provider: ChatProvider,
        builder: 'OntologyBuilder',
        max_iterations: int = 20,
        data_config_path: Optional[str] = None
    ):
        """
        Initialize the interviewer.
        
        Args:
            chat_provider: LLM chat provider
            builder: Ontology builder instance
            max_iterations: Maximum interview rounds
            data_config_path: Path to Go-Doc-Go config with data sources
        """
        self.chat_provider = chat_provider
        self.builder = builder
        self.max_iterations = max_iterations
        self.data_config_path = data_config_path
        self.context = InterviewContext()
    
    def conduct_interview(self) -> Dict[str, Any]:
        """
        Conduct the full interview process.
        
        Returns:
            Generated ontology dictionary
        """
        # Initialize system prompt
        self.context.messages.append({
            "role": "system",
            "content": self._get_system_prompt()
        })
        
        # Phase 1: Domain Understanding
        self._phase_domain_understanding()
        
        # Phase 2: Term Definition
        self._phase_term_definition()
        
        # Phase 3: Entity Extraction
        self._phase_entity_extraction()
        
        # Phase 4: Relationship Mapping
        self._phase_relationship_mapping()
        
        # Phase 5: Refinement
        self._phase_refinement()
        
        # Build final ontology
        return self._build_ontology()
    
    def _get_system_prompt(self) -> str:
        """Get the system prompt for the LLM."""
        return """You are an expert ontology designer helping create domain ontologies for document analysis.
Your role is to guide users through creating ontologies that extract entities and relationships from documents.

Key principles:
1. Start simple and build complexity gradually
2. Focus on the user's specific domain and use cases
3. Suggest relevant terms, entities, and relationships based on the domain
4. Provide concrete examples when possible
5. Validate understanding before moving to the next phase

Output format:
- When suggesting terms, provide them as JSON arrays
- When defining entities, use the format: {"entity_type": "name", "description": "...", "extraction_rules": [...]}
- When defining relationships, use the format: {"source": "entity1", "target": "entity2", "type": "relationship_type"}

Be conversational but focused on gathering the necessary information."""
    
    def _phase_domain_understanding(self):
        """Phase 1: Understand the domain and document types."""
        print("\n📚 Phase 1: Domain Understanding")
        print("-" * 40)
        
        # First, try to analyze actual documents if available
        if self.data_config_path:
            print("\n🔍 Analyzing your configured data sources...")
            self._analyze_sample_documents()
            
            # If we found a domain from the documents, use it
            if not self.context.domain:
                domain = input("\nWhat domain best describes these documents? (e.g., financial, legal, medical): ").strip()
                self.context.domain = domain
        else:
            # Ask about domain if no data sources
            domain = input("What domain or industry are you working with? (e.g., financial, legal, medical): ").strip()
            self.context.domain = domain
            
            # Offer to load data sources
            use_data = input("\nDo you have a Go-Doc-Go config file with data sources? (y/n): ").strip().lower()
            if use_data == 'y':
                self._analyze_sample_documents()
        
        # If domain still not set, ask for it
        if not self.context.domain:
            domain = input("\nWhat domain or industry are you working with? (e.g., financial, legal, medical): ").strip()
            self.context.domain = domain
        
        # Explain what will happen
        print(f"\n📋 Based on '{self.context.domain}' domain, I will:")
        print("  • Suggest common document types for this domain")
        print("  • Generate relevant terms and synonyms")
        print("  • Propose entity types to extract")
        print("  • Recommend relationships between entities")
        print("\nYou can accept, modify, or replace any suggestions.")
        
        # If we don't have document types yet, get suggestions
        if not self.context.document_types:
            # Get LLM suggestions for document types
            self.context.messages.append({
                "role": "user",
                "content": f"I'm working with {self.context.domain} documents. What are common document types in this domain? Be concise."
            })
            
            response = self.chat_provider.chat_completion(self.context.messages)
            self.context.messages.append({"role": "assistant", "content": response})
            print(f"\n🤖 Suggested document types for {self.context.domain}:")
            print(response)
            
            # Ask user to specify their document types
            doc_types_input = input("\n📄 What specific document types will YOU analyze? (comma-separated or press Enter to accept suggestions): ").strip()
            if doc_types_input:
                self.context.document_types = [dt.strip() for dt in doc_types_input.split(",")]
            else:
                # Extract from AI response
                self.context.document_types = self._extract_list_from_response(response)[:5]  # Limit to 5
                print(f"Using suggested types: {', '.join(self.context.document_types)}")
        
        # If we don't have key concepts yet, get them
        if not self.context.key_concepts:
            print("\n🎯 What are the key concepts you want to extract?")
            key_concepts_input = input("List key concepts (comma-separated, or press Enter for AI suggestions): ").strip()
            if key_concepts_input:
                self.context.key_concepts = [kc.strip() for kc in key_concepts_input.split(",")]
            else:
                # Get AI suggestions for key concepts
                prompt = f"For {self.context.domain} documents of type {', '.join(self.context.document_types)}, what are 5-7 key concepts to extract?"
                if self.context.sample_content:
                    prompt += f"\n\nBased on this sample content:\n{self.context.sample_content[:500]}"
                
                self.context.messages.append({
                    "role": "user",
                    "content": prompt
                })
                concept_response = self.chat_provider.chat_completion(self.context.messages)
                self.context.messages.append({"role": "assistant", "content": concept_response})
                self.context.key_concepts = self._extract_list_from_response(concept_response)[:7]
                print(f"Using AI-suggested concepts: {', '.join(self.context.key_concepts)}")
    
    def _analyze_sample_documents(self):
        """Analyze sample documents from configured data sources."""
        try:
            from go_doc_go import Config
            from go_doc_go.content_source.factory import get_content_source
            from go_doc_go.document_parser.factory import get_parser_for_content
            
            print("\n📂 Looking for configured data sources...")
            
            # Try to load config
            config_path = None
            import os
            
            # Use provided config path first
            if self.data_config_path and os.path.exists(self.data_config_path):
                config_path = self.data_config_path
            else:
                # Check common config locations
                for path in ["config.yaml", "config.yml", os.environ.get("GO_DOC_GO_CONFIG_PATH")]:
                    if path and os.path.exists(path):
                        config_path = path
                        break
            
            if not config_path:
                config_input = input("Enter path to config file (or press Enter to skip): ").strip()
                if config_input and os.path.exists(config_input):
                    config_path = config_input
                else:
                    print("⚠️ No config file found, skipping document analysis")
                    return
            
            print(f"📄 Loading config from: {config_path}")
            config = Config(config_path)
            
            # Get content sources
            sources = config.get("content_sources", [])
            if not sources:
                print("⚠️ No content sources configured")
                return
            
            print(f"\n📚 Found {len(sources)} content source(s):")
            for i, source in enumerate(sources):
                print(f"  {i+1}. {source.get('name', 'Unnamed')} ({source.get('type', 'unknown')})")
            
            # Let user select source
            if len(sources) == 1:
                source_idx = 0
            else:
                source_input = input(f"Select source (1-{len(sources)}): ").strip()
                try:
                    source_idx = int(source_input) - 1
                    if source_idx < 0 or source_idx >= len(sources):
                        print("Invalid selection")
                        return
                except ValueError:
                    print("Invalid selection")
                    return
            
            selected_source = sources[source_idx]
            print(f"\n🔍 Analyzing documents from: {selected_source.get('name', 'source')}")
            
            # Create content source
            content_source = get_content_source(selected_source)
            
            # Get sample documents (limit to 5)
            sample_docs = []
            max_samples = 5
            
            # Different content sources have different methods
            if hasattr(content_source, 'list_documents'):
                # FileContentSource uses list_documents
                doc_list = content_source.list_documents()
                for i, doc_info in enumerate(doc_list[:max_samples]):
                    # Fetch the actual document content
                    # Use 'id' field which is the source identifier
                    source_id = doc_info.get('id') or doc_info.get('source_id')
                    doc = content_source.fetch_document(source_id)
                    sample_docs.append(doc)
            elif hasattr(content_source, 'get_documents'):
                # Other sources might use get_documents
                for i, doc in enumerate(content_source.get_documents()):
                    if i >= max_samples:
                        break
                    sample_docs.append(doc)
            else:
                print("⚠️ Content source doesn't support document listing")
                return
            
            if not sample_docs:
                print("⚠️ No documents found in source")
                return
            
            print(f"📊 Analyzing {len(sample_docs)} sample document(s)...")
            
            # Analyze documents and extract patterns
            combined_content = []
            extracted_patterns = {
                "document_types": set(),
                "entities": [],
                "terms": [],
                "metadata_fields": set()
            }
            
            for doc in sample_docs:
                # Parse document
                parser = get_parser_for_content(doc)
                result = parser.parse(doc)
                
                # Extract document type
                doc_type = doc.get("doc_type", "unknown")
                extracted_patterns["document_types"].add(doc_type)
                
                # Extract metadata fields
                metadata = doc.get("metadata", {})
                extracted_patterns["metadata_fields"].update(metadata.keys())
                
                # Get sample content
                elements = result.get("elements", [])
                for element in elements[:10]:  # Sample first 10 elements
                    content = element.get("content_preview", "")
                    if content:
                        combined_content.append(content)
            
            # Store analysis results
            self.context.sample_content = "\n".join(combined_content[:20])  # Limit to 20 snippets
            
            # Update document types if not already set
            if not self.context.document_types:
                self.context.document_types = list(extracted_patterns["document_types"])
                print(f"✅ Detected document types: {', '.join(self.context.document_types)}")
            
            # Show metadata fields found
            if extracted_patterns["metadata_fields"]:
                print(f"✅ Found metadata fields: {', '.join(list(extracted_patterns['metadata_fields'])[:10])}")
                self.context.metadata_fields = list(extracted_patterns["metadata_fields"])
            
            print(f"✅ Analyzed {len(combined_content)} content samples")
            
            # Ask AI to identify patterns in the actual content
            if self.context.sample_content:
                self.context.messages.append({
                    "role": "user",
                    "content": f"Based on this sample content from {self.context.domain} documents:\n\n{self.context.sample_content[:1000]}\n\nWhat entities and patterns do you see?"
                })
                
                pattern_response = self.chat_provider.chat_completion(self.context.messages)
                self.context.messages.append({"role": "assistant", "content": pattern_response})
                print(f"\n🤖 Patterns found in your documents:")
                print(pattern_response[:500])
                
        except Exception as e:
            logger.warning(f"Could not analyze documents: {e}")
            print(f"⚠️ Could not analyze documents: {e}")
            print("Continuing with manual configuration...")
    
    def _extract_list_from_response(self, response: str) -> List[str]:
        """Extract a list of items from an AI response."""
        import re
        # Try to find bullet points, numbered lists, or comma-separated items
        items = []
        
        # Look for bullet points or numbered lists
        lines = response.split('\n')
        for line in lines:
            # Match patterns like "- item", "* item", "1. item", "• item"
            match = re.match(r'^[\-\*•]\s+(.+)$|^\d+\.\s+(.+)$', line.strip())
            if match:
                item = match.group(1) or match.group(2)
                items.append(item.strip())
        
        # If no list found, try comma-separated in first substantial line
        if not items:
            for line in lines:
                if len(line.strip()) > 20 and ',' in line:
                    items = [item.strip() for item in line.split(',')]
                    break
        
        # If still nothing, try to extract quoted items
        if not items:
            items = re.findall(r'"([^"]+)"', response)
        
        return items
    
    def _phase_term_definition(self):
        """Phase 2: Define domain terms and synonyms."""
        print("\n📝 Phase 2: Term Definition")
        print("-" * 40)
        print("\n💡 The AI will now suggest important terms and synonyms based on your domain.")
        print("These terms help identify key concepts in your documents.")
        
        # Get LLM suggestions for terms
        prompt = f"""Based on the {self.context.domain} domain with document types {self.context.document_types} 
        and key concepts {self.context.key_concepts}, suggest important terms and their synonyms.
        
        Format as JSON array: [{{"term": "name", "synonyms": ["syn1", "syn2"], "description": "..."}}]"""
        
        if self.context.sample_content:
            prompt += f"\n\nSample content:\n{self.context.sample_content[:500]}"
        
        self.context.messages.append({"role": "user", "content": prompt})
        response = self.chat_provider.chat_completion(self.context.messages)
        self.context.messages.append({"role": "assistant", "content": response})
        
        print(f"\n🤖 Suggested terms:\n{response}")
        
        # Parse suggested terms
        try:
            import re
            json_match = re.search(r'\[.*\]', response, re.DOTALL)
            if json_match:
                suggested_terms = json.loads(json_match.group())
                self.context.terms.extend(suggested_terms)
        except Exception as e:
            logger.warning(f"Could not parse suggested terms: {e}")
        
        # Allow user to add/modify terms
        while True:
            action = input("\n(a)dd term, (m)odify term, (d)elete term, or (c)ontinue: ").strip().lower()
            
            if action == 'c':
                break
            elif action == 'a':
                term_name = input("Term name: ").strip()
                synonyms = input("Synonyms (comma-separated): ").strip()
                synonyms_list = [s.strip() for s in synonyms.split(",")] if synonyms else []
                description = input("Description: ").strip()
                
                self.context.terms.append({
                    "term": term_name,
                    "synonyms": synonyms_list,
                    "description": description
                })
                print(f"✅ Added term: {term_name}")
            
            elif action == 'm':
                # List terms
                for i, term in enumerate(self.context.terms):
                    print(f"{i}: {term['term']}")
                idx = int(input("Term index to modify: "))
                if 0 <= idx < len(self.context.terms):
                    term = self.context.terms[idx]
                    term['term'] = input(f"Term name [{term['term']}]: ").strip() or term['term']
                    new_synonyms = input(f"Synonyms [{','.join(term.get('synonyms', []))}]: ").strip()
                    if new_synonyms:
                        term['synonyms'] = [s.strip() for s in new_synonyms.split(",")]
                    term['description'] = input(f"Description [{term.get('description', '')}]: ").strip() or term.get('description', '')
                    print(f"✅ Modified term: {term['term']}")
            
            elif action == 'd':
                for i, term in enumerate(self.context.terms):
                    print(f"{i}: {term['term']}")
                idx = int(input("Term index to delete: "))
                if 0 <= idx < len(self.context.terms):
                    deleted = self.context.terms.pop(idx)
                    print(f"✅ Deleted term: {deleted['term']}")
    
    def _phase_entity_extraction(self):
        """Phase 3: Define entities and extraction rules."""
        print("\n🔍 Phase 3: Entity Extraction")
        print("-" * 40)
        print("\n💡 The AI will suggest entities to extract from your documents.")
        print("Entities are specific items like people, companies, dates, amounts, etc.")
        
        # Get LLM suggestions for entities
        prompt = f"""Based on the {self.context.domain} domain with terms {[t['term'] for t in self.context.terms]},
        suggest entity types to extract and their extraction rules.
        
        Consider document types: {self.context.document_types}
        Key concepts: {self.context.key_concepts}"""
        
        # Add metadata fields if available
        if self.context.metadata_fields:
            prompt += f"\n\nAvailable metadata fields in documents: {', '.join(self.context.metadata_fields)}"
            prompt += "\nPlease include metadata-based extraction rules for relevant fields."
        
        # Add sample content if available
        if self.context.sample_content:
            prompt += f"\n\nSample content from actual documents:\n{self.context.sample_content[:500]}"
        
        prompt += """
        
        Format as JSON array: [{{"entity_type": "name", "description": "...", "extraction_rules": [
            {{"type": "METADATA", "path": "metadata.field"}},
            {{"type": "REGEX", "pattern": "pattern"}},
            {{"type": "KEYWORDS", "keywords": ["word1", "word2"]}}
        ]}}]"""
        
        self.context.messages.append({"role": "user", "content": prompt})
        response = self.chat_provider.chat_completion(self.context.messages)
        self.context.messages.append({"role": "assistant", "content": response})
        
        print(f"\n🤖 Suggested entities:\n{response}")
        
        # Parse suggested entities
        try:
            import re
            json_match = re.search(r'\[.*\]', response, re.DOTALL)
            if json_match:
                suggested_entities = json.loads(json_match.group())
                self.context.entities.extend(suggested_entities)
        except Exception as e:
            logger.warning(f"Could not parse suggested entities: {e}")
        
        # Allow user to refine entities
        while True:
            action = input("\n(a)dd entity, (m)odify entity, (d)elete entity, or (c)ontinue: ").strip().lower()
            
            if action == 'c':
                break
            elif action == 'a':
                entity_type = input("Entity type name: ").strip()
                description = input("Description: ").strip()
                element_types = input("Element types to extract from (comma-separated, e.g., paragraph,heading): ").strip()
                
                # Add extraction rules
                rules = []
                while True:
                    rule_type = input("Add extraction rule? (metadata/regex/keywords/none): ").strip().lower()
                    if rule_type == 'none':
                        break
                    elif rule_type == 'metadata':
                        path = input("Metadata path (e.g., metadata.speaker): ").strip()
                        rules.append({"type": "METADATA", "path": path})
                    elif rule_type == 'regex':
                        pattern = input("Regex pattern: ").strip()
                        rules.append({"type": "REGEX", "pattern": pattern})
                    elif rule_type == 'keywords':
                        keywords = input("Keywords (comma-separated): ").strip()
                        rules.append({"type": "KEYWORDS", "keywords": [k.strip() for k in keywords.split(",")]})
                
                self.context.entities.append({
                    "entity_type": entity_type,
                    "description": description,
                    "element_types": [et.strip() for et in element_types.split(",")],
                    "extraction_rules": rules
                })
                print(f"✅ Added entity: {entity_type}")
            
            elif action == 'm' or action == 'd':
                # List entities
                for i, entity in enumerate(self.context.entities):
                    print(f"{i}: {entity['entity_type']} - {entity.get('description', 'No description')}")
                
                if action == 'm':
                    idx = int(input("Entity index to modify: "))
                    if 0 <= idx < len(self.context.entities):
                        # Modify entity (simplified for brevity)
                        entity = self.context.entities[idx]
                        entity['entity_type'] = input(f"Entity type [{entity['entity_type']}]: ").strip() or entity['entity_type']
                        entity['description'] = input(f"Description [{entity.get('description', '')}]: ").strip() or entity.get('description', '')
                        print(f"✅ Modified entity: {entity['entity_type']}")
                
                elif action == 'd':
                    idx = int(input("Entity index to delete: "))
                    if 0 <= idx < len(self.context.entities):
                        deleted = self.context.entities.pop(idx)
                        print(f"✅ Deleted entity: {deleted['entity_type']}")
    
    def _phase_relationship_mapping(self):
        """Phase 4: Define relationships between entities."""
        print("\n🔗 Phase 4: Relationship Mapping")
        print("-" * 40)
        
        if len(self.context.entities) < 2:
            print("ℹ️  Skipping relationship mapping (need at least 2 entities)")
            return
        
        # Get LLM suggestions for relationships
        entity_types = [e['entity_type'] for e in self.context.entities]
        prompt = f"""Given these entity types: {entity_types}
        in the {self.context.domain} domain, suggest relationships between them.
        
        Format as JSON array: [{{"source": "entity1", "target": "entity2", "type": "RELATIONSHIP_TYPE", "description": "..."}}]
        
        Common relationship types: MENTIONS, OWNS, WORKS_FOR, LOCATED_IN, PART_OF, RELATES_TO"""
        
        self.context.messages.append({"role": "user", "content": prompt})
        response = self.chat_provider.chat_completion(self.context.messages)
        self.context.messages.append({"role": "assistant", "content": response})
        
        print(f"\n🤖 Suggested relationships:\n{response}")
        
        # Parse suggested relationships
        try:
            import re
            json_match = re.search(r'\[.*\]', response, re.DOTALL)
            if json_match:
                suggested_relationships = json.loads(json_match.group())
                self.context.relationships.extend(suggested_relationships)
        except Exception as e:
            logger.warning(f"Could not parse suggested relationships: {e}")
        
        # Allow user to refine relationships
        while True:
            action = input("\n(a)dd relationship, (d)elete relationship, or (c)ontinue: ").strip().lower()
            
            if action == 'c':
                break
            elif action == 'a':
                print("Available entities:")
                for i, entity in enumerate(self.context.entities):
                    print(f"  {i}: {entity['entity_type']}")
                
                source_idx = int(input("Source entity index: "))
                target_idx = int(input("Target entity index: "))
                
                if 0 <= source_idx < len(self.context.entities) and 0 <= target_idx < len(self.context.entities):
                    source = self.context.entities[source_idx]['entity_type']
                    target = self.context.entities[target_idx]['entity_type']
                    rel_type = input("Relationship type (e.g., MENTIONS, OWNS, WORKS_FOR): ").strip().upper()
                    description = input("Description: ").strip()
                    
                    self.context.relationships.append({
                        "source": source,
                        "target": target,
                        "type": rel_type,
                        "description": description
                    })
                    print(f"✅ Added relationship: {source} -> {rel_type} -> {target}")
            
            elif action == 'd':
                # List relationships
                for i, rel in enumerate(self.context.relationships):
                    print(f"{i}: {rel['source']} -> {rel['type']} -> {rel['target']}")
                
                idx = int(input("Relationship index to delete: "))
                if 0 <= idx < len(self.context.relationships):
                    deleted = self.context.relationships.pop(idx)
                    print(f"✅ Deleted relationship: {deleted['source']} -> {deleted['type']} -> {deleted['target']}")
    
    def _phase_refinement(self):
        """Phase 5: Final refinement and validation."""
        print("\n✨ Phase 5: Refinement")
        print("-" * 40)
        
        # Show summary
        print("\n📊 Ontology Summary:")
        print(f"  Domain: {self.context.domain}")
        print(f"  Document Types: {', '.join(self.context.document_types)}")
        print(f"  Terms: {len(self.context.terms)}")
        print(f"  Entities: {len(self.context.entities)}")
        print(f"  Relationships: {len(self.context.relationships)}")
        
        # Ask for refinements
        refine = input("\nWould you like to refine any aspect? (y/n): ").strip().lower()
        if refine == 'y':
            print("\nWhat would you like to refine?")
            print("1. Terms")
            print("2. Entities")
            print("3. Relationships")
            print("4. Add derived entities")
            
            choice = input("Choice (1-4): ").strip()
            
            if choice == '1':
                self._phase_term_definition()
            elif choice == '2':
                self._phase_entity_extraction()
            elif choice == '3':
                self._phase_relationship_mapping()
            elif choice == '4':
                self._add_derived_entities()
    
    def _add_derived_entities(self):
        """Add derived entities based on combinations of other entities."""
        print("\n🔮 Derived Entities")
        print("-" * 40)
        
        print("Derived entities are created from combinations of other entities.")
        print("Example: A 'Deal' entity derived from Company + TransactionAmount + Date")
        
        add_derived = input("\nAdd a derived entity? (y/n): ").strip().lower()
        if add_derived != 'y':
            return
        
        name = input("Derived entity name: ").strip()
        description = input("Description: ").strip()
        
        print("\nSelect source entities (comma-separated indices):")
        for i, entity in enumerate(self.context.entities):
            print(f"  {i}: {entity['entity_type']}")
        
        indices = input("Indices: ").strip()
        source_entities = []
        for idx_str in indices.split(","):
            idx = int(idx_str.strip())
            if 0 <= idx < len(self.context.entities):
                source_entities.append(self.context.entities[idx]['entity_type'])
        
        if not hasattr(self.context, 'derived_entities'):
            self.context.derived_entities = []
        
        self.context.derived_entities.append({
            "name": name,
            "description": description,
            "source_entities": source_entities,
            "aggregation_type": "COMBINATION"
        })
        
        print(f"✅ Added derived entity: {name}")
    
    def _build_ontology(self) -> Dict[str, Any]:
        """Build the final ontology from interview context."""
        ontology = {
            "name": f"{self.context.domain.lower().replace(' ', '_')}_ontology",
            "version": "1.0.0",
            "description": f"Ontology for {self.context.domain} domain",
            "metadata": {
                "domain": self.context.domain,
                "document_types": self.context.document_types,
                "key_concepts": self.context.key_concepts,
                "created_by": "ontology_generator_cli"
            }
        }
        
        # Add terms
        if self.context.terms:
            ontology["terms"] = self.context.terms
        
        # Convert entities to element_entity_mappings
        if self.context.entities:
            ontology["element_entity_mappings"] = []
            for entity in self.context.entities:
                mapping = {
                    "entity_type": entity["entity_type"],
                    "description": entity.get("description", ""),
                    "element_types": entity.get("element_types", ["paragraph", "heading"]),
                    "extraction_rules": []
                }
                
                # Convert extraction rules
                for rule in entity.get("extraction_rules", []):
                    if rule["type"] == "METADATA":
                        mapping["extraction_rules"].append({
                            "type": "metadata_field",
                            "field_path": rule["path"],
                            "confidence": 0.9
                        })
                    elif rule["type"] == "REGEX":
                        mapping["extraction_rules"].append({
                            "type": "regex_pattern",
                            "pattern": rule["pattern"],
                            "confidence": 0.8
                        })
                    elif rule["type"] == "KEYWORDS":
                        mapping["extraction_rules"].append({
                            "type": "keyword_match",
                            "keywords": rule["keywords"],
                            "confidence": 0.7
                        })
                
                ontology["element_entity_mappings"].append(mapping)
        
        # Add relationships
        if self.context.relationships:
            ontology["entity_relationship_rules"] = []
            for rel in self.context.relationships:
                ontology["entity_relationship_rules"].append({
                    "name": f"{rel['source']}_{rel['type'].lower()}_{rel['target']}",
                    "source_entity_type": rel["source"],
                    "target_entity_type": rel["target"],
                    "relationship_type": rel["type"],
                    "description": rel.get("description", ""),
                    "confidence_threshold": 0.7
                })
        
        # Add derived entities if any
        if hasattr(self.context, 'derived_entities') and self.context.derived_entities:
            ontology["derived_entities"] = self.context.derived_entities
        
        return ontology