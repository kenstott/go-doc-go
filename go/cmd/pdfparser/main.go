package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kennethstott/go-doc-go/internal/parser"
)

func main() {
	// Define flags
	var (
		inputFile    = flag.String("input", "", "Input PDF file path")
		docID        = flag.String("id", "", "Document ID")
		jsonOutput   = flag.Bool("json", false, "Output in JSON format")
		maxPreview   = flag.Int("max-preview", 100, "Maximum content preview length")
		maxPages     = flag.Int("max-pages", 1000, "Maximum pages to parse")
		extractLinks = flag.Bool("extract-links", true, "Extract links from text")
		verbose      = flag.Bool("verbose", false, "Verbose output")
		stdinMode    = flag.Bool("stdin", false, "Read PDF from stdin")
	)

	flag.Parse()

	// Validate inputs
	if !*stdinMode && *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Either -input or -stdin must be specified\n")
		flag.Usage()
		os.Exit(1)
	}

	// Set default document ID if not provided
	if *docID == "" {
		if *inputFile != "" {
			*docID = *inputFile
		} else {
			*docID = "stdin_pdf"
		}
	}

	// Create parser with configuration
	pdfParser := parser.NewPDFParser()
	pdfParser.MaxContentPreview = *maxPreview
	pdfParser.MaxPages = *maxPages
	pdfParser.ExtractLinks = *extractLinks

	// Parse PDF
	var result *parser.ParseResult
	var err error

	if *stdinMode {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		result, err = pdfParser.Parse(*docID, data)
	} else {
		// Parse from file path
		result, err = pdfParser.Parse(*docID, *inputFile)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing PDF: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if *jsonOutput {
		// Convert to JSON
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
	} else {
		printHumanReadable(result, *verbose)
	}
}

func printHumanReadable(result *parser.ParseResult, verbose bool) {
	fmt.Printf("PDF Document Parsing Results\n")
	fmt.Printf("=============================\n\n")

	// Document info
	fmt.Printf("Document ID: %s\n", result.Document.ID)
	fmt.Printf("Document Type: %s\n", result.Document.DocType)
	if result.Document.Title != "" {
		fmt.Printf("Title: %s\n", result.Document.Title)
	}

	// Metadata
	if result.Document.Metadata != nil {
		fmt.Printf("\nMetadata:\n")
		for key, value := range result.Document.Metadata {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	fmt.Printf("\nStructure:\n")
	fmt.Printf("  Pages: %d\n", countElementsByType(result.Elements, "page"))
	fmt.Printf("  Headers: %d\n", countElementsByType(result.Elements, "header"))
	fmt.Printf("  Paragraphs: %d\n", countElementsByType(result.Elements, "paragraph"))
	fmt.Printf("  Tables: %d\n", countElementsByType(result.Elements, "table"))
	fmt.Printf("  Lists: %d\n", countElementsByType(result.Elements, "list_item"))
	fmt.Printf("  Footnotes: %d\n", countElementsByType(result.Elements, "footnote"))
	fmt.Printf("  Total Elements: %d\n", len(result.Elements))
	fmt.Printf("  Relationships: %d\n", len(result.Relationships))
	fmt.Printf("  Links Extracted: %d\n", len(result.Links))

	if verbose {
		// Show element hierarchy
		fmt.Printf("\nDocument Structure:\n")
		fmt.Printf("===================\n")
		printElementTree(result.Elements, "", "", 0)

		// Show extracted links
		if len(result.Links) > 0 {
			fmt.Printf("\nExtracted Links:\n")
			fmt.Printf("================\n")
			for i, link := range result.Links {
				if i >= 10 && !verbose {
					fmt.Printf("  ... and %d more links\n", len(result.Links)-10)
					break
				}
				fmt.Printf("  [%s] %s\n", link.LinkType, link.LinkTarget)
			}
		}
	} else {
		// Show sample content
		fmt.Printf("\nSample Content:\n")
		fmt.Printf("===============\n")

		// Show first few headers
		headerCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "header" && headerCount < 5 {
				fmt.Printf("  Header: %s\n", elem.ContentPreview)
				headerCount++
			}
		}

		// Show first paragraph
		for _, elem := range result.Elements {
			if elem.ElementType == "paragraph" {
				fmt.Printf("\n  First Paragraph:\n    %s\n", elem.ContentPreview)
				break
			}
		}
	}
}

func countElementsByType(elements []parser.Element, elemType string) int {
	count := 0
	for _, elem := range elements {
		if elem.ElementType == elemType {
			count++
		}
	}
	return count
}

func printElementTree(elements []parser.Element, parentID string, indent string, depth int) {
	if depth > 10 {
		return // Prevent infinite recursion
	}

	for _, elem := range elements {
		if elem.ParentID == parentID {
			// Format element display
			typeStr := strings.ToUpper(elem.ElementType[:1]) + elem.ElementType[1:]
			preview := elem.ContentPreview
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}

			// Print element
			if elem.ElementType == "page" {
				fmt.Printf("%s[%s] %s\n", indent, typeStr, preview)
			} else {
				fmt.Printf("%s├─ [%s] %s\n", indent, typeStr, preview)
			}

			// Print children
			nextIndent := indent
			if elem.ElementType == "page" || elem.ElementType == "body" {
				nextIndent += "  "
			} else {
				nextIndent += "│  "
			}
			printElementTree(elements, elem.ElementID, nextIndent, depth+1)
		}
	}
}