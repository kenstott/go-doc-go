"""
Pytest configuration and fixtures for adapter tests.
"""

import os
import sys
import time
import pytest
import tempfile
import subprocess
import logging
from typing import Dict, Any, Generator, Optional, List
from unittest.mock import MagicMock

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'src'))

# Mock the server module to prevent database initialization
sys.modules['go_doc_go.server'] = MagicMock()

# Set dummy environment variable
os.environ['DOCUMENTS_URI'] = 'file://./test_storage'

logger = logging.getLogger(__name__)

# Try to import boto3 and minio
try:
    import boto3
    from botocore.exceptions import ClientError
    BOTO3_AVAILABLE = True
except ImportError:
    boto3 = None
    ClientError = None
    BOTO3_AVAILABLE = False

try:
    from minio import Minio
    from minio.error import S3Error
    MINIO_AVAILABLE = True
except ImportError:
    Minio = None
    S3Error = None
    MINIO_AVAILABLE = False

# Try to import pymongo
try:
    import pymongo
    from pymongo import MongoClient
    from bson import ObjectId
    PYMONGO_AVAILABLE = True
except ImportError:
    pymongo = None
    MongoClient = None
    ObjectId = None
    PYMONGO_AVAILABLE = False


def wait_for_minio(host: str = "localhost", port: int = 9000, timeout: int = 60) -> bool:
    """
    Wait for Minio to be ready.
    
    Args:
        host: Minio host
        port: Minio port
        timeout: Maximum wait time in seconds
        
    Returns:
        True if Minio is ready, False if timeout
    """
    import socket
    import time
    
    start_time = time.time()
    while time.time() - start_time < timeout:
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(1)
            result = sock.connect_ex((host, port))
            sock.close()
            
            if result == 0:
                # Port is open, try to connect with boto3
                if BOTO3_AVAILABLE:
                    try:
                        client = boto3.client(
                            's3',
                            endpoint_url=f"http://{host}:{port}",
                            aws_access_key_id='minioadmin',
                            aws_secret_access_key='minioadmin',
                            region_name='us-east-1'
                        )
                        client.list_buckets()
                        return True
                    except Exception:
                        pass
                elif MINIO_AVAILABLE:
                    try:
                        client = Minio(
                            f"{host}:{port}",
                            access_key="minioadmin",
                            secret_key="minioadmin",
                            secure=False
                        )
                        client.list_buckets()
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
    Start Docker Compose services for testing.
    """
    compose_file = os.path.join(
        os.path.dirname(__file__), '..', '..', 'docker-compose.test.yml'
    )
    
    # Check if Docker is available
    try:
        subprocess.run(["docker", "--version"], check=True, capture_output=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        pytest.skip("Docker is not available")
    
    # Check if docker-compose file exists
    if not os.path.exists(compose_file):
        pytest.skip(f"Docker Compose file not found: {compose_file}")
    
    # Start services
    try:
        subprocess.run(
            ["docker-compose", "-f", compose_file, "up", "-d"],
            check=True,
            capture_output=True,
            text=True
        )
        
        # Wait for Minio to be ready
        if not wait_for_minio():
            pytest.skip("Minio did not start in time")
        
        yield
        
    finally:
        # Stop services
        subprocess.run(
            ["docker-compose", "-f", compose_file, "down", "-v"],
            capture_output=True,
            text=True
        )


@pytest.fixture(scope="session")
def minio_config() -> Dict[str, Any]:
    """
    Provide Minio configuration for tests.
    """
    return {
        "endpoint_url": "http://localhost:9000",
        "aws_access_key_id": "minioadmin",
        "aws_secret_access_key": "minioadmin",
        "region_name": "us-east-1",
        "bucket_name": "test-bucket"
    }


@pytest.fixture
def s3_client(docker_compose_up, minio_config):
    """
    Create an S3 client configured for Minio.
    """
    if not BOTO3_AVAILABLE:
        pytest.skip("boto3 is not available")
    
    client = boto3.client(
        's3',
        endpoint_url=minio_config["endpoint_url"],
        aws_access_key_id=minio_config["aws_access_key_id"],
        aws_secret_access_key=minio_config["aws_secret_access_key"],
        region_name=minio_config["region_name"]
    )
    
    # Ensure test bucket exists
    try:
        client.create_bucket(Bucket=minio_config["bucket_name"])
    except ClientError as e:
        if e.response['Error']['Code'] != 'BucketAlreadyOwnedByYou':
            raise
    
    yield client
    
    # Cleanup: Delete all objects in test bucket
    try:
        response = client.list_objects_v2(Bucket=minio_config["bucket_name"])
        if 'Contents' in response:
            objects = [{'Key': obj['Key']} for obj in response['Contents']]
            if objects:
                client.delete_objects(
                    Bucket=minio_config["bucket_name"],
                    Delete={'Objects': objects}
                )
    except Exception as e:
        logger.warning(f"Error cleaning up test bucket: {e}")


@pytest.fixture
def minio_client(docker_compose_up, minio_config):
    """
    Create a Minio client for testing.
    """
    if not MINIO_AVAILABLE:
        pytest.skip("minio package is not available")
    
    client = Minio(
        "localhost:9000",
        access_key=minio_config["aws_access_key_id"],
        secret_key=minio_config["aws_secret_access_key"],
        secure=False
    )
    
    # Ensure test bucket exists
    bucket_name = minio_config["bucket_name"]
    if not client.bucket_exists(bucket_name):
        client.make_bucket(bucket_name)
    
    yield client
    
    # Cleanup: Remove all objects from test bucket
    try:
        objects = client.list_objects(bucket_name, recursive=True)
        for obj in objects:
            client.remove_object(bucket_name, obj.object_name)
    except Exception as e:
        logger.warning(f"Error cleaning up test bucket: {e}")


@pytest.fixture
def sample_text_content() -> str:
    """
    Provide sample text content for testing.
    """
    return """# Sample Document

This is a sample document for testing.

## Section 1
This section contains some text content.

## Section 2
This section has a [link](./another-document.md) to another document.
"""


@pytest.fixture
def sample_json_content() -> str:
    """
    Provide sample JSON content for testing.
    """
    return """{
    "title": "Sample JSON Document",
    "type": "test",
    "data": {
        "field1": "value1",
        "field2": 123,
        "nested": {
            "subfield": "subvalue"
        }
    },
    "tags": ["test", "sample", "json"]
}"""


@pytest.fixture
def sample_csv_content() -> str:
    """
    Provide sample CSV content for testing.
    """
    return """Name,Age,Department,Salary
John Doe,30,Engineering,75000
Jane Smith,28,Marketing,65000
Bob Johnson,35,Sales,70000
Alice Williams,32,Engineering,80000
Charlie Brown,29,HR,60000"""


@pytest.fixture
def temp_file() -> Generator[str, None, None]:
    """
    Create a temporary file for testing.
    """
    with tempfile.NamedTemporaryFile(mode='w', delete=False) as f:
        temp_path = f.name
    
    yield temp_path
    
    # Cleanup
    try:
        os.unlink(temp_path)
    except Exception:
        pass


@pytest.fixture
def temp_binary_file() -> Generator[str, None, None]:
    """
    Create a temporary binary file for testing.
    """
    with tempfile.NamedTemporaryFile(mode='wb', delete=False) as f:
        # Write some binary content
        f.write(b'\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10')
        temp_path = f.name
    
    yield temp_path
    
    # Cleanup
    try:
        os.unlink(temp_path)
    except Exception:
        pass


@pytest.fixture
def upload_test_files(s3_client, minio_config):
    """
    Upload test files to S3/Minio.
    """
    bucket_name = minio_config["bucket_name"]
    uploaded_keys = []
    
    def upload(key: str, content: str | bytes, content_type: str = "text/plain"):
        """
        Upload a file to S3/Minio.
        
        Args:
            key: Object key
            content: File content
            content_type: MIME type
        """
        if isinstance(content, str):
            content = content.encode('utf-8')
        
        s3_client.put_object(
            Bucket=bucket_name,
            Key=key,
            Body=content,
            ContentType=content_type
        )
        uploaded_keys.append(key)
        return f"s3://{bucket_name}/{key}"
    
    yield upload
    
    # Cleanup
    if uploaded_keys:
        try:
            s3_client.delete_objects(
                Bucket=bucket_name,
                Delete={'Objects': [{'Key': key} for key in uploaded_keys]}
            )
        except Exception as e:
            logger.warning(f"Error cleaning up uploaded files: {e}")


def wait_for_mongodb(host: str = "localhost", port: int = 27017, timeout: int = 60) -> bool:
    """
    Wait for MongoDB to be ready.
    
    Args:
        host: MongoDB host
        port: MongoDB port
        timeout: Maximum wait time in seconds
        
    Returns:
        True if MongoDB is ready, False if timeout
    """
    import socket
    import time
    
    start_time = time.time()
    while time.time() - start_time < timeout:
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(1)
            result = sock.connect_ex((host, port))
            sock.close()
            
            if result == 0:
                # Port is open, try to connect with pymongo
                if PYMONGO_AVAILABLE:
                    try:
                        client = MongoClient(
                            f"mongodb://admin:admin123@{host}:{port}/",
                            serverSelectionTimeoutMS=5000
                        )
                        client.admin.command('ping')
                        client.close()
                        return True
                    except Exception:
                        pass
        except Exception:
            pass
        
        time.sleep(1)
    
    return False


@pytest.fixture(scope="session")
def mongodb_config() -> Dict[str, Any]:
    """
    Provide MongoDB configuration for tests.
    """
    return {
        "connection_string": "mongodb://admin:admin123@localhost:27017/",
        "database_name": "test_db",
        "collection_name": "test_collection",
        "username": "admin",
        "password": "admin123",
        "host": "localhost",
        "port": 27017
    }


@pytest.fixture
def mongodb_client(docker_compose_up, mongodb_config):
    """
    Create a MongoDB client for testing.
    """
    if not PYMONGO_AVAILABLE:
        pytest.skip("pymongo is not available")
    
    # Wait for MongoDB to be ready
    if not wait_for_mongodb():
        pytest.skip("MongoDB did not start in time")
    
    client = MongoClient(
        mongodb_config["connection_string"],
        serverSelectionTimeoutMS=5000
    )
    
    # Ensure test database exists
    db = client[mongodb_config["database_name"]]
    
    yield client
    
    # Cleanup: Drop test collections created during tests
    try:
        for collection_name in db.list_collection_names():
            if collection_name.startswith("test_"):
                db[collection_name].drop()
    except Exception as e:
        logger.warning(f"Error cleaning up MongoDB collections: {e}")
    
    client.close()


@pytest.fixture
def mongodb_collection(mongodb_client, mongodb_config):
    """
    Get a MongoDB collection for testing.
    """
    db = mongodb_client[mongodb_config["database_name"]]
    collection = db[mongodb_config["collection_name"]]
    
    # Clear collection before test
    collection.delete_many({})
    
    yield collection
    
    # Cleanup after test
    collection.delete_many({})


@pytest.fixture
def insert_test_documents(mongodb_collection):
    """
    Helper to insert test documents into MongoDB.
    """
    inserted_ids = []
    
    def insert(documents: list | dict) -> list:
        """
        Insert documents into the test collection.
        
        Args:
            documents: Single document or list of documents
            
        Returns:
            List of inserted document IDs
        """
        if isinstance(documents, dict):
            documents = [documents]
        
        result = mongodb_collection.insert_many(documents)
        inserted_ids.extend(result.inserted_ids)
        return result.inserted_ids
    
    yield insert
    
    # Cleanup: Remove inserted documents
    if inserted_ids:
        try:
            mongodb_collection.delete_many({"_id": {"$in": inserted_ids}})
        except Exception as e:
            logger.warning(f"Error cleaning up test documents: {e}")


@pytest.fixture
def sample_mongodb_documents() -> List[Dict[str, Any]]:
    """
    Provide sample MongoDB documents for testing.
    """
    return [
        {
            "name": "Document 1",
            "type": "article",
            "content": "This is the content of document 1",
            "tags": ["test", "sample"],
            "metadata": {
                "author": "Test Author 1",
                "created_at": "2024-01-01T00:00:00Z",
                "version": 1
            }
        },
        {
            "name": "Document 2",
            "type": "report",
            "content": "This is a report document with more complex structure",
            "sections": [
                {"title": "Introduction", "text": "Introduction text"},
                {"title": "Analysis", "text": "Analysis text"},
                {"title": "Conclusion", "text": "Conclusion text"}
            ],
            "references": ["doc1", "doc3"],
            "metadata": {
                "author": "Test Author 2",
                "created_at": "2024-01-02T00:00:00Z",
                "version": 2
            }
        },
        {
            "name": "Document 3",
            "type": "data",
            "values": [1, 2, 3, 4, 5],
            "nested": {
                "level1": {
                    "level2": {
                        "level3": "deeply nested value"
                    }
                }
            },
            "metadata": {
                "source": "test_system",
                "timestamp": 1704067200
            }
        }
    ]


# Skip markers for missing dependencies
requires_boto3 = pytest.mark.skipif(
    not BOTO3_AVAILABLE,
    reason="boto3 is not installed"
)

requires_minio = pytest.mark.skipif(
    not MINIO_AVAILABLE,
    reason="minio package is not installed"
)

requires_pymongo = pytest.mark.skipif(
    not PYMONGO_AVAILABLE,
    reason="pymongo is not installed"
)

requires_docker = pytest.mark.skipif(
    subprocess.run(
        ["docker", "--version"],
        capture_output=True
    ).returncode != 0,
    reason="Docker is not available"
)