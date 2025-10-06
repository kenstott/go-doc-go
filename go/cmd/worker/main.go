package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/kennethstott/go-doc-go/internal/embeddings"
	"github.com/kennethstott/go-doc-go/internal/jobcontrol"
	"github.com/kennethstott/go-doc-go/internal/worker"
)

// Config represents the TOML configuration structure
type Config struct {
	Processing struct {
		MaxWorkers int `toml:"max_workers"`
		JobControl struct {
			Backend           string `toml:"backend"` // "sqlite" or "postgres"
			Path              string `toml:"path"`    // SQLite: file path; PostgreSQL: DSN
			ClaimTimeout      int    `toml:"claim_timeout"`
			HeartbeatInterval int    `toml:"heartbeat_interval"`
			MaxRetries        int    `toml:"max_retries"`
		} `toml:"job_control"`
		Neo4jExport struct {
			Enabled            bool   `toml:"enabled"`
			EmptyQueueWaitTime int    `toml:"empty_queue_wait_time"` // Seconds to wait before export
			SourceAnalytics    string `toml:"source_analytics"`      // Source analytics type (e.g., "parquet")
			Connection         struct {
				URI      string `toml:"uri"`
				Username string `toml:"username"`
				Password string `toml:"password"`
				Database string `toml:"database"`
			} `toml:"connection"`
			BatchSize int `toml:"batch_size"`
		} `toml:"neo4j_export"`
	} `toml:"processing"`
	ContentSources []map[string]interface{} `toml:"content_sources"`
	Analytics      struct {
		Enabled bool                     `toml:"enabled"`
		Outputs []map[string]interface{} `toml:"outputs"`
	} `toml:"analytics"`
	Embedding *embeddings.Config `toml:"embedding"`
}

func main() {
	// Command line flags
	var (
		configFile   = flag.String("config", "", "Path to configuration file")
		workerID     = flag.String("worker-id", "", "Custom worker ID (auto-generated if not provided)")
		maxDocuments = flag.Int("max-documents", 0, "Maximum number of documents to process (0 = unlimited)")
		numWorkers   = flag.Int("workers", 0, "Number of concurrent goroutine workers (0 = use NUM_WORKERS env var, default: 1)")
		numInstances = flag.Int("instances", 1, "Number of separate worker processes to spawn (default: 1, Python multiprocessing style)")
	)

	flag.Parse()

	// Get instances from environment variable if not specified via CLI
	if *numInstances == 1 {
		if envInstances := os.Getenv("NUM_INSTANCES"); envInstances != "" {
			if parsed, err := fmt.Sscanf(envInstances, "%d", numInstances); err == nil && parsed == 1 {
				log.Printf("Using NUM_INSTANCES from environment: %d", *numInstances)
			} else {
				log.Printf("Warning: Invalid NUM_INSTANCES value '%s', using default: 1", envInstances)
				*numInstances = 1
			}
		}
	}

	// If instances > 1, spawn multiple child processes and exit
	if *numInstances > 1 {
		spawnInstances(*numInstances, os.Args)
		return
	}

	// Get workers from environment variable if not specified via CLI
	workersFromCLI := *numWorkers != 0
	workersFromEnv := false
	if *numWorkers == 0 {
		if envWorkers := os.Getenv("NUM_WORKERS"); envWorkers != "" {
			if parsed, err := fmt.Sscanf(envWorkers, "%d", numWorkers); err == nil && parsed == 1 {
				log.Printf("Using NUM_WORKERS from environment: %d", *numWorkers)
				workersFromEnv = true
			} else {
				log.Printf("Warning: Invalid NUM_WORKERS value '%s', using default: 1", envWorkers)
				*numWorkers = 1
			}
		} else {
			*numWorkers = 1
		}
	}

	// Determine config file path
	configPath := *configFile
	if configPath == "" {
		configPath = os.Getenv("GO_DOC_GO_CONFIG_PATH")
		if configPath == "" {
			configPath = "./config.toml"
		}
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Configuration file not found: %s\n", configPath)
		os.Exit(1)
	}

	log.Printf("Loading configuration from: %s", configPath)

	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Only use max_workers from config if explicitly set via CLI or env var
	// Default is 1, not from config file
	if !workersFromCLI && !workersFromEnv && config.Processing.MaxWorkers > 0 {
		// Config file is lowest priority - only use if nothing else specified
		// But since we want default=1, we skip config file entirely
		log.Printf("Ignoring max_workers from config (%d), using default: %d", config.Processing.MaxWorkers, *numWorkers)
	}

	// Generate worker ID if not provided
	if *workerID == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown"
		}
		*workerID = fmt.Sprintf("worker_%s_%d", hostname, os.Getpid())
	}

	// Create Neo4j export configuration if enabled
	var neo4jExportConfig *worker.Neo4jExportConfig
	if config.Processing.Neo4jExport.Enabled {
		// Find source analytics path from analytics outputs
		sourcePath := ""
		for _, output := range config.Analytics.Outputs {
			if outputType, ok := output["type"].(string); ok && outputType == config.Processing.Neo4jExport.SourceAnalytics {
				if path, ok := output["path"].(string); ok {
					sourcePath = path
					break
				}
			}
		}

		neo4jExportConfig = &worker.Neo4jExportConfig{
			Enabled:            config.Processing.Neo4jExport.Enabled,
			EmptyQueueWaitTime: config.Processing.Neo4jExport.EmptyQueueWaitTime,
			SourceAnalytics:    config.Processing.Neo4jExport.SourceAnalytics,
			SourcePath:         sourcePath,
			Connection: map[string]interface{}{
				"uri":      config.Processing.Neo4jExport.Connection.URI,
				"username": config.Processing.Neo4jExport.Connection.Username,
				"password": config.Processing.Neo4jExport.Connection.Password,
				"database": config.Processing.Neo4jExport.Connection.Database,
			},
			BatchSize: config.Processing.Neo4jExport.BatchSize,
		}
	}

	// Create worker configuration
	workerConfig := worker.Config{
		WorkerID:   *workerID,
		NumWorkers: *numWorkers,
		JobControlConfig: jobcontrol.Config{
			Backend:           config.Processing.JobControl.Backend,
			Path:              config.Processing.JobControl.Path,
			ClaimTimeout:      config.Processing.JobControl.ClaimTimeout,
			HeartbeatInterval: config.Processing.JobControl.HeartbeatInterval,
			MaxRetries:        config.Processing.JobControl.MaxRetries,
		},
		ContentSources:    config.ContentSources,
		AnalyticsConfigs:  config.Analytics.Outputs,
		EmbeddingConfig:   config.Embedding,
		MaxDocuments:      *maxDocuments,
		DiscoveryInterval: 60,
		Neo4jExportConfig: neo4jExportConfig,
	}

	// Create worker
	w, err := worker.NewWorker(workerConfig)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}
	defer w.Close()

	// Run worker
	log.Printf("========================================")
	log.Printf("STARTING WORKER: %s", *workerID)
	log.Printf("  Max documents: %d", *maxDocuments)
	log.Printf("  Goroutine workers: %d", *numWorkers)
	log.Printf("========================================")

	if err := w.Run(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}

	log.Println("========================================")
	log.Println("WORKER COMPLETED SUCCESSFULLY")
	log.Println("========================================")
}

// spawnInstances spawns multiple child worker processes (Python multiprocessing style)
func spawnInstances(numInstances int, originalArgs []string) {
	log.Printf("========================================")
	log.Printf("SPAWNING %d WORKER INSTANCES", numInstances)
	log.Printf("========================================")

	// Build new args without --instances flag (but keep --workers to inherit)
	var newArgs []string
	skipNext := false
	for _, arg := range originalArgs[1:] { // Skip program name
		if skipNext {
			skipNext = false
			continue
		}
		// Skip --instances flag
		if arg == "--instances" || arg == "-instances" {
			skipNext = true
			continue
		}
		// Also skip --instances=N format
		if len(arg) >= 11 && (arg[:11] == "--instances" || arg[:10] == "-instances") {
			continue
		}
		newArgs = append(newArgs, arg)
	}

	// Add --instances=1 to prevent recursive spawning (--workers is inherited from parent)
	newArgs = append(newArgs, "--instances=1")

	// Get executable path
	executable, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	// Spawn instances
	var wg sync.WaitGroup
	processes := make([]*exec.Cmd, numInstances)

	for i := 0; i < numInstances; i++ {
		wg.Add(1)

		// Create command with same args
		cmd := exec.Command(executable, newArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Remove NUM_INSTANCES from child environment to prevent recursive spawning
		env := os.Environ()
		var cleanEnv []string
		for _, e := range env {
			if !strings.HasPrefix(e, "NUM_INSTANCES=") {
				cleanEnv = append(cleanEnv, e)
			}
		}
		cmd.Env = cleanEnv

		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // Create new process group
		}

		processes[i] = cmd

		// Start instance
		log.Printf("Starting worker instance %d/%d (PID will be assigned)", i+1, numInstances)
		if err := cmd.Start(); err != nil {
			log.Fatalf("Failed to start instance %d: %v", i+1, err)
		}

		log.Printf("Worker instance %d started with PID %d", i+1, cmd.Process.Pid)

		// Wait for this instance in a goroutine
		go func(instanceNum int, process *exec.Cmd) {
			defer wg.Done()
			if err := process.Wait(); err != nil {
				log.Printf("Worker instance %d (PID %d) exited with error: %v", instanceNum, process.Process.Pid, err)
			} else {
				log.Printf("Worker instance %d (PID %d) completed successfully", instanceNum, process.Process.Pid)
			}
		}(i+1, cmd)
	}

	log.Printf("All %d worker instances started, waiting for completion...", numInstances)

	// Wait for all instances to complete
	wg.Wait()

	log.Printf("========================================")
	log.Printf("ALL %d WORKER INSTANCES COMPLETED", numInstances)
	log.Printf("========================================")
}

func loadConfig(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse TOML
	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Debug: Log embedding config
	if config.Embedding != nil {
		log.Printf("DEBUG: Loaded embedding config - Contextual: %v, Predecessor: %d, Successor: %d",
			config.Embedding.Contextual, config.Embedding.PredecessorCount, config.Embedding.SuccessorCount)
	}

	// Set defaults
	if config.Processing.JobControl.ClaimTimeout == 0 {
		config.Processing.JobControl.ClaimTimeout = 300
	}
	if config.Processing.JobControl.HeartbeatInterval == 0 {
		config.Processing.JobControl.HeartbeatInterval = 30
	}
	if config.Processing.JobControl.MaxRetries == 0 {
		config.Processing.JobControl.MaxRetries = 3
	}

	// Convert content sources to proper format
	contentSources := make(map[string]map[string]interface{})
	for _, source := range config.ContentSources {
		if name, ok := source["name"].(string); ok {
			contentSources[name] = source
		}
	}
	config.ContentSources = []map[string]interface{}{}
	for name, sourceConfig := range contentSources {
		sourceConfig["name"] = name
		config.ContentSources = append(config.ContentSources, sourceConfig)
	}

	// Expand paths to be absolute (only for SQLite - PostgreSQL uses connection strings/URIs)
	backend := strings.ToLower(config.Processing.JobControl.Backend)
	if backend == "" || backend == "sqlite" {
		if !filepath.IsAbs(config.Processing.JobControl.Path) {
			configDir := filepath.Dir(path)
			config.Processing.JobControl.Path = filepath.Join(configDir, config.Processing.JobControl.Path)
		}
	}

	return &config, nil
}
