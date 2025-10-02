package embeddings

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// PythonShimGenerator implements EmbeddingGenerator using a persistent Python process
type PythonShimGenerator struct {
	config     Config
	pythonCmd  string
	shimPath   string
	pythonPath string

	// Persistent process
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser
	mu     sync.Mutex
	closed bool
}

// NewPythonShimGenerator creates a new Python shim embedding generator
func NewPythonShimGenerator(config Config) (*PythonShimGenerator, error) {
	// Find Python executable
	pythonCmd := os.Getenv("PYTHON_CMD")
	if pythonCmd == "" {
		pythonCmd = "python3"
	}

	// Find embedding shim script - try multiple locations
	shimPath := filepath.Join("src", "go_doc_go", "embedding_shim.py")
	pythonPath := "src"

	if _, err := os.Stat(shimPath); os.IsNotExist(err) {
		// Try parent directory (when running from tests/)
		shimPath = filepath.Join("..", "src", "go_doc_go", "embedding_shim.py")
		pythonPath = "../src"
		if _, err := os.Stat(shimPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("embedding shim script not found in src/ or ../src/")
		}
	}

	log.Printf("Initialized Python embedding shim (provider: %s, model: %s, dimensions: %d)",
		config.Provider, config.Model, config.Dimensions)

	gen := &PythonShimGenerator{
		config:     config,
		pythonCmd:  pythonCmd,
		shimPath:   shimPath,
		pythonPath: pythonPath,
		closed:     false,
	}

	// Start persistent Python process
	if err := gen.startProcess(); err != nil {
		return nil, fmt.Errorf("failed to start Python process: %w", err)
	}

	return gen, nil
}

// startProcess starts the persistent Python process
func (g *PythonShimGenerator) startProcess() error {
	// Set up PYTHONPATH - use the one determined during initialization
	pythonPath := os.Getenv("PYTHONPATH")
	if pythonPath == "" {
		pythonPath = g.pythonPath
	}

	// Create command
	g.cmd = exec.Command(g.pythonCmd, g.shimPath)
	g.cmd.Env = append(os.Environ(), "PYTHONPATH="+pythonPath)

	// Set up pipes
	stdin, err := g.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	g.stdin = stdin

	stdout, err := g.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	g.stdout = bufio.NewReader(stdout)

	stderr, err := g.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	g.stderr = stderr

	// Start process
	if err := g.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Start stderr reader in background
	go g.readStderr()

	log.Printf("Started persistent Python embedding process (PID: %d)", g.cmd.Process.Pid)

	// Wait for Python to signal it's ready
	readyBytes, err := g.stdout.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("failed to read ready signal: %w", err)
	}

	var readyMsg struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(readyBytes, &readyMsg); err != nil {
		return fmt.Errorf("failed to parse ready signal: %w", err)
	}

	if readyMsg.Status != "ready" {
		return fmt.Errorf("unexpected ready message: %s", string(readyBytes))
	}

	log.Printf("Python embedding process ready")

	return nil
}

// readStderr reads and logs stderr output
func (g *PythonShimGenerator) readStderr() {
	scanner := bufio.NewScanner(g.stderr)
	for scanner.Scan() {
		log.Printf("[Python stderr] %s", scanner.Text())
	}
}

// callPythonShim calls the Python embedding shim with the given operation and data
func (g *PythonShimGenerator) callPythonShim(operation string, data map[string]interface{}) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closed {
		return nil, fmt.Errorf("generator is closed")
	}

	request := map[string]interface{}{
		"operation": operation,
		"config":    g.config,
		"data":      data,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write request with newline delimiter
	if _, err := g.stdin.Write(append(requestJSON, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response (one line)
	responseBytes, err := g.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return responseBytes, nil
}

// Generate generates an embedding for a single text
func (g *PythonShimGenerator) Generate(text string) ([]float64, error) {
	if text == "" {
		// Return zero vector for empty text
		return make([]float64, g.config.Dimensions), nil
	}

	data := map[string]interface{}{
		"text": text,
	}

	output, err := g.callPythonShim("generate", data)
	if err != nil {
		return nil, err
	}

	var response struct {
		Embedding []float64 `json:"embedding"`
		Error     string    `json:"error"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, output: %s", err, string(output))
	}

	if response.Error != "" {
		return nil, fmt.Errorf("embedding generation failed: %s", response.Error)
	}

	return response.Embedding, nil
}

// GenerateBatch generates embeddings for multiple texts
func (g *PythonShimGenerator) GenerateBatch(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	data := map[string]interface{}{
		"texts": texts,
	}

	output, err := g.callPythonShim("generate_batch", data)
	if err != nil {
		return nil, err
	}

	var response struct {
		Embeddings [][]float64 `json:"embeddings"`
		Error      string      `json:"error"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, output: %s", err, string(output))
	}

	if response.Error != "" {
		return nil, fmt.Errorf("batch embedding generation failed: %s", response.Error)
	}

	return response.Embeddings, nil
}

// GetDimensions returns the number of dimensions in the embedding
func (g *PythonShimGenerator) GetDimensions() int {
	return g.config.Dimensions
}

// GetModelName returns the name of the embedding model
func (g *PythonShimGenerator) GetModelName() string {
	return g.config.Model
}

// Close closes the embedding generator and releases resources
func (g *PythonShimGenerator) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closed {
		return nil
	}

	g.closed = true

	// Close stdin to signal Python process to exit
	if g.stdin != nil {
		g.stdin.Close()
	}

	// Wait for process to exit (with timeout would be better)
	if g.cmd != nil && g.cmd.Process != nil {
		log.Printf("Stopping Python embedding process (PID: %d)", g.cmd.Process.Pid)
		g.cmd.Wait()
	}

	return nil
}
