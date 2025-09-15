"""
Utility to fix inconsistent relationship parquet file schemas.
Standardizes all relationship files to use consistent field names.
"""

import os
import logging
from pathlib import Path
from typing import Dict, Any, List, Optional
import pandas as pd
import pyarrow as pa
import pyarrow.parquet as pq

logger = logging.getLogger(__name__)


def standardize_relationship_schema(df: pd.DataFrame) -> pd.DataFrame:
    """
    Standardize relationship DataFrame to consistent schema.
    
    Standard schema:
    - relationship_id: str
    - source_id: str  
    - target_id: str (renamed from target_reference if needed)
    - doc_id: str (if present, kept for reference)
    - relationship_type: str
    - metadata: dict/struct
    - _run_id: str
    - _written_at: str
    """
    # Rename target_reference to target_id if present
    if 'target_reference' in df.columns and 'target_id' not in df.columns:
        df = df.rename(columns={'target_reference': 'target_id'})
        logger.debug(f"Renamed target_reference to target_id")
    
    # Ensure required columns exist
    required_cols = ['relationship_id', 'source_id', 'target_id', 'relationship_type']
    for col in required_cols:
        if col not in df.columns:
            logger.warning(f"Missing required column: {col}")
    
    # Order columns consistently
    standard_order = [
        'relationship_id', 'source_id', 'target_id', 'relationship_type',
        'doc_id', 'metadata', '_run_id', '_written_at'
    ]
    
    # Keep only columns that exist
    cols_to_keep = [col for col in standard_order if col in df.columns]
    
    # Add any extra columns not in standard order
    extra_cols = [col for col in df.columns if col not in standard_order]
    cols_to_keep.extend(extra_cols)
    
    return df[cols_to_keep]


def fix_parquet_file(file_path: str, output_path: Optional[str] = None) -> bool:
    """
    Fix a single parquet file's schema.
    
    Args:
        file_path: Path to the parquet file to fix
        output_path: Optional output path (if None, overwrites original)
    
    Returns:
        True if file was modified, False otherwise
    """
    try:
        # Read the parquet file
        df = pd.read_parquet(file_path)
        original_cols = list(df.columns)
        
        # Standardize schema
        df_fixed = standardize_relationship_schema(df)
        
        # Check if schema changed
        if list(df_fixed.columns) != original_cols:
            # Write the fixed file
            output = output_path or file_path
            df_fixed.to_parquet(output, engine='pyarrow', compression='snappy', index=False)
            logger.info(f"Fixed schema for {file_path}")
            return True
        else:
            logger.debug(f"Schema already correct for {file_path}")
            return False
            
    except Exception as e:
        logger.error(f"Error fixing {file_path}: {e}")
        return False


def fix_all_relationship_files(base_path: str, dry_run: bool = False) -> Dict[str, Any]:
    """
    Fix all relationship parquet files in the data lake.
    
    Args:
        base_path: Base path to the data lake
        dry_run: If True, only report what would be fixed without modifying
    
    Returns:
        Dictionary with statistics about the operation
    """
    stats = {
        'total_files': 0,
        'files_fixed': 0,
        'files_skipped': 0,
        'files_errored': 0,
        'schemas_found': {}
    }
    
    # Find all relationship parquet files
    rel_path = Path(base_path) / 'relationships'
    if not rel_path.exists():
        logger.warning(f"Relationships directory not found: {rel_path}")
        return stats
    
    parquet_files = list(rel_path.rglob('*.parquet'))
    stats['total_files'] = len(parquet_files)
    
    logger.info(f"Found {len(parquet_files)} relationship parquet files")
    
    for file_path in parquet_files:
        try:
            # Read and analyze schema
            df = pd.read_parquet(file_path)
            schema_key = tuple(sorted(df.columns))
            
            # Track unique schemas
            if schema_key not in stats['schemas_found']:
                stats['schemas_found'][schema_key] = []
            stats['schemas_found'][schema_key].append(str(file_path))
            
            if not dry_run:
                # Fix the file
                if fix_parquet_file(str(file_path)):
                    stats['files_fixed'] += 1
                else:
                    stats['files_skipped'] += 1
            else:
                # Check if file would need fixing
                df_fixed = standardize_relationship_schema(df.copy())
                if list(df_fixed.columns) != list(df.columns):
                    stats['files_fixed'] += 1
                    logger.info(f"Would fix: {file_path}")
                else:
                    stats['files_skipped'] += 1
                    
        except Exception as e:
            logger.error(f"Error processing {file_path}: {e}")
            stats['files_errored'] += 1
    
    # Report unique schemas found
    logger.info(f"Found {len(stats['schemas_found'])} unique schemas:")
    for schema, files in stats['schemas_found'].items():
        logger.info(f"  Schema {schema}: {len(files)} files")
    
    return stats


def verify_schema_consistency(base_path: str) -> bool:
    """
    Verify all relationship files have consistent schema.
    
    Args:
        base_path: Base path to the data lake
    
    Returns:
        True if all schemas are consistent, False otherwise
    """
    rel_path = Path(base_path) / 'relationships'
    if not rel_path.exists():
        logger.warning(f"Relationships directory not found: {rel_path}")
        return False
    
    parquet_files = list(rel_path.rglob('*.parquet'))
    if not parquet_files:
        logger.warning("No parquet files found")
        return True
    
    schemas = set()
    for file_path in parquet_files:
        try:
            schema = pq.read_schema(file_path)
            # Convert to a comparable format
            schema_str = str(sorted([(field.name, str(field.type)) for field in schema]))
            schemas.add(schema_str)
        except Exception as e:
            logger.error(f"Error reading schema from {file_path}: {e}")
            return False
    
    if len(schemas) == 1:
        logger.info("All relationship files have consistent schema")
        return True
    else:
        logger.warning(f"Found {len(schemas)} different schemas in relationship files")
        return False


if __name__ == "__main__":
    import argparse
    
    logging.basicConfig(level=logging.INFO, format='%(levelname)s - %(message)s')
    
    parser = argparse.ArgumentParser(description='Fix relationship parquet file schemas')
    parser.add_argument('--data-lake', default='./data-lake', help='Path to data lake')
    parser.add_argument('--dry-run', action='store_true', help='Only show what would be fixed')
    parser.add_argument('--verify', action='store_true', help='Only verify schema consistency')
    
    args = parser.parse_args()
    
    if args.verify:
        # Just verify consistency
        is_consistent = verify_schema_consistency(args.data_lake)
        exit(0 if is_consistent else 1)
    else:
        # Fix schemas
        stats = fix_all_relationship_files(args.data_lake, dry_run=args.dry_run)
        
        print("\nSummary:")
        print(f"  Total files: {stats['total_files']}")
        if args.dry_run:
            print(f"  Would fix: {stats['files_fixed']}")
            print(f"  Would skip: {stats['files_skipped']}")
        else:
            print(f"  Fixed: {stats['files_fixed']}")
            print(f"  Skipped: {stats['files_skipped']}")
        print(f"  Errors: {stats['files_errored']}")
        
        # Verify after fixing
        if not args.dry_run and stats['files_fixed'] > 0:
            print("\nVerifying consistency after fixes...")
            is_consistent = verify_schema_consistency(args.data_lake)
            if is_consistent:
                print("✓ All files now have consistent schema")
            else:
                print("✗ Schema inconsistencies remain")