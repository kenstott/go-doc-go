package versioning

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// VersionManager handles UDML versioning operations
type VersionManager struct {
	backend VersionBackend
	mu      sync.RWMutex

	// Cache for frequently accessed versions
	cache      map[string]*VersionMetadata
	cacheSize  int
	cacheTTL   time.Duration
}

// VersionBackend defines the interface for version storage
type VersionBackend interface {
	// Version CRUD operations
	CreateVersion(ctx context.Context, meta *VersionMetadata) error
	GetVersion(ctx context.Context, version string) (*VersionMetadata, error)
	UpdateVersion(ctx context.Context, meta *VersionMetadata) error
	DeleteVersion(ctx context.Context, version string) error
	ListVersions(ctx context.Context, query *VersionQuery) ([]*VersionMetadata, error)

	// Time-travel queries
	GetVersionAtTime(ctx context.Context, timestamp time.Time) (*VersionMetadata, error)
	GetVersionsInRange(ctx context.Context, start, end time.Time) ([]*VersionMetadata, error)

	// Comparison
	ComputeDiff(ctx context.Context, fromVersion, toVersion string) (*VersionDiff, error)
	GetStats(ctx context.Context, version string) (*VersionStats, error)

	// Migrations
	CreateMigration(ctx context.Context, migration *MigrationRecord) error
	UpdateMigration(ctx context.Context, migration *MigrationRecord) error
	GetMigration(ctx context.Context, fromVersion, toVersion string) (*MigrationRecord, error)
	ListMigrations(ctx context.Context) ([]*MigrationRecord, error)

	// Backend operations
	Close() error
}

// NewVersionManager creates a new version manager
func NewVersionManager(backend VersionBackend) *VersionManager {
	return &VersionManager{
		backend:   backend,
		cache:     make(map[string]*VersionMetadata),
		cacheSize: 100,  // Cache up to 100 versions
		cacheTTL:  5 * time.Minute,
	}
}

// CreateVersion registers a new UDML version
func (vm *VersionManager) CreateVersion(ctx context.Context, meta *VersionMetadata) error {
	if meta == nil {
		return fmt.Errorf("version metadata cannot be nil")
	}

	// Validate version string
	if meta.Version == "" {
		return fmt.Errorf("version string cannot be empty")
	}

	// Set created timestamp if not provided
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}

	// Initialize maps if nil
	if meta.SchemaChanges == nil {
		meta.SchemaChanges = make(map[string]interface{})
	}
	if meta.Tags == nil {
		meta.Tags = []string{}
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Create in backend
	if err := vm.backend.CreateVersion(ctx, meta); err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}

	// Update cache
	vm.cache[meta.Version] = meta

	return nil
}

// GetVersion retrieves metadata for a specific version
func (vm *VersionManager) GetVersion(ctx context.Context, version string) (*VersionMetadata, error) {
	if version == "" {
		return nil, fmt.Errorf("version string cannot be empty")
	}

	// Check cache first
	vm.mu.RLock()
	if cached, ok := vm.cache[version]; ok {
		vm.mu.RUnlock()
		return cached, nil
	}
	vm.mu.RUnlock()

	// Fetch from backend
	meta, err := vm.backend.GetVersion(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %s: %w", version, err)
	}

	// Update cache
	vm.mu.Lock()
	vm.cache[version] = meta
	vm.mu.Unlock()

	return meta, nil
}

// GetLatestVersion returns the most recent version
func (vm *VersionManager) GetLatestVersion(ctx context.Context) (*VersionMetadata, error) {
	versions, err := vm.ListVersions(ctx, &VersionQuery{})
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found")
	}

	// Sort by creation time descending
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	return versions[0], nil
}

// GetVersionAtTime returns the version that was current at a specific time (time-travel)
func (vm *VersionManager) GetVersionAtTime(ctx context.Context, timestamp time.Time) (*VersionMetadata, error) {
	if timestamp.IsZero() {
		return nil, fmt.Errorf("timestamp cannot be zero")
	}

	meta, err := vm.backend.GetVersionAtTime(ctx, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get version at time %s: %w", timestamp.Format(time.RFC3339), err)
	}

	return meta, nil
}

// ListVersions returns versions matching the query
func (vm *VersionManager) ListVersions(ctx context.Context, query *VersionQuery) ([]*VersionMetadata, error) {
	if query == nil {
		query = &VersionQuery{}
	}

	versions, err := vm.backend.ListVersions(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	return versions, nil
}

// UpdateVersion updates version metadata
func (vm *VersionManager) UpdateVersion(ctx context.Context, meta *VersionMetadata) error {
	if meta == nil || meta.Version == "" {
		return fmt.Errorf("invalid version metadata")
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	if err := vm.backend.UpdateVersion(ctx, meta); err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}

	// Update cache
	vm.cache[meta.Version] = meta

	return nil
}

// DeleteVersion removes a version
func (vm *VersionManager) DeleteVersion(ctx context.Context, version string) error {
	if version == "" {
		return fmt.Errorf("version string cannot be empty")
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	if err := vm.backend.DeleteVersion(ctx, version); err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	// Remove from cache
	delete(vm.cache, version)

	return nil
}

// CompareVersions returns differences between two versions
func (vm *VersionManager) CompareVersions(ctx context.Context, fromVersion, toVersion string) (*VersionDiff, error) {
	if fromVersion == "" || toVersion == "" {
		return nil, fmt.Errorf("version strings cannot be empty")
	}

	// Verify versions exist
	if _, err := vm.GetVersion(ctx, fromVersion); err != nil {
		return nil, fmt.Errorf("from version not found: %w", err)
	}
	if _, err := vm.GetVersion(ctx, toVersion); err != nil {
		return nil, fmt.Errorf("to version not found: %w", err)
	}

	diff, err := vm.backend.ComputeDiff(ctx, fromVersion, toVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	return diff, nil
}

// GetStats returns statistics for a version
func (vm *VersionManager) GetStats(ctx context.Context, version string) (*VersionStats, error) {
	if version == "" {
		return nil, fmt.Errorf("version string cannot be empty")
	}

	// Verify version exists
	if _, err := vm.GetVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	stats, err := vm.backend.GetStats(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

// DeprecateVersion marks a version as deprecated
func (vm *VersionManager) DeprecateVersion(ctx context.Context, version string, reason string) error {
	meta, err := vm.GetVersion(ctx, version)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	now := time.Now()
	meta.Deprecated = true
	meta.DeprecatedAt = &now
	meta.Notes = fmt.Sprintf("%s\nDeprecated: %s", meta.Notes, reason)

	return vm.UpdateVersion(ctx, meta)
}

// TagVersion adds tags to a version
func (vm *VersionManager) TagVersion(ctx context.Context, version string, tags ...string) error {
	meta, err := vm.GetVersion(ctx, version)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Add new tags (avoid duplicates)
	tagSet := make(map[string]bool)
	for _, t := range meta.Tags {
		tagSet[t] = true
	}
	for _, t := range tags {
		tagSet[t] = true
	}

	meta.Tags = make([]string, 0, len(tagSet))
	for t := range tagSet {
		meta.Tags = append(meta.Tags, t)
	}
	sort.Strings(meta.Tags)

	return vm.UpdateVersion(ctx, meta)
}

// UntagVersion removes tags from a version
func (vm *VersionManager) UntagVersion(ctx context.Context, version string, tags ...string) error {
	meta, err := vm.GetVersion(ctx, version)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Remove specified tags
	removeSet := make(map[string]bool)
	for _, t := range tags {
		removeSet[t] = true
	}

	filteredTags := []string{}
	for _, t := range meta.Tags {
		if !removeSet[t] {
			filteredTags = append(filteredTags, t)
		}
	}

	meta.Tags = filteredTags
	return vm.UpdateVersion(ctx, meta)
}

// CreateMigration initiates a version migration
func (vm *VersionManager) CreateMigration(ctx context.Context, fromVersion, toVersion string) (*MigrationRecord, error) {
	// Verify versions exist
	if _, err := vm.GetVersion(ctx, fromVersion); err != nil {
		return nil, fmt.Errorf("from version not found: %w", err)
	}
	if _, err := vm.GetVersion(ctx, toVersion); err != nil {
		return nil, fmt.Errorf("to version not found: %w", err)
	}

	migration := &MigrationRecord{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		StartedAt:   time.Now(),
		Status:      MigrationPending,
	}

	if err := vm.backend.CreateMigration(ctx, migration); err != nil {
		return nil, fmt.Errorf("failed to create migration: %w", err)
	}

	return migration, nil
}

// UpdateMigration updates migration progress
func (vm *VersionManager) UpdateMigration(ctx context.Context, migration *MigrationRecord) error {
	if migration == nil {
		return fmt.Errorf("migration record cannot be nil")
	}

	return vm.backend.UpdateMigration(ctx, migration)
}

// GetMigration retrieves a migration record
func (vm *VersionManager) GetMigration(ctx context.Context, fromVersion, toVersion string) (*MigrationRecord, error) {
	return vm.backend.GetMigration(ctx, fromVersion, toVersion)
}

// ListMigrations returns all migration records
func (vm *VersionManager) ListMigrations(ctx context.Context) ([]*MigrationRecord, error) {
	return vm.backend.ListMigrations(ctx)
}

// ExportMetadata exports version metadata as JSON
func (vm *VersionManager) ExportMetadata(ctx context.Context, version string) ([]byte, error) {
	meta, err := vm.GetVersion(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return data, nil
}

// Close releases resources
func (vm *VersionManager) Close() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	vm.cache = nil
	return vm.backend.Close()
}
