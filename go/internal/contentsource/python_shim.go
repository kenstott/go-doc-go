package contentsource

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

// PythonShimSource implements ContentSource by calling Python content source code
type PythonShimSource struct {
	pythonPath string
	scriptPath string
	sourceName string
	config     map[string]interface{}
}

// NewPythonShimSource creates a new Python shim content source
func NewPythonShimSource(sourceName string, config map[string]interface{}) (*PythonShimSource, error) {
	// Find Python
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		pythonPath, err = exec.LookPath("python")
		if err != nil {
			return nil, fmt.Errorf("python not found in PATH: %w", err)
		}
	}

	// The script path will be relative to the project root
	scriptPath := filepath.Join("src", "go_doc_go", "content_source_shim.py")

	return &PythonShimSource{
		pythonPath: pythonPath,
		scriptPath: scriptPath,
		sourceName: sourceName,
		config:     config,
	}, nil
}

// FetchDocument fetches a document via Python
func (s *PythonShimSource) FetchDocument(sourceID string) (*DocumentContent, error) {
	result, err := s.callPython("fetch_document", map[string]interface{}{
		"source_id": sourceID,
	})
	if err != nil {
		return nil, err
	}

	// Parse the result into DocumentContent
	var content DocumentContent
	resultJSON, err := json.Marshal(result["data"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(resultJSON, &content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document content: %w", err)
	}

	return &content, nil
}

// ListDocuments lists documents via Python
func (s *PythonShimSource) ListDocuments() ([]DocumentInfo, error) {
	result, err := s.callPython("list_documents", nil)
	if err != nil {
		return nil, err
	}

	// Parse the result into DocumentInfo slice
	var documents []DocumentInfo
	resultJSON, err := json.Marshal(result["data"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(resultJSON, &documents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal documents: %w", err)
	}

	return documents, nil
}

// HasChanged checks if a document has changed via Python
func (s *PythonShimSource) HasChanged(sourceID string, lastModified interface{}) (bool, error) {
	result, err := s.callPython("has_changed", map[string]interface{}{
		"source_id":     sourceID,
		"last_modified": lastModified,
	})
	if err != nil {
		return false, err
	}

	// Extract the boolean result
	changed, ok := result["data"].(bool)
	if !ok {
		return false, fmt.Errorf("invalid response from Python: expected bool, got %T", result["data"])
	}

	return changed, nil
}

// callPython calls the Python content source shim script
func (s *PythonShimSource) callPython(operation string, params interface{}) (map[string]interface{}, error) {
	// Prepare the call data
	callData := map[string]interface{}{
		"operation":   operation,
		"source_name": s.sourceName,
		"config":      s.config,
		"params":      params,
	}

	jsonData, err := json.Marshal(callData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Call Python script with JSON input
	cmd := exec.Command(s.pythonPath, s.scriptPath)
	cmd.Stdin = nil

	// Pass JSON as argument
	cmd.Args = append(cmd.Args, string(jsonData))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("python call failed: %w, output: %s", err, string(output))
	}

	// Check result
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse python result: %w", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		errorMsg := "unknown error"
		if msg, ok := result["error"].(string); ok {
			errorMsg = msg
		}
		return nil, fmt.Errorf("python operation failed: %s", errorMsg)
	}

	return result, nil
}
