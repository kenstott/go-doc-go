#!/bin/bash
# Validate all TOML code blocks in documentation
# Extracts TOML from markdown and validates syntax

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

VALID=0
INVALID=0
TOTAL=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "=== TOML Validation in Documentation ==="
echo "Project root: $PROJECT_ROOT"
echo "Temp directory: $TEMP_DIR"
echo ""

# Check if we have a TOML validator
VALIDATOR=""
if command -v tomlv &> /dev/null; then
    VALIDATOR="tomlv"
    echo "Using validator: tomlv"
elif command -v toml-test &> /dev/null; then
    VALIDATOR="toml-test"
    echo "Using validator: toml-test"
elif command -v python3 &> /dev/null; then
    # Check if Python tomli/toml module is available
    if python3 -c "import tomli" &> /dev/null 2>&1 || python3 -c "import toml" &> /dev/null 2>&1; then
        VALIDATOR="python"
        echo "Using validator: Python (tomli/toml)"
    fi
fi

if [ -z "$VALIDATOR" ]; then
    echo -e "${YELLOW}⚠ No TOML validator found${NC}"
    echo "  Install one of:"
    echo "    - tomlv: go install github.com/BurntSushi/toml/cmd/tomlv@latest"
    echo "    - Python tomli: pip install tomli"
    exit 1
fi

echo ""

# Python validation script
cat > "$TEMP_DIR/validate_toml.py" <<'EOF'
import sys
try:
    import tomli as toml
except ImportError:
    import toml

try:
    with open(sys.argv[1], 'rb') as f:
        toml.load(f)
    sys.exit(0)
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)
EOF

# Function to validate TOML file
validate_toml() {
    local file="$1"

    case "$VALIDATOR" in
        tomlv)
            if tomlv "$file" &> /dev/null; then
                return 0
            else
                return 1
            fi
            ;;
        python)
            if python3 "$TEMP_DIR/validate_toml.py" "$file" 2>&1; then
                return 0
            else
                return 1
            fi
            ;;
        *)
            echo "Unknown validator: $VALIDATOR" >&2
            return 1
            ;;
    esac
}

# Function to extract and validate TOML blocks from a markdown file
process_markdown_file() {
    local md_file="$1"
    local relative_path="${md_file#$PROJECT_ROOT/}"

    echo -e "${BLUE}Processing: $relative_path${NC}"

    local in_toml_block=0
    local block_num=0
    local line_num=0
    local start_line=0
    local current_block=""

    while IFS= read -r line; do
        line_num=$((line_num + 1))

        # Check for TOML code block start
        if [[ "$line" =~ ^'```'toml ]]; then
            in_toml_block=1
            block_num=$((block_num + 1))
            start_line=$line_num
            current_block=""
            continue
        fi

        # Check for code block end
        if [[ "$line" =~ ^'```'$ ]] && [ $in_toml_block -eq 1 ]; then
            in_toml_block=0
            TOTAL=$((TOTAL + 1))

            # Write block to temp file
            local temp_toml="$TEMP_DIR/${relative_path//\//_}_block_${block_num}.toml"
            echo "$current_block" > "$temp_toml"

            # Validate
            if validate_toml "$temp_toml" 2> "$temp_toml.error"; then
                echo -e "  ${GREEN}✓${NC} Block $block_num (line $start_line)"
                VALID=$((VALID + 1))
            else
                echo -e "  ${RED}✗${NC} Block $block_num (line $start_line)"
                echo -e "    ${YELLOW}Error:${NC}"
                sed 's/^/      /' "$temp_toml.error"
                INVALID=$((INVALID + 1))
            fi

            current_block=""
            continue
        fi

        # Accumulate TOML content
        if [ $in_toml_block -eq 1 ]; then
            current_block+="$line"$'\n'
        fi

    done < "$md_file"
}

# Find all markdown files
echo "Finding markdown files..."
MARKDOWN_FILES=$(find "$PROJECT_ROOT" -name "*.md" -type f \
    ! -path "*/node_modules/*" \
    ! -path "*/.git/*" \
    ! -path "*/vendor/*")

FILE_COUNT=$(echo "$MARKDOWN_FILES" | wc -l | tr -d ' ')
echo "Found $FILE_COUNT markdown files"
echo ""

# Process each markdown file
for md_file in $MARKDOWN_FILES; do
    # Check if file contains TOML code blocks
    if grep -q '```toml' "$md_file"; then
        process_markdown_file "$md_file"
    fi
done

# Summary
echo ""
echo "=== Validation Summary ==="
echo -e "Total TOML blocks:  $TOTAL"
echo -e "Valid:              ${GREEN}$VALID${NC}"
echo -e "Invalid:            ${RED}$INVALID${NC}"

if [ $TOTAL -eq 0 ]; then
    echo -e "\n${YELLOW}⚠ No TOML blocks found in documentation${NC}"
    exit 0
fi

echo ""
PERCENT=$((VALID * 100 / TOTAL))
echo "Success rate: ${PERCENT}%"

if [ $INVALID -gt 0 ]; then
    echo ""
    echo -e "${RED}Some TOML blocks are invalid. Fix them before committing.${NC}"
    exit 1
else
    echo ""
    echo -e "${GREEN}All TOML blocks are valid!${NC}"
    exit 0
fi
