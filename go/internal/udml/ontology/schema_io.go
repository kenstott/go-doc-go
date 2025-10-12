package ontology

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaFormat represents the serialization format for ontology schemas
type SchemaFormat string

const (
	FormatJSON SchemaFormat = "json"
	FormatYAML SchemaFormat = "yaml"
	FormatAuto SchemaFormat = "auto" // Detect from file extension
)

// SaveSchema saves an ontology schema to a file
// Format is detected from file extension (.json, .yaml, .yml) or can be specified explicitly
func SaveSchema(schema *OntologySchema, path string, format SchemaFormat) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Detect format from extension if auto
	if format == FormatAuto || format == "" {
		format = detectFormatFromPath(path)
	}

	// Marshal schema
	var data []byte
	var err error

	switch format {
	case FormatJSON:
		data, err = json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal schema as JSON: %w", err)
		}
	case FormatYAML:
		data, err = yaml.Marshal(schema)
		if err != nil {
			return fmt.Errorf("failed to marshal schema as YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s (use 'json' or 'yaml')", format)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	return nil
}

// LoadSchema loads an ontology schema from a file
// Format is detected from file extension (.json, .yaml, .yml) or can be specified explicitly
func LoadSchema(path string, format SchemaFormat) (*OntologySchema, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Check file exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("schema file not found: %w", err)
	}

	// Detect format from extension if auto
	if format == FormatAuto || format == "" {
		format = detectFormatFromPath(path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Unmarshal schema
	var schema OntologySchema

	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, &schema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema as JSON: %w", err)
		}
	case FormatYAML:
		if err := yaml.Unmarshal(data, &schema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema as YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s (use 'json' or 'yaml')", format)
	}

	// Validate schema
	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("loaded schema is invalid: %w", err)
	}

	return &schema, nil
}

// detectFormatFromPath determines format from file extension
func detectFormatFromPath(path string) SchemaFormat {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return FormatJSON
	case ".yaml", ".yml":
		return FormatYAML
	default:
		// Default to YAML for unknown extensions
		return FormatYAML
	}
}

// SaveSchemaJSON saves schema as JSON with .json extension
func SaveSchemaJSON(schema *OntologySchema, path string) error {
	// Ensure .json extension
	if !strings.HasSuffix(path, ".json") {
		path = path + ".json"
	}
	return SaveSchema(schema, path, FormatJSON)
}

// SaveSchemaYAML saves schema as YAML with .yaml extension
func SaveSchemaYAML(schema *OntologySchema, path string) error {
	// Ensure .yaml extension
	if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
		path = path + ".yaml"
	}
	return SaveSchema(schema, path, FormatYAML)
}

// ValidateSchemaFile validates a schema file without loading it into memory
func ValidateSchemaFile(path string, format SchemaFormat) error {
	_, err := LoadSchema(path, format)
	return err
}
