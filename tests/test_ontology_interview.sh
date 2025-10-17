#!/bin/bash
# Test script for non-interactive ontology interview using Wikipedia corpus

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PARQUET_PATH="$SCRIPT_DIR/test_output/wikipedia_medical_mining_analytics"
OUTPUT_PATH="$SCRIPT_DIR/test_output/wikipedia_ontology.json"
BINARY="$PROJECT_ROOT/bin/ontology_interview"

echo "=================================================="
echo "Wikipedia Ontology Interview Test"
echo "=================================================="
echo "Parquet path: $PARQUET_PATH"
echo "Output path: $OUTPUT_PATH"
echo ""

# Check if parquet data exists
if [ ! -d "$PARQUET_PATH/documents" ]; then
    echo "❌ Error: No document data found in $PARQUET_PATH"
    echo "   Make sure workers have processed some documents first"
    exit 1
fi

# Check for API key
if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "❌ Error: ANTHROPIC_API_KEY environment variable not set"
    exit 1
fi

echo "✓ Parquet data found"
echo "✓ API key configured"
echo ""

# Run non-interactive ontology interview
echo "Running ontology interview in non-interactive mode..."
echo "This will:"
echo "  1. Auto-accept suggested domains"
echo "  2. Auto-accept extraction rules"
echo "  3. Save ontology to: $OUTPUT_PATH"
echo ""

# Run with default diversity threshold (0.85 - filters out >85% similar samples)
# Lower threshold = more strict filtering (fewer samples kept)
# Higher threshold = less strict filtering (more samples kept)
$BINARY \
    --non-interactive \
    "$PARQUET_PATH" \
    "$OUTPUT_PATH"

if [ $? -eq 0 ]; then
    echo ""
    echo "=================================================="
    echo "✓ SUCCESS: Ontology created"
    echo "=================================================="
    echo ""
    echo "Ontology saved to: $OUTPUT_PATH"
    echo ""

    # Show schema summary
    if [ -f "$OUTPUT_PATH" ]; then
        echo "Schema summary:"
        echo "---------------"
        jq -r '
            "Name: \(.name)",
            "Version: \(.version)",
            "Domains: \(.domains | length)",
            "Entity Types: \(.entity_types | length)",
            "Relationships: \(.relationships | length)"
        ' "$OUTPUT_PATH"

        echo ""
        echo "Domain details:"
        jq -r '.domains[] | "  - \(.name): \(.description)"' "$OUTPUT_PATH"
    fi
else
    echo ""
    echo "❌ FAILED: Ontology interview failed"
    exit 1
fi