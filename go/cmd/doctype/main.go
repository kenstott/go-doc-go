package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/detector"
)

func main() {
	var (
		filePath    = flag.String("path", "", "Path to the file to analyze")
		contentStr  = flag.String("content", "", "Base64 encoded content to analyze")
		metadataStr = flag.String("metadata", "", "JSON metadata for content hints")
		jsonOutput  = flag.Bool("json", true, "Output result as JSON")
		method      = flag.String("method", "auto", "Detection method: auto, path, mime, content")
	)
	flag.Parse()

	// Create detector
	det := detector.NewDocumentTypeDetector()

	var content []byte
	var metadata map[string]string
	var docType, detectionMethod string

	// Parse metadata if provided
	if *metadataStr != "" {
		if err := json.Unmarshal([]byte(*metadataStr), &metadata); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing metadata JSON: %v\n", err)
			os.Exit(1)
		}
	}

	// Handle content input
	if *contentStr != "" {
		// Decode base64 content
		decoded, err := base64.StdEncoding.DecodeString(*contentStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding base64 content: %v\n", err)
			os.Exit(1)
		}
		content = decoded
	} else if *filePath != "" {
		// Read file content
		fileContent, err := ioutil.ReadFile(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", *filePath, err)
			os.Exit(1)
		}
		content = fileContent
	}

	// Perform detection based on method
	switch *method {
	case "path":
		if *filePath == "" {
			fmt.Fprintf(os.Stderr, "Path method requires --path argument\n")
			os.Exit(1)
		}
		docType, detectionMethod = det.DetectFromPath(*filePath)

	case "mime":
		if *filePath == "" {
			fmt.Fprintf(os.Stderr, "MIME method requires --path argument\n")
			os.Exit(1)
		}
		docType, detectionMethod = det.DetectFromMime(*filePath)

	case "content":
		if content == nil {
			fmt.Fprintf(os.Stderr, "Content method requires --content or --path argument\n")
			os.Exit(1)
		}
		docType, detectionMethod = det.DetectFromContent(content, metadata)

	case "auto":
		docType, detectionMethod = det.Detect(*filePath, content, metadata)

	default:
		fmt.Fprintf(os.Stderr, "Unknown method: %s\n", *method)
		os.Exit(1)
	}

	// Output result
	if *jsonOutput {
		response := &detector.DetectionResponse{
			DocumentType: docType,
			Method:       detectionMethod,
		}
		jsonStr, err := response.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON response: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(jsonStr)
	} else {
		fmt.Println(docType)
	}
}

// For compatibility with the Python interface, also support positional arguments
func init() {
	// Check if we have positional arguments (legacy mode)
	if len(os.Args) >= 2 && !strings.HasPrefix(os.Args[1], "-") {
		// Legacy mode: first argument is the command
		command := os.Args[1]

		switch command {
		case "detect_from_path":
			if len(os.Args) >= 3 {
				handleLegacyDetectFromPath(os.Args[2])
			} else {
				fmt.Fprintf(os.Stderr, "Usage: %s detect_from_path <path>\n", os.Args[0])
				os.Exit(1)
			}

		case "detect_from_content":
			if len(os.Args) >= 3 {
				var metadata map[string]string
				if len(os.Args) >= 4 {
					json.Unmarshal([]byte(os.Args[3]), &metadata)
				}
				handleLegacyDetectFromContent(os.Args[2], metadata)
			} else {
				fmt.Fprintf(os.Stderr, "Usage: %s detect_from_content <base64_content> [metadata_json]\n", os.Args[0])
				os.Exit(1)
			}

		case "detect":
			path := ""
			contentStr := ""
			var metadata map[string]string

			if len(os.Args) >= 3 {
				path = os.Args[2]
			}
			if len(os.Args) >= 4 {
				contentStr = os.Args[3]
			}
			if len(os.Args) >= 5 {
				json.Unmarshal([]byte(os.Args[4]), &metadata)
			}

			handleLegacyDetect(path, contentStr, metadata)

		default:
			// Continue with normal flag parsing
			return
		}

		// Exit after handling legacy command
		os.Exit(0)
	}
}

func handleLegacyDetectFromPath(path string) {
	det := detector.NewDocumentTypeDetector()
	docType, method := det.DetectFromPath(path)

	response := &detector.DetectionResponse{
		DocumentType: docType,
		Method:       method,
	}
	jsonStr, _ := response.ToJSON()
	fmt.Println(jsonStr)
}

func handleLegacyDetectFromContent(contentStr string, metadata map[string]string) {
	det := detector.NewDocumentTypeDetector()

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding base64 content: %v\n", err)
		os.Exit(1)
	}

	docType, method := det.DetectFromContent(content, metadata)

	response := &detector.DetectionResponse{
		DocumentType: docType,
		Method:       method,
	}
	jsonStr, _ := response.ToJSON()
	fmt.Println(jsonStr)
}

func handleLegacyDetect(path, contentStr string, metadata map[string]string) {
	det := detector.NewDocumentTypeDetector()

	var content []byte
	if contentStr != "" {
		// Decode base64 content
		decoded, err := base64.StdEncoding.DecodeString(contentStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding base64 content: %v\n", err)
			os.Exit(1)
		}
		content = decoded
	} else if path != "" {
		// Read file content
		fileContent, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", path, err)
			os.Exit(1)
		}
		content = fileContent
	}

	docType, method := det.Detect(path, content, metadata)

	response := &detector.DetectionResponse{
		DocumentType: docType,
		Method:       method,
	}
	jsonStr, _ := response.ToJSON()
	fmt.Println(jsonStr)
}