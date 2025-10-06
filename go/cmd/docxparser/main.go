package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kennethstott/go-doc-go/internal/parser"
)

func main() {
	var (
		inputFile       = flag.String("file", "", "Input DOCX file path")
		outputFile      = flag.String("output", "", "Output JSON file path (default: stdout)")
		verbose         = flag.Bool("verbose", false, "Enable verbose output")
		maxPreview      = flag.Int("max-preview", 100, "Maximum content preview length")
		extractHeaders  = flag.Bool("extract-headers", true, "Extract headers and footers")
		extractComments = flag.Bool("extract-comments", true, "Extract comments")
		extractStyles   = flag.Bool("extract-styles", true, "Extract style information")
		help            = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Read from stdin if no file specified
	var inputPath string
	if *inputFile == "" {
		if *verbose {
			log.Println("Reading DOCX file path from stdin...")
		}
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			inputPath = scanner.Text()
		} else {
			log.Fatal("No input provided")
		}
	} else {
		inputPath = *inputFile
	}

	// Validate input file
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		log.Fatalf("Input file does not exist: %s", inputPath)
	}

	// Get absolute path
	absPath, err := filepath.Abs(inputPath)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v", err)
	}

	if *verbose {
		log.Printf("Processing DOCX file: %s", absPath)
	}

	// Create parser with configuration
	docxParser := parser.NewDocxParser()
	docxParser.MaxContentPreview = *maxPreview
	docxParser.ExtractHeadersFooters = *extractHeaders
	docxParser.ExtractComments = *extractComments
	docxParser.ExtractStyles = *extractStyles

	// Create parse request
	request := parser.DocxParseRequest{
		ID:      absPath,
		Content: absPath, // For DOCX, content is the file path
		Metadata: map[string]interface{}{
			"filename":        filepath.Base(absPath),
			"file_extension":  filepath.Ext(absPath),
			"absolute_path":   absPath,
			"parser_version":  "go-1.0.0",
			"max_preview":     *maxPreview,
			"extract_headers": *extractHeaders,
			"extract_comments": *extractComments,
			"extract_styles":  *extractStyles,
		},
	}

	// Parse the document
	if *verbose {
		log.Println("Parsing DOCX document...")
	}

	result, err := docxParser.Parse(request)
	if err != nil {
		log.Fatalf("Error parsing DOCX: %v", err)
	}

	if *verbose {
		log.Printf("Successfully parsed DOCX document:")
		log.Printf("  - Elements: %d", len(result.Elements))
		log.Printf("  - Relationships: %d", len(result.Relationships))
		log.Printf("  - Links: %d", len(result.Links))
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling to JSON: %v", err)
	}

	// Output result
	if *outputFile != "" {
		if *verbose {
			log.Printf("Writing output to: %s", *outputFile)
		}
		err = os.WriteFile(*outputFile, jsonData, 0644)
		if err != nil {
			log.Fatalf("Error writing output file: %v", err)
		}
	} else {
		fmt.Println(string(jsonData))
	}

	if *verbose {
		log.Println("DOCX parsing completed successfully")
	}
}

func showHelp() {
	fmt.Println("DOCX Parser - Go implementation")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  docxparser [OPTIONS]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Parse a DOCX file and output to stdout")
	fmt.Println("  docxparser -file document.docx")
	fmt.Println()
	fmt.Println("  # Parse with custom settings and save to file")
	fmt.Println("  docxparser -file document.docx -output result.json -max-preview 200")
	fmt.Println()
	fmt.Println("  # Parse from stdin (for pipeline usage)")
	fmt.Println("  echo 'document.docx' | docxparser -verbose")
	fmt.Println()
	fmt.Println("  # Disable header/footer extraction")
	fmt.Println("  docxparser -file document.docx -extract-headers=false")
	fmt.Println()
	fmt.Println("OUTPUT:")
	fmt.Println("  The parser outputs a JSON structure containing:")
	fmt.Println("  - document: Document metadata and properties")
	fmt.Println("  - elements: Structured document elements (paragraphs, headers, tables, etc.)")
	fmt.Println("  - relationships: Hierarchical relationships between elements")
	fmt.Println("  - links: Extracted hyperlinks and references")
	fmt.Println()
	fmt.Println("ELEMENT TYPES:")
	fmt.Println("  - document_root: Root document element")
	fmt.Println("  - body: Document body container")
	fmt.Println("  - header: Document headers (H1-H6 equivalent)")
	fmt.Println("  - paragraph: Text paragraphs")
	fmt.Println("  - table: Table structures")
	fmt.Println("  - table_row: Table rows")
	fmt.Println("  - table_cell: Table data cells")
	fmt.Println("  - table_header: Table header cells")
	fmt.Println("  - page_header: Document headers")
	fmt.Println("  - page_footer: Document footers")
	fmt.Println("  - comment: Document comments")
}