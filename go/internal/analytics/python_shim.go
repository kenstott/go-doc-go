package analytics

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

// PythonShimStorage implements Storage by calling Python analytics code
type PythonShimStorage struct {
	pythonPath string
	scriptPath string
	config     map[string]interface{}
}

// NewPythonShimStorage creates a new Python shim storage
func NewPythonShimStorage(config map[string]interface{}) (*PythonShimStorage, error) {
	// Find Python and the analytics script
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		pythonPath, err = exec.LookPath("python")
		if err != nil {
			return nil, fmt.Errorf("python not found in PATH: %w", err)
		}
	}

	// The script path will be relative to the project root
	scriptPath := filepath.Join("src", "go_doc_go", "analytics_shim.py")

	return &PythonShimStorage{
		pythonPath: pythonPath,
		scriptPath: scriptPath,
		config:     config,
	}, nil
}

// AppendDocuments appends documents via Python
func (s *PythonShimStorage) AppendDocuments(documents []Document) error {
	return s.callPython("append_documents", documents)
}

// AppendElements appends elements via Python
func (s *PythonShimStorage) AppendElements(elements []Element) error {
	return s.callPython("append_elements", elements)
}

// AppendRelationships appends relationships via Python
func (s *PythonShimStorage) AppendRelationships(relationships []Relationship) error {
	return s.callPython("append_relationships", relationships)
}

// AppendEmbeddings appends embeddings via Python
func (s *PythonShimStorage) AppendEmbeddings(embeddings []Embedding) error {
	return s.callPython("append_embeddings", embeddings)
}

// callPython calls the Python analytics shim script
func (s *PythonShimStorage) callPython(operation string, data interface{}) error {
	// Prepare the call data
	callData := map[string]interface{}{
		"operation": operation,
		"config":    s.config,
		"data":      data,
	}

	jsonData, err := json.Marshal(callData)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Call Python script with JSON input
	cmd := exec.Command(s.pythonPath, s.scriptPath)
	cmd.Stdin = nil // We'll use arguments instead

	// Set PYTHONPATH to include src directory
	cmd.Env = append(cmd.Environ(), "PYTHONPATH=src")

	// Pass JSON as argument
	cmd.Args = append(cmd.Args, string(jsonData))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("python call failed: %w, output: %s", err, string(output))
	}

	// Check result
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse python result: %w", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		errorMsg := "unknown error"
		if msg, ok := result["error"].(string); ok {
			errorMsg = msg
		}
		return fmt.Errorf("python operation failed: %s", errorMsg)
	}

	return nil
}

// Close closes the storage
func (s *PythonShimStorage) Close() error {
	// Nothing to close for Python shim
	return nil
}
