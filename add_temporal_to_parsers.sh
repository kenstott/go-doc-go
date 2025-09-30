#!/bin/bash

# Script to add temporal support to all Go parsers

PARSERS="docx xlsx csv xml html text markdown"

for parser in $PARSERS; do
    FILE="go/internal/parser/${parser}.go"

    if [ -f "$FILE" ]; then
        echo "Processing $parser..."

        # Check if already has temporal import
        if ! grep -q "internal/temporal" "$FILE"; then
            # Add temporal import after other imports
            sed -i '' '/^import (/a\
	"github.com/kennethstott/go-doc-go/internal/temporal"
' "$FILE"
        fi

        # Add ExtractDates field to parser struct if not present
        if ! grep -q "ExtractDates" "$FILE"; then
            # Find the parser struct and add ExtractDates field
            sed -i '' '/type.*Parser struct {/,/^}/ {
                /^}/ i\
	ExtractDates bool
            }' "$FILE"
        fi

        echo "  ✓ Added temporal support to $parser"
    else
        echo "  ⚠ File not found: $FILE"
    fi
done

echo "Done! All parsers updated with temporal support."