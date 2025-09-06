#!/usr/bin/env python3

import psycopg2
from datetime import datetime, timedelta

def setup_postgres_test_data():
    """Setup PostgreSQL test data"""
    postgres_config = {
        "host": "localhost",
        "port": 5432,
        "database": "testdb", 
        "user": "testuser",
        "password": "testpass"
    }
    
    print("Connecting to PostgreSQL...")
    conn = psycopg2.connect(
        host=postgres_config["host"],
        port=postgres_config["port"],
        database=postgres_config["database"],
        user=postgres_config["user"],
        password=postgres_config["password"]
    )
    conn.autocommit = True
    cursor = conn.cursor()
    
    print("Connected successfully!")
    
    try:
        # Drop tables if they exist (for cleanup)
        print("Dropping existing tables...")
        cursor.execute("DROP TABLE IF EXISTS documents CASCADE")
        cursor.execute("DROP TABLE IF EXISTS binary_docs CASCADE")
        cursor.execute("DROP TABLE IF EXISTS json_records CASCADE")
        
        # Create test tables
        print("Creating tables...")
        cursor.execute("""
            CREATE TABLE documents (
                id SERIAL PRIMARY KEY,
                title TEXT,
                content TEXT,
                doc_type TEXT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                metadata JSONB
            )
        """)
        
        cursor.execute("""
            CREATE TABLE binary_docs (
                id SERIAL PRIMARY KEY,
                filename TEXT,
                content BYTEA,
                content_type TEXT,
                size INTEGER,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)
        
        cursor.execute("""
            CREATE TABLE json_records (
                id SERIAL PRIMARY KEY,
                name TEXT,
                description TEXT,
                data_field TEXT,
                status TEXT,
                config JSONB,
                tags TEXT[],
                timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)
        
        print("Tables created successfully!")
        
        # Insert test data with current timestamps
        print("Inserting test data...")
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
                ON CONFLICT (id) DO UPDATE
                SET title = EXCLUDED.title,
                    content = EXCLUDED.content,
                    doc_type = EXCLUDED.doc_type,
                    created_at = EXCLUDED.created_at,
                    metadata = EXCLUDED.metadata
            """, doc)
        
        # Insert binary test data
        binary_data = b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10\x00\x00\x00\x10'
        cursor.execute("""
            INSERT INTO binary_docs (id, filename, content, content_type, size, created_at)
            VALUES (%s, %s, %s, %s, %s, %s)
            ON CONFLICT (id) DO UPDATE
            SET filename = EXCLUDED.filename,
                content = EXCLUDED.content,
                content_type = EXCLUDED.content_type,
                size = EXCLUDED.size,
                created_at = EXCLUDED.created_at
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
                ON CONFLICT (id) DO UPDATE
                SET name = EXCLUDED.name,
                    description = EXCLUDED.description,
                    data_field = EXCLUDED.data_field,
                    status = EXCLUDED.status,
                    config = EXCLUDED.config,
                    tags = EXCLUDED.tags,
                    timestamp = EXCLUDED.timestamp
            """, record)
        
        print("Documents inserted successfully!")
        
        # Verify data
        cursor.execute("SELECT COUNT(*) FROM documents")
        count = cursor.fetchone()
        print(f"Document count: {count[0]}")
        
        cursor.execute("SELECT COUNT(*) FROM binary_docs")
        count = cursor.fetchone()
        print(f"Binary docs count: {count[0]}")
        
        cursor.execute("SELECT COUNT(*) FROM json_records")
        count = cursor.fetchone()
        print(f"JSON records count: {count[0]}")
        
        print("PostgreSQL test data setup completed successfully!")
        
    except Exception as e:
        print(f"Error setting up PostgreSQL: {e}")
        raise
    finally:
        cursor.close()
        conn.close()

if __name__ == "__main__":
    setup_postgres_test_data()