"""
Loader for domain ontology configurations.
"""
import json
import yaml
from pathlib import Path
from typing import Dict, Any, List, Optional, Union
import logging

from .ontology import DomainOntology

logger = logging.getLogger(__name__)


class OntologyLoader:
    """Loads and manages domain ontology configurations."""
    
    def __init__(self):
        """Initialize the ontology loader."""
        self.ontologies: Dict[str, DomainOntology] = {}
    
    def load_from_file(self, file_path: Union[str, Path]) -> DomainOntology:
        """
        Load ontology from a YAML or JSON file.
        
        Args:
            file_path: Path to the configuration file
            
        Returns:
            Loaded DomainOntology
            
        Raises:
            ValueError: If file format is not supported or configuration is invalid
        """
        file_path = Path(file_path)
        
        if not file_path.exists():
            raise FileNotFoundError(f"Ontology file not found: {file_path}")
        
        # Load file based on extension
        if file_path.suffix.lower() in ['.yaml', '.yml']:
            data = self._load_yaml(file_path)
        elif file_path.suffix.lower() == '.json':
            data = self._load_json(file_path)
        else:
            raise ValueError(f"Unsupported file format: {file_path.suffix}")
        
        # Create ontology from data
        ontology = self.load_from_dict(data)
        
        # Store by name
        self.ontologies[ontology.name] = ontology
        
        logger.info(f"Loaded ontology '{ontology.name}' v{ontology.version} from {file_path}")
        
        return ontology
    
    def load_from_dict(self, data: Dict[str, Any]) -> DomainOntology:
        """
        Load ontology from a dictionary.
        
        Args:
            data: Ontology configuration as dictionary
            
        Returns:
            Loaded DomainOntology
            
        Raises:
            ValueError: If configuration is invalid
        """
        # Create ontology
        ontology = DomainOntology.from_dict(data)
        
        # Validate
        issues = ontology.validate()
        if issues:
            logger.warning(f"Ontology validation issues: {issues}")
            # Optionally raise exception
            # raise ValueError(f"Invalid ontology configuration: {'; '.join(issues)}")
        
        return ontology
    
    def load_from_string(self, content: str, format: str = 'yaml') -> DomainOntology:
        """
        Load ontology from a string.
        
        Args:
            content: Ontology configuration as string
            format: Format of the content ('yaml' or 'json')
            
        Returns:
            Loaded DomainOntology
        """
        if format.lower() in ['yaml', 'yml']:
            data = yaml.safe_load(content)
        elif format.lower() == 'json':
            data = json.loads(content)
        else:
            raise ValueError(f"Unsupported format: {format}")
        
        return self.load_from_dict(data)
    
    def load_directory(self, directory: Union[str, Path], 
                      pattern: str = "*.yaml") -> List[DomainOntology]:
        """
        Load all ontology files from a directory.
        
        Args:
            directory: Directory containing ontology files
            pattern: Glob pattern for files to load
            
        Returns:
            List of loaded ontologies
        """
        directory = Path(directory)
        ontologies = []
        
        for file_path in directory.glob(pattern):
            try:
                ontology = self.load_from_file(file_path)
                ontologies.append(ontology)
            except Exception as e:
                logger.error(f"Failed to load ontology from {file_path}: {e}")
        
        return ontologies
    
    def get_ontology(self, name: str) -> Optional[DomainOntology]:
        """Get loaded ontology by name."""
        return self.ontologies.get(name)
    
    def list_ontologies(self) -> List[str]:
        """List names of all loaded ontologies."""
        return list(self.ontologies.keys())
    
    def clear(self):
        """Clear all loaded ontologies."""
        self.ontologies.clear()
    
    @staticmethod
    def _load_yaml(file_path: Path) -> Dict[str, Any]:
        """Load YAML file."""
        with open(file_path, 'r', encoding='utf-8') as f:
            return yaml.safe_load(f)
    
    @staticmethod
    def _load_json(file_path: Path) -> Dict[str, Any]:
        """Load JSON file."""
        with open(file_path, 'r', encoding='utf-8') as f:
            return json.load(f)


class OntologyManager:
    """
    Manages multiple domain ontologies and provides unified access.
    """
    
    def __init__(self):
        """Initialize the ontology manager."""
        self.loader = OntologyLoader()
        self.active_domains: List[str] = []
    
    def load_ontology(self, file_path: Union[str, Path]) -> str:
        """
        Load an ontology and return its name.
        
        Args:
            file_path: Path to ontology configuration file
            
        Returns:
            Name of the loaded ontology
        """
        ontology = self.loader.load_from_file(file_path)
        return ontology.name
    
    def activate_domain(self, domain_name: str):
        """Activate a domain for use."""
        if domain_name not in self.loader.ontologies:
            raise ValueError(f"Domain '{domain_name}' not loaded")
        
        if domain_name not in self.active_domains:
            self.active_domains.append(domain_name)
            logger.info(f"Activated domain: {domain_name}")
    
    def deactivate_domain(self, domain_name: str):
        """Deactivate a domain."""
        if domain_name in self.active_domains:
            self.active_domains.remove(domain_name)
            logger.info(f"Deactivated domain: {domain_name}")
    
    def get_active_ontologies(self) -> List[DomainOntology]:
        """Get all active domain ontologies."""
        return [
            self.loader.get_ontology(name) 
            for name in self.active_domains
            if self.loader.get_ontology(name)
        ]
    
    def get_all_terms(self) -> Dict[str, List[str]]:
        """
        Get all terms from active domains.
        
        Returns:
            Dictionary mapping domain name to list of term IDs
        """
        result = {}
        for domain_name in self.active_domains:
            ontology = self.loader.get_ontology(domain_name)
            if ontology:
                result[domain_name] = [term.id for term in ontology.terms]
        return result
    
    def find_term(self, term_id: str) -> Optional[tuple[str, Any]]:
        """
        Find a term across all active domains.
        
        Args:
            term_id: Term ID to search for
            
        Returns:
            Tuple of (domain_name, term) if found, None otherwise
        """
        for domain_name in self.active_domains:
            ontology = self.loader.get_ontology(domain_name)
            if ontology:
                term = ontology.get_term(term_id)
                if term:
                    return (domain_name, term)
        return None