package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

// init registers the DuckDB backend in the global registry
func init() {
	Register("duckdb", NewDuckDBBackend)
}

// DuckDBBackend implements QueryBackend using DuckDB for Parquet queries
type DuckDBBackend struct {
	db          *sql.DB
	parquetPath string
	initialized bool
	version     string
}

// NewDuckDBBackend creates a new DuckDB backend
func NewDuckDBBackend(config BackendConfig) (QueryBackend, error) {
	backend := &DuckDBBackend{
		parquetPath: config.ParquetPath,
		version:     "0.10.0", // DuckDB version
	}
	return backend, nil
}

// GetName returns the backend identifier
func (d *DuckDBBackend) GetName() string {
	return "duckdb"
}

// GetVersion returns the DuckDB version
func (d *DuckDBBackend) GetVersion() string {
	return d.version
}

// Initialize opens DuckDB connection and prepares for queries
func (d *DuckDBBackend) Initialize(ctx context.Context, config BackendConfig) error {
	if d.initialized {
		return nil
	}

	// Open DuckDB connection (in-memory by default)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}

	d.db = db
	d.parquetPath = config.ParquetPath
	d.initialized = true

	// Set DuckDB configuration for optimal Parquet reading
	configStatements := []string{
		"SET threads TO 8",                    // Parallel processing
		"SET enable_object_cache TO true",     // Cache Parquet metadata
		"SET preserve_insertion_order TO false", // Allow reordering for performance
	}

	for _, stmt := range configStatements {
		if _, err := d.db.ExecContext(ctx, stmt); err != nil {
			// Non-fatal - continue if config fails
			fmt.Printf("Warning: DuckDB config failed: %s\n", err)
		}
	}

	return nil
}

// Translate converts an Expression to DuckDB SQL
func (d *DuckDBBackend) Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error) {
	if expr == nil {
		return nil, fmt.Errorf("expression cannot be nil")
	}

	// Build SELECT clause
	selectClause := d.buildSelectClause(expr.Select)

	// Build FROM clause with Hive partition glob pattern
	fromClause := d.buildFromClause(expr.From, opts.EnablePartitions)

	// Build WHERE clause
	whereClause := ""
	if expr.Where != nil {
		where, params, err := d.buildWhereClause(expr.Where)
		if err != nil {
			return nil, fmt.Errorf("failed to build WHERE clause: %w", err)
		}
		whereClause = "WHERE " + where
		_ = params // DuckDB uses positional parameters
	}

	// Build ORDER BY clause
	orderByClause := ""
	if len(expr.OrderBy) > 0 {
		orderByClause = "ORDER BY " + d.buildOrderByClause(expr.OrderBy)
	}

	// Build LIMIT/OFFSET clause
	limitClause := ""
	if opts.Limit > 0 || expr.Limit > 0 {
		limit := opts.Limit
		if limit == 0 {
			limit = expr.Limit
		}
		limitClause = fmt.Sprintf("LIMIT %d", limit)

		offset := opts.Offset
		if offset == 0 {
			offset = expr.Offset
		}
		if offset > 0 {
			limitClause += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// Build GROUP BY clause
	groupByClause := ""
	if len(expr.GroupBy) > 0 {
		groupByClause = "GROUP BY " + strings.Join(expr.GroupBy, ", ")
	}

	// Assemble full query
	parts := []string{selectClause, fromClause}
	if whereClause != "" {
		parts = append(parts, whereClause)
	}
	if groupByClause != "" {
		parts = append(parts, groupByClause)
	}
	if orderByClause != "" {
		parts = append(parts, orderByClause)
	}
	if limitClause != "" {
		parts = append(parts, limitClause)
	}

	queryString := strings.Join(parts, " ")

	return &NativeQuery{
		BackendType: "duckdb",
		QueryString: queryString,
		Parameters:  []interface{}{}, // DuckDB uses inline parameters in this implementation
		Metadata: map[string]interface{}{
			"partition_pruning_enabled": opts.EnablePartitions,
			"pushdown_enabled":          opts.EnablePushdown,
		},
		SourceExpr: expr,
	}, nil
}

// buildSelectClause builds SELECT clause from field selections
func (d *DuckDBBackend) buildSelectClause(selections []FieldSelection) string {
	if len(selections) == 0 || (len(selections) == 1 && selections[0].Field == "*") {
		return "SELECT *"
	}

	fields := make([]string, len(selections))
	for i, sel := range selections {
		if sel.Alias != "" {
			fields[i] = fmt.Sprintf("%s AS %s", sel.Field, sel.Alias)
		} else {
			fields[i] = sel.Field
		}
	}

	return "SELECT " + strings.Join(fields, ", ")
}

// buildFromClause builds FROM clause with Hive partition support
func (d *DuckDBBackend) buildFromClause(tableName string, enablePartitions bool) string {
	if d.parquetPath == "" {
		return fmt.Sprintf("FROM %s", tableName)
	}

	if enablePartitions {
		// Use Hive partition glob pattern
		// e.g., 'analytics/elements/element_type=*/version=*/date=*/source=*/*.parquet'
		pattern := fmt.Sprintf("'%s/%s/element_type=*/version=*/date=*/source=*/*.parquet'",
			d.parquetPath, tableName)
		return fmt.Sprintf("FROM %s", pattern)
	}

	// Flat pattern (no partition pruning)
	pattern := fmt.Sprintf("'%s/%s/**/*.parquet'", d.parquetPath, tableName)
	return fmt.Sprintf("FROM %s", pattern)
}

// buildWhereClause builds WHERE clause from predicate
func (d *DuckDBBackend) buildWhereClause(pred *Predicate) (string, []interface{}, error) {
	switch pred.Type {
	case PredicateAnd:
		return d.buildLogicalPredicate(pred, "AND")
	case PredicateOr:
		return d.buildLogicalPredicate(pred, "OR")
	case PredicateNot:
		child, params, err := d.buildWhereClause(pred.Children[0])
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("NOT (%s)", child), params, nil
	case PredicateComparison:
		return d.buildComparisonPredicate(pred)
	case PredicateJSONPath:
		return d.buildJSONPathPredicate(pred)
	case PredicateSimilarity:
		return "", nil, fmt.Errorf("similarity predicates not yet implemented")
	default:
		return "", nil, fmt.Errorf("unknown predicate type: %s", pred.Type)
	}
}

// buildLogicalPredicate builds AND/OR predicates
func (d *DuckDBBackend) buildLogicalPredicate(pred *Predicate, op string) (string, []interface{}, error) {
	if len(pred.Children) == 0 {
		return "", nil, fmt.Errorf("%s predicate requires at least one child", op)
	}

	parts := make([]string, len(pred.Children))
	var allParams []interface{}

	for i, child := range pred.Children {
		part, params, err := d.buildWhereClause(child)
		if err != nil {
			return "", nil, err
		}
		parts[i] = fmt.Sprintf("(%s)", part)
		allParams = append(allParams, params...)
	}

	return strings.Join(parts, " "+op+" "), allParams, nil
}

// buildComparisonPredicate builds comparison predicates
func (d *DuckDBBackend) buildComparisonPredicate(pred *Predicate) (string, []interface{}, error) {
	field := pred.Field
	op := string(pred.Operator)
	value := pred.Value

	switch pred.Operator {
	case OpEqual, OpNotEqual, OpGreaterThan, OpGreaterThanOrEqual, OpLessThan, OpLessThanOrEqual:
		return fmt.Sprintf("%s %s %v", field, op, d.formatValue(value)), nil, nil
	case OpLike, OpILike:
		return fmt.Sprintf("%s %s '%v'", field, op, value), nil, nil
	case OpIn:
		values := d.formatInValues(value)
		return fmt.Sprintf("%s IN (%s)", field, values), nil, nil
	case OpNotIn:
		values := d.formatInValues(value)
		return fmt.Sprintf("%s NOT IN (%s)", field, values), nil, nil
	case OpIsNull:
		return fmt.Sprintf("%s IS NULL", field), nil, nil
	case OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", field), nil, nil
	case OpRegex:
		return fmt.Sprintf("regexp_matches(%s, '%v')", field, value), nil, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", op)
	}
}

// buildJSONPathPredicate builds JSONPath predicates using DuckDB's JSON functions
func (d *DuckDBBackend) buildJSONPathPredicate(pred *Predicate) (string, []interface{}, error) {
	// DuckDB JSON access: json_extract(column, '$.path')
	jsonExpr := fmt.Sprintf("json_extract(%s, '%s')", pred.Field, pred.JSONPath)

	switch pred.Operator {
	case OpEqual:
		return fmt.Sprintf("%s = %v", jsonExpr, d.formatValue(pred.Value)), nil, nil
	case OpNotEqual:
		return fmt.Sprintf("%s != %v", jsonExpr, d.formatValue(pred.Value)), nil, nil
	case OpGreaterThan:
		return fmt.Sprintf("%s > %v", jsonExpr, d.formatValue(pred.Value)), nil, nil
	case OpLessThan:
		return fmt.Sprintf("%s < %v", jsonExpr, d.formatValue(pred.Value)), nil, nil
	default:
		return "", nil, fmt.Errorf("unsupported JSONPath operator: %s", pred.Operator)
	}
}

// buildOrderByClause builds ORDER BY clause
func (d *DuckDBBackend) buildOrderByClause(orderBy []OrderByClause) string {
	parts := make([]string, len(orderBy))
	for i, clause := range orderBy {
		direction := "ASC"
		if clause.Descending {
			direction = "DESC"
		}
		parts[i] = fmt.Sprintf("%s %s", clause.Field, direction)
	}
	return strings.Join(parts, ", ")
}

// formatValue formats a value for SQL
func (d *DuckDBBackend) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// formatInValues formats values for IN clause
func (d *DuckDBBackend) formatInValues(value interface{}) string {
	switch v := value.(type) {
	case []string:
		quoted := make([]string, len(v))
		for i, s := range v {
			quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''"))
		}
		return strings.Join(quoted, ", ")
	case []int:
		strs := make([]string, len(v))
		for i, n := range v {
			strs[i] = fmt.Sprintf("%d", n)
		}
		return strings.Join(strs, ", ")
	case []interface{}:
		strs := make([]string, len(v))
		for i, val := range v {
			strs[i] = d.formatValue(val)
		}
		return strings.Join(strs, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Execute runs a DuckDB query and returns results
func (d *DuckDBBackend) Execute(ctx context.Context, query *NativeQuery) (*QueryResult, error) {
	if !d.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	startTime := time.Now()

	// Execute query
	rows, err := d.db.QueryContext(ctx, query.QueryString, query.Parameters...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column info
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %w", err)
	}

	// Build column info
	columnInfo := make([]ColumnInfo, len(columns))
	for i, col := range columns {
		columnInfo[i] = ColumnInfo{
			Name:     col,
			Type:     d.mapDuckDBType(columnTypes[i].DatabaseTypeName()),
			Nullable: true, // DuckDB columns are nullable by default
		}
	}

	// Parse rows
	resultRows := []Row{}
	for rows.Next() {
		// Create slice for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build row map
		row := make(Row)
		for i, col := range columns {
			row[col] = values[i]
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	executionTime := time.Since(startTime)

	return &QueryResult{
		Columns:       columnInfo,
		Rows:          resultRows,
		RowCount:      len(resultRows),
		ExecutionTime: executionTime,
		BackendType:   "duckdb",
		QueryID:       query.SourceExpr.QueryID,
		Stats: QueryStats{
			RowsScanned: int64(len(resultRows)),
		},
	}, nil
}

// mapDuckDBType maps DuckDB types to generic types
func (d *DuckDBBackend) mapDuckDBType(duckdbType string) string {
	switch strings.ToUpper(duckdbType) {
	case "VARCHAR", "TEXT":
		return "string"
	case "INTEGER", "BIGINT", "SMALLINT", "TINYINT":
		return "int64"
	case "DOUBLE", "FLOAT", "DECIMAL":
		return "float64"
	case "BOOLEAN":
		return "bool"
	case "TIMESTAMP", "DATE", "TIME":
		return "timestamp"
	case "JSON":
		return "json"
	default:
		return "string"
	}
}

// SupportsFeature checks if DuckDB supports a feature
func (d *DuckDBBackend) SupportsFeature(feature string) bool {
	switch feature {
	case "jsonpath":
		return true
	case "regex":
		return true
	case "full_text_search":
		return true // DuckDB has FTS extension
	case "similarity":
		return false // Not implemented yet
	case "graph_traversal":
		return false
	default:
		return false
	}
}

// Explain returns the DuckDB query execution plan
func (d *DuckDBBackend) Explain(ctx context.Context, query *NativeQuery) (*QueryPlan, error) {
	if !d.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	// Get query plan using EXPLAIN
	explainQuery := "EXPLAIN " + query.QueryString
	rows, err := d.db.QueryContext(ctx, explainQuery)
	if err != nil {
		return nil, fmt.Errorf("explain failed: %w", err)
	}
	defer rows.Close()

	// Get column count
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Parse explain output - DuckDB may return multiple columns
	var planLines []string
	for rows.Next() {
		// Create slice for all columns
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan explain output: %w", err)
		}

		// Concatenate all columns (usually just the first contains the plan)
		var parts []string
		for _, val := range values {
			if val != nil {
				if str, ok := val.(string); ok && str != "" {
					parts = append(parts, str)
				} else if bytes, ok := val.([]byte); ok && len(bytes) > 0 {
					parts = append(parts, string(bytes))
				}
			}
		}
		if len(parts) > 0 {
			planLines = append(planLines, strings.Join(parts, " "))
		}
	}

	rawPlan := strings.Join(planLines, "\n")

	// Parse plan steps (simplified)
	steps := d.parsePlanSteps(planLines)

	return &QueryPlan{
		BackendName:   "duckdb",
		RawPlan:       rawPlan,
		Steps:         steps,
		EstimatedCost: 0.0, // DuckDB doesn't provide cost in EXPLAIN output
	}, nil
}

// parsePlanSteps parses DuckDB EXPLAIN output into steps
func (d *DuckDBBackend) parsePlanSteps(planLines []string) []QueryPlanStep {
	steps := []QueryPlanStep{}

	for _, line := range planLines {
		step := QueryPlanStep{
			Operator:    d.extractOperator(line),
			Description: line,
		}
		steps = append(steps, step)
	}

	return steps
}

// extractOperator extracts operator name from plan line
func (d *DuckDBBackend) extractOperator(line string) string {
	// Simple extraction - look for common operators
	line = strings.TrimSpace(line)
	if strings.Contains(line, "SCAN") {
		return "SCAN"
	} else if strings.Contains(line, "FILTER") {
		return "FILTER"
	} else if strings.Contains(line, "PROJECT") {
		return "PROJECT"
	} else if strings.Contains(line, "JOIN") {
		return "JOIN"
	} else if strings.Contains(line, "AGG") {
		return "AGGREGATE"
	} else if strings.Contains(line, "SORT") {
		return "SORT"
	}
	return "UNKNOWN"
}

// Close closes the DuckDB connection
func (d *DuckDBBackend) Close() error {
	if d.db != nil {
		d.initialized = false
		return d.db.Close()
	}
	return nil
}
