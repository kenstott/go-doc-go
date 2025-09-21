"""
Embedding Generator Module for the document pointer system.
This module generates vector embeddings for document elements,
with support for different embedding models and simple contextual embeddings.
"""
import logging
from .base import EmbeddingGenerator
from .contextual_embedding import ContextualEmbeddingGenerator
from ..config import Config

logger = logging.getLogger(__name__)

# Conditionally import embedding providers
HUGGINGFACE_AVAILABLE = False
OPENAI_AVAILABLE = False
FASTEMBED_AVAILABLE = False

# Import HuggingFace provider if available
try:
    from .hugging_face import HuggingFaceEmbeddingGenerator
    HUGGINGFACE_AVAILABLE = True
except ImportError:
    logger.warning("HuggingFace (sentence-transformers) not available. Install with: pip install sentence-transformers")

# Import OpenAI provider if available
try:
    from .openai import OpenAIEmbeddingGenerator
    OPENAI_AVAILABLE = True
except ImportError:
    logger.warning("OpenAI not available. Install with: pip install openai")

# Import FastEmbed provider if available
try:
    from .fastembed import FastEmbedGenerator
    FASTEMBED_AVAILABLE = True
except ImportError:
    logger.warning("FastEmbed not available. Install with: pip install fastembed")


def get_embedding_generator(config: Config) -> EmbeddingGenerator:
    """
    Factory function to create embedding generator from configuration.

    Args:
        config: Configuration object

    Returns:
        EmbeddingGenerator instance
    """
    embeddings = config.config.get("embedding", {})
    if not embeddings:
        raise ValueError("No embedding configuration found in config")

    # Provider is REQUIRED
    provider = embeddings.get("provider")
    if not provider:
        raise ValueError(f"Missing required 'provider' in embedding config. Config: {embeddings}")
    provider = provider.lower()

    # Get optional dimensions configuration
    dimensions = embeddings.get("dimensions", None)

    # Create base generator based on provider
    if provider == "openai":
        if not OPENAI_AVAILABLE:
            raise ImportError("OpenAI library is required for OpenAI embeddings. Install with: pip install openai")

        # Get OpenAI-specific config
        model = embeddings.get("model", "text-embedding-3-small")
        api_key = embeddings.get("api_key", None)

        # Create OpenAI generator
        base_generator = OpenAIEmbeddingGenerator(config, model, api_key, dimensions)
        logger.info(f"Created OpenAI embedding generator with model {model}")

    elif provider == "fastembed":
        if not FASTEMBED_AVAILABLE:
            raise ImportError("FastEmbed library is required for FastEmbed embeddings. Install with: pip install fastembed")

        # Model is REQUIRED for FastEmbed
        model = embeddings.get("model")
        if not model:
            raise ValueError(f"Missing required 'model' for FastEmbed provider. Config: {embeddings}")
        cache_dir = embeddings.get("cache_dir", None)

        # Create FastEmbed generator
        base_generator = FastEmbedGenerator(config, model, dimensions, cache_dir)
        logger.info(f"Created FastEmbed embedding generator with model {model}")

    else:
        # Default to Hugging Face
        if not HUGGINGFACE_AVAILABLE:
            raise ImportError("Sentence-Transformers library is required for HuggingFace embeddings. Install with: pip install sentence-transformers")

        model = embeddings.get("model", "sentence-transformers/all-MiniLM-L6-v2")

        # Create Hugging Face generator
        base_generator = HuggingFaceEmbeddingGenerator(config, model)
        logger.info(f"Created Hugging Face embedding generator with model {model}")

    # Add simple contextual embedding if configured
    if embeddings.get("contextual", False):
        window_size = embeddings.get("window_size", 3)
        overlap_size = embeddings.get("overlap_size", 1)
        predecessor_count = embeddings.get("predecessor_count", 1)
        successor_count = embeddings.get("successor_count", 1)
        ancestor_depth = embeddings.get("ancestor_depth", 1)
        max_tokens = config.get_embedding_max_tokens()

        contextual_generator = ContextualEmbeddingGenerator(
            _config=config,
            base_generator=base_generator,
            window_size=window_size,
            overlap_size=overlap_size,
            predecessor_count=predecessor_count,
            successor_count=successor_count,
            ancestor_depth=ancestor_depth,
            max_tokens=max_tokens,
            use_semantic_tags=False  # Simple text approach
        )
        logger.info("Added simple contextual embedding wrapper")
        return contextual_generator

    return base_generator


def get_embedder_from_analytics_registry(backend_name: str, embedding_config: dict, config: Config) -> EmbeddingGenerator:
    """
    Factory function to create embedding generator from analytics registry backend configuration.

    Args:
        backend_name: Name of the analytics backend
        embedding_config: Embedding configuration from analytics registry backend
        config: Configuration object

    Returns:
        EmbeddingGenerator instance

    Raises:
        ImportError: If required embedding library is not available
        ValueError: If configuration is invalid
    """
    logger.info(f"Creating embedder for analytics backend '{backend_name}' from registry config")

    if not embedding_config:
        raise ValueError(f"No embedding configuration provided for analytics backend '{backend_name}'")

    # Provider is REQUIRED
    provider = embedding_config.get("provider")
    if not provider:
        raise ValueError(f"Missing required 'provider' in embedding config for backend '{backend_name}'. Config: {embedding_config}")
    provider = provider.lower()
    
    # Get optional dimensions configuration
    dimensions = embedding_config.get("dimensions", None)
    
    # Create base generator based on provider
    if provider == "openai":
        if not OPENAI_AVAILABLE:
            raise ImportError("OpenAI library is required for OpenAI embeddings. Install with: pip install openai")

        # Get OpenAI-specific config
        model = embedding_config.get("model", "text-embedding-3-small")
        api_key = embedding_config.get("api_key", None)

        # Create OpenAI generator
        base_generator = OpenAIEmbeddingGenerator(config, model, api_key, dimensions)
        logger.info(f"Created OpenAI embedding generator for backend '{backend_name}' with model {model}")

    elif provider == "fastembed":
        if not FASTEMBED_AVAILABLE:
            raise ImportError("FastEmbed library is required for FastEmbed embeddings. Install with: pip install fastembed")

        # Model is REQUIRED for FastEmbed
        model = embedding_config.get("model")
        if not model:
            raise ValueError(f"Missing required 'model' for FastEmbed provider in backend '{backend_name}'. Config: {embedding_config}")
        cache_dir = embedding_config.get("cache_dir", None)

        # Create FastEmbed generator
        base_generator = FastEmbedGenerator(config, model, dimensions, cache_dir)
        logger.info(f"Created FastEmbed embedding generator for backend '{backend_name}' with model {model}")

    elif provider == "huggingface":
        # HuggingFace provider
        if not HUGGINGFACE_AVAILABLE:
            raise ImportError("Sentence-Transformers library is required for HuggingFace embeddings. Install with: pip install sentence-transformers")

        model = embedding_config.get("model", "sentence-transformers/all-MiniLM-L6-v2")

        # Create Hugging Face generator
        base_generator = HuggingFaceEmbeddingGenerator(config, model)
        logger.info(f"Created Hugging Face embedding generator for backend '{backend_name}' with model {model}")

    else:
        raise ValueError(f"Unknown embedding provider '{provider}' for backend '{backend_name}'")

    # Add contextual embedding if configured in the backend
    if embedding_config.get("contextual", False):
        # Get actually used parameters
        predecessor_count = embedding_config.get("predecessor_count", config.config.get("embedding", {}).get("predecessor_count", 1))
        successor_count = embedding_config.get("successor_count", config.config.get("embedding", {}).get("successor_count", 1))
        max_tokens = config.get_embedding_max_tokens()
        
        # Legacy parameters - passed for backward compatibility but NOT USED
        window_size = embedding_config.get("window_size", 3)  # LEGACY: Not used
        overlap_size = embedding_config.get("overlap_size", 1)  # LEGACY: Not used
        ancestor_depth = embedding_config.get("ancestor_depth", 1)  # LEGACY: Not used
        contextual_generator = ContextualEmbeddingGenerator(
            _config=config,
            base_generator=base_generator,
            predecessor_count=predecessor_count,
            successor_count=successor_count,
            max_tokens=max_tokens,
            # Legacy parameters - kept for compatibility
            window_size=window_size,  # LEGACY
            overlap_size=overlap_size,  # LEGACY
            ancestor_depth=ancestor_depth,  # LEGACY
            use_semantic_tags=False  # Always False
        )
        logger.info(f"Added contextual embedding wrapper for backend '{backend_name}'")
        return contextual_generator

    return base_generator
