package query

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBackendRegistry validates registry creation
func TestNewBackendRegistry(t *testing.T) {
	registry := NewBackendRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.factories)
	assert.NotNil(t, registry.instances)
	assert.Empty(t, registry.ListBackends())
}

// TestRegister validates backend registration
func TestRegister(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend("test"), nil
	}

	err := registry.Register("test", factory)
	assert.NoError(t, err)

	backends := registry.ListBackends()
	require.Len(t, backends, 1)
	assert.Equal(t, "test", backends[0])
}

// TestRegisterDuplicate validates duplicate registration prevention
func TestRegisterDuplicate(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend("test"), nil
	}

	err := registry.Register("test", factory)
	assert.NoError(t, err)

	// Try to register again
	err = registry.Register("test", factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestUnregister validates backend unregistration
func TestUnregister(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend("test"), nil
	}

	err := registry.Register("test", factory)
	assert.NoError(t, err)

	registry.Unregister("test")

	backends := registry.ListBackends()
	assert.Empty(t, backends)
}

// TestCreate validates backend creation
func TestCreate(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	err := registry.Register("mock", factory)
	require.NoError(t, err)

	config := BackendConfig{
		Type:    "mock",
		Timeout: 30 * time.Second,
	}

	backend, err := registry.Create(config)
	require.NoError(t, err)
	require.NotNil(t, backend)

	assert.Equal(t, "mock", backend.GetName())
}

// TestCreateUnknownBackend validates error on unknown backend
func TestCreateUnknownBackend(t *testing.T) {
	registry := NewBackendRegistry()

	config := BackendConfig{
		Type: "unknown",
	}

	backend, err := registry.Create(config)
	assert.Error(t, err)
	assert.Nil(t, backend)
	assert.Contains(t, err.Error(), "unknown backend type")
}

// TestGetOrCreate validates get-or-create pattern
func TestGetOrCreate(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	err := registry.Register("mock", factory)
	require.NoError(t, err)

	config := BackendConfig{
		Type:    "mock",
		Timeout: 30 * time.Second,
	}

	// First call creates instance
	backend1, err := registry.GetOrCreate(config)
	require.NoError(t, err)
	require.NotNil(t, backend1)

	// Second call returns cached instance
	backend2, err := registry.GetOrCreate(config)
	require.NoError(t, err)
	require.NotNil(t, backend2)

	// Should be the same instance
	assert.Equal(t, backend1, backend2)
}

// TestGet validates getting cached backend
func TestGet(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	err := registry.Register("mock", factory)
	require.NoError(t, err)

	config := BackendConfig{
		Type:    "mock",
		Timeout: 30 * time.Second,
	}

	// Create instance first
	_, err = registry.GetOrCreate(config)
	require.NoError(t, err)

	// Get cached instance
	backend, err := registry.Get("mock")
	require.NoError(t, err)
	assert.NotNil(t, backend)
	assert.Equal(t, "mock", backend.GetName())
}

// TestGetNonExistent validates error on getting non-existent backend
func TestGetNonExistent(t *testing.T) {
	registry := NewBackendRegistry()

	backend, err := registry.Get("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, backend)
	assert.Contains(t, err.Error(), "not found")
}

// TestListBackends validates listing registered backends
func TestListBackends(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	backends := []string{"duckdb", "postgresql", "neo4j"}
	for _, backend := range backends {
		err := registry.Register(backend, factory)
		require.NoError(t, err)
	}

	registered := registry.ListBackends()
	assert.Len(t, registered, 3)

	// Verify all backends are registered
	for _, backend := range backends {
		assert.Contains(t, registered, backend)
	}
}

// TestIsRegistered validates registration check
func TestIsRegistered(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	err := registry.Register("mock", factory)
	require.NoError(t, err)

	assert.True(t, registry.IsRegistered("mock"))
	assert.False(t, registry.IsRegistered("unknown"))
}

// TestCloseAll validates closing all backends
func TestCloseAll(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	err := registry.Register("mock1", factory)
	require.NoError(t, err)
	err = registry.Register("mock2", factory)
	require.NoError(t, err)

	// Create instances
	config1 := BackendConfig{Type: "mock1"}
	config2 := BackendConfig{Type: "mock2"}

	_, err = registry.GetOrCreate(config1)
	require.NoError(t, err)
	_, err = registry.GetOrCreate(config2)
	require.NoError(t, err)

	// Close all
	err = registry.CloseAll()
	assert.NoError(t, err)

	// Verify instances are cleared
	_, err = registry.Get("mock1")
	assert.Error(t, err)
	_, err = registry.Get("mock2")
	assert.Error(t, err)
}

// TestGlobalRegistry validates global registry functions
func TestGlobalRegistry(t *testing.T) {
	// Clean up global registry first
	CloseAllBackends()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	// Register in global registry
	err := Register("global_test", factory)
	require.NoError(t, err)

	// Check registration
	assert.True(t, IsBackendRegistered("global_test"))

	// List backends
	backends := ListBackends()
	assert.Contains(t, backends, "global_test")

	// Create backend
	config := BackendConfig{Type: "global_test"}
	backend, err := GetOrCreateBackend(config)
	require.NoError(t, err)
	assert.NotNil(t, backend)

	// Get backend
	backend2, err := GetBackend("global_test")
	require.NoError(t, err)
	assert.Equal(t, backend, backend2)

	// Unregister
	Unregister("global_test")
	assert.False(t, IsBackendRegistered("global_test"))
}

// TestMockBackend validates mock backend functionality
func TestMockBackend(t *testing.T) {
	backend := NewMockBackend("test_mock")

	assert.Equal(t, "test_mock", backend.GetName())
	assert.Equal(t, "1.0.0", backend.GetVersion())

	// Test initialization
	ctx := context.Background()
	config := BackendConfig{Type: "test_mock"}
	err := backend.Initialize(ctx, config)
	assert.NoError(t, err)
	assert.True(t, backend.initialized)

	// Test feature support
	assert.True(t, backend.SupportsFeature("jsonpath"))
	assert.True(t, backend.SupportsFeature("regex"))
	assert.False(t, backend.SupportsFeature("similarity"))
	assert.False(t, backend.SupportsFeature("graph_traversal"))

	// Test translate
	expr := NewExpression("elements").SelectAll()
	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)
	assert.Equal(t, "test_mock", query.BackendType)
	assert.Contains(t, query.QueryString, "SELECT")

	// Test execute
	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Columns, 2)
	assert.Equal(t, 0, result.RowCount)

	// Test explain
	plan, err := backend.Explain(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, "test_mock", plan.BackendName)
	assert.NotEmpty(t, plan.RawPlan)
	assert.Len(t, plan.Steps, 1)

	// Test close
	err = backend.Close()
	assert.NoError(t, err)
	assert.False(t, backend.initialized)
}

// TestBackendConfig validates configuration structure
func TestBackendConfig(t *testing.T) {
	config := BackendConfig{
		Type:               "duckdb",
		Timeout:            30 * time.Second,
		ParquetPath:        "/data/analytics",
		ConnectionString:   "postgres://localhost/db",
		MaxConnections:     10,
		EnableSimilarity:   true,
		EmbeddingModel:     "all-MiniLM-L6-v2",
		Config: map[string]interface{}{
			"memory_limit": "4GB",
			"threads":      8,
		},
	}

	assert.Equal(t, "duckdb", config.Type)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "/data/analytics", config.ParquetPath)
	assert.Equal(t, "postgres://localhost/db", config.ConnectionString)
	assert.Equal(t, 10, config.MaxConnections)
	assert.True(t, config.EnableSimilarity)
	assert.Equal(t, "all-MiniLM-L6-v2", config.EmbeddingModel)
	assert.Equal(t, "4GB", config.Config["memory_limit"])
	assert.Equal(t, 8, config.Config["threads"])
}

// TestTranslateOptions validates translation options
func TestTranslateOptions(t *testing.T) {
	opts := TranslateOptions{
		EnablePushdown:   true,
		EnablePartitions: true,
		Limit:            100,
		Offset:           50,
		Verbose:          true,
	}

	assert.True(t, opts.EnablePushdown)
	assert.True(t, opts.EnablePartitions)
	assert.Equal(t, 100, opts.Limit)
	assert.Equal(t, 50, opts.Offset)
	assert.True(t, opts.Verbose)
}

// TestQueryPlan validates query plan structure
func TestQueryPlan(t *testing.T) {
	plan := QueryPlan{
		BackendName: "duckdb",
		RawPlan:     "EXPLAIN SELECT * FROM elements",
		Steps: []QueryPlanStep{
			{
				Operator:      "SCAN",
				Description:   "Sequential scan on elements",
				EstimatedRows: 1000,
				EstimatedCost: 10.5,
			},
			{
				Operator:      "FILTER",
				Description:   "Filter: element_type = 'paragraph'",
				EstimatedRows: 200,
				EstimatedCost: 2.0,
			},
		},
		Metadata: map[string]interface{}{
			"optimizer_version": "1.0",
		},
		EstimatedCost: 12.5,
	}

	assert.Equal(t, "duckdb", plan.BackendName)
	assert.NotEmpty(t, plan.RawPlan)
	assert.Len(t, plan.Steps, 2)
	assert.Equal(t, "SCAN", plan.Steps[0].Operator)
	assert.Equal(t, 1000, plan.Steps[0].EstimatedRows)
	assert.Equal(t, 12.5, plan.EstimatedCost)
}

// TestBackendCapabilities validates capabilities structure
func TestBackendCapabilities(t *testing.T) {
	caps := BackendCapabilities{
		SupportsJSONPath:        true,
		SupportsSimilarity:      true,
		SupportsRegex:           true,
		SupportsFullTextSearch:  false,
		SupportsGraphTraversal:  false,
		SupportsPartitionPruning: true,
		SupportsVectorized:      true,
		SupportsParallel:        true,
	}

	assert.True(t, caps.SupportsJSONPath)
	assert.True(t, caps.SupportsSimilarity)
	assert.True(t, caps.SupportsRegex)
	assert.False(t, caps.SupportsFullTextSearch)
	assert.False(t, caps.SupportsGraphTraversal)
	assert.True(t, caps.SupportsPartitionPruning)
	assert.True(t, caps.SupportsVectorized)
	assert.True(t, caps.SupportsParallel)
}

// TestConcurrentRegistryAccess validates thread-safe registry operations
func TestConcurrentRegistryAccess(t *testing.T) {
	registry := NewBackendRegistry()

	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	// Register multiple backends concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			backendType := fmt.Sprintf("backend_%d", id)
			err := registry.Register(backendType, factory)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	backends := registry.ListBackends()
	assert.Len(t, backends, 10)
}
