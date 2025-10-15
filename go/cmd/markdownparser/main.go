package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
)

func main() {
	var (
		inputFile        = flag.String("input", "", "Input markdown file path")
		outputFile       = flag.String("output", "", "Output file path (optional)")
		jsonOutput       = flag.Bool("json", false, "Output as JSON")
		docID            = flag.String("id", "", "Document ID")
		stdin            = flag.Bool("stdin", false, "Read from stdin")
		noFrontMatter    = flag.Bool("no-front-matter", false, "Disable front matter extraction")
		paragraphThresh  = flag.Int("paragraph-threshold", 1, "Minimum lines for paragraph")
		maxElements      = flag.Int("max-elements", 1000, "Maximum number of elements")
		noLinks          = flag.Bool("no-links", false, "Disable link extraction")
		noDates          = flag.Bool("no-dates", false, "Disable date extraction")
		noNumbers        = flag.Bool("no-numbers", false, "Disable number extraction")
		noWhitespace     = flag.Bool("no-whitespace", false, "Disable whitespace stripping")
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
	markdownParser := parser.NewMarkdownParser()

	// Configure parser options
	markdownParser.ExtractFrontMatter = !*noFrontMatter
	markdownParser.ParagraphThreshold = *paragraphThresh
	markdownParser.MaxElements = *maxElements
	markdownParser.EnableLinkExtraction = !*noLinks
	markdownParser.ExtractDates = !*noDates
	markdownParser.ExtractNumbers = !*noNumbers
	markdownParser.StripWhitespace = !*noWhitespace

	// Create request
	request := parser.MarkdownParseRequest{
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
	response, err := markdownParser.Parse(request)
	if err != nil {
		log.Fatalf("Error parsing markdown: %v", err)
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

func formatHumanReadable(response *parser.MarkdownParseResponse) string {
	var sb strings.Builder

	sb.WriteString("Markdown Parse Results\n")
	sb.WriteString("======================\n\n")

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
		if headerCount, exists := metadata["header_count"]; exists {
			sb.WriteString(fmt.Sprintf("  Headers: %v\n", headerCount))
		}
		if codeBlockCount, exists := metadata["code_block_count"]; exists {
			sb.WriteString(fmt.Sprintf("  Code Blocks: %v\n", codeBlockCount))
		}
		if listItemCount, exists := metadata["list_item_count"]; exists {
			sb.WriteString(fmt.Sprintf("  List Items: %v\n", listItemCount))
		}
		if hasFrontMatter, exists := metadata["has_front_matter"]; exists {
			sb.WriteString(fmt.Sprintf("  Has Front Matter: %v\n", hasFrontMatter))
		}
		if frontMatter, exists := metadata["front_matter"]; exists && frontMatter != nil {
			sb.WriteString("  Front Matter:\n")
			if fmMap, ok := frontMatter.(map[string]interface{}); ok {
				for key, value := range fmMap {
					sb.WriteString(fmt.Sprintf("    %s: %v\n", key, value))
				}
			}
		}
	}
	sb.WriteString("\n")

	// Elements summary
	sb.WriteString(fmt.Sprintf("Elements (%d):\n", len(response.Elements)))

	// Count elements by type
	elementTypes := make(map[parser.MarkdownElementType]int)
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

		// Show type-specific metadata
		switch element.ElementType {
		case parser.MarkdownElementTypeHeader:
			if level, ok := element.Metadata["level"]; ok {
				sb.WriteString(fmt.Sprintf("      Level: %v\n", level))
			}
		case parser.MarkdownElementTypeCodeBlock:
			if language, ok := element.Metadata["language"]; ok && language != "" {
				sb.WriteString(fmt.Sprintf("      Language: %v\n", language))
			}
		case parser.MarkdownElementTypeList:
			if itemCount, ok := element.Metadata["item_count"]; ok {
				sb.WriteString(fmt.Sprintf("      Items: %v\n", itemCount))
			}
		case parser.MarkdownElementTypeTable:
			if rowCount, ok := element.Metadata["row_count"]; ok {
				sb.WriteString(fmt.Sprintf("      Rows: %v\n", rowCount))
			}
			if colCount, ok := element.Metadata["column_count"]; ok {
				sb.WriteString(fmt.Sprintf("      Columns: %v\n", colCount))
			}
		case parser.MarkdownElementTypeFrontMatter:
			sb.WriteString("      Type: Front Matter\n")
		}

		// Show common metadata
		if lineNumber, ok := element.Metadata["line_number"]; ok {
			sb.WriteString(fmt.Sprintf("      Line: %v\n", lineNumber))
		}
		if length, ok := element.Metadata["length"]; ok {
			sb.WriteString(fmt.Sprintf("      Length: %v chars\n", length))
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
			linkText := ""
			if link.LinkText != "" {
				linkText = fmt.Sprintf(" (%s)", link.LinkText)
			}
			sb.WriteString(fmt.Sprintf("  [%d] %s: %s%s\n", i, link.LinkType, link.LinkTarget, linkText))
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