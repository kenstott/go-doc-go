package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kennethstott/go-doc-go/internal/embeddings"
	"github.com/kennethstott/go-doc-go/internal/jobcontrol"
	"github.com/kennethstott/go-doc-go/internal/worker"
	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration structure
type YAMLConfig struct {
	Processing struct {
		JobControl struct {
			Path              string `yaml:"path"`
			ClaimTimeout      int    `yaml:"claim_timeout"`
			HeartbeatInterval int    `yaml:"heartbeat_interval"`
			MaxRetries        int    `yaml:"max_retries"`
		} `yaml:"job_control"`
	} `yaml:"processing"`
	ContentSources []map[string]interface{} `yaml:"content_sources"`
	Analytics      struct {
		Enabled bool                     `yaml:"enabled"`
		Outputs []map[string]interface{} `yaml:"outputs"`
	} `yaml:"analytics"`
	Embedding *embeddings.Config `yaml:"embedding"`
}

func main() {
	// Command line flags
	var (
		configFile     = flag.String("config", "", "Path to configuration file")
		workerID       = flag.String("worker-id", "", "Custom worker ID (auto-generated if not provided)")
		maxDocuments   = flag.Int("max-documents", 0, "Maximum number of documents to process (0 = unlimited)")
		numWorkers     = flag.Int("workers", 0, "Number of concurrent goroutine workers (0 = use NUM_WORKERS env var, default: 1)")
		batchClaimSize = flag.Int("batch-claim-size", 10, "Number of documents to claim at once (default: 10)")
	)

	flag.Parse()

	// Get workers from environment variable if not specified via CLI
	if *numWorkers == 0 {
		if envWorkers := os.Getenv("NUM_WORKERS"); envWorkers != "" {
			if parsed, err := fmt.Sscanf(envWorkers, "%d", numWorkers); err == nil && parsed == 1 {
				log.Printf("Using NUM_WORKERS from environment: %d", *numWorkers)
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
			configPath = "./config.yaml"
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

	// Generate worker ID if not provided
	if *workerID == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown"
		}
		*workerID = fmt.Sprintf("worker_%s_%d", hostname, os.Getpid())
	}

	// Create worker configuration
	workerConfig := worker.Config{
		WorkerID:       *workerID,
		NumWorkers:     *numWorkers,
		BatchClaimSize: *batchClaimSize,
		JobControlConfig: jobcontrol.Config{
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
	log.Printf("  Batch claim size: %d", *batchClaimSize)
	log.Printf("========================================")

	if err := w.Run(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}

	log.Println("========================================")
	log.Println("WORKER COMPLETED SUCCESSFULLY")
	log.Println("========================================")
}

func loadConfig(path string) (*YAMLConfig, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config YAMLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
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

	// Expand paths to be absolute
	if !filepath.IsAbs(config.Processing.JobControl.Path) {
		configDir := filepath.Dir(path)
		config.Processing.JobControl.Path = filepath.Join(configDir, config.Processing.JobControl.Path)
	}

	return &config, nil
}
