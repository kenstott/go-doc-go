#!/usr/bin/env python3
"""
Google Drive Integration Test Runner
"""

import os
import sys
import re
from pathlib import Path

# Add src to path
sys.path.insert(0, 'src')

def extract_file_id(url_or_id):
    """Extract file ID from various Google Drive URL formats or return as-is if already an ID."""
    if not url_or_id:
        return None
    
    # If it doesn't look like a URL, assume it's already an ID
    if not url_or_id.startswith('http'):
        return url_or_id
    
    # Patterns to extract IDs from various Google URLs
    patterns = [
        r'/file/d/([a-zA-Z0-9_-]+)',  # Drive file URL
        r'/folders/([a-zA-Z0-9_-]+)',  # Drive folder URL
        r'/document/d/([a-zA-Z0-9_-]+)',  # Docs URL
        r'/spreadsheets/d/([a-zA-Z0-9_-]+)',  # Sheets URL
        r'/presentation/d/([a-zA-Z0-9_-]+)',  # Slides URL
        r'[?&]id=([a-zA-Z0-9_-]+)',  # Old style with ?id=
    ]
    
    for pattern in patterns:
        match = re.search(pattern, url_or_id)
        if match:
            return match.group(1)
    
    return url_or_id  # Return as-is if no pattern matches

def setup_environment():
    """Set up environment variables from .env.google_drive file."""
    env_file = Path('.env.google_drive')
    if not env_file.exists():
        print("❌ .env.google_drive file not found!")
        return False
    
    # Read and parse the env file
    with open(env_file) as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith('#'):
                if '=' in line:
                    key, value = line.split('=', 1)
                    # Expand ~ to home directory
                    if value.startswith('~'):
                        value = os.path.expanduser(value)
                    
                    # Extract IDs from URLs for test data
                    if key in ['GOOGLE_DRIVE_TEST_FILE_ID', 'GOOGLE_DRIVE_TEST_FOLDER_ID', 'GOOGLE_DRIVE_TEST_GOOGLE_DOC_ID']:
                        value = extract_file_id(value)
                    
                    os.environ[key] = value
                    print(f"✅ Set {key}")
    
    return True

def verify_credentials():
    """Verify that credentials are properly configured."""
    sa_file = os.environ.get('GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE')
    if sa_file:
        sa_file = os.path.expanduser(sa_file)
        if os.path.exists(sa_file):
            print(f"✅ Service account file found: {sa_file}")
            
            # Check if it's valid JSON
            import json
            try:
                with open(sa_file) as f:
                    data = json.load(f)
                print(f"   Project ID: {data.get('project_id')}")
                print(f"   Client Email: {data.get('client_email')}")
                return True
            except json.JSONDecodeError:
                print("❌ Service account file is not valid JSON!")
                return False
        else:
            print(f"❌ Service account file not found: {sa_file}")
            return False
    
    # Check for OAuth credentials
    creds_file = os.environ.get('GOOGLE_DRIVE_CREDENTIALS_PATH')
    if creds_file:
        creds_file = os.path.expanduser(creds_file)
        if os.path.exists(creds_file):
            print(f"✅ OAuth credentials file found: {creds_file}")
            return True
        else:
            print(f"❌ OAuth credentials file not found: {creds_file}")
    
    print("❌ No credentials configured!")
    return False

def test_connection():
    """Test basic connection to Google Drive."""
    print("\n🔍 Testing Google Drive Connection...")
    
    try:
        from go_doc_go.content_source.google_drive import GoogleDriveContentSource
        
        # Create configuration
        sa_file = os.environ.get('GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE')
        if sa_file:
            sa_file = os.path.expanduser(sa_file)
            config = {
                'auth_type': 'service_account',
                'service_account_file': sa_file,
                'scopes': ['https://www.googleapis.com/auth/drive.readonly'],
                'max_results': 5
            }
        else:
            config = {
                'auth_type': 'oauth',
                'credentials_path': os.path.expanduser(os.environ.get('GOOGLE_DRIVE_CREDENTIALS_PATH', '')),
                'token_path': os.path.expanduser(os.environ.get('GOOGLE_DRIVE_TOKEN_PATH', 'token.pickle')),
                'scopes': ['https://www.googleapis.com/auth/drive.readonly'],
                'max_results': 5
            }
        
        print("   Creating content source...")
        source = GoogleDriveContentSource(config)
        
        print("   Listing documents...")
        documents = list(source.get_documents())
        
        print(f"✅ Connection successful! Found {len(documents)} documents:")
        for doc in documents[:5]:
            name = doc.get('metadata', {}).get('name', 'Unknown')
            size = doc.get('metadata', {}).get('size', 0)
            print(f"   📄 {name} ({size} bytes)")
        
        return True
        
    except ImportError as e:
        print(f"❌ Import error: {e}")
        print("   Install required packages: pip install google-api-python-client google-auth-oauthlib google-auth-httplib2")
        return False
    except Exception as e:
        print(f"❌ Connection failed: {e}")
        import traceback
        traceback.print_exc()
        return False

def test_specific_file():
    """Test fetching a specific file."""
    file_id = os.environ.get('GOOGLE_DRIVE_TEST_FILE_ID')
    if not file_id:
        print("⚠️  No test file ID configured, skipping specific file test")
        return True
    
    print(f"\n🔍 Testing specific file fetch (ID: {file_id})...")
    
    try:
        from go_doc_go.content_source.google_drive import GoogleDriveContentSource
        
        sa_file = os.environ.get('GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE')
        if sa_file:
            sa_file = os.path.expanduser(sa_file)
            config = {
                'auth_type': 'service_account',
                'service_account_file': sa_file,
                'scopes': ['https://www.googleapis.com/auth/drive.readonly']
            }
        else:
            config = {
                'auth_type': 'oauth',
                'credentials_path': os.path.expanduser(os.environ.get('GOOGLE_DRIVE_CREDENTIALS_PATH', '')),
                'token_path': os.path.expanduser(os.environ.get('GOOGLE_DRIVE_TOKEN_PATH', 'token.pickle')),
                'scopes': ['https://www.googleapis.com/auth/drive.readonly']
            }
        
        source = GoogleDriveContentSource(config)
        document = source.fetch_document(file_id)
        
        print(f"✅ Successfully fetched document:")
        print(f"   Name: {document.get('metadata', {}).get('name', 'Unknown')}")
        print(f"   Size: {document.get('metadata', {}).get('size', 0)} bytes")
        print(f"   Type: {document.get('metadata', {}).get('mime_type', 'Unknown')}")
        
        if 'content' in document:
            print(f"   Content preview: {document['content'][:100]}...")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to fetch file: {e}")
        return False

def test_folder():
    """Test fetching files from a specific folder."""
    folder_id = os.environ.get('GOOGLE_DRIVE_TEST_FOLDER_ID')
    if not folder_id:
        print("⚠️  No test folder ID configured, skipping folder test")
        return True
    
    print(f"\n🔍 Testing folder listing (ID: {folder_id})...")
    
    try:
        from go_doc_go.content_source.google_drive import GoogleDriveContentSource
        
        sa_file = os.environ.get('GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE')
        if sa_file:
            sa_file = os.path.expanduser(sa_file)
            config = {
                'auth_type': 'service_account',
                'service_account_file': sa_file,
                'scopes': ['https://www.googleapis.com/auth/drive.readonly'],
                'folders': [folder_id],
                'max_results': 10
            }
        else:
            config = {
                'auth_type': 'oauth',
                'credentials_path': os.path.expanduser(os.environ.get('GOOGLE_DRIVE_CREDENTIALS_PATH', '')),
                'token_path': os.path.expanduser(os.environ.get('GOOGLE_DRIVE_TOKEN_PATH', 'token.pickle')),
                'scopes': ['https://www.googleapis.com/auth/drive.readonly'],
                'folders': [folder_id],
                'max_results': 10
            }
        
        source = GoogleDriveContentSource(config)
        documents = list(source.get_documents())
        
        print(f"✅ Found {len(documents)} documents in folder:")
        for doc in documents[:5]:
            name = doc.get('metadata', {}).get('name', 'Unknown')
            print(f"   📄 {name}")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to list folder: {e}")
        return False

def test_google_doc():
    """Test fetching and exporting a Google Doc."""
    doc_id = os.environ.get('GOOGLE_DRIVE_TEST_GOOGLE_DOC_ID')
    if not doc_id:
        print("⚠️  No Google Doc ID configured, skipping Google Doc test")
        return True
    
    print(f"\n🔍 Testing Google Doc export (ID: {doc_id})...")
    
    try:
        from go_doc_go.content_source.google_drive import GoogleDriveContentSource
        
        sa_file = os.environ.get('GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE')
        if sa_file:
            sa_file = os.path.expanduser(sa_file)
            config = {
                'auth_type': 'service_account',
                'service_account_file': sa_file,
                'scopes': ['https://www.googleapis.com/auth/drive.readonly']
            }
        else:
            config = {
                'auth_type': 'oauth',
                'credentials_path': os.path.expanduser(os.environ.get('GOOGLE_DRIVE_CREDENTIALS_PATH', '')),
                'token_path': os.path.expanduser(os.environ.get('GOOGLE_DRIVE_TOKEN_PATH', 'token.pickle')),
                'scopes': ['https://www.googleapis.com/auth/drive.readonly']
            }
        
        source = GoogleDriveContentSource(config)
        document = source.fetch_document(doc_id)
        
        print(f"✅ Successfully exported Google Doc:")
        print(f"   Name: {document.get('metadata', {}).get('name', 'Unknown')}")
        print(f"   Doc type: {document.get('doc_type', 'Unknown')}")
        print(f"   Content type: {document.get('metadata', {}).get('content_type', 'Unknown')}")
        
        if 'content' in document:
            content_preview = str(document['content'])[:200]
            # For binary content (DOCX), just show length
            if document.get('doc_type') == 'docx':
                print(f"   Binary content: {len(document['content'])} bytes")
            else:
                # Strip HTML tags for preview
                import re
                text_preview = re.sub('<[^<]+?>', '', content_preview)
                print(f"   Content preview: {text_preview[:100]}...")
        elif 'binary_path' in document:
            file_size = os.path.getsize(document['binary_path']) if os.path.exists(document['binary_path']) else 0
            print(f"   Binary file: {document['binary_path']}")
            print(f"   File size: {file_size} bytes")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to export Google Doc: {e}")
        return False

def main():
    """Run all Google Drive tests."""
    print("=" * 60)
    print("🚀 Google Drive Integration Tests")
    print("=" * 60)
    
    # Setup environment
    print("\n📝 Setting up environment...")
    if not setup_environment():
        return 1
    
    # Verify credentials
    print("\n🔐 Verifying credentials...")
    if not verify_credentials():
        return 1
    
    # Run tests
    tests_passed = 0
    tests_total = 4
    
    if test_connection():
        tests_passed += 1
    
    if test_specific_file():
        tests_passed += 1
    
    if test_folder():
        tests_passed += 1
    
    if test_google_doc():
        tests_passed += 1
    
    # Summary
    print("\n" + "=" * 60)
    print(f"📊 Test Results: {tests_passed}/{tests_total} passed")
    if tests_passed == tests_total:
        print("✅ All tests passed!")
    else:
        print(f"⚠️  {tests_total - tests_passed} test(s) failed")
    print("=" * 60)
    
    return 0 if tests_passed == tests_total else 1

if __name__ == "__main__":
    sys.exit(main())