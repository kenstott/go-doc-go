package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kennethstott/go-doc-go/internal/parser"
)

func main() {
	var (
		inputFile        = flag.String("file", "", "Input Parquet file path")
		outputFile       = flag.String("output", "", "Output JSON file path (default: stdout)")
		verbose          = flag.Bool("verbose", false, "Enable verbose output")
		maxPreview       = flag.Int("max-preview", 100, "Maximum content preview length")
		textColumn       = flag.String("text-column", "", "Column to use for text content (auto-detect if empty)")
		groupByColumn    = flag.String("group-by", "", "Column to group rows by (optional)")
		metadataColumns  = flag.String("metadata-columns", "", "Comma-separated list of columns to extract as document metadata")
		help             = flag.Bool("help", false, "Show help message")
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
			log.Println("Reading Parquet file path from stdin...")
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
		log.Printf("Processing Parquet file: %s", absPath)
	}

	// Create parser with configuration
	parquetParser := parser.NewParquetParser()
	parquetParser.MaxContentPreview = *maxPreview
	parquetParser.TextColumn = *textColumn
	parquetParser.GroupByColumn = *groupByColumn

	// Parse metadata columns
	if *metadataColumns != "" {
		cols := strings.Split(*metadataColumns, ",")
		for i := range cols {
			cols[i] = strings.TrimSpace(cols[i])
		}
		parquetParser.MetadataColumns = cols
	}

	// Create parse request
	request := parser.ParquetParseRequest{
		ID:      absPath,
		Content: absPath, // For Parquet, content is the file path
		Metadata: map[string]interface{}{
			"filename":        filepath.Base(absPath),
			"file_extension":  filepath.Ext(absPath),
			"absolute_path":   absPath,
			"parser_version":  "go-1.0.0",
			"max_preview":     *maxPreview,
			"text_column":     *textColumn,
			"group_by_column": *groupByColumn,
		},
	}

	// Parse the document
	if *verbose {
		log.Println("Parsing Parquet file...")
		if *textColumn != "" {
			log.Printf("  - Text column: %s", *textColumn)
		} else {
			log.Println("  - Text column: auto-detect")
		}
		if *groupByColumn != "" {
			log.Printf("  - Group by column: %s", *groupByColumn)
		}
		if len(parquetParser.MetadataColumns) > 0 {
			log.Printf("  - Metadata columns: %v", parquetParser.MetadataColumns)
		}
	}

	result, err := parquetParser.Parse(request)
	if err != nil {
		log.Fatalf("Error parsing Parquet: %v", err)
	}

	if *verbose {
		log.Printf("Successfully parsed Parquet file:")
		log.Printf("  - Elements: %d", len(result.Elements))
		log.Printf("  - Relationships: %d", len(result.Relationships))

		if metadata, ok := result.Document["metadata"].(map[string]interface{}); ok {
			if rowCount, ok := metadata["row_count"].(int64); ok {
				log.Printf("  - Rows: %d", rowCount)
			}
			if columnCount, ok := metadata["column_count"].(int); ok {
				log.Printf("  - Columns: %d", columnCount)
			}
			if groupCount, ok := metadata["group_count"].(int); ok {
				log.Printf("  - Groups: %d", groupCount)
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
		log.Println("Parquet parsing completed successfully")
	}
}

func showHelp() {
	fmt.Println("Parquet Parser - Go implementation for generic tabular data")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  parquetparser [OPTIONS]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Parse a Parquet file and output to stdout")
	fmt.Println("  parquetparser -file data.parquet")
	fmt.Println()
	fmt.Println("  # Parse with specific text column and save to file")
	fmt.Println("  parquetparser -file data.parquet -text-column content -output result.json")
	fmt.Println()
	fmt.Println("  # Parse with grouping and metadata extraction")
	fmt.Println("  parquetparser -file data.parquet -group-by category -metadata-columns \"id,date,author\"")
	fmt.Println()
	fmt.Println("  # Parse from stdin (for pipeline usage)")
	fmt.Println("  echo 'data.parquet' | parquetparser -verbose")
	fmt.Println()
	fmt.Println("OUTPUT:")
	fmt.Println("  The parser outputs a JSON structure containing:")
	fmt.Println("  - document: Document metadata and properties")
	fmt.Println("  - elements: Structured elements (rows as paragraphs)")
	fmt.Println("  - relationships: Hierarchical relationships between elements")
	fmt.Println()
	fmt.Println("ELEMENT STRUCTURE:")
	fmt.Println("  - root: Top-level container")
	fmt.Println("  - body: Document body")
	fmt.Println("  - header: Group headers (when using -group-by)")
	fmt.Println("  - paragraph: Individual data rows")
	fmt.Println()
	fmt.Println("METADATA:")
	fmt.Println("  Each row's data is preserved in the element's metadata field.")
	fmt.Println("  Document-level metadata can be extracted from the first row using -metadata-columns.")
}