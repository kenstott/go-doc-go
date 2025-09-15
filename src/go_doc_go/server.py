import json
import logging
import os
from datetime import datetime
from typing import List

import yaml
from flask import Flask, request, jsonify, render_template_string, send_from_directory, send_file
from flask_cors import CORS
from werkzeug.exceptions import BadRequest, InternalServerError, NotFound, Unauthorized, Forbidden

from go_doc_go.adapter import create_content_resolver
from go_doc_go.config import Config
# Search imports removed as search endpoints have been deleted
# from go_doc_go.search import search_by_text, SearchResult, search_structured
from go_doc_go.api.flask_settings_routes import settings_bp
from go_doc_go.api.pipeline_routes import pipeline_bp

# Configure logging
log_level = os.environ.get('LOG_LEVEL', 'INFO')
log_format = os.environ.get('LOG_FORMAT', '%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logging.basicConfig(level=getattr(logging, log_level), format=log_format)
logger = logging.getLogger(__name__)

# Lazy initialization to prevent import-time database connections
_config = None
db = None
resolver = None

def _ensure_config_loaded():
    """Ensure config is loaded without database connections."""
    global _config
    if _config is None:
        _config = Config(os.environ.get('GO_DOC_GO_CONFIG_PATH', 'config.yaml'))

def _ensure_initialized():
    """Ensure server components are initialized."""
    global _config, db, resolver
    if _config is None:
        _config = Config(os.environ.get('GO_DOC_GO_CONFIG_PATH', 'config.yaml'))
        db = _config.get_document_database()
        db.initialize()
        resolver = create_content_resolver(_config)


def _search_parquet_with_context(query, backend_config, filters, context_config, reconstruction_config):
    """
    Comprehensive search against parquet data lake with full context retrieval.
    Uses a single DuckDB query to get everything in one shot.
    """
    import pandas as pd
    import numpy as np
    import glob
    import re
    from pathlib import Path
    from sklearn.feature_extraction.text import TfidfVectorizer
    from sklearn.metrics.pairwise import cosine_similarity
    import json
    import duckdb
    
    base_path = backend_config.get('base_path', './data-lake')
    
    # Extract filter parameters
    regex_pattern = filters.get('regex_pattern')
    element_types = filters.get('element_types', [])
    cosine_threshold = filters.get('cosine_threshold', 0.0)
    limit = filters.get('limit', 10)
    
    # Extract context parameters  
    num_parents = context_config.get('parents', 2)
    num_siblings = context_config.get('siblings', 3)
    num_semantic_rels = context_config.get('semantic_relationships', 5)
    include_doc_metadata = context_config.get('include_document_metadata', True)
    
    # Use DuckDB to query parquet files directly with hive partitioning
    conn = duckdb.connect(':memory:')
    
    # Paths for data
    elements_path = f'{base_path}/elements/**/*.parquet'
    relationships_path = f'{base_path}/relationships/**/*.parquet'
    documents_path = f'{base_path}/documents/**/*.parquet'
    
    logger.info(f"Using DuckDB with hive partitioning to query data lake at {base_path}")
    
    # Parse element type filters with +/- convention
    include_types = []
    exclude_types = []
    if element_types:
        from go_doc_go.storage.element_element import ElementType
        valid_element_types = {e.value for e in ElementType}
        
        for elem_type in element_types:
            if elem_type.startswith('-'):
                base_type = elem_type[1:]
                if base_type in valid_element_types:
                    exclude_types.append(base_type)
            elif elem_type.startswith('+'):
                base_type = elem_type[1:]
                if base_type in valid_element_types:
                    include_types.append(base_type)
            else:
                if elem_type in valid_element_types:
                    include_types.append(elem_type)
    
    # Build WHERE clause for filtering
    where_clauses = []
    
    # Element type filters
    if include_types:
        types_list = ', '.join([f"'{t}'" for t in include_types])
        where_clauses.append(f"e.element_type IN ({types_list})")
    if exclude_types:
        exclude_list = ', '.join([f"'{t}'" for t in exclude_types])
        where_clauses.append(f"e.element_type NOT IN ({exclude_list})")
    
    # Regex pattern filter
    if regex_pattern:
        where_clauses.append(f"regexp_matches(e.content_preview, '{regex_pattern}')")
    
    # Text search (basic keyword matching)
    if query:
        query_terms = query.lower().split()
        term_conditions = []
        for term in query_terms:
            term_conditions.append(f"lower(e.content_preview) LIKE '%{term}%'")
        if term_conditions:
            where_clauses.append(f"({' OR '.join(term_conditions)})")
    
    where_clause = f"WHERE {' AND '.join(where_clauses)}" if where_clauses else ""
    
    # Single comprehensive DuckDB query to get everything
    comprehensive_query = f"""
    WITH 
    -- Load all elements with filters applied
    filtered_elements AS (
        SELECT * FROM read_parquet('{elements_path}', hive_partitioning=true) e
        {where_clause}
    ),
    
    -- Load all elements (for context lookup)
    all_elements AS (
        SELECT * FROM read_parquet('{elements_path}', hive_partitioning=true)
    ),
    
    -- Load all relationships
    all_relationships AS (
        SELECT * FROM read_parquet('{relationships_path}', hive_partitioning=true)
    ),
    
    -- Load all documents
    all_documents AS (
        SELECT * FROM read_parquet('{documents_path}', hive_partitioning=true)
    ),
    
    -- Get relationships for filtered elements
    element_relationships AS (
        SELECT 
            CASE 
                WHEN r.source_id IN (SELECT element_id FROM filtered_elements) THEN r.source_id
                ELSE r.target_id
            END as element_id,
            r.relationship_id,
            r.relationship_type,
            r.source_id,
            r.target_id,
            r.metadata as rel_metadata
        FROM all_relationships r
        WHERE r.source_id IN (SELECT element_id FROM filtered_elements)
           OR r.target_id IN (SELECT element_id FROM filtered_elements)
    ),
    
    -- Get parent hierarchy
    parent_hierarchy AS (
        SELECT 
            f.element_id,
            f.parent_id as parent1_id,
            p1.element_type as parent1_type,
            p1.content_preview as parent1_content,
            p1.parent_id as parent2_id,
            p2.element_type as parent2_type,
            p2.content_preview as parent2_content,
            p2.parent_id as parent3_id,
            p3.element_type as parent3_type,
            p3.content_preview as parent3_content
        FROM filtered_elements f
        LEFT JOIN all_elements p1 ON f.parent_id = p1.element_id
        LEFT JOIN all_elements p2 ON p1.parent_id = p2.element_id
        LEFT JOIN all_elements p3 ON p2.parent_id = p3.element_id
    ),
    
    -- Get siblings
    element_siblings AS (
        SELECT 
            f.element_id,
            s.element_id as sibling_id,
            s.element_type as sibling_type,
            s.content_preview as sibling_content
        FROM filtered_elements f
        INNER JOIN all_elements s ON f.parent_id = s.parent_id
        WHERE f.element_id != s.element_id
    )
    
    -- Main query combining everything
    SELECT 
        f.*,
        d.metadata as doc_metadata,
        d.source as doc_source,
        ph.parent1_id, ph.parent1_type, ph.parent1_content,
        ph.parent2_id, ph.parent2_type, ph.parent2_content,
        ph.parent3_id, ph.parent3_type, ph.parent3_content,
        r.relationship_id, r.relationship_type, r.source_id, r.target_id, r.rel_metadata,
        s.sibling_id, s.sibling_type, s.sibling_content
    FROM filtered_elements f
    LEFT JOIN all_documents d ON f.doc_id = d.doc_id
    LEFT JOIN parent_hierarchy ph ON f.element_id = ph.element_id
    LEFT JOIN element_relationships r ON f.element_id = r.element_id
    LEFT JOIN element_siblings s ON f.element_id = s.element_id
    """
    
    logger.info(f"Executing comprehensive DuckDB query...")
    
    try:
        # Execute the comprehensive query
        result_df = conn.execute(comprehensive_query).df()
        
        if result_df.empty:
            return {'results': [], 'total': 0, 'message': 'No matching elements found'}
        
        # Apply cosine similarity if query provided
        if query and not result_df.empty:
            # Get unique elements (since joins may create duplicates)
            unique_elements = result_df.drop_duplicates(subset=['element_id'])
            
            # Create TF-IDF vectors
            texts = [query] + unique_elements['content_preview'].astype(str).fillna('').tolist()
            vectorizer = TfidfVectorizer(stop_words='english', max_features=5000)
            tfidf_matrix = vectorizer.fit_transform(texts)
            
            # Calculate similarities
            query_vector = tfidf_matrix[0:1]
            doc_vectors = tfidf_matrix[1:]
            similarities = cosine_similarity(query_vector, doc_vectors).flatten()
            
            # Add similarity scores to unique elements
            unique_elements['similarity_score'] = similarities
            
            # Filter by threshold if specified
            if cosine_threshold > 0.0:
                unique_elements = unique_elements[unique_elements['similarity_score'] >= cosine_threshold]
            
            # Sort by similarity and limit
            unique_elements = unique_elements.sort_values('similarity_score', ascending=False).head(limit)
            
            # Filter main results to only include top elements
            result_df = result_df[result_df['element_id'].isin(unique_elements['element_id'])]
            
            # Add similarity scores to main results
            score_map = dict(zip(unique_elements['element_id'], unique_elements['similarity_score']))
            result_df['similarity_score'] = result_df['element_id'].map(score_map)
        else:
            # No query, just limit results
            unique_ids = result_df['element_id'].unique()[:limit]
            result_df = result_df[result_df['element_id'].isin(unique_ids)]
            result_df['similarity_score'] = 0.0
        
        # Group results by element to build response
        results = []
        for element_id in result_df['element_id'].unique():
            element_rows = result_df[result_df['element_id'] == element_id]
            first_row = element_rows.iloc[0]
            
            # Build graphlet from joined data
            graphlet = {
                'parents': [],
                'siblings': [],
                'semantic_relationships': []
            }
            
            # Extract parents
            if pd.notna(first_row.get('parent1_id')):
                graphlet['parents'].append({
                    'element_id': first_row['parent1_id'],
                    'element_type': first_row['parent1_type'],
                    'content_preview': str(first_row['parent1_content'])[:200] if pd.notna(first_row['parent1_content']) else ''
                })
            if pd.notna(first_row.get('parent2_id')) and len(graphlet['parents']) < num_parents:
                graphlet['parents'].append({
                    'element_id': first_row['parent2_id'],
                    'element_type': first_row['parent2_type'],
                    'content_preview': str(first_row['parent2_content'])[:200] if pd.notna(first_row['parent2_content']) else ''
                })
            if pd.notna(first_row.get('parent3_id')) and len(graphlet['parents']) < num_parents:
                graphlet['parents'].append({
                    'element_id': first_row['parent3_id'],
                    'element_type': first_row['parent3_type'],
                    'content_preview': str(first_row['parent3_content'])[:200] if pd.notna(first_row['parent3_content']) else ''
                })
            
            # Extract siblings (unique)
            seen_siblings = set()
            for _, row in element_rows.iterrows():
                if pd.notna(row.get('sibling_id')) and row['sibling_id'] not in seen_siblings:
                    if len(graphlet['siblings']) < num_siblings:
                        graphlet['siblings'].append({
                            'element_id': row['sibling_id'],
                            'element_type': row['sibling_type'],
                            'content_preview': str(row['sibling_content'])[:200] if pd.notna(row['sibling_content']) else ''
                        })
                        seen_siblings.add(row['sibling_id'])
            
            # Extract relationships (unique)
            seen_relationships = set()
            for _, row in element_rows.iterrows():
                if pd.notna(row.get('relationship_id')) and row['relationship_id'] not in seen_relationships:
                    if len(graphlet['semantic_relationships']) < num_semantic_rels:
                        # Determine the other element
                        other_id = row['target_id'] if row['source_id'] == element_id else row['source_id']
                        
                        # Look up the other element details
                        other_element_query = f"SELECT * FROM all_elements WHERE element_id = '{other_id}'"
                        try:
                            other_result = conn.execute(other_element_query).df()
                            if not other_result.empty:
                                other_row = other_result.iloc[0]
                                graphlet['semantic_relationships'].append({
                                    'relationship_type': row['relationship_type'],
                                    'element_id': other_id,
                                    'element_type': other_row['element_type'],
                                    'content_preview': str(other_row['content_preview'])[:200] if pd.notna(other_row['content_preview']) else '',
                                    'confidence': json.loads(row.get('rel_metadata', '{}')).get('confidence', 1.0) if row.get('rel_metadata') else 1.0
                                })
                                seen_relationships.add(row['relationship_id'])
                        except:
                            pass
            
            # Build document metadata
            doc_metadata = {}
            if include_doc_metadata:
                doc_metadata = {
                    'doc_id': first_row['doc_id'],
                    'source': first_row.get('doc_source', ''),
                    'metadata': json.loads(first_row.get('doc_metadata', '{}')) if pd.notna(first_row.get('doc_metadata')) else {}
                }
            
            # Build reconstruction if requested
            reconstruction = None
            if reconstruction_config.get('format'):
                reconstruction = _reconstruct_element_context(
                    first_row, graphlet, reconstruction_config
                )
            
            result = {
                'element': {
                    'element_id': element_id,
                    'doc_id': first_row['doc_id'],
                    'element_type': first_row['element_type'],
                    'content_preview': str(first_row['content_preview'])[:500] if pd.notna(first_row['content_preview']) else '',
                    'metadata': json.loads(first_row.get('metadata', '{}')) if pd.notna(first_row.get('metadata')) else {}
                },
                'similarity_score': float(first_row.get('similarity_score', 0.0)),
                'graphlet': graphlet,
                'document_metadata': doc_metadata,
                'reconstruction': reconstruction
            }
            
            results.append(result)
        
        # Sort results by similarity score if available
        if query:
            results.sort(key=lambda x: x['similarity_score'], reverse=True)
        
        return {
            'results': results,
            'total': len(results),
            'query': query,
            'filters_applied': filters,
            'backend': backend_config.get('description', 'Unknown')
        }
        
    except Exception as e:
        logger.error(f"Comprehensive DuckDB query failed: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return {'results': [], 'total': 0, 'error': str(e)}


def _load_relationships_data(base_path):
    """Load relationship data from parquet files."""
    import pandas as pd
    import glob
    
    try:
        pattern = f'{base_path}/relationships/**/*.parquet'
        rel_files = glob.glob(pattern, recursive=True)
        
        if rel_files:
            rel_dfs = []
            for file in rel_files:
                try:
                    df = pd.read_parquet(file)
                    rel_dfs.append(df)
                except Exception:
                    continue
            
            if rel_dfs:
                return pd.concat(rel_dfs, ignore_index=True)
    except Exception as e:
        logger.warning(f"Could not load relationships: {e}")
    
    return pd.DataFrame()


def _build_enhanced_graphlet_context(element_id, doc_id, elements_df, element_rels, conn, num_parents, num_siblings, num_semantic_rels, base_path):
    """Build enhanced graphlet context using DuckDB joins for efficiency."""
    import pandas as pd
    
    graphlet = {
        'parents': [],
        'siblings': [],
        'semantic_relationships': []
    }
    
    try:
        # Find the current element
        current_element = elements_df[elements_df['element_id'] == element_id]
        if current_element.empty:
            return graphlet
            
        current_row = current_element.iloc[0]
        parent_id = current_row.get('parent_id')
        
        # Find parents (walk up the hierarchy)
        if parent_id and pd.notna(parent_id):
            parents_found = []
            current_parent = parent_id
            
            for _ in range(num_parents):
                if not current_parent or pd.isna(current_parent):
                    break
                    
                parent_element = elements_df[elements_df['element_id'] == current_parent]
                if parent_element.empty:
                    break
                    
                parent_row = parent_element.iloc[0]
                parents_found.append({
                    'element_id': parent_row['element_id'],
                    'element_type': parent_row['element_type'],
                    'content_preview': parent_row['content_preview'][:200]
                })
                
                # Move to next parent up
                current_parent = parent_row.get('parent_id')
            
            graphlet['parents'] = parents_found
        
        # Find siblings (elements with same parent)
        if parent_id and pd.notna(parent_id):
            siblings = elements_df[
                (elements_df['parent_id'] == parent_id) & 
                (elements_df['element_id'] != element_id)
            ]
            
            graphlet['siblings'] = [
                {
                    'element_id': row['element_id'],
                    'element_type': row['element_type'],
                    'content_preview': row['content_preview'][:200]
                }
                for _, row in siblings.head(num_siblings).iterrows()
            ]
        
        # Find semantic relationships from joined data
        if not element_rels.empty:
            semantic_rels = []
            
            # Get unique relationships (avoiding duplicates)
            unique_rels = element_rels.drop_duplicates(subset=['relationship_id'])
            
            for _, rel_row in unique_rels.head(num_semantic_rels).iterrows():
                # Determine which element is the "other" one
                other_id = rel_row['target_id'] if rel_row['source_id'] == element_id else rel_row['source_id']
                
                if other_id and pd.notna(other_id):
                    # Look up the other element
                    other_element = elements_df[elements_df['element_id'] == other_id]
                    if not other_element.empty:
                        other_row = other_element.iloc[0]
                        semantic_rels.append({
                            'relationship_type': rel_row.get('relationship_type', 'semantic'),
                            'element_id': other_row['element_id'],
                            'element_type': other_row['element_type'],
                            'content_preview': other_row['content_preview'][:200],
                            'confidence': json.loads(rel_row.get('rel_metadata', '{}')).get('confidence', 1.0) if rel_row.get('rel_metadata') else 1.0
                        })
            
            graphlet['semantic_relationships'] = semantic_rels
    
    except Exception as e:
        logger.warning(f"Error building enhanced graphlet for {element_id}: {e}")
    
    return graphlet


def _build_graphlet_context(element_id, doc_id, elements_df, relationship_data, num_parents, num_siblings, num_semantic_rels):
    """Legacy graphlet context builder - kept for compatibility."""
    import pandas as pd
    
    graphlet = {
        'parents': [],
        'siblings': [],
        'semantic_relationships': []
    }
    
    try:
        # Find the current element
        current_element = elements_df[elements_df['element_id'] == element_id]
        if current_element.empty:
            return graphlet
            
        current_row = current_element.iloc[0]
        parent_id = current_row.get('parent_id')
        
        # Find parents (walk up the hierarchy)
        if parent_id and pd.notna(parent_id):
            parents_found = []
            current_parent = parent_id
            
            for _ in range(num_parents):
                if not current_parent or pd.isna(current_parent):
                    break
                    
                parent_element = elements_df[elements_df['element_id'] == current_parent]
                if parent_element.empty:
                    break
                    
                parent_row = parent_element.iloc[0]
                parents_found.append({
                    'element_id': parent_row['element_id'],
                    'element_type': parent_row['element_type'],
                    'content_preview': parent_row['content_preview'][:200]
                })
                
                # Move to next parent up
                current_parent = parent_row.get('parent_id')
            
            graphlet['parents'] = parents_found
        
        # Find siblings (elements with same parent)
        if parent_id and pd.notna(parent_id):
            siblings = elements_df[
                (elements_df['parent_id'] == parent_id) & 
                (elements_df['element_id'] != element_id)
            ]
            
            graphlet['siblings'] = [
                {
                    'element_id': row['element_id'],
                    'element_type': row['element_type'],
                    'content_preview': row['content_preview'][:200]
                }
                for _, row in siblings.head(num_siblings).iterrows()
            ]
        
        # Find semantic relationships (if relationship data available)
        if not relationship_data.empty:
            # Look for relationships where current element is source or target
            related_elements = relationship_data[
                (relationship_data.get('source_element_id', '') == element_id) |
                (relationship_data.get('target_element_id', '') == element_id)
            ]
            
            semantic_rels = []
            for _, rel_row in related_elements.head(num_semantic_rels).iterrows():
                # Determine which element is the "other" one
                other_id = rel_row.get('target_element_id') if rel_row.get('source_element_id') == element_id else rel_row.get('source_element_id')
                
                if other_id:
                    other_element = elements_df[elements_df['element_id'] == other_id]
                    if not other_element.empty:
                        other_row = other_element.iloc[0]
                        semantic_rels.append({
                            'relationship_type': rel_row.get('relationship_type', 'semantic'),
                            'element_id': other_row['element_id'],
                            'element_type': other_row['element_type'],
                            'content_preview': other_row['content_preview'][:200]
                        })
            
            graphlet['semantic_relationships'] = semantic_rels
    
    except Exception as e:
        logger.warning(f"Error building graphlet for {element_id}: {e}")
    
    return graphlet


def _reconstruct_element_context(element_row, graphlet, reconstruction_config):
    """Reconstruct element with context in requested format."""
    format_type = reconstruction_config.get('format', 'markdown')
    include_context = reconstruction_config.get('include_context', True)
    
    try:
        content = element_row['content_preview']
        element_type = element_row['element_type']
        element_id = element_row['element_id']
        
        if format_type == 'markdown':
            reconstruction = f"## Element: {element_type} ({element_id})\n\n{content}\n"
            
            if include_context and graphlet:
                if graphlet['parents']:
                    reconstruction += "\n### Parent Context:\n"
                    for i, parent in enumerate(graphlet['parents']):
                        reconstruction += f"{i+1}. **{parent['element_type']}**: {parent['content_preview']}\n"
                
                if graphlet['siblings']:
                    reconstruction += "\n### Sibling Context:\n"
                    for i, sibling in enumerate(graphlet['siblings']):
                        reconstruction += f"- **{sibling['element_type']}**: {sibling['content_preview']}\n"
                
                if graphlet['semantic_relationships']:
                    reconstruction += "\n### Semantic Relationships:\n"
                    for rel in graphlet['semantic_relationships']:
                        reconstruction += f"- **{rel['relationship_type']}** → {rel['element_type']}: {rel['content_preview']}\n"
        
        elif format_type == 'html':
            reconstruction = f"<div class='element'><h3>Element: {element_type} ({element_id})</h3><p>{content}</p>"
            
            if include_context and graphlet:
                reconstruction += "<div class='context'>"
                if graphlet['parents']:
                    reconstruction += "<h4>Parent Context:</h4><ul>"
                    for parent in graphlet['parents']:
                        reconstruction += f"<li><strong>{parent['element_type']}</strong>: {parent['content_preview']}</li>"
                    reconstruction += "</ul>"
                
                if graphlet['siblings']:
                    reconstruction += "<h4>Sibling Context:</h4><ul>"
                    for sibling in graphlet['siblings']:
                        reconstruction += f"<li><strong>{sibling['element_type']}</strong>: {sibling['content_preview']}</li>"
                    reconstruction += "</ul>"
                
                if graphlet['semantic_relationships']:
                    reconstruction += "<h4>Semantic Relationships:</h4><ul>"
                    for rel in graphlet['semantic_relationships']:
                        reconstruction += f"<li><strong>{rel['relationship_type']}</strong> → {rel['element_type']}: {rel['content_preview']}</li>"
                    reconstruction += "</ul>"
                reconstruction += "</div>"
            
            reconstruction += "</div>"
        
        else:  # plain text
            reconstruction = f"Element: {element_type} ({element_id})\n{content}\n"
            
            if include_context and graphlet:
                if graphlet['parents']:
                    reconstruction += "\nParent Context:\n"
                    for i, parent in enumerate(graphlet['parents']):
                        reconstruction += f"  {i+1}. {parent['element_type']}: {parent['content_preview']}\n"
                
                if graphlet['siblings']:
                    reconstruction += "\nSibling Context:\n"
                    for sibling in graphlet['siblings']:
                        reconstruction += f"  - {sibling['element_type']}: {sibling['content_preview']}\n"
                
                if graphlet['semantic_relationships']:
                    reconstruction += "\nSemantic Relationships:\n"
                    for rel in graphlet['semantic_relationships']:
                        reconstruction += f"  - {rel['relationship_type']} → {rel['element_type']}: {rel['content_preview']}\n"
        
        return reconstruction
        
    except Exception as e:
        logger.warning(f"Error reconstructing element context: {e}")
        return f"Error: Could not reconstruct element context - {str(e)}"


def _load_document_metadata(base_path):
    """Load document metadata from parquet files."""
    import pandas as pd
    import glob
    import json
    
    try:
        pattern = f'{base_path}/documents/**/*.parquet'
        doc_files = glob.glob(pattern, recursive=True)
        
        doc_metadata = {}
        for file in doc_files:
            try:
                df = pd.read_parquet(file)
                for _, row in df.iterrows():
                    doc_id = row.get('doc_id')
                    if doc_id:
                        doc_metadata[doc_id] = {
                            'doc_type': row.get('doc_type'),
                            'source': row.get('source'), 
                            'title': row.get('metadata', {}).get('title'),
                            'created_at': row.get('created_at')
                        }
            except Exception:
                continue
        
        return doc_metadata
    except Exception as e:
        logger.warning(f"Could not load document metadata: {e}")
    
    return {}




def _reconstruct_element_context(element_row, graphlet, reconstruction_config):
    """Reconstruct element with context in specified format."""
    format_type = reconstruction_config.get('format', 'markdown').lower()
    include_context = reconstruction_config.get('include_context', True)
    
    content = element_row['content_preview']
    element_type = element_row['element_type']
    
    if format_type == 'markdown':
        reconstruction = f"## {element_type.title()} Element\n\n{content}\n\n"
        
        if include_context and graphlet:
            if graphlet['parents']:
                reconstruction += "### Parent Context\n\n"
                for parent in graphlet['parents']:
                    reconstruction += f"- **{parent['element_type']}**: {parent['content_preview']}\n"
                reconstruction += "\n"
            
            if graphlet['siblings']:
                reconstruction += "### Sibling Context\n\n" 
                for sibling in graphlet['siblings']:
                    reconstruction += f"- **{sibling['element_type']}**: {sibling['content_preview']}\n"
                reconstruction += "\n"
    
    elif format_type == 'html':
        reconstruction = f"<h2>{element_type.title()} Element</h2>\n<p>{content}</p>\n"
        
        if include_context and graphlet:
            if graphlet['parents']:
                reconstruction += "<h3>Parent Context</h3>\n<ul>\n"
                for parent in graphlet['parents']:
                    reconstruction += f"<li><strong>{parent['element_type']}</strong>: {parent['content_preview']}</li>\n"
                reconstruction += "</ul>\n"
    
    else:  # plain text
        reconstruction = f"{element_type.upper()}: {content}\n\n"
        
        if include_context and graphlet:
            if graphlet['parents']:
                reconstruction += "PARENT CONTEXT:\n"
                for parent in graphlet['parents']:
                    reconstruction += f"  - {parent['element_type']}: {parent['content_preview']}\n"
                reconstruction += "\n"
    
    return reconstruction

# Initialize Flask app
app = Flask(__name__)

# Configure CORS
cors_origins = os.environ.get('CORS_ORIGINS', '*').split(',')
CORS(app, origins=cors_origins)

# Register blueprints
app.register_blueprint(settings_bp)
app.register_blueprint(pipeline_bp)

# Get the directory where server.py is located
SERVER_DIR = os.path.dirname(os.path.abspath(__file__))

# Configuration from environment variables
CONFIG = {
    'HOST': os.environ.get('SERVER_HOST', '0.0.0.0'),
    'PORT': int(os.environ.get('SERVER_PORT', '5000')),
    'DEBUG': os.environ.get('DEBUG', 'False').lower() == 'true',
    'MAX_RESULTS': int(os.environ.get('MAX_RESULTS', '100')),
    'DEFAULT_RESULTS': int(os.environ.get('DEFAULT_RESULTS', '10')),
    'MIN_SCORE_THRESHOLD': float(os.environ.get('MIN_SCORE_THRESHOLD', '0.0')),
    'TIMEOUT': int(os.environ.get('REQUEST_TIMEOUT', '30')),
    'MAX_CONTENT_LENGTH': int(os.environ.get('MAX_CONTENT_LENGTH', '16777216')),  # 16MB
    'RATE_LIMIT': os.environ.get('RATE_LIMIT', '100 per minute'),
    'API_KEY': os.environ.get('API_KEY'),  # Optional API key for authentication
    'API_KEY_HEADER': os.environ.get('API_KEY_HEADER', 'X-API-Key'),
    'OPENAPI_SPEC_PATH': os.environ.get('OPENAPI_SPEC_PATH', os.path.join(SERVER_DIR, 'openapi.yaml')),
    'SWAGGER_UI_ENABLED': os.environ.get('SWAGGER_UI_ENABLED', 'True').lower() == 'true',
    'SWAGGER_UI_PATH': os.environ.get('SWAGGER_UI_PATH', '/docs'),
    'API_SPEC_PATH': os.environ.get('API_SPEC_PATH', '/api/spec'),
}

# Set Flask configuration
app.config['MAX_CONTENT_LENGTH'] = CONFIG['MAX_CONTENT_LENGTH']


# Load OpenAPI specification
def load_openapi_spec():
    """Load the OpenAPI specification from file."""
    try:
        spec_path = CONFIG['OPENAPI_SPEC_PATH']
        if not os.path.exists(spec_path):
            logger.warning(f"OpenAPI spec file not found at {spec_path}")
            return None

        with open(spec_path, 'r') as f:
            if spec_path.endswith('.yaml') or spec_path.endswith('.yml'):
                spec = yaml.safe_load(f)
            else:
                spec = json.load(f)

        # Update server URLs with current configuration
        if 'servers' not in spec:
            spec['servers'] = []

        # Add current server URL
        current_server = f"http://{CONFIG['HOST']}:{CONFIG['PORT']}"
        spec['servers'].insert(0, {
            'url': current_server,
            'description': 'Current server'
        })

        return spec
    except Exception as e:
        logger.error(f"Error loading OpenAPI spec: {str(e)}")
        return None


# Authentication middleware
def check_api_key():
    """Check API key if configured."""
    if CONFIG['API_KEY']:
        api_key = request.headers.get(CONFIG['API_KEY_HEADER'])
        if not api_key or api_key != CONFIG['API_KEY']:
            return jsonify({
                'error': 'Unauthorized',
                'message': 'Invalid or missing API key'
            }), 401
    return None


# Error handlers
@app.errorhandler(400)
def bad_request(error):
    return jsonify({
        'error': 'Bad Request',
        'message': str(error.description)
    }), 400


@app.errorhandler(404)
def not_found(error):
    return jsonify({
        'error': 'Not Found',
        'message': 'Resource not found'
    }), 404


@app.errorhandler(500)
def internal_error(error):
    logger.error(f"Internal server error: {str(error)}")
    return jsonify({
        'error': 'Internal Server Error',
        'message': 'An unexpected error occurred'
    }), 500


# Root endpoint with API documentation links
# Commented out to allow React app to be served at root
# @app.route('/', methods=['GET'])
# def root():
#     """Root endpoint with links to documentation."""
#     response_data = {
#         'name': 'Document Search API',
#         'version': '1.0.0',
#         'status': 'running',
#         'links': {
#             'api_documentation': CONFIG['SWAGGER_UI_PATH'] if CONFIG['SWAGGER_UI_ENABLED'] else None,
#             'openapi_spec': CONFIG['API_SPEC_PATH'],
#             'health': '/health',
#             'api_info': '/api/info'
#         }
#     }
#
#     return jsonify({k: v for k, v in response_data.items() if v is not None})

# API info endpoint (moved from root)
@app.route('/api/info', methods=['GET'])
def api_info():
    """API information endpoint."""
    response_data = {
        'name': 'Document Search API',
        'version': '1.0.0',
        'status': 'running',
        'links': {
            'api_documentation': CONFIG['SWAGGER_UI_PATH'] if CONFIG['SWAGGER_UI_ENABLED'] else None,
            'openapi_spec': CONFIG['API_SPEC_PATH'],
            'health': '/health'
        }
    }

    return jsonify({k: v for k, v in response_data.items() if v is not None})


# OpenAPI specification endpoint
@app.route(CONFIG['API_SPEC_PATH'], methods=['GET'])
def openapi_spec():
    """Serve the OpenAPI specification."""
    spec = load_openapi_spec()
    if spec is None:
        return jsonify({
            'error': 'Not Found',
            'message': 'OpenAPI specification not available'
        }), 404

    return jsonify(spec)


# Swagger UI endpoint
if CONFIG['SWAGGER_UI_ENABLED']:
    # Swagger UI HTML template
    SWAGGER_UI_TEMPLATE = """
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <title>Document Search API - Swagger UI</title>
        <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui.css">
        <style>
            html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
            *, *:before, *:after { box-sizing: inherit; }
            body { margin:0; background: #fafafa; }
        </style>
    </head>
    <body>
        <div id="swagger-ui"></div>
        <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-bundle.js"></script>
        <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-standalone-preset.js"></script>
        <script>
            window.onload = function() {
                const ui = SwaggerUIBundle({
                    url: "{{ openapi_url }}",
                    dom_id: '#swagger-ui',
                    deepLinking: true,
                    presets: [
                        SwaggerUIBundle.presets.apis,
                        SwaggerUIStandalonePreset
                    ],
                    plugins: [
                        SwaggerUIBundle.plugins.DownloadUrl
                    ],
                    layout: "StandaloneLayout",
                    defaultModelsExpandDepth: 1,
                    defaultModelExampleFormat: "value",
                    tryItOutEnabled: true,
                    persistAuthorization: true
                });

                window.ui = ui;
            };
        </script>
    </body>
    </html>
    """


    @app.route(CONFIG['SWAGGER_UI_PATH'], methods=['GET'])
    def swagger_ui():
        """Serve Swagger UI."""
        openapi_url = f"{CONFIG['API_SPEC_PATH']}"
        return render_template_string(SWAGGER_UI_TEMPLATE, openapi_url=openapi_url)


# Health check endpoint
@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint."""
    return jsonify({
        'status': 'healthy',
        'timestamp': datetime.utcnow().isoformat(),
        'version': '1.0.0'
    })


# API Info endpoint - commented out to avoid duplicate
# @app.route('/api/info', methods=['GET'])
# def api_info_old():
#     """Get API information and available endpoints."""
#     info_data = {
#         'name': 'Document Search API',
#         'version': '1.0.0',
#         'endpoints': {
#             '/health': 'Health check',
#             '/api/info': 'API information',
#             Search endpoints removed
#         },
#         'configuration': {
#             'max_results': CONFIG['MAX_RESULTS'],
#             'default_results': CONFIG['DEFAULT_RESULTS'],
#             'min_score_threshold': CONFIG['MIN_SCORE_THRESHOLD'],
#             'timeout': CONFIG['TIMEOUT']
#         }
#     }
# 
#     if CONFIG['SWAGGER_UI_ENABLED']:
#         info_data['documentation'] = CONFIG['SWAGGER_UI_PATH']
#         info_data['openapi_spec'] = CONFIG['API_SPEC_PATH']
# 
#     return jsonify(info_data)


# Helper function to extract topic parameters
def extract_topic_parameters(data):
    """Extract topic-related parameters from request data."""
    include_topics = data.get('include_topics')
    exclude_topics = data.get('exclude_topics')
    min_confidence = data.get('min_confidence')

    # Validate topic parameters
    if include_topics is not None and not isinstance(include_topics, list):
        raise BadRequest("'include_topics' must be a list of strings")

    if exclude_topics is not None and not isinstance(exclude_topics, list):
        raise BadRequest("'exclude_topics' must be a list of strings")

    if min_confidence is not None:
        if not isinstance(min_confidence, (int, float)) or min_confidence < 0.0 or min_confidence > 1.0:
            raise BadRequest("'min_confidence' must be a number between 0.0 and 1.0")

    return include_topics, exclude_topics, min_confidence


# Analytics registry endpoints
@app.route('/api/analytics/registry', methods=['GET'])
def get_analytics_registry():
    """
    Get list of available analytics backends from the registry.
    
    Returns:
        JSON object with backend names, types, and descriptions
    """
    try:
        _ensure_config_loaded()
        
        backends = _config.list_analytics_backends()
        
        # Format for API response (only include enabled backends)
        formatted_backends = {}
        for name, config in backends.items():
            # Check explicit enabled flag (defaults to True if not specified)
            enabled = config.get('enabled', True)
            
            if enabled:
                formatted_backends[name] = {
                    'type': config.get('type', 'unknown'),
                    'description': config.get('description', ''),
                    'enabled': enabled,
                    'available': True,  # Could add actual connectivity check here
                    
                    # Add AI agent selection metadata
                    'search_capabilities': config.get('search_capabilities', {}),
                    'best_for': config.get('best_for', []),
                    'optimal_queries': config.get('optimal_queries', []),
                    'performance': config.get('performance', {}),
                    'data_profile': config.get('data_profile', {})
                }
        
        # Add search configuration info
        search_config = _config.config.get('search', {})
        default_backend = search_config.get('default_backend')
        available_for_search = search_config.get('available_backends', list(backends.keys()))
        
        # Add AI agent selection rules
        agent_selection_rules = _config.config.get('agent_selection_rules', {})
        
        return jsonify({
            'backends': formatted_backends,
            'default_backend': default_backend,
            'available_for_search': available_for_search,
            'agent_selection_rules': agent_selection_rules,
            'total': len(backends)
        })
        
    except Exception as e:
        logger.error(f"Error fetching analytics registry: {str(e)}")
        return jsonify({'error': 'Failed to fetch analytics registry', 'message': str(e)}), 500


@app.route('/api/analytics/registry/<backend_name>', methods=['GET'])
def get_analytics_backend_details(backend_name):
    """
    Get detailed configuration for a specific analytics backend.
    
    Args:
        backend_name: Name of the backend in the registry
        
    Returns:
        JSON object with full backend configuration (sanitized)
    """
    try:
        _ensure_config_loaded()
        
        backend_config = _config.get_analytics_backend(backend_name)
        
        if not backend_config:
            return jsonify({'error': f'Backend {backend_name} not found in registry'}), 404
        
        # Sanitize sensitive information
        sanitized_config = backend_config.copy()
        sensitive_keys = ['password', 'secret_key', 'access_key', 'api_key', 'token']
        
        def sanitize_dict(d):
            for key in list(d.keys()):
                if any(sensitive in key.lower() for sensitive in sensitive_keys):
                    d[key] = '***REDACTED***'
                elif isinstance(d[key], dict):
                    sanitize_dict(d[key])
            return d
        
        sanitized_config = sanitize_dict(sanitized_config)
        
        return jsonify({
            'name': backend_name,
            'config': sanitized_config
        })
        
    except Exception as e:
        logger.error(f"Error fetching backend details for {backend_name}: {str(e)}")
        return jsonify({'error': 'Failed to fetch backend details', 'message': str(e)}), 500


@app.route('/api/analytics/recommend', methods=['POST'])
def recommend_analytics_backend():
    """
    Recommend optimal analytics backend(s) for a given query.
    
    This endpoint helps AI agents automatically select the best backend
    based on query characteristics and requirements.
    
    Expected JSON payload:
    {
        "query": "sales performance trends over time",
        "requirements": {
            "latency": "real_time",  # real_time, interactive, analytical, batch
            "complexity": "analytical_aggregation",  # simple_keyword, analytical_aggregation, etc.
            "data_size": "large_dataset"  # small_dataset, medium_dataset, large_dataset
        },
        "context": {
            "domain": "financial",  # Optional: content domain
            "document_types": ["10-K", "earnings_calls"]  # Optional: specific doc types
        }
    }
    
    Returns:
        JSON with ranked backend recommendations and reasoning
    """
    try:
        _ensure_config_loaded()
        
        data = request.get_json()
        if not data:
            return jsonify({'error': 'JSON payload required'}), 400
            
        query = data.get('query', '').lower()
        requirements = data.get('requirements', {})
        context = data.get('context', {})
        
        # Get agent selection rules and backends
        agent_rules = _config.config.get('agent_selection_rules', {})
        backends = _config.list_analytics_backends()
        
        recommendations = []
        
        # Score backends based on query patterns
        query_patterns = agent_rules.get('query_patterns', {})
        for pattern_name, pattern_config in query_patterns.items():
            keywords = pattern_config.get('keywords', [])
            pattern_backends = pattern_config.get('backends', [])
            
            # Check if query contains pattern keywords
            keyword_matches = sum(1 for keyword in keywords if keyword in query)
            if keyword_matches > 0:
                for backend_name in pattern_backends:
                    if backend_name in backends:
                        recommendations.append({
                            'backend': backend_name,
                            'score': keyword_matches / len(keywords),
                            'reason': f'Query matches {pattern_name} pattern ({keyword_matches}/{len(keywords)} keywords)',
                            'pattern': pattern_name
                        })
        
        # Score based on requirements
        if requirements:
            complexity = requirements.get('complexity')
            latency = requirements.get('latency') 
            data_size = requirements.get('data_size')
            
            complexity_mapping = agent_rules.get('complexity_mapping', {})
            performance_requirements = agent_rules.get('performance_requirements', {})
            data_scale = agent_rules.get('data_scale', {})
            
            # Add complexity-based recommendations
            if complexity and complexity in complexity_mapping:
                for backend_name in complexity_mapping[complexity]:
                    if backend_name in backends:
                        recommendations.append({
                            'backend': backend_name,
                            'score': 0.8,
                            'reason': f'Optimized for {complexity} queries',
                            'criteria': 'complexity'
                        })
            
            # Add performance-based recommendations  
            if latency and latency in performance_requirements:
                for backend_name in performance_requirements[latency]:
                    if backend_name in backends:
                        recommendations.append({
                            'backend': backend_name,
                            'score': 0.9,
                            'reason': f'Meets {latency} latency requirements',
                            'criteria': 'performance'
                        })
            
            # Add data size recommendations
            if data_size and data_size in data_scale:
                for backend_name in data_scale[data_size]:
                    if backend_name in backends:
                        recommendations.append({
                            'backend': backend_name,
                            'score': 0.7,
                            'reason': f'Optimized for {data_size}',
                            'criteria': 'data_scale'
                        })
        
        # Aggregate scores by backend
        backend_scores = {}
        for rec in recommendations:
            backend = rec['backend']
            if backend not in backend_scores:
                backend_scores[backend] = {
                    'backend': backend,
                    'total_score': 0,
                    'reasons': [],
                    'config': backends[backend]
                }
            backend_scores[backend]['total_score'] += rec['score']
            backend_scores[backend]['reasons'].append(rec['reason'])
        
        # Sort by score and return top recommendations
        sorted_recommendations = sorted(
            backend_scores.values(),
            key=lambda x: x['total_score'],
            reverse=True
        )
        
        # Add default backend if no matches
        if not sorted_recommendations:
            default_backend = _config.config.get('search', {}).get('default_backend', 'parquet_lake')
            if default_backend in backends:
                sorted_recommendations = [{
                    'backend': default_backend,
                    'total_score': 0.5,
                    'reasons': ['Default backend - no specific pattern matches'],
                    'config': backends[default_backend]
                }]
        
        return jsonify({
            'query': data.get('query', ''),
            'recommendations': sorted_recommendations[:3],  # Top 3 recommendations
            'total_backends_considered': len(backends),
            'selection_criteria': list(requirements.keys()) if requirements else []
        })
        
    except Exception as e:
        logger.error(f"Error in backend recommendation: {str(e)}")
        return jsonify({'error': 'Failed to recommend backend', 'message': str(e)}), 500


# =============================================================================
# CONFIGURATION AND ONTOLOGY MANAGEMENT API ENDPOINTS
# =============================================================================

@app.route('/api/config', methods=['GET'])
def get_config_endpoint():
    """
    Get current configuration.
    
    Returns the current configuration as JSON.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_config_loaded()
        
        # Get configuration, removing sensitive information
        config_dict = _config.config.copy()
        
        # Remove any sensitive fields if needed
        # (API keys, passwords, etc. - currently none in our config)
        
        return jsonify({
            'config': config_dict,
            'status': 'success'
        })
        
    except Exception as e:
        logger.error(f"Get config error: {str(e)}")
        raise InternalServerError(f"Failed to get configuration: {str(e)}")


@app.route('/api/config', methods=['PUT'])
def update_config_endpoint():
    """
    Update configuration.
    
    Request body should contain the new configuration.
    """
    # Check API key if required  
    auth_response = check_api_key()
    if auth_response:
        return auth_response
        
    try:
        # Parse request JSON
        data = request.get_json()
        if not data or 'config' not in data:
            raise BadRequest("Request body must contain 'config' field")
        
        new_config = data['config']
        
        # Validate the new configuration by creating a temporary Config object
        import tempfile
        import os
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            yaml.dump(new_config, f)
            temp_config_path = f.name
        
        try:
            # Try to create a Config object with the new configuration
            from go_doc_go.config import Config
            test_config = Config(temp_config_path)
            
            # If successful, save to the actual config file
            config_path = os.environ.get('GO_DOC_GO_CONFIG_PATH', 'config.yaml')
            with open(config_path, 'w') as f:
                yaml.dump(new_config, f, default_flow_style=False)
            
            # Reload the global configuration
            global _config, db, resolver
            _config = None  # Force re-initialization
            db = None
            resolver = None
            _ensure_config_loaded()
            
            logger.info("Configuration updated successfully")
            
            return jsonify({
                'status': 'success',
                'message': 'Configuration updated successfully'
            })
            
        finally:
            # Clean up temp file
            os.unlink(temp_config_path)
            
    except Exception as e:
        logger.error(f"Update config error: {str(e)}")
        raise InternalServerError(f"Failed to update configuration: {str(e)}")


@app.route('/api/config/validate', methods=['POST'])
def validate_config_endpoint():
    """
    Validate configuration without saving.
    
    Request body should contain the configuration to validate.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
        
    try:
        # Parse request JSON
        data = request.get_json()
        if not data or 'config' not in data:
            raise BadRequest("Request body must contain 'config' field")
        
        new_config = data['config']
        
        # Validate by creating a temporary Config object
        import tempfile
        import os
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            yaml.dump(new_config, f)
            temp_config_path = f.name
        
        validation_errors = []
        try:
            from go_doc_go.config import Config
            test_config = Config(temp_config_path)
            
        except Exception as e:
            validation_errors.append(str(e))
        finally:
            os.unlink(temp_config_path)
        
        if validation_errors:
            return jsonify({
                'valid': False,
                'errors': validation_errors
            })
        else:
            return jsonify({
                'valid': True,
                'message': 'Configuration is valid'
            })
            
    except Exception as e:
        logger.error(f"Validate config error: {str(e)}")
        raise InternalServerError(f"Failed to validate configuration: {str(e)}")


@app.route('/api/ontologies', methods=['GET'])
def list_ontologies_endpoint():
    """
    List all available ontologies.
    
    Returns a list of ontologies with basic information.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_config_loaded()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'ontologies': [],
                'message': 'Domain detection not enabled'
            })
        
        ontologies = []
        for name in ontology_manager.loader.list_ontologies():
            ontology = ontology_manager.loader.get_ontology(name)
            if ontology:
                ontologies.append({
                    'name': ontology.name,
                    'version': ontology.version,
                    'domain': ontology.domain,
                    'description': ontology.description,
                    'active': name in ontology_manager.active_domains,
                    'terms_count': len(ontology.terms),
                    'entity_mappings_count': len(ontology.element_entity_mappings),
                    'relationship_rules_count': len(ontology.entity_relationship_rules)
                })
        
        return jsonify({
            'ontologies': ontologies,
            'total': len(ontologies),
            'active_domains': ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"List ontologies error: {str(e)}")
        raise InternalServerError(f"Failed to list ontologies: {str(e)}")


@app.route('/api/ontologies/<string:name>', methods=['GET'])
def get_ontology_endpoint(name):
    """
    Get a specific ontology by name.
    
    Returns the full ontology configuration.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_config_loaded()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'error': 'Domain detection not enabled'
            }), 404
        
        ontology = ontology_manager.loader.get_ontology(name)
        if not ontology:
            return jsonify({
                'error': f'Ontology "{name}" not found'
            }), 404
        
        # Convert ontology to dictionary for JSON serialization
        ontology_dict = {
            'name': ontology.name,
            'version': ontology.version,
            'domain': ontology.domain,
            'description': ontology.description,
            'document_types': ontology.document_types,
            'terms': [
                {
                    'id': term.id,
                    'name': term.name,
                    'description': term.description,
                    'synonyms': term.synonyms,
                    'category': term.category,
                    'attributes': term.attributes
                }
                for term in ontology.terms
            ],
            'element_entity_mappings': [
                {
                    'entity_type': mapping.entity_type,
                    'description': mapping.description,
                    'document_types': mapping.document_types,
                    'element_types': mapping.element_types,
                    'extraction_rules': [
                        {
                            'type': rule.type,
                            'pattern': rule.pattern,
                            'field_path': rule.field_path,
                            'confidence': rule.confidence,
                            'required': rule.required,
                            'description': rule.description
                        }
                        for rule in mapping.extraction_rules
                    ]
                }
                for mapping in ontology.element_entity_mappings
            ],
            'entity_relationship_rules': [
                {
                    'name': rule.name,
                    'description': rule.description,
                    'source_entity_type': rule.source_entity_type,
                    'relationship_type': rule.relationship_type,
                    'target_entity_type': rule.target_entity_type,
                    'confidence': rule.confidence,
                    'matching_criteria': {
                        'same_source_element': rule.matching_criteria.same_source_element,
                        'metadata_match': rule.matching_criteria.metadata_match,
                        'content_proximity': rule.matching_criteria.content_proximity
                    }
                }
                for rule in ontology.entity_relationship_rules
            ]
        }
        
        return jsonify({
            'ontology': ontology_dict,
            'active': name in ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"Get ontology error: {str(e)}")
        raise InternalServerError(f"Failed to get ontology: {str(e)}")


@app.route('/api/ontologies/<string:name>', methods=['PUT'])
def update_ontology_endpoint(name):
    """
    Update a specific ontology.
    
    Request body should contain the ontology configuration.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        # Parse request JSON
        data = request.get_json()
        if not data or 'ontology' not in data:
            raise BadRequest("Request body must contain 'ontology' field")
        
        ontology_data = data['ontology']
        
        # Validate ontology name matches URL parameter
        if ontology_data.get('name') != name:
            raise BadRequest("Ontology name in body must match URL parameter")
        
        # Find the ontology file path (this is a simplification)
        import os
        from pathlib import Path
        
        # Look in common ontology directories
        possible_paths = [
            Path('examples/ontologies') / f'{name}.yaml',
            Path('ontologies') / f'{name}.yaml',
            Path('.') / f'{name}.yaml'
        ]
        
        ontology_path = None
        for path in possible_paths:
            if path.exists():
                ontology_path = str(path)
                break
        
        if not ontology_path:
            # Create new ontology file in examples/ontologies
            ontology_path = f'examples/ontologies/{name}.yaml'
            os.makedirs(os.path.dirname(ontology_path), exist_ok=True)
        
        # Validate the ontology by loading it
        try:
            from go_doc_go.domain import OntologyLoader
            loader = OntologyLoader()
            test_ontology = loader.load_from_dict(ontology_data)
            
            # If validation passes, save the file
            with open(ontology_path, 'w') as f:
                yaml.dump(ontology_data, f, default_flow_style=False)
            
            # Reload in the ontology manager
            _ensure_config_loaded()
            ontology_manager = _config.get_ontology_manager()
            if ontology_manager:
                # Clear and reload
                ontology_manager.loader.clear()
                ontology_manager.active_domains.clear()
                
                # Reload all ontologies from config
                domain_config = _config.config.get("relationship_detection", {}).get("domain", {})
                ontologies = domain_config.get("ontologies", [])
                for ontology_config in ontologies:
                    if isinstance(ontology_config, dict):
                        path = ontology_config.get("path")
                        active = ontology_config.get("active", True)
                        
                        if path and os.path.exists(path):
                            try:
                                loaded_name = ontology_manager.load_ontology(path)
                                if active:
                                    ontology_manager.activate_domain(loaded_name)
                            except Exception as e:
                                logger.error(f"Failed to reload ontology from {path}: {e}")
            
            logger.info(f"Ontology '{name}' updated successfully")
            
            return jsonify({
                'status': 'success',
                'message': f'Ontology "{name}" updated successfully'
            })
            
        except Exception as e:
            return jsonify({
                'status': 'error',
                'message': f'Invalid ontology: {str(e)}'
            }), 400
            
    except Exception as e:
        logger.error(f"Update ontology error: {str(e)}")
        raise InternalServerError(f"Failed to update ontology: {str(e)}")


@app.route('/api/domain/active', methods=['GET'])
def get_active_domains_endpoint():
    """
    Get active domains.
    
    Returns list of currently active domains.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_config_loaded()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'active_domains': [],
                'message': 'Domain detection not enabled'
            })
        
        return jsonify({
            'active_domains': ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"Get active domains error: {str(e)}")
        raise InternalServerError(f"Failed to get active domains: {str(e)}")


@app.route('/api/domain/<string:name>/activate', methods=['POST'])
def activate_domain_endpoint(name):
    """
    Activate a domain.
    
    Makes the specified domain active.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_config_loaded()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'error': 'Domain detection not enabled'
            }), 400
        
        # Check if domain exists
        if name not in ontology_manager.loader.ontologies:
            return jsonify({
                'error': f'Domain "{name}" not found'
            }), 404
        
        ontology_manager.activate_domain(name)
        
        return jsonify({
            'status': 'success',
            'message': f'Domain "{name}" activated',
            'active_domains': ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"Activate domain error: {str(e)}")
        raise InternalServerError(f"Failed to activate domain: {str(e)}")


@app.route('/api/domain/<string:name>/deactivate', methods=['POST'])
def deactivate_domain_endpoint(name):
    """
    Deactivate a domain.
    
    Makes the specified domain inactive.
    """
    # Check API key if required
    auth_response = check_api_key()
    if auth_response:
        return auth_response
    
    try:
        _ensure_config_loaded()
        
        ontology_manager = _config.get_ontology_manager()
        if not ontology_manager:
            return jsonify({
                'error': 'Domain detection not enabled'
            }), 400
        
        ontology_manager.deactivate_domain(name)
        
        return jsonify({
            'status': 'success', 
            'message': f'Domain "{name}" deactivated',
            'active_domains': ontology_manager.active_domains
        })
        
    except Exception as e:
        logger.error(f"Deactivate domain error: {str(e)}")
        raise InternalServerError(f"Failed to deactivate domain: {str(e)}")


# Static file serving for React frontend
# Get the path to the frontend build directory
FRONTEND_BUILD_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), 'frontend', 'dist')

@app.route('/')
def serve_index():
    """Serve the React app index.html for the root route."""
    if os.path.exists(os.path.join(FRONTEND_BUILD_DIR, 'index.html')):
        return send_file(os.path.join(FRONTEND_BUILD_DIR, 'index.html'))
    else:
        return jsonify({
            "message": "React frontend not built. Run 'npm run build' in the frontend directory.",
            "api_docs": f"http://{CONFIG['HOST']}:{CONFIG['PORT']}{CONFIG['SWAGGER_UI_PATH']}"
        }), 404

@app.route('/static/<path:filename>')
def serve_static(filename):
    """Serve static files from the React build."""
    return send_from_directory(os.path.join(FRONTEND_BUILD_DIR, 'static'), filename)

@app.route('/assets/<path:filename>')
def serve_assets(filename):
    """Serve asset files from the React build."""
    return send_from_directory(os.path.join(FRONTEND_BUILD_DIR, 'assets'), filename)

# Catch-all route for React Router (must be after API routes)
@app.route('/<path:path>')
def serve_react_app(path):
    """
    Catch-all route to serve the React app for client-side routing.
    This should handle all frontend routes like /config, /ontologies, etc.
    """
    # Don't intercept API routes
    if path.startswith('api/'):
        return jsonify({"error": "API endpoint not found"}), 404
    
    # Serve index.html for all other routes (React Router will handle them)
    if os.path.exists(os.path.join(FRONTEND_BUILD_DIR, 'index.html')):
        return send_file(os.path.join(FRONTEND_BUILD_DIR, 'index.html'))
    else:
        return jsonify({
            "message": "React frontend not built. Run 'npm run build' in the frontend directory.",
            "available_apis": {
                # Search endpoints removed
                "config": f"/api/config", 
                "ontologies": f"/api/ontologies",
                "domains": f"/api/domain"
            }
        }), 404


# Optional: Rate limiting if using Flask-Limiter
try:
    from flask_limiter import Limiter
    from flask_limiter.util import get_remote_address

    # Initialize Limiter with first argument as the key_func (not a parameter name)
    limiter = Limiter(
        get_remote_address,  # First argument is key_func (no parameter name)
        app=app,  # Pass app as a keyword argument
        default_limits=[CONFIG['RATE_LIMIT']],
        storage_uri="memory://",
        strategy="fixed-window"
    )

    # Rate limiting for search endpoints removed as search endpoints have been deleted
    
    # Apply rate limiting to config/ontology endpoints
    limiter.limit(CONFIG['RATE_LIMIT'])(get_config_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(update_config_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(validate_config_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(list_ontologies_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(get_ontology_endpoint)
    limiter.limit(CONFIG['RATE_LIMIT'])(update_ontology_endpoint)

    logger.info(f"Rate limiting enabled: {CONFIG['RATE_LIMIT']}")
except ImportError:
    logger.warning("Flask-Limiter not installed, rate limiting disabled")


# Startup message
def print_startup_info():
    """Print startup information."""
    logger.info("=" * 50)
    logger.info("Document Search API Server Starting")
    logger.info("=" * 50)
    logger.info(f"Server URL: http://{CONFIG['HOST']}:{CONFIG['PORT']}")
    logger.info(f"API Documentation: http://{CONFIG['HOST']}:{CONFIG['PORT']}{CONFIG['SWAGGER_UI_PATH']}")
    logger.info(f"OpenAPI Spec: http://{CONFIG['HOST']}:{CONFIG['PORT']}{CONFIG['API_SPEC_PATH']}")
    logger.info(f"Debug Mode: {CONFIG['DEBUG']}")
    logger.info(f"Authentication: {'Enabled' if CONFIG['API_KEY'] else 'Disabled'}")
    logger.info(f"Rate Limiting: {CONFIG['RATE_LIMIT']}")
    logger.info("Available Endpoints:")
    logger.info("  Search Endpoints:")
    # Search endpoints removed - no longer available
    logger.info("  Configuration Management:")
    logger.info("    GET /api/config - Get current configuration")
    logger.info("    PUT /api/config - Update configuration")
    logger.info("    POST /api/config/validate - Validate configuration")
    logger.info("  Ontology Management:")
    logger.info("    GET /api/ontologies - List all ontologies")
    logger.info("    GET /api/ontologies/<name> - Get specific ontology")
    logger.info("    PUT /api/ontologies/<name> - Update ontology")
    logger.info("  Domain Management:")
    logger.info("    GET /api/domain/active - Get active domains")
    logger.info("    POST /api/domain/<name>/activate - Activate domain")
    logger.info("    POST /api/domain/<name>/deactivate - Deactivate domain")
    logger.info("=" * 50)


# Main entry point
if __name__ == '__main__':
    print_startup_info()
    app.run(
        host=CONFIG['HOST'],
        port=CONFIG['PORT'],
        debug=CONFIG['DEBUG'],
        threaded=True
    )
