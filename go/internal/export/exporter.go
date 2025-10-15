package export

import (
	"fmt"
	"io"

	"github.com/kennethstott/doculyzer-go-conversion/internal/graph"
)

// Exporter defines the interface for exporting graphs to different formats
type Exporter interface {
	// Name returns the exporter identifier (e.g., "rdf", "jsonld", "graphml")
	Name() string

	// Description returns a human-readable description of the export format
	Description() string

	// FileExtension returns the recommended file extension (e.g., ".ttl", ".jsonld", ".graphml")
	FileExtension() string

	// Export exports a graph to the given writer
	Export(g *graph.Graph, w io.Writer) error

	// SupportsFeature checks if the exporter supports a specific feature
	SupportsFeature(feature string) bool
}

// ExportFeature defines standard features that exporters may support
const (
	FeatureProvenance      = "provenance"       // UDML element tracking
	FeatureConfidence      = "confidence"       // Confidence scores
	FeatureDomains         = "domains"          // Data mesh domains
	FeatureTimestamps      = "timestamps"       // Created/Updated timestamps
	FeatureProperties      = "properties"       // Arbitrary property bags
	FeatureMultiLabel      = "multi_label"      // Multiple labels per node
	FeatureDirectedEdges   = "directed_edges"   // Directed relationships
	FeatureUndirectedEdges = "undirected_edges" // Undirected relationships
)

// ExporterRegistry manages available exporters
type ExporterRegistry struct {
	exporters map[string]Exporter
}

// NewExporterRegistry creates a new exporter registry
func NewExporterRegistry() *ExporterRegistry {
	return &ExporterRegistry{
		exporters: make(map[string]Exporter),
	}
}

// Register registers an exporter
func (r *ExporterRegistry) Register(exporter Exporter) error {
	name := exporter.Name()
	if name == "" {
		return fmt.Errorf("exporter name cannot be empty")
	}
	if _, exists := r.exporters[name]; exists {
		return fmt.Errorf("exporter %s already registered", name)
	}
	r.exporters[name] = exporter
	return nil
}

// Get retrieves an exporter by name
func (r *ExporterRegistry) Get(name string) (Exporter, error) {
	exporter, exists := r.exporters[name]
	if !exists {
		return nil, fmt.Errorf("exporter %s not found", name)
	}
	return exporter, nil
}

// ListExporters returns all registered exporter names
func (r *ExporterRegistry) ListExporters() []string {
	names := make([]string, 0, len(r.exporters))
	for name := range r.exporters {
		names = append(names, name)
	}
	return names
}

// GetExporterInfo returns information about a registered exporter
func (r *ExporterRegistry) GetExporterInfo(name string) (string, string, error) {
	exporter, err := r.Get(name)
	if err != nil {
		return "", "", err
	}
	return exporter.Description(), exporter.FileExtension(), nil
}

// Global registry instance
var globalRegistry = NewExporterRegistry()

// RegisterExporter registers an exporter in the global registry
func RegisterExporter(exporter Exporter) error {
	return globalRegistry.Register(exporter)
}

// GetExporter retrieves an exporter from the global registry
func GetExporter(name string) (Exporter, error) {
	return globalRegistry.Get(name)
}

// ListExporters returns all registered exporter names from the global registry
func ListExporters() []string {
	return globalRegistry.ListExporters()
}

// ExportToFile is a convenience function to export a graph to a file
func ExportToFile(g *graph.Graph, format string, w io.Writer) error {
	exporter, err := GetExporter(format)
	if err != nil {
		return fmt.Errorf("failed to get exporter: %w", err)
	}
	return exporter.Export(g, w)
}
