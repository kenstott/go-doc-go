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
		inputFile     = flag.String("input", "", "Input text file path")
		outputFile    = flag.String("output", "", "Output file path (optional)")
		jsonOutput    = flag.Bool("json", false, "Output as JSON")
		docID         = flag.String("id", "", "Document ID")
		stdin         = flag.Bool("stdin", false, "Read from stdin")
		paragraphSep  = flag.String("paragraph-sep", "\n\n", "Paragraph separator")
		minLength     = flag.Int("min-length", 5, "Minimum paragraph length")
		maxElements   = flag.Int("max-elements", 1000, "Maximum number of elements")
		noLinks       = flag.Bool("no-links", false, "Disable link extraction")
		noDates       = flag.Bool("no-dates", false, "Disable date extraction")
		noNumbers     = flag.Bool("no-numbers", false, "Disable number extraction")
		noWhitespace  = flag.Bool("no-whitespace", false, "Disable whitespace stripping")
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
	textParser := parser.NewTextParser()

	// Configure paragraph separator
	textParser.ParagraphSeparator = *paragraphSep

	// Configure minimum paragraph length
	textParser.MinParagraphLength = *minLength

	// Configure maximum elements
	textParser.MaxElements = *maxElements

	// Configure feature toggles
	textParser.EnableLinkExtraction = !*noLinks
	textParser.ExtractDates = !*noDates
	textParser.ExtractNumbers = !*noNumbers
	textParser.StripWhitespace = !*noWhitespace

	// Create request
	request := parser.TextParseRequest{
		ID:      documentID,
		Content: string(content),
		Metadata: map[string]interface{}{
			"source": documentID,
		},
	}

	// Add filename to metadata if reading from file
	if *inputFile != "" {
		request.Metadata["filename"] = *inputFile
	}

	// Parse document
	response, err := textParser.Parse(request)
	if err != nil {
		log.Fatalf("Error parsing text: %v", err)
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

func formatHumanReadable(response *parser.TextParseResponse) string {
	var sb strings.Builder

	sb.WriteString("Text Parse Results\n")
	sb.WriteString("==================\n\n")

	// Document info
	sb.WriteString("Document:\n")
	sb.WriteString(fmt.Sprintf("  ID: %v\n", response.Document["doc_id"]))
	sb.WriteString(fmt.Sprintf("  Type: %v\n", response.Document["doc_type"]))

	if metadata, ok := response.Document["metadata"].(map[string]interface{}); ok {
		if charCount, exists := metadata["character_count"]; exists {
			sb.WriteString(fmt.Sprintf("  Characters: %v\n", charCount))
		}
		if wordCount, exists := metadata["word_count"]; exists {
			sb.WriteString(fmt.Sprintf("  Words: %v\n", wordCount))
		}
		if lineCount, exists := metadata["line_count"]; exists {
			sb.WriteString(fmt.Sprintf("  Lines: %v\n", lineCount))
		}
		if paragraphCount, exists := metadata["paragraph_count"]; exists {
			sb.WriteString(fmt.Sprintf("  Paragraphs: %v\n", paragraphCount))
		}
		if paragraphSep, exists := metadata["paragraph_separator"]; exists {
			sb.WriteString(fmt.Sprintf("  Paragraph Separator: %q\n", paragraphSep))
		}
	}
	sb.WriteString("\n")

	// Elements summary
	sb.WriteString(fmt.Sprintf("Elements (%d):\n", len(response.Elements)))

	// Count elements by type
	elementTypes := make(map[parser.TextElementType]int)
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

		// Show metadata for paragraphs
		if element.ElementType == parser.TextElementTypeParagraph {
			if paragraphIndex, ok := element.Metadata["paragraph_index"]; ok {
				sb.WriteString(fmt.Sprintf("      Index: %v\n", paragraphIndex))
			}
			if length, ok := element.Metadata["length"]; ok {
				sb.WriteString(fmt.Sprintf("      Length: %v chars\n", length))
			}
			if wordCount, ok := element.Metadata["word_count"]; ok {
				sb.WriteString(fmt.Sprintf("      Words: %v\n", wordCount))
			}
			if dates, ok := element.Metadata["dates"]; ok {
				if dateSlice, ok := dates.([]string); ok && len(dateSlice) > 0 {
					sb.WriteString(fmt.Sprintf("      Dates: %v\n", dateSlice))
				}
			}
			if numbers, ok := element.Metadata["numbers"]; ok {
				if numberSlice, ok := numbers.([]interface{}); ok && len(numberSlice) > 0 {
					sb.WriteString(fmt.Sprintf("      Numbers: %v\n", numberSlice))
				}
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