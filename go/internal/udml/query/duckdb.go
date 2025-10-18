package query

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
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

	// Check if this is a JOIN expression (contains "JOIN" keyword)
	if strings.Contains(strings.ToUpper(tableName), " JOIN ") {
		// Parse JOIN expression and build paths for each table
		return d.buildJoinFromClause(tableName)
	}

	// Simple table name - use wildcard pattern with AS alias
	// Partition pruning handled by DuckDB
	pattern := fmt.Sprintf("'%s/%s/**/*.parquet' AS %s", d.parquetPath, tableName, tableName)
	return fmt.Sprintf("FROM %s", pattern)
}

// buildJoinFromClause handles JOIN expressions by expanding table names to parquet paths
func (d *DuckDBBackend) buildJoinFromClause(joinExpr string) string {
	// Replace table names with parquet paths
	// Example: "elements INNER JOIN embeddings ON elements.element_id = embeddings.element_id"
	// Becomes: "'path/elements/**/*.parquet' AS elements INNER JOIN 'path/embeddings/**/*.parquet' AS embeddings ON elements.element_id = embeddings.element_id"

	result := joinExpr

	// Find all table names (words before JOIN or ON keywords)
	// Simple heuristic: look for table names that are valid identifiers
	tableNames := []string{"elements", "embeddings", "documents", "relationships"}

	for _, table := range tableNames {
		// Replace standalone table name (not part of a qualified column reference)
		// We need to be careful not to replace "elements.element_id" -> only replace the table reference itself
		pattern := fmt.Sprintf("'%s/%s/**/*.parquet' AS %s", d.parquetPath, table, table)

		// Replace patterns like "elements INNER JOIN" with pattern
		result = strings.Replace(result, table+" INNER JOIN", pattern+" INNER JOIN", -1)
		result = strings.Replace(result, table+" LEFT JOIN", pattern+" LEFT JOIN", -1)
		result = strings.Replace(result, table+" RIGHT JOIN", pattern+" RIGHT JOIN", -1)
		result = strings.Replace(result, table+" JOIN", pattern+" JOIN", -1)

		// Handle "INNER JOIN tablename ON" - replace the table after JOIN
		result = strings.Replace(result, "INNER JOIN "+table+" ON", "INNER JOIN "+pattern+" ON", -1)
		result = strings.Replace(result, "LEFT JOIN "+table+" ON", "LEFT JOIN "+pattern+" ON", -1)
		result = strings.Replace(result, "RIGHT JOIN "+table+" ON", "RIGHT JOIN "+pattern+" ON", -1)
		result = strings.Replace(result, "JOIN "+table+" ON", "JOIN "+pattern+" ON", -1)

		// Handle case where table is at the start
		if strings.HasPrefix(result, table+" ") {
			result = pattern + result[len(table):]
		}

		// Handle "ON table.column" - need to keep the alias
		result = strings.Replace(result, "ON "+table+".", "ON "+table+".", -1)
	}

	return "FROM " + result
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
		// Similarity predicates require special handling
		// We'll fetch candidates and compute similarity in-memory
		// For now, just return a basic filter that selects rows with the embedding field
		if pred.EmbeddingField == "" {
			return "", nil, fmt.Errorf("similarity predicate requires embedding field")
		}
		// Return a filter that ensures the embedding field exists
		return fmt.Sprintf("%s IS NOT NULL", pred.EmbeddingField), nil, nil
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
	case OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= %v", jsonExpr, d.formatValue(pred.Value)), nil, nil
	case OpLessThan:
		return fmt.Sprintf("%s < %v", jsonExpr, d.formatValue(pred.Value)), nil, nil
	case OpLessThanOrEqual:
		return fmt.Sprintf("%s <= %v", jsonExpr, d.formatValue(pred.Value)), nil, nil
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

// ExecuteRaw executes a raw SQL query with parameters and returns QueryResult
func (d *DuckDBBackend) ExecuteRaw(ctx context.Context, queryString string, args ...interface{}) (*QueryResult, error) {
	if !d.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	startTime := time.Now()

	// Execute query
	rows, err := d.db.QueryContext(ctx, queryString, args...)
	if err != nil {
		return nil, fmt.Errorf("raw query execution failed: %w", err)
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
			Nullable: true,
		}
	}

	// Parse rows
	resultRows := []Row{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

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
		Stats: QueryStats{
			RowsScanned: int64(len(resultRows)),
		},
	}, nil
}

// QueryRaw executes a raw SQL query and returns sql.Rows for direct iteration
// This is useful for streaming large result sets without loading all data into memory
func (d *DuckDBBackend) QueryRaw(ctx context.Context, queryString string, args ...interface{}) (*sql.Rows, error) {
	if !d.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	return d.db.QueryContext(ctx, queryString, args...)
}

// Close closes the DuckDB connection
func (d *DuckDBBackend) Close() error {
	if d.db != nil {
		d.initialized = false
		return d.db.Close()
	}
	return nil
}

// ApplySimilarityFilter applies similarity filtering to query results
// This is done in-memory since DuckDB doesn't have native vector similarity
func ApplySimilarityFilter(results *QueryResult, pred *Predicate) (*QueryResult, error) {
	if pred == nil || pred.Type != PredicateSimilarity {
		return results, nil
	}

	if pred.EmbeddingField == "" {
		return nil, fmt.Errorf("similarity predicate requires embedding field")
	}

	if len(pred.QueryVector) == 0 {
		return nil, fmt.Errorf("similarity predicate requires query vector")
	}

	// Convert query vector to float64 for comparison
	queryVec := ConvertFloat32ToFloat64(pred.QueryVector)

	// Calculate similarity for each row
	type scoredRow struct {
		row        Row
		similarity float64
	}
	scoredRows := make([]scoredRow, 0, len(results.Rows))

	for _, row := range results.Rows {
		// Extract embedding vector from row
		embeddingVal, ok := row[pred.EmbeddingField]
		if !ok {
			continue // Skip rows without embedding
		}

		// Try to parse embedding as []float64
		var embedding []float64
		switch v := embeddingVal.(type) {
		case []float64:
			embedding = v
		case []float32:
			embedding = ConvertFloat32ToFloat64(v)
		case []interface{}:
			// Convert interface slice to float64
			embedding = make([]float64, len(v))
			for i, val := range v {
				switch fval := val.(type) {
				case float64:
					embedding[i] = fval
				case float32:
					embedding[i] = float64(fval)
				default:
					continue // Skip malformed embeddings
				}
			}
		default:
			continue // Skip non-vector fields
		}

		// Calculate cosine similarity
		similarity, err := CosineSimilarityFloat64(queryVec, embedding)
		if err != nil {
			continue // Skip on error
		}

		// Apply threshold filter
		if pred.Threshold > 0 && similarity < pred.Threshold {
			continue
		}

		scoredRows = append(scoredRows, scoredRow{
			row:        row,
			similarity: similarity,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(scoredRows, func(i, j int) bool {
		return scoredRows[i].similarity > scoredRows[j].similarity
	})

	// Apply topK limit
	if pred.TopK > 0 && len(scoredRows) > pred.TopK {
		scoredRows = scoredRows[:pred.TopK]
	}

	// Build filtered result
	filteredRows := make([]Row, len(scoredRows))
	for i, s := range scoredRows {
		filteredRows[i] = s.row
		// Optionally add similarity score to row
		filteredRows[i]["_similarity"] = s.similarity
	}

	return &QueryResult{
		Rows:          filteredRows,
		RowCount:      len(filteredRows),
		Columns:       results.Columns,
		ExecutionTime: results.ExecutionTime,
	}, nil
}
