// Main application demonstrating parser package usage.
package main

import (
	"fmt"
	"strings"

	"parser"
)

func main() {
	fmt.Println("Parser Demo")
	fmt.Println(strings.Repeat("=", 40))

	// Sample input text
	input := `Hello World 123
This is a "test" message
Calculate 10 + 20 = 30
End of input`

	// Validate input
	if err := parser.Validate(input); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}
	fmt.Println("Input validation: PASSED")

	// Tokenize the input
	tokens := parser.Tokenize(input)
	fmt.Printf("\nTokenization results (%d tokens):\n", len(tokens))
	for i, token := range tokens {
		if i < 10 { // Show first 10 tokens
			fmt.Printf("  Token %d: Type=%-12s Value=%-15s Line=%d Col=%d\n",
				i+1, token.Type, token.Value, token.Line, token.Col)
		}
	}
	if len(tokens) > 10 {
		fmt.Printf("  ... and %d more tokens\n", len(tokens)-10)
	}

	// Count words and lines
	wordCount := parser.CountWords(input)
	lineCount := parser.CountLines(input)
	fmt.Printf("\nStatistics:\n")
	fmt.Printf("  Word count: %d\n", wordCount)
	fmt.Printf("  Line count: %d\n", lineCount)

	// Find pattern occurrences
	pattern := "test"
	positions := parser.FindPattern(input, pattern)
	fmt.Printf("\nPattern search for '%s':\n", pattern)
	if len(positions) > 0 {
		fmt.Printf("  Found %d occurrence(s) at positions: %v\n", len(positions), positions)
	} else {
		fmt.Printf("  Pattern not found\n")
	}

	// Search for numbers
	numberPattern := "10"
	numberPositions := parser.FindPattern(input, numberPattern)
	fmt.Printf("\nPattern search for '%s':\n", numberPattern)
	if len(numberPositions) > 0 {
		fmt.Printf("  Found %d occurrence(s) at positions: %v\n", len(numberPositions), numberPositions)
	} else {
		fmt.Printf("  Pattern not found\n")
	}

	fmt.Println("\nParsing complete!")
}
