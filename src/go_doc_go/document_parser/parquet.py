"""
Parquet file parser for structured data including earnings calls.
Handles SEC filing parquet files with speaker metadata.
"""

import json
import logging
from typing import Dict, Any, List, Optional
import pandas as pd

from go_doc_go.document_parser.base import DocumentParser
from go_doc_go.storage import ElementType
from go_doc_go.relationships import RelationshipType

logger = logging.getLogger(__name__)


class ParquetParser(DocumentParser):
    """Parser for Parquet files containing structured data like earnings calls."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize Parquet parser."""
        super().__init__(config)
        self.speaker_column = self.config.get('speaker_column', 'speaker_name')
        self.role_column = self.config.get('role_column', 'speaker_role')
        self.text_column = self.config.get('text_column', 'paragraph_text')
        self.section_column = self.config.get('section_column', 'section_type')
        self.max_content_preview = self.config.get('max_content_preview', 100)
    
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
            
            # Extract document metadata from first row if available
            if not df.empty:
                first_row = df.iloc[0]
                for col in ['company', 'ticker', 'quarter', 'year', 'cik', 'filing_date', 'filing_type']:
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
            
            # Group rows by section if section column exists
            current_section_id = None
            current_section = None
            
            # Process each row
            for idx, row in df.iterrows():
                # Check if we need to create a new section
                if self.section_column in df.columns:
                    section = row.get(self.section_column)
                    if section and section != current_section:
                        # Create new section element
                        current_section = section
                        current_section_id = self._generate_id('section')
                        elements.append({
                            'element_id': current_section_id,
                            'element_type': ElementType.HEADER.value,
                            'parent_id': body_id,
                            'content_preview': f"Section: {section}",
                            'metadata': {'section_type': section}
                        })
                        
                        relationships.append({
                            'relationship_id': self._generate_id('rel'),
                            'source_id': body_id,
                            'target_id': current_section_id,
                            'relationship_type': RelationshipType.CONTAINS.value
                        })
                
                # Create paragraph element for this row
                para_id = self._generate_id('para')
                para_metadata = {
                    'row_index': idx,
                    'paragraph_number': row.get('paragraph_number', idx)
                }
                
                # Add speaker metadata if available
                if self.speaker_column in df.columns and pd.notna(row.get(self.speaker_column)):
                    para_metadata['speaker'] = str(row[self.speaker_column])
                
                if self.role_column in df.columns and pd.notna(row.get(self.role_column)):
                    para_metadata['speaker_role'] = str(row[self.role_column])
                
                # Add section to metadata
                if self.section_column in df.columns and pd.notna(row.get(self.section_column)):
                    para_metadata['section'] = str(row[self.section_column])
                
                # Get text content
                text_content = ''
                if self.text_column in df.columns:
                    text_content = str(row.get(self.text_column, ''))
                
                # Truncate content for preview
                content_preview = text_content[:self.max_content_preview]
                if len(text_content) > self.max_content_preview:
                    content_preview += '...'
                
                # Create paragraph element
                parent_id = current_section_id if current_section_id else body_id
                elements.append({
                    'element_id': para_id,
                    'element_type': ElementType.PARAGRAPH.value,
                    'parent_id': parent_id,
                    'content_preview': content_preview,
                    'metadata': para_metadata
                })
                
                relationships.append({
                    'relationship_id': self._generate_id('rel'),
                    'source_id': parent_id,
                    'target_id': para_id,
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
            
            # Count speakers if available
            if self.speaker_column in df.columns:
                unique_speakers = df[self.speaker_column].dropna().unique()
                if len(unique_speakers) > 0:
                    document['metadata']['speaker_count'] = len(unique_speakers)
                    document['metadata']['speakers'] = unique_speakers.tolist()
            
            return {
                'document': document,
                'elements': elements,
                'relationships': relationships
            }
            
        except Exception as e:
            logger.error(f"Error parsing parquet file: {e}")
            raise