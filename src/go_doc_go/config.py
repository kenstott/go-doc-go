import json
import logging
import os
import re
from typing import Dict, Any, List, Optional

from dotenv import load_dotenv

load_dotenv()

import yaml

logger = logging.getLogger(__name__)


class Config:
    """Configuration manager for the document pointer system."""

    def __init__(self, config_path: Optional[str] = None):
        """
        Initialize configuration from file or default settings.

        Args:
            config_path: Path to configuration file (JSON or YAML)
        """
        self.config = self._get_default_config()
        self._db_instance = None  # Add a database instance cache
        self._ontology_manager = None  # Cache for ontology manager
        self._analytics_registry = None  # Cache for analytics registry

        logger.debug(f"Initializing config, working directory: {os.getcwd()}")

        if config_path and os.path.exists(config_path):
            logger.debug(f"Loading config from: {config_path}")
            self._load_config(config_path)
        else:
            logger.debug("No config path provided or file not found, using defaults")

        # Load analytics registry if it exists
        self._load_analytics_registry()

        self._validate_config()

    @staticmethod
    def _get_default_config() -> Dict[str, Any]:
        """Get default configuration settings."""
        return {
            "storage": {
                "path": "./data",
                "backend": "file",  # Options: file, sqlite, duckdb
                "topic_support": False  # NEW: Enable topic features
            },
            "embedding": {
                "enabled": False,
                "model": "sentence-transformers/all-MiniLM-L6-v2",
                "chunk_size": 512,
                "overlap": 128,
                "max_tokens": 16384
            },
            "content_sources": [],
            "relationship_detection": {
                "enabled": True,
                "link_pattern": r"\[\[(.*?)\]\]|href=[\"\'](.*?)[\"\']"
            },
            "logging": {
                "level": "INFO",
                "file": "./logs/docpointer.log"
            }
        }

    def _replace_env_vars(self, value: Any) -> Any:
        """
        Replace environment variables in string values.

        Args:
            value: The value to process

        Returns:
            The processed value with environment variables replaced
        """
        if isinstance(value, str):
            # Match ${VAR} or $VAR patterns
            pattern = r'\${([^}]+)}|\$([a-zA-Z0-9_]+)'

            def replace_match(match):
                env_var = match.group(1) or match.group(2)
                default_value = None

                # Handle default values with ${VAR:-default} syntax
                if ':-' in env_var:
                    env_var, default_value = env_var.split(':-', 1)

                env_value = os.environ.get(env_var)
                if env_value is None:
                    if default_value is not None:
                        logger.debug(f"Environment variable {env_var} not found, using default: {default_value}")
                        return default_value
                    logger.warning(f"Environment variable {env_var} not found and no default provided")
                    return match.group(0)  # Return the original placeholder if no value found

                logger.debug(f"Replaced environment variable {env_var}")
                return env_value

            return re.sub(pattern, replace_match, value)
        elif isinstance(value, dict):
            return {k: self._replace_env_vars(v) for k, v in value.items()}
        elif isinstance(value, list):
            return [self._replace_env_vars(item) for item in value]
        return value

    def _load_config(self, config_path: str) -> None:
        """
        Load configuration from file.

        Args:
            config_path: Path to configuration file
        """
        try:
            with open(config_path, 'r') as f:
                if config_path.endswith('.json'):
                    loaded_config = json.load(f)
                    logger.debug("Loaded JSON config")
                elif config_path.endswith(('.yaml', '.yml')):
                    loaded_config = yaml.safe_load(f)
                    logger.debug("Loaded YAML config")
                else:
                    message = f"Unsupported config file format: {config_path}"
                    logger.error(message)
                    raise ValueError(message)

            # Replace environment variables in the loaded config
            loaded_config = self._replace_env_vars(loaded_config)
            logger.debug("Replaced environment variables in config")

            # Merge with default config (deep merge)
            self._deep_merge(self.config, loaded_config)
            logger.debug("Config merged with defaults")
        except Exception as e:
            logger.error(f"Error loading config from {config_path}: {str(e)}")
            raise

    def _deep_merge(self, dest: Dict[str, Any], src: Dict[str, Any]) -> None:
        """
        Deep merge two dictionaries.

        Args:
            dest: Destination dictionary (modified in-place)
            src: Source dictionary
        """
        for key, value in src.items():
            if key in dest and isinstance(dest[key], dict) and isinstance(value, dict):
                self._deep_merge(dest[key], value)
            else:
                dest[key] = value

    def _resolve_path_from_project_root(self, path: str) -> str:
        """
        Resolve a path relative to the project root.
        
        Args:
            path: Path to resolve (can be relative or absolute)
            
        Returns:
            Absolute path resolved from project root
        """
        if os.path.isabs(path):
            return path
        
        # Check for project root environment variable
        project_root = os.environ.get('GO_DOC_GO_PROJECT_ROOT')
        if project_root:
            return os.path.join(project_root, path)
        
        # Fallback to current working directory
        return os.path.abspath(path)
    
    def _validate_config(self) -> None:
        """Validate configuration settings."""
        # Ensure storage path exists or create it
        storage_path = self.get_storage_path()
        # Resolve storage path relative to project root
        storage_path = self._resolve_path_from_project_root(storage_path)
        logger.debug(f"Ensuring storage path exists: {storage_path}")
        # Only create directory if it's not a file path (e.g., SQLite db)
        if not storage_path.endswith('.db'):
            os.makedirs(storage_path, exist_ok=True)
        else:
            # For SQLite db files, ensure the parent directory exists
            parent_dir = os.path.dirname(storage_path)
            if parent_dir and parent_dir != '.':
                os.makedirs(parent_dir, exist_ok=True)

        # Ensure logs directory exists
        log_file_path = self.config["logging"]["file"]
        # Resolve log file path relative to project root
        log_file_path = self._resolve_path_from_project_root(log_file_path)
        log_dir = os.path.dirname(log_file_path)
        if log_dir:
            logger.debug(f"Ensuring log directory exists: {log_dir}")
            os.makedirs(log_dir, exist_ok=True)

        # Validate each content source has required fields
        for idx, source in enumerate(self.config.get("content_sources", [])):
            if "type" not in source:
                message = f"Content source at index {idx} is missing 'type' field"
                logger.error(message)
                raise ValueError(message)
            if "name" not in source:
                message = f"Content source at index {idx} is missing 'name' field"
                logger.error(message)
                raise ValueError(message)

            # NEW: Validate topics field if present
            topics = source.get("topics")
            if topics is not None and not isinstance(topics, list):
                message = f"Content source at index {idx}: 'topics' must be a list of strings"
                logger.error(message)
                raise ValueError(message)

        logger.debug("Config validation complete")

    def get_storage_path(self) -> str:
        """Get the storage path for document database."""
        path = self.config.get("storage", {}).get("path", "./data")
        path = os.path.expanduser(path)
        return self._resolve_path_from_project_root(path)

    def get_storage_backend(self) -> str:
        """Get the storage backend type."""
        return self.config["storage"]["backend"]

    # NEW: Check if topic support is enabled
    def is_topic_support_enabled(self) -> bool:
        """Check if topic support is enabled."""
        return self.config.get("storage", {}).get("topic_support", False)

    def is_embedding_enabled(self) -> bool:
        """Check if embeddings are enabled."""
        return self.config["embedding"]["enabled"]

    def get_embedding_model(self) -> str:
        """Get the embedding model name."""
        return self.config["embedding"]["model"]

    def get_embedding_params(self) -> Dict[str, Any]:
        """Get embedding parameters."""
        return self.config["embedding"]
    
    def get_embedding_max_tokens(self) -> int:
        """Get the maximum tokens for contextual embeddings."""
        return self.config["embedding"].get("max_tokens", 16384)

    def get_content_sources(self) -> List[Dict[str, Any]]:
        """Get configured content sources."""
        return self.config["content_sources"]
    
    @property
    def content_sources(self) -> List[Dict[str, Any]]:
        """Get configured content sources (property accessor)."""
        return self.get_content_sources()

    # NEW: Get topics for a content source
    def get_source_topics(self, source_name: str) -> List[str]:
        """
        Get topics configured for a specific content source.

        Args:
            source_name: Name of the content source

        Returns:
            List of topic strings, empty list if no topics configured
        """
        for source in self.get_content_sources():
            if source.get("name") == source_name:
                return source.get("topics", [])
        return []

    def get_relationship_detection_config(self) -> Dict[str, Any]:
        """Get relationship detection configuration."""
        return self.config["relationship_detection"]
    
    @property
    def relationship_detection(self) -> Dict[str, Any]:
        """Get relationship detection configuration (property accessor)."""
        return self.get_relationship_detection_config()

    def get_logging_config(self) -> Dict[str, Any]:
        """Get logging configuration."""
        return self.config["logging"]

    
    def _load_analytics_registry(self):
        """
        Load the analytics registry configuration.
        
        Looks for analytics_registry.yaml in the current directory or 
        from the ANALYTICS_REGISTRY_PATH environment variable.
        """
        registry_paths = [
            os.environ.get('ANALYTICS_REGISTRY_PATH', ''),
            'analytics_registry.yaml',
            'analytics_registry.yml',
            os.path.join(os.path.dirname(__file__), '../../analytics_registry.yaml')
        ]
        
        for path in registry_paths:
            if path and os.path.exists(path):
                logger.debug(f"Loading analytics registry from: {path}")
                try:
                    with open(path, 'r') as f:
                        registry_data = yaml.safe_load(f)
                        # Replace environment variables
                        registry_data = self._replace_env_vars(registry_data)
                        
                        if 'analytics_registry' in registry_data:
                            self._analytics_registry = registry_data['analytics_registry']
                            logger.info(f"Loaded {len(self._analytics_registry)} analytics backends from registry")
                            
                            # Merge search config if present
                            if 'search' in registry_data:
                                if 'search' not in self.config:
                                    self.config['search'] = {}
                                self.config['search'].update(registry_data['search'])
                                logger.debug(f"Updated search config from registry")
                            
                            # Merge processing defaults if present
                            if 'processing' in registry_data:
                                if 'processing' not in self.config:
                                    self.config['processing'] = {}
                                self.config['processing'].update(registry_data['processing'])
                                logger.debug(f"Updated processing config from registry")
                            
                            # Merge agent selection rules if present
                            if 'agent_selection_rules' in registry_data:
                                self.config['agent_selection_rules'] = registry_data['agent_selection_rules']
                                logger.debug(f"Loaded agent selection rules from registry")
                        else:
                            logger.warning(f"Analytics registry file found but no 'analytics_registry' section")
                    return
                except Exception as e:
                    logger.error(f"Error loading analytics registry from {path}: {e}")
        
        logger.debug("No analytics registry file found, analytics registry features disabled")
    
    def get_analytics_backend(self, name: str) -> Optional[Dict[str, Any]]:
        """
        Get analytics backend configuration by name.
        
        Args:
            name: Name of the backend in the registry
            
        Returns:
            Backend configuration dict or None if not found
        """
        if not self._analytics_registry:
            logger.warning("Analytics registry not loaded")
            return None
        
        if name not in self._analytics_registry:
            logger.warning(f"Analytics backend '{name}' not found in registry")
            return None
        
        return self._analytics_registry[name]
    
    def list_analytics_backends(self) -> Dict[str, Dict[str, Any]]:
        """
        List all available analytics backends.
        
        Returns:
            Dictionary of backend names to their configurations
        """
        if not self._analytics_registry:
            return {}
        
        return self._analytics_registry.copy()
    
    def get_search_backend(self) -> Optional[str]:
        """
        Get the default search backend name.
        
        Returns:
            Name of the default search backend or None
        """
        search_config = self.config.get('search', {})
        return search_config.get('default_backend')
    
    def get_search_backend_config(self, backend_name: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Get search backend configuration.
        
        Args:
            backend_name: Optional backend name, uses default if not specified
            
        Returns:
            Backend configuration dict or None if not found
        """
        if not backend_name:
            backend_name = self.get_search_backend()
        
        if not backend_name:
            # Fall back to old storage.backend for compatibility
            return self.config.get('storage', {})
        
        return self.get_analytics_backend(backend_name)
    
    def get_ontology_manager(self):
        """
        Get the domain ontology manager singleton instance.
        
        Returns:
            OntologyManager instance or None if domain detection is disabled
        """
        domain_config = self.config.get("relationship_detection", {}).get("domain", {})
        
        if not domain_config.get("enabled", False):
            return None
            
        if self._ontology_manager is None:
            logger.debug("Creating ontology manager")
            from .domain import OntologyManager
            
            self._ontology_manager = OntologyManager()
            
            # Load configured ontologies
            ontologies = domain_config.get("ontologies", [])
            for ontology_config in ontologies:
                if not isinstance(ontology_config, dict):
                    continue
                    
                path = ontology_config.get("path")
                active = ontology_config.get("active", True)
                
                if path:
                    path = self._resolve_path_from_project_root(path)
                    if os.path.exists(path):
                        try:
                            name = self._ontology_manager.load_ontology(path)
                            if active:
                                self._ontology_manager.activate_domain(name)
                                logger.info(f"Loaded and activated domain ontology: {name}")
                            else:
                                logger.info(f"Loaded domain ontology (inactive): {name}")
                        except Exception as e:
                            logger.error(f"Failed to load ontology from {path}: {e}")
                    else:
                        logger.warning(f"Ontology file not found: {path}")
        
        return self._ontology_manager
    
    def is_domain_detection_enabled(self) -> bool:
        """Check if domain entity extraction and relationship detection is enabled."""
        return self.config.get("relationship_detection", {}).get("domain", {}).get("enabled", False)
    
    def get_entity_extraction_config(self) -> Dict[str, Any]:
        """Get entity extraction configuration."""
        return self.config.get("entity_extraction", {})
    
    def is_entity_extraction_enabled(self) -> bool:
        """Check if entity extraction is enabled."""
        return self.config.get("entity_extraction", {}).get("enabled", False)
    
    def get_extractor_registry(self):
        """
        Get the entity extractor registry.
        
        Returns:
            ExtractorRegistry instance or None if entity extraction is disabled
        """
        if not self.is_entity_extraction_enabled():
            return None
        
        from .extractors.config_loader import ExtractorConfigLoader
        
        loader = ExtractorConfigLoader()
        extraction_config = self.get_entity_extraction_config()
        
        # Load configuration
        loader.load_config(extraction_config)
        
        # Load from external file if specified
        config_file = extraction_config.get('config_file')
        if config_file and os.path.exists(config_file):
            loader.load_config_file(config_file)
        
        return loader.get_registry()

    def add_content_source(self, source_config: Dict[str, Any]) -> None:
        """
        Add a new content source to the configuration.

        Args:
            source_config: Content source configuration
        """
        if "type" not in source_config:
            raise ValueError("Content source is missing 'type' field")
        if "name" not in source_config:
            raise ValueError("Content source is missing 'name' field")

        self.config["content_sources"].append(source_config)
        logger.debug(f"Added content source: {source_config['name']} ({source_config['type']})")

    def get(self, key: str, default: Any = None) -> Any:
        """
        Get configuration value by key.
        
        Args:
            key: Configuration key
            default: Default value if key not found
            
        Returns:
            Configuration value or default
        """
        return self.config.get(key, default)
    
    def save(self, path: str) -> None:
        """
        Save current configuration to file.

        Args:
            path: Path to save configuration file
        """
        logger.debug(f"Saving config to: {path}")
        try:
            with open(path, 'w') as f:
                if path.endswith('.json'):
                    json.dump(self.config, f, indent=2)
                    logger.debug("Saved config as JSON")
                elif path.endswith(('.yaml', '.yml')):
                    yaml.dump(self.config, f, default_flow_style=False)
                    logger.debug("Saved config as YAML")
                else:
                    message = f"Unsupported config file format: {path}"
                    logger.error(message)
                    raise ValueError(message)
        except Exception as e:
            logger.error(f"Error saving config to {path}: {str(e)}")
            raise
