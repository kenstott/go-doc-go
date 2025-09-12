"""
Pytest configuration and fixtures for database tests.
"""

import os
import sys
import sqlite3
import tempfile
import subprocess
import logging
import time
import pytest
from typing import Dict, Any, Generator, Optional, List
from unittest.mock import MagicMock
from pathlib import Path

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'src'))

# Mock the server module to prevent database initialization
sys.modules['go_doc_go.server'] = MagicMock()

# Set dummy environment variable
os.environ['DOCUMENTS_URI'] = 'file://./test_storage'

logger = logging.getLogger(__name__)

# Try to import SQLAlchemy
try:
    import sqlalchemy
    from sqlalchemy import text, create_engine
    SQLALCHEMY_AVAILABLE = True
except ImportError:
    sqlalchemy = None
    SQLALCHEMY_AVAILABLE = False

# Try to import psycopg2 for PostgreSQL
try:
    import psycopg2
    import psycopg2.extras
    PSYCOPG2_AVAILABLE = True
except ImportError:
    psycopg2 = None
    PSYCOPG2_AVAILABLE = False


def wait_for_postgres(host: str = "localhost", port: int = 5432, timeout: int = 60) -> bool:
    """
    Wait for PostgreSQL to be ready.
    
    Args:
        host: PostgreSQL host
        port: PostgreSQL port
        timeout: Maximum wait time in seconds
        
    Returns:
        True if PostgreSQL is ready, False if timeout
    """
    import socket
    
    start_time = time.time()
    while time.time() - start_time < timeout:
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(1)
            result = sock.connect_ex((host, port))
            sock.close()
            
            if result == 0:
                # Port is open, try to connect with psycopg2
                if PSYCOPG2_AVAILABLE:
                    try:
                        conn = psycopg2.connect(
                            host=host,
                            port=port,
                            database="testdb",
                            user="testuser",
                            password="testpass",
                            connect_timeout=5
                        )
                        conn.close()
                        return True
                    except Exception:
                        pass
        except Exception:
            pass
        
        time.sleep(1)
    
    return False


@pytest.fixture(scope="session")
def docker_compose_up():
    """
    Start Docker Compose services for database testing.
    """
    compose_file = os.path.join(
        os.path.dirname(__file__), '..', '..', 'test_containers', 'postgres', 'compose.yaml'
    )
    
    # Check if Docker is available
    try:
        subprocess.run(["docker", "--version"], check=True, capture_output=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        pytest.skip("Docker is not available")
    
    # Check if docker-compose file exists
    if not os.path.exists(compose_file):
        pytest.skip(f"Docker Compose file not found: {compose_file}")
    
    # Check if PostgreSQL is already running
    already_running = False
    try:
        result = subprocess.run(
            ["docker", "ps", "--filter", "name=doculyzer-database-tests-postgres", "--format", "{{.Names}}"],
            capture_output=True,
            text=True,
            check=True
        )
        if "doculyzer-database-tests-postgres" in result.stdout:
            already_running = True
            print("PostgreSQL container already running, using existing container")
    except subprocess.CalledProcessError:
        pass
    
    # Start services if not already running
    if not already_running:
        try:
            # Try docker compose (new syntax) first
            subprocess.run(
                ["docker", "compose", "-f", compose_file, "up", "-d"],
                check=True,
                capture_output=True,
                text=True
            )
        except subprocess.CalledProcessError:
            # Fall back to docker-compose (old syntax)
            subprocess.run(
                ["docker-compose", "-f", compose_file, "up", "-d"],
                check=True,
                capture_output=True,
                text=True
            )
    
    # Wait for PostgreSQL to be ready
    if not wait_for_postgres():
        pytest.skip("PostgreSQL did not start in time")
    
    yield
    
    # Only stop services if we started them
    if not already_running:
        try:
            # Try docker compose (new syntax) first
            subprocess.run(
                ["docker", "compose", "-f", compose_file, "down", "-v"],
                capture_output=True,
                text=True,
                check=True
            )
        except subprocess.CalledProcessError:
            # Fall back to docker-compose (old syntax)
            subprocess.run(
                ["docker-compose", "-f", compose_file, "down", "-v"],
                capture_output=True,
                text=True
            )


@pytest.fixture
def temp_sqlite_db() -> Generator[str, None, None]:
    """
    Create a temporary SQLite database for testing.
    """
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as f:
        db_path = f.name
    
    # Initialize the database
    conn = sqlite3.connect(db_path)
    conn.close()
    
    yield db_path
    
    # Cleanup
    try:
        os.unlink(db_path)
    except Exception:
        pass


@pytest.fixture
def sqlite_test_data(temp_sqlite_db):
    """
    Set up SQLite database with test data.
    """
    conn = sqlite3.connect(temp_sqlite_db)
    cursor = conn.cursor()
    
    # Create test tables
    cursor.execute("""
        CREATE TABLE documents (
            id INTEGER PRIMARY KEY,
            title TEXT NOT NULL,
            content TEXT,
            doc_type TEXT DEFAULT 'text',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            metadata TEXT
        )
    """)
    
    cursor.execute("""
        CREATE TABLE binary_docs (
            id INTEGER PRIMARY KEY,
            filename TEXT NOT NULL,
            content BLOB,
            content_type TEXT,
            size INTEGER,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    
    cursor.execute("""
        CREATE TABLE json_records (
            id INTEGER PRIMARY KEY,
            name TEXT,
            description TEXT,
            data_field TEXT,
            status TEXT,
            config TEXT,
            tags TEXT,
            timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    
    # Insert test data with current timestamps
    from datetime import datetime, timedelta
    current_time = datetime.now()
    test_docs = [
        (1, "Sample Document 1", "This is the content of document 1.", "markdown", (current_time - timedelta(minutes=30)).strftime("%Y-%m-%d %H:%M:%S"), '{"author": "Test Author"}'),
        (2, "Sample Document 2", "# Header\n\nThis is a markdown document with headers.", "markdown", (current_time - timedelta(minutes=25)).strftime("%Y-%m-%d %H:%M:%S"), '{"author": "Another Author", "tags": ["test", "sample"]}'),
        (3, "Plain Text Doc", "Simple plain text content without any formatting.", "text", (current_time - timedelta(minutes=20)).strftime("%Y-%m-%d %H:%M:%S"), '{"source": "test"}'),
        (4, "JSON Document", '{"key": "value", "number": 42, "array": [1, 2, 3]}', "json", (current_time - timedelta(minutes=15)).strftime("%Y-%m-%d %H:%M:%S"), '{"type": "structured"}'),
        (5, "CSV Data", "Name,Age,City\nJohn,30,NYC\nJane,25,LA\nBob,35,Chicago", "csv", (current_time - timedelta(minutes=10)).strftime("%Y-%m-%d %H:%M:%S"), '{"format": "csv"}')
    ]
    
    cursor.executemany("""
        INSERT INTO documents (id, title, content, doc_type, created_at, metadata)
        VALUES (?, ?, ?, ?, ?, ?)
    """, test_docs)
    
    # Insert binary test data
    binary_data = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10\x00\x00\x00\x10'
    cursor.execute("""
        INSERT INTO binary_docs (id, filename, content, content_type, size, created_at)
        VALUES (?, ?, ?, ?, ?, ?)
    """, (1, "test.png", binary_data, "image/png", len(binary_data), (current_time - timedelta(minutes=5)).strftime("%Y-%m-%d %H:%M:%S")))
    
    # Insert JSON records test data
    json_records = [
        (1, "Record 1", "First test record", "data_value_1", "active", '{"setting1": true}', "tag1,tag2", (current_time - timedelta(minutes=8)).strftime("%Y-%m-%d %H:%M:%S")),
        (2, "Record 2", "Second test record", "data_value_2", "inactive", '{"setting2": false}', "tag2,tag3", (current_time - timedelta(minutes=6)).strftime("%Y-%m-%d %H:%M:%S")),
        (3, "Record 3", "Third test record", "data_value_3", "pending", '{"setting3": null}', "tag1,tag3", (current_time - timedelta(minutes=4)).strftime("%Y-%m-%d %H:%M:%S"))
    ]
    
    cursor.executemany("""
        INSERT INTO json_records (id, name, description, data_field, status, config, tags, timestamp)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    """, json_records)
    
    conn.commit()
    conn.close()
    
    return temp_sqlite_db


@pytest.fixture(scope="session")
def postgres_config() -> Dict[str, Any]:
    """
    Provide PostgreSQL configuration for tests.
    """
    return {
        "host": "localhost",
        "port": 5432,
        "database": "testdb",
        "user": "testuser",
        "password": "testpass",
        "connection_string": "postgresql://testuser:testpass@localhost:5432/testdb"
    }


@pytest.fixture
def postgres_connection(postgres_config):
    """
    Create a PostgreSQL connection for testing.
    Skip docker_compose_up to avoid hanging.
    """
    if not PSYCOPG2_AVAILABLE:
        pytest.skip("psycopg2 is not available")
    
    # Check if PostgreSQL is accessible
    import socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(1)
    result = sock.connect_ex((postgres_config["host"], postgres_config["port"]))
    sock.close()
    
    if result != 0:
        pytest.skip("PostgreSQL is not accessible")
    
    try:
        conn = psycopg2.connect(
            host=postgres_config["host"],
            port=postgres_config["port"],
            database=postgres_config["database"],
            user=postgres_config["user"],
            password=postgres_config["password"],
            connect_timeout=5
        )
        conn.autocommit = True
        
        yield conn
        
        conn.close()
    except Exception as e:
        pytest.skip(f"Could not connect to PostgreSQL: {e}")


@pytest.fixture
def postgres_test_data(postgres_connection, postgres_config):
    """
    Set up PostgreSQL database with test data.
    """
    cursor = postgres_connection.cursor()
    
    try:
        
        # Drop tables if they exist (for cleanup)
        cursor.execute("DROP TABLE IF EXISTS documents CASCADE")
        cursor.execute("DROP TABLE IF EXISTS binary_docs CASCADE")
        cursor.execute("DROP TABLE IF EXISTS json_records CASCADE")
        
        # Create test tables
        cursor.execute("""
            CREATE TABLE documents (
                id SERIAL PRIMARY KEY,
                title VARCHAR(255) NOT NULL,
                content TEXT,
                doc_type VARCHAR(50) DEFAULT 'text',
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                metadata JSONB
            )
        """)
        
        cursor.execute("""
            CREATE TABLE binary_docs (
                id SERIAL PRIMARY KEY,
                filename VARCHAR(255) NOT NULL,
                content BYTEA,
                content_type VARCHAR(100),
                size INTEGER,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)
        
        cursor.execute("""
            CREATE TABLE json_records (
                id SERIAL PRIMARY KEY,
                name VARCHAR(255),
                description TEXT,
                data_field TEXT,
                status VARCHAR(50),
                config JSONB,
                tags TEXT[],
                timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)
        
        # Insert test data with current timestamps
        from datetime import datetime, timedelta
        current_time = datetime.now()
        test_docs = [
            (1, "Sample Document 1", "This is the content of document 1.", "markdown", (current_time - timedelta(minutes=30)).strftime("%Y-%m-%d %H:%M:%S"), '{"author": "Test Author"}'),
            (2, "Sample Document 2", "# Header\n\nThis is a markdown document with headers.", "markdown", (current_time - timedelta(minutes=25)).strftime("%Y-%m-%d %H:%M:%S"), '{"author": "Another Author", "tags": ["test", "sample"]}'),
            (3, "Plain Text Doc", "Simple plain text content without any formatting.", "text", (current_time - timedelta(minutes=20)).strftime("%Y-%m-%d %H:%M:%S"), '{"source": "test"}'),
            (4, "JSON Document", '{"key": "value", "number": 42, "array": [1, 2, 3]}', "json", (current_time - timedelta(minutes=15)).strftime("%Y-%m-%d %H:%M:%S"), '{"type": "structured"}'),
            (5, "CSV Data", "Name,Age,City\nJohn,30,NYC\nJane,25,LA\nBob,35,Chicago", "csv", (current_time - timedelta(minutes=10)).strftime("%Y-%m-%d %H:%M:%S"), '{"format": "csv"}')
        ]
        
        for doc in test_docs:
            cursor.execute("""
                INSERT INTO documents (id, title, content, doc_type, created_at, metadata)
                VALUES (%s, %s, %s, %s, %s, %s::jsonb)
            """, doc)
        
        # Insert binary test data
        binary_data = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10\x00\x00\x00\x10'
        cursor.execute("""
            INSERT INTO binary_docs (id, filename, content, content_type, size, created_at)
            VALUES (%s, %s, %s, %s, %s, %s)
        """, (1, "test.png", binary_data, "image/png", len(binary_data), (current_time - timedelta(minutes=5)).strftime("%Y-%m-%d %H:%M:%S")))
        
        # Insert JSON records test data
        json_records = [
            (1, "Record 1", "First test record", "data_value_1", "active", '{"setting1": true}', ["tag1", "tag2"], (current_time - timedelta(minutes=8)).strftime("%Y-%m-%d %H:%M:%S")),
            (2, "Record 2", "Second test record", "data_value_2", "inactive", '{"setting2": false}', ["tag2", "tag3"], (current_time - timedelta(minutes=6)).strftime("%Y-%m-%d %H:%M:%S")),
            (3, "Record 3", "Third test record", "data_value_3", "pending", '{"setting3": null}', ["tag1", "tag3"], (current_time - timedelta(minutes=4)).strftime("%Y-%m-%d %H:%M:%S"))
        ]
        
        for record in json_records:
            cursor.execute("""
                INSERT INTO json_records (id, name, description, data_field, status, config, tags, timestamp)
                VALUES (%s, %s, %s, %s, %s, %s::jsonb, %s, %s)
            """, record)
        
        postgres_connection.commit()
        
    except Exception as e:
        logger.error(f"Error setting up PostgreSQL test data: {e}")
        raise
    
    yield postgres_config["connection_string"]
    
    # Cleanup
    try:
        cursor.execute("DROP TABLE IF EXISTS documents CASCADE")
        cursor.execute("DROP TABLE IF EXISTS binary_docs CASCADE")
        cursor.execute("DROP TABLE IF EXISTS json_records CASCADE")
        postgres_connection.commit()
    except Exception as e:
        logger.warning(f"Error cleaning up PostgreSQL test data: {e}")


@pytest.fixture
def sqlite_engine(sqlite_test_data):
    """
    Create SQLAlchemy engine for SQLite testing.
    """
    if not SQLALCHEMY_AVAILABLE:
        pytest.skip("SQLAlchemy is not available")
    
    engine = create_engine(f'sqlite:///{sqlite_test_data}')
    yield engine
    engine.dispose()


@pytest.fixture
def postgres_engine(postgres_test_data):
    """
    Create SQLAlchemy engine for PostgreSQL testing.
    """
    if not SQLALCHEMY_AVAILABLE:
        pytest.skip("SQLAlchemy is not available")
    
    engine = create_engine(postgres_test_data)
    yield engine
    engine.dispose()


@pytest.fixture
def sample_database_configs():
    """
    Provide sample database configurations for different scenarios.
    """
    return {
        "sqlite_blob": {
            "name": "test-sqlite-source",
            "query": "documents",
            "id_column": "id",
            "content_column": "content",
            "metadata_columns": ["title", "doc_type", "metadata"],
            "timestamp_column": "created_at"
        },
        "sqlite_json": {
            "name": "test-sqlite-json-source",
            "query": "json_records",
            "id_column": "id",
            "json_mode": True,
            "json_columns": ["name", "description", "data_field", "status", "config", "tags"],
            "metadata_columns": ["name", "status"],
            "timestamp_column": "timestamp"
        },
        "postgres_blob": {
            "name": "test-postgres-source",
            "query": "documents",
            "id_column": "id",
            "content_column": "content",
            "metadata_columns": ["title", "doc_type", "metadata"],
            "timestamp_column": "created_at"
        },
        "postgres_json": {
            "name": "test-postgres-json-source",
            "query": "json_records",
            "id_column": "id",
            "json_mode": True,
            "json_columns": ["name", "description", "data_field", "status", "config", "tags"],
            "metadata_columns": ["name", "status"],
            "timestamp_column": "timestamp"
        }
    }


# Skip markers for missing dependencies
requires_sqlalchemy = pytest.mark.skipif(
    not SQLALCHEMY_AVAILABLE,
    reason="SQLAlchemy is not installed"
)

requires_psycopg2 = pytest.mark.skipif(
    not PSYCOPG2_AVAILABLE,
    reason="psycopg2 is not installed"
)

requires_docker = pytest.mark.skipif(
    subprocess.run(
        ["docker", "--version"],
        capture_output=True
    ).returncode != 0,
    reason="Docker is not available"
)