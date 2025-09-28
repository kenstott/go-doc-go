#!/bin/bash

# Process All SEC HTML Files
# This script processes all 147 SEC HTML files found in the directory

echo "🚀 Starting SEC HTML Document Store Processing"
echo "=============================================="
echo "Configuration: configs/sec-html-store.yaml"
echo "Expected files: 147 HTML documents"
echo "Output: Parquet analytics in ./sec_analytics/"
echo ""

# Run the processing worker without document limit to process all files
PYTHONPATH=src python -m go_doc_go worker --config configs/sec-html-store.yaml

echo ""
echo "✅ Processing complete!"
echo ""
echo "📊 Generate analytics report:"
echo "PYTHONPATH=src python -m go_doc_go analytics --config configs/sec-html-store.yaml"
echo ""
echo "📁 View outputs:"
echo "ls -la sec_analytics/"
echo ""
echo "🔍 Check status:"
echo "PYTHONPATH=src python -m go_doc_go status --config configs/sec-html-store.yaml"