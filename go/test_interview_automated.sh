#!/bin/bash
# Automated test of ontology interview system
# Simulates user input to test the full workflow

set -e

echo "=================================="
echo "ONTOLOGY INTERVIEW AUTOMATED TEST"
echo "=================================="
echo ""

# Check API key
if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "ERROR: ANTHROPIC_API_KEY not set"
    exit 1
fi

echo "✓ API key is set"
echo "✓ Corpus location: ./test_interview_corpus"
echo "✓ Output: ./test_ontology.json"
echo ""

# Create simulated user input
# Phase 1 questions: answer with context
# Final approval: approve the schema
cat > /tmp/interview_input.txt <<EOF
Software development organization
Single integrated system
yes
approve
EOF

echo "Starting interview (automated responses)..."
echo ""

# Run interview with simulated input
./bin/ontology_interview ./test_interview_corpus ./test_ontology.json < /tmp/interview_input.txt

echo ""
echo "=================================="
echo "TEST COMPLETE"
echo "=================================="

# Show results
if [ -f "./test_ontology.json" ]; then
    echo "✓ Ontology created successfully"
    echo ""
    echo "Schema summary:"
    echo "---------------"
    jq '. | {name, version, domains: .domains | map(.name), entity_count: (.element_entity_mappings | length), relationship_count: (.entity_relationship_rules | length)}' ./test_ontology.json 2>/dev/null || echo "Install jq for JSON preview"
else
    echo "✗ Ontology file not created"
    exit 1
fi
