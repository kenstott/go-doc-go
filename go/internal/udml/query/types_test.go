package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewExpression validates Expression creation
func TestNewExpression(t *testing.T) {
	expr := NewExpression("elements")

	assert.Equal(t, "elements", expr.From)
	assert.Empty(t, expr.Select)
	assert.Nil(t, expr.Where)
	assert.NotZero(t, expr.Timestamp)
}

// TestExpressionSelectFields validates field selection
func TestExpressionSelectFields(t *testing.T) {
	expr := NewExpression("elements").
		SelectFields("element_id", "content", "element_type")

	require.Len(t, expr.Select, 3)
	assert.Equal(t, "element_id", expr.Select[0].Field)
	assert.Equal(t, "content", expr.Select[1].Field)
	assert.Equal(t, "element_type", expr.Select[2].Field)
}

// TestExpressionSelectAll validates select all
func TestExpressionSelectAll(t *testing.T) {
	expr := NewExpression("elements").SelectAll()

	require.Len(t, expr.Select, 1)
	assert.Equal(t, "*", expr.Select[0].Field)
}

// TestExpressionFilter validates filtering
func TestExpressionFilter(t *testing.T) {
	predicate := NewComparison("element_type", OpEqual, "paragraph")
	expr := NewExpression("elements").
		SelectAll().
		Filter(predicate)

	require.NotNil(t, expr.Where)
	assert.Equal(t, PredicateComparison, expr.Where.Type)
	assert.Equal(t, "element_type", expr.Where.Field)
	assert.Equal(t, OpEqual, expr.Where.Operator)
	assert.Equal(t, "paragraph", expr.Where.Value)
}

// TestExpressionOrder validates ordering
func TestExpressionOrder(t *testing.T) {
	expr := NewExpression("elements").
		SelectAll().
		Order("element_order", false).
		Order("document_position", true)

	require.Len(t, expr.OrderBy, 2)
	assert.Equal(t, "element_order", expr.OrderBy[0].Field)
	assert.False(t, expr.OrderBy[0].Descending)
	assert.Equal(t, "document_position", expr.OrderBy[1].Field)
	assert.True(t, expr.OrderBy[1].Descending)
}

// TestExpressionLimitResults validates limit and offset
func TestExpressionLimitResults(t *testing.T) {
	expr := NewExpression("elements").
		SelectAll().
		LimitResults(100, 50)

	assert.Equal(t, 100, expr.Limit)
	assert.Equal(t, 50, expr.Offset)
}

// TestComplexExpression validates complex query building
func TestComplexExpression(t *testing.T) {
	// Build: SELECT element_id, content FROM elements
	//        WHERE element_type = 'paragraph' AND page_number > 5
	//        ORDER BY document_position ASC
	//        LIMIT 10

	predicate := NewAnd(
		NewComparison("element_type", OpEqual, "paragraph"),
		NewComparison("page_number", OpGreaterThan, 5),
	)

	expr := NewExpression("elements").
		SelectFields("element_id", "content").
		Filter(predicate).
		Order("document_position", false).
		LimitResults(10, 0)

	assert.Equal(t, "elements", expr.From)
	require.Len(t, expr.Select, 2)
	assert.NotNil(t, expr.Where)
	assert.Equal(t, PredicateAnd, expr.Where.Type)
	require.Len(t, expr.OrderBy, 1)
	assert.Equal(t, 10, expr.Limit)
	assert.Equal(t, 0, expr.Offset)
}

// TestNewComparison validates comparison predicate creation
func TestNewComparison(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		op       Operator
		value    interface{}
		expected string
	}{
		{"equal", "element_type", OpEqual, "paragraph", "element_type = paragraph"},
		{"not_equal", "element_type", OpNotEqual, "header", "element_type != header"},
		{"greater_than", "page_number", OpGreaterThan, 10, "page_number > 10"},
		{"less_than_or_equal", "section_level", OpLessThanOrEqual, 3, "section_level <= 3"},
		{"like", "content", OpLike, "%test%", "content LIKE %test%"},
		{"in", "element_type", OpIn, []string{"paragraph", "header"}, "element_type IN [paragraph header]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predicate := NewComparison(tt.field, tt.op, tt.value)

			assert.Equal(t, PredicateComparison, predicate.Type)
			assert.Equal(t, tt.field, predicate.Field)
			assert.Equal(t, tt.op, predicate.Operator)
			assert.Equal(t, tt.value, predicate.Value)
			assert.Contains(t, predicate.String(), tt.field)
		})
	}
}

// TestNewAnd validates AND predicate creation
func TestNewAnd(t *testing.T) {
	pred1 := NewComparison("element_type", OpEqual, "paragraph")
	pred2 := NewComparison("page_number", OpGreaterThan, 5)

	andPred := NewAnd(pred1, pred2)

	assert.Equal(t, PredicateAnd, andPred.Type)
	require.Len(t, andPred.Children, 2)
	assert.Equal(t, pred1, andPred.Children[0])
	assert.Equal(t, pred2, andPred.Children[1])
}

// TestNewOr validates OR predicate creation
func TestNewOr(t *testing.T) {
	pred1 := NewComparison("element_type", OpEqual, "paragraph")
	pred2 := NewComparison("element_type", OpEqual, "header")

	orPred := NewOr(pred1, pred2)

	assert.Equal(t, PredicateOr, orPred.Type)
	require.Len(t, orPred.Children, 2)
	assert.Equal(t, pred1, orPred.Children[0])
	assert.Equal(t, pred2, orPred.Children[1])
}

// TestNewNot validates NOT predicate creation
func TestNewNot(t *testing.T) {
	pred := NewComparison("element_type", OpEqual, "paragraph")
	notPred := NewNot(pred)

	assert.Equal(t, PredicateNot, notPred.Type)
	require.Len(t, notPred.Children, 1)
	assert.Equal(t, pred, notPred.Children[0])
}

// TestNestedPredicates validates nested logical predicates
func TestNestedPredicates(t *testing.T) {
	// Build: (element_type = 'paragraph' OR element_type = 'header')
	//        AND page_number > 5
	//        AND NOT (section_level > 3)

	typeFilter := NewOr(
		NewComparison("element_type", OpEqual, "paragraph"),
		NewComparison("element_type", OpEqual, "header"),
	)

	pageFilter := NewComparison("page_number", OpGreaterThan, 5)

	sectionFilter := NewNot(
		NewComparison("section_level", OpGreaterThan, 3),
	)

	combined := NewAnd(
		NewAnd(typeFilter, pageFilter),
		sectionFilter,
	)

	assert.Equal(t, PredicateAnd, combined.Type)
	require.Len(t, combined.Children, 2)

	// Verify nested structure
	leftAnd := combined.Children[0]
	assert.Equal(t, PredicateAnd, leftAnd.Type)
	assert.Equal(t, PredicateOr, leftAnd.Children[0].Type)

	rightNot := combined.Children[1]
	assert.Equal(t, PredicateNot, rightNot.Type)
}

// TestNewJSONPath validates JSONPath predicate creation
func TestNewJSONPath(t *testing.T) {
	jsonPath := "$.metadata.font_size"
	predicate := NewJSONPath(jsonPath, OpGreaterThan, 12)

	assert.Equal(t, PredicateJSONPath, predicate.Type)
	assert.Equal(t, jsonPath, predicate.JSONPath)
	assert.Equal(t, OpGreaterThan, predicate.Operator)
	assert.Equal(t, 12, predicate.Value)
}

// TestNewSimilarity validates similarity predicate creation
func TestNewSimilarity(t *testing.T) {
	queryVector := []float32{0.1, 0.2, 0.3, 0.4}
	predicate := NewSimilarity("contextual_embedding", queryVector, 0.8, 10)

	assert.Equal(t, PredicateSimilarity, predicate.Type)
	assert.Equal(t, "contextual_embedding", predicate.EmbeddingField)
	assert.Equal(t, queryVector, predicate.QueryVector)
	assert.Equal(t, 0.8, predicate.Threshold)
	assert.Equal(t, 10, predicate.TopK)
}

// TestPredicateString validates String() representation
func TestPredicateString(t *testing.T) {
	tests := []struct {
		name      string
		predicate *Predicate
		contains  []string
	}{
		{
			name:      "comparison",
			predicate: NewComparison("element_type", OpEqual, "paragraph"),
			contains:  []string{"element_type", "=", "paragraph"},
		},
		{
			name: "and",
			predicate: NewAnd(
				NewComparison("page_number", OpGreaterThan, 5),
				NewComparison("section_level", OpLessThan, 3),
			),
			contains: []string{"AND", "page_number", "section_level"},
		},
		{
			name: "or",
			predicate: NewOr(
				NewComparison("element_type", OpEqual, "paragraph"),
				NewComparison("element_type", OpEqual, "header"),
			),
			contains: []string{"OR", "element_type"},
		},
		{
			name:      "not",
			predicate: NewNot(NewComparison("element_type", OpEqual, "code_block")),
			contains:  []string{"NOT", "element_type"},
		},
		{
			name:      "jsonpath",
			predicate: NewJSONPath("$.metadata.color", OpEqual, "#000000"),
			contains:  []string{"JSONPath", "metadata.color"},
		},
		{
			name:      "similarity",
			predicate: NewSimilarity("contextual_embedding", []float32{0.1, 0.2}, 0.8, 10),
			contains:  []string{"Similarity", "contextual_embedding", "0.80", "10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.predicate.String()
			for _, substr := range tt.contains {
				assert.Contains(t, str, substr)
			}
		})
	}
}

// TestExpressionString validates Expression String() representation
func TestExpressionString(t *testing.T) {
	expr := NewExpression("elements").
		SelectFields("element_id", "content").
		Filter(NewComparison("element_type", OpEqual, "paragraph"))

	str := expr.String()
	assert.Contains(t, str, "elements")
	assert.Contains(t, str, "element_id")
	assert.Contains(t, str, "content")
}

// TestNativeQuery validates native query structure
func TestNativeQuery(t *testing.T) {
	expr := NewExpression("elements").SelectAll()

	query := &NativeQuery{
		BackendType: "duckdb",
		QueryString: "SELECT * FROM elements",
		Parameters:  []interface{}{},
		Metadata:    map[string]interface{}{"optimized": true},
		SourceExpr:  expr,
	}

	assert.Equal(t, "duckdb", query.BackendType)
	assert.Equal(t, "SELECT * FROM elements", query.QueryString)
	assert.Empty(t, query.Parameters)
	assert.Equal(t, true, query.Metadata["optimized"])
	assert.Equal(t, expr, query.SourceExpr)
}

// TestQueryResult validates query result structure
func TestQueryResult(t *testing.T) {
	result := &QueryResult{
		Columns: []ColumnInfo{
			{Name: "element_id", Type: "string", Nullable: false},
			{Name: "content", Type: "string", Nullable: true},
		},
		Rows: []Row{
			{"element_id": "elem1", "content": "Test content"},
			{"element_id": "elem2", "content": nil},
		},
		RowCount:    2,
		BackendType: "duckdb",
		QueryID:     "query123",
		Stats: QueryStats{
			RowsScanned:    1000,
			PartitionsUsed: []string{"element_type=paragraph", "element_type=header"},
			BytesProcessed: 1024000,
		},
	}

	require.Len(t, result.Columns, 2)
	assert.Equal(t, "element_id", result.Columns[0].Name)
	assert.Equal(t, "string", result.Columns[0].Type)
	assert.False(t, result.Columns[0].Nullable)

	require.Len(t, result.Rows, 2)
	assert.Equal(t, "elem1", result.Rows[0]["element_id"])
	assert.Equal(t, "Test content", result.Rows[0]["content"])
	assert.Nil(t, result.Rows[1]["content"])

	assert.Equal(t, 2, result.RowCount)
	assert.Equal(t, "duckdb", result.BackendType)
	assert.Equal(t, int64(1000), result.Stats.RowsScanned)
	assert.Equal(t, int64(1024000), result.Stats.BytesProcessed)
}

// TestFieldSelection validates field selection structure
func TestFieldSelection(t *testing.T) {
	tests := []struct {
		name  string
		field FieldSelection
	}{
		{"simple", FieldSelection{Field: "element_id"}},
		{"with_alias", FieldSelection{Field: "element_type", Alias: "type"}},
		{"jsonpath", FieldSelection{Field: "$.metadata.font_size", Alias: "font_size"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.field.Field)
		})
	}
}

// TestJoin validates join structure
func TestJoin(t *testing.T) {
	join := Join{
		Type:    JoinInner,
		Table:   "documents",
		OnLeft:  "doc_id",
		OnRight: "doc_id",
		Predicate: NewComparison("source_name", OpEqual, "test_source"),
	}

	assert.Equal(t, JoinInner, join.Type)
	assert.Equal(t, "documents", join.Table)
	assert.Equal(t, "doc_id", join.OnLeft)
	assert.Equal(t, "doc_id", join.OnRight)
	assert.NotNil(t, join.Predicate)
}

// TestAggregateFunc validates aggregate function structure
func TestAggregateFunc(t *testing.T) {
	tests := []struct {
		name string
		agg  AggregateFunc
	}{
		{"count", AggregateFunc{Function: "count", Field: "*", Alias: "total"}},
		{"sum", AggregateFunc{Function: "sum", Field: "page_number", Alias: "total_pages"}},
		{"avg", AggregateFunc{Function: "avg", Field: "element_order", Alias: "avg_order"}},
		{"min", AggregateFunc{Function: "min", Field: "document_position", Alias: "min_pos"}},
		{"max", AggregateFunc{Function: "max", Field: "document_position", Alias: "max_pos"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.agg.Function)
			assert.NotEmpty(t, tt.agg.Field)
			assert.NotEmpty(t, tt.agg.Alias)
		})
	}
}
