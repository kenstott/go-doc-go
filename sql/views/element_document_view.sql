-- Enhanced element view with document attributes for ontology generation
CREATE OR REPLACE VIEW element_document_enriched AS
SELECT
    -- Element core fields
    e.element_pk,
    e.element_id,
    e.doc_id,
    e.element_type,
    e.parent_id,
    e.content_preview,
    e.content_location,
    e.content_hash,
    e.temporal_value,
    e.metadata as element_metadata,
    e.document_position,

    -- Document attributes
    d.source,
    d.doc_type,
    d.content_hash as doc_content_hash,
    d.metadata as document_metadata,
    d.created_at as doc_created_at,
    d.updated_at as doc_updated_at,

    -- Derived fields for ontology analysis
    CASE
        WHEN e.metadata->>'element_name' IS NOT NULL THEN e.metadata->>'element_name'
        WHEN e.metadata->>'tag_name' IS NOT NULL THEN e.metadata->>'tag_name'
        WHEN e.metadata->>'field_name' IS NOT NULL THEN e.metadata->>'field_name'
        WHEN e.metadata->>'column_name' IS NOT NULL THEN e.metadata->>'column_name'
        ELSE 'unknown'
    END as structural_name,

    CASE
        WHEN e.metadata->>'path' IS NOT NULL THEN e.metadata->>'path'
        WHEN e.metadata->>'xpath' IS NOT NULL THEN e.metadata->>'xpath'
        WHEN e.metadata->>'json_path' IS NOT NULL THEN e.metadata->>'json_path'
        WHEN e.metadata->>'css_selector' IS NOT NULL THEN e.metadata->>'css_selector'
        ELSE NULL
    END as structural_path,

    -- Extract attributes as text for searchability
    CASE
        WHEN e.metadata->'attributes' IS NOT NULL THEN
            string_agg(
                CONCAT(key, ':', value),
                ' '
            ) OVER (PARTITION BY e.element_pk)
        ELSE NULL
    END as attributes_text,

    -- Document type classification
    CASE
        WHEN d.source ILIKE '%.xml' OR d.doc_type = 'xml' THEN 'xml'
        WHEN d.source ILIKE '%.json' OR d.doc_type = 'json' THEN 'json'
        WHEN d.source ILIKE '%.csv' OR d.doc_type = 'csv' THEN 'csv'
        WHEN d.source ILIKE '%.xlsx' OR d.doc_type = 'xlsx' THEN 'xlsx'
        WHEN d.source ILIKE '%.html' OR d.doc_type = 'html' THEN 'html'
        WHEN d.source ILIKE '%.pdf' OR d.doc_type = 'pdf' THEN 'pdf'
        ELSE 'other'
    END as format_type,

    -- SEC form detection
    CASE
        WHEN d.source ILIKE '%form%4%' OR d.source ILIKE '%ownership%' THEN 'sec_form_4'
        WHEN d.source ILIKE '%form%3%' THEN 'sec_form_3'
        WHEN d.source ILIKE '%form%5%' THEN 'sec_form_5'
        WHEN d.source ILIKE '%10-k%' THEN 'sec_10k'
        WHEN d.source ILIKE '%10-q%' THEN 'sec_10q'
        WHEN d.source ILIKE '%earnings%' THEN 'earnings_report'
        ELSE 'other'
    END as document_category,

    -- Temporal indicators
    CASE
        WHEN e.temporal_value IS NOT NULL THEN TRUE
        WHEN e.content_preview ~ '\d{4}-\d{2}-\d{2}' THEN TRUE
        WHEN e.structural_name ILIKE '%date%' THEN TRUE
        ELSE FALSE
    END as has_temporal_value,

    -- Content length for sampling
    LENGTH(e.content_preview) as content_length,

    -- Hierarchy depth
    CASE
        WHEN e.metadata->>'path' IS NOT NULL THEN
            array_length(string_to_array(e.metadata->>'path', '/'), 1) - 1
        ELSE 0
    END as hierarchy_depth

FROM elements e
JOIN documents d ON e.doc_id = d.doc_id
WHERE e.element_type IS NOT NULL;

-- Create indexes for efficient sampling
CREATE INDEX IF NOT EXISTS idx_element_document_format_type
ON element_document_enriched USING HASH(format_type);

CREATE INDEX IF NOT EXISTS idx_element_document_category
ON element_document_enriched USING HASH(document_category);

CREATE INDEX IF NOT EXISTS idx_element_document_structural_name
ON element_document_enriched USING HASH(structural_name);

CREATE INDEX IF NOT EXISTS idx_element_document_element_type
ON element_document_enriched USING HASH(element_type);

-- Statistics view for sampling
CREATE OR REPLACE VIEW element_sampling_stats AS
SELECT
    format_type,
    document_category,
    element_type,
    structural_name,
    COUNT(*) as element_count,
    COUNT(DISTINCT doc_id) as document_count,
    AVG(content_length) as avg_content_length,
    AVG(hierarchy_depth) as avg_hierarchy_depth,
    COUNT(*) FILTER (WHERE has_temporal_value) as temporal_element_count,
    array_agg(DISTINCT structural_path) FILTER (WHERE structural_path IS NOT NULL) as sample_paths
FROM element_document_enriched
GROUP BY format_type, document_category, element_type, structural_name
ORDER BY element_count DESC;