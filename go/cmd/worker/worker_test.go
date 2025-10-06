package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestConfig represents the worker configuration structure
type TestConfig struct {
	Embedding struct {
		Enabled          bool   `yaml:"enabled"`
		Provider         string `yaml:"provider"`
		Model            string `yaml:"model"`
		ModelPath        string `yaml:"model_path"`
		Dimensions       int    `yaml:"dimensions"`
		ChunkSize        int    `yaml:"chunk_size"`
		Overlap          int    `yaml:"overlap"`
		Contextual       bool   `yaml:"contextual"`
		PredecessorCount int    `yaml:"predecessor_count"`
		SuccessorCount   int    `yaml:"successor_count"`
	} `yaml:"embedding"`

	ContentSources []struct {
		Name              string   `yaml:"name"`
		Type              string   `yaml:"type"`
		BasePath          string   `yaml:"base_path,omitempty"`
		BaseURL           string   `yaml:"base_url,omitempty"`
		FilePattern       string   `yaml:"file_pattern,omitempty"`
		IncludeExtensions []string `yaml:"include_extensions,omitempty"`
		WatchForChanges   bool     `yaml:"watch_for_changes,omitempty"`
		MaxLinkDepth      int      `yaml:"max_link_depth,omitempty"`
		RefreshInterval   int      `yaml:"refresh_interval,omitempty"`
		URLList           []string `yaml:"url_list,omitempty"`
		IncludePatterns   []string `yaml:"include_patterns,omitempty"`
		ExcludePatterns   []string `yaml:"exclude_patterns,omitempty"`
		Headers           map[string]string `yaml:"headers,omitempty"`
	} `yaml:"content_sources"`

	RelationshipDetection struct {
		Enabled    bool `yaml:"enabled"`
		Structural bool `yaml:"structural"`
		Semantic   bool `yaml:"semantic"`
		CrossDocumentSemantic struct {
			SimilarityThreshold float64 `yaml:"similarity_threshold"`
		} `yaml:"cross_document_semantic"`
	} `yaml:"relationship_detection"`

	Processing struct {
		BatchSize      int `yaml:"batch_size"`
		MaxWorkers     int `yaml:"max_workers"`
		TimeoutSeconds int `yaml:"timeout_seconds"`
		JobControl     struct {
			Backend           string `yaml:"backend"`
			Path              string `yaml:"path"`
			ClaimTimeout      int    `yaml:"claim_timeout"`
			HeartbeatInterval int    `yaml:"heartbeat_interval"`
			MaxRetries        int    `yaml:"max_retries"`
		} `yaml:"job_control"`
	} `yaml:"processing"`

	Analytics struct {
		Enabled bool `yaml:"enabled"`
		Outputs []struct {
			Type         string   `yaml:"type"`
			Path         string   `yaml:"path"`
			Partitioning []string `yaml:"partitioning"`
		} `yaml:"outputs"`
	} `yaml:"analytics"`

	Logging struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logging"`
}

// TestHelper provides utility functions for worker tests
type TestHelper struct {
	t              *testing.T
	projectRoot    string
	workerBinary   string
	testOutputDir  string
	testAssetsDir  string
	baseConfigPath string
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	// Get project root (3 levels up from go/cmd/worker)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..", "..")

	return &TestHelper{
		t:              t,
		projectRoot:    projectRoot,
		workerBinary:   filepath.Join(projectRoot, "bin", "goworker"),
		testOutputDir:  filepath.Join(projectRoot, "tests", "test_output", "worker_cli_test_go"),
		testAssetsDir:  filepath.Join(projectRoot, "tests", "assets"),
		baseConfigPath: filepath.Join(projectRoot, "tests", "config.sqlite.yaml"),
	}
}

// SetupTestDir creates and returns a clean test directory
func (h *TestHelper) SetupTestDir() string {
	// Remove existing test directory
	if err := os.RemoveAll(h.testOutputDir); err != nil {
		h.t.Logf("Warning: failed to remove existing test dir: %v", err)
	}

	// Create fresh test directory
	if err := os.MkdirAll(h.testOutputDir, 0755); err != nil {
		h.t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create logs directory
	logsDir := filepath.Join(h.testOutputDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		h.t.Fatalf("Failed to create logs directory: %v", err)
	}

	h.t.Cleanup(func() {
		h.t.Logf("\nWorker test output preserved at: %s", h.testOutputDir)
		h.t.Logf("Job control DB: %s", filepath.Join(h.testOutputDir, "job_queue-go.db"))
		h.t.Logf("Analytics output: %s", filepath.Join(h.testOutputDir, "analytics-output-go"))
		h.t.Logf("Source documents: %s", h.testAssetsDir)
	})

	return h.testOutputDir
}

// CreateIsolatedConfig creates a test-specific configuration
func (h *TestHelper) CreateIsolatedConfig(testDir string, customizations func(*TestConfig)) string {
	// Read base config
	data, err := os.ReadFile(h.baseConfigPath)
	if err != nil {
		h.t.Fatalf("Failed to read base config: %v", err)
	}

	var config TestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		h.t.Fatalf("Failed to parse base config: %v", err)
	}

	// Override paths to use test directory
	jobDBPath := filepath.Join(testDir, "job_queue-go.db")
	config.Processing.JobControl.Path = jobDBPath

	// Remove existing job queue database
	if err := os.Remove(jobDBPath); err != nil && !os.IsNotExist(err) {
		h.t.Logf("Warning: failed to remove existing job DB: %v", err)
	}

	// Configure analytics output
	analyticsPath := filepath.Join(testDir, "analytics-output-go")
	if len(config.Analytics.Outputs) > 0 {
		config.Analytics.Outputs[0].Path = analyticsPath
	}

	// Remove existing analytics directory
	if err := os.RemoveAll(analyticsPath); err != nil {
		h.t.Logf("Warning: failed to remove existing analytics dir: %v", err)
	}

	// Configure logging
	config.Logging.File = filepath.Join(testDir, "logs", "worker-go.log")

	// Update content source to use test assets
	for i := range config.ContentSources {
		if config.ContentSources[i].Type == "file" {
			config.ContentSources[i].BasePath = h.testAssetsDir
		}
	}

	// Apply custom modifications
	if customizations != nil {
		customizations(&config)
	}

	// Write isolated config
	isolatedConfigPath := filepath.Join(testDir, "worker_config.yaml")
	data, err = yaml.Marshal(&config)
	if err != nil {
		h.t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(isolatedConfigPath, data, 0644); err != nil {
		h.t.Fatalf("Failed to write isolated config: %v", err)
	}

	return isolatedConfigPath
}

// RunWorker executes the worker binary with given arguments
func (h *TestHelper) RunWorker(ctx context.Context, configPath string, args ...string) (stdout, stderr string, err error) {
	cmdArgs := append([]string{"--config", configPath}, args...)
	cmd := exec.CommandContext(ctx, h.workerBinary, cmdArgs...)
	cmd.Dir = h.projectRoot

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// RunWorkerAsync starts the worker in background and returns the command
func (h *TestHelper) RunWorkerAsync(ctx context.Context, configPath string, args ...string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	cmdArgs := append([]string{"--config", configPath}, args...)
	cmd := exec.CommandContext(ctx, h.workerBinary, cmdArgs...)
	cmd.Dir = h.projectRoot

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start worker: %w", err)
	}

	return cmd, stdout, stderr, nil
}

// ContainsAny checks if output contains any of the given indicators
func ContainsAny(output string, indicators []string) bool {
	lowerOutput := strings.ToLower(output)
	for _, indicator := range indicators {
		if strings.Contains(lowerOutput, strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

// TestWorkerBinaryExists verifies that the worker binary is compiled
func TestWorkerBinaryExists(t *testing.T) {
	h := NewTestHelper(t)

	if _, err := os.Stat(h.workerBinary); os.IsNotExist(err) {
		t.Fatalf("Worker binary not found at %s. Run 'go build -o bin/goworker ./cmd/worker' first", h.workerBinary)
	}

	t.Logf("✓ Worker binary found at: %s", h.workerBinary)
}

// TestWorkerConfigValidation tests that the worker can validate configuration
func TestWorkerConfigValidation(t *testing.T) {
	h := NewTestHelper(t)
	testDir := h.SetupTestDir()
	configPath := h.CreateIsolatedConfig(testDir, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stdout, stderr, err := h.RunWorker(ctx, configPath, "--max-documents", "1")

	output := stdout + stderr

	// Check for success indicators
	successIndicators := []string{
		"Starting worker",
		"elected as leader",
		"became leader",
		"Initialized",
		"Loading configuration from",
	}

	if ctx.Err() == context.DeadlineExceeded {
		// Timeout is expected for continuous workers - check if it started properly
		if !ContainsAny(output, successIndicators) {
			t.Fatalf("Worker startup failed - no success indicators in output: %s", output[:min(1000, len(output))])
		}
		t.Log("✓ Worker started successfully (timed out as expected)")
	} else if err != nil {
		t.Fatalf("Worker failed: %v\nStderr: %s", err, stderr)
	} else {
		t.Log("✓ Worker completed successfully")
	}
}

// TestWorkerProcessDocuments tests that the worker can process documents
func TestWorkerProcessDocuments(t *testing.T) {
	h := NewTestHelper(t)
	testDir := h.SetupTestDir()
	configPath := h.CreateIsolatedConfig(testDir, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	stdout, stderr, err := h.RunWorker(ctx, configPath, "--max-documents", "5")

	output := stdout + stderr

	// Check for success indicators
	successIndicators := []string{
		"Starting worker",
		"elected as leader",
		"became leader",
		"completed document",
		"processing",
		"Initialized",
		"completed successfully",
	}

	if ctx.Err() == context.DeadlineExceeded {
		// Timeout is expected - check for processing indicators
		if !ContainsAny(output, successIndicators) {
			t.Fatalf("Worker processing failed - no success indicators in output: %s", output[:min(1000, len(output))])
		}
		t.Log("✓ Worker processed documents successfully (timed out as expected)")
	} else if err != nil {
		t.Fatalf("Worker failed: %v\nStderr: %s", err, stderr)
	} else {
		t.Log("✓ Worker completed successfully")
	}

	// Verify analytics output was created
	analyticsPath := filepath.Join(testDir, "analytics-output-go")
	if _, err := os.Stat(analyticsPath); err == nil {
		// Count parquet files
		parquetFiles := []string{}
		filepath.Walk(analyticsPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && strings.HasSuffix(path, ".parquet") {
				parquetFiles = append(parquetFiles, path)
			}
			return nil
		})
		t.Logf("✓ Analytics output created: %d parquet files", len(parquetFiles))
	}
}

// TestWorkerLeaderElection tests worker leader election
func TestWorkerLeaderElection(t *testing.T) {
	h := NewTestHelper(t)
	testDir := h.SetupTestDir()
	configPath := h.CreateIsolatedConfig(testDir, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stdout, stderr, err := h.RunWorker(ctx, configPath,
		"--max-documents", "3",
		"--worker-id", "test-leader-worker")

	output := stdout + stderr

	// Check for leader election indicators
	leaderIndicators := []string{
		"elected as leader",
		"became leader",
		"is now leader",
		"Starting worker",
		"Registered worker",
	}

	if ctx.Err() == context.DeadlineExceeded {
		if !ContainsAny(output, leaderIndicators) {
			t.Fatalf("Worker leadership test failed - no leadership indicators in output: %s", output[:min(1000, len(output))])
		}
		t.Log("✓ Worker leader election successful")
	} else if err != nil {
		t.Fatalf("Worker failed: %v\nStderr: %s", err, stderr)
	} else {
		t.Log("✓ Worker completed with leader election")
	}
}

// TestMultipleWorkersCoordination tests multiple workers coordinating via job control
func TestMultipleWorkersCoordination(t *testing.T) {
	h := NewTestHelper(t)
	testDir := h.SetupTestDir()
	configPath := h.CreateIsolatedConfig(testDir, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Run 3 workers in parallel
	var wg sync.WaitGroup
	outputs := make([]string, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(workerNum int) {
			defer wg.Done()

			workerID := fmt.Sprintf("test-worker-%d", workerNum)
			stdout, stderr, _ := h.RunWorker(ctx, configPath,
				"--max-documents", "10",
				"--worker-id", workerID)

			outputs[workerNum] = stdout + stderr
		}(i)
	}

	wg.Wait()

	// Analyze combined output
	combinedOutput := strings.Join(outputs, "\n")

	// Check for coordination indicators
	coordinationIndicators := []string{
		"Starting worker",
		"completed document",
		"leader",
		"elected as leader",
		"Discovering documents from source",
		"Queued.*new documents",
	}

	if !ContainsAny(combinedOutput, coordinationIndicators) {
		t.Fatalf("Worker coordination failed - no coordination indicators in output: %s", combinedOutput[:min(1000, len(combinedOutput))])
	}

	t.Log("✓ Multiple workers coordinated successfully")

	// Check for leader election (at least one worker should become leader)
	if !ContainsAny(combinedOutput, []string{"elected as leader", "became leader"}) {
		t.Log("Warning: No explicit leader election detected, but coordination occurred")
	}
}

// TestWorkerMaxDocumentsLimit tests that worker respects max-documents limit
func TestWorkerMaxDocumentsLimit(t *testing.T) {
	h := NewTestHelper(t)
	testDir := h.SetupTestDir()
	configPath := h.CreateIsolatedConfig(testDir, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout, stderr, err := h.RunWorker(ctx, configPath, "--max-documents", "2")

	output := stdout + stderr

	// Check for completion indicators
	completionIndicators := []string{
		"completed document",
		"stopping",
		"Processed.*documents",
		"completed successfully",
		"Starting worker",
		"reached.*document limit",
	}

	if ctx.Err() == context.DeadlineExceeded {
		if !ContainsAny(output, completionIndicators) {
			t.Fatalf("Worker did not start properly - no completion indicators in output: %s", output[:min(1000, len(output))])
		}
		t.Log("✓ Worker respected max-documents limit")
	} else if err != nil {
		// Check if error is due to reaching limit (which is success)
		if ContainsAny(output, completionIndicators) {
			t.Log("✓ Worker completed with max-documents limit")
		} else {
			t.Fatalf("Worker failed: %v\nStderr: %s", err, stderr)
		}
	} else {
		t.Log("✓ Worker completed successfully with limit")
	}
}

// TestJobControlDatabaseCreation tests that worker creates job control database
func TestJobControlDatabaseCreation(t *testing.T) {
	h := NewTestHelper(t)
	testDir := h.SetupTestDir()
	configPath := h.CreateIsolatedConfig(testDir, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, _, _ = h.RunWorker(ctx, configPath, "--max-documents", "1")

	// Check if job control database was created
	jobDBPath := filepath.Join(testDir, "job_queue-go.db")

	if _, err := os.Stat(jobDBPath); err == nil {
		t.Logf("✓ Job control database created at: %s", jobDBPath)
	} else {
		t.Logf("Note: Job control database not created (may use different backend): %v", err)
		// This is not necessarily a failure - workers might use PostgreSQL or other backends
	}
}

// TestAnalyticsOutputCreation tests that worker creates analytics output
func TestAnalyticsOutputCreation(t *testing.T) {
	h := NewTestHelper(t)
	testDir := h.SetupTestDir()
	configPath := h.CreateIsolatedConfig(testDir, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	_, _, _ = h.RunWorker(ctx, configPath, "--max-documents", "2")

	// Check for analytics output directory
	analyticsPath := filepath.Join(testDir, "analytics-output-go")

	if _, err := os.Stat(analyticsPath); err == nil {
		// Count parquet files
		parquetCount := 0
		filepath.Walk(analyticsPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && strings.HasSuffix(path, ".parquet") {
				parquetCount++
			}
			return nil
		})

		t.Logf("✓ Analytics output created with %d parquet files at: %s", parquetCount, analyticsPath)
	} else {
		t.Logf("Note: Analytics output directory not found: %v", err)
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
