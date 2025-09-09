# Google Drive Integration Setup Guide

This guide explains how to set up Google Drive credentials for the Go-Doc-Go integration tests and production use.

## Prerequisites

1. A Google account
2. Access to [Google Cloud Console](https://console.cloud.google.com/)
3. Python packages: `pip install google-api-python-client google-auth-oauthlib google-auth-httplib2`

## Authentication Methods

Google Drive integration supports two authentication methods:

### Method 1: Service Account (Recommended for Testing)

Service accounts are best for automated testing and server applications because they don't require interactive authentication.

#### Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Select a project" → "New Project"
3. Enter a project name (e.g., "go-doc-go-testing")
4. Click "Create"

#### Step 2: Enable Google Drive API

1. In the Cloud Console, go to "APIs & Services" → "Library"
2. Search for "Google Drive API"
3. Click on it and press "Enable"

#### Step 3: Create a Service Account

1. Go to "APIs & Services" → "Credentials"
2. Click "Create Credentials" → "Service Account"
3. Fill in the service account details:
   - Name: `go-doc-go-service-account`
   - ID: (auto-generated)
   - Description: "Service account for Go-Doc-Go Google Drive integration"
4. Click "Create and Continue"
5. Skip the optional steps (roles and user access)
6. Click "Done"

#### Step 4: Generate a Key

1. Click on the service account you just created
2. Go to the "Keys" tab
3. Click "Add Key" → "Create new key"
4. Choose "JSON" format
5. Click "Create"
6. Save the downloaded JSON file securely

#### Step 5: Share Files/Folders with Service Account

1. Find the service account email in the JSON file (look for `client_email`)
2. In Google Drive, share your test files/folders with this email address
3. Grant "Viewer" permission (or "Editor" if write access is needed)

#### Step 6: Configure Environment

Create a `.env.google_drive` file:

```bash
GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE=/path/to/your-service-account-key.json
GOOGLE_DRIVE_TEST_FILE_ID=<your-test-file-id>
GOOGLE_DRIVE_TEST_FOLDER_ID=<your-test-folder-id>
```

### Method 2: OAuth 2.0 (For Interactive Use)

OAuth is suitable for desktop applications where users can authenticate interactively.

#### Step 1: Create OAuth 2.0 Credentials

1. In Google Cloud Console, go to "APIs & Services" → "Credentials"
2. Click "Create Credentials" → "OAuth client ID"
3. If prompted, configure the OAuth consent screen:
   - User Type: "External" (or "Internal" for Google Workspace)
   - Fill in required fields
   - Add your email to test users
4. For Application type, choose "Desktop app"
5. Name it "Go-Doc-Go Desktop Client"
6. Click "Create"
7. Download the credentials JSON file

#### Step 2: Configure Environment

Create a `.env.google_drive` file:

```bash
GOOGLE_DRIVE_CREDENTIALS_PATH=/path/to/credentials.json
GOOGLE_DRIVE_TOKEN_PATH=/path/to/token.pickle
GOOGLE_DRIVE_TEST_FILE_ID=<your-test-file-id>
```

#### Step 3: Initial Authentication

Run the tests once to trigger authentication:

```bash
# Load environment variables
source .env.google_drive

# Run a test to trigger OAuth flow
pytest tests/test_content_sources/test_google_drive_content_source.py::TestGoogleDriveContentSourceIntegration::test_connection -v
```

This will:
1. Open a browser for authentication
2. Ask you to authorize the application
3. Save the token to `token.pickle` for future use

## Getting File and Folder IDs

You can provide either the full URL or just the ID - the tests will automatically extract IDs from URLs.

### File ID
1. Open the file in Google Drive
2. Look at the URL: `https://drive.google.com/file/d/FILE_ID_HERE/view`
3. You can use either:
   - The full URL: `https://drive.google.com/file/d/1HX39-IKUVUciH7ATuUolYr1DhuFTOy-i/view`
   - Or just the ID: `1HX39-IKUVUciH7ATuUolYr1DhuFTOy-i`

### Folder ID
1. Open the folder in Google Drive
2. Look at the URL: `https://drive.google.com/drive/folders/FOLDER_ID_HERE`
3. You can use either:
   - The full URL: `https://drive.google.com/drive/folders/1g07g0xa9LZfRAEa163x55qfwp7mCYqHq`
   - Or just the ID: `1g07g0xa9LZfRAEa163x55qfwp7mCYqHq`

### Google Docs/Sheets/Slides ID
1. Open the document
2. Look at the URL: `https://docs.google.com/document/d/DOC_ID_HERE/edit`
3. You can use either:
   - The full URL: `https://docs.google.com/document/d/1FbVe1JfKNzQD5qVogL6f0OTdxm-5GgovS7kxpanpPFY/edit`
   - Or just the ID: `1FbVe1JfKNzQD5qVogL6f0OTdxm-5GgovS7kxpanpPFY`

## Document Export Formats

Google Drive integration automatically exports Google's native formats to MS Office formats for better parsing:

- **Google Docs** → **DOCX** (Word format)
- **Google Sheets** → **XLSX** (Excel format)
- **Google Slides** → **PPTX** (PowerPoint format)
- **Google Drawings** → **PNG** (Image format)
- **Google Forms** → **HTML** (Web format)

This allows the existing DOCX, XLSX, and PPTX parsers to handle Google's native documents seamlessly.

## Running Integration Tests

### Using .env.google_drive File

```bash
# Source the environment file
source .env.google_drive

# Export all variables (handles both URLs and IDs)
export GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE=$(eval echo $GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE)
export GOOGLE_DRIVE_TEST_FILE_ID
export GOOGLE_DRIVE_TEST_FOLDER_ID
export GOOGLE_DRIVE_TEST_GOOGLE_DOC_ID

# Run all tests
pytest tests/test_content_sources/test_google_drive_content_source.py \
       tests/test_adapters/test_google_drive_adapter.py -v
```

### With Service Account (Manual)

```bash
# Set environment variables (can use URLs or just IDs)
export GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE=/path/to/service-account-key.json
export GOOGLE_DRIVE_TEST_FILE_ID="https://drive.google.com/file/d/YOUR_ID/view"
export GOOGLE_DRIVE_TEST_FOLDER_ID="https://drive.google.com/drive/folders/YOUR_ID"
export GOOGLE_DRIVE_TEST_GOOGLE_DOC_ID="https://docs.google.com/document/d/YOUR_ID/edit"

# Run tests
pytest tests/test_content_sources/test_google_drive_content_source.py -v
```

### With OAuth

```bash
# Set environment variables
export GOOGLE_DRIVE_CREDENTIALS_PATH=/path/to/credentials.json
export GOOGLE_DRIVE_TOKEN_PATH=/path/to/token.pickle
export GOOGLE_DRIVE_TEST_FILE_ID=your-file-id-or-url

# Run tests
pytest tests/test_content_sources/test_google_drive_content_source.py -v
```

### Quick Test Script

```bash
# Use the provided test runner for quick validation
python test_google_drive.py
```

## CI/CD Configuration

For continuous integration, use service account authentication:

### GitHub Actions Example

```yaml
- name: Run Google Drive Integration Tests
  env:
    GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE: ${{ runner.temp }}/service-account.json
    GOOGLE_DRIVE_TEST_FILE_ID: ${{ secrets.GOOGLE_DRIVE_TEST_FILE_ID }}
    GOOGLE_DRIVE_TEST_FOLDER_ID: ${{ secrets.GOOGLE_DRIVE_TEST_FOLDER_ID }}
  run: |
    # Write service account key from secret
    echo '${{ secrets.GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY }}' > $GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE
    
    # Run tests
    pytest tests/test_content_sources/test_google_drive_content_source.py -v -m integration
```

### GitLab CI Example

```yaml
test:google-drive:
  variables:
    GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE: /tmp/service-account.json
  script:
    - echo "$GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY" > $GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE
    - pytest tests/test_content_sources/test_google_drive_content_source.py -v -m integration
```

## Security Best Practices

1. **Never commit credentials to version control**
   - Add `*.json`, `token.pickle`, and `.env.google_drive` to `.gitignore`

2. **Use minimal permissions**
   - For read-only operations, use `drive.readonly` scope
   - Only grant write access if necessary

3. **Rotate service account keys regularly**
   - Delete old keys in Google Cloud Console
   - Generate new keys periodically

4. **Use secrets management in CI/CD**
   - Store credentials as encrypted secrets
   - Never log or output credentials

5. **Limit service account access**
   - Only share specific test files/folders
   - Don't give service account access to sensitive data

## Troubleshooting

### "Credentials file not found"
- Check the file path in your environment variables
- Ensure the file exists and is readable

### "Permission denied" errors
- Verify the file/folder is shared with the service account email
- Check that the service account has appropriate permissions

### "Invalid credentials" errors
- For OAuth: Delete `token.pickle` and re-authenticate
- For Service Account: Verify the JSON file is valid

### "API not enabled" errors
- Go to Google Cloud Console
- Enable the Google Drive API for your project

### Rate limiting issues
- Google Drive API has quotas
- Implement exponential backoff for retries
- Consider caching responses

## Example Test Data Setup

1. Create a test folder in Google Drive named "Go-Doc-Go Test Data"
2. Add various file types:
   - A PDF document
   - A text file
   - A Google Doc
   - A Google Sheet
   - An image
3. Share the folder with your service account email
4. Use the folder ID in your tests

## Additional Resources

- [Google Drive API Documentation](https://developers.google.com/drive/api/v3/about-sdk)
- [Google Cloud Console](https://console.cloud.google.com/)
- [OAuth 2.0 for Desktop Apps](https://developers.google.com/identity/protocols/oauth2/native-app)
- [Service Account Documentation](https://cloud.google.com/iam/docs/service-accounts)