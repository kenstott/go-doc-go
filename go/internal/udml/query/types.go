package query

import (
	"fmt"
	"time"
)

// Expression is the universal, backend-agnostic query representation
type Expression struct {
	// What to return
	Select []FieldSelection

	// Where to query from
	From string // "elements", "documents", "relationships"

	// Filtering
	Where *Predicate

	// Ordering
	OrderBy []OrderByClause

	// Limiting
	Limit  int
	Offset int

	// Aggregation
	GroupBy   []string
	Aggregate []AggregateFunc

	// Joins (for relationship queries)
	Joins []Join

	// Metadata
	QueryID   string
	Timestamp time.Time
}

// FieldSelection specifies which fields to return
type FieldSelection struct {
	Field string // Field name or JSONPath expression
	Alias string // Optional alias
}

// Predicate represents a filtering condition
type Predicate struct {
	Type     PredicateType // AND, OR, NOT, COMPARISON, JSONPATH, SIMILARITY
	Operator Operator      // For COMPARISON predicates

	// For logical predicates (AND, OR, NOT)
	Children []*Predicate

	// For comparison predicates
	Field string
	Value interface{}

	// For JSONPath predicates
	JSONPath string

	// For similarity predicates
	EmbeddingField string
	QueryVector    []float32
	Threshold      float64
	TopK           int
}

// PredicateType defines the type of predicate
type PredicateType string

const (
	PredicateAnd        PredicateType = "AND"
	PredicateOr         PredicateType = "OR"
	PredicateNot        PredicateType = "NOT"
	PredicateComparison PredicateType = "COMPARISON"
	PredicateJSONPath   PredicateType = "JSONPATH"
	PredicateSimilarity PredicateType = "SIMILARITY"
)

// Operator defines comparison operators
type Operator string

const (
	OpEqual              Operator = "="
	OpNotEqual           Operator = "!="
	OpGreaterThan        Operator = ">"
	OpGreaterThanOrEqual Operator = ">="
	OpLessThan           Operator = "<"
	OpLessThanOrEqual    Operator = "<="
	OpLike               Operator = "LIKE"
	OpILike              Operator = "ILIKE"
	OpIn                 Operator = "IN"
	OpNotIn              Operator = "NOT IN"
	OpIsNull             Operator = "IS NULL"
	OpIsNotNull          Operator = "IS NOT NULL"
	OpRegex              Operator = "REGEX"
)

// OrderByClause specifies ordering
type OrderByClause struct {
	Field      string
	Descending bool
}

// AggregateFunc specifies an aggregation function
type AggregateFunc struct {
	Function string // "count", "sum", "avg", "min", "max"
	Field    string
	Alias    string
}

// Join specifies a join operation
type Join struct {
	Type      JoinType // INNER, LEFT, RIGHT, FULL
	Table     string
	OnLeft    string
	OnRight   string
	Predicate *Predicate // Additional join conditions
}

// JoinType defines the type of join
type JoinType string

const (
	JoinInner JoinType = "INNER"
	JoinLeft  JoinType = "LEFT"
	JoinRight JoinType = "RIGHT"
	JoinFull  JoinType = "FULL"
)

// NativeQuery holds a backend-specific query
type NativeQuery struct {
	BackendType string // "duckdb", "postgresql", etc.
	QueryString string // Native SQL/Cypher/etc. query
	Parameters  []interface{}
	Metadata    map[string]interface{} // Backend-specific metadata
	SourceExpr  *Expression            // Original expression (for debugging)
}

// QueryResult holds query results
type QueryResult struct {
	// Schema
	Columns []ColumnInfo

	// Data
	Rows []Row

	// Metadata
	RowCount      int
	ExecutionTime time.Duration
	BackendType   string
	QueryID       string

	// Statistics
	Stats QueryStats
}

// ColumnInfo describes a result column
type ColumnInfo struct {
	Name     string
	Type     string // "string", "int64", "float64", "bool", "timestamp", "json"
	Nullable bool
}

// Row is a single result row (map of column_name -> value)
type Row map[string]interface{}

// QueryStats contains query execution statistics
type QueryStats struct {
	RowsScanned     int64
	PartitionsUsed  []string
	IndexesUsed     []string
	BytesProcessed  int64
	CacheHitRate    float64
	ParallelWorkers int
}

// Helper methods for building Expressions

// NewExpression creates a new Expression
func NewExpression(from string) *Expression {
	return &Expression{
		From:      from,
		Select:    []FieldSelection{},
		Timestamp: time.Now(),
	}
}

// SelectFields adds field selections
func (e *Expression) SelectFields(fields ...string) *Expression {
	for _, field := range fields {
		e.Select = append(e.Select, FieldSelection{Field: field})
	}
	return e
}

// SelectAll selects all fields
func (e *Expression) SelectAll() *Expression {
	e.Select = []FieldSelection{{Field: "*"}}
	return e
}

// Filter adds a WHERE predicate
func (e *Expression) Filter(predicate *Predicate) *Expression {
	e.Where = predicate
	return e
}

// Order adds ordering
func (e *Expression) Order(field string, desc bool) *Expression {
	e.OrderBy = append(e.OrderBy, OrderByClause{
		Field:      field,
		Descending: desc,
	})
	return e
}

// LimitResults sets limit and offset
func (e *Expression) LimitResults(limit, offset int) *Expression {
	e.Limit = limit
	e.Offset = offset
	return e
}

// Helper methods for building Predicates

// NewComparison creates a comparison predicate
func NewComparison(field string, op Operator, value interface{}) *Predicate {
	return &Predicate{
		Type:     PredicateComparison,
		Operator: op,
		Field:    field,
		Value:    value,
	}
}

// NewAnd creates an AND predicate
func NewAnd(predicates ...*Predicate) *Predicate {
	return &Predicate{
		Type:     PredicateAnd,
		Children: predicates,
	}
}

// NewOr creates an OR predicate
func NewOr(predicates ...*Predicate) *Predicate {
	return &Predicate{
		Type:     PredicateOr,
		Children: predicates,
	}
}

// NewNot creates a NOT predicate
func NewNot(predicate *Predicate) *Predicate {
	return &Predicate{
		Type:     PredicateNot,
		Children: []*Predicate{predicate},
	}
}

// NewJSONPath creates a JSONPath predicate
func NewJSONPath(jsonPath string, op Operator, value interface{}) *Predicate {
	return &Predicate{
		Type:     PredicateJSONPath,
		JSONPath: jsonPath,
		Operator: op,
		Value:    value,
	}
}

// NewSimilarity creates a similarity predicate
func NewSimilarity(embeddingField string, queryVector []float32, threshold float64, topK int) *Predicate {
	return &Predicate{
		Type:           PredicateSimilarity,
		EmbeddingField: embeddingField,
		QueryVector:    queryVector,
		Threshold:      threshold,
		TopK:           topK,
	}
}

// String returns a human-readable representation of the Expression
func (e *Expression) String() string {
	return fmt.Sprintf("SELECT %v FROM %s WHERE %v", e.Select, e.From, e.Where)
}

// String returns a human-readable representation of the Predicate
func (p *Predicate) String() string {
	switch p.Type {
	case PredicateAnd:
		return fmt.Sprintf("(%v AND %v)", p.Children[0], p.Children[1])
	case PredicateOr:
		return fmt.Sprintf("(%v OR %v)", p.Children[0], p.Children[1])
	case PredicateNot:
		return fmt.Sprintf("NOT %v", p.Children[0])
	case PredicateComparison:
		return fmt.Sprintf("%s %s %v", p.Field, p.Operator, p.Value)
	case PredicateJSONPath:
		return fmt.Sprintf("JSONPath(%s) %s %v", p.JSONPath, p.Operator, p.Value)
	case PredicateSimilarity:
		return fmt.Sprintf("Similarity(%s, threshold=%.2f, topK=%d)", p.EmbeddingField, p.Threshold, p.TopK)
	default:
		return fmt.Sprintf("UnknownPredicate(%s)", p.Type)
	}
}
