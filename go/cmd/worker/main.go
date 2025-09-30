package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
}

func main() {
	// Command line flags
	var (
		configFile     = flag.String("config", "", "Path to configuration file")
		workerID       = flag.String("worker-id", "", "Custom worker ID (auto-generated if not provided)")
		maxDocuments   = flag.Int("max-documents", 0, "Maximum number of documents to process (0 = unlimited)")
		numWorkers     = flag.Int("workers", 1, "Number of concurrent goroutine workers (default: 1)")
		batchClaimSize = flag.Int("batch-claim-size", 10, "Number of documents to claim at once (default: 10)")
	)

	flag.Parse()

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
	log.Printf("Starting worker: %s", *workerID)
	if err := w.Run(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}

	log.Println("Worker completed successfully")
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
