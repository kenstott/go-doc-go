import logging
import os

from dotenv import load_dotenv

from go_doc_go.config import Config
from go_doc_go.processing.two_pass import TwoPassProcessor, TwoPassWorker

# Load environment variables from .env file
load_dotenv()

logger = logging.getLogger(__name__)

_config = Config(os.environ.get("GO_DOC_GO_CONFIG_PATH", "./config.yaml"))


def ingest_documents(config: Config, source_configs=None, max_link_depth=None, 
                    processing_mode: str = None, progress_callback=None):
    """
    Ingest documents from configured content sources using two-pass processing.
    
    Pass 1: Document parsing and storage
    Pass 2: Embedding generation and relationship detection
    
    Args:
        config: Configuration object
        source_configs: Optional list of content source configs (overrides config)
        max_link_depth: Optional override for link depth (overrides source config)
        processing_mode: Processing mode ('single', 'distributed', or 'worker'). 
                        Overrides config if specified.
        progress_callback: Optional callback function for progress updates.
                          Called with dict containing: documents, elements, relationships,
                          documents_total, parsing_complete, embedding_complete, etc.
    
    Returns:
        Dictionary with statistics about ingested documents
    """
    logger.debug("Starting document ingestion process")
    
    # Ensure config is a Config instance
    if not isinstance(config, Config):
        from go_doc_go import Config as ConfigClass
        if isinstance(config, dict):
            # Create Config instance from dictionary
            config_obj = ConfigClass()
            config_obj.config = config
            config = config_obj
        elif isinstance(config, str):
            # Create Config instance from path
            config = ConfigClass(config)
        else:
            raise ValueError("config must be a Config instance, dictionary, or path string")
    
    # Handle backward compatibility for processing_mode
    if processing_mode:
        # Map old processing modes to new worker modes
        mode_mapping = {
            'single': 'auto',
            'distributed': 'coordinator',
            'worker': 'worker'
        }
        worker_mode = mode_mapping.get(processing_mode, processing_mode)
        logger.info(f"Mapped legacy processing_mode '{processing_mode}' to worker_mode '{worker_mode}'")
    else:
        # Get worker mode from config (default to 'auto' for local processing)
        worker_mode = config.config.get('processing', {}).get('worker_mode', 'auto')
        
        # Check for legacy 'mode' config for backward compatibility
        if 'mode' in config.config.get('processing', {}):
            legacy_mode = config.config['processing']['mode']
            mode_mapping = {
                'single': 'auto',
                'distributed': 'coordinator', 
                'worker': 'worker'
            }
            worker_mode = mode_mapping.get(legacy_mode, worker_mode)
            logger.info(f"Using legacy mode config: '{legacy_mode}' -> '{worker_mode}'")
    
    logger.info(f"Using worker mode: {worker_mode}")
    
    # Route to appropriate processing method
    if worker_mode == 'coordinator':
        # Coordinator mode: enqueue work for distributed workers
        return _coordinate_two_pass_processing(config, source_configs, max_link_depth, progress_callback)
    elif worker_mode == 'worker':
        # Worker mode: process from queue
        worker = TwoPassWorker(config)
        return worker.run(progress_callback)
    else:
        # Auto mode: local processing (single machine)
        processor = TwoPassProcessor(config)
        return processor.process_local(source_configs, max_link_depth, progress_callback)


def _compute_cross_document_container_relationships(db, processed_doc_ids, config):
    """
    Compute semantic relationships between containers across documents.

    Args:
        db: DocumentDatabase instance
        processed_doc_ids: List of document IDs that were processed
        config: Configuration object

    Returns:
        Number of relationships created
    """
    logger.info(f"Computing cross-document container relationships for {len(processed_doc_ids)} documents")
    logger.debug(f"Input processed_doc_ids: {processed_doc_ids}")

    # Container element types we're interested in
    container_types = ["body", "div", "list", "header", "section", "title", "h1", "h2", "h3", "h4", "h5", "h6"]
    logger.debug(f"Looking for container types: {container_types}")
    
    similarity_threshold = (config.config.get('relationship_detection', {})
                            .get('cross_document_semantic', {}).get('similarity_threshold'))
    logger.debug(f"Similarity threshold from config: {similarity_threshold}")
    logger.debug(f"Config relationship_detection section: {config.config.get('relationship_detection', {})}")
    
    if similarity_threshold is None:
        logger.warning("Similarity threshold not configured - skipping cross-document relationship generation")
        return 0

    # Get all container elements from processed documents
    processed_containers = []

    for doc_id in processed_doc_ids:
        # Get elements from the document
        elements = db.get_document_elements(doc_id)

        # Filter for container elements
        containers = [e for e in elements if e["element_type"] in container_types]

        # Store with document context
        for container in containers:
            processed_containers.append({
                "element": container,
                "doc_id": doc_id
            })

    logger.debug(f"Found {len(processed_containers)} container elements in processed documents")

    # Delete existing semantic relationships for processed elements
    for container_info in processed_containers:
        container = container_info["element"]
        element_id = container["element_id"]
        try:
            db.delete_relationships_for_element(element_id, "semantic_section")
        except Exception as e:
            logger.warning(f"Failed to delete existing relationships for element {element_id}: {e}")

    # Compute new relationships
    new_relationships = []
    relationship_count = 0

    # Create a map of processed containers by element_id for quick lookup
    processed_container_map = {
        container_info["element"]["element_id"]: container_info
        for container_info in processed_containers
    }

    # For each processed container
    for container_info in processed_containers:
        container = container_info["element"]
        source = container_info["doc_id"]
        element_id = container["element_id"]
        element_pk = container["element_pk"]

        # Get embedding
        embedding = db.get_embedding(element_pk)
        if not embedding:
            continue

        # Search for similar containers in other documents
        filter_criteria = {
            "element_type": container_types,
            "exclude_doc_id": [source]  # Exclude containers from the same document
        }

        similar_containers = db.search_by_embedding(
            embedding,
            limit=20,
            filter_criteria=filter_criteria
        )

        # Process results
        for target_id, similarity in similar_containers:
            # Skip if similarity is below threshold
            if similarity < similarity_threshold:
                continue

            # Skip if this is another processed container we've already compared to
            if target_id in processed_container_map:
                target_element = db.get_element(target_id)
                target_doc_id = processed_container_map[target_id]["doc_id"]
            else:
                # Get document ID for this container
                target_element = db.get_element(target_id)
                if not target_element:
                    continue

                target_doc_id = target_element["doc_id"]

            # Create relationship
            new_relationships.append({
                "relationship_id": f"sem_rel_{element_id}_{target_element['element_id']}",
                "source_id": element_id,
                "relationship_type": "semantic_section",
                "target_reference": target_element['element_id'],
                "metadata": {
                    "similarity_score": similarity,
                    "cross_document": True,
                    "source_doc_id": source,
                    "target_doc_id": target_doc_id
                }
            })
            relationship_count += 1

    # Store all new relationships
    for relationship in new_relationships:
        try:
            db.store_relationship(relationship)
        except Exception as e:
            logger.warning(f"Failed to store cross-document relationship {relationship.get('relationship_id', 'unknown')}: {e}")

    logger.info(f"Created {relationship_count} cross-document semantic relationships")
    return relationship_count


def _coordinate_two_pass_processing(config: Config, source_configs=None, max_link_depth=None, progress_callback=None):
    """
    Coordinate two-pass processing for distributed workers.
    
    This function enqueues work for distributed workers to process using the
    two-pass approach. It doesn't process documents itself but coordinates
    the work distribution.
    
    Args:
        config: Configuration object
        source_configs: Optional list of content source configs
        max_link_depth: Optional override for link depth
        progress_callback: Optional callback for progress updates
        
    Returns:
        Dictionary with coordination statistics
    """
    logger.info("Coordinating two-pass processing for distributed workers")
    
    from .work_queue import WorkQueue, RunCoordinator
    from .content_source.factory import get_content_source
    
    try:
        # Initialize database and work queue
        db = config.get_document_database()
        
        # Create run coordinator
        run_config = {
            'content_sources': source_configs or config.get_content_sources(),
            'max_link_depth': max_link_depth,
            'processing': config.config.get('processing', {})
        }
        
        # Generate run ID from config
        run_id = RunCoordinator.get_run_id_from_config(run_config)
        coordinator = RunCoordinator(db, 'coordinator')
        coordinator.ensure_run_exists(run_id, run_config)
        
        # Create work queue
        work_queue = WorkQueue(db, 'coordinator')
        
        # Enqueue documents for Pass 1 (parsing)
        sources_to_process = source_configs or config.get_content_sources()
        total_documents = 0
        
        for source_config in sources_to_process:
            source_name = source_config.get('name')
            logger.info(f"Enqueuing documents from source: {source_name}")
            
            try:
                source = get_content_source(source_config)
                documents = source.list_documents()
                
                for doc in documents:
                    # Enqueue document for Pass 1
                    queue_id = work_queue.add_document(
                        doc_id=doc['id'],
                        source_name=source_name,
                        run_id=run_id,
                        metadata={
                            'processing_pass': 1,
                            'doc_type': doc.get('doc_type'),
                            'source_config': source_config
                        }
                    )
                    total_documents += 1
                    
                logger.info(f"Enqueued {len(documents)} documents from {source_name}")
                
            except Exception as e:
                logger.error(f"Error enqueuing documents from source {source_name}: {e}")
        
        logger.info(f"Enqueued {total_documents} total documents for processing")
        
        # Return coordination statistics
        return {
            'documents_enqueued': total_documents,
            'run_id': run_id,
            'sources': len(sources_to_process),
            'status': 'coordinated'
        }
        
    except Exception as e:
        logger.error(f"Two-pass coordination failed: {str(e)}")
        raise