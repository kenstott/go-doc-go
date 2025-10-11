package query

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSONPath represents a parsed JSONPath expression
type JSONPath struct {
	Expression string
	Segments   []PathSegment
}

// PathSegment represents one segment of a JSONPath
type PathSegment struct {
	Type       SegmentType
	Key        string   // For object access: $.field
	Index      int      // For array access: $[0]
	IsWildcard bool     // For wildcard: $[*] or $.*.field
	Filter     *Predicate // For filter expressions: $[?(@.price > 10)]
}

// SegmentType defines the type of path segment
type SegmentType string

const (
	SegmentRoot      SegmentType = "root"      // $
	SegmentKey       SegmentType = "key"       // .field
	SegmentIndex     SegmentType = "index"     // [0]
	SegmentWildcard  SegmentType = "wildcard"  // [*] or .*
	SegmentRecursive SegmentType = "recursive" // ..field
	SegmentFilter    SegmentType = "filter"    // [?(@.field > 10)]
)

// ParseJSONPath parses a JSONPath expression into segments
func ParseJSONPath(expr string) (*JSONPath, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty JSONPath expression")
	}

	if !strings.HasPrefix(expr, "$") {
		return nil, fmt.Errorf("JSONPath must start with $, got: %s", expr)
	}

	jp := &JSONPath{
		Expression: expr,
		Segments:   []PathSegment{{Type: SegmentRoot}},
	}

	// Remove leading $
	remaining := expr[1:]

	for len(remaining) > 0 {
		segment, consumed, err := parseSegment(remaining)
		if err != nil {
			return nil, fmt.Errorf("failed to parse segment at '%s': %w", remaining, err)
		}

		jp.Segments = append(jp.Segments, segment)
		remaining = remaining[consumed:]
	}

	return jp, nil
}

// parseSegment parses one segment from the remaining string
func parseSegment(s string) (PathSegment, int, error) {
	if len(s) == 0 {
		return PathSegment{}, 0, fmt.Errorf("empty segment")
	}

	switch s[0] {
	case '.':
		// Dot notation: .field or ..field or .*
		if len(s) > 1 && s[1] == '.' {
			// Recursive descent: ..field
			return parseRecursiveDescent(s)
		}
		if len(s) > 1 && s[1] == '*' {
			// Wildcard: .*
			return PathSegment{Type: SegmentWildcard, IsWildcard: true}, 2, nil
		}
		// Regular field: .field
		return parseKey(s)

	case '[':
		// Bracket notation: [0], [*], ['field'], or [?(...)]
		return parseBracket(s)

	default:
		return PathSegment{}, 0, fmt.Errorf("unexpected character: %c", s[0])
	}
}

// parseKey parses a key segment: .field
func parseKey(s string) (PathSegment, int, error) {
	if len(s) < 2 || s[0] != '.' {
		return PathSegment{}, 0, fmt.Errorf("expected ., got: %s", s)
	}

	// Find end of key (next . or [ or end)
	i := 1
	for i < len(s) && s[i] != '.' && s[i] != '[' {
		i++
	}

	key := s[1:i]
	if key == "" {
		return PathSegment{}, 0, fmt.Errorf("empty key")
	}

	return PathSegment{
		Type: SegmentKey,
		Key:  key,
	}, i, nil
}

// parseBracket parses bracket notation: [0], [*], ['field']
func parseBracket(s string) (PathSegment, int, error) {
	if s[0] != '[' {
		return PathSegment{}, 0, fmt.Errorf("expected [, got: %c", s[0])
	}

	// Find closing bracket
	closeIdx := strings.Index(s, "]")
	if closeIdx == -1 {
		return PathSegment{}, 0, fmt.Errorf("unclosed bracket")
	}

	content := s[1:closeIdx]

	// Check for wildcard: [*]
	if content == "*" {
		return PathSegment{
			Type:       SegmentWildcard,
			IsWildcard: true,
		}, closeIdx + 1, nil
	}

	// Check for quoted string: ['field'] or ["field"]
	if (strings.HasPrefix(content, "'") && strings.HasSuffix(content, "'")) ||
		(strings.HasPrefix(content, "\"") && strings.HasSuffix(content, "\"")) {
		key := content[1 : len(content)-1]
		return PathSegment{
			Type: SegmentKey,
			Key:  key,
		}, closeIdx + 1, nil
	}

	// Check for filter: [?(...)]
	if strings.HasPrefix(content, "?") {
		return PathSegment{
			Type: SegmentFilter,
			// Filter parsing is simplified - full implementation would parse the predicate
		}, closeIdx + 1, nil
	}

	// Try to parse as array index: [0]
	index, err := strconv.Atoi(content)
	if err != nil {
		return PathSegment{}, 0, fmt.Errorf("invalid bracket content: %s", content)
	}

	return PathSegment{
		Type:  SegmentIndex,
		Index: index,
	}, closeIdx + 1, nil
}

// parseRecursiveDescent parses recursive descent: ..field
func parseRecursiveDescent(s string) (PathSegment, int, error) {
	if len(s) < 3 || s[0] != '.' || s[1] != '.' {
		return PathSegment{}, 0, fmt.Errorf("expected .., got: %s", s[:2])
	}

	// Find end of key
	i := 2
	for i < len(s) && s[i] != '.' && s[i] != '[' {
		i++
	}

	key := s[2:i]
	if key == "" {
		return PathSegment{}, 0, fmt.Errorf("empty recursive key")
	}

	return PathSegment{
		Type: SegmentRecursive,
		Key:  key,
	}, i, nil
}

// Evaluate evaluates a JSONPath against a JSON value
func (jp *JSONPath) Evaluate(data interface{}) ([]interface{}, error) {
	results := []interface{}{data}

	// Skip root segment
	for i := 1; i < len(jp.Segments); i++ {
		segment := jp.Segments[i]
		var nextResults []interface{}

		for _, current := range results {
			matches, err := evaluateSegment(segment, current)
			if err != nil {
				return nil, err
			}
			nextResults = append(nextResults, matches...)
		}

		results = nextResults
	}

	return results, nil
}

// evaluateSegment evaluates one segment against a value
func evaluateSegment(segment PathSegment, value interface{}) ([]interface{}, error) {
	switch segment.Type {
	case SegmentKey:
		return evaluateKey(segment.Key, value)

	case SegmentIndex:
		return evaluateIndex(segment.Index, value)

	case SegmentWildcard:
		return evaluateWildcard(value)

	case SegmentRecursive:
		return evaluateRecursive(segment.Key, value)

	case SegmentFilter:
		// Simplified: would need full filter evaluation
		return []interface{}{}, nil

	default:
		return nil, fmt.Errorf("unsupported segment type: %s", segment.Type)
	}
}

// evaluateKey evaluates a key segment
func evaluateKey(key string, value interface{}) ([]interface{}, error) {
	switch v := value.(type) {
	case map[string]interface{}:
		if val, ok := v[key]; ok {
			return []interface{}{val}, nil
		}
		return []interface{}{}, nil

	case string:
		// Try to parse as JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(v), &jsonData); err != nil {
			return []interface{}{}, nil
		}
		if val, ok := jsonData[key]; ok {
			return []interface{}{val}, nil
		}
		return []interface{}{}, nil

	default:
		return []interface{}{}, nil
	}
}

// evaluateIndex evaluates an array index segment
func evaluateIndex(index int, value interface{}) ([]interface{}, error) {
	switch v := value.(type) {
	case []interface{}:
		if index < 0 || index >= len(v) {
			return []interface{}{}, nil
		}
		return []interface{}{v[index]}, nil

	case string:
		// Try to parse as JSON array
		var jsonData []interface{}
		if err := json.Unmarshal([]byte(v), &jsonData); err != nil {
			return []interface{}{}, nil
		}
		if index < 0 || index >= len(jsonData) {
			return []interface{}{}, nil
		}
		return []interface{}{jsonData[index]}, nil

	default:
		return []interface{}{}, nil
	}
}

// evaluateWildcard evaluates a wildcard segment
func evaluateWildcard(value interface{}) ([]interface{}, error) {
	switch v := value.(type) {
	case map[string]interface{}:
		results := make([]interface{}, 0, len(v))
		for _, val := range v {
			results = append(results, val)
		}
		return results, nil

	case []interface{}:
		return v, nil

	case string:
		// Try to parse as JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(v), &jsonData); err != nil {
			return []interface{}{}, nil
		}
		return evaluateWildcard(jsonData)

	default:
		return []interface{}{}, nil
	}
}

// evaluateRecursive evaluates a recursive descent segment
func evaluateRecursive(key string, value interface{}) ([]interface{}, error) {
	var results []interface{}

	var recurse func(interface{})
	recurse = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			if target, ok := val[key]; ok {
				results = append(results, target)
			}
			for _, child := range val {
				recurse(child)
			}

		case []interface{}:
			for _, child := range val {
				recurse(child)
			}
		}
	}

	recurse(value)
	return results, nil
}

// String returns the string representation of a JSONPath
func (jp *JSONPath) String() string {
	return jp.Expression
}

// ValidateJSONPath validates a JSONPath expression
func ValidateJSONPath(expr string) error {
	_, err := ParseJSONPath(expr)
	return err
}

// ExtractFromJSON extracts a value from JSON using JSONPath
func ExtractFromJSON(jsonStr string, pathExpr string) ([]interface{}, error) {
	// Parse JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Parse JSONPath
	jp, err := ParseJSONPath(pathExpr)
	if err != nil {
		return nil, err
	}

	// Evaluate
	return jp.Evaluate(data)
}

// CompareJSONPathValue compares a JSONPath result against a value
func CompareJSONPathValue(jsonStr string, pathExpr string, operator Operator, compareValue interface{}) (bool, error) {
	results, err := ExtractFromJSON(jsonStr, pathExpr)
	if err != nil {
		return false, err
	}

	if len(results) == 0 {
		// No match found - handle based on operator
		switch operator {
		case OpIsNull:
			return true, nil
		case OpIsNotNull:
			return false, nil
		default:
			return false, nil
		}
	}

	// Check if any result matches the comparison
	for _, result := range results {
		if match, err := compareValues(result, operator, compareValue); err != nil {
			return false, err
		} else if match {
			return true, nil
		}
	}

	return false, nil
}

// compareValues compares two values using an operator
func compareValues(a interface{}, op Operator, b interface{}) (bool, error) {
	switch op {
	case OpEqual:
		return a == b, nil

	case OpNotEqual:
		return a != b, nil

	case OpGreaterThan:
		return compareNumeric(a, b, func(x, y float64) bool { return x > y })

	case OpGreaterThanOrEqual:
		return compareNumeric(a, b, func(x, y float64) bool { return x >= y })

	case OpLessThan:
		return compareNumeric(a, b, func(x, y float64) bool { return x < y })

	case OpLessThanOrEqual:
		return compareNumeric(a, b, func(x, y float64) bool { return x <= y })

	case OpIsNull:
		return a == nil, nil

	case OpIsNotNull:
		return a != nil, nil

	default:
		return false, fmt.Errorf("unsupported operator for JSONPath comparison: %s", op)
	}
}

// compareNumeric compares numeric values
func compareNumeric(a, b interface{}, cmp func(float64, float64) bool) (bool, error) {
	aFloat, err := toFloat64(a)
	if err != nil {
		return false, err
	}

	bFloat, err := toFloat64(b)
	if err != nil {
		return false, err
	}

	return cmp(aFloat, bFloat), nil
}

// toFloat64 converts a value to float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}
