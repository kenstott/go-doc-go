package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
)

// Document represents a reconstructed hierarchical document from UDML elements
type Document struct {
	DocID      string                 `json:"doc_id"`
	SourceName string                 `json:"source_name,omitempty"`
	Version    string                 `json:"version,omitempty"`
	Elements   []*Element             `json:"elements"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Element represents a single UDML element with optional nested children
type Element struct {
	ElementID        string                 `json:"element_id"`
	ElementType      string                 `json:"element_type"`
	ElementCategory  string                 `json:"element_category,omitempty"`
	Content          string                 `json:"content,omitempty"`
	ContentPreview   string                 `json:"content_preview,omitempty"`
	ContentHash      string                 `json:"content_hash,omitempty"`
	ParentID         string                 `json:"parent_id,omitempty"`
	ElementOrder     float64                `json:"element_order"`
	DocumentPosition float64                `json:"document_position,omitempty"`
	PageNumber       *int                   `json:"page_number,omitempty"`
	SectionLevel     *int                   `json:"section_level,omitempty"`
	RowIndex         *int                   `json:"row_index,omitempty"`
	ColumnIndex      *int                   `json:"column_index,omitempty"`
	TemporalType     string                 `json:"temporal_type,omitempty"`
	TagName          string                 `json:"tag_name,omitempty"`
	ContentLocation  map[string]interface{} `json:"content_location,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	TemporalMetadata map[string]interface{} `json:"temporal_metadata,omitempty"`
	Children         []*Element             `json:"children,omitempty"`
}

// DocumentBuilder constructs hierarchical documents from flat UDML elements
type DocumentBuilder struct {
	backend query.QueryBackend
	options BuildOptions
}

// BuildOptions configures document building behavior
type BuildOptions struct {
	// IncludeContent controls whether full content is included (vs just previews)
	IncludeContent bool

	// MaxDepth limits nesting depth (0 = unlimited)
	MaxDepth int

	// SortChildren sorts children by element_order
	SortChildren bool

	// Filter allows filtering elements during build
	Filter *query.Predicate

	// IncludeMetadata includes JSON overflow fields
	IncludeMetadata bool
}

// DefaultBuildOptions returns sensible defaults
func DefaultBuildOptions() BuildOptions {
	return BuildOptions{
		IncludeContent:  true,
		MaxDepth:        0,
		SortChildren:    true,
		IncludeMetadata: true,
	}
}

// NewDocumentBuilder creates a new document builder using a query backend
func NewDocumentBuilder(backend query.QueryBackend, options BuildOptions) *DocumentBuilder {
	return &DocumentBuilder{
		backend: backend,
		options: options,
	}
}

// BuildDocument reconstructs a hierarchical document from flat UDML elements
func (b *DocumentBuilder) BuildDocument(ctx context.Context, docID string) (*Document, error) {
	// Query all elements for this document
	expr := query.NewExpression("elements").
		SelectAll().
		Filter(query.NewComparison("doc_id", query.OpEqual, docID))

	// Apply optional filter
	if b.options.Filter != nil {
		expr = expr.Filter(query.NewAnd(expr.Where, b.options.Filter))
	}

	// Order by document position for proper reconstruction
	expr = expr.Order("document_position", false)

	// Translate and execute query
	queryObj, err := b.backend.Translate(expr, query.TranslateOptions{
		EnablePartitions: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to translate query: %w", err)
	}

	result, err := b.backend.Execute(ctx, queryObj)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if result.RowCount == 0 {
		return nil, fmt.Errorf("document not found: %s", docID)
	}

	// Convert query.Row to []map[string]interface{} for processing
	rows := make([]map[string]interface{}, len(result.Rows))
	for i, row := range result.Rows {
		rows[i] = map[string]interface{}(row)
	}

	// Convert result rows to elements
	elements, err := b.rowsToElements(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rows: %w", err)
	}

	// Build hierarchy
	tree := b.buildHierarchy(elements)

	// Create document
	doc := &Document{
		DocID:    docID,
		Elements: tree,
	}

	// Extract metadata from first element
	if len(rows) > 0 {
		doc.SourceName = getStringField(rows[0], "source_name")
		doc.Version = getStringField(rows[0], "version")
	}

	return doc, nil
}

// BuildDocumentFromElements builds a document from pre-fetched element data
func (b *DocumentBuilder) BuildDocumentFromElements(docID string, rows []map[string]interface{}) (*Document, error) {
	// Convert rows to elements
	elements, err := b.rowsToElements(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rows: %w", err)
	}

	// Build hierarchy
	tree := b.buildHierarchy(elements)

	// Create document
	doc := &Document{
		DocID:    docID,
		Elements: tree,
	}

	// Extract metadata from first element
	if len(rows) > 0 {
		doc.SourceName = getStringField(rows[0], "source_name")
		doc.Version = getStringField(rows[0], "version")
	}

	return doc, nil
}

// rowsToElements converts query result rows to Element structs
func (b *DocumentBuilder) rowsToElements(rows []map[string]interface{}) ([]*Element, error) {
	elements := make([]*Element, 0, len(rows))

	for _, row := range rows {
		elem := &Element{
			ElementID:        getStringField(row, "element_id"),
			ElementType:      getStringField(row, "element_type"),
			ElementCategory:  getStringField(row, "element_category"),
			ParentID:         getStringField(row, "parent_id"),
			ContentHash:      getStringField(row, "content_hash"),
			ElementOrder:     getFloatField(row, "element_order"),
			DocumentPosition: getFloatField(row, "document_position"),
			TemporalType:     getStringField(row, "temporal_type"),
			TagName:          getStringField(row, "tag_name"),
		}

		// Content fields
		if b.options.IncludeContent {
			elem.Content = getStringField(row, "content")
		}
		elem.ContentPreview = getStringField(row, "content_preview")

		// Optional integer fields
		if pageNum := getIntField(row, "page_number"); pageNum != nil {
			elem.PageNumber = pageNum
		}
		if secLevel := getIntField(row, "section_level"); secLevel != nil {
			elem.SectionLevel = secLevel
		}
		if rowIdx := getIntField(row, "row_index"); rowIdx != nil {
			elem.RowIndex = rowIdx
		}
		if colIdx := getIntField(row, "column_index"); colIdx != nil {
			elem.ColumnIndex = colIdx
		}

		// JSON overflow fields
		if b.options.IncludeMetadata {
			if metadata := getJSONField(row, "metadata"); metadata != nil {
				elem.Metadata = metadata
			}
			if contentLoc := getJSONField(row, "content_location"); contentLoc != nil {
				elem.ContentLocation = contentLoc
			}
			if tempMeta := getJSONField(row, "temporal_metadata"); tempMeta != nil {
				elem.TemporalMetadata = tempMeta
			}
		}

		elements = append(elements, elem)
	}

	return elements, nil
}

// buildHierarchy reconstructs parent-child relationships from flat elements
func (b *DocumentBuilder) buildHierarchy(elements []*Element) []*Element {
	// Create element map for fast lookup
	elementMap := make(map[string]*Element)
	for _, elem := range elements {
		elementMap[elem.ElementID] = elem
	}

	// Build tree structure
	roots := make([]*Element, 0)

	for _, elem := range elements {
		if elem.ParentID == "" {
			// Root element
			roots = append(roots, elem)
		} else {
			// Child element - attach to parent
			if parent, exists := elementMap[elem.ParentID]; exists {
				if parent.Children == nil {
					parent.Children = make([]*Element, 0)
				}
				parent.Children = append(parent.Children, elem)
			} else {
				// Parent not found - treat as root
				roots = append(roots, elem)
			}
		}
	}

	// Sort children if requested
	if b.options.SortChildren {
		b.sortChildren(roots)
	}

	// Apply max depth if specified
	if b.options.MaxDepth > 0 {
		roots = b.limitDepth(roots, 1)
	}

	return roots
}

// sortChildren recursively sorts children by element_order
func (b *DocumentBuilder) sortChildren(elements []*Element) {
	for _, elem := range elements {
		if len(elem.Children) > 0 {
			sort.Slice(elem.Children, func(i, j int) bool {
				return elem.Children[i].ElementOrder < elem.Children[j].ElementOrder
			})
			b.sortChildren(elem.Children)
		}
	}
}

// limitDepth limits tree depth to maxDepth
func (b *DocumentBuilder) limitDepth(elements []*Element, currentDepth int) []*Element {
	if currentDepth >= b.options.MaxDepth {
		// Remove children at max depth
		for _, elem := range elements {
			elem.Children = nil
		}
		return elements
	}

	for _, elem := range elements {
		if len(elem.Children) > 0 {
			elem.Children = b.limitDepth(elem.Children, currentDepth+1)
		}
	}

	return elements
}

// ToJSON serializes the document to JSON bytes
func (d *Document) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// ToJSONCompact serializes the document to compact JSON
func (d *Document) ToJSONCompact() ([]byte, error) {
	return json.Marshal(d)
}

// GetElementCount returns total number of elements (including nested)
func (d *Document) GetElementCount() int {
	count := 0
	for _, elem := range d.Elements {
		count += countElements(elem)
	}
	return count
}

// countElements recursively counts elements
func countElements(elem *Element) int {
	count := 1
	for _, child := range elem.Children {
		count += countElements(child)
	}
	return count
}

// GetMaxDepth returns maximum nesting depth
func (d *Document) GetMaxDepth() int {
	maxDepth := 0
	for _, elem := range d.Elements {
		depth := getDepth(elem, 1)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

// getDepth recursively calculates depth
func getDepth(elem *Element, currentDepth int) int {
	maxChildDepth := currentDepth
	for _, child := range elem.Children {
		childDepth := getDepth(child, currentDepth+1)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}
	return maxChildDepth
}

// Helper functions for field extraction

func getStringField(row map[string]interface{}, key string) string {
	if val, ok := row[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloatField(row map[string]interface{}, key string) float64 {
	if val, ok := row[key]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

func getIntField(row map[string]interface{}, key string) *int {
	if val, ok := row[key]; ok && val != nil {
		switch v := val.(type) {
		case int:
			return &v
		case int64:
			i := int(v)
			return &i
		case float64:
			i := int(v)
			return &i
		}
	}
	return nil
}

func getJSONField(row map[string]interface{}, key string) map[string]interface{} {
	if val, ok := row[key]; ok && val != nil {
		// Already a map
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}

		// Try to parse as JSON string
		if str, ok := val.(string); ok && str != "" {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(str), &result); err == nil {
				return result
			}
		}
	}
	return nil
}
