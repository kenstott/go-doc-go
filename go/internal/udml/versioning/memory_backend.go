package versioning

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemoryBackend implements VersionBackend using in-memory storage
// Suitable for testing and single-process deployments
type MemoryBackend struct {
	versions   map[string]*VersionMetadata
	migrations map[string]*MigrationRecord // Key: "from_version:to_version"
	mu         sync.RWMutex
}

// NewMemoryBackend creates a new in-memory version backend
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		versions:   make(map[string]*VersionMetadata),
		migrations: make(map[string]*MigrationRecord),
	}
}

// CreateVersion stores a new version
func (mb *MemoryBackend) CreateVersion(ctx context.Context, meta *VersionMetadata) error {
	if meta == nil || meta.Version == "" {
		return fmt.Errorf("invalid version metadata")
	}

	mb.mu.Lock()
	defer mb.mu.Unlock()

	if _, exists := mb.versions[meta.Version]; exists {
		return fmt.Errorf("version %s already exists", meta.Version)
	}

	// Deep copy to prevent external modifications
	metaCopy := *meta
	if meta.SchemaChanges != nil {
		metaCopy.SchemaChanges = make(map[string]interface{})
		for k, v := range meta.SchemaChanges {
			metaCopy.SchemaChanges[k] = v
		}
	}
	if meta.Tags != nil {
		metaCopy.Tags = append([]string{}, meta.Tags...)
	}

	mb.versions[meta.Version] = &metaCopy
	return nil
}

// GetVersion retrieves a version by ID
func (mb *MemoryBackend) GetVersion(ctx context.Context, version string) (*VersionMetadata, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	meta, exists := mb.versions[version]
	if !exists {
		return nil, fmt.Errorf("version %s not found", version)
	}

	// Return a copy to prevent external modifications
	metaCopy := *meta
	if meta.SchemaChanges != nil {
		metaCopy.SchemaChanges = make(map[string]interface{})
		for k, v := range meta.SchemaChanges {
			metaCopy.SchemaChanges[k] = v
		}
	}
	if meta.Tags != nil {
		metaCopy.Tags = append([]string{}, meta.Tags...)
	}

	return &metaCopy, nil
}

// UpdateVersion modifies an existing version
func (mb *MemoryBackend) UpdateVersion(ctx context.Context, meta *VersionMetadata) error {
	if meta == nil || meta.Version == "" {
		return fmt.Errorf("invalid version metadata")
	}

	mb.mu.Lock()
	defer mb.mu.Unlock()

	if _, exists := mb.versions[meta.Version]; !exists {
		return fmt.Errorf("version %s not found", meta.Version)
	}

	// Deep copy
	metaCopy := *meta
	if meta.SchemaChanges != nil {
		metaCopy.SchemaChanges = make(map[string]interface{})
		for k, v := range meta.SchemaChanges {
			metaCopy.SchemaChanges[k] = v
		}
	}
	if meta.Tags != nil {
		metaCopy.Tags = append([]string{}, meta.Tags...)
	}

	mb.versions[meta.Version] = &metaCopy
	return nil
}

// DeleteVersion removes a version
func (mb *MemoryBackend) DeleteVersion(ctx context.Context, version string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if _, exists := mb.versions[version]; !exists {
		return fmt.Errorf("version %s not found", version)
	}

	delete(mb.versions, version)
	return nil
}

// ListVersions returns versions matching the query
func (mb *MemoryBackend) ListVersions(ctx context.Context, query *VersionQuery) ([]*VersionMetadata, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	var results []*VersionMetadata

	for _, meta := range mb.versions {
		// Apply filters
		if query != nil {
			// Filter by version
			if query.Version != "" && meta.Version != query.Version {
				continue
			}

			// Filter by tags
			if len(query.Tags) > 0 {
				hasTag := false
				for _, qTag := range query.Tags {
					for _, mTag := range meta.Tags {
						if qTag == mTag {
							hasTag = true
							break
						}
					}
					if hasTag {
						break
					}
				}
				if !hasTag {
					continue
				}
			}

			// Filter by time
			if query.AsOfTime != nil && meta.CreatedAt.After(*query.AsOfTime) {
				continue
			}
		}

		// Copy metadata
		metaCopy := *meta
		if meta.SchemaChanges != nil {
			metaCopy.SchemaChanges = make(map[string]interface{})
			for k, v := range meta.SchemaChanges {
				metaCopy.SchemaChanges[k] = v
			}
		}
		if meta.Tags != nil {
			metaCopy.Tags = append([]string{}, meta.Tags...)
		}

		results = append(results, &metaCopy)
	}

	// Sort by creation time descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	return results, nil
}

// GetVersionAtTime returns the version active at a specific time
func (mb *MemoryBackend) GetVersionAtTime(ctx context.Context, timestamp time.Time) (*VersionMetadata, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	var candidates []*VersionMetadata

	// Find all versions created before or at the timestamp
	for _, meta := range mb.versions {
		if !meta.CreatedAt.After(timestamp) {
			metaCopy := *meta
			if meta.SchemaChanges != nil {
				metaCopy.SchemaChanges = make(map[string]interface{})
				for k, v := range meta.SchemaChanges {
					metaCopy.SchemaChanges[k] = v
				}
			}
			if meta.Tags != nil {
				metaCopy.Tags = append([]string{}, meta.Tags...)
			}
			candidates = append(candidates, &metaCopy)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no version found at time %s", timestamp.Format(time.RFC3339))
	}

	// Sort by creation time descending and return the most recent
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].CreatedAt.After(candidates[j].CreatedAt)
	})

	return candidates[0], nil
}

// GetVersionsInRange returns versions created within a time range
func (mb *MemoryBackend) GetVersionsInRange(ctx context.Context, start, end time.Time) ([]*VersionMetadata, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	var results []*VersionMetadata

	for _, meta := range mb.versions {
		if !meta.CreatedAt.Before(start) && !meta.CreatedAt.After(end) {
			metaCopy := *meta
			if meta.SchemaChanges != nil {
				metaCopy.SchemaChanges = make(map[string]interface{})
				for k, v := range meta.SchemaChanges {
					metaCopy.SchemaChanges[k] = v
				}
			}
			if meta.Tags != nil {
				metaCopy.Tags = append([]string{}, meta.Tags...)
			}
			results = append(results, &metaCopy)
		}
	}

	// Sort by creation time ascending
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.Before(results[j].CreatedAt)
	})

	return results, nil
}

// ComputeDiff calculates differences between two versions
func (mb *MemoryBackend) ComputeDiff(ctx context.Context, fromVersion, toVersion string) (*VersionDiff, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	fromMeta, fromExists := mb.versions[fromVersion]
	if !fromExists {
		return nil, fmt.Errorf("from version %s not found", fromVersion)
	}

	toMeta, toExists := mb.versions[toVersion]
	if !toExists {
		return nil, fmt.Errorf("to version %s not found", toVersion)
	}

	diff := &VersionDiff{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		ComparedAt:  time.Now(),
		FieldChanges: make(map[string]FieldChange),
		TypeChanges:  make(map[string]TypeChange),
	}

	// Calculate data changes
	diff.ElementsAdded = toMeta.ElementCount - fromMeta.ElementCount
	if diff.ElementsAdded < 0 {
		diff.ElementsRemoved = -diff.ElementsAdded
		diff.ElementsAdded = 0
	}

	diff.DocumentsAdded = toMeta.DocumentCount - fromMeta.DocumentCount
	if diff.DocumentsAdded < 0 {
		diff.DocumentsRemoved = -diff.DocumentsAdded
		diff.DocumentsAdded = 0
	}

	// Calculate size delta
	diff.SizeDelta = toMeta.SizeBytes - fromMeta.SizeBytes

	// Detect schema changes (simplified - comparing SchemaChanges maps)
	diff.SchemaChanges = SchemaChanges{
		FieldsAdded:   []string{},
		FieldsRemoved: []string{},
		FieldsChanged: []FieldSchemaChange{},
		TypesAdded:    []string{},
		TypesRemoved:  []string{},
	}

	// This is a simplified implementation
	// A real implementation would deeply analyze schema structures

	return diff, nil
}

// GetStats returns statistics for a version
func (mb *MemoryBackend) GetStats(ctx context.Context, version string) (*VersionStats, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	meta, exists := mb.versions[version]
	if !exists {
		return nil, fmt.Errorf("version %s not found", version)
	}

	stats := &VersionStats{
		Version:        version,
		ElementCounts:  make(map[string]int64),
		DocumentCounts: make(map[string]int64),
		TotalElements:  meta.ElementCount,
		TotalDocuments: meta.DocumentCount,
		SizeBytes:      meta.SizeBytes,
		ComputedAt:     time.Now(),
	}

	// In a real implementation, these would be queried from the data backend
	// For now, return the basic stats from metadata

	return stats, nil
}

// CreateMigration creates a new migration record
func (mb *MemoryBackend) CreateMigration(ctx context.Context, migration *MigrationRecord) error {
	if migration == nil {
		return fmt.Errorf("migration record cannot be nil")
	}

	key := migrationKey(migration.FromVersion, migration.ToVersion)

	mb.mu.Lock()
	defer mb.mu.Unlock()

	if _, exists := mb.migrations[key]; exists {
		return fmt.Errorf("migration from %s to %s already exists", migration.FromVersion, migration.ToVersion)
	}

	migCopy := *migration
	mb.migrations[key] = &migCopy
	return nil
}

// UpdateMigration updates a migration record
func (mb *MemoryBackend) UpdateMigration(ctx context.Context, migration *MigrationRecord) error {
	if migration == nil {
		return fmt.Errorf("migration record cannot be nil")
	}

	key := migrationKey(migration.FromVersion, migration.ToVersion)

	mb.mu.Lock()
	defer mb.mu.Unlock()

	if _, exists := mb.migrations[key]; !exists {
		return fmt.Errorf("migration from %s to %s not found", migration.FromVersion, migration.ToVersion)
	}

	migCopy := *migration
	mb.migrations[key] = &migCopy
	return nil
}

// GetMigration retrieves a migration record
func (mb *MemoryBackend) GetMigration(ctx context.Context, fromVersion, toVersion string) (*MigrationRecord, error) {
	key := migrationKey(fromVersion, toVersion)

	mb.mu.RLock()
	defer mb.mu.RUnlock()

	mig, exists := mb.migrations[key]
	if !exists {
		return nil, fmt.Errorf("migration from %s to %s not found", fromVersion, toVersion)
	}

	migCopy := *mig
	return &migCopy, nil
}

// ListMigrations returns all migration records
func (mb *MemoryBackend) ListMigrations(ctx context.Context) ([]*MigrationRecord, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	var results []*MigrationRecord
	for _, mig := range mb.migrations {
		migCopy := *mig
		results = append(results, &migCopy)
	}

	// Sort by start time descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt.After(results[j].StartedAt)
	})

	return results, nil
}

// Close releases resources
func (mb *MemoryBackend) Close() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	mb.versions = nil
	mb.migrations = nil
	return nil
}

// migrationKey creates a unique key for a migration
func migrationKey(fromVersion, toVersion string) string {
	return fmt.Sprintf("%s:%s", fromVersion, toVersion)
}
