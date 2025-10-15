package udml

import (
	"testing"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSchemaRegistry validates schema registry initialization
func TestNewSchemaRegistry(t *testing.T) {
	registry := NewSchemaRegistry()

	assert.NotNil(t, registry, "Registry should not be nil")
	assert.NotNil(t, registry.schemas, "Schemas map should be initialized")
	assert.NotNil(t, registry.universalSchema, "Universal schema should be initialized")

	// Verify schemas were registered
	assert.Greater(t, len(registry.schemas), 0, "Should have registered element types")
}

// TestUniversalSchema validates the universal schema structure
func TestUniversalSchema(t *testing.T) {
	registry := NewSchemaRegistry()
	schema := registry.getDefaultSchema()

	require.NotNil(t, schema, "Universal schema should not be nil")

	// Expected field names
	expectedFields := []string{
		// Core fields
		"element_id", "doc_id", "source_name", "element_type", "element_category",
		"content", "content_preview", "content_hash", "parent_id",
		"element_order", "document_position",

		// UDML Phase 1: 6 promoted fields
		"page_number", "section_level", "row_index", "column_index",
		"temporal_type", "tag_name",

		// JSON overflow
		"content_location", "metadata", "temporal_metadata",
	}

	assert.Equal(t, len(expectedFields), schema.NumFields(), "Schema should have correct number of fields")

	// Verify all expected fields exist
	for _, expectedField := range expectedFields {
		fields, found := schema.FieldsByName(expectedField)
		assert.True(t, found, "Field %s should exist in schema", expectedField)
		assert.NotEmpty(t, fields, "Field %s should not be empty", expectedField)
	}
}

// TestPromotedFieldsNullability validates that promoted fields are nullable
func TestPromotedFieldsNullability(t *testing.T) {
	registry := NewSchemaRegistry()
	schema := registry.getDefaultSchema()

	promotedFields := []string{
		"page_number", "section_level", "row_index", "column_index",
		"temporal_type", "tag_name",
	}

	for _, fieldName := range promotedFields {
		fields, found := schema.FieldsByName(fieldName)
		require.True(t, found, "Promoted field %s should exist", fieldName)
		require.NotEmpty(t, fields, "Promoted field %s should not be empty", fieldName)
		assert.True(t, fields[0].Nullable, "Promoted field %s should be nullable", fieldName)
	}
}

// TestCoreFieldsRequired validates that core fields are not nullable
func TestCoreFieldsRequired(t *testing.T) {
	registry := NewSchemaRegistry()
	schema := registry.getDefaultSchema()

	requiredFields := []string{
		"element_id", "doc_id", "source_name", "element_type", "element_category",
	}

	for _, fieldName := range requiredFields {
		fields, found := schema.FieldsByName(fieldName)
		require.True(t, found, "Required field %s should exist", fieldName)
		require.NotEmpty(t, fields, "Required field %s should not be empty", fieldName)
		assert.False(t, fields[0].Nullable, "Required field %s should not be nullable", fieldName)
	}
}

// TestPromotedFieldTypes validates data types of promoted fields
func TestPromotedFieldTypes(t *testing.T) {
	registry := NewSchemaRegistry()
	schema := registry.getDefaultSchema()

	// Integer promoted fields
	intFields := []string{"page_number", "section_level", "row_index", "column_index"}
	for _, fieldName := range intFields {
		fields, found := schema.FieldsByName(fieldName)
		require.True(t, found, "Field %s should exist", fieldName)
		require.NotEmpty(t, fields, "Field %s should not be empty", fieldName)
		assert.Equal(t, arrow.PrimitiveTypes.Int64, fields[0].Type, "Field %s should be Int64 type", fieldName)
	}

	// String promoted fields
	stringFields := []string{"temporal_type", "tag_name"}
	for _, fieldName := range stringFields {
		fields, found := schema.FieldsByName(fieldName)
		require.True(t, found, "Field %s should exist", fieldName)
		require.NotEmpty(t, fields, "Field %s should not be empty", fieldName)
		assert.Equal(t, arrow.BinaryTypes.String, fields[0].Type, "Field %s should be String type", fieldName)
	}
}

// TestGetSchema validates schema retrieval for different element types
func TestGetSchema(t *testing.T) {
	registry := NewSchemaRegistry()

	elementTypes := []string{
		"paragraph", "header", "table", "table_cell",
		"list", "code_block", "image",
		"div", "span", "json_field",
	}

	for _, elemType := range elementTypes {
		schema := registry.GetSchema(elemType)
		assert.NotNil(t, schema, "Schema for %s should not be nil", elemType)

		// Verify it's the universal schema
		assert.Equal(t, registry.universalSchema, schema,
			"Schema for %s should be the universal schema", elemType)
	}
}

// TestGetSchemaUnknownType validates fallback for unknown element types
func TestGetSchemaUnknownType(t *testing.T) {
	registry := NewSchemaRegistry()

	// Request schema for an unknown element type
	schema := registry.GetSchema("unknown_element_type")

	assert.NotNil(t, schema, "Should return default schema for unknown type")
	assert.Equal(t, registry.universalSchema, schema,
		"Unknown type should fall back to universal schema")
}

// TestAllElementTypesShareSchema validates that all registered types use the same schema
func TestAllElementTypesShareSchema(t *testing.T) {
	registry := NewSchemaRegistry()

	registeredTypes := registry.GetRegisteredTypes()
	require.Greater(t, len(registeredTypes), 0, "Should have registered types")

	// Get schema for first type
	firstSchema := registry.GetSchema(registeredTypes[0])

	// Verify all types use the same schema instance
	for _, elemType := range registeredTypes {
		schema := registry.GetSchema(elemType)
		assert.Equal(t, firstSchema, schema,
			"All element types should share the same schema instance")
	}
}

// TestHasSchema validates schema existence checks
func TestHasSchema(t *testing.T) {
	registry := NewSchemaRegistry()

	// Test known element types
	knownTypes := []string{"paragraph", "table", "header"}
	for _, elemType := range knownTypes {
		assert.True(t, registry.HasSchema(elemType),
			"Should have schema for %s", elemType)
	}

	// Test unknown element type
	assert.False(t, registry.HasSchema("definitely_not_a_real_element_type"),
		"Should not have schema for unknown type")
}

// TestGetRegisteredTypes validates type enumeration
func TestGetRegisteredTypes(t *testing.T) {
	registry := NewSchemaRegistry()

	types := registry.GetRegisteredTypes()

	assert.Equal(t, 45, len(types), "Should have registered exactly 45 element types (aligned with parser usage)")

	// Verify some expected types are present
	expectedTypes := []string{
		"root", "body", "paragraph", "header",
		"table", "table_row", "table_cell",
		"list", "list_item", "code_block",
		// Verify some of the newly added types
		"slide", "json_object", "xml_element", "front_matter", "blockquote",
	}

	registeredTypesMap := make(map[string]bool)
	for _, t := range types {
		registeredTypesMap[t] = true
	}

	for _, expectedType := range expectedTypes {
		assert.True(t, registeredTypesMap[expectedType],
			"Should have registered type: %s", expectedType)
	}
}

// TestRegisterCustomSchema validates custom schema registration
func TestRegisterCustomSchema(t *testing.T) {
	registry := NewSchemaRegistry()

	// Create a custom schema
	customSchema := arrow.NewSchema([]arrow.Field{
		{Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "custom_field", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
	}, nil)

	// Register custom schema for a new element type
	customType := "custom_element_type"
	registry.RegisterCustomSchema(customType, customSchema)

	// Verify custom schema was registered
	assert.True(t, registry.HasSchema(customType),
		"Should have schema for custom type")

	retrievedSchema := registry.GetSchema(customType)
	assert.Equal(t, customSchema, retrievedSchema,
		"Should retrieve the custom schema")

	// Verify custom schema is different from universal schema
	assert.NotEqual(t, registry.universalSchema, retrievedSchema,
		"Custom schema should be different from universal schema")
}

// TestSchemaConsistency validates schema is consistent across all element types
func TestSchemaConsistency(t *testing.T) {
	registry := NewSchemaRegistry()

	// Get two different element types
	paragraphSchema := registry.GetSchema("paragraph")
	tableSchema := registry.GetSchema("table")

	// They should be identical (same instance)
	assert.Equal(t, paragraphSchema.NumFields(), tableSchema.NumFields(),
		"Schemas should have same number of fields")

	// Compare field by field
	for i := 0; i < paragraphSchema.NumFields(); i++ {
		paragraphField := paragraphSchema.Field(i)
		tableField := tableSchema.Field(i)

		assert.Equal(t, paragraphField.Name, tableField.Name,
			"Field %d should have same name", i)
		assert.Equal(t, paragraphField.Type, tableField.Type,
			"Field %d should have same type", i)
		assert.Equal(t, paragraphField.Nullable, tableField.Nullable,
			"Field %d should have same nullability", i)
	}
}

// TestJSONOverflowFields validates JSON overflow field types
func TestJSONOverflowFields(t *testing.T) {
	registry := NewSchemaRegistry()
	schema := registry.getDefaultSchema()

	jsonFields := []string{"content_location", "metadata", "temporal_metadata"}

	for _, fieldName := range jsonFields {
		fields, found := schema.FieldsByName(fieldName)
		require.True(t, found, "JSON overflow field %s should exist", fieldName)
		require.NotEmpty(t, fields, "JSON overflow field %s should not be empty", fieldName)
		assert.Equal(t, arrow.BinaryTypes.String, fields[0].Type,
			"JSON overflow field %s should be String type (for JSON storage)", fieldName)
		assert.True(t, fields[0].Nullable,
			"JSON overflow field %s should be nullable", fieldName)
	}
}

// TestSchemaFieldCount validates expected field count
func TestSchemaFieldCount(t *testing.T) {
	registry := NewSchemaRegistry()
	schema := registry.getDefaultSchema()

	expectedFieldCount := 20 // 11 core + 6 promoted + 3 JSON overflow
	assert.Equal(t, expectedFieldCount, schema.NumFields(),
		"Schema should have exactly %d fields", expectedFieldCount)
}

// BenchmarkGetSchema benchmarks schema retrieval performance
func BenchmarkGetSchema(b *testing.B) {
	registry := NewSchemaRegistry()
	elementTypes := []string{"paragraph", "table", "header", "list", "code_block"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elemType := elementTypes[i%len(elementTypes)]
		_ = registry.GetSchema(elemType)
	}
}

// BenchmarkNewSchemaRegistry benchmarks registry initialization
func BenchmarkNewSchemaRegistry(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewSchemaRegistry()
	}
}
