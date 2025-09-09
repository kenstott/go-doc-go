#!/usr/bin/env python3
"""
Script to verify Google Drive file sharing with service account.
"""

import json
import os
import sys

def main():
    """Display sharing instructions for Google Drive files."""
    
    print("=" * 60)
    print("🔐 Google Drive File Sharing Instructions")
    print("=" * 60)
    
    # Read service account email
    sa_file = os.path.expanduser("~/credentials/google-drive/service-account-key.json")
    
    if os.path.exists(sa_file):
        with open(sa_file, 'r') as f:
            sa_data = json.load(f)
        service_account_email = sa_data.get('client_email', 'Unknown')
    else:
        service_account_email = "your-service-account@project.iam.gserviceaccount.com"
    
    # Read test file IDs from environment
    test_files = {
        "Test File": "1HX39-IKUVUciH7ATuUolYr1DhuFTOy-i",
        "Test Folder": "1g07g0xa9LZfRAEa163x55qfwp7mCYqHq", 
        "Test Google Doc": "1FbVe1JfKNzQD5qVogL6f0OTdxm-5GgovS7kxpanpPFY"
    }
    
    print(f"\nService Account Email: {service_account_email}")
    print("\n" + "-" * 60)
    print("Files to Share:")
    print("-" * 60)
    
    for name, file_id in test_files.items():
        print(f"\n{name}:")
        print(f"  File ID: {file_id}")
        if name == "Test File":
            print(f"  URL: https://drive.google.com/file/d/{file_id}/view")
        elif name == "Test Folder":
            print(f"  URL: https://drive.google.com/drive/folders/{file_id}")
        elif name == "Test Google Doc":
            print(f"  URL: https://docs.google.com/document/d/{file_id}/edit")
    
    print("\n" + "=" * 60)
    print("📋 How to Share Files with Service Account")
    print("=" * 60)
    
    print(f"""
For each file/folder above:

1. Open the file/folder in Google Drive using the URL

2. Click the "Share" button

3. In the "Add people and groups" field, enter:
   {service_account_email}

4. Set permission to "Viewer" (or "Editor" if write access needed)

5. IMPORTANT: Uncheck "Notify people" (service accounts can't receive email)

6. Click "Share"

7. The file should now be accessible to the service account

Note: If these are not your files, you'll need to:
- Create your own test files in Google Drive
- Update the file IDs in .env.google_drive
- Share them with the service account

After sharing, run the tests again:
   python test_google_drive.py
""")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())