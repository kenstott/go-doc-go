#!/bin/bash
# Add language tags to untagged code blocks

echo "=== Code Block Language Tag Adder ==="
echo "Adding language tags to untagged code blocks..."
echo

FIXED=0

# Function to infer language from context and content
infer_language() {
    local file="$1"
    local line_num="$2"
    
    # Read 3 lines before and 3 lines after for context
    local context=$(sed -n "$((line_num-3)),$((line_num+5))p" "$file")
    
    # Check for explicit language indicators in surrounding text
    if echo "$context" | grep -iq "bash\|shell\|command"; then
        echo "bash"
    elif echo "$context" | grep -iq "toml\|config\.toml"; then
        echo "toml"
    elif echo "$context" | grep -iq "yaml"; then
        echo "yaml"
    elif echo "$context" | grep -iq "python"; then
        echo "python"
    elif echo "$context" | grep -iq "go\|golang"; then
        echo "go"
    elif echo "$context" | grep -iq "json"; then
        echo "json"
    elif echo "$context" | grep -iq "sql\|SELECT\|INSERT"; then
        echo "sql"
    else
        # Check first line of code block for syntax clues
        local first_line=$(sed -n "$((line_num+1))p" "$file")
        
        if echo "$first_line" | grep -q "^#\!/bin/bash\|^#\!/usr/bin/env bash"; then
            echo "bash"
        elif echo "$first_line" | grep -q "^\["; then
            echo "toml"
        elif echo "$first_line" | grep -q "^package\|^func\|^import"; then
            echo "go"
        elif echo "$first_line" | grep -q "^def \|^class \|^import "; then
            echo "python"
        elif echo "$first_line" | grep -q "^SELECT\|^INSERT\|^UPDATE\|^CREATE"; then
            echo "sql"
        elif echo "$first_line" | grep -q "^{"; then
            echo "json"
        elif echo "$first_line" | grep -q "^\$\|^#"; then
            echo "bash"
        else
            echo ""  # Can't determine
        fi
    fi
}

# Function to add language tags to a file
fix_code_blocks() {
    local file="$1"
    
    # Find all untagged code blocks (lines with just ```)
    local untagged_lines=$(grep -n "^\`\`\`$" "$file" 2>/dev/null || echo "")
    
    if [ -z "$untagged_lines" ]; then
        return 0
    fi
    
    echo "Fixing $file"
    
    # Process file line by line, adding language tags
    local temp_file="${file}.tmp"
    local current_line=1
    local fixed_in_file=0
    
    > "$temp_file"  # Create empty temp file
    
    while IFS= read -r line; do
        # Check if this line is an untagged code block opener
        if [ "$line" = '```' ]; then
            # Infer language
            local lang=$(infer_language "$file" "$current_line")
            
            if [ -n "$lang" ]; then
                echo "\`\`\`$lang" >> "$temp_file"
                fixed_in_file=$((fixed_in_file + 1))
            else
                echo "$line" >> "$temp_file"
            fi
        else
            echo "$line" >> "$temp_file"
        fi
        
        current_line=$((current_line + 1))
    done < "$file"
    
    # Replace original file
    mv "$temp_file" "$file"
    
    if [ $fixed_in_file -gt 0 ]; then
        echo "  Fixed $fixed_in_file code blocks"
        FIXED=$((FIXED + fixed_in_file))
    fi
}

# Fix all markdown files
for file in docs/**/*.md *.md; do
    if [ -f "$file" ] && [ "$file" != "docs/GLOSSARY.md" ]; then
        fix_code_blocks "$file"
    fi
done

echo
echo "=== Summary ==="
echo "Fixed $FIXED code blocks"
