"""
YAML configuration loader for entity extractors.

This module allows extractors to be configured and loaded from YAML files,
making the system extensible without code changes.
"""

import re
import yaml
import logging
from typing import Dict, Any, List, Optional, Callable, Union
from pathlib import Path
from .base import EntityExtractor, ExtractorRegistry, RegexExtractor
from .normalizers import normalize, get_normalizer
from .financial import register_financial_extractors
from .temporal import register_temporal_extractors
from .legal import register_legal_extractors
from .medical import register_medical_extractors

logger = logging.getLogger(__name__)


class ConfigurableExtractor(RegexExtractor):
    """Extractor that can be configured from YAML."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize from configuration dictionary.
        
        Args:
            config: Configuration dictionary with:
                - entity_type: Type of entity
                - patterns: List of regex patterns or pattern configs
                - normalizer: Normalizer type or custom function
                - formatter: Formatter type or custom function
                - description: Description of extractor
                - enabled: Whether extractor is enabled
        """
        entity_type = config.get('entity_type', 'custom')
        patterns = self._process_patterns(config.get('patterns', []))
        normalizer = self._get_normalizer_function(config.get('normalizer'))
        formatter = self._get_formatter_function(config.get('formatter'))
        description = config.get('description', f"Configurable extractor for {entity_type}")
        
        super().__init__(
            entity_type=entity_type,
            patterns=patterns,
            normalizer=normalizer,
            formatter=formatter,
            description=description,
            config=config
        )
        
        self.enabled = config.get('enabled', True)
        self.confidence = config.get('confidence', 1.0)
    
    def _process_patterns(self, patterns_config: List[Union[str, Dict]]) -> List[str]:
        """Process pattern configuration."""
        processed = []
        
        for pattern in patterns_config:
            if isinstance(pattern, str):
                processed.append(pattern)
            elif isinstance(pattern, dict):
                # Pattern with options
                regex = pattern.get('regex')
                if regex:
                    # Apply modifiers if specified
                    if pattern.get('word_boundaries', True):
                        regex = r'\b' + regex + r'\b'
                    if pattern.get('case_sensitive', False):
                        # Mark for case-sensitive compilation
                        regex = f"(?-i:{regex})"
                    processed.append(regex)
        
        return processed
    
    def _get_normalizer_function(self, normalizer_config: Union[str, Dict, None]) -> Callable:
        """Get normalizer function from configuration."""
        if not normalizer_config:
            return str.lower
        
        if isinstance(normalizer_config, str):
            # Built-in normalizers
            normalizers = {
                'lower': str.lower,
                'upper': str.upper,
                'title': str.title,
                'strip': str.strip,
                'remove_spaces': lambda x: x.replace(' ', ''),
                'underscore': lambda x: x.replace(' ', '_'),
                'dash': lambda x: x.replace(' ', '-'),
                'entity': lambda x: normalize(x, self._entity_type),
            }
            
            if normalizer_config in normalizers:
                return normalizers[normalizer_config]
            
            # Try to get from normalizer registry
            return get_normalizer(normalizer_config)
        
        elif isinstance(normalizer_config, dict):
            # Composite normalizer
            steps = normalizer_config.get('steps', [])
            
            def composite_normalizer(value: str) -> str:
                result = value
                for step in steps:
                    if step == 'lower':
                        result = result.lower()
                    elif step == 'upper':
                        result = result.upper()
                    elif step == 'strip':
                        result = result.strip()
                    elif step == 'remove_spaces':
                        result = result.replace(' ', '')
                    elif step == 'remove_punctuation':
                        result = re.sub(r'[^\w\s]', '', result)
                    elif isinstance(step, dict):
                        # Regex replacement
                        pattern = step.get('replace_pattern')
                        replacement = step.get('with', '')
                        if pattern:
                            result = re.sub(pattern, replacement, result)
                return result
            
            return composite_normalizer
        
        return str.lower
    
    def _get_formatter_function(self, formatter_config: Union[str, None]) -> Callable:
        """Get formatter function from configuration."""
        if not formatter_config:
            return str
        
        formatters = {
            'lower': str.lower,
            'upper': str.upper,
            'title': str.title,
            'capitalize': str.capitalize,
            'strip': str.strip,
        }
        
        return formatters.get(formatter_config, str)


class ExtractorConfigLoader:
    """Loads and manages extractor configurations from YAML."""
    
    def __init__(self, registry: Optional[ExtractorRegistry] = None):
        """
        Initialize config loader.
        
        Args:
            registry: Extractor registry to populate
        """
        self.registry = registry or ExtractorRegistry()
        self._loaded_configs: Dict[str, Dict] = {}
    
    def load_config_file(self, config_path: Union[str, Path]) -> None:
        """
        Load extractor configurations from YAML file.
        
        Args:
            config_path: Path to YAML configuration file
        """
        config_path = Path(config_path)
        
        if not config_path.exists():
            logger.warning(f"Configuration file not found: {config_path}")
            return
        
        try:
            with open(config_path, 'r') as f:
                config = yaml.safe_load(f)
            
            if config:
                self.load_config(config, source=str(config_path))
                logger.info(f"Loaded extractor configuration from {config_path}")
        
        except Exception as e:
            logger.error(f"Error loading configuration from {config_path}: {e}")
    
    def load_config(self, config: Dict[str, Any], source: str = "inline") -> None:
        """
        Load extractor configuration from dictionary.
        
        Args:
            config: Configuration dictionary
            source: Source identifier for debugging
        """
        # Store config for reference
        self._loaded_configs[source] = config
        
        # Load built-in extractors if enabled
        if config.get('load_builtin', True):
            self._load_builtin_extractors(config.get('builtin', {}))
        
        # Load custom extractors
        custom_extractors = config.get('custom_extractors', [])
        for extractor_config in custom_extractors:
            self._load_custom_extractor(extractor_config)
        
        # Load regex-based extractors
        regex_extractors = config.get('regex_extractors', [])
        for extractor_config in regex_extractors:
            self._load_regex_extractor(extractor_config)
    
    def _load_builtin_extractors(self, builtin_config: Dict[str, Any]) -> None:
        """Load built-in extractors based on configuration."""
        # Financial extractors
        if builtin_config.get('financial', {}).get('enabled', True):
            register_financial_extractors(self.registry)
            logger.debug("Loaded financial extractors")
        
        # Temporal extractors
        if builtin_config.get('temporal', {}).get('enabled', True):
            register_temporal_extractors(self.registry)
            logger.debug("Loaded temporal extractors")
        
        # Legal extractors
        if builtin_config.get('legal', {}).get('enabled', True):
            register_legal_extractors(self.registry)
            logger.debug("Loaded legal extractors")
        
        # Medical extractors
        if builtin_config.get('medical', {}).get('enabled', True):
            register_medical_extractors(self.registry)
            logger.debug("Loaded medical extractors")
    
    def _load_custom_extractor(self, config: Dict[str, Any]) -> None:
        """Load a custom extractor from configuration."""
        name = config.get('name')
        if not name:
            logger.warning("Custom extractor missing 'name' field")
            return
        
        try:
            extractor = ConfigurableExtractor(config)
            self.registry.register(name, extractor)
            logger.debug(f"Loaded custom extractor: {name}")
        
        except Exception as e:
            logger.error(f"Error loading custom extractor '{name}': {e}")
    
    def _load_regex_extractor(self, config: Dict[str, Any]) -> None:
        """Load a simple regex-based extractor."""
        name = config.get('name')
        if not name:
            logger.warning("Regex extractor missing 'name' field")
            return
        
        entity_type = config.get('entity_type', name)
        patterns = config.get('patterns', [])
        
        if not patterns:
            logger.warning(f"Regex extractor '{name}' has no patterns")
            return
        
        try:
            # Create simple regex extractor
            extractor = RegexExtractor(
                entity_type=entity_type,
                patterns=patterns,
                normalizer=self._create_normalizer(config.get('normalizer')),
                formatter=self._create_formatter(config.get('formatter')),
                description=config.get('description', f"Regex extractor for {entity_type}"),
                config=config
            )
            
            self.registry.register(name, extractor)
            logger.debug(f"Loaded regex extractor: {name}")
        
        except Exception as e:
            logger.error(f"Error loading regex extractor '{name}': {e}")
    
    def _create_normalizer(self, config: Union[str, Dict, None]) -> Callable:
        """Create normalizer function from configuration."""
        if not config:
            return str.lower
        
        if isinstance(config, str):
            # Simple normalizer
            if config == 'lower':
                return str.lower
            elif config == 'upper':
                return str.upper
            elif config == 'none':
                return lambda x: x
        
        return str.lower
    
    def _create_formatter(self, config: Union[str, None]) -> Callable:
        """Create formatter function from configuration."""
        if not config:
            return str
        
        if config == 'upper':
            return str.upper
        elif config == 'lower':
            return str.lower
        elif config == 'title':
            return str.title
        
        return str
    
    def get_registry(self) -> ExtractorRegistry:
        """Get the populated extractor registry."""
        return self.registry


def load_extractors_from_yaml(config_path: Union[str, Path]) -> ExtractorRegistry:
    """
    Convenience function to load extractors from YAML file.
    
    Args:
        config_path: Path to YAML configuration file
        
    Returns:
        Populated ExtractorRegistry
    """
    loader = ExtractorConfigLoader()
    loader.load_config_file(config_path)
    return loader.get_registry()


def create_default_config() -> Dict[str, Any]:
    """
    Create default extractor configuration.
    
    Returns:
        Default configuration dictionary
    """
    return {
        'load_builtin': True,
        'builtin': {
            'financial': {'enabled': True},
            'temporal': {'enabled': True},
            'legal': {'enabled': True},
            'medical': {'enabled': True},
        },
        'custom_extractors': [],
        'regex_extractors': []
    }


# Example YAML configuration format
EXAMPLE_CONFIG = """
# Entity Extractor Configuration

# Load built-in extractors
load_builtin: true
builtin:
  financial:
    enabled: true
  temporal:
    enabled: true
  legal:
    enabled: true
  medical:
    enabled: true

# Custom extractors with full configuration
custom_extractors:
  - name: product_code
    entity_type: product_code
    description: Extract product codes
    patterns:
      - regex: '[A-Z]{3}-\\d{4}-[A-Z]'
        word_boundaries: true
      - regex: 'SKU\\s*:\\s*\\w+'
    normalizer: upper
    formatter: upper
    confidence: 0.9
    enabled: true

  - name: email_address
    entity_type: email
    patterns:
      - '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}'
    normalizer:
      steps:
        - lower
        - strip
    formatter: lower

# Simple regex-based extractors
regex_extractors:
  - name: ssn
    entity_type: social_security_number
    patterns:
      - '\\d{3}-\\d{2}-\\d{4}'
    normalizer: none
    description: US Social Security Numbers

  - name: vin
    entity_type: vehicle_identification_number
    patterns:
      - '[A-HJ-NPR-Z0-9]{17}'
    normalizer: upper
    description: Vehicle Identification Numbers

  - name: isbn
    entity_type: isbn
    patterns:
      - 'ISBN(?:-13)?:\\s*978-\\d{1,5}-\\d{1,7}-\\d{1,7}-\\d'
      - 'ISBN(?:-10)?:\\s*\\d{1,5}-\\d{1,7}-\\d{1,7}-[\\dX]'
    normalizer: 
      steps:
        - upper
        - remove_spaces
    description: ISBN book identifiers
"""