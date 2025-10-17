#!/bin/bash
set -e

echo "=================================================="
echo "Hybrid Domain Selection Test"
echo "=================================================="
echo "Testing hybrid domain selection with Wikipedia corpus"
echo ""

# Use the existing Wikipedia parquet data
PARQUET_PATH="/Users/kennethstott/PycharmProjects/doculyzer-go-conversion/tests/test_output/wikipedia_medical_mining_analytics"
OUTPUT_PATH="/Users/kennethstott/PycharmProjects/doculyzer-go-conversion/tests/test_output/hybrid_ontology_test.json"
LOG_PATH="/Users/kennethstott/PycharmProjects/doculyzer-go-conversion/tests/test_output/hybrid_ontology_test.log"

# Check if parquet data exists
if [ ! -d "$PARQUET_PATH" ]; then
    echo "❌ Parquet data not found at: $PARQUET_PATH"
    exit 1
fi

# Check for API key
if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "❌ ANTHROPIC_API_KEY not set"
    exit 1
fi

echo "✓ Parquet data found"
echo "✓ API key configured"
echo ""
echo "Running ontology interview in non-interactive mode..."
echo "Expected behavior: Catalog coverage should be ~70%, so NO hybrid mode triggered"
echo ""

# Run the ontology interview CLI
ONTOLOGY_CATALOG_PATH=./examples/ontologies \
  bin/ontology_interview \
  --non-interactive \
  "$PARQUET_PATH" "$OUTPUT_PATH" \
  2>&1 | tee "$LOG_PATH"

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "=================================================="
    echo "✓ SUCCESS: Hybrid mode test completed"
    echo "=================================================="
    echo ""
    echo "Ontology saved to: $OUTPUT_PATH"
    echo "Log saved to: $LOG_PATH"
    echo ""

    # Verify output file exists
    if [ -f "$OUTPUT_PATH" ]; then
        echo "Ontology file size: $(wc -c < "$OUTPUT_PATH") bytes"
        echo ""
        echo "Domains in ontology:"
        cat "$OUTPUT_PATH" | jq -r '.domains[].name' 2>/dev/null || echo "(could not extract domains)"
    else
        echo "⚠️  Output file not found: $OUTPUT_PATH"
    fi
else
    echo ""
    echo "=================================================="
    echo "❌ FAILED: Test exited with code $EXIT_CODE"
    echo "=================================================="
    exit $EXIT_CODE
fi