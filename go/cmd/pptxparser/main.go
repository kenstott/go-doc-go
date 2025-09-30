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
		inputFile       = flag.String("file", "", "Input PPTX file path")
		outputFile      = flag.String("output", "", "Output JSON file path (default: stdout)")
		verbose         = flag.Bool("verbose", false, "Enable verbose output")
		maxPreview      = flag.Int("max-preview", 100, "Maximum content preview length")
		extractNotes    = flag.Bool("extract-notes", true, "Extract slide notes")
		extractComments = flag.Bool("extract-comments", true, "Extract comments")
		extractShapes   = flag.Bool("extract-shapes", true, "Extract shape information")
		extractTables   = flag.Bool("extract-tables", true, "Extract table content")
		extractImages   = flag.Bool("extract-images", true, "Extract image information")
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
			log.Println("Reading PPTX file path from stdin...")
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
		log.Printf("Processing PPTX file: %s", absPath)
	}

	// Create parser with configuration
	pptxParser := parser.NewPptxParser()
	pptxParser.MaxContentPreview = *maxPreview
	pptxParser.ExtractNotes = *extractNotes
	pptxParser.ExtractComments = *extractComments
	pptxParser.ExtractShapes = *extractShapes
	pptxParser.ExtractTables = *extractTables
	pptxParser.ExtractImages = *extractImages

	// Create parse request
	request := parser.PptxParseRequest{
		ID:      absPath,
		Content: absPath, // For PPTX, content is the file path
		Metadata: map[string]interface{}{
			"filename":         filepath.Base(absPath),
			"file_extension":   filepath.Ext(absPath),
			"absolute_path":    absPath,
			"parser_version":   "go-1.0.0",
			"max_preview":      *maxPreview,
			"extract_notes":    *extractNotes,
			"extract_comments": *extractComments,
			"extract_shapes":   *extractShapes,
			"extract_tables":   *extractTables,
			"extract_images":   *extractImages,
		},
	}

	// Parse the document
	if *verbose {
		log.Println("Parsing PPTX presentation...")
	}

	result, err := pptxParser.Parse(request)
	if err != nil {
		log.Fatalf("Error parsing PPTX: %v", err)
	}

	if *verbose {
		log.Printf("Successfully parsed PPTX presentation:")
		log.Printf("  - Elements: %d", len(result.Elements))
		log.Printf("  - Relationships: %d", len(result.Relationships))
		log.Printf("  - Links: %d", len(result.Links))
		if metadata, ok := result.Document["metadata"].(map[string]interface{}); ok {
			if slideCount, ok := metadata["slide_count"].(int); ok {
				log.Printf("  - Slides: %d", slideCount)
			}
		}
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
		log.Println("PPTX parsing completed successfully")
	}
}

func showHelp() {
	fmt.Println("PPTX Parser - Go implementation")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  pptxparser [OPTIONS]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Parse a PPTX file and output to stdout")
	fmt.Println("  pptxparser -file presentation.pptx")
	fmt.Println()
	fmt.Println("  # Parse with custom settings and save to file")
	fmt.Println("  pptxparser -file presentation.pptx -output result.json -max-preview 200")
	fmt.Println()
	fmt.Println("  # Parse from stdin (for pipeline usage)")
	fmt.Println("  echo 'presentation.pptx' | pptxparser -verbose")
	fmt.Println()
	fmt.Println("  # Parse without extracting notes and shapes")
	fmt.Println("  pptxparser -file presentation.pptx -extract-notes=false -extract-shapes=false")
	fmt.Println()
	fmt.Println("OUTPUT:")
	fmt.Println("  The parser outputs a JSON structure containing:")
	fmt.Println("  - document: Document metadata and properties")
	fmt.Println("  - elements: Structured presentation elements (slides, shapes, text, etc.)")
	fmt.Println("  - relationships: Hierarchical relationships between elements")
	fmt.Println("  - links: Extracted hyperlinks and references")
	fmt.Println()
	fmt.Println("ELEMENT TYPES:")
	fmt.Println("  - presentation_root: Root presentation element")
	fmt.Println("  - presentation_body: Presentation body container")
	fmt.Println("  - slide: Individual slide")
	fmt.Println("  - title: Slide title text")
	fmt.Println("  - subtitle: Slide subtitle text")
	fmt.Println("  - text_box: Text content shape")
	fmt.Println("  - table: Table structure")
	fmt.Println("  - table_cell: Table cell content")
	fmt.Println("  - slide_notes: Speaker notes for a slide")
	fmt.Println("  - comment: Slide comments")
	fmt.Println("  - image: Image placeholder")
	fmt.Println("  - chart: Chart element")
}