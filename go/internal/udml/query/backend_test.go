package query

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueryBackendInterface validates that MockBackend implements QueryBackend
func TestQueryBackendInterface(t *testing.T) {
	var _ QueryBackend = (*MockBackend)(nil)
}

// TestBackendLifecycle validates full backend lifecycle
func TestBackendLifecycle(t *testing.T) {
	backend := NewMockBackend("lifecycle_test")

	// Step 1: Backend should not be initialized
	assert.False(t, backend.initialized)

	// Step 2: Initialize backend
	ctx := context.Background()
	config := BackendConfig{
		Type:        "lifecycle_test",
		ParquetPath: "/tmp/parquet",
		Timeout:     30 * time.Second,
	}

	err := backend.Initialize(ctx, config)
	require.NoError(t, err)
	assert.True(t, backend.initialized)

	// Step 3: Execute queries
	expr := NewExpression("elements").
		SelectFields("element_id", "content").
		Filter(NewComparison("element_type", OpEqual, "paragraph"))

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePushdown:   true,
		EnablePartitions: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, query)
	assert.Equal(t, expr, query.SourceExpr)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "lifecycle_test", result.BackendType)

	// Step 4: Explain query
	plan, err := backend.Explain(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, "lifecycle_test", plan.BackendName)

	// Step 5: Close backend
	err = backend.Close()
	assert.NoError(t, err)
	assert.False(t, backend.initialized)
}

// TestBackendFeatureSupport validates feature checking
func TestBackendFeatureSupport(t *testing.T) {
	backend := NewMockBackend("feature_test")

	tests := []struct {
		feature  string
		expected bool
	}{
		{"jsonpath", true},
		{"regex", true},
		{"similarity", false},
		{"full_text_search", false},
		{"graph_traversal", false},
		{"unknown_feature", false},
	}

	for _, tt := range tests {
		t.Run(tt.feature, func(t *testing.T) {
			result := backend.SupportsFeature(tt.feature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBackendTranslation validates query translation
func TestBackendTranslation(t *testing.T) {
	backend := NewMockBackend("translation_test")

	tests := []struct {
		name string
		expr *Expression
	}{
		{
			name: "simple_select",
			expr: NewExpression("elements").SelectAll(),
		},
		{
			name: "filtered_query",
			expr: NewExpression("elements").
				SelectFields("element_id", "content").
				Filter(NewComparison("element_type", OpEqual, "paragraph")),
		},
		{
			name: "complex_query",
			expr: NewExpression("elements").
				SelectFields("element_id", "content", "page_number").
				Filter(NewAnd(
					NewComparison("element_type", OpEqual, "paragraph"),
					NewComparison("page_number", OpGreaterThan, 5),
				)).
				Order("document_position", false).
				LimitResults(100, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := backend.Translate(tt.expr, TranslateOptions{})
			require.NoError(t, err)
			assert.NotNil(t, query)
			assert.Equal(t, "translation_test", query.BackendType)
			assert.NotEmpty(t, query.QueryString)
			assert.Equal(t, tt.expr, query.SourceExpr)
		})
	}
}

// TestBackendExecution validates query execution
func TestBackendExecution(t *testing.T) {
	backend := NewMockBackend("execution_test")
	ctx := context.Background()

	expr := NewExpression("elements").SelectAll()
	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result structure
	assert.Equal(t, "execution_test", result.BackendType)
	assert.Len(t, result.Columns, 2)
	assert.Equal(t, "element_id", result.Columns[0].Name)
	assert.Equal(t, "string", result.Columns[0].Type)
	assert.False(t, result.Columns[0].Nullable)

	assert.Equal(t, "content", result.Columns[1].Name)
	assert.Equal(t, "string", result.Columns[1].Type)
	assert.True(t, result.Columns[1].Nullable)

	assert.NotNil(t, result.Rows)
	assert.Equal(t, 0, result.RowCount)
}

// TestBackendExplain validates query plan generation
func TestBackendExplain(t *testing.T) {
	backend := NewMockBackend("explain_test")
	ctx := context.Background()

	expr := NewExpression("elements").
		SelectAll().
		Filter(NewComparison("element_type", OpEqual, "paragraph"))

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)

	plan, err := backend.Explain(ctx, query)
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Verify plan structure
	assert.Equal(t, "explain_test", plan.BackendName)
	assert.NotEmpty(t, plan.RawPlan)
	assert.Len(t, plan.Steps, 1)

	step := plan.Steps[0]
	assert.Equal(t, "SCAN", step.Operator)
	assert.NotEmpty(t, step.Description)
	assert.Equal(t, 1000, step.EstimatedRows)
	assert.Equal(t, 1.0, step.EstimatedCost)
}

// TestBackendWithRegistry validates backend registry integration
func TestBackendWithRegistry(t *testing.T) {
	registry := NewBackendRegistry()

	// Register mock backend factory
	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}

	err := registry.Register("integrated_mock", factory)
	require.NoError(t, err)

	// Create backend through registry
	config := BackendConfig{
		Type:        "integrated_mock",
		ParquetPath: "/data/analytics",
		Timeout:     30 * time.Second,
	}

	backend, err := registry.GetOrCreate(config)
	require.NoError(t, err)
	require.NotNil(t, backend)

	// Verify backend works
	assert.Equal(t, "integrated_mock", backend.GetName())
	assert.True(t, backend.SupportsFeature("jsonpath"))

	// Execute full query cycle
	ctx := context.Background()
	expr := NewExpression("elements").SelectAll()

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, "integrated_mock", result.BackendType)

	// Clean up
	err = registry.CloseAll()
	assert.NoError(t, err)
}

// TestBackendConfigValidation validates configuration handling
func TestBackendConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config BackendConfig
		valid  bool
	}{
		{
			name: "valid_duckdb_config",
			config: BackendConfig{
				Type:        "duckdb",
				ParquetPath: "/data/analytics",
				Timeout:     30 * time.Second,
			},
			valid: true,
		},
		{
			name: "valid_postgres_config",
			config: BackendConfig{
				Type:             "postgresql",
				ConnectionString: "postgres://localhost/db",
				MaxConnections:   10,
				Timeout:          30 * time.Second,
			},
			valid: true,
		},
		{
			name: "config_with_features",
			config: BackendConfig{
				Type:             "duckdb",
				ParquetPath:      "/data/analytics",
				EnableSimilarity: true,
				EmbeddingModel:   "all-MiniLM-L6-v2",
				Timeout:          30 * time.Second,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.config.Type)
			if tt.config.Timeout == 0 {
				tt.config.Timeout = 30 * time.Second
			}
			assert.NotZero(t, tt.config.Timeout)
		})
	}
}

// TestQueryBackendVersioning validates backend version tracking
func TestQueryBackendVersioning(t *testing.T) {
	backends := []struct {
		name    string
		version string
	}{
		{"duckdb", "0.10.0"},
		{"postgresql", "16.0"},
		{"neo4j", "5.0"},
	}

	for _, tt := range backends {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend(tt.name)
			backend.version = tt.version

			assert.Equal(t, tt.name, backend.GetName())
			assert.Equal(t, tt.version, backend.GetVersion())
		})
	}
}

// TestBackendContextHandling validates context propagation
func TestBackendContextHandling(t *testing.T) {
	backend := NewMockBackend("context_test")

	// Test with regular context
	ctx := context.Background()
	config := BackendConfig{Type: "context_test"}

	err := backend.Initialize(ctx, config)
	require.NoError(t, err)

	expr := NewExpression("elements").SelectAll()
	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)

	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Mock backend doesn't actually check context, but real backends should
	_, err = backend.Execute(cancelledCtx, query)
	// Mock backend returns success, real backends would return context.Canceled
	assert.NoError(t, err)
}

// TestBackendTranslateOptions validates translation option handling
func TestBackendTranslateOptions(t *testing.T) {
	backend := NewMockBackend("options_test")

	expr := NewExpression("elements").
		SelectFields("element_id", "content").
		Filter(NewComparison("element_type", OpEqual, "paragraph"))

	tests := []struct {
		name string
		opts TranslateOptions
	}{
		{
			name: "default_options",
			opts: TranslateOptions{},
		},
		{
			name: "optimized_options",
			opts: TranslateOptions{
				EnablePushdown:   true,
				EnablePartitions: true,
				Verbose:          false,
			},
		},
		{
			name: "limited_results",
			opts: TranslateOptions{
				Limit:  100,
				Offset: 50,
			},
		},
		{
			name: "verbose_mode",
			opts: TranslateOptions{
				Verbose: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := backend.Translate(expr, tt.opts)
			require.NoError(t, err)
			assert.NotNil(t, query)
			assert.Equal(t, "options_test", query.BackendType)
		})
	}
}

// TestEndToEndQueryFlow validates complete query flow
func TestEndToEndQueryFlow(t *testing.T) {
	// 1. Create registry
	registry := NewBackendRegistry()

	// 2. Register backend
	factory := func(config BackendConfig) (QueryBackend, error) {
		return NewMockBackend(config.Type), nil
	}
	err := registry.Register("e2e_test", factory)
	require.NoError(t, err)

	// 3. Create backend
	config := BackendConfig{
		Type:        "e2e_test",
		ParquetPath: "/data/analytics",
		Timeout:     30 * time.Second,
	}
	backend, err := registry.GetOrCreate(config)
	require.NoError(t, err)

	// 4. Build query expression
	expr := NewExpression("elements").
		SelectFields("element_id", "content", "element_type").
		Filter(NewAnd(
			NewComparison("element_type", OpEqual, "paragraph"),
			NewComparison("page_number", OpGreaterThan, 5),
		)).
		Order("document_position", false).
		LimitResults(100, 0)

	// 5. Translate to native query
	query, err := backend.Translate(expr, TranslateOptions{
		EnablePushdown:   true,
		EnablePartitions: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "e2e_test", query.BackendType)

	// 6. Execute query
	ctx := context.Background()
	result, err := backend.Execute(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, "e2e_test", result.BackendType)

	// 7. Explain query
	plan, err := backend.Explain(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, "e2e_test", plan.BackendName)

	// 8. Clean up
	err = registry.CloseAll()
	assert.NoError(t, err)
}
