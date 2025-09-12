#!/bin/bash
set -e

echo "Starting Solr core initialization..."

# Wait for Solr to start
echo "Waiting for Solr to start..."
solr start -force

# Give Solr a moment to initialize
sleep 5

# Create cores if they don't exist
echo "Creating cores if they don't exist..."
solr create_core -c go_doc_go_documents -d _default || echo "Core go_doc_go_documents already exists or couldn't be created"
solr create_core -c go_doc_go_elements -d _default || echo "Core go_doc_go_elements already exists or couldn't be created"
solr create_core -c go_doc_go_relationships -d _default || echo "Core go_doc_go_relationships already exists or couldn't be created"
solr create_core -c go_doc_go_history -d _default || echo "Core go_doc_go_history already exists or couldn't be created"

# Stop the temporarily started Solr instance
echo "Stopping temporary Solr instance..."
solr stop -force

echo "Initialization complete. Starting Solr in foreground mode..."

# Start Solr in foreground mode
exec solr-foreground
