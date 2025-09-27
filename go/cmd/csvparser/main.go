package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/kennethstott/go-doc-go/internal/parser"
)

func main() {
	var (
		inputFile  = flag.String("input", "", "Input CSV file path")
		outputFile = flag.String("output", "", "Output file path (optional)")
		jsonOutput = flag.Bool("json", false, "Output as JSON")
		docID      = flag.String("id", "", "Document ID")
		stdin      = flag.Bool("stdin", false, "Read from stdin")
		delimiter  = flag.String("delimiter", ",", "CSV delimiter character")
		noHeader   = flag.Bool("no-header", false, "CSV has no header row")
		maxRows    = flag.Int("max-rows", 1000, "Maximum number of rows to process")
	)

	flag.Parse()

	// Validate inputs
	if !*stdin && *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: either -input or -stdin must be provided\n")
		flag.Usage()
		os.Exit(1)
	}

	// Read content
	var content []byte
	var err error

	if *stdin {
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Error reading from stdin: %v", err)
		}
	} else {
		content, err = os.ReadFile(*inputFile)
		if err != nil {
			log.Fatalf("Error reading file %s: %v", *inputFile, err)
		}
	}

	// Determine document ID
	documentID := *docID
	if documentID == "" {
		if *stdin {
			documentID = "stdin"
		} else {
			documentID = *inputFile
		}
	}

	// Create parser with configuration
	csvParser := parser.NewCSVParser()

	// Configure delimiter
	if len(*delimiter) > 0 {
		csvParser.Delimiter = rune((*delimiter)[0])
	}

	// Configure header extraction
	csvParser.ExtractHeader = !*noHeader

	// Configure max rows
	csvParser.MaxRows = *maxRows

	// Create request
	request := parser.CSVParseRequest{
		ID:      documentID,
		Content: string(content),
		Metadata: map[string]interface{}{
			"source": documentID,
		},
	}

	// Parse document
	response, err := csvParser.Parse(request)
	if err != nil {
		log.Fatalf("Error parsing CSV: %v", err)
	}

	// Format output
	var output string
	if *jsonOutput {
		output, err = response.ToJSON()
		if err != nil {
			log.Fatalf("Error converting to JSON: %v", err)
		}
	} else {
		output = formatHumanReadable(response)
	}

	// Write output
	if *outputFile != "" {
		err = os.WriteFile(*outputFile, []byte(output), 0644)
		if err != nil {
			log.Fatalf("Error writing output file: %v", err)
		}
	} else {
		fmt.Print(output)
	}
}

func formatHumanReadable(response *parser.CSVParseResponse) string {
	var sb strings.Builder

	sb.WriteString("CSV Parse Results\n")
	sb.WriteString("=================\n\n")

	// Document info
	sb.WriteString("Document:\n")
	sb.WriteString(fmt.Sprintf("  ID: %v\n", response.Document["doc_id"]))
	sb.WriteString(fmt.Sprintf("  Type: %v\n", response.Document["doc_type"]))

	if metadata, ok := response.Document["metadata"].(map[string]interface{}); ok {
		if rowCount, exists := metadata["row_count"]; exists {
			sb.WriteString(fmt.Sprintf("  Rows: %v\n", rowCount))
		}
		if columnCount, exists := metadata["column_count"]; exists {
			sb.WriteString(fmt.Sprintf("  Columns: %v\n", columnCount))
		}
		if hasHeader, exists := metadata["has_header"]; exists {
			sb.WriteString(fmt.Sprintf("  Has Header: %v\n", hasHeader))
		}
		if headers, exists := metadata["headers"]; exists {
			if headerList, ok := headers.([]string); ok {
				sb.WriteString(fmt.Sprintf("  Headers: %s\n", strings.Join(headerList, ", ")))
			}
		}
		if columnTypes, exists := metadata["column_types"]; exists {
			if typeList, ok := columnTypes.([]string); ok {
				sb.WriteString(fmt.Sprintf("  Column Types: %s\n", strings.Join(typeList, ", ")))
			}
		}
	}
	sb.WriteString("\n")

	// Elements summary
	sb.WriteString(fmt.Sprintf("Elements (%d):\n", len(response.Elements)))

	// Count elements by type
	elementTypes := make(map[parser.CSVElementType]int)
	for _, element := range response.Elements {
		elementTypes[element.ElementType]++
	}

	for elementType, count := range elementTypes {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", elementType, count))
	}
	sb.WriteString("\n")

	// Show sample elements
	sb.WriteString("Sample Elements:\n")
	elementCount := 0
	for _, element := range response.Elements {
		if elementCount >= 10 {
			sb.WriteString(fmt.Sprintf("  ... and %d more elements\n", len(response.Elements)-10))
			break
		}

		sb.WriteString(fmt.Sprintf("  [%d] %s (ID: %s)\n", elementCount, element.ElementType, element.ElementID))
		sb.WriteString(fmt.Sprintf("      Preview: %s\n", element.ContentPreview))

		if element.ParentID != "" {
			sb.WriteString(fmt.Sprintf("      Parent: %s\n", element.ParentID))
		}

		// Show metadata for cells
		if element.ElementType == parser.CSVElementTypeTableCell {
			if row, ok := element.Metadata["row"]; ok {
				if col, ok := element.Metadata["col"]; ok {
					sb.WriteString(fmt.Sprintf("      Position: Row %v, Col %v\n", row, col))
				}
			}
			if header, ok := element.Metadata["header"]; ok && header != "" {
				sb.WriteString(fmt.Sprintf("      Header: %v\n", header))
			}
		}

		elementCount++
	}
	sb.WriteString("\n")

	// Relationships
	if len(response.Relationships) > 0 {
		sb.WriteString(fmt.Sprintf("Relationships (%d):\n", len(response.Relationships)))
		for i, rel := range response.Relationships {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("  ... and %d more relationships\n", len(response.Relationships)-5))
				break
			}
			sb.WriteString(fmt.Sprintf("  [%d] %s -> %s (%s)\n", i,
				rel.SourceElementID[:8], rel.TargetElementID[:8], rel.RelationshipType))
		}
		sb.WriteString("\n")
	}

	// Links
	if len(response.Links) > 0 {
		sb.WriteString(fmt.Sprintf("Links (%d):\n", len(response.Links)))
		for i, link := range response.Links {
			if i >= 10 {
				sb.WriteString(fmt.Sprintf("  ... and %d more links\n", len(response.Links)-10))
				break
			}
			sb.WriteString(fmt.Sprintf("  [%d] %s: %s\n", i, link.LinkType, link.LinkTarget))
		}
		sb.WriteString("\n")
	}

	// Statistics
	sb.WriteString("Statistics:\n")
	sb.WriteString(fmt.Sprintf("  Total Elements: %d\n", len(response.Elements)))
	sb.WriteString(fmt.Sprintf("  Total Relationships: %d\n", len(response.Relationships)))
	sb.WriteString(fmt.Sprintf("  Total Links: %d\n", len(response.Links)))

	return sb.String()
}