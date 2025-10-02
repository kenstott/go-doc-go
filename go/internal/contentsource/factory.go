package contentsource

import (
	"fmt"
)

// NewContentSource creates a content source based on the configuration
func NewContentSource(config map[string]interface{}) (ContentSource, error) {
	// Determine source type
	sourceType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'type' field in content source configuration")
	}

	// Create appropriate source
	switch sourceType {
	case "file":
		return NewFileContentSource(config)
	case "web":
		return NewWebContentSource(config)
	case "s3":
		return NewS3ContentSource(config)
	default:
		// Fall back to Python shim for unsupported types
		name, _ := config["name"].(string)
		return NewPythonShimSource(name, config)
	}
}
