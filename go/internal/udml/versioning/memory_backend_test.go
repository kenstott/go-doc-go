package versioning

import (
	"context"
	"testing"
	"time"
)

func TestMemoryBackend_CreateAndGetVersion(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	meta := &VersionMetadata{
		Version:         "1.0.0",
		CreatedAt:       time.Now(),
		ParserVersion:   "2.1.0",
		EmbeddingModel:  "text-embedding-ada-002",
		ElementCount:    1000,
		DocumentCount:   10,
		SchemaChanges:   map[string]interface{}{"added_fields": []string{"new_field"}},
		Tags:            []string{"production"},
	}

	// Create version
	err := backend.CreateVersion(ctx, meta)
	if err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Retrieve version
	retrieved, err := backend.GetVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	// Verify data
	if retrieved.Version != meta.Version {
		t.Errorf("Expected version %s, got %s", meta.Version, retrieved.Version)
	}
	if retrieved.ParserVersion != meta.ParserVersion {
		t.Errorf("Expected parser version %s, got %s", meta.ParserVersion, retrieved.ParserVersion)
	}
	if retrieved.ElementCount != meta.ElementCount {
		t.Errorf("Expected element count %d, got %d", meta.ElementCount, retrieved.ElementCount)
	}
}

func TestMemoryBackend_CreateDuplicateVersion(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	meta := &VersionMetadata{
		Version:   "1.0.0",
		CreatedAt: time.Now(),
	}

	// Create first time
	err := backend.CreateVersion(ctx, meta)
	if err != nil {
		t.Fatalf("First CreateVersion failed: %v", err)
	}

	// Create second time (should fail)
	err = backend.CreateVersion(ctx, meta)
	if err == nil {
		t.Error("Expected error for duplicate version, got nil")
	}
}

func TestMemoryBackend_UpdateVersion(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	meta := &VersionMetadata{
		Version:       "1.0.0",
		CreatedAt:     time.Now(),
		ElementCount:  1000,
		DocumentCount: 10,
	}

	// Create version
	backend.CreateVersion(ctx, meta)

	// Update version
	meta.ElementCount = 2000
	meta.DocumentCount = 20
	meta.Notes = "Updated"

	err := backend.UpdateVersion(ctx, meta)
	if err != nil {
		t.Fatalf("UpdateVersion failed: %v", err)
	}

	// Verify update
	retrieved, err := backend.GetVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if retrieved.ElementCount != 2000 {
		t.Errorf("Expected element count 2000, got %d", retrieved.ElementCount)
	}
	if retrieved.DocumentCount != 20 {
		t.Errorf("Expected document count 20, got %d", retrieved.DocumentCount)
	}
	if retrieved.Notes != "Updated" {
		t.Errorf("Expected notes 'Updated', got %s", retrieved.Notes)
	}
}

func TestMemoryBackend_DeleteVersion(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	meta := &VersionMetadata{
		Version:   "1.0.0",
		CreatedAt: time.Now(),
	}

	// Create version
	backend.CreateVersion(ctx, meta)

	// Delete version
	err := backend.DeleteVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("DeleteVersion failed: %v", err)
	}

	// Verify deletion
	_, err = backend.GetVersion(ctx, "1.0.0")
	if err == nil {
		t.Error("Expected error for deleted version, got nil")
	}
}

func TestMemoryBackend_ListVersions(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	// Create multiple versions
	versions := []string{"1.0.0", "1.1.0", "2.0.0"}
	for i, v := range versions {
		meta := &VersionMetadata{
			Version:   v,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Hour),
			Tags:      []string{},
		}
		backend.CreateVersion(ctx, meta)
	}

	// List all versions
	list, err := backend.ListVersions(ctx, nil)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(list))
	}

	// Verify sorted by creation time descending
	if list[0].Version != "2.0.0" {
		t.Errorf("Expected first version to be 2.0.0, got %s", list[0].Version)
	}
}

func TestMemoryBackend_ListVersionsWithTagFilter(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	// Create versions with different tags
	meta1 := &VersionMetadata{
		Version:   "1.0.0",
		CreatedAt: time.Now(),
		Tags:      []string{"production"},
	}
	meta2 := &VersionMetadata{
		Version:   "1.1.0",
		CreatedAt: time.Now(),
		Tags:      []string{"beta"},
	}
	meta3 := &VersionMetadata{
		Version:   "2.0.0",
		CreatedAt: time.Now(),
		Tags:      []string{"production", "latest"},
	}

	backend.CreateVersion(ctx, meta1)
	backend.CreateVersion(ctx, meta2)
	backend.CreateVersion(ctx, meta3)

	// Query by tag
	query := &VersionQuery{
		Tags: []string{"production"},
	}
	list, err := backend.ListVersions(ctx, query)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("Expected 2 production versions, got %d", len(list))
	}
}

func TestMemoryBackend_GetVersionAtTime(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	now := time.Now()

	// Create versions at different times
	meta1 := &VersionMetadata{
		Version:   "1.0.0",
		CreatedAt: now.Add(-2 * time.Hour),
	}
	meta2 := &VersionMetadata{
		Version:   "1.1.0",
		CreatedAt: now.Add(-1 * time.Hour),
	}
	meta3 := &VersionMetadata{
		Version:   "2.0.0",
		CreatedAt: now,
	}

	backend.CreateVersion(ctx, meta1)
	backend.CreateVersion(ctx, meta2)
	backend.CreateVersion(ctx, meta3)

	// Query version at 90 minutes ago (should return 1.0.0)
	timestamp := now.Add(-90 * time.Minute)
	version, err := backend.GetVersionAtTime(ctx, timestamp)
	if err != nil {
		t.Fatalf("GetVersionAtTime failed: %v", err)
	}

	if version.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", version.Version)
	}

	// Query version at 30 minutes ago (should return 1.1.0)
	timestamp = now.Add(-30 * time.Minute)
	version, err = backend.GetVersionAtTime(ctx, timestamp)
	if err != nil {
		t.Fatalf("GetVersionAtTime failed: %v", err)
	}

	if version.Version != "1.1.0" {
		t.Errorf("Expected version 1.1.0, got %s", version.Version)
	}
}

func TestMemoryBackend_GetVersionsInRange(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	now := time.Now()

	// Create versions at different times
	meta1 := &VersionMetadata{Version: "1.0.0", CreatedAt: now.Add(-3 * time.Hour)}
	meta2 := &VersionMetadata{Version: "1.1.0", CreatedAt: now.Add(-2 * time.Hour)}
	meta3 := &VersionMetadata{Version: "1.2.0", CreatedAt: now.Add(-1 * time.Hour)}
	meta4 := &VersionMetadata{Version: "2.0.0", CreatedAt: now}

	backend.CreateVersion(ctx, meta1)
	backend.CreateVersion(ctx, meta2)
	backend.CreateVersion(ctx, meta3)
	backend.CreateVersion(ctx, meta4)

	// Query versions between -2.5 hours and -0.5 hours
	start := now.Add(-150 * time.Minute)
	end := now.Add(-30 * time.Minute)

	versions, err := backend.GetVersionsInRange(ctx, start, end)
	if err != nil {
		t.Fatalf("GetVersionsInRange failed: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("Expected 2 versions in range, got %d", len(versions))
	}

	// Verify versions are in ascending order
	if versions[0].Version != "1.1.0" {
		t.Errorf("Expected first version 1.1.0, got %s", versions[0].Version)
	}
	if versions[1].Version != "1.2.0" {
		t.Errorf("Expected second version 1.2.0, got %s", versions[1].Version)
	}
}

func TestMemoryBackend_ComputeDiff(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	// Create two versions with different stats
	meta1 := &VersionMetadata{
		Version:       "1.0.0",
		CreatedAt:     time.Now(),
		ElementCount:  1000,
		DocumentCount: 10,
		SizeBytes:     1024 * 1024,
	}
	meta2 := &VersionMetadata{
		Version:       "1.1.0",
		CreatedAt:     time.Now(),
		ElementCount:  1500,
		DocumentCount: 15,
		SizeBytes:     1536 * 1024,
	}

	backend.CreateVersion(ctx, meta1)
	backend.CreateVersion(ctx, meta2)

	// Compute diff
	diff, err := backend.ComputeDiff(ctx, "1.0.0", "1.1.0")
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	if diff.FromVersion != "1.0.0" {
		t.Errorf("Expected from version 1.0.0, got %s", diff.FromVersion)
	}
	if diff.ToVersion != "1.1.0" {
		t.Errorf("Expected to version 1.1.0, got %s", diff.ToVersion)
	}
	if diff.ElementsAdded != 500 {
		t.Errorf("Expected 500 elements added, got %d", diff.ElementsAdded)
	}
	if diff.DocumentsAdded != 5 {
		t.Errorf("Expected 5 documents added, got %d", diff.DocumentsAdded)
	}
}

func TestMemoryBackend_GetStats(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	meta := &VersionMetadata{
		Version:       "1.0.0",
		CreatedAt:     time.Now(),
		ElementCount:  1000,
		DocumentCount: 10,
		SizeBytes:     1024 * 1024,
	}

	backend.CreateVersion(ctx, meta)

	stats, err := backend.GetStats(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", stats.Version)
	}
	if stats.TotalElements != 1000 {
		t.Errorf("Expected 1000 elements, got %d", stats.TotalElements)
	}
	if stats.TotalDocuments != 10 {
		t.Errorf("Expected 10 documents, got %d", stats.TotalDocuments)
	}
}

func TestMemoryBackend_Migrations(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	migration := &MigrationRecord{
		FromVersion:      "1.0.0",
		ToVersion:        "2.0.0",
		StartedAt:        time.Now(),
		Status:           MigrationPending,
		TotalRecords:     1000,
		ProcessedRecords: 0,
	}

	// Create migration
	err := backend.CreateMigration(ctx, migration)
	if err != nil {
		t.Fatalf("CreateMigration failed: %v", err)
	}

	// Get migration
	retrieved, err := backend.GetMigration(ctx, "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("GetMigration failed: %v", err)
	}

	if retrieved.FromVersion != "1.0.0" {
		t.Errorf("Expected from version 1.0.0, got %s", retrieved.FromVersion)
	}
	if retrieved.Status != MigrationPending {
		t.Errorf("Expected pending status, got %s", retrieved.Status)
	}

	// Update migration
	retrieved.Status = MigrationRunning
	retrieved.ProcessedRecords = 500
	err = backend.UpdateMigration(ctx, retrieved)
	if err != nil {
		t.Fatalf("UpdateMigration failed: %v", err)
	}

	// Verify update
	updated, err := backend.GetMigration(ctx, "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("GetMigration after update failed: %v", err)
	}

	if updated.Status != MigrationRunning {
		t.Errorf("Expected running status, got %s", updated.Status)
	}
	if updated.ProcessedRecords != 500 {
		t.Errorf("Expected 500 processed records, got %d", updated.ProcessedRecords)
	}

	// List migrations
	migrations, err := backend.ListMigrations(ctx)
	if err != nil {
		t.Fatalf("ListMigrations failed: %v", err)
	}

	if len(migrations) != 1 {
		t.Errorf("Expected 1 migration, got %d", len(migrations))
	}
}

func TestMemoryBackend_ConcurrentAccess(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	// Create initial version
	meta := &VersionMetadata{
		Version:   "1.0.0",
		CreatedAt: time.Now(),
	}
	backend.CreateVersion(ctx, meta)

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := backend.GetVersion(ctx, "1.0.0")
			if err != nil {
				t.Errorf("Concurrent GetVersion failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
