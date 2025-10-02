package analytics

import (
	"fmt"
	"log"
)

// NewStorage creates a new analytics storage backend based on configuration
func NewStorage(config map[string]interface{}) (Storage, error) {
	// Determine storage type
	storageType, ok := config["type"].(string)
	if !ok {
		storageType = "parquet" // Default to Parquet
	}

	log.Printf("Creating analytics storage with type: %s", storageType)

	switch storageType {
	case "parquet":
		return NewParquetStorage(config)
	case "python_shim":
		// Fallback to Python shim for backward compatibility
		log.Println("WARNING: Using Python shim for analytics (deprecated, use native Parquet)")
		return NewPythonShimStorage(config)
	default:
		return nil, fmt.Errorf("unsupported analytics storage type: %s", storageType)
	}
}
