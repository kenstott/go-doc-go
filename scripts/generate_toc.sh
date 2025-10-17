#!/bin/bash
# Generate table of contents for markdown files
# Scans headings (H2-H6) and creates anchor links

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROCESSED=0

echo "=== Table of Contents Generator ==="
echo "Project root: $PROJECT_ROOT"
echo ""

# Function to generate TOC for a single file
generate_toc() {
    local file="$1"
    local relative_path="${file#$PROJECT_ROOT/}"

    echo -e "${BLUE}Processing: $relative_path${NC}"

    # Check if file already has TOC
    if grep -q "<!-- TOC -->" "$file"; then
        echo -e "  ${YELLOW}⚠ TOC already exists, skipping${NC}"
        return 0
    fi

    # Extract headings (H2-H6, skip H1)
    local toc=""
    local has_headings=0

    while IFS= read -r line; do
        # Match heading lines (## through ######)
        if [[ "$line" =~ ^(##[#]{0,4})[[:space:]](.+)$ ]]; then
            has_headings=1
            local hashes="${BASH_REMATCH[1]}"
            local title="${BASH_REMATCH[2]}"

            # Calculate indent level (H2=0, H3=1, H4=2, etc.)
            local level=$((${#hashes} - 2))
            local indent=""
            for ((i=0; i<level; i++)); do
                indent="${indent}  "
            done

            # Create anchor (lowercase, replace spaces with hyphens, remove special chars)
            local anchor=$(echo "$title" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9 -]//g' | sed 's/ /-/g')

            # Add TOC entry
            toc="${toc}${indent}- [${title}](#${anchor})\n"
        fi
    done < "$file"

    if [ $has_headings -eq 0 ]; then
        echo -e "  ${YELLOW}⊘ No headings found (H2-H6)${NC}"
        return 0
    fi

    # Find where to insert TOC (after first H1 and any intro paragraph)
    local insert_line=$(awk '/^# / {found=1; next} found && /^$/ {print NR; exit}' "$file")

    if [ -z "$insert_line" ]; then
        # No blank line after H1, insert after H1
        insert_line=$(awk '/^# / {print NR+1; exit}' "$file")
    fi

    if [ -z "$insert_line" ]; then
        echo -e "  ${RED}✗ Could not find insertion point${NC}"
        return 1
    fi

    # Create TOC content
    local toc_content="<!-- TOC -->\n## Table of Contents\n\n${toc}\n<!-- /TOC -->\n"

    # Insert TOC
    awk -v line="$insert_line" -v toc="$toc_content" '
    NR == line {
        printf "%s\n", toc
    }
    { print }
    ' "$file" > "$file.tmp"

    mv "$file.tmp" "$file"

    echo -e "  ${GREEN}✓ TOC generated${NC}"
    PROCESSED=$((PROCESSED + 1))
    return 0
}

# Major documentation files that should have TOC
MAJOR_DOCS=(
    "$PROJECT_ROOT/README.md"
    "$PROJECT_ROOT/QUICK_REFERENCE.md"
    "$PROJECT_ROOT/DEVELOPMENT.md"
    "$PROJECT_ROOT/DISTRIBUTION.md"
    "$PROJECT_ROOT/ONTOLOGY_DEMO_WALKTHROUGH.md"
    "$PROJECT_ROOT/docs/README.md"
    "$PROJECT_ROOT/docs/getting-started/README.md"
    "$PROJECT_ROOT/docs/configuration/README.md"
    "$PROJECT_ROOT/docs/configuration/sources.md"
    "$PROJECT_ROOT/docs/configuration/storage.md"
    "$PROJECT_ROOT/docs/features/embeddings/README.md"
    "$PROJECT_ROOT/docs/features/ontology/README.md"
    "$PROJECT_ROOT/docs/features/udml/specification.md"
    "$PROJECT_ROOT/docs/operations/scaling.md"
    "$PROJECT_ROOT/docs/operations/monitoring.md"
    "$PROJECT_ROOT/docs/operations/troubleshooting.md"
    "$PROJECT_ROOT/docs/reference/cli.md"
    "$PROJECT_ROOT/docs/architecture/worker-design.md"
    "$PROJECT_ROOT/docs/architecture/udml-ontology-complete.md"
)

# Process all major docs
for doc in "${MAJOR_DOCS[@]}"; do
    if [ -f "$doc" ]; then
        generate_toc "$doc"
    else
        echo -e "${YELLOW}⊘ File not found: ${doc#$PROJECT_ROOT/}${NC}"
    fi
    echo ""
done

# Summary
echo "=== Summary ==="
echo -e "Files processed: $PROCESSED"

if [ $PROCESSED -gt 0 ]; then
    echo -e "\n${GREEN}Table of contents added to $PROCESSED files${NC}"
else
    echo -e "\n${YELLOW}No files needed TOC (all already have one or no headings)${NC}"
fi
