#!/bin/sh
# MinIO bucket initialization script

set -e

echo "Waiting for MinIO to be ready..."
sleep 5

# Configure MinIO client
mc config host add minio http://minio-test:9000 minioadmin minioadmin

echo "Creating buckets..."

# Create test buckets
mc mb --ignore-existing minio/test-bucket
mc mb --ignore-existing minio/test-private
mc mb --ignore-existing minio/test-public

echo "Setting bucket policies..."

# Make test-public bucket publicly readable
mc anonymous set download minio/test-public

# Set versioning on test-bucket (optional, for testing versioning features)
mc version enable minio/test-bucket || true

echo "Uploading sample data..."

# Upload sample files if they exist
if [ -d "/sample-data" ]; then
  echo "Found sample data directory, uploading files..."
  
  # Upload all files from sample-data to test-bucket
  mc cp --recursive /sample-data/ minio/test-bucket/ 2>/dev/null || true
  
  # Create some test prefixes/folders
  mc cp /sample-data/README.md minio/test-bucket/documents/README.md 2>/dev/null || true
  mc cp /sample-data/config.yaml minio/test-bucket/configs/config.yaml 2>/dev/null || true
  
  echo "Sample data uploaded"
else
  echo "No sample data directory found, creating minimal test data..."
  
  # Create some test objects directly
  echo "Test content 1" | mc pipe minio/test-bucket/test1.txt
  echo "Test content 2" | mc pipe minio/test-bucket/folder1/test2.txt
  echo '{"test": "data"}' | mc pipe minio/test-bucket/data/test.json
  echo "# Test Markdown" | mc pipe minio/test-bucket/docs/test.md
fi

echo "Creating test IAM user..."

# Create a test user with limited permissions (optional)
mc admin user add minio testuser testpass123 || true

# Create and attach a policy for the test user
cat > /tmp/testuser-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::test-bucket/*",
        "arn:aws:s3:::test-bucket"
      ]
    }
  ]
}
EOF

mc admin policy create minio testuser-policy /tmp/testuser-policy.json || true
mc admin policy attach minio testuser-policy --user testuser || true

echo "Listing bucket contents..."
mc ls minio/test-bucket

echo "MinIO initialization complete!"
echo "Buckets created: test-bucket, test-private, test-public"
echo "Test user created: testuser/testpass123"
echo "Access MinIO at:"
echo "  - S3 API: http://localhost:9000"
echo "  - Console: http://localhost:9001 (minioadmin/minioadmin)"