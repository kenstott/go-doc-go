package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the TOML configuration structure
// Only includes fields needed by ontology commands
type Config struct {
	Analytics struct {
		Enabled bool                     `toml:"enabled"`
		Outputs []map[string]interface{} `toml:"outputs"`
	} `toml:"analytics"`
}

// loadConfig loads and parses a TOML configuration file
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

	return &config, nil
}

// getStoragePath extracts the storage path from analytics outputs
// Returns the path from the first analytics output
func getStoragePath(config *Config) (string, error) {
	if !config.Analytics.Enabled {
		return "", fmt.Errorf("analytics not enabled in config")
	}

	if len(config.Analytics.Outputs) == 0 {
		return "", fmt.Errorf("no analytics outputs configured")
	}

	// Get first output
	output := config.Analytics.Outputs[0]

	path, ok := output["path"].(string)
	if !ok {
		return "", fmt.Errorf("analytics output missing 'path' field")
	}

	return path, nil
}

// getStorageType extracts the storage type from analytics outputs
// Returns the type from the first analytics output (defaults to "parquet")
func getStorageType(config *Config) string {
	if len(config.Analytics.Outputs) == 0 {
		return "parquet"
	}

	output := config.Analytics.Outputs[0]

	if outputType, ok := output["type"].(string); ok {
		return outputType
	}

	return "parquet" // Default
}
