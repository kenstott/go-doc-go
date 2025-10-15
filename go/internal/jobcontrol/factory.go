package jobcontrol

import (
	"fmt"
	"log"
	"strings"
)

// NewJobControl creates a job control instance based on the backend configuration
func NewJobControl(config Config) (JobControl, error) {
	// Determine backend (default to sqlite)
	backend := strings.ToLower(config.Backend)
	if backend == "" {
		backend = "sqlite"
	}

	log.Printf("Creating job control with backend: %s", backend)

	switch backend {
	case "sqlite":
		return NewSQLiteJobControl(config)
	case "postgres", "postgresql":
		return NewPostgreSQLJobControl(config)
	default:
		return nil, fmt.Errorf("unsupported job control backend: %s (supported: sqlite, postgres)", backend)
	}
}
