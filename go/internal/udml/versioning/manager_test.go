package versioning

import (
	"context"
	"testing"
	"time"
)

func TestVersionManager_CreateVersion(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	meta := &VersionMetadata{
		Version:         "1.0.0",
		ParserVersion:   "2.1.0",
		EmbeddingModel:  "text-embedding-ada-002",
		OntologyVersion: "1.0.0",
		Notes:           "Initial release",
		ElementCount:    1000,
		DocumentCount:   10,
		SizeBytes:       1024 * 1024,
		CreatedBy:       "system",
	}

	err := vm.CreateVersion(ctx, meta)
	if err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Verify version was created
	retrieved, err := vm.GetVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if retrieved.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", retrieved.Version)
	}
	if retrieved.ParserVersion != "2.1.0" {
		t.Errorf("Expected parser version 2.1.0, got %s", retrieved.ParserVersion)
	}
	if retrieved.ElementCount != 1000 {
		t.Errorf("Expected element count 1000, got %d", retrieved.ElementCount)
	}
}

func TestVersionManager_CreateVersionNilMetadata(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	err := vm.CreateVersion(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil metadata, got nil")
	}
}

func TestVersionManager_CreateVersionEmptyVersion(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	meta := &VersionMetadata{
		Version: "",
		Notes:   "Invalid version",
	}

	err := vm.CreateVersion(ctx, meta)
	if err == nil {
		t.Error("Expected error for empty version, got nil")
	}
}

func TestVersionManager_GetVersionNotFound(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	_, err := vm.GetVersion(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent version, got nil")
	}
}

func TestVersionManager_GetLatestVersion(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create multiple versions with different timestamps
	versions := []string{"1.0.0", "1.1.0", "1.2.0"}
	for i, v := range versions {
		meta := &VersionMetadata{
			Version:   v,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Hour),
			Notes:     "Version " + v,
		}
		if err := vm.CreateVersion(ctx, meta); err != nil {
			t.Fatalf("CreateVersion failed: %v", err)
		}
	}

	latest, err := vm.GetLatestVersion(ctx)
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}

	if latest.Version != "1.2.0" {
		t.Errorf("Expected latest version 1.2.0, got %s", latest.Version)
	}
}

func TestVersionManager_GetLatestVersionEmpty(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	_, err := vm.GetLatestVersion(ctx)
	if err == nil {
		t.Error("Expected error for empty version list, got nil")
	}
}

func TestVersionManager_UpdateVersion(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create initial version
	meta := &VersionMetadata{
		Version:       "1.0.0",
		Notes:         "Initial",
		ElementCount:  100,
		DocumentCount: 5,
	}
	if err := vm.CreateVersion(ctx, meta); err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Update version
	meta.Notes = "Updated notes"
	meta.ElementCount = 200
	if err := vm.UpdateVersion(ctx, meta); err != nil {
		t.Fatalf("UpdateVersion failed: %v", err)
	}

	// Verify update
	retrieved, err := vm.GetVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if retrieved.Notes != "Updated notes" {
		t.Errorf("Expected updated notes, got %s", retrieved.Notes)
	}
	if retrieved.ElementCount != 200 {
		t.Errorf("Expected element count 200, got %d", retrieved.ElementCount)
	}
}

func TestVersionManager_DeleteVersion(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create version
	meta := &VersionMetadata{
		Version: "1.0.0",
		Notes:   "To be deleted",
	}
	if err := vm.CreateVersion(ctx, meta); err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Delete version
	if err := vm.DeleteVersion(ctx, "1.0.0"); err != nil {
		t.Fatalf("DeleteVersion failed: %v", err)
	}

	// Verify deletion
	_, err := vm.GetVersion(ctx, "1.0.0")
	if err == nil {
		t.Error("Expected error for deleted version, got nil")
	}
}

func TestVersionManager_ListVersions(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create multiple versions
	versions := []string{"1.0.0", "1.1.0", "2.0.0"}
	for _, v := range versions {
		meta := &VersionMetadata{
			Version: v,
			Notes:   "Version " + v,
		}
		if err := vm.CreateVersion(ctx, meta); err != nil {
			t.Fatalf("CreateVersion failed: %v", err)
		}
	}

	// List all versions
	list, err := vm.ListVersions(ctx, nil)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(list))
	}
}

func TestVersionManager_ListVersionsWithQuery(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create versions with tags
	meta1 := &VersionMetadata{
		Version: "1.0.0",
		Tags:    []string{"production"},
	}
	meta2 := &VersionMetadata{
		Version: "1.1.0",
		Tags:    []string{"beta"},
	}
	meta3 := &VersionMetadata{
		Version: "2.0.0",
		Tags:    []string{"production", "latest"},
	}

	for _, meta := range []*VersionMetadata{meta1, meta2, meta3} {
		if err := vm.CreateVersion(ctx, meta); err != nil {
			t.Fatalf("CreateVersion failed: %v", err)
		}
	}

	// Query by tag
	query := &VersionQuery{
		Tags: []string{"production"},
	}
	list, err := vm.ListVersions(ctx, query)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("Expected 2 production versions, got %d", len(list))
	}
}

func TestVersionManager_GetVersionAtTime(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

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

	for _, meta := range []*VersionMetadata{meta1, meta2, meta3} {
		if err := vm.CreateVersion(ctx, meta); err != nil {
			t.Fatalf("CreateVersion failed: %v", err)
		}
	}

	// Query version 90 minutes ago (should return 1.0.0)
	timestamp := now.Add(-90 * time.Minute)
	version, err := vm.GetVersionAtTime(ctx, timestamp)
	if err != nil {
		t.Fatalf("GetVersionAtTime failed: %v", err)
	}

	if version.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", version.Version)
	}

	// Query version 30 minutes ago (should return 1.1.0)
	timestamp = now.Add(-30 * time.Minute)
	version, err = vm.GetVersionAtTime(ctx, timestamp)
	if err != nil {
		t.Fatalf("GetVersionAtTime failed: %v", err)
	}

	if version.Version != "1.1.0" {
		t.Errorf("Expected version 1.1.0, got %s", version.Version)
	}
}

func TestVersionManager_CompareVersions(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create two versions
	meta1 := &VersionMetadata{
		Version:       "1.0.0",
		ElementCount:  1000,
		DocumentCount: 10,
		SizeBytes:     1024 * 1024,
	}
	meta2 := &VersionMetadata{
		Version:       "1.1.0",
		ElementCount:  1500,
		DocumentCount: 15,
		SizeBytes:     1536 * 1024,
	}

	for _, meta := range []*VersionMetadata{meta1, meta2} {
		if err := vm.CreateVersion(ctx, meta); err != nil {
			t.Fatalf("CreateVersion failed: %v", err)
		}
	}

	// Compare versions
	diff, err := vm.CompareVersions(ctx, "1.0.0", "1.1.0")
	if err != nil {
		t.Fatalf("CompareVersions failed: %v", err)
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

func TestVersionManager_DeprecateVersion(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create version
	meta := &VersionMetadata{
		Version: "1.0.0",
		Notes:   "Initial version",
	}
	if err := vm.CreateVersion(ctx, meta); err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Deprecate version
	if err := vm.DeprecateVersion(ctx, "1.0.0", "Superseded by 2.0.0"); err != nil {
		t.Fatalf("DeprecateVersion failed: %v", err)
	}

	// Verify deprecation
	retrieved, err := vm.GetVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if !retrieved.Deprecated {
		t.Error("Expected version to be deprecated")
	}
	if retrieved.DeprecatedAt == nil {
		t.Error("Expected deprecation timestamp")
	}
}

func TestVersionManager_TagVersion(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create version
	meta := &VersionMetadata{
		Version: "1.0.0",
		Tags:    []string{"beta"},
	}
	if err := vm.CreateVersion(ctx, meta); err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Add tags
	if err := vm.TagVersion(ctx, "1.0.0", "production", "stable"); err != nil {
		t.Fatalf("TagVersion failed: %v", err)
	}

	// Verify tags
	retrieved, err := vm.GetVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	expectedTags := map[string]bool{
		"beta":       true,
		"production": true,
		"stable":     true,
	}

	if len(retrieved.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(retrieved.Tags))
	}

	for _, tag := range retrieved.Tags {
		if !expectedTags[tag] {
			t.Errorf("Unexpected tag: %s", tag)
		}
	}
}

func TestVersionManager_UntagVersion(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create version with tags
	meta := &VersionMetadata{
		Version: "1.0.0",
		Tags:    []string{"beta", "production", "stable"},
	}
	if err := vm.CreateVersion(ctx, meta); err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Remove tag
	if err := vm.UntagVersion(ctx, "1.0.0", "beta"); err != nil {
		t.Fatalf("UntagVersion failed: %v", err)
	}

	// Verify tags
	retrieved, err := vm.GetVersion(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
	}

	for _, tag := range retrieved.Tags {
		if tag == "beta" {
			t.Error("Beta tag should have been removed")
		}
	}
}

func TestVersionManager_CreateMigration(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create versions
	meta1 := &VersionMetadata{Version: "1.0.0"}
	meta2 := &VersionMetadata{Version: "2.0.0"}
	vm.CreateVersion(ctx, meta1)
	vm.CreateVersion(ctx, meta2)

	// Create migration
	migration, err := vm.CreateMigration(ctx, "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("CreateMigration failed: %v", err)
	}

	if migration.FromVersion != "1.0.0" {
		t.Errorf("Expected from version 1.0.0, got %s", migration.FromVersion)
	}
	if migration.ToVersion != "2.0.0" {
		t.Errorf("Expected to version 2.0.0, got %s", migration.ToVersion)
	}
	if migration.Status != MigrationPending {
		t.Errorf("Expected pending status, got %s", migration.Status)
	}
}

func TestVersionManager_ExportMetadata(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create version
	meta := &VersionMetadata{
		Version:         "1.0.0",
		ParserVersion:   "2.1.0",
		EmbeddingModel:  "text-embedding-ada-002",
		OntologyVersion: "1.0.0",
		ElementCount:    1000,
	}
	if err := vm.CreateVersion(ctx, meta); err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Export metadata
	data, err := vm.ExportMetadata(ctx, "1.0.0")
	if err != nil {
		t.Fatalf("ExportMetadata failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty JSON data")
	}

	// Verify JSON structure (basic check)
	if len(data) == 0 || data[0] != '{' {
		t.Error("Expected JSON object")
	}
}

func TestVersionManager_GetStats(t *testing.T) {
	backend := NewMemoryBackend()
	vm := NewVersionManager(backend)
	defer vm.Close()

	ctx := context.Background()

	// Create version
	meta := &VersionMetadata{
		Version:       "1.0.0",
		ElementCount:  1000,
		DocumentCount: 10,
		SizeBytes:     1024 * 1024,
	}
	if err := vm.CreateVersion(ctx, meta); err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	// Get stats
	stats, err := vm.GetStats(ctx, "1.0.0")
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
