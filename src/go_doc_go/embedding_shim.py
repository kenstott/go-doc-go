#!/usr/bin/env python3
"""
Python shim for embedding generation from Go.

This script provides a bridge between Go and Python embedding libraries.
It receives JSON requests via command line arguments and returns JSON responses.
"""

import sys
import json
import logging
import os
from typing import List, Dict, Any

# Configure logging - use DEBUG if DEBUG env var is set
log_level = logging.DEBUG if os.getenv('DEBUG') else logging.INFO
logging.basicConfig(level=log_level, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


def create_embedding_generator(config: Dict[str, Any]):
    """
    Create an embedding generator based on configuration.

    Args:
        config: Embedding configuration dictionary

    Returns:
        Embedding generator instance
    """
    provider = config.get('provider', 'fastembed').lower()

    if provider == 'fastembed':
        from go_doc_go.embeddings.fastembed import FastEmbedGenerator

        # Create a minimal Config-like object with required attributes
        class MinimalConfig:
            pass

        config_obj = MinimalConfig()

        # Create FastEmbed generator
        model = config.get('model', 'sentence-transformers/all-MiniLM-L6-v2')
        dimensions = config.get('dimensions', 384)
        cache_dir = config.get('cache_dir', None)

        return FastEmbedGenerator(config_obj, model, dimensions, cache_dir)

    else:
        raise ValueError(f"Unsupported embedding provider: {provider}")


def generate_embedding(config: Dict[str, Any], text: str) -> List[float]:
    """
    Generate embedding for a single text.

    Args:
        config: Embedding configuration
        text: Input text

    Returns:
        Embedding vector
    """
    generator = create_embedding_generator(config)
    embedding = generator.generate(text)
    return embedding


def generate_embeddings_batch(config: Dict[str, Any], texts: List[str]) -> List[List[float]]:
    """
    Generate embeddings for multiple texts.

    Args:
        config: Embedding configuration
        texts: List of input texts

    Returns:
        List of embedding vectors
    """
    generator = create_embedding_generator(config)
    embeddings = generator.generate_batch(texts)
    return embeddings


def main():
    """Main entry point for the embedding shim - runs as persistent process."""
    # Load model once at startup
    generator = None

    # Signal that we're ready to receive requests
    print(json.dumps({"status": "ready"}), flush=True)

    try:
        # Read requests line-by-line from stdin
        request_count = 0
        for line in sys.stdin:
            try:
                line = line.strip()
                if not line:
                    continue

                request_count += 1
                logger.debug(f"Processing request #{request_count}")

                request = json.loads(line)

                operation = request.get('operation')
                config = request.get('config', {})
                data = request.get('data', {})

                # Create generator on first request (lazy initialization)
                if generator is None:
                    logger.info("Initializing embedding generator...")
                    generator = create_embedding_generator(config)
                    logger.info("Embedding generator initialized and ready")

                if operation == 'generate':
                    # Generate single embedding
                    text = data.get('text', '')
                    logger.debug(f"Generating single embedding for text of length {len(text)}")
                    embedding = generator.generate(text)
                    response = {'embedding': embedding}
                    logger.debug(f"Generated embedding with {len(embedding)} dimensions")

                elif operation == 'generate_batch':
                    # Generate batch embeddings
                    texts = data.get('texts', [])
                    logger.debug(f"Generating batch embeddings for {len(texts)} texts")
                    embeddings = generator.generate_batch(texts)
                    response = {'embeddings': embeddings}
                    logger.debug(f"Generated {len(embeddings)} embeddings")

                else:
                    response = {'error': f'Unknown operation: {operation}'}

                # Output response as single line JSON
                response_json = json.dumps(response)
                logger.debug(f"Sending response of length {len(response_json)}")
                print(response_json, flush=True)
                logger.debug(f"Response sent successfully for request #{request_count}")

            except Exception as e:
                logger.error(f"Error processing request #{request_count}: {e}", exc_info=True)
                response = {'error': str(e)}
                print(json.dumps(response), flush=True)

        logger.info(f"Stdin closed after {request_count} requests, exiting normally")

    except KeyboardInterrupt:
        logger.info("Embedding shim received interrupt, shutting down")
    except Exception as e:
        logger.error(f"Fatal error in embedding shim: {e}", exc_info=True)
        sys.exit(1)


if __name__ == '__main__':
    main()
