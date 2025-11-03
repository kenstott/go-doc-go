package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/kennethstott/doculyzer-go-conversion/internal/embeddings"
	"github.com/kennethstott/doculyzer-go-conversion/internal/jobcontrol"
	"github.com/kennethstott/doculyzer-go-conversion/internal/worker"
)

// DiscoveryConfig represents unified discovery/crawling configuration
type DiscoveryConfig struct {
	Enabled         bool     `toml:"enabled"`
	MaxDepth        int      `toml:"max_depth"`
	IncludePatterns []string `toml:"include_patterns"`
	ExcludePatterns []string `toml:"exclude_patterns"`
	Hyperlinks      *HyperlinkDiscoveryConfig      `toml:"hyperlinks"`
	CodeDependencies *CodeDependencyDiscoveryConfig `toml:"code_dependencies"`
}

// HyperlinkDiscoveryConfig controls hyperlink crawling behavior
type HyperlinkDiscoveryConfig struct {
	Enabled             bool `toml:"enabled"`
	MaxDepth            *int `toml:"max_depth"` // nil = use parent config
	SkipExternalDomains bool `toml:"skip_external_domains"`
}

// CodeDependencyDiscoveryConfig controls code import/dependency crawling
type CodeDependencyDiscoveryConfig struct {
	Enabled        bool `toml:"enabled"`
	MaxDepth       *int `toml:"max_depth"` // nil = use parent config
	FollowStdlib   bool `toml:"follow_stdlib"`
	FollowLocal    bool `toml:"follow_local"`
	FollowExternal bool `toml:"follow_external"`
}

// Config represents the TOML configuration structure
type Config struct {
	Processing struct {
		MaxWorkers int `toml:"workers"`
		JobControl struct {
			Backend            string `toml:"backend"` // "sqlite" or "postgres"
			Path               string `toml:"path"`    // SQLite: file path; PostgreSQL: DSN
			ClaimTimeout       int    `toml:"claim_timeout"`
			HeartbeatInterval  int    `toml:"heartbeat_interval"`
			MaxRetries         int    `toml:"max_retries"`
			IdleCheckInterval  int    `toml:"idle_check_interval"`  // Seconds between idle queue checks (default: 300)
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
	ContentSources       []map[string]interface{} `toml:"content_sources"`
	RelationshipDetection struct {
		Enabled    bool `toml:"enabled"`
		Structural bool `toml:"structural"`
		Semantic   bool `toml:"semantic"`
		CrossDocumentSemantic struct {
			Enabled                  bool    `toml:"enabled"`
			SimilarityThreshold      float64 `toml:"similarity_threshold"`
			QueueIdleTriggerMinutes  int     `toml:"queue_idle_trigger_minutes"`
			RateLimitBatchSize       int     `toml:"rate_limit_batch_size"`
			RateLimitSleepMs         int     `toml:"rate_limit_sleep_ms"`
			MinIntervalMinutes       int     `toml:"min_interval_minutes"`
		} `toml:"cross_document_semantic"`
	} `toml:"relationship_detection"`
	Ontology struct {
		Enabled                 bool    `toml:"enabled"`
		SchemaPath              string  `toml:"schema_path"`
		DiversityThreshold      float64 `toml:"diversity_threshold"`       // Cosine similarity threshold for diversity filtering (0.0-1.0)
		QueueIdleTriggerMinutes int     `toml:"queue_idle_trigger_minutes"`
		MinIntervalMinutes      int     `toml:"min_interval_minutes"`
	} `toml:"ontology"`
	Analytics struct {
		Enabled bool                     `toml:"enabled"`
		Outputs []map[string]interface{} `toml:"outputs"`
	} `toml:"analytics"`
	Embedding *embeddings.Config `toml:"embedding"`
}

func main() {
	// Command line flags
	var (
		configFile        = flag.String("config", "", "Path to configuration file")
		workerID          = flag.String("worker-id", "", "Custom worker ID (auto-generated if not provided)")
		maxDocuments      = flag.Int("max-documents", 0, "Maximum number of documents to process (0 = unlimited)")
		numWorkers        = flag.Int("workers", 0, "Number of concurrent goroutine workers (0 = use NUM_WORKERS env var, default: 1)")
		numInstances      = flag.Int("instances", 1, "Number of separate worker processes to spawn (default: 1, Python multiprocessing style)")
		shutdownWhenIdle  = flag.Bool("shutdown-when-idle", false, "Shutdown worker when idle (no work available)")
		shutdownFile      = flag.String("shutdown-file", "", "Path to shutdown control file (worker exits when file exists)")
		idleTimeout       = flag.Duration("idle-timeout", 0, "Shutdown after being idle for this duration (0 = disabled, e.g. 5m, 1h)")
	)

	flag.Parse()

	// Log GOGC setting for memory tuning visibility
	// GOGC controls garbage collection frequency (default 100 = GC when heap doubles)
	// Setting GOGC=20 forces more aggressive GC to prevent memory accumulation
	if gogc := os.Getenv("GOGC"); gogc != "" {
		log.Printf("GOGC environment variable set to: %s (more aggressive GC)", gogc)
	} else {
		log.Printf("GOGC not set, using default (100). Set GOGC=20 for aggressive GC if memory issues occur")
	}

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

	// Only use workers from config if explicitly set via CLI or env var
	// Default is 1, not from config file
	if !workersFromCLI && !workersFromEnv && config.Processing.MaxWorkers > 0 {
		// Config file is lowest priority - only use if nothing else specified
		// But since we want default=1, we skip config file entirely
		log.Printf("Ignoring workers from config (%d), using default: %d", config.Processing.MaxWorkers, *numWorkers)
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

	// Create semantic relationship config if enabled
	var semanticConfig *worker.SemanticConfig
	if config.RelationshipDetection.CrossDocumentSemantic.Enabled {
		semanticConfig = &worker.SemanticConfig{
			Enabled:                  config.RelationshipDetection.CrossDocumentSemantic.Enabled,
			SimilarityThreshold:      config.RelationshipDetection.CrossDocumentSemantic.SimilarityThreshold,
			QueueIdleTriggerMinutes:  config.RelationshipDetection.CrossDocumentSemantic.QueueIdleTriggerMinutes,
			RateLimitBatchSize:       config.RelationshipDetection.CrossDocumentSemantic.RateLimitBatchSize,
			RateLimitSleepMs:         config.RelationshipDetection.CrossDocumentSemantic.RateLimitSleepMs,
			MinIntervalMinutes:       config.RelationshipDetection.CrossDocumentSemantic.MinIntervalMinutes,
		}
	}

	// Create ontology extraction config if enabled
	var ontologyConfig *worker.OntologyConfig
	if config.Ontology.Enabled {
		ontologyConfig = &worker.OntologyConfig{
			Enabled:                 config.Ontology.Enabled,
			SchemaPath:              config.Ontology.SchemaPath,
			DiversityThreshold:      config.Ontology.DiversityThreshold,
			QueueIdleTriggerMinutes: config.Ontology.QueueIdleTriggerMinutes,
			MinIntervalMinutes:      config.Ontology.MinIntervalMinutes,
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
		SemanticConfig:    semanticConfig,
		OntologyConfig:    ontologyConfig,
		ShutdownWhenIdle:  *shutdownWhenIdle,
		IdleTimeout:       *idleTimeout,
		IdleCheckInterval: 0, // 0 triggers default of 5 minutes in NewWorker
	}

	// Create worker
	w, err := worker.NewWorker(workerConfig)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}
	defer w.Close()

	// Setup shutdown mechanisms
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Signal handling (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, initiating graceful shutdown...", sig)
		cancel()
	}()

	// 2. Shutdown file monitoring
	if *shutdownFile != "" {
		log.Printf("Shutdown file monitoring enabled: %s", *shutdownFile)
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, err := os.Stat(*shutdownFile); err == nil {
						log.Printf("Shutdown file detected: %s - initiating shutdown...", *shutdownFile)
						cancel()
						return
					}
				}
			}
		}()
	}

	// 3. Idle timeout monitoring
	if *idleTimeout > 0 {
		log.Printf("Idle timeout enabled: %v", *idleTimeout)
		go func() {
			timer := time.NewTimer(*idleTimeout)
			defer timer.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					log.Printf("Idle timeout reached (%v) - initiating shutdown...", *idleTimeout)
					cancel()
					return
				}
			}
		}()
	}

	// Run worker
	log.Printf("========================================")
	log.Printf("STARTING WORKER: %s", *workerID)
	log.Printf("  Max documents: %d", *maxDocuments)
	log.Printf("  Goroutine workers: %d", *numWorkers)
	if *shutdownWhenIdle {
		log.Printf("  Shutdown when idle: enabled")
	}
	if *shutdownFile != "" {
		log.Printf("  Shutdown file: %s", *shutdownFile)
	}
	if *idleTimeout > 0 {
		log.Printf("  Idle timeout: %v", *idleTimeout)
	}
	log.Printf("========================================")

	// Run worker with context (worker will need to be updated to accept context)
	// For now, we'll run it in a goroutine and wait for completion or cancellation
	done := make(chan error, 1)
	go func() {
		done <- w.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
		log.Println("========================================")
		log.Println("WORKER COMPLETED SUCCESSFULLY")
		log.Println("========================================")
	case <-ctx.Done():
		log.Println("========================================")
		log.Println("SHUTDOWN INITIATED - Waiting for graceful cleanup...")
		log.Println("========================================")
		// Give worker time to finish current task
		select {
		case err := <-done:
			if err != nil {
				log.Printf("Worker exited with error: %v", err)
			} else {
				log.Println("Worker stopped gracefully")
			}
		case <-time.After(30 * time.Second):
			log.Println("Timeout waiting for worker shutdown - forcing exit")
		}
	}
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

	log.Printf("DEBUG: About to start spawn loop with numInstances=%d", numInstances)
	for i := 0; i < numInstances; i++ {
		log.Printf("DEBUG: Spawn loop iteration %d (starting instance %d/%d)", i, i+1, numInstances)
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
	// Relative paths are resolved from the current working directory, not the config file location
	backend := strings.ToLower(config.Processing.JobControl.Backend)
	if backend == "" || backend == "sqlite" {
		if !filepath.IsAbs(config.Processing.JobControl.Path) {
			// Get current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to get current working directory: %w", err)
			}
			// Resolve relative path from current working directory
			config.Processing.JobControl.Path = filepath.Join(cwd, config.Processing.JobControl.Path)
		}
	}

	return &config, nil
}
