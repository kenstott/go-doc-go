-- PostgreSQL initialization script for database tests
-- This script sets up the database with proper permissions and extensions

-- Ensure we're connected to the correct database
\c testdb;

-- Create extensions if available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Grant necessary permissions to test user
GRANT ALL PRIVILEGES ON DATABASE testdb TO testuser;
GRANT ALL PRIVILEGES ON SCHEMA public TO testuser;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO testuser;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO testuser;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO testuser;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO testuser;

-- Create a test function that can be used in tests
CREATE OR REPLACE FUNCTION test_timestamp()
RETURNS TIMESTAMP AS $$
BEGIN
    RETURN CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;