package query

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewDuckDBBackend validates DuckDB backend creation
func TestNewDuckDBBackend(t *testing.T) {
	config := BackendConfig{
		Type:        "duckdb",
		ParquetPath: "/tmp/analytics",
	}

	backend, err := NewDuckDBBackend(config)
	require.NoError(t, err)
	require.NotNil(t, backend)

	assert.Equal(t, "duckdb", backend.GetName())
	assert.NotEmpty(t, backend.GetVersion())

	err = backend.Close()
	assert.NoError(t, err)
}

// TestDuckDBBackendInitialize validates initialization
func TestDuckDBBackendInitialize(t *testing.T) {
	config := BackendConfig{
		Type:        "duckdb",
		ParquetPath: "/tmp/analytics",
	}

	backend, err := NewDuckDBBackend(config)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	err = backend.Initialize(ctx, config)
	require.NoError(t, err)

	duckdb := backend.(*DuckDBBackend)
	assert.True(t, duckdb.initialized)
	assert.NotNil(t, duckdb.db)
}

// TestDuckDBTranslate_SimpleSelect validates simple SELECT translation
func TestDuckDBTranslate_SimpleSelect(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
		initialized: true,
	}

	expr := NewExpression("elements").SelectAll()

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "duckdb", query.BackendType)
	assert.Contains(t, query.QueryString, "SELECT *")
	assert.Contains(t, query.QueryString, "FROM '/data/analytics/elements/element_type=*/version=*/date=*/source=*/*.parquet'")
}

// TestDuckDBTranslate_SelectFields validates field selection translation
func TestDuckDBTranslate_SelectFields(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
		initialized: true,
	}

	expr := NewExpression("elements").
		SelectFields("element_id", "content", "element_type")

	query, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})

	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "SELECT element_id, content, element_type")
}

// TestDuckDBTranslate_WithAlias validates field aliases
func TestDuckDBTranslate_WithAlias(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	expr := NewExpression("elements")
	expr.Select = []FieldSelection{
		{Field: "element_id", Alias: "id"},
		{Field: "element_type", Alias: "type"},
	}

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "element_id AS id")
	assert.Contains(t, query.QueryString, "element_type AS type")
}

// TestDuckDBTranslate_SimpleWhere validates WHERE clause translation
func TestDuckDBTranslate_SimpleWhere(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	expr := NewExpression("elements").
		SelectAll().
		Filter(NewComparison("element_type", OpEqual, "paragraph"))

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "WHERE element_type = 'paragraph'")
}

// TestDuckDBTranslate_ComparisonOperators validates comparison operators
func TestDuckDBTranslate_ComparisonOperators(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	tests := []struct {
		name     string
		operator Operator
		value    interface{}
		expected string
	}{
		{"equal", OpEqual, "paragraph", "element_type = 'paragraph'"},
		{"not_equal", OpNotEqual, "header", "element_type != 'header'"},
		{"greater_than", OpGreaterThan, 5, "page_number > 5"},
		{"greater_equal", OpGreaterThanOrEqual, 10, "page_number >= 10"},
		{"less_than", OpLessThan, 3, "section_level < 3"},
		{"less_equal", OpLessThanOrEqual, 7, "section_level <= 7"},
		{"like", OpLike, "%test%", "content LIKE '%test%'"},
		{"is_null", OpIsNull, nil, "parent_id IS NULL"},
		{"is_not_null", OpIsNotNull, nil, "content IS NOT NULL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var field string
			switch tt.operator {
			case OpEqual, OpNotEqual:
				field = "element_type"
			case OpGreaterThan, OpGreaterThanOrEqual:
				field = "page_number"
			case OpLessThan, OpLessThanOrEqual:
				field = "section_level"
			case OpLike:
				field = "content"
			case OpIsNull:
				field = "parent_id"
			case OpIsNotNull:
				field = "content"
			}

			expr := NewExpression("elements").
				SelectAll().
				Filter(NewComparison(field, tt.operator, tt.value))

			query, err := backend.Translate(expr, TranslateOptions{})
			require.NoError(t, err)
			assert.Contains(t, query.QueryString, tt.expected)
		})
	}
}

// TestDuckDBTranslate_InOperator validates IN operator
func TestDuckDBTranslate_InOperator(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	expr := NewExpression("elements").
		SelectAll().
		Filter(NewComparison("element_type", OpIn, []string{"paragraph", "header", "list"}))

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "element_type IN ('paragraph', 'header', 'list')")
}

// TestDuckDBTranslate_LogicalOperators validates AND/OR/NOT
func TestDuckDBTranslate_LogicalOperators(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	// AND
	andExpr := NewExpression("elements").
		SelectAll().
		Filter(NewAnd(
			NewComparison("element_type", OpEqual, "paragraph"),
			NewComparison("page_number", OpGreaterThan, 5),
		))

	query, err := backend.Translate(andExpr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "element_type = 'paragraph'")
	assert.Contains(t, query.QueryString, "AND")
	assert.Contains(t, query.QueryString, "page_number > 5")

	// OR
	orExpr := NewExpression("elements").
		SelectAll().
		Filter(NewOr(
			NewComparison("element_type", OpEqual, "paragraph"),
			NewComparison("element_type", OpEqual, "header"),
		))

	query, err = backend.Translate(orExpr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "OR")

	// NOT
	notExpr := NewExpression("elements").
		SelectAll().
		Filter(NewNot(NewComparison("element_type", OpEqual, "code_block")))

	query, err = backend.Translate(notExpr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "NOT")
}

// TestDuckDBTranslate_OrderBy validates ORDER BY translation
func TestDuckDBTranslate_OrderBy(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	expr := NewExpression("elements").
		SelectAll().
		Order("document_position", false).
		Order("element_order", true)

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "ORDER BY document_position ASC, element_order DESC")
}

// TestDuckDBTranslate_LimitOffset validates LIMIT/OFFSET translation
func TestDuckDBTranslate_LimitOffset(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	expr := NewExpression("elements").
		SelectAll().
		LimitResults(100, 50)

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "LIMIT 100 OFFSET 50")
}

// TestDuckDBTranslate_GroupBy validates GROUP BY translation
func TestDuckDBTranslate_GroupBy(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	expr := NewExpression("elements")
	expr.Select = []FieldSelection{{Field: "element_type"}, {Field: "COUNT(*)", Alias: "count"}}
	expr.GroupBy = []string{"element_type"}

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "GROUP BY element_type")
}

// TestDuckDBTranslate_JSONPath validates JSONPath translation
func TestDuckDBTranslate_JSONPath(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	// Create JSONPath predicate manually since we need to specify field and path
	pred := &Predicate{
		Type:     PredicateJSONPath,
		Field:    "metadata",
		JSONPath: "$.font_size",
		Operator: OpGreaterThan,
		Value:    12,
	}

	expr := NewExpression("elements").
		SelectAll().
		Filter(pred)

	query, err := backend.Translate(expr, TranslateOptions{})
	require.NoError(t, err)
	assert.Contains(t, query.QueryString, "json_extract(metadata, '$.font_size') > 12")
}

// TestDuckDBTranslate_ComplexQuery validates complex nested query
func TestDuckDBTranslate_ComplexQuery(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	expr := NewExpression("elements").
		SelectFields("element_id", "content", "page_number").
		Filter(NewAnd(
			NewOr(
				NewComparison("element_type", OpEqual, "paragraph"),
				NewComparison("element_type", OpEqual, "header"),
			),
			NewComparison("page_number", OpGreaterThan, 5),
			NewNot(NewComparison("section_level", OpGreaterThan, 3)),
		)).
		Order("document_position", false).
		LimitResults(100, 0)

	query, err := backend.Translate(expr, TranslateOptions{EnablePartitions: true})
	require.NoError(t, err)

	// Verify all clauses present
	assert.Contains(t, query.QueryString, "SELECT element_id, content, page_number")
	assert.Contains(t, query.QueryString, "element_type = 'paragraph'")
	assert.Contains(t, query.QueryString, "OR")
	assert.Contains(t, query.QueryString, "AND")
	assert.Contains(t, query.QueryString, "NOT")
	assert.Contains(t, query.QueryString, "ORDER BY document_position ASC")
	assert.Contains(t, query.QueryString, "LIMIT 100")
}

// TestDuckDBTranslate_PartitionPruning validates partition pruning paths
func TestDuckDBTranslate_PartitionPruning(t *testing.T) {
	backend := &DuckDBBackend{
		parquetPath: "/data/analytics",
	}

	expr := NewExpression("elements").SelectAll()

	// With partition pruning
	queryWithPruning, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: true,
	})
	require.NoError(t, err)
	assert.Contains(t, queryWithPruning.QueryString, "element_type=*/version=*/date=*/source=*")

	// Without partition pruning
	queryWithoutPruning, err := backend.Translate(expr, TranslateOptions{
		EnablePartitions: false,
	})
	require.NoError(t, err)
	assert.Contains(t, queryWithoutPruning.QueryString, "/**/*.parquet")
}

// TestDuckDBSupportsFeature validates feature support
func TestDuckDBSupportsFeature(t *testing.T) {
	backend := &DuckDBBackend{}

	tests := []struct {
		feature  string
		expected bool
	}{
		{"jsonpath", true},
		{"regex", true},
		{"full_text_search", true},
		{"similarity", false},
		{"graph_traversal", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.feature, func(t *testing.T) {
			result := backend.SupportsFeature(tt.feature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDuckDBMapType validates type mapping
func TestDuckDBMapType(t *testing.T) {
	backend := &DuckDBBackend{}

	tests := []struct {
		duckdbType string
		expected   string
	}{
		{"VARCHAR", "string"},
		{"TEXT", "string"},
		{"INTEGER", "int64"},
		{"BIGINT", "int64"},
		{"DOUBLE", "float64"},
		{"FLOAT", "float64"},
		{"BOOLEAN", "bool"},
		{"TIMESTAMP", "timestamp"},
		{"JSON", "json"},
		{"UNKNOWN", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.duckdbType, func(t *testing.T) {
			result := backend.mapDuckDBType(tt.duckdbType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDuckDBFormatValue validates value formatting
func TestDuckDBFormatValue(t *testing.T) {
	backend := &DuckDBBackend{}

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "test", "'test'"},
		{"string_with_quote", "test's", "'test''s'"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backend.formatValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDuckDBFormatInValues validates IN clause value formatting
func TestDuckDBFormatInValues(t *testing.T) {
	backend := &DuckDBBackend{}

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string_array",
			value:    []string{"a", "b", "c"},
			expected: "'a', 'b', 'c'",
		},
		{
			name:     "int_array",
			value:    []int{1, 2, 3},
			expected: "1, 2, 3",
		},
		{
			name:     "interface_array",
			value:    []interface{}{"test", 42, true},
			expected: "'test', 42, true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backend.formatInValues(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDuckDBRegistration validates backend registration
func TestDuckDBRegistration(t *testing.T) {
	registry := NewBackendRegistry()

	// Register DuckDB backend
	err := registry.Register("duckdb", NewDuckDBBackend)
	require.NoError(t, err)

	// Verify registration
	assert.True(t, registry.IsRegistered("duckdb"))

	// Create backend through registry
	config := BackendConfig{
		Type:        "duckdb",
		ParquetPath: "/tmp/analytics",
	}

	backend, err := registry.Create(config)
	require.NoError(t, err)
	assert.Equal(t, "duckdb", backend.GetName())
}

// TestDuckDBBackendInterface validates interface compliance
func TestDuckDBBackendInterface(t *testing.T) {
	var _ QueryBackend = (*DuckDBBackend)(nil)
}

// TestDuckDBTranslate_NilExpression validates error handling
func TestDuckDBTranslate_NilExpression(t *testing.T) {
	backend := &DuckDBBackend{}

	query, err := backend.Translate(nil, TranslateOptions{})
	assert.Error(t, err)
	assert.Nil(t, query)
	assert.Contains(t, err.Error(), "expression cannot be nil")
}

// TestDuckDBExecute_NotInitialized validates initialization check
func TestDuckDBExecute_NotInitialized(t *testing.T) {
	backend := &DuckDBBackend{
		initialized: false,
	}

	ctx := context.Background()
	query := &NativeQuery{
		BackendType: "duckdb",
		QueryString: "SELECT 1",
	}

	result, err := backend.Execute(ctx, query)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestDuckDBExplain_NotInitialized validates initialization check for explain
func TestDuckDBExplain_NotInitialized(t *testing.T) {
	backend := &DuckDBBackend{
		initialized: false,
	}

	ctx := context.Background()
	query := &NativeQuery{
		BackendType: "duckdb",
		QueryString: "SELECT 1",
	}

	plan, err := backend.Explain(ctx, query)
	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestDuckDBClose validates cleanup
func TestDuckDBClose(t *testing.T) {
	backend := &DuckDBBackend{}

	// Close without initialization
	err := backend.Close()
	assert.NoError(t, err)

	// Close with initialization
	config := BackendConfig{Type: "duckdb"}
	ctx := context.Background()
	err = backend.Initialize(ctx, config)
	require.NoError(t, err)

	err = backend.Close()
	assert.NoError(t, err)
	assert.False(t, backend.initialized)
}
