package parser

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PDFPythonShimParser uses Python PdfParser via shim for full functionality
type PDFPythonShimParser struct {
	PythonPath string
	ShimPath   string
}

// NewPDFPythonShimParser creates a new Python-based PDF parser
func NewPDFPythonShimParser() *PDFPythonShimParser {
	// Get paths from environment or use defaults
	pythonPath := os.Getenv("PYTHON_PATH")
	if pythonPath == "" {
		pythonPath = "python3"
	}

	shimPath := os.Getenv("PARSER_SHIM_PATH")
	if shimPath == "" {
		// Default to src/go_doc_go/parser_shim.py relative to project root
		// Worker runs from go/ directory, so go up one level
		shimPath = filepath.Join("..", "src", "go_doc_go", "parser_shim.py")
	}

	return &PDFPythonShimParser{
		PythonPath: pythonPath,
		ShimPath:   shimPath,
	}
}

// Parse calls Python PDF parser via shim
func (p *PDFPythonShimParser) Parse(docID string, content interface{}) (*ParseResult, error) {
	// Convert content to bytes
	var contentBytes []byte
	var err error

	switch v := content.(type) {
	case []byte:
		contentBytes = v
	case string:
		// If it's a file path, read it
		if _, statErr := os.Stat(v); statErr == nil {
			contentBytes, err = os.ReadFile(v)
			if err != nil {
				return nil, fmt.Errorf("failed to read PDF file: %v", err)
			}
		} else {
			contentBytes = []byte(v)
		}
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	// Base64 encode the binary content for JSON transport
	contentBase64 := base64.StdEncoding.EncodeToString(contentBytes)

	// Prepare input for Python shim
	input := map[string]interface{}{
		"parser_type": "pdf",
		"doc_id":      docID,
		"content":     contentBase64,
		"metadata":    make(map[string]interface{}),
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %v", err)
	}

	// Call Python shim
	cmd := exec.Command(p.PythonPath, p.ShimPath)
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Python shim error: %v\nStderr: %s", err, stderr.String())
	}

	// Parse output
	var output struct {
		Success bool                   `json:"success"`
		Error   string                 `json:"error"`
		Result  map[string]interface{} `json:"result"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("failed to parse shim output: %v\nOutput: %s", err, stdout.String())
	}

	if !output.Success {
		return nil, fmt.Errorf("Python parser failed: %s", output.Error)
	}

	// Convert Python result to ParseResult
	return p.convertPythonResult(output.Result, docID)
}

// convertPythonResult converts Python parser output to Go ParseResult
func (p *PDFPythonShimParser) convertPythonResult(pythonResult map[string]interface{}, docID string) (*ParseResult, error) {
	result := &ParseResult{
		Document: Document{
			ID:      docID,
			DocType: "pdf",
		},
		Elements:      []Element{},
		Relationships: []Relationship{},
		Links:         []Link{},
	}

	// Extract document metadata
	if doc, ok := pythonResult["document"].(map[string]interface{}); ok {
		if metadata, ok := doc["metadata"].(map[string]interface{}); ok {
			result.Document.Metadata = metadata
		}
	}

	// Convert elements
	if elements, ok := pythonResult["elements"].([]interface{}); ok {
		for i, elemData := range elements {
			elem, ok := elemData.(map[string]interface{})
			if !ok {
				continue
			}

			element := Element{
				ElementID:       getString(elem, "element_id"),
				ElementType:     getString(elem, "element_type"),
				Content:         getString(elem, "content"),
				ContentPreview:  getString(elem, "content_preview"),
				ParentID:        getString(elem, "parent_id"),
				Position:        i,
				Depth:           getInt(elem, "depth"),
				ContentLocation: getMap(elem, "content_location"),
				Metadata:        getMap(elem, "metadata"),
			}

			result.Elements = append(result.Elements, element)
		}
	}

	// Convert relationships
	if relationships, ok := pythonResult["relationships"].([]interface{}); ok {
		for _, relData := range relationships {
			rel, ok := relData.(map[string]interface{})
			if !ok {
				continue
			}

			relationship := Relationship{
				RelationshipID:   getString(rel, "relationship_id"),
				RelationshipType: getString(rel, "relationship_type"),
				SourceElementID:  getString(rel, "source_id"),
				TargetElementID:  getString(rel, "target_id"),
				Confidence:       getFloat(rel, "confidence"),
				Metadata:         getMap(rel, "metadata"),
			}

			result.Relationships = append(result.Relationships, relationship)
		}
	}

	// Convert links
	if links, ok := pythonResult["links"].([]interface{}); ok {
		for _, linkData := range links {
			link, ok := linkData.(map[string]interface{})
			if !ok {
				continue
			}

			linkObj := Link{
				LinkID:          generateID("link"),
				SourceElementID: getString(link, "source_id"),
				LinkType:        getString(link, "link_type"),
				LinkTarget:      getString(link, "link_target"),
				LinkText:        getString(link, "link_text"),
			}

			result.Links = append(result.Links, linkObj)
		}
	}

	return result, nil
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if subMap, ok := v.(map[string]interface{}); ok {
			return subMap
		}
	}
	return make(map[string]interface{})
}
