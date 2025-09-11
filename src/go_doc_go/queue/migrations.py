"""
Database migration utilities for the work queue system.
"""

import logging
import os
from pathlib import Path
from typing import Optional

logger = logging.getLogger(__name__)


def create_schema(db, force: bool = False) -> None:
    """
    Create the queue schema in the database.
    
    Args:
        db: Database connection
        force: If True, drop existing tables first (DANGEROUS!)
    """
    schema_file = Path(__file__).parent / 'schema.sql'
    
    if not schema_file.exists():
        raise FileNotFoundError(f"Schema file not found: {schema_file}")
    
    with open(schema_file, 'r') as f:
        schema_sql = f.read()
    
    try:
        if force:
            logger.warning("Dropping existing queue tables...")
            drop_sql = """
                DROP TABLE IF EXISTS document_dependencies CASCADE;
                DROP TABLE IF EXISTS run_workers CASCADE;
                DROP TABLE IF EXISTS document_queue CASCADE;
                DROP TABLE IF EXISTS processing_runs CASCADE;
                DROP FUNCTION IF EXISTS update_updated_at_column CASCADE;
                DROP FUNCTION IF EXISTS check_stale_workers CASCADE;
                DROP FUNCTION IF EXISTS reclaim_stale_work CASCADE;
            """
            db.execute_raw(drop_sql)
        
        logger.info("Creating queue schema...")
        db.execute_raw(schema_sql)
        logger.info("Queue schema created successfully")
        
    except Exception as e:
        logger.error(f"Failed to create schema: {e}")
        raise


def check_schema_exists(db) -> bool:
    """
    Check if the queue schema exists in the database.
    
    Args:
        db: Database connection
        
    Returns:
        True if all required tables exist
    """
    required_tables = [
        'processing_runs',
        'document_queue',
        'run_workers',
        'document_dependencies'
    ]
    
    try:
        for table in required_tables:
            result = db.execute("""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_name = %s
                )
            """, (table,))
            
            if not result or not result.get('exists'):
                logger.debug(f"Table {table} does not exist")
                return False
        
        return True
        
    except Exception as e:
        logger.error(f"Error checking schema: {e}")
        return False


def get_schema_version(db) -> Optional[str]:
    """
    Get the current schema version.
    
    Args:
        db: Database connection
        
    Returns:
        Schema version string or None if not versioned
    """
    try:
        # Check if we have a schema_version table (for future use)
        result = db.execute("""
            SELECT EXISTS (
                SELECT FROM information_schema.tables 
                WHERE table_name = 'schema_version'
            )
        """)
        
        if result and result.get('exists'):
            version = db.execute("""
                SELECT version FROM schema_version 
                ORDER BY applied_at DESC 
                LIMIT 1
            """)
            return version.get('version') if version else None
        
        # For now, if schema exists, consider it v1.0.0
        if check_schema_exists(db):
            return "1.0.0"
        
        return None
        
    except Exception as e:
        logger.error(f"Error getting schema version: {e}")
        return None


def migrate_schema(db, target_version: Optional[str] = None) -> None:
    """
    Migrate schema to target version.
    
    Args:
        db: Database connection
        target_version: Target version (None for latest)
    """
    current_version = get_schema_version(db)
    
    if current_version is None:
        # No schema exists, create it
        create_schema(db)
        return
    
    # Future: Add migration logic for version upgrades
    logger.info(f"Current schema version: {current_version}")
    if target_version and target_version != current_version:
        logger.warning(f"Migration from {current_version} to {target_version} not yet implemented")


def validate_schema(db) -> bool:
    """
    Validate that the schema is correctly set up.
    
    Args:
        db: Database connection
        
    Returns:
        True if schema is valid
    """
    if not check_schema_exists(db):
        return False
    
    try:
        # Test basic operations
        with db.transaction():
            # Test we can query the tables
            db.execute("SELECT COUNT(*) FROM processing_runs")
            db.execute("SELECT COUNT(*) FROM document_queue")
            db.execute("SELECT COUNT(*) FROM run_workers")
            db.execute("SELECT COUNT(*) FROM document_dependencies")
        
        logger.info("Schema validation successful")
        return True
        
    except Exception as e:
        logger.error(f"Schema validation failed: {e}")
        return False