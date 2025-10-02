package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PythonShimStorage implements Storage by calling Python analytics code
type PythonShimStorage struct {
	pythonPath    string
	scriptPath    string
	pythonPathEnv string
	config        map[string]interface{}
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

	// Find the script path - try current directory and parent directory
	scriptPath := filepath.Join("src", "go_doc_go", "analytics_shim.py")
	pythonPathEnv := "src"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// Try parent directory (for when running from tests/)
		scriptPath = filepath.Join("..", "src", "go_doc_go", "analytics_shim.py")
		pythonPathEnv = "../src"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("analytics_shim.py not found in src/ or ../src/")
		}
	}

	return &PythonShimStorage{
		pythonPath:    pythonPath,
		scriptPath:    scriptPath,
		pythonPathEnv: pythonPathEnv,
		config:        config,
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

	// Call Python script with JSON input via stdin (to avoid "argument list too long")
	cmd := exec.Command(s.pythonPath, s.scriptPath)

	// Set PYTHONPATH to include src directory
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("PYTHONPATH=%s", s.pythonPathEnv))

	// Use stdin pipe to send data
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start python: %w", err)
	}

	// Write data to stdin
	if _, err := stdin.Write(jsonData); err != nil {
		stdin.Close()
		return fmt.Errorf("failed to write to stdin: %w", err)
	}
	stdin.Close()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("python call failed: %w, stdout: %s, stderr: %s", err, stdout.String(), stderr.String())
	}

	output := stdout.Bytes()

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
