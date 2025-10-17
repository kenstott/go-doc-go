#!/bin/bash
# Update all internal links after documentation reorganization

set -e

echo "=========================================="
echo "Documentation Link Update Script"
echo "=========================================="
echo ""

# Color codes for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Counter for changes
TOTAL_CHANGES=0

# Function to update links in a file
update_links_in_file() {
    local file="$1"
    local changes=0

    if [ ! -f "$file" ]; then
        echo -e "${RED}File not found: $file${NC}"
        return
    fi

    echo "Processing: $file"

    # Create temporary file
    local temp_file=$(mktemp)
    cp "$file" "$temp_file"

    # Update links for moved files
    # Getting Started
    sed -i '' 's|GETTING_STARTED\.md|docs/getting-started/README.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|(getting-started\.md)|(getting-started/README.md)|g' "$file" && ((changes++)) || true

    # Configuration files
    sed -i '' 's|docs/configuration\.md|docs/configuration/README.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|docs/sources\.md|docs/configuration/sources.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|docs/storage\.md|docs/configuration/storage.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|(configuration\.md)|(configuration/README.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(sources\.md)|(configuration/sources.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(storage\.md)|(configuration/storage.md)|g' "$file" && ((changes++)) || true

    # Embeddings
    sed -i '' 's|docs/embeddings\.md|docs/features/embeddings/README.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|(embeddings\.md)|(features/embeddings/README.md)|g' "$file" && ((changes++)) || true

    # Ontology files
    sed -i '' 's|docs/ontology\.md|docs/features/ontology/README.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|docs/ontology-examples\.md|docs/features/ontology/examples.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|docs/ONTOLOGY_QUICK_START\.md|docs/features/ontology/quick-start.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|docs/ONTOLOGY_WORKFLOWS\.md|docs/features/ontology/workflows.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|docs/domain-quickstart\.md|docs/features/ontology/domain-quickstart.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|(ontology\.md)|(features/ontology/README.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(ontology-examples\.md)|(features/ontology/examples.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(ONTOLOGY_QUICK_START\.md)|(features/ontology/quick-start.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(ONTOLOGY_WORKFLOWS\.md)|(features/ontology/workflows.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(domain-quickstart\.md)|(features/ontology/domain-quickstart.md)|g' "$file" && ((changes++)) || true

    # UDML files
    sed -i '' 's|docs/UDML_SPECIFICATION\.md|docs/features/udml/specification.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|docs/UDML_SCHEMAS\.md|docs/features/udml/schemas.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|docs/UDML_ONTOLOGY_SYSTEM\.md|docs/features/udml/ontology-system.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|(UDML_SPECIFICATION\.md)|(features/udml/specification.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(UDML_SCHEMAS\.md)|(features/udml/schemas.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(UDML_ONTOLOGY_SYSTEM\.md)|(features/udml/ontology-system.md)|g' "$file" && ((changes++)) || true

    # Operations
    sed -i '' 's|docs/scaling\.md|docs/operations/scaling.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|(scaling\.md)|(operations/scaling.md)|g' "$file" && ((changes++)) || true

    # Architecture files
    sed -i '' 's|GOROUTINE_WORKER_DESIGN\.md|docs/architecture/worker-design.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|UDML_ONTOLOGY_COMPLETE\.md|docs/architecture/udml-ontology-complete.md|g' "$file" && ((changes++)) || true
    sed -i '' 's|(GOROUTINE_WORKER_DESIGN\.md)|(docs/architecture/worker-design.md)|g' "$file" && ((changes++)) || true
    sed -i '' 's|(UDML_ONTOLOGY_COMPLETE\.md)|(docs/architecture/udml-ontology-complete.md)|g' "$file" && ((changes++)) || true

    # Check if file actually changed
    if ! cmp -s "$file" "$temp_file"; then
        echo -e "  ${GREEN}✓ Updated links${NC}"
        TOTAL_CHANGES=$((TOTAL_CHANGES + 1))
    else
        echo -e "  ${YELLOW}○ No changes needed${NC}"
    fi

    rm -f "$temp_file"
}

echo "Step 1: Updating links in root README.md"
echo "----------------------------------------"
if [ -f "README.md" ]; then
    update_links_in_file "README.md"
else
    echo -e "${RED}README.md not found${NC}"
fi
echo ""

echo "Step 2: Updating links in documentation files"
echo "----------------------------------------------"
# Find all markdown files in docs/ and update them
find docs -name "*.md" -type f | while read -r file; do
    update_links_in_file "$file"
done
echo ""

echo "Step 3: Updating links in other root markdown files"
echo "---------------------------------------------------"
for file in *.md; do
    if [ "$file" != "README.md" ] && [ -f "$file" ]; then
        update_links_in_file "$file"
    fi
done
echo ""

echo "=========================================="
echo "Link Update Summary"
echo "=========================================="
echo -e "${GREEN}Files modified: $TOTAL_CHANGES${NC}"
echo ""
echo "Next steps:"
echo "1. Review the changes with: git diff"
echo "2. Test links by opening markdown files"
echo "3. Commit changes if everything looks good"
echo ""