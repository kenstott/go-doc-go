package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/apache/arrow/go/v18/parquet"
	"github.com/apache/arrow/go/v18/parquet/file"
	"github.com/apache/arrow/go/v18/parquet/pqarrow"
)

// ParquetParser handles parsing of Parquet files containing tabular data
type ParquetParser struct {
	MaxContentPreview int
	TextColumn        string   // Column to use for main text content
	GroupByColumn     string   // Optional column for grouping rows
	MetadataColumns   []string // Columns to extract as document metadata
}

// NewParquetParser creates a new Parquet parser with default settings
func NewParquetParser() *ParquetParser {
	return &ParquetParser{
		MaxContentPreview: 100,
		TextColumn:        "", // Auto-detect if empty
		GroupByColumn:     "",
		MetadataColumns:   []string{},
	}
}

// ParquetParseRequest represents input for Parquet parsing
type ParquetParseRequest struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"` // File path for Parquet files
	Metadata map[string]interface{} `json:"metadata"`
}

// ParquetParseResponse represents the output of Parquet parsing
type ParquetParseResponse struct {
	Document      map[string]interface{} `json:"document"`
	Elements      []map[string]interface{} `json:"elements"`
	Relationships []map[string]interface{} `json:"relationships"`
	Links         []map[string]interface{} `json:"links,omitempty"`
}

// Parse parses a Parquet file and extracts structured data
func (p *ParquetParser) Parse(request ParquetParseRequest) (*ParquetParseResponse, error) {
	// Get file info
	fileInfo, err := os.Stat(request.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %v", err)
	}

	// Create memory allocator
	mem := memory.NewGoAllocator()

	// Open parquet file reader - it expects a file path
	pf, err := file.OpenParquetFile(request.Content, true, file.WithReadProps(parquet.NewReaderProperties(mem)))
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %v", err)
	}
	defer pf.Close()

	// Create arrow file reader
	arrowReader, err := pqarrow.NewFileReader(pf, pqarrow.ArrowReadProperties{}, mem)
	if err != nil {
		return nil, fmt.Errorf("failed to create arrow reader: %v", err)
	}

	// Read the entire table with context
	ctx := context.Background()
	table, err := arrowReader.ReadTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read parquet table: %v", err)
	}
	defer table.Release()

	// Initialize response
	response := &ParquetParseResponse{
		Elements:      []map[string]interface{}{},
		Relationships: []map[string]interface{}{},
		Links:         []map[string]interface{}{},
	}

	// Get column information
	schema := table.Schema()
	columnNames := make([]string, 0, len(schema.Fields()))
	columnTypes := make(map[string]string)

	for _, field := range schema.Fields() {
		columnNames = append(columnNames, field.Name)
		columnTypes[field.Name] = field.Type.String()
	}

	// Auto-detect text column if not specified
	textColumn := p.TextColumn
	if textColumn == "" {
		// Look for first string column
		for _, field := range schema.Fields() {
			if field.Type.ID() == arrow.STRING || field.Type.ID() == arrow.LARGE_STRING {
				textColumn = field.Name
				break
			}
		}
	}

	// Build document metadata
	docMetadata := make(map[string]interface{})
	if request.Metadata != nil {
		for k, v := range request.Metadata {
			docMetadata[k] = v
		}
	}

	// Extract metadata from first row if specified
	if len(p.MetadataColumns) > 0 && table.NumRows() > 0 {
		firstRowData := p.extractRowData(table, 0)
		for _, col := range p.MetadataColumns {
			if val, ok := firstRowData[col]; ok {
				docMetadata[col] = val
			}
		}
	}

	// Add table statistics
	docMetadata["row_count"] = table.NumRows()
	docMetadata["column_count"] = len(columnNames)
	docMetadata["columns"] = columnNames
	docMetadata["column_types"] = columnTypes
	docMetadata["file_size"] = fileInfo.Size()

	if textColumn != "" {
		docMetadata["text_column"] = textColumn
	}

	if p.GroupByColumn != "" && p.containsColumn(schema, p.GroupByColumn) {
		docMetadata["group_column"] = p.GroupByColumn
	}

	// Create document
	response.Document = map[string]interface{}{
		"doc_id":   request.ID,
		"doc_type": "parquet",
		"metadata": docMetadata,
	}

	// Create root element
	rootID := generateID("root")
	response.Elements = append(response.Elements, map[string]interface{}{
		"element_id":      rootID,
		"element_type":    "root",
		"content_preview": fmt.Sprintf("Parquet document with %d rows", table.NumRows()),
		"metadata":        docMetadata,
	})

	// Create body element
	bodyID := generateID("body")
	response.Elements = append(response.Elements, map[string]interface{}{
		"element_id":      bodyID,
		"element_type":    "body",
		"parent_id":       rootID,
		"content_preview": "Document body",
		"metadata":        map[string]interface{}{},
	})

	// Add root->body relationship
	response.Relationships = append(response.Relationships, map[string]interface{}{
		"relationship_id":   generateID("rel"),
		"source_id":        rootID,
		"target_id":        bodyID,
		"relationship_type": "contains",
	})

	// Process rows
	var currentGroupID string
	var currentGroupValue interface{}

	for i := int64(0); i < table.NumRows(); i++ {
		rowData := p.extractRowData(table, i)

		// Check for grouping
		if p.GroupByColumn != "" && p.containsColumn(schema, p.GroupByColumn) {
			groupValue := rowData[p.GroupByColumn]
			if groupValue != currentGroupValue && groupValue != nil {
				currentGroupValue = groupValue
				currentGroupID = generateID("group")

				// Create group element
				response.Elements = append(response.Elements, map[string]interface{}{
					"element_id":      currentGroupID,
					"element_type":    "header",
					"parent_id":       bodyID,
					"content_preview": fmt.Sprintf("Group: %v", groupValue),
					"metadata": map[string]interface{}{
						"group_value":  fmt.Sprintf("%v", groupValue),
						"group_column": p.GroupByColumn,
					},
				})

				// Add body->group relationship
				response.Relationships = append(response.Relationships, map[string]interface{}{
					"relationship_id":   generateID("rel"),
					"source_id":        bodyID,
					"target_id":        currentGroupID,
					"relationship_type": "contains",
				})
			}
		}

		// Create row element
		rowID := generateID("row")

		// Build row metadata (include all columns)
		rowMetadata := map[string]interface{}{
			"row_index": i,
		}
		for k, v := range rowData {
			if v != nil {
				rowMetadata[k] = fmt.Sprintf("%v", v)
			}
		}

		// Get text content for preview
		var textContent string
		if textColumn != "" && rowData[textColumn] != nil {
			textContent = fmt.Sprintf("%v", rowData[textColumn])
		} else {
			// If no text column, concatenate all values
			var parts []string
			for col, val := range rowData {
				if val != nil {
					parts = append(parts, fmt.Sprintf("%s: %v", col, val))
				}
			}
			textContent = strings.Join(parts, "; ")
		}

		// Truncate for preview
		contentPreview := textContent
		if len(contentPreview) > p.MaxContentPreview {
			contentPreview = contentPreview[:p.MaxContentPreview] + "..."
		}

		// Determine parent
		parentID := bodyID
		if currentGroupID != "" {
			parentID = currentGroupID
		}

		// Create row element
		response.Elements = append(response.Elements, map[string]interface{}{
			"element_id":      rowID,
			"element_type":    "paragraph",
			"parent_id":       parentID,
			"content_preview": contentPreview,
			"metadata":        rowMetadata,
		})

		// Add parent->row relationship
		response.Relationships = append(response.Relationships, map[string]interface{}{
			"relationship_id":   generateID("rel"),
			"source_id":        parentID,
			"target_id":        rowID,
			"relationship_type": "contains",
		})
	}

	// Add group statistics if grouping was used
	if p.GroupByColumn != "" && p.containsColumn(schema, p.GroupByColumn) {
		groupCount := p.countUniqueValues(table, p.GroupByColumn)
		docMetadata["group_count"] = groupCount
	}

	return response, nil
}

// extractRowData extracts data from a specific row
func (p *ParquetParser) extractRowData(table arrow.Table, rowIndex int64) map[string]interface{} {
	rowData := make(map[string]interface{})

	for colIdx, field := range table.Schema().Fields() {
		col := table.Column(colIdx)
		if col.Len() <= int(rowIndex) {
			continue
		}

		// Since we're reading the entire table, we can access by index directly
		colData := col.Data()
		if colData.Len() > 0 && int(rowIndex) < colData.Len() {
			// Access the chunked array
			chunks := colData.Chunks()
			rowInChunk := rowIndex
			for _, chunk := range chunks {
				if rowInChunk < int64(chunk.Len()) {
					// Found the chunk containing our row
					if !chunk.IsNull(int(rowInChunk)) {
						rowData[field.Name] = p.getValueFromArray(chunk, int(rowInChunk))
					}
					break
				}
				rowInChunk -= int64(chunk.Len())
			}
		}
	}

	return rowData
}

// getValueFromArray extracts a value from an arrow array
func (p *ParquetParser) getValueFromArray(arr arrow.Array, index int) interface{} {
	switch arr := arr.(type) {
	case *array.String:
		return arr.Value(index)
	case *array.Int64:
		return arr.Value(index)
	case *array.Int32:
		return arr.Value(index)
	case *array.Float64:
		return arr.Value(index)
	case *array.Float32:
		return arr.Value(index)
	case *array.Boolean:
		return arr.Value(index)
	case *array.Binary:
		return string(arr.Value(index))
	default:
		// For other types, use string representation
		return arr.ValueStr(index)
	}
}

// containsColumn checks if a column exists in the schema
func (p *ParquetParser) containsColumn(schema *arrow.Schema, columnName string) bool {
	for _, field := range schema.Fields() {
		if field.Name == columnName {
			return true
		}
	}
	return false
}

// countUniqueValues counts unique values in a column
func (p *ParquetParser) countUniqueValues(table arrow.Table, columnName string) int {
	uniqueValues := make(map[string]bool)

	// Find column index
	colIdx := -1
	for i, field := range table.Schema().Fields() {
		if field.Name == columnName {
			colIdx = i
			break
		}
	}

	if colIdx == -1 {
		return 0
	}

	col := table.Column(colIdx)
	colData := col.Data()
	chunks := colData.Chunks()
	for _, chunk := range chunks {
		for i := 0; i < chunk.Len(); i++ {
			if !chunk.IsNull(i) {
				val := p.getValueFromArray(chunk, i)
				uniqueValues[fmt.Sprintf("%v", val)] = true
			}
		}
	}

	return len(uniqueValues)
}

// ToJSON converts the response to JSON
func (r *ParquetParseResponse) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}