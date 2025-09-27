package main

import (
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
		inputFile         = flag.String("input", "", "Input XML file path")
		docID             = flag.String("id", "", "Document ID")
		jsonOutput        = flag.Bool("json", false, "Output in JSON format")
		stdinMode         = flag.Bool("stdin", false, "Read from stdin")
		maxPreview        = flag.Int("max-preview", 100, "Maximum content preview length")
		extractAttrs      = flag.Bool("extract-attrs", true, "Extract XML attributes")
		flattenNamespaces = flag.Bool("flatten-namespaces", true, "Flatten XML namespaces")
		extractNamespaces = flag.Bool("extract-namespaces", true, "Extract namespace declarations")
		maxDepth          = flag.Int("max-depth", 20, "Maximum parsing depth")
		enableCaching     = flag.Bool("enable-caching", true, "Enable content caching")
		verbose           = flag.Bool("verbose", false, "Verbose output")
	)

	flag.Parse()

	// Validate inputs
	if !*stdinMode && *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Either -input or -stdin must be specified\n")
		flag.Usage()
		os.Exit(1)
	}

	// Read content
	var content string
	var err error

	if *stdinMode {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		content = string(data)
	} else {
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", *inputFile, err)
			os.Exit(1)
		}
		content = string(data)
	}

	// Set default document ID if not provided
	if *docID == "" {
		if *inputFile != "" {
			*docID = *inputFile
		} else {
			*docID = "stdin_xml"
		}
	}

	// Create parser with configuration
	xmlParser := parser.NewXMLParser()
	xmlParser.MaxContentPreview = *maxPreview
	xmlParser.ExtractAttributes = *extractAttrs
	xmlParser.FlattenNamespaces = *flattenNamespaces
	xmlParser.ExtractNamespaces = *extractNamespaces
	xmlParser.MaxDepth = *maxDepth
	xmlParser.EnableCaching = *enableCaching

	// Create parse request
	request := parser.XMLParseRequest{
		ID:      *docID,
		Content: content,
		Metadata: map[string]interface{}{
			"source":   *inputFile,
			"filename": *inputFile,
		},
	}

	// Parse XML
	result, err := xmlParser.Parse(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if *jsonOutput {
		jsonStr, err := result.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(jsonStr)
	} else {
		printHumanReadable(result, *verbose)
	}
}

func printHumanReadable(result *parser.XMLParseResponse, verbose bool) {
	fmt.Printf("XML Document Parsing Results\n")
	fmt.Printf("============================\n\n")

	// Document info
	fmt.Printf("Document ID: %v\n", result.Document["doc_id"])
	fmt.Printf("Document Type: %v\n", result.Document["doc_type"])
	fmt.Printf("Content Hash: %v\n", result.Document["content_hash"])

	// Show namespaces if present
	if namespaces, ok := result.Document["namespaces"].(map[string]string); ok && len(namespaces) > 0 {
		fmt.Printf("\nNamespaces:\n")
		for prefix, uri := range namespaces {
			if prefix == "" {
				fmt.Printf("  (default): %s\n", uri)
			} else {
				fmt.Printf("  %s: %s\n", prefix, uri)
			}
		}
	}

	fmt.Printf("\nElements: %d\n", len(result.Elements))
	fmt.Printf("Relationships: %d\n", len(result.Relationships))
	fmt.Printf("Links: %d\n", len(result.Links))

	if verbose {
		// Show element hierarchy
		fmt.Printf("\nElement Hierarchy:\n")
		fmt.Printf("==================\n")
		printElementTree(result.Elements, "", "")

		// Show relationships
		if len(result.Relationships) > 0 {
			fmt.Printf("\nRelationships:\n")
			fmt.Printf("==============\n")
			for _, rel := range result.Relationships {
				fmt.Printf("  %s -%s-> %s\n",
					truncateID(rel.SourceElementID),
					rel.RelationshipType,
					truncateID(rel.TargetElementID))
			}
		}

		// Show links
		if len(result.Links) > 0 {
			fmt.Printf("\nExtracted Links:\n")
			fmt.Printf("================\n")
			for _, link := range result.Links {
				fmt.Printf("  [%s] %s -> %s\n", link.LinkType, link.LinkText, link.LinkTarget)
			}
		}
	} else {
		// Show summary of elements by type
		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[string(elem.ElementType)]++
		}

		fmt.Printf("\nElement Types:\n")
		for elemType, count := range elementTypes {
			fmt.Printf("  %s: %d\n", elemType, count)
		}
	}
}

func printElementTree(elements []parser.XMLElement, parentID string, indent string) {
	for _, elem := range elements {
		if elem.ParentID == parentID {
			typeStr := string(elem.ElementType)
			preview := elem.ContentPreview
			if len(preview) > 50 {
				preview = preview[:47] + "..."
			}

			// Show element info
			fmt.Printf("%s[%s] %s: %s\n", indent, typeStr, truncateID(elem.ElementID), preview)

			// Show metadata if interesting
			if tagName, ok := elem.Metadata["tag_name"].(string); ok && tagName != "" {
				fmt.Printf("%s    Tag: %s\n", indent, tagName)
			}
			if xmlPath, ok := elem.Metadata["xml_path"].(string); ok && xmlPath != "" {
				fmt.Printf("%s    Path: %s\n", indent, xmlPath)
			}
			if attributes, ok := elem.Metadata["attributes"].(map[string]string); ok && len(attributes) > 0 {
				attrStrs := make([]string, 0, len(attributes))
				for name, value := range attributes {
					attrStrs = append(attrStrs, fmt.Sprintf("%s=\"%s\"", name, value))
				}
				fmt.Printf("%s    Attrs: %s\n", indent, strings.Join(attrStrs, " "))
			}

			// Recursively print children
			printElementTree(elements, elem.ElementID, indent+"  ")
		}
	}
}

func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8] + "..."
	}
	return id
}
