package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kennethstott/go-doc-go/internal/parser"
)

func main() {
	// Define command line flags
	var (
		inputFile  = flag.String("input", "", "Input JSON file path")
		outputFile = flag.String("output", "", "Output file path (optional, defaults to stdout)")
		jsonOutput = flag.Bool("json", false, "Output as JSON")
		docID      = flag.String("id", "", "Document ID")
		help       = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	// Show help if requested or no input provided
	if *help || *inputFile == "" {
		showHelp()
		return
	}

	// Read input file
	content, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Use filename as ID if not provided
	id := *docID
	if id == "" {
		id = *inputFile
	}

	// Create parser
	jsonParser := parser.NewJSONParser()

	// Parse the JSON using new interface
	result, err := jsonParser.Parse(id, string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Format output
	var output string
	if *jsonOutput {
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting to JSON: %v\n", err)
			os.Exit(1)
		}
		output = string(jsonBytes)
	} else {
		// Human-readable format
		output = formatHumanReadable(result)
	}

	// Write output
	if *outputFile != "" {
		err := os.WriteFile(*outputFile, []byte(output), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Print(output)
	}
}

func showHelp() {
	fmt.Println("JSON Parser CLI")
	fmt.Println("Usage:")
	fmt.Println("  jsonparser -input <file> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -input <file>    Input JSON file path (required)")
	fmt.Println("  -output <file>   Output file path (optional, defaults to stdout)")
	fmt.Println("  -json            Output as JSON format")
	fmt.Println("  -id <string>     Document ID (optional, defaults to input filename)")
	fmt.Println("  -help            Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  jsonparser -input document.json")
	fmt.Println("  jsonparser -input document.json -json -output result.json")
	fmt.Println("  jsonparser -input document.json -id my-doc")
}

func formatHumanReadable(parseResult *parser.ParseResult) string {
	var result string

	// Document info
	result += fmt.Sprintf("Document ID: %s\n", parseResult.Document.ID)
	result += fmt.Sprintf("Document Type: %s\n", parseResult.Document.DocType)
	result += "\n"

	// Elements summary
	result += fmt.Sprintf("Elements: %d\n", len(parseResult.Elements))
	elementTypes := make(map[string]int)
	for _, element := range parseResult.Elements {
		elementTypes[element.ElementType]++
	}

	for elementType, count := range elementTypes {
		result += fmt.Sprintf("  %s: %d\n", elementType, count)
	}
	result += "\n"

	// Relationships
	result += fmt.Sprintf("Relationships: %d\n", len(parseResult.Relationships))
	relationshipTypes := make(map[string]int)
	for _, rel := range parseResult.Relationships {
		relationshipTypes[rel.RelationshipType]++
	}

	for relType, count := range relationshipTypes {
		result += fmt.Sprintf("  %s: %d\n", relType, count)
	}
	result += "\n"

	// Links
	result += fmt.Sprintf("Links: %d\n", len(parseResult.Links))
	linkTypes := make(map[string]int)
	for _, link := range parseResult.Links {
		linkTypes[link.LinkType]++
	}

	for linkType, count := range linkTypes {
		result += fmt.Sprintf("  %s: %d\n", linkType, count)
	}
	result += "\n"

	// Sample elements
	result += "Sample Elements:\n"
	for i, element := range parseResult.Elements {
		if i >= 5 { // Show first 5 elements
			result += "  ...\n"
			break
		}
		elemID := element.ElementID
		if len(elemID) > 8 {
			elemID = elemID[:8] + "..."
		}
		result += fmt.Sprintf("  [%d] %s (%s): %s\n",
			i+1,
			elemID,
			element.ElementType,
			element.ContentPreview)
	}

	return result
}