#!/bin/bash
# Check for terminology inconsistencies in documentation

echo "=== Go-Doc-Go Terminology Checker ==="
echo "Checking for non-standard terms..."
echo

ERRORS=0

# Function to check for a term and suggest replacement
check_term() {
    local bad_term="$1"
    local good_term="$2"
    local pattern="$3"
    local files="docs/**/*.md *.md"

    # Use grep to find occurrences (case-insensitive for some terms)
    local matches=$(grep -rn "$pattern" $files 2>/dev/null | grep -v "GLOSSARY.md" | grep -v "Not:" | grep -v "Not This" | grep -v "❌")

    if [ -n "$matches" ]; then
        echo "❌ Found '$bad_term' (should use '$good_term'):"
        echo "$matches" | head -5
        if [ $(echo "$matches" | wc -l) -gt 5 ]; then
            echo "   ... and $(($(echo "$matches" | wc -l) - 5)) more"
        fi
        echo
        ERRORS=$((ERRORS + 1))
    fi
}

echo "Checking terminology..."
echo

# Check for deprecated terms
check_term "worker node" "worker" "worker node"
check_term "worker instance" "worker" "worker instance"
check_term "work queue" "job control" "work queue"
check_term "task queue" "job control" "task queue"
check_term "job queue" "job control" "job queue"
check_term "data source" "content source" "data source"
check_term "document source" "content source" "document source"
check_term "input source" "content source" "input source"

# Check for "doc" instead of "document" (tricky - many false positives)
# Only check in prose, not in code/config examples
check_term "doc/" "document" '\bdoc\b'

# Check for YAML config references
echo "Checking for YAML config references..."
yaml_refs=$(grep -rn "config\.yaml\|\.yaml.*config\|YAML config" docs/**/*.md *.md 2>/dev/null | grep -v "ontology" | grep -v "schema" | grep -v "GLOSSARY.md")
if [ -n "$yaml_refs" ]; then
    echo "❌ Found YAML config references (should use TOML):"
    echo "$yaml_refs" | head -5
    if [ $(echo "$yaml_refs" | wc -l) -gt 5 ]; then
        echo "   ... and $(($(echo "$yaml_refs" | wc -l) - 5)) more"
    fi
    echo
    ERRORS=$((ERRORS + 1))
fi

echo "Checking code block language tags..."
# Check for code blocks without language tags
untagged=$(grep -rn '^```$' docs/**/*.md *.md 2>/dev/null | grep -v "GLOSSARY.md")
if [ -n "$untagged" ]; then
    echo "⚠️  Found code blocks without language tags:"
    echo "$untagged" | head -10
    if [ $(echo "$untagged" | wc -l) -gt 10 ]; then
        echo "   ... and $(($(echo "$untagged" | wc -l) - 10)) more"
    fi
    echo
    # Don't count as error, just warning
fi

echo "Checking heading hierarchy..."
# Check for h1 -> h3 skips (basic check)
echo "  (Checking for major heading hierarchy issues...)"
violations=0
for file in docs/**/*.md *.md; do
    if [ -f "$file" ]; then
        # Count h1 tags (should be exactly 1)
        h1_count=$(grep -c "^# " "$file" 2>/dev/null || echo 0)
        if [ "$h1_count" -gt 1 ]; then
            echo "⚠️  Multiple H1 headings in $file ($h1_count found)"
            violations=$((violations + 1))
        fi
    fi
done

if [ $violations -gt 0 ]; then
    echo "Found $violations heading hierarchy violations"
    echo
fi

echo "=== Summary ==="
if [ $ERRORS -eq 0 ]; then
    echo "✅ No critical terminology issues found!"
else
    echo "❌ Found $ERRORS terminology issues"
    echo "   Review the output above and update documentation to use standard terms"
    echo "   See docs/GLOSSARY.md for standard terminology"
fi

exit $ERRORS
