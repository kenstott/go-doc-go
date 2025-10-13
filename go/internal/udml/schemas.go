package udml

import (
	"github.com/apache/arrow/go/v18/arrow"
)

// SchemaRegistry manages type-specific Parquet schemas for UDML elements
// All element types share the same universal schema with 6 query-optimized promoted fields
type SchemaRegistry struct {
	schemas        map[string]*arrow.Schema
	universalSchema *arrow.Schema
}

// NewSchemaRegistry creates a new schema registry with all element types registered
func NewSchemaRegistry() *SchemaRegistry {
	r := &SchemaRegistry{
		schemas: make(map[string]*arrow.Schema),
	}
	r.registerDefaultSchemas()
	return r
}

// GetSchema returns the schema for a given element type
// If the element type is not found, returns the default universal schema
func (r *SchemaRegistry) GetSchema(elementType string) *arrow.Schema {
	if schema, exists := r.schemas[elementType]; exists {
		return schema
	}
	return r.getDefaultSchema()
}

// getDefaultSchema returns the universal schema used by all element types
func (r *SchemaRegistry) getDefaultSchema() *arrow.Schema {
	return r.universalSchema
}

// registerDefaultSchemas creates and registers the universal schema for all element types
// All element types use the SAME schema with 6 query-optimized nullable fields
// This enables cross-type queries while maintaining Hive partitioning benefits
func (r *SchemaRegistry) registerDefaultSchemas() {
	// Universal schema with 6 promoted fields
	// This schema is shared across ALL element types (paragraph, table, heading, etc.)
	r.universalSchema = arrow.NewSchema([]arrow.Field{
		// Core universal fields (required)
		{Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "element_type", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "element_category", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "content", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "content_preview", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "content_hash", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "parent_id", Type: arrow.BinaryTypes.String, Nullable: true},
		{Name: "element_order", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{Name: "document_position", Type: arrow.PrimitiveTypes.Float64, Nullable: true},

		// UDML Phase 1: 6 Query-Optimized Promoted Fields (all nullable)
		// These fields enable 60-1000x faster queries across all backends
		{Name: "page_number", Type: arrow.PrimitiveTypes.Int64, Nullable: true},      // ~30% populated (PDF/DOCX/PPTX)
		{Name: "section_level", Type: arrow.PrimitiveTypes.Int64, Nullable: true},    // ~15% populated (headings)
		{Name: "row_index", Type: arrow.PrimitiveTypes.Int64, Nullable: true},        // ~20% populated (tables)
		{Name: "column_index", Type: arrow.PrimitiveTypes.Int64, Nullable: true},     // ~20% populated (tables)
		{Name: "temporal_type", Type: arrow.BinaryTypes.String, Nullable: true},      // ~5-10% populated (dates)
		{Name: "tag_name", Type: arrow.BinaryTypes.String, Nullable: true},           // ~25% populated (HTML/XML)

		// JSON overflow (format-specific attributes - stored as JSON strings)
		{Name: "content_location", Type: arrow.BinaryTypes.String, Nullable: true},   // JSON
		{Name: "metadata", Type: arrow.BinaryTypes.String, Nullable: true},           // JSON
		{Name: "temporal_metadata", Type: arrow.BinaryTypes.String, Nullable: true},  // JSON
	}, nil)

	// ALL element types use the same universal schema
	// This enables cross-type queries while maintaining Hive partitioning benefits
	// Define all supported element types (45 types - aligned with actual parser usage)
	elementTypes := []string{
		// Document structure
		"root", "body", "section",

		// Text elements
		"paragraph", "text", "header",

		// List elements
		"list", "list_item",

		// Table elements
		"table", "data_table", "data_tables",
		"table_header", "table_header_row", "table_header_cell",
		"table_row", "table_cell",

		// Code elements
		"code_block",

		// Media elements
		"image", "chart",

		// HTML/XML specific
		"div", "span", "article", "blockquote",

		// Structured data (JSON)
		"json_field", "json_object", "json_array", "json_item",

		// XML specific
		"xml_element",

		// Office document specific - Word
		"headers", "footers", "comments",

		// Office document specific - Excel
		"sheet", "workbook", "comment",
		"merged_cell", "merged_cells",

		// Office document specific - PowerPoint
		"slide",
		"comments_container", "slide_notes", "shape_group",

		// PDF specific
		"page",

		// Markdown specific
		"front_matter",

		// Text parser specific
		"line", "range", "substring",
	}

	// Register the universal schema for all element types
	for _, elemType := range elementTypes {
		r.schemas[elemType] = r.universalSchema
	}
}

// RegisterCustomSchema allows registration of a custom schema for a specific element type
// This is primarily for future extensibility
func (r *SchemaRegistry) RegisterCustomSchema(elementType string, schema *arrow.Schema) {
	r.schemas[elementType] = schema
}

// GetRegisteredTypes returns all registered element types
func (r *SchemaRegistry) GetRegisteredTypes() []string {
	types := make([]string, 0, len(r.schemas))
	for t := range r.schemas {
		types = append(types, t)
	}
	return types
}

// HasSchema returns true if a schema is registered for the given element type
func (r *SchemaRegistry) HasSchema(elementType string) bool {
	_, exists := r.schemas[elementType]
	return exists
}
