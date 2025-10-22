package graph

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

// SaveGraph saves a Graph to disk using GOB encoding
func SaveGraph(g *Graph, filepath string) error {
	if g == nil {
		return fmt.Errorf("cannot save nil graph")
	}

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filepath, err)
	}
	defer file.Close()

	// Create GOB encoder
	encoder := gob.NewEncoder(file)

	// Encode graph
	if err := encoder.Encode(g); err != nil {
		return fmt.Errorf("failed to encode graph: %w", err)
	}

	return nil
}

// LoadGraph loads a Graph from disk using GOB decoding
func LoadGraph(filepath string) (*Graph, error) {
	// Open file
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	// Create GOB decoder
	decoder := gob.NewDecoder(file)

	// Decode graph
	var g Graph
	if err := decoder.Decode(&g); err != nil {
		return nil, fmt.Errorf("failed to decode graph: %w", err)
	}

	return &g, nil
}

// SaveGonumGraph saves a GonumGraph to disk
// Note: This saves the underlying Graph structure, not the gonum-specific data
// To restore, use LoadGraph() then LoadFromGraph() to rebuild GonumGraph
func SaveGonumGraph(gg *GonumGraph, filepath string) error {
	if gg == nil {
		return fmt.Errorf("cannot save nil gonum graph")
	}

	// Convert to Graph and save
	g := gg.ToGraph()
	return SaveGraph(g, filepath)
}

// GraphSnapshot represents a graph with metadata about the snapshot
type GraphSnapshot struct {
	Graph       *Graph                 // The graph data
	CreatedAt   time.Time              // When this snapshot was created
	Description string                 // Optional description
	Metadata    map[string]interface{} // Additional snapshot metadata
}

// SaveSnapshot saves a graph with snapshot metadata
func SaveSnapshot(g *Graph, filepath string, description string) error {
	snapshot := GraphSnapshot{
		Graph:       g,
		CreatedAt:   time.Now(),
		Description: description,
		Metadata:    make(map[string]interface{}),
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file %s: %w", filepath, err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(&snapshot); err != nil {
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	return nil
}

// LoadSnapshot loads a graph snapshot from disk
func LoadSnapshot(filepath string) (*GraphSnapshot, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshot file %s: %w", filepath, err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	var snapshot GraphSnapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot: %w", err)
	}

	return &snapshot, nil
}
