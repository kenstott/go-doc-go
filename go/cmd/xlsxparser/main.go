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
		inputFile       = flag.String("input", "", "Input XLSX file path")
		docID           = flag.String("id", "", "Document ID")
		jsonOutput      = flag.Bool("json", false, "Output in JSON format")
		maxPreview      = flag.Int("max-preview", 100, "Maximum content preview length")
		maxRows         = flag.Int("max-rows", 10000, "Maximum rows to parse per sheet")
		maxCols         = flag.Int("max-cols", 100, "Maximum columns to parse per sheet")
		detectTables    = flag.Bool("detect-tables", true, "Detect table regions in worksheets")
		minTableRows    = flag.Int("min-table-rows", 2, "Minimum rows for table detection")
		minTableCols    = flag.Int("min-table-cols", 2, "Minimum columns for table detection")
		extractComments = flag.Bool("extract-comments", true, "Extract cell comments")
		extractFormulas = flag.Bool("extract-formulas", true, "Extract cell formulas")
		extractLinks    = flag.Bool("extract-links", true, "Extract hyperlinks")
		verbose         = flag.Bool("verbose", false, "Verbose output")
		stdinMode       = flag.Bool("stdin", false, "Read XLSX from stdin")
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
			*docID = "stdin_xlsx"
		}
	}

	// Create parser with configuration
	xlsxParser := parser.NewXLSXParser()
	xlsxParser.MaxContentPreview = *maxPreview
	xlsxParser.MaxRows = *maxRows
	xlsxParser.MaxCols = *maxCols
	xlsxParser.DetectTables = *detectTables
	xlsxParser.MinTableRows = *minTableRows
	xlsxParser.MinTableCols = *minTableCols
	xlsxParser.ExtractComments = *extractComments
	xlsxParser.ExtractFormulas = *extractFormulas
	xlsxParser.ExtractLinks = *extractLinks

	// Parse XLSX
	var result *parser.ParseResult
	var err error

	if *stdinMode {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		result, err = xlsxParser.Parse(*docID, data)
	} else {
		// Parse from file path
		result, err = xlsxParser.Parse(*docID, *inputFile)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing XLSX: %v\n", err)
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
	fmt.Printf("XLSX Document Parsing Results\n")
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
	fmt.Printf("  Worksheets: %d\n", countElementsByType(result.Elements, "worksheet"))
	fmt.Printf("  Tables: %d\n", countElementsByType(result.Elements, "table"))
	fmt.Printf("  Table Headers: %d\n", countElementsByType(result.Elements, "table_header"))
	fmt.Printf("  Table Cells: %d\n", countElementsByType(result.Elements, "table_cell"))
	fmt.Printf("  Merged Cells: %d\n", countElementsByType(result.Elements, "merged_cell"))
	fmt.Printf("  Comments: %d\n", countElementsByType(result.Elements, "comment"))
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

		// Show table information
		tables := filterElementsByType(result.Elements, "table")
		if len(tables) > 0 {
			fmt.Printf("\nTable Details:\n")
			fmt.Printf("==============\n")
			for i, table := range tables {
				if i >= 5 {
					fmt.Printf("  ... and %d more tables\n", len(tables)-5)
					break
				}
				location := table.ContentLocation
				fmt.Printf("  Table %d: %s\n", i+1, table.ContentPreview)
				if location != nil {
					if sheet, ok := location["sheet_name"]; ok {
						fmt.Printf("    Sheet: %v\n", sheet)
					}
					if region, ok := location["cell_range"]; ok {
						fmt.Printf("    Range: %v\n", region)
					}
				}
			}
		}
	} else {
		// Show sample content
		fmt.Printf("\nSample Content:\n")
		fmt.Printf("===============\n")

		// Show worksheets
		worksheets := filterElementsByType(result.Elements, "worksheet")
		if len(worksheets) > 0 {
			fmt.Printf("  Worksheets:\n")
			for i, sheet := range worksheets {
				if i >= 5 {
					fmt.Printf("    ... and %d more sheets\n", len(worksheets)-5)
					break
				}
				fmt.Printf("    - %s\n", sheet.ContentPreview)
			}
		}

		// Show first table
		tables := filterElementsByType(result.Elements, "table")
		if len(tables) > 0 {
			fmt.Printf("\n  First Table:\n")
			fmt.Printf("    %s\n", tables[0].ContentPreview)

			// Show table headers
			tableHeaders := getChildElementsByType(result.Elements, tables[0].ElementID, "table_header")
			if len(tableHeaders) > 0 {
				fmt.Printf("    Headers: ")
				for i, header := range tableHeaders {
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%s", header.ContentPreview)
					if i >= 4 {
						fmt.Printf("...")
						break
					}
				}
				fmt.Printf("\n")
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

func filterElementsByType(elements []parser.Element, elemType string) []parser.Element {
	var filtered []parser.Element
	for _, elem := range elements {
		if elem.ElementType == elemType {
			filtered = append(filtered, elem)
		}
	}
	return filtered
}

func getChildElementsByType(elements []parser.Element, parentID string, elemType string) []parser.Element {
	var children []parser.Element
	for _, elem := range elements {
		if elem.ParentID == parentID && elem.ElementType == elemType {
			children = append(children, elem)
		}
	}
	return children
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
			if elem.ElementType == "worksheet" {
				fmt.Printf("%s[%s] %s\n", indent, typeStr, preview)
			} else if elem.ElementType == "table" {
				fmt.Printf("%s├─ [%s] %s\n", indent, typeStr, preview)
			} else {
				fmt.Printf("%s├─ [%s] %s\n", indent, typeStr, preview)
			}

			// Print children
			nextIndent := indent
			if elem.ElementType == "worksheet" || elem.ElementType == "body" {
				nextIndent += "  "
			} else {
				nextIndent += "│  "
			}
			printElementTree(elements, elem.ElementID, nextIndent, depth+1)
		}
	}
}