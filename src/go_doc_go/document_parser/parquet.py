"""
Parquet file parser for structured tabular data.
Handles Parquet files with configurable metadata extraction.
"""

import json
import logging
from typing import Dict, Any, List, Optional
import pandas as pd

from .base import DocumentParser
from ..storage import ElementType
from ..relationships import RelationshipType

logger = logging.getLogger(__name__)


class ParquetParser(DocumentParser):
    """Parser for Parquet files containing structured tabular data."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize Parquet parser."""
        super().__init__(config)
        # Generic configuration for tabular data
        self.text_column = self.config.get('text_column', None)  # Auto-detect if not specified
        self.group_by_column = self.config.get('group_by_column', None)  # Optional grouping
        self.max_content_preview = self.config.get('max_content_preview', 100)
        # Configurable list of metadata columns to extract from first row
        self.metadata_columns = self.config.get('metadata_columns', [])
    
    def supports_location(self) -> bool:
        """Parquet parser doesn't support location-based content retrieval."""
        return False
    
    def _resolve_element_content(self, element: Dict[str, Any], location: str) -> str:
        """Resolve element content from location (not supported for parquet)."""
        return element.get('content_preview', '')
    
    def _resolve_element_text(self, element: Dict[str, Any]) -> str:
        """Get text content of element."""
        return element.get('content_preview', '')
    
    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse a Parquet file containing structured data.
        
        Args:
            content: Dictionary with 'binary_path' pointing to parquet file
            
        Returns:
            Parsed document with elements and relationships
        """
        doc_id = content.get('id', self._generate_id('doc'))
        metadata = content.get('metadata', {})
        
        # Get path to parquet file
        binary_path = content.get('binary_path')
        if not binary_path:
            raise ValueError("Parquet parser requires 'binary_path' in content")
        
        try:
            # Read parquet file
            df = pd.read_parquet(binary_path)
            
            # Extract document metadata from first row if configured
            if not df.empty and self.metadata_columns:
                first_row = df.iloc[0]
                for col in self.metadata_columns:
                    if col in df.columns and first_row[col]:
                        metadata[col] = str(first_row[col])
            
            elements = []
            relationships = []
            
            # Create document root
            root_id = self._generate_id('root')
            elements.append({
                'element_id': root_id,
                'element_type': ElementType.ROOT.value,
                'content_preview': f"Parquet document with {len(df)} rows",
                'metadata': metadata
            })
            
            # Create body element
            body_id = self._generate_id('body')
            elements.append({
                'element_id': body_id,
                'element_type': ElementType.BODY.value,
                'parent_id': root_id,
                'content_preview': 'Document body',
                'metadata': {}
            })
            
            # Add relationship between root and body
            relationships.append({
                'relationship_id': self._generate_id('rel'),
                'source_id': root_id,
                'target_id': body_id,
                'relationship_type': RelationshipType.CONTAINS.value
            })
            
            # Determine text column if not specified
            if not self.text_column:
                # Auto-detect first string/object column as text column
                for col in df.columns:
                    if df[col].dtype == 'object' or df[col].dtype == 'string':
                        self.text_column = col
                        break

            # Group rows by optional grouping column
            current_group_id = None
            current_group_value = None

            # Process each row
            for idx, row in df.iterrows():
                # Check if we need to create a new group
                if self.group_by_column and self.group_by_column in df.columns:
                    group_value = row.get(self.group_by_column)
                    if group_value and group_value != current_group_value:
                        # Create new group element
                        current_group_value = group_value
                        current_group_id = self._generate_id('group')
                        elements.append({
                            'element_id': current_group_id,
                            'element_type': ElementType.HEADER.value,
                            'parent_id': body_id,
                            'content_preview': f"Group: {group_value}",
                            'metadata': {'group_value': str(group_value), 'group_column': self.group_by_column}
                        })

                        relationships.append({
                            'relationship_id': self._generate_id('rel'),
                            'source_id': body_id,
                            'target_id': current_group_id,
                            'relationship_type': RelationshipType.CONTAINS.value
                        })

                # Create element for this row
                row_element_id = self._generate_id('row')

                # Build metadata from all columns
                row_metadata = {
                    'row_index': idx
                }

                # Add all column values to metadata
                for col in df.columns:
                    value = row.get(col)
                    if pd.notna(value):
                        # Convert to string for JSON serialization
                        row_metadata[col] = str(value)

                # Get text content for preview
                text_content = ''
                if self.text_column and self.text_column in df.columns:
                    text_content = str(row.get(self.text_column, ''))
                elif len(df.columns) > 0:
                    # If no text column specified, concatenate all values
                    text_parts = []
                    for col in df.columns:
                        value = row.get(col)
                        if pd.notna(value):
                            text_parts.append(f"{col}: {value}")
                    text_content = '; '.join(text_parts)

                # Truncate content for preview
                content_preview = text_content[:self.max_content_preview]
                if len(text_content) > self.max_content_preview:
                    content_preview += '...'

                # Create row element
                parent_id = current_group_id if current_group_id else body_id
                elements.append({
                    'element_id': row_element_id,
                    'element_type': ElementType.PARAGRAPH.value,
                    'parent_id': parent_id,
                    'content_preview': content_preview,
                    'metadata': row_metadata
                })

                relationships.append({
                    'relationship_id': self._generate_id('rel'),
                    'source_id': parent_id,
                    'target_id': row_element_id,
                    'relationship_type': RelationshipType.CONTAINS.value
                })
            
            # Create document structure
            document = {
                'doc_id': doc_id,
                'doc_type': 'parquet',
                'metadata': metadata
            }
            
            # Add statistics to metadata
            document['metadata']['row_count'] = len(df)
            document['metadata']['column_count'] = len(df.columns)
            document['metadata']['columns'] = df.columns.tolist()

            # Add column data types for reference
            document['metadata']['column_types'] = {col: str(df[col].dtype) for col in df.columns}

            # If text column was auto-detected, include it in metadata
            if self.text_column:
                document['metadata']['text_column'] = self.text_column

            # If grouping was used, include group statistics
            if self.group_by_column and self.group_by_column in df.columns:
                unique_groups = df[self.group_by_column].dropna().unique()
                document['metadata']['group_count'] = len(unique_groups)
                document['metadata']['group_column'] = self.group_by_column
            
            return {
                'document': document,
                'elements': elements,
                'relationships': relationships
            }
            
        except Exception as e:
            logger.error(f"Error parsing parquet file: {e}")
            raise