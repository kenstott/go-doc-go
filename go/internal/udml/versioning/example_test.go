package versioning_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/versioning"
)

// Example_basicVersionManagement demonstrates creating and retrieving versions
func Example_basicVersionManagement() {
	// Create version manager with memory backend
	backend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create initial UDML version
	v1 := &versioning.VersionMetadata{
		Version:         "1.0.0",
		ParserVersion:   "2.1.0",
		EmbeddingModel:  "text-embedding-ada-002",
		OntologyVersion: "1.0.0",
		Notes:           "Initial UDML release with 6 promoted fields",
		ElementCount:    1000000,
		DocumentCount:   5000,
		SizeBytes:       10 * 1024 * 1024 * 1024, // 10GB
		CreatedBy:       "system",
		Tags:            []string{"production", "stable"},
	}

	if err := vm.CreateVersion(ctx, v1); err != nil {
		log.Fatal(err)
	}

	// Retrieve version
	retrieved, err := vm.GetVersion(ctx, "1.0.0")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Version: %s\n", retrieved.Version)
	fmt.Printf("Parser: %s\n", retrieved.ParserVersion)
	fmt.Printf("Elements: %d\n", retrieved.ElementCount)

	// Output:
	// Version: 1.0.0
	// Parser: 2.1.0
	// Elements: 1000000
}

// Example_schemaUpgrade demonstrates upgrading UDML schema
func Example_schemaUpgrade() {
	backend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Version 1.0.0: Initial schema with 6 promoted fields
	v1 := &versioning.VersionMetadata{
		Version:         "1.0.0",
		ParserVersion:   "2.1.0",
		OntologyVersion: "1.0.0",
		SchemaChanges: map[string]interface{}{
			"promoted_fields": []string{
				"page_number", "section_level", "row_index",
				"column_index", "temporal_type", "tag_name",
			},
		},
		Notes:         "Initial UDML with 6 promoted fields",
		ElementCount:  1000000,
		DocumentCount: 5000,
		Tags:          []string{"production"},
	}
	vm.CreateVersion(ctx, v1)

	// Version 1.1.0: Add 2 new promoted fields
	time.Sleep(100 * time.Millisecond) // Ensure different timestamp
	v2 := &versioning.VersionMetadata{
		Version:         "1.1.0",
		ParserVersion:   "2.2.0",
		OntologyVersion: "1.1.0",
		SchemaChanges: map[string]interface{}{
			"promoted_fields": []string{
				"page_number", "section_level", "row_index",
				"column_index", "temporal_type", "tag_name",
				"speaker_id", "geo_location", // NEW fields
			},
			"fields_added": []string{"speaker_id", "geo_location"},
		},
		Notes:         "Added speaker and geolocation support for transcripts",
		ElementCount:  1200000,
		DocumentCount: 6000,
		Tags:          []string{"production", "latest"},
	}
	vm.CreateVersion(ctx, v2)

	// Compare versions
	diff, err := vm.CompareVersions(ctx, "1.0.0", "1.1.0")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("From: %s\n", diff.FromVersion)
	fmt.Printf("To: %s\n", diff.ToVersion)
	fmt.Printf("Elements added: %d\n", diff.ElementsAdded)
	fmt.Printf("Documents added: %d\n", diff.DocumentsAdded)

	// Output:
	// From: 1.0.0
	// To: 1.1.0
	// Elements added: 200000
	// Documents added: 1000
}

// Example_timeTravel demonstrates querying historical versions
func Example_timeTravel() {
	backend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()
	now := time.Now()

	// Create versions at different times
	v1 := &versioning.VersionMetadata{
		Version:   "1.0.0",
		CreatedAt: now.Add(-2 * time.Hour),
		Notes:     "Initial release",
	}
	v2 := &versioning.VersionMetadata{
		Version:   "1.1.0",
		CreatedAt: now.Add(-1 * time.Hour),
		Notes:     "Added geolocation",
	}
	v3 := &versioning.VersionMetadata{
		Version:   "2.0.0",
		CreatedAt: now,
		Notes:     "Major schema refactor",
	}

	vm.CreateVersion(ctx, v1)
	vm.CreateVersion(ctx, v2)
	vm.CreateVersion(ctx, v3)

	// Time-travel: What version was active 90 minutes ago?
	timestamp := now.Add(-90 * time.Minute)
	version, err := vm.GetVersionAtTime(ctx, timestamp)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Version at %s: %s\n", "90 min ago", version.Version)
	fmt.Printf("Notes: %s\n", version.Notes)

	// Get latest version
	latest, _ := vm.GetLatestVersion(ctx)
	fmt.Printf("Latest: %s\n", latest.Version)

	// Output:
	// Version at 90 min ago: 1.0.0
	// Notes: Initial release
	// Latest: 2.0.0
}

// Example_versionTags demonstrates tagging and filtering versions
func Example_versionTags() {
	backend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create versions with different tags
	v1 := &versioning.VersionMetadata{
		Version: "1.0.0",
		Tags:    []string{"production", "stable"},
	}
	v2 := &versioning.VersionMetadata{
		Version: "1.1.0-beta",
		Tags:    []string{"beta", "testing"},
	}
	v3 := &versioning.VersionMetadata{
		Version: "2.0.0",
		Tags:    []string{"production", "latest"},
	}

	vm.CreateVersion(ctx, v1)
	vm.CreateVersion(ctx, v2)
	vm.CreateVersion(ctx, v3)

	// Query production versions only
	query := &versioning.VersionQuery{
		Tags: []string{"production"},
	}
	prodVersions, err := vm.ListVersions(ctx, query)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Production versions: %d\n", len(prodVersions))
	for _, v := range prodVersions {
		fmt.Printf("  - %s\n", v.Version)
	}

	// Output:
	// Production versions: 2
	//   - 2.0.0
	//   - 1.0.0
}

// Example_versionDeprecation demonstrates deprecating old versions
func Example_versionDeprecation() {
	backend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create old version
	v1 := &versioning.VersionMetadata{
		Version: "1.0.0",
		Notes:   "Old version",
	}
	vm.CreateVersion(ctx, v1)

	// Deprecate it
	err := vm.DeprecateVersion(ctx, "1.0.0", "Superseded by 2.0.0 with better performance")
	if err != nil {
		log.Fatal(err)
	}

	// Check deprecation status
	deprecated, _ := vm.GetVersion(ctx, "1.0.0")
	fmt.Printf("Deprecated: %v\n", deprecated.Deprecated)
	fmt.Printf("Has deprecation time: %v\n", deprecated.DeprecatedAt != nil)

	// Output:
	// Deprecated: true
	// Has deprecation time: true
}

// Example_migrationTracking demonstrates tracking version migrations
func Example_migrationTracking() {
	backend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create versions
	v1 := &versioning.VersionMetadata{Version: "1.0.0"}
	v2 := &versioning.VersionMetadata{Version: "2.0.0"}
	vm.CreateVersion(ctx, v1)
	vm.CreateVersion(ctx, v2)

	// Start migration
	migration, err := vm.CreateMigration(ctx, "1.0.0", "2.0.0")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Migration: %s -> %s\n", migration.FromVersion, migration.ToVersion)
	fmt.Printf("Status: %s\n", migration.Status)

	// Update migration progress
	migration.Status = versioning.MigrationRunning
	migration.TotalRecords = 1000000
	migration.ProcessedRecords = 500000
	vm.UpdateMigration(ctx, migration)

	// Check progress
	updated, _ := vm.GetMigration(ctx, "1.0.0", "2.0.0")
	fmt.Printf("Progress: %d/%d\n", updated.ProcessedRecords, updated.TotalRecords)

	// Output:
	// Migration: 1.0.0 -> 2.0.0
	// Status: pending
	// Progress: 500000/1000000
}

// Example_versionStats demonstrates getting version statistics
func Example_versionStats() {
	backend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create version with stats
	v1 := &versioning.VersionMetadata{
		Version:       "1.0.0",
		ElementCount:  1000000,
		DocumentCount: 5000,
		SizeBytes:     10 * 1024 * 1024 * 1024, // 10GB
	}
	vm.CreateVersion(ctx, v1)

	// Get statistics
	stats, err := vm.GetStats(ctx, "1.0.0")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Elements: %d\n", stats.TotalElements)
	fmt.Printf("Documents: %d\n", stats.TotalDocuments)
	fmt.Printf("Size: %.2f GB\n", float64(stats.SizeBytes)/(1024*1024*1024))

	// Output:
	// Elements: 1000000
	// Documents: 5000
	// Size: 10.00 GB
}

// Example_realWorldWorkflow demonstrates a complete UDML upgrade workflow
func Example_realWorldWorkflow() {
	backend := versioning.NewMemoryBackend()
	vm := versioning.NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Step 1: Deploy new UDML version
	newVersion := &versioning.VersionMetadata{
		Version:         "1.2.0",
		ParserVersion:   "2.3.0",
		EmbeddingModel:  "text-embedding-3-small",
		OntologyVersion: "1.2.0",
		SchemaChanges: map[string]interface{}{
			"embedding_model_changed": "ada-002 -> embedding-3-small",
			"parser_improvements":     "Better table extraction",
		},
		Notes:         "Upgraded to GPT-4 era embeddings",
		CreatedBy:     "deploy-bot",
		Tags:          []string{"production"},
		ElementCount:  0, // Will grow as documents are reprocessed
		DocumentCount: 0,
	}

	if err := vm.CreateVersion(ctx, newVersion); err != nil {
		log.Fatal(err)
	}

	// Step 2: Tag as current production version
	vm.TagVersion(ctx, "1.2.0", "current", "latest")

	// Step 3: Deprecate old version
	vm.DeprecateVersion(ctx, "1.0.0", "Use 1.2.0 for better embeddings")

	// Step 4: Start migration (create old version first for migration)
	oldVersion := &versioning.VersionMetadata{
		Version:       "1.0.0",
		ElementCount:  1000000,
		DocumentCount: 5000,
	}
	vm.CreateVersion(ctx, oldVersion)

	migration, _ := vm.CreateMigration(ctx, "1.0.0", "1.2.0")
	migration.Status = versioning.MigrationRunning
	migration.TotalRecords = 5000
	vm.UpdateMigration(ctx, migration)

	// Step 5: Track reprocessing progress
	// (In real code, this would be updated by the ingestion worker)
	newVersion.ElementCount = 50000
	newVersion.DocumentCount = 250
	vm.UpdateVersion(ctx, newVersion)

	// Step 6: Check deployment status
	current, _ := vm.GetVersion(ctx, "1.2.0")
	fmt.Printf("Version %s deployed\n", current.Version)
	fmt.Printf("Embedding model: %s\n", current.EmbeddingModel)
	fmt.Printf("Documents processed: %d\n", current.DocumentCount)

	// Output:
	// Version 1.2.0 deployed
	// Embedding model: text-embedding-3-small
	// Documents processed: 250
}
