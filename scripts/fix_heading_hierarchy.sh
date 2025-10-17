#!/bin/bash
# Fix heading hierarchy - convert multiple H1s to H2s (keep only first H1)

echo "=== Heading Hierarchy Fixer ==="
echo "Converting multiple H1s to H2s..."
echo

FIXED=0

# Function to fix H1s in a file
fix_h1s() {
    local file="$1"
    
    # Count H1s before
    h1_count_before=$(grep -c "^# " "$file" 2>/dev/null || echo 0)
    
    if [ "$h1_count_before" -le 1 ]; then
        return 0  # File is OK
    fi
    
    echo "Fixing $file ($h1_count_before H1s)"
    
    # Create a temp file with fixed headings
    # Strategy: Keep first H1, convert all other H1s to H2s
    awk '
    BEGIN { first_h1 = 0 }
    /^# / {
        if (first_h1 == 0) {
            print $0  # Keep first H1
            first_h1 = 1
        } else {
            # Convert H1 to H2 (add one more #)
            print "#" $0
        }
        next
    }
    { print $0 }
    ' "$file" > "$file.tmp"
    
    mv "$file.tmp" "$file"
    
    # Count H1s after
    h1_count_after=$(grep -c "^# " "$file" 2>/dev/null || echo 0)
    
    echo "  $h1_count_before H1s → $h1_count_after H1 ✓"
    FIXED=$((FIXED + 1))
}

# Fix all files with multiple H1s
for file in docs/**/*.md *.md; do
    if [ -f "$file" ] && [ "$file" != "docs/GLOSSARY.md" ]; then
        fix_h1s "$file"
    fi
done

echo
echo "=== Summary ==="
echo "Fixed $FIXED files"
