-- Initialize test database schema for go-doc-go
-- This script runs automatically when the container starts

-- Create test user if not exists (in case we need additional users)
-- Main user is created via environment variables

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search

-- Create test schema
CREATE SCHEMA IF NOT EXISTS test_schema;

-- Grant permissions
GRANT ALL ON SCHEMA test_schema TO testuser;
GRANT ALL ON DATABASE go_doc_go_test TO testuser;

-- Create basic tables for testing (these mirror the main schema)
-- Documents table
CREATE TABLE IF NOT EXISTS documents (
    doc_id VARCHAR(255) PRIMARY KEY,
    doc_type VARCHAR(100),
    source TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Elements table
CREATE TABLE IF NOT EXISTS elements (
    element_pk SERIAL PRIMARY KEY,
    element_id VARCHAR(255) UNIQUE NOT NULL,
    doc_id VARCHAR(255) REFERENCES documents(doc_id) ON DELETE CASCADE,
    element_type VARCHAR(100),
    content_preview TEXT,
    document_position INTEGER,
    metadata JSONB,
    parent_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Relationships table
CREATE TABLE IF NOT EXISTS relationships (
    relationship_id SERIAL PRIMARY KEY,
    source_id VARCHAR(255),
    target_id VARCHAR(255),
    relationship_type VARCHAR(100),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, target_id, relationship_type)
);

-- Entities table
CREATE TABLE IF NOT EXISTS entities (
    entity_pk SERIAL PRIMARY KEY,
    entity_id VARCHAR(255) UNIQUE NOT NULL,
    entity_type VARCHAR(100),
    name TEXT,
    attributes JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Work queue tables for distributed processing
CREATE TABLE IF NOT EXISTS processing_runs (
    run_id VARCHAR(16) PRIMARY KEY,
    config JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    leader_worker_id VARCHAR(100),
    leader_elected_at TIMESTAMP,
    leader_heartbeat TIMESTAMP,
    leader_lease_expires TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS document_queue (
    queue_id SERIAL PRIMARY KEY,
    doc_id VARCHAR(255) NOT NULL,
    source_name VARCHAR(255) NOT NULL,
    run_id VARCHAR(16) NOT NULL REFERENCES processing_runs(run_id),
    status VARCHAR(20) DEFAULT 'pending',
    claimed_by VARCHAR(100),
    claimed_at TIMESTAMP,
    heartbeat TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_elements_doc_id ON elements(doc_id);
CREATE INDEX IF NOT EXISTS idx_elements_parent_id ON elements(parent_id);
CREATE INDEX IF NOT EXISTS idx_elements_type ON elements(element_type);
CREATE INDEX IF NOT EXISTS idx_relationships_source ON relationships(source_id);
CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships(target_id);
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_entities_name ON entities(name);
CREATE INDEX IF NOT EXISTS idx_queue_status ON document_queue(status, run_id);
CREATE INDEX IF NOT EXISTS idx_queue_claimed ON document_queue(claimed_by, status);

-- Create the atomic leader election function
CREATE OR REPLACE FUNCTION attempt_leader_election(
    p_run_id VARCHAR(16),
    p_worker_id VARCHAR(100),
    p_lease_duration_seconds INTEGER DEFAULT 60
)
RETURNS BOOLEAN AS $$
DECLARE
    v_now TIMESTAMP := CURRENT_TIMESTAMP;
    v_lease_expires TIMESTAMP := v_now + (p_lease_duration_seconds || ' seconds')::INTERVAL;
    v_updated INTEGER;
BEGIN
    UPDATE processing_runs
    SET 
        leader_worker_id = p_worker_id,
        leader_elected_at = v_now,
        leader_heartbeat = v_now,
        leader_lease_expires = v_lease_expires,
        updated_at = v_now
    WHERE 
        run_id = p_run_id
        AND (
            leader_worker_id IS NULL
            OR leader_lease_expires < v_now
        );
    
    GET DIAGNOSTICS v_updated = ROW_COUNT;
    RETURN v_updated > 0;
END;
$$ LANGUAGE plpgsql;

-- Add some test data
INSERT INTO documents (doc_id, doc_type, source, metadata) 
VALUES 
    ('test-doc-1', 'pdf', 'test.pdf', '{"pages": 10}'),
    ('test-doc-2', 'docx', 'test.docx', '{"author": "Test User"}')
ON CONFLICT (doc_id) DO NOTHING;

-- Print confirmation
DO $$
BEGIN
    RAISE NOTICE 'Test database initialization complete';
END $$;