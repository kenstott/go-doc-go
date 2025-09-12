-- Pipeline Configuration Database Schema
-- SQLite database for storing and managing multiple processing pipelines

-- Pipelines table - stores pipeline configurations
CREATE TABLE IF NOT EXISTS pipelines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    config_yaml TEXT NOT NULL,  -- Full YAML configuration as text
    version INTEGER NOT NULL DEFAULT 1,  -- For optimistic locking
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT,
    is_active BOOLEAN DEFAULT true,
    tags TEXT,  -- JSON array of tags ["financial", "experimental"]
    template_name TEXT  -- Reference to template used to create this pipeline
);

-- Pipeline executions table - tracks pipeline execution history
CREATE TABLE IF NOT EXISTS pipeline_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_id INTEGER NOT NULL,
    pipeline_version INTEGER NOT NULL,  -- Version of pipeline when executed
    run_id TEXT UNIQUE NOT NULL,  -- Unique execution identifier
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'paused', 'cancelled')),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    documents_processed INTEGER DEFAULT 0,
    documents_total INTEGER DEFAULT 0,
    errors_count INTEGER DEFAULT 0,
    warnings_count INTEGER DEFAULT 0,
    worker_count INTEGER DEFAULT 1,
    config_snapshot TEXT,  -- Copy of config YAML at execution time
    execution_log TEXT,    -- Execution logs and error messages
    metadata TEXT,         -- JSON metadata about execution
    FOREIGN KEY (pipeline_id) REFERENCES pipelines(id) ON DELETE CASCADE
);

-- Pipeline execution stages - tracks detailed progress through pipeline stages
CREATE TABLE IF NOT EXISTS execution_stages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    execution_id INTEGER NOT NULL,
    stage_name TEXT NOT NULL,  -- 'ingestion', 'extraction', 'relationships', etc.
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped')),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    documents_processed INTEGER DEFAULT 0,
    progress_percentage REAL DEFAULT 0,
    stage_log TEXT,
    FOREIGN KEY (execution_id) REFERENCES pipeline_executions(id) ON DELETE CASCADE
);

-- Pipeline templates - predefined configurations for common use cases
CREATE TABLE IF NOT EXISTS pipeline_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    category TEXT,  -- 'financial', 'technical', 'legal', 'research', etc.
    config_yaml TEXT NOT NULL,
    is_builtin BOOLEAN DEFAULT false,  -- System templates vs user-created
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tags TEXT  -- JSON array of tags
);

-- Pipeline dependencies - track relationships between pipelines
CREATE TABLE IF NOT EXISTS pipeline_dependencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_id INTEGER NOT NULL,
    depends_on_pipeline_id INTEGER NOT NULL,
    dependency_type TEXT DEFAULT 'data',  -- 'data', 'config', 'resource'
    FOREIGN KEY (pipeline_id) REFERENCES pipelines(id) ON DELETE CASCADE,
    FOREIGN KEY (depends_on_pipeline_id) REFERENCES pipelines(id) ON DELETE CASCADE,
    UNIQUE(pipeline_id, depends_on_pipeline_id)
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_pipelines_name ON pipelines(name);
CREATE INDEX IF NOT EXISTS idx_pipelines_active ON pipelines(is_active);
CREATE INDEX IF NOT EXISTS idx_pipelines_created_at ON pipelines(created_at);
CREATE INDEX IF NOT EXISTS idx_executions_pipeline_id ON pipeline_executions(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_executions_run_id ON pipeline_executions(run_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON pipeline_executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_started_at ON pipeline_executions(started_at);
CREATE INDEX IF NOT EXISTS idx_execution_stages_execution_id ON execution_stages(execution_id);
CREATE INDEX IF NOT EXISTS idx_templates_category ON pipeline_templates(category);

-- Triggers for updating timestamps
CREATE TRIGGER IF NOT EXISTS update_pipelines_updated_at 
    AFTER UPDATE ON pipelines
    FOR EACH ROW
    BEGIN
        UPDATE pipelines SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

-- Insert default pipeline templates
INSERT OR IGNORE INTO pipeline_templates (name, description, category, config_yaml, is_builtin, tags) VALUES 
(
    'Financial Analysis',
    'Complete pipeline for processing SEC filings and financial documents with XBRL support',
    'financial',
    '# Financial Analysis Pipeline Template
storage:
  backend: postgresql
  host: ${DB_HOST:-localhost}
  port: ${DB_PORT:-5432}
  database: ${DB_NAME:-financial_analysis}
  user: ${DB_USER:-postgres}
  password: ${DB_PASSWORD}

embedding:
  enabled: true
  provider: fastembed
  model: "sentence-transformers/all-MiniLM-L6-v2"
  dimensions: 384

entity_extraction:
  enabled: true
  load_builtin: true
  builtin:
    financial:
      enabled: true
    temporal:
      enabled: true

content_sources:
  - name: "sec_filings"
    type: "duckdb"
    database_path: "/data/sec-filings"
    enable_hive_partitioning: true

relationship_detection:
  enabled: true
  structural: true
  semantic: true
  semantic_config:
    similarity_threshold: 0.7

logging:
  level: INFO
  file: "./logs/financial_analysis.log"',
    true,
    '["financial", "sec", "xbrl", "postgresql"]'
),
(
    'Technical Documentation',
    'Pipeline for processing technical documentation from Confluence and file systems',
    'technical',
    '# Technical Documentation Pipeline Template
storage:
  backend: sqlite
  path: ./technical_docs.db

embedding:
  enabled: true
  provider: fastembed
  model: "sentence-transformers/all-MiniLM-L6-v2"
  dimensions: 384

content_sources:
  - name: "confluence_docs"
    type: "confluence"
    base_url: ${CONFLUENCE_URL}
    username: ${CONFLUENCE_USER}
    api_token: ${CONFLUENCE_TOKEN}
    include_pages: true
    include_attachments: false
  
  - name: "local_docs"
    type: "file"
    base_path: "./docs"
    file_pattern: "**/*.{md,rst,txt}"

relationship_detection:
  enabled: true
  structural: true
  cross_document_semantic:
    enabled: true
    similarity_threshold: 0.75

logging:
  level: INFO
  file: "./logs/technical_docs.log"',
    true,
    '["technical", "confluence", "documentation", "sqlite"]'
),
(
    'Legal Processing',
    'Pipeline for processing legal documents with contract and entity extraction',
    'legal',
    '# Legal Processing Pipeline Template
storage:
  backend: neo4j
  uri: ${NEO4J_URI:-bolt://localhost:7687}
  username: ${NEO4J_USER:-neo4j}
  password: ${NEO4J_PASSWORD}

embedding:
  enabled: true
  provider: fastembed
  model: "sentence-transformers/all-MiniLM-L6-v2"
  dimensions: 384

entity_extraction:
  enabled: true
  load_builtin: true
  builtin:
    legal:
      enabled: true
    temporal:
      enabled: true

content_sources:
  - name: "sharepoint_legal"
    type: "sharepoint"
    site_url: ${SHAREPOINT_URL}
    libraries: ["Legal Documents", "Contracts"]
    file_pattern: "*.{pdf,docx}"

relationship_detection:
  enabled: true
  structural: true
  semantic: true
  domain:
    enabled: true

logging:
  level: INFO
  file: "./logs/legal_processing.log"',
    true,
    '["legal", "contracts", "neo4j", "sharepoint"]'
);

-- Views for easier querying
CREATE VIEW IF NOT EXISTS active_pipelines AS
SELECT 
    id,
    name,
    description,
    version,
    created_at,
    updated_at,
    tags
FROM pipelines 
WHERE is_active = true;

CREATE VIEW IF NOT EXISTS recent_executions AS
SELECT 
    pe.id,
    pe.run_id,
    p.name as pipeline_name,
    pe.status,
    pe.started_at,
    pe.completed_at,
    pe.documents_processed,
    pe.documents_total,
    pe.errors_count,
    pe.worker_count,
    CASE 
        WHEN pe.completed_at IS NOT NULL AND pe.started_at IS NOT NULL 
        THEN CAST((julianday(pe.completed_at) - julianday(pe.started_at)) * 24 * 60 AS INTEGER)
        ELSE NULL 
    END as duration_minutes
FROM pipeline_executions pe
JOIN pipelines p ON pe.pipeline_id = p.id
ORDER BY pe.started_at DESC;