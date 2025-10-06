package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kennethstott/go-doc-go/internal/parser"
)

func main() {
	var configFile string
	var inputFile string
	var outputFile string
	var docID string
	var maxPreview int
	var extractDates bool
	var enableCaching bool

	flag.StringVar(&configFile, "config", "", "Configuration file (JSON)")
	flag.StringVar(&inputFile, "input", "", "Input file containing HTML content")
	flag.StringVar(&outputFile, "output", "", "Output file for parsed results (default: stdout)")
	flag.StringVar(&docID, "id", "", "Document ID (required)")
	flag.IntVar(&maxPreview, "max-preview", 100, "Maximum content preview length")
	flag.BoolVar(&extractDates, "extract-dates", true, "Extract dates from content")
	flag.BoolVar(&enableCaching, "enable-caching", true, "Enable caching")
	flag.Parse()

	// Validate required parameters
	if docID == "" {
		fmt.Fprintf(os.Stderr, "Error: Document ID is required (use -id flag)\n")
		os.Exit(1)
	}

	// Read input
	var content string
	if inputFile != "" {
		data, err := ioutil.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
		content = string(data)
	} else {
		// Read from stdin
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		content = string(data)
	}

	// Create parser with configuration
	htmlParser := parser.NewHTMLParser()
	htmlParser.MaxContentPreview = maxPreview
	htmlParser.ExtractDates = extractDates
	htmlParser.EnableCaching = enableCaching

	// Load configuration if provided
	var metadata map[string]interface{}
	if configFile != "" {
		configData, err := ioutil.ReadFile(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}

		var config map[string]interface{}
		if err := json.Unmarshal(configData, &config); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing config file: %v\n", err)
			os.Exit(1)
		}

		// Extract metadata and parser settings
		if meta, ok := config["metadata"].(map[string]interface{}); ok {
			metadata = meta
		}

		if preview, ok := config["max_content_preview"].(float64); ok {
			htmlParser.MaxContentPreview = int(preview)
		}

		if dates, ok := config["extract_dates"].(bool); ok {
			htmlParser.ExtractDates = dates
		}

		if caching, ok := config["enable_caching"].(bool); ok {
			htmlParser.EnableCaching = caching
		}
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Parse the document using new interface
	result, err := htmlParser.Parse(docID, content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing document: %v\n", err)
		os.Exit(1)
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to JSON: %v\n", err)
		os.Exit(1)
	}
	jsonOutput := string(jsonBytes)

	// Write output
	if outputFile != "" {
		err := ioutil.WriteFile(outputFile, []byte(jsonOutput), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(jsonOutput)
	}
}